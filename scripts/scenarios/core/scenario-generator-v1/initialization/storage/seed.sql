-- Scenario Generator V1 - Seed Data
-- Sample campaigns, scenarios, and analytics for testing and demonstration

-- =============================================
-- SAMPLE CAMPAIGNS
-- =============================================

-- Insert sample campaigns to demonstrate the organization system
INSERT INTO campaigns (name, description, color_theme, icon_name, client_name, target_market, business_category, status) VALUES 
    ('AI-Powered Customer Service', 
     'Suite of scenarios focused on automated customer support solutions', 
     'blue', 'headset', 'TechCorp Solutions', 'SMB E-commerce', 'customer-service', 'active'),
    
    ('Content Creation & Marketing', 
     'Automated content generation and marketing workflow scenarios', 
     'purple', 'edit', 'Creative Agency Pro', 'Digital Marketing', 'content-creation', 'active'),
    
    ('Document Processing Pipeline', 
     'Intelligent document analysis and processing scenarios', 
     'green', 'file-text', 'LegalTech Innovations', 'Legal Services', 'document-processing', 'active'),
    
    ('Internal Development Tools', 
     'Scenarios for improving Vrooli development and operations', 
     'orange', 'tool', 'Vrooli Internal', 'Platform Development', 'development-tools', 'active'),
     
    ('Healthcare Automation', 
     'Medical data processing and patient management scenarios', 
     'red', 'activity', 'MedTech Partners', 'Healthcare', 'healthcare', 'active');

-- =============================================
-- SAMPLE SCENARIOS
-- =============================================

-- Get campaign IDs for foreign key references
WITH campaign_ids AS (
    SELECT id, name FROM campaigns
)

INSERT INTO scenarios (
    campaign_id, scenario_id, scenario_name, description, user_request, 
    complexity, category, required_resources, planning_iterations, 
    implementation_iterations, validation_iterations, status, current_phase, 
    progress_percent, estimated_revenue, tags
) 
-- Customer Service Scenarios
SELECT c.id, 'chatbot-support-v1', 'AI Customer Support Chatbot', 
       'Intelligent chatbot for handling customer inquiries, order tracking, and basic support tasks',
       'I need a customer support chatbot that can handle order inquiries, returns, and basic troubleshooting for my e-commerce store. It should integrate with my existing systems and escalate complex issues to human agents.',
       'intermediate', 'customer-service', 
       '["ollama", "n8n", "postgres", "redis"]'::jsonb, 3, 2, 3,
       'completed', 'completed', 100,
       '{"min": 15000, "max": 25000, "currency": "USD"}'::jsonb,
       '{"chatbot", "customer-service", "e-commerce"}'
FROM campaign_ids c WHERE c.name = 'AI-Powered Customer Service'

UNION ALL

SELECT c.id, 'ticket-routing-v1', 'Smart Ticket Routing System',
       'Automated system for categorizing and routing customer support tickets',
       'Build a smart ticket routing system that analyzes customer emails, categorizes them by urgency and type, and automatically assigns them to the right support team members.',
       'intermediate', 'customer-service',
       '["ollama", "n8n", "postgres", "windmill"]'::jsonb, 2, 2, 2,
       'planning', 'planning', 25,
       '{"min": 12000, "max": 20000, "currency": "USD"}'::jsonb,
       '{"automation", "customer-service", "routing"}'
FROM campaign_ids c WHERE c.name = 'AI-Powered Customer Service'

UNION ALL

-- Content Creation Scenarios  
SELECT c.id, 'social-media-generator-v1', 'Automated Social Media Content Generator',
       'AI-powered system for creating social media posts with images and captions',
       'Create a system that takes blog articles as input and automatically generates social media posts for Facebook, Twitter, and Instagram, complete with AI-generated images and optimized captions.',
       'advanced', 'content-creation',
       '["ollama", "comfyui", "n8n", "minio", "postgres"]'::jsonb, 4, 3, 4,
       'implementing', 'implementing', 60,
       '{"min": 20000, "max": 35000, "currency": "USD"}'::jsonb,
       '{"social-media", "content-creation", "ai", "images"}'
