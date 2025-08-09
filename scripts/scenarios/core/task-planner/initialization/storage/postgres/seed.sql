-- Task Planner Seed Data
-- Sample apps, templates, and tasks for demonstration

-- Sample Applications
INSERT INTO apps (id, name, display_name, type, repository_url, documentation_url, api_token, auto_implement, metadata) VALUES
(
  'e-commerce-app',
  'ecommerce_platform',
  'E-Commerce Platform',
  'generated',
  'https://github.com/example/ecommerce-platform',
  'https://docs.example.com/ecommerce',
  'eca_' || encode(gen_random_bytes(16), 'hex'),
  true,
  '{"tech_stack": ["React", "Node.js", "PostgreSQL"], "business_model": "B2C", "target_users": 10000}'
),
(
  'blog-system',
  'blog_cms',
  'Blog Content Management System',
  'generated', 
  'https://github.com/example/blog-cms',
  'https://docs.example.com/blog-cms',
  'bcs_' || encode(gen_random_bytes(16), 'hex'),
  false,
  '{"tech_stack": ["Vue.js", "Express", "MongoDB"], "features": ["SEO", "Analytics", "Comments"]}'
),
(
  'task-automation',
  'workflow_engine',
  'Task Automation Workflow Engine',
  'vrooli',
  'https://github.com/example/workflow-engine',
  'https://docs.example.com/workflows',
  'tae_' || encode(gen_random_bytes(16), 'hex'),
  true,
  '{"tech_stack": ["TypeScript", "Node.js", "Redis", "PostgreSQL"], "integrations": ["n8n", "Zapier"]}'
),
(
  'ai-assistant',
  'ai_chat_assistant',
  'AI Chat Assistant',
  'scenario',
  'https://github.com/example/ai-assistant',
  'https://docs.example.com/ai-assistant',
  'aia_' || encode(gen_random_bytes(16), 'hex'),
  true,
  '{"ai_models": ["GPT-4", "Claude-3"], "capabilities": ["conversation", "task_execution", "code_generation"]}'
);

