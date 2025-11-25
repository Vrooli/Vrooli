#!/bin/bash
# Generic Go unit test runner for scenarios with structured output formatting
# Can be sourced and used by any scenario's test suite
set -euo pipefail

declare -gA TESTING_GO_REQUIREMENT_STATUS=()
declare -gA TESTING_GO_REQUIREMENT_EVIDENCE=()

# ============================================================================
# Output Formatting - Provides clean console output with rich debug files
# ============================================================================
# This formatter provides:
# - Clean console output (2 lines per package max, similar to integration tests)
# - Structured debug files in coverage/unit/ for failures
# - README.md files explaining failures with actionable guidance
# - Smart compilation error grouping
# - Cross-reference to related test failures
#
# Design Philosophy:
# - Console is a dashboard (scannable at a glance)
# - Files contain debug details (screenshots of the problem)
# - READMEs explain what went wrong and how to fix it
# ============================================================================

_testing_go__create_unit_coverage_dirs() {
    local scenario_root="${1:-$(pwd)}"
    local coverage_root="$scenario_root/coverage/unit"

    # Clear previous run's failure reports (keep other artifacts)
    rm -rf "$coverage_root/go" "$coverage_root/compilation" "$coverage_root/README.md" 2>/dev/null || true

    mkdir -p "$coverage_root"/{go,compilation}

    echo "$coverage_root"
}

