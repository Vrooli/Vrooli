-- React Component Library Database Schema
-- Version: 1.0.0

-- Create schema if not exists
CREATE SCHEMA IF NOT EXISTS react_component_library;
SET search_path TO react_component_library;

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Components table
CREATE TABLE IF NOT EXISTS components (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL,
    category VARCHAR(50) NOT NULL,
    description TEXT NOT NULL,
    code TEXT NOT NULL,
    props_schema JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    version VARCHAR(20) DEFAULT '1.0.0',
    author VARCHAR(100) DEFAULT 'anonymous',
    usage_count INTEGER DEFAULT 0,
    accessibility_score DECIMAL(5,2),
    performance_metrics JSONB,
    tags TEXT[], -- Array of tags
    is_active BOOLEAN DEFAULT true,
    dependencies TEXT[], -- Array of dependency names
    screenshots TEXT[], -- Array of screenshot URLs
    example_usage TEXT,
    
    -- Constraints
    CONSTRAINT components_name_check CHECK (length(name) > 0),
    CONSTRAINT components_description_check CHECK (length(description) >= 10),
    CONSTRAINT components_code_check CHECK (length(code) > 0),
    CONSTRAINT components_accessibility_score_check CHECK (accessibility_score >= 0 AND accessibility_score <= 100)
);

-- Component versions table
CREATE TABLE IF NOT EXISTS component_versions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    component_id UUID NOT NULL REFERENCES components(id) ON DELETE CASCADE,
    version VARCHAR(20) NOT NULL,
    code TEXT NOT NULL,
    changelog TEXT,
    breaking_changes TEXT[], -- Array of breaking change descriptions
    deprecated BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Constraints
    UNIQUE(component_id, version)
);

-- Test results table
CREATE TABLE IF NOT EXISTS test_results (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    component_id UUID NOT NULL REFERENCES components(id) ON DELETE CASCADE,
    test_type VARCHAR(50) NOT NULL,
    results JSONB NOT NULL,
    passed BOOLEAN NOT NULL,
    score DECIMAL(5,2) NOT NULL DEFAULT 0,
    tested_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    test_duration_ms INTEGER,
    
    -- Constraints
    CONSTRAINT test_results_test_type_check CHECK (test_type IN ('accessibility', 'performance', 'visual', 'unit_test', 'linting')),
    CONSTRAINT test_results_score_check CHECK (score >= 0 AND score <= 100)
);

-- Usage analytics table
CREATE TABLE IF NOT EXISTS usage_analytics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    component_id UUID NOT NULL REFERENCES components(id) ON DELETE CASCADE,
    scenario VARCHAR(100),
    context VARCHAR(100),
    used_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    user_agent TEXT,
    
    -- Additional analytics fields
    session_id VARCHAR(255),
    ip_address INET
);

-- AI generation logs table
CREATE TABLE IF NOT EXISTS ai_generation_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    request_prompt TEXT NOT NULL,
    response_code TEXT,
    component_name VARCHAR(100),
    success BOOLEAN NOT NULL,
    error_message TEXT,
    generation_time_ms INTEGER,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Associated component (if generation was successful)
    component_id UUID REFERENCES components(id) ON DELETE SET NULL
);

-- Component favorites/bookmarks (for future user system)
CREATE TABLE IF NOT EXISTS component_favorites (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    component_id UUID NOT NULL REFERENCES components(id) ON DELETE CASCADE,
    user_id VARCHAR(100) NOT NULL, -- For future user system
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Constraints
    UNIQUE(component_id, user_id)
);

