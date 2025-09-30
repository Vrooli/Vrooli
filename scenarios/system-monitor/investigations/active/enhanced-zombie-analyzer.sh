#!/bin/bash
# INVESTIGATION_SCRIPT
# NAME: Enhanced Zombie Analyzer
# DESCRIPTION: Advanced zombie process detection with parent analysis and timeline tracking
# CATEGORY: process-analysis
# TRIGGERS: zombie_processes_found, jupyter_issues, long_running_zombies
# OUTPUTS: json
# AUTHOR: claude-agent
# CREATED: 2025-09-11
# LAST_MODIFIED: 2025-09-11
# VERSION: 1.1

set -euo pipefail

# Convert elapsed time (ps etime format) to seconds
elapsed_to_seconds() {
    local etime="$1"
    local days_part rest hours minutes seconds

    if [[ -z "${etime}" || "${etime}" == "-" ]]; then
        printf '0'
        return
    fi

    if [[ "${etime}" == *-* ]]; then
        days_part=${etime%%-*}
        rest=${etime#*-}
    else
        days_part=0
        rest=${etime}
    fi

    IFS=':' read -r first second third <<< "${rest}"

    if [[ -n "${third}" ]]; then
        hours=${first:-0}
        minutes=${second:-0}
        seconds=${third:-0}
    else
        hours=0
        minutes=${first:-0}
        seconds=${second:-0}
    fi

    days_part=${days_part:-0}

    # Use 10# to avoid octal interpretation for values with leading zeros
    printf '%d' $((10#${days_part} * 86400 + 10#${hours} * 3600 + 10#${minutes} * 60 + 10#${seconds}))
}

# Configuration
SCRIPT_NAME="enhanced-zombie-analyzer"
OUTPUT_DIR="../results/$(date +%Y%m%d_%H%M%S)_${SCRIPT_NAME}"
TIMEOUT_SECONDS=45

# Create output directory
mkdir -p "${OUTPUT_DIR}"

# Initialize results
RESULTS_FILE="${OUTPUT_DIR}/results.json"

echo "🧟 Starting Enhanced Zombie Analysis..."

# Function to safely get process info
get_process_info() {
    local pid=$1
    local field=$2
    if [[ -e /proc/${pid}/status ]]; then
        grep "^${field}:" /proc/${pid}/status 2>/dev/null | awk '{print $2}' || echo "unknown"
    else
        echo "unknown"
    fi
}

# Collect zombie data
echo "💀 Detecting zombie processes..."
ZOMBIES=$(ps -eo pid,ppid,etime,state,cmd | grep " Z " | grep -v grep || true)
ZOMBIE_COUNT=0

# Build JSON array of zombies with detailed info
ZOMBIE_ARRAY="[]"
JUPYTER_COUNT=0
PYTHON_COUNT=0
LONG_RUNNING_COUNT=0
if [[ -n "${ZOMBIES}" ]]; then
    echo "🔍 Analyzing ${ZOMBIE_COUNT} zombie processes..."
    while IFS= read -r line; do
        if [[ -z "$line" ]]; then continue; fi
        ((++ZOMBIE_COUNT))
        
        PID=$(echo "$line" | awk '{print $1}')
        PARENT_PID=$(echo "$line" | awk '{print $2}')
        ETIME=$(echo "$line" | awk '{print $3}')
        CMD_RAW=$(echo "$line" | awk '{for(i=5;i<=NF;i++) printf "%s ", $i; print ""}')
        CMD=$(echo "${CMD_RAW}" | sed 's/["\\]/\\&/g')
        CMD_LOWER=$(echo "${CMD_RAW}" | tr 'A-Z' 'a-z')
        ETIME_SECONDS=$(elapsed_to_seconds "${ETIME}")
        if (( ETIME_SECONDS >= 3600 )); then
            ((++LONG_RUNNING_COUNT))
        fi
        if [[ "${CMD_LOWER}" == *python* ]]; then
            ((++PYTHON_COUNT))
        fi
        
        # Get parent process info
        PARENT_CMD="unknown"
        PARENT_USER="unknown"
        PARENT_ETIME="unknown"
        if [[ -n "${PARENT_PID}" ]] && [[ "${PARENT_PID}" != "0" ]]; then
            PARENT_CMD=$(ps -p ${PARENT_PID} -o comm= 2>/dev/null || echo "unknown")
            PARENT_USER=$(ps -p ${PARENT_PID} -o user= 2>/dev/null || echo "unknown")
            PARENT_ETIME=$(ps -p ${PARENT_PID} -o etime= 2>/dev/null || echo "unknown")
        fi
        PARENT_CMD_LOWER=$(echo "${PARENT_CMD}" | tr 'A-Z' 'a-z')
        if [[ "${PARENT_CMD_LOWER}" == *jupyter* || "${PARENT_CMD_LOWER}" == *notebook* ]]; then
            ((++JUPYTER_COUNT))
        fi
        
        # Create JSON object for this zombie
        ZOMBIE_OBJ=$(jq -n \
            --arg pid "$PID" \
            --arg ppid "$PARENT_PID" \
            --arg etime "$ETIME" \
            --arg cmd "$CMD" \
            --arg parent_cmd "$PARENT_CMD" \
            --arg parent_user "$PARENT_USER" \
            --arg parent_etime "$PARENT_ETIME" \
            '{
                pid: $pid,
                ppid: $ppid,
                elapsed_time: $etime,
                command: $cmd,
                parent: {
                    command: $parent_cmd,
                    user: $parent_user,
                    elapsed_time: $parent_etime
                }
            }')
        
        ZOMBIE_ARRAY=$(echo "$ZOMBIE_ARRAY" | jq ". += [$ZOMBIE_OBJ]")
    done <<< "$ZOMBIES"
fi

# Analyze parent processes that have zombies
echo "👨‍👧‍👦 Analyzing parent processes..."
PARENT_ANALYSIS=$(echo "$ZOMBIE_ARRAY" | jq '. as $zombies | [
    .[] | .ppid
] | unique | map(. as $ppid | {
    ppid: $ppid,
    zombie_count: ([$zombies[] | select(.ppid == $ppid)] | length),
    oldest_zombie: ([$zombies[] | select(.ppid == $ppid)] | min_by(.elapsed_time) | .elapsed_time),
    parent_command: ([$zombies[] | select(.ppid == $ppid)] | .[0].parent.command)
})')

# Check for specific problematic patterns
echo "🔍 Detecting patterns..."
JUPYTER_ZOMBIES=${JUPYTER_COUNT}
PYTHON_ZOMBIES=${PYTHON_COUNT}
LONG_RUNNING=${LONG_RUNNING_COUNT}

# Generate recommendations
echo "💡 Building recommendations..."
RECOMMENDATIONS_LIST=()

if [[ ${ZOMBIE_COUNT} -gt 0 ]]; then
    RECOMMENDATIONS_LIST+=("Found ${ZOMBIE_COUNT} zombie processes that need parent process attention")
fi

if [[ ${ZOMBIE_COUNT} -gt 20 ]]; then
    RECOMMENDATIONS_LIST+=("Critical: High zombie count indicates serious subprocess management issue")
    TOP_PARENT_PID=$(echo "$PARENT_ANALYSIS" | jq -r '.[0].ppid // empty')
    TOP_PARENT_COUNT=$(echo "$PARENT_ANALYSIS" | jq -r '.[0].zombie_count // empty')
    if [[ -n "${TOP_PARENT_PID}" && -n "${TOP_PARENT_COUNT}" ]]; then
        RECOMMENDATIONS_LIST+=("Parent process PID ${TOP_PARENT_PID} has ${TOP_PARENT_COUNT} zombie children")
    fi
fi

if [[ ${JUPYTER_ZOMBIES} -gt 0 ]]; then
    RECOMMENDATIONS_LIST+=("Jupyter/notebook server has zombie processes - kernel management issue detected")
    RECOMMENDATIONS_LIST+=("Restart Jupyter server: systemctl restart jupyter or kill -HUP <jupyter_pid>")
fi

if [[ ${PYTHON_ZOMBIES} -gt 10 ]]; then
    RECOMMENDATIONS_LIST+=("Python subprocess.Popen() calls not being properly waited/communicated")
    RECOMMENDATIONS_LIST+=("Add signal.signal(signal.SIGCHLD, signal.SIG_IGN) to parent Python script")
fi

if [[ ${LONG_RUNNING} -gt 5 ]]; then
    RECOMMENDATIONS_LIST+=("Zombies running for hours - parent process not handling SIGCHLD signals")
    RECOMMENDATIONS_LIST+=("Parent process needs wait() or waitpid() implementation")
fi

# Add specific fix for Jupyter
if echo "$PARENT_ANALYSIS" | jq -e '.[0].parent_command | contains("sage-notebook")' > /dev/null 2>&1; then
    RECOMMENDATIONS_LIST+=("SageMath Jupyter notebook detected - known issue with subprocess handling")
    RECOMMENDATIONS_LIST+=("Fix: Restart sage-notebook or implement SIGCHLD handler in sage wrapper")
fi

# Build final JSON result
if (( ${#RECOMMENDATIONS_LIST[@]} > 0 )); then
    RECOMMENDATIONS_JSON=$(printf '%s\n' "${RECOMMENDATIONS_LIST[@]}" | jq -R -s 'split("\n")[:-1]')
else
    RECOMMENDATIONS_JSON='[]'
fi

RESULT=$(jq -n \
    --argjson zombies "$ZOMBIE_ARRAY" \
    --argjson parents "$PARENT_ANALYSIS" \
    --argjson recommendations "$RECOMMENDATIONS_JSON" \
    --arg timestamp "$(date -Iseconds)" \
    --argjson zombie_count "${ZOMBIE_COUNT}" \
    --argjson jupyter_zombies "${JUPYTER_ZOMBIES}" \
    --argjson python_zombies "${PYTHON_ZOMBIES}" \
    --argjson long_running "${LONG_RUNNING}" \
    '{
        investigation: "enhanced-zombie-analyzer",
        timestamp: $timestamp,
        summary: {
            total_zombies: $zombie_count,
            jupyter_related: $jupyter_zombies,
            python_zombies: $python_zombies,
            long_running_zombies: $long_running,
            risk_level: (if $zombie_count > 20 then "high" elif $zombie_count > 10 then "medium" else "low" end)
        },
        zombies: $zombies,
        parent_analysis: $parents,
        recommendations: $recommendations,
        immediate_actions: (if $zombie_count > 20 then [
            "kill -HUP " + ($parents[0].ppid // "unknown"),
            "systemctl restart jupyter",
            "Review /var/log/syslog for subprocess errors"
        ] else [] end)
    }')

# Save and output results
echo "$RESULT" > "${RESULTS_FILE}"
echo "✅ Enhanced zombie analysis complete! Results saved to: ${RESULTS_FILE}"

# Output for API consumption
echo "$RESULT"
