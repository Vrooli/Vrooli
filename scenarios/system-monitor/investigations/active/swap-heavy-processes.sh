#!/bin/bash
# INVESTIGATION_SCRIPT
# NAME: Swap Heavy Process Detector
# DESCRIPTION: Identifies processes consuming excessive swap memory and provides recommendations
# CATEGORY: resource-management
# TRIGGERS: high_swap_usage, swap_thrashing, memory_pressure
# OUTPUTS: json
# AUTHOR: claude-agent
# CREATED: 2025-09-21
# LAST_MODIFIED: 2025-09-21
# VERSION: 1.0

set -euo pipefail

# Get total swap usage
SWAP_TOTAL=$(free -b | grep Swap | awk '{print $2}')
SWAP_USED=$(free -b | grep Swap | awk '{print $3}')
SWAP_PERCENT=$(echo "scale=2; $SWAP_USED * 100 / $SWAP_TOTAL" | bc)

# Find processes using swap (using VmSwap from /proc/*/status)
SWAP_PROCESSES=()
for PID_DIR in /proc/[0-9]*; do
    PID=$(basename "$PID_DIR")
    if [[ -r "$PID_DIR/status" ]]; then
        SWAP_KB=$(grep -E "^VmSwap:" "$PID_DIR/status" 2>/dev/null | awk '{print $2}' || echo "0")
        if [[ "$SWAP_KB" -gt 1024 ]]; then  # Only track processes using > 1MB swap
            CMDLINE=$(tr '\0' ' ' < "$PID_DIR/cmdline" 2>/dev/null | head -c 100 || echo "unknown")
            NAME=$(grep -E "^Name:" "$PID_DIR/status" 2>/dev/null | awk '{print $2}' || echo "unknown")
            RSS_KB=$(grep -E "^VmRSS:" "$PID_DIR/status" 2>/dev/null | awk '{print $2}' || echo "0")
            
            SWAP_PROCESSES+=("{\"pid\":$PID,\"name\":\"$NAME\",\"swap_mb\":$(($SWAP_KB / 1024)),\"rss_mb\":$(($RSS_KB / 1024)),\"cmdline\":\"$CMDLINE\"}")
        fi
    fi
done

# Sort by swap usage and get top 10
TOP_SWAP_JSON=""
if [[ ${#SWAP_PROCESSES[@]} -gt 0 ]]; then
    TOP_SWAP_JSON=$(printf '%s\n' "${SWAP_PROCESSES[@]}" | \
                    jq -s 'sort_by(.swap_mb) | reverse | .[0:10]')
else
    TOP_SWAP_JSON="[]"
fi

# Check swap pressure indicators
SWAP_IN=$(vmstat 1 2 | tail -1 | awk '{print $7}')
SWAP_OUT=$(vmstat 1 2 | tail -1 | awk '{print $8}')

# Determine risk level and recommendations
RISK_LEVEL="low"
RECOMMENDATIONS=()

if (( $(echo "$SWAP_PERCENT > 70" | bc -l) )); then
    RISK_LEVEL="critical"
    RECOMMENDATIONS+=("\"System using $(printf '%.1f' $SWAP_PERCENT)% of swap - performance severely degraded\"")
    RECOMMENDATIONS+=("\"Consider killing non-essential processes or adding more RAM\"")
elif (( $(echo "$SWAP_PERCENT > 50" | bc -l) )); then
    RISK_LEVEL="high"
    RECOMMENDATIONS+=("\"Swap usage at $(printf '%.1f' $SWAP_PERCENT)% - monitor closely\"")
fi

if [[ "$SWAP_IN" -gt 100 || "$SWAP_OUT" -gt 100 ]]; then
    RECOMMENDATIONS+=("\"Active swap thrashing detected (in: $SWAP_IN, out: $SWAP_OUT pages/sec)\"")
    RECOMMENDATIONS+=("\"System is actively swapping - severe performance impact\"")
    RISK_LEVEL="critical"
fi

# Check Chrome memory usage specifically
CHROME_SWAP=$(echo "$TOP_SWAP_JSON" | jq '[.[] | select(.name == "chrome")] | map(.swap_mb) | add // 0')
if [[ "$CHROME_SWAP" -gt 5000 ]]; then
    RECOMMENDATIONS+=("\"Chrome processes using ${CHROME_SWAP}MB of swap\"")
    RECOMMENDATIONS+=("\"Consider closing unnecessary browser tabs or restarting Chrome\"")
fi

# Output findings
jq -n \
  --argjson processes "$TOP_SWAP_JSON" \
  --argjson swap_total "$SWAP_TOTAL" \
  --argjson swap_used "$SWAP_USED" \
  --argjson swap_percent "$SWAP_PERCENT" \
  --arg risk_level "$RISK_LEVEL" \
  --argjson recommendations "[$(IFS=,; echo "${RECOMMENDATIONS[*]:-}")]" \
  '{
    investigation: "swap-heavy-processes",
    timestamp: (now | todateiso8601),
    findings: {
      swap_usage: {
        total_gb: ($swap_total / 1073741824),
        used_gb: ($swap_used / 1073741824),
        percent: $swap_percent
      },
      top_swap_consumers: $processes,
      swap_activity: {
        pages_in_per_sec: '$SWAP_IN',
        pages_out_per_sec: '$SWAP_OUT'
      }
    },
    risk_level: $risk_level,
    recommendations: $recommendations
  }'