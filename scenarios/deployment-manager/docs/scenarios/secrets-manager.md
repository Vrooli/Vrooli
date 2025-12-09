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

## Milestones

1. **Schema Update** — Add deployment strategy fields + validation for infrastructure/service/user secrets.
2. **Planning CLI/API** — `vrooli secrets plan <scenario> --tier desktop` outputs manifest consumed by deployment-manager.
3. **Generation Hooks** — Provide library helpers to generate service secrets (SQLite paths, JWTs) at bundle creation time.
4. **User Prompt SDK** — Shared UI helpers for scenario-to-desktop/mobile to capture secrets securely.

## Success Metrics

- 100% of secrets referenced by deployment-focused scenarios carry a deployment strategy within two releases.
- Secrets plans can be generated in <1s per scenario/tier combination.
- Desktop/mobile bundles never include infrastructure secrets; compliance is enforced automatically.

## Risks

- **Sprawl:** Without governance, teams might define conflicting strategies; enforce linting + review.
- **Storage Safety:** Desktop/mobile key stores vary; need fallback for Linux builds.
- **Rotation:** Generated service secrets must support rotation + revocation workflows downstream.
