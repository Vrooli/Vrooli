#!/usr/bin/env bash
# Verifies that the tech tree seed data exists in the shared Postgres resource.
set -euo pipefail

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

POSTGRES_HOST="${POSTGRES_HOST:-localhost}"
POSTGRES_PORT="${POSTGRES_PORT:-5433}"
POSTGRES_USER="${POSTGRES_USER:-vrooli}"
POSTGRES_DB="${POSTGRES_DB:-tech_tree_designer}"
POSTGRES_CONTAINER_NAME="${POSTGRES_CONTAINER_NAME:-vrooli-postgres-main}"

if [[ -z "${POSTGRES_PASSWORD:-}" ]]; then
  echo "[tech-tree-designer] ERROR: POSTGRES_PASSWORD is not set by the lifecycle environment" >&2
  exit 1
fi

run_psql() {
  local sql=$1
  if command -v psql >/dev/null 2>&1; then
    PGPASSWORD="${POSTGRES_PASSWORD}" \
      psql -h "${POSTGRES_HOST}" \
           -p "${POSTGRES_PORT}" \
           -U "${POSTGRES_USER}" \
           -d "${POSTGRES_DB}" \
           -v ON_ERROR_STOP=1 -tA -F ',' -c "$sql"
    return
  fi

  if ! command -v docker >/dev/null 2>&1; then
    echo "[tech-tree-designer] ERROR: Neither psql nor docker is available to verify Postgres" >&2
    exit 1
  fi

  if ! docker ps --format '{{.Names}}' | grep -q "^${POSTGRES_CONTAINER_NAME}$"; then
    echo "[tech-tree-designer] ERROR: Postgres container '${POSTGRES_CONTAINER_NAME}' is not running" >&2
    exit 1
  fi

  docker exec -e PGPASSWORD="${POSTGRES_PASSWORD}" "${POSTGRES_CONTAINER_NAME}" \
    psql -h localhost -p 5432 -U "${POSTGRES_USER}" -d "${POSTGRES_DB}" -v ON_ERROR_STOP=1 -tA -F ',' -c "$sql"
}

query_result=$(run_psql 'SELECT (SELECT COUNT(*) FROM sectors),(SELECT COUNT(*) FROM progression_stages),(SELECT COUNT(*) FROM strategic_milestones);')
IFS=',' read -r sector_count stage_count milestone_count <<<"${query_result:-0,0,0}"

if [[ "${sector_count:-0}" -eq 0 || "${stage_count:-0}" -eq 0 ]]; then
  echo "[tech-tree-designer] ERROR: seed data missing (sectors=${sector_count:-0}, stages=${stage_count:-0})" >&2
  exit 1
fi

echo "[tech-tree-designer] Verified Postgres seed data: sectors=${sector_count} stages=${stage_count} milestones=${milestone_count:-0}"
