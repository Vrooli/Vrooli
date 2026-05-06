#!/bin/bash
# Script to initialize endpoints in the api-library database

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SEED_FILE="${SCRIPT_DIR}/../initialization/postgres/seed_endpoints.sql"

# Database connection details
DB_HOST="${POSTGRES_HOST:-localhost}"
DB_PORT="${POSTGRES_PORT:-19432}"
DB_USER="${POSTGRES_USER:-vrooli}"
DB_PASSWORD="${POSTGRES_PASSWORD:-vrooli123}"
DB_NAME="${POSTGRES_DB:-api_library}"

echo "🚀 Initializing endpoints in api-library database..."

# Function to execute SQL
execute_sql() {
    local sql="$1"
    docker exec -i vrooli-postgres-main sh -c "PGPASSWORD='$DB_PASSWORD' psql -h localhost -U $DB_USER -d $DB_NAME -c \"$sql\""
}

# Check if endpoints table is empty
ENDPOINT_COUNT=$(docker exec -i vrooli-postgres-main sh -c "PGPASSWORD='$DB_PASSWORD' psql -h localhost -U $DB_USER -d $DB_NAME -t -c 'SELECT COUNT(*) FROM endpoints;'" 2>/dev/null | tr -d ' ' | tr -d '\n')

# Set default if empty
if [ -z "$ENDPOINT_COUNT" ]; then
    ENDPOINT_COUNT="0"
fi

if [ "$ENDPOINT_COUNT" -eq "0" ]; then
    echo "📝 No endpoints found, seeding data..."
    
    # Execute the seed file
    docker exec -i vrooli-postgres-main sh -c "PGPASSWORD='$DB_PASSWORD' psql -h localhost -U $DB_USER -d $DB_NAME" < "$SEED_FILE"
    
    echo "✅ Endpoints seeded successfully!"
else
    echo "ℹ️  Endpoints table already contains $ENDPOINT_COUNT entries, skipping seed"
fi

# Verify the endpoints were added
NEW_COUNT=$(docker exec -i vrooli-postgres-main sh -c "PGPASSWORD='$DB_PASSWORD' psql -h localhost -U $DB_USER -d $DB_NAME -t -c 'SELECT COUNT(*) FROM endpoints;'" 2>/dev/null | tr -d ' ' | tr -d '\n')
echo "📊 Total endpoints in database: $NEW_COUNT"
