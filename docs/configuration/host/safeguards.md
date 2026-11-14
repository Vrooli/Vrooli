# Host Safeguards

## Log-volume bounds and disk watchdog

`log_volume_bounds` limits the host log store and reports a platform verdict.

`harness_file_ownership` hands root-owned coding-agent files back to the operator,
which is what `sudo vrooli setup --include-optional` does for an agent in the
`needs_setup` state. It changes ownership only and never follows a symlink. It acts
only on paths whose parent resolves inside the invoking user's home, and ignores
`GROK_HOME`. See
[Coding-agent updates](tools.md#coding-agent-updates).

`generated_tree_ownership` does the same for the repository's generated Proto trees
(`packages/proto/gen`, the replaced copy `gen.previous` and staging directories). A
root-run materialization can leave root-owned files there that an unprivileged
`make build` cannot replace. Setup is the one place sudo runs, so setup repairs them. It
changes ownership recursively and never follows a symlink. Materialization and
harness installs also hand their files back to the directory's owner when they run as
root, so the repair is a backstop rather than the normal path.
On Linux it configures the supported rotation and rate-limit controls. On
macOS and Windows the current handler reports `unsupported` because native
log-store controls are not implemented yet.

The Linux `host_hardening` safeguard owns the journald drop-in. It explicitly
sets `SystemMaxUse=500M`, `RuntimeMaxUse=100M`, and `MaxRetentionSec=30d`, in
addition to the journald rate limit. The control plane renders and applies
these settings; scenarios only observe the resulting host state.

The emergency watchdog is installed and managed as a Go binary. It checks the
configured free-space floor and reports pressure through the typed storage
client. It does not shell the `vrooli` CLI.

Safeguards are idempotent host-state modifications Vrooli applies to make the host suitable for running its workload — kernel parameters, DNS configuration, firewall rules, NAT protections, TCP tuning, clock synchronization. They differ from host tools in that they *change the host's state* rather than just installing a binary, so they are explicitly opt-in with a risk indicator.

## What lives where

| Concern | File | Field |
|---|---|---|
| What the safeguard does, what it modifies, how to verify | `internal/safeguards/<name>/safeguard.json` | top-level manifest |
| Risk indicator | `internal/safeguards/<name>/safeguard.json` | `risk` (`low` / `medium` / `high`) |
| Capability invariants | `internal/safeguards/<name>/safeguard.json` | `invariants[]`, each with its own applicability and evaluation kind |
| Go handler that implements `Inspect` and `Apply` | `internal/safeguards/<name>/*.go` (registered in `internal/runtime/registry.go`) | `customSafeguardHandlers` map |
| Top-level project requirements | `.vrooli/service.json` | `hostSafeguards[]` (each entry: `hostRequirement`) |
| Per-scenario requirements | `scenarios/<name>/.vrooli/service.json` | `hostSafeguards[]` |
| Per-resource requirements | `resources/<name>/resource.json` | `hostSafeguards[]` |
| Operator opt-in | `~/.vrooli/state/vrooli/vrooli-onboarding/operator-state.json` | `host_safeguards.<name>.opted_in` |

## Native bubblewrap namespaces

`bubblewrap_userns` owns the AppArmor opt-in for `/usr/bin/bwrap` and
`/bin/bwrap`. It is separate from `workspace_sandbox_userns`, which owns the
Workspace Sandbox service launcher. Codex declares the native safeguard as a
host requirement. Neither safeguard disables global AppArmor restrictions.

Inspect with `vrooli host safeguard bubblewrap_userns --dry-run --json`.
Apply with `vrooli host safeguard bubblewrap_userns --sudo-mode ask --json`
from an operator terminal. Before the new CLI is installed, use
`go run ./cmd/vrooli host safeguard bubblewrap_userns --sudo-mode ask --json`
from the repository. Enter the sudo password only in that terminal.

The handler probes actual user, mount, PID and network namespaces. It grants
user namespaces only to the system bubblewrap executable through
`/etc/apparmor.d/vrooli-bubblewrap`. Existing working distribution policy needs
no rewrite. An unfamiliar file at the managed path is not overwritten. Root
setup probes as the identified invoking user; root without that identity is
unverified. macOS and native Windows are not applicable.

Read `applied` and `execution_state` for the safeguard verdict. A successful
repair marks an earlier namespace-probe error as `resolved pre-repair failure`
only after the post-repair probe passes. Older output can contain the raw
pre-repair error beside the success note. Run the read-only inspection above
to verify the current state; do not infer a remaining failure from that old
note. A failed post-repair probe reports `execution_state: failed`, not applied.
The generic `installed` field describes tool installation, not this safeguard.

After applying, run the no-model containment check from the repository:

```bash
go test ./internal/safeguards/bubblewrap-userns -run '^TestLiveNativeCodexSandbox$' -count=1 -v -args -live-native-codex-sandbox
```

It verifies allowed workspace writes, outside/symlink write denial, read-only
mode and host-loopback denial. It removes only its temporary fixture. This is
not qualification of agent tool hooks, remote tools or an entire delivery team.
The profile persists across reboot; a reboot test is separate evidence. To
unapply, an operator must unload the exact profile with `apparmor_parser -R`
and remove `/etc/apparmor.d/vrooli-bubblewrap`; do not change the global sysctl.

## Repository-owned user cron

User cron is host automation, so the control plane owns its declaration and
audit. Add every Vrooli-owned entry to the root `.vrooli/service.json` under
`hostCron`. Do not add a repository path directly to `crontab` without this
declaration.

```json
"hostCron": [
  {
    "name": "example-maintenance",
    "schedule": "15 3 * * *",
    "target": "tools/example-maintenance"
  }
]
```

`target` is a repository-relative path. It must exist, remain inside the
repository, and appear in the installed command with the declared schedule.
After adding or changing a declaration, install the corresponding user-cron
line through the host's normal `crontab` interface, then run:

```bash
vrooli host cron audit
vrooli host cron audit --json
```

The read-only audit reports a declared target that does not exist, a declared
entry that is not installed, and an installed entry that points inside the
repository without a matching declaration. On a platform without the
`crontab` command, it reports `unsupported`; it does not treat missing host
support as a clean audit. The command never installs, removes, or executes a
cron entry.

## How safeguards are discovered

Same drift-protected pattern as tools (see [`tools.md`](tools.md)):

- Filesystem at `internal/safeguards/<name>/safeguard.json` is the canonical list.
- `customSafeguardHandlers` map in `internal/runtime/registry.go` maps handler names to Go constructors.
- The invariant test `TestSafeguardManifestsReferenceRegisteredHandlers` ensures every manifest's `handler` field has a registered constructor.

Onboarding consumes the filesystem registry directly.

To inspect the complete registry from the control plane, including capability,
role, declared platforms, and whether this host has been sampled, run:

```bash
vrooli host safeguard list
vrooli host safeguard list --json
```

The list reports `host_not_sampled` when no handler inspection has been run;
that is intentionally distinct from an applied or failed observation.

## Deployment classification

Safeguards declare the same deployment axes as tools; their canonical meanings
live in [`deployment-contract.md`](../../resources/deployment-contract.md#deployment-eligibility-axes).
Every safeguard declares `privilege` and `bundling`, plus an explicit desktop
deployment profile. A safeguard normally has `privilege: elevated` and
`bundling: prohibited`: it exists to change host state and therefore cannot be
shipped inside a Tier 2 desktop application. Its desktop profile explains that
unsupported status to the operator.

## Platform status tokens

`platform_status` records the implementation state for each host operating
system. Use the tokens as follows:

- `not_implemented` means the capability may apply to the operating system,
  but its provider is not written yet. Count this token as portability debt.
- `not_applicable` means the capability cannot apply to the operating system
  or its host mechanism does not exist there. Treat this token as closed work.

Do not use `unsupported` for a case that is genuinely not applicable. To see
the remaining portability debt, run:

```bash
vrooli host safeguard portability-backlog
vrooli host safeguard portability-backlog --json
```

This is intentionally separate from `risk`. `privilege` is the machine gate
that tells Vrooli whether elevated setup is needed. `risk` is the human-facing
impact label that helps an operator decide whether to opt into a host-state
change. Neither field substitutes for the other.

## Capability invariants

An invariant is an OS-neutral statement about a capability's coupled host
state, declared beside the safeguard that owns the capability. The shared
shape is defined once in `.vrooli/schemas/common.schema.json` and referenced
by `safeguard.schema.json`; there is no second invariant registry. The control
plane aggregates declared sites and reports both `invariants_declared` and
`invariants_evaluated`, including every registered site that could not be
walked. A missing evaluation is coverage evidence, not a zero-invariant pass.

Invariant providers return ordered verdicts: `satisfied_structurally`,
`satisfied`, `undetermined`, `not_applicable`, `not_implemented`, or `failed`.
`undetermined` is not a failure and must retain its reason; callers must not
coerce it into a boolean. Provider-specific package-manager derivation stays
behind the provider seam, while the declaration names only the coupled
capability and invariant kind.

The NVIDIA safeguard is the reference implementation. It declares the
running-kernel module-loadability invariant and the coupled-update atomicity
invariant, and owns the apt policy file under `storage.entries`.

## Safeguard operation risk classes

Handlers classify operations independently from the manifest's human-facing
`risk` label:

| Operation class | Meaning | Consent gate |
|---|---|---|
| `restore_absent` | Restore a missing capability without replacing a live member | operator consent |
| `reload_live` | Replace or reload a live capability member | operator consent + maintenance window |
| `boot_critical` | Change boot/initramfs/kernel state | operator consent + maintenance window and reboot verification |

The setup renderer reports the exact resumable command for a maintenance
window and the reboot-and-verify sequence for a reboot-required result.

## The `risk` field

New as of the configuration substrate work. Operator-facing risk indicator that informs the wizard's display.

| Value | Meaning | Examples |
|---|---|---|
| `low` | No system state changes outside Vrooli's tree. Probes, reads, soft validations | `clock` (clock probe and sync) |
| `medium` | Writes config files outside Vrooli's tree, modifies networking rules, or grants the operator account a host privilege | `dns_resolution` (writes `/etc/systemd/resolved.conf.d/`), `docker_host_firewall` (iptables rules), `nat_protection` (iptables), `tpm_credential_access` (adds the operator account to the TPM device group) |
| `high` | Modifies kernel parameters or requires root in ways that broadly affect host behavior | `kernel_config` (writes `/etc/sysctl.d/99-vrooli.conf`) |

The wizard's host step renders this as a column next to each safeguard so operators can decide informed. Required safeguards (per the consuming manifest's `required: true`) bypass the opt-in but still display risk.

## hostRequirement shape

Identical to tools — a safeguard reference is a `hostRequirement` entry in a `hostSafeguards[]` array. See [`tools.md`](tools.md#hostrequirement-shape) for fields. The same `required: true / false` semantic applies.

## Typed parameters

A safeguard may declare a JSON Schema `config` object in its
`internal/safeguards/<name>/safeguard.json` manifest. The operator's selected
values live beside `opted_in` at
`~/.vrooli/state/vrooli/vrooli-onboarding/operator-state.json → host_safeguards.<name>.config`. Resolution
starts with manifest property defaults, overlays recorded values, validates the
merged object, and passes it to the handler as `ResolvedRequirement.Config`.
Invalid values are reported as an unsupported safeguard with an
`invalid_parameter` blocker; they never silently fall back to a default.

The current parameterized safeguards are:

- `netconsole.target`: kernel target grammar
  `src-port@src-ip/dev,tgt-port@tgt-ip/tgt-mac`; there is no safe default.
- `pstore_ramoops.mem_address` and `.mem_size`: operator-selected physical
  RAM region; `.ecc` defaults to `1`.
- `crashkernel_reserve.reservation`: kernel reservation value, defaulting to
  `512M-:256M`.
- `model_policy_drift.models`: optional per-runner arrays of allowed model
  names, keyed by runner.
- `remote_desktop_access.experience`: `observe-only` (the default),
  `login-screen`, or `direct-desktop`.
- `remote_desktop_access.provider`: `auto` (the default), `gnome-system`,
  `gnome-user-shared`, `gnome-headless`, `xrdp`, `windows-termservice`, or
  `macos-screen-sharing`.

`remote_desktop_access` is deliberately report-first. `observe-only` never
writes host state. Provider selection and liveness come from the typed
`internal/hostinventory` snapshot; safeguards and autoheal do not run their own
`grdctl`, `systemctl`, `sc`, or display-policy probes. The login-screen provider
can enable an existing system-mode GNOME Remote Desktop unit. The user-shared
provider delivers direct-desktop when a display is attached; the headless
provider additionally requires the shared host-inventory Wayland-attainability
fact. Native Windows TermService and macOS Screen Sharing providers deliver
direct-desktop on their own platforms. Only an explicit `WaylandEnable=false`
policy blocks Wayland. The NVIDIA GDM udev marker is diagnostic and never
overrides or disables that policy.

Mutating provider changes are separately opt-in: `allow_enable_system`,
`allow_switch_provider`, `allow_enable_user_unit`,
`allow_disable_system_unit`, `allow_provision_credentials`, and
`allow_enable_native_provider` all default to `false`. A missing permission
yields `manual_action_required` with the exact operator command; inspection
remains read-only. Credential provisioning is intentionally an operator-run
interactive command (`grdctl rdp set-credentials <username> <password>`): the
safeguard never accepts, stores, logs, or passes a password itself.

Use `vrooli-onboarding operator set-safeguard-config --name
remote_desktop_access --key experience --value-json '"direct-desktop"'` to
record one parameter without replacing unrelated operator-state fields. The
onboarding API validates the merged config against the safeguard manifest before
writing it. `vrooli setup status` reports platform-mismatched declarations as
`not_applicable` with the declared and current platforms rather than silently
dropping them.

The `login_keyring_unlock` safeguard is separate from remote-desktop
credential provisioning. It is Linux-only, requires an autologin user, is
optional and default-off, and must be explicitly opted into through operator
state. When enabled, it backs up the autologin user's login keyring before
opening the user's keyring prompt. The operator must choose and confirm a
blank new password; setup reports a manual action until a later status check
confirms the passwordless state. This is a real security reduction: any
process running as that user can read secrets stored in the keyring. The
safeguard never accepts or transmits the remote-desktop password. Without
autologin it reports `not_applicable` and changes nothing.

### Selected local services

`local_service_admission` is optional and Linux-only. Its `bindings` array
contains at most sixteen objects with `id`, `scenario`, `listener` (`ui`),
`source_ip`, `host_ip` and `interface`. Addresses must be canonical private IPv4
addresses on the selected home interface; ports come from the live runtime
registry. Each source is a single device address, not a CIDR or authenticated
identity. Keep machine values in operator state. A consuming scenario can
declare this safeguard in `hostSafeguards[]`; Portal does so for companion/TV
access. Empty configuration is not applicable and does not expose a listener.

After recording validated choices through the existing host/onboarding owner,
`vrooli setup broker --sudo-mode=ask` installs the narrow root-controlled policy.
The same installation occurs during full project setup. The focused command
avoids applying unrelated pending safeguards. Read-only inspection is
`vrooli host safeguard local_service_admission --dry-run --json`.
Runtime convergence, exact managed-rule cleanup and persistent revocation belong
to the [privilege broker](../../architecture/PRIVILEGE_BROKER.md#selected-local-service-admission).
The current adapter and physical reachability must be qualified separately;
an applied host policy is not a successful application session.

### Graphical-session automatic login

`graphical_session` reports the active desktop session, attached display, and
configured automatic-login user. It never changes login policy unless
`allow_enable_automatic_login` is explicitly opted in. The opt-in is a high-risk
physical-security decision: anyone who can restart the host can reach the
configured user's desktop.

On macOS, a username preference alone may not enable automatic login for a
passworded account. When the target user's password is available through an
authorized, ephemeral Bridge credential injection, the safeguard invokes
`/usr/sbin/sysadminctl` and feeds the password through stdin. It is never put in
argv, a plist, a repository file, or the evidence bundle. The typed dispatch
shape is:

```text
--credential-injection <logical-id>:<field>:VROOLI_GRAPHICAL_SESSION_PASSWORD
```

The injected credential is consumed for that bounded job and is not retained by
the safeguard. Do not reuse the Bridge node-store passphrase as the macOS login
password; they are different credentials. If no password injection is supplied,
the safeguard retains the passwordless/defaults path and reports that a
passworded account may still require the typed credential path after reboot.

## Autoheal boot protection

`autoheal_watchdog` is a required project safeguard. `vrooli setup` builds or
refreshes the autoheal loop, installs the invoking user's native scheduler
definition, enables only that scheduler entry, and verifies enablement plus
active state. It does not install a root-owned autoheal service or require a
separate `vrooli-autoheal install` command.

The default `boot_policy` is `dedicated`: Linux setup explicitly enables user
lingering so the user manager can start before login. A shared host can record
`host_safeguards.autoheal_watchdog.config.boot_policy=shared` through the
onboarding operator-state flow; that policy remains login-scoped and does not
enable lingering. Setup status reports service enablement separately from
verified boot protection. An unavailable scheduler bus is reported as
incomplete/degraded with the exact `vrooli setup` recovery command.

## Watchdog executable updates

The emergency-watchdog safeguard serializes executable updates with native
recovery. It probes the staged image for the exact source stamp before replacing
the installed image, streams a checksum-bound prior image into one retained
slot, and persists publication intent before the rename. The installed-image
probe must then match the admitted stamp and bytes. A lost write reply is
reconciled from that receipt. Failed qualification restores only the exact
retained image; an unexpected replacement or corrupt recovery copy remains
unresolved and is reported. The same failed image is suppressed until a changed
release is supplied.

`~/.vrooli/libexec/vrooli-watchdog.update.json` holds the bounded publication
receipt; `.previous` holds the retained executable and `.lock` serializes the
owner. Setup inspection reports pending or corrupt receipts. The independent
native runtime supervisor resumes already-admitted publication/restoration
without a compiler or source-root lookup, even after the publishing caller
dies. It cannot enroll or start another update. The owner removes only its
abandoned regular staging files under that lock.

The receipt's `verified` phase proves executable publication and its version
canary. It does not prove that a scheduled pressure check completed, that changed
scheduler definitions activated, or that root CLI self-update rollback works.
The current-host Linux fixture proves caller SIGKILL followed by restoration
through a fresh isolated native supervisor. Other native platforms remain
unqualified for this recovery path.

Completed live reports identify the producer's build, PID, process incarnation
and validated native invocation ID. Compare those fields with the publication
stamp and native unit's `InvocationID`, `ExecMainPID` and exit status to witness
a scheduled run of the admitted build. Fixture reports omit producer identity.

## Runtime supervisor unit

Root CLI installation stages a bounded executable snapshot before replacing the
canonical path. The existing installer checks the Go package role, native
OS/architecture and stamped `version --json` identity, then verifies the frozen
checksum during publication. The root version response identifies the inspecting
executable; supervisor status identifies the running owner. Basic version probes
skip managed host initialization. Root install locks and shared artifact locks
keep one admitted native file handle and reject observed leaf aliases.

The streamed `.previous` root snapshot is prior canonical byte retention.
Independent root CLI/supervisor activation and rollback still require the
[root control-plane transaction](../../resources/deployment-contract.md#root-control-plane-activation-target).

`runtime_supervisor` is a required project safeguard. Every `vrooli setup` run
renders the supervisor's unit from the shared `platformgo.ServiceDefinition`,
asks the native manager whether it would load it (`systemd-analyze --user
verify`, `plutil -lint`), installs it as the invoking user, enables it,
restarts it when the content changed or its running executable needs adoption,
and then re-inspects to prove it active.
`vrooli runtime supervisor install --user` calls the same `Converge` function,
so the CLI cannot render a unit setup would later call stale. Before 2026-09-02
the supervisor unit was rendered once by the install command and never looked
at again; a unit rendered on 2026-08-18 crash-looped 495 times after a reboot
because its argv no longer parsed.

Inspect reports the unit not-applied when its content differs from the render
(the note names both `ExecStart` lines), when it is not enabled and active, or
when the validator rejects the render. The status carries evidence a
readiness consumer reads back from `vrooli setup status --json --phase
readiness`: `validator_verdict` (accepted, unavailable, or rejected, with the
validator's output), `unit_state` (`ActiveState`, `UnitFileState`, `NRestarts`,
`Result`, `MainPID`), and the executable the unit names. On Linux, inspection
also compares the live process image's file identity through procfs with the
installed executable. Replacing the CLI file does not update a running process.
A mismatch, an unreadable image, or a missing active PID leaves the safeguard
pending. `vrooli host safeguard runtime_supervisor` restarts the unit for an
image mismatch even when its unit content is current, then verifies the adopted
image. Evidence includes `process_image_pid` and `process_image_current`.

The shared Linux convergence owner also performs this image check before
returning success to `vrooli runtime supervisor install --user`. An active unit
with a stale or unreadable executable, or without a valid `MainPID`, fails the
installation result. This check proves the observed image at activation; it
does not provide retained-artifact rollback after activation failure or caller
loss. That recovery path remains a separate update obligation.

## Agent session containment

`agent_session_containment` is a required, user-privilege project safeguard.
It renders `vrooli-agents.slice` from platform-go's slice definition, asks
`systemd-analyze --user verify` whether the user manager would load it,
installs it under `~/.config/systemd/user/` as the invoking user, reloads and
starts it, then reads the LIVE slice back (`ActiveState`, `ControlGroup`, and
the cgroup's `memory.max`, `pids.max`, `cpu.weight`) before it claims applied.
A slice that is written but not loaded, or loaded with other values, is
not-applied; a probe that cannot reach the user manager is `undetermined`
(evidence `probe: undetermined`), never ok. The coding-agent launcher and the
web console start every session in a scope under the slice through
platform-go's `ContainedCommand` / `ContainSelf`, so a build storm is the
slice's problem and not the host's.

| Config key | Default | Meaning |
| --- | --- | --- |
| `cpu_weight` | 50 | systemd `CPUWeight=` of the slice; 100 is neutral, the supervisors keep 400 |
| `memory_high_percent` | 50 | `MemoryHigh=` as a percentage of physical memory (throttling) |
| `memory_max_percent` | 60 | `MemoryMax=` as a percentage of physical memory (the kernel kills inside the slice before the host swaps); never below `memory_high_percent` |
| `tasks_max` | 4096 | `TasksMax=` of the slice; a fork storm stops here |

The operator changes them with `vrooli-onboarding host set-config`; a value
outside its bounds keeps the default so a typo never removes the ceiling.
The defaults are the 2026-09-02 plan's D3 decision and have not been
ratified by the operator.

Ownership boundary: `remote_session_protection` owns the SYSTEM manager's
units (the ancestor reservation in `user.slice.d`, the desktop reservation in
`user-<uid>.slice.d`, and `workload.slice`, Docker's parent) and needs `sudo vrooli setup`. This safeguard owns the USER
manager's `vrooli-agents.slice` and needs no privilege. Neither writes the
other's paths, and the agent slice sits under the desktop reservation.


`remote_session_protection` checks the actual cgroup v2 `memory.min` and
`memory.low` floors on both `user.slice` and the existing UID-1000 desktop
slice. A matching child drop-in alone does not establish effective kernel
protection. Missing, unreadable, invalid or insufficient live values remain
not applied. Its owner writes the ancestor definition under
`/etc/systemd/system/user.slice.d/90-vrooli-remote-session-protection.conf`.
When only reservation proof is missing, it changes only the owned reservation
files and the system manager's runtime `MemoryMin`/`MemoryLow` properties;
it does not reconfigure swap or restart Docker, graphical sessions or apps.
Stronger known allocations are preserved. Unrecognized owned-path settings
are refused before writes. It qualifies the system manager's unit search path,
main fragment and all generic, dashed-prefix and exact-name drop-in layers,
including files not loaded yet. Same-name specificity and search-path precedence
and different-name lexicographic order are checked; persistent startup intent
is kept separate from runtime properties. A later operator reset cannot be
hidden by current kernel values or a new runtime override. Known stronger
operator floors drive the runtime target. Aliases, masks, unsupported memory
syntax, includes and conflicting resets remain unqualified rather than being
silently rewritten. Non-memory generic defaults, such as `TasksMax`, remain
untouched. Owned destination ancestry is checked for symlink or non-directory
redirection before writes. Command replies and actual persistent, manager and
kernel postconditions are checked separately, including reconciliation of a
lost acknowledgement. These are bounded endpoint observations, not a
transaction against arbitrary concurrent privileged configuration writers.
These reservations protect an aggregate reclaim floor. They do not establish
a per-service memory/swap ceiling, guarantee that each descendant remains
available, or prevent every kernel OOM/systemd-oomd decision. File and live
qualification also do not prove physical reboot or sustained-pressure survival.

Evidence tier: Linux is host-verified (systemd 255, cgroup v2). macOS and
Windows have no slice; the launcher applies the same ceilings per session
(an rlimit shim on macOS, a Job Object with quotas on Windows) and those
tiers are fixture-verified only, which the manifest's `platform_status`
says.

## The readiness phase

A safeguard whose manifest entry declares `"when": ["setup", "readiness"]` is
re-inspected, never applied, by `vrooli setup status --json --phase readiness`.
The three boot-path safeguards (`autoheal_watchdog`, `runtime_supervisor`,
`emergency_watchdog`) declare it. Each one runs its rendered unit through the
native validator during Inspect and records the verdict in `evidence`, so a
render that systemd would refuse is visible while the host is healthy instead
of at the next boot. An unavailable validator is recorded as a note: unproven
is not accepted.

Record `host_safeguards.login_keyring_unlock.opted_in=true` through the
onboarding operator-state `apply` command, then run setup from a session that
survives any desktop restart:

```bash
vrooli-onboarding operator apply --body-file <operator-state-json>
sudo vrooli setup --maintenance-window
```

The safeguard remains `not_recorded` until that explicit choice exists. A
missing choice is not consent.

`vrooli setup explain <name>` and `vrooli host safeguard <name> --dry-run`
print the resolved parameter values.

## Credential keyring status and repair

The control plane owns the host credential-store diagnosis and keyring repair.
Use the keyring command group when a credential-backed safeguard reports a
store problem:

```bash
vrooli credentials keyring status
vrooli credentials keyring inspect --format json
vrooli credentials keyring repair --format json
printf '%s' "$PASSPHRASE" | vrooli credentials keyring unlock
```

`status` distinguishes a readable store (`ready`), an absent login collection
(`empty`), a passphrase-locked collection (`locked`), an unresponsive service,
and an unavailable service. A locked store is not the same as a missing
credential: unlock reads the passphrase from standard input only. It never
appears in an argument, environment variable, temporary file, log, or command
output. Inspection is read-only; repair rewrites only Vrooli-owned malformed
entries, makes a backup before writing, and declines foreign entries.

The keyring capability belongs in the control plane because setup and scenario
health checks must use one implementation. Scenarios may report the state and
offer an action, but they must not parse or rewrite the keyring themselves.

## Sysctl drop-in convention

Each sysctl-writing safeguard owns one drop-in under `/etc/sysctl.d/`, with a
distinct `99-vrooli-<safeguard>.conf` name declared in its manifest storage
surface. Applying a safeguard reloads the complete drop-in set with
`sysctl --system`; it does not use `sysctl -p <path>`, because boot ordering and
cross-safeguard visibility must match `systemd-sysctl`.

## Opt-in flow and operator-state states

`~/.vrooli/state/vrooli/vrooli-onboarding/operator-state.json` is the durable record of the operator's choice.
For each optional safeguard, `host_safeguards.<name>.opted_in` has three
observable states:

- `true` — `opted_in`: the operator consented, so setup may apply the safeguard.
- `false` — `declined`: the operator explicitly declined it; setup leaves it
  pending and reports that decision.
- missing — `not_recorded`: no operator choice exists yet; setup leaves the
  safeguard pending and reports that input is required.

Required safeguards bypass this choice because their consuming manifest marks
them as required. `vrooli setup status` and `vrooli setup explain <name>` show
the recorded state, so an absent entry is not silently confused with an
explicit decline. The resolver accepts a missing operator-state file as an
empty state, but rejects malformed state rather than silently falling back.
The onboarding wizard owns writes to this file; manifests remain declarative.

## Idempotency contract

All safeguards must be safe to apply multiple times. The Go handler implements an `Inspect` method that returns whether the safeguard is currently applied; `Apply` is only called when it isn't. This is enforced by the framework and is the reason `healthCheck` on the manifest exists (file-presence checks like `/etc/sysctl.d/99-vrooli.conf` exist).

A safeguard that can't be made idempotent doesn't belong in this system; it should be a one-shot setup script under the relevant scenario.

## Focused repair and reboot-aware safeguards

Use `vrooli host safeguard <name>` when one declared host capability needs a
focused repair. It uses the same handler, privilege policy, and typed status as
`vrooli setup`, without applying unrelated setup requirements. `--dry-run`
shows the exact managed transaction; `--sudo-mode ask` requests elevation from
an interactive terminal when needed.

Some changes cannot become effective in the current boot. Such handlers return
`reboot_required` with the blocking reason `needs_reboot`; this is intentionally
not reported as healthy. Reboot, then rerun the same safeguard to verify the
live state.

`nvidia_driver` is the reference kernel-driver safeguard. It detects NVIDIA
display hardware through PCI sysfs even when `nvidia-smi` is broken, validates
the live NVML/kernel-driver handshake, and on Ubuntu repairs the module package
for the running kernel plus the installed kernel-meta package that carries the
repair forward to future kernel upgrades. It is not a `hostTool`: a GPU
container cannot supply or repair the host kernel driver it depends on.
When a recognized remote-desktop server is active, it fails closed before the
package transaction because module installation can claim DRM/VGA devices
immediately. Schedule a maintenance window with console or SSH recovery and
pass `--maintenance-window` only when that interruption is acceptable. The
same acknowledgement is available on the ordinary `vrooli setup` command;
`sudo vrooli setup --maintenance-window` threads it to every safeguard that
may interrupt the active graphical or remote session.

## Adding a new safeguard

1. Create `internal/safeguards/<name>/safeguard.json` conforming to [`safeguard.schema.json`](../../../.vrooli/schemas/safeguard.schema.json).
2. **Set `risk` honestly.** Operators rely on this; understating the risk is worse than running without the safeguard.
3. Add a Go handler implementing `Inspect` and `Apply` under `path:internal/safeguards/<name>/`, register it in `internal/runtime/registry.go` `customSafeguardHandlers`.
4. Reference the safeguard from the consuming `service.json` or `resource.json` `hostSafeguards[]` array.
5. If the handler needs operator input, declare a typed `config` JSON Schema in
   the safeguard manifest and document each parameter's grammar and defaults.
6. Verify `go test ./internal/runtime/...` passes the manifest-vs-handler and
   strict-schema invariants.

The wizard surfaces the new safeguard automatically.

## Removal

Removing a safeguard from a manifest doesn't undo its host-state changes. If a safeguard has been applied and is later removed from the system, document the manual cleanup steps in the safeguard's `notes` field. A future feature could add an `Unapply` method, but that's out of scope today.

Removing a safeguard is different from disposing of a workload it once
installed. Removal changes the control-plane declaration and stops future
reconciliation; it does not authorize deleting a surviving service, container,
unit, or its data. Workload disposal requires an independent abandoned-workload
classification, path-level evidence, the owning cleanup provider, and the
approval required by that provider's safety tier. An emergency watchdog may
report or propose disposal, but it must not infer disposal from safeguard
absence.

## See also

- [`tools.md`](tools.md) — the binary-install counterpart
- [`../architecture.md#resolution-order`](../architecture.md#resolution-order) — opt-in resolution rules
- [`internal/runtime/registry.go`](../../../internal/runtime/registry.go) — the canonical handler map

## GPU container access diagnosis

For NVIDIA-backed resources, a successful host probe (`nvidia-smi`) and a
configured Docker NVIDIA runtime do not prove that an already-running
container can still open its device. The control plane performs a scoped
`/dev/nvidiactl` open inside each running GPU resource and records
`gpu_state=ok|revoked|unknown` in resource status. `unknown` is never treated
as healthy; `revoked` is reported as `gpu-degraded` (or an unhealthy resource).

Inspect the six GPU resources with:

```text
vrooli resource status ollama --json
vrooli resource status kokoro --json
vrooli resource status whisper --json
vrooli resource status reranker --json
vrooli resource status kyutai-stt --json
vrooli resource status speaker-verification --json
```

The `resource-gpu-access` autoheal check distinguishes critical revoked access
from warning-level probe uncertainty. Its recovery actions restart only the
named revoked resources, or re-verify before restarting all currently revoked
targets. A direct repair is `vrooli resource restart <name>`; do not claim a
GPU repair from host `nvidia-smi` alone.

The daemon-reload path is repair-aware: it snapshots running GPU containers,
reloads systemd, rechecks access, and reports any restart repair. On this host
the disposable-container experiment preserved `/dev/nvidiactl` access across
both the synchronous and non-blocking reload requests, although systemd
returned a D-Bus timeout for the reload request. That timeout is itself
reported as a reload failure and remains actionable rather than being treated
as proof of persistence.

## Recovery services under memory pressure

The minimum recovery plane declared by `api-core/coreset` receives an explicit
`ManagedOOMPreference=avoid` on each live scenario service scope. Shadow
instances, agents, resources and other scenarios retain their ordinary policy.
The native autoheal loop, runtime supervisor and emergency watchdog also render
this preference on their own services. This is a Linux systemd policy; other
platforms retain their existing containment behavior.

A slice with `ManagedOOMMemoryPressure=auto` can still lose descendant services
when an ancestor is monitored. The preference is not recursive, so setting it
only on `vrooli-services.slice` would not protect its leaf scopes. `avoid`
reduces selection priority but permits last-resort reclamation; it is not `omit`,
a memory reservation, or a substitute for workload budgets. User-owned preferences
are respected for the monitored user manager on this host (both cgroups have the
same owner); a root-owned ancestor can ignore them. See the
[systemd resource-control contract](https://www.freedesktop.org/software/systemd/man/latest/systemd.resource-control.html).

Scenario lifecycle placement applies the scope property at launch. Setup owns
rendering/updating the native supervisor units. A successful build does not
change an already-running scenario scope: inspect the live properties after its
normal lifecycle replacement. If the native manager cannot establish the
preference, the raw cgroup fallback refuses to claim that protection and lifecycle
logs a placement failure; service availability remains best-effort.
