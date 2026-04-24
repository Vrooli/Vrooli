# Sandbox Auto-Approve, Diff Schema, and SwarmManager Profile Reconcile — Implementation Plan

## 1. Purpose

Repair three interlocking defects that cause SwarmManager-dispatched agent runs to land in `NEEDS_REVIEW` even when the agent produced zero diff, wasting human review time and preventing the downstream execution pipeline from closing out work autonomously. The fix spans three scenarios (agent-manager, workspace-sandbox, swarm-manager) because the silent failure is only visible end-to-end — each scenario's code looks locally reasonable.

## 2. Required Reading

Executing agents must load the same context that informed this plan:

```bash
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement interoperability-steer
```

Also re-read this plan file in full before editing — Section 5 (Current Technical Context) contains exact file:line references that identify the edit targets, and Section 8 (Contract Decisions) locks the behavior the tests in Section 9 will assert.

## 3. Greenfield Constraint (hard rule)

**This is greenfield work.** All three affected scenarios are internal to the Vrooli monorepo; there are no external API consumers of the sandbox diff response or of the agent-manager run lifecycle. Do not add:

- Backwards-compatibility shims that accept both the old and new diff response shapes.
- Deprecation paths that keep both `DefaultSandboxConfig==nil` semantics and the new defaulted-config behavior alive in parallel.
- Dead-code re-exports, `// removed` comments, or `_unused` rename stubs for deleted fields.
- Migration scripts to "rewrite" old run rows — runs in `NEEDS_REVIEW` are reviewable by humans today; Section 10 covers bulk-approval via the CLI, not schema migration.

If a helper becomes unused after the fix (e.g. `DefaultSandboxConfig` field on the orchestrator config, `wsDiffStats` client struct), delete it outright. The greenfield rule is repeated in Section 13 (Definition of Done) as an objective check.

## 4. Problem Statement

### 4.1 Observed

- In the SwarmManager Runs tab, "Workshop:" runs show the pending-review clock icon and stay there indefinitely; "Heartbeat:" runs (from prompt-manager) close out as complete with a green check.
- Representative stuck run: `4f1bba3d-357e-4ac8-9ff7-237b74dec532` has `run_mode=SANDBOXED`, `exit_code=0`, `status=NEEDS_REVIEW`, `approval_state=PENDING`, `phase=COLLECTING_RESULTS`.
- The sandbox attached to that run (`d5acbd13-694e-4d18-9e86-050130ab3b0f`) is genuinely empty: `GET /api/v1/sandboxes/{id}/diff` returns `{files:[], totalAdded:0, totalDeleted:0, totalModified:0}`.
- `POST /api/v1/sandboxes/{id}/approve` on that sandbox returns `{success:true, applied:0}` — approval of an empty sandbox works when called directly.
- The run's event stream has no approval-related log entries (no "auto-approved empty sandbox", no "auto-approve-if-empty failed"). The auto-approval path is never reached.
- As of 2026-04-24 the workspace-sandbox API reports 174 active sandboxes; the stuck-review backlog is the same order of magnitude.

### 4.2 Root causes (three layers, each independently broken)

**RC1 — Agent-manager silently skips auto-approval when `run.SandboxConfig` is nil.**
`run_executor.go:1593-1597`:
```go
cfg := e.effectiveSandboxConfig()
if cfg == nil { return false }   // silent, no event
```
`effectiveSandboxConfig()` returns nil whenever `run.SandboxConfig` is nil. `resolveSandboxConfig` (service.go:1444-1456) cascades three nils to get there:
- `o.config.DefaultSandboxConfig` — declared at service.go:495 but **never assigned anywhere in the repo**.
- `profile.SandboxConfig` — SwarmManager's profile has no embedded sandbox config.
- `req.SandboxConfig` — SwarmManager's `SpawnBacklog` only sets `InlineConfig.SandboxConfig` when the caller supplied AcceptanceAllow/Deny globs (swarm-manager service.go:347-362). Workshop runs don't.

So `run.SandboxConfig` is nil → `tryAutoApproval` short-circuits → `handleSuccessfulCompletion` (run_executor.go:1338-1341) stamps `Status=NEEDS_REVIEW`. The `autoApproveIfEmpty` helper at run_executor.go:1651-1687 is correct but unreachable.

