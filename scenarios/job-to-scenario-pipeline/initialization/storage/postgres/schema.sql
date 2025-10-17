-- Job-to-Scenario Pipeline Database Schema

CREATE SCHEMA IF NOT EXISTS job_pipeline;

-- Jobs table (mirrors YAML structure for persistence)
CREATE TABLE IF NOT EXISTS job_pipeline.jobs (
    id VARCHAR(50) PRIMARY KEY,
    source VARCHAR(20) NOT NULL CHECK (source IN ('upwork', 'screenshot', 'manual')),
    title TEXT NOT NULL,
    description TEXT,
    budget_min DECIMAL(10, 2),
    budget_max DECIMAL(10, 2),
    budget_currency VARCHAR(3) DEFAULT 'USD',
    skills_required TEXT[], 
    timeline TEXT,
    state VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (state IN ('pending', 'researching', 'evaluated', 'approved', 'building', 'completed', 'archive')),
    imported_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Job metadata
CREATE TABLE IF NOT EXISTS job_pipeline.job_metadata (
    job_id VARCHAR(50) PRIMARY KEY REFERENCES job_pipeline.jobs(id) ON DELETE CASCADE,
    source_url TEXT,
    client_location VARCHAR(100),
    job_type VARCHAR(20) CHECK (job_type IN ('fixed', 'hourly')),
    experience_level VARCHAR(20) CHECK (experience_level IN ('entry', 'intermediate', 'expert')),
    published_at TIMESTAMP WITH TIME ZONE,
    extra_data JSONB
);

-- Research reports
CREATE TABLE IF NOT EXISTS job_pipeline.research_reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id VARCHAR(50) NOT NULL REFERENCES job_pipeline.jobs(id) ON DELETE CASCADE,
    evaluation VARCHAR(20) NOT NULL CHECK (evaluation IN ('NO_ACTION', 'NOT_RECOMMENDED', 'ALREADY_DONE', 'RECOMMENDED')),
    feasibility_score DECIMAL(3, 2) CHECK (feasibility_score >= 0 AND feasibility_score <= 1),
    existing_scenarios TEXT[],
    required_scenarios TEXT[],
    estimated_hours INTEGER,
    technical_analysis TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Proposals
CREATE TABLE IF NOT EXISTS job_pipeline.proposals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id VARCHAR(50) NOT NULL REFERENCES job_pipeline.jobs(id) ON DELETE CASCADE,
    cover_letter TEXT NOT NULL,
    technical_approach TEXT,
    timeline TEXT[],
    deliverables TEXT[],
    price DECIMAL(10, 2),
    generated_scenarios TEXT[],
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    sent_at TIMESTAMP WITH TIME ZONE,
    status VARCHAR(20) DEFAULT 'draft' CHECK (status IN ('draft', 'ready', 'sent', 'accepted', 'rejected'))
);

-- Job history/audit log
CREATE TABLE IF NOT EXISTS job_pipeline.job_history (
    id SERIAL PRIMARY KEY,
    job_id VARCHAR(50) NOT NULL REFERENCES job_pipeline.jobs(id) ON DELETE CASCADE,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    state VARCHAR(20) NOT NULL,
    action VARCHAR(100) NOT NULL,
    notes TEXT,
    performed_by VARCHAR(50) DEFAULT 'system'
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_jobs_state ON job_pipeline.jobs(state);
CREATE INDEX IF NOT EXISTS idx_jobs_source ON job_pipeline.jobs(source);
CREATE INDEX IF NOT EXISTS idx_jobs_imported_at ON job_pipeline.jobs(imported_at DESC);
CREATE INDEX IF NOT EXISTS idx_research_evaluation ON job_pipeline.research_reports(evaluation);
CREATE INDEX IF NOT EXISTS idx_research_feasibility ON job_pipeline.research_reports(feasibility_score);
CREATE INDEX IF NOT EXISTS idx_proposals_status ON job_pipeline.proposals(status);
CREATE INDEX IF NOT EXISTS idx_history_job_id ON job_pipeline.job_history(job_id, timestamp DESC);

-- Full text search
CREATE INDEX IF NOT EXISTS idx_jobs_title_search ON job_pipeline.jobs USING gin(to_tsvector('english', title));
CREATE INDEX IF NOT EXISTS idx_jobs_description_search ON job_pipeline.jobs USING gin(to_tsvector('english', description));

-- Triggers for updated_at
CREATE OR REPLACE FUNCTION job_pipeline.update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_jobs_updated_at BEFORE UPDATE ON job_pipeline.jobs
    FOR EACH ROW EXECUTE FUNCTION job_pipeline.update_updated_at_column();

-- Function to get job statistics
CREATE OR REPLACE FUNCTION job_pipeline.get_job_stats()
RETURNS TABLE (
    state VARCHAR(20),
    count BIGINT,
    avg_budget DECIMAL(10, 2),
    total_estimated_hours BIGINT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        j.state,
        COUNT(*) as count,
        AVG((j.budget_min + j.budget_max) / 2) as avg_budget,
        SUM(r.estimated_hours) as total_estimated_hours
    FROM job_pipeline.jobs j
    LEFT JOIN job_pipeline.research_reports r ON j.id = r.job_id
    GROUP BY j.state
    ORDER BY 
        CASE j.state
            WHEN 'pending' THEN 1
            WHEN 'researching' THEN 2
            WHEN 'evaluated' THEN 3
            WHEN 'approved' THEN 4
            WHEN 'building' THEN 5
            WHEN 'completed' THEN 6
            WHEN 'archive' THEN 7
        END;
END;
$$ LANGUAGE plpgsql;

-- Function to move job to next state
CREATE OR REPLACE FUNCTION job_pipeline.transition_job_state(
    p_job_id VARCHAR(50),
    p_new_state VARCHAR(20),
    p_notes TEXT DEFAULT NULL
)
RETURNS BOOLEAN AS $$
DECLARE
    v_current_state VARCHAR(20);
BEGIN
    -- Get current state
    SELECT state INTO v_current_state FROM job_pipeline.jobs WHERE id = p_job_id;
    
    IF v_current_state IS NULL THEN
        RETURN FALSE;
    END IF;
    
    -- Update job state
    UPDATE job_pipeline.jobs SET state = p_new_state WHERE id = p_job_id;
    
    -- Add history entry
    INSERT INTO job_pipeline.job_history (job_id, state, action, notes)
    VALUES (p_job_id, p_new_state, 'State transition from ' || v_current_state, p_notes);
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

-- View for job overview
CREATE OR REPLACE VIEW job_pipeline.job_overview AS
SELECT 
    j.id,
    j.source,
    j.title,
    j.state,
    j.budget_min,
    j.budget_max,
    j.imported_at,
    r.evaluation,
    r.feasibility_score,
    r.estimated_hours,
    p.price as proposal_price,
    p.status as proposal_status
FROM job_pipeline.jobs j
LEFT JOIN job_pipeline.research_reports r ON j.id = r.job_id
LEFT JOIN job_pipeline.proposals p ON j.id = p.job_id
ORDER BY j.imported_at DESC;

-- Grant permissions (adjust as needed)
GRANT ALL ON SCHEMA job_pipeline TO postgres;
GRANT ALL ON ALL TABLES IN SCHEMA job_pipeline TO postgres;
GRANT ALL ON ALL SEQUENCES IN SCHEMA job_pipeline TO postgres;
GRANT ALL ON ALL FUNCTIONS IN SCHEMA job_pipeline TO postgres;