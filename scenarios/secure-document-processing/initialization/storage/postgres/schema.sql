-- Secure Document Processing Pipeline Database Schema
-- Provides comprehensive document management, workflow storage, and audit capabilities

-- Create schema for the application
CREATE SCHEMA IF NOT EXISTS secure_doc_processing;

-- Set search path
SET search_path TO secure_doc_processing, public;

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================
-- CORE TABLES
-- ============================================

-- Document metadata and management
CREATE TABLE IF NOT EXISTS documents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Document identification
    filename VARCHAR(255) NOT NULL,
    original_filename VARCHAR(255) NOT NULL,
    mime_type VARCHAR(100),
    file_size_bytes BIGINT,
    file_hash VARCHAR(64), -- SHA-256 hash for deduplication
    
    -- Storage information
    storage_path VARCHAR(500), -- MinIO path
    encryption_key_id VARCHAR(255), -- Vault key reference
    storage_bucket VARCHAR(100) DEFAULT 'documents',
    
    -- Processing metadata
    source_type VARCHAR(50) NOT NULL, -- 'upload', 'url', 'drag_drop'
    source_url TEXT, -- If fetched from URL
    upload_method VARCHAR(50),
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    processed_at TIMESTAMP WITH TIME ZONE,
    deleted_at TIMESTAMP WITH TIME ZONE, -- Soft delete
    
    -- User and session
    uploaded_by VARCHAR(255),
    session_id UUID,
    
    -- Document metadata
    metadata JSONB DEFAULT '{}',
    extracted_text TEXT,
    language VARCHAR(10),
    page_count INTEGER,
    
    -- Search and categorization
    tags TEXT[],
    category VARCHAR(100),
    
    -- Status tracking
    status VARCHAR(50) DEFAULT 'pending', -- pending, processing, completed, failed, deleted
    error_message TEXT,
    
    -- Compliance and security
    classification VARCHAR(50), -- public, internal, confidential, restricted
    retention_date DATE,
    compliance_flags JSONB DEFAULT '{}'
);

-- Processing jobs for document batches
CREATE TABLE IF NOT EXISTS processing_jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Job identification
    job_name VARCHAR(255),
    job_type VARCHAR(50) NOT NULL, -- 'prompt', 'workflow', 'batch'
    
    -- Processing configuration
    processing_prompt TEXT,
    workflow_id UUID,
    workflow_name VARCHAR(255),
    configuration JSONB DEFAULT '{}',
    
    -- Storage configuration
    output_folder VARCHAR(255) NOT NULL,
    enable_semantic_search BOOLEAN DEFAULT false,
    
    -- Execution details
    started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    processing_time_ms INTEGER,
    
    -- Status tracking
    status VARCHAR(50) DEFAULT 'created', -- created, queued, processing, completed, failed, cancelled
    progress_percentage INTEGER DEFAULT 0,
    error_message TEXT,
    retry_count INTEGER DEFAULT 0,
    
    -- User and session
    created_by VARCHAR(255),
    session_id UUID,
    
    -- Results and metrics
    documents_processed INTEGER DEFAULT 0,
    documents_failed INTEGER DEFAULT 0,
    total_documents INTEGER DEFAULT 0,
    
    -- Audit and compliance
    audit_trail JSONB DEFAULT '[]',
    compliance_check_results JSONB DEFAULT '{}'
);

-- Link documents to processing jobs
CREATE TABLE IF NOT EXISTS job_documents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    job_id UUID NOT NULL REFERENCES processing_jobs(id) ON DELETE CASCADE,
    document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    
    -- Processing details
    processing_order INTEGER,
    status VARCHAR(50) DEFAULT 'pending',
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    
    -- Results
    output_document_id UUID REFERENCES documents(id),
    processing_result JSONB DEFAULT '{}',
    error_message TEXT,
    
    UNIQUE(job_id, document_id)
);

