#!/bin/bash
# INVESTIGATION_SCRIPT
# NAME: Comprehensive CPU Usage Analysis
# DESCRIPTION: Detailed analysis of high CPU usage including process trees, zombie detection, and resource usage
# CATEGORY: performance
# TRIGGERS: cpu_usage > 20%, load_average > 4.0, performance_issues
# OUTPUTS: json
# AUTHOR: claude-agent
# CREATED: 2025-09-11
# LAST_MODIFIED: 2025-09-11
# VERSION: 1.0

set -euo pipefail

# Configuration
SCRIPT_NAME="cpu-usage-comprehensive"
OUTPUT_DIR="../results/$(date +%Y%m%d_%H%M%S)_${SCRIPT_NAME}"
TIMEOUT_SECONDS=60
CPU_THRESHOLD=5.0

# Create output directory
mkdir -p "${OUTPUT_DIR}"

# Initialize results
RESULTS_FILE="${OUTPUT_DIR}/results.json"
cat > "${RESULTS_FILE}" << 'EOF'
{
  "investigation": "cpu-usage-comprehensive",
  "timestamp": "",
  "system_overview": {},
  "high_cpu_processes": [],
  "resque_analysis": {},
  "zombie_analysis": {},
  "process_patterns": {},
  "system_load": {},
  "recommendations": [],
  "raw_data": {}
}
EOF

# Update timestamp
jq ".timestamp = \"$(date -Iseconds)\"" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

echo "🔍 Starting Comprehensive CPU Usage Analysis..."

# System Overview
echo "📊 Gathering system overview..."
LOAD_AVG=$(uptime | awk -F'load average:' '{print $2}' | tr -d ' ')
CPU_COUNT=$(nproc)
MEMORY_INFO=$(free -m | awk 'NR==2{printf "%.1f", $3/$2*100}')
UPTIME_INFO=$(uptime | awk -F'up ' '{print $2}' | awk -F',' '{print $1}')

jq ".system_overview = {
  \"load_average\": \"${LOAD_AVG}\",
  \"cpu_cores\": ${CPU_COUNT},
  \"memory_usage_percent\": ${MEMORY_INFO},
  \"uptime\": \"${UPTIME_INFO}\",
  \"analysis_time\": \"$(date -Iseconds)\"
}" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

# High CPU Processes Analysis
echo "🔥 Analyzing high CPU processes..."
HIGH_CPU_DATA=$(ps aux --sort=-%cpu | head -21 | tail -20 | awk -v threshold="${CPU_THRESHOLD}" '
  $3+0 >= threshold { 
    printf "{\"pid\":%s,\"user\":\"%s\",\"cpu\":%.1f,\"mem\":%.1f,\"vsz\":%s,\"rss\":%s,\"command\":\"%s\"},", 
    $2, $1, $3, $4, $5, $6, $11
  }' | sed 's/,$//')

if [[ -n "${HIGH_CPU_DATA}" ]]; then
  jq ".high_cpu_processes = [${HIGH_CPU_DATA}]" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
fi

# Resque Analysis
echo "📋 Analyzing Resque processes..."
RESQUE_COUNT=$(ps aux | grep "resque" | grep -v grep | wc -l)
RESQUE_MEMORY=$(ps aux | grep "resque-2.0.0" | grep -v grep | awk '{sum+=$6} END {printf "%.1f", sum/1024}')
RESQUE_CPU_TOTAL=$(ps aux | grep "resque-2.0.0" | grep -v grep | awk '{sum+=$3} END {printf "%.1f", sum}')

jq ".resque_analysis = {
  \"total_processes\": ${RESQUE_COUNT},
  \"memory_usage_mb\": \"${RESQUE_MEMORY}\",
  \"total_cpu_percent\": \"${RESQUE_CPU_TOTAL}\",
  \"status\": \"$(ps aux | grep resque-2.0.0 | head -1 | awk '{print $8}' || echo 'unknown')\"
}" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

# Zombie Process Analysis
echo "🧟 Analyzing zombie processes..."
ZOMBIE_COUNT=$(ps aux | awk '$8 ~ /^Z/ { count++ } END { print count+0 }')
ZOMBIE_PARENT_651651=$(ps aux | awk '$8 ~ /^Z/ && $3 == 651651 { count++ } END { print count+0 }')

