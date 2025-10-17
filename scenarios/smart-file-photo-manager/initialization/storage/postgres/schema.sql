-- Smart File & Photo Manager Database Schema
-- Semantic file management with AI-powered organization

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";
CREATE EXTENSION IF NOT EXISTS "vector";

-- Enum types
CREATE TYPE file_status AS ENUM ('pending', 'processing', 'processed', 'failed', 'archived');
CREATE TYPE file_type AS ENUM ('image', 'document', 'video', 'audio', 'archive', 'other');
CREATE TYPE processing_stage AS ENUM ('uploaded', 'extracted', 'analyzed', 'embedded', 'organized');
CREATE TYPE suggestion_type AS ENUM ('tag', 'folder', 'rename', 'duplicate', 'cleanup');
CREATE TYPE suggestion_status AS ENUM ('pending', 'accepted', 'rejected', 'auto_applied');

-- Files and photos
CREATE TABLE files (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- File identification
    original_name VARCHAR(500) NOT NULL,
    current_name VARCHAR(500),
    file_hash VARCHAR(64) UNIQUE NOT NULL, -- SHA256
    
    -- File properties
    size_bytes BIGINT NOT NULL,
    mime_type VARCHAR(255),
    file_type file_type,
    extension VARCHAR(50),
    
    -- Storage locations
    storage_path TEXT NOT NULL, -- MinIO path
    thumbnail_path TEXT,
    processed_path TEXT,
    
    -- Status and processing
    status file_status DEFAULT 'pending',
    processing_stage processing_stage DEFAULT 'uploaded',
    processing_errors TEXT,
    
    -- Extracted metadata
    width INTEGER, -- For images
    height INTEGER, -- For images
    duration_seconds INTEGER, -- For video/audio
    page_count INTEGER, -- For documents
    exif_data JSONB DEFAULT '{}'::jsonb,
    
    -- AI-extracted content
    description TEXT, -- AI-generated description
    ocr_text TEXT, -- Extracted text from images/PDFs
    detected_objects JSONB DEFAULT '[]'::jsonb, -- For images
    detected_faces INTEGER DEFAULT 0,
    
    -- Embeddings for similarity search
    content_embedding VECTOR(1536),
    visual_embedding VECTOR(1536), -- For images
    
    -- Organization
    folder_path TEXT DEFAULT '/',
    tags TEXT[] DEFAULT '{}',
    categories TEXT[] DEFAULT '{}',
    custom_metadata JSONB DEFAULT '{}'::jsonb,
    
    -- Timestamps
    file_created_at TIMESTAMP, -- Original file creation
    file_modified_at TIMESTAMP, -- Original file modification
    uploaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    processed_at TIMESTAMP,
    last_accessed_at TIMESTAMP,
    
    -- User tracking
    uploaded_by VARCHAR(255),
    owner_id VARCHAR(255),
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Extracted content chunks (for large documents)
CREATE TABLE content_chunks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    file_id UUID REFERENCES files(id) ON DELETE CASCADE,
    
    -- Chunk details
    chunk_index INTEGER NOT NULL,
    page_number INTEGER,
    
    -- Content
    content TEXT NOT NULL,
    chunk_type VARCHAR(50), -- paragraph, table, list, image_caption
    
    -- Embedding for search
    embedding VECTOR(1536),
    
    -- Position in document
    start_char INTEGER,
    end_char INTEGER,
    
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Organization suggestions from AI
CREATE TABLE suggestions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    file_id UUID REFERENCES files(id) ON DELETE CASCADE,
    
    -- Suggestion details
    type suggestion_type NOT NULL,
    status suggestion_status DEFAULT 'pending',
    
    -- Suggestion content
    suggested_value TEXT NOT NULL, -- tag name, folder path, new name, etc.
    current_value TEXT,
    
    -- AI reasoning
    reason TEXT,
    confidence DECIMAL(3,2), -- 0.00 to 1.00
    
    -- Similar files (for duplicate detection)
    similar_file_ids UUID[] DEFAULT '{}',
    similarity_scores DECIMAL(3,2)[] DEFAULT '{}',
    
    -- User interaction
    reviewed_by VARCHAR(255),
    reviewed_at TIMESTAMP,
    applied_at TIMESTAMP,
    
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Folders/Collections
CREATE TABLE folders (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Folder structure
    name VARCHAR(255) NOT NULL,
    path TEXT NOT NULL UNIQUE, -- Full path like /photos/vacation/2024
    parent_path TEXT,
    
    -- Smart folder settings
    is_smart BOOLEAN DEFAULT FALSE,
    smart_query JSONB, -- Query definition for smart folders
    auto_organize BOOLEAN DEFAULT FALSE,
    
    -- Statistics
    file_count INTEGER DEFAULT 0,
    total_size_bytes BIGINT DEFAULT 0,
    
    -- AI-generated metadata
    description TEXT,
    suggested_tags TEXT[] DEFAULT '{}',
    folder_embedding VECTOR(1536),
    
    -- Permissions
    owner_id VARCHAR(255),
    is_public BOOLEAN DEFAULT FALSE,
    
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- File relationships (for grouping related files)
CREATE TABLE file_relationships (
    file_id UUID REFERENCES files(id) ON DELETE CASCADE,
    related_file_id UUID REFERENCES files(id) ON DELETE CASCADE,
    relationship_type VARCHAR(50), -- duplicate, variant, sequence, related
    confidence DECIMAL(3,2),
    
    PRIMARY KEY (file_id, related_file_id)
);

-- Processing queue
CREATE TABLE processing_queue (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    file_id UUID REFERENCES files(id) ON DELETE CASCADE,
    
    -- Queue management
    priority INTEGER DEFAULT 5, -- 1-10, 1 is highest
    processing_type VARCHAR(50), -- extract, analyze, embed, organize
    
    -- Status
    status VARCHAR(50) DEFAULT 'pending', -- pending, processing, completed, failed
    attempts INTEGER DEFAULT 0,
    max_attempts INTEGER DEFAULT 3,
    
    -- Timing
    queued_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    
    -- Error tracking
    last_error TEXT,
    error_count INTEGER DEFAULT 0,
    
    metadata JSONB DEFAULT '{}'::jsonb
);

-- Search history (for improving suggestions)
CREATE TABLE search_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Search details
    query TEXT NOT NULL,
    search_type VARCHAR(50), -- text, semantic, visual
    filters JSONB DEFAULT '{}'::jsonb,
    
    -- Results
    result_count INTEGER,
    clicked_file_ids UUID[] DEFAULT '{}',
    
    -- User info
    user_id VARCHAR(255),
    session_id VARCHAR(255),
    
    -- Embedding for query understanding
    query_embedding VECTOR(1536),
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Batch operations
CREATE TABLE batch_operations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Operation details
    operation_type VARCHAR(50), -- organize, tag, move, delete, export
    status VARCHAR(50) DEFAULT 'pending',
    
    -- Affected files
    file_ids UUID[] NOT NULL,
    total_files INTEGER,
    processed_files INTEGER DEFAULT 0,
    
    -- Operation parameters
    parameters JSONB NOT NULL,
    
    -- Results
    successful_count INTEGER DEFAULT 0,
    failed_count INTEGER DEFAULT 0,
    error_details JSONB DEFAULT '[]'::jsonb,
    
    -- User tracking
    initiated_by VARCHAR(255),
    
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for performance
CREATE INDEX idx_files_status ON files(status);
CREATE INDEX idx_files_type ON files(file_type);
CREATE INDEX idx_files_folder ON files(folder_path);
CREATE INDEX idx_files_hash ON files(file_hash);
CREATE INDEX idx_files_uploaded ON files(uploaded_at DESC);
CREATE INDEX idx_files_owner ON files(owner_id);
CREATE INDEX idx_files_content_embedding ON files USING ivfflat (content_embedding vector_cosine_ops);
CREATE INDEX idx_files_visual_embedding ON files USING ivfflat (visual_embedding vector_cosine_ops);
CREATE INDEX idx_files_name_trgm ON files USING gin (original_name gin_trgm_ops);
CREATE INDEX idx_files_tags ON files USING gin (tags);

CREATE INDEX idx_chunks_file ON content_chunks(file_id);
CREATE INDEX idx_chunks_embedding ON content_chunks USING ivfflat (embedding vector_cosine_ops);

CREATE INDEX idx_suggestions_file ON suggestions(file_id);
CREATE INDEX idx_suggestions_type ON suggestions(type);
CREATE INDEX idx_suggestions_status ON suggestions(status);

CREATE INDEX idx_folders_path ON folders(path);
CREATE INDEX idx_folders_parent ON folders(parent_path);
CREATE INDEX idx_folders_embedding ON folders USING ivfflat (folder_embedding vector_cosine_ops);

CREATE INDEX idx_queue_status ON processing_queue(status);
CREATE INDEX idx_queue_priority ON processing_queue(priority);
CREATE INDEX idx_queue_file ON processing_queue(file_id);

CREATE INDEX idx_search_user ON search_history(user_id);
CREATE INDEX idx_search_created ON search_history(created_at DESC);
CREATE INDEX idx_search_embedding ON search_history USING ivfflat (query_embedding vector_cosine_ops);

-- Update timestamp trigger
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

-- Function to update folder statistics
CREATE OR REPLACE FUNCTION update_folder_stats()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE folders
    SET file_count = (SELECT COUNT(*) FROM files WHERE folder_path = folders.path),
        total_size_bytes = (SELECT COALESCE(SUM(size_bytes), 0) FROM files WHERE folder_path = folders.path)
    WHERE path = COALESCE(NEW.folder_path, OLD.folder_path);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_folder_stats_on_file_change
AFTER INSERT OR UPDATE OR DELETE ON files
FOR EACH ROW EXECUTE FUNCTION update_folder_stats();