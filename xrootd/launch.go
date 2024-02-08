//go:build !windows

/***************************************************************
 *
 * Copyright (C) 2023, Pelican Project, Morgridge Institute for Research
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you
 * may not use this file except in compliance with the License.  You may
 * obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 ***************************************************************/

package xrootd

import (
	"context"
	_ "embed"
	"net/url"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/pelicanplatform/pelican/config"
	"github.com/pelicanplatform/pelican/daemon"
	"github.com/pelicanplatform/pelican/param"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"
)

type (
	PrivilegedXrootdLauncher struct {
		daemonName string
		configPath string
	}

	UnprivilegedXrootdLauncher struct {
		daemon.DaemonLauncher
	}
)

func (launcher PrivilegedXrootdLauncher) Name() string {
	return launcher.daemonName
}

func makeUnprivilegedXrootdLauncher(daemonName string, configPath string, isCache bool) (result UnprivilegedXrootdLauncher, err error) {
	result.DaemonName = daemonName
	result.Uid = -1
	result.Gid = -1
	xrootdRun := param.Xrootd_RunLocation.GetString()
	pidFile := filepath.Join(xrootdRun, "xrootd.pid")
	result.Args = []string{daemonName, "-s", pidFile, "-c", configPath}

	if config.IsRootExecution() {
		result.Uid, err = config.GetDaemonUID()
		if err != nil {
			return
		}
		result.Gid, err = config.GetDaemonGID()
		if err != nil {
			return
		}
	}

	if isCache {
		result.ExtraEnv = []string{"XRD_PLUGINCONFDIR=" + filepath.Join(xrootdRun, "cache-client.plugins.d")}
	}
	return
}

func ConfigureLaunchers(privileged bool, configPath string, useCMSD bool, enableCache bool) (launchers []daemon.Launcher, err error) {
	if privileged {
		launchers = append(launchers, PrivilegedXrootdLauncher{"xrootd", configPath})
		if useCMSD {
			launchers = append(launchers, PrivilegedXrootdLauncher{"cmsd", configPath})
		}
	} else {
		var result UnprivilegedXrootdLauncher
		result, err = makeUnprivilegedXrootdLauncher("xrootd", configPath, enableCache)
		if err != nil {
			return
		}
		launchers = append(launchers, result)
		if useCMSD {
			result, err = makeUnprivilegedXrootdLauncher("cmsd", configPath, false)
			if err != nil {
				return
			}
			launchers = append(launchers, result)
		}
	}
	return
}

func LaunchOriginDaemons(ctx context.Context, launchers []daemon.Launcher, egrp *errgroup.Group) (err error) {
	startupChan := make(chan int)
	readyChan := make(chan bool)
	defer close(readyChan)
	re := regexp.MustCompile(`^------ xrootd [A-Za-z0-9]+@[A-Za-z0-9.\-]+:([0-9]+) initialization complete.*`)
	config.AddFilter(&config.RegexpFilter{
		Name:   "xrootd_startup",
		Regexp: re,
		Levels: []log.Level{log.InfoLevel},
		Fire: func(e *log.Entry) error {
			portStrs := re.FindStringSubmatch(e.Message)
			if len(portStrs) < 1 {
				portStrs = []string{"", ""}
			}
			port, err := strconv.Atoi(portStrs[1])
			if err != nil {
				port = -1
			}
			if _, ok := <-readyChan; ok {
				startupChan <- port
			}
			return nil
		},
	})
	config.AddFilter(&config.RegexpFilter{
		Name:   "xrootd_startup_failed",
		Regexp: regexp.MustCompile(`^------ xrootd [A-Za-z0-9]+@[A-Za-z0-9.\-]+:([0-9]+) initialization failed.*`),
		Levels: []log.Level{log.InfoLevel},
		Fire: func(e *log.Entry) error {
			if _, ok := <-readyChan; ok {
				startupChan <- -1
			}
			return nil
		},
	})
	defer func() {
		config.RemoveFilter("xrootd_startup")
		config.RemoveFilter("xrootd_startup_failed")
		close(startupChan)
	}()

	if err = daemon.LaunchDaemons(ctx, launchers, egrp); err != nil {
		return err
	}

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case readyChan <- true:
		port := <-startupChan
		if port == -1 {
			return errors.New("Xrootd initialization failed")
		} else {
			viper.Set("Origin.Port", port)
			if originUrl, err := url.Parse(param.Origin_Url.GetString()); err == nil {
				originUrl.Host = originUrl.Hostname() + ":" + strconv.Itoa(port)
				viper.Set("Origin.Url", originUrl.String())
				log.Debugln("Resetting Origin.Url to", originUrl.String())
			}
			log.Infoln("Origin startup complete on port", port)
		}
	case <-ticker.C:
		log.Errorln("XRootD did not startup after 10s of waiting")
		return errors.New("XRootD did not startup after 10s of waiting")
	}

	return nil
}
