-- Add OAuth support to existing users table
-- This migration adds OAuth provider storage if not already present

-- Add oauth_providers column if it doesn't exist
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_name = 'users'
                   AND column_name = 'oauth_providers') THEN
        ALTER TABLE users ADD COLUMN oauth_providers JSONB DEFAULT '{}'::jsonb;
    END IF;
END$$;

-- Create index on OAuth providers for faster lookups
CREATE INDEX IF NOT EXISTS idx_users_oauth ON users USING gin (oauth_providers);

-- Add comment explaining the structure
COMMENT ON COLUMN users.oauth_providers IS 'OAuth provider data: {"google": {"id": "...", "name": "...", "picture": "..."}, "github": {...}}';