#!/usr/bin/env bash
# Migration script to add missing columns to campaigns table

set -euo pipefail

echo "🔄 Migrating campaigns table..."

# Get postgres connection details
POSTGRES_HOST="${POSTGRES_HOST:-localhost}"
POSTGRES_PORT="${POSTGRES_PORT:-5432}"
POSTGRES_USER="${POSTGRES_USER:-postgres}"
POSTGRES_DB="${POSTGRES_DB:-vrooli}"

# Create migration SQL
cat << 'SQL' | docker exec -i $(docker ps -q --filter "name=postgres") psql -U "${POSTGRES_USER}" -d "${POSTGRES_DB}"
-- Add missing columns to campaigns table if they don't exist

DO $$
BEGIN
    -- Add parent_id if it doesn't exist
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'prompt_mgr'
        AND table_name = 'campaigns'
        AND column_name = 'parent_id'
    ) THEN
        ALTER TABLE prompt_mgr.campaigns
        ADD COLUMN parent_id UUID REFERENCES prompt_mgr.campaigns(id) ON DELETE CASCADE;
        RAISE NOTICE 'Added parent_id column';
    ELSE
        RAISE NOTICE 'parent_id column already exists';
    END IF;

    -- Add icon if it doesn't exist
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'prompt_mgr'
        AND table_name = 'campaigns'
        AND column_name = 'icon'
    ) THEN
        ALTER TABLE prompt_mgr.campaigns
        ADD COLUMN icon VARCHAR(50) DEFAULT 'folder';
        RAISE NOTICE 'Added icon column';
    ELSE
        RAISE NOTICE 'icon column already exists';
    END IF;
END $$;
SQL

echo "✅ Migration complete"
