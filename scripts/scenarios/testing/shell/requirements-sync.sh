#!/usr/bin/env bash
# Requirements synchronization utilities for test runners
# Syncs test execution results with the requirements registry
set -euo pipefail

# Synchronize requirements registry after test execution
# This function collects phase execution metadata and syncs it with the requirements registry
#
# Usage: testing::requirements::sync SCENARIO_DIR SCENARIO_NAME APP_ROOT
#
# Arguments:
#   $1 - scenario_dir: Path to the scenario directory
#   $2 - scenario_name: Name of the scenario
#   $3 - app_root: Path to the application root directory
#
# Environment Variables:
#   TESTING_REQUIREMENTS_SYNC - Set to "1" to enable sync (default: enabled)
#   TESTING_REQUIREMENTS_SYNC_FORCE - Set to "1" to force sync even with partial test runs
#   TESTING_SUITE_COMMAND_HISTORY - Command history to include in sync manifest
#   TESTING_RUNNER_PHASES - Array of registered test phases
#   TESTING_RUNNER_PHASE_OPTIONAL - Associative array of optional phase flags
#   TESTING_RUNNER_STATUS - Associative array of phase execution statuses
testing::requirements::sync() {
    local scenario_dir="$1"
    local scenario_name="$2"
    local app_root="$3"

    local registry_dir="$scenario_dir/requirements"

    if [ "${TESTING_REQUIREMENTS_SYNC:-1}" != "1" ]; then
        return 0
    fi

    if [ ! -d "$registry_dir" ]; then
        log::warning "⚠️  Skipping requirements sync (requirements/ missing)"
        return 0
    fi

    if ! command -v node >/dev/null 2>&1; then
        log::warning "⚠️  Skipping requirements sync (Node.js not available)"
        return 0
    fi

    local -a phase_entries=()
    local -a missing_required=()
    local -a skipped_required=()

    # Collect phase execution metadata
    if [ ${#TESTING_RUNNER_PHASES[@]} -gt 0 ]; then
        local phase
        for phase in "${TESTING_RUNNER_PHASES[@]}"; do
            local optional_flag="${TESTING_RUNNER_PHASE_OPTIONAL[$phase]:-false}"
            local key="phase:${phase}"
            local status="${TESTING_RUNNER_STATUS[$key]:-}"
            local recorded_flag="false"
            if [ -n "$status" ]; then
                recorded_flag="true"
            fi
            local normalized_status="${status:-not_run}"

            # Track missing/skipped required phases
            if [ "$optional_flag" != "true" ]; then
                if [ "$recorded_flag" = "false" ]; then
                    missing_required+=("$phase")
                elif [[ "$normalized_status" == "skipped" || "$normalized_status" == "missing" || "$normalized_status" == "not_executable" || "$normalized_status" == "not_run" ]]; then
                    skipped_required+=("$phase")
                fi
            fi

            # Build JSON entry for this phase
            local optional_json="false"
            if [ "$optional_flag" = "true" ]; then
                optional_json="true"
            fi
            local recorded_json="false"
            if [ "$recorded_flag" = "true" ]; then
                recorded_json="true"
            fi
            local entry="{\"phase\":\"$phase\",\"status\":\"$normalized_status\",\"optional\":$optional_json,\"recorded\":$recorded_json}"
            phase_entries+=("$entry")
        done
    fi

    # Build phase status JSON array
    local phase_status_json='[]'
    if [ ${#phase_entries[@]} -gt 0 ]; then
        local combined=""
        local first_entry="true"
        local entry
        for entry in "${phase_entries[@]}"; do
            if [ "$first_entry" = "true" ]; then
                combined="$entry"
                first_entry="false"
            else
                combined+=",$entry"
            fi
        done
        phase_status_json="[${combined}]"
    fi

    # Check if we have any phase data
    if [ -z "$phase_status_json" ] || [ "$phase_status_json" = '[]' ]; then
        log::warning "⚠️  Requirements sync skipped: no phase execution metadata recorded"
        log::info "    ↳ Re-run tests via 'test/run-tests.sh' to regenerate coverage"
        return 0
    fi

    # Check if full test suite was executed (unless forced)
    if { [ ${#missing_required[@]} -gt 0 ] || [ ${#skipped_required[@]} -gt 0 ]; } && [ "${TESTING_REQUIREMENTS_SYNC_FORCE:-0}" != "1" ]; then
        log::warning "⚠️  Requirements sync skipped: full suite not executed"
        if [ ${#missing_required[@]} -gt 0 ]; then
            log::info "    Missing phases: ${missing_required[*]}"
        fi
        if [ ${#skipped_required[@]} -gt 0 ]; then
            log::info "    Skipped phases: ${skipped_required[*]}"
        fi
        log::info "    ↳ Run 'test/run-tests.sh' without presets to refresh requirements"
        return 0
    elif { [ ${#missing_required[@]} -gt 0 ] || [ ${#skipped_required[@]} -gt 0 ]; } && [ "${TESTING_REQUIREMENTS_SYNC_FORCE:-0}" = "1" ]; then
        log::warning "⚠️  Forcing requirements sync despite partial suite (TESTING_REQUIREMENTS_SYNC_FORCE=1)"
    fi

    # Build command history JSON
    local commands_json=""
    if [ -n "${TESTING_SUITE_COMMAND_HISTORY:-}" ] && command -v jq >/dev/null 2>&1; then
        commands_json=$(printf '%s\n' "$TESTING_SUITE_COMMAND_HISTORY" | jq -c -R -s 'split("\n") | map(select(length > 0))')
    fi
    if [ -z "$commands_json" ]; then
        commands_json='[]'
    fi

    # Create manifest entry
    local manifest_entry=""
    local manifest_file="$scenario_dir/coverage/requirements-sync/run-log.jsonl"
    local manifest_dir
    manifest_dir=$(dirname "$manifest_file")
    if command -v jq >/dev/null 2>&1; then
        mkdir -p "$manifest_dir"
        local timestamp
        timestamp=$(date -u +%Y-%m-%dT%H:%M:%SZ)
        manifest_entry=$(jq -n \
            --arg ts "$timestamp" \
            --arg scenario "$scenario_name" \
            --argjson commands "$commands_json" \
            '{timestamp:$ts, scenario:$scenario, commands:$commands}')
        printf '%s\n' "$manifest_entry" >> "$manifest_file"
    else
        manifest_file=""
    fi

    # Execute requirements sync command
    local sync_cmd=(node "$app_root/scripts/requirements/report.js" --scenario "$scenario_name" --mode sync)
    local sync_status=1
    if REQUIREMENTS_SYNC_TEST_COMMANDS="$commands_json" \
        REQUIREMENTS_SYNC_PHASE_STATUS="$phase_status_json" \
        REQUIREMENTS_SYNC_MANIFEST_ENTRY="$manifest_entry" \
        REQUIREMENTS_SYNC_RUN_LOG="$manifest_file" \
        "${sync_cmd[@]}" >/dev/null; then
        sync_status=0
    fi

    if [ $sync_status -eq 0 ]; then
        log::info "📋 Requirements registry synced after test run"
    else
        log::warning "⚠️  Failed to sync requirements registry; leaving requirement files unchanged"
    fi
}

# Export function
export -f testing::requirements::sync
