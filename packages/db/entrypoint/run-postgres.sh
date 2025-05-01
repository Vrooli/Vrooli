#!/bin/bash
set -e

###############################################################################
# 1. Start Postgres exactly the way the official entry-point would.
###############################################################################
docker-entrypoint.sh postgres &
postgres_pid=$!

###############################################################################
# 2. Poll until Postgres is ready to accept connections.
###############################################################################
echo "Waiting for PostgreSQL to accept connections…"
until pg_isready -U "${POSTGRES_USER:-postgres}" \
                 -d "${POSTGRES_DB:-postgres}"    \
                 -h localhost -p 5432 -q; do
  echo "PostgreSQL is unavailable – sleeping"
  sleep 1
done
echo "PostgreSQL is ready – running extension and function setup…"

###############################################################################
# 3. Create the required extensions and the Snowflake-ID function.
###############################################################################
psql -v ON_ERROR_STOP=1 --username "${POSTGRES_USER:-postgres}" \
                        --dbname   "${POSTGRES_DB:-postgres}"   <<'EOSQL'
CREATE EXTENSION IF NOT EXISTS citext;
CREATE EXTENSION IF NOT EXISTS "vector";

-- ════════════════════════════════════════════════════════════════════════
-- Snowflake-style 64-bit ID generator
--  • 41 bits time (ms since 2021-01-01T00:00:00Z)
--  • 10 bits worker-/process-ID  (0-1023)
--  • 12 bits per-millisecond sequence (0-4095)
-- ════════════════════════════════════════════════════════════════════════
CREATE OR REPLACE FUNCTION snowflake_id()
RETURNS BIGINT AS $$
DECLARE
    v_epoch          CONSTANT BIGINT := 1609459200000;          -- 2021-01-01
    v_worker_id      BIGINT   := (pg_backend_pid() % 1024);     -- 10 bits
    v_seq            BIGINT;
    v_last_ts_ms     BIGINT;
    v_ts_ms          BIGINT;                                    -- current time
    v_id             BIGINT;                                    -- output
BEGIN
    -- current time in ms
    v_ts_ms := (EXTRACT(EPOCH FROM clock_timestamp()) * 1000)::BIGINT;

    -- fetch previous state (session-local GUCs)
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

    -- guard against clock drift
    IF v_ts_ms < v_last_ts_ms THEN
        RAISE EXCEPTION
          'Clock moved backwards: refusing to generate ID for % ms',
          v_last_ts_ms - v_ts_ms;
    END IF;

    -- same millisecond → bump sequence
    IF v_ts_ms = v_last_ts_ms THEN
        v_seq := (v_seq + 1) & 4095;      -- 12-bit mask
        IF v_seq = 0 THEN
            PERFORM pg_sleep(0.001);      -- wait 1 ms
            v_ts_ms := (EXTRACT(EPOCH FROM clock_timestamp()) * 1000)::BIGINT;
        END IF;
    ELSE
        v_seq := 0;
    END IF;

    -- persist state for this session
    PERFORM set_config('snowflake.last_timestamp', v_ts_ms::text, false);
    PERFORM set_config('snowflake.sequence',       v_seq::text,    false);

    -- compose the 64-bit ID
    v_id := ((v_ts_ms - v_epoch) << 22)  |  (v_worker_id << 12)  |  v_seq;
    RETURN v_id;
END;
$$ LANGUAGE plpgsql STABLE;
EOSQL
echo "✅ Extension and function setup complete."

###############################################################################
# 4. Keep the container running by waiting on the Postgres PID.
###############################################################################
wait "${postgres_pid}"
