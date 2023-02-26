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
	"context"

	"github.com/spf13/cobra"
	"github.com/tcpaddock/shiplot/internal/server"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Starts watching staging directories.",
	Long: `
Run starts a server thread that begins watching the source
directories. The file watcher dynamically sizes itself
based on the lesser value between --maxThreads and the
number of paths in --destinationPaths. The file watcher
will automatically remove destination paths that are full.
The destination path with the most free space will be
preferred.

`,
	Run: func(cmd *cobra.Command, args []string) {
		s, err := server.NewServer(cfg)
		cobra.CheckErr(err)

		ctx := context.Background()
		err = s.Start(ctx)
		cobra.CheckErr(err)
	},
}

func init() {
	RootCmd.AddCommand(runCmd)

	runCmd.PersistentFlags().UintVar(&cfg.MaxThreads, "maxThreads", 4, "Number of concurrent file transfers (default is 4)")
	runCmd.PersistentFlags().BoolVar(&cfg.Server.Enabled, "server.enabled", false, "Enable TCP server (default is false)")
	runCmd.PersistentFlags().StringVar(&cfg.Server.Ip, "server.ip", "0.0.0.0", "Server listen IP (default is 0.0.0.0)")
	runCmd.PersistentFlags().Uint16Var(&cfg.Server.Port, "server.port", 9080, "Server listen port (default is 9080)")
	runCmd.PersistentFlags().BoolVar(&cfg.Client.Enabled, "client.enabled", false, "Enable TCP client (default is false)")
	runCmd.PersistentFlags().StringVar(&cfg.Client.ServerIp, "client.serverIp", "0.0.0.0", "IP of TCP server (default is 0.0.0.0)")
	runCmd.PersistentFlags().Uint16Var(&cfg.Client.ServerPort, "client.serverPort", 9080, "Server listen port (default is 9080)")
	runCmd.PersistentFlags().StringArrayVar(&cfg.StagingPaths, "stagingPaths", nil, "Directory on fast storage used to stage plots")
	runCmd.PersistentFlags().StringArrayVar(&cfg.DestinationPaths, "destinationPaths", nil, "Directories for final plot storage")
}
