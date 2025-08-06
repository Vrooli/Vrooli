-- AI Content Assistant Seed Data
-- Campaign-based content management demo data

-- Insert application metadata
INSERT INTO ai_content_assistant.app_metadata (
    app_name, version, configuration
) VALUES (
    'AI Content Assistant',
    '1.0.0',
    '{
        "features": ["campaigns", "documents", "ai_generation", "semantic_search"],
        "resources": ["postgresql", "qdrant", "minio", "ollama", "comfyui", "unstructured-io", "n8n", "windmill"],
        "business_model": {
            "type": "content_management_saas",
            "target_market": "content_creators_and_agencies",
            "value_proposition": "campaign_based_ai_content_generation"
        }
    }'::jsonb
);

-- Insert sample campaigns
INSERT INTO ai_content_assistant.campaigns (
    id, name, description, brand_guidelines, content_preferences, target_audience
) VALUES 
    (
        '550e8400-e29b-41d4-a716-446655440001'::uuid,
        'SaaS Product Launch',
        'Marketing campaign for launching our new project management SaaS platform',
        '{
            "tone": "professional yet approachable",
            "keywords": ["productivity", "collaboration", "efficiency", "teams"],
            "avoid": ["complicated", "difficult", "expensive"],
            "style": "modern, tech-forward"
        }'::jsonb,
        '{
            "blog_length": "800-1200 words",
            "social_format": "linkedin_and_twitter",
            "image_style": "clean, modern, blue theme"
        }'::jsonb,
        'Small to medium businesses, project managers, remote teams'
    ),
    (
        '550e8400-e29b-41d4-a716-446655440002'::uuid,
        'Health & Wellness Blog',
        'Content creation for wellness coaching business',
        '{
            "tone": "warm, encouraging, expert",
            "keywords": ["wellness", "mindfulness", "health", "balance"],
            "style": "personal, authentic, evidence-based"
        }'::jsonb,
        '{
            "blog_length": "600-900 words",
            "social_format": "instagram_and_facebook",
            "image_style": "natural, calming, earth tones"
        }'::jsonb,
        'Health-conscious individuals, working professionals seeking work-life balance'
    ),
    (
        '550e8400-e29b-41d4-a716-446655440003'::uuid,
        'E-commerce Holiday Campaign',
        'Q4 holiday marketing for online retail store',
        '{
            "tone": "festive, urgent, value-focused",
            "keywords": ["deals", "gifts", "holiday", "savings", "limited time"],
            "style": "energetic, promotional"
        }'::jsonb,
        '{
            "blog_length": "400-600 words",
            "social_format": "all_platforms",
            "image_style": "bright, festive, product-focused"
        }'::jsonb,
        'Online shoppers, gift buyers, deal seekers'
    );

