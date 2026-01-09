-- Home Automation Intelligence Database Schema
-- Self-evolving home automation with multi-user permissions

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- User profiles with device permissions
CREATE TABLE home_profiles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL, -- References scenario-authenticator users
    name VARCHAR(255) NOT NULL,
    permissions JSONB NOT NULL DEFAULT '{}',
    preferences JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    CONSTRAINT unique_user_profile UNIQUE (user_id)
);

-- Create index for user lookups
CREATE INDEX idx_home_profiles_user_id ON home_profiles(user_id);

-- Automation rules created by users or AI
CREATE TABLE automation_rules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    created_by UUID NOT NULL REFERENCES home_profiles(id) ON DELETE CASCADE,
    trigger_type VARCHAR(50) NOT NULL, -- 'schedule', 'device', 'calendar', 'manual'
    trigger_config JSONB NOT NULL DEFAULT '{}',
    conditions JSONB DEFAULT '{}',
    actions JSONB NOT NULL DEFAULT '[]',
    active BOOLEAN DEFAULT true,
    generated_by_ai BOOLEAN DEFAULT false,
    source_code TEXT, -- For AI-generated automations
    execution_count INTEGER DEFAULT 0,
    last_executed TIMESTAMP NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Create indexes for automation queries
CREATE INDEX idx_automation_rules_created_by ON automation_rules(created_by);
CREATE INDEX idx_automation_rules_active ON automation_rules(active);
CREATE INDEX idx_automation_rules_trigger_type ON automation_rules(trigger_type);
CREATE INDEX idx_automation_rules_generated_by_ai ON automation_rules(generated_by_ai);

-- Devices table for device management
CREATE TABLE devices (
    device_id VARCHAR(255) PRIMARY KEY,
    entity_id VARCHAR(255) NOT NULL, -- Home Assistant entity ID
    name VARCHAR(255) NOT NULL,
    device_type VARCHAR(100) NOT NULL,
    manufacturer VARCHAR(255),
    model VARCHAR(255),
    capabilities JSONB DEFAULT '{}',
    room VARCHAR(255),
    available BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Create indexes for device queries
CREATE INDEX idx_devices_type ON devices(device_type);
CREATE INDEX idx_devices_available ON devices(available);
CREATE INDEX idx_devices_room ON devices(room);

-- Device state cache for real-time updates
CREATE TABLE device_states (
    device_id VARCHAR(255) PRIMARY KEY,
    entity_id VARCHAR(255) NOT NULL, -- Home Assistant entity ID
    name VARCHAR(255) NOT NULL,
    device_type VARCHAR(100) NOT NULL,
    state JSONB NOT NULL DEFAULT '{}',
    attributes JSONB DEFAULT '{}',
    available BOOLEAN DEFAULT true,
    last_updated TIMESTAMP DEFAULT NOW(),
    last_seen TIMESTAMP DEFAULT NOW()
);

-- Create indexes for device state queries
CREATE INDEX idx_device_states_type ON device_states(device_type);
CREATE INDEX idx_device_states_available ON device_states(available);
CREATE INDEX idx_device_states_updated ON device_states(last_updated);

-- Smart scenes with AI-suggested optimizations
CREATE TABLE smart_scenes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    created_by UUID NOT NULL REFERENCES home_profiles(id) ON DELETE CASCADE,
    device_states JSONB NOT NULL DEFAULT '{}', -- Device ID -> desired state mapping
    conditions JSONB DEFAULT '{}', -- When this scene should activate
    icon VARCHAR(50) DEFAULT '🏠',
    active BOOLEAN DEFAULT true,
    usage_count INTEGER DEFAULT 0,
    last_used TIMESTAMP NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Create indexes for scene queries
CREATE INDEX idx_smart_scenes_created_by ON smart_scenes(created_by);
CREATE INDEX idx_smart_scenes_active ON smart_scenes(active);

-- Automation execution log for monitoring and debugging
CREATE TABLE automation_executions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    automation_id UUID NOT NULL REFERENCES automation_rules(id) ON DELETE CASCADE,
    trigger_source VARCHAR(100) NOT NULL,
    trigger_data JSONB DEFAULT '{}',
    execution_status VARCHAR(50) NOT NULL DEFAULT 'pending', -- 'pending', 'running', 'completed', 'failed'
    devices_affected TEXT[], -- Array of device IDs
    execution_time_ms INTEGER,
    error_message TEXT,
    executed_at TIMESTAMP DEFAULT NOW()
);

