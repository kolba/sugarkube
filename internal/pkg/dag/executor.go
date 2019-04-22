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

package dag

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/sugarkube/sugarkube/internal/pkg/cmd/cli/cluster"
	"github.com/sugarkube/sugarkube/internal/pkg/constants"
	"github.com/sugarkube/sugarkube/internal/pkg/installer"
	"github.com/sugarkube/sugarkube/internal/pkg/interfaces"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
	"github.com/sugarkube/sugarkube/internal/pkg/registry"
	"github.com/sugarkube/sugarkube/internal/pkg/structs"
	"os"
	"path/filepath"
	"strings"
)

// todo - make this configurable
const parallelisation = 5

// todo - should we add options to skip templating or running post actions?
// Traverses the DAG executing the named action on marked/processable nodes depending on the
// given options
func (d *dag) Execute(action string, stackObj interfaces.IStack, plan bool, apply bool, dryRun bool) error {
	processCh := make(chan NamedNode, parallelisation)
	doneCh := make(chan NamedNode)
	errCh := make(chan error)

	log.Logger.Infof("Executing DAG with plan=%v, apply=%v, dryRun=%v", plan, apply, dryRun)

	// create the worker pool
	for w := uint16(0); w < parallelisation; w++ {
		go worker(d, processCh, doneCh, errCh, action, stackObj, plan, apply, dryRun)
	}

	var finishedCh <-chan bool

	switch action {
	case constants.DagActionInstall:
		finishedCh = d.WalkDown(processCh, doneCh)
		break
	// todo - implement
	//case constants.DagActionDelete:
	//	finishedCh = d.WalkUp(processCh, doneCh)
	//	break
	default:
		return fmt.Errorf("Invalid action on DAG: %s", action)
	}

	for {
		select {
		case err := <-errCh:
			return errors.Wrapf(err, "Error processing kapp")
		case node := <-doneCh:
			if node.shouldProcess {
				log.Logger.Infof("Kapp '%s' processed", node.Name())
			}
		case <-finishedCh:
			log.Logger.Infof("Finished processing kapps")
			break
		}
	}
}

// Processes an installable, either installing/deleting it, running post actions or
// loading its outputs, etc.
func worker(dagObj *dag, processCh <-chan NamedNode, doneCh chan NamedNode, errCh chan error,
	action string, stackObj interfaces.IStack, plan bool, apply bool, dryRun bool) {

	for node := range processCh {
		installableObj := node.installableObj

		kappRootDir := installableObj.GetCacheDir()
		log.Logger.Infof("Processing kapp '%s' in %s", installableObj.FullyQualifiedId(), kappRootDir)

		// todo - print (to stdout) details of the kapp being executed

		_, err := os.Stat(kappRootDir)
		if err != nil {
			msg := fmt.Sprintf("Kapp '%s' doesn't exist in the cache at '%s'", installableObj.Id(), kappRootDir)
			log.Logger.Warn(msg)
			errCh <- errors.Wrap(err, msg)
		}

		// kapp exists, Instantiate an installer in case we need it (for now, this will always be a Make installer)
		installerImpl, err := installer.New(installer.MAKE, stackObj.GetProvider())
		if err != nil {
			errCh <- errors.Wrapf(err, "Error instantiating installer for "+
				"kapp '%s'", installableObj.Id())
		}

		// todo - support installing, deleting, running 'make clean', templating and printing out the vars
		//  for each marked kapp
		switch action {
		case constants.DagActionInstall:
			installOrDelete(true, dagObj, node, installerImpl, stackObj, plan, apply, dryRun, errCh)
			break
		case constants.DagActionDelete:
			installOrDelete(false, dagObj, node, installerImpl, stackObj, plan, apply, dryRun, errCh)
			break
		}

		doneCh <- node
	}
}

