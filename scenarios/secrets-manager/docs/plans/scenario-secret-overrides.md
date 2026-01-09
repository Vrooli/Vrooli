# Scenario Secret Strategy Overrides - Implementation Plan

## Overview

Currently, secret handling strategies are stored globally at the resource+tier level. When a scenario depends on a resource, it inherits that resource's strategies with no ability to customize them. This plan introduces **scenario-level overrides** that allow individual scenarios to customize how their dependent resources' secrets are handled during deployment.

## Problem Statement

1. **No scenario customization**: If resource `postgres` has `DB_PASSWORD` set to "generate" for tier-2-desktop, all scenarios using postgres inherit this - even if one scenario needs "prompt" instead.
2. **Strategy conflicts**: Different scenarios may have different deployment contexts (e.g., one for power users who can configure databases, another for casual users needing a zero-config experience).
3. **Overly broad changes**: Changing a resource's strategy affects ALL scenarios that use it.

## Design Goals

1. **Backward compatible**: Existing behavior (resource-level defaults) works unchanged
2. **Override semantics**: Scenario overrides take precedence over resource defaults when present
3. **Clear UX**: Users can see which values are inherited vs. overridden
4. **Minimal footprint**: Only store overrides, not full copies of every strategy
5. **Data integrity**: Prevent and clean up orphan overrides

---

## Data Integrity Considerations

### Orphan Override Prevention

Since `scenario_name` is stored as a string (not a FK), overrides can become orphaned when:
1. A scenario is deleted or renamed
2. A scenario removes a resource from its dependencies

**Mitigation strategies:**

1. **Validation at creation**: When creating an override, validate that:
   - The scenario exists (check for `service.json` in scenarios directory)
   - The scenario depends on the resource (check `service.json` dependencies)
   - The tier is supported by the scenario (check `service.json` deployment tiers)

2. **Cleanup endpoint**: Add an administrative endpoint to audit and clean orphan overrides:
   ```
   GET  /admin/overrides/orphans         - List overrides for non-existent scenarios or removed dependencies
   POST /admin/overrides/cleanup         - Remove orphan overrides (with dry-run option)
   ```

3. **Soft validation on read**: When fetching overrides, flag any that no longer match the scenario's current dependencies (warn but don't fail).

### Dependency Drift

When a scenario's dependencies change after overrides exist:
- **Resource removed**: Override becomes orphan (handled by cleanup)
- **Resource added**: No issue (new resource uses defaults until override created)
- **Scenario renamed**: All overrides for old name become orphans

**Future consideration**: If a scenarios registry/database is added, migrate `scenario_name` to a FK with `ON DELETE CASCADE`.

---

## Phase 1: Database Schema

### 1.1 New Table: `scenario_secret_strategy_overrides`

```sql
-- Stores scenario-specific overrides for secret handling strategies.
-- Only stores DIFFERENCES from the resource default - null fields mean "inherit".
CREATE TABLE IF NOT EXISTS scenario_secret_strategy_overrides (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    scenario_name VARCHAR(200) NOT NULL,
    resource_secret_id UUID NOT NULL REFERENCES resource_secrets(id) ON DELETE CASCADE,
    tier VARCHAR(50) NOT NULL,

    -- Override fields (null = inherit from resource default)
    handling_strategy VARCHAR(20) CHECK (handling_strategy IS NULL OR handling_strategy IN ('strip', 'generate', 'prompt', 'delegate')),
    fallback_strategy VARCHAR(20),
    requires_user_input BOOLEAN,
    prompt_label TEXT,
    prompt_description TEXT,
    generator_template JSONB,
    bundle_hints JSONB,

    -- Metadata
    override_reason TEXT,  -- Optional: Why this scenario needs different handling
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- Unique constraint: one override per (scenario, secret, tier)
    UNIQUE(scenario_name, resource_secret_id, tier)
);

-- Index for efficient lookup by scenario
CREATE INDEX IF NOT EXISTS idx_scenario_overrides_scenario
    ON scenario_secret_strategy_overrides(scenario_name);

-- Index for efficient lookup by resource secret
CREATE INDEX IF NOT EXISTS idx_scenario_overrides_secret
    ON scenario_secret_strategy_overrides(resource_secret_id);

COMMENT ON TABLE scenario_secret_strategy_overrides IS
    'Stores scenario-specific overrides for secret handling strategies. Null fields inherit from resource defaults.';
```

