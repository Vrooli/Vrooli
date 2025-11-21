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
    local registry_builder="${APP_ROOT}/scripts/scenarios/testing/playbooks/build-registry.mjs"

    if [ ! -d "$playbook_root" ] || [ ! -f "$registry_builder" ]; then
        return 0
    fi

    if command -v node >/dev/null 2>&1; then
        node "$registry_builder" --scenario "$scenario_dir" >/dev/null || true
    else
        echo "⚠️  Node.js not available; skipping playbook registry generation"
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

    # Source the ui-smoke helper to access bundle check function
    local ui_smoke_helper="${APP_ROOT}/scripts/scenarios/testing/shell/ui-smoke.sh"
    if [[ ! -f "$ui_smoke_helper" ]]; then
        if declare -F testing::phase::add_warning >/dev/null 2>&1; then
            testing::phase::add_warning "UI smoke helper missing; skipping bundle freshness check"
        else
            echo "⚠️  UI smoke helper missing; skipping bundle freshness check" >&2
        fi
        return 0
    fi

    # Source the helper to get access to _check_bundle_freshness function
    source "$ui_smoke_helper"

    if ! command -v jq >/dev/null 2>&1; then
        if declare -F testing::phase::add_warning >/dev/null 2>&1; then
            testing::phase::add_warning "jq not available; skipping bundle freshness check"
        else
            echo "⚠️  jq not available; skipping bundle freshness check" >&2
        fi
        return 0
    fi

    # Check bundle freshness
    local bundle_info_json
    bundle_info_json=$(testing::ui_smoke::_check_bundle_freshness "$scenario_dir" 2>/dev/null)
    local bundle_fresh
    bundle_fresh=$(echo "$bundle_info_json" | jq -r '
        if .fresh == false then "false"
        elif .fresh == true then "true"
        else "unknown"
        end' 2>/dev/null || echo "unknown")

    if [[ "$bundle_fresh" = "false" ]]; then
        local bundle_reason
        bundle_reason=$(echo "$bundle_info_json" | jq -r '.reason // "UI bundle missing or outdated"')

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

    testing::integration::build_playbook_registry "$scenario_dir"

    if ! testing::phase::run_bas_automation_validations --scenario "$scenario_name" --manage-runtime auto; then
        local bas_rc=$?
        if [ "$bas_rc" -ne 0 ] && [ "$bas_rc" -ne 200 ]; then
            testing::phase::add_error "Browser Automation Studio workflow validations failed"
        fi
    fi

    force_end=true  # Mark that we're doing normal completion
    testing::phase::auto_lifecycle_end "$summary"
}

export -f testing::integration::check_bundle_freshness
export -f testing::integration::validate_all
