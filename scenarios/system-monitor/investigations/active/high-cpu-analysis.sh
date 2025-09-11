#!/bin/bash
# INVESTIGATION_SCRIPT
# NAME: High CPU Analysis
# DESCRIPTION: Identifies processes consuming excessive CPU and traces their origins
# CATEGORY: performance
# TRIGGERS: cpu_usage > 80%, load_average > 4.0
# OUTPUTS: json
# AUTHOR: claude-agent
# CREATED: 2025-09-11
# LAST_MODIFIED: 2025-09-11
# VERSION: 1.0

set -euo pipefail

# Configuration
SCRIPT_NAME="high-cpu-analysis"
OUTPUT_DIR="../results/$(date +%Y%m%d_%H%M%S)_${SCRIPT_NAME}"
TIMEOUT_SECONDS=60
CPU_THRESHOLD=10.0

# Create output directory
mkdir -p "${OUTPUT_DIR}"

# Initialize results
RESULTS_FILE="${OUTPUT_DIR}/results.json"
cat > "${RESULTS_FILE}" << 'EOF'
{
  "investigation": "high-cpu-analysis",
  "timestamp": "",
  "system_overview": {},
  "high_cpu_processes": [],
  "process_trees": [],
  "patterns": {},
  "recommendations": [],
  "raw_data": {}
}
EOF

# Update timestamp
jq ".timestamp = \"$(date -Iseconds)\"" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

echo "🔍 Starting High CPU Analysis..."

# System Overview
echo "📊 Gathering system overview..."
LOAD_AVG=$(uptime | awk -F'load average:' '{print $2}' | tr -d ' ')
CPU_COUNT=$(nproc)
MEMORY_INFO=$(free -m | awk 'NR==2{printf "%.1f", $3/$2*100}')

jq ".system_overview = {
  \"load_average\": \"${LOAD_AVG}\",
  \"cpu_cores\": ${CPU_COUNT},
  \"memory_usage_percent\": ${MEMORY_INFO},
  \"analysis_time\": \"$(date -Iseconds)\"
}" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

# High CPU Processes
echo "🔥 Identifying high CPU processes..."
HIGH_CPU_PROCS=$(timeout ${TIMEOUT_SECONDS} top -b -n 1 | awk -v threshold="${CPU_THRESHOLD}" '
  NR>7 && $9+0 >= threshold { 
    printf "{\"pid\":%s,\"user\":\"%s\",\"cpu\":%.1f,\"mem\":%.1f,\"command\":\"%s\"},", 
    $1, $2, $9, $10, $12
  }' | sed 's/,$//')

if [[ -n "${HIGH_CPU_PROCS}" ]]; then
  jq ".high_cpu_processes = [${HIGH_CPU_PROCS}]" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
fi

# Process Trees for High CPU Processes
echo "🌳 Building process trees..."
TREE_DATA=""
if [[ -n "${HIGH_CPU_PROCS}" ]]; then
  for pid in $(echo "${HIGH_CPU_PROCS}" | jq -r '.[].pid' 2>/dev/null || echo ""); do
    if [[ -n "${pid}" && "${pid}" =~ ^[0-9]+$ ]]; then
      # Get process tree
      TREE_INFO=$(timeout 10 pstree -p "${pid}" 2>/dev/null | head -20 || echo "tree_unavailable")
      # Get parent chain
      PARENT_CHAIN=$(timeout 10 ps -o pid,ppid,comm --no-headers -p "${pid}" 2>/dev/null || echo "")
      
      if [[ -n "${TREE_INFO}" && "${TREE_INFO}" != "tree_unavailable" ]]; then
        TREE_JSON="{\"pid\":${pid},\"tree\":\"$(echo "${TREE_INFO}" | sed 's/"/\\"/g')\",\"parent_info\":\"$(echo "${PARENT_CHAIN}" | sed 's/"/\\"/g')\"}"
        TREE_DATA="${TREE_DATA}${TREE_JSON},"
      fi
    fi
  done
  TREE_DATA=$(echo "${TREE_DATA}" | sed 's/,$//')
  if [[ -n "${TREE_DATA}" ]]; then
    jq ".process_trees = [${TREE_DATA}]" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
  fi
fi

# Pattern Analysis
echo "🔍 Analyzing patterns..."
DUPLICATE_COMMANDS=$(timeout 20 ps aux | awk '{print $11}' | sort | uniq -c | sort -rn | head -10 | awk '$1 > 5 {printf "{\"command\":\"%s\",\"count\":%d},", $2, $1}' | sed 's/,$//')

PATTERN_DATA="{"
if [[ -n "${DUPLICATE_COMMANDS}" ]]; then
  PATTERN_DATA="${PATTERN_DATA}\"duplicate_commands\":[${DUPLICATE_COMMANDS}],"
fi

# Check for specific problematic patterns
LSOF_COUNT=$(ps aux | grep -c "lsof" || echo "0")
RAKE_COUNT=$(ps aux | grep -c "rake" || echo "0") 
PATTERN_DATA="${PATTERN_DATA}\"lsof_processes\":${LSOF_COUNT},\"rake_processes\":${RAKE_COUNT}"
PATTERN_DATA="${PATTERN_DATA}}"

jq ".patterns = ${PATTERN_DATA}" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

# Generate Recommendations
echo "💡 Generating recommendations..."
RECOMMENDATIONS=""

# Check for lsof abuse
if [[ ${LSOF_COUNT} -gt 10 ]]; then
  RECOMMENDATIONS="${RECOMMENDATIONS}\"Replace lsof polling with /proc filesystem reads for better performance\","
fi

# Check for excessive processes
TOTAL_PROCS=$(ps aux | wc -l)
if [[ ${TOTAL_PROCS} -gt 1000 ]]; then
  RECOMMENDATIONS="${RECOMMENDATIONS}\"High process count (${TOTAL_PROCS}) - investigate process spawning\","
fi

# Check load average
LOAD_1MIN=$(echo "${LOAD_AVG}" | cut -d',' -f1)
if (( $(echo "${LOAD_1MIN} > ${CPU_COUNT}" | bc -l) )); then
  RECOMMENDATIONS="${RECOMMENDATIONS}\"Load average (${LOAD_1MIN}) exceeds CPU count (${CPU_COUNT}) - system overloaded\","
fi

RECOMMENDATIONS=$(echo "${RECOMMENDATIONS}" | sed 's/,$//')
if [[ -n "${RECOMMENDATIONS}" ]]; then
  jq ".recommendations = [${RECOMMENDATIONS}]" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
fi

echo "✅ Analysis complete! Results saved to: ${RESULTS_FILE}"

# Output final results to stdout for API consumption
cat "${RESULTS_FILE}"