_testing_go__group_compilation_errors() {
    local build_output="$1"
    local output_dir="$2"

    if [ ! -s "$build_output" ]; then
        return 0
    fi

    # Check if there are compilation errors
    if ! grep -q "^#" "$build_output" 2>/dev/null; then
        return 0
    fi

    mkdir -p "$output_dir"

    # Group errors by package and error message
    local current_package=""
    local -A package_errors=()
    local -A error_locations=()

    while IFS= read -r line; do
        # Package header: # github.com/...
        if [[ "$line" =~ ^#[[:space:]]+(.*) ]]; then
            current_package="${BASH_REMATCH[1]}"
            continue
        fi

        # Error line: path/file.go:line:col: error message
        if [[ "$line" =~ ^([^:]+):([0-9]+):([0-9]+):[[:space:]]*(.*)$ ]]; then
            local file="${BASH_REMATCH[1]}"
            local lineno="${BASH_REMATCH[2]}"
            local colno="${BASH_REMATCH[3]}"
            local error_msg="${BASH_REMATCH[4]}"

            if [ -n "$current_package" ]; then
                # Group by error message pattern
                local error_key=$(echo "$error_msg" | sed 's/[0-9]\+/N/g; s/"[^"]*"/"..."/g')
                package_errors["$current_package|$error_key"]+="$file:$lineno:$colno: $error_msg"$'\n'

                if [ -z "${error_locations[$current_package|$error_key]:-}" ]; then
                    error_locations["$current_package|$error_key"]="$file:$lineno"
                else
                    error_locations["$current_package|$error_key"]+=", $file:$lineno"
                fi
            fi
        fi
    done < "$build_output"

    # Generate per-package compilation error files
    local -A packages_seen=()
    for key in "${!package_errors[@]}"; do
        local package="${key%%|*}"
        local package_safe=$(echo "$package" | tr '/' '-')

        if [ -z "${packages_seen[$package]:-}" ]; then
            packages_seen[$package]=1

            local package_file="$output_dir/${package_safe}.txt"
            {
                echo "Compilation Errors in Package: $package"
                echo "=============================================="
                echo ""
            } > "$package_file"
        fi

        local package_file="$output_dir/${package_safe}.txt"
        local error_pattern="${key#*|}"
        local locations="${error_locations[$key]}"
        local errors="${package_errors[$key]}"

        {
            echo "Error Pattern:"
            echo "$error_pattern"
            echo ""
            echo "Locations: $locations"
            echo ""
            echo "Full Errors:"
            echo "$errors"
            echo ""
        } >> "$package_file"
    done

    # Generate compilation README
    if [ ${#packages_seen[@]} -gt 0 ]; then
        local readme="$output_dir/README.md"
        {
            echo "# Compilation Failures"
            echo ""
            echo "## Summary"
            echo ""
            echo "${#packages_seen[@]} package(s) failed to compile."
            echo ""
            echo "## Packages"
            echo ""
            for pkg in "${!packages_seen[@]}"; do
                local pkg_safe=$(echo "$pkg" | tr '/' '-')
                echo "- \`$pkg\` - see [${pkg_safe}.txt](./${pkg_safe}.txt)"
            done
            echo ""
            echo "## Next Steps"
            echo ""
            echo "1. Fix compilation errors in the order they appear"
            echo "2. Run \`go build ./...\` to verify fixes"
            echo "3. Rerun tests: \`vrooli test unit $(basename "$(pwd)")\`"
        } > "$readme"
    fi

    # Return count of failed packages
    echo "${#packages_seen[@]}"
}

_testing_go__parse_test_json() {
    local json_output="$1"
    local coverage_root="$2"

    # Parse go test -json output and create structured reports
    declare -gA _go_package_status=()
    declare -gA _go_package_output=()
    declare -gA _go_package_duration=()
    declare -gA _go_package_coverage=()
    declare -gA _go_test_status=()
    declare -gA _go_test_output=()
    declare -ga _go_packages_order=()
    declare -gA _go_package_seen=()

    local current_package=""
    local current_test=""

    while IFS= read -r line; do
        if [ -z "$line" ]; then
            continue
        fi

        local action=$(echo "$line" | jq -r '.Action // empty' 2>/dev/null || true)
        local package=$(echo "$line" | jq -r '.Package // empty' 2>/dev/null || true)
        local test=$(echo "$line" | jq -r '.Test // empty' 2>/dev/null || true)
        local output=$(echo "$line" | jq -r '.Output // empty' 2>/dev/null || true)
        local elapsed=$(echo "$line" | jq -r '.Elapsed // 0' 2>/dev/null || echo "0")

        if [ -z "$action" ]; then
            continue
        fi

        # Track package (deduplicate using separate array)
        if [ -n "$package" ] && [ -z "${_go_package_seen[$package]:-}" ]; then
            _go_packages_order+=("$package")
            _go_package_seen["$package"]=1
        fi

        # Accumulate output
        if [ -n "$output" ] && [ -n "$package" ]; then
            if [ -n "$test" ]; then
                _go_test_output["$package|$test"]+="$output"
            else
                _go_package_output["$package"]+="$output"
            fi
        fi

        # Handle test completion
        if [ -n "$test" ]; then
            case "$action" in
                pass)
                    _go_test_status["$package|$test"]="passed"
                    ;;
                fail)
                    _go_test_status["$package|$test"]="failed"
                    ;;
                skip)
                    _go_test_status["$package|$test"]="skipped"
                    ;;
            esac
        fi

        # Handle package completion
        if [ -z "$test" ]; then
            case "$action" in
                pass)
                    _go_package_status["$package"]="passed"
                    _go_package_duration["$package"]="$elapsed"
                    ;;
                fail)
                    _go_package_status["$package"]="failed"
                    _go_package_duration["$package"]="$elapsed"
                    ;;
                skip)
                    _go_package_status["$package"]="skipped"
                    _go_package_duration["$package"]="$elapsed"
                    ;;
            esac
        fi
    done < "$json_output"

    # Extract coverage from package output
    for package in "${_go_packages_order[@]}"; do
        local pkg_output="${_go_package_output[$package]:-}"
        if [ -n "$pkg_output" ]; then
            local coverage=$(echo "$pkg_output" | grep -oP 'coverage: \K[0-9.]+(?=% of statements)' | head -1 || echo "")
            if [ -n "$coverage" ]; then
                _go_package_coverage["$package"]="$coverage"
            fi
        fi
    done
}

