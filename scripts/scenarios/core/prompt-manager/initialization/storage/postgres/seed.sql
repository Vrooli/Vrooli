-- Seed data for Prompt Manager with AI Maintenance Tasks
-- Default user, campaigns, tags, and maintenance prompts

-- Insert default user
INSERT INTO users (username, name, email) VALUES 
    ('default', 'AI Maintenance User', 'maintenance@vrooli.com')
ON CONFLICT (username) DO NOTHING;

-- Insert maintenance-focused tags
INSERT INTO tags (name, color, description) VALUES
    ('maintenance', '#6366f1', 'AI maintenance and code quality tasks'),
    ('testing', '#10B981', 'Test quality and coverage improvements'),
    ('performance', '#F59E0B', 'Performance optimization tasks'),
    ('security', '#EF4444', 'Security review and vulnerability fixes'),
    ('accessibility', '#8B5CF6', 'Accessibility and WCAG compliance'),
    ('code-quality', '#3B82F6', 'Code quality and refactoring'),
    ('documentation', '#6B7280', 'Documentation and comment improvements'),
    ('cleanup', '#EC4899', 'Code cleanup and dead code removal'),
    ('error-handling', '#14B8A6', 'Error handling and logging improvements'),
    ('startup', '#F97316', 'Development environment and startup issues'),
    ('feature-development', '#10B981', 'Feature development and enhancement'),
    ('infrastructure', '#F97316', 'Infrastructure and resource development'),
    ('automation', '#8B5CF6', 'Workflow and automation development'),
    ('application', '#EC4899', 'Application and scenario development'),
    ('platform', '#6B7280', 'Platform and core feature development'),
    ('guide', '#6366f1', 'Guide and reference prompts')
ON CONFLICT (name) DO NOTHING;

-- Insert campaigns for organizing maintenance tasks
INSERT INTO campaigns (name, description, color, icon, sort_order, is_favorite) VALUES
    ('AI Maintenance', 'Automated maintenance tasks for code quality and health', '#6366f1', 'wrench', 1, true),
    ('Test Improvements', 'Test quality, coverage, and reliability tasks', '#10B981', 'check-circle', 2, true),
    ('Performance', 'Performance optimization and monitoring tasks', '#F59E0B', 'zap', 3, true),
    ('Security & Safety', 'Security reviews and vulnerability assessments', '#EF4444', 'shield', 4, true),
    ('User Experience', 'UX, accessibility, and loading state improvements', '#8B5CF6', 'users', 5, false),
    ('Code Health', 'Code quality, cleanup, and refactoring tasks', '#3B82F6', 'heart', 6, false),
    ('Documentation', 'Documentation, comments, and knowledge management', '#6B7280', 'book', 7, false),
    ('Feature Development', 'Comprehensive guides for building new Vrooli capabilities', '#10B981', 'plus-circle', 8, true),
    ('Resource Development', 'Building and enhancing local services and infrastructure', '#F97316', 'server', 9, true),
    ('Automation Development', 'Creating workflows and process automation', '#8B5CF6', 'workflow', 10, true),
    ('Application Development', 'Building complete scenarios and business applications', '#EC4899', 'application', 11, true),
    ('Platform Development', 'Core Vrooli platform and infrastructure improvements', '#6B7280', 'cog', 12, false)
ON CONFLICT (name) DO NOTHING;

-- Get campaign and tag IDs for relationships
DO $$ 
DECLARE
    maintenance_campaign_id UUID;
    test_campaign_id UUID;
    perf_campaign_id UUID;
    security_campaign_id UUID;
    ux_campaign_id UUID;
    code_campaign_id UUID;
    docs_campaign_id UUID;
    feature_campaign_id UUID;
    resource_campaign_id UUID;
    automation_campaign_id UUID;
    application_campaign_id UUID;
    platform_campaign_id UUID;
    
    maintenance_tag_id UUID;
    testing_tag_id UUID;
    performance_tag_id UUID;
    security_tag_id UUID;
    accessibility_tag_id UUID;
    code_quality_tag_id UUID;
    documentation_tag_id UUID;
    cleanup_tag_id UUID;
    error_handling_tag_id UUID;
    startup_tag_id UUID;
    feature_development_tag_id UUID;
    infrastructure_tag_id UUID;
    automation_tag_id UUID;
    application_tag_id UUID;
    platform_tag_id UUID;
    guide_tag_id UUID;
