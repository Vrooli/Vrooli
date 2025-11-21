-- Document Manager Seed Data
-- Sample data for testing and demonstration

-- Sample applications
INSERT INTO applications (id, name, repository_url, documentation_path, health_score, last_scan_result, notification_settings) VALUES 
('550e8400-e29b-41d4-a716-446655440001', 'Vrooli Platform', 'https://github.com/VrooliCorp/vrooli', '/docs', 0.85, '{"last_scan": "2024-01-15T10:00:00Z", "issues_found": 12, "improvements_suggested": 8}', '{"email": true, "slack": true, "severity_threshold": "medium"}'),
('550e8400-e29b-41d4-a716-446655440002', 'API Gateway Service', 'https://github.com/company/api-gateway', '/documentation', 0.72, '{"last_scan": "2024-01-14T15:30:00Z", "issues_found": 18, "improvements_suggested": 15}', '{"email": false, "slack": true, "severity_threshold": "high"}'),
('550e8400-e29b-41d4-a716-446655440003', 'Mobile App SDK', 'https://github.com/company/mobile-sdk', '/docs', 0.91, '{"last_scan": "2024-01-16T08:45:00Z", "issues_found": 5, "improvements_suggested": 3}', '{"email": true, "slack": false, "severity_threshold": "low"}');

-- Sample agent templates (system templates)
INSERT INTO agent_templates (id, name, description, type, default_config, default_schedule, is_system_template) VALUES
('660e8400-e29b-41d4-a716-446655440001', 'Basic Drift Detector', 'Detects when documentation diverges from code implementation', 'drift_detector', '{"scan_patterns": ["*.md", "*.mdx", "*.rst"], "code_patterns": ["*.ts", "*.js", "*.py"], "drift_threshold": 0.3, "semantic_analysis": true}', '0 */6 * * *', true),
('660e8400-e29b-41d4-a716-446655440002', 'Advanced Link Validator', 'Comprehensive link validation with retry logic', 'link_validator', '{"check_external": true, "timeout_ms": 5000, "retry_count": 3, "ignore_patterns": ["localhost", "example.com"], "check_anchors": true}', '0 2 * * *', true),
('660e8400-e29b-41d4-a716-446655440003', 'Documentation Organizer', 'Analyzes and suggests better document structure', 'organization_analyzer', '{"max_depth": 5, "ideal_sections": ["Overview", "Installation", "Usage", "API", "Examples", "FAQ"], "check_navigation": true, "suggest_toc": true}', '0 0 * * 1', true),
('660e8400-e29b-41d4-a716-446655440004', 'Coverage Analyzer', 'Identifies undocumented code and features', 'coverage_checker', '{"min_coverage": 80, "check_functions": true, "check_classes": true, "check_modules": true, "ignore_private": true, "generate_stubs": false}', '0 0 * * *', true);

-- Sample agents for applications
INSERT INTO agents (id, name, type, application_id, config, schedule_cron, auto_apply_threshold, last_performance_score, enabled, last_run, next_run) VALUES
('770e8400-e29b-41d4-a716-446655440001', 'Vrooli Docs Drift Monitor', 'drift_detector', '550e8400-e29b-41d4-a716-446655440001', '{"scan_patterns": ["docs/**/*.md"], "code_patterns": ["packages/**/*.ts"], "drift_threshold": 0.25}', '0 */4 * * *', 0.0, 0.87, true, '2024-01-15T12:00:00Z', '2024-01-15T16:00:00Z'),
('770e8400-e29b-41d4-a716-446655440002', 'Vrooli Link Checker', 'link_validator', '550e8400-e29b-41d4-a716-446655440001', '{"check_external": true, "timeout_ms": 3000, "retry_count": 2}', '0 1 * * *', 0.8, 0.92, true, '2024-01-15T01:00:00Z', '2024-01-16T01:00:00Z'),
('770e8400-e29b-41d4-a716-446655440003', 'API Gateway Coverage Monitor', 'coverage_checker', '550e8400-e29b-41d4-a716-446655440002', '{"min_coverage": 75, "check_functions": true, "ignore_private": true}', '0 3 * * *', 0.0, 0.74, true, '2024-01-14T03:00:00Z', '2024-01-15T03:00:00Z'),
('770e8400-e29b-41d4-a716-446655440004', 'Mobile SDK Meta Optimizer', 'meta_optimizer', '550e8400-e29b-41d4-a716-446655440003', '{"analyze_performance": true, "suggest_new_agents": true, "performance_window_days": 14}', '0 0 * * 0', 0.0, 0.95, true, '2024-01-14T00:00:00Z', '2024-01-21T00:00:00Z');

