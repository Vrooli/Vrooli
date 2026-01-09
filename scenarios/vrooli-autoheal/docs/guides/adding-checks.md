# Adding Health Checks

How to extend Autoheal with custom health checks.

## Check Interface

Every health check implements this interface:

```go
type Check interface {
    ID() string                         // Unique identifier
    Description() string                // Human-readable description
    IntervalSeconds() int               // How often to run
    Platforms() []platform.Type         // Which platforms (nil = all)
    Run(ctx context.Context) Result     // Execute the check
}
```

## Creating a New Check

### Step 1: Create the Check File

Create a new file in `api/internal/checks/infra/` or `api/internal/checks/vrooli/`:

```go
// api/internal/checks/infra/memory.go
package infra

import (
    "context"
    "fmt"
    "runtime"

    "vrooli-autoheal/internal/checks"
    "vrooli-autoheal/internal/platform"
)

// MemoryCheck monitors system memory usage
type MemoryCheck struct {
    warningThreshold  float64
    criticalThreshold float64
}

// NewMemoryCheck creates a memory usage check
func NewMemoryCheck(warningPct, criticalPct float64) *MemoryCheck {
    return &MemoryCheck{
        warningThreshold:  warningPct,
        criticalThreshold: criticalPct,
    }
}

func (c *MemoryCheck) ID() string {
    return "system-memory"
}

func (c *MemoryCheck) Description() string {
    return "Monitor system memory usage"
}

func (c *MemoryCheck) IntervalSeconds() int {
    return 60 // Check every minute
}

func (c *MemoryCheck) Platforms() []platform.Type {
    return nil // Run on all platforms
}

func (c *MemoryCheck) Run(ctx context.Context) checks.Result {
    var m runtime.MemStats
    runtime.ReadMemStats(&m)

    usedPct := float64(m.Alloc) / float64(m.Sys) * 100

    if usedPct >= c.criticalThreshold {
        return checks.Result{
            CheckID: c.ID(),
            Status:  checks.StatusCritical,
            Message: fmt.Sprintf("Memory usage critical: %.1f%%", usedPct),
            Details: map[string]interface{}{
                "used_pct": usedPct,
                "alloc":    m.Alloc,
                "sys":      m.Sys,
            },
        }
    }

    if usedPct >= c.warningThreshold {
        return checks.Result{
            CheckID: c.ID(),
            Status:  checks.StatusWarning,
            Message: fmt.Sprintf("Memory usage elevated: %.1f%%", usedPct),
        }
    }

    return checks.Result{
        CheckID: c.ID(),
        Status:  checks.StatusOK,
        Message: fmt.Sprintf("Memory usage normal: %.1f%%", usedPct),
    }
}
```

### Step 2: Register the Check

Add registration in `api/internal/bootstrap/checks.go`:

```go
func RegisterDefaultChecks(registry *checks.Registry, caps *platform.Capabilities) {
    // Existing checks...

    // Add your new check
    registry.Register(infra.NewMemoryCheck(70.0, 90.0))
}
```

### Step 3: Add Tests

Create tests in `api/internal/checks/infra/memory_test.go`:

```go
package infra

import (
    "context"
    "testing"

    "vrooli-autoheal/internal/checks"
)

func TestMemoryCheck_ID(t *testing.T) {
    c := NewMemoryCheck(70, 90)
    if c.ID() != "system-memory" {
        t.Errorf("expected system-memory, got %s", c.ID())
    }
}

func TestMemoryCheck_Run(t *testing.T) {
    c := NewMemoryCheck(70, 90)
    result := c.Run(context.Background())

    if result.CheckID != "system-memory" {
        t.Errorf("expected system-memory, got %s", result.CheckID)
    }

    // Memory check should always return a result
    if result.Status != checks.StatusOK &&
       result.Status != checks.StatusWarning &&
       result.Status != checks.StatusCritical {
        t.Errorf("unexpected status: %v", result.Status)
    }
}
```

### Step 4: Rebuild and Test

```bash
# Rebuild the API
cd scenarios/vrooli-autoheal
make build

# Restart the scenario
make restart

# Verify the check appears
curl http://localhost:PORT/api/v1/checks | jq '.[] | select(.id == "system-memory")'
```

## Platform-Specific Checks

If your check only works on certain platforms, specify them:

```go
func (c *SystemdCheck) Platforms() []platform.Type {
    return []platform.Type{platform.Linux}
}
```

For checks that depend on capabilities:

```go
type DockerCheck struct {
    caps *platform.Capabilities
}

func NewDockerCheck(caps *platform.Capabilities) *DockerCheck {
    return &DockerCheck{caps: caps}
}

func (c *DockerCheck) Platforms() []platform.Type {
    if c.caps.HasDocker {
        return nil // Run on all platforms if Docker available
    }
    return []platform.Type{} // Empty slice = skip on all platforms
}
```

## Check Best Practices

### 1. Use Timeouts

Always use the context for timeouts:

```go
func (c *NetworkCheck) Run(ctx context.Context) checks.Result {
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    // Use ctx in operations...
}
```

### 2. Return Helpful Messages

Messages should help users understand and fix issues:

```go
// Good
Message: "PostgreSQL connection refused on localhost:5432 - is the service running?"

// Bad
Message: "connection refused"
```

### 3. Include Details

Add structured details for debugging:

```go
Details: map[string]interface{}{
    "host":       host,
    "port":       port,
    "error":      err.Error(),
    "suggestion": "Check firewall rules",
}
```

### 4. Handle Errors Gracefully

Never panic; always return a Result:

```go
func (c *MyCheck) Run(ctx context.Context) checks.Result {
    data, err := c.fetchData()
    if err != nil {
        return checks.Result{
            CheckID: c.ID(),
            Status:  checks.StatusCritical,
            Message: fmt.Sprintf("Failed to fetch data: %v", err),
        }
    }
    // Continue with healthy path...
}
```

### 5. Choose Appropriate Intervals

| Check Type | Recommended Interval |
|------------|---------------------|
| Network | 30-60 seconds |
| Services | 60 seconds |
| Resources | 60-120 seconds |
| Disk/System | 300 seconds (5 min) |
| Certificates | 3600 seconds (1 hour) |

## Check Categories

Organize checks by purpose:

| Directory | Purpose | Examples |
|-----------|---------|----------|
| `checks/infra/` | Infrastructure health | network, dns, docker |
| `checks/vrooli/` | Vrooli resources/scenarios | postgres, redis, scenarios |
| `checks/system/` | OS-level health | disk, memory, processes |
