-- Research Assistant Seed Data
-- Purpose: Initial report templates, example schedules, and sample data

SET search_path TO research_assistant;

-- ==========================================
-- Report Templates
-- ==========================================

INSERT INTO report_templates (name, description, category, default_depth, default_length, sections, required_sources, search_queries, system_prompt, analysis_prompts) VALUES
(
    'Daily AI Progress',
    'Track latest developments in artificial intelligence and machine learning',
    'technology',
    'standard',
    3,
    '[
        {"name": "Executive Summary", "required": true, "max_words": 200},
        {"name": "Key Developments", "required": true, "subsections": ["Research Papers", "Industry News", "Product Launches"]},
        {"name": "Market Impact", "required": true},
        {"name": "Technical Details", "required": false},
        {"name": "Future Implications", "required": true}
    ]'::jsonb,
    15,
    '[
        "artificial intelligence breakthroughs today",
        "machine learning news latest 24 hours",
        "AI research papers arxiv today",
        "OpenAI Anthropic Google DeepMind news"
    ]'::jsonb,
    'You are an AI research analyst providing daily intelligence briefings on artificial intelligence progress. Focus on genuinely new developments from the last 24-48 hours. Prioritize breakthrough research, significant product launches, and industry shifts.',
    '{
        "development_analysis": "Analyze the technical significance and real-world applicability of each development",
        "trend_identification": "Identify emerging patterns across multiple developments",
        "impact_assessment": "Evaluate potential industry and societal impacts"
    }'::jsonb
),
(
    'Competitive Intelligence Report',
    'Comprehensive analysis of competitors and market positioning',
    'business',
    'deep',
    8,
    '[
        {"name": "Executive Overview", "required": true, "max_words": 300},
        {"name": "Competitor Profiles", "required": true, "subsections": ["Products/Services", "Market Share", "Recent Activities", "Strengths/Weaknesses"]},
        {"name": "Market Dynamics", "required": true, "subsections": ["Trends", "Opportunities", "Threats"]},
        {"name": "Strategic Positioning", "required": true},
        {"name": "Financial Analysis", "required": false},
        {"name": "Technology Stack Comparison", "required": false},
        {"name": "Customer Sentiment Analysis", "required": true},
        {"name": "Recommendations", "required": true}
    ]'::jsonb,
    25,
    '[
        "{{COMPANY}} competitors market share",
        "{{COMPANY}} vs {{COMPETITORS}} comparison",
        "{{INDUSTRY}} market analysis trends",
        "{{COMPANY}} customer reviews sentiment",
        "{{COMPANY}} recent news announcements"
    ]'::jsonb,
    'You are a strategic business analyst specializing in competitive intelligence. Provide data-driven insights with specific examples and quantifiable metrics where possible. Maintain objectivity while highlighting actionable intelligence.',
    '{
        "competitor_analysis": "Deep dive into each competitor''s strategy, capabilities, and market position",
        "differentiation": "Identify unique value propositions and competitive advantages",
        "gap_analysis": "Find market gaps and unserved customer needs",
        "risk_assessment": "Evaluate competitive threats and market risks"
    }'::jsonb
),
(
    'Market Research Deep Dive',
    'Comprehensive market analysis for strategic decision-making',
    'business',
    'deep',
    10,
    '[
        {"name": "Market Overview", "required": true},
        {"name": "Market Size & Growth", "required": true, "subsections": ["Current Market Value", "Growth Projections", "Regional Analysis"]},
        {"name": "Customer Segmentation", "required": true},
        {"name": "Competitive Landscape", "required": true},
        {"name": "Technology Trends", "required": true},
        {"name": "Regulatory Environment", "required": false},
        {"name": "Investment Analysis", "required": true},
        {"name": "Risk Factors", "required": true},
        {"name": "Opportunities", "required": true},
        {"name": "Strategic Recommendations", "required": true}
    ]'::jsonb,
    30,
    '[
        "{{MARKET}} size revenue forecast",
        "{{MARKET}} growth rate CAGR analysis",
        "{{MARKET}} key players competitors",
        "{{MARKET}} customer demographics behavior",
        "{{MARKET}} technology trends innovations",
        "{{MARKET}} regulatory compliance requirements",
        "{{MARKET}} investment funding venture capital"
    ]'::jsonb,
    'You are a senior market research analyst preparing strategic intelligence for C-level executives. Provide comprehensive, data-backed analysis with clear visualizations of market dynamics. Focus on actionable insights for strategic planning.',
    '{
        "market_sizing": "Calculate TAM, SAM, and SOM with supporting data",
        "growth_drivers": "Identify and analyze key factors driving market growth",
        "competitive_dynamics": "Map competitive landscape and market positioning",
        "opportunity_identification": "Highlight specific opportunities for market entry or expansion"
    }'::jsonb
),
(
    'Technology Trend Analysis',
    'Analyze emerging technologies and their potential impact',
    'technology',
    'standard',
    5,
    '[
        {"name": "Trend Summary", "required": true},
        {"name": "Technology Deep Dive", "required": true},
        {"name": "Industry Applications", "required": true},
        {"name": "Adoption Timeline", "required": true},
        {"name": "Key Players", "required": true},
        {"name": "Challenges & Limitations", "required": true},
        {"name": "Investment Landscape", "required": false},
        {"name": "Future Outlook", "required": true}
    ]'::jsonb,
    20,
    '[
        "{{TECHNOLOGY}} latest developments breakthroughs",
        "{{TECHNOLOGY}} industry applications use cases",
        "{{TECHNOLOGY}} market adoption statistics",
        "{{TECHNOLOGY}} challenges limitations problems",
        "{{TECHNOLOGY}} investment funding trends"
    ]'::jsonb,
    'You are a technology analyst specializing in emerging tech trends. Provide balanced analysis covering both potential and limitations. Include concrete examples and case studies where available.',
    '{
        "technical_analysis": "Explain the technology in accessible terms while maintaining technical accuracy",
        "adoption_analysis": "Assess adoption readiness and barriers",
        "impact_assessment": "Evaluate potential disruption and transformation effects"
    }'::jsonb
),
(
    'Academic Literature Review',
    'Comprehensive review of academic research on a specific topic',
    'research',
    'deep',
    7,
    '[
        {"name": "Abstract", "required": true, "max_words": 250},
        {"name": "Introduction", "required": true},
        {"name": "Methodology", "required": true},
        {"name": "Key Findings", "required": true, "subsections": ["Consensus Views", "Contradictions", "Emerging Theories"]},
        {"name": "Critical Analysis", "required": true},
        {"name": "Research Gaps", "required": true},
        {"name": "Future Directions", "required": true},
        {"name": "References", "required": true}
    ]'::jsonb,
    40,
    '[
        "{{TOPIC}} systematic review meta-analysis",
        "{{TOPIC}} peer-reviewed research papers",
        "{{TOPIC}} academic journals publications",
        "{{TOPIC}} research methodology approaches",
        "{{TOPIC}} theoretical frameworks models"
    ]'::jsonb,
    'You are an academic researcher conducting a systematic literature review. Maintain scholarly rigor, cite sources properly, and provide critical analysis of methodologies and findings. Identify patterns, contradictions, and gaps in current research.',
    '{
        "methodology_critique": "Evaluate research methodologies and their limitations",
        "synthesis": "Synthesize findings across multiple studies",
        "gap_identification": "Identify specific gaps in current research",
        "theoretical_analysis": "Analyze theoretical frameworks and their applications"
    }'::jsonb
);

