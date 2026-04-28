# Protected-Mode Runners — Capability Matrix

> **Status (2026-04-28):** the runner-fork pilot has landed for the
> `claude_code` runner's primary `Execute` path. Protected-mode requests
> now route the agent process through workspace-sandbox `/processes` via
> the `runner.Launcher` seam (introduced by this work) and its
> `SandboxLauncher` implementation. The `codex` and `opencode` runners,
> and the `claude_code` durable-transcript and continuation paths, still
> use the host launch path; extending the seam to them is the next slice.
>
> **What changed in 2026-04-28:**
>
>   - `runner.Launcher` interface introduced. `claude_code.Execute` now
>     depends on it instead of `exec.Command` directly.
>   - `runner.HostLauncher` wraps the existing `managedProcess` machinery
>     — zero behavior change for non-protected runs.
>   - `sandbox.SandboxLauncher` implements `runner.Launcher` against the
>     workspace-sandbox `/processes`, `/files/content`, and
>     `/processes/{pid}/logs` endpoints.
>   - Workspace-sandbox `StartProcess` now enforces the protected-mode
>     git allowlist (previously `Exec` only — agents could bypass via
>     `/processes`).
>   - Validation gate flipped: `SandboxConfig.Mode = protected` is no
>     longer rejected. Falls back to host launch with an explicit warn
>     event when no `SandboxLauncherFactory` is wired.

## What "protected mode" means today (claude_code Execute)

| Layer | Effect when `SandboxConfig.Mode == protected` |
|---|---|
| Sandbox creation (`Provider.Create`) | `Behavior.Protected.GitAllowlist` is set to the locked default. Workspace-sandbox stores it on the sandbox. |
| Coding-agent launch (claude_code Execute) | **Pilot path**: `runner.SandboxLauncher.Launch` POSTs to `/processes`. The agent process runs inside bwrap with the sandbox's network and git guardrails applied at the OS level, not just on its file output. |
| Direct `/exec` calls (any caller using `Provider.ExecProcess`) | Git verb allowlist enforced server-side — non-listed verbs return structured 403. Surfaces as `ExecProcessResult.Blocked` for callers. |
| `/processes` background launches (the new pilot path) | Git verb allowlist enforced server-side. Surfaces as a typed `*sandbox.LaunchBlocked` on `Launcher.Launch`. |
| Bwrap network isolation | `NetworkMode` translated by the adapter: `none`/empty → full isolation, `localhost` → vrooli-aware (loopback only), `full` → unrestricted. |
| Resource limits | Forwarded via `/processes` body, clamped by workspace-sandbox `ExecutionConfig`. |
| Apply-at-run-end | Identical to tracking mode — provenance write, acceptance filter, manual-review TTL all unchanged. |

## What "protected mode" does **not** yet mean

| Layer | Behavior in protected mode today |
|---|---|
| `claude_code` durable-transcript path (`executeWithDurableTranscript`) | Still uses host launch. Migrating it to `Launcher` is mechanical given the seam, but every transcript test would need re-validation; deferred to next slice. |
| `claude_code` continuation path (`Continue`) | Still uses host launch. Same rationale. |
| `codex` runner | Still uses host launch in all modes. Codex's `--full-auto` is its own enforcement layer; the runner-fork next slice keeps `--full-auto` as defense-in-depth. |
| `opencode` runner | Still uses host launch in all modes. |
| Truly interactive stdin | Workspace-sandbox `/processes` doesn't expose a stdin pipe; the pilot stages stdin as a file in the sandbox and uses a `bash -c 'exec ... < prompt.txt'` wrapper. Suitable for prompt-via-stdin (the actual claude_code pattern); not for mid-run interactive prompts. |
| Stdout/stderr stream separation | Workspace-sandbox merges them into a single log file. The pilot tolerates this because `parseStreamEvents` skips non-JSON lines, but stderr accumulation is empty in protected mode. Future work: ws-sb could split streams. |

## Per-runner deltas (target end state)

| Runner | Tracking mode | Protected mode (today, 2026-04-28) | Protected mode (target) |
|---|---|---|---|
| `claude_code` Execute | host exec, transcript parsing, heartbeat | **✅ shipped** — routes through `SandboxLauncher` via `Launcher` seam | streaming via long-running `/processes` (works today, polling-based) |
| `claude_code` durable transcript | host exec | same as tracking (host) | route through `Launcher` |
| `claude_code` Continue | host exec | same as tracking (host) | route through `Launcher` |
| `codex` | host exec, `--full-auto` for network | same as tracking (host) | route through `Launcher`; `--full-auto` retained |
| `opencode` | host exec, watchdog | same as tracking (host) | route through `Launcher` |

