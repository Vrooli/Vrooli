# Plan: protected-sandbox-agent-launch

**Initiative:** `protected-agent-sandboxing`
**Item:** `execute/protected-sandbox-agent-launch`
**Effort:** XL
**Authored:** 2026-04-27

## Goal

Convert the three coding-agent runners (`claude_code`, `codex`, `opencode`) so that, when `SandboxConfig.Mode == protected`, the agent process tree itself launches **through workspace-sandbox execution APIs** rather than as a host `exec.Command`. The `Provider.ExecProcess` seam already exists and is tested; the runners do not yet route through it. This plan closes that gap, adds the streaming/interactive surface the runners need, and flips `Protected` to the default sandbox mode.

## User decisions baked into this plan

1. **All three runners in one bundle.** Single PR / single execution run, not per-runner sequencing. The runners share enough surface that splitting them costs more than it saves.
2. **Default flip to `Protected` once it's real.** Network default = `localhost` (local resources reachable, wider internet blocked) so existing local-resource workflows keep working out of the box. `RunMode.SANDBOXED` + `SandboxConfig.Mode = Sandboxed` (tracking-only) remains as the explicit per-spawn opt-out for runs that legitimately need full host capability (e.g., a `git push` after review, or scraping a remote URL during a research run).
3. **In-scope cleanup beyond the spec:** also fix the pre-existing `ecosystem-manager TestDetectVrooliRootFromEnv` failure (already done in the same session) and the swarm-manager CLI fingerprint-mismatch warning if the root cause is in `packages/cli-core/cliutil/stalechecker.go`.

## Critical constraint (from spec)

**No prompt or agent-side behavior changes.** The agent must not know it's in a sandbox. The orchestration layer changes how the process is launched; argv, env, working directory all flow identically to the agent's perspective.

## Workstream A — Extend `Provider.ExecProcess` to runner-grade parity

`Provider.ExecProcess` today is built for short, fire-and-forget `/exec` calls. Coding-agent runners need:

| Need | Source of truth in the runner | Provider extension |
|---|---|---|
| Long-lived stdout/stderr streaming | runners consume tokens line-by-line for UI streaming | `LaunchProcess(ctx, req) (Handle, error)`; `Handle.Stdout()/.Stderr() io.Reader` |
| Interactive stdin | Claude Code prompts mid-run; runner pipes stdin | `Handle.Stdin() io.WriteCloser` |
| Cancellation parity | runner respects `ctx.Done()` to kill the agent | `Handle.Kill(sig) error`; ctx cancel terminates the remote process group |
| Env injection | per-runner env (CLAUDE_CODE_*, OPENAI_*, OPENCODE_*) | `LaunchRequest.Env map[string]string` |
| Working dir | always merged overlay | `LaunchRequest.WorkingDir string` |
| Exit code + signal | runner reports run status | `Handle.Wait() (ExitState, error)` with `ExitCode int`, `Signal syscall.Signal`, `OOMKilled bool` |
| Heartbeat / liveness | runner watchdogs detect hung agents | `Handle.IsAlive() bool` (cheap, no roundtrip required) |

### Server side (workspace-sandbox)

`scenarios/workspace-sandbox/api/internal/handlers/process.go` already has long-running `/processes` endpoints (per `EXECUTION_MODES.md`). The plan:

- Add `POST /processes/launch` returning `process_id` + WebSocket/SSE stream URLs for stdout/stderr/stdin.
- Reuse existing process tracking, signal handling, and cleanup paths — no new lifecycle.
- Apply `evaluateProtectedGitAllowlist` and `NetworkMode` translation to launched processes the same way they apply to `/exec`. Both are already implemented for `/exec`; share the helpers.

### Client side (agent-manager)

`scenarios/agent-manager/api/internal/adapters/sandbox/workspace_sandbox.go`:

