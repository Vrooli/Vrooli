-- Seed data for Prompt Manager
-- Default user, campaigns, tags, and sample prompts

-- Insert default user
INSERT INTO users (username, name, email) VALUES 
    ('default', 'Personal User', 'user@localhost')
ON CONFLICT (username) DO NOTHING;

-- Insert default campaign tags
INSERT INTO tags (name, color, description) VALUES
    ('debugging', '#EF4444', 'Code debugging and troubleshooting'),
    ('ux-design', '#8B5CF6', 'User experience and interface design'),
    ('coding', '#10B981', 'Software development and programming'),
    ('writing', '#F59E0B', 'Content writing and documentation'),
    ('analysis', '#3B82F6', 'Data analysis and research'),
    ('automation', '#6B7280', 'Process automation and workflows'),
    ('creative', '#EC4899', 'Creative tasks and ideation'),
    ('learning', '#14B8A6', 'Learning and skill development')
ON CONFLICT (name) DO NOTHING;

-- Insert default campaigns with proper ordering
INSERT INTO campaigns (name, description, color, icon, sort_order, is_favorite) VALUES
    ('Debugging', 'Prompts for troubleshooting code issues and finding bugs', '#EF4444', 'bug', 1, true),
    ('UX Design', 'User experience research, design thinking, and interface improvements', '#8B5CF6', 'palette', 2, true),
    ('Coding', 'Software development, architecture, and programming best practices', '#10B981', 'code', 3, true),
    ('Writing', 'Documentation, content creation, and communication', '#F59E0B', 'pen-tool', 4, false),
    ('Analysis', 'Data analysis, research, and problem-solving approaches', '#3B82F6', 'chart-bar', 5, false),
    ('Automation', 'Workflow automation and process optimization', '#6B7280', 'cog', 6, false),
    ('Learning', 'Skill development and educational content', '#14B8A6', 'book-open', 7, false)
ON CONFLICT (name) DO NOTHING;

-- Get campaign and tag IDs for relationships
DO $$ 
DECLARE
    debugging_campaign_id UUID;
    ux_campaign_id UUID;
    coding_campaign_id UUID;
    debugging_tag_id UUID;
    ux_tag_id UUID;
    coding_tag_id UUID;
BEGIN
    -- Get campaign IDs
    SELECT id INTO debugging_campaign_id FROM campaigns WHERE name = 'Debugging';
    SELECT id INTO ux_campaign_id FROM campaigns WHERE name = 'UX Design';
    SELECT id INTO coding_campaign_id FROM campaigns WHERE name = 'Coding';
    
    -- Get tag IDs
    SELECT id INTO debugging_tag_id FROM tags WHERE name = 'debugging';
    SELECT id INTO ux_tag_id FROM tags WHERE name = 'ux-design';
    SELECT id INTO coding_tag_id FROM tags WHERE name = 'coding';

    -- Insert sample prompts for debugging campaign
    INSERT INTO prompts (campaign_id, title, content, description, usage_count, is_favorite, effectiveness_rating) VALUES
        (debugging_campaign_id, 
         'Debug SQL Performance', 
         'I have a slow SQL query that needs optimization. Here''s the query:

[QUERY]

The current execution time is [TIME] and it''s running against a table with [ROWS] rows. Please analyze this query and suggest optimizations, including:

1. Index recommendations
2. Query restructuring possibilities
3. Potential bottlenecks
4. Alternative approaches

Database: [DATABASE_TYPE]
Table schema: [SCHEMA]', 
         'Template for debugging slow SQL queries with systematic analysis',
         3, true, 5),
         
        (debugging_campaign_id,
         'JavaScript Error Analysis',
         'I''m getting this JavaScript error and need help debugging:

Error: [ERROR_MESSAGE]
Stack trace: [STACK_TRACE]
Context: [CONTEXT_DESCRIPTION]
Browser/Environment: [ENVIRONMENT]

Please help me:
1. Understand what this error means
2. Identify the root cause
3. Provide a fix
4. Suggest how to prevent similar issues

Code snippet where error occurs:
```javascript
[CODE_SNIPPET]
```',
         'Systematic approach to debugging JavaScript errors',
         5, true, 4),

        (debugging_campaign_id,
         'API Integration Debug',
         'I''m having issues with an API integration. Here are the details:

API Endpoint: [ENDPOINT]
HTTP Method: [METHOD]
Request payload: [PAYLOAD]
Response received: [RESPONSE]
Expected response: [EXPECTED]

Error details:
- Status code: [STATUS_CODE]
- Error message: [ERROR_MESSAGE]
- Headers: [HEADERS]

Please help me debug this API integration and suggest fixes.',
         'Template for troubleshooting API integration problems',
         2, false, 4);

    -- Insert sample prompts for UX Design campaign
    INSERT INTO prompts (campaign_id, title, content, description, usage_count, effectiveness_rating) VALUES
        (ux_campaign_id,
         'User Journey Analysis',
         'I need to analyze the user journey for [FEATURE/PRODUCT]. Here''s the current flow:

Current user steps:
1. [STEP_1]
2. [STEP_2]
3. [STEP_3]
[etc.]

Pain points identified:
- [PAIN_POINT_1]
- [PAIN_POINT_2]

Goals:
- [GOAL_1]
- [GOAL_2]

Please help me:
1. Identify friction points in this journey
2. Suggest improvements to reduce cognitive load
3. Recommend better user flow alternatives
4. Consider accessibility and mobile experience',
         'Comprehensive user journey analysis template',
         4, true, 5),

        (ux_campaign_id,
         'Design System Component',
         'I need to design a [COMPONENT_TYPE] component for our design system.

Requirements:
- Use case: [USE_CASE]
- Target users: [USER_TYPES]
- Platform: [PLATFORM]
- Brand guidelines: [BRAND_INFO]

Current design challenges:
- [CHALLENGE_1]
- [CHALLENGE_2]

Please help me design this component considering:
1. Usability best practices
2. Accessibility (WCAG compliance)
3. Consistency with existing design system
4. Responsive behavior
5. Interaction patterns',
         'Template for creating design system components',
         2, false, 4);

    -- Insert sample prompts for coding campaign  
    INSERT INTO prompts (campaign_id, title, content, description, usage_count, effectiveness_rating) VALUES
        (coding_campaign_id,
         'Code Review Request',
         'Please review this [LANGUAGE] code for:

```[language]
[CODE_BLOCK]
```

Context: [CONTEXT_DESCRIPTION]
Purpose: [PURPOSE]

Please focus on:
1. Code quality and best practices
2. Performance considerations  
3. Security vulnerabilities
4. Maintainability and readability
5. Test coverage recommendations

Specific concerns: [SPECIFIC_CONCERNS]',
         'Comprehensive code review template',
         6, true, 5),

        (coding_campaign_id,
         'Architecture Decision',
         'I need to make an architecture decision for [PROJECT/FEATURE].

Current situation:
- [CURRENT_STATE]
- [CONSTRAINTS]
- [REQUIREMENTS]

Options being considered:
1. [OPTION_1]: [DESCRIPTION]
   Pros: [PROS]
   Cons: [CONS]

2. [OPTION_2]: [DESCRIPTION]
   Pros: [PROS] 
   Cons: [CONS]

Please help me evaluate these options considering:
- Scalability
- Maintainability  
- Performance
- Development velocity
- Technical debt implications',
         'Template for evaluating architecture decisions',
         3, true, 4);

    -- Create prompt-tag relationships
    INSERT INTO prompt_tags (prompt_id, tag_id) 
    SELECT p.id, debugging_tag_id 
    FROM prompts p 
    JOIN campaigns c ON p.campaign_id = c.id 
    WHERE c.name = 'Debugging';

    INSERT INTO prompt_tags (prompt_id, tag_id)
    SELECT p.id, ux_tag_id
    FROM prompts p
    JOIN campaigns c ON p.campaign_id = c.id
    WHERE c.name = 'UX Design';

    INSERT INTO prompt_tags (prompt_id, tag_id)
    SELECT p.id, coding_tag_id
    FROM prompts p
    JOIN campaigns c ON p.campaign_id = c.id
    WHERE c.name = 'Coding';

