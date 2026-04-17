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

# Create database if it doesn't exist (create-database is idempotent)
echo "📦 Creating database: $DB_NAME"
resource-postgres content create-database --instance main "$DB_NAME"

# Reset any pre-existing tables so the schema applies cleanly
resource-postgres content execute --instance main --database "$DB_NAME" \
    --sql "DROP TABLE IF EXISTS escalations, messages, conversations, intent_patterns, daily_analytics, chatbots CASCADE;"

# Run schema.sql
if [ -f "$SCENARIO_ROOT/initialization/storage/postgres/schema.sql" ]; then
    echo "📄 Applying schema..."
    resource-postgres content execute --instance main --database "$DB_NAME" \
        --file "$SCENARIO_ROOT/initialization/storage/postgres/schema.sql"
    echo "✅ Schema applied successfully"
else
    echo "⚠️  Schema file not found"
fi

# Run seed.sql if it exists
if [ -f "$SCENARIO_ROOT/initialization/storage/postgres/seed.sql" ]; then
    echo "🌱 Applying seed data..."
    resource-postgres content execute --instance main --database "$DB_NAME" \
        --file "$SCENARIO_ROOT/initialization/storage/postgres/seed.sql"
    echo "✅ Seed data applied successfully"
else
    echo "ℹ️  No seed data file found (this is OK)"
fi

echo "🎉 Database initialization complete!"