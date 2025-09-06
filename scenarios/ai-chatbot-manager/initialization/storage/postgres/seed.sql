-- AI Chatbot Manager Seed Data
-- Sample chatbot configurations for testing and demonstration

-- Insert sample chatbots with different personalities and use cases
INSERT INTO chatbots (
    name, 
    description, 
    personality, 
    knowledge_base,
    model_config,
    widget_config
) VALUES 
(
    'Sales Assistant Pro',
    'High-converting sales chatbot for SaaS products',
    'You are an expert sales assistant helping prospects discover how our software can solve their business problems. You are enthusiastic but not pushy, always ask qualifying questions, and focus on understanding customer needs before presenting solutions. Always try to capture contact information for follow-up.',
    'Our SaaS platform offers project management, team collaboration, and analytics tools. Pricing starts at $29/month for teams up to 10 users. Key features include: real-time collaboration, automated reporting, integrations with 50+ tools, mobile apps, and enterprise-grade security. We offer a 14-day free trial.',
    '{"model": "llama3.2", "temperature": 0.7, "max_tokens": 800}',
    '{"theme": "professional", "position": "bottom-right", "primaryColor": "#2563eb", "greeting": "Hi! I''m here to help you discover how our platform can streamline your team''s workflow. What challenges is your team facing?"}'
),
(
    'Support Helper',
    'Customer support chatbot for technical assistance',
    'You are a helpful technical support assistant. You are patient, clear, and thorough in your explanations. Always try to understand the user''s technical level and adjust your language accordingly. If you cannot solve an issue, offer to connect them with a human agent.',
    'Common issues include: password resets (direct users to forgot password page), billing questions (explain our pricing tiers), feature requests (collect feedback), bug reports (ask for steps to reproduce), and integration help (we support Slack, Microsoft Teams, Google Workspace, Salesforce, and HubSpot).',
    '{"model": "mistral", "temperature": 0.5, "max_tokens": 600}',
    '{"theme": "support", "position": "bottom-left", "primaryColor": "#059669", "greeting": "Welcome to our support center! I''m here to help you with any questions or issues. How can I assist you today?"}'
),
(
    'Lead Qualifier',
    'B2B lead qualification and appointment setting',
    'You are a professional lead qualification specialist. Your goal is to understand prospect needs, budget, timeline, and decision-making process. You are consultative and focus on building rapport while gathering qualification information. Always aim to schedule a demo or consultation call.',
    'We provide enterprise software solutions for companies with 100+ employees. Our solutions include: CRM systems ($50K-200K), data analytics platforms ($30K-150K), and custom integrations. Typical implementation takes 3-6 months. We serve Fortune 500 companies and growing mid-market businesses.',
    '{"model": "llama3.2", "temperature": 0.6, "max_tokens": 900}',
    '{"theme": "enterprise", "position": "center", "primaryColor": "#7c3aed", "greeting": "Hello! I help connect growing businesses with solutions that scale. What brings you here today?"}'
),
(
    'E-commerce Assistant',
    'Product recommendation and shopping support',
    'You are an enthusiastic shopping assistant who loves helping customers find the perfect products. You ask about preferences, usage needs, and budget to make personalized recommendations. You create excitement about products while being honest about features and limitations.',
    'We sell premium outdoor gear including: hiking boots ($150-400), backpacks ($80-300), tents ($200-800), and camping accessories. All products come with a lifetime warranty. We offer free shipping over $100 and 30-day returns. Popular brands include Patagonia, REI Co-op, and our house brand.',
    '{"model": "llama3.2", "temperature": 0.8, "max_tokens": 700}',
    '{"theme": "ecommerce", "position": "bottom-right", "primaryColor": "#dc2626", "greeting": "Ready for your next adventure? I''m here to help you find the perfect gear! What outdoor activities are you planning?"}'
);

-- Insert sample conversations for demonstration
DO $$
DECLARE
    sales_bot_id UUID;
    support_bot_id UUID;
    conv_id_1 UUID;
    conv_id_2 UUID;
    conv_id_3 UUID;
