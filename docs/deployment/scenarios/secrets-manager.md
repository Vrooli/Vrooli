# Scenario: secrets-manager (Deployment Role)

## Current Role

- Stores secrets for local development using filesystem vaults, `.env` files, or HashiCorp Vault.
- Provides APIs/CLIs for retrieving secrets when scenarios boot.

## Deployment Requirements

1. **Classification** — Each secret must be tagged as Infrastructure, Service, or User (see [Secrets Guide](../guides/secrets-management.md)).
2. **Strategy Definitions** — For every tier, secrets-manager should describe how to handle the secret:
   - `strip` (infrastructure)
   - `generate` (service)
   - `prompt` (user)
   - `delegate` (ask for provider-managed secret)
3. **Template Engine** — Provide templates for generated secrets (file paths, env vars, config files).
4. **Audit Trail** — Track which deployments received which generated secrets for rotation/revocation.
5. **API for deployment-manager** — `GET /deployment/secrets?scenario=picker-wheel&tier=desktop` returning the manifest.

## Integration Points

- Deployment-manager uses the API/CLI to show secret plans.
- scenario-to-desktop/mobile/cloud embeds prompts/generators according to the manifest.
- app-issue-tracker receives tasks if a secret lacks a deployment strategy.

## Near-Term Tasks

- Extend secret metadata schema to include deployment strategies.
- Build CLI command to preview secret plans (`vrooli secrets plan ...`).
- Document best practices for storing user-provided secrets on desktop/mobile (safeStorage, Keychain, etc.).