-- Sample Task Templates
INSERT INTO task_templates (id, app_id, name, description, title_template, description_template, investigation_template, implementation_template, default_priority, default_tags, default_labels) VALUES
(
  uuid_generate_v4(),
  'e-commerce-app',
  'Add Payment Method',
  'Template for adding new payment integrations',
  'Add {payment_provider} payment integration',
  'Implement {payment_provider} payment processing with secure transaction handling, error management, and user-friendly checkout flow.',
  'Research {payment_provider} API documentation, security requirements, compliance needs (PCI DSS), error handling patterns, and integration complexity. Evaluate sandbox testing capabilities and production requirements.',
  '1. Set up {payment_provider} developer account and API credentials\n2. Install and configure SDK/libraries\n3. Implement payment form components\n4. Add backend payment processing endpoints\n5. Implement webhook handling for payment events\n6. Add comprehensive error handling and user feedback\n7. Implement security measures and input validation\n8. Add unit and integration tests\n9. Test in sandbox environment\n10. Document integration and deployment steps',
  'high',
  ARRAY['payment', 'integration', 'security', 'backend', 'frontend'],
  '{"complexity": "high", "security_critical": true, "external_dependency": true}'
),
(
  uuid_generate_v4(),
  'blog-system',
  'SEO Enhancement',
  'Template for SEO-related improvements',
  'Improve SEO: {seo_aspect}',
  'Enhance search engine optimization by implementing {seo_aspect} to improve search visibility and organic traffic.',
  'Research current SEO best practices for {seo_aspect}, analyze competitor implementations, evaluate available tools and libraries, and assess impact on page performance and user experience.',
  '1. Audit current {seo_aspect} implementation\n2. Research and select appropriate tools/libraries\n3. Implement {seo_aspect} enhancements\n4. Update content templates and components\n5. Add structured data markup if applicable\n6. Implement performance optimizations\n7. Add SEO testing and validation\n8. Update documentation with SEO guidelines\n9. Monitor and measure improvement',
  'medium',
  ARRAY['seo', 'content', 'performance', 'marketing'],
  '{"impact": "medium", "measurable": true, "content_related": true}'
),
(
  uuid_generate_v4(),
  'task-automation',
  'Workflow Integration',
  'Template for adding new workflow integrations',
  'Add {service_name} workflow integration',
  'Integrate {service_name} to expand workflow automation capabilities and enable seamless data flow between systems.',
  'Research {service_name} API capabilities, authentication methods, rate limiting, data formats, error handling, and integration patterns. Evaluate compatibility with existing workflow engine architecture.',
  '1. Study {service_name} API documentation and authentication\n2. Design integration architecture and data flow\n3. Implement authentication and connection handling\n4. Create workflow nodes/components for {service_name}\n5. Add error handling and retry logic\n6. Implement rate limiting and request optimization\n7. Add comprehensive logging and monitoring\n8. Create unit and integration tests\n9. Document integration setup and usage\n10. Add configuration management',
  'medium',
  ARRAY['integration', 'workflow', 'api', 'automation'],
  '{"integration_type": "api", "complexity": "medium", "reusable": true}'
),
(
  uuid_generate_v4(),
  'ai-assistant',
  'AI Model Integration',
  'Template for integrating new AI models',
  'Integrate {ai_model} for {capability}',
  'Add support for {ai_model} to enhance {capability} and provide users with improved AI assistance capabilities.',
  'Research {ai_model} capabilities, API requirements, pricing structure, rate limits, context windows, and integration complexity. Evaluate performance characteristics and suitability for {capability} use cases.',
  '1. Set up {ai_model} API access and authentication\n2. Design model abstraction layer for compatibility\n3. Implement {ai_model} client and connection handling\n4. Add prompt engineering and optimization\n5. Implement response processing and validation\n6. Add error handling and fallback mechanisms\n7. Implement usage tracking and cost monitoring\n8. Add comprehensive testing with various inputs\n9. Optimize for performance and cost efficiency\n10. Document usage patterns and best practices',
  'high',
  ARRAY['ai', 'model', 'integration', 'capability'],
  '{"ai_related": true, "complexity": "high", "cost_impact": true}'
);

-- Sample Tasks in Various Stages
INSERT INTO tasks (id, app_id, title, description, status, priority, tags, estimated_hours, confidence_score, metadata) VALUES
-- E-commerce App Tasks
(
  uuid_generate_v4(),
  'e-commerce-app',
  'Implement shopping cart persistence',
  'Add functionality to save cart contents across browser sessions using localStorage and sync with user account when logged in',
  'backlog',
  'high',
  ARRAY['frontend', 'cart', 'persistence', 'user-experience'],
  6.0,
  null,
  '{"category": "feature", "user_impact": "high", "technical_debt": false}'
),
(
  uuid_generate_v4(),
  'e-commerce-app', 
  'Add product recommendation engine',
  'Implement AI-powered product recommendations based on user behavior, purchase history, and similar user patterns',
  'backlog',
  'medium',
  ARRAY['ai', 'recommendation', 'analytics', 'personalization'],
  20.0,
  null,
  '{"category": "feature", "complexity": "high", "requires_data": true}'
),
(
  uuid_generate_v4(),
  'e-commerce-app',
  'Fix checkout flow timeout issue',
  'Resolve timeout errors during checkout process that occur when payment processing takes longer than 30 seconds',
  'staged',
  'critical',
  ARRAY['bug', 'checkout', 'payment', 'timeout'],
  4.0,
  0.85,
  '{"category": "bug", "severity": "critical", "affects_revenue": true}'
),

