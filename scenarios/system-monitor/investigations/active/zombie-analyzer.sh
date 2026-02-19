#!/bin/bash
# INVESTIGATION_SCRIPT
# NAME: Zombie Process Analyzer
# DESCRIPTION: Consolidated zombie process detection with parent analysis, pattern detection, and optional container scanning
# CATEGORY: process-analysis
# TRIGGERS: zombie_processes_found, container_zombies, process_cleanup_issues, jupyter_issues, long_running_zombies
# OUTPUTS: json
# AUTHOR: claude-agent
# CREATED: 2026-02-18
# LAST_MODIFIED: 2026-02-18
# VERSION: 2.0

set -euo pipefail

source "$(dirname "$0")/lib/common.sh"

# Parse arguments
CONTAINER_MODE="${INVESTIGATION_ZOMBIE_CONTAINER:-0}"
while [[ $# -gt 0 ]]; do
    case "$1" in
        --container) CONTAINER_MODE=1; shift ;;
        *) shift ;;
    esac
done

init_investigation "zombie-analyzer"

# Initialize results JSON
jq -n --arg ts "$(date -Iseconds)" '{
    investigation: "zombie-analyzer",
    timestamp: $ts,
    summary: {},
    zombie_processes: [],
    parent_analysis: [],
    patterns: {},
    container_analysis: {},
    recommendations: [],
    findings: [],
    immediate_actions: []
}' > "${RESULTS_FILE}"

# ── Collect zombies ─────────────────────────────────────────────────────────

collect_zombies() {
    local zombie_array="[]"
    local total=0 python_count=0 jupyter_count=0 long_running=0

    while IFS= read -r line; do
        [[ -z "${line}" ]] && continue
        local pid ppid etime state cmd_raw
        pid=$(awk '{print $1}' <<< "$line")
        ppid=$(awk '{print $2}' <<< "$line")
        etime=$(awk '{print $3}' <<< "$line")
        state=$(awk '{print $4}' <<< "$line")
        [[ "${state}" != Z* ]] && continue
        cmd_raw=$(awk '{for(i=5;i<=NF;i++) printf "%s ", $i}' <<< "$line")

        ((++total))

        local secs
        secs=$(elapsed_to_seconds "${etime}")
        (( secs >= 3600 )) && ((++long_running))

        local cmd_lower
        cmd_lower=$(tr 'A-Z' 'a-z' <<< "${cmd_raw}")
        [[ "${cmd_lower}" == *python* ]] && ((++python_count))

        # Parent info
        local parent_cmd="unknown" parent_user="unknown"
        if [[ "${ppid}" != "0" ]]; then
            parent_cmd=$(ps -p "${ppid}" -o comm= 2>/dev/null || echo "unknown")
            parent_user=$(ps -p "${ppid}" -o user= 2>/dev/null || echo "unknown")
        fi
        local parent_lower
        parent_lower=$(tr 'A-Z' 'a-z' <<< "${parent_cmd}")
        [[ "${parent_lower}" == *jupyter* || "${parent_lower}" == *notebook* ]] && ((++jupyter_count))

        local obj
        obj=$(jq -n \
            --arg pid "$pid" --arg ppid "$ppid" --arg etime "$etime" \
            --arg cmd "$cmd_raw" --arg pcmd "$parent_cmd" --arg puser "$parent_user" \
            '{pid:$pid, ppid:$ppid, elapsed_time:$etime, command:$cmd,
              parent:{command:$pcmd, user:$puser}}')
        zombie_array=$(jq --argjson obj "$obj" '. += [$obj]' <<< "$zombie_array")
    done < <(ps -eo pid,ppid,etime,state,cmd 2>/dev/null | tail -n +2)

    update_json_value '.zombie_processes = $value' "$zombie_array"
    update_json ".summary.total_zombies = ${total}
        | .summary.python_zombies = ${python_count}
        | .summary.jupyter_related = ${jupyter_count}
        | .summary.long_running_zombies = ${long_running}"

    # Export for use in other functions
    TOTAL_ZOMBIES=${total}
    PYTHON_ZOMBIES=${python_count}
    JUPYTER_ZOMBIES=${jupyter_count}
    LONG_RUNNING_ZOMBIES=${long_running}
}

# ── Analyze parents ─────────────────────────────────────────────────────────

analyze_parents() {
    local parents
    parents=$(jq '[.zombie_processes[] | .ppid] | unique | map(. as $pp | {
        ppid: $pp,
        zombie_count: ([input_filename] | length),
    })' "${RESULTS_FILE}" 2>/dev/null || echo "[]")

    # Rebuild with proper aggregation
    parents=$(jq '[.zombie_processes | group_by(.ppid)[] | {
        ppid: .[0].ppid,
        zombie_count: length,
        parent_command: .[0].parent.command,
        parent_user: .[0].parent.user
    }]' "${RESULTS_FILE}")
    update_json_value '.parent_analysis = $value' "$parents"
}

