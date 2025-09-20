#!/bin/bash
# INVESTIGATION_SCRIPT
# NAME: Judge0 Memory Analyzer
# DESCRIPTION: Analyzes Judge0 worker containers for memory exhaustion and provides optimization recommendations
# CATEGORY: resource-management
# TRIGGERS: judge0_memory_high, container_memory_limit, worker_memory_exhaustion
# OUTPUTS: json
# AUTHOR: claude-agent
# CREATED: 2025-09-14
# LAST_MODIFIED: 2025-09-14
# VERSION: 1.0

set -euo pipefail

# Configuration
SCRIPT_NAME="judge0-memory-analyzer"
OUTPUT_DIR="../results/$(date +%Y%m%d_%H%M%S)_${SCRIPT_NAME}"
TIMEOUT_SECONDS=30
MEMORY_THRESHOLD=90  # Percentage

# Create output directory
mkdir -p "${OUTPUT_DIR}"

# Initialize results
RESULTS_FILE="${OUTPUT_DIR}/results.json"
cat > "${RESULTS_FILE}" << 'EOF'
{
  "investigation": "judge0-memory-analyzer",
  "timestamp": "",
  "judge0_workers": [],
  "memory_issues": [],
  "configuration_analysis": {},
  "recommendations": []
}
EOF

# Update timestamp
jq ".timestamp = \"$(date -Iseconds)\"" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

echo "🔍 Starting Judge0 Memory Analysis..."

# Analyze Judge0 worker containers
echo "📊 Analyzing Judge0 worker memory usage..."
JUDGE0_WORKERS=$(docker ps --format "{{.Names}}" | grep -E "judge0|worker" || true)

if [[ -n "${JUDGE0_WORKERS}" ]]; then
  WORKER_DATA="["
  for worker in ${JUDGE0_WORKERS}; do
    # Get container stats
    STATS=$(timeout 5 docker stats "$worker" --no-stream --format "json" 2>/dev/null || echo '{}')
    if [[ "$STATS" != '{}' ]]; then
      MEM_USAGE=$(echo "$STATS" | jq -r '.MemUsage')
      MEM_PERCENT=$(echo "$STATS" | jq -r '.MemPerc' | tr -d '%')
      CPU_PERCENT=$(echo "$STATS" | jq -r '.CPUPerc' | tr -d '%')
      
      WORKER_DATA="${WORKER_DATA}{\"name\":\"$worker\",\"memory_usage\":\"$MEM_USAGE\",\"memory_percent\":$MEM_PERCENT,\"cpu_percent\":$CPU_PERCENT},"
      
      # Check if memory usage is critical
      if (( $(echo "$MEM_PERCENT > $MEMORY_THRESHOLD" | bc -l) )); then
        jq ".memory_issues += [{
          \"container\": \"$worker\",
          \"severity\": \"critical\",
          \"memory_percent\": $MEM_PERCENT,
          \"message\": \"Memory usage exceeds ${MEMORY_THRESHOLD}% threshold\"
        }]" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
      fi
    fi
  done
  WORKER_DATA=$(echo "$WORKER_DATA" | sed 's/,$/]/')
  
  jq ".judge0_workers = $WORKER_DATA" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
fi

# Check Judge0 configuration
echo "⚙️ Checking Judge0 configuration..."
JUDGE0_CONFIG=$(docker inspect config-judge0-workers-1 2>/dev/null | jq '.[0] | {
  "memory_limit": (.HostConfig.Memory // 0),
  "memory_reservation": (.HostConfig.MemoryReservation // 0),
  "memory_swap": (.HostConfig.MemorySwap // 0),
  "cpu_shares": (.HostConfig.CpuShares // 0),
  "restart_policy": .HostConfig.RestartPolicy.Name
}' || echo '{}')

if [[ "$JUDGE0_CONFIG" != '{}' ]]; then
  jq ".configuration_analysis = $JUDGE0_CONFIG" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
fi

# Check for memory-related errors in Judge0 logs
echo "🔎 Checking Judge0 logs for memory issues..."
for worker in ${JUDGE0_WORKERS}; do
  MEM_ERRORS=$(timeout 5 docker logs "$worker" 2>&1 | tail -200 | grep -ciE "(out of memory|memory limit|oom|malloc|heap)" || echo "0")
  if [[ ${MEM_ERRORS} -gt 0 ]]; then
    jq ".memory_issues += [{
      \"container\": \"$worker\",
      \"severity\": \"warning\",
      \"error_count\": $MEM_ERRORS,
      \"message\": \"Memory-related errors found in logs\"
    }]" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
  fi
done

# Generate recommendations
echo "💡 Generating recommendations..."
RECOMMENDATIONS=""

# Check for specific Judge0 memory issues
if docker stats --no-stream --format "{{.MemPerc}}" config-judge0-workers-1 2>/dev/null | grep -qE "9[5-9]\.|100\."; then
  RECOMMENDATIONS="${RECOMMENDATIONS}\"Judge0 worker 1 at critical memory usage - increase memory limit or reduce concurrent executions\","
fi

if docker stats --no-stream --format "{{.MemPerc}}" config-judge0-workers-2 2>/dev/null | grep -qE "9[5-9]\.|100\."; then
  RECOMMENDATIONS="${RECOMMENDATIONS}\"Judge0 worker 2 at critical memory usage - increase memory limit or reduce concurrent executions\","
fi

# General Judge0 recommendations
RECOMMENDATIONS="${RECOMMENDATIONS}\"Consider implementing memory limits per code execution in Judge0 configuration\","
RECOMMENDATIONS="${RECOMMENDATIONS}\"Review Judge0 worker pool size vs available system memory\","
RECOMMENDATIONS="${RECOMMENDATIONS}\"Enable swap accounting if not already enabled for better container memory management\","

RECOMMENDATIONS=$(echo "${RECOMMENDATIONS}" | sed 's/,$//')
if [[ -n "${RECOMMENDATIONS}" ]]; then
  jq ".recommendations = [${RECOMMENDATIONS}]" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
fi

echo "✅ Judge0 memory analysis complete! Results: ${RESULTS_FILE}"

# Output final results
cat "${RESULTS_FILE}"