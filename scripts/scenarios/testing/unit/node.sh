#!/bin/bash
# Generic Node.js unit test runner for scenarios
# Can be sourced and used by any scenario's test suite
set -euo pipefail

declare -gA TESTING_NODE_REQUIREMENT_STATUS=()
declare -gA TESTING_NODE_REQUIREMENT_EVIDENCE=()

# Run Node.js unit tests for a scenario
# Usage: testing::unit::run_node_tests [options]
# Options:
#   --dir PATH              Directory containing Node.js code (default: ui)
#   --timeout MS            Test timeout in milliseconds (default: 30000)
#   --test-cmd CMD          Custom test command (default: reads from package.json)
#   --verbose               Verbose test output (default: false)
#   --coverage-warn PERCENT Coverage warning threshold (default: 80)
#   --coverage-error PERCENT Coverage error threshold (default: 50)
testing::unit::run_node_tests() {
    local node_dir="ui"
    local timeout="30000"
    local test_cmd=""
    local verbose=false
    local coverage_warn_threshold=80
    local coverage_error_threshold=0
    local coverage_error_enforced=false
    if [ "${TESTING_NODE_COVERAGE_FAIL:-0}" = "1" ]; then
        coverage_error_enforced=true
    fi

    # Reset any previously exported coverage metadata
    unset -v TESTING_NODE_COVERAGE_COLLECTED TESTING_NODE_COVERAGE_PERCENT \
        TESTING_NODE_COVERAGE_TOTALS_JSON TESTING_NODE_COVERAGE_SUMMARY_PATH \
        TESTING_NODE_COVERAGE_LCOV_PATH TESTING_NODE_COVERAGE_DIR \
        TESTING_NODE_COVERAGE_LINK_PATH 2>/dev/null || true
    TESTING_NODE_REQUIREMENT_STATUS=()
    TESTING_NODE_REQUIREMENT_EVIDENCE=()

    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --dir)
                node_dir="$2"
                shift 2
                ;;
            --timeout)
                timeout="$2"
                shift 2
                ;;
            --test-cmd)
                test_cmd="$2"
                shift 2
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
    
    echo "📦 Running Node.js unit tests..."
    
    # Check if Node.js is available
    if ! command -v node >/dev/null 2>&1; then
        echo "❌ Node.js is not installed"
        return 1
    fi
    
    # Check if we have Node.js code
    if [ ! -d "$node_dir" ]; then
        echo "ℹ️  No $node_dir directory found, skipping Node.js tests"
        return 0
    fi
    
    if [ ! -f "$node_dir/package.json" ]; then
        echo "ℹ️  No package.json found in $node_dir, skipping Node.js tests"
        return 0
    fi
    
    # Save current directory and change to Node directory
    local original_dir=$(pwd)
    local node_abs_dir
    node_abs_dir=$(cd "$node_dir" && pwd)
    cd "$node_abs_dir"

    local coverage_root_on_disk="$node_abs_dir/coverage"
    local coverage_storage_root="$coverage_root_on_disk"
    if [ -n "${TESTING_UNIT_WORK_DIR:-}" ]; then
        coverage_storage_root="${TESTING_UNIT_WORK_DIR%/}/node"
        rm -rf "$coverage_root_on_disk"
        mkdir -p "$coverage_storage_root"
        ln -s "$coverage_storage_root" "$coverage_root_on_disk"
        declare -g TESTING_NODE_COVERAGE_LINK_PATH="$coverage_root_on_disk"
    else
        mkdir -p "$coverage_storage_root"
    fi
    declare -g TESTING_NODE_COVERAGE_DIR="$coverage_storage_root"
    
    # Check if test script is defined in package.json
    if [ -z "$test_cmd" ]; then
        test_cmd=$(node -e "const pkg=require('./package.json'); console.log(pkg.scripts?.test || '')" 2>/dev/null || echo "")
    fi
    
    if [ -z "$test_cmd" ] || [ "$test_cmd" == "echo \"Error: no test specified\" && exit 1" ]; then
        echo "ℹ️  No test script defined in package.json"
        
        # Check if there are test files anyway
        local test_files=$(find . \( -name "*.test.js" -o -name "*.spec.js" -o -name "*.test.ts" -o -name "*.spec.ts" \) -type f | wc -l)
        if [ "$test_files" -gt 0 ]; then
            echo "   Found $test_files test file(s) but no test script configured"
            echo "   💡 Add a test script to package.json"
        fi
        
        cd "$original_dir"
        return 0
    fi
    
    # Detect preferred package manager (pnpm/yarn/npm)
    local package_manager="npm"
    if [ -f "pnpm-lock.yaml" ] || grep -q '"packageManager"\s*:\s*"pnpm@' package.json 2>/dev/null; then
        package_manager="pnpm"
    elif [ -f "yarn.lock" ]; then
        package_manager="yarn"
    fi

    # Install dependencies if node_modules doesn't exist (or validate lockfiles)
    if [ ! -d "node_modules" ]; then
        echo "📦 Installing Node.js dependencies ($package_manager)..."
        case "$package_manager" in
            pnpm)
                if ! command -v pnpm >/dev/null 2>&1; then
                    echo "❌ pnpm is not installed"
                    cd "$original_dir"
                    return 1
                fi
                if ! pnpm install --frozen-lockfile --ignore-scripts --silent; then
                    echo "❌ Failed to install dependencies with pnpm"
                    cd "$original_dir"
                    return 1
                fi
                ;;
            yarn)
                if ! command -v yarn >/dev/null 2>&1; then
                    echo "❌ yarn is not installed"
                    cd "$original_dir"
                    return 1
                fi
                if ! yarn install --silent --frozen-lockfile >/dev/null 2>&1; then
                    echo "❌ Failed to install dependencies with yarn"
                    cd "$original_dir"
                    return 1
                fi
                ;;
            *)
                if ! command -v npm >/dev/null 2>&1; then
                    echo "❌ npm is not installed"
                    cd "$original_dir"
                    return 1
                fi
                if ! npm install --silent; then
                    echo "❌ Failed to install Node.js dependencies"
                    cd "$original_dir"
                    return 1
                fi
                ;;
        esac
    else
        case "$package_manager" in
            pnpm)
                if command -v pnpm >/dev/null 2>&1; then
                    pnpm install --frozen-lockfile --ignore-scripts --silent --offline >/dev/null 2>&1 || true
                fi
                ;;
            yarn)
                if command -v yarn >/dev/null 2>&1; then
                    yarn install --silent --frozen-lockfile >/dev/null 2>&1 || true
                fi
                ;;
            *)
                :
                ;;
        esac
    fi
    
    echo "🧪 Running Node.js tests..."

    local uses_vitest="false"
    local package_manifest="$node_abs_dir/package.json"
    if [ -f "$package_manifest" ]; then
        if command -v jq >/dev/null 2>&1; then
            uses_vitest=$(jq -r '[(.devDependencies.vitest // empty), (.dependencies.vitest // empty), (.peerDependencies.vitest // empty)]
                | map(select(. != null))
                | length
                | if . > 0 then "true" else "false" end' "$package_manifest" 2>/dev/null || echo "false")
        else
            if grep -q '"vitest"' "$package_manifest" 2>/dev/null; then
                uses_vitest="true"
            fi
        fi
    fi

    # Set test timeout environment variable if supported
    export JEST_TIMEOUT="$timeout"
    export MOCHA_TIMEOUT="$timeout"
    export VITEST_TIMEOUT="$timeout"
    
    # Run tests with coverage
    local test_output
    local test_success=false

    # Note: For vitest 2.x, detailed coverage config (reporter, thresholds, etc.)
    # must be in vite.config.ts. The --coverage flag is supported to enable coverage,
    # but --coverage.* CLI flags are not supported and will cause errors.
    # The test script in package.json should include --coverage for vitest.
    local -a runner_args=()
    if [ "$verbose" != true ]; then
        runner_args+=("--silent")
    fi

    case "$package_manager" in
        pnpm)
            test_output=$(pnpm test "${runner_args[@]}" 2>&1)
            ;;
        yarn)
            test_output=$(yarn test "${runner_args[@]}" 2>&1)
            ;;
        *)
            test_output=$(npm test -- "${runner_args[@]}" 2>&1)
            ;;
    esac

    local node_exit=$?
    testing::unit::_node_collect_requirement_tags "$test_output"

    local enforce_requirements="${TESTING_REQUIREMENTS_ENFORCE:-${VROOLI_REQUIREMENTS_ENFORCE:-0}}"
    if [ "$enforce_requirements" = "1" ] && [ ${#TESTING_NODE_REQUIREMENT_STATUS[@]} -eq 0 ]; then
        echo "⚠️  No REQ:<ID> tags detected in Node test output; add requirement tags to attribute coverage."
    fi

    if [ $node_exit -ne 0 ]; then
        echo "$test_output"
        echo "❌ Node.js unit tests failed"
        cd "$original_dir"
        return 1
    fi

    test_success=true
    echo "$test_output"

    if [ "$test_success" = true ]; then
        echo "✅ Node.js unit tests completed successfully"

        # Prefer structured coverage data; fall back to parsing stdout
        if [[ "$coverage_storage_root" != "$coverage_root_on_disk" ]]; then
            if [ ! -f "$coverage_storage_root/coverage-summary.json" ] && [ -f "$coverage_root_on_disk/coverage-summary.json" ]; then
                rm -rf "$coverage_storage_root"
                mkdir -p "$coverage_storage_root"
                cp -R "$coverage_root_on_disk/." "$coverage_storage_root/" 2>/dev/null || true
            fi
        fi

        local coverage_percent=""
        if [ -f "$coverage_storage_root/coverage-summary.json" ]; then
            coverage_percent=$(node -e "const summary=require('./coverage/coverage-summary.json'); const pct=summary?.total?.statements?.pct; if (typeof pct === 'number') { process.stdout.write(pct.toString()); }" 2>/dev/null || echo "")
        fi

        if [ -z "$coverage_percent" ]; then
            local coverage_line=$(echo "$test_output" | grep -E "(All files|Statements|Lines|Functions|Branches).*[0-9]+(\.[0-9]+)?%" | head -1)
            if [ -n "$coverage_line" ]; then
                coverage_percent=$(echo "$coverage_line" | grep -o '[0-9]\+\(\.[0-9]\+\)\?%' | head -1 | sed 's/%//')
            fi
        fi

        if [ -z "$coverage_percent" ]; then
            coverage_percent=$(echo "$test_output" | grep -o '[0-9]\+\(\.[0-9]\+\)\?%' | head -1 | sed 's/%//')
        fi

        if [ -n "$coverage_percent" ]; then
            local coverage_display
            coverage_display=$(printf '%.2f' "$coverage_percent" 2>/dev/null || echo "$coverage_percent")

            echo ""
            if awk "BEGIN {exit !($coverage_percent+0 < $coverage_error_threshold)}"; then
                if [ "$coverage_error_enforced" = true ]; then
                    echo "❌ ERROR: Node.js test coverage (${coverage_display}%) is below error threshold ($coverage_error_threshold%)"
                    echo "   This indicates insufficient test coverage. Please add more comprehensive tests."
                    cd "$original_dir"
                    return 1
                else
                    echo "⚠️  WARNING: Node.js test coverage (${coverage_display}%) is below target ($coverage_error_threshold%)"
                fi
            elif awk "BEGIN {exit !($coverage_percent+0 < $coverage_warn_threshold)}"; then
                echo "⚠️  WARNING: Node.js test coverage (${coverage_display}%) is below warning threshold ($coverage_warn_threshold%)"
                echo "   Consider adding more tests to improve code coverage."
            else
                echo "✅ Node.js test coverage (${coverage_display}%) meets quality thresholds"
            fi
        else
            echo "ℹ️  No coverage information found in test output or summary. Coverage may not be configured."
        fi

        # Export coverage metadata for downstream aggregation
        declare -g TESTING_NODE_COVERAGE_COLLECTED="true"
        if [ -n "$coverage_percent" ]; then
            declare -g TESTING_NODE_COVERAGE_PERCENT="$coverage_percent"
        fi

        if [ -f "$coverage_storage_root/coverage-summary.json" ]; then
            declare -g TESTING_NODE_COVERAGE_SUMMARY_PATH="$coverage_storage_root/coverage-summary.json"
            local totals_json
            totals_json=$(node -e "const summary=require('./coverage/coverage-summary.json'); const totals=(summary && summary.total) || {}; const pct=value=> (value && typeof value.pct === 'number') ? value.pct : null; const result={statements:pct(totals.statements), branches:pct(totals.branches), functions:pct(totals.functions), lines:pct(totals.lines)}; process.stdout.write(JSON.stringify(result));" 2>/dev/null || echo "")
            if [ -n "$totals_json" ]; then
                declare -g TESTING_NODE_COVERAGE_TOTALS_JSON="$totals_json"
            fi
        fi

        if [ -f "$coverage_storage_root/lcov.info" ]; then
            declare -g TESTING_NODE_COVERAGE_LCOV_PATH="$coverage_storage_root/lcov.info"
        fi

        cd "$original_dir"
        return 0
    fi
}

# Export function for use by sourcing scripts
export -f testing::unit::run_node_tests

testing::unit::_node_status_rank() {
    case "$1" in
        failed) echo 3 ;;
        skipped) echo 2 ;;
        passed) echo 1 ;;
        *) echo 0 ;;
    esac
}

