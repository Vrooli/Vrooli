# Documentation Seams

This file defines the boundaries between documentation layers.

## Project-Level Docs

Project-level docs should explain:

- what Vrooli is
- how the platform is organized
- how the root CLI and control plane work
- how scenarios and resources relate
- what deployment maturity looks like at a high level

They now primarily live under:

- `path:docs/concepts/`
- `path:docs/guides/`
- `path:docs/reference/`
- `path:docs/operations/`
- `path:docs/deployment/`
- `path:docs/strategy/`

## Scenario-System Docs

`path:docs/scenarios/` should explain:

- how the scenario ecosystem works
- scenario authoring norms
- scenario validation and deployment patterns

Individual scenario docs should explain the specific scenario.

## Resource-System Docs

`path:docs/resources/` should explain:

- how resources are modeled
- resource templates and blueprints
- resource governance and lifecycle policy

Individual resource docs should explain the specific resource.

## Plans

Scratch implementation plans should be authored through Plan Manager:
`plan-manager author start`, `plan-manager author continue`, and
`plan-manager author finalize`. Plan Manager creates the structured canonical
record and publishes a rendered markdown mirror outside the git worktree under
the repo-contract runtime-home `plans` directory.

Root `vrooli plans` has been retired. Plan lifecycle, inspection, import,
archive, authoring, and render/export workflows belong on `plan-manager`
directly. The only project-level plan integration is hygiene:
`vrooli hygiene --plans` / `vrooli hygiene --plans-only`, which delegates
reconciliation to Plan Manager.

Root hygiene talks to Plan Manager through `internal/app/hygiene::PlanReconciler`
and `ReconcilePlans`. Unavailable/timeout errors degrade to the static advisory
scan, while canonical invalid/not-found/server errors remain authoritative
failures. Under `--fix-safe --plans`, hygiene asks Plan Manager to repair
mirrors, canonicalize misplaced markdown sources, and retire source files only
after Plan Manager reports them canonical/imported/duplicate.
Parse failures and conflicts are guided hygiene issues, not automatic fixes:
safe-fix leaves those source files untouched, and the next step is to inspect
`vrooli hygiene --plans-only --details`, repair the markdown into an importable
plan or move non-plan notes out of plan source locations, then rerun hygiene.

`path:docs/plans/` are not current truth by default. They are promoted design and migration artifacts unless a canonical doc explicitly points to them as active.

## Scenario Lifecycle Wait-Contract Seams

## React Component Library version ledger seam

The RCL `version_ledger` is a replayable projection boundary, not a second
source of truth. `internal/versionledger.Repository.Rebuild` reads indexed
version identity, version-scoped gate evidence, compact test rollups, and
adoption facts, then upserts one stable `(library_id, version)` row. Lifecycle
verbs use the same repository to protect the reference graph before changing
manifest state or reclaiming a version folder. Progression consumers and
measures call the ledger read surface; they do not issue equivalent ad hoc
queries.

Root control-plane code seams introduced by the scenario start wait contract
(`docs/plans/scenario-lifecycle-start-wait-contract-plan.md`); each exists so
the next agent extends it instead of hand-rolling a parallel mechanism:

- `lifecycle.Await` + `AwaitPolicy` (`internal/lifecycle/await.go`) — the ONLY
  wait/poll primitive for lifecycle orchestration; every timeout/interval is a
  row in its policy table, so bespoke `for { sleep }` loops are prohibited.
- `lifecycle.ProgressSink` / `ProgressEvent` (`internal/lifecycle/progress.go`)
  — start progress as typed events; the Runner's built-in sink renders the
  historical human lines byte-for-byte, and additional sinks (registry record,
  test capture) compose via `WithProgressSink`.
- Start-operation record (`internal/scenarioruntime/startop.go`,
  `internal/lifecycle/operation_record.go`) — durable per-start progress in
  the runtime registry (`runtime_start_operations`, `runtime_phase_durations`).
  Progress only, NEVER authority: health/ports authority stays on
  `runtime_instances`/`runtime_port_claims`, and a dead-initiator record is
  reported `abandoned`, never a reason to refuse a start.
  Schema note: the runtime registry and capacity ledger keep one declarative
  `schemaSQL` each plus a `user_version` stamp with hard-error guards — no
  in-code migration ladders while greenfield; existing local DBs convert via
  one-shot operator scripts (see "Persisted-Data Schema Convention" in
  `docs/package-governance.md`).
- Attach/wait (`internal/lifecycle/wait.go`) — `Runner.WaitScenario` is the
  single blocking primitive behind `vrooli scenario wait`, concurrent-start
  attach, and the agent-manager Waiter; owner death converts an attach into a
  takeover.
- `cliutil.ParkProducerLifecycle` (`packages/cli-core/cliutil/park.go`) with
  the matching Waiter in
  `scenarios/agent-manager/api/internal/orchestration/waiter.go` — parks
  agent-manager runs on scenario waits (key `"<scenario>/<variant>"`).

## Test Genie Descriptor-Backed Phase Seams

Provider-backed Test Genie phase ownership is split deliberately:

- Provider scenarios own `scenarios/<provider>/.vrooli/test-genie.json`. That descriptor is the single checked-in source for phase identity, policy, applicability declaration, runnability capabilities, docs path, validation transport, and the embedded maturity/finding contract.
- `providerdescriptor` (`scenarios/test-genie/api/internal/orchestrator/providerdescriptor`) loads and validates descriptors without provider, network, or runtime side effects. It also rejects retired sibling `.vrooli/maturity.json` files.
- `phaseregistry` (`scenarios/test-genie/api/internal/orchestrator/phaseregistry`) validates the effective registry and binds descriptor sources to Test Genie-owned runner implementations. It owns ordering and duplicate/missing/unsupported-source diagnostics, but not applicability.
- `applicability` (`scenarios/test-genie/api/internal/orchestrator/applicability`) is a pure predicate evaluator over target facts. It never calls providers and never starts targets.
- `phasepolicy` (`scenarios/test-genie/api/internal/orchestrator/phasepolicy`) owns selection, provider-readiness, lifecycle, freshness, result-gating, and unavailable semantics. New code must use these policy dimensions instead of expanding the old `optional` boolean.
- `providerreadiness` (`scenarios/test-genie/api/internal/orchestrator/providerreadiness`) is the execution-time seam that checks, starts, restarts, and probes only providers selected for the current applicable plan.

Inspection surfaces are part of the seam: `test-genie phases list|inspect|applicability|plan --json` and `test-genie provider-contract scan --json` expose the registry, applicability reasons, policy, readiness posture, descriptor source, and conformance status so agents do not need to infer plan state from logs.