BEGIN
    -- Get campaign IDs
    SELECT id INTO maintenance_campaign_id FROM campaigns WHERE name = 'AI Maintenance';
    SELECT id INTO test_campaign_id FROM campaigns WHERE name = 'Test Improvements';
    SELECT id INTO perf_campaign_id FROM campaigns WHERE name = 'Performance';
    SELECT id INTO security_campaign_id FROM campaigns WHERE name = 'Security & Safety';
    SELECT id INTO ux_campaign_id FROM campaigns WHERE name = 'User Experience';
    SELECT id INTO code_campaign_id FROM campaigns WHERE name = 'Code Health';
    SELECT id INTO docs_campaign_id FROM campaigns WHERE name = 'Documentation';
    SELECT id INTO feature_campaign_id FROM campaigns WHERE name = 'Feature Development';
    SELECT id INTO resource_campaign_id FROM campaigns WHERE name = 'Resource Development';
    SELECT id INTO automation_campaign_id FROM campaigns WHERE name = 'Automation Development';
    SELECT id INTO application_campaign_id FROM campaigns WHERE name = 'Application Development';
    SELECT id INTO platform_campaign_id FROM campaigns WHERE name = 'Platform Development';
    
    -- Get tag IDs
    SELECT id INTO maintenance_tag_id FROM tags WHERE name = 'maintenance';
    SELECT id INTO testing_tag_id FROM tags WHERE name = 'testing';
    SELECT id INTO performance_tag_id FROM tags WHERE name = 'performance';
    SELECT id INTO security_tag_id FROM tags WHERE name = 'security';
    SELECT id INTO accessibility_tag_id FROM tags WHERE name = 'accessibility';
    SELECT id INTO code_quality_tag_id FROM tags WHERE name = 'code-quality';
    SELECT id INTO documentation_tag_id FROM tags WHERE name = 'documentation';
    SELECT id INTO cleanup_tag_id FROM tags WHERE name = 'cleanup';
    SELECT id INTO error_handling_tag_id FROM tags WHERE name = 'error-handling';
    SELECT id INTO startup_tag_id FROM tags WHERE name = 'startup';
    SELECT id INTO feature_development_tag_id FROM tags WHERE name = 'feature-development';
    SELECT id INTO infrastructure_tag_id FROM tags WHERE name = 'infrastructure';
    SELECT id INTO automation_tag_id FROM tags WHERE name = 'automation';
    SELECT id INTO application_tag_id FROM tags WHERE name = 'application';
    SELECT id INTO platform_tag_id FROM tags WHERE name = 'platform';
    SELECT id INTO guide_tag_id FROM tags WHERE name = 'guide';

    -- Insert the main guide
    INSERT INTO prompts (campaign_id, title, file_path, description, variables, usage_count, is_favorite, effectiveness_rating, quick_access_key) VALUES
        (maintenance_campaign_id, 
         'AI Maintenance Task Guide', 
         'maintenance/_TASK_GUIDE.md',
         'Complete guide for AI maintenance tracking system with all task IDs and instructions',
         '[]'::jsonb,
         0, true, 5, 'guide');

    -- Insert test-related maintenance tasks
    INSERT INTO prompts (campaign_id, title, file_path, description, variables, usage_count, is_favorite, effectiveness_rating, quick_access_key) VALUES
        (test_campaign_id, 
         'Test Quality Review', 
         'maintenance/TEST_QUALITY.md',
         'Find and fix tests written to pass rather than test actual behavior',
         '[]'::jsonb,
         0, true, 5, 'test-quality'),
         
        (test_campaign_id,
         'Test Coverage Improvement',
         'maintenance/TEST_COVERAGE.md',
         'Improve test coverage for critical paths and edge cases',
         '[]'::jsonb,
         0, false, 4, 'test-coverage');

    -- Insert performance-related tasks
    INSERT INTO prompts (campaign_id, title, file_path, description, variables, usage_count, is_favorite, effectiveness_rating, quick_access_key) VALUES
        (perf_campaign_id,
         'React Performance Optimization',
         'maintenance/REACT_PERF.md',
         'Optimize React component rendering and performance issues',
         '[]'::jsonb,
         0, true, 5, 'react-perf'),
         
        (perf_campaign_id,
         'General Performance Optimization',
         'maintenance/PERF_GENERAL.md',
         'Improve overall application performance',
         '[]'::jsonb,
         0, false, 4, 'perf');

    -- Insert security and safety tasks
    INSERT INTO prompts (campaign_id, title, file_path, description, variables, usage_count, is_favorite, effectiveness_rating, quick_access_key) VALUES
        (security_campaign_id,
         'Security Review',
         'maintenance/SECURITY.md',
         'Identify and fix security vulnerabilities',
         '[]'::jsonb,
         0, true, 5, 'security'),
         
        (security_campaign_id,
         'Type Safety Improvements',
         'maintenance/TYPE_SAFETY.md',
         'Improve TypeScript type safety and eliminate any types',
         '[]'::jsonb,
         0, false, 4, 'type-safety'),
         
        (security_campaign_id,
         'Error Handling Enhancement',
         'maintenance/ERROR_HANDLING.md',
         'Improve error handling, logging, and user feedback',
         '[]'::jsonb,
         0, true, 5, 'errors');

    -- Insert UX and accessibility tasks
    INSERT INTO prompts (campaign_id, title, file_path, description, variables, usage_count, is_favorite, effectiveness_rating, quick_access_key) VALUES
        (ux_campaign_id,
         'Accessibility Improvements',
         'maintenance/ACCESSIBILITY.md',
         'Enhance WCAG compliance and accessibility features',
         '[]'::jsonb,
         0, false, 5, 'a11y'),
         
        (ux_campaign_id,
         'Loading States UX',
         'maintenance/LOADING_STATES.md',
         'Improve loading states and user feedback',
         '[]'::jsonb,
         0, false, 4, 'loading');

    -- Insert code health tasks
    INSERT INTO prompts (campaign_id, title, file_path, description, variables, usage_count, is_favorite, effectiveness_rating, quick_access_key) VALUES
        (code_campaign_id,
         'Code Quality Review',
         'maintenance/CODE_QUALITY.md',
         'General code quality improvements and refactoring',
         '[]'::jsonb,
         0, false, 4, 'quality'),
         
        (code_campaign_id,
         'Dead Code Elimination',
         'maintenance/DEAD_CODE.md',
         'Identify and remove unused code',
         '[]'::jsonb,
         0, false, 4, 'dead-code'),
         
        (code_campaign_id,
         'Import Cleanup',
         'maintenance/IMPORTS.md',
         'Clean up and optimize import statements',
         '[]'::jsonb,
         0, false, 3, 'imports'),
         
        (code_campaign_id,
         'TODO/FIXME Cleanup',
         'maintenance/TODO_CLEANUP.md',
         'Address TODO and FIXME comments in codebase',
         '[]'::jsonb,
         0, false, 3, 'todos');

    -- Insert documentation tasks
    INSERT INTO prompts (campaign_id, title, file_path, description, variables, usage_count, is_favorite, effectiveness_rating, quick_access_key) VALUES
        (docs_campaign_id,
         'Comment Quality',
         'maintenance/COMMENTS.md',
         'Improve code comments and documentation',
         '[]'::jsonb,
         0, false, 3, 'comments');

    -- Insert special/priority tasks
    INSERT INTO prompts (campaign_id, title, file_path, description, variables, usage_count, is_favorite, effectiveness_rating, quick_access_key) VALUES
        (maintenance_campaign_id,
         'Easy Wins Identification',
         'maintenance/EASY_WINS.md',
         'Find quick, high-impact improvements',
         '[]'::jsonb,
         0, true, 4, 'easy'),
         
        (maintenance_campaign_id,
         'Startup Error Resolution',
         'maintenance/STARTUP_ERRORS.md',
         'Identify and fix development environment startup issues',
         '[]'::jsonb,
         0, true, 5, 'startup');

    -- Create prompt-tag relationships for test tasks
    INSERT INTO prompt_tags (prompt_id, tag_id) 
    SELECT p.id, testing_tag_id 
    FROM prompts p 
    WHERE p.file_path IN ('maintenance/TEST_QUALITY.md', 'maintenance/TEST_COVERAGE.md');

    -- Create prompt-tag relationships for performance tasks
    INSERT INTO prompt_tags (prompt_id, tag_id)
    SELECT p.id, performance_tag_id
    FROM prompts p
    WHERE p.file_path IN ('maintenance/REACT_PERF.md', 'maintenance/PERF_GENERAL.md');

    -- Create prompt-tag relationships for security tasks
    INSERT INTO prompt_tags (prompt_id, tag_id)
    SELECT p.id, security_tag_id
    FROM prompts p
    WHERE p.file_path IN ('maintenance/SECURITY.md', 'maintenance/TYPE_SAFETY.md');

    -- Create prompt-tag relationships for error handling
    INSERT INTO prompt_tags (prompt_id, tag_id)
    SELECT p.id, error_handling_tag_id
    FROM prompts p
    WHERE p.file_path = 'maintenance/ERROR_HANDLING.md';

    -- Create prompt-tag relationships for accessibility
    INSERT INTO prompt_tags (prompt_id, tag_id)
    SELECT p.id, accessibility_tag_id
    FROM prompts p
    WHERE p.file_path = 'maintenance/ACCESSIBILITY.md';

    -- Create prompt-tag relationships for code quality
    INSERT INTO prompt_tags (prompt_id, tag_id)
    SELECT p.id, code_quality_tag_id
    FROM prompts p
    WHERE p.file_path IN ('maintenance/CODE_QUALITY.md', 'maintenance/IMPORTS.md');

    -- Create prompt-tag relationships for cleanup
    INSERT INTO prompt_tags (prompt_id, tag_id)
    SELECT p.id, cleanup_tag_id
    FROM prompts p
    WHERE p.file_path IN ('maintenance/DEAD_CODE.md', 'maintenance/TODO_CLEANUP.md');

    -- Create prompt-tag relationships for documentation
    INSERT INTO prompt_tags (prompt_id, tag_id)
    SELECT p.id, documentation_tag_id
    FROM prompts p
    WHERE p.file_path = 'maintenance/COMMENTS.md';

    -- Create prompt-tag relationships for startup
    INSERT INTO prompt_tags (prompt_id, tag_id)
    SELECT p.id, startup_tag_id
    FROM prompts p
    WHERE p.file_path = 'maintenance/STARTUP_ERRORS.md';

    -- Tag all maintenance tasks
    INSERT INTO prompt_tags (prompt_id, tag_id)
    SELECT p.id, maintenance_tag_id
    FROM prompts p
    WHERE p.file_path LIKE 'maintenance/%';

    -- Insert feature development prompts
    INSERT INTO prompts (campaign_id, title, file_path, description, variables, usage_count, is_favorite, effectiveness_rating, quick_access_key) VALUES
        (feature_campaign_id,
         'Feature Development Guide',
         'features/feature-development-guide.md',
         'Meta-prompt to help choose which development prompt to use',
         '[]'::jsonb,
         0, true, 5, 'dev-guide'),
         
        (resource_campaign_id,
         'Resource Development and Enhancement',
         'features/add-fix-resource.md',
         'Comprehensive guide for adding or fixing Vrooli resources (local services)',
         '[]'::jsonb,
         0, true, 5, 'resource'),
         
        (automation_campaign_id,
         'N8n Workflow Development and Enhancement',
         'features/add-fix-n8n-workflow.md',
         'Complete guide for creating and fixing n8n automation workflows',
         '[]'::jsonb,
         0, true, 5, 'workflow'),
         
        (application_campaign_id,
         'Scenario Development and Enhancement',
         'features/add-fix-scenario.md',
         'Comprehensive guide for building complete Vrooli scenarios (applications)',
         '[]'::jsonb,
         0, true, 5, 'scenario'),
         
        (platform_campaign_id,
         'Core Vrooli Feature Development',
         'features/add-fix-core-features.md',
         'Guide for developing foundational Vrooli platform features',
         '[]'::jsonb,
         0, true, 5, 'core');

    -- Create prompt-tag relationships for feature development prompts
    INSERT INTO prompt_tags (prompt_id, tag_id)
    SELECT p.id, guide_tag_id
    FROM prompts p
    WHERE p.file_path = 'features/feature-development-guide.md';

    INSERT INTO prompt_tags (prompt_id, tag_id)
    SELECT p.id, feature_development_tag_id
    FROM prompts p
    WHERE p.file_path LIKE 'features/%';

    INSERT INTO prompt_tags (prompt_id, tag_id)
    SELECT p.id, infrastructure_tag_id
    FROM prompts p
    WHERE p.file_path = 'features/add-fix-resource.md';

    INSERT INTO prompt_tags (prompt_id, tag_id)
    SELECT p.id, automation_tag_id
    FROM prompts p
    WHERE p.file_path = 'features/add-fix-n8n-workflow.md';

    INSERT INTO prompt_tags (prompt_id, tag_id)
    SELECT p.id, application_tag_id
    FROM prompts p
    WHERE p.file_path = 'features/add-fix-scenario.md';

    INSERT INTO prompt_tags (prompt_id, tag_id)
    SELECT p.id, platform_tag_id
    FROM prompts p
    WHERE p.file_path = 'features/add-fix-core-features.md';

END $$;

-- Update campaign last_used timestamps and prompt counts
UPDATE campaigns SET 
    last_used = CURRENT_TIMESTAMP,
    prompt_count = (SELECT COUNT(*) FROM prompts WHERE campaign_id = campaigns.id)
WHERE name IN ('AI Maintenance', 'Test Improvements', 'Performance', 'Security & Safety', 'User Experience', 'Code Health', 'Documentation', 'Feature Development', 'Resource Development', 'Automation Development', 'Application Development', 'Platform Development');

-- Function to load prompt content from files (placeholder for production)
CREATE OR REPLACE FUNCTION load_prompt_content(path VARCHAR)
RETURNS TEXT AS $$
DECLARE
    base_path TEXT := '/app/initialization/prompts/';
    full_path TEXT;
BEGIN
    full_path := base_path || path;
    -- In production, this would read from the file system
    -- For now, return the path as a placeholder
    RETURN 'Content from file: ' || full_path;
END;
$$ LANGUAGE plpgsql;

-- Update content_cache for all prompts (in production, this would read actual files)
UPDATE prompts 
SET content_cache = load_prompt_content(file_path)
WHERE file_path IS NOT NULL;