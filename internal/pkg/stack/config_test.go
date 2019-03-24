/*
 * Copyright 2019 The Sugarkube Authors
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

package stack

import (
	"github.com/stretchr/testify/assert"
	"github.com/sugarkube/sugarkube/internal/pkg/acquirer"
	"github.com/sugarkube/sugarkube/internal/pkg/structs"
	"gopkg.in/yaml.v2"
	"testing"
)

// Helper to get acquirers in a single-valued context
func discardErr(acquirer acquirer.Acquirer, err error) acquirer.Acquirer {
	if err != nil {
		panic(err)
	}

	return acquirer
}

func TestLoadStackConfigGarbagePath(t *testing.T) {
	_, err := loadStackConfigFile("fake-path", "/fake/~/some?/~/garbage")
	assert.Error(t, err)
}

func TestLoadStackConfigNonExistentPath(t *testing.T) {
	_, err := loadStackConfigFile("missing-path", "/missing/stacks.yaml")
	assert.Error(t, err)
}

func TestLoadStackConfigDir(t *testing.T) {
	_, err := loadStackConfigFile("dir-path", "../../testdata")
	assert.Error(t, err)
}

func GetTestManifests() (*Manifest, *Manifest) {
	manifest1 := structs.ManifestDescriptor{
		Id:  "",
		Uri: "../../testdata/manifests/manifest1.yaml",
		Overrides: map[string]interface{}{
			"kappA": map[interface{}]interface{}{
				"state": "absent",
				"sources": map[interface{}]interface{}{
					"pathA": map[interface{}]interface{}{
						"options": map[interface{}]interface{}{
							"branch": "stable",
						},
					},
				},
				"vars": map[interface{}]interface{}{
					"sizeVar":  "mediumOverridden",
					"stackVar": "setInOverrides",
				},
			},
		},
	}

	manifest1KappDescriptors := []structs.KappDescriptor{
		{
			Id:    "kappA",
			State: "present",
			Sources: []structs.Source{
				{
					Uri: "git@github.com:sugarkube/kapps-A.git//some/pathA#kappA-0.1.0",
				},
			},
			Vars: map[string]interface{}{
				"sizeVar": "big",
				"colours": []interface{}{
					"red",
					"black",
				},
			},
		},
	}

	manifest1.UnparsedKapps = manifest1UnparsedKapps

	manifest2 := structs.ManifestDescriptor{
		Id:  "exampleManifest2",
		Uri: "../../testdata/manifests/manifest2.yaml",
		Options: ManifestOptions{
			Parallelisation: uint16(1),
		},
	}

	manifest2KappDescriptors := []structs.KappDescriptor{
		{
			Id:    "kappC",
			State: "present",
			Sources: []structs.Source{
				{
					Id:  "special",
					Uri: "git@github.com:sugarkube/kapps-C.git//kappC/some/special-path#kappC-0.3.0",
				},
				{Uri: "git@github.com:sugarkube/kapps-C.git//kappC/some/pathZ#kappZ-0.3.0"},
				{Uri: "git@github.com:sugarkube/kapps-C.git//kappC/some/pathX#kappX-0.3.0"},
				{Uri: "git@github.com:sugarkube/kapps-C.git//kappC/some/pathY#kappY-0.3.0"},
			},
		},
		{
			Id:    "kappB",
			State: "present",
			Sources: []structs.Source{
				{Uri: "git@github.com:sugarkube/kapps-B.git//some/pathB#kappB-0.2.0"},
			},
		},
		{
			Id:    "kappD",
			State: "present",
			Sources: []structs.Source{
				{
					Uri: "git@github.com:sugarkube/kapps-D.git//some/pathD#kappD-0.2.0",
					Options: map[string]interface{}{
						"branch": "kappDBranch",
					},
				},
			},
		},
		{
			Id:    "kappA",
			State: "present",
			Sources: []structs.Source{
				{IncludeValues: false,
					Uri: "git@github.com:sugarkube/kapps-A.git//some/pathA#kappA-0.2.0"},
			},
		},
	}

	manifest2.UnparsedKapps = manifest2UnparsedKapps

	return &manifest1, &manifest2
}

func TestLoadStackConfig(t *testing.T) {

	manifest1, manifest2 := GetTestManifests()

	expected := &structs.Stack{
		Name:        "large",
		FilePath:    "../../testdata/stacks.yaml",
		Provider:    "local",
		Provisioner: "minikube",
		Profile:     "local",
		Cluster:     "large",
		ProviderVarsDirs: []string{
			"./stacks/",
		},
		TemplateDirs: []string{
			"templates1/",
			"templates2/",
		},
		Manifests: []*Manifest{
			manifest1,
			manifest2,
		},
		KappVarsDirs: []string{
			"sample-kapp-vars/",
		},
	}

	actual, err := loadStackConfigFile("large", "../../testdata/stacks.yaml")
	assert.Nil(t, err)
	assert.Equal(t, expected, actual, "unexpected stack")
}

func TestLoadStackConfigMissingStackName(t *testing.T) {
	_, err := loadStackConfigFile("missing-stack-name", "../../testdata/stacks.yaml")
	assert.Error(t, err)
}

func TestDir(t *testing.T) {
	stack := structs.Stack{
		FilePath: "../../testdata/stacks.yaml",
	}

	expected := "../../testdata"
	actual := stack.Dir()

	assert.Equal(t, expected, actual, "Unexpected config dir")
}

// this should return the path to the current working dir, but it's difficult
// to meaningfully test.
func TestDirBlank(t *testing.T) {
	stack := StackConfig{}
	actual := stack.Dir()

	assert.NotNil(t, actual, "Unexpected config dir")
	assert.NotEmpty(t, actual, "Unexpected config dir")
}

func TestGetKappVarsFromFiles(t *testing.T) {

	manifest1, manifest2 := GetTestManifests()

	stackConfig := structs.Stack{
		Name:        "large",
		FilePath:    "../../testdata/stacks.yaml",
		Provider:    "test-provider",
		Provisioner: "test-provisioner",
		Profile:     "test-profile",
		Cluster:     "test-cluster",
		Account:     "test-account",
		Region:      "test-region1",
		ProviderVarsDirs: []string{
			"./stacks/",
		},
		KappVarsDirs: []string{
			"./sample-kapp-vars",
			"./sample-kapp-vars/kapp-vars/",
			"./sample-kapp-vars/kapp-vars2/",
		},
		Manifests: []*Manifest{
			manifest1,
			manifest2,
		},
	}

	expected := `globals:
 account: test-account-val
kapp: kappA-val
kappASisterDir: extra-val
kappOverride: kappA-val-override
profile: test-profile-val
region: test-region1-val
regionOverride: region-val-override
`

	kappObj := &stackConfig.Manifests[0].Installables()[0]
	results, err := kappObj.GetVarsFromFiles(&stackConfig)
	assert.Nil(t, err)

	yamlResults, err := yaml.Marshal(results)
	assert.Nil(t, err)

	assert.Equal(t, expected, string(yamlResults[:]))
}

func TestAllManifests(t *testing.T) {
	stackConfig := StackConfig{
		Manifests: []*Manifest{
			{
				UnparsedKapps: []Kapp{
					{
						Id: "kapp2",
					},
				},
			},
			{
				UnparsedKapps: []Kapp{
					{
						Id: "kapp3",
					},
				},
			},
		},
	}

	assert.Equal(t, 2, len(stackConfig.Manifests))
}