#!/usr/bin/env bash
# Deletes the web-console test-fixture session folders listed in p2-fixture-dirs.txt (P2-D3).
# Each folder is deleted only if ALL checks pass:
#   - it is directly under ~/.vrooli/state/vrooli/web-console/sessions/codex/
#   - it contains exactly one regular file (symlinks are ignored and never followed)
#   - that file is named rollout-{test123,e2e,stale,resume,backfill,mid-session-test}.jsonl
#   - that file has no "session_meta" line (every real Codex rollout starts with one)
# Anything else is skipped and reported. Dry run by default; pass --delete to delete.
set -euo pipefail

base="$HOME/.vrooli/state/vrooli/web-console/sessions/codex"
list="$(dirname "$0")/p2-fixture-dirs.txt"
mode="${1:-dry-run}"
fixture='^rollout-(test123|e2e|stale|resume|backfill|mid-session-test)\.jsonl$'
ok=0 skipped=0

while IFS=$'\t' read -r dir _; do
  dir="${dir%"${dir##*[![:space:]]}"}"
  if [[ "$(dirname "$dir")" != "$base" || ! -d "$dir" || -L "$dir" ]]; then
    echo "SKIP (not a fixture dir under $base): $dir"; skipped=$((skipped + 1)); continue
  fi
  mapfile -t files < <(find "$dir" -type f)
  if [[ ${#files[@]} -ne 1 ]] || ! [[ "$(basename "${files[0]}")" =~ $fixture ]] || grep -q session_meta "${files[0]}"; then
    echo "SKIP (contents are not a single fixture rollout): $dir"; skipped=$((skipped + 1)); continue
  fi
  if [[ "$mode" == "--delete" ]]; then
    rm -r -- "$dir"
  fi
  ok=$((ok + 1))
done < "$list"

if [[ "$mode" == "--delete" ]]; then
  echo "deleted $ok fixture folders; skipped $skipped"
else
  echo "dry run: $ok fixture folders would be deleted; skipped $skipped (run with --delete to delete)"
fi
