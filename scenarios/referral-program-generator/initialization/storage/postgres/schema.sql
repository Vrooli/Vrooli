-- Referral Program Generator Database Schema
-- Version: 1.0.0
-- Compatible with PostgreSQL 12+

-- Create extension for UUID generation if not exists
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Create referral programs table
CREATE TABLE IF NOT EXISTS referral_programs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scenario_name VARCHAR(255) NOT NULL,
    commission_rate DECIMAL(5,4) NOT NULL CHECK (commission_rate >= 0 AND commission_rate <= 1),
    tracking_code VARCHAR(50) UNIQUE NOT NULL,
    landing_page_url TEXT,
    branding_config JSONB,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create referral links table
CREATE TABLE IF NOT EXISTS referral_links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    program_id UUID NOT NULL REFERENCES referral_programs(id) ON DELETE CASCADE,
    referrer_id UUID NOT NULL, -- References users table in scenario-authenticator
    tracking_code VARCHAR(32) UNIQUE NOT NULL,
    clicks INTEGER DEFAULT 0 CHECK (clicks >= 0),
    conversions INTEGER DEFAULT 0 CHECK (conversions >= 0),
    total_commission DECIMAL(12,2) DEFAULT 0.00 CHECK (total_commission >= 0),
    is_active BOOLEAN DEFAULT true,
    last_click_at TIMESTAMP WITH TIME ZONE,
    last_conversion_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create commissions table
CREATE TABLE IF NOT EXISTS commissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    link_id UUID NOT NULL REFERENCES referral_links(id) ON DELETE CASCADE,
    customer_id UUID NOT NULL, -- References customer/user in the purchasing scenario
    amount DECIMAL(12,2) NOT NULL CHECK (amount >= 0),
    status VARCHAR(20) NOT NULL DEFAULT 'pending' 
        CHECK (status IN ('pending', 'approved', 'paid', 'disputed', 'cancelled')),
    transaction_date TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    approved_at TIMESTAMP WITH TIME ZONE,
    paid_date TIMESTAMP WITH TIME ZONE,
    payment_method VARCHAR(50),
    payment_reference VARCHAR(255),
    notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create referral events table for detailed tracking
CREATE TABLE IF NOT EXISTS referral_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    link_id UUID NOT NULL REFERENCES referral_links(id) ON DELETE CASCADE,
    event_type VARCHAR(20) NOT NULL CHECK (event_type IN ('click', 'conversion', 'signup', 'purchase')),
    customer_id UUID,
    ip_address INET,
    user_agent TEXT,
    referrer_url TEXT,
    landing_page TEXT,
    conversion_value DECIMAL(12,2),
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create campaign table for organizing referral programs
CREATE TABLE IF NOT EXISTS referral_campaigns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    start_date TIMESTAMP WITH TIME ZONE,
    end_date TIMESTAMP WITH TIME ZONE,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Junction table for program-campaign relationships
CREATE TABLE IF NOT EXISTS program_campaigns (
    program_id UUID NOT NULL REFERENCES referral_programs(id) ON DELETE CASCADE,
    campaign_id UUID NOT NULL REFERENCES referral_campaigns(id) ON DELETE CASCADE,
    PRIMARY KEY (program_id, campaign_id)
);

-- Create indexes for performance optimization
-- Referral programs indexes
CREATE INDEX IF NOT EXISTS idx_referral_programs_scenario_name ON referral_programs(scenario_name);
CREATE INDEX IF NOT EXISTS idx_referral_programs_tracking_code ON referral_programs(tracking_code);
CREATE INDEX IF NOT EXISTS idx_referral_programs_created_at ON referral_programs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_referral_programs_active ON referral_programs(is_active) WHERE is_active = true;