-- Blog System Tasks  
(
  uuid_generate_v4(),
  'blog-system',
  'Add social media sharing buttons',
  'Implement shareable links for Facebook, Twitter, LinkedIn, and other social platforms with proper meta tags',
  'completed',
  'medium',
  ARRAY['social', 'sharing', 'engagement', 'seo'],
  3.0,
  0.9,
  '{"category": "feature", "completion_date": "2024-01-15", "user_requested": true}'
),
(
  uuid_generate_v4(),
  'blog-system',
  'Implement comment moderation system',
  'Add administrator interface for reviewing, approving, and managing user comments with spam detection',
  'in_progress',
  'medium', 
  ARRAY['moderation', 'comments', 'admin', 'spam-detection'],
  12.0,
  0.75,
  '{"category": "feature", "started_date": "2024-01-10", "progress": 60}'
),
(
  uuid_generate_v4(),
  'blog-system',
  'Optimize image loading performance',
  'Implement lazy loading, image compression, and WebP format support to improve page load times',
  'backlog',
  'medium',
  ARRAY['performance', 'images', 'optimization', 'lazy-loading'],
  8.0,
  null,
  '{"category": "performance", "impact": "medium", "seo_benefit": true}'
),

-- Task Automation Engine Tasks
(
  uuid_generate_v4(),
  'task-automation',
  'Add Slack integration for notifications',
  'Create workflow nodes for sending Slack messages, creating channels, and managing team notifications',
  'staged',
  'high',
  ARRAY['integration', 'slack', 'notifications', 'workflow'],
  10.0,
  0.8,
  '{"category": "integration", "external_service": "Slack", "user_demand": "high"}'
),
(
  uuid_generate_v4(),
  'task-automation',
  'Implement workflow scheduling system',
  'Add cron-like scheduling capabilities for recurring workflows with timezone support and failure handling',
  'backlog',
  'high',
  ARRAY['scheduling', 'cron', 'automation', 'recurring'],
  15.0,
  null,
  '{"category": "feature", "complexity": "high", "core_functionality": true}'
),

-- AI Assistant Tasks
(
  uuid_generate_v4(),
  'ai-assistant',
  'Add code execution sandbox',
  'Implement secure code execution environment for running user-generated code with proper isolation and timeouts',
  'in_progress',
  'critical',
  ARRAY['security', 'sandbox', 'code-execution', 'isolation'],
  25.0,
  0.7,
  '{"category": "feature", "security_critical": true, "complexity": "very_high"}'
),
(
  uuid_generate_v4(),
  'ai-assistant',
  'Improve conversation memory management',
  'Optimize how conversation context is stored and retrieved to handle longer conversations efficiently',
  'completed',
  'medium',
  ARRAY['memory', 'conversation', 'optimization', 'context'],
  8.0,
  0.9,
  '{"category": "optimization", "performance_impact": true, "completed_date": "2024-01-12"}'
);

-- Add some completed tasks with implementation details
UPDATE tasks SET 
  investigation_report = 'Social media sharing is a critical feature for content virality and user engagement. Research shows that adding native sharing buttons can increase social engagement by 25-40%. Key requirements include: proper Open Graph and Twitter Card meta tags, tracking of share events for analytics, mobile-responsive design, and support for the most popular platforms. Technical approach will use native Web Share API where available, falling back to platform-specific URL schemes.',
  implementation_plan = '1. Research and select social sharing library or implement native solution\n2. Design responsive button layout and icons\n3. Implement Open Graph and Twitter Card meta tag generation\n4. Add sharing buttons component with click tracking\n5. Implement Web Share API with graceful fallbacks\n6. Add analytics integration for share event tracking\n7. Test across different devices and platforms\n8. Add configuration for enabling/disabling specific platforms\n9. Update content templates to include sharing buttons\n10. Document usage and customization options',
  technical_requirements = 'Frontend: React/Vue component library, CSS frameworks, social media platform APIs. Backend: Meta tag generation, analytics tracking, configuration management. Third-party: Google Analytics or similar for tracking, optional social sharing service. Performance: Lazy loading of sharing scripts, minimal impact on page load time.',
  implementation_code = 'import React from \'react\';\nimport { trackEvent } from \'../analytics\';\n\nconst SocialShare = ({ title, url, description }) => {\n  const shareData = { title, url, text: description };\n  \n  const handleShare = async (platform) => {\n    if (navigator.share && platform === \'native\') {\n      await navigator.share(shareData);\n    } else {\n      const shareUrl = generatePlatformUrl(platform, shareData);\n      window.open(shareUrl, \'_blank\', \'width=600,height=400\');\n    }\n    trackEvent(\'social_share\', { platform, url });\n  };\n  \n  return (\n    <div className="social-share">\n      <button onClick={() => handleShare(\'facebook\')}>Facebook</button>\n      <button onClick={() => handleShare(\'twitter\')}>Twitter</button>\n      <button onClick={() => handleShare(\'linkedin\')}>LinkedIn</button>\n      {navigator.share && (\n        <button onClick={() => handleShare(\'native\')}>Share</button>\n      )}\n    </div>\n  );\n};\n\nexport default SocialShare;',
  implementation_result = 'Successfully implemented social media sharing buttons with Web Share API support, proper meta tags, and analytics tracking. Component is reusable and configurable. Testing completed across major platforms and devices.',
  test_results = 'All tests passing: Component rendering (✓), Share URL generation (✓), Analytics tracking (✓), Web Share API detection (✓), Mobile responsiveness (✓), Cross-browser compatibility (✓)',
  completed_at = '2024-01-15 14:30:00'