testing::unit::_node_store_requirement_result() {
    local requirement_id="$1"
    local status="$2"
    local detail="$3"

    if [ -z "$requirement_id" ] || [ -z "$status" ]; then
        return
    fi

    local current_status="${TESTING_NODE_REQUIREMENT_STATUS["$requirement_id"]:-}"
    if [ -z "$current_status" ]; then
        TESTING_NODE_REQUIREMENT_STATUS["$requirement_id"]="$status"
        TESTING_NODE_REQUIREMENT_EVIDENCE["$requirement_id"]="$detail"
        return
    fi

    local new_rank
    local current_rank
    new_rank=$(testing::unit::_node_status_rank "$status")
    current_rank=$(testing::unit::_node_status_rank "$current_status")

    if [ "$new_rank" -gt "$current_rank" ]; then
        TESTING_NODE_REQUIREMENT_STATUS["$requirement_id"]="$status"
        TESTING_NODE_REQUIREMENT_EVIDENCE["$requirement_id"]="$detail"
    elif [ "$new_rank" -eq "$current_rank" ]; then
        local existing_detail="${TESTING_NODE_REQUIREMENT_EVIDENCE["$requirement_id"]:-}"
        if [ -n "$existing_detail" ]; then
            TESTING_NODE_REQUIREMENT_EVIDENCE["$requirement_id"]="${existing_detail}; ${detail}"
        else
            TESTING_NODE_REQUIREMENT_EVIDENCE["$requirement_id"]="$detail"
        fi
    fi
}

