-- Migration: 003_add_user_preferences.sql
-- Description: Add user preferences table for storing user settings
-- Author: Vrooli PostgreSQL Resource  
-- Date: 2025-01-31

-- Create user_preferences table
CREATE TABLE IF NOT EXISTS user_preferences (
    user_id INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    theme VARCHAR(20) DEFAULT 'light',
    language VARCHAR(10) DEFAULT 'en',
    timezone VARCHAR(50) DEFAULT 'UTC',
    email_notifications BOOLEAN DEFAULT true,
    push_notifications BOOLEAN DEFAULT false,
    weekly_digest BOOLEAN DEFAULT true,
    preferences_data JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Add update trigger
CREATE TRIGGER update_user_preferences_updated_at BEFORE UPDATE
    ON user_preferences FOR EACH ROW EXECUTE PROCEDURE update_updated_at_column();

-- Migrate existing users to have default preferences
INSERT INTO user_preferences (user_id)
SELECT id FROM users
ON CONFLICT (user_id) DO NOTHING;

-- Add new column to users table for quick access
ALTER TABLE users ADD COLUMN IF NOT EXISTS preferred_language VARCHAR(10) DEFAULT 'en';

-- Update users table with language from preferences
UPDATE users u
SET preferred_language = up.language
FROM user_preferences up
WHERE u.id = up.user_id;