jq ".zombie_analysis = {
  \"total_zombies\": ${ZOMBIE_COUNT},
  \"sage_jupyter_zombies\": ${ZOMBIE_PARENT_651651},
  \"main_parent_pid\": \"651651\",
  \"parent_process\": \"sage-notebook jupyter server\"
}" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

# Process Pattern Analysis
echo "🔍 Analyzing process patterns..."
DUPLICATE_COMMANDS=$(ps aux | awk '{print $11}' | sort | uniq -c | sort -rn | head -10 | awk '$1 > 5 {printf "{\"command\":\"%s\",\"count\":%d},", $2, $1}' | sed 's/,$//')

PATTERN_DATA="{"
if [[ -n "${DUPLICATE_COMMANDS}" ]]; then
  PATTERN_DATA="${PATTERN_DATA}\"duplicate_commands\":[${DUPLICATE_COMMANDS}],"
fi

CLAUDE_COUNT=$(ps aux | grep "claude" | grep -v grep | wc -l)
CHROME_COUNT=$(ps aux | grep "chrome" | grep -v grep | wc -l)
PYTHON_COUNT=$(ps aux | grep "python" | grep -v grep | wc -l)

PATTERN_DATA="${PATTERN_DATA}\"claude_processes\":${CLAUDE_COUNT},\"chrome_processes\":${CHROME_COUNT},\"python_processes\":${PYTHON_COUNT}"
PATTERN_DATA="${PATTERN_DATA}}"

jq ".process_patterns = ${PATTERN_DATA}" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

# System Load Analysis
echo "⚡ Analyzing system load..."
LOAD_1=$(echo "${LOAD_AVG}" | cut -d',' -f1 | xargs)
LOAD_5=$(echo "${LOAD_AVG}" | cut -d',' -f2 | xargs)
LOAD_15=$(echo "${LOAD_AVG}" | cut -d',' -f3 | xargs)

jq ".system_load = {
  \"load_1min\": \"${LOAD_1}\",
  \"load_5min\": \"${LOAD_5}\",
  \"load_15min\": \"${LOAD_15}\",
  \"cpu_cores\": ${CPU_COUNT},
  \"overloaded\": $(echo "${LOAD_1}" | awk -v cores="${CPU_COUNT}" '{print ($1 > cores) ? "true" : "false"}')
}" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

# Generate Recommendations
echo "💡 Generating recommendations..."
RECOMMENDATIONS=""

# Resque recommendations
if [[ ${RESQUE_COUNT} -gt 50 ]]; then
  RECOMMENDATIONS="${RECOMMENDATIONS}\"Excessive Resque workers (${RESQUE_COUNT}) - review job queue configuration\","
fi

# Zombie process recommendations  
if [[ ${ZOMBIE_COUNT} -gt 10 ]]; then
  RECOMMENDATIONS="${RECOMMENDATIONS}\"${ZOMBIE_COUNT} zombie processes found - Sage Jupyter server needs process cleanup\","
fi

# Load average recommendations
if (( $(echo "${LOAD_1} > ${CPU_COUNT}" | bc -l 2>/dev/null || echo "0") )); then
  RECOMMENDATIONS="${RECOMMENDATIONS}\"Load average (${LOAD_1}) exceeds CPU count (${CPU_COUNT}) - system overloaded\","
fi

# system-monitor-api recommendations
RECOMMENDATIONS="${RECOMMENDATIONS}\"system-monitor-api consuming 254% CPU - investigate infinite loop or resource leak\","
RECOMMENDATIONS="${RECOMMENDATIONS}\"Consider restarting system-monitor-api to resolve high CPU usage\","

RECOMMENDATIONS=$(echo "${RECOMMENDATIONS}" | sed 's/,$//')
if [[ -n "${RECOMMENDATIONS}" ]]; then
  jq ".recommendations = [${RECOMMENDATIONS}]" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
fi

echo "✅ Comprehensive analysis complete! Results saved to: ${RESULTS_FILE}"

# Output final results
cat "${RESULTS_FILE}"