-- === Required extensions ====================================================
CREATE EXTENSION IF NOT EXISTS citext;
CREATE EXTENSION IF NOT EXISTS "vector";

-- === Snowflake-style 64-bit ID generator ====================================
CREATE OR REPLACE FUNCTION snowflake_id()
RETURNS BIGINT
LANGUAGE plpgsql
STABLE
AS $$
DECLARE
    v_epoch          CONSTANT BIGINT := 1609459200000;          -- 2021-01-01
    v_worker_id      BIGINT   := (pg_backend_pid() % 1024);     -- 10 bits
    v_seq            BIGINT;
    v_last_ts_ms     BIGINT;
    v_ts_ms          BIGINT;
    v_id             BIGINT;
BEGIN
    v_ts_ms := (EXTRACT(EPOCH FROM clock_timestamp()) * 1000)::BIGINT;

    BEGIN
        v_last_ts_ms := current_setting('snowflake.last_timestamp')::BIGINT;
    EXCEPTION WHEN others THEN
        v_last_ts_ms := 0;
    END;

    BEGIN
        v_seq := current_setting('snowflake.sequence')::BIGINT;
    EXCEPTION WHEN others THEN
        v_seq := 0;
    END;

    IF v_ts_ms < v_last_ts_ms THEN
        RAISE EXCEPTION
          'Clock moved backwards: refusing to generate ID for % ms',
          v_last_ts_ms - v_ts_ms;
    END IF;

    IF v_ts_ms = v_last_ts_ms THEN
        v_seq := (v_seq + 1) & 4095;
        IF v_seq = 0 THEN
            PERFORM pg_sleep(0.001);
            v_ts_ms := (EXTRACT(EPOCH FROM clock_timestamp()) * 1000)::BIGINT;
        END IF;
    ELSE
        v_seq := 0;
    END IF;

    PERFORM set_config('snowflake.last_timestamp', v_ts_ms::text, false);
    PERFORM set_config('snowflake.sequence',       v_seq::text,    false);

    v_id := ((v_ts_ms - v_epoch) << 22) | (v_worker_id << 12) | v_seq;
    RETURN v_id;
END;
$$;
