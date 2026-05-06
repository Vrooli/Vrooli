# Port Allocation

This page is the canonical reference for scenario listener ports in Vrooli.
Templates, scaffolding, the `vrooli scenario audit` rule, and the manifest
validator all point here rather than duplicating policy.

## TL;DR

Declare scenario ports in `.vrooli/service.json` using these canonical bands.
All bands sit **below 32768** so no listener overlaps the Linux ephemeral
range (the OS can otherwise steal the number as an outbound source port while
the scenario is down, which surfaces as intermittent "port already in use"
restart failures).

| Role      | Env var     | Canonical band   |
|-----------|-------------|------------------|
| API       | `API_PORT`  | `15000-19999`    |
| UI        | `UI_PORT`   | `20000-24999`    |
| WebSocket | `WS_PORT`   | `25000-29999`    |
| Reserved headroom (future roles) | —           | `30000-32767`    |

Fixed ports (e.g. Cloudflare-tunneled UIs) pin a single value inside the band
for that role.

## Why 32768 matters

Every operating system maintains an "ephemeral" port pool that the kernel
allocates as **source** ports for outbound TCP/UDP connections. Defaults vary:

| OS      | Default ephemeral range | How to verify                                                                |
|---------|-------------------------|------------------------------------------------------------------------------|
| Linux   | `32768-60999`           | `cat /proc/sys/net/ipv4/ip_local_port_range`                                 |
| macOS   | `49152-65535`           | `sysctl net.inet.ip.portrange.first net.inet.ip.portrange.last`              |
| Windows | `49152-65535`           | `netsh int ipv4 show dynamicport tcp`                                        |
| IANA (RFC 6335) | `49152-65535`  | — (specification, not an enforced value)                                     |

Linux's default starts **much lower** than the IANA recommendation and than
any other supported OS. A listener port in `32768-60999` on Linux is a
roulette bet: whenever the scenario is down, another process's outbound
socket can consume that port as its source, leaving `bind()` to fail on the
next start with no visible permanent listener to blame.

This class of failure is the reason the canonical bands stop at `32767`.

## How Vrooli enforces the policy

Three layers work together so a regression is caught as early as possible:

1. **Manifest validator** — `internal/scenario/validate_ports.go` runs at
   `ReadService` time. It rejects any fixed port or range that overlaps the
   **live** OS ephemeral window (detected once per process via
   `internal/portspec.OSEphemeralRange`). Broken manifests fail fast with a
   link back to this document.
2. **Auditor rule** — `scenarios/scenario-auditor/api/rules/config/service_ports.go`
   runs during `vrooli scenario audit`. It applies the same canonical-band
   rules against a **static reference** ephemeral range (Linux's default
   `32768-60999`) so CI output is reproducible regardless of the machine
   running the audit.
3. **Template defaults** — `templates/scenarios/react-vite/.vrooli/service.json`
   and `templates/scenarios/landing-page-react-vite/.vrooli/service.json`
   declare the canonical bands. New scenarios inherit them automatically.

### Escape hatch

If a manifest legitimately must live outside the policy (e.g., a legacy
integration you are in the middle of migrating), set
`VROOLI_PORT_VALIDATION=off` when loading. This is intentionally limited to
the manifest validator — the auditor still flags the finding. Do not leave it
on permanently.

## Failure modes this prevents

The following three symptoms all collapse onto the same "port already in use"
error message in the current CLI; the port-allocation policy addresses each.

1. **Ephemeral-range steal (Linux only).** Kernel assigns your listener's port
   as an outbound source while the scenario is stopped, so `bind()` fails on
   restart. Fix: keep every listener below 32768 (this policy).
2. **Orphan listeners from unclean termination.** A leftover child process
   continues listening on the fixed port after the CLI terminated
   abnormally. Fix (unrelated to this page): the start-time cleanup path in
   `internal/lifecycle/lifecycle.go` now kills env-less orphans on canonical
   ports; see that code's tests for the contract.
3. **Lock-before-bind race.** Port lock file was written before the child
   actually bound; a crash between the two steps strands the lock. Fix
   (also unrelated to this page): `ConfirmLock` / `AbandonLock` in
   `internal/ports/ports.go` split the allocation and the confirmation
   steps.

## Migration

The `path:cmd/vrooli-ports-migrate` tool computes `newPort = oldPort - 15000` for
any scenario UI port currently in `35xxx-36xxx` and rewrites
`.vrooli/service.json` accordingly. Dry-run by default:

```bash
go run ./cmd/vrooli-ports-migrate         # print before→after table
go run ./cmd/vrooli-ports-migrate --apply # write changes in-place
go run ./cmd/vrooli-ports-migrate --json  # machine-readable output
```

Scenarios with public Cloudflare tunnels or other external consumers are
flagged in the tool's output so the tunnel configuration can be updated in
the same pass. The migration is idempotent — running it twice after a
successful apply is a no-op.

## See also

- [`docs/operations/troubleshooting.md`](../operations/troubleshooting.md) —
  `vrooli diagnose-port` for live conflict triage.
- [`docs/reference/cli-commands.md`](cli-commands.md) — lifecycle commands that
  surface port allocation.
- `path:internal/portspec/` — constants and OS probe.
- `internal/scenario/validate_ports.go` — validator.
- `scenarios/scenario-auditor/api/rules/config/service_ports.go` — audit
  rule.
