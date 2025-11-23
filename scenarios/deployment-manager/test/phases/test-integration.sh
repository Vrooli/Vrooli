#!/bin/bash
# Runs Browser Automation Studio workflow automations and CLI BATS tests from requirements registry.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/integration.sh"

# Custom integration validation that runs both BATS CLI tests and BAS workflows
# This overrides the default testing::integration::validate_all to add BATS support
testing::integration::validate_all_with_bats() {
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
    if ! testing::integration::check_bundle_freshness "$scenario_dir" "$scenario_name"; then
        # Clear expected requirements to avoid misleading "missing coverage" warnings
        TESTING_PHASE_EXPECTED_REQUIREMENTS=()
        force_end=true
        testing::phase::auto_lifecycle_end "Integration tests blocked by stale UI bundle"
        return 1
    fi

    # Run BATS CLI tests and record requirement results
    CLI_TEST_DIR="${scenario_dir}/test/cli"
    if [ -d "$CLI_TEST_DIR" ] && command -v bats >/dev/null 2>&1; then
        echo "🧪 Running CLI BATS tests..."
        # Unset API_PORT to let CLI detect it dynamically
        unset API_PORT

        # Run BATS and capture TAP output
        BATS_OUTPUT=$(mktemp)
        trap "rm -f '$BATS_OUTPUT'" EXIT

        if bats --tap "$CLI_TEST_DIR"/*.bats > "$BATS_OUTPUT" 2>&1; then
            BATS_EXIT_CODE=0
            echo "✅ BATS tests passed"
        else
            BATS_EXIT_CODE=$?
            echo "❌ BATS tests failed" >&2
        fi

        # Parse TAP output to extract requirement results
        # TAP format: "ok 1 [REQ:ID] Test description" or "not ok 1 [REQ:ID] Test description"
        while IFS= read -r line; do
            # Extract requirement ID from test name
            if [[ "$line" =~ ^(ok|not\ ok)\ [0-9]+\ \[REQ:([A-Z0-9-]+)\] ]]; then
                test_status="${BASH_REMATCH[1]}"
                req_id="${BASH_REMATCH[2]}"
                test_desc="${line#*\]}"  # Everything after [REQ:ID]
                test_desc="${test_desc# }"     # Trim leading space

                if [ "$test_status" = "ok" ]; then
                    testing::phase::add_requirement --id "$req_id" --status passed --evidence "CLI: $test_desc"
                    testing::phase::add_test passed
                else
                    testing::phase::add_requirement --id "$req_id" --status failed --evidence "CLI: $test_desc"
                    testing::phase::add_test failed
                fi
            elif [[ "$line" =~ ^(ok|not\ ok)\ [0-9]+ ]]; then
                # Test without requirement tag
                test_status="${BASH_REMATCH[1]}"
                if [ "$test_status" = "ok" ]; then
                    testing::phase::add_test passed
                else
                    testing::phase::add_test failed
                fi
            fi
        done < "$BATS_OUTPUT"

        if [ $BATS_EXIT_CODE -ne 0 ]; then
            force_end=true
            testing::phase::auto_lifecycle_end "BATS tests failed"
            return 1
        fi
    fi

    # Build playbook registry and run BAS workflow validations
    testing::integration::build_playbook_registry "$scenario_dir"
    if ! testing::phase::run_bas_automation_validations --manage-runtime auto; then
        local bas_rc=$?
        if [ "$bas_rc" -ne 0 ] && [ "$bas_rc" -ne 200 ]; then
            testing::phase::add_error "Browser Automation Studio workflow validations failed"
        fi
    fi

    force_end=true  # Mark that we're doing normal completion
    testing::phase::auto_lifecycle_end "$summary"
}

# Run custom integration validation with BATS support
testing::integration::validate_all_with_bats
