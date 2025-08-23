-- Seed data for Idea Generator
-- Purpose: Provide realistic sample data for demonstration and testing

-- Sample users
INSERT INTO users (id, username, email, preferences) VALUES 
('550e8400-e29b-41d4-a716-446655440001', 'demo_user', 'demo@ideagen.com', '{"theme": "light", "notifications": true}'),
('550e8400-e29b-41d4-a716-446655440002', 'sarah_innovator', 'sarah@innovation.co', '{"theme": "dark", "dice_animation": "bounce"}'),
('550e8400-e29b-41d4-a716-446655440003', 'alex_creative', 'alex@creative.studio', '{"theme": "light", "chat_sound": false}');

-- Sample campaigns with different colors and contexts
INSERT INTO campaigns (id, name, description, color, context, owner_id) VALUES 
(
    '650e8400-e29b-41d4-a716-446655440001',
    'SaaS Product Innovation',
    'Brainstorming innovative SaaS solutions for small businesses',
    '#10B981', -- Emerald green
    '{
        "target_audience": "Small business owners (1-50 employees)",
        "focus_areas": ["productivity", "automation", "cost-efficiency"],
        "constraints": ["budget-conscious", "easy-to-use", "quick-setup"],
        "market_research": {
            "pain_points": ["manual processes", "data silos", "poor communication"],
            "competitors": ["Notion", "Airtable", "Monday.com"],
            "trends": ["AI integration", "no-code tools", "remote work"]
        }
    }',
    '550e8400-e29b-41d4-a716-446655440001'
),
(
    '650e8400-e29b-41d4-a716-446655440002', 
    'Sustainable Tech Solutions',
    'Environmentally conscious technology products and services',
    '#059669', -- Dark green
    '{
        "target_audience": "Environmentally conscious consumers and businesses",
        "focus_areas": ["renewable energy", "waste reduction", "circular economy"],
        "constraints": ["carbon neutral", "sustainable materials", "local sourcing"],
        "market_research": {
            "pain_points": ["high energy costs", "waste management", "supply chain transparency"],
            "growth_sectors": ["clean energy", "sustainable manufacturing", "green transport"],
            "regulations": ["carbon reporting", "waste regulations", "renewable targets"]
        }
    }',
    '550e8400-e29b-41d4-a716-446655440002'
),
(
    '650e8400-e29b-41d4-a716-446655440003',
    'AI-Powered Education',
    'Leveraging artificial intelligence to transform learning experiences',
    '#3B82F6', -- Blue
    '{
        "target_audience": "Students, educators, and educational institutions",
        "focus_areas": ["personalized learning", "assessment automation", "accessibility"],
        "constraints": ["privacy compliance", "age-appropriate", "affordable"],
        "market_research": {
            "pain_points": ["one-size-fits-all learning", "teacher workload", "student engagement"],
            "opportunities": ["remote learning", "skill gaps", "lifelong learning"],
            "technology_trends": ["adaptive learning", "natural language processing", "computer vision"]
        }
    }',
    '550e8400-e29b-41d4-a716-446655440003'
);

