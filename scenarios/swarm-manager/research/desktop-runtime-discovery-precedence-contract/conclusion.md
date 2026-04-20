# Research Conclusion: Desktop Runtime Discovery And Precedence Contract

## Research Question
When downloaded Vrooli desktop apps (bundles) run alongside each other and/or alongside a host Vrooli swarm, how do they resolve inter-scenario calls that use the standard discovery package (`packages/api-core/discovery`), and what precedence and cross-platform behavior must they guarantee so the set behaves as one coherent ecosystem?

Specifically, the contract must define:
1. Default precedence when a downloaded app and a local-swarm instance of the same scenario both exist.
2. Resolution semantics for inter-scenario calls routed through the standard discovery package.
3. How bundled apps expose Vrooli-compatible discovery behavior.
4. The cross-platform contract for macOS, Windows, and Linux.
5. A future seam for explicit override/configuration without blocking the default behavior now.
6. The regression cases required to prove downloaded+downloaded and downloaded+local-swarm topologies work correctly.

## Summary
All four core design decisions and all three publication/scope decisions are now settled.

**The contract in one paragraph:** The standard discovery package (`packages/api-core/discovery`) is unchanged — all inter-scenario resolution continues to flow through `vrooli scenario port`. Downloaded desktop apps participate by writing the **same** process record shape as the host swarm into the **same** `~/.vrooli/processes/scenarios/<name>/<step>.json` location on start, and removing it on stop, with an added `source: "bundle"` tag on every bundle-written record. The "downloaded wins over local swarm" precedence is enforced inside `internal/app/scenario/service.go::Port()` as a simple record comparison (`source=bundle` beats `source=swarm` — and an empty `source` is treated as `source=swarm` for precedence purposes). Stale records from crashed bundles are filtered on the reader side using the existing `process.LiveRecords()` PID-liveness check. The bundle-local `vrooli` shim (today bash on macOS/Linux, a cross-compiled `vrooli.exe` helper on Windows) reads the same registry for unknown scenarios, so inter-bundle discovery works on hosts that have never installed the host Vrooli CLI. The future per-process override seam is `VROOLI_SCENARIO_<NAME>_URL`, read inside the resolver before it shells out.

**Why this lands cleanly:** there is zero new discovery infrastructure. Every mechanism in the contract is a small additive change to systems that already exist. The file-based process registry is already read by the CLI's `Port()` function and its liveness filter (`process.LiveRecords`) already removes dead PIDs. `api-core/discovery` stays agnostic — desktop semantics live entirely in the CLI and the bundle.

## Methodology
- Read `packages/api-core/discovery/resolve.go` to characterize the current discovery path (CLI-mediated `vrooli scenario port` lookup).
- Read `scenarios/scenario-to-desktop/api/bundle/staging.go` to characterize the bundle-local `vrooli` shim that is staged for non-Windows targets.
- Cross-referenced the initiative siblings (`execute/cross-platform-bundled-discovery-parity`, `execute/desktop-ecosystem-interop-regression-coverage`) to understand downstream consumers of this contract.
- Reviewed `orchestration-summary.md` for the planning session's working assumptions, in particular that downloaded apps should win over same-scenario local swarm instances by default.
- **Round 2:** traced `vrooli scenario port` end-to-end through `internal/cli/scenariocli/commands.go` → `internal/cli/scenariohandlers/handlers.go` → `internal/app/scenario/service.go::Port()` → `internal/scenario/runtime_state.go::RuntimePortBindings()` to characterize the existing process-record resolver the CLI already uses. Examined `internal/process/process.go` for the record schema and `internal/lifecycle/phases.go` + `internal/lifecycle/lifecycle.go` for start/stop registration and cleanup semantics.
- **Round 3:** verified that `internal/process/process.go::LiveRecords()` (lines 146-161) already performs PID-based liveness filtering on the full record set before consumers see it — so D3=A (reader-side liveness check) is not a new primitive, it is the primitive the CLI already uses. Cross-checked `ReadScenarioRecords()` (lines 115-144) to confirm that an unknown `source` value on a legacy (swarm-written) record will Unmarshal cleanly because the proposed field is additive JSON with omitempty. Surveyed `scenarios/scenario-to-desktop/runtime/` to identify the natural registration call site (no existing references to `~/.vrooli/processes/` from the bundle runtime — this is net-new integration).