FROM campaign_ids c WHERE c.name = 'Content Creation & Marketing'

UNION ALL

SELECT c.id, 'blog-writer-v1', 'AI Blog Writing Assistant',
       'Automated blog post generation and editing system',
       'I need an AI system that can research topics, generate comprehensive blog posts, and format them for our CMS. It should include SEO optimization and fact-checking.',
       'intermediate', 'content-creation',
       '["ollama", "searxng", "n8n", "postgres", "minio"]'::jsonb, 3, 2, 3,
       'validating', 'validating', 85,
       '{"min": 15000, "max": 28000, "currency": "USD"}'::jsonb,
       '{"blogging", "seo", "content-creation", "research"}'
FROM campaign_ids c WHERE c.name = 'Content Creation & Marketing'

UNION ALL

-- Document Processing Scenarios
SELECT c.id, 'contract-analyzer-v1', 'Legal Contract Analysis System',
       'Automated system for analyzing legal contracts and identifying key terms',
       'Build a system that can process legal contracts, extract key terms and clauses, identify potential risks, and generate summary reports for our legal team review.',
       'advanced', 'document-processing',
       '["unstructured-io", "ollama", "n8n", "postgres", "minio", "qdrant"]'::jsonb, 4, 3, 5,
       'requested', 'requested', 0,
       '{"min": 25000, "max": 45000, "currency": "USD"}'::jsonb,
       '{"legal", "contract-analysis", "document-processing"}'
FROM campaign_ids c WHERE c.name = 'Document Processing Pipeline'

UNION ALL

SELECT c.id, 'invoice-processor-v1', 'Automated Invoice Processing System',
       'System for extracting data from invoices and integrating with accounting software',
       'Create an automated invoice processing system that can extract data from PDF invoices, validate the information, and automatically enter it into QuickBooks.',
       'intermediate', 'document-processing',
       '["unstructured-io", "n8n", "postgres", "minio"]'::jsonb, 3, 2, 3,
       'completed', 'completed', 100,
       '{"min": 18000, "max": 30000, "currency": "USD"}'::jsonb,
       '{"invoices", "accounting", "automation", "quickbooks"}'
FROM campaign_ids c WHERE c.name = 'Document Processing Pipeline'

UNION ALL

-- Development Tools Scenarios
SELECT c.id, 'code-reviewer-v1', 'Automated Code Review Assistant',
       'AI system for reviewing code quality and suggesting improvements',
       'Build an automated code review system that can analyze pull requests, check coding standards, identify potential bugs, and suggest improvements using AI.',
       'advanced', 'development-tools',
       '["ollama", "n8n", "postgres", "redis", "windmill"]'::jsonb, 3, 3, 4,
       'implementing', 'implementing', 45,
       '{"min": 22000, "max": 38000, "currency": "USD"}'::jsonb,
       '{"code-review", "development", "quality-assurance"}'
FROM campaign_ids c WHERE c.name = 'Internal Development Tools'

UNION ALL

-- Healthcare Scenarios
SELECT c.id, 'patient-data-processor-v1', 'Patient Data Processing System',
       'HIPAA-compliant system for processing and analyzing patient medical records',
       'Create a secure, HIPAA-compliant system that can process patient medical records, extract key information, and generate analysis reports while maintaining strict privacy controls.',
       'advanced', 'healthcare',
       '["unstructured-io", "ollama", "postgres", "vault", "windmill"]'::jsonb, 5, 4, 5,
       'planning', 'planning', 15,
       '{"min": 35000, "max": 60000, "currency": "USD"}'::jsonb,
       '{"healthcare", "hipaa", "patient-data", "medical-records"}'
FROM campaign_ids c WHERE c.name = 'Healthcare Automation';

