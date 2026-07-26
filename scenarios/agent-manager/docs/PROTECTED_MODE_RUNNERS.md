# Protected-Mode Runners — Capability Matrix

## Tool restriction enforcement

`allowedTools` and `deniedTools` are canonical profile controls, not runner CLI
passthrough. Claude Code and Grok enforce both per launch and map canonical
names to their native vocabulary (`--allowedTools` / `--disallowedTools` and
`--allow` / `--deny`) on execute and continuation. Codex and OpenCode report
no per-launch tool enforcement; a restricted profile therefore fails closed unless its
declaration explicitly selects advisory policy. Policy fallback rechecks this
capability after every runner switch and skips an unsupported candidate under
enforced policy. The live codec registry is authoritative: inspect it with
`agent-manager runner tools`.

## Run-control fidelity and isolation

`effort` is a canonical run control with the values `low`, `medium`, `high`,
`xhigh`, and `max`. Claude Code and Grok receive `--effort`; Codex receives
`-c model_reasoning_effort=<level>`; OpenCode receives its provider-specific
`--variant` equivalent. Execute, continuation, and
interactive launch paths preserve the resolved model, effort, prompt, and
applicable tool controls.

Codec-pipe children receive a minimal environment allowlist rather than the
web-console process environment. In particular, inherited runner-state
variables are excluded. Codex and Grok receive an explicit, run-scoped session
home outside the sandbox overlay; it persists across turns so a continuation
can read the first turn's rollout without exposing the parent home.

Default sandbox lifecycle configuration checkpoints between turns and deletes
at terminal status. Manual-review and explicitly preserving configurations are
the only deliberate preservation paths. These claims are executable contracts:
`internal/contracts/run_control_test.go`,
`internal/adapters/runner/env_utils_test.go`, and
`internal/orchestration/session_home_test.go` run without a runner binary or
network access; the sanitized codec corpus adds representative recorded
transcripts under `internal/adapters/runner/codecs/testdata/corpus/`.

> **Status (2026-04-27, Slices 1 + 4 complete):** every coding-agent
> runner — `claude_code`, `codex`, and `opencode` — routes **every
> launch path** (streaming Execute, durable-transcript Execute, durable-
> transcript Continue, streaming Continue) through the
> `runner.Launcher` seam. When `SandboxConfig.Mode == Protected` and a
> sandbox factory is wired, the agent process tree itself runs inside
> workspace-sandbox bwrap isolation — not just on the host with a
> tracked overlay. Protected is now the **default** sandbox mode
> produced by `domain.DefaultSandboxConfig()`; tracking-mode is the
> documented operator opt-out for runs that legitimately need full host
> capability.
>
> **What changed in Slice 1 (claude_code/codex/opencode remainder):**
>
>   - All three runners' durable-transcript paths
>     (`runDurableCommand` / `runTranscriptCommand`) now take a
>     `Launcher` + `LaunchRequest` instead of a raw `*exec.Cmd`. Stdout
>     is `io.Copy`'d from the launcher's pipe into the transcript file
>     in a goroutine; the previous `cmd.Stdout = file` direct-pipe
>     pattern is gone. `transcript.OnProcessStart(pid, pid)` is fed by
>     `proc.PID()`.
>   - All three runners' streaming Continue paths
>     (`claude_code.Continue`, `codex.Continue`, `opencode.Continue`)
>     route through `r.selector.PickFor` using `req.GetConfig()` and
>     `req.SandboxID`.
>   - `ContinueRequest` gained `ResolvedConfig *domain.RunConfig` and
>     `SandboxID *uuid.UUID` fields plus a `GetConfig()` method,
>     mirroring `ExecuteRequest`. Orchestration's continuation call
>     site (`orchestration/service.go:2204`) populates them from the
>     stored `Run.ResolvedConfig` and `Run.SandboxID` so protected
>     runs continue in the same launcher.
>   - `launcherSelector.Pick(ctx, req ExecuteRequest)` is now a thin
>     wrapper over the new `PickFor(ctx, runID, cfg, sandboxID, sink)`
>     primitive — both Execute and Continue paths share one routing
>     implementation.
>   - The legacy `r.runs map[uuid.UUID]*exec.Cmd` field is gone from
>     all three runners. `r.launched map[uuid.UUID]LaunchedProcess` is
>     the single registry. `Stop()` consults only this map and uses
>     `proc.Signal(grace) + proc.Kill()` for the SIGTERM/SIGKILL
>     escalation.
>   - Wait-error handling unified via `runner.extractExitCode(err)`
>     which understands both `*exec.ExitError` (host) and
>     `*sandbox.remoteExitError` (sandbox) by satisfying a small
>     `ExitCode() int` interface.
>
> **What changed in Slice 4 (default flip):**
>
>   - `domain.DefaultSandboxConfig()` now returns
>     `Mode = SandboxModeProtected`. The auditability-contract test
>     `TestDefaultSandboxConfig_LockedDefaults` was updated to pin
>     this.
>   - `Orchestrator.resolveSandboxConfig` starts from
>     `DefaultSandboxConfig()` (instead of zero-valuing) and backfills
>     `Mode` and `NetworkMode` after the override clone, so partial
>     `SandboxConfig` overrides (notably swarm-manager's
>     acceptance-only inline configs) don't silently strip Mode to
>     unspecified.
>   - `SandboxMode` doc comments rewritten: Tracking is now described
>     as the explicit operator opt-out, Protected as the production
>     default.
>
> **What changed in earlier slices (5 + 3 + 2):**
>
>   - Slice 5 fixed the CLI fingerprint rebuild loop, the SandboxMode
>     doc-drift, added the SEAMS.md `1c. Process Launcher` section,
>     and tracked four follow-up items under
>     `protected-agent-sandboxing`.
>   - Slice 3 extracted `launcherSelector` and
>     `buildEnvWrappedLaunchRequest`. Eight selector routing tests
>     replaced the per-runner copies.
>   - Slice 2 wired the streaming Execute paths of `codex` and
>     `opencode` through the seam. Six routing tests for each landed.
>   - The original pilot (Slice H) introduced `runner.Launcher`,
>     `HostLauncher`, `SandboxLauncher`, and the `/processes`
>     git-allowlist enforcement.

