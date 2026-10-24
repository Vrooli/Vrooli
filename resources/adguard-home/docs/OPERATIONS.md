# Operations

`adguard-home` is a native managed-service resource for managed local DNS
filtering. Its release archive is acquired and verified by the shared
fact-predicated acquisition contract; no container runtime is involved.

## Architecture Boundary

Keep responsibilities split cleanly:

- `resource.json` owns declarative lifecycle, runtime, port, health, and export metadata.
- `cli/` owns the binary entrypoint, wiring, and delegated command registration.
- `cli/internal/adguard` owns the narrow AdGuard Home control API client.
- `cli/internal/health` owns AdGuard-specific API diagnostics that cannot be expressed through manifest health checks.

Do not turn `cli/main.go` into the primary implementation surface. If the
resource needs specialized bootstrap, rotation, rollback, or diagnostics, grow
the matching package under `cli/internal/` first and keep commands thin.

## Operator Checklist

- Confirm the pinned release and every target checksum before upgrades. The current manifest uses AdGuard Home `v0.107.73`.
- Keep runtime storage rooted in `${RESOURCE_CONFIG_DIR}`, `${RESOURCE_DATA_DIR}`, `${RESOURCE_CACHE_DIR}`, `${RESOURCE_LOGS_DIR}`, and `${RESOURCE_STATE_DIR}` rather than repo-local `data/`.
- Back up `${RESOURCE_CONFIG_DIR}` and `${RESOURCE_DATA_DIR}` together. The managed process reads configuration from the former and keeps runtime work data in the latter.
- Treat the manifest HTTP check as liveness only. Use `resource-adguard-home api-health --json` for authenticated control-plane readiness.
- Use `resource-adguard-home bootstrap --base-url http://localhost:3000 --json` for first-run setup. It stores credentials through the credential authority and does not print the generated password.
- Use `resource-adguard-home config preview --json` to inspect upstream drift before Network Manager policy code applies any persistent resolver change.
- Use `resource-adguard-home clients list --json` for client/device evidence without reading query-level DNS history.
- Use `resource-adguard-home querylog privacy --json` to confirm query-log posture before reporting privacy compliance.
- Prefer shared control-plane behavior first; use `cli/internal/install`, `cli/internal/runtime`, `cli/internal/status`, `cli/internal/health`, and `cli/internal/env` only for real specialization.
- Keep `environment_exports` as the canonical scenario-facing contract; do not reintroduce shell-based export wrappers.
- Treat custom CLI commands as an exception path, not the default architecture.
- Validate both the resource manifest and at least one consuming scenario before treating the scaffold as complete.

## DNS Listener Binding

The manifest binds DNS on `192.168.1.173:53/tcp` and `192.168.1.173:53/udp`.
That is the honest production shape for network-wide DNS filtering on this
host, while preserving `systemd-resolved` on loopback addresses such as
`127.0.0.53`.

Before router-wide rollout:

1. Reserve `192.168.1.173` for this server in router DHCP, or update
   `resource.json` to the server's reserved/static LAN address.
2. Start AdGuard through `vrooli resource start adguard-home`.
3. If `api-health` reports `setup_required`, run
   `resource-adguard-home bootstrap --base-url http://localhost:3000 --json`.
4. Verify authenticated health using the stored secret:
   `resource-adguard-home api-health --base-url http://localhost:3000 --username admin --password "$PASSWORD" --json`.
5. Configure Network Manager with
   `network-manager resolver configure-adguard --base-url http://localhost:3000 --username admin --credential-ref vrooli/adguard-home --json`.
6. Point router DHCP DNS and IPv6 RDNSS/DNS guidance to the same LAN address
   only after AdGuard is healthy.

If `vrooli resource start adguard-home` fails because the configured LAN
listener is occupied, identify the owner with `vrooli diagnose-port 53` and move
or disable the conflicting resolver only through the owning service's supported
configuration.

Do not silently remap DNS to a high port and then claim network-wide ad
blocking; clients will not use that high port unless explicitly configured.

## Credentials

The manifest declares `vrooli/adguard-home` as the canonical
credential reference. The bootstrap command stores username/password there
through the Vrooli secret flow. Resource diagnostics may also read
`ADGUARD_HOME_USERNAME` and `ADGUARD_HOME_PASSWORD` for local smoke tests, but
prefer one-shot shell variables injected by the credential authority when
performing live checks.

Never pass the plain password into Network Manager. Network Manager should store
only the credential reference and fail closed on missing or invalid credentials.

## Health States

`resource-adguard-home api-health --json` reports:

| State | Meaning |
|---|---|
| `healthy` | Control API reachable and protection is enabled with minimal query-log posture. |
| `degraded` | Control API reachable but protection is disabled, filtering state is unknown, query logs are enabled, or a secondary endpoint is unavailable. |
| `setup_required` | AdGuard setup/control API is not ready yet. |
| `auth_failed` | Credentials were rejected. |
| `unreachable` | Admin/control endpoint could not be reached. |

Only `healthy` should allow Network Manager to describe AdGuard protection as
active. Network-wide enforcement still requires client/router DNS evidence.

## Upgrade / Rollback

1. Capture Network Manager resolver health and a home snapshot before changing
   the image tag.
2. Stop the resource through `vrooli resource stop adguard-home`.
3. Back up the resource config/data directories.
4. Update the image tag in `resource.json`.
5. Start through `vrooli resource start adguard-home`.
6. Run `resource-adguard-home api-health --json`.
7. Run Network Manager resolver health and a post-upgrade snapshot.

If the API health regresses, restore the prior tag and config/data backup before
attempting policy changes.
