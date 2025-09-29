-- Missing tables for home-automation scenario
-- These tables are referenced by the API but not in the main schema

-- Safety rules table for automation validation
CREATE TABLE IF NOT EXISTS safety_rules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    rule_type VARCHAR(50) NOT NULL, -- 'device', 'automation', 'scene', 'global'
    conditions JSONB NOT NULL DEFAULT '{}',
    actions JSONB DEFAULT '{}',
    priority INTEGER DEFAULT 100,
    is_enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Create indexes for safety rules
CREATE INDEX IF NOT EXISTS idx_safety_rules_enabled ON safety_rules(is_enabled);
CREATE INDEX IF NOT EXISTS idx_safety_rules_type ON safety_rules(rule_type);
CREATE INDEX IF NOT EXISTS idx_safety_rules_priority ON safety_rules(priority);

-- Active contexts table for calendar scheduler
CREATE TABLE IF NOT EXISTS active_contexts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    context_name VARCHAR(255) NOT NULL,
    profile_id UUID REFERENCES home_profiles(id) ON DELETE CASCADE,
    scene_id UUID REFERENCES smart_scenes(id) ON DELETE SET NULL,
    automation_overrides JSONB DEFAULT '{}',
    activated_at TIMESTAMP DEFAULT NOW(),
    expires_at TIMESTAMP,
    is_active BOOLEAN DEFAULT true,
    priority INTEGER DEFAULT 100
);

-- Create indexes for active contexts
CREATE INDEX IF NOT EXISTS idx_active_contexts_name ON active_contexts(context_name);
CREATE INDEX IF NOT EXISTS idx_active_contexts_active ON active_contexts(is_active);
CREATE INDEX IF NOT EXISTS idx_active_contexts_profile ON active_contexts(profile_id);

-- Insert default safety rules
INSERT INTO safety_rules (name, description, rule_type, conditions, is_enabled) VALUES
    ('Prevent simultaneous door locks', 'Prevent all doors from being unlocked simultaneously', 'device', '{"type": "lock", "max_simultaneous": 2}', true),
    ('Energy usage limit', 'Limit total energy usage during peak hours', 'global', '{"max_watts": 5000, "peak_hours": [16, 21]}', true),
    ('Temperature safety', 'Prevent extreme temperature settings', 'device', '{"type": "thermostat", "min_temp": 60, "max_temp": 85}', true)
ON CONFLICT DO NOTHING;

-- Create schema for home_automation if it doesn't exist
CREATE SCHEMA IF NOT EXISTS home_automation;

-- Create active_contexts in home_automation schema as well
CREATE TABLE IF NOT EXISTS home_automation.active_contexts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    context_name VARCHAR(255) NOT NULL,
    profile_id UUID,
    scene_id UUID,
    automation_overrides JSONB DEFAULT '{}',
    activated_at TIMESTAMP DEFAULT NOW(),
    expires_at TIMESTAMP,
    is_active BOOLEAN DEFAULT true,
    priority INTEGER DEFAULT 100
);