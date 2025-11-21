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

export -f testing::integration::validate_all
