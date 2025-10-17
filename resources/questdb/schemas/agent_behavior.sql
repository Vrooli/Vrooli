-- Agent behavior and interaction tracking
-- Monitors Agent-S2, Browserless, and other agent activities

CREATE TABLE IF NOT EXISTS agent_behavior (
    timestamp TIMESTAMP,                    -- Time of action
    agent_id SYMBOL,                       -- Agent identifier (agent-s2, browserless, etc.)
    session_id SYMBOL,                     -- Session identifier
    action_type SYMBOL,                    -- Action category (click, type, navigate, screenshot, etc.)
    target_element STRING,                 -- Element interacted with (selector, coordinates)
    target_url STRING,                     -- URL if web interaction
    duration_ms DOUBLE,                    -- Time taken for action
    success BOOLEAN,                       -- Whether action succeeded
    retry_count INT,                       -- Number of retries needed
    screenshot_taken BOOLEAN,              -- Whether screenshot was captured
    ai_confidence DOUBLE,                  -- AI confidence score (0-1)
    ai_reasoning STRING,                   -- AI's reasoning for action
    error_type SYMBOL,                     -- Error category if failed
    error_message STRING,                  -- Detailed error message
    context STRING,                        -- Additional context as JSON
    parent_workflow_id SYMBOL,             -- Associated workflow if any
    resource_usage STRING                  -- CPU/memory usage during action
) timestamp(timestamp) PARTITION BY DAY;