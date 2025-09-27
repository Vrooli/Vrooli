#!/usr/bin/env bash
################################################################################
# SQLite Resource - Default Configuration
# 
# This file defines default settings for the SQLite resource.
# These can be overridden by environment variables.
################################################################################

# Database storage location
VROOLI_DATA="${VROOLI_DATA:-${HOME}/.vrooli/data}"
export SQLITE_DATABASE_PATH="${SQLITE_DATABASE_PATH:-${VROOLI_DATA}/sqlite/databases}"

# Default database settings
export SQLITE_MAX_CONNECTIONS="${SQLITE_MAX_CONNECTIONS:-10}"
export SQLITE_JOURNAL_MODE="${SQLITE_JOURNAL_MODE:-WAL}"  # Write-Ahead Logging
export SQLITE_BUSY_TIMEOUT="${SQLITE_BUSY_TIMEOUT:-10000}"  # milliseconds (increased for better concurrency)
export SQLITE_CACHE_SIZE="${SQLITE_CACHE_SIZE:-2000}"      # pages
export SQLITE_PAGE_SIZE="${SQLITE_PAGE_SIZE:-4096}"        # bytes

# Backup settings
export SQLITE_BACKUP_PATH="${SQLITE_BACKUP_PATH:-${VROOLI_DATA}/sqlite/backups}"
export SQLITE_BACKUP_RETENTION_DAYS="${SQLITE_BACKUP_RETENTION_DAYS:-7}"

# Performance settings
export SQLITE_SYNCHRONOUS="${SQLITE_SYNCHRONOUS:-NORMAL}"  # NORMAL, FULL, or OFF
export SQLITE_TEMP_STORE="${SQLITE_TEMP_STORE:-MEMORY}"    # MEMORY or FILE
export SQLITE_MMAP_SIZE="${SQLITE_MMAP_SIZE:-268435456}"   # 256MB memory-mapped I/O

# Security settings
export SQLITE_FILE_PERMISSIONS="${SQLITE_FILE_PERMISSIONS:-600}"  # Read/write for owner only

# CLI settings
export SQLITE_CLI_TIMEOUT="${SQLITE_CLI_TIMEOUT:-30}"  # Command timeout in seconds
export SQLITE_VERBOSE="${SQLITE_VERBOSE:-false}"        # Verbose output

# Health check settings (for compatibility, even though SQLite is serverless)
export SQLITE_HEALTH_CHECK_INTERVAL="${SQLITE_HEALTH_CHECK_INTERVAL:-60}"  # seconds

# Replication settings
export SQLITE_REPLICATION_ENABLED="${SQLITE_REPLICATION_ENABLED:-true}"
export SQLITE_REPLICATION_PATH="${SQLITE_REPLICATION_PATH:-${VROOLI_DATA}/sqlite/replicas}"
export SQLITE_REPLICATION_SYNC_INTERVAL="${SQLITE_REPLICATION_SYNC_INTERVAL:-300}"  # seconds (5 minutes)
export SQLITE_REPLICATION_MONITOR_ENABLED="${SQLITE_REPLICATION_MONITOR_ENABLED:-false}"  # Auto-monitoring off by default
export SQLITE_REPLICATION_MONITOR_INTERVAL="${SQLITE_REPLICATION_MONITOR_INTERVAL:-60}"  # seconds

# Web UI settings
export SQLITE_UI_PORT="${SQLITE_UI_PORT:-8297}"
export SQLITE_UI_HOST="${SQLITE_UI_HOST:-127.0.0.1}"

# Resource metadata
export SQLITE_VERSION="${SQLITE_VERSION:-3.43.0}"
export SQLITE_RESOURCE_VERSION="${SQLITE_RESOURCE_VERSION:-1.0.0}"