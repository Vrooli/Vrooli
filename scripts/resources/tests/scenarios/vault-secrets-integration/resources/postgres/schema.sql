-- Vault Secrets Integration Audit Schema
-- This schema tracks all secret access operations for compliance and security

-- Create schema if not exists
CREATE SCHEMA IF NOT EXISTS vault_audit;

-- Secret access audit table
CREATE TABLE IF NOT EXISTS vault_audit.secret_access_log (
    id SERIAL PRIMARY KEY,
    access_id UUID DEFAULT gen_random_uuid(),
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Request details
    user_id VARCHAR(255),
    service_name VARCHAR(255) NOT NULL DEFAULT 'n8n',
    workflow_id VARCHAR(255),
    webhook_path VARCHAR(255),
    
    -- Vault interaction details
    vault_path VARCHAR(500) NOT NULL,
    vault_operation VARCHAR(50) NOT NULL DEFAULT 'READ',
    vault_version INTEGER,
    
    -- Response details
    success BOOLEAN NOT NULL DEFAULT false,
    response_code INTEGER,
    error_message TEXT,
    
    -- Security metadata
    source_ip VARCHAR(45),
    user_agent TEXT,
    request_headers JSONB,
    
    -- Performance metrics
    duration_ms INTEGER,
    vault_response_time_ms INTEGER,
    
    -- Compliance fields
    data_classification VARCHAR(50),
    compliance_tags TEXT[],
    retention_days INTEGER DEFAULT 2555, -- 7 years default
    
    -- Indexes for performance
    CONSTRAINT unique_access_id UNIQUE (access_id)
);

-- Create indexes for common queries
CREATE INDEX idx_secret_access_timestamp ON vault_audit.secret_access_log(timestamp DESC);
CREATE INDEX idx_secret_access_user ON vault_audit.secret_access_log(user_id);
CREATE INDEX idx_secret_access_service ON vault_audit.secret_access_log(service_name);
CREATE INDEX idx_secret_access_vault_path ON vault_audit.secret_access_log(vault_path);
CREATE INDEX idx_secret_access_success ON vault_audit.secret_access_log(success);
CREATE INDEX idx_secret_access_workflow ON vault_audit.secret_access_log(workflow_id);

-- Secret rotation tracking
CREATE TABLE IF NOT EXISTS vault_audit.secret_rotation_log (
    id SERIAL PRIMARY KEY,
    rotation_id UUID DEFAULT gen_random_uuid(),
    rotated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Secret identification
    secret_path VARCHAR(500) NOT NULL,
    secret_type VARCHAR(100) NOT NULL, -- 'api_key', 'database_password', etc.
    
    -- Rotation details
    old_version INTEGER,
    new_version INTEGER,
    rotation_reason VARCHAR(255),
    initiated_by VARCHAR(255),
    
    -- Status tracking
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, completed, failed
    completed_at TIMESTAMP WITH TIME ZONE,
    error_details TEXT,
    
    -- Affected services
    affected_services TEXT[],
    notifications_sent JSONB,
    
    CONSTRAINT unique_rotation_id UNIQUE (rotation_id)
);

CREATE INDEX idx_rotation_timestamp ON vault_audit.secret_rotation_log(rotated_at DESC);
CREATE INDEX idx_rotation_path ON vault_audit.secret_rotation_log(secret_path);
CREATE INDEX idx_rotation_status ON vault_audit.secret_rotation_log(status);

-- Compliance summary view
CREATE OR REPLACE VIEW vault_audit.compliance_summary AS
SELECT 
    DATE_TRUNC('day', timestamp) as access_date,
    service_name,
    COUNT(*) as total_accesses,
    COUNT(DISTINCT user_id) as unique_users,
    COUNT(DISTINCT vault_path) as unique_secrets,
    SUM(CASE WHEN success THEN 1 ELSE 0 END) as successful_accesses,
    SUM(CASE WHEN NOT success THEN 1 ELSE 0 END) as failed_accesses,
    AVG(duration_ms) as avg_duration_ms,
    PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY duration_ms) as p95_duration_ms
FROM vault_audit.secret_access_log
GROUP BY DATE_TRUNC('day', timestamp), service_name;

-- Function to clean up old audit logs based on retention policy
CREATE OR REPLACE FUNCTION vault_audit.cleanup_old_logs() RETURNS void AS $$
BEGIN
    DELETE FROM vault_audit.secret_access_log 
    WHERE timestamp < NOW() - INTERVAL '1 day' * retention_days;
    
    -- Log the cleanup operation
    INSERT INTO vault_audit.secret_access_log (
        user_id, 
        service_name, 
        vault_path, 
        vault_operation,
        success,
        response_code
    ) VALUES (
        'system',
        'audit_cleanup',
        'internal/cleanup',
        'DELETE',
        true,
        200
    );
END;
$$ LANGUAGE plpgsql;

-- Create a stored procedure for recording access
CREATE OR REPLACE FUNCTION vault_audit.record_secret_access(
    p_user_id VARCHAR(255),
    p_workflow_id VARCHAR(255),
    p_vault_path VARCHAR(500),
    p_success BOOLEAN,
    p_duration_ms INTEGER DEFAULT NULL,
    p_error_message TEXT DEFAULT NULL
) RETURNS UUID AS $$
DECLARE
    v_access_id UUID;
BEGIN
    INSERT INTO vault_audit.secret_access_log (
        user_id,
        workflow_id,
        vault_path,
        success,
        duration_ms,
        error_message,
        response_code
    ) VALUES (
        p_user_id,
        p_workflow_id,
        p_vault_path,
        p_success,
        p_duration_ms,
        p_error_message,
        CASE WHEN p_success THEN 200 ELSE 500 END
    ) RETURNING access_id INTO v_access_id;
    
    RETURN v_access_id;
END;
$$ LANGUAGE plpgsql;

-- Grant permissions (adjust based on your user setup)
GRANT USAGE ON SCHEMA vault_audit TO PUBLIC;
GRANT SELECT ON ALL TABLES IN SCHEMA vault_audit TO PUBLIC;
GRANT INSERT ON vault_audit.secret_access_log TO PUBLIC;
GRANT INSERT ON vault_audit.secret_rotation_log TO PUBLIC;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA vault_audit TO PUBLIC;