-- Sample app settings
INSERT INTO app_settings (application_id, auto_apply_enabled, auto_apply_max_severity, notification_channels, scan_frequency_hours, retention_days, custom_rules) VALUES
('550e8400-e29b-41d4-a716-446655440001', false, 'low', '["slack", "email"]', 6, 90, '{"ignore_generated_files": true, "custom_severity_rules": {"broken_internal_link": "high"}}'),
('550e8400-e29b-41d4-a716-446655440002', true, 'low', '["slack"]', 12, 60, '{"auto_fix_typos": true, "suggest_missing_examples": true}'),
('550e8400-e29b-41d4-a716-446655440003', false, 'medium', '["email"]', 24, 120, '{"require_changelog_updates": true}');

-- Sample improvement queue items
INSERT INTO improvement_queue (id, agent_id, application_id, type, severity, title, description, suggested_fix, user_feedback, status, created_at, reviewed_at, reviewed_by, review_notes) VALUES
('880e8400-e29b-41d4-a716-446655440001', '770e8400-e29b-41d4-a716-446655440001', '550e8400-e29b-41d4-a716-446655440001', 'outdated_content', 'medium', 'API documentation out of sync with implementation', 'The authentication section in docs/api/auth.md describes endpoints that no longer exist in the current codebase.', '{"files": ["docs/api/auth.md"], "suggested_changes": [{"line": 45, "old": "POST /auth/login", "new": "POST /api/v2/auth/signin"}], "confidence": 0.89}', null, 'pending', '2024-01-15T09:30:00Z', null, null, null),
('880e8400-e29b-41d4-a716-446655440002', '770e8400-e29b-41d4-a716-446655440002', '550e8400-e29b-41d4-a716-446655440001', 'broken_link', 'high', 'Broken link to deployment guide', 'Link to deployment guide returns 404: https://docs.vrooli.com/deploy/kubernetes', '{"files": ["docs/README.md"], "suggested_changes": [{"line": 23, "old": "https://docs.vrooli.com/deploy/kubernetes", "new": "https://docs.vrooli.com/deployment/k8s-guide"}], "confidence": 0.95}', 'This looks correct, please apply', 'approved', '2024-01-15T10:15:00Z', '2024-01-15T14:22:00Z', 'john.doe', 'Verified the new URL exists and is correct'),
('880e8400-e29b-41d4-a716-446655440003', '770e8400-e29b-41d4-a716-446655440003', '550e8400-e29b-41d4-a716-446655440002', 'missing_documentation', 'medium', 'Undocumented API endpoints detected', 'Found 8 API endpoints in the codebase that are not documented in the API reference.', '{"endpoints": ["/api/v1/health", "/api/v1/metrics", "/api/v1/cache/clear"], "suggested_location": "docs/api-reference.md", "template": "standard_endpoint_template"}', null, 'pending', '2024-01-14T16:45:00Z', null, null, null),
('880e8400-e29b-41d4-a716-446655440004', '770e8400-e29b-41d4-a716-446655440004', '550e8400-e29b-41d4-a716-446655440003', 'agent_optimization', 'low', 'Link validator performance optimization', 'The link validator agent could be optimized by caching DNS lookups and running checks in parallel batches.', '{"optimization_type": "performance", "estimated_improvement": "40% faster execution", "changes": [{"file": "agent_config", "setting": "parallel_batch_size", "new_value": 10}]}', null, 'revision_requested', '2024-01-14T08:00:00Z', '2024-01-14T12:30:00Z', 'jane.smith', 'Please provide more specific implementation details');

