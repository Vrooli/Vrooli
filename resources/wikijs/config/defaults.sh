#!/usr/bin/env bash
################################################################################
# Wiki.js Default Configuration - v2.0 Universal Contract Compliant
# 
# Default configuration values for Wiki.js resource
#
################################################################################

# Service configuration
export WIKIJS_CONTAINER="${WIKIJS_CONTAINER:-wikijs}"
export WIKIJS_IMAGE="${WIKIJS_IMAGE:-requarks/wiki:2}"
export WIKIJS_PORT="${WIKIJS_PORT:-3010}"

# Database configuration
export WIKIJS_DB_TYPE="${WIKIJS_DB_TYPE:-postgres}"
export WIKIJS_DB_HOST="${WIKIJS_DB_HOST:-localhost}"
export WIKIJS_DB_PORT="${WIKIJS_DB_PORT:-5433}"
export WIKIJS_DB_NAME="${WIKIJS_DB_NAME:-wikijs}"
export WIKIJS_DB_USER="${WIKIJS_DB_USER:-wikijs}"
export WIKIJS_DB_PASS="${WIKIJS_DB_PASS:-wikijs_pass}"

# Storage configuration
export WIKIJS_DATA_DIR="${WIKIJS_DATA_DIR:-${var_DATA_DIR:-/var/lib/vrooli}/resources/wikijs}"
export WIKIJS_BACKUP_DIR="${WIKIJS_BACKUP_DIR:-${WIKIJS_DATA_DIR}/backups}"

# Git configuration (optional)
export WIKIJS_GIT_URL="${WIKIJS_GIT_URL:-}"
export WIKIJS_GIT_BRANCH="${WIKIJS_GIT_BRANCH:-main}"
export WIKIJS_GIT_USER="${WIKIJS_GIT_USER:-}"
export WIKIJS_GIT_PASS="${WIKIJS_GIT_PASS:-}"

# Search configuration
export WIKIJS_SEARCH_ENGINE="${WIKIJS_SEARCH_ENGINE:-db}"  # db, elasticsearch, algolia

# Authentication
export WIKIJS_ADMIN_EMAIL="${WIKIJS_ADMIN_EMAIL:-admin@vrooli.local}"
export WIKIJS_ADMIN_PASS="${WIKIJS_ADMIN_PASS:-Admin123!}"

# Performance tuning
export WIKIJS_MAX_FILE_SIZE="${WIKIJS_MAX_FILE_SIZE:-5242880}"  # 5MB
export WIKIJS_SESSION_SECRET="${WIKIJS_SESSION_SECRET:-$(openssl rand -hex 32 2>/dev/null || echo 'default-session-secret')}"

# Startup configuration
export WIKIJS_STARTUP_TIMEOUT="${WIKIJS_STARTUP_TIMEOUT:-35}"
export WIKIJS_HEALTH_CHECK_INTERVAL="${WIKIJS_HEALTH_CHECK_INTERVAL:-5}"
export WIKIJS_HEALTH_CHECK_RETRIES="${WIKIJS_HEALTH_CHECK_RETRIES:-10}"