/*
Copyright © 2023 Taylor Paddock

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	version     = "dev"
	buildTarget = "unknown"
	buildDate   = "unknown"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version number",
	Long: `
Show the shiplot version number, build target platform,
build date and time, and runtime OS type and architecture.

For example:

$ shiplot version
shiplot v1.0.0
- build target: linux_amd64
- build date: 2006-01-02T15:04:05Z
- os type: linux
- os arch: amd64
	
`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("shiplot %s\n", version)
		fmt.Printf("- build target: %s\n", buildTarget)
		fmt.Printf("- build date: %s\n", buildDate)
		fmt.Printf("- os type: %s\n", runtime.GOOS)
		fmt.Printf("- os type: %s\n", runtime.GOARCH)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
