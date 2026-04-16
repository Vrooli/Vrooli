# Vault Resource

Managed HashiCorp Vault runtime for local secret storage and resource-secret workflows.

## Intent

- Resource ID: `vault`
- Category: `storage`
- Driver: `docker-service`
- Portability tier: `full`

## Use Cases

- Provide a central local secret store for resources and scenarios.
- Validate and export resource-specific secrets from a shared contract.
- Support development and internal automation that should not hardcode credentials.

## Architecture

This resource is being aligned to the updated `docker-service` structure.

- `resource.json` is the declarative authority for lifecycle, runtime, ports, exports, health, and freshness metadata.
- `cli/` is the thin binary entrypoint and delegated command wiring surface.
- `cli/internal/` is the default home for Vault-specific Go logic when the manifest and shared control plane are not enough.
- `lib/` still contains retained shell behavior during the migration. That behavior should move into `cli/internal/...` over time rather than back into `cli/main.go`.

The intended escalation path is:

1. express behavior in `resource.json`
2. rely on the shared `vrooli resource ...` control plane
3. add Vault-specific Go code under `cli/internal/...` only where specialization is real
4. add custom CLI commands only when the resource truly needs resource-local operator actions beyond the standard lifecycle surface

Current internal package boundaries:

- `cli/internal/install`: install/bootstrap helpers unique to Vault
- `cli/internal/runtime`: runtime and config materialization helpers
- `cli/internal/status`: richer Vault status interpretation
- `cli/internal/health`: Vault-specific probe helpers
- `cli/internal/env`: environment export and derived-config helpers

## Usage

```bash
# Install or validate the resource contract
vrooli resource install vault

# Check status through the shared control plane
resource-vault status
```

Connection defaults:

- URL: `http://localhost:8200`
- Dev token: `myroot`

## Notes

- Keep `cli/main.go` thin. Do not treat it as the implementation surface for secret, template, or validation workflows.
- Keep runtime storage rooted in `${RESOURCE_*_DIR}` paths rather than repo-local mutable directories.
- Existing shell-heavy workflows in `lib/` are transitional. New logic should land in Go under `cli/internal/...`.
- Use [docs/OPERATIONS.md](/home/matthalloran8/Vrooli/resources/vault/docs/OPERATIONS.md) as the architecture boundary for future migrations.
