#!/usr/bin/env bash

# UI Smoke command module
scenario::smoke::run() {
    local scenario_name=""
    local json_output=false

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --json)
                json_output=true
                shift
                ;;
            --help|-h)
                scenario::smoke::help
                return 0
                ;;
            *)
                if [[ -z "$scenario_name" ]]; then
                    scenario_name="$1"
                else
                    log::error "Unexpected argument: $1"
                    scenario::smoke::help
                    return 1
                fi
                shift
                ;;
        esac
    done

    if [[ -z "$scenario_name" ]]; then
        log::error "Scenario name is required"
        scenario::smoke::help
        return 1
    fi

    local scenario_dir="${APP_ROOT}/scenarios/${scenario_name}"
    if [[ ! -d "$scenario_dir" ]]; then
        log::error "Scenario directory not found: $scenario_dir"
        return 1
    fi

    local smoke_helper="${APP_ROOT}/scripts/scenarios/testing/shell/ui-smoke.sh"
    if [[ ! -f "$smoke_helper" ]]; then
        log::error "UI smoke helper not found at $smoke_helper"
        return 1
    fi

    local summary_json
    local helper_status
    set +e
    summary_json=$(bash "$smoke_helper" --scenario "$scenario_name" --scenario-dir "$scenario_dir" --json)
    helper_status=$?
    set -e

    scenario::smoke::emit_output "$summary_json" "$json_output"
    return $helper_status
}

scenario::smoke::emit_output() {
    local summary_json="$1"
    local json_output="$2"

    if [[ -z "$summary_json" ]]; then
        log::error "UI smoke harness did not return summary output"
        return 1
    fi

    if [[ "$json_output" = true ]]; then
        echo "$summary_json"
        return 0
    fi

    scenario::smoke::render_text_summary "$summary_json"
}

scenario::smoke::render_text_summary() {
    local summary_json="$1"

    if ! command -v jq >/dev/null 2>&1; then
        log::info "$summary_json"
        return 0
    fi

    local scenario status message ui_url duration handshake_status handshake_duration handshake_error dependency_present screenshot_path console_path handshake_present dependency_field_present storage_summary patched_count
    scenario=$(echo "$summary_json" | jq -r '.scenario // ""')
    status=$(echo "$summary_json" | jq -r '.status // "unknown"')
    message=$(echo "$summary_json" | jq -r '.message // ""')
    ui_url=$(echo "$summary_json" | jq -r '.ui_url // ""')
    duration=$(echo "$summary_json" | jq -r '.duration_ms // 0')
    handshake_status=$(echo "$summary_json" | jq -r '.raw.handshake.signaled // false')
    handshake_duration=$(echo "$summary_json" | jq -r '.raw.handshake.durationMs // 0')
    handshake_error=$(echo "$summary_json" | jq -r '.raw.handshake.error // empty')
    handshake_present=$(echo "$summary_json" | jq -r 'if (.raw? and .raw.handshake?) then "true" else "false" end' 2>/dev/null || echo "false")
    dependency_field_present=$(echo "$summary_json" | jq -r 'if (.iframe_bridge? // empty | type == "object") and (.iframe_bridge | has("dependency_present")) then "true" else "false" end' 2>/dev/null || echo "false")
    dependency_present=$(echo "$summary_json" | jq -r '.iframe_bridge.dependency_present // false')
    screenshot_path=$(echo "$summary_json" | jq -r '.artifacts.screenshot // empty')
    console_path=$(echo "$summary_json" | jq -r '.artifacts.console // empty')
    storage_summary=$(echo "$summary_json" | jq -c '.storage_shim // []')

    log::info "Scenario: $scenario"
    log::info "UI URL: ${ui_url:-n/a}"
    log::info "Status: ${status} (${duration}ms)"
    if [[ -n "$message" && "$message" != "null" ]]; then
        log::info "Detail: $message"
    fi

    if [[ "$dependency_field_present" = "true" ]]; then
        if [[ "$dependency_present" != "true" ]]; then
            log::error "Iframe bridge dependency: missing (@vrooli/iframe-bridge)"
        else
            log::success "Iframe bridge dependency: present"
        fi
    fi

    if [[ "$handshake_present" = "true" ]]; then
        if [[ "$handshake_status" = "true" ]]; then
            log::success "Bridge handshake: ✅ (${handshake_duration}ms)"
        else
            local detail="${handshake_error:-Bridge never signaled ready}"
            log::error "Bridge handshake: ❌ ${detail}"
        fi
    elif [[ "$status" = "skipped" ]]; then
        log::info "Bridge handshake: (skipped)"
    fi

    if [[ -n "$screenshot_path" && "$screenshot_path" != "null" ]]; then
        log::info "Screenshot: $screenshot_path"
    fi
    if [[ -n "$console_path" && "$console_path" != "null" ]]; then
        log::info "Console log: $console_path"
    fi
    if [[ -n "$storage_summary" && "$storage_summary" != "[]" ]]; then
        patched_count=$(echo "$storage_summary" | jq '[.[] | select(.patched == true)] | length' 2>/dev/null || echo "0")
        if [[ "$patched_count" != "0" ]]; then
            local patched_props
            patched_props=$(echo "$storage_summary" | jq -r '[.[] | select(.patched == true) | .prop] | join(", ")' 2>/dev/null || echo "localStorage")
            log::warning "Storage shim: enabled for ${patched_props} (third-party storage blocked)"
        fi
    fi
}

scenario::smoke::help() {
    cat <<'EOF'
Usage: vrooli scenario ui-smoke <name> [--json]

Run the Browserless-backed UI smoke harness for a scenario.

Options:
  --json        Emit JSON summary instead of human-readable output
  --help        Show this help message
EOF
}
