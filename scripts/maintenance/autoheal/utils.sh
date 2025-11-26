#!/usr/bin/env bash
# =============================================================================
# Autoheal Utilities
# Logging, locking, and common helper functions
# =============================================================================

# =============================================================================
# LOGGING UTILITIES
# =============================================================================

log() {
    local level="$1"
    shift
    printf '%s :: [%s] %s\n' "$(date -Is)" "$level" "$*" | tee -a "$LOG_FILE" >/dev/null
}

log_stream() {
    local level="$1"
    while IFS= read -r line; do
        [[ -z "$line" ]] && continue
        log "$level" "$line"
    done
    return 0
}

fatal() {
    log "ERROR" "$*"
    exit 1
}

log_section() {
    log "INFO" "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    log "INFO" "$*"
    log "INFO" "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
}

# =============================================================================
# PREREQUISITES & SETUP
# =============================================================================

ensure_prereqs() {
    command -v vrooli >/dev/null 2>&1 || fatal "vrooli CLI not found in PATH"
    command -v jq >/dev/null 2>&1 || fatal "jq is required but not found in PATH"
    command -v curl >/dev/null 2>&1 || fatal "curl is required but not found in PATH"
    command -v lsof >/dev/null 2>&1 || fatal "lsof is required but not found in PATH"

    if command -v timeout >/dev/null 2>&1; then
        TIMEOUT_BIN="timeout"
    else
        TIMEOUT_BIN=""
        log "WARN" "timeout command not available; commands will run without execution limits"
    fi

    local target_log="$LOG_FILE"
    local fallback_log="${HOME}/.vrooli/logs/vrooli-autoheal.log"

    if ! mkdir -p "$(dirname "$target_log")" 2>/dev/null || ! touch "$target_log" 2>/dev/null; then
        LOG_FILE="$fallback_log"
        mkdir -p "$(dirname "$LOG_FILE")" 2>/dev/null || true
        if ! touch "$LOG_FILE" 2>/dev/null; then
            printf '%s\n' "ERROR: Unable to create log file at '$target_log' or fallback '$LOG_FILE'" >&2
            exit 1
        fi
        printf '%s :: [WARN] Falling back to log file %s (unable to write %s)\n' "$(date -Is)" "$LOG_FILE" "$target_log" >>"$LOG_FILE"
    else
        LOG_FILE="$target_log"
    fi
}

# =============================================================================
# LOCK FILE MANAGEMENT
# =============================================================================

acquire_lock() {
    local lock_dir
    lock_dir="$(dirname "$LOCK_FILE")"
    if ! mkdir -p "$lock_dir" 2>/dev/null; then
        log "WARN" "Unable to create lock directory $lock_dir; falling back to /tmp"
        LOCK_FILE=/tmp/vrooli-autoheal.lock
        lock_dir="$(dirname "$LOCK_FILE")"
        mkdir -p "$lock_dir" 2>/dev/null || true
    fi

    if command -v flock >/dev/null 2>&1; then
        exec 9>"$LOCK_FILE"
        if ! flock -n 9; then
            log "DEBUG" "Another autoheal run is already in progress"
            return 1
        fi
        LOCK_METHOD="flock"
    else
        if ( set -o noclobber; : >"$LOCK_FILE" ) 2>/dev/null; then
            LOCK_METHOD="noclobber"
            trap cleanup_lock EXIT
        else
            log "DEBUG" "Another autoheal run is already in progress"
            return 1
        fi
    fi
    return 0
}

cleanup_lock() {
    if [[ "${LOCK_METHOD:-}" == "noclobber" ]]; then
        rm -f "$LOCK_FILE"
    fi
}

# =============================================================================
# HELPER UTILITIES
# =============================================================================

trim_token() {
    printf '%s' "$1" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//'
}

parse_list() {
    local raw="$1"
    local IFS=',' token trimmed
    for token in $raw; do
        trimmed=$(trim_token "$token")
        [[ -n "$trimmed" ]] && printf '%s\n' "$trimmed"
    done
}

run_with_timeout_capture() {
    local __output_var="$1"
    shift
    local output
    local rc

    if [[ -n "${TIMEOUT_BIN}" ]]; then
        if output=$("$TIMEOUT_BIN" "$CMD_TIMEOUT" "$@" 2>/dev/null); then
            rc=0
        else
            rc=$?
        fi
    else
        if output=$("$@" 2>/dev/null); then
            rc=0
        else
            rc=$?
        fi
    fi

    printf -v "$__output_var" '%s' "$output"
    return $rc
}

run_with_timeout_stream() {
    if [[ -n "${TIMEOUT_BIN}" ]]; then
        "$TIMEOUT_BIN" "$CMD_TIMEOUT" "$@"
    else
        "$@"
    fi
}