### 1.2 Migration Script

File: `initialization/storage/postgres/migrations/002_scenario_overrides.sql`

```sql
-- Migration: Add scenario_secret_strategy_overrides table
-- Apply with: psql -d secrets_manager -f migrations/002_scenario_overrides.sql

BEGIN;

-- Create table if not exists
CREATE TABLE IF NOT EXISTS scenario_secret_strategy_overrides (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    scenario_name VARCHAR(200) NOT NULL,
    resource_secret_id UUID NOT NULL REFERENCES resource_secrets(id) ON DELETE CASCADE,
    tier VARCHAR(50) NOT NULL,
    handling_strategy VARCHAR(20) CHECK (handling_strategy IS NULL OR handling_strategy IN ('strip', 'generate', 'prompt', 'delegate')),
    fallback_strategy VARCHAR(20),
    requires_user_input BOOLEAN,
    prompt_label TEXT,
    prompt_description TEXT,
    generator_template JSONB,
    bundle_hints JSONB,
    override_reason TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(scenario_name, resource_secret_id, tier)
);

CREATE INDEX IF NOT EXISTS idx_scenario_overrides_scenario
    ON scenario_secret_strategy_overrides(scenario_name);
CREATE INDEX IF NOT EXISTS idx_scenario_overrides_secret
    ON scenario_secret_strategy_overrides(resource_secret_id);

-- Add trigger for updated_at
DROP TRIGGER IF EXISTS update_scenario_overrides_updated_at ON scenario_secret_strategy_overrides;
CREATE TRIGGER update_scenario_overrides_updated_at
    BEFORE UPDATE ON scenario_secret_strategy_overrides
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

COMMIT;
```

---

## Phase 2: Backend API Changes

### 2.1 API Design Decisions

**Tier Parameter Consistency**: All endpoints use tier as a path parameter for consistency:
- `GET /{scenario}/overrides` - List ALL overrides for scenario (all tiers) - useful for overview pages
- `GET /{scenario}/overrides/{tier}` - List overrides for specific tier
- `GET /{scenario}/overrides/{tier}/{resource}/{secret}` - Get specific override
- `POST /{scenario}/overrides/{tier}/{resource}/{secret}` - Create/update override
- `DELETE /{scenario}/overrides/{tier}/{resource}/{secret}` - Remove override
- `GET /{scenario}/effective/{tier}` - Get effective strategies

**Bulk Operations**: For common workflows like cloning overrides:
- `POST /{scenario}/overrides/copy-from-tier` - Copy all overrides from one tier to another
  - Body: `{ "source_tier": "tier-2-desktop", "target_tier": "tier-3-saas", "overwrite": false }`
- `POST /{scenario}/overrides/copy-from-scenario` - Copy overrides from another scenario
  - Body: `{ "source_scenario": "data-analytics", "tier": "tier-2-desktop", "overwrite": false }`

**Scenario-Resource Validation**: When creating an override, the API validates that:
1. The scenario exists (check for `service.json` in scenarios directory)
2. The scenario depends on the resource (check `service.json` dependencies)
3. The tier is supported by the scenario (check `service.json` deployment tiers, if specified)

This prevents orphan overrides for resources the scenario doesn't use. Returns 400 with clear error message on validation failure.

**Wildcard Tier Support (v2)**: Future enhancement - allow `tier = '*'` to apply an override to all tiers at once. For v1, users must create separate overrides per tier.

### 2.2 New Types

File: `api/types.go` - Add:

