-- Secrets Manager Database Schema
-- Tracks resource secret requirements and validation status

-- Create extension for UUID generation
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Table for tracking resource secret requirements
CREATE TABLE IF NOT EXISTS resource_secrets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    resource_name VARCHAR(100) NOT NULL,
    secret_key VARCHAR(200) NOT NULL,
    secret_type VARCHAR(50) NOT NULL CHECK (secret_type IN ('env_var', 'api_key', 'credential', 'token', 'password', 'certificate')),
    required BOOLEAN NOT NULL DEFAULT true,
    description TEXT,
    validation_pattern VARCHAR(500),
    documentation_url TEXT,
    default_value TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Composite unique constraint to prevent duplicates
    UNIQUE(resource_name, secret_key)
);

-- Table for tracking secret validation results
CREATE TABLE IF NOT EXISTS secret_validations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    resource_secret_id UUID NOT NULL REFERENCES resource_secrets(id) ON DELETE CASCADE,
    validation_status VARCHAR(20) NOT NULL CHECK (validation_status IN ('missing', 'invalid', 'valid', 'expired')),
    validation_method VARCHAR(20) NOT NULL CHECK (validation_method IN ('env', 'vault', 'file', 'api')),
    validation_timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    error_message TEXT,
    validation_details JSONB
);

-- Table for tracking scan operations
CREATE TABLE IF NOT EXISTS secret_scans (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    scan_type VARCHAR(50) NOT NULL DEFAULT 'full',
    resources_scanned TEXT[], -- Array of resource names
    secrets_discovered INTEGER DEFAULT 0,
    scan_duration_ms INTEGER,
    scan_timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    scan_status VARCHAR(20) NOT NULL DEFAULT 'completed' CHECK (scan_status IN ('running', 'completed', 'failed')),
    error_message TEXT,
    scan_metadata JSONB
);

-- Table for tracking secret provisioning operations  
CREATE TABLE IF NOT EXISTS secret_provisions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    resource_secret_id UUID NOT NULL REFERENCES resource_secrets(id) ON DELETE CASCADE,
    storage_method VARCHAR(20) NOT NULL CHECK (storage_method IN ('vault', 'env', 'file')),
    storage_location TEXT NOT NULL,
    provisioned_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    provisioned_by VARCHAR(100),
    provision_status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (provision_status IN ('active', 'expired', 'revoked')),
    expiration_date TIMESTAMP WITH TIME ZONE
);

-- View for current secret status summary
CREATE OR REPLACE VIEW secret_health_summary AS
SELECT 
    rs.resource_name,
    COUNT(rs.id) as total_secrets,
    COUNT(CASE WHEN rs.required = true THEN 1 END) as required_secrets,
    COUNT(CASE WHEN sv.validation_status = 'valid' THEN 1 END) as valid_secrets,
    COUNT(CASE WHEN sv.validation_status = 'missing' AND rs.required = true THEN 1 END) as missing_required_secrets,
    COUNT(CASE WHEN sv.validation_status = 'invalid' THEN 1 END) as invalid_secrets,
    MAX(sv.validation_timestamp) as last_validation
FROM resource_secrets rs
LEFT JOIN (
    SELECT DISTINCT ON (resource_secret_id) 
        resource_secret_id, 
        validation_status,
        validation_timestamp
    FROM secret_validations 
    ORDER BY resource_secret_id, validation_timestamp DESC
) sv ON rs.id = sv.resource_secret_id
GROUP BY rs.resource_name
ORDER BY rs.resource_name;

-- View for missing secrets that need attention
CREATE OR REPLACE VIEW missing_secrets_report AS
SELECT 
    rs.resource_name,
    rs.secret_key,
    rs.secret_type,
    rs.description,
    rs.documentation_url,
    sv.validation_status,
    sv.error_message,
    sv.validation_timestamp
FROM resource_secrets rs
LEFT JOIN (
    SELECT DISTINCT ON (resource_secret_id)
        resource_secret_id,
        validation_status,
        error_message,
        validation_timestamp
    FROM secret_validations
    ORDER BY resource_secret_id, validation_timestamp DESC
) sv ON rs.id = sv.resource_secret_id
WHERE rs.required = true 
AND (sv.validation_status IS NULL OR sv.validation_status IN ('missing', 'invalid'))
ORDER BY rs.resource_name, rs.secret_key;

-- Function to update the updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Trigger to automatically update updated_at on resource_secrets
DROP TRIGGER IF EXISTS update_resource_secrets_updated_at ON resource_secrets;
CREATE TRIGGER update_resource_secrets_updated_at 
BEFORE UPDATE ON resource_secrets 
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Function to clean up old validation records (keep last 100 per secret)
CREATE OR REPLACE FUNCTION cleanup_old_validations()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER := 0;
BEGIN
    WITH ranked_validations AS (
        SELECT id,
               ROW_NUMBER() OVER (PARTITION BY resource_secret_id ORDER BY validation_timestamp DESC) as rn
        FROM secret_validations
    )
    DELETE FROM secret_validations 
    WHERE id IN (
        SELECT id FROM ranked_validations WHERE rn > 100
    );
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Index for performance optimization
CREATE INDEX IF NOT EXISTS idx_resource_secrets_resource_name ON resource_secrets(resource_name);
CREATE INDEX IF NOT EXISTS idx_resource_secrets_required ON resource_secrets(required) WHERE required = true;
CREATE INDEX IF NOT EXISTS idx_secret_validations_status ON secret_validations(validation_status);
CREATE INDEX IF NOT EXISTS idx_secret_validations_timestamp ON secret_validations(validation_timestamp);

-- Comments for documentation
COMMENT ON TABLE resource_secrets IS 'Tracks all secret requirements discovered from resource configurations';
COMMENT ON TABLE secret_validations IS 'Stores validation results for each secret over time';
COMMENT ON TABLE secret_scans IS 'Logs all resource scanning operations';
COMMENT ON TABLE secret_provisions IS 'Tracks secret provisioning and storage operations';
COMMENT ON VIEW secret_health_summary IS 'Summary view of secret validation status by resource';
COMMENT ON VIEW missing_secrets_report IS 'Report of missing required secrets that need attention';

-- Grant permissions (adjust as needed for your setup)
-- GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO secrets_manager_api;
-- GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO secrets_manager_api;
