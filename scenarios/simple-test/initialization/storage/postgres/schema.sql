-- Simple Test App Database Schema
-- Minimal schema for framework validation

-- Create schema for the application
CREATE SCHEMA IF NOT EXISTS simple_test;

-- Set search path
SET search_path TO simple_test, public;

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================
-- CORE TABLES
-- ============================================

-- Simple test data table
CREATE TABLE IF NOT EXISTS test_data (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Basic fields
    name VARCHAR(255) NOT NULL,
    value TEXT,
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Test results table
CREATE TABLE IF NOT EXISTS test_results (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Test information
    test_name VARCHAR(255) NOT NULL,
    status VARCHAR(50) DEFAULT 'pending', -- pending, running, passed, failed
    result_data JSONB DEFAULT '{}',
    
    -- Timestamps
    started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE
);

-- ============================================
-- INDEXES
-- ============================================

CREATE INDEX idx_test_data_name ON test_data(name);
CREATE INDEX idx_test_results_status ON test_results(status);
CREATE INDEX idx_test_results_created ON test_results(started_at DESC);

-- ============================================
-- PERMISSIONS
-- ============================================

-- Grant appropriate permissions
GRANT USAGE ON SCHEMA simple_test TO PUBLIC;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA simple_test TO PUBLIC;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA simple_test TO PUBLIC;