```go
// ScenarioSecretOverride represents a scenario-specific strategy override.
type ScenarioSecretOverride struct {
    ID                string          `json:"id"`
    ScenarioName      string          `json:"scenario_name"`
    ResourceSecretID  string          `json:"resource_secret_id"`
    ResourceName      string          `json:"resource_name"`      // Joined from resource_secrets
    SecretKey         string          `json:"secret_key"`         // Joined from resource_secrets
    Tier              string          `json:"tier"`
    HandlingStrategy  *string         `json:"handling_strategy,omitempty"`
    FallbackStrategy  *string         `json:"fallback_strategy,omitempty"`
    RequiresUserInput *bool           `json:"requires_user_input,omitempty"`
    PromptLabel       *string         `json:"prompt_label,omitempty"`
    PromptDescription *string         `json:"prompt_description,omitempty"`
    GeneratorTemplate json.RawMessage `json:"generator_template,omitempty"`
    BundleHints       json.RawMessage `json:"bundle_hints,omitempty"`
    OverrideReason    *string         `json:"override_reason,omitempty"`
    CreatedAt         time.Time       `json:"created_at"`
    UpdatedAt         time.Time       `json:"updated_at"`
}

// EffectiveSecretStrategy represents the merged strategy (resource default + scenario override).
type EffectiveSecretStrategy struct {
    ResourceName      string `json:"resource_name"`
    SecretKey         string `json:"secret_key"`
    Tier              string `json:"tier"`
    HandlingStrategy  string `json:"handling_strategy"`
    FallbackStrategy  string `json:"fallback_strategy,omitempty"`
    RequiresUserInput bool   `json:"requires_user_input"`
    PromptLabel       string `json:"prompt_label,omitempty"`
    PromptDescription string `json:"prompt_description,omitempty"`

    // Origin tracking
    IsOverridden      bool   `json:"is_overridden"`      // True if scenario override exists
    OverriddenFields  []string `json:"overridden_fields,omitempty"` // Which fields are overridden
    OverrideReason    string `json:"override_reason,omitempty"`
}
```

### 2.3 New Handler: Scenario Overrides

File: `api/scenario_override_handlers.go`

```go
package main

// RegisterRoutes mounts scenario override endpoints under the provided router.
// Routes:
//   GET    /scenarios/{scenario}/overrides                            - List ALL overrides (all tiers)
//   GET    /scenarios/{scenario}/overrides/{tier}                     - List overrides for scenario+tier
//   GET    /scenarios/{scenario}/overrides/{tier}/{resource}/{secret} - Get override for specific secret
//   POST   /scenarios/{scenario}/overrides/{tier}/{resource}/{secret} - Create/update override
//   DELETE /scenarios/{scenario}/overrides/{tier}/{resource}/{secret} - Remove override (revert to default)
//   GET    /scenarios/{scenario}/effective/{tier}                     - Get effective strategies for deployment
//   POST   /scenarios/{scenario}/overrides/copy-from-tier             - Copy overrides between tiers
//   POST   /scenarios/{scenario}/overrides/copy-from-scenario         - Copy overrides from another scenario

func (h *ScenarioOverrideHandlers) RegisterRoutes(router *mux.Router) {
    router.HandleFunc("/{scenario}/overrides", h.ListAllOverrides).Methods("GET")
    router.HandleFunc("/{scenario}/overrides/{tier}", h.ListOverrides).Methods("GET")
    router.HandleFunc("/{scenario}/overrides/{tier}/{resource}/{secret}", h.GetOverride).Methods("GET")
    router.HandleFunc("/{scenario}/overrides/{tier}/{resource}/{secret}", h.SetOverride).Methods("POST")
    router.HandleFunc("/{scenario}/overrides/{tier}/{resource}/{secret}", h.DeleteOverride).Methods("DELETE")
    router.HandleFunc("/{scenario}/effective/{tier}", h.GetEffectiveStrategies).Methods("GET")
    router.HandleFunc("/{scenario}/overrides/copy-from-tier", h.CopyFromTier).Methods("POST")
    router.HandleFunc("/{scenario}/overrides/copy-from-scenario", h.CopyFromScenario).Methods("POST")
}
```

### 2.4 Core Query Functions

File: `api/scenario_override_queries.go`