-- ==========================================
-- Example Report Schedules
-- ==========================================

INSERT INTO report_schedules (name, description, cron_expression, timezone, topic_template, depth, target_length, search_filters, custom_prompts, notification_emails) VALUES
(
    'Daily AI Intelligence Briefing',
    'Morning briefing on AI developments and breakthroughs',
    '0 9 * * *',
    'America/New_York',
    'Latest artificial intelligence and machine learning developments as of {{DATE}}',
    'standard',
    3,
    '{
        "time_range": "last_24_hours",
        "exclude_domains": ["blogspot.com", "medium.com"],
        "prefer_domains": ["arxiv.org", "openai.com", "anthropic.com", "deepmind.com", "nature.com", "science.org"]
    }'::jsonb,
    '{
        "focus_areas": ["research breakthroughs", "industry applications", "regulatory updates"],
        "tone": "professional but accessible"
    }'::jsonb,
    ARRAY['research-team@example.com']
),
(
    'Weekly Competitive Analysis',
    'Weekly competitive intelligence report for strategic planning',
    '0 10 * * 1',
    'America/New_York',
    'Competitive intelligence for technology sector - Week of {{WEEK_START}}',
    'deep',
    8,
    '{
        "time_range": "last_7_days",
        "focus_companies": ["OpenAI", "Anthropic", "Google", "Microsoft", "Meta"],
        "include_financial": true
    }'::jsonb,
    '{
        "emphasis": ["product launches", "strategic partnerships", "funding rounds", "executive changes"],
        "comparison_framework": "SWOT analysis"
    }'::jsonb,
    ARRAY['strategy@example.com', 'executives@example.com']
),
(
    'Monthly Market Trends Report',
    'Comprehensive monthly analysis of market trends and opportunities',
    '0 8 1 * *',
    'America/Los_Angeles',
    'Market trends and analysis for {{MONTH}} {{YEAR}}',
    'deep',
    10,
    '{
        "time_range": "last_30_days",
        "geographic_focus": ["North America", "Europe", "Asia-Pacific"],
        "data_sources": ["government", "industry reports", "financial data"]
    }'::jsonb,
    '{
        "sections_emphasis": ["emerging opportunities", "risk factors", "investment trends"],
        "include_forecasts": true,
        "quantitative_analysis": true
    }'::jsonb,
    ARRAY['investors@example.com', 'board@example.com']
);