**RC2 — Diff response schema mismatch between workspace-sandbox and agent-manager.**
- Server type (workspace-sandbox types.go:331-344) emits `{files, unifiedDiff, totalAdded, totalDeleted, totalModified}`.
- Client type (agent-manager workspace_sandbox.go:521-545) decodes into `{files, unifiedDiff, stats: {filesChanged, ...}}`.
- `stats` is never present in the wire payload, so `diff.Stats.FilesChanged` is always 0 regardless of the actual sandbox contents.
- Today this coincidentally helps: with RC1 fixed, `autoApproveIfEmpty` would call `Approve` whenever `FilesChanged == 0`, which is *always*. Any sandbox with real changes would be silently auto-approved and committed without review.
- The existing unit test (workspace_sandbox_test.go:146-167) hides this by mocking a fake server that emits `stats`. The real binary never has.

**RC3 — SwarmManager agent profile row is desynced from its own code.**
- Code (swarm-manager profile.go:69-70) declares `RequiresSandbox: false, RequiresApproval: true`.
- DB (profile `b5fdee32-353f-4ea7-ad87-ec40557ae32a`, created 2026-04-17, updated 2026-04-23) has `requires_sandbox: true, requires_approval: true`.
- `EnsureProfile` (agent-manager service.go:759-761) returns the existing DB row verbatim and ignores `req.Defaults` unless `UpdateExisting: true`. SwarmManager's `defaultProfileRef()` never sets that flag.
- If the profile really were InPlace as the code intends, `handleSuccessfulCompletion` (run_executor.go:1330-1333) would short-circuit to `Status=Complete` and never enter the approval path — making RC1 irrelevant for this caller. Instead the runs ship as Sandboxed-with-review and hit RC1 head-on.
- Code intent is ambiguous: is SwarmManager's research/workshop agent supposed to run InPlace (no sandbox, direct mutations) or Sandboxed (auto-approve-if-empty, manual review otherwise)? This plan's Section 8 commits to one.

### 4.3 Why heartbeat runs are unaffected

Prompt-manager's heartbeat scheduler configures `RequiresSandbox: false, RequiresApproval: false`. With `RequiresApproval=false`, `handleSuccessfulCompletion` (run_executor.go:1342-1346) sets `Status=Complete` directly and never touches `tryAutoApproval`. Heartbeat's DB profile was presumably created after the current code was in place, so RC3 doesn't bite. The contrast is diagnostic: every path around the RC1 short-circuit completes cleanly; the RC1 path is the trap.

## 5. Scope

### 5.1 In scope

- Agent-manager run-orchestration code: `resolveSandboxConfig`, `tryAutoApproval`, `effectiveSandboxConfig`, `autoApproveIfEmpty`, and the `DefaultSandboxConfig` field on the orchestrator config (delete it; it was never assigned).
- Workspace-sandbox diff wire shape (`types.DiffResult`) and agent-manager's matching decoder (`wsDiffResponse`).
- Swarm-manager profile lifecycle: `EnsureProfile` call site, default profile config, and the decision between InPlace and Sandboxed run modes for workshop/backlog/execution runs.
- Bulk cleanup of the existing `NEEDS_REVIEW` backlog after the fix ships.
- Telemetry that would have made the original bug visible (a warn-level event when auto-approval is skipped due to missing config).

### 5.2 Out of scope

- Partial-accept / acceptance allowlist/denylist semantics (swarm-manager service.go:347-362 path). Not broken; leave behavior untouched.
- Prompt-manager heartbeat path. Works correctly; do not refactor as part of this plan.
- UI changes to the "All Runs" list or the review pane. Screenshot-visible symptom disappears once the runs stop landing in NEEDS_REVIEW.
- Broader profile management UX (editing profiles in the agent-manager UI). `EnsureProfile` semantics change is scoped to the SwarmManager caller.
- Investigation runs path (investigation.go:434) — it's a different caller with its own contract; inspect for regression but do not redesign.

## 6. Current Technical Context

Exact file:line anchors for the edit targets. Line numbers are against the `agi` branch as of 2026-04-24; executing agent should re-grep to confirm before editing.