Key functions:
1. `fetchScenarioOverrides(ctx, db, scenario, tier)` - Get all overrides for a scenario
2. `fetchEffectiveStrategy(ctx, db, scenario, resource, secret, tier)` - Merge resource default with scenario override
3. `upsertScenarioOverride(ctx, db, override)` - Create or update an override
4. `deleteScenarioOverride(ctx, db, scenario, resource, secret, tier)` - Remove an override

### 2.5 Modify FetchSecrets for Scenario Context

File: `api/deployment_manifest_fetcher.go`

Update `SecretStore` interface:
```go
type SecretStore interface {
    // FetchSecrets retrieves secrets for the given resources and tier.
    // If scenario is non-empty, applies scenario-specific overrides.
    FetchSecrets(ctx context.Context, scenario, tier string, resources []string, includeOptional bool) ([]DeploymentSecretEntry, error)

    // ... rest of interface
}
```

Update `PostgresSecretStore.FetchSecrets` query to LEFT JOIN with `scenario_secret_strategy_overrides` and use COALESCE to prefer override values:

```sql
SELECT
    rs.id,
    rs.resource_name,
    rs.secret_key,
    -- ... other fields ...

    -- COALESCE precedence (first non-null wins):
    --   1. Scenario override (sso) - highest priority, scenario-specific customization
    --   2. Resource default (sds) - tier-specific default for this resource
    --   3. Fallback value       - empty string/false if neither exists
    --
    -- This means: override > default > fallback
    -- A scenario override ALWAYS wins over the resource default when present.
    COALESCE(sso.handling_strategy, sds.handling_strategy, '') as handling_strategy,
    COALESCE(sso.fallback_strategy, sds.fallback_strategy, '') as fallback_strategy,
    COALESCE(sso.requires_user_input, sds.requires_user_input, false) as requires_user_input,
    COALESCE(sso.prompt_label, sds.prompt_label, '') as prompt_label,
    COALESCE(sso.prompt_description, sds.prompt_description, '') as prompt_description,
    COALESCE(sso.generator_template, sds.generator_template) as generator_template,
    COALESCE(sso.bundle_hints, sds.bundle_hints) as bundle_hints,

    -- Track if overridden for UI (any override row exists for this scenario+secret+tier)
    (sso.id IS NOT NULL) as is_overridden,
    sso.override_reason
FROM resource_secrets rs
LEFT JOIN secret_deployment_strategies sds
    ON sds.resource_secret_id = rs.id AND sds.tier = $1
LEFT JOIN scenario_secret_strategy_overrides sso
    ON sso.resource_secret_id = rs.id
    AND sso.tier = $1
    AND sso.scenario_name = $2
-- ... rest of query
```

### 2.6 Update ManifestBuilder

File: `api/deployment_manifest_builder.go`

Pass scenario to `FetchSecrets`:
```go
// Before:
entries, err := b.secretStore.FetchSecrets(ctx, tier, resolved.Effective, req.IncludeOptional)

// After:
entries, err := b.secretStore.FetchSecrets(ctx, scenario, tier, resolved.Effective, req.IncludeOptional)
```

---

## Phase 3: UI Changes

### 3.1 New API Functions

File: `ui/src/lib/api.ts`

```typescript
export interface ScenarioSecretOverride {
  id: string;
  scenario_name: string;
  resource_name: string;
  secret_key: string;
  tier: string;
  handling_strategy?: string;
  fallback_strategy?: string;
  requires_user_input?: boolean;
  prompt_label?: string;
  prompt_description?: string;
  override_reason?: string;
  is_overridden: boolean;
  overridden_fields?: string[];
}

// Note: tier is passed as path param, not in payload
export interface SetOverridePayload {
  handling_strategy?: string;
  fallback_strategy?: string;
  requires_user_input?: boolean;
  prompt_label?: string;
  prompt_description?: string;
  generator_template?: Record<string, unknown>;  // v2: UI controls for this
  bundle_hints?: Record<string, unknown>;        // v2: UI controls for this
  override_reason?: string;
}

export const fetchScenarioOverrides = (scenario: string, tier: string) =>
  jsonFetch<{ overrides: ScenarioSecretOverride[] }>(
    `/scenarios/${encodeURIComponent(scenario)}/overrides/${encodeURIComponent(tier)}`
  );

export const setScenarioOverride = (
  scenario: string,
  tier: string,
  resource: string,
  secret: string,
  payload: SetOverridePayload
) =>
  jsonFetch<ScenarioSecretOverride>(
    `/scenarios/${encodeURIComponent(scenario)}/overrides/${encodeURIComponent(tier)}/${encodeURIComponent(resource)}/${encodeURIComponent(secret)}`,
    { method: "POST", body: JSON.stringify(payload) }
  );

export const deleteScenarioOverride = (
  scenario: string,
  tier: string,
  resource: string,
  secret: string
) =>
  jsonFetch<{ success: boolean }>(
    `/scenarios/${encodeURIComponent(scenario)}/overrides/${encodeURIComponent(tier)}/${encodeURIComponent(resource)}/${encodeURIComponent(secret)}`,
    { method: "DELETE" }
  );
```

