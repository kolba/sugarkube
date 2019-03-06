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

package provider

import (
	"github.com/stretchr/testify/assert"
	"github.com/sugarkube/sugarkube/internal/pkg/kapp"
	"testing"
)

func TestStackConfigVars(t *testing.T) {
	sc, err := kapp.LoadStackConfig("large", "../../testdata/stacks.yaml")
	assert.Nil(t, err)

	expected := map[string]interface{}{
		"provisioner": map[interface{}]interface{}{
			"binary": "minikube",
			"params": map[interface{}]interface{}{
				"start": map[interface{}]interface{}{
					"disk_size": "120g",
					"memory":    4096,
					"cpus":      4,
				},
			},
		},
	}

	providerImpl, err := newProviderImpl(sc.Provider, sc)
	assert.Nil(t, err)

	actual, err := LoadProviderVars(providerImpl, sc)
	assert.Nil(t, err)
	assert.Equal(t, expected, actual, "Mismatching vars")
}

func TestNewNonExistentProvider(t *testing.T) {
	sc, err := kapp.LoadStackConfig("large", "../../testdata/stacks.yaml")
	assert.Nil(t, err)
	actual, err := newProviderImpl("bananas", sc)
	assert.NotNil(t, err)
	assert.Nil(t, actual)
}

func TestNewProviderError(t *testing.T) {
	sc, err := kapp.LoadStackConfig("large", "../../testdata/stacks.yaml")
	assert.Nil(t, err)
	actual, err := newProviderImpl("nonsense", sc)
	assert.NotNil(t, err)
	assert.Nil(t, actual)
}

func TestNewLocalProvider(t *testing.T) {
	sc, err := kapp.LoadStackConfig("large", "../../testdata/stacks.yaml")
	assert.Nil(t, err)
	actual, err := newProviderImpl(LOCAL, sc)
	assert.Nil(t, err)
	assert.Equal(t, &LocalProvider{}, actual)
}

func TestNewAWSProvider(t *testing.T) {
	sc, err := kapp.LoadStackConfig("large", "../../testdata/stacks.yaml")
	assert.Nil(t, err)
	actual, err := newProviderImpl(AWS, sc)
	assert.Nil(t, err)
	assert.Equal(t, &AwsProvider{}, actual)
}
