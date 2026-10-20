# Owned operation runtime deadline probe

This opt-in Linux probe qualifies the manager's real `RuntimeMaxSec` behavior
through `platform-go`'s public owned-operation API. It starts only the current
Go test executable with fixed test names and fixed fixture modes; the fixture
does not invoke a shell or load an imported script. Each operation intent is
written and synced before dispatch. Cleanup is restricted to the exact unit
returned for that persisted intent.

## Result

The initial probe exposed a real gap: `RuntimeMaxSec` killed a running main
process and its child but did not bound a descendant-only service lifetime
with the transient-service default. The smallest owner correction,
`ExitType=cgroup`, was added to the fixed service properties. Repeating both
modes now shows that the manager kills the descendant-only tree at its deadline
too. The existing manual-stop, fast-success, and failed-exec probes also remain
green. On this Linux manager, the shared owner has current-host deadline and
tree-cleanup evidence; other managers/platforms remain unqualified.

## Exact run evidence

Host: Linux `7.0.0-30-generic`, x86_64; user manager systemd `255`
(`255.4-1ubuntu8.18`). The host manager reports overall state `degraded`, but
the exact transient services below were accepted and queried normally.

Command:

```text
cd packages/platform-go && PLATFORM_GO_RUNTIME_MAX_PROBE=1 go test . -run '^TestOwnedOperationCurrentHostRuntimeMaxProbe$' -count=1 -v -timeout=20s
```

Before the owner correction, the desired-behavior probe intentionally failed
the descendant-only case:

```text
runtime-max probe mode=parent-running unit=vrooli-op-platform-go-1d49d7ec7c371d839e5c9ea6.service invocation=5f5020364c414535b5690adcfbfef205 child_pid=3880907 runtime_max=2s elapsed=2.122s result=timeout exit_known=true populated=false root_alive=false child_alive_after=false
runtime-max probe incomplete mode=parent-exits unit=vrooli-op-platform-go-530235372579f83eb5bc60d0.service invocation=b5f847c2748144d2b073fefcf49aebca child_pid=3881215 runtime_max=2s elapsed=7.01s wait_error=platform: inspect owned operation: systemctl manager omitted required operation field LoadState observation={Ref:{Intent:{Owner:platform-go AttemptID:runtime-max-parent-exits-1792467140035117851 InvocationID:e653e1d3159699e854281663ee6c821d SpecDigest:70ab858b78291a5744f41df6085f1e1cd8ad5dbc7766a8e2104a9f79c858f255} Unit:vrooli-op-platform-go-530235372579f83eb5bc60d0.service Scope:cgroup:/user.slice/user-1000.slice/user@1000.service/vrooli.slice/vrooli-operations.slice/vrooli-op-platform-go-530235372579f83eb5bc60d0.service RootPID:3881207 RootStartTime:411824765 NativeInvocationID:b5f847c2748144d2b073fefcf49aebca ScopeDevice:31 ScopeInode:6948636} State:running Populated:true RootAlive:false Result:success ExecMainCode:1 ExecMainStatus:0 ExitKnown:false} inspect_error=<nil> manager_state={LoadState:loaded ActiveState:active SubState:exited ControlGroup:/user.slice/user-1000.slice/user@1000.service/vrooli.slice/vrooli-operations.slice/vrooli-op-platform-go-530235372579f83eb5bc60d0.service InvocationID:b5f847c2748144d2b073fefcf49aebca Result:success Job: JobType: JobPath: MainPID:0 ExecMainCode:1 ExecMainStatus:0 ExitKnown:true HasPendingJob:false} manager_error=<nil> child_alive=true child_error=<nil>
    operation_runtimemax_linux_test.go:100: wait for manager-enforced deadline: platform: inspect owned operation: systemctl manager omitted required operation field LoadState
--- FAIL: TestOwnedOperationCurrentHostRuntimeMaxProbe (9.32s)
    --- PASS: TestOwnedOperationCurrentHostRuntimeMaxProbe/parent-running (2.22s)
    --- FAIL: TestOwnedOperationCurrentHostRuntimeMaxProbe/parent-exits (7.10s)
```

The context expiry cancels the last `systemctl show` used by `Wait`, so that
call's parser sees incomplete output. The independent manager query captured
immediately afterward is the decisive evidence: the unit remained
`active/exited`, its main PID was zero, its cgroup remained populated, and the
recorded child was alive. Test cleanup then stopped and released that exact
unit. Follow-up read-only checks returned `LoadState=not-found` for the exact
unit and no `/proc` entry for child PID `3881215`; no `vrooli-op-*` units were
left loaded.

After the owner correction, the same bounded probe passed:

