-- SCENARIO_NAME_PLACEHOLDER Database Schema
-- Core database structure for API and application data

-- Resources table: Main entity management
CREATE TABLE IF NOT EXISTS resources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    type VARCHAR(100),
    status VARCHAR(50) DEFAULT 'active',
    config JSONB DEFAULT '{}',
    metadata JSONB DEFAULT '{}',
    
    -- Relationships
    parent_id UUID REFERENCES resources(id) ON DELETE SET NULL,
    owner_id VARCHAR(255),
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    
    -- Constraints
    CONSTRAINT unique_name_per_type UNIQUE(name, type)
);

-- Workflows table: Workflow definitions
CREATE TABLE IF NOT EXISTS workflows (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    platform VARCHAR(50) NOT NULL CHECK (platform IN ('n8n', 'node-red')),
    
    -- Workflow configuration
    definition JSONB NOT NULL,
    input_schema JSONB,
    output_schema JSONB,
    config JSONB DEFAULT '{}',
    
    -- Metadata
    version INTEGER DEFAULT 1,
    is_active BOOLEAN DEFAULT true,
    tags TEXT[],
    
    -- Statistics
    execution_count INTEGER DEFAULT 0,
    success_count INTEGER DEFAULT 0,
    failure_count INTEGER DEFAULT 0,
    avg_duration_ms INTEGER,
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Executions table: Workflow execution history
CREATE TABLE IF NOT EXISTS executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id UUID REFERENCES workflows(id) ON DELETE CASCADE,
    
    -- Execution details
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    input_data JSONB,
    output_data JSONB,
    error_message TEXT,
    
    -- Performance metrics
    started_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE,
    duration_ms INTEGER,
    
    -- Resource usage
    resource_usage JSONB DEFAULT '{}',
    
    -- Context
    triggered_by VARCHAR(255),
    correlation_id UUID,
    metadata JSONB DEFAULT '{}'
);

