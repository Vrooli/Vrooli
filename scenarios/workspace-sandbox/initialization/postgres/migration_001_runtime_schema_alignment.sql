-- Align older workspace-sandbox databases with the lifecycle-managed schema.
-- This migration replaces the legacy API-side startup mutations so schema
-- evolution happens through tracked SQL artifacts instead of hidden runtime DDL.

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE OR REPLACE FUNCTION vrooli_uuid_v4()
RETURNS UUID AS $$
BEGIN
    BEGIN
        RETURN uuid_generate_v4();
    EXCEPTION
        WHEN undefined_function THEN
            RETURN gen_random_uuid();
    END;
END;
$$ LANGUAGE plpgsql;

CREATE TABLE IF NOT EXISTS applied_changes (
    id UUID PRIMARY KEY DEFAULT vrooli_uuid_v4(),
    sandbox_id UUID REFERENCES sandboxes(id) ON DELETE SET NULL,
    sandbox_owner TEXT,
    sandbox_owner_type TEXT,
    file_path TEXT NOT NULL,
    project_root TEXT NOT NULL,
    change_type TEXT NOT NULL,
    file_size BIGINT DEFAULT 0,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    committed_at TIMESTAMPTZ,
    commit_hash TEXT,
    commit_message TEXT,
    agent_manager_run_id TEXT,
    CONSTRAINT valid_applied_change_type CHECK (change_type IN ('added', 'modified', 'deleted'))
);

ALTER TABLE sandboxes ADD COLUMN IF NOT EXISTS reserved_path TEXT;
ALTER TABLE sandboxes ADD COLUMN IF NOT EXISTS reserved_paths TEXT[] DEFAULT '{}';
ALTER TABLE sandboxes ADD COLUMN IF NOT EXISTS no_lock BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE sandboxes ADD COLUMN IF NOT EXISTS behavior JSONB NOT NULL DEFAULT '{}'::jsonb;

ALTER TABLE applied_changes ADD COLUMN IF NOT EXISTS agent_manager_run_id TEXT;

UPDATE sandboxes
SET reserved_path = scope_path
WHERE reserved_path IS NULL
  AND no_lock = false;

UPDATE sandboxes
SET reserved_paths = ARRAY[COALESCE(reserved_path, scope_path)]
WHERE (reserved_paths IS NULL OR array_length(reserved_paths, 1) IS NULL OR array_length(reserved_paths, 1) = 0)
  AND no_lock = false;

UPDATE sandboxes
SET behavior = '{}'::jsonb
WHERE behavior IS NULL;

CREATE INDEX IF NOT EXISTS idx_sandboxes_reserved_path ON sandboxes(reserved_path);
CREATE INDEX IF NOT EXISTS idx_applied_changes_sandbox_id ON applied_changes(sandbox_id);
CREATE INDEX IF NOT EXISTS idx_applied_changes_file_path ON applied_changes(file_path);
CREATE INDEX IF NOT EXISTS idx_applied_changes_project_root ON applied_changes(project_root);
CREATE INDEX IF NOT EXISTS idx_applied_changes_pending ON applied_changes(committed_at) WHERE committed_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_applied_changes_run_id ON applied_changes(agent_manager_run_id);

CREATE OR REPLACE FUNCTION check_scope_overlap(
    new_scope TEXT,
    new_project TEXT,
    exclude_id UUID DEFAULT NULL
) RETURNS TABLE(id UUID, scope_path TEXT, status sandbox_status) AS $$
BEGIN
    RETURN QUERY
    SELECT s.id, existing_prefix, s.status
    FROM sandboxes s,
         LATERAL unnest(
            CASE
                WHEN s.reserved_paths IS NOT NULL AND array_length(s.reserved_paths, 1) > 0 THEN s.reserved_paths
                ELSE ARRAY[COALESCE(s.reserved_path, s.scope_path)]
            END
         ) AS existing_prefix
    WHERE s.project_root = new_project
      AND s.no_lock = false
      AND s.status IN ('creating', 'active', 'stopped')
      AND (exclude_id IS NULL OR s.id != exclude_id)
      AND (
          existing_prefix LIKE new_scope || '/%'
          OR existing_prefix = new_scope
          OR new_scope LIKE existing_prefix || '/%'
      );
END;
$$ LANGUAGE plpgsql;
