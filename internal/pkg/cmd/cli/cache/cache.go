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

package cache

import (
	"fmt"
	"github.com/spf13/cobra"
	"io"
)

func NewCacheCmds(out io.Writer) *cobra.Command {

	cmd := &cobra.Command{
		Use:   "cache [command]",
		Short: fmt.Sprintf("Work with kapp caches"),
		Long:  `Create and refresh kapp caches`,
	}

	cmd.AddCommand(
		newCreateCmd(out),
		newDiffCmd(out),
	)

	return cmd
}