### Agent-manager (`scenarios/agent-manager/api/internal/`)

| Concern | Path | Line | Current behavior |
|---|---|---|---|
| Silent short-circuit | `orchestration/run_executor.go` | 1593-1597 | `tryAutoApproval` returns false if `cfg == nil` |
| Config resolution | `orchestration/run_executor.go` | 1563-1568 | `effectiveSandboxConfig` reads `e.run.SandboxConfig` |
| Config cascade | `orchestration/service.go` | 1444-1456 | `resolveSandboxConfig` returns nil when all inputs nil |
| Unused default slot | `orchestration/service.go` | 495 | `DefaultSandboxConfig *domain.SandboxConfig` (never assigned) |
| InPlace short-circuit | `orchestration/run_executor.go` | 1330-1334 | Works; preserve |
| RequiresApproval=false path | `orchestration/run_executor.go` | 1342-1346 | Works; preserve |
| Auto-approve-if-empty | `orchestration/run_executor.go` | 1651-1687 | Logic correct; check on `FilesChanged` is correct if RC2 fixed |
| Diff decoder (client) | `adapters/sandbox/workspace_sandbox.go` | 154-181, 521-577 | Expects `stats` object that server never emits |
| Profile resolution | `orchestration/service.go` | 746-811 | `EnsureProfile` ignores `Defaults` when row exists and `UpdateExisting=false` |
| Run-event logger | `orchestration/run_executor.go` | 1494-1505 | `emitSystemEvent`; used to add missing warn event |

### Workspace-sandbox (`scenarios/workspace-sandbox/api/internal/`)

| Concern | Path | Line | Current behavior |
|---|---|---|---|
| Server diff type | `types/types.go` | 331-344 | Emits `totalAdded/totalDeleted/totalModified`, no `stats` |
| Diff builder | `diff/diff.go` | 209-217 | Populates the above totals |
| Diff handler | `handlers/diff.go` | 18-85 | Passes result straight through `JSONSuccess` |

### Swarm-manager (`scenarios/swarm-manager/api/internal/agentmanager/`)

| Concern | Path | Line | Current behavior |
|---|---|---|---|
| Profile defaults | `profile.go` | 53-72 | `RequiresSandbox: false, RequiresApproval: true` |
| Profile build | `profile.go` | 87-103 | Maps ProfileConfig → `domainpb.AgentProfile` |
| Profile ref | `profile.go` | 119-127 | Builds `ProfileRef` with `Defaults` but never sets `UpdateExisting` |
| SpawnBacklog dispatch | `service.go` | 295-377 | Inline SandboxConfig only on explicit allow/deny |
| SpawnInitiative dispatch | `service.go` | ~383+ | Same pattern; confirm during edit |
| SpawnResearch dispatch | `service.go` | ~243+ | Same pattern; confirm during edit |

### Empirical verification (already confirmed during RCA)

```bash
# Live diff shape — note the absence of any "stats" key
curl -s http://localhost:15120/api/v1/sandboxes/<id>/diff | jq 'keys'

# Live profile state
agent-manager profile get b5fdee32-353f-4ea7-ad87-ec40557ae32a --json | jq '.profile | {requires_sandbox, requires_approval, updated_at}'

# Stuck runs
agent-manager run list --status needs_review --limit 10 --json | jq '.runs[] | {id, run_mode, approval_state, tag}'
```

## 7. Target End State

1. A SwarmManager run that produces zero changes ends in `Status=Complete`, `ApprovalState` is either `None` (InPlace) or `Approved` with actor `auto-approve-empty` (Sandboxed) — depending on the run-mode decision in Section 8.1. Either way: no human review needed.
2. A SwarmManager run that produces changes ends in `Status=NEEDS_REVIEW`, `ApprovalState=Pending`, and the diff reflects the actual file count. The run event stream contains a log entry stating why auto-approval did not fire (non-empty sandbox).
3. The workspace-sandbox diff response and the agent-manager decoder agree on shape; the agent-manager `FilesChanged` value equals the real file count at every call site.
4. The `DefaultSandboxConfig` field is removed from agent-manager's orchestrator config; `resolveSandboxConfig` either always returns a valid config or the callers tolerate nil without silent failure — Section 8.2 commits to one.
5. SwarmManager's stored profile row matches the code's intent on every `CreateRun` call. If code changes later, the DB reflects it on next dispatch, not on manual intervention.
6. Existing `NEEDS_REVIEW` runs with empty sandboxes are closed out in bulk via CLI — no DB rewrites, no manual UI clicks per run.
7. A regression test (contract test against a real workspace-sandbox binary or a schema-parity test) exists such that a future schema drift in either direction fails CI before it ships.

