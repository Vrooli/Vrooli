-- Migration: Add scenario_secret_strategy_overrides table
-- Apply with: psql -d secrets_manager -f migrations/002_scenario_overrides.sql

BEGIN;

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

-- Index for efficient lookup by tier
CREATE INDEX IF NOT EXISTS idx_scenario_overrides_tier
    ON scenario_secret_strategy_overrides(tier);

-- Add trigger for updated_at
DROP TRIGGER IF EXISTS update_scenario_overrides_updated_at ON scenario_secret_strategy_overrides;
CREATE TRIGGER update_scenario_overrides_updated_at
    BEFORE UPDATE ON scenario_secret_strategy_overrides
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE scenario_secret_strategy_overrides IS
    'Stores scenario-specific overrides for secret handling strategies. Null fields inherit from resource defaults.';

COMMIT;