-- Referral links indexes
CREATE INDEX IF NOT EXISTS idx_referral_links_program_id ON referral_links(program_id);
CREATE INDEX IF NOT EXISTS idx_referral_links_referrer_id ON referral_links(referrer_id);
CREATE INDEX IF NOT EXISTS idx_referral_links_tracking_code ON referral_links(tracking_code);
CREATE INDEX IF NOT EXISTS idx_referral_links_created_at ON referral_links(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_referral_links_active ON referral_links(is_active) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_referral_links_stats ON referral_links(clicks, conversions, total_commission);

-- Commissions indexes
CREATE INDEX IF NOT EXISTS idx_commissions_link_id ON commissions(link_id);
CREATE INDEX IF NOT EXISTS idx_commissions_customer_id ON commissions(customer_id);
CREATE INDEX IF NOT EXISTS idx_commissions_status ON commissions(status);
CREATE INDEX IF NOT EXISTS idx_commissions_transaction_date ON commissions(transaction_date DESC);
CREATE INDEX IF NOT EXISTS idx_commissions_amount ON commissions(amount);
CREATE INDEX IF NOT EXISTS idx_commissions_pending ON commissions(status) WHERE status = 'pending';

-- Referral events indexes
CREATE INDEX IF NOT EXISTS idx_referral_events_link_id ON referral_events(link_id);
CREATE INDEX IF NOT EXISTS idx_referral_events_type ON referral_events(event_type);
CREATE INDEX IF NOT EXISTS idx_referral_events_customer_id ON referral_events(customer_id);
CREATE INDEX IF NOT EXISTS idx_referral_events_created_at ON referral_events(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_referral_events_ip_address ON referral_events(ip_address);

-- Campaign indexes
CREATE INDEX IF NOT EXISTS idx_referral_campaigns_name ON referral_campaigns(name);
CREATE INDEX IF NOT EXISTS idx_referral_campaigns_dates ON referral_campaigns(start_date, end_date);
CREATE INDEX IF NOT EXISTS idx_referral_campaigns_active ON referral_campaigns(is_active) WHERE is_active = true;

-- Create trigger function for updating updated_at timestamps
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create triggers for automatic timestamp updates
CREATE TRIGGER update_referral_programs_updated_at 
    BEFORE UPDATE ON referral_programs 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_referral_links_updated_at 
    BEFORE UPDATE ON referral_links 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_commissions_updated_at 
    BEFORE UPDATE ON commissions 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_referral_campaigns_updated_at 
    BEFORE UPDATE ON referral_campaigns 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Create views for common queries
-- View for referral link statistics
CREATE OR REPLACE VIEW referral_link_stats AS
SELECT 
    rl.id,
    rl.tracking_code,
    rl.referrer_id,
    rp.scenario_name,
    rp.commission_rate,
    rl.clicks,
    rl.conversions,
    rl.total_commission,
    CASE 
        WHEN rl.clicks = 0 THEN 0
        ELSE ROUND((rl.conversions::DECIMAL / rl.clicks) * 100, 2)
    END as conversion_rate,
    rl.created_at,
    rl.last_click_at,
    rl.last_conversion_at
FROM referral_links rl
JOIN referral_programs rp ON rl.program_id = rp.id
WHERE rl.is_active = true;

-- View for commission summary by referrer
CREATE OR REPLACE VIEW referrer_commission_summary AS
SELECT 
    rl.referrer_id,
    COUNT(DISTINCT rl.id) as total_links,
    SUM(rl.clicks) as total_clicks,
    SUM(rl.conversions) as total_conversions,
    SUM(c.amount) FILTER (WHERE c.status = 'pending') as pending_commission,
    SUM(c.amount) FILTER (WHERE c.status = 'approved') as approved_commission,
    SUM(c.amount) FILTER (WHERE c.status = 'paid') as paid_commission,
    SUM(c.amount) as total_commission
FROM referral_links rl
LEFT JOIN commissions c ON rl.id = c.link_id
WHERE rl.is_active = true
GROUP BY rl.referrer_id;

-- View for program performance
CREATE OR REPLACE VIEW program_performance AS
SELECT 
    rp.id,
    rp.scenario_name,
    rp.commission_rate,
    COUNT(DISTINCT rl.id) as total_links,
    SUM(rl.clicks) as total_clicks,
    SUM(rl.conversions) as total_conversions,
    SUM(c.amount) as total_commissions,
    AVG(c.amount) as avg_commission,
    CASE 
        WHEN SUM(rl.clicks) = 0 THEN 0
        ELSE ROUND((SUM(rl.conversions)::DECIMAL / SUM(rl.clicks)) * 100, 2)
    END as overall_conversion_rate,
    rp.created_at
FROM referral_programs rp
LEFT JOIN referral_links rl ON rp.id = rl.program_id AND rl.is_active = true
LEFT JOIN commissions c ON rl.id = c.link_id
WHERE rp.is_active = true
GROUP BY rp.id, rp.scenario_name, rp.commission_rate, rp.created_at;

-- Create function for fraud detection
CREATE OR REPLACE FUNCTION detect_suspicious_activity(
    p_link_id UUID,
    p_ip_address INET DEFAULT NULL,
    p_timeframe_minutes INTEGER DEFAULT 60
)
RETURNS BOOLEAN AS $$
DECLARE
    click_count INTEGER;
    conversion_count INTEGER;
    unique_ips INTEGER;
BEGIN
    -- Count recent activities for this link
    SELECT 
        COUNT(*) FILTER (WHERE event_type = 'click'),
        COUNT(*) FILTER (WHERE event_type = 'conversion'),
        COUNT(DISTINCT ip_address)
    INTO click_count, conversion_count, unique_ips
    FROM referral_events
    WHERE link_id = p_link_id
        AND created_at >= NOW() - (p_timeframe_minutes || ' minutes')::INTERVAL
        AND (p_ip_address IS NULL OR ip_address = p_ip_address);
    
    -- Detect suspicious patterns
    RETURN (
        click_count > 100 OR                    -- Too many clicks
        conversion_count > 10 OR                -- Too many conversions
        (click_count > 0 AND unique_ips = 1)    -- All clicks from same IP
    );
END;
$$ LANGUAGE plpgsql;

-- Create function for calculating commission tiers
CREATE OR REPLACE FUNCTION calculate_tier_commission(
    p_referrer_id UUID,
    p_base_amount DECIMAL,
    p_program_id UUID
)
RETURNS DECIMAL AS $$
DECLARE
    base_rate DECIMAL;
    total_referrals INTEGER;
    tier_multiplier DECIMAL := 1.0;
BEGIN
    -- Get base commission rate
    SELECT commission_rate INTO base_rate
    FROM referral_programs
    WHERE id = p_program_id;
    
    -- Get total successful referrals for this referrer
    SELECT COUNT(*)
    INTO total_referrals
    FROM referral_links rl
    JOIN commissions c ON rl.id = c.link_id
    WHERE rl.referrer_id = p_referrer_id
        AND rl.program_id = p_program_id
        AND c.status IN ('approved', 'paid');
    
    -- Calculate tier multiplier
    IF total_referrals >= 100 THEN
        tier_multiplier := 1.5;  -- 50% bonus for 100+ referrals
    ELSIF total_referrals >= 50 THEN
        tier_multiplier := 1.3;  -- 30% bonus for 50+ referrals
    ELSIF total_referrals >= 25 THEN
        tier_multiplier := 1.2;  -- 20% bonus for 25+ referrals
    ELSIF total_referrals >= 10 THEN
        tier_multiplier := 1.1;  -- 10% bonus for 10+ referrals
    END IF;
    
    RETURN ROUND(p_base_amount * base_rate * tier_multiplier, 2);
END;
$$ LANGUAGE plpgsql;

-- Insert default data
INSERT INTO referral_campaigns (name, description, is_active) VALUES 
('Default Campaign', 'Default referral campaign for all programs', true)
ON CONFLICT DO NOTHING;

-- Comments for documentation
COMMENT ON TABLE referral_programs IS 'Main referral program configurations';
COMMENT ON TABLE referral_links IS 'Individual referral links created by users';
COMMENT ON TABLE commissions IS 'Commission records for successful referrals';
COMMENT ON TABLE referral_events IS 'Detailed tracking of all referral activities';
COMMENT ON TABLE referral_campaigns IS 'Marketing campaigns that group referral programs';

COMMENT ON COLUMN referral_programs.commission_rate IS 'Commission rate as decimal (0.1 = 10%)';
COMMENT ON COLUMN referral_links.tracking_code IS 'Unique code used in referral URLs';
COMMENT ON COLUMN commissions.status IS 'Commission status: pending, approved, paid, disputed, cancelled';
COMMENT ON COLUMN referral_events.event_type IS 'Type of event: click, conversion, signup, purchase';

-- Grant permissions (adjust as needed for your user setup)
-- GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO vrooli;
-- GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO vrooli;