# Sandbox Launch & Auto-Approve Fixes — Implementation Plan

**Status:** Ready to execute
**Created:** 2026-04-28
**Branch:** `agi`
**Constraint:** **Greenfield only.** No compatibility shims, no legacy field
fallbacks, no `// removed for X` placeholders. The auditability contract is
already locked; we are fixing it where it is broken, not adding alternate
modes.

---

## 1. Purpose

The recent `Sandboxing auto-approval p1..p5` series (commits `3e8b004704`,
`e6b21b8da4`, `6a345c7eb2`, `26af7314ab`, etc.) made protected-sandbox the
default execution mode for agent-manager runs spawned through swarm-manager.
That cutover surfaced four stacked defects that together make the swarm-manager
feedback / research / execution agents non-functional from the user's
perspective: the agent never actually runs, the failure is not categorized,
and the run is parked in `needs_review` instead of `failed`.

This plan fixes all four root causes in one coordinated change so that:

1. Sandbox-routed agents actually execute the runner inside the bwrap
   namespace.
2. Bwrap exit codes and stderr propagate cleanly back to agent-manager events.
3. Most runs auto-accept on success (sandbox is for auditability, per the
   user's stated direction); manual review is opt-in, not the default.
4. Silent launch failures cannot masquerade as `complete + needs_review`.

A future agent should be able to execute this plan from cold with no chat
context.

---

## 2. Required Reading (run these first)

```bash
# Plan-authoring foundations
prompt-manager skill read implementation-plan-authoring
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement

# Topical skills directly relevant to this plan
prompt-manager skill read signal-and-feedback-surface-design
prompt-manager skill read failure-topography-and-graceful-degradation
prompt-manager skill read error-semantics-recovery-path-design
prompt-manager skill read documentation-health
prompt-manager skill read interoperability-steer
prompt-manager skill read scientific-debugging
```

Then read the prior context this plan continues:

- `path:docs/plans/sandbox-auto-approve-and-profile-reconcile-plan.md` (2026-04-24
  predecessor — defines `resolveSandboxConfig` non-nil contract,
  `defaultProfileRef.UpdateExisting=true`, diff-stats wire shape).
- `path:scenarios/agent-manager/docs/AUDITABILITY_CONTRACT.md` (locked contract;
  do not violate).
- `path:scenarios/workspace-sandbox/docs/AUDITABILITY_CONTRACT.md` (locked contract).
- `path:scenarios/agent-manager/api/internal/orchestration/run_executor.go`
  (around `applyAtRunEnd`, `useSandboxedWorkspace`, the env-var injection at
  line ~1254).
- `path:scenarios/agent-manager/api/internal/adapters/sandbox/sandbox_launcher.go`
  (entire file).
- `path:scenarios/workspace-sandbox/api/internal/driver/bwrap.go` (especially
  `buildBwrapArgs`, `BuildExecCommand`).
- `path:scenarios/workspace-sandbox/api/internal/handlers/process.go`
  (`StreamProcessLogs`, the OnExit closure starting around line 389).
- `path:scenarios/swarm-manager/api/internal/agentmanager/profile.go`.

---

## 3. Problem Statement (with command-level evidence)

### 3.1 Symptom (reported)

User dispatched a swarm-manager initiative-feedback agent against
`git-control-tower-ai-provenance`. The run completed in 134 ms with 9 events,
status `needs_review`, exit_code 0, and **no assistant output at all**. The
agent did nothing.

### 3.2 Reproduction evidence

Run id: `d6df2636-9d56-4621-b51b-5584a452f655`.

```
$ agent-manager run get d6df2636-9d56-4621-b51b-5584a452f655 --json
  → status: RUN_STATUS_NEEDS_REVIEW
  → exit_code: 0
  → started_at .. ended_at: 134 ms
  → resolved_config.sandbox_config.manual_review: true
  → resolved_config.sandbox_config.mode: SANDBOX_MODE_PROTECTED

$ agent-manager run events d6df2636-... --json
  → 1× RUN_EVENT_TYPE_MESSAGE (the user prompt)
  → 4× phase logs
  → 2× status events (starting→running→complete in 100ms)
  → 1× "apply deferred: manualReview=true (operator approval required)"
  → 0× assistant message events

$ ls -la scenarios/agent-manager/data/runs/d6df2636-.../
  → transcript.ndjson: 0 bytes
  → stderr.log:        0 bytes
  → meta.json: working_dir = /home/.../workspace-sandbox/<sb>/merged

$ cat /home/matthalloran8/.local/share/workspace-sandbox/<sb>/logs/<pid>.stderr.log
  → bwrap: Can't chdir to /home/.../workspace-sandbox/<sb>/merged: No such file or directory
  → === Process Exited: code 1 signal 0 oom false ===
```

The same `bwrap chdir` error appears in **every** sandbox launched by
agent-manager since the cutover. Behavior split:

- Fast-path: `complete` in ~100 ms, exit_code 0 (race won by the client).
- Slow-path: `failed` after the 3600 s timeout (race lost; SSE exit event
  never delivered).

Both paths are the same underlying bug.

### 3.3 Root causes (four stacked defects)

**RC-1 — Working-dir host/namespace mismatch (critical).**
`run_executor.go:681-691` sets `e.workDir = sandbox.GetWorkspacePath()`,
which returns the **host** merged path
(`/home/.../workspace-sandbox/<id>/merged`). That value flows into
`SandboxLauncher.Launch` (`sandbox_launcher.go:99`) as `WorkingDir`, is
forwarded to workspace-sandbox `POST /api/v1/sandboxes/{id}/processes`,
and bwrap is invoked with `--chdir <host_path>`. But inside the bwrap mount
namespace the merged dir is bind-mounted at `/workspace`
(`workspace-sandbox/.../driver/bwrap.go:334`). The host path does not exist
inside the namespace, so bwrap fails immediately with
`Can't chdir to ...: No such file or directory` and exits 1 before claude
ever launches.

The unit test at `sandbox_launcher_test.go:266` hardcodes `WorkingDir:
"/workspace"` — so tests pass; production passes the wrong value. Same
problem applies to the `VROOLI_SANDBOX_MERGED` env var injected at
`run_executor.go:1254`, which leaks the host path into the namespace.

**RC-2 — SSE exit-event race drops bwrap's non-zero exit (high).**
`workspace-sandbox/.../handlers/process.go:737-759` (`StreamProcessLogs`)
calls `StreamLog` (which blocks until logs flush), THEN queries
`ProcessTracker.GetExitInfo`, THEN emits `event: exit`. The OnExit reaper
that records exit info runs in a separate goroutine spawned by
`spawnExitReaper` (`bwrap.go:711`). When bwrap fails in <100 ms, the log
writer closes before the reaper has called `RecordExit`, so `GetExitInfo`
returns nil and **no exit event is ever sent**.

On the client side (`sandbox_launcher.go`), `sandboxLaunchedProcess.Wait()`
returns nil when no exit event is received — so
`claude_code.go:280-321` lands in the success branch with
`result.Success=true, ExitCode=0`. The captured stderr is buffered in
`errorOutput` (`claude_code.go:200-206`) but only emitted when `err != nil`,
so the bwrap diagnostic is silently dropped.

**RC-3 — `ManualReview=true` default forces NEEDS_REVIEW even on silent
failures (UX/contract).**
`path:scenarios/swarm-manager/.../agentmanager/profile.go:82` hardcodes
`ManualReview: true` for every swarm-manager profile. Combined with
`defaultProfileRef.UpdateExisting=true` (which overwrites the DB row on
every dispatch), every swarm-manager run lands in `needs_review` regardless
of outcome — including silently-failed ones.

This contradicts the user's stated direction: sandbox is for **auditability,
not protection**, and most runs should auto-accept. The doc comment at
`profile.go:60-64` justifies the current default with stale reasoning
("research/workshop diffs should be human-reviewable") that predates the
auditability cutover.

**RC-4 — No "ran-too-fast / no-output" categorizer (high).**
The signal pattern *protected sandbox + zero
`RUN_EVENT_TYPE_MESSAGE` events + runtime <2 s + exit 0* unambiguously
means "the launch itself failed and the runner never produced output."
Today there is no heuristic that catches this: the orchestrator rolls
straight into `collecting_results → complete`, then `manualReview` flips
it to `needs_review`. Operator sees "ran fine, awaiting review" instead
of "launch failed; here is the bwrap stderr."

---

## 4. Scope

### In scope
- Translate `WorkingDir` and host-path env vars at the SandboxLauncher
  boundary (RC-1).
- Make workspace-sandbox `StreamProcessLogs` synchronously await the
  recorded exit info before closing the SSE stream, AND make
  agent-manager treat "stream ended without exit event" as a failure
  (RC-2, belt-and-suspenders).
- Surface non-empty captured stderr on the success path when present
  (RC-2).
- Flip `ManualReview` default on the swarm-manager profile to `false`,
  update doc comment (RC-3).
- Add a post-run "launch-likely-failed" categorizer in the run executor
  with a clear, structured RUN_FAILED event (RC-4).
- Tests covering each fix at the right seam (unit + integration).
- Documentation updates (`SEAMS.md`, `ERROR-SEMANTICS.md`,
  `INTEROP_AUDIT.md` for both agent-manager and workspace-sandbox).
- Clean up the existing 17+ leaked `needs_review` rows for the affected
  swarm-manager runs (one-shot script, separate from the binary fix).

### Out of scope
- Rewriting the workspace-sandbox lifecycle reconciler (the 386 leaked
  sandbox dirs are a related but separate problem; flagged as a follow-up).
- Adding a per-skill manual-review override system. Today's contract is
  "default false, opt-in true via SandboxConfig"; per-skill knobs are a
  later refinement.
- Changing the proto schema for `SandboxConfig` or `RunStatus`.
- Refactoring `claude_code.go` beyond what RC-2 surfacing requires.

### Explicitly prohibited (per user constraint)
- No legacy/compat shims. If a behavior was wrong, change it; do not
  preserve the old shape behind a flag.
- No `// removed for X` placeholder comments. Delete dead code.
- No backward-compat helpers like "fall back to host path if /workspace
  doesn't exist."

---

## 5. Current Technical Context (key files & components)

### 5.1 agent-manager (`path:scenarios/agent-manager/api/`)

| Path | Role |
|---|---|
| `path:internal/orchestration/run_executor.go` | Owns `e.workDir`, `e.sandboxID`, the env-var injection at line ~1254, and `applyAtRunEnd` at line ~1698 (manualReview defer). |
| `path:internal/orchestration/service.go` | Calls `GetWorkspacePath` at lines 1127, 2004, 2374. |
| `path:internal/adapters/sandbox/sandbox_launcher.go` | `Launch` posts `workingDir` to `/processes`. `runStream` parses SSE. `finalizeWaitErr` decides `waitErr`. |
| `path:internal/adapters/sandbox/workspace_sandbox.go` | `GetWorkspacePath` returns host path. |
| `path:internal/adapters/runner/claude_code.go` | Lines 200-321: stderr capture and Wait-error type-switch. |
| `path:internal/adapters/runner/exit_code.go` | `extractExitCode` uniform error decoder. |
| `path:internal/domain/types.go` | `SandboxConfig.ManualReview` (line 298). |

### 5.2 workspace-sandbox (`path:scenarios/workspace-sandbox/api/`)

| Path | Role |
|---|---|
| `path:internal/driver/bwrap.go` | `buildBwrapArgs` (line 303), `BuildExecCommand` (line 624), `StartProcess` (line 674), `spawnExitReaper`. The merged dir is bind-mounted at `/workspace` at line 334. |
| `path:internal/handlers/process.go` | `StartProcess` handler (OnExit closure ~line 389), `StreamProcessLogs` (line 711). |
| `path:internal/process/tracker.go` | `ExitInfo` shape (line 40), `RecordExit`, `GetExitInfo`. |

### 5.3 swarm-manager (`path:scenarios/swarm-manager/api/`)

| Path | Role |
|---|---|
| `path:internal/agentmanager/profile.go` | `DefaultProfileConfig` line 65 — sets `ManualReview: true` at line 82. |

### 5.4 Run-time evidence locations

| Path | What's there |
|---|---|
| `path:scenarios/agent-manager/data/runs/<run_id>/` | `meta.json`, `transcript.ndjson`, `stderr.log`, `cursor.json`. |
| `/home/matthalloran8/.local/share/workspace-sandbox/<sb>/logs/<pid>.stderr.log` | The actual bwrap stderr we are missing in run events. |

---

## 6. Target End State

After this plan, dispatching a swarm-manager initiative-feedback agent
yields a run that:

1. **Actually executes claude inside bwrap.** The runner produces ≥1
   `RUN_EVENT_TYPE_MESSAGE` event with assistant content, the transcript
   file is non-empty, and a `mutation_list` JSON block lands in the
   feedback round file as expected.
2. **On clean success, auto-accepts.** The run reaches `RUN_STATUS_COMPLETE`
   with `approval_state = APPROVED` (via `applyAtRunEnd`), the sandbox is
   applied, and the run does NOT land in `needs_review` unless the user
   explicitly set `manualReview=true` per task.
3. **On bwrap or runner failure, reports `RUN_STATUS_FAILED`** with:
   - A `RUN_EVENT_TYPE_ERROR` carrying `code=LAUNCH_FAILED` (RC-4) or
     the actual non-zero exit code (RC-2).
   - A `RUN_EVENT_TYPE_LOG` (level=warn) containing the captured bwrap
     stderr.
   - `exit_code` matching the bwrap exit (1, not 0).
4. **Cannot reach `complete` with zero assistant messages.** The
   categorizer (RC-4) demotes that pattern to `failed` with code
   `LAUNCH_FAILED_NO_OUTPUT`.

A regression test gates each of these properties.

---

## 7. Implementation Strategy (phased)

The phases are dependency-ordered. Each phase is independently mergeable
and leaves the system in a working state. Run `vrooli scenario restart
agent-manager` and `vrooli scenario restart workspace-sandbox` after each
phase. Always restart agent-manager AFTER workspace-sandbox so the
launcher reconnects to the new server.

### Phase A — RC-1: WorkingDir + env-var translation at the SandboxLauncher boundary

**Files:**
- `path:scenarios/agent-manager/api/internal/adapters/sandbox/sandbox_launcher.go`
- `path:scenarios/agent-manager/api/internal/adapters/sandbox/sandbox_launcher_test.go`
- `path:scenarios/agent-manager/api/internal/orchestration/run_executor.go`

**Steps:**

1. Introduce a `const SandboxNamespacePath = "/workspace"` in
   `sandbox_launcher.go` with a comment pointing at
   `workspace-sandbox/api/internal/driver/bwrap.go:334` so future
   agents see why the constant must match.
2. In `SandboxLauncher.Launch`, **rewrite** `req.WorkingDir` and any
   host-path env values before the POST:
   - If `req.WorkingDir` matches `<sandbox_root>/merged` (exact prefix
     match against the resolved sandbox path), replace with
     `SandboxNamespacePath`.
   - Otherwise, if `req.WorkingDir == ""`, use `SandboxNamespacePath`.
   - If `req.WorkingDir` is some other absolute host path that we can't
     translate, return `LaunchBlocked{Code: "workdir_outside_sandbox"}`
     — this is a contract violation, not a fallback case.
   - Apply the same translation to the env map: any value that is
     exactly the host merged path becomes `SandboxNamespacePath`.
   - Add tests that cover all three branches.
3. Add a small helper `translateHostPathToNamespace(value, hostMerged
   string) string` and unit-test it directly.
4. Update `run_executor.go` line ~1254
   (`"VROOLI_SANDBOX_MERGED": e.workDir`) — keep the env var name (it
   remains the agent's "where the workspace is" pointer), but document
   that it carries the **host** path; the SandboxLauncher will translate
   it. The agent inside the namespace sees `/workspace`.
5. Add a regression test in `sandbox_launcher_test.go` that asserts
   the production code path (executor + launcher together) ends up
   POSTing `workingDir: "/workspace"`, not the host path. Use a
   fake-server harness that captures the request body.

**Acceptance:**
- `go test ./internal/adapters/sandbox/...` green.
- Manual: dispatch a swarm-manager feedback run, then
  `cat /home/.../workspace-sandbox/<sb>/logs/<pid>.stderr.log`
  must NOT contain `bwrap: Can't chdir`.

### Phase B — RC-2 server-side: deterministic exit-event delivery in workspace-sandbox

**Files:**
- `path:scenarios/workspace-sandbox/api/internal/handlers/process.go`
- `path:scenarios/workspace-sandbox/api/internal/process/tracker.go`
- `path:scenarios/workspace-sandbox/api/internal/handlers/process_test.go` (new
  or extended)

**Steps:**

1. Add `ProcessTracker.WaitForExit(sandboxID, pid, ctx) (*ExitInfo,
   error)` that blocks on a per-process exit channel (closed by
   `RecordExit`). The channel is created at `Track()` time and closed
   exactly once when `RecordExit` runs.
2. In `StreamProcessLogs` (`process.go:711`), after `StreamLog` returns,
   call `WaitForExit` with a short bounded timeout (e.g. 5s) instead of
   the current "best-effort `GetExitInfo` then emit event: exit if
   present" pattern.
   - If `WaitForExit` returns the ExitInfo → emit `event: exit` with
     the JSON payload.
   - If it times out → emit `event: error` with
     `data: exit info unavailable after stream close`. Client treats
     this as exit 1 (covered in Phase C).
3. Add a unit test that races a fast-failing process against
   `StreamProcessLogs` and asserts the SSE stream always ends with an
   `event: exit` carrying a non-zero `exitCode` for a process that
   actually exited non-zero — even when the process exited before the
   SSE subscriber attached.

**Acceptance:**
- `go test ./internal/handlers/...` and
  `go test ./internal/process/...` green.
- Manual smoke: launch a process via
  `agent-manager run create --task-id ... --profile-id ...` with
  bwrap intentionally broken (e.g. by `--chdir /nonexistent` in a
  test) and verify the run reaches `failed` with exit_code 1.

### Phase C — RC-2 client-side: never silently treat "no exit event" as success

**Files:**
- `path:scenarios/agent-manager/api/internal/adapters/sandbox/sandbox_launcher.go`
- `path:scenarios/agent-manager/api/internal/adapters/runner/claude_code.go`
- `sandbox_launcher_test.go`, `claude_code_test.go`

**Steps:**

1. In `sandboxLaunchedProcess.finalizeWaitErr` (around line 394), when
   `p.exitInfo == nil` and `p.killed.Load() == false`, set
   `p.waitErr = errors.New("sandbox process ended without exit info")`
   and emit a `LaunchedProcess`-level marker. Replace the comment
   "Treat as success unless we have other state" — that policy is
   wrong; under the new contract, missing exit info is a failure.
2. Define `var ErrSandboxNoExitInfo = errors.New(...)` so callers can
   `errors.Is` it.
3. In `claude_code.go` lines ~280-321, in the success branch (exit 0,
   no rate-limit), if `errorOutput.Len() > 0`, emit a warn-level log
   event carrying the captured stderr **before** returning. This is
   independent of Phase D's categorizer; it ensures bwrap stderr
   reaches the run events surface even when the run is technically
   successful.
4. Test: race-fail in the launcher → claude_code returns
   `Success=false, ExitCode=-1, ErrorMessage="sandbox process ended
   without exit info"`. Run executor maps this to RUN_STATUS_FAILED.

**Acceptance:**
- New test `TestSandboxLauncher_NoExitInfo_ReportsFailure` asserts the
  exact error.
- New test `TestClaudeCode_StderrSurfacedOnSuccess` asserts that a
  successful run with non-empty stderr emits a warn log event.

### Phase D — RC-4: "launch-likely-failed" categorizer in the run executor

**Files:**
- `path:scenarios/agent-manager/api/internal/orchestration/run_executor.go`
- `path:scenarios/agent-manager/api/internal/orchestration/run_executor_test.go`
- `path:scenarios/agent-manager/api/internal/domain/errors.go` (or wherever
  the RUN_FAILED error codes live — verify by `rg "RunErrorCode\b"`).

**Steps:**

1. Add a new structured error code `RUN_ERROR_CODE_LAUNCH_FAILED` to the
   error code enum (proto-side and domain-side). Pick a name consistent
   with existing codes — verify by reading the enum first.
2. After the runner returns a successful result but BEFORE
   `applyAtRunEnd`, run a single-pass post-execution validator with
   these rules:
   - If `cfg.Mode == SandboxModeProtected` AND `result.Success` AND
     duration < 2s AND zero `RUN_EVENT_TYPE_MESSAGE` events were
     emitted → demote to failed with `RUN_ERROR_CODE_LAUNCH_FAILED`
     and a structured error message that includes the captured
     stderr (truncated to 4 KB).
3. Surface the validator as `validateRunOutcome(...)` so it is
   independently testable. Emit a single warn log event explaining the
   demotion so operators can see the heuristic fired.
4. Tests:
   - `TestValidateRunOutcome_DemotesSilentLaunchFailure` — protected
     sandbox + 0 messages + 100 ms + exit 0 → demoted.
   - `TestValidateRunOutcome_KeepsHonestSuccess` — protected sandbox + 1
     message + 30 s + exit 0 → preserved.
   - `TestValidateRunOutcome_IgnoresInPlace` — in-place mode + 0
     messages + 100 ms + exit 0 → preserved (different policy; no
     bwrap path to fail).

**Error semantics (per error-semantics-recovery-path-design):**

| Category | Code | Trigger | Recovery path |
|---|---|---|---|
| Connectivity | `RUN_ERROR_CODE_LAUNCH_FAILED` | bwrap chdir, missing executable, namespace setup failure | Operator inspects the captured stderr → fixes config → retries. Not auto-retried. |
| Internal | `RUN_ERROR_CODE_RUNNER_TIMEOUT` (existing) | Runner exceeded timeout | Operator extends timeout or splits work. |
| Connectivity | new `RUN_ERROR_CODE_SANDBOX_NO_EXIT_INFO` | SSE stream closed without exit event despite Phase B fix | Bug indicator. Logs alert; investigate workspace-sandbox health. |

Document this table in
`path:scenarios/agent-manager/docs/internal/ERROR-SEMANTICS.md`.

### Phase E — RC-3: remove `ManualReview` from the swarm-manager profile entirely

**Locked contract (operator decision 2026-04-28):** swarm-manager runs
**always** auto-accept on success. There is no per-skill, per-task, or
per-anything override that flips `ManualReview` to true within
swarm-manager. The flag is removed from swarm-manager's local
`ProfileConfig` struct so no future code path can re-introduce it. If a
later workflow needs operator-gated apply, that's a separate plan and a
separate scenario.

**Files:**
- `path:scenarios/swarm-manager/api/internal/agentmanager/profile.go`
- `path:scenarios/swarm-manager/api/internal/agentmanager/service_test.go`
- Any other swarm-manager file the audit (step 4 below) finds.

**Steps:**

1. In `profile.go`:
   - Delete the `ManualReview bool` field from `ProfileConfig` (line 34).
   - Delete the doc comment block at lines 18-24 referring to
     `RequiresApproval` legacy and `ManualReview` forwarding; replace
     with: *"swarm-manager runs always auto-accept on success.
     Sandbox is the auditability layer — per-run file diffs are
     preserved in workspace-sandbox regardless. There is no
     manual-review knob; runs are either successful (auto-applied)
     or failed (sandbox preserved for inspection)."*
   - Delete the `ManualReview` rationale block at lines 58-64.
   - Delete the `ManualReview: true` line at 82.
   - In `buildProfile` (line 99), remove the
     `SandboxConfig: &domainpb.SandboxConfig{ManualReview: cfg.ManualReview}`
     assignment at line 114. Replace with whatever the proto default
     produces — i.e., omit `SandboxConfig` from the `AgentProfile`
     entirely, OR set it to `&domainpb.SandboxConfig{}` if a non-nil
     value is required by the agent-manager API contract. Verify which
     by reading `agent-manager/api/internal/orchestration/service.go`
     resolveSandboxConfig and checking what nil vs empty does in
     proto unmarshalling.
2. In `service_test.go`:
   - Line 35-36: invert. Test now asserts the resolved config has
     `ManualReview=false` (or that `SandboxConfig` is nil/empty),
     with rationale comment: *"swarm-manager auto-accepts; no
     manual-review path exists in this profile."*
   - Line 60-61: same.
   - Line 82: remove `ManualReview: true` from the test fixture.
   - Line 101-102: invert.
3. After the edits, run:
   ```bash
   rg "ManualReview" scenarios/swarm-manager/ --type go
   ```
   Only matches that should remain are in `path:cli/cmd_stats.go` (which
   reads the `manual_review` Prometheus label from agent-manager's
   metrics — read-only, leave alone). If anything else matches,
   delete it.
4. Verify `defaultProfileRef.UpdateExisting=true`
   (`profile.go:143-147`) is unchanged. That's how the new shape
   overwrites the existing DB row on the next dispatch.
5. **Drain script** (separate from the source change). Create
   `path:scripts/cleanup/2026-04-28-drain-needs-review-launch-failures.sh`:
   - Lists every `agent-manager run` with status=needs_review.
   - For each, fetches events; classifies as silent-launch-failure
     when `0 RUN_EVENT_TYPE_MESSAGE events AND duration <2s`.
   - Calls `agent-manager run reject <id> --reason "silent launch
     failure pre-fix; see docs/plans/sandbox-launch-and-auto-approve-fixes-plan.md"`.
   - Has a `--dry-run` mode (default) and prints the cohort first.
     Operator confirms before mutating.
   - Header documents that the script is deleted in a separate
     commit after the drain completes (no permanent residue).

**Acceptance:**
- `go test ./internal/agentmanager/... -timeout 300s` green.
- `rg "ManualReview" scenarios/swarm-manager/api/` returns zero hits.
- Manual: dispatch any swarm-manager run that succeeds → resolved
  config shows `manual_review: false` (or absent), run reaches
  `RUN_STATUS_COMPLETE` with `APPROVAL_STATE_APPROVED`, sandbox
  applied.

### Phase F — Documentation health (per documentation-health skill)

**Files:**
- `path:scenarios/agent-manager/docs/internal/SEAMS.md`
- `path:scenarios/agent-manager/docs/internal/ERROR-SEMANTICS.md`
- `path:scenarios/agent-manager/docs/internal/PROBLEMS.md`
- `path:scenarios/workspace-sandbox/docs/internal/SEAMS.md`
- `path:scenarios/workspace-sandbox/docs/internal/ERROR-SEMANTICS.md`
- `path:scenarios/workspace-sandbox/docs/internal/PROBLEMS.md`
- `path:docs/plans/sandbox-launch-and-auto-approve-fixes-plan.md` (this file
  — mark complete on close-out).

**Steps:**

1. **agent-manager `SEAMS.md`** — document the SandboxLauncher
   boundary as the single point of host-path → namespace-path
   translation. Add `[CODE: scenarios/agent-manager/api/internal/adapters/sandbox/sandbox_launcher.go#translateHostPathToNamespace]`.
2. **agent-manager `ERROR-SEMANTICS.md`** — add the table from Phase
   D. Mark `LAUNCH_FAILED` and `SANDBOX_NO_EXIT_INFO` as terminal
   failures (no auto-retry).
3. **workspace-sandbox `SEAMS.md`** — document the
   `WaitForExit` contract and the
   `StreamProcessLogs → WaitForExit → emit event: exit` ordering.
4. **`PROBLEMS.md` (both)** — record the pre-fix symptom + commit
   range so the next agent has a tombstone if a regression appears.
5. Add `// DOC:` comments at the new functions
   (`translateHostPathToNamespace`, `validateRunOutcome`,
   `WaitForExit`) pointing at the relevant SEAMS sections.

### Phase G — Final integration test + smoke verification

**Files:**
- `path:scenarios/agent-manager/api/internal/orchestration/run_executor_e2e_test.go`
  (new, optional if existing e2e is already comprehensive — verify
  before adding).
- Manual smoke checklist (below).

**Steps:**

1. Add an end-to-end test that:
   - Spins up a real workspace-sandbox via the fake-provider harness.
   - Dispatches a sandboxed run with a trivial echo prompt.
   - Asserts: ≥1 message event, status complete, approval_state
     approved, sandbox applied.
2. Smoke:
   ```bash
   vrooli scenario restart workspace-sandbox
   vrooli scenario restart agent-manager
   vrooli scenario restart swarm-manager
   ```
   Then dispatch a real swarm-manager initiative-feedback run from the
   UI for `git-control-tower-ai-provenance` and verify everything
   end-to-end:
   - `agent-manager run get <id> --json` shows `manual_review: false`,
     `mode: protected`, ≥1 message event, exit_code 0, status
     complete, approval_state approved.
   - `transcript.ndjson` is non-empty.
   - The feedback round file in
     `path:scenarios/swarm-manager/initiatives/git-control-tower-ai-provenance/feedback/`
     contains a real `mutation_list` proposal.

---

## 8. Contract Decisions (locked)

These are the precise behavioral contracts this plan establishes. Every
test must align with these.

| Contract | Value | Where enforced |
|---|---|---|
| Sandbox namespace working dir | `/workspace` | `SandboxLauncher` constant + bwrap bind in workspace-sandbox |
| Host path → namespace translation | At SandboxLauncher boundary, never inside runner | `translateHostPathToNamespace` |
| Untranslatable host workdir | `LaunchBlocked{Code: "workdir_outside_sandbox"}` | `SandboxLauncher.Launch` |
| Missing SSE exit event after Phase B fix | `ErrSandboxNoExitInfo`, run goes to FAILED | client `finalizeWaitErr` |
| `WaitForExit` timeout | 5s after `StreamLog` returns | server `StreamProcessLogs` |
| Stderr surfacing on success | Always emit warn log when `errorOutput` non-empty | `claude_code.go` |
| "No-output protected run" categorizer | duration <2s AND 0 message events AND `mode == PROTECTED` | `validateRunOutcome` |
| swarm-manager `ManualReview` | **Field removed.** Always auto-accept. | `ProfileConfig` (no field), `buildProfile` (no assignment) |
| Other-scenario `ManualReview=true` defaults (scenario-to-cloud, scenario-to-desktop, ecosystem-manager, app-issue-tracker) | Untouched. Different semantics (trip-wire, not diff-review). | their respective `service.go` files |
| Auto-accept-on-success for swarm-manager | `applyAtRunEnd` always runs and applies | `applyAtRunEnd` (unchanged; the swarm-manager-side change makes `cfg.ManualReview` false at the source) |

---

## 9. Testing Plan (automated, per "prefer automated tests" feedback)

| Phase | New tests | Existing tests touched |
|---|---|---|
| A | `TestTranslateHostPathToNamespace` (table-driven, 4 branches); `TestSandboxLauncher_LaunchTranslatesHostMergedPath`; `TestSandboxLauncher_LaunchRejectsUntranslatableHostPath` | `TestSandboxLauncher_LaunchAndStreamLog` (no behavior change; assertion already uses `/workspace`) |
| B | `TestStreamProcessLogs_DeliversExitEventForFastFailure`; `TestProcessTracker_WaitForExit_ChannelSemantics` | `TestStartProcess_OnExitRecordsExit` (verify still green after `WaitForExit` is added) |
| C | `TestSandboxLauncher_NoExitInfo_ReportsFailure`; `TestClaudeCode_StderrSurfacedOnSuccess` | `TestClaudeCode_Execute_*` family — sweep for any test that expected `Success=true` on a no-exit-event launch and update |
| D | `TestValidateRunOutcome_DemotesSilentLaunchFailure`; `TestValidateRunOutcome_KeepsHonestSuccess`; `TestValidateRunOutcome_IgnoresInPlace` | `run_executor_test.go` apply-deferred tests (verify they still cover the manualReview=true path explicitly) |
| E | Update `TestDefaultProfileConfig_ManualReview` (rename if needed; assertion flips); update assertions at `service_test.go:35,60,101`. | All swarm-manager tests that incidentally check ManualReview must be inverted in lockstep. |
| G | `TestRunExecutor_E2E_SandboxedAutoApprove_HappyPath`; `TestRunExecutor_E2E_SandboxedLaunchFailure_ReportsFailed` | n/a |

**Test gates (must all pass before merge):**

```bash
cd scenarios/agent-manager/api && go test ./... -timeout 600s
cd scenarios/workspace-sandbox/api && go test ./... -timeout 600s
cd scenarios/swarm-manager/api && go test ./... -timeout 600s
cd packages/cli-core && go test ./...     # in case of shared utilities
cd packages/api-core && go test ./...     # in case of shared utilities
```

Lint:
```bash
golangci-lint run scenarios/agent-manager/api/... scenarios/workspace-sandbox/api/... scenarios/swarm-manager/api/...
gofumpt -l scenarios/agent-manager/api scenarios/workspace-sandbox/api scenarios/swarm-manager/api    # must produce zero output
```

---

## 10. Rollout / Validation Checklist

After all phases land:

- [ ] `vrooli scenario restart workspace-sandbox` healthy.
- [ ] `vrooli scenario restart agent-manager` healthy.
- [ ] `vrooli scenario restart swarm-manager` healthy.
- [ ] All three scenarios show `running healthy 2 processes` in
      `vrooli scenario status`.
- [ ] One real dispatch from the swarm-manager initiative-feedback page
      passes the Phase G smoke test end-to-end.
- [ ] No `bwrap: Can't chdir` line in any
      `~/.local/share/workspace-sandbox/<sb>/logs/*.stderr.log` for
      sandboxes created after the cutover.
- [ ] At least 3 fresh runs reach `RUN_STATUS_COMPLETE` +
      `APPROVAL_STATE_APPROVED` automatically.
- [ ] One intentional failure dispatch (e.g. invalid prompt that fails
      validation) reaches `RUN_STATUS_FAILED` with a populated
      `error_msg` and a warn-level stderr log event.
- [ ] Drain script run on existing leaked `needs_review` rows;
      `agent-manager run list --status needs_review` is empty (or
      contains only legitimately-pending operator reviews).
- [ ] Doc updates committed (Phase F).

---

## 11. Risks + Mitigations

| Risk | Likelihood | Mitigation |
|---|---|---|
| Phase B `WaitForExit` hangs because of an OnExit path that never fires | Low | Use a bounded timeout (5s); on timeout emit `event: error`. Phase C client treats this as failure, not silence. |
| Phase E auto-accept causes destructive operations to apply silently | Medium | The auditability contract preserves per-run diffs in workspace-sandbox; `ApplyAtRunEnd` writes provenance. Operator decision is "always auto-accept for swarm-manager"; if a destructive class of work emerges later that genuinely needs gating, that is a separate plan (likely a new scenario or a per-task config on the agent-manager side, not a swarm-manager-profile knob). |
| Phase A path translation breaks in-place runs (no sandbox) | Low | Translation only triggers when `SandboxLauncher` is used; in-place runs go through `HostLauncher`, untouched. Test `TestValidateRunOutcome_IgnoresInPlace` covers this. |
| Phase D categorizer false-positives a fast-but-honest run that emitted no messages (e.g. tool-only run) | Low | The heuristic requires `mode==PROTECTED` AND zero `RUN_EVENT_TYPE_MESSAGE` events. A real tool run still emits assistant-tool-call messages. If a legitimate runner type produces zero messages, exempt it explicitly via runner capability flag. |
| Drain script over-rejects | Medium | Script previews IDs first (`--dry-run`), requires explicit confirmation before mutating. Operator reviews the list. |
| Tests left asserting `ManualReview=true` slip through | Medium | Phase E step 2 explicitly enumerates the three asserts to invert. Plus `rg "ManualReview\s*=\s*true|ManualReview:\s*true" scenarios/swarm-manager` at the end of Phase E to confirm zero remaining matches. |
| New error codes break proto compatibility | Low | Adding enum values is additive (per interoperability-steer §5). Verify with `cd packages/proto && make breaking`. |

---

## 12. Non-goals / Prohibited Patterns

- **No** `if hostPath { fallback }` shims in the launcher. The contract
  is: SandboxLauncher always translates, and the host path never
  reaches the namespace.
- **No** swarm-manager-side `ManualReview=true` re-introduction. The
  field is **removed** from swarm-manager's `ProfileConfig`; there is
  no opt-in path. Auto-accept is unconditional for swarm-manager.
- **No** wrapping the existing `applyAtRunEnd` in a new gate. The
  existing gate already honors `ManualReview`; we are flipping the
  default upstream.
- **No** adding a "soft-fail" status between `complete` and `failed`.
  Silent-launch-failures are real failures.
- **No** retrying bwrap chdir failures inside the runner. They are
  configuration bugs, not transient.
- **No** new global error framework. We add specific error codes to
  the existing `RunErrorCode` enum.
- **No** snake_case → camelCase translation layer in this plan; the
  proto schema and `useProtoNames: true` already handle that. If
  drift is found while implementing, file a separate interop fix.

---

## 13. Definition of Done

A future agent (or this one resuming with no chat context) considers the
plan done when **all** of the following are true:

1. All seven phases (A–G) committed; each phase's acceptance bullets
   pass.
2. Test gates from §9 all green; lint clean.
3. `agent-manager run list --status needs_review` no longer contains
   silent-launch-failure rows from the pre-fix window.
4. End-to-end smoke (§10) passed against a real swarm-manager dispatch.
5. Documentation updates from §7 Phase F merged; cross-references
   (`// DOC:` ↔ `[CODE: ...]`) verified by
   `knowledge-observatory docs audit agent-manager` and the
   workspace-sandbox equivalent.
6. The new memory entry is written:
   `~/.claude/projects/-home-matthalloran8-Vrooli/memory/project_sandbox_launcher_workdir_translation.md`
   summarizing the new contracts (workdir translation, exit-info wait
   semantics, manual-review default flip, launch-failed categorizer).
   Index entry added to `MEMORY.md`.
7. The drain script has run, has been deleted, and the deletion is in a
   separate commit.

---

## 14. Operator decisions (resolved 2026-04-28)

1. **Default flip scope:** swarm-manager profile only. The four other
   scenarios that hardcode `ManualReview=true` (scenario-to-cloud,
   scenario-to-desktop, ecosystem-manager, app-issue-tracker) use it
   as a "writes shouldn't happen here" trip-wire and remain untouched.
2. **Per-skill / per-task override:** explicitly **not added**.
   swarm-manager runs **always** auto-accept on success — there is no
   per-skill knob, no per-task escape hatch, and the field is removed
   from `ProfileConfig` to prevent accidental re-introduction. If a
   future workflow needs operator-gated apply, that is a separate plan.
3. **Drain policy:** reject (not delete) the silent-launch-failure
   rows. Preserves the audit trail with explicit rationale referencing
   this plan.

---

## 15. Appendix — Quick reference: relevant lines for the implementing agent

```
agent-manager:
  internal/orchestration/run_executor.go:681-691   GetWorkspacePath → e.workDir
  internal/orchestration/run_executor.go:1103,1128 WorkingDir flows into LaunchRequest
  internal/orchestration/run_executor.go:1248-1254 VROOLI_SANDBOX_MERGED env injection
  internal/orchestration/run_executor.go:1698-1721 applyAtRunEnd manualReview branch
  internal/adapters/sandbox/sandbox_launcher.go:73-118  Launch → /processes
  internal/adapters/sandbox/sandbox_launcher.go:333-377 runStream / SSE event types
  internal/adapters/sandbox/sandbox_launcher.go:393-412 finalizeWaitErr
  internal/adapters/runner/claude_code.go:200-206  errorOutput stderr capture
  internal/adapters/runner/claude_code.go:280-321  Wait error type-switch
  internal/adapters/runner/exit_code.go            extractExitCode

workspace-sandbox:
  internal/driver/bwrap.go:303-407   buildBwrapArgs (incl. line 334 /workspace bind)
  internal/driver/bwrap.go:624-642   BuildExecCommand
  internal/driver/bwrap.go:674-714   StartProcess (incl. spawnExitReaper)
  internal/handlers/process.go:387-414  OnExit closure → RecordExit
  internal/handlers/process.go:711-759  StreamProcessLogs (the race lives here)
  internal/process/tracker.go:40-45     ExitInfo
  internal/process/tracker.go            (RecordExit, GetExitInfo)

swarm-manager:
  internal/agentmanager/profile.go:65-84  DefaultProfileConfig (line 82 = ManualReview)
  internal/agentmanager/profile.go:99-117 buildProfile → SandboxConfig.ManualReview
  internal/agentmanager/profile.go:133-148 defaultProfileRef.UpdateExisting=true
  internal/agentmanager/service_test.go:35-36, 60-61, 82, 101-102
```

---

*End of plan.*
