-- Migration: Drop old model_id triggers and columns
-- This migration removes legacy schema elements that conflict with the current schema
-- Run this if you see errors like: pq: record "new" has no field "model_id"

\c financial_calculators_hub;

-- Drop triggers that reference model_id (if they exist)
DROP TRIGGER IF EXISTS set_model_id_trigger ON calculations;
DROP TRIGGER IF EXISTS set_model_id_trigger ON net_worth_entries;
DROP TRIGGER IF EXISTS set_model_id_trigger ON saved_scenarios;
DROP TRIGGER IF EXISTS set_model_id_trigger ON tax_calculations;

-- Drop the trigger function (if it exists)
DROP FUNCTION IF EXISTS set_model_id() CASCADE;

-- Drop model_id columns (if they exist)
ALTER TABLE IF EXISTS calculations DROP COLUMN IF EXISTS model_id;
ALTER TABLE IF EXISTS net_worth_entries DROP COLUMN IF EXISTS model_id;
ALTER TABLE IF EXISTS saved_scenarios DROP COLUMN IF EXISTS model_id;
ALTER TABLE IF EXISTS tax_calculations DROP COLUMN IF EXISTS model_id;

-- Confirm migration
SELECT 'Migration 001_drop_model_id_trigger completed successfully' AS status;
