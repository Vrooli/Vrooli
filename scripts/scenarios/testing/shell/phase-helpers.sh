#!/usr/bin/env bash
# Phase lifecycle helpers - Eliminates boilerplate from test phase scripts
# Usage: source this file at the beginning of your phase script, then use the helpers
set -euo pipefail

# Global variables for phase state
TESTING_PHASE_NAME=""
TESTING_PHASE_TARGET_TIME=""
TESTING_PHASE_START_TIME=""
TESTING_PHASE_ERROR_COUNT=0
TESTING_PHASE_TEST_COUNT=0
TESTING_PHASE_WARNING_COUNT=0
TESTING_PHASE_SKIPPED_COUNT=0
TESTING_PHASE_SCENARIO_DIR=""
TESTING_PHASE_SCENARIO_NAME=""
TESTING_PHASE_APP_ROOT=""
TESTING_PHASE_CLEANUP_FUNCTIONS=()

# Initialize phase environment
# Usage: testing::phase::init [options]
# Options:
#   --phase-name NAME       Override auto-detected phase name
#   --target-time TIME      Target execution time (e.g., "15s", "2m")
#   --skip-runtime-check    Skip runtime availability check
#   --require-runtime       Require runtime to be available
testing::phase::init() {
    local phase_name=""
    local target_time=""
    local skip_runtime_check=false
    local require_runtime=false
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --phase-name)
                phase_name="$2"
                shift 2
                ;;
            --target-time)
                target_time="$2"
                shift 2
                ;;
            --skip-runtime-check)
                skip_runtime_check=true
                shift
                ;;
            --require-runtime)
                require_runtime=true
                shift
                ;;
            *)
                echo "Unknown option to testing::phase::init: $1" >&2
                return 1
                ;;
        esac
    done
    
    # Auto-detect directories
    local phase_script_dir="$(cd "$(dirname "${BASH_SOURCE[1]}")" && pwd)"
    TESTING_PHASE_SCENARIO_DIR="$(cd "$phase_script_dir/../.." && pwd)"
    TESTING_PHASE_APP_ROOT="${APP_ROOT:-$(builtin cd "${TESTING_PHASE_SCENARIO_DIR}/../.." && builtin pwd)}"
    TESTING_PHASE_SCENARIO_NAME="$(basename "$TESTING_PHASE_SCENARIO_DIR")"
    
    # Auto-detect phase name from script filename if not provided
    if [ -z "$phase_name" ]; then
        local script_name="$(basename "${BASH_SOURCE[1]}")"
        phase_name="${script_name#test-}"
        phase_name="${phase_name%.sh}"
    fi
    TESTING_PHASE_NAME="$phase_name"
    TESTING_PHASE_TARGET_TIME="$target_time"
    
    # Source required libraries
    source "${TESTING_PHASE_APP_ROOT}/scripts/lib/utils/log.sh"
    source "${TESTING_PHASE_APP_ROOT}/scripts/scenarios/testing/shell/core.sh"
    source "${TESTING_PHASE_APP_ROOT}/scripts/scenarios/testing/shell/connectivity.sh"
    
    # Source scenario-specific utilities if they exist
    if [ -f "${TESTING_PHASE_SCENARIO_DIR}/test/utils/cli.sh" ]; then
        source "${TESTING_PHASE_SCENARIO_DIR}/test/utils/cli.sh"
    fi
    
    # Display phase header
    local header="=== ${TESTING_PHASE_NAME^} Phase"
    if [ -n "$TESTING_PHASE_TARGET_TIME" ]; then
        header="$header (Target: <${TESTING_PHASE_TARGET_TIME})"
    fi
    header="$header ==="
    echo "$header"
    
    # Start timing
    TESTING_PHASE_START_TIME=$(date +%s)
    
    # Handle runtime requirements
    if [ "$require_runtime" = true ] && [ "$skip_runtime_check" = false ]; then
        if ! testing::core::ensure_runtime_or_skip "$TESTING_PHASE_SCENARIO_NAME" "${TESTING_PHASE_NAME} tests"; then
            local status=$?
            if [ "$status" -eq 200 ]; then
                exit 200  # Skipped
            else
                exit 1    # Failed
            fi
        fi
    fi
    
    # Change to scenario directory by default
    cd "$TESTING_PHASE_SCENARIO_DIR"
    
    # Set up cleanup trap
    trap 'testing::phase::cleanup' EXIT
}

