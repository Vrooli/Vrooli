-- ROI Fit Analysis Database Schema

-- Analysis table for storing ROI analyses
CREATE TABLE IF NOT EXISTS roi_analyses (
    id SERIAL PRIMARY KEY,
    analysis_id VARCHAR(255) UNIQUE NOT NULL,
    idea TEXT NOT NULL,
    budget DECIMAL(12, 2) NOT NULL,
    timeline VARCHAR(50) NOT NULL,
    skills JSONB,
    market_focus TEXT,
    roi_percentage DECIMAL(8, 2),
    estimated_revenue DECIMAL(12, 2),
    payback_months INTEGER,
    risk_level VARCHAR(20),
    confidence_score DECIMAL(5, 2),
    detailed_analysis JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    status VARCHAR(20) DEFAULT 'completed'
);

-- Market research data
CREATE TABLE IF NOT EXISTS market_research (
    id SERIAL PRIMARY KEY,
    analysis_id INTEGER REFERENCES roi_analyses(id) ON DELETE CASCADE,
    data_source VARCHAR(255),
    market_trend TEXT,
    competitor_analysis JSONB,
    pricing_data JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Financial projections
CREATE TABLE IF NOT EXISTS financial_projections (
    id SERIAL PRIMARY KEY,
    analysis_id INTEGER REFERENCES roi_analyses(id) ON DELETE CASCADE,
    month INTEGER NOT NULL,
    revenue DECIMAL(12, 2),
    costs DECIMAL(12, 2),
    profit DECIMAL(12, 2),
    cumulative_roi DECIMAL(8, 2),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Risk factors
CREATE TABLE IF NOT EXISTS risk_factors (
    id SERIAL PRIMARY KEY,
    analysis_id INTEGER REFERENCES roi_analyses(id) ON DELETE CASCADE,
    factor_type VARCHAR(100),
    description TEXT,
    impact_level VARCHAR(20),
    mitigation_strategy TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Opportunities tracking
CREATE TABLE IF NOT EXISTS opportunities (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    category VARCHAR(100),
    roi_score DECIMAL(8, 2),
    payback_months INTEGER,
    investment_required DECIMAL(12, 2),
    market_size DECIMAL(15, 2),
    priority VARCHAR(20),
    status VARCHAR(50) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- User preferences for analysis
CREATE TABLE IF NOT EXISTS user_preferences (
    id SERIAL PRIMARY KEY,
    preference_key VARCHAR(100) UNIQUE NOT NULL,
    preference_value TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_analyses_status ON roi_analyses(status);
CREATE INDEX IF NOT EXISTS idx_analyses_created ON roi_analyses(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_opportunities_roi ON opportunities(roi_score DESC);
CREATE INDEX IF NOT EXISTS idx_opportunities_status ON opportunities(status);

-- Create update trigger for updated_at columns
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_roi_analyses_updated_at BEFORE UPDATE ON roi_analyses
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_opportunities_updated_at BEFORE UPDATE ON opportunities
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Insert sample opportunities for initial testing
INSERT INTO opportunities (name, category, roi_score, payback_months, investment_required, market_size, priority) VALUES
('AI-Powered CRM Solution', 'SaaS', 85.5, 14, 75000, 125000000000, 'high'),
('E-commerce Platform', 'E-commerce', 72.3, 18, 50000, 85000000000, 'medium'),
('EdTech Solution', 'Education', 91.2, 12, 100000, 200000000000, 'high'),
('FinTech Mobile App', 'Finance', 78.9, 16, 80000, 150000000000, 'high'),
('Healthcare Analytics', 'Healthcare', 82.1, 20, 120000, 300000000000, 'medium')
ON CONFLICT DO NOTHING;

-- Insert sample preferences
INSERT INTO user_preferences (preference_key, preference_value) VALUES
('default_risk_tolerance', 'medium'),
('preferred_payback_period', '18'),
('minimum_roi_score', '60'),
('default_market_focus', 'technology')
ON CONFLICT (preference_key) DO NOTHING;