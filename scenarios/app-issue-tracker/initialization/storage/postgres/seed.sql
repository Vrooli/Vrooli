-- Seed data for Issue Tracker
-- Initial agent prompts and sample apps

-- Insert sample apps (these represent different Vrooli components and generated apps)
INSERT INTO apps (id, name, display_name, scenario_name, type, api_token, environment, status) VALUES 
    ('a0000001-0000-0000-0000-000000000001', 'vrooli-core', 'Vrooli Core Platform', NULL, 'vrooli-core', 'token_vrooli_core_' || gen_random_uuid(), 'production', 'active'),
    ('a0000001-0000-0000-0000-000000000002', 'research-assistant', 'Research Assistant App', 'research-assistant', 'scenario', 'token_research_' || gen_random_uuid(), 'production', 'active'),
    ('a0000001-0000-0000-0000-000000000003', 'analytics-dashboard', 'Analytics Dashboard', 'analytics-dashboard', 'scenario', 'token_analytics_' || gen_random_uuid(), 'production', 'active'),
    ('a0000001-0000-0000-0000-000000000004', 'document-processor', 'Document Processing System', 'secure-document-processing', 'scenario', 'token_docproc_' || gen_random_uuid(), 'production', 'active'),
    ('a0000001-0000-0000-0000-000000000005', 'image-generator', 'AI Image Generation Pipeline', 'image-generation-pipeline', 'scenario', 'token_imagegen_' || gen_random_uuid(), 'production', 'active');

-- Insert unified agent template
INSERT INTO agents (id, name, display_name, description, system_prompt, user_prompt_template, capabilities, max_tokens, temperature, model, is_active) VALUES
    (
        'b0000001-0000-0000-0000-000000000001',
        'unified-resolver',
        'Unified Issue Resolver',
        'Single-pass agent that triages, investigates, and drafts remediation plans for issues.',
        'You are an elite software engineer who performs end-to-end incident response:
1. Triage the issue and summarise current impact
2. Investigate logs, code, and configuration to uncover the root cause
3. Propose code-level fixes with clear implementation steps
4. Design automated and manual tests to validate the remediation
5. Recommend rollback strategies and post-fix monitoring

Always output structured, actionable findings with confidence scores.',
        'Investigate and resolve the following issue:

Issue Title: {{issue_title}}
Issue Description: {{issue_description}}
Issue Type: {{issue_type}}
Priority: {{issue_priority}}
App: {{app_name}}

Error Message: {{error_message}}
Stack Trace:
{{stack_trace}}

Affected Files: {{affected_files}}

Produce:
1. Investigation summary
2. Root cause analysis
3. Suggested remediation
4. Implementation steps
5. Test plan
6. Rollback plan
7. Confidence score',
        '{triage,investigate,fix,test}',
        8192,
        0.2,
        'claude-3-opus-20240229',
        true
    );

-- Insert sample pattern groups
INSERT INTO pattern_groups (id, name, description, pattern_type, common_keywords, total_issues) VALUES
    ('c0000001-0000-0000-0000-000000000001', 'Authentication Errors', 'Issues related to authentication and authorization', 'error_pattern', '{auth,login,token,permission,403,401}', 0),
    ('c0000001-0000-0000-0000-000000000002', 'Database Connection Issues', 'Problems with database connectivity and queries', 'error_pattern', '{database,connection,postgres,timeout,ECONNREFUSED}', 0),
    ('c0000001-0000-0000-0000-000000000003', 'Memory Leaks', 'Memory usage and leak patterns', 'performance_pattern', '{memory,leak,heap,garbage,OOM}', 0),
    ('c0000001-0000-0000-0000-000000000004', 'API Rate Limiting', 'Rate limit and throttling issues', 'behavior_pattern', '{rate,limit,throttle,429,quota}', 0);

-- Create a function to generate API tokens for apps
CREATE OR REPLACE FUNCTION generate_api_token(app_name VARCHAR)
RETURNS VARCHAR AS $$
BEGIN
    RETURN 'tk_' || replace(lower(app_name), ' ', '_') || '_' || substr(md5(random()::text), 1, 24);
END;
$$ LANGUAGE plpgsql;

-- Create initial metrics view
CREATE OR REPLACE VIEW issue_metrics AS
SELECT 
    a.name as app_name,
    COUNT(i.id) as total_issues,
    COUNT(CASE WHEN i.status IN ('open', 'active') THEN 1 END) as active_issues,
    COUNT(CASE WHEN i.status = 'completed' THEN 1 END) as completed_issues,
    COUNT(CASE WHEN i.priority = 'critical' THEN 1 END) as critical_issues,
    AVG(EXTRACT(EPOCH FROM (i.resolved_at - i.created_at))/3600)::DECIMAL(10,2) as avg_resolution_hours,
    COUNT(DISTINCT i.pattern_group_id) as pattern_groups
FROM apps a
LEFT JOIN issues i ON a.id = i.app_id
GROUP BY a.id, a.name;

-- Create agent performance view
CREATE OR REPLACE VIEW agent_performance AS
SELECT 
    ag.name as agent_name,
    ag.display_name,
    ag.total_runs,
    ag.successful_runs,
    CASE 
        WHEN ag.total_runs > 0 
        THEN (ag.successful_runs::DECIMAL / ag.total_runs * 100)::DECIMAL(5,2)
        ELSE 0 
    END as success_rate_percent,
    ag.avg_completion_time_seconds,
    ag.total_tokens_used,
    ag.total_cost_usd,
    COUNT(DISTINCT ar.issue_id) as unique_issues_handled
FROM agents ag
LEFT JOIN agent_runs ar ON ag.id = ar.agent_id
GROUP BY ag.id;

-- Sample data for testing
-- Insert a test issue to verify the system
INSERT INTO issues (app_id, title, description, type, priority, status, error_message, stack_trace) 
SELECT 
    'a0000001-0000-0000-0000-000000000001',
    'Sample: Authentication timeout in user service',
    'Users are experiencing intermittent authentication failures during peak hours. The auth service times out after 30 seconds.',
    'bug',
    'high',
    'open',
    'Error: Authentication service timeout after 30000ms',
    E'Error: Authentication service timeout after 30000ms\n    at AuthService.authenticate (/app/src/services/auth.js:45:11)\n    at async UserController.login (/app/src/controllers/user.js:23:20)\n    at async /app/src/middleware/errorHandler.js:15:5'
WHERE NOT EXISTS (
    SELECT 1 FROM issues WHERE title = 'Sample: Authentication timeout in user service'
);

-- Grant necessary permissions (adjust based on your setup)
GRANT ALL ON ALL TABLES IN SCHEMA public TO issue_tracker_user;
GRANT ALL ON ALL SEQUENCES IN SCHEMA public TO issue_tracker_user;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO issue_tracker_user;
