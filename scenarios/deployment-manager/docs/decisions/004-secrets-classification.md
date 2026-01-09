# ADR-004: Four-Class Secrets Model

## Status

Accepted

## Context

Secrets handling is one of the hardest problems in portable deployments:

- **Tier 1 (local dev)**: Secrets stored in `.env` files, Vault, or environment variables
- **Tier 2+ (bundles)**: No access to infrastructure. How do bundles get secrets?

Naive approaches failed:
1. **Bundle secrets directly**: Security disaster. Secrets visible in package.
2. **Require network**: Breaks offline-capable bundles.
3. **Ignore secrets**: Apps fail on first run with cryptic errors.

Different secrets have fundamentally different characteristics:
- Database passwords: Tied to infrastructure, can't leave the server
- JWT signing keys: Unique per installation, can be generated
- API keys: User must provide from external service
- Session secrets: Can be randomly generated on first run

## Decision

Classify all secrets into **four classes** based on how they can be provisioned in bundles:

| Class | Description | Bundle Strategy |
|-------|-------------|-----------------|
| `infrastructure` | Tied to external systems (DB passwords, Vault tokens) | **Never bundled** - must be swapped out |
| `per_install_generated` | Unique per installation, can be auto-generated | Generated on first run |
| `user_prompt` | User must provide (API keys, licenses) | First-run wizard prompts |
| `remote_fetch` | Retrieved from external vault at runtime | Fetched on start (requires network) |

### Manifest Secret Schema

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
  "target": { "type": "env", "name": "JWT_SECRET" }
}
```

```json
{
  "id": "openai_api_key",
  "class": "user_prompt",
  "description": "OpenAI API key for AI features",
  "format": "^sk-[a-zA-Z0-9]{48}$",
  "prompt": {
    "label": "OpenAI API Key",
    "description": "Enter your OpenAI API key from platform.openai.com"
  },
  "target": { "type": "env", "name": "OPENAI_API_KEY" }
}
```

### Validation Rules

1. **infrastructure secrets block bundling**: If a scenario requires infrastructure secrets and doesn't swap them out, bundle fitness = 0
2. **per_install_generated must have generator**: Manifest validation fails without generator config
3. **user_prompt must have prompt**: Must define label and description for wizard
4. **remote_fetch requires network**: Warns that offline mode won't work

## Consequences

### Positive

- **Clear blocking rules**: Infrastructure secrets automatically block unsafe bundling
- **User-friendly onboarding**: First-run wizard knows exactly what to ask
- **Automation**: per_install_generated secrets need no user interaction
- **Flexibility**: Four classes cover all real-world cases we've encountered
- **Audit trail**: Manifest documents what secrets exist and how they're handled

### Negative

- **Migration work**: Existing scenarios must classify their secrets
- **Complexity**: Developers must understand classification to write manifests
- **Partial offline**: remote_fetch class prevents true offline operation

### Neutral

- secrets-manager scenario owns classification logic
- deployment-manager queries secrets-manager for classification
- Runtime supervisor handles generation and injection

## Alternatives Considered

### Two Classes (Bundleable/Not Bundleable)

Simple binary classification.

**Rejected because**: Doesn't distinguish between auto-generated and user-prompted. Both are "bundleable" but handled very differently.

### External Secrets Only

All secrets fetched from external vault (no bundled secrets).

**Rejected because**: Breaks offline operation. Requires network on every start. Many users want true standalone apps.

### User Provides All Secrets

No auto-generation. User provides everything in first-run wizard.

**Rejected because**: Poor UX. Users shouldn't have to generate random strings for internal secrets like JWT keys.

### Six Classes (More Granular)

Split user_prompt into required/optional, split infrastructure into database/service.

**Rejected because**: Diminishing returns. Four classes handle 99% of cases. Adding more increases cognitive load.

## References

- [Secrets Management Guide](../guides/secrets-management.md)
- [secrets-manager Scenario](/scenarios/secrets-manager/)
- [Example Manifest Secrets](../examples/manifests/desktop-happy.json)
