# E2/E3 native operation owner decision

Owner implementation evidence, 2026-10-20. No FFmpeg or Blender renderer was launched, no tool was installed, no Device Studio runtime integration was changed, and no foreign-platform validation was run. The initial `process-owner/` directory was absent before the first evidence write; `prewrite.sha256` retains the source baseline.

## Decision

Do not dispatch E2/E3 render children through `scenario-to-desktop/runtime/infra.RealProcessRunner` as-is, and do not relax platform-go's service/test/agent scope guards. Extend the shared `packages/platform-go` owner with one narrow, typed **owned operation** surface, then let Device Studio jobs call it. This is the smallest boundary that can join the existing durable job attempt to an OS-enforced process tree without introducing a scenario-private supervisor.

`packages/platform-go/operation.go` and `operation_linux.go` now expose `NewOperationIntent`, `StartOwnedOperation`, `LookupOwnedOperation`, `InspectOwnedOperation`, `StopOwnedOperation`, and `ReleaseOwnedOperation`. The caller creates and durably stores intent before dispatch; it binds owner namespace, never-reused attempt/invocation IDs, and a digest of the complete resolved executable/argv/env/working-directory/resource spec. The owner launches a retained transient service in `vrooli-operations.slice`, requiring `systemd-run`, a manager `InvocationID`, finite `RuntimeMaxSec`, and `ExitType=cgroup`; a manager that rejects the required property fails the start, with no fallback. `ExitType=cgroup` requires systemd 250 or newer. Start waits for systemd's `service-type=exec` running acknowledgment or an explicit, consistent terminal manager result; a pending `Job`, incomplete state fields, and ambiguous dispatch cannot prove start or termination. A terminal service whose `ControlGroup` is empty is recoverable only through the exact manager `InvocationID` and the dedicated operations slice's manager-reported cgroup path plus deterministic unit name. Linux refs bind the exact unit and invocation, plus PID/start time and cgroup device/inode when a main process still exists. Missing manager identity is an unknown outcome, never gone. `StopOwnedOperation` signals only the exact service cgroup, sends SIGTERM then bounded SIGKILL, and reports native quiescence only when the manager is in a consistent terminal state with no pending job, MainPID zero, and recursive `cgroup.events populated` zero, or when the exact cgroup directory is gone under that terminal manager evidence. A missing `cgroup.events` inside an existing directory is unknown/error, never empty. An activating service with no process is canceled through the exact manager unit before quiescence can be reported. Launcher reap is separate; no local render-root reap is claimed. `ReleaseOwnedOperation` is explicit and only for a terminal service with known exit evidence and an empty tree after the caller persists output/job evidence; it destroys manager-side recovery evidence and clears a retained failed unit after checking the same invocation.

The implemented Linux path uses systemd transient **services**, not `ContainedCommand` scopes, so the manager retains terminal `Result` and `ExecMainStatus` for short work while the service remains loaded. The fixed `systemd-run --no-block --remain-after-exit --service-type=exec` launcher starts `/usr/bin/env -i <explicit entries> <absolute executable> <fixed argv>` in the service. It sets memory/task/CPU ceilings, `RuntimeMaxSec`, `ExitType=cgroup`, `KillMode=control-group`, a bounded stop grace, and journal stdout/stderr rate limits. Current-host probe evidence covers both a still-running main process and a main-exits/descendant-remains service; details are in `runtime-max/README.md`. Environment values are not included in evidence receipts and are not a credential channel because manager arguments/properties may expose them to the same user. The package does not read back journal output. The host fixture exercised lookup by pre-persisted intent in the same test process; a real Device Studio API crash/restart adapter remains unimplemented. Do not generalize `StopScopeContext` or change its agent/test/service guards. macOS and Windows operation adapters return `ErrUnsupported`; shared contracts exist, but the operation path is not cross-platform qualified.

Persisted job/attempt intent is the pre-spawn ownership record. A crash between intent and process start reconciles as no operation only if the owner proves no matching unit exists. A crash after native start but before the API stores the returned ref can be looked up by deterministic unit name derived from the persisted intent. InvocationID is checked against the exact manager unit; PID reuse is additionally checked with process start time while a main process is live. No process-name or ancestry-only matching. After caller-confirmed output persistence, `ReleaseOwnedOperation` unloads the service and intentionally ends manager recovery.

## Existing owner evidence

The real RuntimeMaxSec probe initially found that the default service exit
mode dropped the runtime deadline when the main process exited but descendants
remained. The shared Linux owner now configures `ExitType=cgroup`; the corrected
host probe confirms deadline termination and empty-tree evidence for both a
running main process and a descendant-only service. Exact pre/post-correction
manager evidence, systemd version notes, commands, and test results are in
[`runtime-max/README.md`](runtime-max/README.md). This qualifies the tested
Linux user manager only; it does not establish support on other platforms.

