#!/usr/bin/env bash
# Load platform configurations into database

set -euo pipefail

echo "📋 Loading platform configurations..."

# Wait for PostgreSQL to be ready
echo "Waiting for PostgreSQL to be ready..."
timeout 60 bash -c 'until pg_isready -h localhost -p ${RESOURCE_PORTS[postgres]} > /dev/null 2>&1; do sleep 1; done'

# Load platform configurations from JSON file
CONFIG_FILE="initialization/configuration/platform-configs.json"

if [[ -f "$CONFIG_FILE" ]]; then
    echo "Loading platform configurations from $CONFIG_FILE..."
    
    # Insert or update platform configurations
    psql -h localhost -p "${RESOURCE_PORTS[postgres]}" -U postgres -d scraper_manager \
        -c "INSERT INTO platform_configs (data) VALUES ('$(cat "$CONFIG_FILE" | tr -d '\n' | sed "s/'/''/'g")') 
            ON CONFLICT ((md5(data::text))) DO NOTHING;" || echo "Configuration already loaded"
    
    echo "✅ Platform configurations loaded successfully"
else
    echo "⚠️  Platform configuration file not found: $CONFIG_FILE"
fi