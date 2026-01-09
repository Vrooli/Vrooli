# deps - Dependency Resolution

The `deps` package provides dependency resolution utilities, primarily topological sorting for service startup order.

## Overview

Services often depend on other services (e.g., an API depends on a database). This package implements dependency resolution algorithms to determine safe startup and shutdown ordering.

## Key Functions

| Function | Purpose |
|----------|---------|
| `TopoSort(services)` | Sort services in dependency order |
| `FindService(services, id)` | Look up a service by ID |

## Topological Sort

Returns services ordered so dependencies start before dependents:

```go
services := []manifest.Service{
    {ID: "api", Dependencies: []string{"db", "cache"}},
    {ID: "db"},
    {ID: "cache"},
    {ID: "worker", Dependencies: []string{"api"}},
}

order, err := deps.TopoSort(services)
// order = ["db", "cache", "api", "worker"]
```

## Cycle Detection

Circular dependencies are detected and reported:

```go
services := []manifest.Service{
    {ID: "a", Dependencies: []string{"b"}},
    {ID: "b", Dependencies: []string{"a"}},
}

_, err := deps.TopoSort(services)
// err: "cycle detected in dependencies"
```

## Algorithm

Uses Kahn's algorithm:
1. Build adjacency list and in-degree counts
2. Start with nodes having zero in-degree
3. Process each node, decrementing dependents' in-degree
4. Repeat until all nodes processed or cycle detected

Time complexity: O(V + E) where V = services, E = dependencies.

## Usage in Runtime

```go
// Startup: launch in dependency order
order, err := deps.TopoSort(manifest.Services)
for _, id := range order {
    svc := deps.FindService(manifest.Services, id)
    if err := supervisor.startService(ctx, *svc); err != nil {
        // handle error
    }
}

// Shutdown: reverse order
for i := len(order) - 1; i >= 0; i-- {
    supervisor.stopService(ctx, order[i])
}
```

## Error

| Error | Description |
|-------|-------------|
| `ErrCyclicDependency` | Circular dependency detected |

## Dependencies

- **Depends on**: `manifest`
- **Depended on by**: `bundleruntime`