```text
cd packages/platform-go && PLATFORM_GO_RUNTIME_MAX_PROBE=1 go test . -run '^TestOwnedOperationCurrentHostRuntimeMaxProbe$' -count=1 -v -timeout=20s
runtime-max probe mode=parent-running unit=vrooli-op-platform-go-e91ca1c3f2a995ba0d52e4cd.service invocation=b6ce71b437a9490e9441c5e7d54f54ac child_pid=3916524 runtime_max=2s exit_type=cgroup elapsed=2.03s result=timeout exit_known=true populated=false root_alive=false child_alive_after=false
runtime-max probe mode=parent-exits unit=vrooli-op-platform-go-92f75e5edae1be0e32f8734f.service invocation=be2db5397b9f401ebbb0a082c4bacfdb child_pid=3916765 runtime_max=2s exit_type=cgroup elapsed=2.134s result=timeout exit_known=true populated=false root_alive=false child_alive_after=false
PASS
```

The probe reads back both manager properties from each live exact unit before
waiting; `ExitType=cgroup` is observed in addition to the requested
`RuntimeMaxUSec=2s`.

The combined owned-operation regression probes passed after correction:

```text
cd packages/platform-go && PLATFORM_GO_RUNTIME_MAX_PROBE=1 PLATFORM_GO_OPERATION_PROBE=1 go test . -run '^TestOwnedOperationCurrentHost(RuntimeMax|Tree|Terminal)Probe$' -count=1 -v -timeout=25s
TreeProbe: PASS; captured direct and nested child PIDs were gone after exact-unit stop.
TerminalProbe: fast-success result=success status=0; failed-exec result=exit-code status=127; both quiescent and released.
RuntimeMaxProbe: parent-running PASS; parent-exits PASS; each RuntimeMaxUSec=2s, manager result=timeout, empty tree.
PASS
```

`cd packages/platform-go && go test -race . -count=1` also passed (`ok github.com/vrooli/platform-go`, 2.864s). After these
probes, read-only manager queries found none of their exact units loaded and
the test-recorded child PIDs had no `/proc` entries.

The opt-in probe's default, non-host path compiles and skips successfully:

```text
cd packages/platform-go && go test . -run '^TestOwnedOperationCurrentHostRuntimeMaxProbe$' -count=1
ok github.com/vrooli/platform-go
```

## Interpretation and next owner correction

**Observed:** the current fixed transient service sets a finite `RuntimeMaxSec`
and default service exit semantics. The host accepted `RuntimeMaxUSec=2s` and
enforced it while the main process was running. In the second case, the same
manager reported `ActiveState=active`, `SubState=exited`, `MainPID=0`,
`Result=success`, while recursive cgroup inspection and `/proc` confirmed a
live descendant.

**Inference from the pre-correction result:** `ExitType=main` (systemd's
documented default) ended the service's main-process lifecycle before the owned
process tree was empty, so its runtime deadline no longer bounded
descendant-only execution. The owner now sets `ExitType=cgroup` on the
transient service, keeping it active while any process remains in the cgroup.
The post-correction probe demonstrates that the manager deadline then kills
both service forms and retains timeout outcome evidence. Keep the fixed
`Type=exec`, `KillMode=control-group`, and persisted intent/ref checks. Do not
add a scenario watchdog or use `CPUWeight` as a wall deadline.

The host's installed `systemd.service(5)` says `ExitType=cgroup` keeps a service
running while any process in its cgroup remains, defaults to `main`, and was
added in systemd 250. It documents `RuntimeMaxSec` as terminating a service
that exceeds its active runtime; systemd-run's `--property=` sets properties on
the transient service. Thus a runtime owner relying on this correction needs a
manager with `ExitType=cgroup` support (systemd 250 or newer), or must return
an explicit unsupported capability. This is documentation support, not proof
that the corrected owner path works; the runtime-limit-plus-descendant probe
must pass on a supported manager before admission.

References: [systemd.service(5), ExitType and RuntimeMaxSec](https://www.freedesktop.org/software/systemd/man/253/systemd.service.html),
[systemd transient unit settings](https://github.com/systemd/systemd/blob/main/docs/TRANSIENT-SETTINGS.md).
The host command `man -P cat systemd.service` reported systemd 255 and the
same version boundary locally; `man -P cat systemd-run` confirmed transient
service properties are supplied through `--property=`.

## Reproduction boundaries

- Probe source: `packages/platform-go/operation_runtimemax_linux_test.go`.
- Prewrite snapshot: [prewrite.txt](prewrite.txt), which records that the test
  source and this evidence directory were absent before this probe was added,
  plus hashes of the pre-existing host probe and process-owner README.
- Production-owner prewrite hashes: [owner-prewrite.txt](owner-prewrite.txt).
- The pre-correction parent-exits failure is retained above. The corrected
  owner passes both cases on systemd 255.
- The shared platform operation owner was changed per the admitted P1 correction
  to require `ExitType=cgroup`; no Device Studio worker/product integration
  changed. No process outside the exact probe units was targeted, and all
  created units were confirmed removed afterward.
- This is Linux user-manager evidence only. It is not Device Studio integration,
  cross-platform qualification, or generated-code sandbox evidence.