-- =============================================
-- SAMPLE CLAUDE INTERACTIONS
-- =============================================

-- Add sample Claude Code interactions for completed scenarios
WITH completed_scenarios AS (
    SELECT id FROM scenarios WHERE status = 'completed' LIMIT 2
)
INSERT INTO claude_interactions (
    scenario_id, phase, iteration_number, interaction_type, 
    prompt_template, prompt_content, response_content,
    started_at, completed_at, duration_ms, success,
    input_tokens, output_tokens, estimated_cost_usd,
    claude_model, temperature
)
SELECT 
    s.id, 'planning', 1, 'prompt',
    'initial-planning-prompt',
    'Generate a comprehensive implementation plan for: AI Customer Support Chatbot for e-commerce...',
    'Based on your requirements, I recommend the following implementation approach...',
    NOW() - INTERVAL '2 days', NOW() - INTERVAL '2 days' + INTERVAL '3 minutes',
    180000, true, 1250, 2800, 0.045,
    'claude-3-sonnet', 0.7
FROM completed_scenarios s
LIMIT 1

UNION ALL

SELECT 
    s.id, 'implementation', 1, 'response',
    'implementation-prompt',
    'Implement the chatbot scenario based on the approved plan...',
    'Here is the complete implementation with n8n workflows, database schema...',
    NOW() - INTERVAL '1 day', NOW() - INTERVAL '1 day' + INTERVAL '8 minutes',
    480000, true, 2100, 5200, 0.095,
    'claude-3-sonnet', 0.7
FROM completed_scenarios s
LIMIT 1;

-- =============================================
-- SAMPLE IMPROVEMENT PATTERNS
-- =============================================

-- Add sample improvement patterns based on common issues
INSERT INTO improvement_patterns (
    issue_category, issue_description, solution_description,
    scenario_phase, complexity, resources_involved,
    occurrence_count, success_rate, confidence_score,
    suggested_improvements, prevention_strategies
) VALUES
    ('Resource Configuration', 
     'n8n workflows failing to connect to Ollama due to incorrect endpoint configuration',
     'Updated resource URLs configuration to use localhost:11434 instead of 127.0.0.1:11434',
     'implementation', 'intermediate', '["n8n", "ollama"]'::jsonb,
     5, 90.0, 0.85,
     'Standardize all resource endpoint configurations to use localhost',
     'Add endpoint validation step in the planning phase'),
     
    ('Schema Design',
     'PostgreSQL foreign key constraints causing deployment failures',
     'Modified schema generation to include proper constraint ordering and cascade rules',
     'validation', 'advanced', '["postgres"]'::jsonb,
     3, 95.0, 0.90,
     'Include database constraint validation in the implementation phase',
     'Use schema validation tools before deployment'),
     
    ('Claude Prompt Engineering',
     'Generated scenarios missing critical error handling code',
     'Enhanced prompts to explicitly request comprehensive error handling and validation',
     'implementation', 'all', '["claude-code"]'::jsonb,
     8, 75.0, 0.80,
     'Add error handling requirements to all implementation prompts',
     'Include error handling checklist in prompt templates');

-- =============================================
-- SAMPLE VALIDATION RESULTS
-- =============================================

-- Add sample validation results
INSERT INTO validation_results (
    scenario_id, validation_attempt, validation_type,
    success, exit_code, stdout_output, stderr_output,
    started_at, completed_at, duration_ms,
    issues_found, fixes_applied
)
SELECT 
    s.id, 1, 'dry_run',
    true, 0, 
    'Scenario validation successful. All resources configured correctly.',
    '',
    NOW() - INTERVAL '1 day', NOW() - INTERVAL '1 day' + INTERVAL '2 minutes',
    120000,
    '[]'::jsonb, '[]'::jsonb
FROM scenarios s 
WHERE s.status = 'completed' 
LIMIT 1

UNION ALL

