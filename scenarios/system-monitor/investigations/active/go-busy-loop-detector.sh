#!/bin/bash
# INVESTIGATION_SCRIPT
# NAME: Go Busy Loop Detector
# DESCRIPTION: Detects Go processes with busy polling loops consuming excessive CPU
# CATEGORY: performance
# TRIGGERS: high_go_cpu, busy_loop_suspected, missing_sleep
# OUTPUTS: json
# AUTHOR: claude-agent
# CREATED: 2025-09-11
# LAST_MODIFIED: 2025-09-11
# VERSION: 1.0

set -euo pipefail

# Configuration
SCRIPT_NAME="go-busy-loop-detector"
OUTPUT_DIR="../results/$(date +%Y%m%d_%H%M%S)_${SCRIPT_NAME}"
TIMEOUT_SECONDS=30
CPU_THRESHOLD=50.0

# Create output directory
mkdir -p "${OUTPUT_DIR}"

# Initialize results
RESULTS_FILE="${OUTPUT_DIR}/results.json"
cat > "${RESULTS_FILE}" << 'EOF'
{
  "investigation": "go-busy-loop-detector",
  "timestamp": "",
  "go_processes": [],
  "busy_loops_detected": [],
  "io_patterns": {},
  "syscall_patterns": {},
  "recommendations": []
}
EOF

# Update timestamp
jq ".timestamp = \"$(date -Iseconds)\"" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

echo "🔍 Starting Go Busy Loop Detection..."

# Find all Go processes with high CPU
echo "🎯 Locating high-CPU Go processes..."
GO_PROCS=$(ps aux | grep -E "(system-monitor-api|api/main|go-build)" | grep -v grep | awk -v threshold="${CPU_THRESHOLD}" '$3+0 >= threshold {print $2":"$3":"$11}')

if [[ -z "${GO_PROCS}" ]]; then
  echo "✅ No high-CPU Go processes found"
  jq '.go_processes = []' "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
else
  PROCS_JSON=""
  for PROC in ${GO_PROCS}; do
    PID=$(echo "${PROC}" | cut -d: -f1)
    CPU=$(echo "${PROC}" | cut -d: -f2)
    CMD=$(echo "${PROC}" | cut -d: -f3)
    
    echo "📊 Analyzing PID ${PID} (${CPU}% CPU)..."
    
    # Check I/O stats to detect tight loops
    if [[ -e "/proc/${PID}/io" ]]; then
      READ_BYTES=$(cat /proc/${PID}/io | grep read_bytes | cut -d: -f2 | tr -d ' ')
      WRITE_BYTES=$(cat /proc/${PID}/io | grep write_bytes | cut -d: -f2 | tr -d ' ')
      SYSCALLS=$(cat /proc/${PID}/io | grep syscr | cut -d: -f2 | tr -d ' ' || echo "0")
      
      # Calculate I/O rate (bytes per second, approximate)
      RUNTIME_SECS=$(ps -o etimes= -p ${PID} 2>/dev/null | tr -d ' ' || echo "1")
      if [[ ${RUNTIME_SECS} -gt 0 ]]; then
        READ_RATE=$((READ_BYTES / RUNTIME_SECS))
        WRITE_RATE=$((WRITE_BYTES / RUNTIME_SECS))
        SYSCALL_RATE=$((SYSCALLS / RUNTIME_SECS))
      else
        READ_RATE=0
        WRITE_RATE=0
        SYSCALL_RATE=0
      fi
      
      # Check voluntary context switches (low = tight loop)
      VOL_CTXT_SW=$(cat /proc/${PID}/status | grep voluntary_ctxt_switches | awk '{print $2}' || echo "0")
      NONVOL_CTXT_SW=$(cat /proc/${PID}/status | grep nonvoluntary_ctxt_switches | awk '{print $2}' || echo "0")
      
      # Detect busy loop pattern: high CPU, low I/O, low voluntary context switches
      BUSY_LOOP="false"
      LOOP_CONFIDENCE=0
      
      if (( $(echo "${CPU} > 100" | bc -l) )); then
        LOOP_CONFIDENCE=$((LOOP_CONFIDENCE + 30))
      fi
      
      if [[ ${READ_RATE} -lt 1000 && ${WRITE_RATE} -lt 1000 ]]; then
        LOOP_CONFIDENCE=$((LOOP_CONFIDENCE + 20))
      fi
      
      if [[ ${RUNTIME_SECS} -gt 60 ]]; then
        VOL_SW_RATE=$((VOL_CTXT_SW / RUNTIME_SECS))
        if [[ ${VOL_SW_RATE} -lt 100 ]]; then
          LOOP_CONFIDENCE=$((LOOP_CONFIDENCE + 30))
        fi
      fi
      
      # High syscall rate without I/O suggests polling
      if [[ ${SYSCALL_RATE} -gt 1000 && ${READ_RATE} -lt 1000 ]]; then
        LOOP_CONFIDENCE=$((LOOP_CONFIDENCE + 20))
      fi
      
      if [[ ${LOOP_CONFIDENCE} -ge 60 ]]; then
        BUSY_LOOP="true"
      fi
      
      PROCS_JSON="${PROCS_JSON}{
        \"pid\": ${PID},
        \"cpu_percent\": ${CPU},
        \"command\": \"${CMD}\",
        \"runtime_seconds\": ${RUNTIME_SECS},
        \"read_bytes_per_sec\": ${READ_RATE},
        \"write_bytes_per_sec\": ${WRITE_RATE},
        \"syscalls_per_sec\": ${SYSCALL_RATE},
        \"voluntary_ctxt_switches\": ${VOL_CTXT_SW},
        \"nonvoluntary_ctxt_switches\": ${NONVOL_CTXT_SW},
        \"busy_loop_detected\": ${BUSY_LOOP},
        \"confidence_score\": ${LOOP_CONFIDENCE}
      },"
    fi
  done
  
  # Remove trailing comma and update results
  PROCS_JSON=$(echo "${PROCS_JSON}" | sed 's/,$//')
  jq ".go_processes = [${PROCS_JSON}]" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
