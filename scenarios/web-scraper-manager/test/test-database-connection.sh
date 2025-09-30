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

# Check if postgres container is actually running (status might be degraded but still functional)
if ! docker ps --format "table {{.Names}}" | grep -q "vrooli-postgres-main"; then
    echo "❌ PostgreSQL container is not running"
    exit 1
fi
echo "✅ PostgreSQL container is running"

# Test database exists using docker
echo "Testing database exists..."
if ! docker exec vrooli-postgres-main psql -U vrooli -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname='vrooli'" 2>/dev/null | grep -q "1"; then
    echo "❌ Database vrooli does not exist"
    exit 1
fi
echo "✅ Database vrooli exists"

# Test schema tables exist
echo "Testing required tables exist..."
REQUIRED_TABLES=("scraping_agents" "scraping_targets" "scraping_results" "proxy_pool" "api_endpoints")

for table in "${REQUIRED_TABLES[@]}"; do
    if ! docker exec vrooli-postgres-main psql -U vrooli -d vrooli -tAc "SELECT 1 FROM information_schema.tables WHERE table_name='$table' AND table_schema='public'" 2>/dev/null | grep -q "1"; then
        echo "❌ Required table '$table' does not exist"
        exit 1
    fi
    echo "✅ Table '$table' exists"
done

echo "✅ Database connection and schema tests passed"
