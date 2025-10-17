-- Vrooli Orchestrator Database Schema
-- Manages startup profiles and activation history

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Profiles table - stores startup profile configurations
CREATE TABLE IF NOT EXISTS profiles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) UNIQUE NOT NULL, -- kebab-case identifier
    display_name VARCHAR(200) NOT NULL, -- human-readable name
    description TEXT,
    
    -- Profile metadata
    metadata JSONB NOT NULL DEFAULT '{}',
    
    -- Configuration arrays
    resources TEXT[] DEFAULT '{}', -- resources to start
    scenarios TEXT[] DEFAULT '{}', -- scenarios to run
    auto_browser TEXT[] DEFAULT '{}', -- URLs to open in browser
    
    -- Environment and behavior
    environment_vars JSONB DEFAULT '{}', -- env var overrides
    idle_shutdown_minutes INTEGER DEFAULT NULL, -- auto-shutdown timer
    dependencies TEXT[] DEFAULT '{}', -- other profiles this depends on
    
    -- Status and timestamps
    status VARCHAR(20) DEFAULT 'inactive' CHECK (status IN ('active', 'inactive', 'error')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_by VARCHAR(100) DEFAULT 'system'
);

-- Profile activations table - tracks activation history and analytics
CREATE TABLE IF NOT EXISTS profile_activations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    profile_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    
    -- Activation lifecycle
    activated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deactivated_at TIMESTAMP WITH TIME ZONE DEFAULT NULL,
    
    -- Context and results
    user_context TEXT, -- who/what triggered activation
    activation_method VARCHAR(50), -- cli, api, auto, etc.
    success BOOLEAN DEFAULT FALSE,
    error_details TEXT,
    
    -- What actually got started
    started_resources TEXT[] DEFAULT '{}',
    started_scenarios TEXT[] DEFAULT '{}',
    opened_urls TEXT[] DEFAULT '{}',
    failed_resources TEXT[] DEFAULT '{}',
    failed_scenarios TEXT[] DEFAULT '{}',
    
    -- Performance metrics
    total_activation_time_ms INTEGER,
    resource_start_time_ms INTEGER,
    scenario_start_time_ms INTEGER
);

-- Active profile tracking - only one profile can be active at a time
CREATE TABLE IF NOT EXISTS active_profile (
    id INTEGER PRIMARY KEY DEFAULT 1 CHECK (id = 1), -- singleton table
    profile_id UUID REFERENCES profiles(id) ON DELETE SET NULL,
    activation_id UUID REFERENCES profile_activations(id) ON DELETE SET NULL,
    activated_at TIMESTAMP WITH TIME ZONE,
    
    -- Constraint ensures only one row exists
    CONSTRAINT single_active_profile CHECK (id = 1)
);

-- Insert the singleton row
INSERT INTO active_profile (id, profile_id, activation_id, activated_at) 
VALUES (1, NULL, NULL, NULL)
ON CONFLICT (id) DO NOTHING;

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_profiles_name ON profiles(name);
CREATE INDEX IF NOT EXISTS idx_profiles_status ON profiles(status);
CREATE INDEX IF NOT EXISTS idx_profiles_created_at ON profiles(created_at);
CREATE INDEX IF NOT EXISTS idx_activations_profile_id ON profile_activations(profile_id);
CREATE INDEX IF NOT EXISTS idx_activations_activated_at ON profile_activations(activated_at);
CREATE INDEX IF NOT EXISTS idx_activations_success ON profile_activations(success);

-- Trigger to update the updated_at timestamp on profiles
CREATE OR REPLACE FUNCTION update_profile_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER profiles_updated_at
    BEFORE UPDATE ON profiles
    FOR EACH ROW
    EXECUTE FUNCTION update_profile_updated_at();

-- Function to get current active profile
CREATE OR REPLACE FUNCTION get_active_profile()
RETURNS TABLE (
    profile_id UUID,
    profile_name TEXT,
    activation_id UUID,
    activated_at TIMESTAMP WITH TIME ZONE
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        ap.profile_id,
        p.name::TEXT,
        ap.activation_id,
        ap.activated_at
    FROM active_profile ap
    LEFT JOIN profiles p ON ap.profile_id = p.id
    WHERE ap.id = 1;
END;
$$ LANGUAGE plpgsql;

-- Function to set active profile
CREATE OR REPLACE FUNCTION set_active_profile(
    p_profile_id UUID,
    p_activation_id UUID
)
RETURNS VOID AS $$
BEGIN
    UPDATE active_profile 
    SET 
        profile_id = p_profile_id,
        activation_id = p_activation_id,
        activated_at = NOW()
    WHERE id = 1;
END;
$$ LANGUAGE plpgsql;

-- Function to clear active profile
CREATE OR REPLACE FUNCTION clear_active_profile()
RETURNS VOID AS $$
BEGIN
    UPDATE active_profile 
    SET 
        profile_id = NULL,
        activation_id = NULL,
        activated_at = NULL
    WHERE id = 1;
END;
$$ LANGUAGE plpgsql;

-- View for profile analytics
CREATE OR REPLACE VIEW profile_analytics AS
SELECT 
    p.id,
    p.name,
    p.display_name,
    p.status,
    COUNT(pa.id) as total_activations,
    COUNT(CASE WHEN pa.success = true THEN 1 END) as successful_activations,
    COUNT(CASE WHEN pa.success = false THEN 1 END) as failed_activations,
    AVG(pa.total_activation_time_ms) as avg_activation_time_ms,
    MAX(pa.activated_at) as last_activated_at,
    p.created_at,
    p.updated_at
FROM profiles p
LEFT JOIN profile_activations pa ON p.id = pa.profile_id
GROUP BY p.id, p.name, p.display_name, p.status, p.created_at, p.updated_at;