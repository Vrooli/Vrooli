#!/usr/bin/env bash
# Integration phase helper utilities
set -euo pipefail

testing::integration::resolve_workflow_definition() {
    local workflow_path="$1"
    local scenario_dir="$2"
    if [ -z "$workflow_path" ] || [ ! -f "$workflow_path" ]; then
        return 1
    fi
    local resolver_root
    resolver_root="${APP_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../../" && pwd)}"
    local resolver_script="${resolver_root}/scripts/scenarios/testing/playbooks/resolve-workflow.py"
    if [ -f "$resolver_script" ]; then
        local python_cmd=""
        if command -v python3 >/dev/null 2>&1; then
            python_cmd="python3"
        elif command -v python >/dev/null 2>&1; then
            python_cmd="python"
        fi
        if [ -n "$python_cmd" ]; then
            "$python_cmd" "$resolver_script" --workflow "$workflow_path" --scenario "$scenario_dir"
            return $?
        fi
        echo "⚠️  python is not available; skipping workflow fixture resolution" >&2
    fi
    cat "$workflow_path"
}

testing::integration::build_playbook_registry() {
    local scenario_dir="${1:-${TESTING_PHASE_SCENARIO_DIR:-$(pwd)}}"
    local playbook_root="$scenario_dir/test/playbooks"

    if [ ! -d "$playbook_root" ]; then
        return 0
    fi

    # Prefer test-genie CLI (cross-platform, no runtime dependency)
    if command -v test-genie >/dev/null 2>&1; then
        test-genie registry build --scenario "$scenario_dir" >/dev/null 2>&1 || true
        return 0
    fi

    # Fallback to legacy Node.js script (deprecated)
    local registry_builder="${APP_ROOT}/scripts/scenarios/testing/playbooks/build-registry.mjs"
    if [ -f "$registry_builder" ] && command -v node >/dev/null 2>&1; then
        node "$registry_builder" --scenario "$scenario_dir" >/dev/null 2>&1 || true
    fi
}

# Check bundle freshness before running integration tests
# Integration tests often interact with the UI, so outdated bundles cause false failures
testing::integration::check_bundle_freshness() {
    local scenario_dir="$1"
    local scenario_name="$2"

    # Skip if no UI directory exists
    if [[ ! -d "$scenario_dir/ui" ]]; then
        return 0
    fi

    # Get bundle check config from service.json if available
    local service_json="$scenario_dir/.vrooli/service.json"
    local config='{}'
    if [[ -f "$service_json" ]] && command -v jq >/dev/null 2>&1; then
        local candidate
        candidate=$(jq -c '.lifecycle.setup.condition.checks[]? | select(.type == "ui-bundle")' "$service_json" 2>/dev/null | head -n 1)
        if [[ -n "$candidate" ]]; then
            config="$candidate"
        fi
    fi

    # Call bundle check script directly
    local bundle_check_script="${APP_ROOT}/scripts/lib/setup-conditions/ui-bundle-check.sh"
    if [[ ! -f "$bundle_check_script" ]]; then
        if declare -F testing::phase::add_warning >/dev/null 2>&1; then
            testing::phase::add_warning "Bundle check script missing; skipping freshness check"
        else
            echo "⚠️  Bundle check script missing; skipping freshness check" >&2
        fi
        return 0
    fi

    # Run the check - exit 0 means stale/missing, exit 1 means fresh
    local check_output
    local check_result
    set +e
    check_output=$(APP_ROOT="$scenario_dir" bash "$bundle_check_script" "$config" 2>&1)
    check_result=$?
    set -e

    # Exit code 0 = setup needed (bundle stale/missing)
    # Exit code 1 = bundle is current
    if [[ $check_result -eq 0 ]]; then
        local bundle_reason="${check_output:-UI bundle missing or outdated}"
        # Clean up debug prefix if present
        bundle_reason="${bundle_reason#\[DEBUG\] }"

        # Use log functions if available, otherwise use echo
        if declare -F log::error >/dev/null 2>&1; then
            log::error "Integration tests blocked: ${bundle_reason}"
            log::error "  ↳ Fix: vrooli scenario restart ${scenario_name}"
            log::error "  ↳ Then rerun: vrooli test integration ${scenario_name}"
        else
            echo "❌ Integration tests blocked: ${bundle_reason}" >&2
            echo "  ↳ Fix: vrooli scenario restart ${scenario_name}" >&2
            echo "  ↳ Then rerun: vrooli test integration ${scenario_name}" >&2
        fi

        if declare -F testing::phase::add_error >/dev/null 2>&1; then
            testing::phase::add_error "Stale UI bundle detected"
        fi

        return 1
    fi

    # Bundle is fresh or no bundle check needed
    return 0
}

