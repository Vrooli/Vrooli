# Secrets Management for Deployment

Deployment fails fast if secrets leak or cannot be provisioned. This guide formalizes the secret classification model and how secrets are handled across deployment tiers.

## Quick Reference

| Secret Class | Bundle Behavior | Example |
|--------------|-----------------|---------|
| `infrastructure` | **Never bundled** | DB passwords, Vault tokens |
| `per_install_generated` | Auto-generated on first run | JWT keys, session secrets |
| `user_prompt` | User provides during first-run wizard | API keys, SMTP creds |
| `remote_fetch` | Fetched from external vault at runtime | Enterprise secrets |

---

## Secret Classes

### 1. Infrastructure Secrets (Never Ship)

- **Examples**: master DB passwords, Vault root tokens, Cloudflare tunnel certs, secrets-manager encryption keys.
- **Handling**: Remain only on the core Vrooli server. Deployment bundles reference placeholders.
- **Manifest class**: `"class": "infrastructure"` (but these should be removed via swaps, not included)

> **Critical**: If a scenario requires infrastructure secrets, it cannot be bundled until those dependencies are swapped out. For example, swapping `postgres` → `sqlite` removes the need for `POSTGRES_PASSWORD`.

### 2. Per-Install Generated Secrets (Auto-Created)

- **Examples**: API JWT signing keys, per-install database credentials, inter-service auth tokens.
- **Handling**: The runtime supervisor generates these on first run using the `generator` config in the manifest.
- **Manifest class**: `"class": "per_install_generated"`

```json
{
  "id": "jwt_secret",
  "class": "per_install_generated",
  "description": "JWT signing key",
  "format": "^[A-Za-z0-9]{32,}$",
  "generator": {
    "type": "random",
    "length": 32,
    "charset": "alnum"
  },
  "target": {
    "type": "env",
    "name": "JWT_SECRET"
  }
}
```

**Generator types:**
- `random` - Random string with configurable `length` and `charset` (`alnum`, `hex`, `base64`)
- `uuid` - UUID v4

### 3. User-Prompted Secrets (Provided at Runtime)

- **Examples**: OpenRouter API keys, OAuth client secrets provided by the customer, SMTP credentials.
- **Handling**: Bundles include first-run prompts. The user provides these during the first-run wizard.
- **Manifest class**: `"class": "user_prompt"`

```json
{
  "id": "openrouter_api_key",
  "class": "user_prompt",
  "description": "API key for AI features",
  "format": "^sk-or-.*$",
  "prompt": {
    "label": "OpenRouter API Key",
    "description": "Enter your OpenRouter API key to enable AI features. Get one at openrouter.ai"
  },
  "required": true,
  "target": {
    "type": "env",
    "name": "OPENROUTER_API_KEY"
  }
}
```

### 4. Remote-Fetch Secrets (Enterprise/Advanced)

- **Examples**: Secrets stored in HashiCorp Vault, AWS Secrets Manager, Azure Key Vault.
- **Handling**: Manifest contains a reference; the runtime fetches at startup.
- **Manifest class**: `"class": "remote_fetch"`

---

## First-Run Wizard

When a bundled desktop app launches for the first time, the runtime supervisor handles secret initialization:

### Wizard Flow

```
┌─────────────────────────────────────────────────────────────┐
│                    First-Run Wizard                         │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Step 1: Auto-Generate Secrets                              │
│  ─────────────────────────────                              │
│  ✓ JWT_SECRET generated (32 chars)                          │
│  ✓ SESSION_KEY generated (32 chars)                         │
│                                                             │
│  Step 2: User-Provided Secrets                              │
│  ─────────────────────────────                              │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐    │
│  │ OpenRouter API Key                                  │    │
│  │ Enter your OpenRouter API key to enable AI features │    │
│  │                                                     │    │
│  │ [sk-or-________________________]                    │    │
│  │                                                     │    │
│  │ ℹ️ Get one at openrouter.ai                         │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                             │
│  Step 3: Validation                                         │
│  ─────────────────                                          │
│  ✓ All required secrets provided                            │
│  ✓ Format validation passed                                 │
│                                                             │
│                              [Continue →]                   │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Implementation Details

**Where secrets are stored:**
- **Electron apps**: Uses `safeStorage` API for encrypted storage
- **Location**: `~/.config/<app-name>/secrets.json` (encrypted)

**Validation:**
- Secrets with `format` field are validated against the regex
- Required secrets block startup until provided
- Optional secrets can be skipped

**Re-configuration:**
- Users can update secrets via the app's Settings menu
- The runtime exposes `POST /secrets` on the control API

### Runtime Control API for Secrets

```bash
# Get current secret status (not values)
curl http://127.0.0.1:<ipc_port>/secrets \
  -H "Authorization: Bearer <token>"

# Response:
{
  "secrets": [
    {"id": "jwt_secret", "status": "configured", "source": "generated"},
    {"id": "openrouter_api_key", "status": "configured", "source": "user_provided"},
    {"id": "smtp_password", "status": "missing", "required": false}
  ]
}

# Submit a secret
curl -X POST http://127.0.0.1:<ipc_port>/secrets \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"id": "smtp_password", "value": "secret123"}'
```

---

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

---

## CLI Commands

```bash
# Identify required secrets for a profile
deployment-manager secrets identify <profile-id>

# Generate a secrets template (for documentation/setup)
deployment-manager secrets template <profile-id> --format env
deployment-manager secrets template <profile-id> --format json

# Validate that all required secrets are configured
deployment-manager secrets validate <profile-id>
```

---

## Related Documentation

- [Bundle Manifest Schema](bundle-manifest-schema.md) - Secret field definitions
- [Desktop Workflow](../workflows/desktop-deployment.md) - Phase 4 covers secrets
- [CLI Secret Commands](../cli/secret-commands.md) - Full command reference
- [Fitness Scoring](fitness-scoring.md) - How secrets affect deployment readiness
