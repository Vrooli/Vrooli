#!/usr/bin/env bash
# Splink Default Configuration

# Service configuration
export SPLINK_PORT="${SPLINK_PORT:-8096}"
export SPLINK_HOST="${SPLINK_HOST:-0.0.0.0}"
export SPLINK_BACKEND="${SPLINK_BACKEND:-duckdb}"

# Performance settings
export SPLINK_MAX_WORKERS="${SPLINK_MAX_WORKERS:-4}"
export SPLINK_MAX_DATASET_SIZE="${SPLINK_MAX_DATASET_SIZE:-10000000}"  # 10M records
export SPLINK_MEMORY_LIMIT="${SPLINK_MEMORY_LIMIT:-4G}"
export SPLINK_TIMEOUT="${SPLINK_TIMEOUT:-3600}"  # 1 hour for large jobs

# Backend configurations
export DUCKDB_MEMORY_LIMIT="${DUCKDB_MEMORY_LIMIT:-2GB}"
export DUCKDB_THREADS="${DUCKDB_THREADS:-4}"

# Spark configuration (optional)
export SPARK_MASTER_URL="${SPARK_MASTER_URL:-}"
export SPARK_EXECUTOR_MEMORY="${SPARK_EXECUTOR_MEMORY:-2g}"
export SPARK_EXECUTOR_CORES="${SPARK_EXECUTOR_CORES:-2}"

# Storage configuration
export DATA_DIR="${VROOLI_ROOT:-$HOME/Vrooli}/.vrooli/data/splink"
export DATASETS_DIR="${DATA_DIR}/datasets"
export RESULTS_DIR="${DATA_DIR}/results"
export JOBS_DIR="${DATA_DIR}/jobs"

# Database configuration (optional)
export POSTGRES_HOST="${POSTGRES_HOST:-localhost}"
export POSTGRES_PORT="${POSTGRES_PORT:-5433}"
export POSTGRES_DB="${POSTGRES_DB:-splink}"
export POSTGRES_USER="${POSTGRES_USER:-splink}"
export POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-}"

# Redis configuration (optional)
export REDIS_HOST="${REDIS_HOST:-localhost}"
export REDIS_PORT="${REDIS_PORT:-6380}"
export REDIS_DB="${REDIS_DB:-0}"

# MinIO configuration (optional)
export MINIO_HOST="${MINIO_HOST:-localhost}"
export MINIO_PORT="${MINIO_PORT:-9000}"
export MINIO_ACCESS_KEY="${MINIO_ACCESS_KEY:-}"
export MINIO_SECRET_KEY="${MINIO_SECRET_KEY:-}"
export MINIO_BUCKET="${MINIO_BUCKET:-splink}"

# Logging configuration
export LOG_LEVEL="${LOG_LEVEL:-INFO}"
export LOG_FORMAT="${LOG_FORMAT:-json}"
export LOG_FILE="${VROOLI_ROOT:-$HOME/Vrooli}/.vrooli/logs/resources/splink.log"

# Health check configuration
export HEALTH_CHECK_INTERVAL="${HEALTH_CHECK_INTERVAL:-30}"
export HEALTH_CHECK_TIMEOUT="${HEALTH_CHECK_TIMEOUT:-5}"

# API configuration
export API_RATE_LIMIT="${API_RATE_LIMIT:-100}"  # requests per minute
export API_MAX_UPLOAD_SIZE="${API_MAX_UPLOAD_SIZE:-100MB}"
export API_CORS_ORIGINS="${API_CORS_ORIGINS:-*}"

# Default linkage settings
export DEFAULT_THRESHOLD="${DEFAULT_THRESHOLD:-0.9}"
export DEFAULT_LINK_TYPE="${DEFAULT_LINK_TYPE:-one_to_one}"
export DEFAULT_BLOCKING_RULES="${DEFAULT_BLOCKING_RULES:-}"