-- Insert default configuration values
INSERT INTO ai_content_assistant.app_config (config_key, config_value, config_type, description) VALUES 
    -- Feature flags
    ('feature_campaign_management', 'true'::jsonb, 'feature_flag', 'Enable campaign-based content organization'),
    ('feature_document_processing', 'true'::jsonb, 'feature_flag', 'Enable document upload and text extraction'),
    ('feature_semantic_search', 'true'::jsonb, 'feature_flag', 'Enable vector-based document search'),
    ('feature_ai_content_generation', 'true'::jsonb, 'feature_flag', 'Enable AI-powered content generation'),
    ('feature_image_generation', 'true'::jsonb, 'feature_flag', 'Enable AI image generation via ComfyUI'),
    ('feature_content_templates', 'true'::jsonb, 'feature_flag', 'Enable structured content templates'),
    
    -- Application settings
    ('app_name', '"AI Content Assistant"'::jsonb, 'setting', 'Application display name'),
    ('app_description', '"Campaign-based AI content assistant with intelligent context management"'::jsonb, 'setting', 'Application description'),
    ('max_concurrent_users', '25'::jsonb, 'setting', 'Maximum concurrent users'),
    ('session_timeout_minutes', '60'::jsonb, 'setting', 'User session timeout in minutes'),
    ('enable_usage_analytics', 'true'::jsonb, 'setting', 'Track usage analytics'),
    
    -- Content generation settings
    ('ai_model_temperature', '0.7'::jsonb, 'resource_config', 'AI model temperature for text generation'),
    ('max_context_documents', '10'::jsonb, 'resource_config', 'Maximum documents to use for context'),
    ('context_similarity_threshold', '0.7'::jsonb, 'resource_config', 'Minimum similarity score for document relevance'),
    ('max_generation_tokens', '2048'::jsonb, 'resource_config', 'Maximum tokens for content generation'),
    
    -- File handling
    ('max_file_size_mb', '50'::jsonb, 'resource_config', 'Maximum file upload size in MB'),
    ('supported_file_types', '["pdf", "docx", "txt", "md", "html"]'::jsonb, 'resource_config', 'Supported document file types'),
    ('document_chunk_size', '1000'::jsonb, 'resource_config', 'Text chunk size for embedding'),
    ('document_chunk_overlap', '100'::jsonb, 'resource_config', 'Overlap between text chunks'),
    
    -- UI customization
    ('ui_theme', '"modern"'::jsonb, 'setting', 'UI theme selection'),
    ('ui_primary_color', '"#2563eb"'::jsonb, 'setting', 'Primary UI color'),
    ('ui_brand_name', '"AI Content Assistant"'::jsonb, 'setting', 'Brand name displayed in UI'),
    ('ui_show_campaign_tabs', 'true'::jsonb, 'setting', 'Show campaign tabs in main interface'),
    ('ui_enable_dark_mode', 'true'::jsonb, 'setting', 'Enable dark mode option');

-- Insert sample documents for demonstration
INSERT INTO ai_content_assistant.campaign_documents (
    id, campaign_id, filename, original_filename, file_path, file_size_bytes, content_type, extracted_text, processing_status
) VALUES 
    (
        '650e8400-e29b-41d4-a716-446655440001'::uuid,
        '550e8400-e29b-41d4-a716-446655440001'::uuid, -- SaaS Product Launch
        'product-specs.pdf',
        'Product Specifications.pdf',
        '/campaign-documents/saas-launch/product-specs.pdf',
        245760,
        'application/pdf',
        'Our project management platform offers advanced features including real-time collaboration, automated workflow management, comprehensive reporting dashboards, and seamless integrations with popular development tools. Key benefits include 40% increase in team productivity, reduced project delivery time, and improved stakeholder communication.',
        'completed'
    ),
    (
        '650e8400-e29b-41d4-a716-446655440002'::uuid,
        '550e8400-e29b-41d4-a716-446655440002'::uuid, -- Health & Wellness Blog
        'wellness-research.docx',
        'Wellness Research Studies.docx',
        '/campaign-documents/wellness-blog/wellness-research.docx',
        128540,
        'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
        'Recent studies show that mindfulness practices can reduce stress by up to 30% and improve focus by 25%. Regular meditation, even 10 minutes daily, has been linked to better sleep quality, enhanced emotional regulation, and increased overall life satisfaction. Work-life balance strategies that incorporate mindfulness show the highest success rates.',
        'completed'
    ),
    (
        '650e8400-e29b-41d4-a716-446655440003'::uuid,
        '550e8400-e29b-41d4-a716-446655440003'::uuid, -- E-commerce Holiday Campaign
        'product-catalog.txt',
        'Holiday Product Catalog.txt',
        '/campaign-documents/holiday-campaign/product-catalog.txt',
        89320,
        'text/plain',
        'Featured holiday products: Premium wireless headphones (40% off), Smart home starter kits (35% off), Fitness tracking bundles (50% off), Tech accessories gift sets (30% off). Limited quantities available. Free shipping on orders over $75. Extended return policy through January 31st.',
        'completed'
    );

-- Insert sample user sessions for demonstration
INSERT INTO ai_content_assistant.user_sessions (
    session_id, user_identifier, current_campaign_id, context_data, preferences
) VALUES 
    ('demo-session-001', 'content-creator-demo', 
     '550e8400-e29b-41d4-a716-446655440001'::uuid,
     '{"source": "demo", "use_case": "saas_marketing", "expertise_level": "intermediate"}'::jsonb,
     '{"notifications": true, "theme": "light", "language": "en", "preferred_content_length": "medium"}'::jsonb),
     
    ('demo-session-002', 'agency-user',
     '550e8400-e29b-41d4-a716-446655440002'::uuid,
     '{"source": "agency_demo", "environment": "development", "team_size": "small"}'::jsonb,
     '{"notifications": true, "theme": "dark", "language": "en", "preferred_content_length": "long"}'::jsonb);