- `packages/platform-go/containment.go:18-27` describes the shared native containment design and evidence status: Linux host-verified; macOS/Windows compile/fixture-verified.
- `packages/platform-go/containment.go:96-147` exposes `ContainedSpec` / `ContainedCommand`; it accepts executable path, argv, env, dir, scope name and limits, but does not take a durable owner/job record. `StartContext` (`:154-175`) performs start and then post-start acquisition.
- `packages/platform-go/containment.go:178-213` already treats failed scope acquisition as unverified, attempts bounded owner cleanup, and waits for the root. This is the right shared layer to extend, not copy into Device Studio.
- `packages/platform-go/containment.go:267-285` explicitly says Linux stop is guarded to agent/test slices and macOS/Windows return unsupported. Keep those protections intact.
- `packages/platform-go/containment_linux.go:32-76` is still the existing scope runner with direct-start cgroup-write fallback at `:78-143`. The new operation path instead uses `systemd-run` transient services in `operation_linux.go` and refuses fallback; it does not weaken existing containment guards.
- `packages/platform-go/containment_linux.go:474-521` limits stop to the guarded `freezableCgroup` path, asks systemd to stop or uses `cgroup.kill`; `:561-582` reads exact cgroup membership. Existing tests cover agent/test scopes and bounds, not a production operation owner: `containment_linux_test.go:32-76,79-144`.
- `packages/platform-go/containment_darwin.go:16-26,153-169` uses process groups and per-process rlimits but explicitly has no stop implementation. `packages/platform-go/containment_windows.go:15-19,188-212` describes Job Object limits but process enumeration/stop are unsupported. `stopscope_unsupported_test.go:10-16` pins honest unsupported behavior.
- `scenarios/scenario-to-desktop/runtime/infra/process.go:11-27,39-60,70-107` is a useful portable start/wait/root signal abstraction. It uses `exec.CommandContext`, starts first, then `assignProcessContainment`; `Kill()` targets the root handle only. It has no durable pre-spawn intent, native scope lookup, or descendant-empty proof. `process_reaping_unix_test.go:16-46` proves one process-group stop fixture on Unix; it does not establish process cleanup/recovery across supported targets.
- `docs/concepts/ARCHITECTURE.md:162-179` says durable ownership records outrank ancestry and require PID plus start-time match; apply the same rule to render operation recovery.

## Tool resolution and closure

`internal/tools/ffmpeg/tool.json` declares FFmpeg `bundling: host-required`, but marks Linux, macOS, and Windows `unqualified`; packages list Homebrew and winget only. This is a tool-catalog entry, not a pinned renderer profile or a packaged executable closure. Image Tools `.vrooli/service.json:239-257` only declares optional host FFmpeg for format conversion. It does not give Device Studio a qualified video decoder/encoder profile.

There is no `internal/tools/blender/tool.json`. Device Studio's retained v8 probe is historical evidence only: `renderer-design/README.md:116-143` records Blender 4.0.2+dfsg-1ubuntu8 and FFmpeg 6.1.1-3ubuntu5 locally, but no Blender GPU kernel payload and only CPU enumeration. It is neither managed package closure nor GPU qualification.

Scenario Dependency Analyzer currently has a deliberately closed host-tool install surface for `npm/hyperframes` (`scenarios/scenario-dependency-analyzer/api/internal/installgateway/host_tool.go:15-31`); its existence does not mean arbitrary FFmpeg/Blender provisioning is available. The root `internal/tools/ffmpeg` catalog remains the right owner for a future governed profile, while scenario-to-desktop owns signed app/bundle packaging. Do not install or resolve from ambient PATH in a production job. Until those owners provide a pinned binary/profile digest, OS/architecture closure, license manifest, and packaged-runtime path, video export and Blender rendering must report unavailable rather than use a developer's host binary. PNG stills can remain Go-native.

## Safe E2/E3 command boundary

**E2 fixed native argv:** compile a typed render request into a fixed, allowlisted FFmpeg command builder. Never accept command strings, shell snippets, user-supplied flags, arbitrary filter expressions, or output paths from a job payload. Resolve executable/profile through the governed tool owner; use immutable input refs staged into a per-attempt workspace, explicit output path, fixed non-secret environment, wall deadline, output/temp quota and memory/task ceilings. Avoid shell invocation. The owner routes stdout/stderr to the systemd journal with per-service rate limits; it offers no readback API, so output artifacts and render receipts must be explicit job-owned files. Use the operation owner for dispatch, cancellation, manager lookup, and verified descendant exit. Keep Go PNG encode separate; FFmpeg is only needed for supported video decode/encode.

**E3 fixed shipped runner:** invoke the pinned Blender executable with package-owned fixed argv and a shipped, reviewed adapter script consuming a validated data-only scene recipe. Disable Blender autoexec and reject arbitrary `.blend`/Python imports. Disable GPU selection until an exact bundled backend is enumerated and a representative render is qualified; offer explicit CPU only if its profile is present and visual/performance scope passes. Blender autoexec-off and resource ceilings do not sandbox Python or prevent file/network access. Do not enable generated/imported scripts or arbitrary scene files until a shared cross-platform sandbox can enforce read-only runtime/input mounts, one writable output workspace, network denial, bounded storage/resources, and auditable cleanup. The agent workspace sandbox is not a desktop render-job sandbox and should not be repurposed.

## Minimal implementation scope and tests