WHERE title = 'Add social media sharing buttons';

UPDATE tasks SET
  investigation_report = 'Conversation memory management is crucial for AI assistant performance and user experience. Current implementation stores full conversation history in memory, leading to exponential memory usage and slower response times as conversations grow. Research into sliding window approaches, conversation summarization, and context compression techniques shows potential for 60-80% memory reduction while maintaining conversation quality. Key challenges include context preservation, important information retention, and seamless user experience during context switches.',
  implementation_plan = '1. Analyze current memory usage patterns and conversation lengths\n2. Research and prototype sliding window approaches\n3. Implement conversation summarization using AI models\n4. Design context compression algorithm preserving key information\n5. Create memory management service with configurable policies\n6. Implement gradual memory optimization (don\'t truncate mid-conversation)\n7. Add conversation quality metrics and monitoring\n8. Test with various conversation lengths and topics\n9. Optimize for both memory usage and response quality\n10. Deploy with gradual rollout and monitoring',
  technical_requirements = 'Backend: Node.js memory management, Redis for conversation storage, AI model for summarization. Algorithms: Sliding window, text summarization, importance scoring. Monitoring: Memory usage metrics, conversation quality scores, response time tracking. Infrastructure: Scalable Redis cluster, monitoring dashboards.',
  implementation_code = 'class ConversationMemoryManager {\n  constructor(options = {}) {\n    this.maxTokens = options.maxTokens || 4000;\n    this.windowSize = options.windowSize || 20;\n    this.summaryModel = options.summaryModel || \'gpt-3.5-turbo\';\n  }\n  \n  async optimizeMemory(conversation) {\n    if (this.getTokenCount(conversation) <= this.maxTokens) {\n      return conversation;\n    }\n    \n    const recentMessages = conversation.slice(-this.windowSize);\n    const olderMessages = conversation.slice(0, -this.windowSize);\n    \n    if (olderMessages.length === 0) return recentMessages;\n    \n    const summary = await this.summarizeMessages(olderMessages);\n    return [{ role: \'system\', content: summary }, ...recentMessages];\n  }\n  \n  async summarizeMessages(messages) {\n    const text = messages.map(m => `${m.role}: ${m.content}`).join(\'\\n\');\n    return await this.summaryModel.summarize(text, {\n      maxLength: 200,\n      preserveKey: true\n    });\n  }\n}',
  implementation_result = 'Implemented intelligent conversation memory management with 70% memory reduction and improved response times. Sliding window with AI summarization maintains conversation context while optimizing resource usage.',
  test_results = 'Memory usage tests: 70% reduction achieved (✓), Response quality maintained: 95% user satisfaction (✓), Performance improvement: 45% faster responses (✓), Long conversation handling (✓), Context preservation accuracy: 92% (✓)',
  completed_at = '2024-01-12 16:45:00'
WHERE title = 'Improve conversation memory management';