testing::unit::_node_collect_requirement_tags() {
    local raw_output="$1"
    if [ -z "$raw_output" ]; then
        return
    fi

    # Strip ANSI escape sequences for reliable parsing
    local cleaned_output
    cleaned_output=$(printf '%s\n' "$raw_output" | sed -E $'s/\x1B\[[0-9;]*[A-Za-z]//g')

    while IFS= read -r line; do
        [[ "$line" == *"REQ:"* ]] || continue

        local normalized_status=""
        if [[ "$line" =~ (FAIL|✗|✘|✕|×) ]]; then
            normalized_status="failed"
        elif [[ "$line" =~ (SKIP|skip|PENDING|pending|○|⧗) ]]; then
            normalized_status="skipped"
        elif [[ "$line" =~ (PASS|✓|✔|√|SUCCESS) ]]; then
            normalized_status="passed"
        fi

        if [ -z "$normalized_status" ]; then
            continue
        fi

        local tokens
        tokens=$(printf '%s\n' "$line" | grep -o 'REQ:[A-Za-z0-9_-]\+' || true)
        if [ -z "$tokens" ]; then
            continue
        fi

        local trimmed_line
        trimmed_line=$(echo "$line" | sed 's/^\s\+//')

        local token
        for token in $tokens; do
            local requirement_id="${token#REQ:}"
            testing::unit::_node_store_requirement_result "$requirement_id" "$normalized_status" "Node test ${trimmed_line}"
        done
    done <<< "$cleaned_output"
}