-- Saved workflows for reuse
CREATE TABLE IF NOT EXISTS workflows (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Workflow identification
    name VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(100),
    tags TEXT[],
    
    -- Workflow definition
    workflow_type VARCHAR(50) NOT NULL, -- 'n8n', 'custom', 'ai_generated'
    workflow_definition JSONB NOT NULL,
    configuration JSONB DEFAULT '{}',
    
    -- Source information
    created_from_prompt TEXT,
    source_job_id UUID REFERENCES processing_jobs(id),
    
    -- Versioning
    version INTEGER DEFAULT 1,
    parent_workflow_id UUID REFERENCES workflows(id),
    is_active BOOLEAN DEFAULT true,
    
    -- Usage tracking
    usage_count INTEGER DEFAULT 0,
    last_used_at TIMESTAMP WITH TIME ZONE,
    average_processing_time_ms INTEGER,
    success_rate DECIMAL(5,2),
    
    -- Metadata
    created_by VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Access control
    is_public BOOLEAN DEFAULT false,
    allowed_users TEXT[],
    required_permissions TEXT[]
);

-- Semantic search indices
CREATE TABLE IF NOT EXISTS search_indices (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    
    -- Vector storage reference
    vector_id VARCHAR(255), -- Qdrant vector ID
    collection_name VARCHAR(100) DEFAULT 'document-embeddings',
    
    -- Embedding metadata
    embedding_model VARCHAR(100),
    embedding_dimension INTEGER,
    chunk_index INTEGER,
    chunk_text TEXT,
    
    -- Search optimization
    keywords TEXT[],
    summary TEXT,
    
    -- Timestamps
    indexed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_accessed TIMESTAMP WITH TIME ZONE,
    access_count INTEGER DEFAULT 0
);

-- Comprehensive audit trail
CREATE TABLE IF NOT EXISTS audit_trail (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Event identification
    event_timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    event_type VARCHAR(100) NOT NULL,
    event_category VARCHAR(50), -- security, compliance, operation, access
    
    -- Resource references
    document_id UUID REFERENCES documents(id),
    job_id UUID REFERENCES processing_jobs(id),
    workflow_id UUID REFERENCES workflows(id),
    
    -- Actor information
    user_id VARCHAR(255),
    session_id UUID,
    ip_address INET,
    user_agent TEXT,
    
    -- Event details
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(50),
    resource_id VARCHAR(255),
    
    -- Change tracking
    old_values JSONB,
    new_values JSONB,
    
    -- Security and compliance
    risk_level VARCHAR(20), -- low, medium, high, critical
    compliance_impact BOOLEAN DEFAULT false,
    requires_review BOOLEAN DEFAULT false,
    
    -- Additional context
    metadata JSONB DEFAULT '{}',
    error_message TEXT,
    
    -- Retention
    retention_required BOOLEAN DEFAULT true,
    can_be_deleted_after DATE
);

-- User sessions for tracking
CREATE TABLE IF NOT EXISTS user_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_token VARCHAR(255) UNIQUE NOT NULL,
    
    -- User information
    user_id VARCHAR(255),
    user_email VARCHAR(255),
    user_role VARCHAR(50),
    
    -- Session details
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_activity TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE,
    is_active BOOLEAN DEFAULT true,
    
    -- Session metadata
    ip_address INET,
    user_agent TEXT,
    
    -- Permissions and limits
    permissions JSONB DEFAULT '{}',
    rate_limit_remaining INTEGER DEFAULT 100,
    
    -- Activity tracking
    documents_processed INTEGER DEFAULT 0,
    jobs_created INTEGER DEFAULT 0,
    storage_used_bytes BIGINT DEFAULT 0
);

