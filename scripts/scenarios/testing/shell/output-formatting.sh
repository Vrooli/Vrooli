#!/usr/bin/env bash
# Output formatting functions for test runner
# Handles executive summaries, phase progress, and failure analysis
set -euo pipefail

# -----------------------------------------------------------------------------
# Utility Functions
# -----------------------------------------------------------------------------

_testing_runner_join_list() {
    local separator="$1"
    shift || true
    local result=""
    local first=true
    for item in "$@"; do
        if [ "$first" = true ]; then
            result="$item"
            first=false
        else
            result+="$separator$item"
        fi
    done
    echo "$result"
}

_testing_runner_friendly_label() {
    local key="$1"
    local type="${TESTING_RUNNER_ITEM_TYPE[$key]:-}"
    local display="${TESTING_RUNNER_ITEM_DISPLAY[$key]:-}"

    case "$type" in
        phase)
            echo "phase '${display#phase-}'"
            ;;
        test)
            echo "test '${display#test-}'"
            ;;
        *)
            echo "$display"
            ;;
    esac
}

# -----------------------------------------------------------------------------
# Header and Summary Functions
# -----------------------------------------------------------------------------

_testing_runner_print_header() {
    local selection_ref_name="$1"
    local verbose="$2"
    local parallel="$3"
    local dry_run="$4"
    local manage_runtime="$5"
    local -n selection_ref="$selection_ref_name"

    # Print executive summary box
    echo "╔═══════════════════════════════════════════════════════════════╗"
    printf "║  %-61s║\n" "${TESTING_RUNNER_SCENARIO_NAME^^} COMPREHENSIVE TEST SUITE"
    echo "╠═══════════════════════════════════════════════════════════════╣"
    printf "║  %-61s║\n" "Scenario: ${TESTING_RUNNER_SCENARIO_NAME}"
    printf "║  %-61s║\n" "Test directory: ${TESTING_RUNNER_TEST_DIR##*/scenarios/}"

    # Count phases and estimate total time
    local phase_count=0
    local estimated_time=0
    for item in "${selection_ref[@]}"; do
        if _testing_runner_is_phase "$item"; then
            ((phase_count++)) || true
            local timeout="${TESTING_RUNNER_PHASE_TIMEOUT[$item]:-60}"
            ((estimated_time += timeout)) || true
        fi
    done

    printf "║  %-61s║\n" "Phases: ${phase_count} • Estimated: ~${estimated_time}s"
    echo "╚═══════════════════════════════════════════════════════════════╝"
    echo ""

    # Print test plan
    echo "Test Plan:"
    local index=0

    # Phase descriptions map
    declare -A phase_descriptions=(
        ["structure"]="Validate structure & start runtime"
        ["dependencies"]="Check runtimes, packages, resources"
        ["unit"]="Go + Node.js unit tests"
        ["integration"]="BAS workflow automations"
        ["business"]="Endpoints, CLI, websockets"
        ["performance"]="Lighthouse audits"
    )

    for item in "${selection_ref[@]}"; do
        if _testing_runner_is_phase "$item"; then
            ((index++)) || true
            local timeout="${TESTING_RUNNER_PHASE_TIMEOUT[$item]:-60}"
            local description="${phase_descriptions[$item]:-Run $item tests}"
            printf "  [%d/%d] %-15s (±%ds)  → %s\n" "$index" "$phase_count" "$item" "$timeout" "$description"
        fi
    done

    echo ""
    if [ "$dry_run" = "true" ]; then
        echo "🔍 DRY RUN MODE - No tests will be executed"
        echo ""
    else
        echo "Starting execution..."
        echo ""
    fi
}

