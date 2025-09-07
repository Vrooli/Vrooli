-- Scenario Generator V1 - Database Seed Data
-- Populate initial scenario templates and sample data

-- Insert additional scenario templates for comprehensive coverage
INSERT INTO scenario_templates (name, description, category, prompt_template, resources_suggested, complexity_level, estimated_revenue_min, estimated_revenue_max) VALUES
('Real Estate Investment Analyzer', 'AI-powered property analysis and investment recommendation system', 'analytics', 'Create a comprehensive real estate investment analysis platform that scrapes MLS listings, performs market comparisons, calculates ROI metrics, generates investment reports, and provides property recommendations. Include automated valuation models, rental yield calculations, and market trend analysis.', ARRAY['postgres', 'claude-code', 'minio'], 'advanced', 35000, 60000),

('Social Media Content Engine', 'Automated content creation and posting across multiple platforms', 'content-marketing', 'Build an AI-powered social media management system that generates posts, creates images, schedules content across Instagram, Twitter, LinkedIn, and Facebook. Include hashtag optimization, engagement tracking, and performance analytics.', ARRAY['postgres', 'comfyui', 'claude-code'], 'intermediate', 20000, 40000),

('Healthcare Patient Portal', 'Secure patient management and telemedicine platform', 'healthcare', 'Create a HIPAA-compliant patient portal with appointment scheduling, medical record management, telemedicine video calls, prescription tracking, and billing integration. Include patient communication tools and health monitoring dashboards.', ARRAY['postgres', 'vault'], 'advanced', 40000, 75000),

('Educational Course Builder', 'Interactive online learning platform with AI tutoring', 'education', 'Build a comprehensive e-learning platform with course creation tools, video hosting, interactive quizzes, AI-powered tutoring assistance, student progress tracking, and certification management. Include payment processing and instructor analytics.', ARRAY['postgres', 'claude-code', 'minio'], 'advanced', 30000, 55000),

('Supply Chain Optimizer', 'AI-driven inventory and logistics management system', 'business-automation', 'Create an intelligent supply chain management platform that optimizes inventory levels, predicts demand, manages supplier relationships, tracks shipments, and automates reordering. Include cost optimization and risk assessment features.', ARRAY['postgres', 'claude-code'], 'advanced', 45000, 80000),

('Financial Planning Assistant', 'Personal and business financial management with AI insights', 'financial', 'Build a comprehensive financial planning tool that connects to bank accounts, categorizes transactions, provides budgeting recommendations, forecasts cash flow, and offers investment advice. Include tax planning and retirement calculations.', ARRAY['postgres', 'claude-code', 'vault'], 'intermediate', 25000, 45000);

-- Insert sample scenarios for demonstration
INSERT INTO scenarios (id, name, description, prompt, status, complexity, category, estimated_revenue, created_at, updated_at) VALUES
('demo-001', 'E-Commerce Analytics Dashboard', 'Comprehensive sales and customer analytics for online stores', 'Create a business intelligence dashboard for e-commerce companies that tracks sales performance, customer behavior, inventory levels, and marketing campaign effectiveness. Include predictive analytics for demand forecasting and customer lifetime value calculations.', 'completed', 'intermediate', 'analytics', 28000, NOW() - INTERVAL '7 days', NOW() - INTERVAL '6 days'),

('demo-002', 'AI-Powered Customer Service Hub', 'Intelligent customer support with automated ticket routing', 'Build a customer service platform that uses AI to automatically categorize support tickets, route them to appropriate agents, suggest responses, and escalate complex issues. Include knowledge base integration and customer satisfaction tracking.', 'completed', 'advanced', 'customer-service', 42000, NOW() - INTERVAL '14 days', NOW() - INTERVAL '12 days'),

('demo-003', 'Document Processing Pipeline', 'Automated extraction and processing of business documents', 'Create a system that automatically processes invoices, contracts, and forms using OCR and AI. Extract key data points, validate information, integrate with accounting systems, and flag exceptions for human review.', 'ready', 'intermediate', 'data-processing', 32000, NOW() - INTERVAL '3 days', NOW() - INTERVAL '2 days');

-- Update usage count for templates that have been used
UPDATE scenario_templates 
SET usage_count = CASE 
    WHEN name = 'Simple SaaS Dashboard' THEN 5
    WHEN name = 'AI Document Processor' THEN 3
    WHEN name = 'Customer Support Bot' THEN 7
    ELSE usage_count
END,
success_rate = CASE
    WHEN name = 'Simple SaaS Dashboard' THEN 80.0
    WHEN name = 'AI Document Processor' THEN 75.0
    WHEN name = 'Customer Support Bot' THEN 85.0
    ELSE success_rate
END
WHERE name IN ('Simple SaaS Dashboard', 'AI Document Processor', 'Customer Support Bot');

-- Insert sample generation logs
INSERT INTO generation_logs (scenario_id, step, prompt, response, success, started_at, completed_at, duration_seconds) VALUES
('demo-001', 'initial_planning', 'Generate implementation plan for e-commerce analytics dashboard...', 'Created comprehensive plan with 5 phases: data collection, analytics engine, visualization layer, reporting system, and deployment.', true, NOW() - INTERVAL '7 days 2 hours', NOW() - INTERVAL '7 days 1 hour 55 minutes', 300),
('demo-001', 'implementation', 'Implement the planned e-commerce analytics dashboard...', 'Successfully generated all components including database schema, API endpoints, dashboard UI, and deployment configuration.', true, NOW() - INTERVAL '7 days 1 hour 30 minutes', NOW() - INTERVAL '7 days 45 minutes', 2700),
('demo-001', 'validation', 'Validate generated scenario against requirements...', 'All validation checks passed. Scenario is ready for deployment.', true, NOW() - INTERVAL '7 days 30 minutes', NOW() - INTERVAL '7 days 25 minutes', 300),

('demo-002', 'initial_planning', 'Generate implementation plan for AI customer service hub...', 'Created detailed architecture with microservices approach, AI integration points, and scalability considerations.', true, NOW() - INTERVAL '14 days 3 hours', NOW() - INTERVAL '14 days 2 hours 45 minutes', 900),
('demo-002', 'plan_refinement', 'Refine the customer service hub plan based on best practices...', 'Enhanced plan with security considerations, compliance requirements, and performance optimization strategies.', true, NOW() - INTERVAL '14 days 2 hours 30 minutes', NOW() - INTERVAL '14 days 2 hours 15 minutes', 900),
('demo-002', 'implementation', 'Implement the refined customer service hub...', 'Generated complete solution with AI routing, knowledge base integration, agent dashboard, and customer portal.', true, NOW() - INTERVAL '14 days 2 hours', NOW() - INTERVAL '14 days 30 minutes', 5400);

-- Create indexes for better query performance on seed data
ANALYZE scenarios;
ANALYZE scenario_templates;
ANALYZE generation_logs;