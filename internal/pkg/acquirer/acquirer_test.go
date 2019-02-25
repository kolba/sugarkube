/*
 * Copyright 2018 The Sugarkube Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package acquirer

import (
	"github.com/stretchr/testify/assert"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"testing"
)

func init() {
	log.ConfigureLogger("debug", false)
}

func TestNewAcquirerError(t *testing.T) {
	actual, err := acquirerFactory("nonsense", map[string]string{})
	assert.NotNil(t, err)
	assert.Nil(t, actual)
}

func TestNewGitAcquirerPartial(t *testing.T) {
	actual, err := acquirerFactory(GIT_ACQUIRER, map[string]string{
		"branch": "master",
	})
	assert.NotNil(t, err)
	assert.Nil(t, actual)
}

var defaultSettings = map[string]string{
	"uri":    "git@github.com:sugarkube/kapps.git",
	"branch": "master",
	"path":   "incubator/tiller/",
}

var expectedAcquirer = &GitAcquirer{
	id:            "tiller",
	uri:           "git@github.com:sugarkube/kapps.git",
	branch:        "master",
	path:          "incubator/tiller/",
	includeValues: true,
}

func TestNewGitAcquirerFull(t *testing.T) {
	actual, err := acquirerFactory(GIT_ACQUIRER, defaultSettings)
	assert.Nil(t, err)
	assert.Equal(t, expectedAcquirer, actual,
		"Fully-defined git acquirer incorrectly created")
}

func TestNewAcquirerGit(t *testing.T) {
	actual, err := NewAcquirer(defaultSettings)
	assert.Nil(t, err)
	assert.Equal(t, expectedAcquirer, actual)
}

func TestNewAcquirerGitExplicit(t *testing.T) {
	actual, err := NewAcquirer(
		map[string]string{
			ACQUIRER_KEY: GIT_ACQUIRER,
			"uri":        "git@github.com:sugarkube/kapps.git",
			"branch":     "master",
			"path":       "incubator/tiller/",
		})
	assert.Nil(t, err)
	assert.Equal(t, expectedAcquirer, actual)
}

func TestNewAcquirerNilUriError(t *testing.T) {
	actual, err := NewAcquirer(map[string]string{
		"uri": "",
	})
	assert.NotNil(t, err)
	assert.Nil(t, actual)
}
