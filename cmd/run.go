/*
Copyright © 2023 Taylor Paddock <tcpaddock@gmail.com>

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
	"context"

	"github.com/spf13/cobra"
	"github.com/tcpaddock/shiplot/internal/server"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Starts server",
	Long:  `Starts server`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		s, err := server.NewServer(ctx, cfg)
		cobra.CheckErr(err)

		err = s.Start()
		cobra.CheckErr(err)
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.PersistentFlags().UintVar(&cfg.MaxThreads, "maxThreads", 4, "Number of concurrent file transfers (default is 4)")
	runCmd.PersistentFlags().UintVar(&cfg.Port, "port", 9080, "Server listen port (default is 9080)")
	runCmd.PersistentFlags().StringVar(&cfg.StagingPath, "stagingPath", "", "Directory on fast storage used to stage plots")
	runCmd.PersistentFlags().StringArrayVar(&cfg.DestinationPaths, "destinationPaths", nil, "Directories for final plot storage")
}