-- Insert sample generated content
INSERT INTO ai_content_assistant.generated_content (
    id, campaign_id, content_type, title, prompt, generated_content, used_documents, session_id, user_identifier, generation_time_ms
) VALUES 
    (
        '750e8400-e29b-41d4-a716-446655440001'::uuid,
        '550e8400-e29b-41d4-a716-446655440001'::uuid,
        'blog_post',
        'Revolutionizing Team Productivity: The Future of Project Management',
        'Write a blog post about how our project management platform improves team productivity',
        '# Revolutionizing Team Productivity: The Future of Project Management\n\nIn today''s fast-paced business environment, teams are constantly seeking ways to improve efficiency and deliver projects faster. Our advanced project management platform represents a significant leap forward in collaborative work management.\n\n## Key Benefits That Matter\n\nOur platform delivers measurable results: teams experience a 40% increase in productivity while significantly reducing project delivery time. The secret lies in our comprehensive approach to workflow management and stakeholder communication.\n\n## Real-Time Collaboration Features\n\nThe platform offers seamless real-time collaboration tools that keep everyone synchronized. Whether your team is working remotely or in-office, the intuitive interface ensures everyone stays on the same page.\n\n## Automated Workflow Management\n\nBy automating routine tasks and providing intelligent workflow suggestions, teams can focus on what matters most - delivering exceptional results for their stakeholders.',
        ARRAY['650e8400-e29b-41d4-a716-446655440001'::uuid],
        (SELECT id FROM ai_content_assistant.user_sessions WHERE session_id = 'demo-session-001'),
        'content-creator-demo',
        2340
    ),
    (
        '750e8400-e29b-41d4-a716-446655440002'::uuid,
        '550e8400-e29b-41d4-a716-446655440002'::uuid,
        'social_media',
        'Daily Mindfulness Tip',
        'Create a LinkedIn post about the benefits of daily mindfulness practice',
        '🧠 Just 10 minutes of daily mindfulness can transform your work-life balance! \n\nRecent research shows:\n✅ 30% reduction in stress levels\n✅ 25% improvement in focus\n✅ Better sleep quality & emotional regulation\n\nStart small: try a 5-minute morning meditation and build from there. Your future self will thank you! 💪\n\n#Mindfulness #WorkLifeBalance #WellnessTips #ProductivityHacks',
        ARRAY['650e8400-e29b-41d4-a716-446655440002'::uuid],
        (SELECT id FROM ai_content_assistant.user_sessions WHERE session_id = 'demo-session-002'),
        'agency-user',
        1150
    );

-- Insert sample activity log entries
INSERT INTO ai_content_assistant.activity_log (
    event_type, event_data, campaign_id, session_id, user_identifier, processing_time_ms, status
) VALUES 
    ('campaign_created', 
     '{"campaign_name": "SaaS Product Launch", "initial_documents": 1}'::jsonb,
     '550e8400-e29b-41d4-a716-446655440001'::uuid,
     (SELECT id FROM ai_content_assistant.user_sessions WHERE session_id = 'demo-session-001'),
     'content-creator-demo', 180, 'completed'),
     
    ('document_uploaded',
     '{"filename": "product-specs.pdf", "file_size_mb": 0.24, "processing_time_ms": 850}'::jsonb,
     '550e8400-e29b-41d4-a716-446655440001'::uuid,
     (SELECT id FROM ai_content_assistant.user_sessions WHERE session_id = 'demo-session-001'),
     'content-creator-demo', 850, 'completed'),
     
    ('content_generated',
     '{"content_type": "blog_post", "word_count": 245, "documents_used": 1}'::jsonb,
     '550e8400-e29b-41d4-a716-446655440001'::uuid,
     (SELECT id FROM ai_content_assistant.user_sessions WHERE session_id = 'demo-session-001'),
     'content-creator-demo', 2340, 'completed');

