-- Seed data for Issue Tracker
-- Initial agent prompts and sample apps

-- Insert sample apps (these represent different Vrooli components and generated apps)
INSERT INTO apps (id, name, display_name, scenario_name, type, api_token, environment, status) VALUES 
    ('a0000001-0000-0000-0000-000000000001', 'vrooli-core', 'Vrooli Core Platform', NULL, 'vrooli-core', 'token_vrooli_core_' || gen_random_uuid(), 'production', 'active'),
    ('a0000001-0000-0000-0000-000000000002', 'research-assistant', 'Research Assistant App', 'research-assistant', 'scenario', 'token_research_' || gen_random_uuid(), 'production', 'active'),
    ('a0000001-0000-0000-0000-000000000003', 'analytics-dashboard', 'Analytics Dashboard', 'analytics-dashboard', 'scenario', 'token_analytics_' || gen_random_uuid(), 'production', 'active'),
    ('a0000001-0000-0000-0000-000000000004', 'document-processor', 'Document Processing System', 'secure-document-processing', 'scenario', 'token_docproc_' || gen_random_uuid(), 'production', 'active'),
    ('a0000001-0000-0000-0000-000000000005', 'image-generator', 'AI Image Generation Pipeline', 'image-generation-pipeline', 'scenario', 'token_imagegen_' || gen_random_uuid(), 'production', 'active');

