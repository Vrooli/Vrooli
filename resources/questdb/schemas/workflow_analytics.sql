-- Workflow execution analytics
-- Tracks performance of n8n, Node-RED, and other automation workflows

CREATE TABLE IF NOT EXISTS workflow_metrics (
    timestamp TIMESTAMP,                    -- Workflow start time
    workflow_id SYMBOL,                    -- Unique workflow identifier
    workflow_name SYMBOL,                  -- Human-readable workflow name
    workflow_version STRING,               -- Workflow version
    platform SYMBOL,                       -- Platform (n8n, node-red, huginn)
    trigger_type SYMBOL,                   -- How workflow was triggered
    execution_id SYMBOL,                   -- Unique execution identifier
    execution_time_ms DOUBLE,              -- Total execution time
    steps_total INT,                       -- Total number of steps
    steps_completed INT,                   -- Successfully completed steps
    steps_failed INT,                      -- Failed steps
    status SYMBOL,                         -- Final status (success, failed, timeout, cancelled)
    error_node SYMBOL,                     -- Node where error occurred
    error_message STRING,                  -- Error details
    input_size_bytes LONG,                 -- Size of input data
    output_size_bytes LONG,                -- Size of output data
    resources_used STRING,                 -- JSON array of resources used
    cost DOUBLE,                          -- Estimated cost if applicable
    user_id SYMBOL,                        -- User who triggered workflow
    metadata STRING                        -- Additional metadata as JSON
) timestamp(timestamp) PARTITION BY DAY;