### 3.2 Update useResourcePanel Hook

File: `ui/src/hooks/useResourcePanel.ts`

Add scenario context support:
```typescript
interface UseResourcePanelOptions {
  scenario?: string;  // If provided, enables override mode
}

export function useResourcePanel(options?: UseResourcePanelOptions) {
  const { scenario } = options ?? {};
  // When scenario is provided, start in override mode by default.
  // This makes sense because if you're viewing secrets in a scenario context,
  // you likely want to customize for that scenario.
  const [isOverrideMode, setIsOverrideMode] = useState(!!scenario);

  // ... existing state ...

  // New: Fetch scenario overrides when in override mode
  const overridesQuery = useQuery({
    queryKey: ["scenario-overrides", scenario, activeResource],
    queryFn: () => fetchScenarioOverrides(scenario!, strategyTier),
    enabled: !!scenario && !!activeResource,
  });

  // Modify handleStrategyApply to create override instead of modifying resource default
  const handleStrategyApply = useCallback(async () => {
    if (!activeResource || !selectedSecretKey) return;

    if (scenario && isOverrideMode) {
      // Create scenario override (tier is now a path param)
      await setScenarioOverride(scenario, strategyTier, activeResource, selectedSecretKey, {
        handling_strategy: strategyHandling,
        // ... other fields
      });
    } else {
      // Existing behavior: update resource default
      await updateSecretStrategy(activeResource, selectedSecretKey, { ... });
    }
  }, [scenario, isOverrideMode, /* ... other deps */]);

  return {
    // ... existing returns ...
    isOverrideMode,
    setIsOverrideMode,
    scenarioOverrides: overridesQuery.data?.overrides,
  };
}
```

### 3.3 Update SecretDetail Component

File: `ui/src/features/resource-panel/SecretDetail.tsx`

Add override mode UI:

