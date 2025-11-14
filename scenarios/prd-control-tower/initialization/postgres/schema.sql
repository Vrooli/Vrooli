-- PRD Control Tower Database Schema

-- Drafts table: stores PRD drafts with metadata
CREATE TABLE IF NOT EXISTS drafts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type VARCHAR(20) NOT NULL CHECK (entity_type IN ('scenario', 'resource')),
    entity_name VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    owner VARCHAR(255),
    source_backlog_id UUID NULL,  -- Optional link to originating backlog entry
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    status VARCHAR(20) DEFAULT 'draft' CHECK (status IN ('draft', 'validating', 'ready', 'published')),
    UNIQUE(entity_type, entity_name)
);

-- Index for faster lookups
CREATE INDEX IF NOT EXISTS idx_drafts_entity ON drafts(entity_type, entity_name);
CREATE INDEX IF NOT EXISTS idx_drafts_status ON drafts(status);
CREATE INDEX IF NOT EXISTS idx_drafts_updated ON drafts(updated_at DESC);

-- Backlog entries capture scenario/resource ideas before draft creation
CREATE TABLE IF NOT EXISTS backlog_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    idea_text TEXT NOT NULL,
    entity_type VARCHAR(20) NOT NULL DEFAULT 'scenario' CHECK (entity_type IN ('scenario', 'resource')),
    suggested_name VARCHAR(255) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'converted', 'archived')),
    converted_draft_id UUID REFERENCES drafts(id),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_backlog_status ON backlog_entries(status);
CREATE INDEX IF NOT EXISTS idx_backlog_created ON backlog_entries(created_at DESC);

-- Audit results cache: stores validation results to avoid repeated scenario-auditor calls
CREATE TABLE IF NOT EXISTS audit_results (
    draft_id UUID REFERENCES drafts(id) ON DELETE CASCADE,
    violations JSONB NOT NULL,
    cached_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (draft_id)
);

-- Index for cache invalidation checks
CREATE INDEX IF NOT EXISTS idx_audit_cached ON audit_results(cached_at DESC);