## 8. Contract Decisions

These commit the behavior that tests in Section 9 will assert. No ambiguity; if a decision here conflicts with code, change the code.

### 8.1 SwarmManager run mode: Sandboxed with auto-approve-if-empty

SwarmManager's workshop, backlog, and initiative runs **will be Sandboxed**. Research agents do produce file diffs (they create plan files, update docs, write outputs) and we want those diffs reviewable. Empty sandboxes (agents that read-only inspect and report via conversation only) will auto-approve.

- Profile: `RequiresSandbox: true, RequiresApproval: true`.
- Run mode: `SANDBOXED`.
- Acceptance: auto-approve when sandbox is empty; human review when non-empty.
- This choice resolves RC3's ambiguity by aligning code to the current DB state (flipping the code, not the DB).

### 8.2 Agent-manager: `resolveSandboxConfig` returns a non-nil default

`resolveSandboxConfig` will always return a non-nil `*domain.SandboxConfig` when the run mode is Sandboxed. A caller providing no inputs gets a zero-valued struct that `normalizeSandboxConfig` fills in with `Acceptance.Mode = "allowlist"` and all flags at Go zero values (which means `DisableAutoApproveIfEmpty=false`, i.e. auto-approve-if-empty is on).

- Delete the `DefaultSandboxConfig` field from the orchestrator config (service.go:495). It is dead weight; no assignments exist. `investigation.go:434-435` must stop reading it.
- `tryAutoApproval` no longer needs the `if cfg == nil { return false }` guard; if the contract holds, cfg is never nil for sandboxed runs. Keep a defensive check *with* a warn-level event so a future bug is visible, not silent.

### 8.3 Diff response shape: one source of truth

The workspace-sandbox diff response will expose a structured `stats` object and the legacy top-level totals will be removed. Greenfield: change both sides in the same PR.

Server response shape (new):
```json
{
  "sandboxId": "...",
  "files": [...],
  "unifiedDiff": "...",
  "generated": "...",
  "stats": {
    "filesChanged": N,
    "filesAdded": N,
    "filesModified": N,
    "filesDeleted": N,
    "linesAdded": N,
    "linesRemoved": N,
    "totalBytes": N
  },
  "mode": "diff"
}
```

- `filesChanged = filesAdded + filesModified + filesDeleted`.
- Agent-manager's `wsDiffResponse` and `DiffResult` / `DiffStats` stay as they are (they already match this shape).
- Delete `TotalAdded`, `TotalDeleted`, `TotalModified` from workspace-sandbox `types.DiffResult`. Any internal readers (grep before delete) convert to `stats.*`.

### 8.4 `EnsureProfile` semantics for declarative callers

SwarmManager's `defaultProfileRef` will pass `UpdateExisting: true` so that each dispatch treats the code-declared profile as authoritative. The DB row becomes a cache of the last declared state, not a stale override.

- Add `UpdateExisting` (bool) to `apipb.ProfileRef` if not already present; plumb through `EnsureProfileRequest.UpdateExisting`.
- If the field already exists, just set it from swarm-manager.

### 8.5 Telemetry

`tryAutoApproval` defensive path emits a warn log: `"auto-approval skipped: run has no sandbox config (resolve bug — please report)"`. This is belt-and-suspenders: after 8.2 it should never fire, but if it does we'll know.

`autoApproveIfEmpty` non-empty path (`FilesChanged > 0`) emits an info log: `"auto-approval skipped: <N> files changed — review required"`. This replaces the current silent return so a run's event stream explains why it's awaiting review.

## 9. Implementation Strategy

Phased by component to avoid large simultaneous blast radius. Each phase builds and tests to green before the next begins.

### Phase 1 — Diff schema parity (fixes RC2)

