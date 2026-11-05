#!/usr/bin/env bash
# feed-snapshot.sh — semantic parity harness for the next-action feed.
#
# A next-action projection is operator-safety-critical: a wrong action
# misdirects the human loop. Performance work on the projection must therefore
# prove it changed cost, not meaning. This script captures the feed's decision
# fields — ranking order, action id, enabled flag, reason, blockers, target —
# and diffs a fresh capture against the committed golden file.
#
#   scripts/feed-snapshot.sh capture   # (re)write test/perf/feed-golden.json
#   scripts/feed-snapshot.sh diff      # non-zero exit when semantics changed
#
# created_at is deliberately dropped: it is raw item data passed through the
# projection, so including it would turn unrelated backlog edits into false
# parity failures. Its only projection role — ranking tiebreak — is already
# captured by the preserved entry order.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(dirname "$SCRIPT_DIR")"
SCENARIO_NAME="swarm-manager"
GOLDEN_PATH="${SCENARIO_DIR}/test/perf/feed-golden.json"

MODE="${1:-diff}"
case "$MODE" in
  capture|diff) ;;
  -h|--help)
    sed -n '2,18p' "$0" | sed 's/^# \{0,1\}//'
    exit 0
    ;;
  *)
    echo "feed-snapshot: unknown mode: $MODE (expected capture or diff)" >&2
    exit 2
    ;;
esac

API_PORT="${SWARM_MANAGER_API_PORT:-}"
if [[ -z "$API_PORT" ]]; then
  API_PORT="$(vrooli scenario status "$SCENARIO_NAME" --json 2>/dev/null | jq -r '.scenario.ports.API_PORT // empty')"
fi
if [[ -z "$API_PORT" || "$API_PORT" == "null" ]]; then
  echo "feed-snapshot: cannot resolve API port — start the scenario with 'vrooli scenario start $SCENARIO_NAME'" >&2
  exit 1
fi

# The projection fields whose meaning this plan must preserve.
NORMALIZE='{
  entries: [
    .entries[] | {
      entity_kind,
      entity_ref,
      entity_title,
      tier,
      goal_priority,
      backlog_rank,
      chained_ref,
      action: {
        id: .action.id,
        compact_label: .action.compact_label,
        expanded_label: .action.expanded_label,
        enabled: .action.enabled,
        reason: .action.reason,
        target: .action.target,
        blockers: (.action.blockers // [] | map({code, message, forceable}))
      }
    }
  ]
}'

capture_to() {
  curl -fsS "http://localhost:${API_PORT}/api/v1/next-actions/feed" \
    | jq -S "$NORMALIZE" >"$1"
}

if [[ "$MODE" == "capture" ]]; then
  mkdir -p "$(dirname "$GOLDEN_PATH")"
  capture_to "$GOLDEN_PATH"
  echo "feed-snapshot: captured $(jq '.entries | length' "$GOLDEN_PATH") entries to ${GOLDEN_PATH#"$SCENARIO_DIR"/}"
  exit 0
fi

if [[ ! -f "$GOLDEN_PATH" ]]; then
  echo "feed-snapshot: no golden snapshot at ${GOLDEN_PATH} — run 'scripts/feed-snapshot.sh capture' first" >&2
  exit 1
fi

CURRENT="$(mktemp -t feed-snapshot.XXXXXX.json)"
trap 'rm -f "$CURRENT"' EXIT
capture_to "$CURRENT"

if diff -u "$GOLDEN_PATH" "$CURRENT" >/dev/null; then
  echo "feed-snapshot: parity holds ($(jq '.entries | length' "$CURRENT") entries)"
  exit 0
fi

echo "feed-snapshot: SEMANTIC DIFF against golden snapshot" >&2
diff -u "$GOLDEN_PATH" "$CURRENT" >&2 || true
exit 1
