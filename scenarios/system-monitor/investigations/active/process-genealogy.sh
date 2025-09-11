#!/bin/bash
# INVESTIGATION_SCRIPT
# NAME: Process Genealogy Tracer
# DESCRIPTION: Traces parent-child relationships to find process spawn sources
# CATEGORY: process-analysis
# TRIGGERS: excessive_processes, memory_pressure, fork_bomb_suspected
# OUTPUTS: json
# AUTHOR: claude-agent
# CREATED: 2025-09-11
# LAST_MODIFIED: 2025-09-11
# VERSION: 1.0

set -euo pipefail

SCRIPT_NAME="process-genealogy"
OUTPUT_DIR="../results/$(date +%Y%m%d_%H%M%S)_${SCRIPT_NAME}"
TIMEOUT_SECONDS=30

mkdir -p "${OUTPUT_DIR}"
RESULTS_FILE="${OUTPUT_DIR}/results.json"

cat > "${RESULTS_FILE}" << 'EOF'
{
  "investigation": "process-genealogy",
  "timestamp": "",
  "process_families": [],
  "suspicious_patterns": [],
  "top_spawners": [],
  "recommendations": []
}
EOF

jq ".timestamp = \"$(date -Iseconds)\"" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

echo "🧬 Starting Process Genealogy Analysis..."

# Find processes with many children
echo "👨‍👩‍👧‍👦 Finding prolific parent processes..."
PARENT_COUNTS=$(timeout ${TIMEOUT_SECONDS} ps -eo ppid | sort | uniq -c | sort -rn | head -20 | awk 'NR>1 && $1>5 {printf "{\"ppid\":%s,\"child_count\":%d},", $2, $1}' | sed 's/,$//')

if [[ -n "${PARENT_COUNTS}" ]]; then
  jq ".top_spawners = [${PARENT_COUNTS}]" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
fi

# Analyze process families for specific patterns
echo "🔍 Analyzing process families..."
FAMILIES=""
for ppid in $(echo "${PARENT_COUNTS}" | jq -r '.[].ppid' 2>/dev/null || echo ""); do
  if [[ -n "${ppid}" && "${ppid}" =~ ^[0-9]+$ && "${ppid}" != "0" ]]; then
    # Get parent process info
    PARENT_INFO=$(timeout 5 ps -p "${ppid}" -o pid,comm,args --no-headers 2>/dev/null | head -1 || echo "")
    if [[ -n "${PARENT_INFO}" ]]; then
      PARENT_CMD=$(echo "${PARENT_INFO}" | awk '{print $2}')
      PARENT_ARGS=$(echo "${PARENT_INFO}" | cut -d' ' -f3- | sed 's/"/\\"/g')
      
      # Get child processes
      CHILDREN=$(timeout 10 ps --ppid "${ppid}" -o pid,comm --no-headers 2>/dev/null | wc -l || echo "0")
      CHILD_COMMANDS=$(timeout 10 ps --ppid "${ppid}" -o comm --no-headers 2>/dev/null | sort | uniq -c | sort -rn | head -5 | awk '{printf "\"%s\":%d,", $2, $1}' | sed 's/,$//' || echo "")
      
      FAMILY_JSON="{\"ppid\":${ppid},\"parent_command\":\"${PARENT_CMD}\",\"parent_args\":\"${PARENT_ARGS}\",\"child_count\":${CHILDREN},\"child_types\":{${CHILD_COMMANDS}}}"
      FAMILIES="${FAMILIES}${FAMILY_JSON},"
    fi
  fi
done

FAMILIES=$(echo "${FAMILIES}" | sed 's/,$//')
if [[ -n "${FAMILIES}" ]]; then
  jq ".process_families = [${FAMILIES}]" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
fi

# Detect suspicious patterns
echo "🚨 Detecting suspicious patterns..."
SUSPICIOUS=""