```tsx
interface SecretDetailProps {
  // ... existing props ...
  scenario?: string;
  isOverrideMode?: boolean;
  existingOverride?: ScenarioSecretOverride;
  onToggleOverrideMode?: (enabled: boolean) => void;
  onRevertToDefault?: () => void;
}

export const SecretDetail = ({
  selectedSecret,
  scenario,
  isOverrideMode,
  existingOverride,
  onToggleOverrideMode,
  onRevertToDefault,
  // ... other props
}: SecretDetailProps) => {
  return (
    <div className="rounded-2xl border border-white/10 bg-black/30 p-4">
      {/* Scenario override banner */}
      {scenario && (
        <div className="mb-4 flex items-center justify-between rounded-xl border border-purple-400/30 bg-purple-400/10 px-3 py-2">
          <div className="flex items-center gap-2">
            <span className="text-xs text-purple-200">
              {isOverrideMode
                ? `Editing override for ${scenario}`
                : `Using resource defaults`}
            </span>
            {existingOverride && (
              <span className="rounded-full bg-purple-500/20 px-2 py-0.5 text-[10px] text-purple-100">
                Overridden
              </span>
            )}
          </div>
          <div className="flex items-center gap-2">
            <Button
              variant="ghost"
              size="sm"
              onClick={() => onToggleOverrideMode?.(!isOverrideMode)}
            >
              {isOverrideMode ? "Edit default" : "Create override"}
            </Button>
            {existingOverride && (
              <Button
                variant="ghost"
                size="sm"
                onClick={onRevertToDefault}
                className="text-red-300 hover:text-red-200"
              >
                Revert
              </Button>
            )}
          </div>
        </div>
      )}

      {/* Strategy form - visual indicator for overridden fields */}
      <div className="space-y-2">
        <label className="text-xs text-white/60">
          <div className="flex items-center gap-1">
            <span>Handling Strategy</span>
            {existingOverride?.overridden_fields?.includes("handling_strategy") && (
              <span className="rounded bg-purple-500/30 px-1 text-[9px] text-purple-200">
                overridden
              </span>
            )}
          </div>
          <select
            value={strategyHandling}
            onChange={(e) => onSetStrategyHandling(e.target.value)}
            className={cn(
              "mt-1 w-full rounded-xl border px-3 py-2 text-sm text-white",
              existingOverride?.overridden_fields?.includes("handling_strategy")
                ? "border-purple-400/40 bg-purple-500/10"
                : "border-white/10 bg-slate-800"
            )}
          >
            {/* options */}
          </select>
        </label>

        {/* Override reason field (only shown in override mode) */}
        {isOverrideMode && (
          <label className="text-xs text-white/60">
            Why does {scenario} need different handling?
            <textarea
              value={overrideReason}
              onChange={(e) => setOverrideReason(e.target.value)}
              className="mt-1 w-full rounded-xl border border-white/10 bg-white/5 px-3 py-2 text-sm text-white"
              placeholder="Optional: Explain why this scenario needs a different strategy"
              rows={2}
            />
          </label>
        )}
      </div>

      {/* ... rest of component */}
    </div>
  );
};
```

### 3.4 Update ResourcePanel to Pass Scenario Context

File: `ui/src/features/resource-panel/ResourcePanel.tsx`

```tsx
interface ResourcePanelProps {
  // ... existing props ...
  scenario?: string;  // New: scenario context for override mode
}
```

### 3.5 Update App.tsx to Pass Scenario

File: `ui/src/App.tsx`

When opening ResourcePanel from deployment tab, pass the selected scenario:

```tsx
<ResourcePanel
  // ... existing props ...
  scenario={activeTab === "deployment" ? selectedScenario : undefined}
/>
```

### 3.6 Add Diff View for Overrides (Optional Enhancement)

When an override exists, show a side-by-side diff of "Resource Default → Override":

```tsx
{existingOverride && (
  <div className="mt-3 rounded-xl border border-white/10 bg-black/20 p-3">
    <div className="mb-2 text-xs font-medium text-white/60">Changes from default</div>
    <div className="grid grid-cols-2 gap-4 text-xs">
      <div>
        <div className="mb-1 text-white/40">Resource Default</div>
        <div className="rounded bg-red-500/10 px-2 py-1 text-red-200">
          {resourceDefault.handling_strategy}
        </div>
      </div>
      <div>
        <div className="mb-1 text-white/40">Scenario Override</div>
        <div className="rounded bg-green-500/10 px-2 py-1 text-green-200">
          {existingOverride.handling_strategy}
        </div>
      </div>
    </div>
  </div>
)}
```

This helps users immediately understand what they're changing and why.

### 3.7 Add HelpDialog for Override Mode

Update `SecretDetail.tsx` help dialogs:

```tsx
<HelpDialog title="Scenario Overrides">
  <p>When editing secrets in the context of a specific scenario, you can create
     <strong className="text-purple-200"> overrides</strong> that only apply to that scenario.</p>

  <div className="mt-3 space-y-2">
    <p><strong className="text-white">How it works:</strong></p>
    <ul className="ml-4 space-y-1 text-white/80">
      <li>• <strong>Resource defaults</strong> apply to all scenarios that use this resource</li>
      <li>• <strong>Scenario overrides</strong> customize handling for just one scenario</li>
      <li>• Overrides take precedence over defaults during deployment</li>
    </ul>
  </div>

  <div className="mt-3 space-y-2">
    <p><strong className="text-white">Example:</strong></p>
    <p className="text-white/70">
      The <code>postgres</code> resource might have <code>DB_PASSWORD</code> set to
      "generate" by default. But your <code>data-analytics</code> scenario needs users
      to provide their own password, so you create an override setting it to "prompt".
    </p>
  </div>
</HelpDialog>
```