_testing_go__create_failure_reports() {
    local coverage_root="$1"
    local go_dir="$2"
    local scenario_root="${3:-$(pwd)}"

    for package in "${_go_packages_order[@]}"; do
        local status="${_go_package_status[$package]:-unknown}"

        if [ "$status" != "failed" ]; then
            continue
        fi

        # Create package failure directory
        # Use last 2 path components to avoid collisions (e.g., "automation-executor" from "github.com/.../automation/executor")
        local package_safe=$(echo "$package" | awk -F'/' '{if (NF >= 2) print $(NF-1) "-" $NF; else print $NF}')
        local pkg_dir="$coverage_root/go/$package_safe"
        mkdir -p "$pkg_dir/tests"

        # Save full package output
        local pkg_output="${_go_package_output[$package]:-}"
        if [ -n "$pkg_output" ]; then
            echo "$pkg_output" > "$pkg_dir/output.txt"
        fi

        # Count test results
        local test_passed=0
        local test_failed=0
        local test_skipped=0
        local failed_tests=()

        for key in "${!_go_test_status[@]}"; do
            if [[ "$key" == "$package|"* ]]; then
                local test_name="${key#$package|}"
                local test_status="${_go_test_status[$key]}"

                case "$test_status" in
                    passed) ((test_passed++)) || true ;;
                    failed)
                        ((test_failed++)) || true
                        failed_tests+=("$test_name")
                        ;;
                    skipped) ((test_skipped++)) || true ;;
                esac
            fi
        done

        # Create per-test failure reports
        for test_name in "${failed_tests[@]}"; do
            local test_safe=$(echo "$test_name" | tr '/' '-')
            local test_dir="$pkg_dir/tests/$test_safe"
            mkdir -p "$test_dir"

            local test_output="${_go_test_output[$package|$test_name]:-}"
            if [ -n "$test_output" ]; then
                echo "$test_output" > "$test_dir/output.txt"
            fi

            # Generate test failure README
            local test_readme="$test_dir/README.md"
            local test_readme_abs="$(cd "$(dirname "$test_readme")" && pwd)/$(basename "$test_readme")"
            local output_txt_abs="$test_dir/output.txt"
            local output_exists="❌ missing"
            if [ -f "$output_txt_abs" ]; then
                output_exists="✅ exists"
            fi

            {
                echo "# Test Failure: $test_name"
                echo ""
                echo "**📍 Path Context**"
                echo "- **This Report:** \`$test_readme_abs\`"
                echo "- **Scenario Root:** \`$scenario_root\`"
                echo "- **Full Test Output:** \`$output_txt_abs\` ($output_exists)"
                echo ""
                echo "## Package"
                echo "\`$package\`"
                echo ""
                echo "## Error Output"
                echo "\`\`\`"
                echo "$test_output"
                echo "\`\`\`"
                echo ""
                echo "## Quick Reproduce"
                echo "\`\`\`bash"
                echo "# From anywhere, run:"
                echo "cd $scenario_root/$go_dir && \\"
                echo "  go test -v $package -run ^${test_name}$"
                echo "\`\`\`"
                echo ""
                echo "## Next Steps"
                echo "1. Read the error output above"
                echo "2. Run the reproduction command to debug interactively"
                echo "3. Fix the issue and rerun: \`vrooli test unit $(basename "$scenario_root")\`"
            } > "$test_readme"
        done

        # Generate package README
        local pkg_readme="$pkg_dir/README.md"
        local pkg_readme_abs="$(cd "$(dirname "$pkg_readme")" && pwd)/$(basename "$pkg_readme")"
        local pkg_output_abs="$pkg_dir/output.txt"
        local pkg_output_exists="❌ missing"
        if [ -f "$pkg_output_abs" ]; then
            pkg_output_exists="✅ exists"
        fi

        {
            echo "# Package Failure: $package"
            echo ""
            echo "**📍 Path Context**"
            echo "- **This Report:** \`$pkg_readme_abs\`"
            echo "- **Scenario Root:** \`$scenario_root\`"
            echo "- **Full Package Output:** \`$pkg_output_abs\` ($pkg_output_exists)"
            echo ""
            echo "## Summary"
            echo "- ✅ Passed: $test_passed"
            echo "- ❌ Failed: $test_failed"
            echo "- ⏭️  Skipped: $test_skipped"
            echo ""

            if [ ${#failed_tests[@]} -gt 0 ]; then
                echo "## Failed Tests"
                echo ""
                for test_name in "${failed_tests[@]}"; do
                    local test_safe=$(echo "$test_name" | tr '/' '-')
                    local test_readme_path="$pkg_dir/tests/$test_safe/README.md"
                    local test_exists="❌ missing"
                    if [ -f "$test_readme_path" ]; then
                        test_exists="✅ exists"
                    fi
                    echo "- \`$test_name\` - \`$test_readme_path\` ($test_exists)"
                done
                echo ""
            fi

            echo "## Quick Reproduce"
            echo "\`\`\`bash"
            echo "# From anywhere, run:"
            echo "cd $scenario_root/$go_dir && \\"
            echo "  go test -v $package"
            echo "\`\`\`"
            echo ""
            echo "## Next Steps"
            echo "1. Review failed test details above (click absolute paths to open)"
            echo "2. Run the reproduction command to debug interactively"
            echo "3. Fix failures and rerun: \`vrooli test unit $(basename "$scenario_root")\`"
        } > "$pkg_readme"
    done
}

_testing_go__print_console_summary() {
    local compilation_failed="$1"
    local coverage_root="$2"
    local scenario_root="${3:-$(pwd)}"

    echo ""

    if [ "$compilation_failed" -gt 0 ]; then
        # Show first few compilation errors inline for immediate context
        local compilation_dir="$coverage_root/compilation"
        local first_error_file=$(find "$compilation_dir" -name "*.txt" -type f 2>/dev/null | head -1)

        echo "🔨 Compilation: ❌ $compilation_failed package(s) failed to compile"
        echo ""

        if [ -n "$first_error_file" ] && [ -f "$first_error_file" ]; then
            echo "Top compilation errors:"
            # Extract first 5 unique error patterns
            grep "^[^:]\+:[0-9]\+:[0-9]\+:" "$first_error_file" 2>/dev/null | head -5 | while IFS= read -r line; do
                echo "  $line"
            done
            local total_errors=$(grep -c "^[^:]\+:[0-9]\+:[0-9]\+:" "$first_error_file" 2>/dev/null || echo "0")
            if [ "$total_errors" -gt 5 ]; then
                echo "  ... and $((total_errors - 5)) more errors"
            fi
            echo ""
        fi

        local readme_abs="$scenario_root/coverage/unit/compilation/README.md"
        echo "   ↳ Full details: $readme_abs"
        return 1
    fi

    echo "🔨 Compilation: ✅ All packages compile"

    local total_tests_passed=0
    local total_tests_failed=0
    local total_tests_skipped=0
    local total_packages_failed=0
    local has_failures=false

    # First pass: count test totals (not package totals)
    for package in "${_go_packages_order[@]}"; do
        local pkg_status="${_go_package_status[$package]:-unknown}"

        # Count individual tests within each package
        for key in "${!_go_test_status[@]}"; do
            if [[ "$key" == "$package|"* ]]; then
                local test_status="${_go_test_status[$key]}"
                case "$test_status" in
                    passed) ((total_tests_passed++)) || true ;;
                    failed) ((total_tests_failed++)) || true ;;
                    skipped) ((total_tests_skipped++)) || true ;;
                esac
            fi
        done

        # Track package-level failures for display
        if [ "$pkg_status" = "failed" ]; then
            ((total_packages_failed++)) || true
            has_failures=true
        fi
    done

    # Only show failures (if any)
    if [ "$has_failures" = true ]; then
        echo ""
        echo "🧪 Failed packages:"
        for package in "${_go_packages_order[@]}"; do
            local status="${_go_package_status[$package]:-unknown}"

            if [ "$status" = "failed" ]; then
                local duration="${_go_package_duration[$package]:-0}"

                # Shorten package name for display
                local pkg_display="$package"
                pkg_display="${pkg_display#github.com/vrooli/*/}" # Remove common prefix

                local duration_fmt=$(printf "%.3fs" "$duration" 2>/dev/null || echo "${duration}s")

                # Count actual test pass/fail for better messaging
                local test_passed=0
                local test_failed=0
                for key in "${!_go_test_status[@]}"; do
                    if [[ "$key" == "$package|"* ]]; then
                        local test_status="${_go_test_status[$key]}"
                        case "$test_status" in
                            passed) ((test_passed++)) || true ;;
                            failed) ((test_failed++)) || true ;;
                        esac
                    fi
                done

                local package_safe=$(echo "$package" | awk -F'/' '{if (NF >= 2) print $(NF-1) "-" $NF; else print $NF}')
                local readme_abs="$scenario_root/coverage/unit/go/$package_safe/README.md"
                echo "❌ $pkg_display ($duration_fmt, $test_passed passed, $test_failed failed)"
                echo "   ↳ Read $readme_abs"
            fi
        done
    fi

    echo ""
    echo "📊 Go Test Summary:"
    echo "   ✅ $total_tests_passed tests passed  ❌ $total_tests_failed tests failed  ⏭️ $total_tests_skipped tests skipped"
    echo "   📦 $total_packages_failed of ${#_go_packages_order[@]} packages had failures"

    if [ "$has_failures" = true ]; then
        local readme_abs="$scenario_root/coverage/unit/README.md"
        echo ""
        echo "❌ Unit tests failed - see $readme_abs"
        return 1
    fi

    return 0
}

