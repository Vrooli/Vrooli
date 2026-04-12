# HashiCorp Vault Resource

Vault remains the central secret store for Vrooli, but its supported CLI surface is now intentionally narrow. The retained commands are the ones active code paths use today:

- `status`
- `content add`
- `content get`
- `content remove`
- `secrets check`
- `secrets init`
- `secrets validate`
- `secrets export`
- `secrets create-template`

Unused admin, audit, monitoring, migration, and recovery subcommands were removed instead of being carried into migration work.

## Quick Start

### Check Vault health
```bash
vrooli resource vault status
```

### Manage resource secrets
```bash
# Check whether a resource has the required secrets
vrooli resource vault secrets check openrouter

# Initialize a resource's secrets interactively
vrooli resource vault secrets init openrouter

# Validate all configured resource secrets
vrooli resource vault secrets validate

# Export a resource's secrets as environment variables
vrooli resource vault secrets export openrouter > openrouter-env.sh
source openrouter-env.sh

# Create a new secrets.yaml template for a resource
vrooli resource vault secrets create-template my-resource
```

### Read and write Vault content
```bash
# Store a plain value
vrooli resource vault content add --path "resources/openrouter/api_key" --value "sk-..."

# Store a keyed value inside a path
vrooli resource vault content add --path "resources/postgres" --key "db_password" --value "secret"

# Read a raw value
vrooli resource vault content get --path "resources/openrouter/api_key" --format raw

# Read a keyed value
vrooli resource vault content get --path "resources/postgres" --key "db_password" --format raw

# Remove a path
vrooli resource vault content remove --path "resources/openrouter/api_key"
```

## Resource Secrets Workflow

Vault integrates with resource `config/secrets.yaml` files.

1. A resource defines required and optional secrets in `config/secrets.yaml`.
2. `vrooli resource vault secrets check <resource>` reports whether those secrets are present.
3. `vrooli resource vault secrets init <resource>` helps populate missing secrets.
4. `vrooli resource vault secrets export <resource>` emits environment variables for consumers that still expect env-based configuration.

See [SECRETS-STANDARD.md](docs/SECRETS-STANDARD.md) for the shared contract.

## Supported Commands

### Standard resource lifecycle
- `install`
- `uninstall`
- `start`
- `stop`
- `restart`
- `status`
- `logs`

### Supported Vault-specific commands
- `content add --path <path> --value <value> [--key <key>]`
- `content get --path <path> [--key <key>] [--format raw]`
- `content remove --path <path>`
- `secrets check <resource>`
- `secrets init <resource>`
- `secrets validate`
- `secrets export <resource>`
- `secrets create-template <resource>`

## Notes

- The Vault resource is still active, but most of the historical shell surface was not used by non-Vault code.
- The reduced interface is intentional and is meant to keep the upcoming native migration focused on the code paths the repo actually depends on.
