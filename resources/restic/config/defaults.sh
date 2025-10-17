#!/usr/bin/env bash
# Restic Resource - Default Configuration

# Repository configuration
export RESTIC_REPOSITORY="${RESTIC_REPOSITORY:-/repository}"
export RESTIC_BACKEND="${RESTIC_BACKEND:-local}"
export RESTIC_PASSWORD="${RESTIC_PASSWORD:-changeme}"  # Should be overridden in production

# Backup configuration
export RESTIC_BACKUP_SCHEDULE="${RESTIC_BACKUP_SCHEDULE:-0 2 * * *}"  # Daily at 2 AM
export RESTIC_BACKUP_PATHS="${RESTIC_BACKUP_PATHS:-/data}"
export RESTIC_EXCLUDE_PATTERNS="${RESTIC_EXCLUDE_PATTERNS:-}"

# Retention policy
export RESTIC_KEEP_DAILY="${RESTIC_KEEP_DAILY:-7}"
export RESTIC_KEEP_WEEKLY="${RESTIC_KEEP_WEEKLY:-4}"
export RESTIC_KEEP_MONTHLY="${RESTIC_KEEP_MONTHLY:-12}"
export RESTIC_KEEP_YEARLY="${RESTIC_KEEP_YEARLY:-3}"

# Performance settings
export RESTIC_COMPRESSION="${RESTIC_COMPRESSION:-auto}"
export RESTIC_CACHE_DIR="${RESTIC_CACHE_DIR:-/cache}"
export RESTIC_PARALLEL="${RESTIC_PARALLEL:-2}"

# S3 backend configuration (if using S3/MinIO)
export AWS_ACCESS_KEY_ID="${AWS_ACCESS_KEY_ID:-}"
export AWS_SECRET_ACCESS_KEY="${AWS_SECRET_ACCESS_KEY:-}"
export AWS_DEFAULT_REGION="${AWS_DEFAULT_REGION:-us-east-1}"
export RESTIC_S3_ENDPOINT="${RESTIC_S3_ENDPOINT:-}"

# Network configuration
export RESTIC_NETWORK="${RESTIC_NETWORK:-vrooli-network}"

# Resource limits
export RESTIC_MEMORY_LIMIT="${RESTIC_MEMORY_LIMIT:-2g}"
export RESTIC_CPU_LIMIT="${RESTIC_CPU_LIMIT:-2}"