-- Application configuration
CREATE TABLE IF NOT EXISTS app_config (
    id SERIAL PRIMARY KEY,
    config_key VARCHAR(255) UNIQUE NOT NULL,
    config_value JSONB NOT NULL,
    config_type VARCHAR(50) DEFAULT 'setting',
    
    -- Metadata
    description TEXT,
    is_active BOOLEAN DEFAULT true,
    is_sensitive BOOLEAN DEFAULT false,
    
    -- Versioning
    version INTEGER DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_by VARCHAR(255)
);

-- ============================================
-- INDEXES FOR PERFORMANCE
-- ============================================

-- Documents indexes
CREATE INDEX idx_documents_status ON documents(status);
CREATE INDEX idx_documents_created_at ON documents(created_at DESC);
CREATE INDEX idx_documents_file_hash ON documents(file_hash);
CREATE INDEX idx_documents_session ON documents(session_id);
CREATE INDEX idx_documents_classification ON documents(classification);
CREATE INDEX idx_documents_tags ON documents USING GIN(tags);
CREATE INDEX idx_documents_metadata ON documents USING GIN(metadata);

-- Processing jobs indexes
CREATE INDEX idx_jobs_status ON processing_jobs(status);
CREATE INDEX idx_jobs_created_at ON processing_jobs(started_at DESC);
CREATE INDEX idx_jobs_session ON processing_jobs(session_id);
CREATE INDEX idx_jobs_workflow ON processing_jobs(workflow_id);

-- Job documents indexes
CREATE INDEX idx_job_docs_job ON job_documents(job_id);
CREATE INDEX idx_job_docs_document ON job_documents(document_id);
CREATE INDEX idx_job_docs_status ON job_documents(status);

-- Workflows indexes
CREATE INDEX idx_workflows_name ON workflows(name);
CREATE INDEX idx_workflows_active ON workflows(is_active);
CREATE INDEX idx_workflows_tags ON workflows USING GIN(tags);
CREATE INDEX idx_workflows_usage ON workflows(usage_count DESC);

-- Search indices indexes
CREATE INDEX idx_search_document ON search_indices(document_id);
CREATE INDEX idx_search_keywords ON search_indices USING GIN(keywords);
CREATE INDEX idx_search_accessed ON search_indices(last_accessed DESC);

-- Audit trail indexes
CREATE INDEX idx_audit_timestamp ON audit_trail(event_timestamp DESC);
CREATE INDEX idx_audit_event_type ON audit_trail(event_type);
CREATE INDEX idx_audit_document ON audit_trail(document_id);
CREATE INDEX idx_audit_job ON audit_trail(job_id);
CREATE INDEX idx_audit_user ON audit_trail(user_id);
CREATE INDEX idx_audit_risk ON audit_trail(risk_level);
CREATE INDEX idx_audit_compliance ON audit_trail(compliance_impact);

-- Sessions indexes
CREATE INDEX idx_sessions_token ON user_sessions(session_token);
CREATE INDEX idx_sessions_user ON user_sessions(user_id);
CREATE INDEX idx_sessions_active ON user_sessions(is_active, expires_at);

-- ============================================
-- VIEWS FOR MONITORING
-- ============================================

-- Job status overview
CREATE OR REPLACE VIEW job_status_summary AS
SELECT 
    status,
    COUNT(*) as job_count,
    AVG(processing_time_ms) as avg_processing_time_ms,
    SUM(documents_processed) as total_documents_processed,
    SUM(documents_failed) as total_documents_failed
FROM processing_jobs
WHERE started_at > NOW() - INTERVAL '7 days'
GROUP BY status;

-- Document processing metrics
CREATE OR REPLACE VIEW document_metrics AS
SELECT 
    DATE_TRUNC('hour', created_at) as hour,
    COUNT(*) as documents_uploaded,
    AVG(file_size_bytes) as avg_file_size,
    COUNT(DISTINCT session_id) as unique_sessions,
    SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) as completed_count,
    SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed_count
FROM documents
GROUP BY DATE_TRUNC('hour', created_at)
ORDER BY hour DESC;

