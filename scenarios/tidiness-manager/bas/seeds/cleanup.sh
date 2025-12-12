#!/usr/bin/env bash
# Clean up BAS test seed data for tidiness-manager integration tests
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "$SCRIPT_DIR/../../.." && pwd)"

# Source database configuration
if [ -f "$SCENARIO_DIR/.env" ]; then
    set -a
    source "$SCENARIO_DIR/.env"
    set +a
fi

DB_NAME="${DB_NAME:-tidiness_manager}"
DB_USER="${DB_USER:-tidiness}"
DB_PASSWORD="${DB_PASSWORD:-tidiness_pass}"
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"

echo "🧹 Cleaning up test seed data..."

# Clean up test issues
PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c \
    "DELETE FROM issues WHERE scenario IN ('picker-wheel', 'tidiness-manager')" 2>/dev/null || true

echo "✅ Test seed data cleaned up"