# Add a test result
# Usage: testing::phase::add_test [passed|failed|skipped]
testing::phase::add_test() {
    local result="${1:-passed}"
    TESTING_PHASE_TEST_COUNT=$((TESTING_PHASE_TEST_COUNT + 1))
    
    case "$result" in
        failed)
            TESTING_PHASE_ERROR_COUNT=$((TESTING_PHASE_ERROR_COUNT + 1))
            ;;
        skipped)
            TESTING_PHASE_SKIPPED_COUNT=$((TESTING_PHASE_SKIPPED_COUNT + 1))
            ;;
    esac
}

# Add an error (convenience function)
# Usage: testing::phase::add_error [message]
testing::phase::add_error() {
    local message="${1:-}"
    TESTING_PHASE_ERROR_COUNT=$((TESTING_PHASE_ERROR_COUNT + 1))
    if [ -n "$message" ]; then
        log::error "$message"
    fi
}

# Add a warning
# Usage: testing::phase::add_warning [message]
testing::phase::add_warning() {
    local message="${1:-}"
    TESTING_PHASE_WARNING_COUNT=$((TESTING_PHASE_WARNING_COUNT + 1))
    if [ -n "$message" ]; then
        log::warning "$message"
    fi
}

# Check a condition and report result
# Usage: testing::phase::check "description" command [args...]
# Example: testing::phase::check "API health endpoint" curl -sf "$API_URL/health"
testing::phase::check() {
    local description="$1"
    shift
    
    echo -n "🔍 Checking $description... "
    if "$@" >/dev/null 2>&1; then
        log::success "✅ Passed"
        testing::phase::add_test passed
        return 0
    else
        log::error "❌ Failed"
        testing::phase::add_test failed
        return 1
    fi
}

