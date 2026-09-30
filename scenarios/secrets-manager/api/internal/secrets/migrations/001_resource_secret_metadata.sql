-- Add metadata columns to existing resource_secrets tables.
-- The IF NOT EXISTS clauses make this forward-only migration safe to reapply.
ALTER TABLE resource_secrets
    ADD COLUMN IF NOT EXISTS classification VARCHAR(20) NOT NULL DEFAULT 'service' CHECK (classification IN ('infrastructure', 'service', 'user')),
    ADD COLUMN IF NOT EXISTS owner_team TEXT,
    ADD COLUMN IF NOT EXISTS owner_contact TEXT,
    ADD COLUMN IF NOT EXISTS rotation_period_days INTEGER DEFAULT 0,
    ADD COLUMN IF NOT EXISTS last_rotated_at TIMESTAMP WITH TIME ZONE;

