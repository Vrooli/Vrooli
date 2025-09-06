-- Social Media Scheduler Database Schema
-- This schema supports multi-tenant SaaS architecture with OAuth integration

-- Enable UUID generation
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Create database if it doesn't exist (handled by Vrooli setup)
-- CREATE DATABASE vrooli_social_media_scheduler;

-- Users table for multi-tenant authentication
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    subscription_tier VARCHAR(50) DEFAULT 'free' CHECK (subscription_tier IN ('free', 'professional', 'agency', 'enterprise')),
    subscription_status VARCHAR(50) DEFAULT 'active' CHECK (subscription_status IN ('active', 'cancelled', 'suspended', 'trial')),
    timezone VARCHAR(100) DEFAULT 'UTC',
    preferences JSONB DEFAULT '{}',
    usage_limits JSONB DEFAULT '{"posts_per_month": 10, "platforms": 2}',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    last_login_at TIMESTAMP
);

-- Social media platform accounts linked to users
CREATE TABLE social_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    platform VARCHAR(50) NOT NULL CHECK (platform IN ('twitter', 'instagram', 'linkedin', 'facebook', 'tiktok')),
    platform_user_id VARCHAR(255) NOT NULL,
    username VARCHAR(255) NOT NULL,
    display_name VARCHAR(255),
    avatar_url TEXT,
    access_token TEXT NOT NULL,
    refresh_token TEXT,
    token_expires_at TIMESTAMP,
    token_scopes TEXT[] DEFAULT '{}',
    account_data JSONB DEFAULT '{}', -- Store platform-specific metadata
    is_active BOOLEAN DEFAULT true,
    last_used_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(user_id, platform, platform_user_id)
);

-- Campaigns for organizing related posts (optional integration with campaign-content-studio)
CREATE TABLE campaigns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    external_campaign_id UUID, -- Reference to campaign-content-studio if applicable
    name VARCHAR(255) NOT NULL,
    description TEXT,
    brand_guidelines JSONB DEFAULT '{}',
    status VARCHAR(50) DEFAULT 'active' CHECK (status IN ('active', 'paused', 'completed', 'archived')),
    start_date DATE,
    end_date DATE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Main scheduled posts table
CREATE TABLE scheduled_posts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    campaign_id UUID REFERENCES campaigns(id) ON DELETE SET NULL,
    title VARCHAR(500) NOT NULL,
    content TEXT NOT NULL,
    platform_variants JSONB DEFAULT '{}', -- Platform-specific adapted content
    media_urls TEXT[] DEFAULT '{}',
    platforms VARCHAR(50)[] NOT NULL,
    scheduled_at TIMESTAMP NOT NULL,
    timezone VARCHAR(100) NOT NULL,
    status VARCHAR(50) DEFAULT 'scheduled' CHECK (status IN ('draft', 'scheduled', 'posting', 'posted', 'failed', 'cancelled')),
    posted_at TIMESTAMP,
    error_message TEXT,
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3,
    analytics_data JSONB DEFAULT '{}',
    metadata JSONB DEFAULT '{}', -- Additional post metadata
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Individual platform posts (one record per platform per scheduled post)
CREATE TABLE platform_posts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scheduled_post_id UUID REFERENCES scheduled_posts(id) ON DELETE CASCADE,
    social_account_id UUID REFERENCES social_accounts(id) ON DELETE CASCADE,
    platform VARCHAR(50) NOT NULL,
    platform_post_id VARCHAR(255), -- ID from the social platform after posting
    optimized_content TEXT NOT NULL,
    optimized_media_urls TEXT[] DEFAULT '{}',
    hashtags TEXT[] DEFAULT '{}',
    status VARCHAR(50) DEFAULT 'scheduled' CHECK (status IN ('scheduled', 'posting', 'posted', 'failed')),
    posted_at TIMESTAMP,
    error_message TEXT,
    retry_count INTEGER DEFAULT 0,
    engagement_data JSONB DEFAULT '{}', -- Likes, shares, comments, etc.
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Analytics tracking for performance insights
CREATE TABLE post_analytics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    platform_post_id UUID REFERENCES platform_posts(id) ON DELETE CASCADE,
    recorded_at TIMESTAMP DEFAULT NOW(),
    metrics JSONB NOT NULL, -- Platform-specific metrics (likes, shares, comments, reach, impressions)
    engagement_rate DECIMAL(5,4),
    reach INTEGER,
    impressions INTEGER,
    clicks INTEGER,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Media files uploaded by users
CREATE TABLE media_files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    filename VARCHAR(255) NOT NULL,
    original_filename VARCHAR(255) NOT NULL,
    file_type VARCHAR(100) NOT NULL,
    file_size BIGINT NOT NULL,
    dimensions JSONB, -- {width: 1920, height: 1080}
    minio_path TEXT NOT NULL,
    public_url TEXT,
    platform_variants JSONB DEFAULT '{}', -- Different sizes/formats per platform
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT NOW()
);