fi

# Generate recommendations
echo "💡 Generating recommendations..."
RECOMMENDATIONS=""

# Check for specific issues
if jq -e '.go_processes[] | select(.busy_loop_detected == true)' "${RESULTS_FILE}" > /dev/null 2>&1; then
  RECOMMENDATIONS="${RECOMMENDATIONS}\"Busy loop detected in Go process - add time.Sleep() to polling loops\","
  RECOMMENDATIONS="${RECOMMENDATIONS}\"Review monitoring service loops for missing delays between iterations\","
  RECOMMENDATIONS="${RECOMMENDATIONS}\"Check WebSocket handlers for busy-wait patterns\","
  RECOMMENDATIONS="${RECOMMENDATIONS}\"Consider using channels or condition variables instead of polling\","
fi

if jq -e '.go_processes[] | select(.cpu_percent > 200)' "${RESULTS_FILE}" > /dev/null 2>&1; then
  RECOMMENDATIONS="${RECOMMENDATIONS}\"Process using multiple CPU cores inefficiently - review goroutine synchronization\","
  RECOMMENDATIONS="${RECOMMENDATIONS}\"Consider adding runtime.Gosched() calls in tight loops\","
fi

if jq -e '.go_processes[] | select(.syscalls_per_sec > 10000)' "${RESULTS_FILE}" > /dev/null 2>&1; then
  RECOMMENDATIONS="${RECOMMENDATIONS}\"Excessive syscall rate detected - batch operations or add buffering\","
fi

RECOMMENDATIONS="${RECOMMENDATIONS}\"Restart the system-monitor-api process to immediately resolve high CPU\","
RECOMMENDATIONS=$(echo "${RECOMMENDATIONS}" | sed 's/,$//')

if [[ -n "${RECOMMENDATIONS}" ]]; then
  jq ".recommendations = [${RECOMMENDATIONS}]" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
fi

echo "✅ Go Busy Loop detection complete! Results: ${RESULTS_FILE}"

# Output final results
cat "${RESULTS_FILE}"