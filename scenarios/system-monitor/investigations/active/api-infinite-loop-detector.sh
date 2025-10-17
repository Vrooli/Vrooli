#!/bin/bash
# INVESTIGATION_SCRIPT
# NAME: API Infinite Loop Detector
# DESCRIPTION: Detects and diagnoses infinite loop conditions in Go API services
# CATEGORY: performance
# TRIGGERS: high_cpu_api, infinite_loop_suspected, go_service_issue
# OUTPUTS: json
# AUTHOR: claude-agent
# CREATED: 2025-09-11
# LAST_MODIFIED: 2025-09-11
# VERSION: 1.0

set -euo pipefail

# Configuration
SCRIPT_NAME="api-infinite-loop-detector"
OUTPUT_DIR="../results/$(date +%Y%m%d_%H%M%S)_${SCRIPT_NAME}"
TIMEOUT_SECONDS=30

# Create output directory
mkdir -p "${OUTPUT_DIR}"

# Initialize results
RESULTS_FILE="${OUTPUT_DIR}/results.json"
cat > "${RESULTS_FILE}" << 'EOF'
{
  "investigation": "api-infinite-loop-detector",
  "timestamp": "",
  "target_processes": [],
  "diagnosis": {},
  "evidence": {},
  "recommendations": []
}
EOF

# Update timestamp
jq ".timestamp = \"$(date -Iseconds)\"" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

echo "🔍 Starting API Infinite Loop Detection..."

# Find high CPU Go processes
echo "🎯 Finding high CPU Go processes..."
HIGH_CPU_APIS=$(ps aux --sort=-%cpu | head -20 | grep -E "\.(api|server|monitor)" | grep -v grep | head -5 || true)

if [[ -z "${HIGH_CPU_APIS}" ]]; then
  echo "✅ No high CPU API processes found"
  jq '.diagnosis = {"status": "healthy", "message": "No high CPU API processes detected"}' "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
else
  # Analyze each high CPU process
  PROCESS_DATA="["
  
  while IFS= read -r proc_line; do
    if [[ -n "${proc_line}" ]]; then
      PID=$(echo "${proc_line}" | awk '{print $2}')
      CPU=$(echo "${proc_line}" | awk '{print $3}')
      CMD=$(echo "${proc_line}" | awk '{print $11}')
      
      echo "📊 Analyzing ${CMD} (PID: ${PID}, CPU: ${CPU}%)..."
      
      # Get runtime
      RUNTIME=$(ps -p ${PID} -o etime= 2>/dev/null | tr -d ' ' || echo "unknown")
      
      # Check I/O stats to detect busy loops
      IO_READ=0
      IO_WRITE=0
      if [[ -r "/proc/${PID}/io" ]]; then
        IO_READ=$(grep read_bytes /proc/${PID}/io 2>/dev/null | cut -d: -f2 | tr -d ' ' || echo 0)
        IO_WRITE=$(grep write_bytes /proc/${PID}/io 2>/dev/null | cut -d: -f2 | tr -d ' ' || echo 0)
      fi
      
      # Check thread count
      THREADS=$(cat /proc/${PID}/status 2>/dev/null | grep Threads | awk '{print $2}' || echo 0)
      
      # Calculate I/O rate (bytes per CPU percent - low means tight loop)
      IO_RATE=0
      if (( $(echo "${CPU} > 0" | bc -l 2>/dev/null || echo 0) )); then
        IO_TOTAL=$((IO_READ + IO_WRITE))
        IO_RATE=$(echo "scale=2; ${IO_TOTAL}/${CPU}" | bc 2>/dev/null || echo 0)
      fi
      
      # Determine if it's likely an infinite loop
      LOOP_SUSPECTED="false"
      LOOP_REASON=""
      
      if (( $(echo "${CPU} > 100" | bc -l 2>/dev/null || echo 0) )); then
        if (( $(echo "${IO_RATE} < 10000" | bc -l 2>/dev/null || echo 0) )); then
          LOOP_SUSPECTED="true"
          LOOP_REASON="High CPU with minimal I/O activity"
        fi
      fi
      
      PROCESS_DATA="${PROCESS_DATA}{
        \"pid\": ${PID},
        \"command\": \"${CMD}\",
        \"cpu_percent\": ${CPU},
        \"runtime\": \"${RUNTIME}\",
        \"threads\": ${THREADS},
        \"io_read_bytes\": ${IO_READ},
        \"io_write_bytes\": ${IO_WRITE},
        \"io_per_cpu\": ${IO_RATE},
        \"loop_suspected\": ${LOOP_SUSPECTED},
        \"loop_reason\": \"${LOOP_REASON}\"
      },"
    fi
  done <<< "${HIGH_CPU_APIS}"
  
  PROCESS_DATA=$(echo "${PROCESS_DATA}" | sed 's/,$//')
  PROCESS_DATA="${PROCESS_DATA}]"
  
  # Update results
  jq ".target_processes = ${PROCESS_DATA}" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
  
  # Generate diagnosis
  LOOP_COUNT=$(echo "${PROCESS_DATA}" | jq '[.[] | select(.loop_suspected == true)] | length')
  
  if [[ ${LOOP_COUNT} -gt 0 ]]; then
    jq '.diagnosis = {
      "status": "critical",
      "message": "Infinite loop conditions detected",
      "affected_processes": '"${LOOP_COUNT}"'
    }' "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
    
    # Add evidence
    jq '.evidence = {
      "pattern": "high_cpu_low_io",
      "description": "Process consuming CPU without corresponding I/O indicates tight computational loop",
      "typical_causes": [
        "Missing time.Sleep() in polling loops",
        "Busy-wait on channel or condition",
        "Infinite recursion without exit condition",
        "Websocket handler in tight read loop",
        "HTTP handler missing response.Write or return"
      ]
    }' "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
    
    # Generate recommendations
    RECOMMENDATIONS=""
    RECOMMENDATIONS="${RECOMMENDATIONS}\"IMMEDIATE: Restart affected API services to restore normal operation\","
    RECOMMENDATIONS="${RECOMMENDATIONS}\"Review recent code changes for polling loops without delays\","
    RECOMMENDATIONS="${RECOMMENDATIONS}\"Add pprof CPU profiling to identify hot code paths\","
    RECOMMENDATIONS="${RECOMMENDATIONS}\"Check websocket handlers for proper connection management\","
    RECOMMENDATIONS="${RECOMMENDATIONS}\"Verify HTTP handlers complete responses properly\","
    RECOMMENDATIONS="${RECOMMENDATIONS}\"Add rate limiting and circuit breakers to prevent runaway loops\","
    RECOMMENDATIONS="${RECOMMENDATIONS}\"Implement health check endpoints with timeout protection\""
    
    jq ".recommendations = [${RECOMMENDATIONS}]" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
  else
    jq '.diagnosis = {
      "status": "warning",
      "message": "High CPU usage detected but no clear infinite loop pattern",
      "affected_processes": '"${LOOP_COUNT}"'
    }' "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
  fi
fi

echo "✅ Infinite loop detection complete! Results: ${RESULTS_FILE}"

# Output final results
cat "${RESULTS_FILE}"
