#!/bin/bash
# Wrapper script to start the Go API server
# This works around a lifecycle runner issue

# Use environment variables with defaults
export API_PORT="${API_PORT:-17865}"
export UI_PORT="${UI_PORT:-37865}"
export AUTH_SERVICE_URL="${AUTH_SERVICE_URL:-http://localhost:15785}"
export AUTH_PORT="${AUTH_PORT:-15785}"
export STORAGE_PATH="${STORAGE_PATH:-${PWD}/data/files}"
export POSTGRES_URL="${POSTGRES_URL:-postgresql://vrooli:lUq9qvemypKpuEeXCV6Vnxak1@localhost:5433/vrooli?sslmode=disable}"
export REDIS_URL="${REDIS_URL:-redis://localhost:6379}"
export MAX_FILE_SIZE="${MAX_FILE_SIZE:-10485760}"
export DEFAULT_EXPIRY_HOURS="${DEFAULT_EXPIRY_HOURS:-24}"
export THUMBNAIL_SIZE="${THUMBNAIL_SIZE:-200}"

# Change to the API directory
cd "$(dirname "$0")"

# Run the Go API server
exec ./device-sync-hub-api