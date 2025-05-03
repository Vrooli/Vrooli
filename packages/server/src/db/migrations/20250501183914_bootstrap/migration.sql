-- === Required extensions ====================================================
CREATE EXTENSION IF NOT EXISTS citext;
CREATE EXTENSION IF NOT EXISTS "vector";

-- === Snowflake-style 64-bit ID generator ====================================
CREATE OR REPLACE FUNCTION snowflake_id()
RETURNS BIGINT
LANGUAGE plpgsql
VOLATILE
AS $$
DECLARE
    v_epoch          CONSTANT BIGINT := 1609459200000;          -- 2021-01-01
    v_worker_id      BIGINT   := (pg_backend_pid() % 1024);     -- 10 bits for worker ID
    v_seq_bits       CONSTANT INT    := 12;                     -- 12 bits for sequence
    v_max_seq        CONSTANT BIGINT := (1 << v_seq_bits) - 1;  -- Max sequence value (4095)
    v_seq            BIGINT;
    v_last_ts_ms     BIGINT;
    v_ts_ms          BIGINT;
    v_id             BIGINT;
    v_clock_drift_tolerance_ms CONSTANT BIGINT := 10; -- Allow 10ms backward drift
BEGIN
    -- Use a loop to handle sequence overflow and ensure time moves forward
    LOOP
        v_ts_ms := (EXTRACT(EPOCH FROM clock_timestamp()) * 1000)::BIGINT;

        BEGIN
            v_last_ts_ms := current_setting('snowflake.last_timestamp')::BIGINT;
        EXCEPTION WHEN others THEN
            v_last_ts_ms := 0; -- Initialize if not set in the session
        END;

        BEGIN
            v_seq := current_setting('snowflake.sequence')::BIGINT;
        EXCEPTION WHEN others THEN
            v_seq := 0; -- Initialize if not set
        END;

        -- Handle clock moving backwards
        IF v_ts_ms < v_last_ts_ms THEN
            -- If drift is within tolerance, use the last timestamp
            IF v_last_ts_ms - v_ts_ms <= v_clock_drift_tolerance_ms THEN
                v_ts_ms := v_last_ts_ms; -- Lock timestamp to the previous value
            ELSE
                -- If drift is too large, raise an error
                RAISE EXCEPTION
                    'Clock moved backwards significantly: refusing to generate ID for % ms',
                    v_last_ts_ms - v_ts_ms;
            END IF;
        END IF;

        IF v_ts_ms = v_last_ts_ms THEN
            v_seq := (v_seq + 1) & v_max_seq; -- Increment sequence, wrap around using bitwise AND
            -- If sequence wrapped, wait until the next millisecond
            IF v_seq = 0 THEN
                -- Loop until the clock actually advances
                WHILE v_ts_ms <= v_last_ts_ms LOOP
                   PERFORM pg_sleep(0.0001); -- Sleep very briefly
                   v_ts_ms := (EXTRACT(EPOCH FROM clock_timestamp()) * 1000)::BIGINT;
                END LOOP;
                -- Reset sequence for the new timestamp
                v_seq := 0;
                -- Re-enter the outer loop to process the new timestamp
                CONTINUE;
            END IF;
        ELSE
            -- New millisecond, reset sequence
            v_seq := 0;
        END IF;

        -- If we successfully generated a sequence number for v_ts_ms, exit loop
        EXIT;

    END LOOP;

    -- Update session settings with the timestamp and sequence used
    PERFORM set_config('snowflake.last_timestamp', v_ts_ms::text, false);
    PERFORM set_config('snowflake.sequence',       v_seq::text,    false);

    -- Construct the final ID
    v_id := ((v_ts_ms - v_epoch) << (v_seq_bits + 10)) -- Timestamp component (shifted left by 22 bits)
          | (v_worker_id << v_seq_bits)               -- Worker ID component (shifted left by 12 bits)
          | v_seq;                                    -- Sequence component

    RETURN v_id;
END;
$$;
