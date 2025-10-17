#!/usr/bin/env bash
# Load platform configurations into postgres using managed resource tooling

set -euo pipefail

echo "📋 Loading platform configurations..."

if ! command -v resource-postgres >/dev/null 2>&1; then
    echo "⚠️  resource-postgres CLI not available; skipping platform configuration load"
    exit 0
fi

# Ensure the postgres resource is reachable before attempting inserts
if ! resource-postgres status >/dev/null 2>&1; then
    echo "⚠️  PostgreSQL resource is not running; skipping platform configuration load"
    exit 0
fi

CONFIG_FILE="initialization/configuration/platform-configs.json"

if [[ ! -f "$CONFIG_FILE" ]]; then
    echo "⚠️  Platform configuration file not found: $CONFIG_FILE"
    exit 0
fi

if ! command -v jq >/dev/null 2>&1; then
    echo "⚠️  jq is required to process configuration payload; skipping"
    exit 0
fi

CONFIG_PAYLOAD=$(jq -c '.' "$CONFIG_FILE" 2>/dev/null || true)

if [[ -z "$CONFIG_PAYLOAD" || "$CONFIG_PAYLOAD" == "null" ]]; then
    echo "⚠️  Platform configuration file is empty; skipping"
    exit 0
fi

# Escape single quotes for SQL literal
SQL_PAYLOAD=$(printf "%s" "$CONFIG_PAYLOAD" | sed "s/'/''/g")

SQL_STATEMENT="INSERT INTO platform_configs (data) VALUES ('$SQL_PAYLOAD') ON CONFLICT ((md5((data)::text))) DO NOTHING;"

if POSTGRES_DEFAULT_DB="scraper_manager" resource-postgres content execute "$SQL_STATEMENT" >/dev/null 2>&1; then
    echo "✅ Platform configurations loaded successfully"
else
    echo "⚠️  Failed to load platform configurations via resource-postgres"
fi
