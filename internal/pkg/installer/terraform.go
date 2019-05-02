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

package installer

import (
	"fmt"
	"github.com/sugarkube/sugarkube/internal/pkg/constants"
	"github.com/sugarkube/sugarkube/internal/pkg/interfaces"
	"github.com/sugarkube/sugarkube/internal/pkg/log"
)

// Installs kapps with terraform
type TerraformInstaller struct {
	provider interfaces.IProvider
}

func (i TerraformInstaller) Install(installableObj interfaces.IInstallable, stack interfaces.IStack, approved bool, dryRun bool) error {
	log.Logger.Errorf("Not implemented - terraform install")
	return nil
}

func (i TerraformInstaller) Delete(installableObj interfaces.IInstallable, stack interfaces.IStack, approved bool, dryRun bool) error {
	log.Logger.Errorf("Not implemented - terraform delete")
	return nil
}

func (i TerraformInstaller) Clean(installableObj interfaces.IInstallable, stack interfaces.IStack, dryRun bool) error {
	log.Logger.Errorf("Not implemented - terraform clean")
	return nil
}

func (i TerraformInstaller) Output(installableObj interfaces.IInstallable, stack interfaces.IStack, dryRun bool) error {
	log.Logger.Errorf("Not implemented - terraform output")
	return nil
}

func (i TerraformInstaller) Name() string {
	return constants.InstallerTerraformInstaller
}

func (i TerraformInstaller) GetVars(action string, approved bool) map[string]interface{} {
	return map[string]interface{}{
		constants.InstallerAction:   action,
		constants.InstallerApproved: fmt.Sprintf("%v", approved)}
}
