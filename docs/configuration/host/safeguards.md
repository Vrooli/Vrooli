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

## The `risk` field

New as of the configuration substrate work. Operator-facing risk indicator that informs the wizard's display.

| Value | Meaning | Examples |
|---|---|---|
| `low` | No system state changes outside Vrooli's tree. Probes, reads, soft validations | `clock` (clock probe and sync) |
| `medium` | Writes config files outside Vrooli's tree, or modifies networking rules | `dns_resolution` (writes `/etc/systemd/resolved.conf.d/`), `docker_host_firewall` (iptables rules), `nat_protection` (iptables) |
| `high` | Modifies kernel parameters or requires root in ways that broadly affect host behavior | `kernel_config` (writes `/etc/sysctl.d/99-vrooli.conf`) |

The wizard's host step renders this as a column next to each safeguard so operators can decide informed. Required safeguards (per the consuming manifest's `required: true`) bypass the opt-in but still display risk.

## hostRequirement shape

Identical to tools — a safeguard reference is a `hostRequirement` entry in a `hostSafeguards[]` array. See [`tools.md`](tools.md#hostrequirement-shape) for fields. The same `required: true / false` semantic applies.

## Opt-in flow

```
1. .vrooli/operator-state.json → host_safeguards.<name>.opted_in   (if present, use it)
2. fall back to the consuming manifest's hostRequirement `required` field
3. default (not applied — safeguards are opt-in by nature)
```

The wizard writes only to layer 1. Manifests own layer 2.

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
pass `--maintenance-window` only when that interruption is acceptable.

## Adding a new safeguard

1. Create `internal/safeguards/<name>/safeguard.json` conforming to [`safeguard.schema.json`](../../../.vrooli/schemas/safeguard.schema.json).
2. **Set `risk` honestly.** Operators rely on this; understating the risk is worse than running without the safeguard.
3. Add a Go handler implementing `Inspect` and `Apply` under `path:internal/safeguards/<name>/`, register it in `internal/runtime/registry.go` `customSafeguardHandlers`.
4. Reference the safeguard from the consuming `service.json` or `resource.json` `hostSafeguards[]` array.
5. Verify `go test ./internal/runtime/...` passes the manifest-vs-handler invariant.

The wizard surfaces the new safeguard automatically.

## Removal

Removing a safeguard from a manifest doesn't undo its host-state changes. If a safeguard has been applied and is later removed from the system, document the manual cleanup steps in the safeguard's `notes` field. A future feature could add an `Unapply` method, but that's out of scope today.

## See also

- [`tools.md`](tools.md) — the binary-install counterpart
- [`../architecture.md#resolution-order`](../architecture.md#resolution-order) — opt-in resolution rules
- [`internal/runtime/registry.go`](../../../internal/runtime/registry.go) — the canonical handler map
