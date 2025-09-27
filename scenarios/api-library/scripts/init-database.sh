#!/bin/bash

# Database initialization script for api-library
set -e

echo "🔧 Initializing api-library database..."

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
    
    if psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f "$sql_file" &>/dev/null; then
        echo "    ✅ $description applied successfully"
    else
        echo "    ⚠️  Some warnings while applying $description (continuing...)"
    fi
}

# Apply schemas
echo "📋 Applying database schemas..."
execute_sql "initialization/postgres/schema.sql" "main schema"
execute_sql "initialization/postgres/schema_update_v2.sql" "v2 updates"
execute_sql "initialization/postgres/schema_integration_snippets.sql" "integration snippets"

# Apply seed data
echo "🌱 Applying seed data..."
execute_sql "initialization/postgres/seed.sql" "initial API data"

echo "✅ Database initialization complete!"
echo ""
echo "📊 Database statistics:"
export PGPASSWORD="$DB_PASS"
API_COUNT=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT COUNT(*) FROM apis" 2>/dev/null || echo "0")
SNIPPET_COUNT=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT COUNT(*) FROM integration_snippets" 2>/dev/null || echo "0")

echo "  • APIs loaded: $API_COUNT"
echo "  • Snippets loaded: $SNIPPET_COUNT"