-- Sample Task Transitions (Activity Log)
INSERT INTO task_transitions (task_id, from_status, to_status, triggered_by, trigger_type, reason, created_at) VALUES
((SELECT id FROM tasks WHERE title = 'Add social media sharing buttons'), 'unstructured', 'backlog', 'text-parser-workflow', 'automated', 'Task parsed from user input', '2024-01-10 09:15:00'),
((SELECT id FROM tasks WHERE title = 'Add social media sharing buttons'), 'backlog', 'staged', 'research-workflow', 'automated', 'Research and planning completed successfully', '2024-01-12 14:22:00'),
((SELECT id FROM tasks WHERE title = 'Add social media sharing buttons'), 'staged', 'in_progress', 'implementation-workflow', 'automated', 'Implementation started by Claude Code', '2024-01-15 10:30:00'),
((SELECT id FROM tasks WHERE title = 'Add social media sharing buttons'), 'in_progress', 'completed', 'implementation-workflow', 'automated', 'Implementation completed successfully', '2024-01-15 14:30:00'),

((SELECT id FROM tasks WHERE title = 'Improve conversation memory management'), 'backlog', 'staged', 'research-workflow', 'automated', 'Research completed with high confidence', '2024-01-08 11:45:00'),
((SELECT id FROM tasks WHERE title = 'Improve conversation memory management'), 'staged', 'in_progress', 'windmill-ui', 'manual', 'Started implementation after review', '2024-01-10 09:00:00'),
((SELECT id FROM tasks WHERE title = 'Improve conversation memory management'), 'in_progress', 'completed', 'implementation-workflow', 'automated', 'Memory optimization implementation completed', '2024-01-12 16:45:00');

-- Sample Agent Runs (AI Processing History)
INSERT INTO agent_runs (task_id, action, agent_name, model_used, successful, tokens_used, execution_time_ms, cost_usd, confidence, started_at, completed_at) VALUES
((SELECT id FROM tasks WHERE title = 'Add social media sharing buttons'), 'parse', 'text-parser', 'llama3.2', true, 450, 2300, 0.0012, 0.85, '2024-01-10 09:15:00', '2024-01-10 09:15:02'),
((SELECT id FROM tasks WHERE title = 'Add social media sharing buttons'), 'research', 'research-coordinator', 'multi-agent', true, 2800, 45000, 0.0420, 0.90, '2024-01-12 14:20:00', '2024-01-12 14:22:15'),
((SELECT id FROM tasks WHERE title = 'Add social media sharing buttons'), 'implement', 'claude-code', 'claude-3-opus-20240229', true, 6200, 180000, 0.0930, 0.92, '2024-01-15 10:30:00', '2024-01-15 14:28:00'),

((SELECT id FROM tasks WHERE title = 'Improve conversation memory management'), 'research', 'research-coordinator', 'multi-agent', true, 3200, 52000, 0.0480, 0.88, '2024-01-08 11:40:00', '2024-01-08 11:45:20'),
((SELECT id FROM tasks WHERE title = 'Improve conversation memory management'), 'implement', 'claude-code', 'claude-3-opus-20240229', true, 8500, 380000, 0.1275, 0.89, '2024-01-10 09:00:00', '2024-01-12 16:43:20');

-- Sample Research Artifacts
INSERT INTO research_artifacts (task_id, type, source_url, title, content, relevance_score, metadata) VALUES
((SELECT id FROM tasks WHERE title = 'Add social media sharing buttons'), 'documentation', 'https://developer.mozilla.org/en-US/docs/Web/API/Web_Share_API', 'Web Share API Documentation', 'The Web Share API allows web apps to share content using the platform''s native sharing mechanisms. Supported on mobile browsers and progressive web apps.', 0.95, '{"source": "MDN", "technology": "Web API"}'),
((SELECT id FROM tasks WHERE title = 'Add social media sharing buttons'), 'tutorial', 'https://css-tricks.com/simple-social-sharing-links/', 'Simple Social Sharing Links', 'Comprehensive guide to implementing social sharing buttons without heavy JavaScript libraries. Covers URL formats for major platforms.', 0.88, '{"source": "CSS-Tricks", "approach": "lightweight"}'),
((SELECT id FROM tasks WHERE title = 'Add social media sharing buttons'), 'code_example', 'https://github.com/nygardk/react-share', 'React Share Library', 'Popular React library for social media sharing components with 2.8k GitHub stars. Provides pre-built components for all major platforms.', 0.82, '{"github_stars": 2800, "language": "React"}'),

