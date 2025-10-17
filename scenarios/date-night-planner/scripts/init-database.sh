#!/bin/bash
# Initialize Date Night Planner database schema

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(dirname "$SCRIPT_DIR")"

# Database connection details
DB_HOST="${POSTGRES_HOST:-localhost}"
DB_PORT="${POSTGRES_PORT:-5432}"
DB_USER="${POSTGRES_USER:-postgres}"
DB_PASSWORD="${POSTGRES_PASSWORD:-postgres}"
DB_NAME="${POSTGRES_DB:-vrooli}"

# Check if PostgreSQL is running
echo "Checking PostgreSQL availability..."
if ! nc -z "$DB_HOST" "$DB_PORT" 2>/dev/null; then
    echo "❌ PostgreSQL is not available at $DB_HOST:$DB_PORT"
    echo "   Please start PostgreSQL first: vrooli resource postgres start"
    exit 1
fi

echo "✅ PostgreSQL is available"

# Use docker to execute SQL file
SCHEMA_FILE="$SCENARIO_DIR/initialization/storage/postgres/schema.sql"
SEED_FILE="$SCENARIO_DIR/initialization/storage/postgres/seed.sql"

if [ ! -f "$SCHEMA_FILE" ]; then
    echo "❌ Schema file not found: $SCHEMA_FILE"
    exit 1
fi

echo "📋 Applying database schema..."
docker exec -i vrooli-postgres psql -U "$DB_USER" -d "$DB_NAME" < "$SCHEMA_FILE" 2>&1 | grep -v "NOTICE" | grep -v "already exists" || true

echo "✅ Schema applied successfully"

# Apply seed data if it exists
if [ -f "$SEED_FILE" ]; then
    echo "🌱 Applying seed data..."
    docker exec -i vrooli-postgres psql -U "$DB_USER" -d "$DB_NAME" < "$SEED_FILE" 2>&1 | grep -v "NOTICE" || true
    echo "✅ Seed data applied"
fi

# Verify schema was created
echo "🔍 Verifying schema creation..."
SCHEMA_EXISTS=$(docker exec -i vrooli-postgres psql -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT EXISTS(SELECT 1 FROM information_schema.schemata WHERE schema_name = 'date_night_planner');" | tr -d '[:space:]')

if [ "$SCHEMA_EXISTS" = "t" ]; then
    echo "✅ Database schema 'date_night_planner' verified successfully"
    
    # List tables in the schema
    echo ""
    echo "📊 Tables in date_night_planner schema:"
    docker exec -i vrooli-postgres psql -U "$DB_USER" -d "$DB_NAME" -c "SELECT table_name FROM information_schema.tables WHERE table_schema = 'date_night_planner' ORDER BY table_name;" | grep -E '^\s+\w+' || true
else
    echo "❌ Schema verification failed"
    exit 1
fi

echo ""
echo "🎉 Date Night Planner database initialization complete!"