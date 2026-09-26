# Desktop telemetry

Generated desktop applications record local lifecycle and deployment events in
`deployment-telemetry.jsonl` under the application’s user-data directory.

## Event purpose

Telemetry may describe:

- application start and readiness;
- dependency or server unavailability;
- runtime/service exit;
- migration and secret-status outcomes;
- update checks, downloads, replacement, and errors;
- GPU or host-capability decisions.

Telemetry must not contain credential values, bearer tokens, private endpoints,
generated operator configuration, or captured video bytes.

## Local paths

| Platform | Default location |
| --- | --- |
| Windows | `%APPDATA%/<App Name>/deployment-telemetry.jsonl` |
| macOS | `~/Library/Application Support/<App Name>/deployment-telemetry.jsonl` |
| Linux | `~/.config/<App Name>/deployment-telemetry.jsonl` |

The bundled runtime may keep service logs and runtime telemetry below its
application-data root. The install directory is not a mutable state directory.

## Collection

When an operator has consented to share a diagnostic file, ingest it with:

```bash
scenario-to-desktop telemetry ingest "my-scenario" --file "telemetry.jsonl"
```

Collection is separate from release evidence. Governance receives metadata and
references, not secret values or video bytes.

## Tier 1 management

Auto-starting or stopping a local Tier 1 scenario is an explicit thin-client
configuration. It is not part of bundled-runtime ownership and must never be
enabled for a thin client that points at a separately managed server.

See [the runtime reference](../../runtime/README.md), [logging guidance](../guides/logging-bundled-desktop.md),
and the [desktop evidence contract](../../../../docs/reference/scenario-to-desktop-evidence-and-tier-contract.md).
