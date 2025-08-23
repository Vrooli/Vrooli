-- Rollback Migration: 003_add_user_preferences.sql
-- Description: Rollback user preferences changes
-- Author: Vrooli PostgreSQL Resource
-- Date: 2025-01-31

-- Remove the column from users table
ALTER TABLE users DROP COLUMN IF EXISTS preferred_language;

-- Drop the user_preferences table
DROP TABLE IF EXISTS user_preferences CASCADE;

-- Note: This rollback will permanently delete all user preference data
-- Consider backing up the data before running this rollback:
-- pg_dump -t user_preferences > user_preferences_backup.sql