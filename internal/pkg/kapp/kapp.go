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

package kapp

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/acquirer"
	"github.com/sugarkube/sugarkube/internal/pkg/convert"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"gopkg.in/yaml.v2"
	"path/filepath"
	"strings"
)

type Template struct {
	Source string
	Dest   string
}

// Populated from the kapp's sugarkube.yaml file
type Config struct {
	EnvVars    map[string]interface{} `yaml:"envVars"`
	Version    string
	TargetArgs map[string]map[string][]map[string]string `yaml:"targets"`
}

type Kapp struct {
	Id       string
	manifest *Manifest
	cacheDir string
	Config   Config
	// if true, this kapp should be present after completing, otherwise it
	// should be absent. This is here instead of e.g. putting all kapps into
	// an enclosing struct with 'present' and 'absent' properties so we can
	// preserve ordering. This approach lets users strictly define the ordering
	// of installation and deletion operations.
	ShouldBePresent bool
	// todo - merge these values with the rest of the merged values prior to invoking a kapp
	vars      map[string]interface{}
	Sources   []acquirer.Acquirer
	Templates []Template
}

const PRESENT_KEY = "present"
const ABSENT_KEY = "absent"
const SOURCES_KEY = "sources"
const TEMPLATES_KEY = "templates"
const VARS_KEY = "vars"
const ID_KEY = "id"

// Sets the root cache directory the kapp is checked out into
func (k *Kapp) SetCacheDir(cacheDir string) {
	log.Logger.Debugf("Setting cache dir on kapp '%s' to '%s'",
		k.FullyQualifiedId(), cacheDir)
	k.cacheDir = cacheDir
}

// Returns the fully-qualified ID of a kapp
func (k Kapp) FullyQualifiedId() string {
	if k.manifest == nil {
		return k.Id
	} else {
		return strings.Join([]string{k.manifest.Id, k.Id}, ":")
	}
}

// Returns the physical path to this kapp in a cache
func (k Kapp) CacheDir() string {
	cacheDir := filepath.Join(k.cacheDir, k.manifest.Id, k.Id)

	// if no cache dir has been set (e.g. because the user is doing a dry-run),
	// don't return an absolute path
	if k.cacheDir != "" {
		absCacheDir, err := filepath.Abs(cacheDir)
		if err != nil {
			panic(fmt.Sprintf("Couldn't convert path to absolute path: %#v", err))
		}

		cacheDir = absCacheDir
	} else {
		log.Logger.Debug("No cache dir has been set on kapp. Cache dir will " +
			"not be converted to an absolute path.")
	}

	return cacheDir
}

// Returns certain kapp data that should be exposed as variables when running kapps
func (k Kapp) GetIntrinsicData() map[string]string {
	return map[string]string{
		"id":              k.Id,
		"shouldBePresent": fmt.Sprintf("%#v", k.ShouldBePresent),
		"cacheRoot":       k.CacheDir(),
	}
}

// Instantiates a new Kapp, returning errors if any required settings are missing
func newKapp(manifest *Manifest, id string, shouldBePresent bool, vars map[string]interface{},
	templates []Template, sources []acquirer.Acquirer) (*Kapp, error) {

	if id == "" {
		return nil, errors.New("Can't instantiate a kapp with no ID")
	}

	if manifest == nil {
		return nil, errors.New("Can't instantiate a kapp with no associated manifest")
	}

	kappObj := Kapp{
		Id:              id,
		manifest:        manifest,
		ShouldBePresent: shouldBePresent,
		vars:            vars,
		Templates:       templates,
		Sources:         sources,
	}

	log.Logger.Debugf("Instantiated kapp: %#v", kappObj)

	return &kappObj, nil
}

// Parses kapps and adds them to an array
func parseKapps(manifest *Manifest, kapps *[]Kapp, kappDefinitions []interface{}, shouldBePresent bool) error {

	// parse each kapp definition
	for _, v := range kappDefinitions {
		log.Logger.Debugf("Parsing kapp from values: %#v", v)

		valuesMap, err := convert.MapInterfaceInterfaceToMapStringInterface(v.(map[interface{}]interface{}))
		if err != nil {
			return errors.Wrapf(err, "Error converting manifest value to map")
		}

		log.Logger.Debugf("kapp valuesMap=%#v", valuesMap)

		// Return a useful error message if no ID has been declared. We need to do this here as well as when
		// instantiating a kapp because this catches if the ID key is missing altogether
		id, ok := valuesMap[ID_KEY].(string)
		if !ok {
			return errors.New(fmt.Sprintf("No ID declared for kapp: %#v", valuesMap))
		}

		vars, err := parseVariables(valuesMap)
		if err != nil {
			return errors.WithStack(err)
		}

		templates, err := parseTemplates(valuesMap)
		if err != nil {
			return errors.WithStack(err)
		}

		acquirers, err := parseAcquirers(valuesMap)
		if err != nil {
			return errors.WithStack(err)
		}

		kapp, err := newKapp(manifest, id, shouldBePresent, vars, templates, acquirers)

		*kapps = append(*kapps, *kapp)
	}

	return nil
}