# Check for potential fork bombs (processes spawning many identical children)
IDENTICAL_CHILDREN=$(timeout 15 ps -eo ppid,comm | awk '{print $1 ":" $2}' | sort | uniq -c | sort -rn | awk '$1 > 50 {printf "{\"pattern\":\"identical_children\",\"ppid\":\"%s\",\"command\":\"%s\",\"count\":%d},", substr($2,1,index($2,":")-1), substr($2,index($2,":")+1), $1}' | sed 's/,$//')

# Check for excessive lsof/find/grep processes
SYSTEM_TOOL_ABUSE=$(ps aux | awk '$11 ~ /^(lsof|find|grep|awk)$/ {print $11}' | sort | uniq -c | sort -rn | awk '$1 > 10 {printf "{\"pattern\":\"tool_abuse\",\"command\":\"%s\",\"count\":%d},", $2, $1}' | sed 's/,$//')

# Check for containers spawning excessive workers
CONTAINER_WORKERS=$(timeout 10 ps aux | grep -E "(rake|worker|celery)" | wc -l || echo "0")
if [[ ${CONTAINER_WORKERS} -gt 50 ]]; then
  SUSPICIOUS="${SUSPICIOUS}{\"pattern\":\"excessive_workers\",\"count\":${CONTAINER_WORKERS}},"
fi

# Combine suspicious patterns
ALL_SUSPICIOUS="${IDENTICAL_CHILDREN}"
if [[ -n "${SYSTEM_TOOL_ABUSE}" ]]; then
  ALL_SUSPICIOUS="${ALL_SUSPICIOUS},${SYSTEM_TOOL_ABUSE}"
fi
ALL_SUSPICIOUS="${ALL_SUSPICIOUS}${SUSPICIOUS}"
ALL_SUSPICIOUS=$(echo "${ALL_SUSPICIOUS}" | sed 's/^,//' | sed 's/,$//')

if [[ -n "${ALL_SUSPICIOUS}" ]]; then
  jq ".suspicious_patterns = [${ALL_SUSPICIOUS}]" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
fi

# Generate recommendations
echo "💡 Generating recommendations..."
RECOMMENDATIONS=""

# Check for systemd/init as top spawner (normal)
TOP_SPAWNER_PID=$(echo "${PARENT_COUNTS}" | jq -r '.[0].ppid' 2>/dev/null || echo "")
if [[ -n "${TOP_SPAWNER_PID}" && "${TOP_SPAWNER_PID}" != "1" ]]; then
  TOP_SPAWNER_CMD=$(ps -p "${TOP_SPAWNER_PID}" -o comm --no-headers 2>/dev/null || echo "unknown")
  RECOMMENDATIONS="${RECOMMENDATIONS}\"Investigate process ${TOP_SPAWNER_PID} (${TOP_SPAWNER_CMD}) - unusual top spawner\","
fi

# Recommendations based on suspicious patterns
if [[ -n "${ALL_SUSPICIOUS}" ]]; then
  # Count different pattern types
  PATTERN_COUNT=$(echo "${ALL_SUSPICIOUS}" | jq -s 'length' 2>/dev/null || echo "0")
  if [[ ${PATTERN_COUNT} -gt 0 ]]; then
    RECOMMENDATIONS="${RECOMMENDATIONS}\"Found ${PATTERN_COUNT} suspicious process patterns - review and potentially limit process spawning\","
  fi
fi

# Container worker recommendations
if [[ ${CONTAINER_WORKERS} -gt 50 ]]; then
  RECOMMENDATIONS="${RECOMMENDATIONS}\"Excessive worker processes (${CONTAINER_WORKERS}) - check container memory limits and worker configuration\","
fi

RECOMMENDATIONS=$(echo "${RECOMMENDATIONS}" | sed 's/,$//')
if [[ -n "${RECOMMENDATIONS}" ]]; then
  jq ".recommendations = [${RECOMMENDATIONS}]" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
fi

echo "✅ Process genealogy analysis complete! Results: ${RESULTS_FILE}"
cat "${RESULTS_FILE}"