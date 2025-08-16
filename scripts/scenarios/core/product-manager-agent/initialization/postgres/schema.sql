-- Product Manager Agent Database Schema

-- Products table
CREATE TABLE IF NOT EXISTS products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    vision TEXT,
    mission TEXT,
    target_market TEXT,
    status VARCHAR(50) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Features table
CREATE TABLE IF NOT EXISTS features (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID REFERENCES products(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    user_story TEXT,
    acceptance_criteria JSONB,
    priority VARCHAR(20) DEFAULT 'medium',
    effort_estimate INTEGER,
    business_value INTEGER,
    risk_level VARCHAR(20),
    status VARCHAR(50) DEFAULT 'backlog',
    sprint_id UUID,
    assigned_to VARCHAR(255),
    tags TEXT[],
    dependencies UUID[],
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Decisions table
CREATE TABLE IF NOT EXISTS decisions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID REFERENCES products(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    context TEXT,
    decision TEXT NOT NULL,
    rationale TEXT,
    alternatives_considered JSONB,
    impact_analysis JSONB,
    stakeholders TEXT[],
    decision_date DATE DEFAULT CURRENT_DATE,
    review_date DATE,
    status VARCHAR(50) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Roadmaps table
CREATE TABLE IF NOT EXISTS roadmaps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID REFERENCES products(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    timeframe VARCHAR(50),
    vision TEXT,
    goals JSONB,
    milestones JSONB,
    status VARCHAR(50) DEFAULT 'draft',
    published_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Sprints table
CREATE TABLE IF NOT EXISTS sprints (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID REFERENCES products(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    goal TEXT,
    start_date DATE,
    end_date DATE,
    velocity INTEGER,
    capacity INTEGER,
    status VARCHAR(50) DEFAULT 'planning',
    retrospective JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- User feedback table
CREATE TABLE IF NOT EXISTS user_feedback (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID REFERENCES products(id) ON DELETE CASCADE,
    feature_id UUID REFERENCES features(id) ON DELETE SET NULL,
    source VARCHAR(100),
    user_id VARCHAR(255),
    feedback_type VARCHAR(50),
    content TEXT NOT NULL,
    sentiment VARCHAR(20),
    priority VARCHAR(20),
    status VARCHAR(50) DEFAULT 'new',
    tags TEXT[],
    metadata JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    processed_at TIMESTAMP
);

-- Market research table
CREATE TABLE IF NOT EXISTS market_research (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID REFERENCES products(id) ON DELETE CASCADE,
    research_type VARCHAR(100),
    title VARCHAR(255) NOT NULL,
    summary TEXT,
    findings JSONB,
    competitors JSONB,
    market_trends JSONB,
    opportunities TEXT[],
    threats TEXT[],
    source_urls TEXT[],
    research_date DATE DEFAULT CURRENT_DATE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Metrics table
CREATE TABLE IF NOT EXISTS metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID REFERENCES products(id) ON DELETE CASCADE,
    metric_name VARCHAR(255) NOT NULL,
    metric_type VARCHAR(100),
    value NUMERIC,
    unit VARCHAR(50),
    target_value NUMERIC,
    measurement_date DATE DEFAULT CURRENT_DATE,
    tags TEXT[],
    metadata JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ROI calculations table
CREATE TABLE IF NOT EXISTS roi_calculations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    feature_id UUID REFERENCES features(id) ON DELETE CASCADE,
    investment_cost NUMERIC,
    expected_revenue NUMERIC,
    time_to_value INTEGER,
    payback_period INTEGER,
    roi_percentage NUMERIC,
    assumptions JSONB,
    risks JSONB,
    calculated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for better performance
CREATE INDEX idx_features_product_id ON features(product_id);
CREATE INDEX idx_features_status ON features(status);
CREATE INDEX idx_features_priority ON features(priority);
CREATE INDEX idx_decisions_product_id ON decisions(product_id);
CREATE INDEX idx_roadmaps_product_id ON roadmaps(product_id);
CREATE INDEX idx_sprints_product_id ON sprints(product_id);
CREATE INDEX idx_sprints_status ON sprints(status);
CREATE INDEX idx_user_feedback_product_id ON user_feedback(product_id);
CREATE INDEX idx_user_feedback_status ON user_feedback(status);
CREATE INDEX idx_market_research_product_id ON market_research(product_id);
CREATE INDEX idx_metrics_product_id ON metrics(product_id);
CREATE INDEX idx_metrics_measurement_date ON metrics(measurement_date);

-- Create update trigger for updated_at timestamps
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_products_updated_at BEFORE UPDATE ON products
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_features_updated_at BEFORE UPDATE ON features
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_decisions_updated_at BEFORE UPDATE ON decisions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_roadmaps_updated_at BEFORE UPDATE ON roadmaps
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_sprints_updated_at BEFORE UPDATE ON sprints
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_market_research_updated_at BEFORE UPDATE ON market_research
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();