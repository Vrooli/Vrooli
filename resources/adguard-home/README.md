# AdGuard Home

Managed AdGuard Home DNS filtering and resolver service

This resource uses the `managed-service` template. The control plane acquires
and verifies the official AdGuard Home release for the selected
OS/architecture, then supervises the native process directly.

## Intent

- Resource ID: `adguard-home`
- Category: `dns`
- Driver: `managed-service`
- Portability tier: `cross-platform native`
- Pinned release: `v0.107.73`

## Use Cases

- Provide the governed local DNS filtering backend for Network Manager.
- Export stable AdGuard connection details to scenarios through Vrooli resource environment exports.
- Keep AdGuard configuration and work data in resource-managed durable directories.
- Let resource-local diagnostics prove AdGuard control-plane status before any scenario claims ad blocking is active.

## Architecture

The CLI stays thin on purpose.

- `resource.json` is the declarative authority for lifecycle, install, invoke, freshness, exports, and runtime metadata.
- `cli/` is the single binary entrypoint and command wiring surface.
- `cli/internal/adguard` is the narrow AdGuard Home control API client used by diagnostics.
- `cli/internal/health` classifies setup, auth, protection, upstream, and query-log posture from that client.

## Ports

The manifest declares:

| Name | Bind | Process | Purpose |
|---|---:|---:|---|
| `admin` | `3000/tcp` | `3000/tcp` | AdGuard setup/admin/control API |
| `dns-tcp` | `192.168.1.173:53/tcp` | `53/tcp` | DNS service |
| `dns-udp` | `192.168.1.173:53/udp` | `53/udp` | DNS service |

The default is loopback-only so a clean install cannot accidentally become a
network-wide resolver. Change the declarative bind address only with an
explicit host firewall and LAN exposure policy.

## Environment Exports

Scenarios should consume these exports rather than hard-coded values:

```text
ADGUARD_HOME_HOST
ADGUARD_HOME_PORT
ADGUARD_HOME_DNS_TCP_PORT
ADGUARD_HOME_DNS_UDP_PORT
ADGUARD_HOME_DNS_BIND_IP
ADGUARD_HOME_URL
ADGUARD_HOME_BASE_URL
ADGUARD_HOME_CREDENTIAL_REF
```

`ADGUARD_HOME_CREDENTIAL_REF` points at the credential-authority identity
`vrooli/adguard-home`.
Network Manager should receive that reference only; it must not store or log the
plain admin password.

## Diagnostics

```bash
resource-adguard-home api-health --json
resource-adguard-home api-health --base-url http://localhost:3000 --json
resource-adguard-home bootstrap --base-url http://localhost:3000 --json
resource-adguard-home config preview --upstream https://dns.quad9.net/dns-query --json
resource-adguard-home clients list --json
resource-adguard-home querylog privacy --json
```

Credentials default from `ADGUARD_HOME_USERNAME` and `ADGUARD_HOME_PASSWORD`.
When `ADGUARD_HOME_PASSWORD` is omitted, diagnostics resolve
`--credential-ref`, `ADGUARD_HOME_CREDENTIAL_REF`, or
`vrooli/adguard-home` through the credential authority and read only
the `username` and `password` fields.
The `api-health` command reports:

- `setup_required` when the control API is not mounted yet.
- `auth_failed` when credentials are rejected.
- `unreachable` when the admin endpoint cannot be reached.
- `degraded` when protection is disabled, filtering state is unknown, or query logs are enabled.
- `healthy` only when protection is enabled and query logs are disabled or minimal.

Additional diagnostics are intentionally narrow:

- `config preview` reads current upstream DNS config and previews a proposed upstream set. It can optionally call AdGuard's upstream test endpoint with `--test-upstreams`, but it does not mutate configuration.
- `clients list` reads configured and automatically discovered AdGuard clients without reading query-level DNS log entries.
- `querylog privacy` reports query-log configuration through `/control/querylog/config`, falling back to the legacy `/control/querylog_info` endpoint when necessary.

Persistent DNS filtering changes should go through Network Manager policy and rollback ledgers, not resource-local ad hoc commands.

## First-Run Bootstrap

Use the resource-local bootstrap command after `vrooli resource start
adguard-home` reports the managed service is running:

```bash
resource-adguard-home bootstrap --base-url http://localhost:3000 --json
```

The command calls AdGuard Home's first-install control API, generates a
high-entropy admin password when none is supplied, stores `username` and
`password` under `vrooli/adguard-home` through the credential authority, and
prints only the credential reference. It also
hardens privacy by disabling query logging after setup when the AdGuard API
accepts the current config shape.

After bootstrap, configure Network Manager with the secret reference only:

```bash
network-manager resolver configure-adguard \
  --base-url http://localhost:3000 \
  --username admin \
  --credential-ref vrooli/adguard-home \
  --json
```

Network Manager must continue to report `not_configured`, `setup_required`,
`auth_failed`, or `unreachable` until the resource is reachable,
authenticated, and protection/filtering is verified.

## Current Host DNS Design

The current host runs `systemd-resolved` on loopback port `53`, which is fine.
The AdGuard resource publishes DNS on `192.168.1.173:53` instead. If that LAN
address or port is occupied, `vrooli resource start adguard-home` should fail
clearly. Do not claim "ad blocking active" until AdGuard is running,
authenticated, and serving DNS on the intended LAN listener.
## Maturity

M4 (2026-08-05): lifecycle, readiness, pinned runtime, and Go CLI tests are covered by the fleet contract.
