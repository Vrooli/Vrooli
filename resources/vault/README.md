# Vault Resource

Managed HashiCorp Vault runtime for local Vrooli secret storage and resource-secret workflows.

## Supported Surface

- `resource.json` is the lifecycle authority for the signed native Vault server, its loopback listener, data path, health check, and provider policy.
- `resource-vault` is the supported CLI for resources and scenarios.
- `resource-vault content ...` is the supported machine interface for reading and writing KV v2 secrets.
- `vrooli credentials ...` is the metadata-safe credential inventory and
  provisioning surface for manifest descriptors.

## Runtime Posture

The default local runtime is a Vrooli-signed Vault server installed under the per-user resource artifact store. It uses file storage under `${RESOURCE_DATA_DIR}`, writes a non-secret loopback-only configuration under `${RESOURCE_CONFIG_DIR}`, and never exposes a root token through the normal lifecycle or CLI path. Initialization, unseal, and scoped client credentials are provider-authorized operations; supply the resulting scoped `VAULT_TOKEN` explicitly when using `resource-vault content`.

This is suitable for local durable resource secrets such as Kopia repository passphrases. It is not an enterprise production Vault deployment: no HA, auto-unseal, TLS listener, namespaces, dynamic database credentials, PKI, or SSH CA are implemented here.

## Lifecycle

```bash
vrooli resource install vault
resource-vault start
resource-vault status
resource-vault logs
resource-vault stop
```

The lifecycle commands delegate to the shared Vrooli resource control plane. `vrooli resource status vault` is equivalent for status.

## Secret Content

Canonical commands:

```bash
resource-vault content set --path secret/resources/example/api/key --key value --value "secret"
resource-vault content get --path secret/resources/example/api/key --key value --format raw
resource-vault content get --path secret/resources/example/api/key --format json
resource-vault content list --path secret/resources/example/
resource-vault content delete --path secret/resources/example/api/key
```

Contract:

- KV v2 paths are passed as Vault CLI paths, for example `secret/resources/kopia/repo/<repo>/passphrase`.
- `--key` defaults to `value`.
- `--format raw` prints only the secret value and exits non-zero when the path or field is absent.
- `content set` uses `kv patch` and falls back to `kv put`, preserving sibling fields when the path already exists.
- Values beginning with `@` are rejected so Vault cannot interpret them as host file references.

## Resource Secret Inventory

Resources declare expected credentials in `resource.json` under
`credentials.descriptors`.

```bash
vrooli credentials status --identity vrooli/openrouter --field api-key
```

`check` and `validate` never print secret values. `export` prints shell-safe `export KEY='value'` lines only for present secrets with `default_env` declarations. Dynamic paths such as `{repo-name}` are reported as dynamic inventory entries and are not treated as globally missing.

## Kopia

`resource-kopia` stores repository passphrases through:

```text
secret/resources/kopia/repo/<repo>/passphrase
```

with field:

```text
passphrase
```

Kopia treats a missing passphrase for an existing repository as a hard error and distinguishes missing secrets from Vault service outages.
