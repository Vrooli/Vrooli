-- Resource Improver Seed Data
-- Initial data for testing and bootstrapping the resource improvement system

-- Available resources in the Vrooli ecosystem (discovered automatically)
INSERT INTO available_resources (name, display_name, description, path, current_compliance_score, current_health_score, improvement_friendly, complexity_level) VALUES
('postgres', 'PostgreSQL', 'Relational database server', 'resources/postgres', 85.0, 92.0, true, 'medium'),
('redis', 'Redis', 'In-memory key-value store', 'resources/redis', 75.0, 88.0, true, 'simple'),
('qdrant', 'Qdrant Vector DB', 'Vector database for embeddings', 'resources/qdrant', 65.0, 85.0, true, 'medium'),
('ollama', 'Ollama', 'Local LLM inference server', 'resources/ollama', 70.0, 80.0, true, 'complex'),
('minio', 'MinIO', 'S3-compatible object storage', 'resources/minio', 80.0, 90.0, true, 'medium'),
('questdb', 'QuestDB', 'Time-series database', 'resources/questdb', 55.0, 70.0, true, 'medium'),
('claude-code', 'Claude Code', 'AI development assistance', 'resources/claude-code', 90.0, 95.0, false, 'complex'),
('browserless', 'Browserless', 'Headless browser automation', 'resources/browserless', 45.0, 65.0, true, 'medium'),
('judge0', 'Judge0', 'Code execution engine', 'resources/judge0', 50.0, 70.0, true, 'complex'),
('vault', 'HashiCorp Vault', 'Secrets management', 'resources/vault', 85.0, 90.0, true, 'complex'),
('litellm', 'LiteLLM', 'LLM proxy and gateway', 'resources/litellm', 60.0, 75.0, true, 'medium'),
('comfyui', 'ComfyUI', 'Node-based UI for Stable Diffusion', 'resources/comfyui', 40.0, 60.0, true, 'complex')
ON CONFLICT (name) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    description = EXCLUDED.description,
    path = EXCLUDED.path,
    current_compliance_score = EXCLUDED.current_compliance_score,
    current_health_score = EXCLUDED.current_health_score,
    improvement_friendly = EXCLUDED.improvement_friendly,
    complexity_level = EXCLUDED.complexity_level,
    updated_at = CURRENT_TIMESTAMP;

-- Initial resource metrics (will be updated by analysis)
INSERT INTO resource_metrics (resource_name, v2_compliance_score, health_reliability, cli_coverage, doc_completeness, status, recommendations) VALUES
('postgres', 85.0, 92.0, 80.0, 75.0, 'good', '["Add timeout handling to health checks", "Improve CLI help documentation"]'),
('redis', 75.0, 88.0, 70.0, 60.0, 'needs-improvement', '["Implement v2.0 lib directory structure", "Add content management commands", "Create comprehensive README"]'),
('qdrant', 65.0, 85.0, 60.0, 55.0, 'needs-improvement', '["Add missing lifecycle hooks", "Implement robust health checking", "Standardize CLI commands"]'),
('ollama', 70.0, 80.0, 75.0, 65.0, 'needs-improvement', '["Add model validation checks", "Improve startup grace period", "Document environment variables"]'),
('minio', 80.0, 90.0, 85.0, 80.0, 'good', '["Add bucket lifecycle management", "Improve monitoring endpoints"]'),
('questdb', 55.0, 70.0, 50.0, 45.0, 'poor', '["Critical: Implement v2.0 contract compliance", "Add health check timeouts", "Create CLI interface"]'),
('browserless', 45.0, 65.0, 40.0, 35.0, 'poor', '["Critical: Add lib directory structure", "Implement proper health checking", "Create standard CLI commands"]'),
('judge0', 50.0, 70.0, 45.0, 40.0, 'poor', '["Critical: Implement v2.0 compliance", "Add security validations", "Improve error handling"]')
ON CONFLICT (resource_name) DO UPDATE SET
    v2_compliance_score = EXCLUDED.v2_compliance_score,
    health_reliability = EXCLUDED.health_reliability,
    cli_coverage = EXCLUDED.cli_coverage,
    doc_completeness = EXCLUDED.doc_completeness,
    status = EXCLUDED.status,
    recommendations = EXCLUDED.recommendations,
    updated_at = CURRENT_TIMESTAMP;

