#!/usr/bin/env bash
# Apply BAS test seed data for tidiness-manager integration tests
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "$SCRIPT_DIR/../../.." && pwd)"

# Source database configuration
if [ -f "$SCENARIO_DIR/.env" ]; then
    set -a
    source "$SCENARIO_DIR/.env"
    set +a
fi

# Get API port for database access
API_PORT="${API_PORT:-$(vrooli scenario port tidiness-manager API_PORT 2>/dev/null | grep -oP '(?<=API_PORT=)\d+' || echo '16820')}"
DB_NAME="${DB_NAME:-tidiness_manager}"
DB_USER="${DB_USER:-tidiness}"
DB_PASSWORD="${DB_PASSWORD:-tidiness_pass}"
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"

echo "🌱 Seeding test data for tidiness-manager integration tests..."

# Insert test scenario data via API to ensure proper schema
SEED_DATA='[
  {
    "scenario": "picker-wheel",
    "file_path": "api/main.go",
    "category": "lint",
    "severity": "medium",
    "title": "Unused variable",
    "description": "Variable foo is declared but never used",
    "line_number": 42,
    "column_number": 5
  },
  {
    "scenario": "picker-wheel",
    "file_path": "ui/src/App.tsx",
    "category": "type",
    "severity": "high",
    "title": "Type mismatch",
    "description": "Expected string but got number",
    "line_number": 15,
    "column_number": 10
  },
  {
    "scenario": "tidiness-manager",
    "file_path": "api/handlers.go",
    "category": "lint",
    "severity": "low",
    "title": "Line too long",
    "description": "Line exceeds 120 characters",
    "line_number": 100,
    "column_number": 1
  }
]'

# Insert issues via API
echo "$SEED_DATA" | jq -c '.[]' | while IFS= read -r issue; do
    response=$(curl -s -X POST "http://localhost:${API_PORT}/api/v1/agent/issues" \
        -H "Content-Type: application/json" \
        -d "$issue")
    if echo "$response" | jq -e '.id' >/dev/null 2>&1; then
        : # Success - issue created
    else
        echo "⚠️  Failed to insert issue (may already exist): $response" >&2
    fi
done

echo "✅ Test seed data applied successfully"
