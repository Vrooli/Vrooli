#!/bin/bash
# INVESTIGATION_SCRIPT
# NAME: Memory Leak and Process Health Detector
# DESCRIPTION: Detects memory leaks, long-running processes, and resource consumption patterns
# CATEGORY: resource-management
# TRIGGERS: high_memory_usage, memory_growth_pattern, resource_exhaustion
# OUTPUTS: json
# AUTHOR: claude-agent
# CREATED: 2025-09-14
# LAST_MODIFIED: 2025-09-14
# VERSION: 1.0

set -euo pipefail

# Configuration
SCRIPT_NAME="memory-leak-detector"
OUTPUT_DIR="../results/$(date +%Y%m%d_%H%M%S)_${SCRIPT_NAME}"
TIMEOUT_SECONDS=60
MEMORY_THRESHOLD=1000  # MB

# Create output directory
mkdir -p "${OUTPUT_DIR}"

# Initialize results
RESULTS_FILE="${OUTPUT_DIR}/results.json"
cat > "${RESULTS_FILE}" << 'EOF'
{
  "investigation": "memory-leak-detector",
  "timestamp": "",
  "memory_overview": {},
  "high_memory_processes": [],
  "long_running_processes": [],
  "container_memory": [],
  "python_processes": [],
  "recommendations": []
}
EOF

# Update timestamp
jq ".timestamp = \"$(date -Iseconds)\"" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

echo "🔍 Starting Memory Leak and Process Health Detection..."

# Memory Overview
echo "📊 Analyzing system memory..."
TOTAL_MEM=$(free -m | awk 'NR==2{print $2}')
USED_MEM=$(free -m | awk 'NR==2{print $3}')
FREE_MEM=$(free -m | awk 'NR==2{print $4}')
CACHE_MEM=$(free -m | awk 'NR==2{print $6}')
MEM_PERCENT=$(awk "BEGIN {printf \"%.1f\", ($USED_MEM/$TOTAL_MEM)*100}")

jq ".memory_overview = {
  \"total_mb\": $TOTAL_MEM,
  \"used_mb\": $USED_MEM,
  \"free_mb\": $FREE_MEM,
  \"cache_mb\": $CACHE_MEM,
  \"usage_percent\": $MEM_PERCENT
}" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

# High Memory Processes
echo "🔥 Finding high memory processes..."
HIGH_MEM_PROCS=$(ps aux --sort=-%mem | head -20 | awk 'NR>1{
  printf "{\"pid\": %s, \"user\": \"%s\", \"mem_percent\": %.1f, \"rss_kb\": %s, \"command\": \"%s\"},\n", 
  $2, $1, $4, $6, substr($0, index($0,$11))
}' | sed 's/,$//' | jq -s '.')

jq ".high_memory_processes = $HIGH_MEM_PROCS" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

# Long Running Processes (potential memory leaks)
echo "⏱️ Analyzing long-running processes..."
LONG_RUNNERS=$(ps aux --sort=-etime | head -20 | awk 'NR>1{
  # Extract elapsed time in a readable format
  etime=$10
  printf "{\"pid\": %s, \"user\": \"%s\", \"mem_percent\": %.1f, \"cpu_percent\": %.1f, \"elapsed\": \"%s\", \"command\": \"%s\"},\n",
  $2, $1, $4, $3, etime, substr($0, index($0,$11))
}' | sed 's/,$//' | jq -s '.')

jq ".long_running_processes = $LONG_RUNNERS" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

# Container Memory Analysis
echo "🐳 Checking container memory usage..."
if command -v docker &> /dev/null; then
  CONTAINER_MEM=$(timeout 5 docker stats --no-stream --format "json" 2>/dev/null | jq -s '[.[] | {
    "container": .Name,
    "memory_usage": .MemUsage,
    "memory_percent": .MemPerc,
    "cpu_percent": .CPUPerc
  }]' 2>/dev/null || echo "[]")
  
  jq ".container_memory = $CONTAINER_MEM" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
fi

# Python Process Analysis (since we found many zombie Python processes)
echo "🐍 Analyzing Python processes..."
PYTHON_PROCS=$(ps aux | grep -E "python[0-9]*" | grep -v grep | awk '{
  printf "{\"pid\": %s, \"user\": \"%s\", \"mem_percent\": %.1f, \"cpu_percent\": %.1f, \"state\": \"%s\", \"command\": \"%s\"},\n",
  $2, $1, $4, $3, $8, substr($0, index($0,$11))
}' | sed 's/,$//' | jq -s '.' 2>/dev/null || echo "[]")

jq ".python_processes = $PYTHON_PROCS" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

# Generate Recommendations
echo "💡 Generating recommendations..."
RECOMMENDATIONS=[]

# Check for memory pressure
if (( $(echo "$MEM_PERCENT > 80" | bc -l) )); then
  RECOMMENDATIONS+=("High memory usage ($MEM_PERCENT%) - investigate top memory consumers")
fi

# Check for Python subprocess issues
PYTHON_COUNT=$(echo "$PYTHON_PROCS" | jq 'length')
if [ "$PYTHON_COUNT" -gt 20 ]; then
  RECOMMENDATIONS+=("Many Python processes ($PYTHON_COUNT) detected - review subprocess management")
fi

# Check for container memory issues
if [ -n "$CONTAINER_MEM" ] && [ "$CONTAINER_MEM" != "[]" ]; then
  HIGH_CONTAINER_COUNT=$(echo "$CONTAINER_MEM" | jq '[.[] | select(.memory_percent | gsub("%";"") | tonumber > 50)] | length')
  if [ "$HIGH_CONTAINER_COUNT" -gt 0 ]; then
    RECOMMENDATIONS+=("$HIGH_CONTAINER_COUNT containers using >50% memory - review container limits")
  fi
fi

# Check for specific problematic processes
if echo "$HIGH_MEM_PROCS" | jq -r '.[].command' | grep -q "sage"; then
  RECOMMENDATIONS+=("SageMath process consuming significant memory - check for computation issues")
fi

# Convert recommendations to JSON
RECS_JSON=$(printf '%s\n' "${RECOMMENDATIONS[@]}" | jq -R . | jq -s .)
jq ".recommendations = $RECS_JSON" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

echo "✅ Memory leak detection complete! Results: ${OUTPUT_DIR}/results.json"
cat "${RESULTS_FILE}" | jq '.'