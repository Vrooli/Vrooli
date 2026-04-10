#!/bin/bash
# Shared library for investigation scripts
# Sourced by consolidated investigation scripts to reduce duplication
# NOTE: This directory is automatically skipped by ScriptService (script.go:70)

# ── Numeric helpers ──────────────────────────────────────────────────────────

# Sanitize a value to a non-negative integer (strips non-digits, defaults to 0)
sanitize_int() {
    local value="${1:-}"
    value="${value//[!0-9]/}"
    printf '%s' "${value:-0}"
}

# Convert ps etime format (e.g. "3-02:15:30", "1:05", "45") to seconds
elapsed_to_seconds() {
    local etime="$1"
    if [[ -z "${etime}" || "${etime}" == "-" ]]; then
        printf '0'
        return
    fi

    local days_part=0 rest="${etime}"
    if [[ "${etime}" == *-* ]]; then
        days_part=${etime%%-*}
        rest=${etime#*-}
    fi

    local hours=0 minutes=0 seconds=0
    IFS=':' read -r first second third <<< "${rest}"
    if [[ -n "${third}" ]]; then
        hours=${first:-0}; minutes=${second:-0}; seconds=${third:-0}
    else
        minutes=${first:-0}; seconds=${second:-0}
    fi

    printf '%d' $(( 10#${days_part:-0} * 86400 + 10#${hours:-0} * 3600 + 10#${minutes:-0} * 60 + 10#${seconds:-0} ))
}

# ── Investigation lifecycle ──────────────────────────────────────────────────

# Initialize an investigation: sets OUTPUT_DIR, RESULTS_FILE, creates dirs, sets trap
# Usage: init_investigation "script-name"
init_investigation() {
    local name="$1"
    SCRIPT_NAME="${name}"
    OUTPUT_DIR="../results/$(date +%Y%m%d_%H%M%S)_${name}"
    RESULTS_FILE="${OUTPUT_DIR}/results.json"
    mkdir -p "${OUTPUT_DIR}"

    trap '_investigation_cleanup' EXIT
}

_investigation_cleanup() {
    # Ensure partial results are still available on failure
    if [[ -f "${RESULTS_FILE:-}" ]]; then
        cat "${RESULTS_FILE}" 2>/dev/null || true
    fi
}

# Output the final results JSON to stdout (for API consumption)
finalize_investigation() {
    trap - EXIT  # disable cleanup trap since we're outputting intentionally
    if [[ -f "${RESULTS_FILE}" ]]; then
        cat "${RESULTS_FILE}"
    fi
}

# ── JSON helpers ─────────────────────────────────────────────────────────────

# Apply a jq filter to the results file (atomic tmp-file swap)
# Usage: update_json '.timestamp = "now"'
update_json() {
    local filter="$1"
    jq "${filter}" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
}

# Apply a jq filter with an --argjson value to the results file
# Usage: update_json_value '.memory_overview = $value' '{"total":1024}'
update_json_value() {
    local filter="$1"
    local json_payload="$2"
    jq --argjson value "${json_payload}" "${filter}" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
}

# Append a string to the .findings array using safe jq --arg
# Usage: add_finding "High memory usage detected"
add_finding() {
    local text="$1"
    jq --arg text "${text}" '.findings += [$text]' "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
}

# Append a string to the .recommendations array using safe jq --arg
# Usage: add_recommendation "Restart the service"
add_recommendation() {
    local text="$1"
    jq --arg text "${text}" '.recommendations += [$text]' "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
}

# ── Command helpers ──────────────────────────────────────────────────────────

# Run a command with a timeout, returning exit 0 on timeout (graceful degradation)
# Usage: safe_timeout 10 docker stats --no-stream
safe_timeout() {
    local seconds="$1"
    shift
    timeout "${seconds}" "$@" 2>/dev/null || true
}