-- Create indexes for execution log queries  
CREATE INDEX idx_automation_executions_automation_id ON automation_executions(automation_id);
CREATE INDEX idx_automation_executions_status ON automation_executions(execution_status);
CREATE INDEX idx_automation_executions_executed_at ON automation_executions(executed_at);

-- Permission audit log for security
CREATE TABLE permission_audit_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,
    profile_id UUID REFERENCES home_profiles(id) ON DELETE SET NULL,
    action_type VARCHAR(100) NOT NULL, -- 'device_control', 'automation_create', 'scene_activate'
    resource_type VARCHAR(50) NOT NULL, -- 'device', 'automation', 'scene'
    resource_id VARCHAR(255) NOT NULL,
    permission_granted BOOLEAN NOT NULL,
    ip_address INET,
    user_agent TEXT,
    additional_data JSONB DEFAULT '{}',
    timestamp TIMESTAMP DEFAULT NOW()
);

-- Create indexes for audit queries
CREATE INDEX idx_permission_audit_user_id ON permission_audit_log(user_id);
CREATE INDEX idx_permission_audit_timestamp ON permission_audit_log(timestamp);
CREATE INDEX idx_permission_audit_resource ON permission_audit_log(resource_type, resource_id);

-- Energy usage tracking for optimization
CREATE TABLE energy_usage (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    device_id VARCHAR(255) NOT NULL,
    measurement_type VARCHAR(50) NOT NULL, -- 'power', 'energy', 'cost'
    value DECIMAL(10,4) NOT NULL,
    unit VARCHAR(20) NOT NULL, -- 'W', 'kWh', 'USD'
    timestamp TIMESTAMP DEFAULT NOW()
);

-- Create indexes for energy queries
CREATE INDEX idx_energy_usage_device_timestamp ON energy_usage(device_id, timestamp);
CREATE INDEX idx_energy_usage_type_timestamp ON energy_usage(measurement_type, timestamp);

-- Calendar integration events for context-aware automation
CREATE TABLE calendar_contexts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    calendar_event_id VARCHAR(255) NOT NULL,
    context_name VARCHAR(255) NOT NULL, -- 'work', 'sleep', 'entertainment', 'away'
    scene_id UUID REFERENCES smart_scenes(id) ON DELETE SET NULL,
    automation_overrides JSONB DEFAULT '{}',
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP NOT NULL,
    all_day BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Create indexes for calendar context queries
CREATE INDEX idx_calendar_contexts_event_id ON calendar_contexts(calendar_event_id);
CREATE INDEX idx_calendar_contexts_time_range ON calendar_contexts(start_time, end_time);
CREATE INDEX idx_calendar_contexts_context_name ON calendar_contexts(context_name);

