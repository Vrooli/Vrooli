#!/usr/bin/env bash
# Shared test helper functions for Vrooli testing infrastructure
# Provides common utilities used across all test phases

# shellcheck disable=SC1091
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
[[ -z "${SCRIPT_DIR:-}" ]] && SCRIPT_DIR="${APP_ROOT}/__test"
source "$SCRIPT_DIR/shared/logging.bash"

# File existence and validation helpers
assert_file_exists() {
    local file_path="$1"
    local description="${2:-file}"
    
    if [[ -f "$file_path" ]]; then
        log_file_found "$file_path" "$description"
        return 0
    else
        log_file_missing "$file_path" "$description"
        return 1
    fi
}

assert_directory_exists() {
    local dir_path="$1"
    local description="${2:-directory}"
    
    if [[ -d "$dir_path" ]]; then
        log_file_found "$dir_path" "$description"
        return 0
    else
        log_file_missing "$dir_path" "$description"
        return 1
    fi
}

assert_file_executable() {
    local file_path="$1"
    local description="${2:-script}"
    
    if [[ -x "$file_path" ]]; then
        is_verbose && log_success "$description is executable: $file_path"
        return 0
    else
        log_error "$description is not executable: $file_path"
        return 1
    fi
}

# Content validation helpers
assert_file_not_empty() {
    local file_path="$1"
    local description="${2:-file}"
    
    if [[ -s "$file_path" ]]; then
        is_verbose && log_success "$description is not empty: $file_path"
        return 0
    else
        log_error "$description is empty: $file_path"
        return 1
    fi
}

assert_json_valid() {
    local file_path="$1"
    local description="${2:-JSON file}"
    
    if command -v jq >/dev/null 2>&1; then
        if jq empty < "$file_path" >/dev/null 2>&1; then
            is_verbose && log_success "$description has valid JSON: $file_path"
            return 0
        else
            log_error "$description has invalid JSON: $file_path"
            return 1
        fi
    else
        log_warning "jq not available, skipping JSON validation for: $file_path"
        return 0
    fi
}

# Shell script validation helpers
validate_shell_syntax() {
    local script_path="$1"
    local description="${2:-shell script}"
    
    log_command "bash -n $script_path"
    
    if is_dry_run; then
        log_info "[DRY RUN] Would validate syntax: $script_path"
        return 0
    fi
    
    if bash -n "$script_path" 2>/dev/null; then
        log_command_success "bash -n $script_path"
        is_verbose && log_success "$description has valid syntax: $script_path"
        return 0
    else
        log_command_failure "bash -n $script_path"
        log_error "$description has syntax errors: $script_path"
        return 1
    fi
}

run_shellcheck() {
    local script_path="$1"
    local description="${2:-shell script}"
    
    if ! command -v shellcheck >/dev/null 2>&1; then
        log_warning "shellcheck not available, skipping static analysis for: $script_path"
        return 0
    fi
    
    log_command "shellcheck $script_path"
    
    if is_dry_run; then
        log_info "[DRY RUN] Would run shellcheck on: $script_path"
        return 0
    fi
    
    if shellcheck "$script_path" 2>/dev/null; then
        log_command_success "shellcheck $script_path"
        is_verbose && log_success "$description passes shellcheck: $script_path"
        return 0
    else
        log_command_failure "shellcheck $script_path"
        log_error "$description fails shellcheck: $script_path"
        return 1
    fi
}

# File discovery helpers
find_shell_scripts() {
    local search_dir="${1:-$PROJECT_ROOT}"
    local pattern="${2:-*.sh}"
    
    find "$search_dir" -type f -name "$pattern" 2>/dev/null | sort
}

find_bats_tests() {
    local search_dir="${1:-$PROJECT_ROOT}"
    
    find "$search_dir" -type f -name "*.bats" 2>/dev/null | sort
}

find_json_files() {
    local search_dir="${1:-$PROJECT_ROOT}"
    local pattern="${2:-*.json}"
    
    find "$search_dir" -type f -name "$pattern" 2>/dev/null | sort
}

# Resource helpers
get_enabled_resources() {
    local service_json="${PROJECT_ROOT}/.vrooli/service.json"
    
    if [[ -f "$service_json" ]]; then
        if command -v jq >/dev/null 2>&1; then
            # Handle nested structure - resources are grouped by category
            # Resources structure: resources.category.resource_name.enabled
            # We need to get all resource names where enabled=true
            jq -r '.resources | to_entries[] | .value | to_entries[] | select(.value.enabled == true) | .key' "$service_json" 2>/dev/null || true
        else
            log_warning "jq not available, cannot parse enabled resources"
        fi
    else
        log_warning "service.json not found at: $service_json"
    fi
}