Dependencies: none. Do this first so Phase 2's tests can trust `FilesChanged`.

1. In `scenarios/workspace-sandbox/api/internal/types/types.go`, change `DiffResult` to carry a `Stats` field (match the agent-manager `wsDiffStats` shape). Delete `TotalAdded`, `TotalDeleted`, `TotalModified`.
2. In `scenarios/workspace-sandbox/api/internal/diff/diff.go:209-217`, populate `Stats` from the existing `added/deleted/modified` counters plus `LinesAdded/LinesRemoved` (sum from per-file diffs) and a `TotalBytes` sum.
3. `grep -rn 'TotalAdded\|TotalDeleted\|TotalModified' scenarios/workspace-sandbox/` and update every internal reader. Delete unused code.
4. In `scenarios/agent-manager/api/internal/adapters/sandbox/workspace_sandbox_test.go:136-191`, replace the hand-rolled mock server with a contract test that either (a) spins up the real workspace-sandbox binary via testcontainers or (b) reuses a shared fixture file that both server tests and client tests load from. Option (a) is preferred per `seam-discovery-and-enforcement`.
5. Build both scenarios: `cd scenarios/workspace-sandbox/api && go build ./... && go test ./...`, same for agent-manager.
6. `vrooli scenario restart workspace-sandbox && vrooli scenario restart agent-manager`.
7. Verify: `curl -s http://localhost:15120/api/v1/sandboxes/<any>/diff | jq '.stats.filesChanged'` returns a number.

### Phase 2 — Agent-manager auto-approval reliability (fixes RC1)

Dependencies: Phase 1 (tests rely on correct `FilesChanged`).

1. In `scenarios/agent-manager/api/internal/orchestration/service.go`:
   - Delete the `DefaultSandboxConfig *domain.SandboxConfig` field (line 495) and every reader. `grep -rn DefaultSandboxConfig scenarios/agent-manager/` should return no hits after this step.
   - In `resolveSandboxConfig` (line 1444): if after the three cascading assignments `cfg == nil` and the run is sandboxed, allocate `cfg = &domain.SandboxConfig{}`. Then run `normalizeSandboxConfig` as today. The returned value is always non-nil for sandboxed runs.
   - Caller at line 1022 should not need to change; both branches now return non-nil.
2. In `scenarios/agent-manager/api/internal/orchestration/run_executor.go`:
   - `tryAutoApproval` (line 1593): keep the `if cfg == nil` guard but emit a warn event with the text in Section 8.5 before returning false. In production this should never fire.
   - `autoApproveIfEmpty` (line 1664): when `FilesChanged > 0`, emit an info event `fmt.Sprintf("auto-approval skipped: %d files changed — review required", diff.Stats.FilesChanged)` before returning false.
3. In `scenarios/agent-manager/api/internal/orchestration/investigation.go:434-435`: the read of `o.config.DefaultSandboxConfig` disappears with the field. Replace with a local `domain.SandboxConfig{}` literal if that function needs a config, or delete if it doesn't.
4. Unit tests:
   - New table-driven test for `handleSuccessfulCompletion` covering (InPlace), (Sandboxed + RequiresApproval=false), (Sandboxed + empty diff + auto-approve success), (Sandboxed + non-empty diff → NEEDS_REVIEW + info event emitted), (Sandboxed + GetDiff error → NEEDS_REVIEW + warn event), (Sandboxed + Approve error → NEEDS_REVIEW + warn event).
   - Assertion for the previously-silent path: emit a warn event when SandboxConfig is nil in a sandboxed run.
5. Build + test: `cd scenarios/agent-manager/api && go build ./... && go test ./... -timeout 600s`.
6. `vrooli scenario restart agent-manager` and verify health.

### Phase 3 — SwarmManager profile + run mode reconciliation (fixes RC3)

Dependencies: Phases 1 and 2 (tests assume diff-parity and live auto-approval).

1. In `scenarios/swarm-manager/api/internal/agentmanager/profile.go:53-72`, change `DefaultProfileConfig`:
   - `RequiresSandbox: true` (align to decision 8.1).
   - `RequiresApproval: true` (unchanged).
