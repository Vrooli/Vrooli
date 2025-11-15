# Secrets Management for Deployment

Deployment fails fast if secrets leak or cannot be provisioned. This guide formalizes the three-tier secret model and how `secrets-manager` will integrate with deployment-manager.

## Secret Classes

1. **Infrastructure Secrets (Never Ship)**
   - Examples: master DB passwords, Vault root tokens, Cloudflare tunnel certs, secrets-manager encryption keys.
   - Handling: Remain only on the core Vrooli server. Deployment bundles reference placeholders.

2. **Service Secrets (Regenerate Per Deployment)**
   - Examples: API JWT signing keys, per-install database credentials, inter-service auth tokens.
   - Handling: deployment-manager requests fresh values from secrets-manager during bundle creation. For desktop/mobile tiers this might be a generated file stored locally; for SaaS tiers they go into provider-specific secret stores.

3. **User Secrets (Provided at Runtime)**
   - Examples: OpenRouter API keys, OAuth client secrets provided by the customer, SMTP credentials.
   - Handling: Bundles include first-run prompts or configuration UIs. secrets-manager defines the schema and validation rules.

## Metadata Example

```json
{
  "secret_id": "postgres_connection",
  "deployment_strategy": {
    "local_dev": {
      "type": "infrastructure",
      "source": "secrets_vault",
      "value": "postgresql://localhost:5432/app"
    },
    "desktop": {
      "type": "service",
      "action": "generate_sqlite_path",
      "template": "file://{{USER_DATA}}/app.db"
    },
    "cloud": {
      "type": "service",
      "action": "environment_variable",
      "template": "{{POSTGRES_URL}}"
    }
  }
}
```

## Process Flow

1. Deployment-manager asks secrets-manager for the secret manifest filtered by tier.
2. Secrets-manager returns:
   - Which secrets to strip (infra).
   - Which secrets to generate and embed.
   - Which secrets to prompt the user for, including UI copy and validation rules.
3. scenario-to-* packagers embed the generated secrets, prompt logic, and secure storage (e.g., Electron `safeStorage`, mobile Keychain, cloud secret stores).

## Tooling Wishlist

- CLI: `vrooli secrets plan --scenario picker-wheel --tier desktop` → shows what will be bundled, generated, prompted.
- Integration: deployment-manager UI highlights missing secret strategies.
- Auditing: secrets-manager tracks when/where secrets are deployed to support revocation.

## Interim Guidance

- Document secret requirements in scenario READMEs now, explicitly marking which category they fall under.
- Never copy infrastructure secrets into bundles or test artifacts.
- When building thin clients, prompt users to connect to an existing Tier 1 server and supply their own API keys.
