# Test Genie Seams

This document tracks the intentional seams that let Test Genie evolve without forcing broad edits across the codebase.

## Primary seams

### Queue staleness policy

- Package: `api/internal/queue`
- Surface: `ActiveQueueWindow()` plus repository-owned snapshot logic
- Why it exists: queue telemetry needs a policy boundary for "active" vs "stale" queued work. The HTTP layer and CLI should consume that decision, not invent their own filters.

### Scenario summary projection

- Package: `api/internal/scenarios`
- Surface: `ScenarioDirectoryRepository`
- Why it exists: scenario catalog views should summarize queue and execution state without coupling callers to raw SQL or raw queue rows. The repository now shares the same active-queue cutoff as queue telemetry.

### Playbooks registry normalization

- Package: `api/internal/playbooks/registry`
- Surface: `Builder.Build()` and `Loader.Load()`
- Why it exists: BAS workflow files are authoring inputs; `bas/registry.json` is the execution contract. Registry normalization is where legacy labels, execution-mode hints, ordering, fixtures, and requirements become one stable manifest.

### Playbooks execution mode

- Package: `api/internal/playbooks/types`
- Surface: `Registry.Metadata.ExecutionMode`, `Registry.UsesObserverMode()`
- Why it exists: observer-mode vs mutating playbooks is a configuration decision that should be expressed in data, then consumed by the playbooks phase. This keeps self-target safety out of ad hoc string checks in orchestration code.

### CLI API-response parsing

- Package: `cli/internal/apijson`
- Surface: `Parse[T]`
- Why it exists: empty-body and malformed-response diagnostics are transport concerns shared across CLI commands. Centralizing them prevents drift between `status`, `generate`, `runlocal`, `execute`, and playbooks seed commands.

### Registry-build ownership

- Package: `cli/internal/registry`
- Surface: thin wrapper over `api/internal/playbooks/registry`
- Why it exists: the CLI command is a delivery surface, not a second source of truth for registry schema. The wrapper keeps the command UX local while making the shared API builder the only normalization pipeline.

## Secondary seams

### HTTP transport adapters

- Package: `api/internal/app/httpserver`
- Surface: handler interfaces such as suite queue, execution service, and scenario directory service
- Why it exists: handlers should be testable with stubs and should not own domain rules.

### Phase command executors

- Package: `api/internal/orchestrator/phases`
- Surface: command execution helpers and phase-scoped dependencies
- Why it exists: phase implementations often need shell execution or lifecycle hooks; keeping those behind seams makes targeted tests possible without invoking the real world.

### Requirements reporting/rendering

- Package: `api/internal/requirements/reporting`
- Surface: renderer interfaces and format-specific builders
- Why it exists: markdown, JSON, and trace outputs should share the same domain summary while varying only presentation.

## What should not become a seam

- Queue stale filtering in the CLI. That belongs in persistence/repository code.
- Playbooks observer-mode detection in raw workflow execution. That belongs in registry data and orchestration setup.
- A second registry schema in the CLI. The command should delegate to the shared builder.
