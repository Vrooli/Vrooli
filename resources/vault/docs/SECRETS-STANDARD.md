# Vrooli Resource Secrets Standard

Resources can declare required and optional secrets in `resources/<resource>/config/secrets.yaml`. Vault uses those declarations for inventory, validation, and environment export.

## Minimal Shape

```yaml
version: "1.0"
resource: "example"
description: "Credentials for the example resource"

secrets:
  api_keys:
    - name: "primary_api_key"
      path: "secret/resources/{resource}/api/primary"
      description: "Main API key"
      required: true
      format: "string"
      default_env: "EXAMPLE_API_KEY"
      validation:
        pattern: "^[A-Za-z0-9_-]{20,}$"

initialization:
  auto_generate:
    - name: "internal_token"
      type: "random-32"
      path: "secret/resources/{resource}/internal/token"
```

Supported fields for inventory are `name`, `path`, `description`, `required`, `format`, `default_env`, `fallback`, `fields`, and `validation.pattern`.

`{resource}` and `{resource-name}` are expanded to the declaring resource. Other templates such as `{repo-name}` are treated as dynamic paths; they appear in inventory but are not considered globally missing because they require runtime context.

## Commands

```bash
resource-vault secrets scan
resource-vault secrets check <resource>
resource-vault secrets validate [resource]
resource-vault secrets export <resource>
resource-vault secrets provision <resource>
```

`resource-vault secrets init <resource>` is an alias for `provision` for resources that already call it. It is non-interactive.

## Semantics

- `scan` lists resources with declarations and secret counts.
- `check` verifies present/missing status for one resource without printing values.
- `validate` checks one resource or all resources and fails if required static secrets are missing.
- `export` prints shell-safe `export KEY='value'` lines only for present secrets with `default_env`.
- `provision` creates supported `initialization.auto_generate` secrets when they do not already exist.

Supported non-interactive auto-generation types are `uuid`, `token`, `password`, `random-32`, and `random-N`.

## Content Contract

The declaration path maps to a Vault KV v2 path. Single-value secrets use field `value` unless a resource-specific command explicitly writes another field.

```bash
resource-vault content set --path secret/resources/example/api/primary --key value --value "$EXAMPLE_API_KEY"
resource-vault content get --path secret/resources/example/api/primary --key value --format raw
```

Resources should use `resource-vault` only. They must not call Vault HTTP APIs or the `vault` binary directly.

## Limitations

The standard is currently an inventory and local secret workflow. It does not yet implement interactive prompting, provider-specific credential tests, per-resource Vault policies, TTL renewal, or secret rotation automation.
