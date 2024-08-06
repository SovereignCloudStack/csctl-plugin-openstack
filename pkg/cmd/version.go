/*
Copyright 2024 The Kubernetes Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	// Version of csctl-openstack.
	Version = "dev"
	// Commit against which csctl-openstack version is cut.
	Commit = "unknown"
)

var versionCmd = &cobra.Command{
	Use:          "version",
	Short:        "prints the latest version of csctl-openstack",
	Run:          printVersion,
	SilenceUsage: true,
}

func printVersion(_ *cobra.Command, _ []string) {
	fmt.Println("csctl-openstack version:", Version)
	fmt.Println("commit:", Commit)
}