SELECT 
    s.id, 1, 'dry_run',
    false, 1,
    'Resource validation failed',
    'Error: PostgreSQL connection failed - database does not exist',
    NOW() - INTERVAL '3 hours', NOW() - INTERVAL '3 hours' + INTERVAL '1 minute',
    60000,
    '[{"type": "database_missing", "resource": "postgres", "severity": "high"}]'::jsonb,
    '[{"action": "create_database", "result": "success"}]'::jsonb
FROM scenarios s 
WHERE s.status = 'validating'
LIMIT 1;

-- =============================================
-- ANALYTICS DATA
-- =============================================

-- Add sample daily analytics
INSERT INTO daily_analytics (
    analytics_date, scenarios_requested, scenarios_completed, scenarios_failed,
    avg_generation_time_minutes, avg_planning_iterations, avg_implementation_iterations,
    avg_validation_iterations, success_rate, first_time_validation_success_rate,
    most_used_resources, category_breakdown, complexity_breakdown,
    estimated_revenue_total, total_claude_interactions, total_tokens_used, total_claude_cost_usd
) VALUES
    (CURRENT_DATE - INTERVAL '7 days', 3, 2, 0, 45.5, 3.0, 2.5, 3.0, 85.0, 75.0,
     '["ollama", "n8n", "postgres"]'::jsonb,
     '{"customer-service": 2, "content-creation": 1}'::jsonb,
     '{"intermediate": 2, "advanced": 1}'::jsonb,
     75000, 12, 15400, 1.85),
     
    (CURRENT_DATE - INTERVAL '6 days', 2, 1, 1, 52.0, 3.5, 2.0, 4.0, 50.0, 60.0,
     '["ollama", "comfyui", "n8n"]'::jsonb,
     '{"content-creation": 1, "document-processing": 1}'::jsonb,
     '{"intermediate": 1, "advanced": 1}'::jsonb,
     35000, 8, 11200, 1.45),
     
    (CURRENT_DATE - INTERVAL '1 day', 4, 3, 0, 38.2, 2.7, 2.3, 2.7, 88.0, 80.0,
     '["ollama", "n8n", "postgres", "minio"]'::jsonb,
     '{"customer-service": 2, "development-tools": 1, "healthcare": 1}'::jsonb,
     '{"intermediate": 2, "advanced": 2}'::jsonb,
     125000, 18, 22800, 2.95);

-- =============================================
-- VERIFICATION QUERIES
-- =============================================

-- Display summary of seeded data
DO $$
BEGIN
    RAISE NOTICE 'Scenario Generator V1 seed data inserted successfully';
    RAISE NOTICE 'Campaigns: %', (SELECT COUNT(*) FROM campaigns);
    RAISE NOTICE 'Scenarios: %', (SELECT COUNT(*) FROM scenarios);
    RAISE NOTICE 'Claude Interactions: %', (SELECT COUNT(*) FROM claude_interactions);
    RAISE NOTICE 'Improvement Patterns: %', (SELECT COUNT(*) FROM improvement_patterns);
    RAISE NOTICE 'Validation Results: %', (SELECT COUNT(*) FROM validation_results);
    RAISE NOTICE 'Analytics Entries: %', (SELECT COUNT(*) FROM daily_analytics);
END $$;

-- Show campaign summary
SELECT 
    'Campaign Summary' as report_type,
    name,
    scenario_count,
    success_rate || '%' as success_rate,
    '$' || total_revenue_potential as revenue_potential
FROM campaigns
ORDER BY name;

-- Show scenario status summary  
SELECT 
    'Scenario Status' as report_type,
    status,
    COUNT(*) as count,
    ROUND(AVG(progress_percent), 1) || '%' as avg_progress
FROM scenarios
GROUP BY status
ORDER BY 
    CASE status
        WHEN 'completed' THEN 1
        WHEN 'implementing' THEN 2
        WHEN 'validating' THEN 3
        WHEN 'planning' THEN 4
        WHEN 'requested' THEN 5
        ELSE 6
    END;