2. In the same file at `defaultProfileRef` (line 119), change the returned `apipb.ProfileRef` to include `UpdateExisting: true`. If the field does not exist on `ProfileRef`, add it to the proto at `packages/proto/schemas/agent-manager/v1/api/` (follow `interoperability-steer` §4–§5 for proto workflow, then `make generate`).
3. `scenarios/agent-manager/api/internal/orchestration/service.go:759-761` (EnsureProfile): already honors `UpdateExisting`; just make sure `req.UpdateExisting` flows from the proto through `EnsureProfileRequest`. Grep to confirm.
4. Unit test: new test asserting that `SpawnBacklog` with a pre-existing DB profile and updated code-side defaults results in the DB row being updated to match the code values on the next call.
5. Build + test both scenarios.
6. Restart both: `vrooli scenario restart agent-manager && vrooli scenario restart swarm-manager`.
7. Verify profile: `agent-manager profile get <id> --json | jq '.profile | {requires_sandbox, requires_approval, updated_at}'` — `updated_at` should advance to the restart time and flags should match `DefaultProfileConfig`.

### Phase 4 — Backlog cleanup

Dependencies: Phases 1–3 in production, verified by at least one fresh SwarmManager run completing as Complete.

1. Enumerate currently-stuck runs: `agent-manager run list --status needs_review --limit 500 --json` → filter to tag prefix `swarm-manager:` → for each, fetch the sandbox diff and confirm it is empty.
2. For empty-diff runs: use the existing `agent-manager` CLI to approve or mark complete. Prefer `agent-manager run approve <id>` (verify the CLI has this; if not, add it as a thin wrapper over the HTTP approve endpoint — do not script raw `curl` per memory `feedback_skills_use_cli_never_api`).
3. Keep a short script at `scripts/approve-empty-swarm-runs.sh` for auditability; remove after the backlog is drained.
4. Spot-check the Runs tab in the SwarmManager UI; no empty runs should show the clock icon.

### Phase 5 — Final cleanup & verification

Per `plan-skill-discovery` §3b, every scenario-touching plan ends with:

1. Run `golangci-lint run` in each modified scenario and fix every warning in modified files, including pre-existing ones. Do not rationalize pre-existing issues as out of scope — per memory `feedback_planning_guidelines`, fix them.
2. Run `gofumpt -w` on every modified Go file.
3. Run full test suites: `cd scenarios/<name>/api && go test ./... -timeout 600s` for each of agent-manager, workspace-sandbox, swarm-manager.
4. `vrooli scenario restart agent-manager && vrooli scenario restart workspace-sandbox && vrooli scenario restart swarm-manager`.
5. Health verify each: `curl -sf http://localhost:<port>/health || echo "UNHEALTHY: <name>"`. If anything returns UNHEALTHY, investigate root cause; do not merge.
6. Spawn a fresh SwarmManager workshop run and confirm:
   - It lands Sandboxed (check via `agent-manager run get <id> --json | jq .run.run_mode`).
   - If the agent produces no diff → run ends Complete + Approved (actor `auto-approve-empty`) and the event log shows `"auto-approved empty sandbox (no changes detected)"`.
   - If the agent produces a diff → run ends NEEDS_REVIEW and the event log shows the new info message `"auto-approval skipped: N files changed — review required"`.

## 10. Testing Plan

Per user memory `feedback_testing_over_manual`, favor automated tests over manual checklists. Every code change below has a corresponding automated gate.

### 10.1 Unit tests (Go)

- `scenarios/workspace-sandbox/api/internal/diff/diff_test.go`: extend existing tests to assert the new `Stats.FilesChanged/FilesAdded/FilesModified/FilesDeleted` values against canned upper/lower-dir fixtures covering add, modify, delete, and no-change scenarios.
- `scenarios/agent-manager/api/internal/orchestration/run_executor_test.go`: table-driven test for `handleSuccessfulCompletion`/`tryAutoApproval` across the six cases in Phase 2 step 4. Assert both terminal state *and* emitted events (use an in-memory event sink).
- `scenarios/agent-manager/api/internal/orchestration/service_test.go`: new test for `resolveSandboxConfig` confirming it returns a non-nil, normalized config with `Acceptance.Mode="allowlist"` when all inputs are nil.
- `scenarios/agent-manager/api/internal/orchestration/service_test.go`: new test for `EnsureProfile` covering `UpdateExisting: true` updating an existing row, `UpdateExisting: false` ignoring defaults.
- `scenarios/swarm-manager/api/internal/agentmanager/service_test.go`: new test asserting `defaultProfileRef()` returns a `ProfileRef` with `UpdateExisting=true` and the profile defaults matching `DefaultProfileConfig()`.