## Interactive execution mode is rejected for protected runs

Everything below describes the **codec-pipe** execution path, where agent-manager
owns the agent child process and can route it through the `runner.Launcher` seam
into workspace-sandbox bwrap isolation. The **interactive** execution mode
(`ExecutionMode == interactive`, see
[interactive-runner-design.md](interactive-runner-design.md)) launches the real
agent CLI inside a web-console tmux session on the host — there is no
agent-manager-owned process tree to place inside a protected sandbox, so its
guarantees cannot hold. Interactive mode is therefore **rejected at run
validation** for any sandboxed run (`RunMode == RunModeSandboxed`, which covers
both the protected and tracking sandbox modes), with a clear error; such runs
stay on the codec-pipe / sandbox-launcher path documented here. Interactive mode
requires an in-place run (`RunMode == RunModeInPlace`, i.e. sandbox off). The
gate lives in `internal/domain/validation.go` and is pinned by
`internal/domain/interactive_gate_test.go`.

## What "protected mode" means today (every runner, every path)

| Layer | Effect when `SandboxConfig.Mode == Protected` |
|---|---|
| Sandbox creation (`Provider.Create`) | `Behavior.Protected.GitAllowlist` is set to the locked default. Workspace-sandbox stores it on the sandbox. |
| Coding-agent launch (every runner / every path) | `runner.SandboxLauncher.Launch` POSTs to `/processes`. The agent process runs inside bwrap with the sandbox's network and git guardrails applied at the OS level, not just on its file output. Applies to streaming Execute, durable-transcript Execute, durable-transcript Continue, and streaming Continue paths uniformly. |
| Direct `/exec` calls (any caller using `Provider.ExecProcess`) | Git verb allowlist enforced server-side — non-listed verbs return structured 403. Surfaces as `ExecProcessResult.Blocked` for callers. |
| `/processes` background launches (the runner-fork path) | Git verb allowlist enforced server-side. Surfaces as a typed `*sandbox.LaunchBlocked` on `Launcher.Launch`. |
| Network isolation | The agent process always launches under the `vrooli-aware` profile (`localhost` network mode); direct `/exec` calls translate `NetworkMode`: `none`/empty → full isolation, `localhost` → vrooli-aware, `full` → unrestricted. **Known gap:** no containment backend enforces loopback-only network, so `localhost` currently grants *unrestricted* network on every OS (tracked as scenario-qa bug `knw-1784006975589682125`). The launcher selector emits a warn run-event naming the missing `network-loopback-only` enforcement on every protected launch until a backend claims it. |
| Resource limits | Forwarded via `/processes` body, clamped by workspace-sandbox `ExecutionConfig`. |
| Apply-at-run-end | Identical between protected and tracking modes. Agent-manager lifecycle policy chooses `/turn-checkpoint` for continuable turns, which records provenance, applies acceptance filtering, and parks the sandbox as `checkpointed`; final apply paths use workspace-sandbox's final apply/approval endpoint. |
| Exit code propagation | Both host and sandbox launches surface the exit code through `Wait()`'s error; `runner.extractExitCode` reads either via the `ExitCode() int` interface. |