-- User activity log for audit and analytics
CREATE TABLE activity_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(100), -- 'post', 'campaign', 'account', etc.
    resource_id UUID,
    details JSONB DEFAULT '{}',
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

-- System jobs for background processing (complement to Redis queue)
CREATE TABLE system_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_type VARCHAR(100) NOT NULL,
    job_data JSONB NOT NULL,
    status VARCHAR(50) DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'failed', 'cancelled')),
    priority INTEGER DEFAULT 1,
    attempts INTEGER DEFAULT 0,
    max_attempts INTEGER DEFAULT 3,
    scheduled_for TIMESTAMP DEFAULT NOW(),
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    error_message TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

-- OAuth states for secure authentication flow
CREATE TABLE oauth_states (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    platform VARCHAR(50) NOT NULL,
    state_token VARCHAR(255) UNIQUE NOT NULL,
    redirect_uri TEXT,
    expires_at TIMESTAMP NOT NULL,
    used_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Create indexes for performance
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_subscription ON users(subscription_tier, subscription_status);

CREATE INDEX idx_social_accounts_user_platform ON social_accounts(user_id, platform);
CREATE INDEX idx_social_accounts_active ON social_accounts(is_active, last_used_at);
CREATE INDEX idx_social_accounts_token_expires ON social_accounts(token_expires_at) WHERE token_expires_at IS NOT NULL;

CREATE INDEX idx_campaigns_user_status ON campaigns(user_id, status);
CREATE INDEX idx_campaigns_dates ON campaigns(start_date, end_date);

CREATE INDEX idx_scheduled_posts_user_status ON scheduled_posts(user_id, status);
CREATE INDEX idx_scheduled_posts_scheduled_at ON scheduled_posts(scheduled_at, status);
CREATE INDEX idx_scheduled_posts_campaign ON scheduled_posts(campaign_id);
CREATE INDEX idx_scheduled_posts_platforms ON scheduled_posts USING GIN(platforms);

CREATE INDEX idx_platform_posts_scheduled_post ON platform_posts(scheduled_post_id);
CREATE INDEX idx_platform_posts_social_account ON platform_posts(social_account_id);
CREATE INDEX idx_platform_posts_status_posted ON platform_posts(status, posted_at);
CREATE INDEX idx_platform_posts_platform_id ON platform_posts(platform, platform_post_id);

CREATE INDEX idx_post_analytics_platform_post ON post_analytics(platform_post_id);
CREATE INDEX idx_post_analytics_recorded_at ON post_analytics(recorded_at);

CREATE INDEX idx_media_files_user ON media_files(user_id, created_at);
CREATE INDEX idx_media_files_type ON media_files(file_type);

CREATE INDEX idx_activity_logs_user_action ON activity_logs(user_id, action, created_at);
CREATE INDEX idx_activity_logs_resource ON activity_logs(resource_type, resource_id);

CREATE INDEX idx_system_jobs_status_priority ON system_jobs(status, priority, scheduled_for);
CREATE INDEX idx_system_jobs_type_status ON system_jobs(job_type, status);

CREATE INDEX idx_oauth_states_token ON oauth_states(state_token);
CREATE INDEX idx_oauth_states_expires ON oauth_states(expires_at);

-- Create functions for common operations
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Add updated_at triggers
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_social_accounts_updated_at BEFORE UPDATE ON social_accounts FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_campaigns_updated_at BEFORE UPDATE ON campaigns FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_scheduled_posts_updated_at BEFORE UPDATE ON scheduled_posts FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_platform_posts_updated_at BEFORE UPDATE ON platform_posts FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Function to clean up expired OAuth states
CREATE OR REPLACE FUNCTION cleanup_expired_oauth_states()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM oauth_states WHERE expires_at < NOW() - INTERVAL '1 day';
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Function to get user's posting statistics
CREATE OR REPLACE FUNCTION get_user_post_stats(user_uuid UUID, period_days INTEGER DEFAULT 30)
RETURNS TABLE (
    total_posts BIGINT,
    posted_posts BIGINT,
    failed_posts BIGINT,
    scheduled_posts BIGINT,
    platforms_used TEXT[],
    avg_engagement_rate DECIMAL
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        COUNT(*) as total_posts,
        COUNT(*) FILTER (WHERE sp.status = 'posted') as posted_posts,
        COUNT(*) FILTER (WHERE sp.status = 'failed') as failed_posts,
        COUNT(*) FILTER (WHERE sp.status = 'scheduled') as scheduled_posts,
        array_agg(DISTINCT unnest(sp.platforms)) as platforms_used,
        AVG(
            CASE 
                WHEN pa.engagement_rate IS NOT NULL THEN pa.engagement_rate
                ELSE 0
            END
        ) as avg_engagement_rate
    FROM scheduled_posts sp
    LEFT JOIN platform_posts pp ON sp.id = pp.scheduled_post_id
    LEFT JOIN post_analytics pa ON pp.id = pa.platform_post_id
    WHERE sp.user_id = user_uuid 
    AND sp.created_at >= NOW() - INTERVAL '%d days' % period_days;
END;
$$ LANGUAGE plpgsql;

-- Function to check user usage limits
CREATE OR REPLACE FUNCTION check_user_usage_limit(user_uuid UUID, limit_type VARCHAR)
RETURNS BOOLEAN AS $$
DECLARE
    user_limits JSONB;
    current_usage INTEGER;
    limit_value INTEGER;
BEGIN
    -- Get user limits
    SELECT usage_limits INTO user_limits FROM users WHERE id = user_uuid;
    
    -- Get limit value
    limit_value := (user_limits->>limit_type)::INTEGER;
    
    -- Calculate current usage based on limit type
    CASE limit_type
        WHEN 'posts_per_month' THEN
            SELECT COUNT(*) INTO current_usage 
            FROM scheduled_posts 
            WHERE user_id = user_uuid 
            AND created_at >= date_trunc('month', NOW());
        ELSE
            RETURN false; -- Unknown limit type
    END CASE;
    
    -- Return true if under limit, false if over
    RETURN current_usage < limit_value;
END;
$$ LANGUAGE plpgsql;

-- Create views for common queries
CREATE VIEW user_dashboard_stats AS
SELECT 
    u.id as user_id,
    u.email,
    u.subscription_tier,
    COUNT(DISTINCT sa.id) as connected_accounts,
    COUNT(DISTINCT sp.id) FILTER (WHERE sp.status = 'scheduled') as scheduled_posts,
    COUNT(DISTINCT sp.id) FILTER (WHERE sp.status = 'posted' AND sp.posted_at >= NOW() - INTERVAL '30 days') as posts_last_30d,
    COUNT(DISTINCT c.id) as active_campaigns,
    u.last_login_at,
    u.created_at as user_since
FROM users u
LEFT JOIN social_accounts sa ON u.id = sa.user_id AND sa.is_active = true
LEFT JOIN scheduled_posts sp ON u.id = sp.user_id
LEFT JOIN campaigns c ON u.id = c.user_id AND c.status = 'active'
GROUP BY u.id, u.email, u.subscription_tier, u.last_login_at, u.created_at;

-- Create view for platform performance analytics
CREATE VIEW platform_performance AS
SELECT 
    pp.platform,
    DATE_TRUNC('day', pp.posted_at) as date,
    COUNT(*) as posts_count,
    AVG(pa.engagement_rate) as avg_engagement_rate,
    SUM(pa.reach) as total_reach,
    SUM(pa.impressions) as total_impressions,
    SUM(pa.clicks) as total_clicks
FROM platform_posts pp
LEFT JOIN post_analytics pa ON pp.id = pa.platform_post_id
WHERE pp.status = 'posted' AND pp.posted_at IS NOT NULL
GROUP BY pp.platform, DATE_TRUNC('day', pp.posted_at)
ORDER BY date DESC, platform;

-- Insert initial system configuration
INSERT INTO system_jobs (job_type, job_data, status, priority) VALUES
('cleanup_expired_tokens', '{"interval_hours": 24}', 'pending', 1),
('refresh_analytics', '{"interval_hours": 6}', 'pending', 2),
('cleanup_old_logs', '{"retention_days": 90}', 'pending', 3);

-- Comments for documentation
COMMENT ON TABLE users IS 'Multi-tenant user accounts with subscription management';
COMMENT ON TABLE social_accounts IS 'OAuth-connected social media platform accounts';
COMMENT ON TABLE campaigns IS 'Optional campaign organization for posts';
COMMENT ON TABLE scheduled_posts IS 'Main posts scheduled across multiple platforms';
COMMENT ON TABLE platform_posts IS 'Individual platform-specific posts with optimization';
COMMENT ON TABLE post_analytics IS 'Engagement and performance metrics per platform post';
COMMENT ON TABLE media_files IS 'User-uploaded media with platform-specific variants';
COMMENT ON TABLE activity_logs IS 'Audit trail for user actions';
COMMENT ON TABLE system_jobs IS 'Background job queue with retry logic';
COMMENT ON TABLE oauth_states IS 'Secure OAuth flow state management';

-- Schema version tracking
CREATE TABLE schema_migrations (
    version VARCHAR(255) PRIMARY KEY,
    applied_at TIMESTAMP DEFAULT NOW()
);

INSERT INTO schema_migrations (version) VALUES ('001_initial_schema');