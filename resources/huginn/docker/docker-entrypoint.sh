#!/bin/bash
set -e
# Enhanced entrypoint for Huginn with Vrooli integration

echo "Starting Huginn with enhanced entrypoint..."

# Set up PATH
export PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:$PATH"

# Set up environment for Huginn
export RAILS_ENV="${RAILS_ENV:-production}"
export DATABASE_ADAPTER="${DATABASE_ADAPTER:-postgresql}"
export DATABASE_ENCODING="${DATABASE_ENCODING:-utf8}"
export DATABASE_RECONNECT="${DATABASE_RECONNECT:-true}"
export DATABASE_POOL="${DATABASE_POOL:-20}"

# Ensure required environment variables are set
: "${DATABASE_USERNAME:?DATABASE_USERNAME must be set}"
: "${DATABASE_PASSWORD:?DATABASE_PASSWORD must be set}"
: "${DATABASE_NAME:?DATABASE_NAME must be set}"
: "${DATABASE_HOST:?DATABASE_HOST must be set}"
: "${DATABASE_PORT:?DATABASE_PORT must be set}"

# Function to wait for database
wait_for_database() {
    echo "Waiting for MySQL to be ready..."
    local max_attempts=30
    local attempt=0
    
    while [ $attempt -lt $max_attempts ]; do
        if mysqladmin ping -h "$DATABASE_HOST" -P "$DATABASE_PORT" -u "$DATABASE_USERNAME" -p"$DATABASE_PASSWORD" --silent 2>/dev/null; then
            echo "MySQL is ready!"
            return 0
        fi
        
        attempt=$((attempt + 1))
        echo "Waiting for MySQL... (attempt $attempt/$max_attempts)"
        sleep 2
    done
    
    echo "ERROR: MySQL did not become ready in time"
    return 1
}

# Create required directories
mkdir -p /app/log /app/tmp/pids /app/tmp/cache /app/uploads 2>/dev/null || true

echo "Running as user: $(whoami)"

# Wait for database to be ready
wait_for_database || exit 1

# Clean up old PID files
rm -f /app/tmp/pids/server.pid 2>/dev/null || true

# Pass control to the original Huginn entrypoint
echo "Passing control to default Huginn entrypoint..."
exec /scripts/init