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

### Execution bootstrap

- Package: `api/internal/orchestrator`
- Surface: prepared execution setup/finalization inside `SuiteOrchestrator`
- Why it exists: streaming and non-streaming suite execution must agree on runtime URL resolution, plan construction, artifact directories, result summaries, and requirement sync behavior. The shared bootstrap/finalization path keeps those decisions in one place.

### Target runtime lifecycle

- Package: `api/internal/orchestrator/targetruntime`
- Surface: `Manager.EnsureRunning()`, `RestartWithEnv()`, `Restore()`, and `Cleanup()`
- Why it exists: runtime-backed phases need one path-aware lifecycle boundary for starting and discovering the scenario under test. This avoids phase-local port fallbacks and keeps temporary generated scenarios isolated from Test Genie's own runtime environment.

### Workflow seed DB detection

- Package: `api/internal/playbooks/dbdetect`
- Surface: `Collector`, `Filesystem`, `Manifest` interfaces; `Resolver` + declarative `Profile` table; `DetectionReport`
- Why it exists: legacy workflow seed helpers need to decide which databases a target scenario actually uses (Postgres, Redis, SQLite, or any combination) before they provision temp resources. Putting the decision behind a resolver with explicit evidence sources keeps the rules in one declarative place: collectors emit raw observations, profiles pick which observations count for each DB, the resolver records the highest-priority decision plus corroboration and conflicts, and the result is printed as a `db-detect:` block in compatibility logs so every decision is traceable.

### Playbooks registry normalization

- Package: `api/internal/playbooks/registry`
- Surface: `Builder.Build()` and `Loader.Load()`
- Why it exists: BAS workflow files are authoring inputs; `bas/registry.json` is the execution contract. Registry normalization is where legacy labels, execution-mode hints, ordering, fixtures, and requirements become one stable manifest.

### Legacy workflow execution mode

- Package: `api/internal/playbooks/types`
- Surface: `Registry.Metadata.ExecutionMode`, `Registry.UsesObserverMode()`
- Why it exists: observer-mode vs mutating workflow metadata is a configuration decision that should be expressed in data. The active workflow phase delegates that safety decision to workflow-health; Test Genie keeps this type for compatibility with legacy seed/artifact paths.

### CLI API-response parsing

- Package: `cli/internal/apijson`
- Surface: `Parse[T]`
- Why it exists: empty-body and malformed-response diagnostics are transport concerns shared across CLI commands. Centralizing them prevents drift between `status`, `generate`, `runlocal`, `execute`, and playbooks seed commands.

### Requirements view loading

- Package: `api/internal/app/httpserver`
- Surface: `loadScenarioRequirementsView()` plus shared requirements-module parsing helpers
- Why it exists: the requirements UI needs one coherent projection of three sources of truth: cached summary snapshots, source `requirements/*.json` files, and sync metadata. The handler owns the projection; callers do not need to know where each field originated.

### Registry-build ownership

- Package: `cli/internal/registry`
- Surface: thin wrapper over `api/internal/playbooks/registry`
- Why it exists: the CLI command is a delivery surface, not a second source of truth for registry schema. The wrapper keeps the command UX local while making the shared API builder the only normalization pipeline.

### Tree digest (run freshness identity)

- Package: `packages/freshness-go/treedigest` (shared module; extracted from `api/internal/shared/treedigest` so cached status readers compute the same identity)
- Surface: `Compute()` / `ComputeWithRunner()` (injectable command runner), `CollectGitContext()`
- Why it exists: "has this scenario's current change-set been tested?" needs a byte-exact identity for the working tree a run executed against. The digest spec (sha256 over sorted relpath+content hashes of git-enumerable files, generated/state dirs excluded) is frozen here so the run-start stamper, `RunsService.CheckFreshness`, and every consumer compare the same value. The change-set fan-out lives in the CLI (`test-genie runs freshness --changed`, `cli/runs/freshness_changed.go`): it maps the git change-set to scenarios and checks each concurrently, degrading to `{checked:false}` instead of erroring. `vrooli hygiene` consumes that JSON as its advisory `test_freshness` check (warning findings, non-blocking), which is how the pre-commit flow sees staleness — git-control-tower deliberately knows nothing about freshness. Documented v1 limitation: scope is the scenario dir only — shared `packages/*` edits do not invalidate freshness. The required-phase set the freshness check defaults to is `phases.FreshnessRequired()` (≡ the quick preset), a code-level SSOT that is deliberately NOT per-scenario configurable.

### Business findings producer

- Package: `api/internal/orchestrator/phases` (`phase_business_findings.go`)
- Surface: `businessFindings()` over `business.RunResult.Issues` (rule-keyed structural issues) + `.Index` (parsed registry)
- Why it exists: the business phase's drift signal must reach the EM closed loop as typed `FINDING_SOURCE_BUSINESS` findings, not prose observations. The producer keys on `ValidationIssue.Rule` (set by `internal/requirements/validation`) so issue→code mapping never parses message text, and the registry-drift checks (starter template, no-validation, prd_ref unmatched) read the already-parsed module index instead of re-running discovery.

## Secondary seams

### HTTP transport adapters

- Package: `api/internal/app/httpserver`
- Surface: handler interfaces such as suite queue, execution service, and scenario directory service
- Why it exists: handlers should be testable with stubs and should not own domain rules.

### Generation phase control surface

- Package: `ui/src/pages/Generate`
- Surface: `PHASES_FOR_GENERATION` plus task-specific copy overrides in `PhaseSelector`
- Why it exists: the generation UI only exposes a small, intentional set of phase levers. Labels, descriptions, and button states should come from one control surface so the dialog, selector, and tests do not drift.

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
- An implicit Test Genie WebSocket endpoint in integration checks. Real-time agent updates come from `agent-manager`, so Test Genie's core integration phase should not assume an internal socket exists.
