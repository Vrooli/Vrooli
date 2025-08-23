-- AI model inference and performance metrics
-- Tracks all AI/LLM operations across the platform

CREATE TABLE IF NOT EXISTS ai_inference (
    timestamp TIMESTAMP,                    -- Time of inference
    model SYMBOL,                          -- Model name (llama3.2, gpt-4, etc.)
    task_type SYMBOL,                      -- Task category (chat, completion, embedding, etc.)
    user_id SYMBOL,                        -- User or session identifier
    request_id SYMBOL,                     -- Unique request identifier
    response_time_ms DOUBLE,               -- Total response time in milliseconds
    time_to_first_token_ms DOUBLE,        -- Time to first token (streaming)
    tokens_input INT,                      -- Number of input tokens
    tokens_output INT,                     -- Number of output tokens
    tokens_per_second DOUBLE,              -- Generation speed
    cost DOUBLE,                          -- Estimated cost in USD
    temperature DOUBLE,                    -- Model temperature parameter
    success BOOLEAN,                       -- Whether inference succeeded
    error_type SYMBOL,                     -- Error category if failed
    error_message STRING,                  -- Detailed error message
    metadata STRING                        -- Additional metadata as JSON
) timestamp(timestamp) PARTITION BY DAY;