-- Sample resource reports (typical issues found)
INSERT INTO resource_reports (resource_name, issue_type, description, context, severity) VALUES
('redis', 'v2-compliance', 'Missing lib/ directory structure required for v2.0 contract', '{"missing_files": ["lib/common.sh", "lib/health.sh", "lib/lifecycle.sh"]}', 'high'),
('qdrant', 'health-check', 'Health check lacks timeout and retry logic', '{"current_timeout": null, "retry_attempts": 1}', 'medium'),
('ollama', 'cli-enhancement', 'CLI missing standard content management commands', '{"missing_commands": ["content list", "content add", "content remove"]}', 'medium'),
('questdb', 'documentation', 'README lacks usage examples and troubleshooting section', '{"missing_sections": ["Usage Examples", "Troubleshooting", "Environment Variables"]}', 'low'),
('browserless', 'v2-compliance', 'Critical v2.0 contract violations - no lib directory, no health check', '{"compliance_score": 15, "missing_components": ["lib/", "health check", "lifecycle hooks"]}', 'critical'),
('judge0', 'cli-enhancement', 'No CLI interface available for resource management', '{"cli_exists": false, "management_method": "docker_only"}', 'high');

-- Sample improvement queue items (pending improvements)
INSERT INTO improvement_queue (id, title, description, type, target, priority, estimates, status, created_by) VALUES
('redis-v2-compliance-001', 'Implement v2.0 contract for Redis', 'Add missing lib/ directory structure with required scripts for v2.0 compliance', 'v2-compliance', 'redis', 'high', '{"impact": 8, "urgency": "high", "success_prob": 0.9, "resource_cost": "moderate"}', 'pending', 'system'),
('qdrant-health-timeout-002', 'Add timeout and retry logic to Qdrant health checks', 'Improve health check reliability with proper timeouts and exponential backoff', 'health-check', 'qdrant', 'medium', '{"impact": 6, "urgency": "medium", "success_prob": 0.8, "resource_cost": "low"}', 'pending', 'system'),
('ollama-cli-enhancement-003', 'Add content management commands to Ollama CLI', 'Implement standard content list, add, and remove commands for model management', 'cli-enhancement', 'ollama', 'medium', '{"impact": 7, "urgency": "medium", "success_prob": 0.85, "resource_cost": "moderate"}', 'pending', 'system'),
('browserless-critical-v2-004', 'Critical v2.0 compliance fixes for Browserless', 'Complete v2.0 contract implementation including lib/ structure and health checks', 'v2-compliance', 'browserless', 'critical', '{"impact": 9, "urgency": "critical", "success_prob": 0.7, "resource_cost": "heavy"}', 'pending', 'system'),
('questdb-documentation-005', 'Improve QuestDB documentation completeness', 'Add usage examples, environment variables, and troubleshooting guide to README', 'documentation', 'questdb', 'low', '{"impact": 4, "urgency": "low", "success_prob": 0.95, "resource_cost": "low"}', 'pending', 'system');

-- Sample improvement memory entries (learning from past improvements)
INSERT INTO improvement_memory (queue_item_id, resource_name, improvement_type, success, changes, test_results, patterns) VALUES
('postgres-health-fix-historic', 'postgres', 'health-check', true, 'Modified lib/health.sh to add timeout and retry logic', 'All health checks passed', '{"pattern": "timeout_retry", "effective_timeout": "5s", "max_retries": 3, "success_indicators": ["curl response", "service status"]}'),
('minio-v2-upgrade-historic', 'minio', 'v2-compliance', true, 'Added lib/lifecycle.sh, lib/content.sh, updated service.json', 'v2.0 compliance increased to 95%', '{"pattern": "v2_compliance_boost", "key_files": ["lib/lifecycle.sh", "lib/content.sh", "service.json"], "compliance_gain": 40}'),
('redis-cli-standardization-historic', 'redis', 'cli-enhancement', true, 'Added help command, JSON output support, verbose mode', 'CLI coverage increased to 85%', '{"pattern": "cli_standardization", "standard_commands": ["help", "status", "logs"], "output_formats": ["json", "plain"]}');

-- Update last_analyzed_at timestamps to be realistic
UPDATE resource_metrics SET last_analyzed_at = CURRENT_TIMESTAMP - INTERVAL '1 day' WHERE resource_name IN ('postgres', 'redis', 'minio');
UPDATE resource_metrics SET last_analyzed_at = CURRENT_TIMESTAMP - INTERVAL '3 days' WHERE resource_name IN ('qdrant', 'ollama');
UPDATE resource_metrics SET last_analyzed_at = CURRENT_TIMESTAMP - INTERVAL '7 days' WHERE resource_name IN ('questdb', 'browserless', 'judge0');