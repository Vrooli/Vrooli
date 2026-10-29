#!/usr/bin/env bash
# Deletes the Agent Manager runs imported from web-console test fixtures (P2-D3), listed in
# p2-fixture-runs.txt as "run_id|transcript_path". A run is deleted only if its transcript path is
# under web-console's codex sessions dir and is named rollout-{test123,e2e,stale,resume,backfill,
# mid-session-test}.jsonl. Uses the governed `agent-manager run delete` (operator context required).
# Dry run by default; pass --delete to delete. Run p2-delete-fixture-dirs.sh first or after; the
# indexer skips these files either way.
set -euo pipefail

list="$(dirname "$0")/p2-fixture-runs.txt"
mode="${1:-dry-run}"
fixture="^$HOME/\.vrooli/state/vrooli/web-console/sessions/codex/[^/]+/.*/rollout-(test123|e2e|stale|resume|backfill|mid-session-test)\.jsonl$"
ok=0 skipped=0 failed=0

while IFS='|' read -r id path; do
  if ! [[ "$id" =~ ^[0-9a-f-]{36}$ && "$path" =~ $fixture ]]; then
    echo "SKIP (not a fixture run): $id $path"; skipped=$((skipped + 1)); continue
  fi
  if [[ "$mode" == "--delete" ]]; then
    if agent-manager run delete "$id" >/dev/null 2>&1; then ok=$((ok + 1)); else echo "FAILED: $id"; failed=$((failed + 1)); fi
  else
    ok=$((ok + 1))
  fi
done < "$list"

if [[ "$mode" == "--delete" ]]; then
  echo "deleted $ok fixture runs; skipped $skipped; failed $failed"
else
  echo "dry run: $ok fixture runs would be deleted; skipped $skipped (run with --delete to delete)"
fi
