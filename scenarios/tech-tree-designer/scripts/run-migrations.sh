#!/usr/bin/env bash
set -euo pipefail

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
APP_ROOT="$(cd "${SCENARIO_DIR}/../.." && pwd)"

# shellcheck disable=SC1091
source "${APP_ROOT}/resources/postgres/config/exports.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/resources/postgres/config/defaults.sh"

if [[ -z "${POSTGRES_PASSWORD:-}" ]]; then
  echo "[tech-tree-designer] ERROR: POSTGRES_PASSWORD is not set by the resource exports" >&2
  exit 1
fi

MIGRATIONS_DIR="${SCENARIO_DIR}/initialization/postgres/migrations"

echo "Running database migrations for tech-tree-designer..."

# Check if migration directory exists
if [ ! -d "$MIGRATIONS_DIR" ]; then
    echo "No migrations directory found at: $MIGRATIONS_DIR"
    exit 0
fi

# Get list of migration files sorted by name
MIGRATION_FILES=$(find "$MIGRATIONS_DIR" -name "*.sql" -type f | sort)

if [ -z "$MIGRATION_FILES" ]; then
    echo "No migration files found"
    exit 0
fi

run_migration() {
  local file=$1
  local sql=$(cat "$file")

  if command -v psql >/dev/null 2>&1; then
    PGPASSWORD="${POSTGRES_PASSWORD}" \
      psql -h "${POSTGRES_HOST}" \
           -p "${POSTGRES_PORT}" \
           -U "${POSTGRES_USER}" \
           -d "${POSTGRES_DB}" \
           -v ON_ERROR_STOP=1 <<EOF
$sql
EOF
    return
  fi

  if ! command -v docker >/dev/null 2>&1; then
    echo "[tech-tree-designer] ERROR: Neither psql nor docker is available to run migrations" >&2
    exit 1
  fi

  local container_name="${POSTGRES_CONTAINER_PREFIX}-${POSTGRES_INSTANCE:-main}"
  if ! docker ps --format '{{.Names}}' | grep -q "^${container_name}$"; then
    echo "[tech-tree-designer] ERROR: Postgres container '${container_name}' is not running" >&2
    exit 1
  fi

  docker exec -e PGPASSWORD="${POSTGRES_PASSWORD}" "$container_name" \
    psql -h localhost -p 5432 -U "${POSTGRES_USER}" -d "${POSTGRES_DB}" -v ON_ERROR_STOP=1 <<EOF
$sql
EOF
}

# Run each migration
for MIGRATION_FILE in $MIGRATION_FILES; do
    MIGRATION_NAME=$(basename "$MIGRATION_FILE")
    echo "  → Running migration: $MIGRATION_NAME"
    run_migration "$MIGRATION_FILE"
done

echo "✅ All migrations completed successfully"
