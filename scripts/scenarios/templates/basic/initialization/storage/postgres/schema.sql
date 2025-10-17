-- Basic Scenario Template Database Schema
-- This file contains the database schema for the basic template
-- Designed for simple integration testing

-- Basic test table for validating database connectivity
CREATE TABLE IF NOT EXISTS test_data (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Simple configuration table for testing resource connections
CREATE TABLE IF NOT EXISTS basic_config (
    key VARCHAR(100) PRIMARY KEY,
    value JSONB NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Index for faster configuration lookups
CREATE INDEX IF NOT EXISTS idx_basic_config_key ON basic_config(key);

-- Update trigger for updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_test_data_updated_at
    BEFORE UPDATE ON test_data
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();