# Ensure the running UI/API processes were started after the latest build artifacts.
# This prevents running E2E against stale binaries even when dist files look fresh.
testing::integration::check_runtime_freshness() {
    local scenario_dir="$1"
    local scenario_name="$2"
    local now
    now=$(date +%s)

    _mtime() {
        local path="$1"
        if [ ! -f "$path" ]; then
            echo ""
            return
        fi
        stat -c %Y "$path" 2>/dev/null || stat -f %m "$path" 2>/dev/null || echo ""
    }

    _pid_listening_on_port() {
        local port="$1"
        # Prefer lsof
        if command -v lsof >/dev/null 2>&1; then
            lsof -tiTCP:"$port" -sTCP:LISTEN 2>/dev/null | head -1
            return
        fi
        # Fallback to ss
        if command -v ss >/dev/null 2>&1; then
            ss -ltnp 2>/dev/null | awk -v p=":$port" '$4 ~ p {gsub("pid=","",$NF); split($NF,a,","); print a[1]; exit}'
            return
        fi
        echo ""
    }

    _youngest_start_epoch() {
        local pattern="$1"
        local port="$2"
        local best_start=""
        local pid=""

        # If port provided, try to resolve PID from listener
        if [ -n "$port" ]; then
            pid=$(_pid_listening_on_port "$port")
        fi

        # Fallback to pattern search
        if [ -z "$pid" ]; then
            for pid in $(pgrep -f "$pattern" 2>/dev/null || true); do
                :
            done
        fi

        for target_pid in $pid; do
            local etimes
            etimes=$(ps -o etimes= -p "$target_pid" 2>/dev/null | tr -d '[:space:]')
            [ -z "$etimes" ] && continue
            local start_epoch=$((now - etimes))
            if [ -z "$best_start" ] || [ "$start_epoch" -gt "$best_start" ]; then
                best_start="$start_epoch"
            fi
        done

        echo "$best_start"
    }

    local api_binary="${scenario_dir}/api/browser-automation-studio-api"
    local ui_bundle="${scenario_dir}/ui/dist/index.html"

    # Resolve ports for the scenario being tested
    # IMPORTANT: Always query the scenario's actual ports, don't trust inherited environment
    # variables which may belong to a different scenario (e.g. when testing from within another
    # scenario's context like ecosystem-manager)
    if command -v vrooli >/dev/null 2>&1; then
        local resolved_api_port resolved_ui_port
        resolved_api_port=$(vrooli scenario port "$scenario_name" API_PORT 2>/dev/null || echo "")
        resolved_ui_port=$(vrooli scenario port "$scenario_name" UI_PORT 2>/dev/null || echo "")
        # Only use resolved ports if they were successfully retrieved
        [ -n "$resolved_api_port" ] && API_PORT="$resolved_api_port"
        [ -n "$resolved_ui_port" ] && UI_PORT="$resolved_ui_port"
    fi

    local api_build_mtime ui_build_mtime
    api_build_mtime=$(_mtime "$api_binary")
    ui_build_mtime=$(_mtime "$ui_bundle")

    # Check API freshness
    if [ -n "$api_build_mtime" ]; then
        local api_start_epoch
        api_start_epoch=$(_youngest_start_epoch "browser-automation-studio-api" "${API_PORT:-}")
        if [ -z "$api_start_epoch" ]; then
            log::error "API process not running. Restart scenario: vrooli scenario restart ${scenario_name}"
            testing::phase::add_error "API runtime missing; restart scenario"
            return 1
        fi
        if [ "$api_build_mtime" -gt "$api_start_epoch" ]; then
            log::error "API binary newer than running process (stale runtime). Restart scenario: vrooli scenario restart ${scenario_name}"
            testing::phase::add_error "API runtime stale; restart scenario to load latest build"
            return 1
        fi
    fi

    # Check UI freshness
    if [ -n "$ui_build_mtime" ]; then
        local ui_start_epoch
        ui_start_epoch=$(_youngest_start_epoch "browser-automation-studio/ui/server.js" "${UI_PORT:-}")
        if [ -z "$ui_start_epoch" ]; then
            log::error "UI server not running. Restart scenario: vrooli scenario restart ${scenario_name}"
            testing::phase::add_error "UI runtime missing; restart scenario"
            return 1
        fi
        if [ "$ui_build_mtime" -gt "$ui_start_epoch" ]; then
            log::error "UI bundle newer than running server (stale runtime). Restart scenario: vrooli scenario restart ${scenario_name}"
            testing::phase::add_error "UI runtime stale; restart scenario to load latest build"
            return 1
        fi
    fi

    return 0
}