## Per-OS containment backend

Protected mode's OS-level guardrails are carried out by a per-OS *containment
backend* selected inside workspace-sandbox behind a platform-neutral seam. The
sandbox reports the guarantees its backend actually enforces
(`GET /api/v1/driver/containment`, mirrored into `runner.Containment`), and the
launcher selector warns when the enforced set is missing anything protected mode
depends on — `filesystem-write-containment` and `network-deny`
(`Containment.MissingProtectedEnforcements`). Protected mode is never silently
downgraded: the run proceeds, but the gap is surfaced loudly.

| Enforcement | Linux (`bwrap`) | macOS (`seatbelt`) | Other (`none`) |
|-------------|:---------------:|:------------------:|:--------------:|
| `filesystem-write-containment` | ✅ | ✅ | ❌ |
| `network-deny` | ✅ | ✅ | ❌ |
| `pid-namespace` | ✅ | ❌ | ❌ |
| `path-illusion` | ✅ | ❌ | ❌ |
| **Protected mode honored (no gap warn)** | ✅ | ✅ | ❌ (both required enforcements missing) |

macOS Seatbelt (via `sandbox-exec`) is **partial by design**: it enforces both
protected-mode-required guarantees, so protected mode runs on macOS without a
degradation warning, but it provides no path illusion (the workspace stays at
its host `mergedDir`) and no pid-namespace isolation. See
[workspace-sandbox EXECUTION_MODES.md](../../workspace-sandbox/docs/EXECUTION_MODES.md#per-os-containment)
for the backend detail.

## Per-runner / per-path matrix

Every cell is shipped via the launcher seam. Tracking-mode and Protected
mode share the same code path; the only difference is whether `Pick`
returns the host or sandbox launcher.

| Runner / path | Mechanism today | Notes |
|---|---|---|
| `claude_code.Execute` (streaming) | `r.selector.Pick` + `buildEnvWrappedLaunchRequest` | the original Slice-H pilot |
| `claude_code.executeWithDurableTranscript` | `r.selector.Pick` + `buildEnvWrappedLaunchRequest`, `runDurableCommand(launcher, request)` | Slice 1 |
| `claude_code.continueWithDurableTranscript` | `r.selector.PickFor(req.RunID, req.GetConfig(), req.SandboxID, …)` + builder, `runDurableCommand` | Slice 1 |
| `claude_code.Continue` (streaming) | `r.selector.PickFor` + builder; idle reset via `proc.ResetIdleTimer` | Slice 1 |
| `codex.executeWithJSONStream` (streaming) | `r.selector.Pick` + builder; `--full-auto` retained as defense-in-depth | Slice 2 |
| `codex.executeWithWrapper` (streaming fallback) | `r.selector.Pick` with raw `LaunchRequest` (resource-codex `--tag` flag, no env shim) | Slice 2 |
| `codex.executeWithJSONTranscript` | `r.selector.Pick` + builder, `runTranscriptCommand(spec)` | Slice 1 |
| `codex.executeWithWrapperTranscript` | `r.selector.Pick` with raw `LaunchRequest`, `runTranscriptCommand(spec)` | Slice 1 |
| `codex.continueWithJSONTranscript` | `r.selector.PickFor` + builder, `runTranscriptCommand` | Slice 1 |
| `codex.Continue` (streaming) | `r.selector.PickFor` + builder | Slice 1 |
| `opencode.Execute` (streaming) | `r.selector.Pick` + builder; `step_finish` early-exit via `proc.Kill()` | Slice 2 |
| `opencode.executeWithTranscript` | `r.selector.Pick` + builder, `runTranscriptCommand(spec)` | Slice 1 |
| `opencode.continueWithTranscript` | `r.selector.PickFor` + builder, `runTranscriptCommand` | Slice 1 |
| `opencode.Continue` (streaming) | `r.selector.PickFor` + builder | Slice 1 |

> **OpenCode invocation contract (2026-06-22).** The OpenCode codec
> (`codecs/opencode.go`) invokes the **raw `opencode` binary** directly
> — `opencode run <prompt> --format json --print-logs [-m <model>]
> [--session <id>]` — not the `resource-opencode` wrapper (whose `run`/
> `status` subcommands no longer exist). Availability is a plain
> `exec.LookPath("opencode")`. Capabilities reported: `SupportsStreaming`
> and `SupportsCostTracking` are **true** (the JSON event stream carries
> `step_finish` token/cost data), alongside continuation and cancellation.
> This matches the codex/claude-code raw-binary contract; sandbox launch
> routing through `launcherSelector` is unchanged.

## Codec capability contract (SSOT, 2026-06-22)

This is the single source of truth for each codec's
`Capabilities()` struct (`internal/adapters/runner/codecs/{claude,codex,opencode,grok}.go`).
The drift guard `codecs.TestCapabilitiesConformance` pins code to this
table — update both together. Parity means "same capability wherever the
upstream CLI allows," not faked support.

| Capability | claude_code | codex | opencode | grok | Notes |
|---|:---:|:---:|:---:|:---:|---|
| Messages | ✅ | ✅ | ✅ | ✅ | structured assistant/user messages |
| Tool events | ✅ | ✅ | ✅ | ❌ | tool_call / tool_result. **grok:** headless stdout never surfaces tool events even when a tool runs (trace-verified) |
| Cost tracking | ✅ | ✅ | ✅ | ❌ | token + cost from the JSON stream. **grok:** no usage/cost in the stream |
| Streaming | ✅ | ✅ | ✅ | ✅ | line-delimited JSON event stream (grok: `--output-format streaming-json`) |
| Cancellation | ✅ | ✅ | ✅ | ✅ | mid-run kill via the launcher |
| Continuation | ✅ | ✅ | ✅ | ✅ | claude `--resume`; codex `exec resume`; opencode `--session`; grok `--resume` |
| Image attachments | ✅ | ✅ | ✅ | ❌ | claude embeds paths in the prompt; codex `-i/--image`; opencode `-f/--file`; **grok:** no headless image flag |
| Local models (Ollama) | ❌ | ✅ | ✅ | ❌ | **Acknowledged difference:** claude-code is Anthropic-native (litellm proxy retired). codex routes `ollama/*` models via `--oss --local-provider ollama`; opencode via its first-class `ollama` provider block. grok is xAI-native. |

> **Grok is an intentional non-uniform row (no vaporware).** Unlike the
> other three coding agents, grok's headless `-p … --output-format
> streaming-json` output carries only reasoning (`thought`), assistant text
> (`text`), a terminal `end` event with the session id, and `error` events —
> verified against `codecs/testdata/grok_trace.jsonl`. So tool events, cost,
> and image attachments are honestly **false**. Permission posture (D4): the
> codec maps `SkipPermissionPrompt` → grok's `--always-approve`; per-run
> AllowedTools/DeniedTools are not written (they would race on the single
> global `~/.grok/config.toml` and collide with the resource's `agentpolicy`
> gate). The grok resource's PreToolUse deny hook enforces. Per-run
> enforcement is a filed follow-up.

**Model advertisement (`SupportedModels`).** Raw codecs contain no curated
cloud model IDs. `SupportedModels` is limited to locally-pulled `ollama/*`
models that codex + opencode discover via a cached (60s TTL) exec to the probe
SSOT (`resource-ollama models list --json`, wrapped in `codecs/ollama.go`).
The probe is agent-safe: an unreachable daemon or absent SSOT reports no
dynamic entries, never an error. Resource-owned `policy resolve` supplies the
concrete cloud model only when Agent Manager creates a run, and the result is
retained in that run's immutable snapshot. `SupportedFeatures` /
`AllowedExtraFlags` differ per upstream CLI (claude exposes
`EnableBrowser` / `--disallowedTools`; codex + opencode expose `--verbose`)
— these are genuine upstream differences, documented not reconciled.

**Update surface.** All three resource CLIs share one
`upstream-check` verb (`github.com/vrooli/cli-core/upstreamcheck`, wired via
`upstreamverb.Commands`). `vrooli resource upstream-check [--all] [--json]`
aggregates the three (degrades any unresolved resource to `unknown`, always
exit 0). Each resource exposes an opt-in `update` verb off its
`lib/install.sh` that reinstalls to the pin (never silent auto-update).

## Trade-offs the seam used to lean on (all resolved 2026-04-28)

| Concern | Resolution | Item |
|---|---|---|
| Native stdin pipe | `POST /processes/{pid}/stdin?close=true` streams stdin bytes into a real pipe wired to the bwrap'd process; the launcher posts `req.Stdin` directly. The old file-staging and `bash -c 'exec ... < prompt.txt'` wrapper are gone. | `execute/ws-sb-native-stdin-pipe` |
| Stdout/stderr stream separation | Workspace-sandbox writes `{pid}.stdout.log` and `{pid}.stderr.log` separately; `/processes/{pid}/logs` and `/logs/stream` both require `?stream=stdout\|stderr`. The launcher opens two SSE streams (one per fd) and surfaces a real `Stderr()` reader on `LaunchedProcess`. | `execute/ws-sb-stdout-stderr-split` |
| Stream transport | The launcher now consumes Server-Sent Events from `/processes/{pid}/logs/stream`; chunks are pushed by the server's logWriter fan-out as bytes are written, with no client-side polling. The `PollInterval` field is removed. | `execute/ws-sb-streaming-process-logs` |
| Exit-code precision | The driver's wait reaper records structured `ExitInfo{ExitCode, Signal, OOMKilled}` via `Tracker.RecordExit`, and the SSE stream emits one `event: exit` carrying the JSON-encoded info. The launcher surfaces this as `*remoteExitError` with `ExitCode()`, `signal`, and `oomKilled` fields. | `execute/ws-sb-structured-exit-codes` |

## How protected mode is exercised

- `runner.Launcher` interface: `internal/adapters/runner/launcher.go`
- `runner.HostLauncher` (host path): `internal/adapters/runner/host_launcher.go`
- `sandbox.SandboxLauncher` (protected path): `internal/adapters/sandbox/sandbox_launcher.go`
- `launcherSelector` (shared routing seam): `internal/adapters/runner/launcher_selector.go`
- `buildEnvWrappedLaunchRequest` (shared LaunchRequest builder): `internal/adapters/runner/launch_request.go`
- `extractExitCode` (host/sandbox-uniform exit-code helper): `internal/adapters/runner/exit_code.go`
- Per-runner Execute + Continue + durable wiring:
  - `internal/adapters/runner/claude_code.go`
  - `internal/adapters/runner/codex_runner.go`
  - `internal/adapters/runner/opencode_runner.go`
- Wire-encoder that materializes `Behavior.Protected.GitAllowlist`:
  `internal/adapters/sandbox/wire_encoder.go` (covered by `wire_encoder_test.go`)
- Workspace-sandbox `/exec` git-allowlist enforcement:
  `scenarios/workspace-sandbox/api/internal/handlers/process.go` (`Exec`),
  tests in `process_git_allowlist_test.go`
- Workspace-sandbox `/processes` git-allowlist enforcement:
  same file (`StartProcess`), tests in `process_start_git_allowlist_test.go`
- Default-mode flip + safe override merge:
  `internal/orchestration/service.go::resolveSandboxConfig` plus
  `internal/domain/types.go::DefaultSandboxConfig`

### Test coverage

| Concern | Test file | Status |
|---|---|---|
| `Launcher` contract / `HostLauncher` behavior | `runner/launcher_test.go` | 7 tests, all pass |
| `launcherSelector` Pick + PickFor routing (Execute and Continue) | `runner/launcher_selector_test.go` | 11 tests, all pass |
| `claude_code` runner ↔ selector wiring | covered by selector tests + integration tests |  |
| `codex` runner ↔ selector wiring + protected/tracking routing + wrapper fallback | `runner/codex_runner_routing_test.go` | 6 tests, all pass |
| `opencode` runner ↔ selector wiring + protected/tracking routing + env-shim guard | `runner/opencode_runner_routing_test.go` | 6 tests, all pass |
| `SandboxLauncher` lifecycle (start, stream, kill, ctx-cancel, 403) | `sandbox/sandbox_launcher_test.go` | 5 tests + helpers, all pass |
| `Exec` git allowlist | `handlers/process_git_allowlist_test.go` | unchanged, pass |
| `StartProcess` git allowlist | `handlers/process_start_git_allowlist_test.go` | 4 tests, all pass |
| Default-config Mode = Protected | `domain/auditability_contract_test.go::TestDefaultSandboxConfig_LockedDefaults` | passes |

## Default-mode policy

`domain.DefaultSandboxConfig()` returns `Mode = SandboxModeProtected`.
This is the production default. Spawn surfaces should clone the
default and apply field-wise overrides on top.

`SandboxModeTracking` is the explicit operator opt-out for runs that
need full host capability (e.g. a `git push` after review, scraping a
remote URL during a research run, or spawn surfaces that legitimately
need to spawn the agent on the host). Set it explicitly per-spawn;
nothing defaults to it.

`RunMode.IN_PLACE` remains the full-bypass mode for the rare cases
where even tracking-mode auditability is wrong (e.g., agent-manager
developing itself, self-modifying scenarios).

`Orchestrator.resolveSandboxConfig` field-wise backfills `Mode` and
`NetworkMode` from the default after the override clone, so partial
inline configs (notably swarm-manager's acceptance-only overrides)
don't silently strip these fields back to the proto zero-value.

## How to extend the seam to a new runner

1. Replace the runner's `exec.CommandContext(...) + startManagedProcess(...)
   + cmd.StderrPipe + cmd.StdinPipe` block with:

   ```go
   launcher := r.selector.Pick(ctx, req)            // or PickFor for Continue
   launchReq := buildEnvWrappedLaunchRequest(
       "MY_RUNNER_AGENT_TAG", r.binaryPath, args,
       req.GetTag(), prompt, r.buildEnv(req), req.WorkingDir,
   )
   proc, err := launcher.Launch(ctx, launchReq)
   ```

2. Read from `proc.Stdout()` / `proc.Stderr()` instead of `mp.Stdout()`
   and `cmd.StderrPipe()`. Call `proc.ResetIdleTimer()` instead of
   `mp.ResetTimer()`. Call `proc.Wait()` instead of `mp.Wait()`.

3. Track in `r.launched[req.RunID]` (LaunchedProcess map) — there is
   no longer an `r.runs` *exec.Cmd map. `Stop()` consults `r.launched`
   only.

4. For wait-error handling, use `extractExitCode(err)` to handle both
   host (`*exec.ExitError`) and sandbox (`*remoteExitError`) cases
   uniformly.

5. Add a routing test paralleling `codex_runner_routing_test.go` to
   pin the protected-vs-host fork.

## Open follow-on work

None outstanding for this initiative as of 2026-04-28. The four
ws-sb-* items shipped together as part of the same change that flipped
the protected-mode trade-offs above. See the [executions](
../../swarm-manager/execute/) for each:

- `execute/ws-sb-stdout-stderr-split` — completed
- `execute/ws-sb-native-stdin-pipe` — completed
- `execute/ws-sb-streaming-process-logs` — completed
- `execute/ws-sb-structured-exit-codes` — completed
