#!/usr/bin/env bash
# cleanup-duplicate-schedules.sh
# Remove duplicate sample schedules created by re-seeding
#
# Usage: ./scripts/cleanup-duplicate-schedules.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
API_PORT="${API_PORT:-18446}"
API_URL="http://localhost:${API_PORT}"

echo "🔧 Cleaning up duplicate sample schedules..."

# Get all schedules
SCHEDULES=$(curl -sf "${API_URL}/api/schedules" || { echo "❌ Failed to fetch schedules"; exit 1; })

# Find duplicates (schedules with same name and owner)
echo "$SCHEDULES" | jq -r '
    group_by(.name + "|" + .owner) |
    map(select(length > 1)) |
    .[] |
    .[1:] |
    .[] |
    .id
' | while read -r schedule_id; do
    if [ -n "$schedule_id" ]; then
        echo "  Deleting duplicate: $schedule_id"
        curl -sf -X DELETE "${API_URL}/api/schedules/${schedule_id}" &>/dev/null || true
    fi
done

# Show final count
FINAL_COUNT=$(curl -sf "${API_URL}/api/schedules" | jq 'length')
echo "✅ Cleanup complete. Remaining schedules: ${FINAL_COUNT}"

# Show unique schedule names
echo ""
echo "Unique schedules:"
curl -sf "${API_URL}/api/schedules" | jq -r 'group_by(.name) | map({name: .[0].name, count: length, owner: .[0].owner}) | .[] | "  - \(.name) (\(.owner))"'
