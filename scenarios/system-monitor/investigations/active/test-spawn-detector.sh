#!/bin/bash
# INVESTIGATION_SCRIPT
# NAME: Test Spawn Detector
# DESCRIPTION: Detects test processes, spawning behavior, and fork patterns for debugging
# CATEGORY: test
# TRIGGERS: test_spawn, fork_testing, process_debugging
# OUTPUTS: json
# AUTHOR: claude-agent
# CREATED: 2025-09-11
# LAST_MODIFIED: 2025-09-11
# VERSION: 1.0

set -euo pipefail

# Configuration
SCRIPT_NAME="test-spawn-detector"
OUTPUT_DIR="../results/$(date +%Y%m%d_%H%M%S)_${SCRIPT_NAME}"
TIMEOUT_SECONDS=30

# Create output directory
mkdir -p "${OUTPUT_DIR}"

# Initialize results
RESULTS_FILE="${OUTPUT_DIR}/results.json"
cat > "${RESULTS_FILE}" << 'EOF'
{
  "investigation": "test-spawn-detector",
  "timestamp": "",
  "test_processes": [],
  "spawn_processes": [],
  "fork_activity": {},
  "multiprocessing_status": {},
  "jupyter_zombies": {},
  "recommendations": []
}
EOF

# Update timestamp
jq ".timestamp = \"$(date -Iseconds)\"" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

echo "🧪 Starting Test Spawn Detection..."

# Find test-related processes
echo "🔍 Searching for test processes..."
TEST_PROCS=$(ps aux | grep -E "(test|Test|TEST|spec|Spec)" | grep -v -E "(grep|kernel|kworker|protest)" | head -20 | \
  awk '{printf "{\"pid\":%s,\"user\":\"%s\",\"cpu\":%.1f,\"mem\":%.1f,\"command\":\"%s\"},", $2, $1, $3, $4, substr($0, index($0,$11))}' | sed 's/,$//')

if [[ -n "${TEST_PROCS}" ]]; then
  jq ".test_processes = [${TEST_PROCS}]" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
fi

# Find spawn-related processes
echo "🔄 Detecting spawn and fork activity..."
SPAWN_PROCS=$(ps aux | grep -E "(spawn|fork|multiprocessing)" | grep -v -E "(grep|kernel|kworker)" | head -20 | \
  awk '{printf "{\"pid\":%s,\"user\":\"%s\",\"cpu\":%.1f,\"mem\":%.1f,\"command\":\"%s\"},", $2, $1, $3, $4, substr($0, index($0,$11))}' | sed 's/,$//')

if [[ -n "${SPAWN_PROCS}" ]]; then
  jq ".spawn_processes = [${SPAWN_PROCS}]" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
fi

# Analyze fork activity
echo "📊 Analyzing fork/spawn patterns..."
FORK_COUNT=$(ps aux | grep -c "fork" || echo "0")
SPAWN_COUNT=$(ps aux | grep -c "spawn" || echo "0")
MULTIPROC_COUNT=$(ps aux | grep -c "multiprocessing" || echo "0")

jq ".fork_activity = {
  \"fork_processes\": ${FORK_COUNT},
  \"spawn_processes\": ${SPAWN_COUNT},
  \"multiprocessing_processes\": ${MULTIPROC_COUNT}
}" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

# Check Python multiprocessing health
echo "🐍 Checking Python multiprocessing status..."
PYTHON_MP_PROCS=$(ps aux | grep "multiprocessing" | grep -v grep | wc -l || echo "0")
PYTHON_MP_TRACKERS=$(ps aux | grep "resource_tracker" | grep -v grep | wc -l || echo "0")

jq ".multiprocessing_status = {
  \"multiprocessing_workers\": ${PYTHON_MP_PROCS},
  \"resource_trackers\": ${PYTHON_MP_TRACKERS},
  \"healthy\": $([ ${PYTHON_MP_TRACKERS} -gt 0 ] && [ ${PYTHON_MP_PROCS} -gt 0 ] && echo "true" || echo "false")
}" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

# Special check for Jupyter zombies
echo "📓 Checking Jupyter zombie situation..."
JUPYTER_PID=$(ps aux | grep -E "jupyter|notebook" | grep -v grep | head -1 | awk '{print $2}')
JUPYTER_ZOMBIES=0
JUPYTER_PARENT=""

if [[ -n "${JUPYTER_PID}" ]]; then
  JUPYTER_ZOMBIES=$(ps --ppid ${JUPYTER_PID} 2>/dev/null | grep -c "<defunct>" || echo "0")
  JUPYTER_PARENT=$(ps -p ${JUPYTER_PID} -o comm= 2>/dev/null || echo "unknown")
fi

jq ".jupyter_zombies = {
  \"jupyter_pid\": \"${JUPYTER_PID:-none}\",
  \"zombie_count\": ${JUPYTER_ZOMBIES},
  \"jupyter_command\": \"${JUPYTER_PARENT}\"
}" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

# Generate recommendations
echo "💡 Generating recommendations..."
RECOMMENDATIONS=""

# Test process recommendations
if [[ $(echo "${TEST_PROCS}" | grep -c "pid") -gt 5 ]]; then
  RECOMMENDATIONS="${RECOMMENDATIONS}\"Multiple test processes detected - ensure tests are cleaning up properly\","
fi

# Spawn/fork recommendations
if [[ ${SPAWN_COUNT} -gt 20 ]] || [[ ${FORK_COUNT} -gt 20 ]]; then
  RECOMMENDATIONS="${RECOMMENDATIONS}\"High spawn/fork activity detected - check for runaway process creation\","
fi

# Multiprocessing recommendations
if [[ ${PYTHON_MP_PROCS} -gt 10 ]]; then
  RECOMMENDATIONS="${RECOMMENDATIONS}\"Many Python multiprocessing workers - verify pool size configuration\","
fi

# Jupyter zombie recommendations
if [[ ${JUPYTER_ZOMBIES} -gt 10 ]]; then
  RECOMMENDATIONS="${RECOMMENDATIONS}\"Jupyter has ${JUPYTER_ZOMBIES} zombie children - restart notebook server\","
  RECOMMENDATIONS="${RECOMMENDATIONS}\"Consider upgrading Jupyter or reviewing kernel management\","
fi

# General recommendations
RECOMMENDATIONS="${RECOMMENDATIONS}\"Monitor process creation rate with 'vmstat 1'\","
RECOMMENDATIONS="${RECOMMENDATIONS}\"Use strace -f to trace fork/spawn calls for debugging\""

RECOMMENDATIONS=$(echo "${RECOMMENDATIONS}" | sed 's/,$//')
jq ".recommendations = [${RECOMMENDATIONS}]" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

echo "✅ Test spawn detection complete! Results: ${RESULTS_FILE}"

# Output final results
cat "${RESULTS_FILE}"