## Findings

### Finding 1: Standard discovery is CLI-mediated and has no override seam today
`packages/api-core/discovery/resolve.go` resolves a scenario name to a port by shelling out to `vrooli scenario port <scenario> <portKey>` on every call. The resolver has:
- `VrooliPath` (default `"vrooli"`), `Host` (default `"localhost"`), and `Scheme` (default `"http"`) config fields.
- A `StaticBaseURL` test hook that bypasses the CLI entirely.
- No environment-variable override (e.g., `VROOLI_SCENARIO_<NAME>_URL`) and no registry/file lookup.

Implication: whoever provides the `vrooli` binary on `PATH` decides what "discovery" means for any process the desktop app spawns. Precedence is currently **implicit in PATH ordering**, not enforced at a higher layer.

### Finding 2: A bundle-local shim already emulates `vrooli scenario port` — but only for its own bundle and only for non-Windows
`scenarios/scenario-to-desktop/api/bundle/staging.go` stages a thin bash shim at `<bundle_root>/bin/vrooli` that:
- Reads `<bundle_root>/bundle.json` to discover services by `type` (`api-binary` → `API_PORT`, `ui-bundle` → `UI_PORT`).
- Reads the bundle's runtime IPC port from `$XDG_CONFIG_HOME/<app_name>/runtime/ipc_port` (falling back to the manifest-declared default).
- Execs `runtimectl port --service <svcId> --port-name <portName>` to ask the bundled runtime for the live port.
- Falls through to generic `runtimectl <cmd> <args>` for any other subcommand.

Two explicit gaps in the current shim:
- **Windows is skipped** (`if strings.HasPrefix(platform, "win") { return nil }`). There is no Windows equivalent staged today.
- **Only own-bundle services are resolvable.** The shim's manifest lookup is limited to the current `bundle.json`. A bundled app asking for *another* downloaded app's scenario via this shim will receive "no service for type …" and fail.

### Finding 3: `vrooli scenario port` already reads a file-based process registry — this is the leverage point
`internal/app/scenario/service.go::Port()` (lines 223–273) resolves a port by:
1. Calling `Scenarios.Detail(name)` and `BuildListPorts(manifest, runtime.Records)`.
2. Reading JSON process records from `~/.vrooli/processes/scenarios/<scenario-name>/<step-name>.json`.
3. Running `resolveRequestedPort()` to pick the requested port name from the record set.

Record schema (`internal/process/process.go:17-30`) already includes `PID`, `PGID`, `Port`, `Scenario`, `Step`, `StartedAt`, `Status`. Records are written by `startTrackedProcess()` in `internal/lifecycle/phases.go:350–375` on scenario start and removed by `process.RemoveScenarioRecord()` via `lifecycle.Stop()` (`internal/lifecycle/lifecycle.go:515–540`).

**This reframes the contract:** there is no need to invent a new registry. A downloaded app can participate by writing the same record shape into the same directory on start and removing it on stop. The existing CLI resolver picks it up for free. The only new logic is (a) a `source: "bundle"` field to distinguish origin, and (b) a precedence rule in `Port()` that prefers `source=bundle` over `source=swarm` when both exist for the same scenario/port.

### Finding 4: `LiveRecords()` already implements the D3 liveness strategy
`internal/process/process.go::LiveRecords()` (lines 146-161) iterates every record through `isPIDRunningFn(record.PID)` (which wraps `IsPIDRunning`) and drops records whose PID is not alive. This is precisely the strategy chosen in D3 (reader-side liveness check). `Port()` already feeds records through this filter before resolving. Stale bundle records from crashed apps will therefore be filtered transparently using the same code path that already handles dead swarm records — no new mechanism is needed.

PID-reuse risk (a recycled PID making a stale record look alive) is the same risk the swarm already tolerates. Comparing `StartedAt` against process start time is a future hardening opportunity, not a contract requirement.

