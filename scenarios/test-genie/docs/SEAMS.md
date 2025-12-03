# Test Genie Seams

This document describes the key architectural boundaries (seams) in Test Genie that enable testing, extensibility, and clean separation of concerns.

## Suite Orchestrator

| Seam | Purpose | Implementation |
|------|---------|----------------|
| `SuiteOrchestrator` | Central test execution engine that runs phases sequentially | `internal/orchestrator/orchestrator.go` - Coordinates phase execution, collects results, triggers requirements sync |
| `PhaseRunner` interface | Pluggable phase implementations | `internal/orchestrator/phases/` - Each phase (structure, dependencies, unit, integration, e2e, business, performance) implements the same interface |
| `PhaseRegistry` | Declarative phase catalog with metadata | `internal/orchestrator/phases/catalog.go` - Defines ordering, timeouts, and phase categories |
| `Workspace` | Scenario filesystem abstraction | `internal/orchestrator/workspace/` - Resolves paths, validates structure, loads configs |

## Requirements System

| Seam | Purpose | Implementation |
|------|---------|----------------|
| `requirements.Service` | Unified requirements operations | `internal/requirements/service.go` - Sync, report, validate operations |
| `discovery.Scanner` | Requirement file enumeration | `internal/requirements/discovery/` - Finds and resolves requirement JSON files |
| `parsing.Parser` | JSON parsing and normalization | `internal/requirements/parsing/` - Loads and normalizes requirement structures |
| `evidence.Loader` | Test evidence aggregation | `internal/requirements/evidence/` - Loads phase results, vitest output, manual validations |
| `enrichment.Enricher` | Live status computation | `internal/requirements/enrichment/` - Merges evidence with requirements |
| `sync.Syncer` | File synchronization | `internal/requirements/sync/` - Updates requirement files with live status |
| `reporting.Reporter` | Output generation | `internal/requirements/reporting/` - JSON, Markdown, trace formats |

## API Runtime & HTTP Surface

| Seam | Purpose | Implementation |
|------|---------|----------------|
| `runtime.LoadConfig` | Environment/config parsing | `internal/app/config.go` - Isolated config resolution, testable without env vars |
| `runtime.BuildDependencies` | Dependency injection bootstrap | `internal/app/dependencies.go` - Wires DB, orchestrator, services |
| `httpserver.Dependencies` | HTTP handler dependencies | `internal/app/httpserver/` - Injects services via interfaces for testability |
| `suite.ExecutionHistory` | Execution record abstraction | `internal/suite/` - Interface-based access to execution data |

## Phase Runners

Each phase implements the `PhaseRunner` interface:

```go
type PhaseRunner interface {
    Name() string
    Run(ctx context.Context, workspace *Workspace, opts *RunOptions) (*PhaseResult, error)
}
```

| Phase | File | Purpose |
|-------|------|---------|
| Structure | `phase_structure.go` | File/config validation |
| Dependencies | `phase_dependencies.go` | Runtime/tool availability |
| Unit | `phase_unit.go` | Go, Vitest, Python unit tests |
| Integration | `phase_integration.go` | API/CLI connectivity |
| E2E | `phase_playbooks.go` | BAS browser automation |
| Business | `phase_business.go` | Requirements coverage |
| Performance | `phase_performance.go` | Benchmarks |

## Command Execution

| Seam | Purpose | Implementation |
|------|---------|----------------|
| `CommandLookup` | OS command resolution | Allows stubbing command availability in tests |
| `Executor` | Process execution | Abstracts shell-out operations for testability |

## Extension Points

### Adding a New Phase

1. Implement `PhaseRunner` interface in `internal/orchestrator/phases/`
2. Register in `catalog.go` with ordering weight
3. Add timeout/config support in `.vrooli/testing.json` schema

### Adding Evidence Sources

1. Implement loader in `internal/requirements/evidence/`
2. Register with `evidence.Loader` aggregator
3. Add matcher rules in `enrichment/matcher.go`

### Custom Reporting

1. Implement renderer in `internal/requirements/reporting/`
2. Add format option to `Reporter.Report()` method

## Testing Strategy

All seams are designed for testability:

- **Interfaces**: Core components use interfaces for easy mocking
- **Dependency injection**: No global state, all dependencies passed explicitly
- **Command stubs**: OS interactions abstracted for deterministic tests
- **Fixture support**: Config/workspace can be loaded from test fixtures

## See Also

- [Architecture](concepts/architecture.md) - System design overview
- [Phase Catalog](reference/phase-catalog.md) - Phase definitions
- [API Endpoints](reference/api-endpoints.md) - REST API reference
