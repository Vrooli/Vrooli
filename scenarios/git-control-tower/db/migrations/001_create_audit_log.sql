-- [REQ:GCT-OT-P0-007] PostgreSQL audit logging
-- Creates the git_audit_log table for tracking mutating operations

CREATE TABLE IF NOT EXISTS git_audit_log (
    id BIGSERIAL PRIMARY KEY,
    operation VARCHAR(50) NOT NULL,
    repo_dir TEXT NOT NULL,
    branch VARCHAR(255),
    paths TEXT[],
    commit_hash VARCHAR(50),
    commit_message TEXT,
    success BOOLEAN NOT NULL DEFAULT false,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    metadata JSONB
);

-- Index for querying by operation type
CREATE INDEX IF NOT EXISTS idx_audit_log_operation ON git_audit_log(operation);

-- Index for querying by timestamp (most common query pattern)
CREATE INDEX IF NOT EXISTS idx_audit_log_created_at ON git_audit_log(created_at DESC);

-- Index for querying by branch
CREATE INDEX IF NOT EXISTS idx_audit_log_branch ON git_audit_log(branch);

-- Composite index for common filter combinations
CREATE INDEX IF NOT EXISTS idx_audit_log_op_created ON git_audit_log(operation, created_at DESC);

COMMENT ON TABLE git_audit_log IS 'Audit log for git-control-tower mutating operations (stage, unstage, commit)';
COMMENT ON COLUMN git_audit_log.operation IS 'Type of operation: stage, unstage, commit';
COMMENT ON COLUMN git_audit_log.repo_dir IS 'Repository directory path';
COMMENT ON COLUMN git_audit_log.branch IS 'Branch name at time of operation';
COMMENT ON COLUMN git_audit_log.paths IS 'Files affected by the operation';
COMMENT ON COLUMN git_audit_log.commit_hash IS 'Commit hash for commit operations';
COMMENT ON COLUMN git_audit_log.commit_message IS 'Commit message for commit operations';
COMMENT ON COLUMN git_audit_log.success IS 'Whether the operation succeeded';
COMMENT ON COLUMN git_audit_log.error_message IS 'Error message if operation failed';
COMMENT ON COLUMN git_audit_log.metadata IS 'Additional context in JSON format';
