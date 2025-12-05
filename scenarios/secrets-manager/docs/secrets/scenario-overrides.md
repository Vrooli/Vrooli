# Scenario Secret Strategy Overrides

This document describes how to use scenario-specific overrides to customize secret handling strategies for individual scenarios, without modifying the global resource defaults.

## Overview

By default, each resource secret has a **tier-specific handling strategy** defined at the resource level. These strategies apply globally to all scenarios that use the resource.

**Scenario overrides** allow you to customize how a specific scenario handles a secret, without affecting other scenarios. This is useful when:

- A scenario needs to exclude a secret that's normally required (e.g., a desktop app that doesn't use a database)
- A scenario needs user prompts instead of auto-generation
- A scenario has special security requirements

## Data Model

### Database Table

Overrides are stored in `scenario_secret_strategy_overrides`:

```sql
CREATE TABLE scenario_secret_strategy_overrides (
    id UUID PRIMARY KEY,
    scenario_name VARCHAR(200) NOT NULL,
    resource_secret_id UUID NOT NULL REFERENCES resource_secrets(id),
    tier VARCHAR(50) NOT NULL,

    -- Override fields (null = inherit from resource default)
    handling_strategy VARCHAR(20),
    fallback_strategy VARCHAR(20),
    requires_user_input BOOLEAN,
    prompt_label TEXT,
    prompt_description TEXT,
    generator_template JSONB,
    bundle_hints JSONB,

    override_reason TEXT,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,

    UNIQUE(scenario_name, resource_secret_id, tier)
);
```

### Strategy Precedence

When generating deployment manifests, strategies are resolved with COALESCE:

```
Effective Strategy = COALESCE(
    scenario_override,      -- Highest priority
    resource_tier_default,  -- Resource-level tier strategy
    'unspecified'           -- Fallback
)
```

## API Endpoints

### List All Overrides for a Scenario

```http
GET /api/v1/scenarios/{scenario}/overrides
```

Returns all overrides for a scenario across all tiers.

### List Overrides for Scenario + Tier

```http
GET /api/v1/scenarios/{scenario}/overrides/{tier}
```

### Get Specific Override

```http
GET /api/v1/scenarios/{scenario}/overrides/{tier}/{resource}/{secret}
```

### Create/Update Override

```http
POST /api/v1/scenarios/{scenario}/overrides/{tier}/{resource}/{secret}
Content-Type: application/json

{
  "handling_strategy": "strip",
  "override_reason": "Desktop app doesn't use this database"
}
```

**Valid handling strategies:** `strip`, `generate`, `prompt`, `delegate`

### Delete Override (Revert to Default)

```http
DELETE /api/v1/scenarios/{scenario}/overrides/{tier}/{resource}/{secret}
```

### Get Effective Strategies

Returns the merged view of resource defaults + scenario overrides:

```http
GET /api/v1/scenarios/{scenario}/effective/{tier}?resources=postgres,redis
```

Response includes `is_overridden` and `overridden_fields` for each secret.

### Copy Overrides Between Tiers

```http
POST /api/v1/scenarios/{scenario}/overrides/copy-from-tier
Content-Type: application/json

{
  "source_tier": "tier-2-desktop",
  "target_tier": "tier-3-mobile",
  "overwrite": false
}
```

### Copy Overrides from Another Scenario

```http
POST /api/v1/scenarios/{scenario}/overrides/copy-from-scenario
Content-Type: application/json

{
  "source_scenario": "other-scenario",
  "tier": "tier-2-desktop",
  "overwrite": false
}
```

## Admin Endpoints

### List Orphan Overrides

Finds overrides for scenarios that no longer exist or no longer depend on the resource:

```http
GET /api/v1/admin/overrides/orphans
```

### Cleanup Orphan Overrides

```http
POST /api/v1/admin/overrides/cleanup
Content-Type: application/json

{
  "dry_run": true
}
```

Set `dry_run: false` to actually delete orphans.

## UI Usage

In the secrets-manager UI:

1. Select a scenario from the dropdown
2. Open the Resource Panel for a secret
3. Check "Override for scenario: {name}" checkbox
4. Configure the desired strategy
5. Optionally add an override reason
6. Click "Apply scenario override"

The UI shows:
- A purple indicator when a secret has an active override
- The override reason (if provided)
- A "Remove override" button to revert to resource defaults

## Examples

### Example 1: Exclude Database Secret for Desktop App

A desktop app doesn't use PostgreSQL, but the resource declares `POSTGRES_PASSWORD` as required.

```bash
curl -X POST http://localhost:PORT/api/v1/scenarios/desktop-app/overrides/tier-2-desktop/postgres/POSTGRES_PASSWORD \
  -H "Content-Type: application/json" \
  -d '{
    "handling_strategy": "strip",
    "override_reason": "Desktop version uses local SQLite instead"
  }'
```

### Example 2: Require User Input for API Key

A scenario needs users to provide their own API key instead of using a generated one:

```bash
curl -X POST http://localhost:PORT/api/v1/scenarios/byok-app/overrides/tier-2-desktop/openai/OPENAI_API_KEY \
  -H "Content-Type: application/json" \
  -d '{
    "handling_strategy": "prompt",
    "requires_user_input": true,
    "prompt_label": "Your OpenAI API Key",
    "prompt_description": "Enter your personal OpenAI API key. You can get one at platform.openai.com",
    "override_reason": "Users must provide their own API key for this app"
  }'
```

### Example 3: Copy Desktop Overrides to Mobile

```bash
curl -X POST http://localhost:PORT/api/v1/scenarios/my-app/overrides/copy-from-tier \
  -H "Content-Type: application/json" \
  -d '{
    "source_tier": "tier-2-desktop",
    "target_tier": "tier-3-mobile",
    "overwrite": false
  }'
```

## Validation

The API validates:

1. **Strategy values** - Only `strip`, `generate`, `prompt`, `delegate` are allowed
2. **Secret existence** - The resource/secret must exist in `resource_secrets`
3. **Scenario dependency** - The scenario should depend on the resource (warning only)

## Integration with Deployment Manifests

When `POST /api/v1/deployment/secrets` is called with a scenario, the manifest automatically:

1. Fetches resource secrets
2. Joins with `secret_deployment_strategies` for tier defaults
3. Left-joins with `scenario_secret_strategy_overrides` for scenario overrides
4. Uses COALESCE to apply override precedence
5. Returns the effective strategy in the manifest

No additional configuration is needed - overrides are automatically applied.

## Database Migration

Run the migration to create the overrides table:

```bash
psql -d secrets_manager -f initialization/storage/postgres/migrations/002_scenario_overrides.sql
```

Or if using the full schema:

```bash
psql -d secrets_manager -f initialization/storage/postgres/schema.sql
```
