-- Git Control Tower Database Schema
-- Metadata storage for commit history, audit logs, and draft commits

-- Commit metadata table
CREATE TABLE IF NOT EXISTS git_commits (
    id SERIAL PRIMARY KEY,
    commit_hash VARCHAR(40) NOT NULL,
    message TEXT NOT NULL,
    author_name VARCHAR(255),
    author_email VARCHAR(255),
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    branch VARCHAR(255),
    files_changed INTEGER DEFAULT 0,
    insertions INTEGER DEFAULT 0,
    deletions INTEGER DEFAULT 0,
    UNIQUE(commit_hash)
);

-- Audit log table
CREATE TABLE IF NOT EXISTS git_audit_log (
    id SERIAL PRIMARY KEY,
    operation VARCHAR(50) NOT NULL,
    entity_type VARCHAR(50),
    entity_id VARCHAR(255),
    details JSONB,
    user_context VARCHAR(255),
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Draft commit queue table
CREATE TABLE IF NOT EXISTS git_draft_commits (
    id SERIAL PRIMARY KEY,
    draft_id UUID NOT NULL DEFAULT gen_random_uuid(),
    message TEXT NOT NULL,
    staged_files JSONB NOT NULL,
    ai_generated BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    status VARCHAR(20) DEFAULT 'draft',
    UNIQUE(draft_id)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_git_commits_timestamp ON git_commits(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_git_commits_branch ON git_commits(branch);
CREATE INDEX IF NOT EXISTS idx_git_audit_log_timestamp ON git_audit_log(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_git_audit_log_operation ON git_audit_log(operation);
CREATE INDEX IF NOT EXISTS idx_git_draft_commits_status ON git_draft_commits(status);

-- Grant permissions
GRANT ALL ON git_commits TO vrooli;
GRANT ALL ON git_audit_log TO vrooli;
GRANT ALL ON git_draft_commits TO vrooli;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO vrooli;