### 10.2 Contract test (agent-manager ↔ workspace-sandbox)

Replace the fake-server unit test at `scenarios/agent-manager/api/internal/adapters/sandbox/workspace_sandbox_test.go:136-191` with a real-binary contract test. Pattern:
- Start the workspace-sandbox binary via `testcontainers-go` (or reuse an existing scenario-integration pattern if one already exists — grep before adding a new dependency).
- Create a sandbox, write known files into the upper dir, call `GET /diff` via the real adapter, assert that every field in `DiffResult.Stats` matches expectations.
- Also assert the empty-sandbox case: creating a sandbox and immediately calling diff yields `FilesChanged=0` and `len(Files)=0`.

This test is the regression gate for future schema drift in either direction. If a future change removes `stats` from the server or renames a field, this test fails before the change ships.

### 10.3 Integration test (full SwarmManager → agent-manager → workspace-sandbox loop)

Add a scenario-level test (fits under SwarmManager's `vrooli scenario test` suite) that:
1. Dispatches a SpawnBacklog call with a no-op prompt (agent reads one file and exits).
2. Polls the run until terminal.
3. Asserts `status=Complete`, `approval_state=Approved`, `actor=auto-approve-empty`, and a matching event in the run's event log.

This is the end-to-end gate that would have caught the original bug.

### 10.4 Lint / format

- `golangci-lint run ./...` in each modified scenario — zero warnings on modified files (pre-existing in modified files included, per memory `feedback_planning_guidelines`).
- `gofumpt -l` — no output.

## 11. Rollout / Validation Checklist

In order. Check each before moving on.

- [ ] Phase 1 builds and tests green in both scenarios; `vrooli scenario restart workspace-sandbox && vrooli scenario restart agent-manager` returns healthy.
- [ ] Live curl to `/diff` shows the new `stats` object with correct counts for a sandbox known to have changes.
- [ ] Phase 2 tests green; `run_executor_test.go` table test passes all six cases.
- [ ] `grep -rn DefaultSandboxConfig scenarios/agent-manager/` returns zero results.
- [ ] Phase 3 tests green; `agent-manager profile get <swarm-manager-profile-id>` shows flags matching the new `DefaultProfileConfig` values and `updated_at` advanced.
- [ ] Fresh SwarmManager workshop run with a no-op prompt completes as `Complete + Approved (auto-approve-empty)` within one typical execution window.
- [ ] Fresh SwarmManager workshop run with a file-modifying prompt completes as `NEEDS_REVIEW + Pending` and the event log shows the new info-level skip message with the correct file count.
- [ ] Phase 4 bulk cleanup drains the existing backlog; `agent-manager run list --status needs_review --tag-prefix swarm-manager: --json | jq '.runs | length'` returns a small, human-reviewable number (non-empty diffs only).
- [ ] Phase 5 full lint/format/test/restart/health sweep is green across all three scenarios.
- [ ] The contract test from 10.2 runs in CI (or in `vrooli scenario test agent-manager` / `workspace-sandbox`) without requiring manual setup.

## 12. Risks + Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Phase 1 breaks another consumer of `totalAdded/totalDeleted/totalModified` that grep missed | Medium | Medium — breaks the UI diff view or a CLI | Grep the whole monorepo (not just workspace-sandbox) for those three field names before deleting. Touch UIs or other Go consumers in the same PR. |
| `EnsureProfile` with `UpdateExisting: true` clobbers a profile an admin hand-edited via the UI | Low | Medium — admin tweaks get reverted on next dispatch | Document the semantic change in the PR body. If admins need to edit this profile, they should edit the code defaults, not the DB row. Alternative: `UpdateExisting: true` only overlays fields declared in `Defaults`, leaves others alone — confirm this during implementation. |
| Flipping SwarmManager to `RequiresSandbox: true` changes the execution surface for existing backlog items | High | Low — agents will now run in a sandbox whose scope is `scenarios/agent-manager/api`, which may be wrong for some tasks | Section 8.1 matches the current DB state, which is what production has been doing for weeks — this is a code-catches-up-to-prod change, not a behavioral change. Verify by running a representative set of existing skills in a test environment. |
| Contract test (10.2) requires a binary build at test time — slows CI | Medium | Low — longer feedback loop | Build once per CI job, cache between tests. If too slow, shrink to a "shape parity" assertion that decodes a captured real response against the client struct and asserts no zero-valued fields that should be populated. |
| Phase 4 bulk approve accidentally approves runs that *do* have diffs (script bug) | Low | High — unreviewed changes applied to canonical repo | The script must filter by `files == []` from the live diff endpoint, not just by run status. Dry-run mode first; human-visible confirmation on count before mutation. Better: use the agent-manager `run approve` CLI which already invokes the sandbox Approve endpoint — it will only approve what's in the sandbox. |
| Event-log emission for non-empty sandboxes spams long-running runs | Very low | Very low — one extra log line per run | Non-concern; it's one line per terminal decision per run. |

## 13. Non-goals / Prohibited Patterns

- **No compatibility shim accepting both `totalAdded/...` and `stats.filesChanged/...`.** Greenfield: change both sides in one PR. If review is blocked because the PR is cross-scenario, split into "add stats + keep totals" → "remove totals" chained PRs behind a feature flag only if reviewer explicitly asks — and remove the flag on merge to main.
- **No dual-path in `tryAutoApproval`** keeping the nil-cfg silent-return as a "legacy fallback". Section 8.2 commits to always-non-nil for sandboxed runs; the nil path becomes a warn-event-emitting defensive check.
- **No fake-server unit tests for the diff contract** once the contract test in 10.2 exists. Delete `TestWorkspaceSandboxProvider_GetDiff` as currently written; the real-binary test supersedes it.
- **No manual UI clicking** to drain the Phase 4 backlog. CLI only.
- **No memory/DB migration for stuck runs** — approve them via the live API, which is exactly what a human review click does. Do not write SQL that flips `status` directly.
- **No `_unused` / `_removed` / commented-out stubs** for the deleted `DefaultSandboxConfig` field or the deleted `TotalAdded/TotalDeleted/TotalModified` fields. Delete outright.

## 14. Definition of Done

Objective, falsifiable checks. A failing one blocks merge.

1. A fresh SwarmManager workshop run that performs only reads (no file writes) ends `status=Complete, approval_state=Approved, actor=auto-approve-empty` with the expected info log in its event stream.
2. A fresh SwarmManager workshop run that writes a file ends `status=NEEDS_REVIEW, approval_state=Pending` with the new `"auto-approval skipped: N files changed"` info event showing `N == <actual file count>`.
3. `grep -rn DefaultSandboxConfig scenarios/agent-manager/` → zero results.
4. `grep -rn 'TotalAdded\|TotalDeleted\|TotalModified' scenarios/workspace-sandbox/ scenarios/agent-manager/` → zero results (unless matched by `Stats.LinesAdded`/etc., which are different field names).
5. Workspace-sandbox `/diff` response contains a `stats` object with `filesChanged, filesAdded, filesModified, filesDeleted, linesAdded, linesRemoved, totalBytes`.
6. `agent-manager profile get <swarm-manager-profile-id> --json` shows `requires_sandbox: true, requires_approval: true` and `updated_at` matches the most recent SwarmManager restart.
7. The contract test from 10.2 runs and passes as part of `vrooli scenario test agent-manager` (or `workspace-sandbox` — pick one home).
8. `agent-manager run list --status needs_review --tag-prefix swarm-manager:` returns only runs whose sandbox diff is genuinely non-empty (verified by spot-check of three).
9. `golangci-lint run` on each of agent-manager, workspace-sandbox, swarm-manager returns zero warnings in modified files.
10. `gofumpt -l` on modified files returns empty.
11. All three scenarios report `healthy` via `vrooli scenario status` after final restart.
12. No backwards-compatibility shims, no commented-out code, no `_unused` renames introduced — spot-check the diff against the prohibitions in Section 13.
