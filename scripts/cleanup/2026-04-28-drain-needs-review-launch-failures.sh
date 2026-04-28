#!/usr/bin/env bash
# 2026-04-28-drain-needs-review-launch-failures.sh
#
# Drain the silent-launch-failure rows that accumulated in
# RUN_STATUS_NEEDS_REVIEW between the protected-sandbox cutover (commit
# 3e8b004704, Sandboxing auto-approval p1) and the four-phase fix in
# docs/plans/sandbox-launch-and-auto-approve-fixes-plan.md. For each
# such run, the runner never produced output: bwrap chdir-failed before
# claude launched, the SSE exit event was raced away, and the run
# landed at "complete + manual review". Per the operator decision on
# 2026-04-28, these are REJECTED (not deleted) so the audit trail
# preserves an explicit rationale.
#
# Heuristic for "silent launch failure":
#   - status == NEEDS_REVIEW
#   - 0 RUN_EVENT_TYPE_MESSAGE events
#   - duration < 2s
#
# Usage:
#   scripts/cleanup/2026-04-28-drain-needs-review-launch-failures.sh             # dry-run (default)
#   REJECT=1 scripts/cleanup/2026-04-28-drain-needs-review-launch-failures.sh    # actually reject
#
# This script is one-shot. Delete it in a separate commit after the
# drain completes — see Definition of Done in the plan.

set -euo pipefail

REJECT=${REJECT:-0}
TAG_PREFIX=${TAG_PREFIX:-swarm-manager:}
RATIONALE="silent launch failure pre-fix; see docs/plans/sandbox-launch-and-auto-approve-fixes-plan.md"

# duration_ms below which a successful protected run is considered a
# silent launch failure. Aligns with launchFailedMaxDuration in
# scenarios/agent-manager/api/internal/orchestration/run_executor.go.
DURATION_MS_MAX=${DURATION_MS_MAX:-2000}

echo "Mode: $([[ $REJECT == 1 ]] && echo REJECT || echo DRY-RUN)"
echo "Tag prefix: $TAG_PREFIX"
echo "Max duration treated as launch-failure: ${DURATION_MS_MAX}ms"
echo

offset=0
candidates=()
while :; do
	page=$(agent-manager run list \
		--status needs_review \
		--tag-prefix "$TAG_PREFIX" \
		--limit 100 \
		--offset "$offset" \
		--json)
	page_ids=$(echo "$page" | python3 -c "
import sys, json
for r in json.load(sys.stdin).get('runs', []):
    print(r['id'])
")
	[[ -z "$page_ids" ]] && break
	while IFS= read -r run_id; do
		[[ -z "$run_id" ]] && continue

		# Pull events JSON to count messages and read timing.
		ev=$(agent-manager run events "$run_id" --json 2>/dev/null || echo '{"events":[]}')
		summary=$(echo "$ev" | python3 -c "
import sys, json, datetime as dt
e = json.load(sys.stdin).get('events', [])
msgs = sum(1 for x in e if x.get('eventType') == 'message')
ts = [x.get('timestamp') or x.get('createdAt') for x in e if x.get('timestamp') or x.get('createdAt')]
def parse(s):
    try: return dt.datetime.fromisoformat(s.replace('Z','+00:00'))
    except: return None
parsed = [t for t in (parse(t) for t in ts) if t]
duration_ms = 0
if len(parsed) >= 2:
    duration_ms = int((max(parsed) - min(parsed)).total_seconds() * 1000)
print(f'{msgs} {duration_ms}')
")
		read -r msgs duration_ms <<<"$summary"
		if [[ "$msgs" == "0" && "$duration_ms" -lt "$DURATION_MS_MAX" ]]; then
			candidates+=("$run_id")
			echo "  candidate: $run_id (msgs=$msgs, duration=${duration_ms}ms)"
		fi
	done <<<"$page_ids"
	offset=$((offset + 100))
done

echo
echo "Found ${#candidates[@]} silent-launch-failure candidate(s)."
if [[ ${#candidates[@]} -eq 0 ]]; then
	exit 0
fi

if [[ "$REJECT" != 1 ]]; then
	echo "Dry-run only. Re-run with REJECT=1 to mark these runs as rejected."
	exit 0
fi

echo "Rejecting ${#candidates[@]} run(s)…"
for run_id in "${candidates[@]}"; do
	if agent-manager run reject "$run_id" --reason "$RATIONALE"; then
		echo "  rejected: $run_id"
	else
		echo "  FAILED to reject: $run_id" >&2
	fi
done
