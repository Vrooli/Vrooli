#!/usr/bin/env bash
# Integration phase helper utilities
set -euo pipefail

# Run Browser Automation Studio workflow validations (or other workflow-based checks)
testing::integration::validate_all() {
    local summary="Integration tests completed"
    testing::phase::auto_lifecycle_start \
        --phase-name "integration" \
        --default-target-time "240s" \
        --summary "$summary" \
        --config-phase-key "integration" \
        || true

    local scenario_name="${TESTING_PHASE_SCENARIO_NAME:-$(basename "$(pwd)")}" 

    if ! testing::phase::run_bas_automation_validations --scenario "$scenario_name" --manage-runtime auto; then
        local bas_rc=$?
        if [ "$bas_rc" -ne 0 ] && [ "$bas_rc" -ne 200 ]; then
            testing::phase::add_error "Browser Automation Studio workflow validations failed"
        fi
    fi

    testing::phase::auto_lifecycle_end "$summary"
}

export -f testing::integration::validate_all
