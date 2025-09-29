#!/usr/bin/env bash
# Test PostgreSQL database connectivity and schema

set -euo pipefail

echo "🔍 Testing database connection and schema..."

# Test connection
echo "Testing PostgreSQL resource status..."
if ! command -v resource-postgres >/dev/null 2>&1; then
    echo "⚠️  resource-postgres CLI not available; skipping database test"
    exit 0
fi

if ! resource-postgres status >/dev/null 2>&1; then
    echo "❌ PostgreSQL resource is not running"
    exit 1
fi

# Test database exists using managed CLI
echo "Testing database exists..."
DB_EXISTS_SQL="SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = 'scraper_manager') AS exists;"

if ! POSTGRES_DEFAULT_DB="postgres" resource-postgres content execute "$DB_EXISTS_SQL" | \
    awk 'NR==3 {gsub(/^[ \t]+|[ \t]+$/, "", $0); print $0}' | grep -q '^t$'; then
    echo "❌ Database scraper_manager does not exist"
    exit 1
fi

# Test schema tables exist
echo "Testing required tables exist..."
REQUIRED_TABLES=("scraping_agents" "scraping_targets" "scraping_results" "proxy_pool" "api_endpoints")

for table in "${REQUIRED_TABLES[@]}"; do
    TABLE_EXISTS_SQL="SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = '$table') AS exists;"
    if ! POSTGRES_DEFAULT_DB="scraper_manager" resource-postgres content execute "$TABLE_EXISTS_SQL" | \
        awk 'NR==3 {gsub(/^[ \t]+|[ \t]+$/, "", $0); print $0}' | grep -q '^t$'; then
        echo "❌ Required table '$table' does not exist"
        exit 1
    fi
done

echo "✅ Database connection and schema tests passed"
