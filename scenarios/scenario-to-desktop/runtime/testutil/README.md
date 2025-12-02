# testutil - Test Utilities and Mocks

The `testutil` package provides reusable mock implementations and test helpers for the runtime package.

## Overview

Testing the runtime requires mocking infrastructure dependencies (filesystem, network, time, etc.). This package provides consistent mock implementations that can be shared across all test files.

## Mock Implementations

| Mock | Interface | Key Features |
|------|-----------|--------------|
| `MockClock` | `infra.Clock` | Controllable time, `Advance()` method |
| `MockTicker` | `infra.Ticker` | Manual tick triggering |
| `MockFileSystem` | `infra.FileSystem` | In-memory files, error injection |
| `MockDialer` | `infra.NetworkDialer` | Configurable connection results |
| `MockProcess` | `infra.Process` | Configurable wait/signal behavior |
| `MockProcessRunner` | `infra.ProcessRunner` | Track started processes |
| `MockCommandRunner` | `infra.CommandRunner` | Scripted command outputs |
| `MockPortAllocator` | `ports.Allocator` | Pre-configured port mapping |

## Usage

### MockClock

```go
clock := testutil.NewMockClock(time.Now())

// Get current time
now := clock.Now()

// Advance time
clock.Advance(5 * time.Second)

// Create ticker (use clock.Tick() to trigger)
ticker := clock.NewTicker(time.Second)
```

### MockFileSystem

```go
fs := testutil.NewMockFileSystem()

// Pre-populate files
fs.Files["config.json"] = []byte(`{"key": "value"}`)

// Set up directories
fs.Dirs["/app/data"] = true

// Inject errors
fs.ReadErr = os.ErrNotExist

// Check what was written
data := fs.Files["output.txt"]
```

### MockDialer

```go
dialer := testutil.NewMockDialer()

// Port is available (connection refused)
dialer.Ports[8080] = false

// Port is in use (connection succeeds)
dialer.Ports[3000] = true
```

### MockCommandRunner

```go
runner := testutil.NewMockCommandRunner()

// Set up outputs for specific commands
runner.SetOutputs(map[string]string{
    "nvidia-smi": "GPU 0: NVIDIA GeForce RTX 3080",
})

// Inject error
runner.SetShouldErr(true)

// Check what commands were run
commands := runner.Commands()
```

### MockPortAllocator

```go
alloc := testutil.NewMockPortAllocator()

// Pre-configure ports
alloc.Ports["api"] = map[string]int{"http": 8080}
alloc.Ports["db"] = map[string]int{"postgres": 5432}
```

## Creating Tests

```go
func TestSomething(t *testing.T) {
    // Create mocks
    clock := testutil.NewMockClock(time.Now())
    fs := testutil.NewMockFileSystem()
    dialer := testutil.NewMockDialer()

    // Pre-populate state
    fs.Files["secrets.json"] = []byte(`{"API_KEY": "test"}`)

    // Create supervisor with mocks
    supervisor, err := bundleruntime.NewSupervisor(bundleruntime.Options{
        Manifest:      manifest,
        BundlePath:    "/bundle",
        Clock:         clock,
        FileSystem:    fs,
        NetworkDialer: dialer,
    })

    // Test behavior
    // ...

    // Verify interactions
    if _, ok := fs.Files["output.txt"]; !ok {
        t.Error("expected output.txt to be written")
    }
}
```

## Thread Safety

Mock implementations are NOT thread-safe by default. For concurrent tests, use appropriate synchronization or create separate mocks per goroutine.

## Dependencies

- **Depends on**: `infra`, `ports`, `manifest`
- **Depended on by**: Test files in `bundleruntime`
