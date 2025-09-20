#!/bin/bash
# INVESTIGATION_SCRIPT
# NAME: Container Zombie Analyzer
# DESCRIPTION: Advanced zombie process analysis for containerized applications with parent tracking
# CATEGORY: process-analysis
# TRIGGERS: container_zombies, jupyter_zombies, docker_process_issues
# OUTPUTS: json
# AUTHOR: claude-agent
# CREATED: 2025-09-11
# LAST_MODIFIED: 2025-09-11
# VERSION: 1.1

set -euo pipefail

# Configuration
SCRIPT_NAME="container-zombie-analyzer"
OUTPUT_DIR="../results/$(date +%Y%m%d_%H%M%S)_${SCRIPT_NAME}"
TIMEOUT_SECONDS=45

# Create output directory
mkdir -p "${OUTPUT_DIR}"

# Initialize results
RESULTS_FILE="${OUTPUT_DIR}/results.json"
cat > "${RESULTS_FILE}" << 'EOF'
{
  "investigation": "container-zombie-analyzer",
  "timestamp": "",
  "zombie_summary": {},
  "container_analysis": {},
  "parent_process_tree": [],
  "root_cause_analysis": {},
  "recommendations": []
}
EOF

# Update timestamp
jq ".timestamp = \"$(date -Iseconds)\"" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

echo "🐳 Starting Container Zombie Analysis..."

# Get zombie process information with complete parent chain
echo "🧟 Detecting zombie processes..."
ZOMBIE_DATA=$(ps aux | grep "<defunct>" | grep -v grep | while read line; do
    PID=$(echo "$line" | awk '{print $2}')
    USER=$(echo "$line" | awk '{print $1}')
    CMD=$(echo "$line" | awk '{print $11}' | sed 's/\[//g' | sed 's/\]//g')
    
    # Get parent info from /proc
    if [[ -e /proc/${PID}/stat ]]; then
        PARENT_PID=$(awk '{print $4}' /proc/${PID}/stat 2>/dev/null || echo "unknown")
        PARENT_CMD=$(ps -p ${PARENT_PID} -o comm= 2>/dev/null || echo "unknown")
        PARENT_ARGS=$(ps -p ${PARENT_PID} -o args= 2>/dev/null || echo "unknown")
        
        # Check if parent is in container
        IS_CONTAINER="false"
        if [[ -e /proc/${PARENT_PID}/cgroup ]]; then
            if grep -q "docker\|containerd" /proc/${PARENT_PID}/cgroup 2>/dev/null; then
                IS_CONTAINER="true"
            fi
        fi
        
        echo "{\"zombie_pid\":${PID},\"user\":\"${USER}\",\"command\":\"${CMD}\",\"parent_pid\":${PARENT_PID},\"parent_cmd\":\"${PARENT_CMD}\",\"parent_args\":\"${PARENT_ARGS//\"/\\\"}\",\"in_container\":${IS_CONTAINER}},"
    fi
done | sed 's/,$//')

ZOMBIE_COUNT=$(ps aux | grep -c "<defunct>" || echo "0")

# Analyze container-specific issues
echo "🐳 Analyzing container processes..."
CONTAINER_DATA="{"

# Check Docker containers with zombies
if command -v docker &> /dev/null; then
    CONTAINERS_WITH_ISSUES=$(docker ps --format "{{.ID}} {{.Names}}" | while read cid cname; do
        # Check for zombies in container
        CONTAINER_ZOMBIES=$(docker exec ${cid} ps aux 2>/dev/null | grep -c "<defunct>" || echo "0")
        if [[ ${CONTAINER_ZOMBIES} -gt 0 ]]; then
            echo "\"${cname}\":${CONTAINER_ZOMBIES},"
        fi
    done | sed 's/,$//')
    
    if [[ -n "${CONTAINERS_WITH_ISSUES}" ]]; then
        CONTAINER_DATA="${CONTAINER_DATA}\"containers_with_zombies\":{${CONTAINERS_WITH_ISSUES}},"
    fi
fi

