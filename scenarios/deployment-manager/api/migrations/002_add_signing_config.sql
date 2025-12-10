-- Migration: Add signing_config column to profiles table
-- Version: 002
-- Description: Adds code signing configuration storage for desktop bundle deployments

-- Up Migration
-- The signing_config column stores platform-specific code signing settings as JSONB.
-- This enables Windows Authenticode, macOS Developer ID, and Linux GPG signing.

-- Check if column exists before adding (idempotent)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'profiles' AND column_name = 'signing_config'
    ) THEN
        ALTER TABLE profiles ADD COLUMN signing_config JSONB DEFAULT NULL;

        -- Add a comment describing the column
        COMMENT ON COLUMN profiles.signing_config IS
            'Code signing configuration for desktop deployments. Contains platform-specific settings for Windows, macOS, and Linux signing.';

        -- Add partial index for profiles with signing enabled
        CREATE INDEX IF NOT EXISTS idx_profiles_signing_enabled
            ON profiles ((signing_config->>'enabled'))
            WHERE signing_config IS NOT NULL;
    END IF;
END $$;

-- Down Migration (for rollback)
-- Run this manually if you need to undo the migration:
-- ALTER TABLE profiles DROP COLUMN IF EXISTS signing_config;
-- DROP INDEX IF EXISTS idx_profiles_signing_enabled;
