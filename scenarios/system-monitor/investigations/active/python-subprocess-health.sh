#!/bin/bash
# INVESTIGATION_SCRIPT
# NAME: Python Subprocess Health Monitor
# DESCRIPTION: Detects Python processes with zombie children and poor subprocess management
# CATEGORY: process-analysis
# TRIGGERS: python_zombies, jupyter_issues, subprocess_leaks
# OUTPUTS: json
# AUTHOR: claude-agent
# CREATED: 2025-09-11
# LAST_MODIFIED: 2025-09-11
# VERSION: 1.0

set -euo pipefail

# Configuration
SCRIPT_NAME="python-subprocess-health"
OUTPUT_DIR="../results/$(date +%Y%m%d_%H%M%S)_${SCRIPT_NAME}"
TIMEOUT_SECONDS=45

# Create output directory
mkdir -p "${OUTPUT_DIR}"

# Initialize results
RESULTS_FILE="${OUTPUT_DIR}/results.json"
cat > "${RESULTS_FILE}" << 'EOF'
{
  "investigation": "python-subprocess-health",
  "timestamp": "",
  "python_processes": [],
  "jupyter_status": {},
  "zombie_analysis": {},
  "subprocess_issues": [],
  "recommendations": []
}
EOF

# Update timestamp
jq ".timestamp = \"$(date -Iseconds)\"" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

echo "🐍 Starting Python Subprocess Health Check..."

# Find all Python processes
echo "📊 Analyzing Python processes..."
PYTHON_PROCS=$(ps aux | grep -E "python[0-9]?" | grep -v grep | awk '{print $2}' | head -20)

PYTHON_DETAILS="["
for pid in ${PYTHON_PROCS}; do
  if [[ -e /proc/${pid}/cmdline ]]; then
    CMD=$(tr '\0' ' ' < /proc/${pid}/cmdline | head -c 200)
    # Count zombie children
    ZOMBIE_COUNT=$(ps --ppid ${pid} | grep -c "<defunct>" || echo "0")
    # Get memory usage
    MEM=$(ps -p ${pid} -o rss= 2>/dev/null || echo "0")
    # Get CPU usage
    CPU=$(ps -p ${pid} -o %cpu= 2>/dev/null || echo "0")
    
    PYTHON_DETAILS="${PYTHON_DETAILS}{\"pid\":${pid},\"zombie_children\":${ZOMBIE_COUNT},\"memory_mb\":$((MEM/1024)),\"cpu_percent\":${CPU},\"command\":\"${CMD}\"},"
  fi
done
PYTHON_DETAILS=$(echo "${PYTHON_DETAILS}" | sed 's/,$//')"]"

# Update Python process details
jq ".python_processes = ${PYTHON_DETAILS}" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

# Check for Jupyter specifically
echo "📓 Checking Jupyter notebook servers..."
JUPYTER_PIDS=$(ps aux | grep -E "jupyter|notebook" | grep -v grep | awk '{print $2}' | head -5)
JUPYTER_STATUS="{"

if [[ -n "${JUPYTER_PIDS}" ]]; then
  JUPYTER_COUNT=$(echo "${JUPYTER_PIDS}" | wc -l)
  JUPYTER_STATUS="${JUPYTER_STATUS}\"running\":true,\"count\":${JUPYTER_COUNT},"
  
  # Check for zombie accumulation in Jupyter
  TOTAL_JUPYTER_ZOMBIES=0
  for pid in ${JUPYTER_PIDS}; do
    ZOMBIES=$(ps --ppid ${pid} 2>/dev/null | grep -c "<defunct>" || echo "0")
    TOTAL_JUPYTER_ZOMBIES=$((TOTAL_JUPYTER_ZOMBIES + ZOMBIES))
  done
  JUPYTER_STATUS="${JUPYTER_STATUS}\"total_zombies\":${TOTAL_JUPYTER_ZOMBIES}"
else
  JUPYTER_STATUS="${JUPYTER_STATUS}\"running\":false,\"count\":0,\"total_zombies\":0"
fi
JUPYTER_STATUS="${JUPYTER_STATUS}}"

# Update Jupyter status
jq ".jupyter_status = ${JUPYTER_STATUS}" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

