# Open Issues

## Resolved 2026-07-10 — Descriptor phase additions broke fixed test registries

**Symptom:** The full API suite failed after the descriptor-owned `templates`
phase appeared: catalog guards lacked templates-specific surface/skip entries,
and the orchestration fixture called the live provider because its no-op runner
list named only the older phases.

**Root cause:** Test infrastructure retained three fixed phase registries after
production orchestration became descriptor-driven. The tests therefore required
coordinated Test Genie changes for a provider-owned catalog extension.

**Resolution:** Orchestration fixtures now replace every discovered runner and
detach provider transport generically. Capability guards validate normalized
descriptor ownership without pinning phase surface metadata, and skip-variable
guards derive the stable public name from each machine key. A synthetic future
phase fixture proves planning and execution require no phase registration.

**Owner:** Test Genie descriptor/catalog test architecture.

**Refs:** `api/internal/orchestrator/suite_execution_test.go`;
`api/internal/orchestrator/phases/catalog_guard_test.go`.

## Resolved 2026-07-10 — Foreground execute dropped aggregate duration and unavailable failures

**Symptom:** A lifecycle-managed `vrooli scenario test agent-manager --json`
run published the terminal verdict, phase results, and findings, but its
`phaseSummary.durationSeconds` was zero even though the phase durations were
nonzero. A `provider_unavailable` phase also increased `total` without
increasing any outcome bucket.

**Root cause:** The foreground FollowRun renderer rebuilt its terminal summary
with a private counting loop. Unlike the canonical wait/show summarizer, that
loop did not accumulate phase duration and did not classify
`provider_unavailable` as failed evidence.

**Resolution:** Foreground execute, wait, and show rendering now share the same
phase summarizer for total, outcome counts, duration, and observations. A
regression fixture covers nonzero durations and unavailable providers. Run
`20260711-034608-b6bdcd60` proved the repaired foreground summary and persisted
show projection both contain two failed phases totaling three seconds, while
the findings endpoint returns both phases.

**Owner:** Test Genie CLI durable execution projection.

**Refs:** `cli/execute/durable.go`; `cli/execute/standing.go`;
`cli/execute/durable_test.go`.

## Resolved 2026-07-10 — Terminal wait and show disagreed after durable fallback

**Symptom:** `test-genie runs wait --json swarm-manager 20260710-142937-ae6a753e`
returned terminal `failed` with zero phases and zero duration, while `runs show`
for the same run returned 20 persisted phase results spanning 14:47:02–14:59:57
UTC.

**Historical root cause:** Terminal wait projected the run manager's reduced
live/durable status fallback while show converted the persisted run record; the
two paths did not share a versioned terminal snapshot projector.

**Historical workaround:** Treat terminal wait output with missing phase data as
degraded and inspect the same run with show. Do not relabel the zero-phase
result as a pass or start a replacement run to manufacture baseline evidence.

**Resolution:** New runs atomically persist one schema-versioned terminal
snapshot and both wait and show project it before and after retirement/restart.
Legacy, partial, and corrupt records remain explicitly degraded. The original
run remains a diagnostic fixture; only a fresh exact-SHA lifecycle run may
replace it as authoritative evidence.

**Owner:** test-genie run lifecycle.

**Refs:** `api/internal/app/runs/lifecycle.go`;
`api/internal/app/runs/convert.go`; run `20260710-142937-ae6a753e`.

## Test Gaps — typed artifact catalog live proof

Phase 4 has race-enabled unit and contract coverage for atomic catalogs,
runtime-emitted unknown kinds, legacy discovery, screenshot/video enumeration,
opaque list/detail/byte access, missing files, cross-run IDs, symlink escape,
and active-content headers. Phase 10's copy-first rehearsal also proved repeated
wait/get/catalog projections and opaque resolution over 116 copied historical
Test Genie runs and 2,237 artifacts without changing the copied byte tree; all
116 honestly remained legacy because they predate canonical snapshots and
persisted catalogs. The remaining proof is a clean, lifecycle-managed exact-SHA
run confirming that new ui-health screenshots and workflow-health recordings
are persisted and streamed through opaque routes. Copied historical evidence is
strong migration evidence, but it is not an authoritative post-repair run.

- Detection refactor 2026-05: playbooks DB detection moved to `internal/playbooks/dbdetect`; the silent Postgres+Redis fallback is eliminated. Scenarios with no manifest/godeps/source evidence now provision nothing for legacy workflow seed helpers — fix at the source (declare the resource or add the driver import) rather than reinstating a fallback.
- Historical queue, generation, and separate fix/requirements-improve gaps listed here were retired in July 2026. The supported workflow is now evidence-backed remediation from a completed execution; do not reintroduce those independent paths to address old ledger entries.

# Failure Topography (2025-12-03)
- **Critical flows mapped**
  - *Suite request ingestion*: depends on API payload validation and embedded SQLite writes. Failure modes: invalid requested types/priority (client) vs. DB access failures (infra). Current mitigation: validation errors stay 400 and non-validation paths now emit structured logging; storage failures still bubble a 500—documented for follow-up.
  - *Suite execution orchestrator*: preflight includes provider-backed catalog phases and artifact persistence. Dependencies: filesystem layout, `.vrooli/service.json`, provider availability, language runtimes, package managers, and manifest-declared resources.
- **Observed failure modes**
  - Missing directories or manifest drift silently mapped to 500s before this loop. They are now classified as `misconfiguration` with remediation text so UI/API callers can render contextual actions.
  - Dependency gaps were previously reported one-at-a-time; now the Go phase aggregates all missing commands and surfaces a single actionable error plus per-phase observations to avoid repeated API calls.
  - Optional data (e.g., Node workspaces without lockfiles, manifests without required resources) now degrade gracefully by issuing warnings/observations instead of failing the phase.
- **Remaining risks**
  - Execution persistence is still a single SQL INSERT with no retry or circuit breaker. If the SQLite store cannot be opened or written, the orchestrator returns a 500 even though the phase output is available in memory. Future loop: buffer execution records locally and retry asynchronously.
  - The Go phases emit failure classifications, but the UI/API have not consumed the new integration/perf telemetry yet—wire it into operator dashboards so failure context surfaces to users.
  - The new history endpoints expose per-phase failures to operators, but they still require a manual poll; future work should stream execution status so long-running suites can surface progress before completion.

# Deferred Ideas
- Evaluate which pieces of the archived Go services or CLI should be ported verbatim versus redesigned.
