-- Applications/Scenarios Management Schema
-- Tracks scenarios that integrate with the authentication service

-- Create applications table to track integrated scenarios
CREATE TABLE IF NOT EXISTS applications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) UNIQUE NOT NULL,
    display_name VARCHAR(255) NOT NULL,
    description TEXT,
    scenario_type VARCHAR(50), -- 'api', 'ui', 'workflow', 'hybrid'
    
    -- Authentication configuration
    api_key_hash VARCHAR(255) UNIQUE NOT NULL,
    api_secret_hash VARCHAR(255) NOT NULL,
    allowed_origins JSONB DEFAULT '[]'::jsonb, -- CORS allowed origins
    redirect_uris JSONB DEFAULT '[]'::jsonb, -- OAuth redirect URIs
    
    -- Permissions and limits
    permissions JSONB DEFAULT '[]'::jsonb,
    rate_limit INTEGER DEFAULT 1000, -- requests per hour
    max_users INTEGER, -- user limit for this app
    
    -- Integration details
    integration_type VARCHAR(50) DEFAULT 'jwt', -- 'jwt', 'api_key', 'oauth2'
    webhook_url VARCHAR(500), -- for auth events
    metadata JSONB DEFAULT '{}'::jsonb,
    
    -- Status tracking
    is_active BOOLEAN DEFAULT TRUE,
    last_accessed TIMESTAMP,
    total_users INTEGER DEFAULT 0,
    total_authentications INTEGER DEFAULT 0,
    
    -- Timestamps
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by UUID REFERENCES users(id)
);

-- Create indexes for applications
CREATE INDEX idx_applications_name ON applications(name) WHERE is_active = TRUE;
CREATE INDEX idx_applications_api_key ON applications(api_key_hash) WHERE is_active = TRUE;
CREATE INDEX idx_applications_created_at ON applications(created_at);

-- Create application_users junction table
-- Tracks which users have accounts in which applications
CREATE TABLE IF NOT EXISTS application_users (
    application_id UUID REFERENCES applications(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    app_specific_data JSONB DEFAULT '{}'::jsonb, -- app-specific user data
    first_login TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_login TIMESTAMP,
    login_count INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT TRUE,
    PRIMARY KEY (application_id, user_id)
);

-- Create indexes for application_users
CREATE INDEX idx_application_users_app ON application_users(application_id) WHERE is_active = TRUE;
CREATE INDEX idx_application_users_user ON application_users(user_id) WHERE is_active = TRUE;

-- Create application_sessions table
-- Tracks active sessions per application
CREATE TABLE IF NOT EXISTS application_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    application_id UUID REFERENCES applications(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    session_token_hash VARCHAR(255) UNIQUE NOT NULL,
    ip_address INET,
    user_agent TEXT,
    metadata JSONB DEFAULT '{}'::jsonb,
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_activity TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL,
    ended_at TIMESTAMP
);

-- Create indexes for application_sessions
CREATE INDEX idx_app_sessions_app ON application_sessions(application_id) WHERE ended_at IS NULL;
CREATE INDEX idx_app_sessions_user ON application_sessions(user_id) WHERE ended_at IS NULL;
CREATE INDEX idx_app_sessions_token ON application_sessions(session_token_hash) WHERE ended_at IS NULL;
CREATE INDEX idx_app_sessions_expires ON application_sessions(expires_at) WHERE ended_at IS NULL;

-- Update audit_logs to track application context
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS application_id UUID REFERENCES applications(id);
CREATE INDEX idx_audit_logs_application ON audit_logs(application_id);

-- Update API keys to support application-level keys
ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS application_id UUID REFERENCES applications(id);
CREATE INDEX idx_api_keys_application ON api_keys(application_id) WHERE revoked_at IS NULL;

-- Create view for application statistics
CREATE OR REPLACE VIEW application_stats AS
    SELECT 
        a.id,
        a.name,
        a.display_name,
        a.is_active,
        COUNT(DISTINCT au.user_id) as total_users,
        COUNT(DISTINCT CASE WHEN as2.ended_at IS NULL THEN as2.id END) as active_sessions,
        COUNT(DISTINCT al.id) as total_events_today,
        a.rate_limit,
        a.last_accessed,
        a.created_at
    FROM applications a
    LEFT JOIN application_users au ON a.id = au.application_id
    LEFT JOIN application_sessions as2 ON a.id = as2.application_id
    LEFT JOIN audit_logs al ON a.id = al.application_id 
        AND al.created_at >= CURRENT_DATE
    GROUP BY a.id;

-- Create function to register a new application
CREATE OR REPLACE FUNCTION register_application(
    p_name VARCHAR(100),
    p_display_name VARCHAR(255),
    p_scenario_type VARCHAR(50),
    p_created_by UUID
) RETURNS TABLE(
    application_id UUID,
    api_key VARCHAR(255),
    api_secret VARCHAR(255)
) AS $$
DECLARE
    v_app_id UUID;
    v_api_key VARCHAR(255);
    v_api_secret VARCHAR(255);
BEGIN
    -- Generate API credentials
    v_api_key := 'ak_' || encode(gen_random_bytes(32), 'hex');
    v_api_secret := 'as_' || encode(gen_random_bytes(32), 'hex');
    
    -- Insert application
    INSERT INTO applications (
        name, display_name, scenario_type, 
        api_key_hash, api_secret_hash, created_by
    ) VALUES (
        p_name, p_display_name, p_scenario_type,
        crypt(v_api_key, gen_salt('bf')),
        crypt(v_api_secret, gen_salt('bf')),
        p_created_by
    ) RETURNING id INTO v_app_id;
    
    -- Return credentials (only time they're available in plaintext)
    RETURN QUERY SELECT v_app_id, v_api_key, v_api_secret;
END;
$$ LANGUAGE plpgsql;

-- Create trigger to update application stats
CREATE OR REPLACE FUNCTION update_application_stats()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_TABLE_NAME = 'application_users' THEN
        UPDATE applications 
        SET total_users = (
            SELECT COUNT(*) FROM application_users 
            WHERE application_id = NEW.application_id AND is_active = TRUE
        )
        WHERE id = NEW.application_id;
    ELSIF TG_TABLE_NAME = 'audit_logs' AND NEW.action LIKE 'auth.%' THEN
        UPDATE applications 
        SET total_authentications = total_authentications + 1,
            last_accessed = CURRENT_TIMESTAMP
        WHERE id = NEW.application_id;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_app_stats_on_user
    AFTER INSERT OR UPDATE ON application_users
    FOR EACH ROW EXECUTE FUNCTION update_application_stats();

CREATE TRIGGER update_app_stats_on_auth
    AFTER INSERT ON audit_logs
    FOR EACH ROW 
    WHEN (NEW.action LIKE 'auth.%')
    EXECUTE FUNCTION update_application_stats();

-- Grant permissions
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO scenario_authenticator;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO scenario_authenticator;
GRANT ALL PRIVILEGES ON ALL FUNCTIONS IN SCHEMA public TO scenario_authenticator;