# Identify specific Jupyter/notebook issues
JUPYTER_PARENT=$(ps aux | grep -E "(jupyter|notebook)" | grep -v grep | head -1 | awk '{print $2}' || echo "")
if [[ -n "${JUPYTER_PARENT}" ]]; then
    JUPYTER_ZOMBIES=$(ps --ppid ${JUPYTER_PARENT} | grep -c "<defunct>" || echo "0")
    CONTAINER_DATA="${CONTAINER_DATA}\"jupyter_parent_pid\":${JUPYTER_PARENT},\"jupyter_zombies\":${JUPYTER_ZOMBIES},"
fi

CONTAINER_DATA=$(echo "${CONTAINER_DATA}" | sed 's/,$//')
CONTAINER_DATA="${CONTAINER_DATA}}"

# Build parent process tree for zombies
echo "🌳 Building process genealogy tree..."
PROCESS_TREE=""
for zombie_pid in $(ps aux | grep "<defunct>" | awk '{print $2}'); do
    CURRENT_PID=${zombie_pid}
    TREE_PATH="${zombie_pid}"
    
    # Trace up to 5 levels of parents
    for i in {1..5}; do
        if [[ -e /proc/${CURRENT_PID}/stat ]]; then
            PARENT_PID=$(awk '{print $4}' /proc/${CURRENT_PID}/stat 2>/dev/null || break)
            if [[ "${PARENT_PID}" == "1" ]] || [[ "${PARENT_PID}" == "0" ]]; then
                TREE_PATH="${TREE_PATH} -> init/systemd"
                break
            fi
            PARENT_NAME=$(ps -p ${PARENT_PID} -o comm= 2>/dev/null || echo "pid${PARENT_PID}")
            TREE_PATH="${TREE_PATH} -> ${PARENT_NAME}(${PARENT_PID})"
            CURRENT_PID=${PARENT_PID}
        else
            break
        fi
    done
    
    PROCESS_TREE="${PROCESS_TREE}\"${TREE_PATH}\","
done
PROCESS_TREE=$(echo "${PROCESS_TREE}" | sed 's/,$//')

# Root cause analysis
echo "🔍 Performing root cause analysis..."
ROOT_CAUSE_DATA="{"

# Analyze patterns
PYTHON_ZOMBIES=$(ps aux | grep "<defunct>" | grep -c "python" || echo "0")
NODE_ZOMBIES=$(ps aux | grep "<defunct>" | grep -c "node" || echo "0")
SHELL_ZOMBIES=$(ps aux | grep "<defunct>" | grep -c -E "sh|bash" || echo "0")

ROOT_CAUSE_DATA="${ROOT_CAUSE_DATA}\"python_zombies\":${PYTHON_ZOMBIES},"
ROOT_CAUSE_DATA="${ROOT_CAUSE_DATA}\"node_zombies\":${NODE_ZOMBIES},"
ROOT_CAUSE_DATA="${ROOT_CAUSE_DATA}\"shell_zombies\":${SHELL_ZOMBIES},"

# Check for specific issues
if [[ ${PYTHON_ZOMBIES} -gt 10 ]]; then
    ROOT_CAUSE_DATA="${ROOT_CAUSE_DATA}\"likely_cause\":\"Python subprocess management issue\","
    ROOT_CAUSE_DATA="${ROOT_CAUSE_DATA}\"severity\":\"high\","
elif [[ ${ZOMBIE_COUNT} -gt 20 ]]; then
    ROOT_CAUSE_DATA="${ROOT_CAUSE_DATA}\"likely_cause\":\"Systematic parent process issue\","
    ROOT_CAUSE_DATA="${ROOT_CAUSE_DATA}\"severity\":\"critical\","
elif [[ ${ZOMBIE_COUNT} -gt 5 ]]; then
    ROOT_CAUSE_DATA="${ROOT_CAUSE_DATA}\"likely_cause\":\"Moderate zombie accumulation\","
    ROOT_CAUSE_DATA="${ROOT_CAUSE_DATA}\"severity\":\"medium\","
else
    ROOT_CAUSE_DATA="${ROOT_CAUSE_DATA}\"likely_cause\":\"Minor zombie presence\","
    ROOT_CAUSE_DATA="${ROOT_CAUSE_DATA}\"severity\":\"low\","