# Test execution helpers
run_test_with_timeout() {
    local timeout_seconds="$1"
    local test_command="$2"
    local test_name="${3:-unknown test}"
    
    log_test_start "$test_name"
    log_command "$test_command"
    
    if is_dry_run; then
        log_info "[DRY RUN] Would run: $test_command"
        return 0
    fi
    
    if timeout "$timeout_seconds" bash -c "$test_command" 2>/dev/null; then
        log_command_success "$test_command"
        log_test_pass "$test_name"
        return 0
    else
        local exit_code=$?
        if [[ $exit_code -eq 124 ]]; then
            log_test_fail "$test_name" "Timeout after ${timeout_seconds}s"
        else
            log_command_failure "$test_command" "$exit_code"
            log_test_fail "$test_name" "Exit code: $exit_code"
        fi
        return 1
    fi
}

# Counter utilities for test tracking
declare -g TEST_COUNTER_TOTAL=0
declare -g TEST_COUNTER_PASSED=0
declare -g TEST_COUNTER_FAILED=0
declare -g TEST_COUNTER_SKIPPED=0

reset_test_counters() {
    TEST_COUNTER_TOTAL=0
    TEST_COUNTER_PASSED=0
    TEST_COUNTER_FAILED=0
    TEST_COUNTER_SKIPPED=0
}

increment_test_counter() {
    local status="$1"  # passed, failed, skipped
    ((TEST_COUNTER_TOTAL++))
    
    case "$status" in
        passed)
            ((TEST_COUNTER_PASSED++))
            ;;
        failed)
            ((TEST_COUNTER_FAILED++))
            ;;
        skipped)
            ((TEST_COUNTER_SKIPPED++))
            ;;
        *)
            log_error "Invalid test status: $status"
            ;;
    esac
}

get_test_counters() {
    echo "$TEST_COUNTER_TOTAL $TEST_COUNTER_PASSED $TEST_COUNTER_FAILED $TEST_COUNTER_SKIPPED"
}

print_test_summary() {
    log_summary "$TEST_COUNTER_TOTAL" "$TEST_COUNTER_PASSED" "$TEST_COUNTER_FAILED" "$TEST_COUNTER_SKIPPED"
}

# Utility functions
relative_path() {
    local file_path="$1"
    local base_path="${2:-$PROJECT_ROOT}"
    
    # Remove base path prefix if present
    echo "${file_path#$base_path/}"
}

is_shell_script() {
    local file_path="$1"
    
    # Check file extension
    if [[ "$file_path" =~ \.sh$ ]]; then
        return 0
    fi
    
    # Check shebang
    if [[ -f "$file_path" ]] && head -n1 "$file_path" | grep -q '^#!/.*sh'; then
        return 0
    fi
    
    return 1
}

# Validate and fix file permissions for testing
validate_test_file_permissions() {
    local file_path="$1"
    local required_permission="${2:-readable}" # readable, executable, writable
    local description="${3:-file}"
    
    if [[ ! -f "$file_path" ]]; then
        log_error "$description does not exist: $file_path"
        return 1
    fi
    
    case "$required_permission" in
        "readable")
            if [[ ! -r "$file_path" ]]; then
                log_error "$description is not readable: $file_path"
                return 1
            fi
            ;;
        "executable")
            if [[ ! -x "$file_path" ]]; then
                log_warning "$description is not executable, attempting to fix: $file_path"
                if chmod +x "$file_path" 2>/dev/null; then
                    log_success "Fixed executable permissions for $description: $file_path"
                else
                    log_error "Cannot make $description executable: $file_path"
                    return 1
                fi
            fi
            ;;
        "writable")
            if [[ ! -w "$file_path" ]]; then
                log_error "$description is not writable: $file_path"
                return 1
            fi
            ;;
        *)
            log_error "Invalid permission type: $required_permission"
            return 1
            ;;
    esac
    
    return 0
}

# Validate directory permissions
validate_directory_permissions() {
    local dir_path="$1"
    local required_permission="${2:-readable}" # readable, writable
    local description="${3:-directory}"
    local create_if_missing="${4:-false}"
    
    if [[ ! -d "$dir_path" ]]; then
        if [[ "$create_if_missing" == "true" ]]; then
            log_info "Creating missing $description: $dir_path"
            if mkdir -p "$dir_path" 2>/dev/null; then
                log_success "Created $description: $dir_path"
            else
                log_error "Cannot create $description: $dir_path"
                return 1
            fi
        else
            log_error "$description does not exist: $dir_path"
            return 1
        fi
    fi
    
    case "$required_permission" in
        "readable")
            if [[ ! -r "$dir_path" ]]; then
                log_error "$description is not readable: $dir_path"
                return 1
            fi
            ;;
        "writable")
            if [[ ! -w "$dir_path" ]]; then
                log_error "$description is not writable: $dir_path"
                return 1
            fi
            ;;
        *)
            log_error "Invalid permission type: $required_permission"
            return 1
            ;;
    esac
    
    return 0
}

# Export functions for use in subshells
export -f assert_file_exists assert_directory_exists assert_file_executable
export -f assert_file_not_empty assert_json_valid
export -f validate_shell_syntax run_shellcheck
export -f find_shell_scripts find_bats_tests find_json_files
export -f get_enabled_resources run_test_with_timeout
export -f reset_test_counters increment_test_counter get_test_counters print_test_summary
export -f relative_path is_shell_script
export -f validate_test_file_permissions validate_directory_permissions