# Native service definitions

Every long-lived process the control plane or a scenario installs into a
host's native scheduler renders from one typed description:
`platformgo.ServiceDefinition` in `packages/platform-go/servicedef.go`. Nothing
on the boot path writes a unit body by hand. This page is the contract; the
renderers and validators are the implementation.

## The contract

| Field | Meaning |
|---|---|
| `Name`, `Label` | systemd unit base name; launchd reverse-DNS label. Windows task and service names come from `CoreUnit.Windows`. |
| `Executable`, `Args` | absolute path plus one token per argument. Argv for `vrooli` comes from the `cliinvoke` catalog and carries no global flags. |
| `Env` | rendered in key order so a unit compares byte-for-byte across runs. |
| `Kind` | `daemon`, `oneshot`, or `timer` (a timer renders its service beside it). |
| `Restart` | mode, delay, and a burst limit inside a window. |
| `OnFailureUnit` | systemd `OnFailure=`; ignored by other targets. |
| `Scope` | `user` or `system`. A user unit is wanted by `default.target`; `multi-user.target` and `network-online.target` exist only in the system manager. |
| `Protections` | `CPUWeight`, `MemoryMin`, `OOMScoreAdjust`: the supervisor must outlive the pressure it reports. |
| `Username` | the principal; required for the Windows task, rendered for system-scope units only. |
| `Logs` | stdout and stderr files; empty means the manager's default. |

`Validate()` rejects what no renderer can load: a missing name or executable, a
relative executable, a newline in any string, a timer without a schedule of at
least one minute, a `DocumentationURL` that is not an `https://` URL.

## Renderers and validators

| Target | Renderer | Validator | Verdict states |
|---|---|---|---|
| Linux | `RenderSystemd` (`render_systemd.go`) | `systemd-analyze --user verify` (or `--system`) | `accepted`, `unavailable`, `rejected` |
| macOS | `RenderLaunchd` (`render_launchd.go`) | `plutil -lint` | same |
| Windows | `RenderWindowsTaskXML` (`render_windowstask.go`) | XML namespace and principal check on every host | same |

`RenderDefinition(d, target)` dispatches on the target token; `NormalizeTarget`
accepts `linux`, `darwin`, `macos`, and `windows`. Every installer calls
`ValidateArtifact` before it enables an artifact and stores the `Verdict` in the
safeguard's `evidence.validator_verdict`. A rejected render never replaces a
working unit. An unavailable validator is recorded as a note: unproven is not
accepted.

Quoting is per directive and the tests assert it against systemd itself
(`TestSystemdRenderPassesSystemdAnalyze`): `Environment=` and `ExecStart=` take
quoted values, `WorkingDirectory=` and the log paths take the rest of the line
verbatim.

## The PATH table

`DefaultPath(target, home)` is the one PATH a rendered unit carries. Linux and
macOS entries include `/opt/homebrew/bin`, `/usr/local/go/bin`,
`/usr/local/bin`, `$HOME/.cargo/bin`, `$HOME/go/bin`, `$HOME/.local/bin`,
`$HOME/bin`, and the runtime-home bin entry from repo-contract, then the
system directories. The Go toolchain entry matters: the autoheal recovery
floor runs `go mod download` from inside the unit, and a PATH without it
burned a breaker slot per attempt on 2026-09-02. `internal/hostreqkit` consumes
this table; it no longer carries its own.

## Core units

`CoreUnits()` is the only list of "our units". Every probe that needs the set
derives it from here (`CoreDaemonUnits`, `CoreSystemdUnits`, `NativeName`).

| ID | Kind | systemd | launchd | Windows | Owner |
|---|---|---|---|---|---|
| `autoheal-loop` | daemon | `vrooli-autoheal.service` | `com.vrooli.autoheal` | `VrooliAutoheal` | `internal/safeguards/autoheal-watchdog/handler.go` |
| `runtime-supervisor` | daemon | `vrooli-runtime-supervisor.service` | `com.vrooli.runtime-supervisor` | `VrooliRuntimeSupervisor` | `internal/runtimesupervisor/service_install.go` |
| `emergency-watchdog` | oneshot | `vrooli-emergency-watchdog.service` | `com.vrooli.emergency-watchdog` | `VrooliEmergencyWatchdog` | `internal/safeguards/emergency-watchdog/handler.go` |
| `emergency-watchdog-timer` | timer | `vrooli-emergency-watchdog.timer` | shares the service plist | shares the task | same |

The definitions live beside the type: `WatchdogDefinition`,
`RuntimeSupervisorDefinition`, `EmergencyWatchdogDefinition`. The bridge agent
(`scenarios/vrooli-bridge/agent/internal/service`) keeps its own install API
and projects onto the same type, so it is not a core unit but renders the same
way. `Documentation=` is always
`https://github.com/Vrooli/Vrooli/blob/master/<owner path>`.

Each core unit is converged by a safeguard on every `vrooli setup` and
re-inspected by `vrooli setup status --json --phase readiness`:
`autoheal_watchdog`, `runtime_supervisor`, `emergency_watchdog`. See
[host safeguards](../configuration/host/safeguards.md).

## Writers left for a later consolidation

These write native unit text outside the seam. They are drop-ins, oneshot
timers, or root-owned system units with no boot-recovery role, so they were
left as they are. Any new long-lived unit must use `ServiceDefinition`.

| Writer | What it writes | Why it is not on the seam yet |
|---|---|---|
| `internal/safeguards/remote-session-protection/handler.go:36,439` | a `[Service]` drop-in and a system unit | drop-in fragments have no definition shape |
| `internal/safeguards/keyring-daemon-limits/handler.go:73` | a `[Service]` drop-in for `gnome-keyring-daemon` | drop-in fragment |
| `internal/privilegebroker/install.go:142` | the root-owned broker system unit | system scope through the privileged installer |
| `internal/safeguards/nvidia-driver/handler.go:55` | a root-owned oneshot | system scope |
| `internal/safeguards/pstore-observability/handler.go:338,350` | root-owned collector service and timer | system scope |
| `internal/safeguards/kdump-observability/handler.go:400,412` | root-owned collector service and timer | system scope |
| `internal/resources/securestore/schedule_linux.go:43`, `schedule_darwin.go:45` | credential-store copy timer and LaunchAgent | user-scope timer; the first candidate to migrate |

The vendored copy under `packages/proto/vendor/github.com/vrooli/platform-go`
is refreshed with `make -C packages/proto refresh-vendor`; do not edit it.