# Check for required files
# Usage: testing::phase::check_files file1 file2 ...
testing::phase::check_files() {
    local files=("$@")
    local missing_files=()
    
    echo "🔍 Checking required files..."
    for file in "${files[@]}"; do
        if [ ! -f "$file" ]; then
            missing_files+=("$file")
            TESTING_PHASE_ERROR_COUNT=$((TESTING_PHASE_ERROR_COUNT + 1))
        fi
    done
    
    if [ ${#missing_files[@]} -gt 0 ]; then
        log::error "❌ Missing required files:"
        printf "   - %s\n" "${missing_files[@]}"
        return 1
    else
        log::success "✅ All required files present"
        return 0
    fi
}

# Check for required directories
# Usage: testing::phase::check_directories dir1 dir2 ...
testing::phase::check_directories() {
    local dirs=("$@")
    local missing_dirs=()
    
    echo "🔍 Checking required directories..."
    for dir in "${dirs[@]}"; do
        if [ ! -d "$dir" ]; then
            missing_dirs+=("$dir")
            TESTING_PHASE_ERROR_COUNT=$((TESTING_PHASE_ERROR_COUNT + 1))
        fi
    done
    
    if [ ${#missing_dirs[@]} -gt 0 ]; then
        log::error "❌ Missing required directories:"
        printf "   - %s\n" "${missing_dirs[@]}"
        return 1
    else
        log::success "✅ All required directories present"
        return 0
    fi
}

# Register a cleanup function
# Usage: testing::phase::register_cleanup function_name
testing::phase::register_cleanup() {
    local func="$1"
    TESTING_PHASE_CLEANUP_FUNCTIONS+=("$func")
}

# Run cleanup functions
testing::phase::cleanup() {
    for func in "${TESTING_PHASE_CLEANUP_FUNCTIONS[@]}"; do
        if declare -f "$func" >/dev/null; then
            "$func" || true
        fi
    done
}

# End phase with summary and exit
# Usage: testing::phase::end_with_summary [custom_message]
testing::phase::end_with_summary() {
    local custom_message="${1:-}"
    local end_time=$(date +%s)
    local duration=$((end_time - TESTING_PHASE_START_TIME))
    
    echo ""
    
    # Build summary message
    local summary_parts=()
    if [ $TESTING_PHASE_TEST_COUNT -gt 0 ]; then
        summary_parts+=("$TESTING_PHASE_TEST_COUNT tests")
    fi
    if [ $TESTING_PHASE_ERROR_COUNT -gt 0 ]; then
        summary_parts+=("$TESTING_PHASE_ERROR_COUNT errors")
    fi
    if [ $TESTING_PHASE_WARNING_COUNT -gt 0 ]; then
        summary_parts+=("$TESTING_PHASE_WARNING_COUNT warnings")  
    fi
    if [ $TESTING_PHASE_SKIPPED_COUNT -gt 0 ]; then
        summary_parts+=("$TESTING_PHASE_SKIPPED_COUNT skipped")
    fi
    
    local summary=""
    if [ ${#summary_parts[@]} -gt 0 ]; then
        summary=" ($(IFS=', '; echo "${summary_parts[*]}"))"
    fi
    
    # Check if we exceeded target time
    local time_warning=""
    if [ -n "$TESTING_PHASE_TARGET_TIME" ]; then
        local target_seconds
        if [[ "$TESTING_PHASE_TARGET_TIME" =~ ^([0-9]+)s$ ]]; then
            target_seconds="${BASH_REMATCH[1]}"
        elif [[ "$TESTING_PHASE_TARGET_TIME" =~ ^([0-9]+)m$ ]]; then
            target_seconds=$((BASH_REMATCH[1] * 60))
        else
            target_seconds=0
        fi
        
        if [ $target_seconds -gt 0 ] && [ $duration -gt $target_seconds ]; then
            time_warning=" ⚠️  Exceeded ${TESTING_PHASE_TARGET_TIME} target"
        fi
    fi
    
    # Display final status
    if [ $TESTING_PHASE_ERROR_COUNT -eq 0 ]; then
        if [ -n "$custom_message" ]; then
            log::success "✅ $custom_message in ${duration}s${summary}${time_warning}"
        else
            log::success "✅ ${TESTING_PHASE_NAME^} phase completed successfully in ${duration}s${summary}${time_warning}"
        fi
        exit 0
    else
        if [ -n "$custom_message" ]; then
            log::error "❌ $custom_message in ${duration}s${summary}${time_warning}"
        else
            log::error "❌ ${TESTING_PHASE_NAME^} phase failed in ${duration}s${summary}${time_warning}"
        fi
        exit 1
    fi
}

# Measure and report execution time for a command
# Usage: testing::phase::timed_exec "description" command [args...]
testing::phase::timed_exec() {
    local description="$1"
    shift
    
    echo "⏱️  $description..."
    local start=$(date +%s)
    
    if "$@"; then
        local end=$(date +%s)
        local duration=$((end - start))
        log::success "✅ Completed in ${duration}s"
        return 0
    else
        local end=$(date +%s)
        local duration=$((end - start))
        log::error "❌ Failed after ${duration}s"
        return 1
    fi
}

# Export functions for use by phase scripts
export -f testing::phase::init
export -f testing::phase::add_test
export -f testing::phase::add_error
export -f testing::phase::add_warning
export -f testing::phase::check
export -f testing::phase::check_files
export -f testing::phase::check_directories
export -f testing::phase::register_cleanup
export -f testing::phase::cleanup
export -f testing::phase::end_with_summary
export -f testing::phase::timed_exec