# Go Migration Guide (Internal)

> **Audience**: Test Genie developers migrating from legacy bash to Go

## Overview

This document tracks the migration from bash-based testing (`scripts/scenarios/testing/`) to Go-native orchestration within test-genie.

## Migration Status

### Completed Phases

| Phase | Bash Script | Go Implementation | Status |
|-------|-------------|-------------------|--------|
| Structure | `test-structure.sh` | `phases/structure.go` | ✅ Complete |
| Dependencies | `test-dependencies.sh` | `phases/dependencies.go` | ✅ Complete |
| Unit | `test-unit.sh` | `phases/unit.go` | ✅ Complete |
| Integration | `test-integration.sh` | `phases/integration.go` | ✅ Complete |
| Business | `test-business.sh` | `phases/business.go` | ✅ Complete |
| Performance | `test-performance.sh` | `phases/performance.go` | ✅ Complete |

### Pending Components

| Component | Legacy Location | Go Location | Status |
|-----------|-----------------|-------------|--------|
| Requirements Sync | `scripts/requirements/` | `orchestrator/requirements/` | 🔄 In Progress |
| Phase Helpers | `shell/phase-helpers.sh` | `orchestrator/phases/` | ✅ Replaced |
| Test Runner | `unit/run-all.sh` | `phases/unit.go` | ✅ Replaced |

## Migration Approach

### 1. Interface-First

Define Go interfaces that match bash script behavior:

```go
// Before: bash function
// testing::phase::run_structure() { ... }

// After: Go interface
type PhaseRunner interface {
    Run(ctx context.Context, workspace Workspace) (*Result, error)
}
```

### 2. Parallel Operation

Run both systems during transition:

```go
func (o *Orchestrator) RunPhase(phase string, scenario string) (*Result, error) {
    if goPhase := o.registry.Get(phase); goPhase != nil {
        return goPhase.Run(ctx, workspace)
    }
    // Fallback to bash
    return o.runBashPhase(phase, scenario)
}
```

### 3. Feature Flags

Control migration with environment variables:

```bash
# Force Go implementation
export TEST_GENIE_USE_GO_PHASES=true

# Fall back to bash for specific phases
export TEST_GENIE_BASH_PHASES=integration,performance
```

## Key Differences

### Exit Code Handling

```bash
# Bash: Exit codes set directly
exit 0  # success
exit 1  # failure
exit 2  # skipped
```

```go
// Go: Return typed results
return &Result{
    Status: StatusPassed,
    Phase:  "unit",
    Output: output,
}, nil
```

### Logging

```bash
# Bash: Echo to stdout/stderr
echo "✅ Test passed"
log::error "Test failed"
```

```go
// Go: Structured logging
logger.Info("test passed", "phase", "unit", "tests", 42)
logger.Error("test failed", "error", err)
```

### Configuration

```bash
# Bash: Source config files
source "$APP_ROOT/scripts/scenarios/testing/config.sh"
```

```go
// Go: Parse .vrooli/testing.json
config, err := testing.LoadConfig(scenarioDir)
```

## Testing the Migration

### Unit Tests

Each Go phase has corresponding tests:

```go
// phases/structure_test.go
func TestStructurePhase(t *testing.T) {
    workspace := NewMockWorkspace(t)
    phase := NewStructurePhase()

    result, err := phase.Run(context.Background(), workspace)

    assert.NoError(t, err)
    assert.Equal(t, StatusPassed, result.Status)
}
```

### Integration Tests

Compare Go and bash output:

```bash
# Run both implementations
bash_result=$(./test/phases/test-structure.sh 2>&1)
go_result=$(go run ./cmd/phase-runner structure 2>&1)

# Compare outputs
diff <(echo "$bash_result") <(echo "$go_result")
```

## Rollback Procedure

If Go implementation causes issues:

1. Set environment variable: `export TEST_GENIE_USE_GO_PHASES=false`
2. Restart test-genie: `make restart`
3. File issue with reproduction steps

## Deprecation Timeline

| Milestone | Date | Action |
|-----------|------|--------|
| Go phases complete | 2024-12-01 | All phases in Go |
| Parallel operation | 2024-12-15 | Both systems running |
| Bash deprecated | 2025-01-01 | Bash scripts marked deprecated |
| Bash removed | 2025-02-01 | Legacy scripts deleted |

## See Also

- [Architecture](../concepts/architecture.md) - Go architecture overview
- [Requirements Sync Plan](requirements-sync-plan.md) - Requirements migration
- [PROGRESS.md](../PROGRESS.md) - Implementation progress