-- System configuration and feature flags
CREATE TABLE system_config (
    key VARCHAR(255) PRIMARY KEY,
    value JSONB NOT NULL,
    description TEXT,
    updated_by UUID,
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Insert default configuration
INSERT INTO system_config (key, value, description) VALUES
    ('ai_automation_enabled', 'true', 'Enable AI-powered automation generation'),
    ('max_concurrent_automations', '10', 'Maximum number of automations that can run simultaneously'),
    ('device_command_timeout_ms', '5000', 'Timeout for device control commands'),
    ('energy_monitoring_enabled', 'true', 'Enable energy usage tracking'),
    ('calendar_integration_enabled', 'true', 'Enable calendar-based context switching'),
    ('permission_audit_enabled', 'true', 'Enable permission and access auditing'),
    ('home_assistant_connection', '{"base_url":"http://localhost:8123","token":"","mock_mode":true,"last_status":"unconfigured","last_message":"Home Assistant base URL not configured"}', 'Home Assistant integration settings');

-- Functions for common operations

-- Function to update device state with timestamp
CREATE OR REPLACE FUNCTION update_device_state(
    p_device_id VARCHAR(255),
    p_entity_id VARCHAR(255),
    p_name VARCHAR(255),
    p_device_type VARCHAR(100),
    p_state JSONB,
    p_attributes JSONB DEFAULT NULL,
    p_available BOOLEAN DEFAULT true
) RETURNS VOID AS $$
BEGIN
    INSERT INTO device_states (
        device_id, entity_id, name, device_type, state, attributes, available, last_updated, last_seen
    ) VALUES (
        p_device_id, p_entity_id, p_name, p_device_type, p_state, p_attributes, p_available, NOW(), NOW()
    )
    ON CONFLICT (device_id) DO UPDATE SET
        entity_id = EXCLUDED.entity_id,
        name = EXCLUDED.name,
        device_type = EXCLUDED.device_type,
        state = EXCLUDED.state,
        attributes = COALESCE(EXCLUDED.attributes, device_states.attributes),
        available = EXCLUDED.available,
        last_updated = NOW(),
        last_seen = CASE 
            WHEN EXCLUDED.available = true THEN NOW()
            ELSE device_states.last_seen
        END;
END;
$$ LANGUAGE plpgsql;

-- Function to log automation execution
CREATE OR REPLACE FUNCTION log_automation_execution(
    p_automation_id UUID,
    p_trigger_source VARCHAR(100),
    p_trigger_data JSONB DEFAULT '{}',
    p_devices_affected TEXT[] DEFAULT '{}',
    p_execution_time_ms INTEGER DEFAULT NULL,
    p_status VARCHAR(50) DEFAULT 'completed',
    p_error_message TEXT DEFAULT NULL
) RETURNS UUID AS $$
DECLARE
    execution_id UUID;
BEGIN
    INSERT INTO automation_executions (
        automation_id, trigger_source, trigger_data, execution_status,
        devices_affected, execution_time_ms, error_message
    ) VALUES (
        p_automation_id, p_trigger_source, p_trigger_data, p_status,
        p_devices_affected, p_execution_time_ms, p_error_message
    ) RETURNING id INTO execution_id;
    
    -- Update automation rule execution count
    UPDATE automation_rules 
    SET execution_count = execution_count + 1,
        last_executed = NOW()
    WHERE id = p_automation_id;
    
    RETURN execution_id;
END;
$$ LANGUAGE plpgsql;

-- Function to check user device permissions
CREATE OR REPLACE FUNCTION check_device_permission(
    p_user_id UUID,
    p_device_id VARCHAR(255),
    p_action VARCHAR(100) DEFAULT 'control'
) RETURNS BOOLEAN AS $$
DECLARE
    user_permissions JSONB;
    allowed_devices TEXT[];
BEGIN
    -- Get user's device permissions
    SELECT permissions->'devices' INTO user_permissions
    FROM home_profiles 
    WHERE user_id = p_user_id;
    
    -- If no permissions found, deny access
    IF user_permissions IS NULL THEN
        RETURN false;
    END IF;
    
    -- Check for wildcard permission
    IF user_permissions ? '*' THEN
        RETURN true;
    END IF;
    
    -- Check for specific device permission
    IF user_permissions ? p_device_id THEN
        RETURN true;
    END IF;
    
    -- Default deny
    RETURN false;
END;
$$ LANGUAGE plpgsql;

-- Triggers for automatic timestamp updates
CREATE OR REPLACE FUNCTION update_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create triggers for tables with updated_at columns
CREATE TRIGGER trigger_home_profiles_updated_at
    BEFORE UPDATE ON home_profiles
    FOR EACH ROW EXECUTE FUNCTION update_timestamp();

CREATE TRIGGER trigger_automation_rules_updated_at
    BEFORE UPDATE ON automation_rules
    FOR EACH ROW EXECUTE FUNCTION update_timestamp();

CREATE TRIGGER trigger_smart_scenes_updated_at
    BEFORE UPDATE ON smart_scenes
    FOR EACH ROW EXECUTE FUNCTION update_timestamp();

CREATE TRIGGER trigger_system_config_updated_at
    BEFORE UPDATE ON system_config
    FOR EACH ROW EXECUTE FUNCTION update_timestamp();

-- Comments for documentation
COMMENT ON TABLE home_profiles IS 'User profiles with device access permissions and preferences';
COMMENT ON TABLE automation_rules IS 'User-created and AI-generated automation rules';
COMMENT ON TABLE device_states IS 'Real-time cache of Home Assistant device states';
COMMENT ON TABLE smart_scenes IS 'Predefined device state combinations for different contexts';
COMMENT ON TABLE automation_executions IS 'Log of automation rule executions for monitoring';
COMMENT ON TABLE permission_audit_log IS 'Security audit trail for permission checks';
COMMENT ON TABLE energy_usage IS 'Energy consumption tracking for optimization';
COMMENT ON TABLE calendar_contexts IS 'Calendar event integration for context-aware automation';
COMMENT ON TABLE system_config IS 'System configuration and feature flags';

-- Grant permissions for application user
-- Note: In production, create a dedicated application user with limited permissions
-- GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO home_automation_app;
-- GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO home_automation_app;
