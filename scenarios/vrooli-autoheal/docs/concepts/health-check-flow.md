# Health Check Flow

How health checks execute from request to result.

## Tick Execution Flow

```mermaid
flowchart TD
    A[Tick Request] --> B[Registry.RunAll]
    B --> C{For each check}
    C --> D{Platform compatible?}
    D -->|No| E[Skip - wrong platform]
    D -->|Yes| F{Interval elapsed?}
    F -->|No| G[Return cached result]
    F -->|Yes| H[Execute check.Run]
    H --> I{Check succeeded?}
    I -->|Yes| J[Store OK result]
    I -->|No| K[Store Warning/Critical]
    J --> L[Update last-run time]
    K --> L
    L --> M{More checks?}
    E --> M
    G --> M
    M -->|Yes| C
    M -->|No| N[Compute summary]
    N --> O[Return results]
```

## Platform Filtering

Checks specify which platforms they support:

```go
type Check interface {
    ID() string
    Description() string
    IntervalSeconds() int
    Platforms() []platform.Type  // nil = all platforms
    Run(ctx context.Context) Result
}
```

Example: RDP check only runs on platforms that support RDP:

```go
func (c *RDPCheck) Platforms() []platform.Type {
    if c.caps.SupportsRdp {
        return []platform.Type{platform.Linux, platform.Windows}
    }
    return nil // Skip on all platforms if no RDP support
}
```

## Interval Enforcement

Each check has an interval that controls how often it runs:

```mermaid
sequenceDiagram
    participant Client
    participant Registry
    participant Check
    participant Cache

    Client->>Registry: RunAll()
    Registry->>Cache: GetLastRun(checkId)
    Cache-->>Registry: lastRun: 30s ago

    alt interval (60s) not elapsed
        Registry-->>Client: Return cached result
    else interval elapsed
        Registry->>Check: Run()
        Check-->>Registry: Result{status: ok}
        Registry->>Cache: SetLastRun(checkId, now)
        Registry-->>Client: Return new result
    end
```

The `--force` flag bypasses interval checks:

```bash
# Normal: respects intervals
vrooli-autoheal tick

# Force: runs all checks regardless of interval
vrooli-autoheal tick --force
```

## Result Status Levels

```mermaid
graph LR
    subgraph "Status Hierarchy"
        OK["OK<br/>✅ Healthy"]
        Warning["Warning<br/>⚠️ Degraded"]
        Critical["Critical<br/>❌ Failed"]
    end

    OK --> Warning
    Warning --> Critical

    style OK fill:#10b981,color:#fff
    style Warning fill:#f59e0b,color:#fff
    style Critical fill:#ef4444,color:#fff
```

**Aggregation Rule**: Overall status is the worst status among all checks.

```go
func AggregateStatus(results []Result) Status {
    worst := StatusOK
    for _, r := range results {
        if r.Status > worst {
            worst = r.Status
        }
    }
    return worst
}
```

## Check Result Structure

Each check returns a Result:

```go
type Result struct {
    CheckID   string                 // "infra-network"
    Status    Status                 // OK, Warning, Critical
    Message   string                 // Human-readable description
    Details   map[string]interface{} // Optional structured data
    Timestamp time.Time              // When the check ran
    Duration  time.Duration          // How long it took
}
```

## Persistence Flow

```mermaid
sequenceDiagram
    participant Check
    participant Registry
    participant Store
    participant DB

    Check->>Registry: Return Result
    Registry->>Store: SaveResult(result)
    Store->>DB: INSERT INTO health_results
    DB-->>Store: OK

    Note over Store,DB: Results kept for 24 hours

    Store->>DB: DELETE WHERE timestamp < 24h ago
```

## Error Handling

Checks should never panic. Errors are reported as Critical status:

```go
func (c *NetworkCheck) Run(ctx context.Context) checks.Result {
    conn, err := net.DialTimeout("tcp", c.target, 5*time.Second)
    if err != nil {
        return checks.Result{
            CheckID: c.ID(),
            Status:  checks.StatusCritical,
            Message: fmt.Sprintf("Network unreachable: %v", err),
        }
    }
    conn.Close()
    return checks.Result{
        CheckID: c.ID(),
        Status:  checks.StatusOK,
        Message: "Network connectivity OK",
    }
}
```