BEGIN
    -- Get chatbot IDs
    SELECT id INTO sales_bot_id FROM chatbots WHERE name = 'Sales Assistant Pro';
    SELECT id INTO support_bot_id FROM chatbots WHERE name = 'Support Helper';
    
    -- Sample sales conversation
    INSERT INTO conversations (
        chatbot_id, 
        session_id, 
        user_ip, 
        started_at, 
        ended_at, 
        lead_captured, 
        lead_data
    ) VALUES (
        sales_bot_id, 
        'demo_session_001', 
        '192.168.1.100'::inet,
        NOW() - INTERVAL '2 days',
        NOW() - INTERVAL '2 days' + INTERVAL '15 minutes',
        true,
        '{"email": "john.doe@example.com", "name": "John Doe", "company": "TechCorp", "role": "Product Manager", "team_size": "15-25"}'
    ) RETURNING id INTO conv_id_1;
    
    -- Messages for sales conversation
    INSERT INTO messages (conversation_id, role, content, metadata) VALUES
    (conv_id_1, 'user', 'Hi, I''m looking for a project management solution for my team.', '{"intent": "inquiry", "confidence": 0.95}'),
    (conv_id_1, 'assistant', 'Great! I''d love to help you find the right solution. How large is your team currently?', '{"response_time_ms": 850}'),
    (conv_id_1, 'user', 'We have about 20 people across 3 departments.', '{"intent": "team_size", "confidence": 0.88}'),
    (conv_id_1, 'assistant', 'Perfect! Our platform is designed for teams exactly that size. What are your biggest project management challenges right now?', '{"response_time_ms": 750}'),
    (conv_id_1, 'user', 'We struggle with visibility across projects and keeping everyone aligned.', '{"intent": "pain_point", "confidence": 0.92}'),
    (conv_id_1, 'assistant', 'Those are exactly the problems we solve! Our real-time collaboration and automated reporting features give managers complete visibility. Would you like to see a demo of how this works?', '{"response_time_ms": 920}'),
    (conv_id_1, 'user', 'Yes, that sounds helpful. My email is john.doe@example.com', '{"intent": "interested", "confidence": 0.97, "lead_captured": true}');
    
    -- Sample support conversation
    INSERT INTO conversations (
        chatbot_id, 
        session_id, 
        user_ip, 
        started_at, 
        ended_at, 
        lead_captured
    ) VALUES (
        support_bot_id, 
        'support_session_001', 
        '10.0.0.55'::inet,
        NOW() - INTERVAL '1 day',
        NOW() - INTERVAL '1 day' + INTERVAL '8 minutes',
        false
    ) RETURNING id INTO conv_id_2;
    
    -- Messages for support conversation
    INSERT INTO messages (conversation_id, role, content) VALUES
    (conv_id_2, 'user', 'I can''t log into my account. It says my password is wrong but I''m sure it''s correct.'),
    (conv_id_2, 'assistant', 'I understand how frustrating that can be! Let me help you get back into your account. Have you tried using the "Forgot Password" link on the login page?'),
    (conv_id_2, 'user', 'No, I haven''t tried that yet. Where do I find it?'),
    (conv_id_2, 'assistant', 'You''ll find the "Forgot Password" link right below the password field on our login page. Click that and enter your email address - we''ll send you a secure reset link within a few minutes.'),
    (conv_id_2, 'user', 'Great, I found it. Thanks for the help!'),
    (conv_id_2, 'assistant', 'You''re welcome! The reset email should arrive shortly. If you don''t see it in your inbox within 10 minutes, please check your spam folder. Is there anything else I can help you with today?'),
    (conv_id_2, 'user', 'No, that''s all. Thanks again!');
    
    -- Sample shorter conversation for analytics variety
    INSERT INTO conversations (
        chatbot_id, 
        session_id, 
        user_ip, 
        started_at, 
        ended_at
    ) VALUES (
        sales_bot_id, 
        'quick_session_001', 
        '172.16.0.10'::inet,
        NOW() - INTERVAL '3 hours',
        NOW() - INTERVAL '3 hours' + INTERVAL '2 minutes'
    ) RETURNING id INTO conv_id_3;
    
    INSERT INTO messages (conversation_id, role, content) VALUES
    (conv_id_3, 'user', 'What''s your pricing?'),
    (conv_id_3, 'assistant', 'Our pricing starts at $29/month for teams up to 10 users. For a team of 20 like yours, you''d be looking at our Professional plan. Would you like me to show you what''s included?'),
    (conv_id_3, 'user', 'Too expensive for us right now.'),
    (conv_id_3, 'assistant', 'I understand budget is important. We do offer a 14-day free trial so you can see the value first. Would that be helpful?'),
    (conv_id_3, 'user', 'Maybe later, thanks.');
END $$;

-- Generate some analytics data for the past week
DO $$
DECLARE
    bot_record RECORD;
    target_date DATE;
BEGIN
    -- Update analytics for each chatbot for the past 7 days
    FOR bot_record IN SELECT id FROM chatbots LOOP
        FOR target_date IN SELECT generate_series(CURRENT_DATE - 6, CURRENT_DATE, '1 day'::interval)::date LOOP
            PERFORM update_daily_analytics(target_date, bot_record.id);
        END LOOP;
    END LOOP;
END $$;

-- Insert some intent patterns for demonstration
INSERT INTO intent_patterns (chatbot_id, pattern, intent_name, confidence, occurrence_count) 
SELECT 
    c.id, 
    pattern_data.pattern, 
    pattern_data.intent_name, 
    pattern_data.confidence,
    pattern_data.occurrence_count
FROM chatbots c,
(VALUES 
    ('pricing|cost|price|expensive|cheap|budget', 'pricing_inquiry', 0.85, 15),
    ('demo|demonstration|show me|see how', 'demo_request', 0.90, 8),
    ('team size|how many|users|people', 'team_sizing', 0.75, 12),
    ('integrat|connect|sync|api', 'integration_question', 0.80, 6),
    ('help|support|problem|issue|trouble', 'support_request', 0.95, 25),
    ('login|password|account|access', 'account_issue', 0.88, 18),
    ('features|capabilities|what does|functionality', 'feature_inquiry', 0.82, 20)
) AS pattern_data(pattern, intent_name, confidence, occurrence_count)
WHERE c.name IN ('Sales Assistant Pro', 'Support Helper');

-- Add some comments for the demo data
COMMENT ON TABLE chatbots IS 'Contains 4 sample chatbots: Sales Assistant Pro, Support Helper, Lead Qualifier, and E-commerce Assistant';

-- Show summary of seeded data
DO $$
BEGIN
    RAISE NOTICE 'AI Chatbot Manager seed data loaded successfully:';
    RAISE NOTICE '- 4 sample chatbots with different personalities';
    RAISE NOTICE '- 3 sample conversations with realistic message exchanges';
    RAISE NOTICE '- Historical analytics data for the past week';
    RAISE NOTICE '- Intent patterns for machine learning training';
    RAISE NOTICE 'Ready for testing and demonstration!';
END $$;