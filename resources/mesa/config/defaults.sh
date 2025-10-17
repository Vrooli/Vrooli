#!/usr/bin/env bash
# Mesa Default Configuration

# Service configuration
export MESA_HOST="${MESA_HOST:-0.0.0.0}"
export MESA_PORT="${MESA_PORT:-9512}"
export MESA_WORKERS="${MESA_WORKERS:-4}"
export MESA_LOG_LEVEL="${MESA_LOG_LEVEL:-info}"

# Simulation defaults
export MESA_DEFAULT_STEPS="${MESA_DEFAULT_STEPS:-100}"
export MESA_DEFAULT_SEED="${MESA_DEFAULT_SEED:-42}"
export MESA_MAX_AGENTS="${MESA_MAX_AGENTS:-10000}"
export MESA_BATCH_SIZE="${MESA_BATCH_SIZE:-10}"

# Resource limits
export MESA_MAX_MEMORY="${MESA_MAX_MEMORY:-512M}"
export MESA_TIMEOUT="${MESA_TIMEOUT:-300}"

# Optional integrations
export MESA_REDIS_URL="${MESA_REDIS_URL:-}"
export MESA_QDRANT_URL="${MESA_QDRANT_URL:-}"
export MESA_POSTGRES_URL="${MESA_POSTGRES_URL:-}"

# Export paths
export MESA_EXPORT_PATH="${MESA_EXPORT_PATH:-/tmp/mesa_exports}"
export MESA_METRICS_PATH="${MESA_METRICS_PATH:-/tmp/mesa_metrics}"