-- Seed data for Secure Document Processing Pipeline
-- Provides initial configuration, sample workflows, and default settings

-- Set search path
SET search_path TO secure_doc_processing, public;

-- ============================================
-- APPLICATION CONFIGURATION
-- ============================================

INSERT INTO app_config (config_key, config_value, config_type, description, is_sensitive) VALUES
    -- System configuration
    ('system.app_name', '"Secure Document Processing Pipeline"'::jsonb, 'setting', 'Application name', false),
    ('system.version', '"1.0.0"'::jsonb, 'setting', 'Application version', false),
    ('system.environment', '"production"'::jsonb, 'setting', 'Deployment environment', false),
    
    -- Feature flags
    ('features.semantic_search', 'true'::jsonb, 'feature_flag', 'Enable semantic search functionality', false),
    ('features.ai_processing', 'true'::jsonb, 'feature_flag', 'Enable AI-powered document processing', false),
    ('features.workflow_conversion', 'true'::jsonb, 'feature_flag', 'Allow converting prompts to workflows', false),
    ('features.encryption', 'true'::jsonb, 'feature_flag', 'Enable document encryption', false),
    ('features.audit_trail', 'true'::jsonb, 'feature_flag', 'Enable comprehensive audit logging', false),
    ('features.compliance_checks', 'true'::jsonb, 'feature_flag', 'Enable compliance validation', false),
    
    -- Processing configuration
    ('processing.max_file_size_mb', '100'::jsonb, 'setting', 'Maximum file size for upload in MB', false),
    ('processing.supported_formats', '["pdf", "docx", "txt", "html", "jpg", "png", "xlsx", "csv"]'::jsonb, 'setting', 'Supported file formats', false),
    ('processing.timeout_seconds', '300'::jsonb, 'setting', 'Processing timeout in seconds', false),
    ('processing.max_concurrent_jobs', '10'::jsonb, 'setting', 'Maximum concurrent processing jobs', false),
    ('processing.retry_attempts', '3'::jsonb, 'setting', 'Number of retry attempts for failed jobs', false),
    
    -- Storage configuration
    ('storage.retention_days', '90'::jsonb, 'setting', 'Default document retention period in days', false),
    ('storage.encryption_algorithm', '"AES-256-GCM"'::jsonb, 'setting', 'Encryption algorithm for documents', false),
    ('storage.compression_enabled', 'true'::jsonb, 'setting', 'Enable document compression', false),
    ('storage.versioning_enabled', 'true'::jsonb, 'setting', 'Enable document versioning', false),
    
    -- AI configuration
    ('ai.default_model', '"llama3.1:8b"'::jsonb, 'setting', 'Default AI model for processing', false),
    ('ai.embedding_model', '"nomic-embed-text"'::jsonb, 'setting', 'Model for generating embeddings', false),
    ('ai.temperature', '0.7'::jsonb, 'setting', 'AI temperature setting', false),
    ('ai.max_tokens', '4096'::jsonb, 'setting', 'Maximum tokens for AI processing', false),
    
    -- Security configuration
    ('security.require_authentication', 'false'::jsonb, 'setting', 'Require user authentication', false),
    ('security.session_timeout_minutes', '30'::jsonb, 'setting', 'Session timeout in minutes', false),
    ('security.max_login_attempts', '5'::jsonb, 'setting', 'Maximum login attempts before lockout', false),
    ('security.audit_retention_days', '365'::jsonb, 'setting', 'Audit trail retention in days', false),
    
    -- Compliance configuration
    ('compliance.standards', '["GDPR", "HIPAA", "SOC2"]'::jsonb, 'setting', 'Supported compliance standards', false),
    ('compliance.pii_detection', 'true'::jsonb, 'setting', 'Enable PII detection in documents', false),
    ('compliance.redaction_enabled', 'true'::jsonb, 'setting', 'Enable automatic PII redaction', false),
    
    -- UI configuration
    ('ui.theme', '"light"'::jsonb, 'setting', 'Default UI theme', false),
    ('ui.items_per_page', '20'::jsonb, 'setting', 'Default items per page in lists', false),
    ('ui.enable_drag_drop', 'true'::jsonb, 'setting', 'Enable drag and drop uploads', false),
    ('ui.show_processing_preview', 'true'::jsonb, 'setting', 'Show document preview during processing', false);

-- ============================================
-- SAMPLE WORKFLOWS
-- ============================================

