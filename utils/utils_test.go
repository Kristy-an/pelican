/***************************************************************
*
* Copyright (C) 2024, Pelican Project, Morgridge Institute for Research
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

package utils

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test the functionality of CheckValidQuery and all its edge cases
func TestValidQuery(t *testing.T) {
	// Test recursive query passes
	t.Run("testValidRecursive", func(t *testing.T) {
		transferStr := "pelican://something/here?recursive=true"
		transferUrl, err := url.Parse(transferStr)
		assert.NoError(t, err)

		err = CheckValidQuery(transferUrl)
		assert.NoError(t, err)
	})

	// Test directread query passes
	t.Run("testValidDirectRead", func(t *testing.T) {
		transferStr := "pelican://something/here?directread"
		transferUrl, err := url.Parse(transferStr)
		assert.NoError(t, err)

		err = CheckValidQuery(transferUrl)
		assert.NoError(t, err)
	})

	// Test pack query passes
	t.Run("testValidPack", func(t *testing.T) {
		transferStr := "pelican://something/here?pack=tar.gz"
		transferUrl, err := url.Parse(transferStr)
		assert.NoError(t, err)

		err = CheckValidQuery(transferUrl)
		assert.NoError(t, err)
	})

	// Test a typo/invalid query fails
	t.Run("testInvalidQuery", func(t *testing.T) {
		transferStr := "pelican://something/here?recrustive=true"
		transferUrl, err := url.Parse(transferStr)
		assert.NoError(t, err)

		err = CheckValidQuery(transferUrl)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid query parameter(s) recrustive=true provided in url pelican://something/here?recrustive=true")
	})

	// Test that both pack and recursive queries are not allowed together (only in plugin case)
	t.Run("testBothPackAndRecursiveFailure", func(t *testing.T) {
		transferStr := "pelican://something/here?pack=tar.gz&recursive=true"
		transferUrl, err := url.Parse(transferStr)
		assert.NoError(t, err)

		err = CheckValidQuery(transferUrl)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot have both recursive and pack query parameters")
	})

	// Test we pass with both pack and directread
	t.Run("testBothPackAndDirectReadSuccess", func(t *testing.T) {
		transferStr := "pelican://something/here?pack=tar.gz&directread"
		transferUrl, err := url.Parse(transferStr)
		assert.NoError(t, err)

		err = CheckValidQuery(transferUrl)
		assert.NoError(t, err)
	})

	// Test we pass with both recursive and directread (just plugin case)
	t.Run("testBothRecursiveAndDirectReadSuccess", func(t *testing.T) {
		transferStr := "pelican://something/here?recursive=true&directread"
		transferUrl, err := url.Parse(transferStr)
		assert.NoError(t, err)

		err = CheckValidQuery(transferUrl)
		assert.NoError(t, err)
	})

	// Test if we have a value assigned to directread, we fail
	t.Run("testValueOnDirectReadNoFailure", func(t *testing.T) {
		transferStr := "pelican://something/here?directread=false"
		transferUrl, err := url.Parse(transferStr)
		assert.NoError(t, err)

		err = CheckValidQuery(transferUrl)
		assert.NoError(t, err)
	})
}

func TestApplyIPMask(t *testing.T) {
	t.Run("testValidIPv4", func(t *testing.T) {
		expectedMaskedIP := "192.168.1.0"
		validIPv4 := "192.168.1.1"
		maskedIP, ok := ApplyIPMask(validIPv4)
		assert.True(t, ok)
		assert.Equal(t, expectedMaskedIP, maskedIP)
	})

	t.Run("testValidIPv6", func(t *testing.T) {
		expectedMaskedIP := "2001:db8:3333:4444::"
		validIPv6 := "2001:0db8:3333:4444:5555:6666:7777:8888"
		maskedIP, ok := ApplyIPMask(validIPv6)
		assert.True(t, ok)
		assert.Equal(t, expectedMaskedIP, maskedIP)
	})

	t.Run("testInvalidInput", func(t *testing.T) {
		invalid := "abc.123"
		maskedIP, ok := ApplyIPMask(invalid)
		assert.False(t, ok)
		assert.Equal(t, maskedIP, invalid)
	})
}

func TestExtractAndMaskIP(t *testing.T) {
	t.Run("testWrappedIPv4", func(t *testing.T) {
		expectedMaskedIP := "192.168.1.0"
		wrappedValidIPv4 := "[192.168.1.1]"
		maskedIP, ok := ExtractAndMaskIP(wrappedValidIPv4)
		assert.True(t, ok)
		assert.Equal(t, expectedMaskedIP, maskedIP)
	})

	t.Run("testWrappedIPv6", func(t *testing.T) {
		expectedMaskedIP := "2001:db8:3333:4444::"
		wrappedValidIPv6 := "[2001:0db8:3333:4444:5555:6666:7777:8888]"
		maskedIP, ok := ExtractAndMaskIP(wrappedValidIPv6)
		assert.True(t, ok)
		assert.Equal(t, expectedMaskedIP, maskedIP)
	})

	t.Run("testWrappedInvalid", func(t *testing.T) {
		invalid := "[abc.123]"
		expected := "abc.123"
		maskedIP, ok := ExtractAndMaskIP(invalid)
		assert.False(t, ok)
		assert.Equal(t, maskedIP, expected)
	})

	t.Run("testUnwrappedIPv4", func(t *testing.T) {
		expectedMaskedIP := "192.168.1.0"
		validIPv4 := "192.168.1.1"
		maskedIP, ok := ExtractAndMaskIP(validIPv4)
		assert.True(t, ok)
		assert.Equal(t, expectedMaskedIP, maskedIP)
	})

	t.Run("testUnwrappedIPv6", func(t *testing.T) {
		expectedMaskedIP := "2001:db8:3333:4444::"
		validIPv6 := "2001:0db8:3333:4444:5555:6666:7777:8888"
		maskedIP, ok := ExtractAndMaskIP(validIPv6)
		assert.True(t, ok)
		assert.Equal(t, expectedMaskedIP, maskedIP)
	})

	t.Run("testWrappedReal", func(t *testing.T) {
		real := "[::ffff:79.110.62.117]"
		expected := "79.110.62.0"
		maskedIP, ok := ExtractAndMaskIP(real)
		assert.True(t, ok)
		assert.Equal(t, expected, maskedIP)
	})
}