-- Insert agent templates with sophisticated prompts
INSERT INTO agents (id, name, display_name, description, system_prompt, user_prompt_template, capabilities, max_tokens, temperature, model, is_active) VALUES 
    (
        'b0000001-0000-0000-0000-000000000001',
        'deep-investigator',
        'Deep Code Investigator',
        'Performs thorough investigation of issues by analyzing code structure, dependencies, and execution flow',
        'You are an expert software debugger and code analyst. Your role is to investigate issues by:
1. Analyzing the codebase structure and dependencies
2. Tracing execution paths and data flow
3. Identifying root causes based on error patterns
4. Suggesting specific, actionable fixes
5. Checking for similar issues in the codebase

Always provide confidence scores for your analysis and cite specific code locations.',
        'Investigate the following issue in the {{app_name}} application:

Title: {{issue_title}}
Description: {{issue_description}}
Type: {{issue_type}}
Priority: {{issue_priority}}

Error Message: {{error_message}}
Stack Trace:
{{stack_trace}}

Affected Files: {{affected_files}}

Please analyze:
1. The root cause of this issue
2. All code components involved
3. Potential side effects of fixes
4. Similar patterns in the codebase
5. Recommended solution with implementation steps

Provide confidence scores (0-1) for each finding.',
        '{investigate}',
        8192,
        0.2,
        'claude-3-opus-20240229',
        true
    ),
    (
        'b0000001-0000-0000-0000-000000000002',
        'auto-fixer',
        'Automated Fix Generator',
        'Generates and validates fixes for identified issues',
        'You are an expert software engineer specialized in generating fixes for bugs. Your approach:
1. Analyze the investigation report thoroughly
2. Generate minimal, targeted fixes
3. Ensure backward compatibility
4. Add appropriate error handling
5. Include test cases for the fix
6. Document changes clearly

Focus on correctness over optimization. Always validate your fixes against the existing test suite.',
        'Generate a fix for the following investigated issue:

Issue ID: {{issue_id}}
Root Cause: {{root_cause}}
Investigation Report:
{{investigation_report}}

Affected Components:
{{affected_components}}

Requirements:
- Minimal code changes
- Maintain backward compatibility
- Include error handling
- Add/update tests
- Document the fix

Generate:
1. Code changes (with file paths and line numbers)
2. Test cases
3. Verification steps
4. Rollback plan if needed',
        '{fix,test}',
        8192,
        0.15,
        'claude-3-opus-20240229',
        true
    ),
    (
        'b0000001-0000-0000-0000-000000000003',
        'pattern-detector',
        'Cross-App Pattern Analyzer',
        'Identifies patterns and correlations across issues from different applications',
        'You are a pattern recognition specialist analyzing issues across multiple applications. Focus on:
1. Identifying common error patterns
2. Finding shared root causes
3. Detecting cascading failures
4. Recognizing performance bottlenecks
5. Spotting security vulnerabilities

Look for both obvious and subtle patterns. Consider temporal, spatial, and causal relationships.',
        'Analyze the following set of issues for patterns:

Issues Count: {{issue_count}}
Apps Affected: {{app_names}}
Time Range: {{time_range}}

Issues Summary:
{{issues_summary}}

Common Keywords: {{common_keywords}}
Common Errors: {{common_errors}}

Identify:
1. Common patterns across these issues
2. Potential shared root causes
3. Risk of cascade failures
4. Systemic problems
5. Recommended preventive measures

Provide pattern confidence scores and affected app correlations.',
        '{investigate}',
        8192,
        0.3,
        'claude-3-opus-20240229',
        true
    ),
    (
        'b0000001-0000-0000-0000-000000000004',
        'quick-analyzer',
        'Quick Triage Analyzer',
        'Performs rapid initial assessment for issue prioritization',
        'You are a triage specialist providing quick initial assessments. Be concise and actionable. Focus on:
1. Immediate severity assessment
2. Impact radius
3. Quick mitigation steps
4. Priority recommendation
5. Agent assignment suggestion

Provide assessments in under 30 seconds of analysis.',
        'Quickly assess this issue:

Title: {{issue_title}}
Error: {{error_message}}
App: {{app_name}}
Reporter: {{reporter_name}}

Provide:
1. Severity (critical/high/medium/low)
2. Impact assessment (users affected)
3. Immediate mitigation if critical
4. Recommended investigation priority
5. Suggested specialist agent

Keep response under 200 words.',
        '{investigate}',
        2048,
        0.1,
        'claude-3-haiku-20240307',
        true
    ),
    (
        'b0000001-0000-0000-0000-000000000005',
        'test-validator',
        'Test Suite Validator',
        'Validates fixes by running and analyzing test results',
        'You are a QA engineer specialized in test validation. Your responsibilities:
1. Design comprehensive test scenarios
2. Validate fix effectiveness
3. Check for regression
4. Verify edge cases
5. Ensure performance is maintained

Be thorough but efficient. Prioritize critical path testing.',
        'Validate the following fix:

Issue: {{issue_title}}
Fix Applied: {{fix_description}}
Changed Files: {{changed_files}}

Test Requirements:
1. Verify the fix resolves the issue
2. Check for regressions
3. Test edge cases
4. Validate performance impact
5. Confirm no new issues introduced

Provide:
- Test plan
- Test results
- Coverage report
- Performance metrics
- Recommendation (approve/reject/modify)',
        '{test}',
        4096,
        0.1,
        'claude-3-opus-20240229',
        true
    ),
    (
        'b0000001-0000-0000-0000-000000000006',
        'documentation-updater',
        'Documentation Generator',
        'Updates documentation based on fixes and new patterns',
        'You are a technical writer specializing in developer documentation. Focus on:
1. Clear, concise explanations
2. Code examples
3. Migration guides
4. Troubleshooting sections
5. Best practices

Write for developers of varying experience levels.',
        'Document the following change:

Issue Fixed: {{issue_title}}
Solution Applied: {{solution_description}}
Affected Components: {{affected_components}}

Create/Update:
1. Changelog entry
2. Migration guide (if breaking changes)
3. Troubleshooting section
4. Code examples
5. Best practices to prevent recurrence

Format in Markdown. Include code snippets where helpful.',
        '{document}',
        4096,
        0.3,
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
    COUNT(CASE WHEN i.status IN ('open', 'investigating', 'in_progress') THEN 1 END) as active_issues,
    COUNT(CASE WHEN i.status = 'fixed' THEN 1 END) as fixed_issues,
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