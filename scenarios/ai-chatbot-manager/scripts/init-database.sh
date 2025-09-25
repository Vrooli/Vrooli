#!/bin/bash

# Database initialization script for AI Chatbot Manager
set -e

echo "🔄 Initializing AI Chatbot Manager database..."

# Get database credentials
DB_HOST="${POSTGRES_HOST:-localhost}"
DB_PORT="${POSTGRES_PORT:-5433}"
DB_USER="${POSTGRES_USER:-postgres}"
DB_PASS="${POSTGRES_PASSWORD:-postgres}"
DB_NAME="${POSTGRES_DB:-ai_chatbot_manager}"

# Find the scenario root directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
SCENARIO_ROOT="$(dirname "$SCRIPT_DIR")"

echo "📁 Using scenario root: $SCENARIO_ROOT"

# Check if resource-postgres is available
if ! command -v resource-postgres &> /dev/null; then
    echo "❌ resource-postgres CLI not found"
    exit 1
fi

# Create database if it doesn't exist
echo "📦 Creating database: $DB_NAME"
resource-postgres content execute --instance main "CREATE DATABASE $DB_NAME;" 2>/dev/null || echo "Database may already exist"

# Run schema.sql
if [ -f "$SCENARIO_ROOT/initialization/storage/postgres/schema.sql" ]; then
    echo "📄 Applying schema..."
    # Use a temporary file to combine commands
    TEMP_SQL=$(mktemp)
    echo "\\c $DB_NAME" > "$TEMP_SQL"
    cat "$SCENARIO_ROOT/initialization/storage/postgres/schema.sql" >> "$TEMP_SQL"
    
    resource-postgres content execute --instance main --file "$TEMP_SQL"
    rm -f "$TEMP_SQL"
    echo "✅ Schema applied successfully"
else
    echo "⚠️  Schema file not found"
fi

# Run seed.sql if it exists
if [ -f "$SCENARIO_ROOT/initialization/storage/postgres/seed.sql" ]; then
    echo "🌱 Applying seed data..."
    TEMP_SQL=$(mktemp)
    echo "\\c $DB_NAME" > "$TEMP_SQL"
    cat "$SCENARIO_ROOT/initialization/storage/postgres/seed.sql" >> "$TEMP_SQL"
    
    resource-postgres content execute --instance main --file "$TEMP_SQL"
    rm -f "$TEMP_SQL"
    echo "✅ Seed data applied successfully"
else
    echo "ℹ️  No seed data file found (this is OK)"
fi

echo "🎉 Database initialization complete!"