-- Sample ideas for each campaign
INSERT INTO ideas (id, campaign_id, title, content, category, tags, generation_prompt, generation_model, status, created_by) VALUES
(
    '750e8400-e29b-41d4-a716-446655440001',
    '650e8400-e29b-41d4-a716-446655440001',
    'Smart Invoice Processing Assistant',
    'An AI-powered tool that automatically extracts data from invoices, categorizes expenses, and integrates with accounting software. Uses OCR and machine learning to handle various invoice formats, reducing manual data entry by 90%. Features include duplicate detection, vendor management, and approval workflows.',
    'automation',
    ARRAY['AI', 'accounting', 'OCR', 'workflow'],
    'Generate innovative SaaS ideas for small business productivity',
    'llama3.2',
    'refined',
    '550e8400-e29b-41d4-a716-446655440001'
),
(
    '750e8400-e29b-41d4-a716-446655440002',
    '650e8400-e29b-41d4-a716-446655440001',
    'Team Knowledge Graph Builder',
    'A collaborative platform that automatically builds knowledge graphs from team communications, documents, and activities. Helps small teams discover expertise, avoid duplicate work, and onboard new members faster. Integrates with Slack, email, and project management tools.',
    'collaboration',
    ARRAY['knowledge management', 'graph database', 'team collaboration'],
    'Create tools to help small businesses work more efficiently',
    'llama3.2',
    'draft',
    '550e8400-e29b-41d4-a716-446655440001'
),
(
    '750e8400-e29b-41d4-a716-446655440003',
    '650e8400-e29b-41d4-a716-446655440002',
    'Carbon Footprint API Service',
    'A comprehensive API that calculates carbon footprints for products, services, and supply chains. Integrates with existing e-commerce platforms and ERP systems. Provides real-time emissions data, offset recommendations, and sustainability scoring. Helps businesses meet ESG reporting requirements.',
    'sustainability',
    ARRAY['carbon tracking', 'API', 'ESG reporting', 'supply chain'],
    'Develop environmental solutions that help businesses become more sustainable',
    'llama3.2',
    'refined',
    '550e8400-e29b-41d4-a716-446655440002'
),
(
    '750e8400-e29b-41d4-a716-446655440004',
    '650e8400-e29b-41d4-a716-446655440002',
    'Local Resource Sharing Network',
    'A hyperlocal platform connecting businesses and individuals to share underutilized resources like equipment, spaces, and expertise. Reduces waste by maximizing resource utilization within communities. Features include trust scoring, insurance integration, and impact tracking.',
    'sharing economy',
    ARRAY['resource sharing', 'local community', 'waste reduction'],
    'Create solutions for sustainable resource usage',
    'mistral',
    'draft',
    '550e8400-e29b-41d4-a716-446655440002'
),
(
    '750e8400-e29b-41d4-a716-446655440005',
    '650e8400-e29b-41d4-a716-446655440003',
    'Adaptive Learning Companion',
    'An AI tutor that adapts to each student\'s learning style, pace, and interests. Uses multimodal learning (text, audio, visual, interactive) to explain concepts. Tracks progress, identifies knowledge gaps, and provides personalized study plans. Integrates with curriculum standards and existing LMS platforms.',
    'personalized learning',
    ARRAY['AI tutor', 'adaptive learning', 'multimodal', 'curriculum'],
    'Design AI-powered educational tools for personalized learning',
    'llama3.2',
    'refined',
    '550e8400-e29b-41d4-a716-446655440003'
),
(
    '750e8400-e29b-41d4-a716-446655440006',
    '650e8400-e29b-41d4-a716-446655440003',
    'Skill Gap Analysis Platform',
    'A comprehensive platform that analyzes job market trends, identifies emerging skill requirements, and creates personalized learning pathways. Helps students and professionals stay ahead of industry changes. Features include industry partnerships, certification tracking, and career guidance.',
    'career development',
    ARRAY['skill analysis', 'career guidance', 'job market', 'learning paths'],
    'Create tools to help people prepare for future job markets',
    'mistral',
    'draft',
    '550e8400-e29b-41d4-a716-446655440003'
);

-- Sample chat sessions (for ideas that have been refined)
INSERT INTO chat_sessions (id, idea_id, user_id, session_type, status, context) VALUES
(
    '850e8400-e29b-41d4-a716-446655440001',
    '750e8400-e29b-41d4-a716-446655440001',
    '550e8400-e29b-41d4-a716-446655440001',
    'refinement',
    'completed',
    '{"refinement_focus": ["market_validation", "technical_feasibility"], "iterations": 3}'
),
(
    '850e8400-e29b-41d4-a716-446655440002',
    '750e8400-e29b-41d4-a716-446655440003',
    '550e8400-e29b-41d4-a716-446655440002',
    'critique',
    'completed',
    '{"critique_areas": ["business_model", "competitive_analysis"], "improvements_made": true}'
);

-- Sample chat messages showing agent interactions
INSERT INTO chat_messages (id, session_id, sender_type, sender_name, message_text, message_type, metadata) VALUES
(
    '950e8400-e29b-41d4-a716-446655440001',
    '850e8400-e29b-41d4-a716-446655440001',
    'user',
    'demo_user',
    'I like this invoice processing idea, but I want to make it more specific to small restaurants. Can you help refine it?',
    'text',
    '{}'
),
(
    '950e8400-e29b-41d4-a716-446655440002',
    '850e8400-e29b-41d4-a716-446655440001',
    'agent',
    'revise',
    'Great direction! Let me refine this for the restaurant industry specifically. Restaurants have unique needs like ingredient tracking, supplier invoice variations, and food cost management. Here''s how we could adapt the idea...',
    'text',
    '{"agent_reasoning": "Identified restaurant-specific pain points", "context_used": ["campaign_context", "industry_knowledge"]}'
),
(
    '950e8400-e29b-41d4-a716-446655440003',
    '850e8400-e29b-41d4-a716-446655440002',
    'user',
    'sarah_innovator',
    'What are the potential challenges with implementing this carbon footprint API?',
    'text',
    '{}'
),
(
    '950e8400-e29b-41d4-a716-446655440004',
    '850e8400-e29b-41d4-a716-446655440002',
    'agent',
    'critique',
    'Several key challenges to consider: 1) Data accuracy and standardization across different industries, 2) Integration complexity with existing systems, 3) Regulatory compliance across different regions, 4) Real-time calculation performance at scale...',
    'text',
    '{"agent_reasoning": "Analyzed technical and business challenges", "focus_areas": ["scalability", "compliance", "data_quality"]}'
);

