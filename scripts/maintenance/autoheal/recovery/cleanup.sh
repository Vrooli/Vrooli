#!/usr/bin/env bash
# =============================================================================
# Recovery & Cleanup Functions
# Zombie process reaping and port cleanup for scenarios
# =============================================================================

# =============================================================================
# ZOMBIE PROCESS CLEANUP
# =============================================================================

cleanup_zombie_processes() {
    log "INFO" "Cleaning up zombie processes"

    # Get parent PIDs of zombie processes
    local zombie_parents
    zombie_parents=$(ps -eo ppid,stat | awk '$2 ~ /Z/ {print $1}' | sort -u)

    if [[ -z "$zombie_parents" ]]; then
        log "INFO" "No zombie parent processes found"
        return 0
    fi

    log "INFO" "Sending SIGCHLD to parent processes: $zombie_parents"

    for ppid in $zombie_parents; do
        if [[ "$ppid" -gt 1 ]]; then  # Don't signal init
            kill -CHLD "$ppid" 2>/dev/null || true
        fi
    done

    sleep 2

    # Check if zombies were reaped
    local remaining
    remaining=$(ps aux | awk '$8 ~ /Z/ {print}' | wc -l)

    log "INFO" "Zombie processes after cleanup: $remaining"

    if [[ "$remaining" -lt "$ZOMBIE_THRESHOLD" ]]; then
        log "INFO" "Zombie cleanup successful"
        return 0
    else
        log "WARN" "Zombie cleanup incomplete - $remaining zombies remain"
        return 1
    fi
}

# =============================================================================
# PORT CLEANUP FOR SCENARIOS
# =============================================================================

kill_port_listener() {
    local name="$1" port="$2"
    if [[ -z "$port" || ! "$port" =~ ^[0-9]+$ ]]; then
        return 0
    fi

    local pids pid
    if ! pids=$(lsof -t -iTCP:"$port" -sTCP:LISTEN 2>/dev/null | sort -u); then
        return 0
    fi
    if [[ -z "$pids" ]]; then
        return 0
    fi

    log "WARN" "Scenario '$name' port $port still in use; forcing shutdown of listener(s): $pids"
    for pid in $pids; do
        kill -TERM "$pid" 2>/dev/null || true
    done
    sleep 2
    for pid in $pids; do
        if kill -0 "$pid" 2>/dev/null; then
            kill -KILL "$pid" 2>/dev/null || true
        fi
    done
    return 0
}

cleanup_scenario_ports() {
    local name="$1"
    local -a ports
    mapfile -t ports < <(collect_scenario_ports "$name" 2>/dev/null | sort -n)
    if [[ "${#ports[@]}" -eq 0 ]]; then
        return 0
    fi

    local port
    for port in "${ports[@]}"; do
        kill_port_listener "$name" "$port"
    done

    local proc_dir="${HOME}/.vrooli/processes/scenarios/${name}"
    if [[ -d "$proc_dir" ]]; then
        rm -f "$proc_dir"/*.pid "$proc_dir"/*.json 2>/dev/null || true
    fi
}
