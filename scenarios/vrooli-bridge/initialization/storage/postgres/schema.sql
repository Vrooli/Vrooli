-- Vrooli Bridge Database Schema
-- Manages external project integrations and tracking

-- Project registry table
CREATE TABLE IF NOT EXISTS projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    path TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    type TEXT NOT NULL DEFAULT 'generic',
    vrooli_version TEXT,
    bridge_version TEXT,
    integration_status TEXT CHECK (integration_status IN ('active', 'outdated', 'missing', 'error')) DEFAULT 'missing',
    last_updated TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    metadata JSONB DEFAULT '{}'::jsonb
);

-- Integration history table
CREATE TABLE IF NOT EXISTS integration_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    action TEXT NOT NULL CHECK (action IN ('created', 'updated', 'removed', 'scanned')),
    vrooli_version TEXT,
    bridge_version TEXT,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    files_modified JSONB DEFAULT '[]'::jsonb,
    details JSONB DEFAULT '{}'::jsonb
);

-- Project files tracking (for integrity checking)
CREATE TABLE IF NOT EXISTS project_files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    file_path TEXT NOT NULL,
    file_type TEXT CHECK (file_type IN ('vrooli_integration', 'claude_additions', 'readme_update')) NOT NULL,
    checksum TEXT,
    last_verified TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(project_id, file_path)
);

-- Project tags for categorization
CREATE TABLE IF NOT EXISTS project_tags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    tag TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(project_id, tag)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_projects_type ON projects(type);
CREATE INDEX IF NOT EXISTS idx_projects_status ON projects(integration_status);
CREATE INDEX IF NOT EXISTS idx_projects_updated ON projects(last_updated);
CREATE INDEX IF NOT EXISTS idx_history_project_id ON integration_history(project_id);
CREATE INDEX IF NOT EXISTS idx_history_timestamp ON integration_history(timestamp);
CREATE INDEX IF NOT EXISTS idx_files_project_id ON project_files(project_id);
CREATE INDEX IF NOT EXISTS idx_tags_project_id ON project_tags(project_id);

-- Views for common queries
CREATE OR REPLACE VIEW project_summary AS
SELECT 
    p.id,
    p.name,
    p.path,
    p.type,
    p.integration_status,
    p.last_updated,
    p.vrooli_version,
    COUNT(pf.id) as file_count,
    COUNT(pt.id) as tag_count,
    MAX(ih.timestamp) as last_action
FROM projects p
LEFT JOIN project_files pf ON p.id = pf.project_id
LEFT JOIN project_tags pt ON p.id = pt.project_id  
LEFT JOIN integration_history ih ON p.id = ih.project_id
GROUP BY p.id, p.name, p.path, p.type, p.integration_status, p.last_updated, p.vrooli_version;

-- Integration health view
CREATE OR REPLACE VIEW integration_health AS
SELECT 
    p.type,
    p.integration_status,
    COUNT(*) as count,
    AVG(EXTRACT(EPOCH FROM (NOW() - p.last_updated))/86400) as avg_days_since_update
FROM projects p
GROUP BY p.type, p.integration_status;

-- Functions
CREATE OR REPLACE FUNCTION update_project_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.last_updated = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Triggers
DROP TRIGGER IF EXISTS trigger_update_project_timestamp ON projects;
CREATE TRIGGER trigger_update_project_timestamp
    BEFORE UPDATE ON projects
    FOR EACH ROW
    EXECUTE FUNCTION update_project_timestamp();

-- Function to add integration history entry
CREATE OR REPLACE FUNCTION add_integration_history(
    p_project_id UUID,
    p_action TEXT,
    p_vrooli_version TEXT DEFAULT NULL,
    p_bridge_version TEXT DEFAULT NULL,
    p_files_modified JSONB DEFAULT '[]'::jsonb,
    p_details JSONB DEFAULT '{}'::jsonb
) RETURNS UUID AS $$
DECLARE
    history_id UUID;
BEGIN
    INSERT INTO integration_history (
        project_id, action, vrooli_version, bridge_version, 
        files_modified, details
    ) VALUES (
        p_project_id, p_action, p_vrooli_version, p_bridge_version,
        p_files_modified, p_details
    ) RETURNING id INTO history_id;
    
    RETURN history_id;
END;
$$ LANGUAGE plpgsql;