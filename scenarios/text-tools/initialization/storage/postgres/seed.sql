-- Text Tools Seed Data
-- Version: 1.0.0
-- Description: Initial templates, rules, and sample data

SET search_path TO text_tools, public;

-- Insert default text templates
INSERT INTO text_templates (name, category, content, variables, description, tags) VALUES
('markdown_to_html', 'conversion', '<!DOCTYPE html>
<html>
<head>
    <title>{{title}}</title>
    <style>{{styles}}</style>
</head>
<body>
    {{content}}
</body>
</html>', 
'[{"name": "title", "type": "string", "required": true}, {"name": "styles", "type": "string", "required": false}, {"name": "content", "type": "string", "required": true}]'::jsonb,
'Basic HTML template for markdown conversion', ARRAY['html', 'markdown', 'conversion']),

('json_api_response', 'api', '{
  "success": {{success}},
  "data": {{data}},
  "message": "{{message}}",
  "timestamp": "{{timestamp}}"
}',
'[{"name": "success", "type": "boolean", "required": true}, {"name": "data", "type": "any", "required": false}, {"name": "message", "type": "string", "required": false}, {"name": "timestamp", "type": "string", "required": true}]'::jsonb,
'Standard API response template', ARRAY['api', 'json', 'response']),

('email_template', 'communication', 'Subject: {{subject}}

Dear {{recipient_name}},

{{body}}

Best regards,
{{sender_name}}
{{sender_title}}',
'[{"name": "subject", "type": "string", "required": true}, {"name": "recipient_name", "type": "string", "required": true}, {"name": "body", "type": "string", "required": true}, {"name": "sender_name", "type": "string", "required": true}, {"name": "sender_title", "type": "string", "required": false}]'::jsonb,
'Basic email template', ARRAY['email', 'communication']),

('code_documentation', 'documentation', '/**
 * {{function_name}}
 * 
 * {{description}}
 * 
 * @param {{parameters}}
 * @returns {{return_type}} {{return_description}}
 * @example
 * {{example}}
 */
',
'[{"name": "function_name", "type": "string", "required": true}, {"name": "description", "type": "string", "required": true}, {"name": "parameters", "type": "string", "required": true}, {"name": "return_type", "type": "string", "required": true}, {"name": "return_description", "type": "string", "required": false}, {"name": "example", "type": "string", "required": false}]'::jsonb,
'JSDoc style code documentation template', ARRAY['code', 'documentation', 'jsdoc']);

-- Insert default extraction rules
INSERT INTO text_extraction_rules (name, source_format, rule_type, pattern, description, is_active) VALUES
('extract_emails', 'plain', 'regex', '[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}', 'Extract email addresses from text', true),
('extract_urls', 'plain', 'regex', 'https?://[^\s<>"{}|\\^`\[\]]+', 'Extract URLs from text', true),
('extract_phone_numbers', 'plain', 'regex', '(\+?[1-9]\d{0,2}[-.\s]?)?(\(?\d{3}\)?[-.\s]?)?\d{3}[-.\s]?\d{4}', 'Extract phone numbers (US format)', true),
('extract_dates', 'plain', 'regex', '\b(\d{1,2}[/-]\d{1,2}[/-]\d{2,4}|\d{4}[/-]\d{1,2}[/-]\d{1,2})\b', 'Extract dates in various formats', true),
('extract_json_values', 'json', 'json_path', '$..value', 'Extract all "value" fields from JSON', true),
('extract_html_text', 'html', 'css_selector', 'p, h1, h2, h3, h4, h5, h6, li', 'Extract text content from HTML', true),
('extract_markdown_headers', 'markdown', 'regex', '^#{1,6}\s+(.+)$', 'Extract headers from markdown', true),
('extract_code_blocks', 'markdown', 'regex', '```[\s\S]*?```', 'Extract code blocks from markdown', true);

-- Insert sample text processing pipelines
INSERT INTO text_pipelines (name, description, steps, input_format, output_format, is_public) VALUES
('clean_and_analyze', 'Clean text and perform basic analysis', 
'[
  {"type": "sanitize", "params": {"remove_html": true, "normalize_whitespace": true}},
  {"type": "transform", "params": {"case": "lower"}},
  {"type": "analyze", "params": {"extract": ["entities", "keywords"]}}
]'::jsonb,
'html', 'json', true),

('document_summary', 'Extract and summarize document content',
'[
  {"type": "extract", "params": {"format": "auto"}},
  {"type": "analyze", "params": {"summary_length": 200}},
  {"type": "transform", "params": {"format": "markdown"}}
]'::jsonb,
'any', 'markdown', true),

('code_documentation_generator', 'Generate documentation from code',
'[
  {"type": "extract", "params": {"pattern": "function|class|method"}},
  {"type": "analyze", "params": {"extract": ["structure", "complexity"]}},
  {"type": "template", "params": {"template": "code_documentation"}}
]'::jsonb,
'plain', 'markdown', true);

-- Insert sample API usage stats (for testing dashboard)
INSERT INTO api_usage_stats (endpoint, method, scenario_name, request_size_bytes, response_size_bytes, duration_ms, status_code) VALUES
('/api/v1/text/diff', 'POST', 'test-scenario', 1024, 2048, 45, 200),
('/api/v1/text/search', 'POST', 'notes', 512, 1024, 23, 200),
('/api/v1/text/extract', 'POST', 'document-manager', 5120, 10240, 156, 200),
('/api/v1/text/analyze', 'POST', 'research-assistant', 2048, 4096, 234, 200),
('/api/v1/text/transform', 'POST', 'code-smell', 1536, 1536, 12, 200);

-- Create default admin notifications
INSERT INTO text_documents (name, content, format, size_bytes, language, metadata) VALUES
('Welcome to Text Tools', 'Text Tools is now initialized and ready to process text for all Vrooli scenarios.', 'plain', 79, 'en', 
'{"type": "system", "category": "notification"}'::jsonb),
('API Documentation', '# Text Tools API

## Endpoints
- POST /api/v1/text/diff - Compare texts
- POST /api/v1/text/search - Search patterns
- POST /api/v1/text/transform - Transform text
- POST /api/v1/text/extract - Extract from documents
- POST /api/v1/text/analyze - Analyze with NLP

See full documentation at /docs', 'markdown', 245, 'en',
'{"type": "documentation", "category": "api"}'::jsonb);

-- Refresh materialized view
REFRESH MATERIALIZED VIEW operation_statistics;