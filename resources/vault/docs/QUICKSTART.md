# Vault Quick Start

## Start The Resource

```bash
vrooli resource install vault
resource-vault start
resource-vault status
```

The local runtime uses a signed Vault server with file storage in the resource data directory and a loopback-only listener. `resource-vault` requires an explicitly supplied scoped `VAULT_TOKEN` for secret operations; bootstrap and unseal are authorized provider operations and never print or discover a root token on the normal path.

## Store And Read A Secret

```bash
resource-vault content set \
  --path secret/capabilities/example/quickstart \
  --key value \
  --value "hello-vault"

resource-vault content get \
  --path secret/capabilities/example/quickstart \
  --key value \
  --format raw
```

Machine consumers should use `--format raw` when they need exactly the field value on stdout.

## List And Delete

```bash
resource-vault content list --path secret/capabilities/example/
resource-vault content delete --path secret/capabilities/example/quickstart
```

## Resource Declarations

```bash
vrooli credentials status --identity vrooli/openrouter --field api-key
```

`check` and `validate` report presence without values. `export` emits shell-safe `export KEY='value'` lines for present secrets declared with `default_env`.

## Ordinary Credentials

Ordinary resource, integration, and backup credentials do not use Vault. They
are provisioned through the control-plane credential authority:

```bash
vrooli credentials provision --identity vrooli/example --field api-key
vrooli credentials status --identity vrooli/example --field api-key
```

The authority uses the operating system's native key service or encrypted
portable storage. Use `resource-vault` only for a capability that explicitly
requires Vault KV or Transit semantics.

## Current Limits

This resource is a durable local Vault service for Vrooli resources. It is not a complete enterprise Vault deployment. HA, auto-unseal, production TLS, namespaces, dynamic database credentials, PKI, SSH CA, and audited multi-tenant policy workflows remain out of scope for this version.
