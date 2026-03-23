#!/usr/bin/env bash
# Speaker Verification - Status Reporting

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# Source status framework if available
if [[ -f "${APP_ROOT}/scripts/resources/lib/status-args.sh" ]]; then
    # shellcheck disable=SC1091
    source "${APP_ROOT}/scripts/resources/lib/status-args.sh"
fi

# Helper: emit key-value pair as two lines (framework expects alternating key/value)
_sv_kv() {
    echo "$1"
    echo "$2"
}

#######################################
# Collect status data
# Arguments: --fast (optional, skip expensive checks)
# Returns: alternating key/value lines on stdout
#######################################
speaker_verification::status::collect_data() {
    local fast="false"
    [[ "${1:-}" == "--fast" ]] && fast="true"

    _sv_kv "name" "speaker-verification"
    _sv_kv "category" "ai-ml"
    _sv_kv "description" "Speaker verification using NeMo TitaNet"
    _sv_kv "container_name" "${SPEAKER_VERIFICATION_CONTAINER_NAME}"
    _sv_kv "port" "${SPEAKER_VERIFICATION_PORT}"
    _sv_kv "base_url" "${SPEAKER_VERIFICATION_BASE_URL}"
    _sv_kv "data_dir" "${SPEAKER_VERIFICATION_DATA_DIR}"
    _sv_kv "profiles_dir" "${SPEAKER_VERIFICATION_PROFILES_DIR}"
    _sv_kv "device" "${SPEAKER_VERIFICATION_DEVICE}"
    _sv_kv "model" "${SPEAKER_VERIFICATION_MODEL}"
    _sv_kv "threshold" "${SPEAKER_VERIFICATION_DEFAULT_THRESHOLD}"
    _sv_kv "gpu_enabled" "${SPEAKER_VERIFICATION_GPU_ENABLED}"

    # Check installation
    if common::container_exists; then
        _sv_kv "installed" "true"
    else
        _sv_kv "installed" "false"
        _sv_kv "running" "false"
        _sv_kv "healthy" "false"
        return
    fi

    # Check running
    if common::is_running; then
        _sv_kv "running" "true"
    else
        _sv_kv "running" "false"
        _sv_kv "healthy" "false"
        return
    fi

    # Check health
    if speaker_verification::is_healthy; then
        _sv_kv "healthy" "true"
        if speaker_verification::is_ready; then
            _sv_kv "health_message" "Healthy and ready"
        else
            _sv_kv "health_message" "Alive but model not yet loaded"
        fi
    else
        _sv_kv "healthy" "false"
        _sv_kv "health_message" "Health check failed"
    fi

    # Container status
    local container_status
    container_status=$(docker inspect --format '{{.State.Status}}' "$SPEAKER_VERIFICATION_CONTAINER_NAME" 2>/dev/null || echo "unknown")
    _sv_kv "container_status" "${container_status}"

    # Skip expensive checks in fast mode
    if [[ "$fast" == "true" ]]; then
        return
    fi

    # Profile count
    local profile_count=0
    if [[ -d "${SPEAKER_VERIFICATION_PROFILES_DIR}" ]]; then
        profile_count=$(find "${SPEAKER_VERIFICATION_PROFILES_DIR}" -name "profile.json" 2>/dev/null | wc -l)
    fi
    _sv_kv "profile_count" "${profile_count}"

    # Data size
    local data_size="0"
    if [[ -d "${SPEAKER_VERIFICATION_DATA_DIR}" ]]; then
        data_size=$(du -sh "${SPEAKER_VERIFICATION_DATA_DIR}" 2>/dev/null | cut -f1)
    fi
    _sv_kv "data_size" "${data_size}"

    # Container resource usage
    if common::is_running; then
        local stats
        stats=$(docker stats --no-stream --format '{{.CPUPerc}}|{{.MemUsage}}' "$SPEAKER_VERIFICATION_CONTAINER_NAME" 2>/dev/null || echo "|")
        _sv_kv "cpu_usage" "$(echo "$stats" | cut -d'|' -f1)"
        _sv_kv "memory_usage" "$(echo "$stats" | cut -d'|' -f2)"
    fi

    # API info (if ready)
    if speaker_verification::is_ready; then
        local info
        if info=$(timeout 5 curl -sf "${SPEAKER_VERIFICATION_BASE_URL}/v1/info" 2>/dev/null); then
            local backend model_name device_in_use
            backend=$(echo "$info" | jq -r '.backend // "unknown"' 2>/dev/null)
            model_name=$(echo "$info" | jq -r '.model // "unknown"' 2>/dev/null)
            device_in_use=$(echo "$info" | jq -r '.device // "unknown"' 2>/dev/null)
            _sv_kv "active_backend" "${backend}"
            _sv_kv "active_model" "${model_name}"
            _sv_kv "active_device" "${device_in_use}"
        fi
    fi
}
export -f speaker_verification::status::collect_data

