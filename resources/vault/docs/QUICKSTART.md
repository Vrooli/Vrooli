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
  --path secret/test/quickstart \
  --key value \
  --value "hello-vault"

resource-vault content get \
  --path secret/test/quickstart \
  --key value \
  --format raw
```

Machine consumers should use `--format raw` when they need exactly the field value on stdout.

## List And Delete

```bash
resource-vault content list --path secret/test/
resource-vault content delete --path secret/test/quickstart
```

## Resource Declarations

```bash
resource-vault secrets scan
resource-vault secrets check kopia
resource-vault secrets validate
resource-vault secrets export opencode
```

`check` and `validate` report presence without values. `export` emits shell-safe `export KEY='value'` lines for present secrets declared with `default_env`.

## Kopia Passphrase Path

Kopia stores one generated passphrase per repository:

```bash
resource-vault content get \
  --path secret/resources/kopia/repo/<repo>/passphrase \
  --key passphrase \
  --format raw
```

Do not store Kopia passphrases in config files, logs, command history, or scenario databases.

## Current Limits

This resource is a durable local Vault service for Vrooli resources. It is not a complete enterprise Vault deployment. HA, auto-unseal, production TLS, namespaces, dynamic database credentials, PKI, SSH CA, and audited multi-tenant policy workflows remain out of scope for this version.
