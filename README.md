# shiplot

[![codecov](https://codecov.io/gh/tcpaddock/shiplot/branch/main/graph/badge.svg?token=N52TPZ8AWX)](https://codecov.io/gh/tcpaddock/shiplot)

Chia plot file shipper.

## Current State

This tool is currently in early development. Test heavily before using.

## Install

- Download the binary from [releases](https://github.com/tcpaddock/shiplot/releases).
- Create a config file in same directory as binary. (See [example](example.shiplot.yaml))

## Usage

The tool currently has two primary modes. It will either move plots locally or ship them across the network.

View the help text by running:
```bash
# Linux/macOS
shiplot help

# Windows
shiplot.exe help
```

View the version by running:
```bash
# Linux/macOS
shiplot version

# Windows
shiplot.exe version
```

### Local 

Start moving plots:
```bash
# Linux/macOS
shiplot run --maxthreads=12 --stagingPaths="/staging/*" --destinationPaths="/mnt/dest,/mnt/jbod*"

# Windows
shiplot.exe run --maxthreads=12 --stagingPaths="C:/staging/*" --destinationPaths="D:/,E:/"
```

### Network

In network mode, the destinationPaths parameter is ignored.

Start server on destination:
```bash
# Linux/macOS
shiplot run --maxthreads=12 --destinationPaths="/mnt/dest,/mnt/jbod*" --server.enabled=true

# Windows
shiplot.exe run --maxthreads=12 --destinationPaths="D:/,E:/" --server.enabled=true
```

Start client on plotter:
```bash
# Linux/macOS
shiplot run --maxthreads=12 --stagingPaths="/staging/*" --client.enabled=true --client.serverIp="192.168.0.2"

# Windows
shiplot.exe run --maxthreads=12 --stagingPaths="/staging/*" --client.enabled=true --client.serverIp="192.168.0.2"
```

## License

Licensed under the [MIT license](LICENSE).

Copyright © 2023 Taylor Paddock