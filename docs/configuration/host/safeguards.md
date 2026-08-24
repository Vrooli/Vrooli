# Host Safeguards

Safeguards are idempotent host-state modifications Vrooli applies to make the host suitable for running its workload — kernel parameters, DNS configuration, firewall rules, NAT protections, TCP tuning, clock synchronization. They differ from host tools in that they *change the host's state* rather than just installing a binary, so they are explicitly opt-in with a risk indicator.

## What lives where

| Concern | File | Field |
|---|---|---|
| What the safeguard does, what it modifies, how to verify | `internal/safeguards/<name>/safeguard.json` | top-level manifest |
| Risk indicator | `internal/safeguards/<name>/safeguard.json` | `risk` (`low` / `medium` / `high`) |
| Go handler that implements `Inspect` and `Apply` | `internal/safeguards/<name>/*.go` (registered in `internal/runtime/registry.go`) | `customSafeguardHandlers` map |
| Top-level project requirements | `.vrooli/service.json` | `hostSafeguards[]` (each entry: `hostRequirement`) |
| Per-scenario requirements | `scenarios/<name>/.vrooli/service.json` | `hostSafeguards[]` |
| Per-resource requirements | `resources/<name>/resource.json` | `hostSafeguards[]` |
| Operator opt-in | `.vrooli/operator-state.json` | `host_safeguards.<name>.opted_in` |

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

This is intentionally separate from `risk`. `privilege` is the machine gate
that tells Vrooli whether elevated setup is needed. `risk` is the human-facing
impact label that helps an operator decide whether to opt into a host-state
change. Neither field substitutes for the other.

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
`.vrooli/operator-state.json → host_safeguards.<name>.config`. Resolution
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

`.vrooli/operator-state.json` is the durable record of the operator's choice.
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
vrooli resource status ollama --json --no-stale-check
vrooli resource status kokoro --json --no-stale-check
vrooli resource status whisper --json --no-stale-check
vrooli resource status reranker --json --no-stale-check
vrooli resource status kyutai-stt --json --no-stale-check
vrooli resource status speaker-verification --json --no-stale-check
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