END $$;

-- Insert useful prompt templates
INSERT INTO templates (name, description, content, category, variables) VALUES
    ('Code Analysis',
     'Template for analyzing code quality and suggesting improvements',
     'Please analyze this [LANGUAGE] code:

```[language]
[CODE]
```

Focus on:
1. [FOCUS_AREA_1]
2. [FOCUS_AREA_2] 
3. [FOCUS_AREA_3]

Provide specific recommendations for improvement.',
     'development',
     '["LANGUAGE", "CODE", "FOCUS_AREA_1", "FOCUS_AREA_2", "FOCUS_AREA_3"]'::jsonb),

    ('Feature Specification',
     'Template for creating detailed feature specifications',
     'Feature: [FEATURE_NAME]

## Overview
[BRIEF_DESCRIPTION]

## User Stories
- As a [USER_TYPE], I want [GOAL] so that [BENEFIT]
- [ADDITIONAL_STORIES]

## Acceptance Criteria
1. [CRITERIA_1]
2. [CRITERIA_2]
3. [CRITERIA_3]

## Technical Considerations
- [TECH_REQUIREMENT_1]
- [TECH_REQUIREMENT_2]

## Dependencies
- [DEPENDENCY_1]
- [DEPENDENCY_2]',
     'planning',
     '["FEATURE_NAME", "BRIEF_DESCRIPTION", "USER_TYPE", "GOAL", "BENEFIT"]'::jsonb),

    ('Bug Report Analysis',
     'Template for systematic bug analysis and resolution',
     'Bug Report: [BUG_TITLE]

## Current Behavior
[ACTUAL_BEHAVIOR]

## Expected Behavior  
[EXPECTED_BEHAVIOR]

## Steps to Reproduce
1. [STEP_1]
2. [STEP_2]
3. [STEP_3]

## Environment
- Browser/Platform: [ENVIRONMENT]
- Version: [VERSION]
- Additional context: [CONTEXT]

Please help me:
1. Identify the root cause
2. Suggest a fix
3. Recommend prevention strategies',
     'debugging',
     '["BUG_TITLE", "ACTUAL_BEHAVIOR", "EXPECTED_BEHAVIOR", "ENVIRONMENT", "VERSION"]'::jsonb);

-- Update campaign last_used timestamps and prompt counts
UPDATE campaigns SET 
    last_used = CURRENT_TIMESTAMP,
    prompt_count = (SELECT COUNT(*) FROM prompts WHERE campaign_id = campaigns.id)
WHERE name IN ('Debugging', 'UX Design', 'Coding');