fi

ROOT_CAUSE_DATA=$(echo "${ROOT_CAUSE_DATA}" | sed 's/,$//')
ROOT_CAUSE_DATA="${ROOT_CAUSE_DATA}}"

# Generate targeted recommendations
echo "💡 Generating targeted recommendations..."
RECOMMENDATIONS=""

if [[ ${ZOMBIE_COUNT} -gt 0 ]]; then
    RECOMMENDATIONS="${RECOMMENDATIONS}\"Total of ${ZOMBIE_COUNT} zombie processes detected requiring attention\","
fi

if [[ -n "${JUPYTER_PARENT}" ]] && [[ ${JUPYTER_ZOMBIES} -gt 0 ]]; then
    RECOMMENDATIONS="${RECOMMENDATIONS}\"Jupyter notebook (PID ${JUPYTER_PARENT}) has ${JUPYTER_ZOMBIES} zombie children - restart the container or implement proper subprocess handling\","
    RECOMMENDATIONS="${RECOMMENDATIONS}\"For containerized Jupyter: docker restart sagemath-main or similar container\","
    RECOMMENDATIONS="${RECOMMENDATIONS}\"Configure Jupyter kernel to properly handle subprocess termination\","
fi

if [[ ${PYTHON_ZOMBIES} -gt 10 ]]; then
    RECOMMENDATIONS="${RECOMMENDATIONS}\"Critical: ${PYTHON_ZOMBIES} Python zombies indicate subprocess.Popen() without wait() or communicate()\","
    RECOMMENDATIONS="${RECOMMENDATIONS}\"Add signal handler: signal.signal(signal.SIGCHLD, signal.SIG_IGN) in Python parent process\","
    RECOMMENDATIONS="${RECOMMENDATIONS}\"Use context managers: with subprocess.Popen() as proc: proc.wait()\","
fi

if [[ -n "${CONTAINERS_WITH_ISSUES}" ]]; then
    RECOMMENDATIONS="${RECOMMENDATIONS}\"Container process management issue detected - consider adding init system like tini or dumb-init\","
    RECOMMENDATIONS="${RECOMMENDATIONS}\"Update Dockerfile: Add --init flag to docker run or use tini as entrypoint\","
fi

# Add general best practices
RECOMMENDATIONS="${RECOMMENDATIONS}\"Monitor with: watch 'ps aux | grep defunct | wc -l' to track zombie count\","
RECOMMENDATIONS="${RECOMMENDATIONS}\"Investigate parent processes that aren't reaping children properly\","
RECOMMENDATIONS="${RECOMMENDATIONS}\"Consider implementing a zombie reaper service if issue persists\""

RECOMMENDATIONS=$(echo "${RECOMMENDATIONS}" | sed 's/,$//')

# Build final JSON
jq ".zombie_summary = {
  \"total_zombies\": ${ZOMBIE_COUNT},
  \"detection_time\": \"$(date -Iseconds)\"
}" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

if [[ -n "${ZOMBIE_DATA}" ]]; then
    echo "[${ZOMBIE_DATA}]" | jq '.' > "${OUTPUT_DIR}/zombie_data.json"
    jq ".zombie_processes = $(cat ${OUTPUT_DIR}/zombie_data.json)" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
fi

echo "${CONTAINER_DATA}" | jq '.' > "${OUTPUT_DIR}/container_data.json"
jq ".container_analysis = $(cat ${OUTPUT_DIR}/container_data.json)" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

if [[ -n "${PROCESS_TREE}" ]]; then
    jq ".parent_process_tree = [${PROCESS_TREE}]" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
fi

echo "${ROOT_CAUSE_DATA}" | jq '.' > "${OUTPUT_DIR}/root_cause.json"
jq ".root_cause_analysis = $(cat ${OUTPUT_DIR}/root_cause.json)" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

if [[ -n "${RECOMMENDATIONS}" ]]; then
    jq ".recommendations = [${RECOMMENDATIONS}]" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
fi

echo "✅ Container zombie analysis complete! Results: ${RESULTS_FILE}"

# Output final results for API consumption
cat "${RESULTS_FILE}"