-- Events table: Application event log
CREATE TABLE IF NOT EXISTS events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type VARCHAR(100) NOT NULL,
    event_name VARCHAR(255) NOT NULL,
    
    -- Event data
    payload JSONB,
    source VARCHAR(100),
    target VARCHAR(100),
    
    -- Context
    resource_id UUID REFERENCES resources(id) ON DELETE CASCADE,
    execution_id UUID REFERENCES executions(id) ON DELETE CASCADE,
    user_id VARCHAR(255),
    session_id VARCHAR(255),
    
    -- Metadata
    severity VARCHAR(20) DEFAULT 'info',
    tags TEXT[],
    
    -- Timestamp
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Configuration table: Application settings
CREATE TABLE IF NOT EXISTS configuration (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key VARCHAR(255) UNIQUE NOT NULL,
    value JSONB NOT NULL,
    
    -- Configuration metadata
    category VARCHAR(100),
    description TEXT,
    is_secret BOOLEAN DEFAULT false,
    is_active BOOLEAN DEFAULT true,
    
    -- Validation
    schema JSONB,
    default_value JSONB,
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Metrics table: Performance and usage metrics
CREATE TABLE IF NOT EXISTS metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    metric_name VARCHAR(255) NOT NULL,
    metric_type VARCHAR(50) NOT NULL,
    
    -- Metric data
    value NUMERIC NOT NULL,
    unit VARCHAR(50),
    
    -- Context
    resource_type VARCHAR(100),
    resource_id VARCHAR(255),
    
    -- Dimensions for grouping
    dimensions JSONB DEFAULT '{}',
    
    -- Timestamp
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Sessions table: User session management
CREATE TABLE IF NOT EXISTS sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_token VARCHAR(255) UNIQUE NOT NULL,
    user_id VARCHAR(255),
    
    -- Session data
    data JSONB DEFAULT '{}',
    ip_address INET,
    user_agent TEXT,
    
    -- Status
    is_active BOOLEAN DEFAULT true,
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_activity TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_resources_type ON resources(type) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_resources_status ON resources(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_resources_owner ON resources(owner_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_resources_created ON resources(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_workflows_platform ON workflows(platform) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_workflows_tags ON workflows USING GIN(tags);
CREATE INDEX IF NOT EXISTS idx_workflows_active ON workflows(is_active);

CREATE INDEX IF NOT EXISTS idx_executions_workflow ON executions(workflow_id);
CREATE INDEX IF NOT EXISTS idx_executions_status ON executions(status);
CREATE INDEX IF NOT EXISTS idx_executions_started ON executions(started_at DESC);
CREATE INDEX IF NOT EXISTS idx_executions_correlation ON executions(correlation_id);

CREATE INDEX IF NOT EXISTS idx_events_type ON events(event_type);
CREATE INDEX IF NOT EXISTS idx_events_resource ON events(resource_id);
CREATE INDEX IF NOT EXISTS idx_events_execution ON events(execution_id);
CREATE INDEX IF NOT EXISTS idx_events_created ON events(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_events_tags ON events USING GIN(tags);

CREATE INDEX IF NOT EXISTS idx_configuration_category ON configuration(category) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_configuration_key ON configuration(key) WHERE is_active = true;

CREATE INDEX IF NOT EXISTS idx_metrics_name ON metrics(metric_name);
CREATE INDEX IF NOT EXISTS idx_metrics_timestamp ON metrics(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_metrics_resource ON metrics(resource_type, resource_id);

CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(session_token) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_sessions_activity ON sessions(last_activity DESC) WHERE is_active = true;

-- Update triggers for timestamps
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

DROP TRIGGER IF EXISTS update_resources_updated_at ON resources;
CREATE TRIGGER update_resources_updated_at BEFORE UPDATE ON resources
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_workflows_updated_at ON workflows;
CREATE TRIGGER update_workflows_updated_at BEFORE UPDATE ON workflows
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_configuration_updated_at ON configuration;
CREATE TRIGGER update_configuration_updated_at BEFORE UPDATE ON configuration
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Function to update workflow statistics after execution
CREATE OR REPLACE FUNCTION update_workflow_stats()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.status IN ('success', 'failed') AND NEW.workflow_id IS NOT NULL THEN
        UPDATE workflows SET
            execution_count = execution_count + 1,
            success_count = success_count + CASE WHEN NEW.status = 'success' THEN 1 ELSE 0 END,
            failure_count = failure_count + CASE WHEN NEW.status = 'failed' THEN 1 ELSE 0 END,
            avg_duration_ms = CASE
                WHEN execution_count = 0 THEN NEW.duration_ms
                ELSE ((avg_duration_ms * execution_count + COALESCE(NEW.duration_ms, 0)) / (execution_count + 1))::INTEGER
            END,
            updated_at = CURRENT_TIMESTAMP
        WHERE id = NEW.workflow_id;
    END IF;
    RETURN NEW;
END;
$$ language 'plpgsql';

DROP TRIGGER IF EXISTS update_workflow_stats_on_execution ON executions;
CREATE TRIGGER update_workflow_stats_on_execution
    AFTER INSERT OR UPDATE OF status ON executions
    FOR EACH ROW
    WHEN (NEW.status IN ('success', 'failed'))
    EXECUTE FUNCTION update_workflow_stats();

-- Useful views for monitoring
CREATE OR REPLACE VIEW workflow_performance AS
SELECT 
    w.name,
    w.platform,
    w.execution_count,
    w.success_count,
    w.failure_count,
    CASE WHEN w.execution_count > 0 
         THEN (w.success_count::FLOAT / w.execution_count * 100)::NUMERIC(5,2)
         ELSE 0 
    END as success_rate,
    w.avg_duration_ms,
    w.is_active
FROM workflows w
ORDER BY w.execution_count DESC;

CREATE OR REPLACE VIEW recent_executions AS
SELECT 
    e.id,
    w.name as workflow_name,
    e.status,
    e.started_at,
    e.completed_at,
    e.duration_ms,
    e.error_message
FROM executions e
JOIN workflows w ON e.workflow_id = w.id
ORDER BY e.started_at DESC
LIMIT 100;

CREATE OR REPLACE VIEW resource_summary AS
SELECT 
    type,
    status,
    COUNT(*) as count,
    MAX(created_at) as latest_created,
    MAX(updated_at) as latest_updated
FROM resources
WHERE deleted_at IS NULL
GROUP BY type, status
ORDER BY type, status;
-- File Tools Database Schema
-- Complete schema for file operations and management

-- Create database if not exists
-- Note: This needs to be run as superuser or database owner
-- CREATE DATABASE file_tools;

-- File Assets table: Core file metadata storage
CREATE TABLE IF NOT EXISTS file_assets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(500) NOT NULL,
    path TEXT NOT NULL,
    size_bytes BIGINT NOT NULL,
    mime_type VARCHAR(255),
    file_extension VARCHAR(50),
    checksum_md5 VARCHAR(32),
    checksum_sha1 VARCHAR(40), 
    checksum_sha256 VARCHAR(64),
    
    -- File attributes
    permissions VARCHAR(10),
    owner VARCHAR(255),
    group_name VARCHAR(255),
    
    -- Content categorization
    content_type VARCHAR(50) CHECK (content_type IN ('text', 'image', 'video', 'audio', 'binary', 'archive', 'document')),
    is_archived BOOLEAN DEFAULT false,
    archive_location TEXT,
    tags TEXT[],
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    
    -- Timestamps
    file_created_at TIMESTAMP WITH TIME ZONE,
    file_modified_at TIMESTAMP WITH TIME ZONE,
    file_accessed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Indexes
    CONSTRAINT unique_file_path UNIQUE(path)
);

-- File Metadata table: Extended metadata extraction
CREATE TABLE IF NOT EXISTS file_metadata (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id UUID NOT NULL REFERENCES file_assets(id) ON DELETE CASCADE,
    metadata_type VARCHAR(50) CHECK (metadata_type IN ('exif', 'properties', 'extended_attributes', 'content_analysis')),
    extraction_method VARCHAR(100),
    raw_metadata JSONB,
    processed_metadata JSONB,
    confidence_score DECIMAL(3,2),
    extracted_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    extractor_version VARCHAR(50),
    validation_status VARCHAR(20) CHECK (validation_status IN ('valid', 'invalid', 'partial'))
);

-- Duplicate Groups table: Track duplicate files
CREATE TABLE IF NOT EXISTS duplicate_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    similarity_type VARCHAR(50) CHECK (similarity_type IN ('exact', 'hash_match', 'content_similar', 'name_similar')),
    similarity_score DECIMAL(5,4),
    group_hash VARCHAR(64),
    file_count INTEGER NOT NULL,
    total_size_bytes BIGINT,
    potential_savings_bytes BIGINT,
    resolution_status VARCHAR(20) CHECK (resolution_status IN ('pending', 'resolved', 'ignored')),
    resolution_action TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_verified TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Duplicate Files junction table
CREATE TABLE IF NOT EXISTS duplicate_files (
    group_id UUID NOT NULL REFERENCES duplicate_groups(id) ON DELETE CASCADE,
    file_id UUID NOT NULL REFERENCES file_assets(id) ON DELETE CASCADE,
    is_primary BOOLEAN DEFAULT false,
    added_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (group_id, file_id)
);

-- File Operations table: Track all file operations
CREATE TABLE IF NOT EXISTS file_operations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    operation_type VARCHAR(50) CHECK (operation_type IN ('copy', 'move', 'delete', 'compress', 'extract', 'split', 'merge', 'checksum')),
    source_files JSONB,
    target_location TEXT,
    parameters JSONB,
    status VARCHAR(20) CHECK (status IN ('pending', 'running', 'completed', 'failed', 'cancelled')),
    progress_percentage INTEGER CHECK (progress_percentage >= 0 AND progress_percentage <= 100),
    bytes_processed BIGINT,
    error_message TEXT,
    created_by VARCHAR(255),
    rollback_data JSONB,
    start_time TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    end_time TIMESTAMP WITH TIME ZONE
);

-- File Relationships table: Track file dependencies
CREATE TABLE IF NOT EXISTS file_relationships (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    parent_file_id UUID NOT NULL REFERENCES file_assets(id) ON DELETE CASCADE,
    child_file_id UUID NOT NULL REFERENCES file_assets(id) ON DELETE CASCADE,
    relationship_type VARCHAR(50) CHECK (relationship_type IN ('contains', 'references', 'depends_on', 'version_of', 'similar_to')),
    strength DECIMAL(3,2),
    discovered_by VARCHAR(50) CHECK (discovered_by IN ('content_analysis', 'user_defined', 'automatic')),
    discovered_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    validated BOOLEAN DEFAULT false,
    notes TEXT,
    CONSTRAINT unique_relationship UNIQUE(parent_file_id, child_file_id, relationship_type)
);

-- Archives table: Track compressed archives
CREATE TABLE IF NOT EXISTS archives (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id UUID NOT NULL REFERENCES file_assets(id) ON DELETE CASCADE,
    archive_format VARCHAR(20) CHECK (archive_format IN ('zip', 'tar', 'gzip', 'bzip2', '7z', 'rar')),
    compression_level INTEGER,
    original_size_bytes BIGINT,
    compressed_size_bytes BIGINT,
    compression_ratio DECIMAL(5,2),
    files_count INTEGER,
    is_encrypted BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Archive Contents table: Files within archives
CREATE TABLE IF NOT EXISTS archive_contents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    archive_id UUID NOT NULL REFERENCES archives(id) ON DELETE CASCADE,
    file_path TEXT NOT NULL,
    file_size BIGINT,
    compressed_size BIGINT,
    checksum VARCHAR(64),
    extracted_metadata JSONB
);

-- Split Files table: Track split file operations
CREATE TABLE IF NOT EXISTS split_files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    original_file_id UUID REFERENCES file_assets(id) ON DELETE SET NULL,
    split_pattern VARCHAR(255),
    part_size_bytes BIGINT,
    total_parts INTEGER,
    checksum_verification VARCHAR(64),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Split Parts table: Individual split file parts
CREATE TABLE IF NOT EXISTS split_parts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    split_id UUID NOT NULL REFERENCES split_files(id) ON DELETE CASCADE,
    part_number INTEGER NOT NULL,
    file_id UUID REFERENCES file_assets(id) ON DELETE CASCADE,
    checksum VARCHAR(64),
    CONSTRAINT unique_split_part UNIQUE(split_id, part_number)
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_file_assets_name ON file_assets(name);
CREATE INDEX IF NOT EXISTS idx_file_assets_path ON file_assets(path);
CREATE INDEX IF NOT EXISTS idx_file_assets_mime_type ON file_assets(mime_type);
CREATE INDEX IF NOT EXISTS idx_file_assets_size ON file_assets(size_bytes);
CREATE INDEX IF NOT EXISTS idx_file_assets_checksums ON file_assets(checksum_md5, checksum_sha256);
CREATE INDEX IF NOT EXISTS idx_file_assets_tags ON file_assets USING GIN(tags);
CREATE INDEX IF NOT EXISTS idx_file_assets_created ON file_assets(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_file_metadata_file ON file_metadata(file_id);
CREATE INDEX IF NOT EXISTS idx_file_metadata_type ON file_metadata(metadata_type);

CREATE INDEX IF NOT EXISTS idx_duplicate_groups_type ON duplicate_groups(similarity_type);
CREATE INDEX IF NOT EXISTS idx_duplicate_groups_status ON duplicate_groups(resolution_status);

CREATE INDEX IF NOT EXISTS idx_file_operations_type ON file_operations(operation_type);
CREATE INDEX IF NOT EXISTS idx_file_operations_status ON file_operations(status);
CREATE INDEX IF NOT EXISTS idx_file_operations_created ON file_operations(start_time DESC);

CREATE INDEX IF NOT EXISTS idx_file_relationships_parent ON file_relationships(parent_file_id);
CREATE INDEX IF NOT EXISTS idx_file_relationships_child ON file_relationships(child_file_id);
CREATE INDEX IF NOT EXISTS idx_file_relationships_type ON file_relationships(relationship_type);

CREATE INDEX IF NOT EXISTS idx_archives_file ON archives(file_id);
CREATE INDEX IF NOT EXISTS idx_archives_format ON archives(archive_format);

-- Update triggers for timestamps
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

DROP TRIGGER IF EXISTS update_file_assets_updated_at ON file_assets;
CREATE TRIGGER update_file_assets_updated_at BEFORE UPDATE ON file_assets
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_duplicate_groups_updated_at ON duplicate_groups;
CREATE TRIGGER update_duplicate_groups_updated_at BEFORE UPDATE ON duplicate_groups
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Useful views for monitoring
CREATE OR REPLACE VIEW file_statistics AS
SELECT 
    COUNT(*) as total_files,
    SUM(size_bytes) as total_size_bytes,
    AVG(size_bytes) as avg_file_size,
    COUNT(DISTINCT mime_type) as unique_mime_types,
    COUNT(DISTINCT file_extension) as unique_extensions
FROM file_assets;

CREATE OR REPLACE VIEW duplicate_summary AS
SELECT 
    dg.id,
    dg.similarity_type,
    dg.file_count,
    dg.potential_savings_bytes,
    dg.resolution_status,
    COUNT(df.file_id) as actual_file_count
FROM duplicate_groups dg
LEFT JOIN duplicate_files df ON dg.id = df.group_id
GROUP BY dg.id, dg.similarity_type, dg.file_count, dg.potential_savings_bytes, dg.resolution_status;

CREATE OR REPLACE VIEW recent_operations AS
SELECT 
    id,
    operation_type,
    status,
    progress_percentage,
    start_time,
    end_time,
    EXTRACT(EPOCH FROM (COALESCE(end_time, CURRENT_TIMESTAMP) - start_time)) as duration_seconds,
    error_message
FROM file_operations
ORDER BY start_time DESC
LIMIT 100;

CREATE OR REPLACE VIEW compression_performance AS
SELECT 
    a.archive_format,
    COUNT(*) as total_archives,
    AVG(a.compression_ratio) as avg_compression_ratio,
    SUM(a.original_size_bytes) as total_original_size,
    SUM(a.compressed_size_bytes) as total_compressed_size,
    SUM(a.original_size_bytes - a.compressed_size_bytes) as total_saved_bytes
FROM archives a
GROUP BY a.archive_format
ORDER BY total_archives DESC;