_testing_runner_print_summary() {
    local total_duration="$1"
    local coverage_requested="$2"

    local total=0
    local passed=0
    local failed=0
    local skipped=0

    for item in "${TESTING_RUNNER_EXECUTION_ITEMS[@]}"; do
        ((total++)) || true
        case "${TESTING_RUNNER_STATUS[$item]}" in
            passed)
                ((passed++)) || true
                ;;
            skipped)
                ((skipped++)) || true
                ;;
            "")
                ;;
            *)
                ((failed++)) || true
                ;;
        esac
    done

    echo ""
    echo "╔═══════════════════════════════════════════════════════════════╗"
    printf "║  %-61s║\n" "TEST SUITE COMPLETE"
    echo "╠═══════════════════════════════════════════════════════════════╣"
    local results_line="Results: ${passed} passed • ${failed} failed • Duration: ${total_duration}s"
    printf "║  %-61s║\n" "$results_line"
    echo "╚═══════════════════════════════════════════════════════════════╝"
    echo ""

    if [ $failed -gt 0 ]; then
        # Analyze failures for root causes and blocking relationships
        _testing_runner_analyze_failures "$total_duration"
    fi

    # Show phase results summary
    echo "PHASE RESULTS:"

    # First show passed phases
    local has_passed=false
    for item in "${TESTING_RUNNER_EXECUTION_ITEMS[@]}"; do
        if [[ "$item" == phase:* ]]; then
            local phase_name="${item#phase:}"
            local status="${TESTING_RUNNER_STATUS[$item]}"
            local duration="${TESTING_RUNNER_DURATION[$item]:-0}"

            if [ "$status" = "passed" ]; then
                has_passed=true
                local display="${TESTING_RUNNER_ITEM_DISPLAY[$item]:-$phase_name}"
                local warnings=""

                # Check for warnings in log
                local log_path="${TESTING_RUNNER_LOG_PATH[$item]:-}"
                if [ -f "$log_path" ]; then
                    local warning_count=$(grep -c "\[WARNING:" "$log_path" 2>/dev/null || echo "0")
                    # Clean up any newlines or whitespace
                    warning_count=$(echo "$warning_count" | tr -d '\n' | tr -d ' ')
                    if [ -n "$warning_count" ] && [ "$warning_count" -gt 0 ] 2>/dev/null; then
                        warnings=" ($warning_count warnings)"
                    fi
                fi

                printf "  ✅ %-20s • %3ds%s\n" "$display" "$duration" "$warnings"
            fi
        fi
    done

    # Then show failed phases
    echo ""
    for item in "${TESTING_RUNNER_EXECUTION_ITEMS[@]}"; do
        if [[ "$item" == phase:* ]]; then
            local phase_name="${item#phase:}"
            local status="${TESTING_RUNNER_STATUS[$item]}"
            local duration="${TESTING_RUNNER_DURATION[$item]:-0}"

            if [ "$status" != "passed" ] && [ "$status" != "skipped" ]; then
                local display="${TESTING_RUNNER_ITEM_DISPLAY[$item]:-$phase_name}"
                local log_path="${TESTING_RUNNER_LOG_PATH[$item]:-}"
                local reason=""

                # Try to extract error reason from log
                if [ -f "$log_path" ]; then
                    if grep -q "UI bundle.*stale\|bundle.*outdated" "$log_path" 2>/dev/null; then
                        reason=" (UI bundle stale)"
                    elif grep -q "TypeScript.*error\|Cannot find name" "$log_path" 2>/dev/null; then
                        reason=" (TypeScript error)"
                    elif grep -q "test.*failed" "$log_path" 2>/dev/null; then
                        local fail_count=$(grep -c "❌.*failed" "$log_path" 2>/dev/null || echo "?")
                        reason=" ($fail_count test failures)"
                    fi
                fi

                printf "  ❌ %-20s • %3ds%s\n" "$display" "$duration" "$reason"
            elif [ "$status" = "skipped" ]; then
                local display="${TESTING_RUNNER_ITEM_DISPLAY[$item]:-$phase_name}"
                printf "  ⏭️  %-20s • %3ds • Skipped\n" "$display" "$duration"
            fi
        fi
    done

    echo ""

    # Show artifact summary
    testing::artifacts::summary
    echo ""

    # Show cache statistics
    testing::cache::stats
    echo ""

    if [ $failed -eq 0 ] && [ $skipped -eq 0 ]; then
        log::success "🎉 All tests passed successfully!"
        log::success "✅ ${TESTING_RUNNER_SCENARIO_NAME^} testing infrastructure is working correctly"
    elif [ $failed -eq 0 ]; then
        log::success "✅ Test execution completed (with some skipped)"
    fi

    if [ "$coverage_requested" = "true" ]; then
        echo ""
        echo "📈 Coverage reports may be available in:"
        echo "   • Go: api/coverage.html"
        echo "   • Node.js: ui/coverage/lcov-report/index.html"
        echo "   • Python: htmlcov/index.html"
    fi

    echo ""
    echo "📚 DOCUMENTATION:"
    local app_root
    app_root=$(cd "${TESTING_RUNNER_SCENARIO_DIR}/../.." && pwd)
    echo "   • ${app_root}/docs/testing/architecture/PHASED_TESTING.md"
    echo "   • ${app_root}/docs/testing/guides/requirement-tracking-quick-start.md"
    echo "   • ${app_root}/docs/testing/guides/writing-testable-uis.md"
    echo "   • ${app_root}/docs/testing/guides/ui-automation-with-bas.md"
    echo "   • Test files in: $TESTING_RUNNER_TEST_DIR"
    echo ""
}

