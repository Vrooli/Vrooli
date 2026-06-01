# Product Requirements Document - Vault Resource

## Purpose

Vault provides local durable secret storage for Vrooli resources and scenarios. The immediate production-readiness target is reliable storage and retrieval of resource secrets such as Kopia repository passphrases through a single supported CLI surface.

## Implemented

- [x] Docker-service resource manifest with managed lifecycle.
- [x] Standard resource lifecycle commands through `resource-vault`: `info`, `status`, `install`, `uninstall`, `start`, `stop`, `restart`, `logs`.
- [x] Durable local Vault file storage under canonical resource data/config/log directories.
- [x] Local first-use initialization and unseal for the managed container.
- [x] KV v2 content commands: `content get`, `content set`, `content add`, `content delete`, `content remove`, `content list`.
- [x] `content get --format raw` machine contract for direct value reads.
- [x] `content get --format json` passthrough for Vault JSON.
- [x] `content set` patch-then-put behavior to preserve sibling fields.
- [x] Rejection of `@`-prefixed values to avoid implicit file reads.
- [x] Resource secret inventory commands: `secrets scan`, `secrets check`, `secrets validate`, `secrets export`, `secrets provision`.
- [x] `secrets init` alias for non-interactive provisioning where existing resources call it.
- [x] Kopia consumer integration through `resource-vault content get/set`.
- [x] Kopia missing-secret handling distinguishes absence from Vault/Docker failure.

## Not Implemented

- [ ] Enterprise HA Vault.
- [ ] Production TLS listener configuration.
- [ ] Auto-unseal with KMS/HSM.
- [ ] Namespaces or tenant isolation.
- [ ] Dynamic database credentials.
- [ ] PKI or SSH CA workflows.
- [ ] Interactive secret prompting.
- [ ] Secret rotation automation.
- [ ] Audit device enablement and security scoring CLI.
- [ ] Per-resource Vault policies and scoped tokens.
- [ ] Bulk migration/export commands from legacy shell workflows.

## CLI Contract

Resources and scenarios must use `resource-vault`.

```bash
resource-vault content set --path <kv-path> [--key <field>] --value <value>
resource-vault content get --path <kv-path> [--key <field>] [--format raw|json]
resource-vault content list --path <kv-prefix>
resource-vault content delete --path <kv-path>
```

Single-value secrets use field `value` by default. Kopia passphrases use field `passphrase`.

Resource declarations:

```bash
resource-vault secrets scan
resource-vault secrets check <resource>
resource-vault secrets validate [resource]
resource-vault secrets export <resource>
resource-vault secrets provision <resource>
```

## Runtime Posture

The local runtime is durable across container restarts. It is acceptable for local backup passphrase storage, assuming the operator protects the resource data directory that contains local bootstrap material.

This resource is not yet a fully managed external production Vault. External production use requires explicit work on TLS, unseal strategy, policy design, audit logging, storage backup, and HA.

## Quality Gates

- `resources/vault/cli`: `GOWORK=off go test ./...`
- `resources/vault`: `make check`
- `resource-vault status`
- `resource-vault content set/get/delete` smoke test.
- Restart persistence test.
- `resource-kopia` repository create/status/snapshot/restore validation using a temporary filesystem repository.
