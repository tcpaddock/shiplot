//go:build windows

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
package service

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/tcpaddock/shiplot/internal/util"
	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc/mgr"
)

var (
	exePath, installPath string
)

const sampleConfig = `# Maximum number of concurrent file transfers.
maxThreads: 4

# Server section contains settings for a TCP server to support network transfers.
server:
  # Whether to run a server. Default is false.
  enabled: false
  # The IP the server should bind to. Default is 0.0.0.0.
  ip: "0.0.0.0"
  # The port the server should bind to. Default is 9080.
  port: 9080

# Client section contains settings for a TCP client to support network transfers.
# The client move all plot files to the server and will discard any destination paths.
client:
  # Whether to enable moving plots over the network. Default is false.
  enabled: false
  # The IP of the server. Default is 0.0.0.0.
  serverIp: "0.0.0.0"
  # The port of the server. Default is 9080.
  serverport: 9080


# The paths that are watched for new *.plot files.
stagingPaths:
  # - "C:/staging"

# List of paths to move *.plot files to.
destinationPaths:
  # - "C:/destination"
`

func install() (err error) {
	// Get current executable path
	if exePath, err = os.Executable(); err != nil {
		return err
	}

	// Get install path
	if installPath, err = getInstallPath(); err != nil {
		return err
	}

	// Create install folder if not exists
	if err = createInstallFolder(); err != nil {
		return err
	}

	// Copy executable if not exists
	if err = copyExe(); err != nil {
		return err
	}

	// Create sample config if not exists
	if err = createConfig(); err != nil {
		return err
	}

	// Create service
	return createService()
}

func getInstallPath() (name string, err error) {
	pd := os.Getenv("ProgramData")
	if pd == "" {
		return "", fmt.Errorf("failed to find ProgramData path")
	}

	return filepath.Join(pd, "shiplot"), nil
}

func createInstallFolder() (err error) {
	if _, err := os.Stat(installPath); errors.Is(err, fs.ErrNotExist) {
		if err = os.Mkdir(installPath, 0750); err != nil {
			return err
		}
	}

	return nil
}

func copyExe() (err error) {
	if _, err := os.Stat(filepath.Join(installPath, "shiplot.exe")); errors.Is(err, fs.ErrNotExist) {
		if _, err = util.CopyFile(exePath, filepath.Join(installPath, "shiplot.exe")); err != nil {
			return err
		}
	}

	return nil
}

func createConfig() (err error) {
	if _, err := os.Stat(filepath.Join(installPath, "shiplot.yaml")); errors.Is(err, fs.ErrNotExist) {
		f, err := os.Create(filepath.Join(installPath, "shiplot.yaml"))
		if err != nil {
			_ = f.Close()
			return err
		}
		defer f.Close()

		_, err = f.WriteString(sampleConfig)
		if err != nil {
			return err
		}
	}

	return nil
}

func createService() (err error) {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	// Check if service exists
	s, err := m.OpenService(serviceName)
	if err == nil {
		s.Close()
		return fmt.Errorf("service %s already exists", serviceName)
	}

	s, err = m.CreateService(serviceName, exePath, mgr.Config{DisplayName: serviceName, Description: "Chia plot shipper"}, "run")
	if err != nil {
		return err
	}
	defer s.Close()

	err = eventlog.InstallAsEventCreate(serviceName, eventlog.Error|eventlog.Warning|eventlog.Info)
	if err != nil {
		s.Delete()
		return err
	}

	return nil
}

func uninstall() (err error) {
	// Get current executable path
	if exePath, err = os.Executable(); err != nil {
		return err
	}

	// Get install path
	if installPath, err = getInstallPath(); err != nil {
		return err
	}

	// Remove service
	if err = removeService(); err != nil {
		return err
	}

	// Delete install directory
	if err = os.RemoveAll(installPath); err != nil {
		return err
	}

	return nil
}

func removeService() (err error) {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(serviceName)
	if err != nil {
		return fmt.Errorf("service %s is not installed", serviceName)
	}
	defer s.Close()

	err = s.Delete()
	if err != nil {
		return err
	}

	err = eventlog.Remove(serviceName)
	if err != nil {
		return err
	}

	return nil
}

func enable() (err error) {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(serviceName)
	if err != nil {
		return fmt.Errorf("service %s is not installed", serviceName)
	}
	defer s.Close()

	c, err := s.Config()
	if err != nil {
		return err
	}

	c.StartType = 2

	return s.UpdateConfig(c)
}

func disable() (err error) {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(serviceName)
	if err != nil {
		return fmt.Errorf("service %s is not installed", serviceName)
	}
	defer s.Close()

	c, err := s.Config()
	if err != nil {
		return err
	}

	c.StartType = 3

	return s.UpdateConfig(c)
}
