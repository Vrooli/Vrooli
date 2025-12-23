#!/bin/bash
# INVESTIGATION_SCRIPT
# NAME: Zombie Process Detector
# DESCRIPTION: Detects and analyzes zombie (defunct) processes to identify parent process issues
# CATEGORY: process-analysis
# TRIGGERS: zombie_processes_found, process_cleanup_issues, high_process_count
# OUTPUTS: json
# AUTHOR: claude-agent
# CREATED: 2025-09-11
# LAST_MODIFIED: 2025-12-22
# VERSION: 1.1

set -euo pipefail

# Configuration
SCRIPT_NAME="zombie-process-detector"
OUTPUT_DIR="../results/$(date +%Y%m%d_%H%M%S)_${SCRIPT_NAME}"
TIMEOUT_SECONDS=30

# Create output directory
mkdir -p "${OUTPUT_DIR}"

# Initialize results
RESULTS_FILE="${OUTPUT_DIR}/results.json"
cat > "${RESULTS_FILE}" << 'EOF'
{
  "investigation": "zombie-process-detector",
  "timestamp": "",
  "zombie_summary": {},
  "zombie_processes": [],
  "parent_analysis": [],
  "patterns": {},
  "recommendations": []
}
EOF

# Update timestamp
jq ".timestamp = \"$(date -Iseconds)\"" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

echo "🧟 Starting Zombie Process Detection..."

# Count and identify zombie processes
echo "💀 Searching for zombie processes..."
ZOMBIE_COUNT=$(ps -eo stat= | awk '$1 ~ /Z/ {count++} END {print count+0}')
ZOMBIE_LIST=$(ps -eo pid=,ppid=,user=,stat=,comm= | awk '$4 ~ /Z/ {printf "{\"pid\":%s,\"ppid\":%s,\"user\":\"%s\",\"command\":\"%s\"},", $1, $2, $3, $5}' | sed 's/,$//')

# Get detailed zombie info with parent PIDs
ZOMBIE_DETAILS=""
if [[ ${ZOMBIE_COUNT} -gt 0 ]]; then
  echo "🔍 Analyzing zombie parent processes..."
  for pid in $(ps -eo pid=,stat= | awk '$2 ~ /Z/ {print $1}'); do
    if [[ -e /proc/${pid}/stat ]]; then
      PARENT_PID=$(awk '{print $4}' /proc/${pid}/stat 2>/dev/null || echo "unknown")
      PARENT_CMD=$(ps -p ${PARENT_PID} -o comm= 2>/dev/null || echo "unknown")
      ZOMBIE_DETAILS="${ZOMBIE_DETAILS}{\"zombie_pid\":${pid},\"parent_pid\":${PARENT_PID},\"parent_cmd\":\"${PARENT_CMD}\"},"
    fi
  done
  ZOMBIE_DETAILS=$(echo "${ZOMBIE_DETAILS}" | sed 's/,$//')
fi

# Update zombie summary
jq ".zombie_summary = {
  \"total_zombies\": ${ZOMBIE_COUNT},
  \"detection_time\": \"$(date -Iseconds)\"
}" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

# Add zombie list if any found
if [[ -n "${ZOMBIE_LIST}" ]]; then
  jq ".zombie_processes = [${ZOMBIE_LIST}]" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
fi

# Add parent analysis if available
if [[ -n "${ZOMBIE_DETAILS}" ]]; then
  jq ".parent_analysis = [${ZOMBIE_DETAILS}]" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
fi

# Pattern Analysis
echo "🔍 Analyzing patterns..."
PATTERN_DATA="{"

# Check for common zombie patterns
PYTHON_ZOMBIES=$(ps -eo stat=,comm= | awk '$1 ~ /Z/ && tolower($2) ~ /python/ {count++} END {print count+0}')
SHELL_ZOMBIES=$(ps -eo stat=,comm= | awk '$1 ~ /Z/ && tolower($2) ~ /(sh|bash)/ {count++} END {print count+0}')
NODE_ZOMBIES=$(ps -eo stat=,comm= | awk '$1 ~ /Z/ && tolower($2) ~ /node/ {count++} END {print count+0}')

PATTERN_DATA="${PATTERN_DATA}\"python_zombies\":${PYTHON_ZOMBIES},"
PATTERN_DATA="${PATTERN_DATA}\"shell_zombies\":${SHELL_ZOMBIES},"
PATTERN_DATA="${PATTERN_DATA}\"node_zombies\":${NODE_ZOMBIES}"
PATTERN_DATA="${PATTERN_DATA}}"

jq ".patterns = ${PATTERN_DATA}" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

# Generate Recommendations
echo "💡 Generating recommendations..."
RECOMMENDATIONS=""

if [[ ${ZOMBIE_COUNT} -gt 0 ]]; then
  RECOMMENDATIONS="${RECOMMENDATIONS}\"Found ${ZOMBIE_COUNT} zombie processes - parent processes not properly reaping children\","
fi

if [[ ${ZOMBIE_COUNT} -gt 10 ]]; then
  RECOMMENDATIONS="${RECOMMENDATIONS}\"High number of zombies indicates systematic parent process issue\","
  RECOMMENDATIONS="${RECOMMENDATIONS}\"Check parent process signal handling and wait() calls\","
fi

if [[ ${PYTHON_ZOMBIES} -gt 5 ]]; then
  RECOMMENDATIONS="${RECOMMENDATIONS}\"Multiple Python zombies detected - review Python subprocess management\","
fi

# Check for init as parent (orphaned zombies)
INIT_ZOMBIES=$(ps -eo ppid=,stat= | awk '$2 ~ /Z/ && $1 == 1 {count++} END {print count+0}')
if [[ ${INIT_ZOMBIES} -gt 0 ]]; then
  RECOMMENDATIONS="${RECOMMENDATIONS}\"${INIT_ZOMBIES} zombies orphaned to init - original parent crashed\","
fi

# Add general recommendations
if [[ ${ZOMBIE_COUNT} -gt 0 ]]; then
  RECOMMENDATIONS="${RECOMMENDATIONS}\"Review parent process code for proper child reaping\","
  RECOMMENDATIONS="${RECOMMENDATIONS}\"Consider implementing SIGCHLD signal handler\","
  RECOMMENDATIONS="${RECOMMENDATIONS}\"Use wait() or waitpid() to collect child exit status\""
fi

RECOMMENDATIONS=$(echo "${RECOMMENDATIONS}" | sed 's/,$//')
if [[ -n "${RECOMMENDATIONS}" ]]; then
  jq ".recommendations = [${RECOMMENDATIONS}]" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
fi

echo "✅ Zombie process detection complete! Results: ${RESULTS_FILE}"

# Output final results to stdout for API consumption
cat "${RESULTS_FILE}"
