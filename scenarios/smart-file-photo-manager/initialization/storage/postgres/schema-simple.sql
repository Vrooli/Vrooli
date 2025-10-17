-- Smart File Photo Manager Database Schema (Simplified - No Vector)
-- This schema provides core functionality without vector search

-- Enable necessary extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- Main files table
CREATE TABLE IF NOT EXISTS files (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    original_name VARCHAR(255) NOT NULL,
    current_name VARCHAR(255),
    file_hash VARCHAR(64) NOT NULL,
    size_bytes BIGINT NOT NULL,
    mime_type VARCHAR(100),
    file_type VARCHAR(50),
    extension VARCHAR(20),
    storage_path TEXT NOT NULL,
    thumbnail_path TEXT,
    status VARCHAR(50) DEFAULT 'pending',
    processing_stage VARCHAR(100) DEFAULT 'uploaded',
    description TEXT,
    ocr_text TEXT,
    detected_objects JSONB,
    folder_path VARCHAR(500) DEFAULT '/',
    tags TEXT[],
    categories TEXT[],
    custom_metadata JSONB DEFAULT '{}',
    file_created_at TIMESTAMP,
    file_modified_at TIMESTAMP,
    uploaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    processed_at TIMESTAMP,
    last_accessed_at TIMESTAMP,
    uploaded_by VARCHAR(255),
    owner_id VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Suggestions table
CREATE TABLE IF NOT EXISTS suggestions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    file_id UUID NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL,
    status VARCHAR(50) DEFAULT 'pending',
    suggested_value TEXT NOT NULL,
    current_value TEXT,
    reason TEXT,
    confidence FLOAT,
    similar_file_ids UUID[],
    similarity_scores FLOAT[],
    reviewed_by VARCHAR(255),
    reviewed_at TIMESTAMP,
    applied_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Folders table for organization
CREATE TABLE IF NOT EXISTS folders (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    path VARCHAR(500) UNIQUE NOT NULL,
    parent_path VARCHAR(500),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Search history for improving suggestions
CREATE TABLE IF NOT EXISTS search_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    query TEXT NOT NULL,
    query_type VARCHAR(50) DEFAULT 'text',
    results_count INT DEFAULT 0,
    clicked_file_ids UUID[],
    search_timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    user_id VARCHAR(255)
);

-- Processing history for tracking
CREATE TABLE IF NOT EXISTS processing_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    file_id UUID NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    action VARCHAR(100) NOT NULL,
    status VARCHAR(50) NOT NULL,
    error_message TEXT,
    metadata JSONB DEFAULT '{}',
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP
);

-- Duplicate groups
CREATE TABLE IF NOT EXISTS duplicate_groups (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    file_ids UUID[] NOT NULL,
    similarity_scores FLOAT[],
    detection_method VARCHAR(50),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_files_status ON files(status);
CREATE INDEX IF NOT EXISTS idx_files_file_type ON files(file_type);
CREATE INDEX IF NOT EXISTS idx_files_folder_path ON files(folder_path);
CREATE INDEX IF NOT EXISTS idx_files_hash ON files(file_hash);
CREATE INDEX IF NOT EXISTS idx_files_uploaded_at ON files(uploaded_at DESC);
CREATE INDEX IF NOT EXISTS idx_files_owner ON files(owner_id);

-- Text search indexes
CREATE INDEX IF NOT EXISTS idx_files_name_trgm ON files USING gin (original_name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_files_description_trgm ON files USING gin (description gin_trgm_ops) WHERE description IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_files_ocr_trgm ON files USING gin (ocr_text gin_trgm_ops) WHERE ocr_text IS NOT NULL;

-- Array indexes
CREATE INDEX IF NOT EXISTS idx_files_tags ON files USING gin (tags);

-- JSONB indexes
CREATE INDEX IF NOT EXISTS idx_files_metadata ON files USING gin (custom_metadata);
CREATE INDEX IF NOT EXISTS idx_files_objects ON files USING gin (detected_objects);

-- Suggestions indexes
CREATE INDEX IF NOT EXISTS idx_suggestions_file_id ON suggestions(file_id);
CREATE INDEX IF NOT EXISTS idx_suggestions_status ON suggestions(status);

-- Folders indexes
CREATE INDEX IF NOT EXISTS idx_folders_path ON folders(path);
CREATE INDEX IF NOT EXISTS idx_folders_parent ON folders(parent_path);

-- Update triggers
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_files_updated_at BEFORE UPDATE ON files 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_folders_updated_at BEFORE UPDATE ON folders 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();