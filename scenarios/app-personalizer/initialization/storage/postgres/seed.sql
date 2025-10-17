-- App Personalizer Seed Data
-- Sample data for testing and development

-- Sample modification templates
INSERT INTO modification_templates (name, category, framework, description, modification_pattern, required_inputs) VALUES
('React Theme Colors', 'ui_theme', 'react', 'Change primary and secondary colors in React theme', 
 '{"files": ["src/styles/theme.js"], "pattern": "colors: {primary: \\"{{primary_color}}\\", secondary: \\"{{secondary_color}}\\"}"}',
 '["primary_color", "secondary_color"]'),

('Vue Brand Assets', 'branding', 'vue', 'Update logo and favicon for Vue apps',
 '{"files": ["public/logo.png", "public/favicon.ico"], "pattern": "replace_assets"}',
 '["logo_url", "favicon_url"]'),

('Content Personalization', 'content', 'react', 'Personalize welcome messages and default content',
 '{"files": ["src/data/defaults.js"], "pattern": "export const defaults = {welcomeMessage: \\"{{welcome_message}}\\", userName: \\"{{user_name}}\\"}"}',
 '["welcome_message", "user_name"]'),

('AI Personality Config', 'behavior', 'react', 'Configure AI assistant personality and interaction style',
 '{"files": ["src/config/ai.config.js"], "pattern": "personality: \\"{{personality_type}}\\", style: \\"{{interaction_style}}\\"}"}',
 '["personality_type", "interaction_style"]');

-- Sample validation rules
INSERT INTO validation_rules (rule_name, rule_type, framework, validation_command, expected_output_pattern, timeout_seconds, is_critical) VALUES
('React Build', 'build', 'react', 'npm run build', 'Build completed successfully', 300, true),
('React Lint', 'lint', 'react', 'npm run lint', 'No linting errors found', 60, false),
('React Test', 'test', 'react', 'npm test -- --watchAll=false', 'Tests passed', 120, false),
('Vue Build', 'build', 'vue', 'npm run build', 'Build complete', 300, true),
('Vue Lint', 'lint', 'vue', 'npm run lint', 'All files pass linting', 60, false),
('Next.js Build', 'build', 'next.js', 'npm run build', 'Compiled successfully', 300, true);

-- Sample integration configurations (for testing)
INSERT INTO integrations (service_name, endpoint_url, auth_type, connection_status, rate_limit_config) VALUES
('personal-digital-twin', 'http://localhost:8200', 'api_token', 'active', '{"requests_per_minute": 60, "burst": 10}'),
('brand-manager', 'http://localhost:8100', 'api_key', 'active', '{"requests_per_minute": 100, "burst": 20}'),
('claude-code', 'http://claude-code:8080', 'none', 'active', '{"requests_per_minute": 30, "burst": 5}');

-- Create indexes for performance (if they don't exist from schema)
DO $$ 
BEGIN
    -- Check if index exists before creating
    IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = 'idx_modification_templates_category') THEN
        CREATE INDEX idx_modification_templates_category ON modification_templates(category);
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = 'idx_modification_templates_framework') THEN
        CREATE INDEX idx_modification_templates_framework ON modification_templates(framework);
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = 'idx_validation_rules_framework') THEN
        CREATE INDEX idx_validation_rules_framework ON validation_rules(framework);
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = 'idx_integrations_service') THEN
        CREATE INDEX idx_integrations_service ON integrations(service_name);
    END IF;
END $$;