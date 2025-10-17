#!/bin/bash

# Migration script for api-library database
set -e

echo "🔧 Applying database migrations for api-library..."

# Database connection details
DB_HOST="${POSTGRES_HOST:-localhost}"
DB_PORT="${POSTGRES_PORT:-5433}"
DB_USER="${POSTGRES_USER:-vrooli}"
DB_PASS="${POSTGRES_PASSWORD:-vrooli}"
DB_NAME="${POSTGRES_DB:-api_library}"

# Check if postgres is running
if ! nc -z "$DB_HOST" "$DB_PORT" 2>/dev/null; then
    echo "❌ PostgreSQL is not running on $DB_HOST:$DB_PORT"
    echo "   Start it with: vrooli resource postgres manage start"
    exit 1
fi

# Function to execute SQL file
execute_sql() {
    local sql_file=$1
    local description=$2
    
    echo "  → Applying $description..."
    export PGPASSWORD="$DB_PASS"
    
    if psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f "$sql_file" 2>&1; then
        echo "    ✅ $description applied successfully"
        return 0
    else
        echo "    ❌ Failed to apply $description"
        return 1
    fi
}

# Apply migrations
echo "📋 Applying database migrations..."

# First ensure uuid extension is available
export PGPASSWORD="$DB_PASS"
psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";" 2>/dev/null || true

# Apply analytics migration
if execute_sql "initialization/postgres/migration_004_analytics.sql" "Analytics migration"; then
    echo "✅ Analytics tables created"
fi

# Apply webhooks migration
if execute_sql "initialization/postgres/migration_005_webhooks.sql" "Webhooks migration"; then
    echo "✅ Webhook tables created"
fi

# Apply health monitoring migration
if execute_sql "initialization/postgres/migration_006_health_monitoring.sql" "Health monitoring migration"; then
    echo "✅ Health monitoring tables created"
fi

echo ""
echo "📊 Migration status:"
export PGPASSWORD="$DB_PASS"

# Check if new tables exist
WEBHOOK_EXISTS=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'webhook_subscriptions'" 2>/dev/null || echo "0")
HEALTH_EXISTS=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'health_checks'" 2>/dev/null || echo "0")
ANALYTICS_EXISTS=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'api_usage_events'" 2>/dev/null || echo "0")

if [ "${WEBHOOK_EXISTS// /}" = "1" ]; then
    echo "  ✅ Webhook tables exist"
else
    echo "  ⚠️  Webhook tables missing"
fi

if [ "${HEALTH_EXISTS// /}" = "1" ]; then
    echo "  ✅ Health monitoring tables exist"
else
    echo "  ⚠️  Health monitoring tables missing"
fi

if [ "${ANALYTICS_EXISTS// /}" = "1" ]; then
    echo "  ✅ Analytics tables exist"  
else
    echo "  ⚠️  Analytics tables missing"
fi

echo ""
echo "✅ Migration process complete!"