### Finding 5: The Source field is additive and backward-compatible
Adding a `Source string \`json:"source,omitempty"\`` field to `Record` in `internal/process/process.go` is backward-compatible in both directions:
- Legacy records (no `source` key in JSON) unmarshal cleanly via `json.Unmarshal()` — the field zero-values to `""`.
- **Empty `source` is treated as `"swarm"` for precedence purposes** (settled in round 3, d3=A). This preserves the invariant that all existing records on disk today belong to the swarm, and makes the precedence rule a pure `bundle > swarm|empty` comparison.
- Older readers that do not know about the field ignore it (`omitempty` on write keeps the JSON narrow when unset).

This means the swarm lifecycle does not have to change in lockstep with the bundle lifecycle — the swarm can start writing `source: "swarm"` explicitly as a later hardening, or never at all, without breaking precedence. The bundle must always write `source: "bundle"`.

### Finding 6: Precedence enforcement is a ~10-line patch inside `Port()`
With records coming in already liveness-filtered and carrying a `Source` tag, the precedence rule reduces to: after `BuildListPorts` returns candidate records for the requested port name, if any candidate has `Source == "bundle"`, filter the candidate set to just those records. Otherwise, return the set unchanged (treating `Source == ""` identically to `Source == "swarm"`). This preserves current behavior on all-swarm or all-bundle topologies, and enforces "downloaded wins" on mixed topologies. The rule lives entirely in `internal/app/scenario/service.go::Port()` — no api-core change, no resolver change.

### Finding 7: Cross-platform state-dir divergence is a latent hazard — but tractable
`internal/config/config.go` hardcodes `~/.vrooli/` — there is no XDG support, and Windows uses the same `$HOME/.vrooli/` path (via `os.UserHomeDir()`). The contract keeps this simple: the bundle registration code uses the same `config.HomeDir()` resolver the rest of the CLI uses, so bundle and host always look at the same location on every platform. Moving to XDG-native paths is a follow-up, not a blocker.

### Finding 8: The bundle registration call site is greenfield
A survey of `scenarios/scenario-to-desktop/runtime/` turned up **no existing references** to `~/.vrooli/processes/` from the bundle runtime. This means the registration integration is net-new code with no backwards-compat concerns inside the bundle runtime itself. The natural call site is after the bundle runtime successfully launches an `api-binary` service and has the real port; the natural deregistration site is the bundle runtime's clean shutdown path. Exact function placement is an implementation concern for `execute/cross-platform-bundled-discovery-parity`, not a contract concern.

### Finding 9: Siblings encode downstream assumptions the contract must satisfy
- `execute/cross-platform-bundled-discovery-parity` (L) will implement Windows parity, consistent `vrooli scenario port`-style lookups inside downloaded apps, runtime precedence (downloaded > local swarm), docs, and tests — and leaves room for a future explicit override layer.
- `execute/desktop-ecosystem-interop-regression-coverage` (L) will validate (a) downloaded → downloaded calls, (b) downloaded peer preferred over same-scenario local swarm, (c) mixed topologies, (d) core cross-platform interop cases, (e) evidence capture feeding deployment review.

The contract from this research must be concrete enough for both of these to implement/validate without further design.

## Contract Summary (Final Form)

### Default precedence
For a given `(scenario, port_name)` tuple, when both a downloaded-app record (`source=bundle`) and a local-swarm record (`source=swarm` **or empty**) exist and are live, the **bundle record wins**. Empty `source` is treated as `swarm` for precedence purposes — no migration or backfill of existing records is required. If multiple bundle records exist for the same tuple, the resolver currently returns the first per existing `resolveRequestedPort` semantics; if this becomes a real problem, a secondary "most-recently-started wins" tiebreaker is the obvious next step and does not break the contract.

### Resolution semantics
All resolution continues to flow through `vrooli scenario port <scenario> <port_name>` via `packages/api-core/discovery/resolve.go`. No changes to the api-core public surface. The precedence rule lives in the CLI resolver.

