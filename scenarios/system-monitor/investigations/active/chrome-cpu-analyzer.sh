#!/bin/bash
# INVESTIGATION_SCRIPT
# NAME: Chrome High CPU Analyzer
# DESCRIPTION: Detects and analyzes Chrome processes consuming excessive CPU
# CATEGORY: performance
# TRIGGERS: high_chrome_cpu, browser_performance, renderer_overload
# OUTPUTS: json
# AUTHOR: claude-agent
# CREATED: 2025-09-14
# LAST_MODIFIED: 2025-09-14
# VERSION: 1.0

set -euo pipefail

# Configuration
SCRIPT_NAME="chrome-cpu-analyzer"
OUTPUT_DIR="../results/$(date +%Y%m%d_%H%M%S)_${SCRIPT_NAME}"
TIMEOUT_SECONDS=30
CPU_THRESHOLD=50.0

# Create output directory
mkdir -p "${OUTPUT_DIR}"

# Initialize results
RESULTS_FILE="${OUTPUT_DIR}/results.json"
cat > "${RESULTS_FILE}" << 'EOF'
{
  "investigation": "chrome-cpu-analyzer",
  "timestamp": "",
  "chrome_summary": {},
  "high_cpu_processes": [],
  "tab_analysis": {},
  "recommendations": [],
  "auto_fix_applied": false
}
EOF

# Update timestamp
jq ".timestamp = \"$(date -Iseconds)\"" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

echo "🔍 Starting Chrome CPU Analysis..."

# Count Chrome processes
CHROME_COUNT=$(ps aux | grep -E "chrome|chromium" | grep -v grep | wc -l)
CHROME_CPU_TOTAL=$(ps aux | grep -E "chrome|chromium" | grep -v grep | awk '{sum+=$3} END {printf "%.1f", sum}')
CHROME_MEM_TOTAL=$(ps aux | grep -E "chrome|chromium" | grep -v grep | awk '{sum+=$4} END {printf "%.1f", sum}')

jq ".chrome_summary = {
  \"total_processes\": ${CHROME_COUNT},
  \"total_cpu_percent\": \"${CHROME_CPU_TOTAL}\",
  \"total_memory_percent\": \"${CHROME_MEM_TOTAL}\",
  \"detection_time\": \"$(date -Iseconds)\"
}" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

# Find high CPU Chrome processes
echo "🔥 Analyzing high CPU Chrome processes..."
HIGH_CPU_CHROME=$(ps aux | grep -E "chrome|chromium" | grep -v grep | awk -v threshold="${CPU_THRESHOLD}" '
  $3+0 >= threshold { 
    gsub(/"/,"\\\"", $0)
    match($0, /--type=([^ ]+)/, type_arr)
    type = (type_arr[1] ? type_arr[1] : "main")
    printf "{\"pid\":%s,\"cpu\":%.1f,\"mem\":%.1f,\"type\":\"%s\",\"full_cmd\":\"%s\"},", 
    $2, $3, $4, type, substr($0, index($0,$11))
  }' | sed 's/,$//')

if [[ -n "${HIGH_CPU_CHROME}" ]]; then
  jq ".high_cpu_processes = [${HIGH_CPU_CHROME}]" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
  
  # Count affected tabs/types
  RENDERER_COUNT=$(echo "${HIGH_CPU_CHROME}" | grep -o '"type":"renderer"' | wc -l)
  GPU_COUNT=$(echo "${HIGH_CPU_CHROME}" | grep -o '"type":"gpu-process"' | wc -l)
  UTILITY_COUNT=$(echo "${HIGH_CPU_CHROME}" | grep -o '"type":"utility"' | wc -l)
  
  jq ".tab_analysis = {
    \"high_cpu_renderers\": ${RENDERER_COUNT},
    \"high_cpu_gpu\": ${GPU_COUNT},
    \"high_cpu_utility\": ${UTILITY_COUNT}
  }" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
fi

# Generate recommendations
echo "💡 Generating recommendations..."
RECOMMENDATIONS=""
AUTO_FIX=false

# Check for runaway renderer processes (>100% CPU)
if ps aux | grep -E "chrome.*--type=renderer" | grep -v grep | awk '$3 > 100 {exit 0} END {exit 1}'; then
  RECOMMENDATIONS="${RECOMMENDATIONS}\"Found Chrome renderer process(s) consuming >100% CPU - likely JavaScript infinite loop\","
  RECOMMENDATIONS="${RECOMMENDATIONS}\"Consider closing affected tabs or restarting Chrome\","
  
  # Auto-fix: Kill renderer processes over 200% CPU
  KILLED_PIDS=""
  while read -r pid cpu; do
    if (( $(echo "$cpu > 200" | bc -l 2>/dev/null || echo "0") )); then
      echo "🔧 Auto-fixing: Killing runaway Chrome renderer PID $pid (${cpu}% CPU)"
      kill -9 "$pid" 2>/dev/null && KILLED_PIDS="${KILLED_PIDS}${pid}," && AUTO_FIX=true || true
    fi
  done < <(ps aux | grep -E "chrome.*--type=renderer" | grep -v grep | awk '{print $2, $3}')
  
  if [[ -n "${KILLED_PIDS}" ]]; then
    RECOMMENDATIONS="${RECOMMENDATIONS}\"Auto-fix applied: Killed Chrome renderer PIDs [${KILLED_PIDS%,}] due to extreme CPU usage\","
  fi
fi

# Check total Chrome CPU usage
if (( $(echo "${CHROME_CPU_TOTAL} > 100" | bc -l 2>/dev/null || echo "0") )); then
  RECOMMENDATIONS="${RECOMMENDATIONS}\"Chrome using ${CHROME_CPU_TOTAL}% total CPU - consider reducing open tabs\","
fi

# Check for GPU process issues
if ps aux | grep "chrome.*--type=gpu-process" | grep -v grep | awk '$3 > 50 {exit 0} END {exit 1}'; then
  RECOMMENDATIONS="${RECOMMENDATIONS}\"Chrome GPU process consuming high CPU - check hardware acceleration settings\","
fi

RECOMMENDATIONS=$(echo "${RECOMMENDATIONS}" | sed 's/,$//')
if [[ -n "${RECOMMENDATIONS}" ]]; then
  jq ".recommendations = [${RECOMMENDATIONS}]" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
fi

# Update auto-fix status
jq ".auto_fix_applied = ${AUTO_FIX,,}" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

echo "✅ Chrome CPU analysis complete! Results: ${RESULTS_FILE}"

# Output final results
cat "${RESULTS_FILE}"