# Analyze zombie patterns
echo "🧟 Analyzing zombie patterns..."
TOTAL_PYTHON_ZOMBIES=$(ps aux | grep "<defunct>" | grep -c "python" || echo "0")
ZOMBIE_PARENTS=$(ps aux | grep "<defunct>" | awk '{print $3}' | sort -u | head -10)

ZOMBIE_ANALYSIS="{\"total_python_zombies\":${TOTAL_PYTHON_ZOMBIES},\"unique_parents\":$(echo "${ZOMBIE_PARENTS}" | wc -l)}"
jq ".zombie_analysis = ${ZOMBIE_ANALYSIS}" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

# Check for subprocess issues
echo "🔍 Detecting subprocess management issues..."
SUBPROCESS_ISSUES="["

# Check for processes with many zombies
for pid in ${PYTHON_PROCS}; do
  ZOMBIE_COUNT=$(ps --ppid ${pid} 2>/dev/null | grep -c "<defunct>" || echo "0")
  if [[ ${ZOMBIE_COUNT} -gt 5 ]]; then
    SUBPROCESS_ISSUES="${SUBPROCESS_ISSUES}{\"pid\":${pid},\"issue\":\"excessive_zombies\",\"count\":${ZOMBIE_COUNT}},"
  fi
done

# Check for processes with too many children
for pid in ${PYTHON_PROCS}; do
  CHILD_COUNT=$(ps --ppid ${pid} 2>/dev/null | wc -l || echo "0")
  if [[ ${CHILD_COUNT} -gt 50 ]]; then
    SUBPROCESS_ISSUES="${SUBPROCESS_ISSUES}{\"pid\":${pid},\"issue\":\"too_many_children\",\"count\":${CHILD_COUNT}},"
  fi
done

SUBPROCESS_ISSUES=$(echo "${SUBPROCESS_ISSUES}" | sed 's/,$//')"]"
jq ".subprocess_issues = ${SUBPROCESS_ISSUES}" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

# Generate recommendations
echo "💡 Generating recommendations..."
RECOMMENDATIONS=""

if [[ ${TOTAL_PYTHON_ZOMBIES} -gt 10 ]]; then
  RECOMMENDATIONS="${RECOMMENDATIONS}\"Implement proper subprocess.wait() or communicate() in Python scripts\","
  RECOMMENDATIONS="${RECOMMENDATIONS}\"Use subprocess.run() instead of subprocess.Popen() when possible\","
fi

if [[ $(echo "${JUPYTER_STATUS}" | grep -o '"total_zombies":[0-9]*' | cut -d: -f2) -gt 10 ]]; then
  RECOMMENDATIONS="${RECOMMENDATIONS}\"Jupyter kernel management issue - consider restarting notebook server\","
  RECOMMENDATIONS="${RECOMMENDATIONS}\"Review Jupyter kernel lifecycle configuration\","
fi

if [[ $(echo "${SUBPROCESS_ISSUES}" | grep -c "excessive_zombies") -gt 0 ]]; then
  RECOMMENDATIONS="${RECOMMENDATIONS}\"Add signal.signal(signal.SIGCHLD, signal.SIG_IGN) to prevent zombies\","
  RECOMMENDATIONS="${RECOMMENDATIONS}\"Use context managers for subprocess management\","
fi

if [[ $(echo "${SUBPROCESS_ISSUES}" | grep -c "too_many_children") -gt 0 ]]; then
  RECOMMENDATIONS="${RECOMMENDATIONS}\"Implement subprocess pooling or rate limiting\","
  RECOMMENDATIONS="${RECOMMENDATIONS}\"Check for subprocess leak in loops\","
fi

# Add general best practices
RECOMMENDATIONS="${RECOMMENDATIONS}\"Monitor Python process health with 'ps --forest' regularly\","
RECOMMENDATIONS="${RECOMMENDATIONS}\"Consider using multiprocessing.Pool for parallel tasks\""

RECOMMENDATIONS=$(echo "${RECOMMENDATIONS}" | sed 's/,$//')
jq ".recommendations = [${RECOMMENDATIONS}]" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

echo "✅ Python subprocess health check complete! Results: ${RESULTS_FILE}"

# Output final results
cat "${RESULTS_FILE}"