---

## Phase 4: Testing

### 4.1 Database Tests

File: `api/scenario_override_queries_test.go`

Test cases:
1. `TestUpsertOverride_CreateNew` - Create a new override
2. `TestUpsertOverride_UpdateExisting` - Update an existing override
3. `TestDeleteOverride_Success` - Delete an override
4. `TestDeleteOverride_NotFound` - Delete non-existent returns gracefully
5. `TestFetchEffectiveStrategy_NoOverride` - Returns resource default
6. `TestFetchEffectiveStrategy_WithOverride` - Returns merged strategy
7. `TestFetchEffectiveStrategy_PartialOverride` - Some fields overridden, others inherited

### 4.2 API Handler Tests

File: `api/scenario_override_handlers_test.go`

Test cases:
1. `TestListOverrides_Empty` - No overrides returns empty list
2. `TestListOverrides_WithData` - Returns correct overrides
3. `TestListAllOverrides_MultiTier` - Returns overrides across all tiers
4. `TestSetOverride_ValidPayload` - Creates override successfully
5. `TestSetOverride_InvalidStrategy` - Rejects invalid handling_strategy
6. `TestSetOverride_NonexistentSecret` - Returns 404 for unknown secret
7. `TestSetOverride_ScenarioDoesNotExist` - Returns 400 if scenario doesn't exist
8. `TestSetOverride_ScenarioDoesNotDependOnResource` - Returns 400 if scenario doesn't use the resource
9. `TestSetOverride_UnsupportedTier` - Returns 400 if tier not in scenario's deployment tiers
10. `TestDeleteOverride_Success` - Deletes and returns success
11. `TestGetEffectiveStrategies_MergesCorrectly` - Verifies merge logic

### 4.3 Bulk Operations Tests

File: `api/scenario_override_bulk_test.go`

Test cases:
1. `TestCopyFromTier_Success` - Copies all overrides from one tier to another
2. `TestCopyFromTier_OverwriteFalse` - Skips existing overrides when overwrite=false
3. `TestCopyFromTier_OverwriteTrue` - Replaces existing overrides when overwrite=true
4. `TestCopyFromTier_SourceEmpty` - Returns success with count=0 when source has no overrides
5. `TestCopyFromScenario_Success` - Copies overrides from another scenario
6. `TestCopyFromScenario_ValidatesDependencies` - Only copies overrides for resources the target depends on

### 4.4 Admin Endpoint Tests

File: `api/admin_handlers_test.go`

Test cases:
1. `TestListOrphans_NoOrphans` - Returns empty list when all overrides valid
2. `TestListOrphans_DeletedScenario` - Detects overrides for non-existent scenarios
3. `TestListOrphans_RemovedDependency` - Detects overrides for resources no longer in dependencies
4. `TestCleanupOrphans_DryRun` - Reports what would be deleted without deleting
5. `TestCleanupOrphans_Execute` - Actually removes orphan overrides

### 4.5 Integration Tests

File: `api/deployment_manifest_test.go` - Add:

1. `TestManifestGeneration_WithScenarioOverrides` - Verify overrides applied in manifest
2. `TestManifestGeneration_OverridePrecedence` - Override wins over default
3. `TestManifestGeneration_PartialOverride` - Non-overridden fields use default

### 4.6 UI Tests

File: `ui/src/features/resource-panel/__tests__/SecretDetail.test.tsx`

Test cases:
1. Override banner shown when scenario prop provided
2. Override mode toggle works
3. Overridden fields visually indicated
4. Revert button calls delete API
5. Form submission creates override in override mode
6. Diff view shows resource default vs override

---

## Phase 5: Documentation

### 5.1 In-App Help

Already covered in Phase 3.6 with HelpDialog updates.

### 5.2 Developer Documentation

File: `docs/secrets/scenario-overrides.md`