-- Sample action history
INSERT INTO action_history (queue_item_id, agent_id, action_type, action_data, result, execution_time_ms, created_at) VALUES
('880e8400-e29b-41d4-a716-446655440002', '770e8400-e29b-41d4-a716-446655440002', 'apply_fix', '{"file": "docs/README.md", "changes_applied": 1}', 'success', 1250, '2024-01-15T14:25:00Z'),
('880e8400-e29b-41d4-a716-446655440001', '770e8400-e29b-41d4-a716-446655440001', 'analyze_drift', '{"files_scanned": 45, "issues_found": 3}', 'success', 15800, '2024-01-15T09:30:00Z');

-- Sample agent metrics
INSERT INTO agent_metrics (agent_id, metric_date, runs_count, suggestions_count, approved_count, denied_count, revision_count, avg_execution_time_ms, error_count) VALUES
('770e8400-e29b-41d4-a716-446655440001', '2024-01-14', 4, 12, 8, 2, 2, 14500, 0),
('770e8400-e29b-41d4-a716-446655440002', '2024-01-14', 1, 3, 2, 0, 1, 8200, 0),
('770e8400-e29b-41d4-a716-446655440003', '2024-01-14', 1, 8, 0, 0, 0, 22100, 1),
('770e8400-e29b-41d4-a716-446655440001', '2024-01-15', 2, 5, 3, 1, 1, 13200, 0);

-- Sample documentation coverage
INSERT INTO documentation_coverage (application_id, scan_date, total_files, documented_files, coverage_percentage, missing_docs, outdated_docs, quality_score) VALUES
('550e8400-e29b-41d4-a716-446655440001', '2024-01-15T10:00:00Z', 127, 108, 85.04, '["utils/helpers.ts", "components/advanced/*"]', '["docs/api/auth.md", "docs/getting-started.md"]', 7.8),
('550e8400-e29b-41d4-a716-446655440002', '2024-01-14T15:30:00Z', 89, 64, 71.91, '["controllers/admin/*", "middleware/security.js"]', '["README.md", "docs/deployment/README.md"]', 6.2),
('550e8400-e29b-41d4-a716-446655440003', '2024-01-16T08:45:00Z', 156, 142, 91.03, '["internal/debug.ts"]', '[]', 8.9);

-- Sample notification preferences
INSERT INTO notification_preferences (user_id, application_id, channel, severity_threshold, enabled, schedule_window_start, schedule_window_end, timezone) VALUES
('john.doe', '550e8400-e29b-41d4-a716-446655440001', 'slack', 'medium', true, '09:00', '17:00', 'UTC'),
('john.doe', '550e8400-e29b-41d4-a716-446655440001', 'email', 'high', true, null, null, 'UTC'),
('jane.smith', '550e8400-e29b-41d4-a716-446655440002', 'slack', 'low', true, '08:00', '18:00', 'America/New_York'),
('admin', null, 'slack', 'critical', true, null, null, 'UTC');

-- Sample improvement feedback
INSERT INTO improvement_feedback (improvement_id, feedback_type, effectiveness_rating, feedback_text, would_revert, additional_suggestions, created_by, created_at) VALUES
('880e8400-e29b-41d4-a716-446655440002', 'positive', 5, 'Perfect fix, the link now works correctly and users can access the deployment guide.', false, 'Consider adding link validation as a pre-commit hook to catch these earlier.', 'john.doe', '2024-01-15T16:00:00Z');

-- Sample audit log entries
INSERT INTO audit_log (entity_type, entity_id, action, old_values, new_values, performed_by, ip_address, created_at) VALUES
('improvement_queue', '880e8400-e29b-41d4-a716-446655440002', 'status_change', '{"status": "pending"}', '{"status": "approved", "reviewed_by": "john.doe", "review_notes": "Verified the new URL exists and is correct"}', 'john.doe', '192.168.1.100', '2024-01-15T14:22:00Z'),
('agent', '770e8400-e29b-41d4-a716-446655440001', 'config_update', '{"drift_threshold": 0.3}', '{"drift_threshold": 0.25}', 'admin', '192.168.1.50', '2024-01-15T11:30:00Z'),
('application', '550e8400-e29b-41d4-a716-446655440001', 'health_score_update', '{"health_score": 0.82}', '{"health_score": 0.85}', 'system', null, '2024-01-15T12:00:00Z');