-- Workflow usage statistics
CREATE OR REPLACE VIEW workflow_usage_stats AS
SELECT 
    w.id,
    w.name,
    w.usage_count,
    w.success_rate,
    w.average_processing_time_ms,
    COUNT(DISTINCT pj.id) as recent_jobs,
    MAX(pj.started_at) as last_used
FROM workflows w
LEFT JOIN processing_jobs pj ON pj.workflow_id = w.id 
    AND pj.started_at > NOW() - INTERVAL '30 days'
WHERE w.is_active = true
GROUP BY w.id, w.name, w.usage_count, w.success_rate, w.average_processing_time_ms
ORDER BY w.usage_count DESC;

-- Security audit summary
CREATE OR REPLACE VIEW security_audit_summary AS
SELECT 
    DATE_TRUNC('day', event_timestamp) as day,
    event_category,
    risk_level,
    COUNT(*) as event_count,
    COUNT(DISTINCT user_id) as unique_users,
    SUM(CASE WHEN compliance_impact THEN 1 ELSE 0 END) as compliance_events,
    SUM(CASE WHEN requires_review THEN 1 ELSE 0 END) as pending_reviews
FROM audit_trail
WHERE event_timestamp > NOW() - INTERVAL '30 days'
GROUP BY DATE_TRUNC('day', event_timestamp), event_category, risk_level
ORDER BY day DESC, risk_level DESC;

-- ============================================
-- STORED PROCEDURES
-- ============================================

-- Function to update workflow statistics
CREATE OR REPLACE FUNCTION update_workflow_stats(workflow_id UUID)
RETURNS void AS $$
BEGIN
    UPDATE workflows w
    SET 
        usage_count = (
            SELECT COUNT(*) FROM processing_jobs 
            WHERE workflow_id = w.id
        ),
        last_used_at = (
            SELECT MAX(started_at) FROM processing_jobs 
            WHERE workflow_id = w.id
        ),
        average_processing_time_ms = (
            SELECT AVG(processing_time_ms) FROM processing_jobs 
            WHERE workflow_id = w.id AND status = 'completed'
        ),
        success_rate = (
            SELECT 
                CASE 
                    WHEN COUNT(*) > 0 THEN 
                        (SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END)::DECIMAL / COUNT(*)) * 100
                    ELSE 0
                END
            FROM processing_jobs 
            WHERE workflow_id = w.id
        ),
        updated_at = NOW()
    WHERE w.id = workflow_id;
END;
$$ LANGUAGE plpgsql;

-- Function to log audit events
CREATE OR REPLACE FUNCTION log_audit_event(
    p_event_type VARCHAR(100),
    p_action VARCHAR(100),
    p_user_id VARCHAR(255),
    p_resource_type VARCHAR(50),
    p_resource_id VARCHAR(255),
    p_metadata JSONB DEFAULT '{}'
)
RETURNS UUID AS $$
DECLARE
    audit_id UUID;
BEGIN
    INSERT INTO audit_trail (
        event_type, 
        action, 
        user_id, 
        resource_type, 
        resource_id, 
        metadata
    ) VALUES (
        p_event_type,
        p_action,
        p_user_id,
        p_resource_type,
        p_resource_id,
        p_metadata
    ) RETURNING id INTO audit_id;
    
    RETURN audit_id;
END;
$$ LANGUAGE plpgsql;

-- ============================================
-- TRIGGERS
-- ============================================

-- Update timestamps trigger
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_documents_updated_at BEFORE UPDATE ON documents
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_workflows_updated_at BEFORE UPDATE ON workflows
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_app_config_updated_at BEFORE UPDATE ON app_config
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================
-- PERMISSIONS
-- ============================================

-- Grant appropriate permissions (adjust based on your setup)
GRANT USAGE ON SCHEMA secure_doc_processing TO PUBLIC;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA secure_doc_processing TO PUBLIC;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA secure_doc_processing TO PUBLIC;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA secure_doc_processing TO PUBLIC;