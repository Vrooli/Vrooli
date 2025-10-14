#!/bin/bash
# Database migration helper for financial-calculators-hub
# Applies migrations in order to update existing database schemas

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MIGRATION_DIR="${SCRIPT_DIR}/../initialization/postgres/migrations"

# Check if postgres environment variables are set
if [ -z "$POSTGRES_HOST" ] || [ -z "$POSTGRES_PORT" ] || [ -z "$POSTGRES_USER" ]; then
    echo "❌ Error: Required PostgreSQL environment variables not set"
    echo ""
    echo "Required variables:"
    echo "  POSTGRES_HOST"
    echo "  POSTGRES_PORT"
    echo "  POSTGRES_USER"
    echo "  POSTGRES_PASSWORD (optional but recommended)"
    echo ""
    echo "Example:"
    echo "  export POSTGRES_HOST=localhost"
    echo "  export POSTGRES_PORT=5433"
    echo "  export POSTGRES_USER=postgres"
    echo "  export POSTGRES_PASSWORD=your_password"
    exit 1
fi

echo "🔄 Financial Calculators Hub - Database Migrations"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "Database: financial_calculators_hub"
echo "Host: $POSTGRES_HOST:$POSTGRES_PORT"
echo "User: $POSTGRES_USER"
echo ""

# Build psql connection string
PGPASSWORD="${POSTGRES_PASSWORD}"

# Check if migration directory exists
if [ ! -d "$MIGRATION_DIR" ]; then
    echo "⚠️  No migrations directory found at: $MIGRATION_DIR"
    echo "Nothing to migrate."
    exit 0
fi

# Count migrations
migration_count=$(find "$MIGRATION_DIR" -name "*.sql" -type f | wc -l)
if [ "$migration_count" -eq 0 ]; then
    echo "⚠️  No migration files found."
    exit 0
fi

echo "Found $migration_count migration(s) to apply"
echo ""

# Apply migrations in order
for migration_file in "$MIGRATION_DIR"/*.sql; do
    if [ -f "$migration_file" ]; then
        migration_name=$(basename "$migration_file")
        echo "📝 Applying migration: $migration_name"

        # Extract SQL commands and execute via resource-postgres
        # Skip \c command and execute the rest
        if grep -v '^\\c ' "$migration_file" | resource-postgres content execute --database main 2>&1; then
            echo "✅ Migration applied successfully: $migration_name"
        else
            echo "❌ Failed to apply migration: $migration_name"
            exit 1
        fi
        echo ""
    fi
done

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✅ All migrations completed successfully!"
echo ""
echo "Next steps:"
echo "  1. Restart the scenario: make stop && make start"
echo "  2. Run tests: make test"
