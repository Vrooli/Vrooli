-- Vrooli Orchestrator Default Profiles
-- Insert default startup profiles for common use cases

-- Developer Profile - Full development environment
INSERT INTO profiles (
    name, 
    display_name, 
    description,
    metadata,
    resources,
    scenarios,
    auto_browser,
    idle_shutdown_minutes
) VALUES (
    'developer-full',
    'Developer (Full Stack)',
    'Complete development environment with all tools, monitoring dashboards, and debugging capabilities',
    '{
        "target_audience": "developers",
        "resource_footprint": "high",
        "use_case": "development_and_testing",
        "category": "development"
    }',
    ARRAY['postgres', 'n8n', 'ollama', 'browserless', 'qdrant', 'redis'],
    ARRAY['system-monitor', 'app-debugger', 'app-monitor', 'core-debugger'],
    ARRAY['http://localhost:3000/system-monitor', 'http://localhost:3001/app-debugger'],
    60
);

-- Lightweight Developer Profile 
INSERT INTO profiles (
    name,
    display_name, 
    description,
    metadata,
    resources,
    scenarios,
    auto_browser,
    idle_shutdown_minutes
) VALUES (
    'developer-light',
    'Developer (Lightweight)',
    'Essential development tools without heavy monitoring - good for laptop/limited resources',
    '{
        "target_audience": "developers",
        "resource_footprint": "medium", 
        "use_case": "mobile_development",
        "category": "development"
    }',
    ARRAY['postgres', 'n8n', 'ollama'],
    ARRAY['app-debugger'],
    ARRAY[],
    45
);

-- Business User Profile
INSERT INTO profiles (
    name,
    display_name,
    description, 
    metadata,
    resources,
    scenarios,
    auto_browser,
    idle_shutdown_minutes
) VALUES (
    'business-productivity',
    'Business & Productivity',
    'Business-focused scenarios for productivity, communication, and data analysis',
    '{
        "target_audience": "business_users",
        "resource_footprint": "medium",
        "use_case": "business_productivity", 
        "category": "business"
    }',
    ARRAY['postgres', 'n8n', 'ollama'],
    ARRAY['document-manager', 'contact-book', 'calendar', 'task-planner', 'email-triage'],
    ARRAY['http://localhost:3000/document-manager', 'http://localhost:3001/calendar'],
    90
);

-- Creative Profile  
INSERT INTO profiles (
    name,
    display_name,
    description,
    metadata,
    resources, 
    scenarios,
    auto_browser,
    idle_shutdown_minutes
) VALUES (
    'creative-suite',
    'Creative & Content',
    'Creative tools for content creation, image generation, and storytelling',
    '{
        "target_audience": "content_creators",
        "resource_footprint": "high",
        "use_case": "content_creation",
        "category": "creative"
    }',
    ARRAY['postgres', 'n8n', 'ollama', 'minio'],
    ARRAY['image-generation-pipeline', 'bedtime-story-generator', 'brand-manager', 'campaign-content-studio'],
    ARRAY['http://localhost:3000/image-generation-pipeline'],
    120
);

-- Gaming Profile
INSERT INTO profiles (
    name,
    display_name,
    description,
    metadata,
    resources,
    scenarios,
    auto_browser,
    idle_shutdown_minutes
) VALUES (
    'gaming-entertainment',
    'Gaming & Entertainment', 
    'Fun and entertaining scenarios - games, random selectors, and interactive experiences',
    '{
        "target_audience": "casual_users",
        "resource_footprint": "low",
        "use_case": "entertainment",
        "category": "entertainment"
    }',
    ARRAY['postgres', 'n8n'],
    ARRAY['retro-game-launcher', 'picker-wheel', 'quiz-generator', 'typing-test'],
    ARRAY['http://localhost:3000/retro-game-launcher', 'http://localhost:3001/picker-wheel'],
    180
);

-- Household Management Profile
INSERT INTO profiles (
    name,
    display_name,
    description,
    metadata,
    resources,
    scenarios,
    auto_browser,
    idle_shutdown_minutes
) VALUES (
    'household-management',
    'Household Management',
    'Family and home management tools - chores, schedules, shopping, and planning',
    '{
        "target_audience": "families",
        "resource_footprint": "medium",
        "use_case": "household_management",
        "category": "lifestyle"
    }',
    ARRAY['postgres', 'n8n', 'ollama'],
    ARRAY['chore-tracking', 'calendar', 'smart-shopping-assistant', 'home-automation', 'contact-book'],
    ARRAY['http://localhost:3000/chore-tracking', 'http://localhost:3001/calendar'],
    240
);

-- Demo/Showcase Profile
INSERT INTO profiles (
    name,
    display_name,
    description,
    metadata,
    resources,
    scenarios,
    auto_browser,
    idle_shutdown_minutes
) VALUES (
    'demo-showcase',
    'Demo & Showcase',
    'Curated selection of impressive scenarios for demonstrations and showcasing Vrooli capabilities',
    '{
        "target_audience": "prospects",
        "resource_footprint": "medium",
        "use_case": "demonstration",
        "category": "showcase"
    }',
    ARRAY['postgres', 'n8n', 'ollama', 'browserless'],
    ARRAY['system-monitor', 'image-generation-pipeline', 'fall-foliage-explorer', 'chart-generator', 'scenario-surfer'],
    ARRAY['http://localhost:3000/scenario-surfer'],
    30
);

-- Research & Analysis Profile  
INSERT INTO profiles (
    name,
    display_name,
    description,
    metadata,
    resources,
    scenarios,
    auto_browser,
    idle_shutdown_minutes
) VALUES (
    'research-analysis',
    'Research & Analysis',
    'Research tools, data analysis, and intelligence gathering scenarios',
    '{
        "target_audience": "researchers",
        "resource_footprint": "high", 
        "use_case": "research_and_analysis",
        "category": "research"
    }',
    ARRAY['postgres', 'n8n', 'ollama', 'qdrant', 'browserless'],
    ARRAY['research-assistant', 'data-structurer', 'bookmark-intelligence-hub', 'competitor-change-monitor', 'roi-fit-analysis'],
    ARRAY['http://localhost:3000/research-assistant'],
    90
);

-- Minimal Profile - Bare minimum for testing
INSERT INTO profiles (
    name,
    display_name,
    description,
    metadata,
    resources,
    scenarios,
    auto_browser,
    idle_shutdown_minutes
) VALUES (
    'minimal',
    'Minimal Test',
    'Bare minimum resources for testing and troubleshooting - just postgres and one scenario',
    '{
        "target_audience": "system_administrators",
        "resource_footprint": "low",
        "use_case": "testing_troubleshooting",
        "category": "testing"
    }',
    ARRAY['postgres'],
    ARRAY['system-monitor'],
    ARRAY[],
    15
);

-- Customer Deployment Profile Template
INSERT INTO profiles (
    name,
    display_name,
    description,
    metadata,
    resources,
    scenarios,
    auto_browser,
    idle_shutdown_minutes
) VALUES (
    'customer-template',
    'Customer Deployment Template',
    'Template profile for customer-specific deployments - customize resources and scenarios per customer needs',
    '{
        "target_audience": "customers",
        "resource_footprint": "customizable",
        "use_case": "customer_deployment",
        "category": "template",
        "is_template": true
    }',
    ARRAY['postgres', 'n8n'],
    ARRAY['scenario-surfer'],
    ARRAY['http://localhost:3000/scenario-surfer'],
    NULL
);

-- Set default profile to developer-light (good balance for most users)
SELECT set_active_profile(
    (SELECT id FROM profiles WHERE name = 'developer-light' LIMIT 1),
    NULL
);