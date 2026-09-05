# Vault Resource

Managed HashiCorp Vault runtime for explicitly declared Vault-backed capabilities.

## Supported Surface

- `resource.json` is the lifecycle authority for the signed native Vault server, its loopback listener, data path, health check, and provider policy.
- `resource-vault` is the supported CLI for resources and scenarios.
- `resource-vault content ...` is the supported machine interface for reading and writing KV v2 secrets.
- `vrooli credentials ...` is the metadata-safe credential inventory and
  provisioning surface for manifest descriptors.

## Runtime Posture

The default local runtime is a Vrooli-signed Vault server installed under the per-user resource artifact store. It uses file storage under `${RESOURCE_DATA_DIR}`, writes a non-secret loopback-only configuration under `${RESOURCE_CONFIG_DIR}`, and never exposes a root token through the normal lifecycle or CLI path. Initialization, unseal, and scoped client credentials are provider-authorized operations; supply the resulting scoped `VAULT_TOKEN` explicitly when using `resource-vault content`.

Vault is an optional capability service, not the ordinary credential authority for Vrooli. New resource and integration credentials use the control-plane credential authority, backed by the operating system's native key service or an encrypted portable backend. Use this resource only when a capability explicitly requires Vault KV or Transit semantics. It is not an enterprise production Vault deployment: no HA, auto-unseal, TLS listener, namespaces, dynamic database credentials, PKI, or SSH CA are implemented here.

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
resource-vault content set --path secret/capabilities/example/api/key --key value --value "secret"
resource-vault content get --path secret/capabilities/example/api/key --key value --format raw
resource-vault content get --path secret/capabilities/example/api/key --format json
resource-vault content list --path secret/capabilities/example/
resource-vault content delete --path secret/capabilities/example/api/key
```

Contract:

- KV v2 paths are passed as Vault CLI paths owned by the explicit Vault-backed capability.
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

## Relationship To The Credential Authority

Ordinary credentials, including Kopia repository passphrases and backup backend credentials, are stored and retrieved through the control-plane credential authority. `resource-kopia` uses a stable authority identity per repository and does not mirror those values into Vault. The authority selects the operating system key service or encrypted portable storage and provides recovery bundles when requested.

Capabilities that explicitly need Vault KV or Transit continue to use `resource-vault`; that boundary is opt-in and does not make Vault a project-wide secret store.
## Maturity

M4 (2026-08-05): lifecycle, health, platform gates, and Go CLI test evidence are covered by the fleet contract.
