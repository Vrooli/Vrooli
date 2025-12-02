# infra - Infrastructure Abstractions

The `infra` package provides interfaces and implementations for operating system primitives, enabling dependency injection and testability throughout the runtime.

## Overview

This package defines the boundary between the runtime and the operating system. By abstracting time, filesystem, network, and process operations behind interfaces, the runtime can be thoroughly tested without actual system calls.

## Key Interfaces

| Interface | Purpose | Real Implementation |
|-----------|---------|---------------------|
| `Clock` | Time operations (`Now()`, `After()`, `NewTicker()`) | `RealClock` |
| `Ticker` | Periodic time events | `time.Ticker` wrapper |
| `FileSystem` | File operations (read, write, mkdir, stat) | `RealFileSystem` |
| `File` | Open file handle | `os.File` wrapper |
| `NetworkDialer` | TCP/UDP connections | `RealNetworkDialer` |
| `HTTPClient` | HTTP requests | `RealHTTPClient` |
| `ProcessRunner` | Start external processes | `RealProcessRunner` |
| `Process` | Running process handle | `realProcess` |
| `CommandRunner` | Execute commands with output capture | `RealCommandRunner` |
| `EnvReader` | Read environment variables | `RealEnvReader` |

## Usage

```go
// Production: use real implementations
clock := infra.RealClock{}
fs := infra.RealFileSystem{}
dialer := infra.RealNetworkDialer{}

// Testing: inject mocks
supervisor, _ := bundleruntime.NewSupervisor(bundleruntime.Options{
    Clock:      mockClock,
    FileSystem: mockFS,
    // ...
})
```

## Design Principles

1. **Thin Wrappers**: Real implementations are thin wrappers around stdlib
2. **Interface Segregation**: Each interface is focused and minimal
3. **No Business Logic**: This package has zero domain knowledge
4. **Cross-Platform**: Abstractions work on Linux, macOS, and Windows

## Files

| File | Contents |
|------|----------|
| `clock.go` | `Clock`, `Ticker`, `RealClock` |
| `filesystem.go` | `FileSystem`, `File`, `RealFileSystem` |
| `network.go` | `NetworkDialer`, `HTTPClient`, `RealNetworkDialer`, `RealHTTPClient` |
| `process.go` | `ProcessRunner`, `Process`, `CommandRunner`, `RealProcessRunner`, `RealCommandRunner` |
| `env.go` | `EnvReader`, `RealEnvReader` |
| `signals.go` | Cross-platform signal constants (`Interrupt`, `Kill`) |

## Testing

Mock implementations live in `runtime/testutil/mocks.go`. See `testutil/README.md` for usage.

## Dependencies

- **Depends on**: Standard library only (`os`, `time`, `net`, `os/exec`)
- **Depended on by**: All other runtime packages