-- Text extraction workflow
INSERT INTO workflows (
    name, 
    description, 
    category, 
    tags, 
    workflow_type,
    workflow_definition,
    configuration,
    is_public,
    created_by
) VALUES (
    'Text Extraction',
    'Extract and structure text content from documents',
    'extraction',
    ARRAY['extraction', 'text', 'parsing'],
    'n8n',
    '{
        "nodes": [
            {
                "id": "extract-text",
                "type": "unstructured-io",
                "operation": "extract_text",
                "parameters": {
                    "strategy": "auto",
                    "include_metadata": true
                }
            }
        ]
    }'::jsonb,
    '{
        "output_format": "text",
        "preserve_formatting": true,
        "extract_tables": true
    }'::jsonb,
    true,
    'system'
);

-- Document summarization workflow
INSERT INTO workflows (
    name,
    description,
    category,
    tags,
    workflow_type,
    workflow_definition,
    configuration,
    is_public,
    created_by
) VALUES (
    'Document Summarization',
    'Generate concise summaries of document content',
    'analysis',
    ARRAY['summary', 'analysis', 'ai'],
    'n8n',
    '{
        "nodes": [
            {
                "id": "extract",
                "type": "unstructured-io",
                "operation": "extract_text"
            },
            {
                "id": "summarize",
                "type": "ollama",
                "operation": "generate",
                "parameters": {
                    "prompt": "Summarize the following document in 3-5 key points",
                    "model": "llama3.1:8b"
                }
            }
        ]
    }'::jsonb,
    '{
        "summary_length": "short",
        "include_keywords": true,
        "extract_entities": true
    }'::jsonb,
    true,
    'system'
);

-- PII redaction workflow
INSERT INTO workflows (
    name,
    description,
    category,
    tags,
    workflow_type,
    workflow_definition,
    configuration,
    is_public,
    created_by
) VALUES (
    'PII Redaction',
    'Detect and redact personally identifiable information',
    'compliance',
    ARRAY['compliance', 'pii', 'redaction', 'privacy'],
    'n8n',
    '{
        "nodes": [
            {
                "id": "extract",
                "type": "unstructured-io",
                "operation": "extract_text"
            },
            {
                "id": "detect-pii",
                "type": "custom",
                "operation": "detect_pii",
                "parameters": {
                    "entities": ["email", "phone", "ssn", "credit_card", "address"]
                }
            },
            {
                "id": "redact",
                "type": "custom",
                "operation": "redact_entities",
                "parameters": {
                    "redaction_type": "mask"
                }
            }
        ]
    }'::jsonb,
    '{
        "compliance_standard": "GDPR",
        "redaction_method": "mask",
        "generate_report": true
    }'::jsonb,
    true,
    'system'
);

-- Format conversion workflow
INSERT INTO workflows (
    name,
    description,
    category,
    tags,
    workflow_type,
    workflow_definition,
    configuration,
    is_public,
    created_by
) VALUES (
    'Format Conversion',
    'Convert documents between different formats',
    'conversion',
    ARRAY['conversion', 'format', 'transform'],
    'n8n',
    '{
        "nodes": [
            {
                "id": "convert",
                "type": "unstructured-io",
                "operation": "convert",
                "parameters": {
                    "target_format": "pdf",
                    "preserve_formatting": true
                }
            }
        ]
    }'::jsonb,
    '{
        "supported_outputs": ["pdf", "docx", "html", "markdown"],
        "quality": "high",
        "compress": false
    }'::jsonb,
    true,
    'system'
);

-- Data extraction workflow
INSERT INTO workflows (
    name,
    description,
    category,
    tags,
    workflow_type,
    workflow_definition,
    configuration,
    is_public,
    created_by
) VALUES (
    'Structured Data Extraction',
    'Extract structured data like tables, forms, and key-value pairs',
    'extraction',
    ARRAY['extraction', 'structured', 'tables', 'forms'],
    'n8n',
    '{
        "nodes": [
            {
                "id": "extract-structured",
                "type": "unstructured-io",
                "operation": "extract_structured",
                "parameters": {
                    "extract_tables": true,
                    "extract_forms": true,
                    "extract_metadata": true
                }
            },
            {
                "id": "format-output",
                "type": "custom",
                "operation": "format_json",
                "parameters": {
                    "schema": "auto-detect"
                }
            }
        ]
    }'::jsonb,
    '{
        "output_format": "json",
        "include_confidence_scores": true,
        "validate_data": true
    }'::jsonb,
    true,
    'system'
);