// Implements the install action. Nodes that should be processed are installed. All nodes load any outputs
// and merge them with their parents' outputs.
func installOrDelete(install bool, dagObj *dag, node NamedNode, installerImpl interfaces.IInstaller, stackObj interfaces.IStack,
	plan bool, apply bool, dryRun bool, errCh chan error) {

	installableObj := node.installableObj

	// only plan or apply kapps that have been flagged for processing
	if node.shouldProcess && plan {
		if install {
			err := installerImpl.Install(installableObj, stackObj, false, true, dryRun)
			if err != nil {
				errCh <- errors.Wrapf(err, "Error planning kapp '%s'", installableObj.Id())
			}
		} else {
			err := installerImpl.Delete(installableObj, stackObj, false, true, dryRun)
			if err != nil {
				errCh <- errors.Wrapf(err, "Error planning kapp '%s'", installableObj.Id())
			}
		}
	}

	if node.shouldProcess && apply {
		if install {
			err := installerImpl.Install(installableObj, stackObj, true, true, dryRun)
			if err != nil {
				errCh <- errors.Wrapf(err, "Error installing kapp '%s'", installableObj.Id())
			}
		} else {
			err := installerImpl.Delete(installableObj, stackObj, true, true, dryRun)
			if err != nil {
				errCh <- errors.Wrapf(err, "Error deleting kapp '%s'", installableObj.Id())
			}
		}
	}

	var outputs map[string]interface{}

	// try to load kapp outputs and fail if we can't (assume we only need to do this when installing)
	if install && installableObj.HasOutputs() {
		// run the output target to write outputs to files
		err := installerImpl.Output(installableObj, stackObj, true, dryRun)
		if err != nil {
			errCh <- errors.Wrapf(err, "Error writing output for kapp '%s'", installableObj.Id())
		}

		// load and parse outputs
		outputs, err = installableObj.GetOutputs(dryRun)
		if err != nil {
			errCh <- errors.Wrapf(err, "Error loading the output of kapp '%s'", installableObj.Id())
		}
	}

	// build the kapp's local registry
	buildLocalRegistry(dagObj, node, outputs, errCh)

	// rerender templates so they can use kapp outputs (e.g. before adding the paths to rendered templates as provider vars)
	err := renderKappTemplates(stackObj, installableObj, dryRun)
	if err != nil {
		errCh <- errors.WithStack(err)
	}

	// execute any post actions if we've just actually installed the kapp.
	if node.shouldProcess && apply && len(installableObj.PostActions()) > 0 {
		for _, postAction := range installableObj.PostActions() {
			executePostAction(postAction, installableObj, stackObj, errCh, dryRun)
		}
	}
}

// Instantiates a new registry local to the installable and populates it with the result of merging
// each local registry of each parent. If the parent's manifest ID is different to the current node's
// manifest ID registry keys for non fully-qualified installable IDs will be deleted from the registry
// before merging. In all cases the special value 'this' will not be merged either.
func buildLocalRegistry(dagObj *dag, node NamedNode, outputs map[string]interface{}, errCh chan<- error) {
	localRegistry := registry.New()

	parents := dagObj.graph.To(node.ID())

	for parents.Next() {
		parent := parents.Node().(NamedNode)

		parentRegistry := parent.installableObj.GetLocalRegistry()

		// if the parent was in a different manifest, strip out all non fully-qualified registry
		// entries
		if parent.installableObj.ManifestId() != node.installableObj.ManifestId() {
			deleteNonFullyQualifiedOutputs(parentRegistry)
		}

		// always delete the special key 'this'
		deleteSpecialThisOutput(parentRegistry)

		for k, v := range parentRegistry.AsMap() {
			err := localRegistry.Set(k, v)
			if err != nil {
				errCh <- errors.WithStack(err)
			}
		}
	}

	// only add outputs if any were passed in
	if outputs != nil && len(outputs) > 0 {
		err := addOutputsToRegistry(node.installableObj, outputs, localRegistry)
		if err != nil {
			errCh <- errors.WithStack(err)
		}
	}

	node.installableObj.SetLocalRegistry(localRegistry)
}

