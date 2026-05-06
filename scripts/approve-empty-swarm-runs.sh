#!/usr/bin/env bash
# approve-empty-swarm-runs.sh
#
# Drain the swarm-manager NEEDS_REVIEW backlog that accumulated before the
# 2026-04-24 auto-approve fix.
#
# For every run whose tag starts with "swarm-manager:" and whose status is
# NEEDS_REVIEW, fetch the live sandbox diff via the workspace-sandbox HTTP API
# (single curl; no logic is inferred from it beyond "diff is empty"). If the
# diff has zero files changed, approve the run via the agent-manager CLI.
# Runs with real diffs are left for human review.
#
# Requires: agent-manager CLI, workspace-sandbox running on its service port.
#
# Usage:
#   scripts/approve-empty-swarm-runs.sh            # dry-run by default
#   APPROVE=1 scripts/approve-empty-swarm-runs.sh  # actually approve
#
# Remove this script after the one-time backlog drain is complete.
set -euo pipefail

APPROVE=${APPROVE:-0}
SANDBOX_URL=${SANDBOX_URL:-http://localhost:15120}

echo "Fetching NEEDS_REVIEW runs with swarm-manager: tag prefix (paginated, 100/page)..."
offset=0
run_ids=""
while :; do
    page=$(agent-manager run list \
        --status needs_review \
        --tag-prefix swarm-manager: \
        --limit 100 \
        --offset "$offset" \
        --json)
    page_ids=$(echo "$page" | python3 -c "
import sys, json
for r in json.load(sys.stdin).get('runs', []):
    print(r['id'])
")
    if [[ -z "$page_ids" ]]; then
        break
    fi
    run_ids="${run_ids}${page_ids}"$'\n'
    page_n=$(echo "$page_ids" | grep -c .)
    offset=$((offset + page_n))
    if [[ "$page_n" -lt 100 ]]; then
        break
    fi
done

run_count=$(echo -n "$run_ids" | grep -c . || true)
echo "Found $run_count candidate runs"

echo "Building run_id -> sandbox_id map from workspace-sandbox..."
sandbox_map=$(curl -sf "${SANDBOX_URL}/api/v1/sandboxes?limit=1000" | python3 -c "
import sys, json
for s in json.load(sys.stdin).get('sandboxes', []):
    if s.get('ownerType') == 'run' and s.get('owner'):
        print(s['owner'] + '\t' + s['id'])
")

# Build joined pairs: run_id <tab> sandbox_id (or empty if no sandbox)
all_pairs=$(python3 <<PY
import sys
sbx_map = {}
for line in """$sandbox_map""".strip().splitlines():
    if '\t' in line:
        owner, sid = line.split('\t', 1)
        sbx_map[owner] = sid
for rid in """$run_ids""".strip().splitlines():
    print(rid + '\t' + sbx_map.get(rid, ''))
PY
)

if [[ "$run_count" == 0 ]]; then
    echo "Nothing to do."
    exit 0
fi

approved=0
skipped_with_diff=0
skipped_no_sandbox=0
errored=0

while IFS=$'\t' read -r run_id sandbox_id; do
    if [[ -z "$run_id" ]]; then
        continue
    fi
    if [[ -z "$sandbox_id" || "$sandbox_id" == "null" ]]; then
        skipped_no_sandbox=$((skipped_no_sandbox + 1))
        continue
    fi
    files_changed=$(curl -sf "${SANDBOX_URL}/api/v1/sandboxes/${sandbox_id}/diff" \
        | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('stats',{}).get('filesChanged',0))" 2>/dev/null || echo "err")
    if [[ "$files_changed" == "err" ]]; then
        errored=$((errored + 1))
        echo "  [$run_id] sandbox=$sandbox_id — could not reach diff endpoint, skipping"
        continue
    fi
    if [[ "$files_changed" == "0" ]]; then
        if [[ "$APPROVE" == "1" ]]; then
            if agent-manager run approve "$run_id" >/dev/null 2>&1; then
                echo "  [$run_id] empty sandbox — APPROVED"
                approved=$((approved + 1))
            else
                echo "  [$run_id] empty sandbox — approve CLI failed"
                errored=$((errored + 1))
            fi
        else
            echo "  [$run_id] empty sandbox — would approve (dry-run)"
            approved=$((approved + 1))
        fi
    else
        skipped_with_diff=$((skipped_with_diff + 1))
        echo "  [$run_id] $files_changed files changed — leaving for human review"
    fi
done < <(echo -n "$all_pairs")

echo ""
echo "Summary:"
echo "  approved:            $approved"
echo "  left for review:     $skipped_with_diff"
echo "  no sandbox attached: $skipped_no_sandbox"
echo "  errored:             $errored"

if [[ "$APPROVE" != "1" ]]; then
    echo ""
    echo "(dry-run) re-run with APPROVE=1 to actually approve."
fi
