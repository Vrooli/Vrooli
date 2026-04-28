# Plan: Verify and Finalize App-Monitor's Hard Cutover from App-Issue-Tracker to Swarm-Manager

## Required Reading
- `scientific-debugging` — for any verification gaps that surface as bugs during the audit (root-cause before patching).
- `swarm-manager-backlog-tools` — canonical CLI/contract for any backlog interactions inside the migration mapping doc examples.

## Problem & Goal
The in-flight migration of app-monitor from the `app-issue-tracker` scenario to the `swarm-manager` scenario must be a **hard cutover**: zero residual references, zero compat shims, zero feature flags. We must:
1. Audit the existing migration to confirm parity.
2. Eliminate every residual `app-issue-tracker` reference in `scenarios/app-monitor/**`.
3. Produce a migration-mapping doc that can serve as the template for the cross-scenario sweep that depends on this item (`execute/cross-scenario-issue-tracker-cutover-sweep`).

## Current-State Snapshot (round 1 audit)
Run from `scenarios/app-monitor`:

- **Code references to `app-issue-tracker` / `appIssueTracker` / `app_issue_tracker`:** Zero matches in source (.go, .ts, .tsx). Only residue: `.vrooli/deployment/deployment-report.json` (5 matches — stale generated snapshot from 2025-11-16 that still lists `app-issue-tracker` as a declared scenario dependency with its full child resource tree).
- **service.json (`.vrooli/service.json`):** Declares only `swarm-manager` (required, `>=1.0.0`) and `scenario-auditor` under `dependencies.scenarios`. No `app-issue-tracker` entry. ✅
- **API → swarm-manager call sites:**
  - `api/services/app_reports.go:485-505` — direct HTTP POST to swarm-manager backlog endpoint with proper error handling.
  - `api/toolregistry/issue_tools.go:71,129` — tool registry tags reference `swarm-manager`.
  - `api/services/app_diagnostics_aggregator.go` — modified to route through swarm-manager.
  - `api/services/app_issues.go` — **deleted** (was the AIT integration module).
  - New file: `api/services/app_fixes.go` (replaces issues semantics with "fixes" terminology).
- **UI references:** `ui/src/state/scenarioIssuesStore.ts`, `ReportIssueDialog.tsx`, `useReportExistingIssues.ts`, `useReportIssueState.ts`, `DiagnosticsTab.tsx`, `LighthouseTab.tsx`, `services/api.ts`, `types/index.ts` — all modified, none reference AIT names.
- **Tests:** `apps_test.go`, `app_business_logic_test.go`, `app_reports_test.go`, `app_service_test.go`, `PreviewPane.test.tsx` modified; new `test_helpers_test.go` added. Need to run to confirm green without AIT.
- **Migration mapping doc:** Does not exist yet.

## Scope
**In scope (this item):**
- Final residual cleanup inside `scenarios/app-monitor/**` (specifically `.vrooli/deployment/deployment-report.json`).
- Authoring the migration mapping doc.
- Executing the test suite to confirm cleanness.
- The verification audit (inventory + parity check).

**Out of scope:**
- swarm-manager-side documentation cleanup (`scenarios/swarm-manager/docs/internal/INTEROP_AUDIT.md`, `INTENT.md`, `reference/operational-targets.md`, `guides/research-notes.md` still reference AIT — these are owned by `chore/app-issue-tracker-deprecation`).
- Migrating other scenarios — that is `execute/cross-scenario-issue-tracker-cutover-sweep` (depends on this item).
- Archiving/deleting the `app-issue-tracker` scenario directory itself (owned by the deprecation chore).

## Approach
1. **Inventory phase** — re-run the canonical acceptance grep:
   ```bash
   rg -n 'app-issue-tracker|appIssueTracker|app_issue_tracker' scenarios/app-monitor
   ```
   Confirm only `.vrooli/deployment/deployment-report.json` matches.