// Executes post actions
func executePostAction(postAction structs.PostAction, installableObj interfaces.IInstallable,
	stackObj interfaces.IStack, errCh chan error, dryRun bool) {
	switch postAction.Id {
	case constants.PostActionClusterUpdate:
		log.Logger.Info("Running cluster update action")
		err := cluster.UpdateCluster(os.Stdout, stackObj, true, dryRun)
		if err != nil {
			errCh <- errors.Wrapf(err, "Error updating cluster, triggered by kapp '%s'",
				installableObj.Id())
		}
		break
	case constants.PostActionAddProviderVarsFiles:
		log.Logger.Infof("Running action to add provider vars dirs")
		// todo - run each path through the templater
		for _, path := range postAction.Params {
			if !filepath.IsAbs(path) {
				// convert the relative path to absolute
				path = filepath.Join(installableObj.GetConfigFileDir(), path)
			}

			log.Logger.Debugf("Adding provider vars dir: %s", path)
			stackObj.GetProvider().AddVarsPath(path)
		}

		// refresh the provider vars so the extra vars files we've just added are loaded
		err := stackObj.RefreshProviderVars()
		if err != nil {
			errCh <- errors.WithStack(err)
		}
		break
	}
}

// Deletes all outputs from the registry that aren't fully qualified
func deleteNonFullyQualifiedOutputs(registry interfaces.IRegistry) {
	outputs, ok := registry.Get(constants.RegistryKeyOutputs)
	if !ok {
		return
	}

	// iterate through all the keys for those that aren't fully qualified and delete them
	for k, _ := range outputs.(map[string]interface{}) {
		if !strings.Contains(k, constants.NamespaceSeparator) {
			fullKey := strings.Join([]string{
				constants.RegistryKeyOutputs, k}, constants.RegistryFieldSeparator)
			registry.Delete(fullKey)
		}
	}
}

// deletes the special constant key "this" from the registry
func deleteSpecialThisOutput(registry interfaces.IRegistry) {
	registry.Delete(strings.Join([]string{constants.RegistryKeyOutputs,
		constants.RegistryKeyThis}, constants.RegistryFieldSeparator))
}

// Adds output from an installable to the registry
func addOutputsToRegistry(installableObj interfaces.IInstallable, outputs map[string]interface{},
	registry interfaces.IRegistry) error {

	// We convert kapp IDs to have underscores because Go's templating library throws its toys out
	// the pram when it find a map key with a hyphen in. K8s is the opposite, so this seems like
	// the least worst way of accommodating both
	underscoredInstallableId := strings.Replace(installableObj.Id(), "-", "_", -1)
	underscoredInstallableFQId := strings.Replace(installableObj.FullyQualifiedId(), "-", "_", -1)

	prefixes := []string{
		// "outputs.this"
		strings.Join([]string{constants.RegistryKeyOutputs, constants.RegistryKeyThis}, constants.RegistryFieldSeparator),
		// short prefix - can be used by other kapps in the manifest
		strings.Join([]string{constants.RegistryKeyOutputs, underscoredInstallableId},
			constants.RegistryFieldSeparator),
		// fully-qualified prefix - can be used by kapps in other manifests
		strings.Join([]string{constants.RegistryKeyOutputs,
			underscoredInstallableFQId}, constants.RegistryFieldSeparator),
	}

	// store the output under various keys
	for outputId, output := range outputs {
		for _, prefix := range prefixes {
			key := strings.Join([]string{prefix, outputId}, constants.RegistryFieldSeparator)
			err := registry.Set(key, output)
			if err != nil {
				return errors.WithStack(err)
			}
		}
	}

	return nil
}

// Renders templates for a kapp
func renderKappTemplates(stackObj interfaces.IStack, installableObj interfaces.IInstallable,
	dryRun bool) error {

	// todo - this should take a copy of the registry so we don't mutate the global one.
	//  Remember to use the the global registry as the base fragment

	templatedVars, err := stackObj.GetTemplatedVars(installableObj, nil)
	if err != nil {
		return errors.WithStack(err)
	}

	_, err = installableObj.RenderTemplates(templatedVars, stackObj.GetConfig(), dryRun)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}