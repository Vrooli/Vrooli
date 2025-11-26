#!/usr/bin/env bash
# =============================================================================
# Resource Health Checks
# PostgreSQL, Redis, Qdrant, Ollama, Searxng, Browserless, etc.
# =============================================================================

check_resource() {
    local name="$1"
    local info running healthy total_instances healthy_instances

    if ! run_with_timeout_capture info vrooli resource status "$name" --json; then
        log "WARN" "Resource '$name' status check failed; attempting restart"
        restart_resource "$name"
        return
    fi

    running=$(jq -r '.running // false' <<<"$info")
    healthy=$(jq -r '.healthy // false' <<<"$info")
    total_instances=$(jq -r '.total_instances // empty' <<<"$info")
    healthy_instances=$(jq -r '.healthy_instances // empty' <<<"$info")

    if [[ "$healthy" != "true" && -n "$total_instances" && -n "$healthy_instances" ]]; then
        if (( healthy_instances > 0 )); then
            log "WARN" "Resource '$name' degraded but $healthy_instances/$total_instances instances healthy; skipping restart"
            healthy="true"
        fi
    fi

    if [[ "$running" != "true" || "$healthy" != "true" ]]; then
        restart_resource "$name" "$running" "$healthy"
    fi
}

restart_resource() {
    local name="$1"
    local running="${2:-unknown}" healthy="${3:-unknown}"

    log "INFO" "Restarting resource '$name' (running=$running healthy=$healthy)"

    if ! run_with_timeout_stream vrooli resource stop "$name" >>"$LOG_FILE" 2>&1; then
        log "DEBUG" "Stop command for '$name' returned non-zero, continuing"
    fi

    if ! run_with_timeout_stream vrooli resource start "$name" >>"$LOG_FILE" 2>&1; then
        log "ERROR" "Failed to start resource '$name'"
    else
        log "INFO" "Resource '$name' restart requested"
    fi
}