# ── Detect patterns ─────────────────────────────────────────────────────────

detect_patterns() {
    local node_count=0 shell_count=0 orphan_count=0
    node_count=$(sanitize_int "$(ps -eo stat=,comm= | awk '$1 ~ /Z/ && tolower($2) ~ /node/ {c++} END {print c+0}')")
    shell_count=$(sanitize_int "$(ps -eo stat=,comm= | awk '$1 ~ /Z/ && tolower($2) ~ /(sh|bash)/ {c++} END {print c+0}')")
    orphan_count=$(sanitize_int "$(ps -eo ppid=,stat= | awk '$2 ~ /Z/ && $1 == 1 {c++} END {print c+0}')")

    local patterns
    patterns=$(jq -n \
        --argjson python "${PYTHON_ZOMBIES}" \
        --argjson node "${node_count}" \
        --argjson shell "${shell_count}" \
        --argjson orphans "${orphan_count}" \
        '{python_zombies:$python, node_zombies:$node, shell_zombies:$shell, orphaned_to_init:$orphans}')
    update_json_value '.patterns = $value' "$patterns"

    if (( orphan_count > 0 )); then
        add_finding "${orphan_count} zombies orphaned to init - original parent crashed"
    fi
}

# ── Container scanning (optional) ──────────────────────────────────────────

scan_containers() {
    [[ "${CONTAINER_MODE}" != "1" ]] && return
    if ! command -v docker &>/dev/null; then
        add_finding "Container mode requested but docker not available"
        return
    fi

    local container_json="{}"
    while IFS=' ' read -r cid cname; do
        [[ -z "${cid}" ]] && continue
        local cps count
        cps=$(safe_timeout 10 docker exec "${cid}" ps aux 2>/dev/null || true)
        if [[ -n "${cps}" ]]; then
            count=$(grep -c "<defunct>" <<< "${cps}" || true)
            count=$(sanitize_int "${count}")
            if (( count > 0 )); then
                container_json=$(jq --arg name "$cname" --argjson cnt "$count" \
                    '. + {($name): $cnt}' <<< "$container_json")
                add_finding "Container ${cname} has ${count} zombie processes"
            fi
        fi
    done < <(safe_timeout 10 docker ps --format "{{.ID}} {{.Names}}" 2>/dev/null || true)

    update_json_value '.container_analysis = $value' "$container_json"
}

# ── Recommendations ─────────────────────────────────────────────────────────

generate_recommendations() {
    local risk="low"
    if (( TOTAL_ZOMBIES > 20 )); then
        risk="high"
    elif (( TOTAL_ZOMBIES > 10 )); then
        risk="medium"
    fi
    update_json ".summary.risk_level = \"${risk}\""

    if (( TOTAL_ZOMBIES > 0 )); then
        add_recommendation "Found ${TOTAL_ZOMBIES} zombie processes - parent processes not properly reaping children"
        add_recommendation "Review parent process code for proper child reaping with wait()/waitpid()"
    fi

    if [[ "${risk}" == "high" ]]; then
        add_recommendation "Critical: High zombie count indicates systematic subprocess management issue"
        # Build immediate actions
        local top_ppid
        top_ppid=$(jq -r '.parent_analysis | sort_by(-.zombie_count) | .[0].ppid // empty' "${RESULTS_FILE}")
        if [[ -n "${top_ppid}" ]]; then
            update_json ".immediate_actions += [\"kill -HUP ${top_ppid}\", \"Review /var/log/syslog for subprocess errors\"]"
        fi
    fi

    if (( JUPYTER_ZOMBIES > 0 )); then
        add_recommendation "Jupyter/notebook server has zombie processes - kernel management issue detected"
        add_recommendation "Restart Jupyter server or implement proper subprocess handling"
    fi

    if (( PYTHON_ZOMBIES > 5 )); then
        add_recommendation "Python subprocess.Popen() calls not being properly waited - add signal.signal(signal.SIGCHLD, signal.SIG_IGN)"
    fi

    if (( LONG_RUNNING_ZOMBIES > 5 )); then
        add_recommendation "Zombies running for hours - parent process needs SIGCHLD signal handling"
    fi

    if [[ "${CONTAINER_MODE}" == "1" ]]; then
        local has_container_zombies
        has_container_zombies=$(jq '.container_analysis | length > 0' "${RESULTS_FILE}")
        if [[ "${has_container_zombies}" == "true" ]]; then
            add_recommendation "Container zombie issue - consider adding tini or dumb-init as init system"
            add_recommendation "Use --init flag with docker run or configure tini as entrypoint"
        fi
    fi
}

# ── Main ────────────────────────────────────────────────────────────────────

TOTAL_ZOMBIES=0
PYTHON_ZOMBIES=0
JUPYTER_ZOMBIES=0
LONG_RUNNING_ZOMBIES=0

collect_zombies
analyze_parents
detect_patterns
scan_containers
generate_recommendations
finalize_investigation