((SELECT id FROM tasks WHERE title = 'Improve conversation memory management'), 'documentation', 'https://platform.openai.com/docs/guides/gpt/managing-tokens', 'Managing Tokens in GPT Models', 'Official OpenAI documentation on token limits, context windows, and strategies for handling long conversations in AI applications.', 0.98, '{"source": "OpenAI", "topic": "token_management"}'),
((SELECT id FROM tasks WHERE title = 'Improve conversation memory management'), 'research_paper', 'https://arxiv.org/abs/2304.03442', 'Efficient Memory Management in Conversational AI', 'Academic paper discussing sliding window approaches, conversation summarization, and memory optimization techniques for chat systems.', 0.91, '{"paper_type": "arxiv", "citations": 45}'),
((SELECT id FROM tasks WHERE title = 'Improve conversation memory management'), 'code_example', 'https://github.com/microsoft/DialoGPT', 'DialoGPT Memory Implementation', 'Microsoft''s implementation of conversation memory management in their DialoGPT model, including context compression algorithms.', 0.85, '{"organization": "Microsoft", "implementation": "production"}');

-- Update app statistics
UPDATE apps SET 
  total_tasks = (SELECT COUNT(*) FROM tasks WHERE app_id = apps.id),
  completed_tasks = (SELECT COUNT(*) FROM tasks WHERE app_id = apps.id AND status = 'completed'),
  avg_completion_hours = (SELECT AVG(EXTRACT(EPOCH FROM (completed_at - created_at)) / 3600) FROM tasks WHERE app_id = apps.id AND status = 'completed')
WHERE id IN ('e-commerce-app', 'blog-system', 'task-automation', 'ai-assistant');

-- Create some unstructured sessions for demonstration
INSERT INTO unstructured_sessions (id, app_id, raw_text, input_type, processed, tasks_extracted, submitted_by, submitted_at, processed_at) VALUES
(
  uuid_generate_v4(),
  'e-commerce-app',
  '- Add wishlist functionality\n- Fix mobile checkout UI\n- Implement inventory alerts\nTODO: Add product reviews system\nIdea: What about a loyalty program?',
  'markdown',
  true,
  5,
  'product_manager',
  '2024-01-08 10:30:00',
  '2024-01-08 10:30:15'
),
(
  uuid_generate_v4(),
  'blog-system',
  'Need to improve the blog performance and SEO. Some ideas:\n\n1. Add image compression\n2. Implement lazy loading\n3. Better meta tags\n4. Schema markup for articles\n5. Sitemap generation\n\nAlso the comment system needs spam protection.',
  'plaintext',
  true,
  6,
  'marketing_team',
  '2024-01-10 14:20:00',
  '2024-01-10 14:20:08'
),
(
  uuid_generate_v4(),
  'ai-assistant',
  'High priority bugs to fix:\n- Memory leaks in long conversations\n- API timeout issues\n- Error handling for model failures\n\nNew features requested:\n- File upload support\n- Better code syntax highlighting\n- Conversation export functionality',
  'structured_list',
  false,
  0,
  'development_team',
  '2024-01-15 16:45:00',
  null
);

-- Add some sample similar task relationships (simulated embedding similarity)
-- In a real system, these would be calculated by vector similarity
-- This is just for demonstration purposes

COMMIT;

-- Analyze tables for better query performance
ANALYZE apps;
ANALYZE tasks;
ANALYZE unstructured_sessions;
ANALYZE agent_runs;
ANALYZE task_transitions;
ANALYZE research_artifacts;
ANALYZE task_templates;