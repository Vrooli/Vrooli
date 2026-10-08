#!/usr/bin/env bash
# Reproduce every measurement the program-runtime reuse loop cites.
# Pass a database path, or default to the live program-runtime store.
set -euo pipefail
DB="${1:-/home/matthalloran8/Vrooli/scenarios/program-runtime/data/program-runtime.db}"
HERE="$(cd "$(dirname "$0")" && pwd)"
echo "# measured $(date -u +%Y-%m-%dT%H:%M:%SZ) against ${DB}"
sqlite3 "$DB" < "${HERE}/measure.sql"