- Implement `WorkspaceSandboxProvider.LaunchProcess(ctx, req)` returning a `*launchHandle` that owns the WS/SSE connections.
- `Handle.Stdout()/.Stderr()` adapt the WS/SSE frames to `io.Reader` for drop-in compatibility with the runner's existing transcript consumer.
- `Handle.Stdin()` wraps a WS write half.
- `Handle.Kill()` posts to a `DELETE /processes/{id}` (or sends a kill frame on the WS).
- `Handle.Wait()` blocks on terminal frame; surfaces structured exit info.
- Tests with a mock workspace-sandbox HTTP/WS server.

## Workstream B — Wire each runner

Three files, identical pattern:

```go
// adapters/runner/claude_code.go (and codex_runner.go, opencode_runner.go)
func (r *ClaudeCodeRunner) Run(ctx context.Context, req runner.Request) (runner.Result, error) {
    if req.SandboxConfig != nil && req.SandboxConfig.Mode.Effective() == domain.SandboxModeProtected {
        return r.runProtected(ctx, req)
    }
    return r.runHost(ctx, req)
}
```

`runProtected` builds the **same argv, env, cwd** as `runHost`, then calls `provider.LaunchProcess`. The transcript-parsing/streaming consumer is reused unchanged because `Handle.Stdout()` returns an `io.Reader`. The heartbeat / watchdog logic remains as-is (it's runner-side, not exec-site).

### Per-runner specifics

- **claude_code**: Handles transcript JSON streaming. `runProtected` must preserve the line-buffered Reader semantics. Interactive prompts (Claude asking for confirmation) flow through `Handle.Stdin()`.
- **codex**: Already uses `--full-auto` for network blocking at the runner side. We **keep `--full-auto`** as defense-in-depth even though workspace-sandbox now also enforces network isolation. Memory note: per the existing project memory, dual enforcement is intentional.
- **opencode**: Has a watchdog that kills hung subprocess. Watchdog handoff must use `Handle.Kill()` instead of `cmd.Process.Kill()`.

## Workstream C — Capability matrix completion

Update `scenarios/agent-manager/docs/PROTECTED_MODE_RUNNERS.md`:

| Capability | claude_code | codex | opencode |
|---|---|---|---|
| Token streaming | ✅ verified by test | ✅ | ✅ |
| Interactive stdin | ✅ | ❌ N/A (no interactive mode) | ❌ N/A |
| Cancellation < 500ms | ✅ | ✅ | ✅ |
| Env injection | ✅ | ✅ | ✅ |
| Working dir | ✅ | ✅ | ✅ |
| Exit code propagation | ✅ | ✅ | ✅ |
| Child-process discovery | ✅ all children inside sandbox process group | ✅ | ✅ |
| OOM detection | ✅ | ✅ | ✅ |

Any cell that can't be ✅ at end of work gets a documented limitation row + owner.

## Workstream D — Default-mode flip

Once Workstreams A–C are green:

1. **Default `SandboxConfig.Mode`** flips from `Sandboxed` to `Protected` in:
   - `domain.DefaultSandboxConfig()` (or wherever the default is materialized)
   - `QuickRunDialog.tsx` UI default
   - swarm-manager spawn helpers (`freshConversationID()` co-location — same call site)
   - settings.json fixtures
2. **Default `SandboxConfig.NetworkMode = localhost`** — local resources still reachable, public internet blocked. Document the rationale and the per-spawn opt-out path.
3. **Operator escape hatch:** `RunMode.SANDBOXED` + `SandboxConfig.Mode = Sandboxed` continues to work and is documented as "tracking-only mode for runs that need host capabilities."
4. **`RunMode.IN_PLACE`** remains the full-bypass mode for the rare case where even sandboxing is wrong (e.g., self-modifying scenarios, agent-manager developing itself).
5. Update metrics: `agent_manager_sandbox_adoption_total{sandbox_mode=...}` should now show predominantly `protected` after the flip — confirm in `swarm-manager stats sandbox-adoption`.

## Workstream E — End-to-end tests

Add (under `scenarios/agent-manager/api/internal/adapters/runner/` and integration-test sites):

1. `runner_protected_streaming_test.go` — protected run streams identically to host run; same byte sequence into the transcript consumer.
2. `runner_protected_cancellation_test.go` — `ctx.Cancel()` terminates the agent process within 500ms; child processes are reaped.
3. `runner_protected_provenance_test.go` — protected run produces identical provenance records (`pending_changes`, `applied_changes`, `conversation_id`, `cost_usd`, `schema_version`) compared to a tracking-mode reference run.
4. `runner_protected_git_allowlist_test.go` — agent that calls `git push` mid-run gets a structured 403 surfaced as a typed run-level error (not a hang or corrupt run state).
5. `runner_protected_network_isolation_test.go` — `NetworkMode = localhost` blocks `curl https://example.com` but allows `curl http://localhost:5432` (or a stub local resource).
6. `runner_protected_runner_isolation_test.go` — assert that when `Mode == protected`, the host's `exec.Command` is **never** invoked for the agent binary (only `provider.LaunchProcess`).

Plus end-to-end smoke: spawn a real run via the UI in protected mode, observe successful completion + provenance landing.

## Workstream F — Cleanup the previous purge skipped

Per the supplementary cleanup section of the plan:

- Sweep agent-manager UI + swarm-manager UI for stale "approval required" copy and dead types via `pnpm exec eslint .` (one-shot, fix any new findings).
- Confirm `agentRequiresApproval` is `reserved` (not deleted outright) in proto with a comment explaining wire-incompat preservation.
- Verify migration `down` paths exist for `requires_approval` drop and `migration_002_provenance_schema_version.sql`.
- Grep test fixtures for stale fields: `**/test-fixtures/**/*.json` for `agentRequiresApproval`, `requires_approval`, `autoApprove`.
- Doc sweep: `scenarios/agent-manager/docs/`, top-level `docs/`, AGENTS.md, scenario READMEs for any legacy "approval policy" wording.

## Workstream G — Pre-existing fixes (in-scope per user)

- ✅ **Done in advance:** `TestDetectVrooliRootFromEnv` fixed by overriding both `VROOLI_SOURCE_ROOT` and `VROOLI_ROOT` in the test (root cause: `repo-contract-go` checks `VROOLI_SOURCE_ROOT` first by intentional design; test was missing that env override).
- **swarm-manager CLI fingerprint loop:** investigate `packages/cli-core/cliutil/stalechecker.go`. Build-time fingerprint and run-time recomputation disagree on the same source tree. Add a `--debug-stale` flag that prints which file's hash differs, then fix root cause (likely glob mismatch between installer and runtime checker, or a mode/timestamp leak into the hash).

## Workstream H — Architectural polish (screaming architecture)

- **Promote runner protected-mode to a first-class seam** via `Launcher` interface:
  ```go
  type Launcher interface {
      Launch(ctx context.Context, req LaunchRequest) (LaunchHandle, error)
  }
  ```
  with `HostLauncher` and `SandboxLauncher` implementations. Each runner depends on `Launcher`, not on a switch. Drops a switch-statement test surface and matches "agents shouldn't care where the process executes."
- **Wrap-not-use principle**: add a section to `PROTECTED_MODE_RUNNERS.md` linking the runner-fork to the long-run goal of forbidding direct external-tool use.

## Workstream I — Validation gate

After every workstream:

1. `go build ./...` + `go test ./... -count=1 -timeout 300s` for agent-manager, workspace-sandbox, swarm-manager, sandbox-provenance, repo-contract-go.
2. `golangci-lint run ./...` + `gofumpt -l .` clean across all four Go modules.
3. `pnpm run type-check` clean for agent-manager/ui, swarm-manager/ui, web-console/ui.
4. `vrooli scenario restart agent-manager workspace-sandbox swarm-manager` and confirm `/health` healthy + `/metrics` exposes adoption counters.
5. Real protected-mode run via UI; confirm provenance lands identically to tracking-mode reference.

## Acceptance criteria

This item is done when:

1. ✅ All three runners route through `provider.LaunchProcess` when `SandboxConfig.Mode == Protected`; the runner-isolation test (E.6) proves it.
2. ✅ Streaming, interactive, cancellation, env, cwd, exit-code, OOM detection all preserved (Workstream C matrix all-green).
3. ✅ `Protected` is the default sandbox mode; `NetworkMode = localhost` is the default network mode; `Sandboxed` (tracking-only) is the documented operator opt-out.
4. ✅ Provenance shape identical between protected and tracking modes (E.3).
5. ✅ Git allowlist enforcement surfaces as a typed run-level error, not a corrupt run (E.4).
6. ✅ All Workstream I health checks green.
7. ✅ Capability matrix in `PROTECTED_MODE_RUNNERS.md` filled with verified ✅/❌ for every cell, no `?`s.
8. ✅ Cleanup sweeps (Workstream F) leave zero stale references to deleted approval/decision fields.
9. ✅ Pre-existing CLI fingerprint warning is gone after Workstream G fix.

## Risks and mitigations

| Risk | Mitigation |
|---|---|
| `/processes/launch` streaming has higher latency than host exec; agent UX feels laggy | Benchmark in E.1; if >50ms p95 on token-stream, batch frames or fall back to a UNIX-socket transport for local |
| Codex `--full-auto` and workspace-sandbox network isolation conflict | Memory note already says dual enforcement is intentional; add an integration test that codex still functions correctly under both layers |
| Interactive mode: WS stdin frame ordering vs PTY semantics | claude_code is line-oriented, not full-PTY; line-buffered WS suffices. If a future runner needs a real PTY, add a separate code path then |
| Default-mode flip breaks an unforeseen consumer | Roll out in stages: (a) flip default in code, (b) leave per-spawn override as escape, (c) monitor `sandbox_adoption_total` metric for one week before considering removing the `Sandboxed` (tracking-only) mode |
| Watchdog races: `Handle.Kill()` racing with natural exit | `Wait()` returns idempotently after Kill; runner code already handles this for host case |

## Out of scope (explicit non-goals)

- Multi-runner per-run support (one run = one runner today).
- Replacing `RunMode.IN_PLACE`. It stays as the full-bypass mode.
- Live-migration of running tracking-mode runs to protected mode.
- Removing the tracking-only mode entirely. We keep it for legitimate full-capability use cases.
- New agent runners (Cursor, Aider, etc.). Add them later via the same `Launcher` pattern.

## Sequencing

```
Day 1–3:   Workstream A (extend Provider; workspace-sandbox /processes/launch + tests)
Day 4–6:   Workstream B (wire all 3 runners + parity tests for one — claude_code as pilot)
Day 7–8:   Finish Workstream B for codex + opencode
Day 9–10:  Workstream E (full E2E test suite)
Day 11:    Workstream C (capability matrix doc finalize)
Day 12:    Workstream D (default flip + metrics verify)
Day 13:    Workstream F + G + H (cleanup + pre-existing fixes + architectural polish)
Day 14:    Workstream I (final validation gate)
```

Total: ~14 working days for a focused engineer. PR should be reviewable in chunks per workstream.

## References

- Spec: `swarm-manager backlog get --kind execute --name protected-sandbox-agent-launch`
- Existing seam: `scenarios/agent-manager/api/internal/adapters/sandbox/workspace_sandbox.go`
- Capability matrix doc (will be updated): `scenarios/agent-manager/docs/PROTECTED_MODE_RUNNERS.md`
- Git allowlist enforcement: `scenarios/workspace-sandbox/api/internal/handlers/process.go`
- Network isolation memory note: `~/.claude/projects/-home-matthalloran8-Vrooli/memory/project_agent_manager_network_isolation.md`
- Wrap-not-use principle: AGENTS.md § Wrap-Not-Use Principle