-- Insert sample resource metrics for demonstration
INSERT INTO ai_content_assistant.resource_metrics (
    resource_type, resource_name, operation, duration_ms, success, 
    tokens_used, estimated_cost_usd, session_id, request_metadata
) VALUES 
    ('ai', 'ollama', 'text_generation', 2340, true, 580, 0.0058,
     (SELECT id FROM ai_content_assistant.user_sessions WHERE session_id = 'demo-session-001'),
     '{"model": "llama3.1:8b", "prompt_length": 420, "context_documents": 1}'::jsonb),
    
    ('ai', 'ollama', 'embedding_generation', 450, true, null, 0.001,
     (SELECT id FROM ai_content_assistant.user_sessions WHERE session_id = 'demo-session-001'),
     '{"model": "nomic-embed-text", "text_chunks": 3}'::jsonb),
    
    ('storage', 'qdrant', 'vector_search', 85, true, null, 0.0001,
     (SELECT id FROM ai_content_assistant.user_sessions WHERE session_id = 'demo-session-001'),
     '{"collection": "campaign_documents", "query_vector_size": 384, "results": 5}'::jsonb),
    
    ('storage', 'minio', 'file_upload', 320, true, null, 0.0001,
     (SELECT id FROM ai_content_assistant.user_sessions WHERE session_id = 'demo-session-001'),
     '{"bucket": "campaign-documents", "file_size_mb": 0.24}'::jsonb),
    
    ('ai', 'unstructured_io', 'document_processing', 850, true, null, 0.003,
     (SELECT id FROM ai_content_assistant.user_sessions WHERE session_id = 'demo-session-001'),
     '{"file_type": "pdf", "pages_processed": 4, "extracted_text_length": 1247}'::jsonb),
    
    ('automation', 'n8n', 'workflow_execution', 680, true, null, 0.0002,
     (SELECT id FROM ai_content_assistant.user_sessions WHERE session_id = 'demo-session-001'),
     '{"workflow_name": "document-processing", "steps_executed": 6}'::jsonb),
    
    ('storage', 'postgres', 'query_execution', 32, true, null, 0.00001,
     (SELECT id FROM ai_content_assistant.user_sessions WHERE session_id = 'demo-session-001'),
     '{"query_type": "INSERT", "table": "generated_content"}'::jsonb);

-- Add content generation templates
INSERT INTO ai_content_assistant.app_config (config_key, config_value, config_type, description) VALUES 
    ('content_templates', '{
        "blog_post": {
            "structure": ["headline", "introduction", "main_points", "conclusion", "call_to_action"],
            "min_words": 600,
            "max_words": 1200,
            "seo_focus": true
        },
        "social_media": {
            "platforms": {
                "linkedin": {"max_chars": 3000, "tone": "professional"},
                "twitter": {"max_chars": 280, "tone": "conversational"},
                "facebook": {"max_chars": 2000, "tone": "engaging"}
            },
            "include_hashtags": true,
            "include_call_to_action": true
        },
        "email": {
            "structure": ["subject", "greeting", "body", "closing", "signature"],
            "max_words": 400,
            "personalization": true
        }
    }'::jsonb, 'resource_config', 'Content generation templates and guidelines');

-- Verify the seeded data
SELECT 'Application metadata:' as info, COUNT(*) as count FROM ai_content_assistant.app_metadata
UNION ALL
SELECT 'Campaigns:', COUNT(*) FROM ai_content_assistant.campaigns
UNION ALL
SELECT 'Documents:', COUNT(*) FROM ai_content_assistant.campaign_documents
UNION ALL
SELECT 'Generated content:', COUNT(*) FROM ai_content_assistant.generated_content
UNION ALL
SELECT 'Configuration entries:', COUNT(*) FROM ai_content_assistant.app_config
UNION ALL
SELECT 'User sessions:', COUNT(*) FROM ai_content_assistant.user_sessions
UNION ALL
SELECT 'Activity log entries:', COUNT(*) FROM ai_content_assistant.activity_log
UNION ALL
SELECT 'Resource metrics:', COUNT(*) FROM ai_content_assistant.resource_metrics;

-- Display sample campaign summary
SELECT * FROM ai_content_assistant.campaign_summary;