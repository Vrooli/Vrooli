#!/usr/bin/env bash
# Verifies that the tech tree seed data exists in the shared Postgres resource.
set -euo pipefail

POSTGRES_DB="${POSTGRES_DB:-vrooli_tech_tree_designer}"

count_rows() {
  local table=$1
  resource-postgres content execute --database "${POSTGRES_DB}" --sql "SELECT COUNT(*) FROM ${table};" 2>/dev/null \
    | awk '/^[[:space:]]*[0-9]+[[:space:]]*$/ { gsub(/ /, ""); print; exit }'
}

sector_count=$(count_rows sectors)
stage_count=$(count_rows progression_stages)
milestone_count=$(count_rows strategic_milestones)

if [[ "${sector_count:-0}" -eq 0 || "${stage_count:-0}" -eq 0 ]]; then
  echo "[tech-tree-designer] ERROR: seed data missing (sectors=${sector_count:-0}, stages=${stage_count:-0})" >&2
  exit 1
fi

echo "[tech-tree-designer] Verified Postgres seed data: sectors=${sector_count} stages=${stage_count} milestones=${milestone_count:-0}"
