#!/usr/bin/env bash
# Test PostgreSQL database connectivity and schema

set -euo pipefail

echo "🔍 Testing database connection and schema..."

# Test connection
echo "Testing PostgreSQL connection..."
if ! pg_isready -h localhost -p "${RESOURCE_PORTS[postgres]}" -U postgres > /dev/null 2>&1; then
    echo "❌ PostgreSQL is not ready"
    exit 1
fi

# Test database exists
echo "Testing database exists..."
if ! psql -h localhost -p "${RESOURCE_PORTS[postgres]}" -U postgres -lqt | cut -d \| -f 1 | grep -qw scraper_manager; then
    echo "❌ Database scraper_manager does not exist"
    exit 1
fi

# Test schema tables exist
echo "Testing required tables exist..."
REQUIRED_TABLES=("scraping_agents" "scraping_targets" "scraping_results" "proxy_pool" "api_endpoints")

for table in "${REQUIRED_TABLES[@]}"; do
    if ! psql -h localhost -p "${RESOURCE_PORTS[postgres]}" -U postgres -d scraper_manager \
        -c "SELECT 1 FROM information_schema.tables WHERE table_name = '$table';" | grep -q "1"; then
        echo "❌ Required table '$table' does not exist"
        exit 1
    fi
done

echo "✅ Database connection and schema tests passed"