-- Translation workflow
INSERT INTO workflows (
    name,
    description,
    category,
    tags,
    workflow_type,
    workflow_definition,
    configuration,
    is_public,
    created_by
) VALUES (
    'Document Translation',
    'Translate documents to different languages',
    'translation',
    ARRAY['translation', 'language', 'ai'],
    'n8n',
    '{
        "nodes": [
            {
                "id": "extract",
                "type": "unstructured-io",
                "operation": "extract_text"
            },
            {
                "id": "translate",
                "type": "ollama",
                "operation": "generate",
                "parameters": {
                    "prompt_template": "Translate the following text to {target_language}. Preserve formatting and technical terms.",
                    "model": "llama3.1:8b"
                }
            }
        ]
    }'::jsonb,
    '{
        "supported_languages": ["Spanish", "French", "German", "Italian", "Portuguese", "Chinese", "Japanese"],
        "preserve_formatting": true,
        "technical_glossary": true
    }'::jsonb,
    true,
    'system'
);

-- OCR workflow for images
INSERT INTO workflows (
    name,
    description,
    category,
    tags,
    workflow_type,
    workflow_definition,
    configuration,
    is_public,
    created_by
) VALUES (
    'OCR Text Extraction',
    'Extract text from images and scanned documents',
    'extraction',
    ARRAY['ocr', 'images', 'scanning', 'extraction'],
    'n8n',
    '{
        "nodes": [
            {
                "id": "ocr",
                "type": "unstructured-io",
                "operation": "ocr",
                "parameters": {
                    "languages": ["eng"],
                    "enhance_image": true
                }
            },
            {
                "id": "post-process",
                "type": "custom",
                "operation": "clean_text",
                "parameters": {
                    "fix_spelling": true,
                    "remove_noise": true
                }
            }
        ]
    }'::jsonb,
    '{
        "image_preprocessing": true,
        "confidence_threshold": 0.8,
        "output_format": "text"
    }'::jsonb,
    true,
    'system'
);

-- Compliance check workflow
INSERT INTO workflows (
    name,
    description,
    category,
    tags,
    workflow_type,
    workflow_definition,
    configuration,
    is_public,
    created_by
) VALUES (
    'Compliance Validation',
    'Check documents for compliance with various standards',
    'compliance',
    ARRAY['compliance', 'validation', 'audit', 'security'],
    'n8n',
    '{
        "nodes": [
            {
                "id": "extract",
                "type": "unstructured-io",
                "operation": "extract_text"
            },
            {
                "id": "check-compliance",
                "type": "custom",
                "operation": "compliance_check",
                "parameters": {
                    "standards": ["GDPR", "HIPAA", "SOC2"],
                    "generate_report": true
                }
            }
        ]
    }'::jsonb,
    '{
        "strict_mode": true,
        "auto_remediation": false,
        "report_format": "pdf"
    }'::jsonb,
    true,
    'system'
);

-- ============================================
-- SAMPLE USER SESSION
-- ============================================

INSERT INTO user_sessions (
    session_token,
    user_id,
    user_email,
    user_role,
    expires_at,
    permissions
) VALUES (
    'demo-session-token-001',
    'demo-user',
    'demo@example.com',
    'user',
    NOW() + INTERVAL '7 days',
    '{
        "can_upload": true,
        "can_process": true,
        "can_create_workflows": true,
        "can_use_ai": true,
        "max_file_size_mb": 100,
        "allowed_formats": ["pdf", "docx", "txt", "jpg", "png"]
    }'::jsonb
);

-- ============================================
-- SAMPLE AUDIT ENTRIES
-- ============================================

INSERT INTO audit_trail (
    event_type,
    event_category,
    action,
    user_id,
    resource_type,
    resource_id,
    risk_level,
    metadata
) VALUES
    ('system_startup', 'operation', 'initialize', 'system', 'application', 'secure-doc-processing', 'low', 
     '{"version": "1.0.0", "environment": "production"}'::jsonb),
    
    ('config_loaded', 'operation', 'load', 'system', 'configuration', 'app_config', 'low',
     '{"config_count": 30, "feature_flags": 6}'::jsonb),
    
    ('workflows_initialized', 'operation', 'create', 'system', 'workflow', 'default-workflows', 'low',
     '{"workflow_count": 8, "categories": ["extraction", "analysis", "compliance", "conversion"]}'::jsonb);

-- ============================================
-- INITIAL STATISTICS UPDATE
-- ============================================

-- Update workflow statistics for initial workflows
UPDATE workflows SET 
    usage_count = 0,
    average_processing_time_ms = 2000,
    success_rate = 100.0
WHERE created_by = 'system';

-- ============================================
-- VERIFICATION QUERIES
-- ============================================

-- Display seeded data summary
SELECT 'Configuration entries:' as info, COUNT(*) as count FROM app_config
UNION ALL
SELECT 'Sample workflows:', COUNT(*) FROM workflows
UNION ALL
SELECT 'Active sessions:', COUNT(*) FROM user_sessions WHERE is_active = true
UNION ALL
SELECT 'Audit entries:', COUNT(*) FROM audit_trail
ORDER BY info;