### Bundle participation
A downloaded desktop app:
- Writes `process.Record{Scenario, Step, Port, PID, StartedAt, Source: "bundle", ...}` to `<home>/.vrooli/processes/scenarios/<scenario>/<step>.json` when its service is up and the real port is known.
- Removes the record on clean shutdown.
- Stale records from crashes are handled by the resolver's existing PID-liveness filter — the bundle does not need heartbeating.
- The bundle-local `vrooli` shim reads the same registry when resolving peer scenarios not present in its own `bundle.json`.

### Schema change
Add `Source string \`json:"source,omitempty"\`` to `process.Record` in `internal/process/process.go`. Accepted values: `""` (legacy, treated as `swarm`), `"swarm"`, `"bundle"`. Additive and backward-compatible. No coordinated swarm-lifecycle change is required; existing swarm records keep working with `source` unset.

### Cross-platform contract
| Platform | Shim form | Registry path |
|----------|-----------|---------------|
| Linux    | Existing bash shim + registry-read path for unknown scenarios | `$HOME/.vrooli/processes/scenarios/<name>/` |
| macOS    | Existing bash shim + registry-read path for unknown scenarios | `$HOME/.vrooli/processes/scenarios/<name>/` |
| Windows  | New cross-compiled `vrooli.exe` helper (Go), covers same surface natively | `%USERPROFILE%\.vrooli\processes\scenarios\<name>\` (via `os.UserHomeDir()`) |

### Future override seam
`packages/api-core/discovery/resolve.go` gains an environment-variable lookup **before** its existing CLI shell-out: if `VROOLI_SCENARIO_<UPPER_NAME>_URL` is set, short-circuit to that URL. No other behavior changes. This is the sanctioned per-process override for tests, scripts, and operator-driven precedence inversion. Broader config-file overrides are out of scope; this seam does not preclude them later.

## Regression Matrix

The matrix below is the set of cases `execute/desktop-ecosystem-interop-regression-coverage` must cover to prove the contract holds. Each row is a distinct topology + observable behavior.

**Scope rule (settled in round 3, d2=A):** Cases 1–9 are **mandatory** for the contract to be considered proven. Case 10 (concurrent-start race) is **nice-to-have / future hardening** — it is a robustness property rather than a contract property, and a reliable repro may require a dedicated harness. It should be tracked as a follow-up if it is not delivered in the first implementation pass.

| # | Class | Topology | Call | Expected |
|---|-------|----------|------|----------|
| 1 | Mandatory (core) | Downloaded-A only (running) | A→A internal port lookup via shim | Resolves via own `bundle.json` path (existing behavior, regression guard) |
| 2 | Mandatory (core) | Downloaded-A + Downloaded-B (both running) | A→B via shim | Shim consults shared registry, returns B's bundle-written port |
| 3 | Mandatory (core) | Downloaded-A + Local-swarm-B (both running, different scenarios) | A→B via shim | Shim consults registry, returns swarm B's port |
| 4 | Mandatory (core) | Downloaded-A + Local-swarm-A (same scenario) | Third scenario (host swarm caller) → A via `vrooli scenario port` | Returns **downloaded A's** port (bundle beats swarm) |
| 5 | Mandatory (core) | Downloaded-A + Local-swarm-A, downloaded-A crashes | Same as #4, after crash | PID liveness filter drops dead record; resolver falls through to swarm A |
| 6 | Mandatory (core) | Downloaded-A alone, bundle shut down cleanly | Any caller asks for A | Deregistration removed the record; resolver returns "not found" |
| 7 | Mandatory (core) | Override seam | Any caller with `VROOLI_SCENARIO_A_URL` set | Resolver short-circuits to the env var URL, ignores registry entirely |
| 8 | Mandatory (Windows parity) | Windows host: Downloaded-A + Downloaded-B | Same as #2, Windows | `vrooli.exe` helper resolves identically to Unix shim |
| 9 | Mandatory (Windows parity) | Windows host: Downloaded-A + Local-swarm-A | Same as #4, Windows | `vrooli.exe` enforces same precedence as Unix |
| 10 | **Nice-to-have / future hardening** | Concurrent start: two bundles of the same scenario start near-simultaneously | Caller resolves mid-race | Resolver returns one consistent live record per call; never a mix of two records for one response |

Cases 1–7 are the core semantic matrix. Cases 8–9 prove cross-platform parity. Case 10 is a stress/robustness case and is explicitly non-blocking for initial contract proof.

## Actions

### Action 1: Update document — Publish the authoritative contract spec under internal docs
- **File**: `docs/internal/desktop-runtime-discovery-contract.md`
- **Change**: Create a new internal spec document using the "Contract Summary (Final Form)" section above as the normative content (default precedence, resolution semantics, bundle participation, schema change, cross-platform table, override seam), plus the Regression Matrix as an appendix. This is the authoritative engineering contract — both `execute/cross-platform-bundled-discovery-parity` and `execute/desktop-ecosystem-interop-regression-coverage` will cite it. Keep it alongside other SEAMS-style internal specs for discoverability by engineers.

### Action 2: Update document — Publish a deployment-doc explainer that links to the internal spec
- **File**: `docs/deployment/desktop-runtime-discovery.md`
- **Change**: Create a short (half-page) deployment-focused explainer covering: what the behavior means for operators of Vrooli desktop apps (downloaded apps take precedence, stale records self-heal, override seam for pinning), with a prominent link to `docs/internal/desktop-runtime-discovery-contract.md` as the normative source. Do not restate the contract body here — only the operator-visible consequences. Link from `docs/deployment/README.md`.

### Action 3: Update document — Add pointer to the internal spec from the architecture docs index
- **File**: `docs/README.md`
- **Change**: Add a short "Desktop runtime discovery" entry in the internal/architecture section pointing to `docs/internal/desktop-runtime-discovery-contract.md`, so engineers looking for discovery/runtime behavior can find it from the docs root.

### Action 4: Hand off — `execute/cross-platform-bundled-discovery-parity` has everything it needs
- The Contract Summary is sufficiently concrete for implementation without further design work. No new backlog item needed.

### Action 5: Hand off — `execute/desktop-ecosystem-interop-regression-coverage` has everything it needs
- The Regression Matrix (with Cases 1–9 mandatory, Case 10 follow-up) is the input that item needs to scope test topologies. No new backlog item needed.

### Action 6: No separate backlog item for the schema or precedence patch
- The `Source` field addition to `process.Record` and the ~10-line precedence check in `Port()` are small, coherent parts of `cross-platform-bundled-discovery-parity`. Splitting them out would fragment the change.

## Limitations
- "Most-recently-started wins" as a tiebreaker between multiple bundle records for the same scenario is noted but not specified in this contract. If it becomes material during implementation, extend the contract rather than re-research.
- PID-reuse edge case (a recycled PID making a dead record look alive) is inherited from the existing swarm liveness strategy and is not hardened here. A `StartedAt`-vs-process-start-time comparison is the known future hardening.
- Default precedence assumes exactly one `source=bundle` record is live per `(scenario, port)` in a healthy topology. Multi-install-of-same-app scenarios (e.g., two installs of the same bundle on one user account) are out of scope; if that becomes a real use case, a tiebreaker specification is required.
- Cross-platform state-dir layout stays on `$HOME/.vrooli/` on all platforms. Migration to XDG / Windows-native paths is out of scope for this contract.
- No mDNS/socket-advertisement path was specified. The contract is filesystem-only. If a future remote-host topology becomes a goal, a network-advertisement layer would be additive on top of the file registry, not a replacement.
- Concurrent-start races (Regression Matrix case 10) are acknowledged but explicitly deferred as non-blocking for initial contract proof. If reliable reproduction proves hard during implementation, it should become its own follow-up backlog item rather than gating this work.
- Dual-publication of the contract (internal spec + deployment explainer, per d1=C) introduces a small drift risk if the two documents are edited independently over time. The explainer is intentionally short and points to the internal spec as normative; that is the mitigation. If drift becomes a real problem, collapsing back to a single document is cheap.
- Confidence in findings 1–9 is high (direct source references with file paths and line numbers). The Contract Summary and Regression Matrix are derived from settled decisions and verified code paths.