-- Sample agent interactions (historical record)
INSERT INTO agent_interactions (id, idea_id, session_id, interaction_type, agent_prompt, agent_response, model_used, metadata) VALUES
(
    'a50e8400-e29b-41d4-a716-446655440001',
    '750e8400-e29b-41d4-a716-446655440001',
    '850e8400-e29b-41d4-a716-446655440001',
    'revise',
    'Refine this invoice processing idea specifically for small restaurants, considering their unique operational needs and constraints.',
    'Restaurant-Focused Invoice Processing Assistant: An AI-powered tool specifically designed for small restaurants that handles supplier invoices, ingredient cost tracking, and food cost management. Features include: seasonal price variation tracking, supplier relationship scoring, food waste calculation from over-ordering, menu profitability analysis, and integration with POS systems for real-time cost tracking.',
    'llama3.2',
    '{"refinement_iterations": 1, "focus_areas": ["industry_specific", "cost_management"]}'
),
(
    'a50e8400-e29b-41d4-a716-446655440002',
    '750e8400-e29b-41d4-a716-446655440003',
    '850e8400-e29b-41d4-a716-446655440002',
    'critique',
    'Provide constructive criticism of this carbon footprint API idea, focusing on potential implementation challenges and market obstacles.',
    'Key challenges: 1) Data standardization - carbon calculations vary by methodology and region, 2) Integration complexity - existing ERP systems may lack necessary data granularity, 3) Verification requirements - customers will need third-party validation for credible reporting, 4) Competitive landscape - large players like Microsoft and Google already offer similar services, 5) Pricing model - need to balance accuracy with cost for smaller businesses.',
    'mistral',
    '{"critique_focus": ["implementation", "market_competition"], "improvement_suggestions": 3}'
);

-- Sample idea versions (showing evolution)
INSERT INTO idea_versions (id, idea_id, session_id, version_number, title, content, changed_by, change_summary) VALUES
(
    'v50e8400-e29b-41d4-a716-446655440001',
    '750e8400-e29b-41d4-a716-446655440001',
    '850e8400-e29b-41d4-a716-446655440001',
    1,
    'Smart Invoice Processing Assistant',
    'An AI-powered tool that automatically extracts data from invoices, categorizes expenses, and integrates with accounting software.',
    'user',
    'Initial idea generation'
),
(
    'v50e8400-e29b-41d4-a716-446655440002',
    '750e8400-e29b-41d4-a716-446655440001',
    '850e8400-e29b-41d4-a716-446655440001',
    2,
    'Restaurant Invoice Processing Assistant',
    'An AI-powered tool specifically designed for small restaurants that handles supplier invoices, ingredient cost tracking, and food cost management. Features seasonal price tracking, supplier scoring, and POS integration.',
    'agent:revise',
    'Specialized for restaurant industry with specific features for food cost management'
);

-- Sample search queries (for analytics)
INSERT INTO search_queries (id, user_id, campaign_id, query_text, results_count) VALUES
(
    's50e8400-e29b-41d4-a716-446655440001',
    '550e8400-e29b-41d4-a716-446655440001',
    '650e8400-e29b-41d4-a716-446655440001',
    'automation tools for accounting',
    5
),
(
    's50e8400-e29b-41d4-a716-446655440002',
    '550e8400-e29b-41d4-a716-446655440002',
    '650e8400-e29b-41d4-a716-446655440002',
    'carbon footprint tracking solutions',
    8
),
(
    's50e8400-e29b-41d4-a716-446655440003',
    '550e8400-e29b-41d4-a716-446655440003',
    '650e8400-e29b-41d4-a716-446655440003',
    'AI educational assessment',
    3
);

-- Update sequences to avoid conflicts
SELECT setval(pg_get_serial_sequence('users', 'id'), 1000);
SELECT setval(pg_get_serial_sequence('campaigns', 'id'), 1000);
SELECT setval(pg_get_serial_sequence('ideas', 'id'), 1000);
SELECT setval(pg_get_serial_sequence('chat_sessions', 'id'), 1000);
SELECT setval(pg_get_serial_sequence('chat_messages', 'id'), 1000);
SELECT setval(pg_get_serial_sequence('agent_interactions', 'id'), 1000);
SELECT setval(pg_get_serial_sequence('idea_versions', 'id'), 1000);
SELECT setval(pg_get_serial_sequence('search_queries', 'id'), 1000);