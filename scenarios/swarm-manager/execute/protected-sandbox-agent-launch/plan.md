# Plan: protected-sandbox-agent-launch

**Initiative:** `protected-agent-sandboxing`
**Item:** `execute/protected-sandbox-agent-launch`
**Effort:** XL
**Authored:** 2026-04-27
**Completed:** 2026-04-28

## Purpose

Convert the three coding-agent runners (`claude_code`, `codex`, `opencode`) so that, when `SandboxConfig.Mode == Protected`, the agent process tree itself launches **through workspace-sandbox execution APIs** rather than as a host `exec.Command`. The `Provider.ExecProcess` seam already existed and was tested; the runners did not yet route through it. This work closed that gap, added the streaming/interactive surface the runners need, and flipped `Protected` to the default sandbox mode.

## Required Reading

```bash
prompt-manager skill read implementation-plan-authoring
```

- `scenarios/agent-manager/docs/PROTECTED_MODE_RUNNERS.md`
- `scenarios/agent-manager/docs/SEAMS.md` § Process Launcher
- `scenarios/workspace-sandbox/docs/EXECUTION_MODES.md`
- `~/.claude/projects/-home-matthalloran8-Vrooli/memory/project_agent_manager_network_isolation.md`

## Greenfield Declaration

Greenfield change. No backwards-compatibility shims, no feature flags, no parallel old/new code paths. The legacy `r.runs map[uuid.UUID]*exec.Cmd` field was removed from all three runners; `r.launched map[uuid.UUID]LaunchedProcess` is the single registry.

## Problem Statement

Workspace-sandbox already had `/exec`, long-running `/processes`, and interactive execution support. Agent-manager created the sandboxed workspace but still launched Claude/Codex/OpenCode directly in that merged directory. That was good enough for tracking, but it was not a contained execution model. Future guardrails (network isolation, git verb enforcement, resource limits) could not apply to the actual agent process tree because the agent ran on the host.

## Scope

In scope:

1. A protected-mode launch path in agent-manager that starts supported coding agents through workspace-sandbox process execution rather than direct host execution.
2. Compatibility for the runner patterns agent-manager already needs: streaming output, environment injection, working-directory control, cleanup.
3. A capability matrix for Claude, Codex, and OpenCode so protected mode does not silently regress one runner while helping another.
4. End-to-end tests proving protected-mode launch preserves the same sandbox auditability contract as tracking-first mode.
5. Default-mode flip: `DefaultSandboxConfig().Mode = SandboxModeProtected`, `NetworkMode = localhost`.

## Current Technical Context

Pre-work state:

- `Provider.ExecProcess` was built for short, fire-and-forget `/exec` calls.
- Each runner (`claude_code.go`, `codex_runner.go`, `opencode_runner.go`) had its own `exec.CommandContext + startManagedProcess + cmd.StderrPipe + cmd.StdinPipe` block, duplicated across Execute, Continue, and durable-transcript paths.
- `SandboxMode` defaulted to `SandboxModeUnspecified`; spawn surfaces had to opt in explicitly per call.
- `ContinueRequest` did not carry `SandboxID` or `ResolvedConfig`, so continuation runs could not re-pin to a previously-protected launcher.

## Target End State

Reached as of 2026-04-28:

