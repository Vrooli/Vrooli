#!/bin/bash

# Migration script for api-library database using Docker
set -e

echo "🔧 Applying database migrations for api-library using Docker..."

# Database connection details
DB_HOST="${POSTGRES_HOST:-localhost}"
DB_PORT="${POSTGRES_PORT:-5433}"
DB_USER="${POSTGRES_USER:-vrooli}"
DB_PASS="${POSTGRES_PASSWORD:-vrooli}"
DB_NAME="${POSTGRES_DB:-api_library}"

# Container name
CONTAINER="vrooli-postgres-main"

# Check if postgres container is running
if ! docker ps | grep -q "$CONTAINER"; then
    echo "❌ PostgreSQL container $CONTAINER is not running"
    exit 1
fi

# Function to execute SQL file using docker
execute_sql() {
    local sql_file=$1
    local description=$2
    
    echo "  → Applying $description..."
    
    # Copy SQL file to container
    docker cp "$sql_file" "$CONTAINER:/tmp/migration.sql"
    
    # Execute the SQL
    if docker exec -e PGPASSWORD="$DB_PASS" "$CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -f "/tmp/migration.sql" 2>&1; then
        echo "    ✅ $description applied successfully"
        return 0
    else
        echo "    ❌ Failed to apply $description"
        return 1
    fi
}

# First ensure uuid extension is available
echo "📋 Preparing database..."
docker exec -e PGPASSWORD="$DB_PASS" "$CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -c "CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";" 2>/dev/null || true

echo "📋 Applying database migrations..."

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

# Check if new tables exist
WEBHOOK_EXISTS=$(docker exec -e PGPASSWORD="$DB_PASS" "$CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'webhook_subscriptions'" 2>/dev/null || echo "0")
HEALTH_EXISTS=$(docker exec -e PGPASSWORD="$DB_PASS" "$CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'health_checks'" 2>/dev/null || echo "0")
ANALYTICS_EXISTS=$(docker exec -e PGPASSWORD="$DB_PASS" "$CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'api_usage_events'" 2>/dev/null || echo "0")

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