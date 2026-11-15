#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
dbm_root="$repo_root/scenarios/data-backup-manager"
tmp_root="$(mktemp -d "${TMPDIR:-/tmp}/dbm-db6-phase.XXXXXX")"
matrix_output="$tmp_root/support-matrix.md"
trap 'rm -rf "$tmp_root"' EXIT
cp "$dbm_root/docs/reference/support-matrix.md" "$matrix_output"
export DBM_PLATFORM_MATRIX_PATH="$matrix_output"
export VROOLI_HOME="$tmp_root/vrooli"
export DBM_PLATFORM_RECEIPTS_PATH="$tmp_root/dbm-platform-receipts.jsonl"

# This focused proving ground exercises the actual owners with their existing
# fake observers/remediators. It never calls host volume commands or touches a
# repository. Test Genie owns the DBM scenario unit phase; this supplemental
# runner includes the control-plane owner, which is a separate Go module.
(
  cd "$repo_root"
  go test ./internal/volumeremediation
)
(
  cd "$dbm_root/api"
  go test ./internal/destinationreadiness ./internal/platformreceipts ./internal/runs ./internal/restores ./handlers/destinations
)
(
  cd "$dbm_root/scripts"
  python3 -m unittest -v test_generate_support_matrix.py
)
python3 "$dbm_root/scripts/write_db6_fixture_receipts.py"
python3 "$dbm_root/scripts/generate_support_matrix.py"
python3 "$dbm_root/scripts/generate_support_matrix.py" --check
cat "$matrix_output" | sed -n '/BEGIN GENERATED PLATFORM MATRIX/,/END GENERATED PLATFORM MATRIX/p'

printf 'DB6 fault-injection owner checks and receipt-only matrix regeneration passed.\n'