_testing_go__generate_overall_readme() {
    local coverage_root="$1"
    local compilation_failed="$2"
    local has_test_failures="$3"
    local scenario_root="${4:-$(pwd)}"

    local readme="$coverage_root/README.md"
    local readme_abs="$(cd "$(dirname "$readme")" && pwd)/$(basename "$readme")"
    local full_output_abs="$coverage_root/full-output.txt"
    local full_output_exists="❌ missing"
    if [ -f "$full_output_abs" ]; then
        full_output_exists="✅ exists"
    fi

    {
        echo "# Unit Test Results"
        echo ""
        echo "**📍 Path Context**"
        echo "- **This Report:** \`$readme_abs\`"
        echo "- **Scenario Root:** \`$scenario_root\`"
        echo "- **Full Test Output:** \`$full_output_abs\` ($full_output_exists)"
        echo ""
        echo "Generated: $(date -u +"%Y-%m-%d %H:%M:%S UTC")"
        echo ""

        if [ "$compilation_failed" -gt 0 ]; then
            echo "## ❌ Compilation Failed"
            echo ""
            echo "$compilation_failed package(s) failed to compile. Fix compilation errors before tests can run."
            echo ""
            echo "See [compilation/README.md](./compilation/README.md) for details."
            echo ""
        elif [ "$has_test_failures" = "true" ]; then
            echo "## ❌ Tests Failed"
            echo ""
            echo "Some tests failed. See package-specific reports below."
            echo ""
            echo "## Failed Packages"
            echo ""
            for package in "${_go_packages_order[@]}"; do
                local status="${_go_package_status[$package]:-unknown}"
                if [ "$status" = "failed" ]; then
                    local package_safe=$(echo "$package" | awk -F'/' '{if (NF >= 2) print $(NF-1) "-" $NF; else print $NF}')
                    local pkg_readme_path="$coverage_root/go/$package_safe/README.md"
                    local pkg_exists="❌ missing"
                    if [ -f "$pkg_readme_path" ]; then
                        pkg_exists="✅ exists"
                    fi
                    echo "- \`$package\` - \`$pkg_readme_path\` ($pkg_exists)"
                fi
            done
            echo ""
        else
            echo "## ✅ All Tests Passed"
            echo ""
            echo "All compilation and tests completed successfully."
            echo ""
        fi

        echo "## Files"
        echo "- [full-output.txt](./full-output.txt) - Complete test output"
        if [ "$compilation_failed" -gt 0 ]; then
            echo "- [compilation/](./compilation/) - Compilation error details"
        fi
        if [ "$has_test_failures" = "true" ]; then
            echo "- [go/](./go/) - Test failure details by package"
        fi
        echo ""
        echo "## Next Steps"
        if [ "$compilation_failed" -gt 0 ]; then
            echo "1. Fix compilation errors in [compilation/README.md](./compilation/README.md)"
            echo "2. Run \`go build ./...\` to verify"
        elif [ "$has_test_failures" = "true" ]; then
            echo "1. Review failed packages above"
            echo "2. Run individual package tests for focused debugging"
            echo "3. Fix failures and rerun: \`vrooli test unit $(basename "$(pwd)")\`"
        else
            echo "No action required - all tests passing!"
        fi
    } > "$readme"
}

