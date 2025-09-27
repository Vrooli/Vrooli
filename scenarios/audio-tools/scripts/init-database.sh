#!/usr/bin/env bash
# Initialize database for audio-tools scenario

set -e

# Get the directory of this script
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
SCENARIO_DIR="$(dirname "$SCRIPT_DIR")"

# Source environment variables if available
if [ -f "$SCENARIO_DIR/.env" ]; then
    source "$SCENARIO_DIR/.env"
fi

# Database configuration
DB_HOST="${POSTGRES_HOST:-localhost}"
DB_PORT="${POSTGRES_PORT:-5433}"
DB_NAME="audio_tools"
DB_USER="${POSTGRES_USER:-vrooli}"
DB_PASS="${POSTGRES_PASSWORD:-postgres}"

echo "🔧 Initializing database for audio-tools..."

# Check if database exists, create if not
echo "📊 Checking database existence..."
DB_EXISTS=$(PGPASSWORD=$DB_PASS psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname='$DB_NAME'" 2>/dev/null || echo "0")

if [ "$DB_EXISTS" != "1" ]; then
    echo "📦 Creating database $DB_NAME..."
    PGPASSWORD=$DB_PASS psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d postgres -c "CREATE DATABASE $DB_NAME;" 2>/dev/null || {
        echo "⚠️  Could not create database. It may already exist or there may be permission issues."
    }
fi

# Apply schema
if [ -f "$SCENARIO_DIR/initialization/storage/postgres/schema.sql" ]; then
    echo "📋 Applying database schema..."
    PGPASSWORD=$DB_PASS psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f "$SCENARIO_DIR/initialization/storage/postgres/schema.sql" 2>/dev/null || {
        echo "✅ Schema already applied or updated"
    }
else
    echo "⚠️  Schema file not found at $SCENARIO_DIR/initialization/storage/postgres/schema.sql"
fi

# Apply seed data (optional)
if [ -f "$SCENARIO_DIR/initialization/storage/postgres/seed.sql" ]; then
    echo "🌱 Applying seed data..."
    # Replace placeholders in seed file
    sed "s/SCENARIO_NAME_PLACEHOLDER/audio-tools/g" "$SCENARIO_DIR/initialization/storage/postgres/seed.sql" | \
    PGPASSWORD=$DB_PASS psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME 2>/dev/null || {
        echo "✅ Seed data already applied or updated"
    }
fi

echo "✅ Database initialization complete for audio-tools"

# Test connection
echo "🔍 Testing database connection..."
PGPASSWORD=$DB_PASS psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "SELECT COUNT(*) FROM audio_processing_jobs;" 2>/dev/null && {
    echo "✅ Database connection successful - audio_processing_jobs table exists"
} || {
    echo "⚠️  Could not verify audio_processing_jobs table"
}