-- Component ratings/reviews (for future marketplace features)
CREATE TABLE IF NOT EXISTS component_reviews (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    component_id UUID NOT NULL REFERENCES components(id) ON DELETE CASCADE,
    user_id VARCHAR(100) NOT NULL, -- For future user system
    rating INTEGER NOT NULL,
    review_text TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Constraints
    CONSTRAINT component_reviews_rating_check CHECK (rating >= 1 AND rating <= 5),
    UNIQUE(component_id, user_id)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_components_name ON components(name);
CREATE INDEX IF NOT EXISTS idx_components_category ON components(category);
CREATE INDEX IF NOT EXISTS idx_components_created_at ON components(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_components_usage_count ON components(usage_count DESC);
CREATE INDEX IF NOT EXISTS idx_components_accessibility_score ON components(accessibility_score DESC);
CREATE INDEX IF NOT EXISTS idx_components_is_active ON components(is_active);
CREATE INDEX IF NOT EXISTS idx_components_tags ON components USING GIN(tags);

CREATE INDEX IF NOT EXISTS idx_component_versions_component_id ON component_versions(component_id);
CREATE INDEX IF NOT EXISTS idx_component_versions_created_at ON component_versions(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_test_results_component_id ON test_results(component_id);
CREATE INDEX IF NOT EXISTS idx_test_results_test_type ON test_results(test_type);
CREATE INDEX IF NOT EXISTS idx_test_results_tested_at ON test_results(tested_at DESC);
CREATE INDEX IF NOT EXISTS idx_test_results_passed ON test_results(passed);

CREATE INDEX IF NOT EXISTS idx_usage_analytics_component_id ON usage_analytics(component_id);
CREATE INDEX IF NOT EXISTS idx_usage_analytics_used_at ON usage_analytics(used_at DESC);
CREATE INDEX IF NOT EXISTS idx_usage_analytics_scenario ON usage_analytics(scenario);

CREATE INDEX IF NOT EXISTS idx_ai_generation_logs_created_at ON ai_generation_logs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ai_generation_logs_success ON ai_generation_logs(success);
CREATE INDEX IF NOT EXISTS idx_ai_generation_logs_component_id ON ai_generation_logs(component_id);

-- Full-text search indexes
CREATE INDEX IF NOT EXISTS idx_components_search ON components USING GIN(
    to_tsvector('english', name || ' ' || description || ' ' || array_to_string(tags, ' '))
);

-- Triggers for updating timestamps
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_components_updated_at 
    BEFORE UPDATE ON components 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_component_reviews_updated_at 
    BEFORE UPDATE ON component_reviews 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Views for common queries
CREATE OR REPLACE VIEW popular_components AS
SELECT 
    c.*,
    COUNT(ua.id) as recent_usage_count,
    AVG(cr.rating) as avg_rating,
    COUNT(cr.id) as review_count
FROM components c
LEFT JOIN usage_analytics ua ON c.id = ua.component_id 
    AND ua.used_at > CURRENT_DATE - INTERVAL '30 days'
LEFT JOIN component_reviews cr ON c.id = cr.component_id
WHERE c.is_active = true
GROUP BY c.id
ORDER BY recent_usage_count DESC, c.usage_count DESC;

CREATE OR REPLACE VIEW component_stats AS
SELECT 
    c.id,
    c.name,
    c.category,
    c.usage_count,
    c.accessibility_score,
    (SELECT COUNT(*) FROM test_results tr WHERE tr.component_id = c.id AND tr.passed = true) as passed_tests,
    (SELECT COUNT(*) FROM test_results tr WHERE tr.component_id = c.id) as total_tests,
    (SELECT AVG(score) FROM test_results tr WHERE tr.component_id = c.id AND tr.test_type = 'accessibility') as avg_accessibility_score,
    (SELECT AVG(score) FROM test_results tr WHERE tr.component_id = c.id AND tr.test_type = 'performance') as avg_performance_score,
    AVG(cr.rating) as avg_rating,
    COUNT(cr.id) as review_count
FROM components c
LEFT JOIN component_reviews cr ON c.id = cr.component_id
WHERE c.is_active = true
GROUP BY c.id;

-- Sample data for development/testing
INSERT INTO components (
    name, 
    category, 
    description, 
    code, 
    tags,
    example_usage,
    accessibility_score
) VALUES 
(
    'Button',
    'form',
    'A customizable button component with multiple variants and accessibility features',
    'const Button = ({ children, variant = "primary", onClick, ...props }) => (
      <button 
        className={`btn btn-${variant}`} 
        onClick={onClick} 
        {...props}
      >
        {children}
      </button>
    );',
    ARRAY['button', 'form', 'interactive'],
    '<Button variant="primary" onClick={() => console.log("clicked")}>Click me</Button>',
    95.5
),
(
    'Modal',
    'feedback',
    'A flexible modal dialog component with backdrop and keyboard navigation support',
    'const Modal = ({ isOpen, onClose, children, title }) => {
      if (!isOpen) return null;
      
      return (
        <div className="modal-backdrop" onClick={onClose}>
          <div className="modal-content" onClick={e => e.stopPropagation()}>
            <div className="modal-header">
              <h2>{title}</h2>
              <button onClick={onClose} aria-label="Close">×</button>
            </div>
            <div className="modal-body">{children}</div>
          </div>
        </div>
      );
    };',
    ARRAY['modal', 'dialog', 'overlay'],
    '<Modal isOpen={true} title="Example" onClose={() => setOpen(false)}>Content</Modal>',
    88.2
),
(
    'Card',
    'layout',
    'A versatile card component for displaying content with header, body, and footer sections',
    'const Card = ({ title, children, footer, className = "" }) => (
      <div className={`card ${className}`}>
        {title && <div className="card-header"><h3>{title}</h3></div>}
        <div className="card-body">{children}</div>
        {footer && <div className="card-footer">{footer}</div>}
      </div>
    );',
    ARRAY['card', 'container', 'layout'],
    '<Card title="Example Card">This is the card content</Card>',
    92.0
) ON CONFLICT DO NOTHING;

-- Insert sample test results
INSERT INTO test_results (component_id, test_type, results, passed, score)
SELECT 
    c.id,
    'accessibility',
    '{"violations": [], "passes": ["color-contrast", "aria-labels"], "score": 95.5}',
    true,
    95.5
FROM components c 
WHERE c.name = 'Button'
ON CONFLICT DO NOTHING;

INSERT INTO test_results (component_id, test_type, results, passed, score)
SELECT 
    c.id,
    'performance',
    '{"render_time_ms": 12.3, "bundle_size_kb": 3.2, "memory_usage_mb": 1.8}',
    true,
    87.5
FROM components c 
WHERE c.name = 'Button'
ON CONFLICT DO NOTHING;

COMMIT;