// Parse variables in a Kapp values map
func parseVariables(valuesMap map[string]interface{}) (map[string]interface{}, error) {
	var kappVars map[string]interface{}

	rawKappVars, ok := valuesMap[VARS_KEY]
	if ok {
		varsBytes, err := yaml.Marshal(rawKappVars)
		if err != nil {
			return nil, errors.Wrapf(err, "Error marshalling vars in kapp: %#v", valuesMap)
		}

		err = yaml.Unmarshal(varsBytes, &kappVars)
		if err != nil {
			return nil, errors.Wrapf(err, "Error unmarshalling vars for kapp: %#v", valuesMap)
		}

		log.Logger.Debugf("Parsed vars from kapp: %s", kappVars)
	} else {
		log.Logger.Debugf("No vars found in kapp")
	}

	return kappVars, nil
}

// Parse templates from a Kapp values map
func parseTemplates(valuesMap map[string]interface{}) ([]Template, error) {
	templateBytes, err := yaml.Marshal(valuesMap[TEMPLATES_KEY])
	if err != nil {
		return nil, errors.Wrapf(err, "Error marshalling kapp templates: %#v", valuesMap)
	}

	log.Logger.Debugf("Marshalled templates YAML: %s", templateBytes)

	templates := []Template{}
	err = yaml.Unmarshal(templateBytes, &templates)
	if err != nil {
		return nil, errors.Wrapf(err, "Error unmarshalling template YAML: %s", templateBytes)
	}

	return templates, nil
}

// Parse acquirers from a Kapp values map
func parseAcquirers(valuesMap map[string]interface{}) ([]acquirer.Acquirer, error) {
	sourcesBytes, err := yaml.Marshal(valuesMap[SOURCES_KEY])
	if err != nil {
		return nil, errors.Wrapf(err, "Error marshalling sources yaml: %#v", valuesMap)
	}

	log.Logger.Debugf("Marshalled sources YAML: %s", sourcesBytes)

	sourcesMaps := []map[interface{}]interface{}{}
	err = yaml.UnmarshalStrict(sourcesBytes, &sourcesMaps)
	if err != nil {
		return nil, errors.Wrapf(err, "Error unmarshalling yaml: %s", sourcesBytes)
	}

	log.Logger.Debugf("kapp sourcesMaps=%#v", sourcesMaps)

	kappSources := make([]acquirer.Acquirer, 0)
	// now we have a list of sources, get the acquirer for each one
	for _, sourceMap := range sourcesMaps {
		sourceStringMap, err := convert.MapInterfaceInterfaceToMapStringString(sourceMap)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		acquirerImpl, err := acquirer.NewAcquirer(sourceStringMap)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		log.Logger.Debugf("Got acquirer %#v", acquirerImpl)

		kappSources = append(kappSources, acquirerImpl)
	}

	return kappSources, nil
}

// Parses a manifest YAML. It separately parses all kapps that should be present and all those that should be
// absent, and returns a single list containing them all.
func parseManifestYaml(manifest *Manifest, data map[string]interface{}) ([]Kapp, error) {
	kapps := make([]Kapp, 0)

	log.Logger.Debugf("Manifest data to parse: %#v", data)

	presentKapps, ok := data[PRESENT_KEY]
	if ok {
		err := parseKapps(manifest, &kapps, presentKapps.([]interface{}), true)
		if err != nil {
			return nil, errors.Wrap(err, "Error parsing present kapps")
		}
	}

	absentKapps, ok := data[ABSENT_KEY]
	if ok {
		err := parseKapps(manifest, &kapps, absentKapps.([]interface{}), false)
		if err != nil {
			return nil, errors.Wrap(err, "Error parsing absent kapps")
		}
	}

	log.Logger.Debugf("Parsed kapps to install and remove: %#v", kapps)

	return kapps, nil
}