# Run Browser Automation Studio workflow validations (or other workflow-based checks)
# NOTE: We do NOT run explicit linting before workflow execution. The BAS API validates
# workflows during execution, so separate linting is redundant and adds unnecessary complexity.
testing::integration::validate_all() {
    local summary="Integration tests completed"
    local force_end=false

    # Set up signal handler to ensure completion logic runs on timeout
    _integration_cleanup() {
        if declare -F _testing_playbooks__cleanup_seeds >/dev/null 2>&1; then
            _testing_playbooks__cleanup_seeds || true
        fi
        if [ "$force_end" = true ]; then
            return 0  # Already cleaned up
        fi
        force_end=true

        if [ "$TESTING_PHASE_AUTO_MANAGED" = true ]; then
            # Add note that phase was interrupted
            testing::phase::add_error "Phase timed out - recording partial results"
            testing::phase::auto_lifecycle_end "Integration tests timed out (partial results)"
        fi
    }
    trap _integration_cleanup SIGTERM SIGINT EXIT

    testing::phase::auto_lifecycle_start \
        --phase-name "integration" \
        --default-target-time "240s" \
        --summary "$summary" \
        --config-phase-key "integration" \
        || true

    local scenario_dir="${TESTING_PHASE_SCENARIO_DIR:-$(pwd)}"
    local scenario_name="$(basename "$scenario_dir")"

    # Set browser session strategy from config
    local config_file="$scenario_dir/.vrooli/testing.json"
    local session_strategy="clean"  # Safe default
    if [ -f "$config_file" ] && command -v jq >/dev/null 2>&1; then
        local configured_strategy
        configured_strategy=$(jq -r '.phases.integration.browser_session_strategy // empty' "$config_file" 2>/dev/null || echo "")
        if [ -n "$configured_strategy" ]; then
            session_strategy="$configured_strategy"
        fi
    fi
    export BAS_SESSION_STRATEGY="$session_strategy"
    log::info "Browser session strategy: $session_strategy"

    # Check bundle freshness before running integration tests
    # This prevents wasting time on tests that will fail due to outdated UI
    if ! testing::integration::check_bundle_freshness "$scenario_dir" "$scenario_name"; then
        # Clear expected requirements to avoid misleading "missing coverage" warnings
        # since the tests never ran
        TESTING_PHASE_EXPECTED_REQUIREMENTS=()
        force_end=true
        testing::phase::auto_lifecycle_end "Integration tests blocked by stale UI bundle"
        return 1
    fi

    # Ensure running UI/API processes are not older than the latest build artifacts
    if ! testing::integration::check_runtime_freshness "$scenario_dir" "$scenario_name"; then
        TESTING_PHASE_EXPECTED_REQUIREMENTS=()
        force_end=true
        testing::phase::auto_lifecycle_end "Integration tests blocked by stale runtime"
        return 1
    fi

    testing::integration::build_playbook_registry "$scenario_dir"

    # Run BAS workflow validations
    # NOTE: Do NOT pass --scenario here. The workflow runner defaults to "browser-automation-studio"
    # which is the scenario that executes workflows. $scenario_name is the scenario BEING TESTED.
    if ! testing::phase::run_bas_automation_validations --manage-runtime auto; then
        local bas_rc=$?
        if [ "$bas_rc" -ne 0 ] && [ "$bas_rc" -ne 200 ]; then
            testing::phase::add_error "Browser Automation Studio workflow validations failed"
        fi
    fi

    # Check if any tests actually ran
    # If test count is 0, this indicates the phase was effectively skipped
    # (e.g., due to API_PORT issues, missing BAS CLI, or no playbooks)
    if [ "$TESTING_PHASE_TEST_COUNT" -eq 0 ]; then
        # Override phase status to "skipped" instead of "passed"
        force_end=true
        # Provide informative message about why no tests ran
        local skip_reason="No tests executed"
        if [ "$TESTING_PHASE_ERROR_COUNT" -gt 0 ]; then
            skip_reason="Prerequisites not met (check errors above)"
        fi
        testing::phase::end_with_summary "$skip_reason" "skipped"
        return 200
    fi

    force_end=true  # Mark that we're doing normal completion
    testing::phase::auto_lifecycle_end "$summary"
}

export -f testing::integration::check_bundle_freshness
export -f testing::integration::check_runtime_freshness
export -f testing::integration::validate_all
