#!/bin/bash
# Universal unit test runner that runs all detected language tests
# Sources individual language runners and executes them
set -euo pipefail

# Get the directory where this script is located
UNIT_TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source all language-specific runners
source "$UNIT_TEST_DIR/go.sh"
source "$UNIT_TEST_DIR/node.sh"
source "$UNIT_TEST_DIR/python.sh"

# Run all unit tests for a scenario
# Usage: testing::unit::run_all_tests [options]
# Options:
#   --go-dir PATH               Directory for Go tests (default: api)
#   --node-dir PATH             Directory for Node.js tests (default: ui)
#   --python-dir PATH           Directory for Python tests (default: .)
#   --skip-go                   Skip Go tests
#   --skip-node                 Skip Node.js tests
#   --skip-python               Skip Python tests
#   --verbose                   Verbose output for all tests
#   --fail-fast                 Stop on first test failure
#   --coverage-warn PERCENT     Coverage warning threshold (default: 80)
#   --coverage-error PERCENT    Coverage error threshold (default: 50)
testing::unit::run_all_tests() {
    local go_dir="api"
    local node_dir="ui"
    local python_dir="."
    local skip_go=false
    local skip_node=false
    local skip_python=false
    local verbose=false
    local fail_fast=false
    local coverage_warn_threshold=80
    local coverage_error_threshold=50
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --go-dir)
                go_dir="$2"
                shift 2
                ;;
            --node-dir)
                node_dir="$2"
                shift 2
                ;;
            --python-dir)
                python_dir="$2"
                shift 2
                ;;
            --skip-go)
                skip_go=true
                shift
                ;;
            --skip-node)
                skip_node=true
                shift
                ;;
            --skip-python)
                skip_python=true
                shift
                ;;
            --verbose)
                verbose=true
                shift
                ;;
            --fail-fast)
                fail_fast=true
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
    
    local error_count=0
    local test_count=0
    local skipped_count=0

    local scenario_root=$(pwd)
    local coverage_root_dir="$scenario_root/coverage/test-genie"
    local coverage_entries=()
    local jq_available=true

    # Ensure previous aggregated output is cleared without touching other coverage assets
    rm -rf "$coverage_root_dir"

    if ! command -v jq >/dev/null 2>&1; then
        jq_available=false
        echo "⚠️  jq not found; coverage aggregation report will be skipped" >&2
    fi
    
    echo "🧪 Running all unit tests..."
    echo ""
    
    # Run Go tests
    if [ "$skip_go" = false ]; then
        local go_args="--dir $go_dir --coverage-warn $coverage_warn_threshold --coverage-error $coverage_error_threshold"
        if [ "$verbose" = true ]; then
            go_args="$go_args --verbose"
        fi
        
        if testing::unit::run_go_tests $go_args; then
            ((test_count++)) || true

            if [ "${TESTING_GO_COVERAGE_COLLECTED:-}" = "true" ]; then
                local go_dest_dir="$coverage_root_dir/go"
                mkdir -p "$go_dest_dir"

                local go_profile_rel=""
                if [ -n "${TESTING_GO_COVERAGE_PROFILE:-}" ] && [ -f "${TESTING_GO_COVERAGE_PROFILE}" ]; then
                    cp "${TESTING_GO_COVERAGE_PROFILE}" "$go_dest_dir/coverage.out"
                    go_profile_rel="coverage/test-genie/go/coverage.out"
                fi

                local go_html_rel=""
                if [ -n "${TESTING_GO_COVERAGE_HTML:-}" ] && [ -f "${TESTING_GO_COVERAGE_HTML}" ]; then
                    cp "${TESTING_GO_COVERAGE_HTML}" "$go_dest_dir/coverage.html"
                    go_html_rel="coverage/test-genie/go/coverage.html"
                fi

                if [ "$jq_available" = true ]; then
                    local go_stat_arg="${TESTING_GO_COVERAGE_PERCENT:-null}"
                    local go_entry
                    go_entry=$(jq -n \
                        --arg lang "go" \
                        --arg profile "$go_profile_rel" \
                        --arg html "$go_html_rel" \
                        --argjson statements "$go_stat_arg" \
                        '{
                            language: $lang,
                            metrics: { statements: $statements },
                            artifacts: (
                                {}
                                | if ($profile | length) > 0 then .cover_profile = $profile else . end
                                | if ($html | length) > 0 then .html_report = $html else . end
                            )
                        }')
                    coverage_entries+=("$go_entry")
                fi
            fi
        else
            ((error_count++)) || true
            if [ "$fail_fast" = true ]; then
                echo "❌ Stopping due to Go test failure (fail-fast enabled)"
                return 1
            fi
        fi
        echo ""
    else
        ((skipped_count++)) || true
    fi
    
    # Run Node.js tests
    if [ "$skip_node" = false ]; then
        local node_args="--dir $node_dir --coverage-warn $coverage_warn_threshold --coverage-error $coverage_error_threshold"
        if [ "$verbose" = true ]; then
            node_args="$node_args --verbose"
        fi
        
        if testing::unit::run_node_tests $node_args; then
            ((test_count++)) || true

            if [ "${TESTING_NODE_COVERAGE_COLLECTED:-}" = "true" ]; then
                local node_dest_dir="$coverage_root_dir/node"
                mkdir -p "$node_dest_dir"

                local node_summary_rel=""
                if [ -n "${TESTING_NODE_COVERAGE_SUMMARY_PATH:-}" ] && [ -f "${TESTING_NODE_COVERAGE_SUMMARY_PATH}" ]; then
                    cp "${TESTING_NODE_COVERAGE_SUMMARY_PATH}" "$node_dest_dir/coverage-summary.json"
                    node_summary_rel="coverage/test-genie/node/coverage-summary.json"
                fi

                local node_lcov_rel=""
                if [ -n "${TESTING_NODE_COVERAGE_LCOV_PATH:-}" ] && [ -f "${TESTING_NODE_COVERAGE_LCOV_PATH}" ]; then
                    cp "${TESTING_NODE_COVERAGE_LCOV_PATH}" "$node_dest_dir/lcov.info"
                    node_lcov_rel="coverage/test-genie/node/lcov.info"
                fi

                if [ "$jq_available" = true ]; then
                    local node_metrics_json='{}'
                    if [ -n "${TESTING_NODE_COVERAGE_TOTALS_JSON:-}" ]; then
                        node_metrics_json="${TESTING_NODE_COVERAGE_TOTALS_JSON}"
                    elif [ -n "${TESTING_NODE_COVERAGE_PERCENT:-}" ]; then
                        node_metrics_json="{\"statements\": ${TESTING_NODE_COVERAGE_PERCENT}}"
                    fi

                    local node_entry
                    node_entry=$(jq -n \
                        --arg lang "node" \
                        --arg summary "$node_summary_rel" \
                        --arg lcov "$node_lcov_rel" \
                        --argjson metrics "$node_metrics_json" \
                        '{
                            language: $lang,
                            metrics: $metrics,
                            artifacts: (
                                {}
                                | if ($summary | length) > 0 then .summary = $summary else . end
                                | if ($lcov | length) > 0 then .lcov = $lcov else . end
                            )
                        }')
                    coverage_entries+=("$node_entry")
                fi
            fi
        else
            ((error_count++)) || true
            if [ "$fail_fast" = true ]; then
                echo "❌ Stopping due to Node.js test failure (fail-fast enabled)"
                return 1
            fi
        fi
        echo ""
    else
        ((skipped_count++)) || true
    fi
    
    # Run Python tests
    if [ "$skip_python" = false ]; then
        local python_args="--dir $python_dir"
        if [ "$verbose" = true ]; then
            python_args="$python_args --verbose"
        fi
        
        # Note: Python test runner doesn't have coverage thresholds implemented yet
        if testing::unit::run_python_tests $python_args; then
            ((test_count++)) || true
        else
            ((error_count++)) || true
            if [ "$fail_fast" = true ]; then
                echo "❌ Stopping due to Python test failure (fail-fast enabled)"
                return 1
            fi
        fi
        echo ""
    else
        ((skipped_count++)) || true
    fi
    
    if [ "$jq_available" = true ] && [ ${#coverage_entries[@]} -gt 0 ]; then
        mkdir -p "$coverage_root_dir"
        local languages_json
        languages_json=$(printf '%s\n' "${coverage_entries[@]}" | jq -s '.')
        local timestamp
        timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
        local aggregate_json
        aggregate_json=$(jq -n \
            --arg generated_at "$timestamp" \
            --argjson languages "$languages_json" \
            '{generated_at: $generated_at, languages: $languages}')
        local aggregate_path="$coverage_root_dir/aggregate.json"
        printf '%s\n' "$aggregate_json" > "$aggregate_path"
        local aggregate_rel="${aggregate_path#$scenario_root/}"
        echo "ℹ️  Coverage aggregate written to ${aggregate_rel:-$aggregate_path}"
    fi
    
    # Summary
    echo "📊 Unit Test Summary:"
    echo "   Tests passed: $test_count"
    echo "   Tests failed: $error_count"
    echo "   Tests skipped: $skipped_count"
    
    if [ $error_count -eq 0 ]; then
        echo ""
        echo "✅ All unit tests passed!"
        return 0
    else
        echo ""
        echo "❌ Some unit tests failed"
        return 1
    fi
}

# Export main function
export -f testing::unit::run_all_tests

# If script is executed directly (not sourced), run all tests
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    testing::unit::run_all_tests "$@"
fi
