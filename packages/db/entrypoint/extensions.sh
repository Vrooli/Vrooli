psql -U ${POSTGRES_USER} -d ${POSTGRES_DB} -c 'CREATE EXTENSION IF NOT EXISTS citext'
psql -U ${POSTGRES_USER} -d ${POSTGRES_DB} -c 'CREATE EXTENSION IF NOT EXISTS "vector"'

# Create the snowflake_id function for distributed ID generation
cat << 'EOF' | psql -U ${POSTGRES_USER} -d ${POSTGRES_DB}
-- Snowflake ID PostgreSQL function
-- Creates a time-ordered, distributed unique ID generation function
-- Similar to Twitter's Snowflake ID system

CREATE OR REPLACE FUNCTION snowflake_id()
RETURNS BIGINT AS $$
DECLARE
    -- Constants
    epoch BIGINT := 1609459200000; -- Custom epoch (Jan 1, 2021 UTC)
    worker_id BIGINT;
    sequence_id BIGINT;
    
    -- Last timestamp used for a Snowflake ID
    last_timestamp BIGINT;
    
    -- Current timestamp
    current_timestamp BIGINT;
    
    -- The Snowflake ID to return
    snowflake_id BIGINT;
BEGIN
    -- Use database PID or connection ID as worker_id
    -- This will be unique per connection but may reuse across connections
    -- In production, you'd want a more stable worker ID strategy
    worker_id := (pg_backend_pid() % 1024);
    
    -- Get current timestamp in milliseconds
    current_timestamp := (EXTRACT(EPOCH FROM NOW()) * 1000)::BIGINT;
    
    -- Get last timestamp from a session variable 
    -- Default to current timestamp if not set
    BEGIN
        last_timestamp := current_setting('snowflake.last_timestamp')::BIGINT;
    EXCEPTION
        WHEN OTHERS THEN
            last_timestamp := 0;
    END;
    
    -- Get sequence from a session variable
    -- Default to 0 if not set
    BEGIN
        sequence_id := current_setting('snowflake.sequence')::BIGINT;
    EXCEPTION
        WHEN OTHERS THEN
            sequence_id := 0;
    END;
    
    -- If timestamp moved backwards, throw an error
    -- In production, you'd want to handle this more gracefully
    IF current_timestamp < last_timestamp THEN
        RAISE EXCEPTION 'Clock moved backwards, refusing to generate ID for % milliseconds', 
            last_timestamp - current_timestamp;
    END IF;
    
    -- If current timestamp is same as last timestamp, increment sequence
    IF current_timestamp = last_timestamp THEN
        sequence_id := (sequence_id + 1) & 4095; -- 4095 is the max sequence (12 bits)
        
        -- If sequence overflows, wait for next millisecond
        IF sequence_id = 0 THEN
            -- In a real implementation, we'd wait until next millisecond
            -- But this simple version just bumps the timestamp
            current_timestamp := current_timestamp + 1;
        END IF;
    ELSE
        -- If timestamp changed, reset sequence
        sequence_id := 0;
    END IF;
    
    -- Save last timestamp and sequence for next call
    PERFORM set_config('snowflake.last_timestamp', current_timestamp::TEXT, FALSE);
    PERFORM set_config('snowflake.sequence', sequence_id::TEXT, FALSE);
    
    -- Calculate Snowflake ID
    -- 41 bits for timestamp, 10 bits for worker ID, 12 bits for sequence
    snowflake_id := ((current_timestamp - epoch) << 22) |
                   (worker_id << 12) |
                   sequence_id;
                  
    RETURN snowflake_id;
END;
$$ LANGUAGE plpgsql;
EOF

# Confirm the function was created
echo "✅ Added snowflake_id() function to PostgreSQL"
