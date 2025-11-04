#!/bin/bash
# Generic Go unit test runner for scenarios
# Can be sourced and used by any scenario's test suite
set -euo pipefail

declare -gA TESTING_GO_REQUIREMENT_STATUS=()
declare -gA TESTING_GO_REQUIREMENT_EVIDENCE=()

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
    cd "$api_dir"
    
    # Check if there are any test files
    local test_files=$(find . -name "*_test.go" -type f | wc -l)
    if [ "$test_files" -eq 0 ]; then
        echo "ℹ️  No Go test files (*_test.go) found"
        cd "$original_dir"
        return 0
    fi
    
    echo "📦 Downloading Go module dependencies..."
    if ! go mod download; then
        echo "❌ Failed to download Go dependencies"
        cd "$original_dir"
        return 1
    fi
    
    # Build test command
    local test_cmd="go test"
    if [ "$verbose" = true ]; then
        test_cmd="$test_cmd -v"
    fi
    test_cmd="$test_cmd ./... -timeout $timeout"
    if [ "$coverage" = true ]; then
        test_cmd="$test_cmd -cover -coverprofile=coverage.out"
    fi
    
    echo "🧪 Running Go tests..."

    local go_output_file
    go_output_file=$(mktemp)
    local go_exit=0

    set +e
    eval "$test_cmd" | tee "$go_output_file"
    go_exit=${PIPESTATUS[0]}
    set -e

    testing::unit::_go_collect_requirement_tags "$go_output_file"

    local enforce_requirements="${TESTING_REQUIREMENTS_ENFORCE:-${VROOLI_REQUIREMENTS_ENFORCE:-0}}"
    if [ "$enforce_requirements" = "1" ] && [ ${#TESTING_GO_REQUIREMENT_STATUS[@]} -eq 0 ]; then
        echo "⚠️  No REQ:<ID> tags detected in Go test output; add requirement tags to tie unit tests to coverage."
    fi

    if [ "$go_exit" -eq 0 ]; then
        echo "✅ Go unit tests completed successfully"

        # Display coverage summary if coverage file exists
        if [ "$coverage" = true ] && [ -f "coverage.out" ]; then
            echo ""
            echo "📊 Go Test Coverage Summary:"
            local coverage_line=$(go tool cover -func=coverage.out | tail -1)
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
                    rm -f "$go_output_file"
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
            local coverage_profile_rel="${api_dir%/}/coverage.out"
            declare -g TESTING_GO_COVERAGE_PROFILE="$coverage_profile_rel"
            # Generate HTML coverage report for manual inspection
            if go tool cover -html=coverage.out -o coverage.html 2>/dev/null; then
                local coverage_html_rel="${api_dir%/}/coverage.html"
                declare -g TESTING_GO_COVERAGE_HTML="$coverage_html_rel"
                echo "ℹ️  HTML coverage report generated: $coverage_html_rel"
            fi
        fi

        rm -f "$go_output_file"
        cd "$original_dir"
        return 0
    else
        echo "❌ Go unit tests failed"
        rm -f "$go_output_file"
        cd "$original_dir"
        return 1
    fi
}

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