```markdown
# Scenario Secret Strategy Overrides

## Overview

Scenario overrides allow individual scenarios to customize how their dependent
resources' secrets are handled during deployment, without affecting other scenarios.

## When to Use Overrides

Use scenario overrides when:
- A scenario has different deployment requirements than the resource default
- You want to test different strategies without affecting production
- Different user personas need different secret handling

## API Reference

### List All Overrides (All Tiers)
GET /api/v1/scenarios/{scenario}/overrides

### List Overrides (Specific Tier)
GET /api/v1/scenarios/{scenario}/overrides/{tier}

### Get Override
GET /api/v1/scenarios/{scenario}/overrides/{tier}/{resource}/{secret}

### Set Override
POST /api/v1/scenarios/{scenario}/overrides/{tier}/{resource}/{secret}
Body: { handling_strategy?, fallback_strategy?, requires_user_input?, prompt_label?, prompt_description?, override_reason? }

### Delete Override
DELETE /api/v1/scenarios/{scenario}/overrides/{tier}/{resource}/{secret}

### Get Effective Strategies
GET /api/v1/scenarios/{scenario}/effective/{tier}

### Bulk Operations

#### Copy Overrides Between Tiers
POST /api/v1/scenarios/{scenario}/overrides/copy-from-tier
Body: { source_tier: string, target_tier: string, overwrite?: boolean }

#### Copy Overrides From Another Scenario
POST /api/v1/scenarios/{scenario}/overrides/copy-from-scenario
Body: { source_scenario: string, tier: string, overwrite?: boolean }

### Admin Endpoints

#### List Orphan Overrides
GET /api/v1/admin/overrides/orphans

#### Cleanup Orphan Overrides
POST /api/v1/admin/overrides/cleanup
Body: { dry_run?: boolean }

## Database Schema

See `initialization/storage/postgres/migrations/002_scenario_overrides.sql`
```

---

## Implementation Order

### Phase 1: Database & Core Backend
- [ ] Add migration script and update schema.sql
- [ ] Add new types to types.go
- [ ] Implement scenario_override_queries.go
- [ ] Implement scenario_override_handlers.go (including bulk operations)
- [ ] Implement admin handlers for orphan cleanup
- [ ] Add scenario/tier validation logic
- [ ] Update PostgresSecretStore.FetchSecrets with override logic
- [ ] Update ManifestBuilder to pass scenario

### Phase 2: Backend Testing
- [ ] Write unit tests for query functions
- [ ] Write API handler tests
- [ ] Write integration tests for manifest generation
- [ ] Manual testing with curl/httpie

### Phase 3: UI Implementation
- [ ] Add API functions to api.ts
- [ ] Update useResourcePanel hook
- [ ] Update SecretDetail component with override UI
- [ ] Update ResourcePanel to pass scenario context
- [ ] Update App.tsx to pass selectedScenario

### Phase 4: UI Polish & Docs
- [ ] Add HelpDialog explanations
- [ ] Visual polish for override indicators
- [ ] Write developer documentation
- [ ] End-to-end testing
- [ ] Update README if needed

---

## Rollback Plan

If issues arise:
1. The migration is additive - no existing data is modified
2. Backend changes are behind scenario parameter - empty scenario = existing behavior
3. UI changes are conditional on scenario prop
4. To fully revert: drop the new table and remove handler registration

---

## Future Enhancements (v2+)

1. **Wildcard tier support**: Allow `tier = '*'` to apply an override to all tiers at once
2. **Generator template UI**: Add visual controls for editing generator_template and bundle_hints JSONB fields
3. **Override templates**: Predefined override sets for common patterns (e.g., "zero-config", "power-user")
4. **Override inheritance**: Scenarios can inherit overrides from parent scenarios
5. **Audit trail**: Track who created/modified overrides and when (requires auth integration)
6. **Override validation**: Warn if an override might cause deployment issues (e.g., "prompt" strategy with no prompt_label)
7. **Batch delete**: Delete all overrides for a scenario/tier in one operation
8. **Export/import**: Export overrides to JSON for backup or migration between environments