-- ==========================================
-- Sample User Preferences
-- ==========================================

INSERT INTO user_preferences (user_id, theme, language, timezone, default_depth, default_length, email_notifications, preferred_search_engines, blocked_domains, chat_personality, show_sources) VALUES
(
    'default-user',
    'light',
    'en',
    'America/New_York',
    'standard',
    5,
    true,
    ARRAY['google', 'bing', 'duckduckgo', 'arxiv', 'scholar'],
    ARRAY['pinterest.com', 'facebook.com'],
    'professional',
    true
),
(
    'research-team',
    'dark',
    'en',
    'Europe/London',
    'deep',
    8,
    true,
    ARRAY['scholar', 'arxiv', 'pubmed', 'semantic_scholar'],
    ARRAY['blogspot.com', 'tumblr.com'],
    'concise',
    true
);

-- ==========================================
-- Sample Completed Report (for demonstration)
-- ==========================================

INSERT INTO reports (
    title,
    topic,
    depth,
    target_length,
    markdown_content,
    summary,
    key_findings,
    sources_count,
    word_count,
    confidence_score,
    status,
    requested_at,
    completed_at,
    processing_time_seconds,
    requested_by,
    tags,
    category
) VALUES
(
    'AI Language Models: December 2024 Progress Report',
    'Latest developments in large language models and their applications',
    'standard',
    5,
    '# AI Language Models: December 2024 Progress Report

## Executive Summary

The field of large language models (LLMs) continues to advance rapidly with significant breakthroughs in efficiency, capability, and application domains...

## Key Developments

### 1. Model Efficiency Improvements
Recent research has demonstrated 40% reduction in computational requirements while maintaining performance...

### 2. Multi-Modal Integration
New architectures successfully integrate vision, audio, and text processing in unified models...

### 3. Reasoning Capabilities
Enhanced chain-of-thought prompting and structured reasoning show promising results...

## Market Impact

The LLM market is projected to reach $40 billion by 2025, with enterprise adoption accelerating...

## Future Implications

These developments suggest a trajectory toward more capable, efficient, and specialized AI systems...',
    'Comprehensive analysis of recent LLM developments showing significant progress in efficiency (40% reduction in compute), multi-modal capabilities, and reasoning. Market growth remains strong with enterprise adoption accelerating.',
    '[
        {"finding": "40% reduction in computational requirements achieved", "confidence": 0.92, "sources": 5},
        {"finding": "Multi-modal models now production-ready", "confidence": 0.88, "sources": 8},
        {"finding": "Enterprise adoption increased 3x year-over-year", "confidence": 0.95, "sources": 12}
    ]'::jsonb,
    23,
    2847,
    0.91,
    'completed',
    NOW() - INTERVAL '2 hours',
    NOW() - INTERVAL '1 hour 45 minutes',
    900,
    'research-team',
    ARRAY['ai', 'llm', 'technology', 'research'],
    'technology'
);

-- ==========================================
-- Sample Chat Conversation
-- ==========================================

INSERT INTO chat_conversations (id, title, user_id, is_active, message_count) VALUES
(
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
    'Discussion about AI trends',
    'default-user',
    true,
    2
);

INSERT INTO chat_messages (conversation_id, role, content, confidence_score) VALUES
(
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
    'user',
    'Can you summarize the latest developments in AI language models?',
    NULL
),
(
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
    'assistant',
    'Based on our latest research report from December 2024, here are the key developments in AI language models:

1. **Efficiency Breakthroughs**: Researchers have achieved a 40% reduction in computational requirements while maintaining model performance, making LLMs more accessible.

2. **Multi-Modal Integration**: Production-ready models now seamlessly integrate vision, audio, and text processing capabilities.

3. **Enhanced Reasoning**: New chain-of-thought prompting techniques have significantly improved structured reasoning capabilities.

4. **Market Growth**: The LLM market is on track to reach $40 billion by 2025, with enterprise adoption increasing 3x year-over-year.

These developments indicate a strong trajectory toward more capable and efficient AI systems. Would you like me to generate a detailed report on any specific aspect?',
    0.91
);

-- ==========================================
-- Update schedule next run times
-- ==========================================

UPDATE report_schedules 
SET next_run_at = 
    CASE 
        WHEN cron_expression = '0 9 * * *' THEN date_trunc('day', NOW()) + INTERVAL '1 day 9 hours'
        WHEN cron_expression = '0 10 * * 1' THEN date_trunc('week', NOW()) + INTERVAL '1 week 10 hours'
        WHEN cron_expression = '0 8 1 * *' THEN date_trunc('month', NOW()) + INTERVAL '1 month 8 hours'
        ELSE NOW() + INTERVAL '1 day'
    END;