2. **Parity phase** — for each prior AIT integration point (issue-report flow, diagnostics→issue creation, agent-spawn flows), trace the equivalent swarm-manager call. Capture endpoint and type mappings as we go (this becomes the mapping doc).

3. **Residual cleanup phase** — handle `.vrooli/deployment/deployment-report.json` per decision d2.

4. **Doc phase** — write migration mapping doc per decision d1.

5. **Test phase** — run app-monitor test suite (Go API + UI) per decision d3 and confirm zero AIT runtime dependencies.

6. **Acceptance phase** — re-run all four acceptance checks.

## Migration Mapping (preliminary, to be finalized in the doc)
| Concern | App-Issue-Tracker (before) | Swarm-Manager (after) |
|---|---|---|
| Issue creation | AIT issue API | swarm-manager backlog `fix` item creation (`POST /backlog/items`) |
| Issue listing | AIT scoped per-scenario | swarm-manager backlog list filtered by scenario tag |
| Diagnostics→issue | AIT-specific aggregator helper | `app_diagnostics_aggregator.go` → `app_reports.go` swarm-manager POST |
| Domain term | "issue" | "fix" (item kind) — UI "Report Issue" remains a user-facing label |
| service.json dep | `app-issue-tracker` | `swarm-manager` (required) |
| Tool registry tags | `["app-issue-tracker", ...]` | `["swarm-manager", "fixes", ...]` |

(The complete mapping — including request/response shapes — is the deliverable.)

## Test Plan
- `cd scenarios/app-monitor && make test` — full suite (depth set by d3).
- Stop `app-issue-tracker` scenario (or confirm not running) before testing — acceptance demands tests pass without it.
- Acceptance gate runs:
  - `rg -n 'app-issue-tracker|appIssueTracker|app_issue_tracker' scenarios/app-monitor` → zero (or only the migration doc itself if d1 places it under app-monitor).
  - `jq '.dependencies.scenarios | keys' scenarios/app-monitor/.vrooli/service.json` excludes `app-issue-tracker`.
  - Migration mapping doc exists at the path chosen in d1.

## Risks
- **R1 — `.vrooli/deployment/deployment-report.json` regeneration semantics:** This file is timestamped 2025-11-16 (well before migration). If it's regenerated by `vrooli scenario deploy` or similar, hand-editing will be overwritten; if it is manually maintained, it must be edited or the file deleted. Resolution depends on d2.
- **R2 — Hidden runtime AIT dependency:** Code search is clean, but a config/env/CLI invocation could still expect AIT to be running. Test phase mitigates by running with AIT stopped.
- **R3 — Mapping doc drift:** If the doc lives only under app-monitor but the cross-scenario sweep needs it as a template, it could be missed. Resolution per d1.
- **R4 — UI naming inertia:** `scenarioIssuesStore.ts`, `ReportIssueDialog.tsx` keep "issue" terminology even though backend kind is `fix`. This is intentional UX — the user-facing word is "issue"; the backend kind is "fix". Documenting this in the mapping doc avoids future confusion.

## Cross-initiative implications (initiative neighborhood)
- `execute/cross-scenario-issue-tracker-cutover-sweep` (downstream, depends on this item) — needs the mapping doc as its template. Doc location (d1) directly affects discoverability for the sweep.
- `chore/app-issue-tracker-deprecation` (downstream, depends on this) — owns the swarm-manager-side doc cleanup and final scenario archival. Out of scope here, flagged for orchestrator.
- `execute/swarm-manager-identity-adoption` (sibling, status=failed) — does not gate this item; current swarm-manager identity gaps are unrelated to the cutover.

## Open Questions / Decisions
Tracked in `workshop/round-NNN.json` — see d1–d4 in round-001 for the open set.

## Out-of-Scope Reminder for the Orchestrator
After this item completes, the swarm-manager-side AIT references in `scenarios/swarm-manager/docs/{internal/INTEROP_AUDIT.md, internal/INTENT.md, reference/operational-targets.md, guides/research-notes.md}` still need cleanup as part of `chore/app-issue-tracker-deprecation`.
