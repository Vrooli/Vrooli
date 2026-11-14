# Operations

`vault` is a signed `managed-service` resource managed through the shared Vrooli lifecycle.

## Boundaries

- `resource.json` owns the pinned server artifact, loopback configuration, provider policy, ports, health checks, and lifecycle metadata.
- `cli/` owns the `resource-vault` binary and custom resource commands.
- `cli/internal/content` owns the canonical KV v2 content wrapper.
- Credential inventory and provisioning are owned by the `vrooli credentials`
  control-plane command and manifest descriptors.

## Runtime Mode

The `control-plane` target defaults to one verified, Vrooli-owned per-user
Vault host with a signed server artifact and file storage:

```json
storage "file" { path = "${RESOURCE_DATA_DIR}" }
```

Data, config, and logs use canonical resource storage directories. The listener is loopback-only; desktop bundles allocate a private port at launch, while the local control plane uses the declared resource port after proving it is not already occupied. The service artifact is checked before launch. This mode is durable across service restarts, but it is not the default storage path for backup passphrases or ordinary resource credentials.

The user host initializes a fresh Vault once and stores its recovery material
only in the operating-system credential store. It recovers an initialized
Vault from that store on restart, then issues short-lived app-scoped tokens.
If the credential-store adapter is unavailable, startup fails before
initialization. It never falls back to a plaintext token/state file. This is
not HA or auto-unseal.

## Provider Modes

`managed-shared` is the control-plane default user-host mode. The Vrooli resource host owns
its lifecycle and the broker issues a per-app child token without exposing its
management token. `managed-private` creates an isolated instance only when a
scenario declares that requirement. `remote-vrooli` is a scenario API boundary, not
a direct desktop-to-Vault connection. When `VROOLI_MANAGED_PROVIDER` is
`remote-vrooli`, `resource-vault` rejects direct content and status requests;
the desktop client must use the scenario API instead.

An organization-managed Vault is attach-only. Validate it explicitly without
adopting or managing it:

```bash
vrooli resource status vault --provider=attach-only --endpoint=https://vault.example.internal
```

The endpoint URL cannot include credentials, a query, or a fragment. Vrooli
only requests its declared health path (`/v1/sys/health`) and will never start,
stop, initialize, unseal, or rewrite an attach-only endpoint.

Attach-only requests must declare either `read-only` or `read-write` access in
the resource provider policy. Neither capability grants lifecycle authority.

## Operational Checks

```bash
resource-vault status
vrooli resource status vault
resource-vault content set --path secret/capabilities/example/ops --key value --value ok
resource-vault content get --path secret/capabilities/example/ops --key value --format raw
resource-vault content delete --path secret/capabilities/example/ops
```

For persistence:

```bash
resource-vault content set --path secret/capabilities/example/persistence --key value --value survives-restart
vrooli resource restart vault
resource-vault content get --path secret/capabilities/example/persistence --key value --format raw
resource-vault content delete --path secret/capabilities/example/persistence
```

## Target-Host Smoke Evidence

Run the managed-service integration test on each claimed operating-system and
architecture pair with that target's verified, signed Vault artifact. The test
starts the real server through the native private bootstrap adapter, requires
post-bootstrap health (never `501`), restarts it, then proves that an
app-scoped token cannot write another app's path and can read its persisted
path after native recovery. It also exercises the verified
`managed-discovered` executable path with the same artifact.

```bash
VROOLI_VAULT_INTEGRATION_BINARY=/absolute/path/to/vault_target \
  go test ./internal/resources -run '^TestManagedVaultArtifactIntegration$' -count=1 -v

VROOLI_VAULT_INTEGRATION_BINARY=/absolute/path/to/vault_target \
  go test ./scenarios/scenario-to-desktop/runtime/resources -run '^TestDesktopVaultArtifactIntegration$' -count=1 -v
```

On Windows PowerShell:

```powershell
$env:VROOLI_VAULT_INTEGRATION_BINARY = 'C:\artifacts\vault_windows_amd64.exe'
go test ./internal/resources -run '^TestManagedVaultArtifactIntegration$' -count=1 -v
```

Use the platform's native executable name: the test accepts both a Unix-style
`vault_target` artifact and a Windows `vault_target.exe` artifact. This is a
target-host smoke test; cross-compilation alone does not establish runtime
support.

Linux also requires a live Secret Service implementation exposed by
`secret-tool` for the turnkey user-host path. If `secret-tool` is unavailable,
retain Linux as `conditional` and do not initialize Vault. Artifact and
private-mode tests do not establish turnkey bootstrap evidence.

Install the declared host tool only through Vrooli, then let the managed
resource perform its non-secret credential-store probe:

```bash
vrooli host install secret-tool --sudo-mode=ask
vrooli resource start vault
vrooli resource status vault
```

An interactive administrator prompt can be required for the host-tool install.
If the Secret Service probe still fails, unlock the active user's Secret
Service session and retry the resource start. Never replace this prerequisite
with a plaintext bootstrap or recovery file.

## Secret Handling

Default status, check, validate, list, and lifecycle commands must not print
credential values. Ordinary credential values have no Vault CLI read or export
workflow.

## Production Gaps

Before using Vault for external production tenants, implement and validate at least:

- TLS listener and certificate management for non-loopback deployments.
- Externalized unseal strategy and a proper operator runbook.
- Policy and token model with least privilege per resource.
- Audit device enablement and log retention.
- Backup and restore of Vault storage and bootstrap material.
- HA storage and disaster recovery design.
