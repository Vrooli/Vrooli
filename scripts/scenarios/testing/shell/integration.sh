#!/usr/bin/env bash
# Integration phase helper utilities
set -euo pipefail

# Run Browser Automation Studio workflow validations (or other workflow-based checks)
testing::integration::validate_all() {
    local summary="Integration tests completed"
    local force_end=false

    # Set up signal handler to ensure completion logic runs on timeout
    _integration_cleanup() {
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