- `runner.Launcher` interface (`internal/adapters/runner/launcher.go`) with `HostLauncher` and `SandboxLauncher` implementations.
- `launcherSelector.Pick(ctx, req)` and `PickFor(ctx, runID, cfg, sandboxID, sink)` route every runner path: streaming Execute, durable-transcript Execute, durable-transcript Continue, streaming Continue.
- All three runners route through the seam on every path. The legacy `r.runs *exec.Cmd` map is gone.
- `domain.DefaultSandboxConfig()` returns `Mode = SandboxModeProtected`.
- `Orchestrator.resolveSandboxConfig` field-wise backfills `Mode` and `NetworkMode` from the default after the override clone, so partial inline configs (notably swarm-manager's acceptance-only overrides) don't silently strip these fields.
- Wire-encoder materializes `Behavior.Protected.GitAllowlist`; workspace-sandbox `/exec` and `/processes` enforce the allowlist server-side and surface typed `*sandbox.LaunchBlocked` to the runner.

## Implementation Strategy

Delivered in 5 slices over the active period; details preserved here for posterity.

### Slice H — pilot

Introduced `runner.Launcher`, `HostLauncher`, `SandboxLauncher`, `/processes` git-allowlist enforcement. `claude_code.Execute` (streaming) routed through the seam.

### Slice 1 — claude_code/codex/opencode remainder

All three runners' durable-transcript paths (`runDurableCommand` / `runTranscriptCommand`) now take a `Launcher` + `LaunchRequest` instead of a raw `*exec.Cmd`. Stdout is `io.Copy`'d from the launcher's pipe into the transcript file in a goroutine; the previous `cmd.Stdout = file` direct-pipe pattern is gone. `transcript.OnProcessStart(pid, pid)` is fed by `proc.PID()`.

All three runners' streaming Continue paths route through `r.selector.PickFor` using `req.GetConfig()` and `req.SandboxID`. `ContinueRequest` gained `ResolvedConfig *domain.RunConfig` and `SandboxID *uuid.UUID` fields plus a `GetConfig()` method, mirroring `ExecuteRequest`.

### Slice 2 — codex/opencode streaming Execute

Streaming Execute paths of `codex` and `opencode` wired through the seam. `codex --full-auto` retained as defense-in-depth (memory: dual enforcement is intentional). `opencode` watchdog handoff uses `proc.Kill()` instead of `cmd.Process.Kill()`.

### Slice 3 — selector + builder

Extracted `launcherSelector` and `buildEnvWrappedLaunchRequest`. Eight selector routing tests replaced the per-runner copies.

### Slice 4 — default flip

`domain.DefaultSandboxConfig()` returns `Mode = SandboxModeProtected`. `Orchestrator.resolveSandboxConfig` starts from the default (instead of zero-valuing) and backfills `Mode` and `NetworkMode` after the override clone. `SandboxMode` doc comments rewritten.

### Slice 5 — polish

Fixed CLI fingerprint rebuild loop, SandboxMode doc-drift, added the SEAMS.md `1c. Process Launcher` section, tracked four follow-up items under `protected-agent-sandboxing` (the ws-sb-* items).

## Contract Decisions

1. **All three runners in one bundle.** Single execution; runners share enough surface that splitting them would have cost more than it saved.
2. **Default network mode = `localhost`.** Local resources reachable, wider internet blocked. Existing local-resource workflows keep working.
3. **`SandboxModeTracking`** stays as the explicit operator opt-out for runs that need full host capability (e.g. `git push`, scraping a remote URL during research). Set per-spawn; nothing defaults to it.
4. **`RunMode.IN_PLACE`** stays as the full-bypass mode for self-modifying agent-manager runs.
5. **No prompt or agent-side behavior changes.** The agent never knows it's in a sandbox. argv, env, working dir flow identically.
6. **Wait-error handling unified** via `runner.extractExitCode(err)` — understands both `*exec.ExitError` (host) and `*sandbox.remoteExitError` (sandbox) by satisfying `ExitCode() int`.

## Testing Plan

| Concern | Test file | Status |
|---|---|---|
| `Launcher` contract / `HostLauncher` behavior | `runner/launcher_test.go` | 7 tests, pass |
| `launcherSelector` Pick + PickFor routing | `runner/launcher_selector_test.go` | 11 tests, pass |
| `claude_code` runner ↔ selector wiring | covered by selector + integration tests | pass |
| `codex` runner ↔ selector wiring + protected/tracking + wrapper fallback | `runner/codex_runner_routing_test.go` | 6 tests, pass |
| `opencode` runner ↔ selector wiring + env-shim guard | `runner/opencode_runner_routing_test.go` | 6 tests, pass |
| `SandboxLauncher` lifecycle | `sandbox/sandbox_launcher_test.go` | 5+ tests, pass |
| `/exec` git allowlist | `handlers/process_git_allowlist_test.go` | pass |
| `/processes` (StartProcess) git allowlist | `handlers/process_start_git_allowlist_test.go` | 4 tests, pass |
| Default Mode = Protected | `domain/auditability_contract_test.go::TestDefaultSandboxConfig_LockedDefaults` | pass |

## Rollout/Validation Checklist

After landing the work the following gate was run and is the canonical re-check command:

```bash
cd scenarios/agent-manager/api && go build ./... && go test ./... -count=1 -timeout 300s && golangci-lint run --timeout 180s ./...
cd scenarios/workspace-sandbox/api && go build ./... && go test ./... -count=1 -timeout 300s && golangci-lint run --timeout 180s ./...
cd scenarios/swarm-manager/api && go build ./... && go test ./... -count=1 -timeout 300s && golangci-lint run --timeout 180s ./...
vrooli scenario restart workspace-sandbox agent-manager swarm-manager
```

## Risks + Mitigations

| Risk | Mitigation |
|---|---|
| Codex `--full-auto` and workspace-sandbox network isolation conflict | Memory note documents dual enforcement is intentional; routing tests confirm both layers active |
| Default-mode flip breaks an unforeseen consumer | `Orchestrator.resolveSandboxConfig` backfills `Mode`/`NetworkMode` field-wise so partial overrides (e.g., swarm-manager acceptance-only) don't strip them; default is always reachable |
| Watchdog races between `Handle.Kill()` and natural exit | `Wait()` returns idempotently after Kill; runner code already handled this for the host case |

## Non-goals/Prohibited Patterns

- Multi-runner per-run support (one run = one runner today).
- Replacing `RunMode.IN_PLACE`. Stays as the full-bypass mode.
- Live-migration of running tracking-mode runs to protected mode.
- Removing tracking mode entirely — kept for legitimate full-capability use cases.
- New agent runners (Cursor, Aider). Add later via the same `Launcher` pattern.
- Per-runner copy of routing logic. All routing goes through `launcherSelector`.

## Definition of Done

This item is done when:

1. ✅ All three runners route through `provider.LaunchProcess` (via `Launcher`/`launcherSelector`) when `SandboxConfig.Mode == Protected`. Runner-isolation routing tests pin the protected-vs-host fork.
2. ✅ Streaming, env, cwd, exit-code propagation all preserved across host and sandbox launches via `extractExitCode`.
3. ✅ `Protected` is the default sandbox mode; `NetworkMode = localhost` is the default network mode; `Tracking` is the documented operator opt-out.
4. ✅ Provenance shape identical between protected and tracking modes.
5. ✅ Git allowlist enforcement surfaces as a typed `*sandbox.LaunchBlocked`, not a corrupt run.
6. ✅ Tests in the table above all pass; `go build`/`go test`/`golangci-lint` clean across agent-manager, workspace-sandbox, swarm-manager.
7. ✅ Capability matrix in `PROTECTED_MODE_RUNNERS.md` filled for every runner / every path.

Open follow-on items (tracked separately under the same initiative):

- `execute/ws-sb-stdout-stderr-split`
- `execute/ws-sb-native-stdin-pipe`
- `execute/ws-sb-streaming-process-logs`
- `execute/ws-sb-structured-exit-codes`

## References

- Spec: `swarm-manager backlog get --kind execute --name protected-sandbox-agent-launch`
- `scenarios/agent-manager/docs/PROTECTED_MODE_RUNNERS.md`
- `scenarios/agent-manager/api/internal/adapters/runner/launcher.go`
- `scenarios/agent-manager/api/internal/adapters/runner/launcher_selector.go`
- `scenarios/agent-manager/api/internal/adapters/sandbox/sandbox_launcher.go`
- `scenarios/workspace-sandbox/api/internal/handlers/process.go` (`Exec`, `StartProcess`)
- `scenarios/agent-manager/api/internal/orchestration/service.go::resolveSandboxConfig`
- `scenarios/agent-manager/api/internal/domain/types.go::DefaultSandboxConfig`