# -----------------------------------------------------------------------------
# Failure Analysis
# -----------------------------------------------------------------------------

_testing_runner_analyze_failures() {
    local total_duration="$1"

    # Collect all failed phases with their logs
    local -a failed_phases=()
    local -a failed_logs=()
    declare -A error_patterns=(
        ["typescript"]="TypeScript.*error|Cannot find name|error TS"
        ["ui_bundle"]="UI bundle.*stale|bundle.*outdated|Bundle Status:.*stale"
        ["build_failure"]="build.*failed|compilation.*failed|ELIFECYCLE"
        ["test_failure"]="tests? failed|❌.*failed"
        ["dependency"]="not_installed|not healthy|missing"
        ["timeout"]="timed out|timeout|exceeded.*time"
        ["api_port"]="Unable to resolve API_PORT|API.*not responding|HTTP 502|Bad Gateway"
    )

    for item in "${TESTING_RUNNER_EXECUTION_ITEMS[@]}"; do
        if [[ "$item" == phase:* ]]; then
            local status="${TESTING_RUNNER_STATUS[$item]}"
            if [ "$status" != "passed" ] && [ "$status" != "skipped" ]; then
                local phase_name="${item#phase:}"
                local log_path="${TESTING_RUNNER_LOG_PATH[$item]:-}"
                failed_phases+=("$phase_name")
                failed_logs+=("$log_path")
            fi
        fi
    done

    # =========================================================================
    # CONSOLIDATED ERROR DIGEST
    # =========================================================================
    echo "╔═══════════════════════════════════════════════════════════════╗"
    echo "║  ERROR DIGEST                                                 ║"
    echo "╚═══════════════════════════════════════════════════════════════╝"
    echo ""

    # Extract and display specific error messages for each failed phase
    for i in "${!failed_phases[@]}"; do
        local phase="${failed_phases[$i]}"
        local log="${failed_logs[$i]}"

        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo "❌ Phase: ${phase^^}"
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

        if [ ! -f "$log" ]; then
            echo "   Error: Log file not found"
            echo "   File:  $log"
            echo ""
            continue
        fi

        # Extract specific error based on phase type
        case "$phase" in
            unit)
                # Extract Go/Node test failures
                if grep -qE "❌.*failed" "$log" 2>/dev/null; then
                    local go_failures=$(grep -E "❌.*\(.*passed.*failed\)" "$log" 2>/dev/null | head -3)
                    if [ -n "$go_failures" ]; then
                        echo "   Test Failures:"
                        echo "$go_failures" | sed 's/^/     /'
                        echo ""
                        echo "   Details:"
                        while IFS= read -r failure_line; do
                            if [[ "$failure_line" =~ coverage/unit/go/([^/]+)/README.md ]]; then
                                echo "     → ${TESTING_RUNNER_SCENARIO_DIR}/coverage/unit/go/${BASH_REMATCH[1]}/README.md"
                            fi
                        done < <(grep "coverage/unit" "$log" 2>/dev/null | head -5)
                    fi
                fi
                ;;
            integration)
                # Extract integration-specific errors
                if grep -qE "${error_patterns[api_port]}" "$log" 2>/dev/null; then
                    echo "   Cause:  API not available"
                    local error_msg=$(grep -E "Unable to resolve API_PORT|HTTP 502|API.*not responding" "$log" 2>/dev/null | head -1 | sed 's/^[[:space:]]*//')
                    echo "   Error:  $error_msg"
                    echo ""
                    echo "   Fix:    vrooli scenario restart ${TESTING_RUNNER_SCENARIO_NAME}"
                fi
                ;;
            performance)
                # Extract performance audit failures
                if grep -qE "Lighthouse.*failed|performance:.*FAIL" "$log" 2>/dev/null; then
                    echo "   Lighthouse Failures:"
                    grep -E "❌.*failed|performance:.*%.*FAIL|accessibility:.*%.*FAIL" "$log" 2>/dev/null | head -5 | sed 's/^/     /'
                    echo ""
                    local lighthouse_dir="${TESTING_RUNNER_SCENARIO_DIR}/test/artifacts/lighthouse"
                    echo "   Reports: $lighthouse_dir/"
                fi
                if grep -qE "UI bundle.*exceeds limit" "$log" 2>/dev/null; then
                    local bundle_msg=$(grep "UI bundle.*exceeds limit" "$log" | head -1)
                    echo "   Bundle:  $bundle_msg"
                fi
                ;;
            structure)
                # Extract structure validation errors
                if grep -qE "UI smoke.*failed|HTTP 502" "$log" 2>/dev/null; then
                    echo "   Cause:  UI smoke test failed"
                    local smoke_error=$(grep -A1 "UI smoke.*failed" "$log" 2>/dev/null | tail -1 | sed 's/^[[:space:]]*//')
                    echo "   Error:  $smoke_error"
                    echo ""
                    echo "   Fix:    Check if scenario is running properly"
                fi
                ;;
            *)
                # Generic error extraction
                local error_line=$(grep -E "\[ERROR\]|❌" "$log" 2>/dev/null | head -1 | sed 's/^\[ERROR\][[:space:]]*//' | sed 's/^[[:space:]]*//')
                if [ -n "$error_line" ]; then
                    echo "   Error:  ${error_line:0:90}"
                fi
                ;;
        esac

        echo "   Log:    $log"
        echo ""
    done

    # =========================================================================
    # CRITICAL PATH ANALYSIS
    # =========================================================================
    echo "╔═══════════════════════════════════════════════════════════════╗"
    echo "║  CRITICAL PATH ANALYSIS                                       ║"
    echo "╚═══════════════════════════════════════════════════════════════╝"

    # Analyze errors and determine root cause
    local primary_blocker=""
    local primary_blocker_phase=""
    local primary_blocker_details=""
    local -a secondary_issues=()
    local -a blocked_phases=()

    for i in "${!failed_phases[@]}"; do
        local phase="${failed_phases[$i]}"
        local log="${failed_logs[$i]}"

        if [ ! -f "$log" ]; then
            continue
        fi

        # Check for TypeScript errors (high priority blocker)
        if grep -qE "${error_patterns[typescript]}" "$log" 2>/dev/null; then
            if [ -z "$primary_blocker" ]; then
                primary_blocker="TypeScript compilation error"
                primary_blocker_phase="$phase"
                local ts_error=$(grep -E "error TS[0-9]+|Cannot find name" "$log" 2>/dev/null | head -1 | sed 's/^[[:space:]]*//')
                primary_blocker_details="$ts_error"
                blocked_phases+=("integration" "performance")
            fi
        # Check for UI bundle staleness
        elif grep -qE "${error_patterns[ui_bundle]}" "$log" 2>/dev/null; then
            if [ -z "$primary_blocker" ] || [ "$primary_blocker_phase" = "$phase" ]; then
                primary_blocker="Stale UI bundle"
                primary_blocker_phase="$phase"
                primary_blocker_details="UI bundle is out of date with source files"
                blocked_phases+=("integration" "performance")
            fi
        # Check for build failures
        elif grep -qE "${error_patterns[build_failure]}" "$log" 2>/dev/null; then
            if [ -z "$primary_blocker" ]; then
                primary_blocker="Build failure"
                primary_blocker_phase="$phase"
                local build_error=$(grep -E "error|failed" "$log" 2>/dev/null | grep -v "^\[" | head -1 | sed 's/^[[:space:]]*//')
                primary_blocker_details="$build_error"
            fi
        # Test failures are secondary issues
        elif grep -qE "${error_patterns[test_failure]}" "$log" 2>/dev/null; then
            local test_count=$(grep -cE "❌|failed" "$log" 2>/dev/null || echo "?")
            secondary_issues+=("$phase: $test_count test failures")
        fi
    done

    # Determine which phases were blocked by the primary issue
    local -a actually_blocked=()
    for blocked in "${blocked_phases[@]}"; do
        for failed in "${failed_phases[@]}"; do
            if [ "$blocked" = "$failed" ] && [ "$blocked" != "$primary_blocker_phase" ]; then
                actually_blocked+=("$failed")
            fi
        done
    done

    if [ -n "$primary_blocker" ]; then
        echo ""
        echo "  🔴 PRIMARY BLOCKER: $primary_blocker"
        echo "     Phase: $primary_blocker_phase"
        if [ -n "$primary_blocker_details" ]; then
            local truncated_details="${primary_blocker_details:0:100}"
            echo "     Details: $truncated_details"
        fi
        if [ ${#actually_blocked[@]} -gt 0 ]; then
            local blocked_str=$(IFS=', '; echo "${actually_blocked[*]}")
            echo "     Impact: Blocks ${#actually_blocked[@]} phase(s): $blocked_str"
        fi
        echo ""
    fi

    if [ ${#secondary_issues[@]} -gt 0 ]; then
        echo "  🟡 SECONDARY ISSUES:"
        for issue in "${secondary_issues[@]}"; do
            echo "     • $issue"
        done
        echo ""
    fi

    # =========================================================================
    # QUICK FIX GUIDE (RECOMMENDED ACTIONS)
    # =========================================================================
    echo "╔═══════════════════════════════════════════════════════════════╗"
    echo "║  QUICK FIX GUIDE                                              ║"
    echo "╚═══════════════════════════════════════════════════════════════╝"
    echo ""
    local action_num=1

    if [[ "$primary_blocker" == *"TypeScript"* ]]; then
        echo "  ${action_num}. Fix TypeScript compilation error"
        ((action_num++))
        for log in "${failed_logs[@]}"; do
            if [ -f "$log" ]; then
                local file_line=$(grep -E "\.ts.*error TS|\.ts\([0-9]+,[0-9]+\)" "$log" 2>/dev/null | head -1 | sed 's/:.*$//' | sed 's/^[[:space:]]*//')
                if [ -n "$file_line" ]; then
                    echo "     Check:   $file_line"
                    break
                fi
            fi
        done
        echo "     Action:  Fix compilation errors in TypeScript files"
        echo "     Rerun:   make test"
        echo ""
    elif [[ "$primary_blocker" == *"bundle"* ]] || [[ "$primary_blocker" == *"Build"* ]]; then
        echo "  ${action_num}. Rebuild and restart scenario"
        ((action_num++))
        echo "     Run:     vrooli scenario restart ${TESTING_RUNNER_SCENARIO_NAME}"
        echo "     Effect:  Rebuilds UI and unblocks ${#actually_blocked[@]} phase(s)"
        echo "     Rerun:   make test"
        echo ""
    fi

    if [ ${#secondary_issues[@]} -gt 0 ]; then
        echo "  ${action_num}. Address test failures"
        ((action_num++))
        for i in "${!failed_phases[@]}"; do
            local phase="${failed_phases[$i]}"
            local log="${failed_logs[$i]}"
            if [ -f "$log" ] && grep -qE "${error_patterns[test_failure]}" "$log" 2>/dev/null; then
                local coverage_readme=""
                # Check common coverage paths - use absolute paths
                local coverage_unit_go="${TESTING_RUNNER_SCENARIO_DIR}/coverage/unit/go"
                if [ -d "$coverage_unit_go" ]; then
                    coverage_readme=$(find "$coverage_unit_go" -name README.md 2>/dev/null | grep -E "${phase}|executor|database" | head -1)
                fi
                if [ -n "$coverage_readme" ]; then
                    echo "     Review:  $coverage_readme"
                else
                    echo "     Review:  $log"
                fi
            fi
        done
        echo "     Run individually:"
        for phase in "${failed_phases[@]}"; do
            case "$phase" in
                unit)
                    echo "       • cd ${TESTING_RUNNER_SCENARIO_DIR}/api && go test -v ./... -run <TestName>"
                    echo "       • cd ${TESTING_RUNNER_SCENARIO_DIR}/ui && pnpm test <test-file>"
                    ;;
                integration)
                    echo "       • vrooli test integration ${TESTING_RUNNER_SCENARIO_NAME}"
                    ;;
                performance)
                    echo "       • vrooli test performance ${TESTING_RUNNER_SCENARIO_NAME}"
                    ;;
            esac
        done
        echo ""
    fi

    # =========================================================================
    # PHASE-SPECIFIC DEBUG GUIDES
    # =========================================================================
    echo "╔═══════════════════════════════════════════════════════════════╗"
    echo "║  PHASE-SPECIFIC DEBUG GUIDES                                  ║"
    echo "╚═══════════════════════════════════════════════════════════════╝"
    echo ""

    for phase in "${failed_phases[@]}"; do
        case "$phase" in
            unit)
                echo "━━━ UNIT PHASE ━━━"
                echo "Common issues:"
                echo "  • Database migrations not applied"
                echo "  • Missing test dependencies"
                echo "  • Stale module cache"
                echo ""
                echo "Quick fixes:"
                echo "  → Check migrations:   ls ${TESTING_RUNNER_SCENARIO_DIR}/initialization/postgres/migrations/"
                echo "  → Clear Go cache:     go clean -testcache"
                echo "  → Reinstall modules:  cd ${TESTING_RUNNER_SCENARIO_DIR}/ui && pnpm install"
                echo ""
                ;;
            integration)
                echo "━━━ INTEGRATION PHASE ━━━"
                echo "Common issues:"
                echo "  • Scenario not running (API_PORT unavailable)"
                echo "  • Stale UI bundle"
                echo "  • Missing BAS CLI"
                echo ""
                echo "Quick fixes:"
                local scenarios_root="${TESTING_RUNNER_SCENARIO_DIR}/.."
                echo "  → Restart scenario:   vrooli scenario restart ${TESTING_RUNNER_SCENARIO_NAME}"
                echo "  → Check scenario:     vrooli scenario status ${TESTING_RUNNER_SCENARIO_NAME}"
                echo "  → Install BAS:        cd ${scenarios_root}/browser-automation-studio && make install-cli"
                echo ""
                ;;
            performance)
                echo "━━━ PERFORMANCE PHASE ━━━"
                echo "Common issues:"
                echo "  • Lighthouse audits below threshold"
                echo "  • UI bundle too large"
                echo "  • Slow page load times"
                echo ""
                echo "Quick fixes:"
                echo "  → View reports:       open ${TESTING_RUNNER_SCENARIO_DIR}/test/artifacts/lighthouse/*.html"
                echo "  → Analyze bundle:     cd ${TESTING_RUNNER_SCENARIO_DIR}/ui && pnpm run analyze"
                echo "  → Check bundle size:  ls -lh ${TESTING_RUNNER_SCENARIO_DIR}/ui/dist/assets/*.js"
                echo ""
                ;;
            structure)
                echo "━━━ STRUCTURE PHASE ━━━"
                echo "Common issues:"
                echo "  • UI smoke test failing (API not responding)"
                echo "  • Missing required files"
                echo "  • Invalid JSON configuration"
                echo ""
                echo "Quick fixes:"
                local app_root
                app_root=$(cd "${TESTING_RUNNER_SCENARIO_DIR}/../.." && pwd)
                echo "  → Check API logs:     tail -f ${app_root}/logs/${TESTING_RUNNER_SCENARIO_NAME}-api.log"
                echo "  → Restart scenario:   vrooli scenario restart ${TESTING_RUNNER_SCENARIO_NAME}"
                echo "  → Validate JSON:      jq . ${TESTING_RUNNER_SCENARIO_DIR}/.vrooli/service.json"
                echo ""
                ;;
        esac
    done

    # =========================================================================
    # DETAILED REFERENCES
    # =========================================================================
    echo "╔═══════════════════════════════════════════════════════════════╗"
    echo "║  DETAILED REFERENCES                                          ║"
    echo "╚═══════════════════════════════════════════════════════════════╝"
    echo ""
    echo "  Logs:"
    local artifacts_dir="${TESTING_RUNNER_SCENARIO_DIR}/test/artifacts"
    echo "    Phase logs:      ${artifacts_dir}/"
    local coverage_unit="${TESTING_RUNNER_SCENARIO_DIR}/coverage/unit"
    if [ -d "$coverage_unit" ]; then
        echo "    Coverage:        ${coverage_unit}/"
    fi
    local requirements_index="${TESTING_RUNNER_SCENARIO_DIR}/requirements/index.json"
    if [ -f "$requirements_index" ]; then
        echo "    Requirements:    $requirements_index"
    fi
    echo ""
    echo "  Documentation:"
    local app_root
    app_root=$(cd "${TESTING_RUNNER_SCENARIO_DIR}/../.." && pwd)
    echo "    Architecture:    ${app_root}/docs/testing/architecture/PHASED_TESTING.md"
    echo "    Quick Start:     ${app_root}/docs/testing/guides/requirement-tracking-quick-start.md"
    echo "    UI Testing:      ${app_root}/docs/testing/guides/ui-automation-with-bas.md"
    echo ""
}
