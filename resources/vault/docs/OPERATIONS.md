# Operations

`vault` is a `docker-service` resource managed through the shared Vrooli lifecycle.

## Boundaries

- `resource.json` owns Docker image, runtime command, volumes, ports, health checks, and lifecycle metadata.
- `cli/` owns the `resource-vault` binary and custom resource commands.
- `cli/internal/content` owns the canonical KV v2 content wrapper.
- `cli/internal/secrets` owns `config/secrets.yaml` inventory workflows.
- `lib/` is retained shell-era reference code and is not authoritative.

## Runtime Mode

The default local mode uses Vault file storage:

```json
{"storage":{"file":{"path":"/vault/file"}}}
```

Data, config, and logs are mounted from canonical resource storage directories. This mode is durable across container restarts and is the expected mode for backup passphrases.

The CLI initializes and unseals a fresh local instance on first secret operation. Bootstrap material is stored in the resource config mount and should be treated as sensitive local operator material. This keeps local development and internal automation usable, but it is not HA or auto-unseal.

## Operational Checks

```bash
resource-vault status
vrooli resource status vault
resource-vault content set --path secret/test/ops --key value --value ok
resource-vault content get --path secret/test/ops --key value --format raw
resource-vault content delete --path secret/test/ops
```

For persistence:

```bash
resource-vault content set --path secret/test/persistence --key value --value survives-restart
vrooli resource restart vault
resource-vault content get --path secret/test/persistence --key value --format raw
resource-vault content delete --path secret/test/persistence
```

## Secret Handling

Default status, check, validate, list, and lifecycle commands must not print secret values. Commands that reveal values are limited to direct `content get` and `secrets export` requests.

## Production Gaps

Before using Vault for external production tenants, implement and validate at least:

- TLS listener and certificate management.
- Externalized unseal strategy or a proper operator runbook.
- Policy and token model with least privilege per resource.
- Audit device enablement and log retention.
- Backup and restore of Vault storage and bootstrap material.
- HA storage and disaster recovery design.
