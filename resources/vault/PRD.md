# Product Requirements Document - Vault Resource

## Purpose

Vault provides an optional local KV and Transit capability for Vrooli resources and scenarios that explicitly require Vault semantics. It is not the ordinary credential authority; ordinary resource, integration, and backup credentials use the control-plane credential authority.

## Implemented

- [x] Signed native managed-service resource manifest with shared lifecycle.
- [x] Standard resource lifecycle commands through `resource-vault`: `info`, `status`, `install`, `uninstall`, `start`, `stop`, `restart`, `logs`.
- [x] Durable local Vault file storage under canonical resource data/config/log directories.
- [x] Native runtime preserves the established durable data directory without a Docker compatibility path.
- [x] KV v2 content commands: `content get`, `content set`, `content add`, `content delete`, `content remove`, `content list`.
- [x] `content get --format raw` machine contract for direct value reads.
- [x] `content get --format json` passthrough for Vault JSON.
- [x] `content set` patch-then-put behavior to preserve sibling fields.
- [x] Rejection of `@`-prefixed values to avoid implicit file reads.
- [x] Vault capability lifecycle and scoped content commands.
- [x] `secrets init` alias for non-interactive provisioning where existing resources call it.
- [x] Explicit capability boundary: ordinary credentials are handled by the control-plane credential authority, while only declared Vault-backed capabilities use `resource-vault`.
- [x] Broker-authorized shared use issues per-app, lease-bounded Vault child tokens with isolated KV v2 policy prefixes; management credentials are never persisted or returned to applications.
- [x] Explicit attach-only endpoint health validation without lifecycle authority.

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
- [ ] Bulk migration/export commands from legacy shell workflows.

## CLI Contract

Capabilities that explicitly require Vault semantics must use `resource-vault`.

```bash
resource-vault content set --path <kv-path> [--key <field>] --value <value>
resource-vault content get --path <kv-path> [--key <field>] [--format raw|json]
resource-vault content list --path <kv-prefix>
resource-vault content delete --path <kv-path>
```

Single-value Vault content uses field `value` by default. Capability-specific consumers define any other fields they require.

Resource declarations:

```bash
vrooli credentials status --identity <logical-id> --field <field>
```

## Runtime Posture

The local runtime is durable across native managed-service restarts. It is not the default storage path for backup passphrases or ordinary resource credentials. Those values belong to the control-plane credential authority, assuming the operator protects the selected native key service or encrypted portable backend. Normal Vault lifecycle paths do not write, discover, or expose bootstrap material.

This resource is not yet a fully managed external production Vault. External production use requires explicit work on TLS, unseal strategy, policy design, audit logging, storage backup, and HA.

## Maturity Assessment

Vault currently meets **M4 (template-conformant), with conditional desktop
profiles** under the resource maturity playbook. Its manifest is authoritative,
normal lifecycle and diagnostics use Go-native managed-service paths, the
release catalog pins a verified artifact for every declared desktop target, and
normal operations do not require Docker or Bash. Focused runtime integration
also proves lifecycle restart and app-scope isolation with a real Vault
artifact; explicitly governed Vault consumers retain their supported CLI contracts.

It must not be classified as M5 yet. Each macOS and Windows profile remains
conditional until the target-host smoke test runs against that platform's
signed artifact. The full Plan Manager baseline also remains non-comparable
because of pre-existing affected-scenario failures. See
[`docs/OPERATIONS.md`](docs/OPERATIONS.md#target-host-smoke-evidence) for the
native-host evidence command.

## Capability Evidence Map

| Capability | Focused evidence |
| --- | --- |
| Verified private lifecycle, restart, and persistence | `TestManagedVaultArtifactIntegration` and `TestDesktopVaultArtifactIntegration` with a staged signed Vault artifact |
| Artifact integrity and lifecycle ownership | `TestManagedServiceSupervisorStartsVerifiedArtifactAndStops`, `TestManagedServiceSupervisorRejectsTamperedArtifact`, and `TestManagedServiceArtifactPathUsesInstalledSignedArtifactStore` |
| Shared app isolation and consent | `TestVaultCredentialIssuerCreatesLeaseBoundScopedToken`, `TestProviderPolicyUsesTargetDefaultsWithoutOverridingExplicitChoice`, and the desktop runtime shared-broker tests |
| Attach-only safety | `TestManagedServiceDriverRefusesAttachOnlyLifecycle` and `TestManagedServiceDriverValidatesExplicitAttachOnlyEndpointWithoutLifecycle` |
| Remote scenario boundary | `TestNativeRunnerRejectsDirectRemoteVrooliAccess` and `TestStatusRejectsDirectRemoteVrooliAccess` |
| Consumer compatibility | `go test ./...` in explicitly governed Vault consumers |

Run the target-host smoke test for each conditional target before upgrading its
support claim. The evidence table is a correspondence map, not a substitute
for native target execution.

## Quality Gates

- `resources/vault/cli`: `GOWORK=off go test ./...`
- `resources/vault`: `make check`
- `resource-vault status`
- `resource-vault content set/get/delete` smoke test.
- Restart persistence test.
