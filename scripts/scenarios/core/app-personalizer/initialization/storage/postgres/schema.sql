-- App Personalizer Database Schema

CREATE DATABASE IF NOT EXISTS app_personalizer;
\c app_personalizer;

-- Registry of apps available for personalization
CREATE TABLE app_registry (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    app_name VARCHAR(255) NOT NULL,
    app_path TEXT NOT NULL,
    app_type VARCHAR(50), -- 'generated', 'external'
    framework VARCHAR(50), -- 'react', 'vue', 'next.js', etc.
    version VARCHAR(50),
    service_json JSONB,
    personalization_points JSONB DEFAULT '[]',
    supported_personalizations JSONB DEFAULT '[]',
    last_analyzed TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Personalization records
CREATE TABLE personalizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id UUID REFERENCES app_registry(id),
    persona_id UUID, -- from personal-digital-twin
    brand_id UUID, -- from brand-manager
    personalization_name VARCHAR(255),
    description TEXT,
    deployment_mode VARCHAR(50), -- 'copy', 'patch', 'multi_tenant'
    modifications JSONB NOT NULL,
    original_app_path TEXT,
    personalized_app_path TEXT,
    backup_path TEXT,
    status VARCHAR(50) DEFAULT 'pending',
    validation_results JSONB,
    created_by VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    applied_at TIMESTAMP,
    rollback_at TIMESTAMP
);

-- Modification templates for common personalizations
CREATE TABLE modification_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    category VARCHAR(100), -- 'ui_theme', 'content', 'functionality', 'behavior'
    framework VARCHAR(50),
    description TEXT,
    modification_pattern JSONB NOT NULL,
    required_inputs JSONB DEFAULT '[]',
    example_usage JSONB,
    success_rate FLOAT,
    usage_count INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Track modification history
CREATE TABLE modification_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    personalization_id UUID REFERENCES personalizations(id),
    modification_type VARCHAR(50),
    file_path TEXT,
    original_content TEXT,
    modified_content TEXT,
    diff_patch TEXT,
    applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    applied_by VARCHAR(50) -- 'claude-code', 'manual', 'automated'
);

-- App backups before personalization
CREATE TABLE app_backups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id UUID REFERENCES app_registry(id),
    personalization_id UUID REFERENCES personalizations(id),
    backup_type VARCHAR(50), -- 'full', 'differential', 'config-only'
    backup_path TEXT NOT NULL,
    backup_size_bytes BIGINT,
    checksum VARCHAR(64),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP
);

-- Integration connections
CREATE TABLE integrations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    service_name VARCHAR(50) NOT NULL, -- 'personal-digital-twin', 'brand-manager'
    endpoint_url TEXT NOT NULL,
    auth_type VARCHAR(50), -- 'api_token', 'api_key', 'oauth'
    auth_credentials JSONB, -- encrypted
    connection_status VARCHAR(50) DEFAULT 'active',
    last_verified TIMESTAMP,
    rate_limit_config JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Validation rules for personalizations
CREATE TABLE validation_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_name VARCHAR(255) NOT NULL,
    rule_type VARCHAR(50), -- 'build', 'lint', 'test', 'startup'
    framework VARCHAR(50),
    validation_command TEXT,
    expected_output_pattern TEXT,
    timeout_seconds INTEGER DEFAULT 300,
    is_critical BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Multi-tenant configurations
CREATE TABLE multi_tenant_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id UUID REFERENCES app_registry(id),
    tenant_id VARCHAR(255) NOT NULL, -- could be persona_id
    config_type VARCHAR(50), -- 'theme', 'content', 'features'
    configuration JSONB NOT NULL,
    is_active BOOLEAN DEFAULT true,
    priority INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(app_id, tenant_id, config_type)
);

-- Deployment tracking
CREATE TABLE deployments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    personalization_id UUID REFERENCES personalizations(id),
    deployment_type VARCHAR(50), -- 'local', 'docker', 'kubernetes'
    deployment_target TEXT,
    deployment_config JSONB,
    status VARCHAR(50) DEFAULT 'pending',
    health_check_url TEXT,
    last_health_check TIMESTAMP,
    health_status VARCHAR(50),
    deployed_at TIMESTAMP,
    terminated_at TIMESTAMP
);

-- Indexes for performance
CREATE INDEX idx_app_registry_name ON app_registry(app_name);
CREATE INDEX idx_app_registry_type ON app_registry(app_type);
CREATE INDEX idx_personalizations_app ON personalizations(app_id);
CREATE INDEX idx_personalizations_persona ON personalizations(persona_id);
CREATE INDEX idx_personalizations_brand ON personalizations(brand_id);
CREATE INDEX idx_personalizations_status ON personalizations(status);
CREATE INDEX idx_modification_history_personalization ON modification_history(personalization_id);
CREATE INDEX idx_multi_tenant_app_tenant ON multi_tenant_configs(app_id, tenant_id);
CREATE INDEX idx_deployments_personalization ON deployments(personalization_id);
CREATE INDEX idx_deployments_status ON deployments(status);

-- Update triggers
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER app_registry_updated_at BEFORE UPDATE ON app_registry
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER multi_tenant_configs_updated_at BEFORE UPDATE ON multi_tenant_configs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();