## How protected mode is exercised today

- `runner.Launcher` interface: `internal/adapters/runner/launcher.go`
- `runner.HostLauncher` (legacy host path): `internal/adapters/runner/host_launcher.go`
- `sandbox.SandboxLauncher` (protected path): `internal/adapters/sandbox/sandbox_launcher.go`
- `claude_code.Execute` routing logic: `internal/adapters/runner/claude_code.go` → `selectLauncher`
- Wire-encoder that materializes `Behavior.Protected.GitAllowlist`:
  `internal/adapters/sandbox/wire_encoder.go` (covered by `wire_encoder_test.go`)
- Workspace-sandbox `/exec` git-allowlist enforcement:
  `scenarios/workspace-sandbox/api/internal/handlers/process.go` (`Exec`),
  tests in `process_git_allowlist_test.go`
- Workspace-sandbox `/processes` git-allowlist enforcement (NEW):
  same file (`StartProcess`), tests in `process_start_git_allowlist_test.go`

### Test coverage

| Concern | Test file | Status |
|---|---|---|
| `Launcher` contract / `HostLauncher` behavior | `runner/launcher_test.go` | 7 tests, all pass |
| `claude_code` routing logic | `runner/claude_code_routing_test.go` | 7 tests, all pass |
| `SandboxLauncher` lifecycle (start, stream, kill, ctx-cancel, 403) | `sandbox/sandbox_launcher_test.go` | 5 tests + helpers, all pass |
| `Exec` git allowlist | `handlers/process_git_allowlist_test.go` | unchanged, pass |
| `StartProcess` git allowlist (NEW) | `handlers/process_start_git_allowlist_test.go` | 4 tests, all pass |

## Default-mode policy

Protected is **NOT yet the default** SandboxConfig.Mode. The pilot only
covers `claude_code.Execute`; flipping the default before codex/opencode
and the durable-transcript/continuation paths route through the seam
would silently downgrade those runs back to host execution (with a warn
event). Default flip is sequenced behind those follow-on slices.

To opt a single run into protected mode today, set
`run.ResolvedConfig.SandboxConfig.Mode = "protected"` at request time.

## How to extend the seam to a new runner

1. Replace the runner's `exec.CommandContext(...) + startManagedProcess(...)
   + cmd.StderrPipe + cmd.StdinPipe` block with:

   ```go
   launcher := r.selectLauncher(ctx, req)
   proc, err := launcher.Launch(ctx, runner.LaunchRequest{
       Command:     "env",
       Args:        envArgs,
       Env:         r.buildEnv(req),
       WorkingDir:  req.WorkingDir,
       Stdin:       strings.NewReader(prompt),
       IdleTimeout: DefaultStreamIdleTimeout,
   })
   ```

2. Read from `proc.Stdout()` / `proc.Stderr()` instead of `mp.Stdout()`
   and `cmd.StderrPipe()`. Call `proc.ResetIdleTimer()` instead of
   `mp.ResetTimer()`. Call `proc.Wait()` instead of `mp.Wait()`.

3. Track in `r.launched[req.RunID]` (LaunchedProcess map) instead of
   `r.runs[req.RunID]` (`*exec.Cmd` map). Update `Stop()` to consult
   the new map first (claude_code's pattern).

4. Add a routing test paralleling `claude_code_routing_test.go` to pin
   the protected-vs-host fork.

## Open follow-on work

Tracked under the `protected-agent-sandboxing` initiative:

- Migrate `claude_code` durable-transcript and continuation paths to the
  `Launcher` seam.
- Wire `codex` and `opencode` runners through the seam.
- Workspace-sandbox: split stdout/stderr in `/processes` log capture so
  protected runs can populate the runner's `errorOutput` buffer.
- Workspace-sandbox: native stdin pipe on `/processes` (replaces the
  bash-wrapper file-redirect workaround).
- Default-mode flip: once all runners route through the seam, change
  `DefaultSandboxConfig().Mode` from `tracking` to `protected` and
  document the per-spawn opt-out.
