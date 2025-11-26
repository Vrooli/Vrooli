#!/usr/bin/env bash
# =============================================================================
# Vrooli API Health Check
# Monitors the main Vrooli orchestration API
# =============================================================================

check_api_health() {
    if [[ -z "$API_URL" ]]; then
        return 0
    fi

    local api_output=""
    if ! run_with_timeout_capture api_output curl --max-time "$API_TIMEOUT" --silent --show-error "$API_URL"; then
        log "ERROR" "Vrooli API unreachable at $API_URL"
        if [[ -n "$API_RECOVERY" ]]; then
            log "INFO" "Running API recovery command"
            if ! run_with_timeout_stream bash -c "$API_RECOVERY" >>"$LOG_FILE" 2>&1; then
                log "ERROR" "API recovery command failed"
            else
                log "INFO" "API recovery command executed"
            fi
        fi
        return 1
    fi

    if [[ -n "$api_output" ]]; then
        local status
        status=$(jq -r '.status // "unknown"' <<<"$api_output" 2>/dev/null || echo "unknown")
        if [[ "$status" == "failed" || "$status" == "unknown" ]]; then
            log "ERROR" "Vrooli API health check failed: status=$status"
            printf '%s\n' "$api_output" >>"$LOG_FILE"
            if [[ -n "$API_RECOVERY" ]]; then
                log "INFO" "Running API recovery command"
                if ! run_with_timeout_stream bash -c "$API_RECOVERY" >>"$LOG_FILE" 2>&1; then
                    log "ERROR" "API recovery command failed"
                else
                    log "INFO" "API recovery command executed"
                fi
            fi
            return 1
        elif [[ "$status" == "degraded" ]]; then
            log "WARN" "Vrooli API is degraded but responsive (status=$status)"
            return 0
        else
            log "DEBUG" "Vrooli API health check passed (status=$status)"
            return 0
        fi
    fi
    return 0
}