1. Implemented owner code is scoped to `packages/platform-go/operation*.go`; existing generic stop guards are unchanged. Intent is pre-dispatch, owner+attempt namespace is validated, direct cgroup-write fallback is refused, and no scheduler/job DB was added.
2. Focused package tests cover spec digest/limits/redaction, fixed no-shell argv and runtime bounds, malformed manager status, recursive population parsing, ref/unit/invocation/slice mismatch, mismatch stop refusal, and unsupported adapters in source. `go test .` passes on the current Linux host.
3. One bounded current-host fixture is present behind `PLATFORM_GO_OPERATION_PROBE=1`. It ran a `/bin/sh` fixture that exited after spawning a direct `/bin/sleep` and a nested shell/sleep. Lookup recovered its retained invocation; inspect saw the cgroup populated after the main shell exit; stop verified both PIDs gone; release unloaded the exact retained unit. This is not an FFmpeg/Blender qualification, an API restart test, or cleanup proof for other platforms.
4. Add job-owner integration tests that persist attempt before dispatch, kill/cancel across API disconnect/restart, reject success until full descendant quiescence plus output verification, and reconcile dispatch-before-ref crash boundaries. Keep these separate from process-owner unit tests.
5. Add fixture-only argv-builder tests for metacharacters/path edge cases, input/output containment, bounded flags, fixed shipped Blender script identity, disabled arbitrary scene/script import, CPU/GPU no-silent-fallback and missing-profile unavailability. Do not launch Blender/FFmpeg in generic tests; reserve one bounded actual encoder/renderer integration proof after a governed profile and operation owner pass.

## Search and verification record

Required discovery was attempted first. `search-hub query "run local FFmpeg and Blender subprocesses with cancellation, descendant cleanup, restart recovery" --type library,skill,command` failed before query execution because CLI build admission denied host memory PSI above the SB20 setpoint. The mandated one-time fallback, `prompt-manager discover ... --type all`, failed for the same admission condition. Repository source inspection proceeded independently; these outages do not establish absence beyond the specific code/config paths inspected here.

Read-only commands used (repository root):

```text
search-hub query "run local FFmpeg and Blender subprocesses with cancellation, descendant cleanup, restart recovery" --type library,skill,command
prompt-manager discover "run local FFmpeg and Blender subprocesses with cancellation, descendant cleanup, restart recovery" --type all
sed -n '1,260p' scenarios/device-studio/docs/internal/goal/epochs/E2.md
sed -n '1,260p' docs/concepts/ARCHITECTURE.md
sed -n '1,280p' scenarios/device-studio/docs/domains/rendering.md
sed -n '1,280p' docs/TESTING.md
rg -n "ProcessRunner|StartProcess|RunCommand|StopScopeContext|Kill\\(" packages/platform-go scenarios/scenario-to-desktop/runtime -g '*.go'
rg -n '"name": "ffmpeg"|ffmpeg.*host-tool' scenarios/scenario-dependency-analyzer -g '*.json' -g '*.yaml' -g '*.go' --glob '!*.freshness.json'
```

Exact focused verification and host-probe evidence:

```text
cd packages/platform-go && go test .
ok github.com/vrooli/platform-go

cd packages/platform-go && PLATFORM_GO_OPERATION_PROBE=1 go test . -run '^TestOwnedOperationCurrentHost(Tree|Terminal)Probe$' -count=1 -v
operation probe unit=vrooli-op-device-studio-a8d9c7b4552c806e61788146.service native_invocation=f6fbf6c8d46543e5904770f98efedaa0 children=[3519747 3519748] exit_known=true result=success quiescent=true
operation terminal probe case=fast-success unit=vrooli-op-device-studio-bd0588312bd4c56343f23a7a.service invocation=d8f4a8b127364cd7963b06094677c92e result=success exec_status=0 quiescent=true
operation terminal probe case=failed-exec unit=vrooli-op-device-studio-4305d241c2cb92d9fa08c8af.service invocation=6da2d9ed09d54d5dab2bc120f5de3606 result=exit-code exec_status=127 quiescent=true
PASS
```

`cd packages/platform-go && go test -race .` also passed. Post-probe read-only checks confirmed the exact descendant unit was `LoadState=not-found`, `ActiveState=inactive`, both captured PIDs had no `/proc/<pid>/stat`, and `systemctl --user list-units --all 'vrooli-op-*'` returned no units. `/usr/bin/true` returned retained `success`, status 0; a deliberate nonexistent executable returned `exit-code`, status 127. Both had empty process trees and were released. The manager's real `not-found` response omits `JobType` and `JobPath`; the parser requires `LoadState`, `ActiveState`, `SubState`, `ControlGroup`, `InvocationID`, `Result`, `MainPID`, exec status fields, and `Job`, treating `JobType`/`JobPath` as optional. A regression confirms a recovered pending job is addressable by persisted intent yet cannot imply quiescence; cancellation uses a fake manager sequence because a naturally queued user-service job could not be produced safely in this bounded host probe. This is current-host Linux owner evidence, not Device Studio E2 acceptance or arbitrary generated-code sandbox proof.