# ============================================================================
# Main Test Runner - Public Interface
# ============================================================================

# Run Go unit tests for a scenario
# Usage: testing::unit::run_go_tests [options]
# Options:
#   --dir PATH              Directory containing Go code (default: api)
#   --timeout SEC           Test timeout in seconds (default: 30)
#   --coverage              Generate coverage report (default: true)
#   --verbose               Verbose test output (default: false)
#   --coverage-warn PERCENT Coverage warning threshold (default: 80)
#   --coverage-error PERCENT Coverage error threshold (default: 50)
testing::unit::run_go_tests() {
    local api_dir="api"
    local timeout="30s"
    local coverage=true
    local verbose=false
    local coverage_warn_threshold=80
    local coverage_error_threshold=50

    # Reset any previously exported coverage metadata
    unset -v TESTING_GO_COVERAGE_COLLECTED TESTING_GO_COVERAGE_PERCENT \
        TESTING_GO_COVERAGE_PROFILE TESTING_GO_COVERAGE_HTML 2>/dev/null || true
    TESTING_GO_REQUIREMENT_STATUS=()
    TESTING_GO_REQUIREMENT_EVIDENCE=()

    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --dir)
                api_dir="$2"
                shift 2
                ;;
            --timeout)
                timeout="${2}s"
                shift 2
                ;;
            --no-coverage)
                coverage=false
                shift
                ;;
            --verbose)
                verbose=true
                shift
                ;;
            --coverage-warn)
                coverage_warn_threshold="$2"
                shift 2
                ;;
            --coverage-error)
                coverage_error_threshold="$2"
                shift 2
                ;;
            *)
                echo "Unknown option: $1"
                return 1
                ;;
        esac
    done

    echo "🐹 Running Go unit tests..."

    # Check if Go is available
    if ! command -v go >/dev/null 2>&1; then
        echo "❌ Go is not installed"
        return 1
    fi

    # Check if we have Go code
    if [ ! -d "$api_dir" ]; then
        echo "ℹ️  No $api_dir directory found, skipping Go tests"
        return 0
    fi

    if [ ! -f "$api_dir/go.mod" ]; then
        echo "ℹ️  No go.mod found in $api_dir, skipping Go tests"
        return 0
    fi

    # Save current directory and change to API directory
    local original_dir=$(pwd)
    local scenario_name=$(basename "$original_dir")
    cd "$api_dir"

    # Check if there are any test files
    local test_files=$(find . -name "*_test.go" -type f | wc -l)
    if [ "$test_files" -eq 0 ]; then
        echo "ℹ️  No Go test files (*_test.go) found"
        cd "$original_dir"
        return 0
    fi

    # Create coverage directory structure
    local coverage_root=$(_testing_go__create_unit_coverage_dirs "$original_dir")

    echo -n "📦 Downloading Go module dependencies... "
    if ! go mod download >"$coverage_root/go-mod-download.log" 2>&1; then
        echo "failed"
        echo "❌ Failed to download Go dependencies"
        cd "$original_dir"
        return 1
    fi
    echo "done"

    local build_output="$coverage_root/build-output.txt"
    local json_output="$coverage_root/test-output.json"
    local full_output="$coverage_root/full-output.txt"

    # Build test command
    local test_cmd="go test -json ./... -timeout $timeout"
    local coverage_profile_path="coverage.out"
    local coverage_html_path="coverage.html"

    if [ "$coverage" = true ]; then
        if [ -n "${TESTING_UNIT_WORK_DIR:-}" ]; then
            local go_work_dir="${TESTING_UNIT_WORK_DIR%/}/go"
            mkdir -p "$go_work_dir"
            coverage_profile_path="$go_work_dir/coverage.out"
            coverage_html_path="$go_work_dir/coverage.html"
        fi
        test_cmd="$test_cmd -cover -coverprofile=$coverage_profile_path"
    fi

    # Run tests and capture output
    local go_exit=0
    set +e

    # Try to build first to separate compilation errors from test failures
    echo -n "🔨 Building packages... "
    if ! go build ./... >"$build_output" 2>&1; then
        echo "failed"

        # Compilation failed - group errors and report
        local compilation_failed=$(_testing_go__group_compilation_errors "$build_output" "$coverage_root/compilation")

        if [ "$compilation_failed" -gt 0 ]; then
            _testing_go__print_console_summary "$compilation_failed" "$coverage_root" "$original_dir"
            _testing_go__generate_overall_readme "$coverage_root" "$compilation_failed" "false"

            cd "$original_dir"
            set -e
            return 1
        fi
    fi
    echo "success"

    # Compilation succeeded, now run tests (capture silently, show progress)
    echo -n "🧪 Running tests (this may take a while)... "
    eval "$test_cmd" >"$full_output" 2>&1
    go_exit=$?
    echo "done"

    # Copy full output to JSON file for parsing (they're the same in -json mode)
    cp "$full_output" "$json_output"
    set -e

    # Parse JSON output and create structured reports
    _testing_go__parse_test_json "$json_output" "$coverage_root"

    # Collect requirement tags from output
    testing::unit::_go_collect_requirement_tags "$full_output"

    local enforce_requirements="${TESTING_REQUIREMENTS_ENFORCE:-${VROOLI_REQUIREMENTS_ENFORCE:-0}}"
    if [ "$enforce_requirements" = "1" ] && [ ${#TESTING_GO_REQUIREMENT_STATUS[@]} -eq 0 ]; then
        echo "⚠️  No REQ:<ID> tags detected in Go test output; add requirement tags to tie unit tests to coverage."
    fi

    # Create failure reports for failed packages/tests
    if [ $go_exit -ne 0 ]; then
        _testing_go__create_failure_reports "$coverage_root" "$api_dir" "$original_dir"
    fi

    # Print console summary
    local has_test_failures="false"
    if ! _testing_go__print_console_summary "0" "$coverage_root" "$original_dir"; then
        has_test_failures="true"
    fi

    # Generate overall README
    _testing_go__generate_overall_readme "$coverage_root" "0" "$has_test_failures" "$original_dir"

    # Handle coverage reporting
    if [ "$coverage" = true ] && [ -f "$coverage_profile_path" ]; then
        echo ""
        echo "📊 Go Test Coverage Summary:"
        local coverage_line=$(go tool cover -func="$coverage_profile_path" | tail -1)
        echo "$coverage_line"

        # Extract coverage percentage
        local coverage_percent=$(echo "$coverage_line" | grep -o '[0-9]*\.[0-9]*%' | sed 's/%//' | head -1)
        if [ -n "$coverage_percent" ]; then
            local coverage_num=$(echo "$coverage_percent" | cut -d. -f1)

            # Check coverage thresholds
            echo ""
            if [ "$coverage_num" -lt "$coverage_error_threshold" ]; then
                echo "❌ ERROR: Go test coverage ($coverage_percent%) is below error threshold ($coverage_error_threshold%)"
                echo "   This indicates insufficient test coverage. Please add more comprehensive tests."
                cd "$original_dir"
                return 1
            elif [ "$coverage_num" -lt "$coverage_warn_threshold" ]; then
                echo "⚠️  WARNING: Go test coverage ($coverage_percent%) is below warning threshold ($coverage_warn_threshold%)"
                echo "   Consider adding more tests to improve code coverage."
            else
                echo "✅ Go test coverage ($coverage_percent%) meets quality thresholds"
            fi

            # Export parsed percentage for downstream aggregation
            declare -g TESTING_GO_COVERAGE_PERCENT="$coverage_percent"
        else
            echo "⚠️  WARNING: Could not parse coverage percentage from: $coverage_line"
        fi

        # Always record artifact paths so they can be copied later
        declare -g TESTING_GO_COVERAGE_COLLECTED="true"
        local profile_abs_path="$coverage_profile_path"
        if [[ "$profile_abs_path" != /* ]]; then
            profile_abs_path="$PWD/$profile_abs_path"
        fi
        declare -g TESTING_GO_COVERAGE_PROFILE="$profile_abs_path"

        if go tool cover -html="$coverage_profile_path" -o "$coverage_html_path" 2>/dev/null; then
            local html_abs_path="$coverage_html_path"
            if [[ "$html_abs_path" != /* ]]; then
                html_abs_path="$PWD/$html_abs_path"
            fi
            declare -g TESTING_GO_COVERAGE_HTML="$html_abs_path"
            local html_rel
            html_rel=$(testing::core::format_path "$html_abs_path" "${original_dir:-$scenario_root}")
            echo "ℹ️  HTML coverage report generated: $html_rel"
        fi
    fi

    cd "$original_dir"

    if [ "$has_test_failures" = "true" ]; then
        return 1
    fi

    return 0
}

# ============================================================================
# Requirement Tracking - Legacy Support
# ============================================================================

testing::unit::_go_status_rank() {
    case "$1" in
        failed) echo 3 ;;
        skipped) echo 2 ;;
        passed) echo 1 ;;
        *) echo 0 ;;
    esac
}

testing::unit::_go_store_requirement_result() {
    local requirement_id="$1"
    local status="$2"
    local detail="$3"

    if [ -z "$requirement_id" ] || [ -z "$status" ]; then
        return
    fi

    local current_status="${TESTING_GO_REQUIREMENT_STATUS["$requirement_id"]:-}"
    if [ -z "$current_status" ]; then
        TESTING_GO_REQUIREMENT_STATUS["$requirement_id"]="$status"
        TESTING_GO_REQUIREMENT_EVIDENCE["$requirement_id"]="$detail"
        return
    fi

    local new_rank
    local current_rank
    new_rank=$(testing::unit::_go_status_rank "$status")
    current_rank=$(testing::unit::_go_status_rank "$current_status")

    if [ "$new_rank" -gt "$current_rank" ]; then
        TESTING_GO_REQUIREMENT_STATUS["$requirement_id"]="$status"
        TESTING_GO_REQUIREMENT_EVIDENCE["$requirement_id"]="$detail"
    elif [ "$new_rank" -eq "$current_rank" ]; then
        local existing_detail="${TESTING_GO_REQUIREMENT_EVIDENCE["$requirement_id"]:-}"
        if [ -n "$existing_detail" ]; then
            TESTING_GO_REQUIREMENT_EVIDENCE["$requirement_id"]="${existing_detail}; ${detail}"
        else
            TESTING_GO_REQUIREMENT_EVIDENCE["$requirement_id"]="$detail"
        fi
    fi
}

testing::unit::_go_collect_requirement_tags() {
    local output_path="$1"
    if [ -z "$output_path" ] || [ ! -f "$output_path" ]; then
        return
    fi

    while IFS= read -r line; do
        [[ "$line" =~ ---\ (PASS|FAIL|SKIP):\ (.*)$ ]] || continue
        local raw_status="${BASH_REMATCH[1]}"
        local test_name="${BASH_REMATCH[2]}"
        local normalized_status
        case "$raw_status" in
            PASS) normalized_status="passed" ;;
            FAIL) normalized_status="failed" ;;
            SKIP) normalized_status="skipped" ;;
            *) normalized_status="" ;;
        esac
        if [ -z "$normalized_status" ]; then
            continue
        fi

        # Trim trailing timing information
        test_name="${test_name%% (*}"

        local tokens
        tokens=$(printf '%s\n' "$test_name" | grep -o 'REQ:[A-Za-z0-9_-]\+' || true)
        if [ -z "$tokens" ]; then
            continue
        fi

        local token
        for token in $tokens; do
            local requirement_id="${token#REQ:}"
            testing::unit::_go_store_requirement_result "$requirement_id" "$normalized_status" "Go test ${test_name}"
        done
    done < "$output_path"
}

# Export function for use by sourcing scripts
export -f testing::unit::run_go_tests
