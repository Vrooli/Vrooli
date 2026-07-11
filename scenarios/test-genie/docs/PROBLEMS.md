# Open Issues

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
and active-content headers. The remaining proof belongs to Phase 10: a clean,
lifecycle-managed run must confirm that real ui-health screenshots and
workflow-health recordings are catalogued and streamed through opaque routes.
Focused fixtures are not authoritative live evidence.

- Detection refactor 2026-05: playbooks DB detection moved to `internal/playbooks/dbdetect`; the silent Postgres+Redis fallback is eliminated. Scenarios with no manifest/godeps/source evidence now provision nothing for legacy workflow seed helpers — fix at the source (declare the resource or add the driver import) rather than reinstating a fallback.
- The scenario still needs to rebuild the CLI delegation workflow that triggers suite generation remotely.
- Requirement modules now have `[REQ:TESTGENIE-*]` tags across Go + CLI suites, but we still need to run the orchestrator through the lifecycle to refresh requirement snapshots and add UI/E2E coverage (vault dashboard, delegated flows) before OT-P0-002 is truly multi-layer.
- Requirements sync now runs directly from the Go orchestrator, but it only fires after full-suite executions triggered via the API—`coverage/run-tests.sh` and the UI still need to delegate to that path so coverage snapshots stay fresh without manual commands.
- Queue telemetry now surfaces in `/health` and the CLI, but there is still no alerting when items stay queued for too long or when execution failures spike—ecosystem-manager will need to subscribe to the new signals to close that gap.
- The React dashboard now exposes queue metrics and runner triggers, but it still lacks visibility into delegated issue IDs, coverage/vault analytics, and historical suite grouping, so ops personas cannot yet audit whether AI generation actually closed the gaps they queued.
- The new quick-focus rail is still ephemeral; we need to persist the chosen scenario and outstanding queue/execution context (local storage + API) so operators can resume their intent after refreshes instead of retyping every session.
- Personas still share the same overview density; watchers and decision-makers need a lighter "state of testing" digest surface that summarizes focus, backlog, vault coverage, and failure stories without the instrumentation-heavy controls that builders expect.
- Flow highlight cards now surface the oldest high-priority queue entry and the most recent failed execution, but guidance still stops there—coverage/vault analytics and automated alert thresholds remain missing, so ops personas cannot yet tell when AI assistance produced enough suites or which vault needs attention next.

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
