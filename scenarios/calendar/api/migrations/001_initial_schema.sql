-- Migration: 001_initial_schema
-- Description: Initial database schema for calendar system
-- Author: AI Agent
-- Date: 2025-01-12

-- Create schema_migrations table to track applied migrations
CREATE TABLE IF NOT EXISTS schema_migrations (
    version VARCHAR(50) PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    checksum VARCHAR(32) NOT NULL
);

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";
CREATE EXTENSION IF NOT EXISTS "btree_gin";

-- Users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    auth_user_id UUID NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL UNIQUE,
    display_name VARCHAR(255) NOT NULL,
    timezone VARCHAR(50) NOT NULL DEFAULT 'UTC',
    preferences JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Events table
CREATE TABLE events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    timezone VARCHAR(50) NOT NULL DEFAULT 'UTC',
    location VARCHAR(500),
    event_type VARCHAR(50) NOT NULL DEFAULT 'meeting',
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    metadata JSONB DEFAULT '{}',
    automation_config JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT chk_events_time_order CHECK (start_time < end_time),
    CONSTRAINT chk_events_status CHECK (status IN ('active', 'cancelled', 'completed')),
    CONSTRAINT chk_events_type CHECK (event_type IN ('meeting', 'appointment', 'task', 'reminder', 'block', 'personal', 'work', 'travel'))
);

-- Event reminders table
CREATE TABLE event_reminders (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    minutes_before INTEGER NOT NULL,
    notification_type VARCHAR(20) NOT NULL DEFAULT 'email',
    scheduled_time TIMESTAMPTZ NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    notification_id VARCHAR(255),
    delivered_at TIMESTAMPTZ,
    error_message TEXT,
    retry_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT chk_reminder_minutes CHECK (minutes_before >= 0),
    CONSTRAINT chk_reminder_type CHECK (notification_type IN ('email', 'sms', 'push', 'webhook')),
    CONSTRAINT chk_reminder_status CHECK (status IN ('pending', 'sent', 'delivered', 'failed', 'cancelled')),
    CONSTRAINT chk_reminder_retry CHECK (retry_count >= 0 AND retry_count <= 10)
);

-- Recurring patterns table
CREATE TABLE recurring_patterns (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    parent_event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    pattern_type VARCHAR(20) NOT NULL,
    interval_value INTEGER NOT NULL DEFAULT 1,
    days_of_week INTEGER[],
    days_of_month INTEGER[],
    weeks_of_month INTEGER[],
    months_of_year INTEGER[],
    end_date TIMESTAMPTZ,
    max_occurrences INTEGER,
    exceptions TIMESTAMPTZ[],
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT chk_recurring_pattern_type CHECK (pattern_type IN ('daily', 'weekly', 'monthly', 'yearly', 'custom')),
    CONSTRAINT chk_recurring_interval CHECK (interval_value > 0)
);

-- Event embeddings table
CREATE TABLE event_embeddings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    qdrant_point_id UUID NOT NULL,
    embedding_version VARCHAR(20) NOT NULL DEFAULT 'v1.0',
    content_hash VARCHAR(64) NOT NULL,
    keywords TEXT[],
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT uq_event_embedding UNIQUE (event_id)
);

-- Record this migration as applied
INSERT INTO schema_migrations (version, checksum) 
VALUES ('001_initial_schema', 'a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6');