#######################################
# Display status as text
# Arguments: alternating key/value pairs
#######################################
speaker_verification::status::display_text() {
    # Build associative array from alternating key/value args
    local -A kv=()
    local i=0
    while [[ $i -lt $# ]]; do
        local key="${!i+1}"
        local val_idx=$((i + 2))
        key="${@:$((i+1)):1}"
        val="${@:$((i+2)):1}"
        kv["$key"]="$val"
        i=$((i + 2))
    done

    echo "=== Speaker Verification Status ==="
    echo

    if [[ "${kv[installed]:-}" != "true" ]]; then
        echo "Status: NOT INSTALLED"
        echo "Install with: resource-speaker-verification manage install"
        return
    fi

    if [[ "${kv[running]:-}" != "true" ]]; then
        echo "Status: STOPPED"
        echo "Start with: resource-speaker-verification manage start"
        return
    fi

    if [[ "${kv[healthy]:-}" == "true" ]]; then
        echo "Status: HEALTHY"
    else
        echo "Status: UNHEALTHY"
    fi

    [[ -n "${kv[health_message]:-}" ]] && echo "Health: ${kv[health_message]}"

    echo
    echo "Connection:"
    echo "  URL:       ${kv[base_url]:-}"
    echo "  Port:      ${kv[port]:-}"
    echo "  Container: ${kv[container_name]:-}"

    echo
    echo "Configuration:"
    echo "  Model:     ${kv[model]:-}"
    echo "  Device:    ${kv[device]:-}"
    echo "  Threshold: ${kv[threshold]:-}"
    echo "  GPU:       ${kv[gpu_enabled]:-}"

    if [[ -n "${kv[profile_count]:-}" ]] || [[ -n "${kv[data_size]:-}" ]]; then
        echo
        echo "Data:"
        [[ -n "${kv[profile_count]:-}" ]] && echo "  Profiles:  ${kv[profile_count]}"
        [[ -n "${kv[data_size]:-}" ]] && echo "  Data size: ${kv[data_size]}"
    fi

    if [[ -n "${kv[cpu_usage]:-}" ]] || [[ -n "${kv[memory_usage]:-}" ]]; then
        echo
        echo "Resources:"
        [[ -n "${kv[cpu_usage]:-}" ]] && echo "  CPU:    ${kv[cpu_usage]}"
        [[ -n "${kv[memory_usage]:-}" ]] && echo "  Memory: ${kv[memory_usage]}"
    fi
}
export -f speaker_verification::status::display_text

#######################################
# Main status entry point
#######################################
speaker_verification::status() {
    if command -v status::run_standard &>/dev/null; then
        status::run_standard "speaker-verification" "speaker_verification::status::collect_data" "speaker_verification::status::display_text" "$@"
    else
        # Fallback if framework not available
        local data
        data=$(speaker_verification::status::collect_data "$@")

        # Parse for exit code
        local healthy="false" running="false"
        while IFS= read -r line; do
            case "$line" in
                healthy) IFS= read -r healthy ;;
                running) IFS= read -r running ;;
            esac
        done <<< "$data"

        # Display
        local -a data_array
        mapfile -t data_array <<< "$data"
        speaker_verification::status::display_text "${data_array[@]}"

        if [[ "$healthy" == "true" ]]; then
            return 0
        elif [[ "$running" == "true" ]]; then
            return 1
        else
            return 2
        fi
    fi
}
export -f speaker_verification::status
