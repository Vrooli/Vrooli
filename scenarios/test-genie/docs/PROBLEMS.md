# Open Issues
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
  - *Suite request ingestion*: depends on API payload validation and Postgres writes. Failure modes: invalid requested types/priority (client) vs. DB outages (infra). Current mitigation: validation errors stay 400 and non-validation paths now emit structured logging; DB outage still bubbles a 500—documented for follow-up.
  - *Suite execution orchestrator*: preflight includes Go-native structure + dependency phases, scenario script registry, and artifact persistence. Dependencies: filesystem layout, `.vrooli/service.json`, toolchain availability (`bash`, `curl`, `jq`, language runtimes, package managers), and manifest-declared resources.
- **Observed failure modes**
  - Missing directories or manifest drift silently mapped to 500s before this loop. They are now classified as `misconfiguration` with remediation text so UI/API callers can render contextual actions.
  - Dependency gaps were previously reported one-at-a-time; now the Go phase aggregates all missing commands and surfaces a single actionable error plus per-phase observations to avoid repeated API calls.
  - Optional data (e.g., Node workspaces without lockfiles, manifests without required resources) now degrade gracefully by issuing warnings/observations instead of failing the phase.
- **Remaining risks**
  - Execution persistence is still a single SQL INSERT with no retry or circuit breaker. If Postgres is down, the orchestrator returns a 500 even though the phase output is available in memory. Future loop: buffer execution records locally and retry asynchronously.
  - The Go phases emit failure classifications, but the UI/API have not consumed the new integration/perf telemetry yet—wire it into operator dashboards so failure context surfaces to users.
  - The new history endpoints expose per-phase failures to operators, but they still require a manual poll; future work should stream execution status so long-running suites can surface progress before completion.

# Deferred Ideas
- Evaluate which pieces of the archived Go services or CLI should be ported verbatim versus redesigned.
