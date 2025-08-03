#!/usr/bin/env bash
# Changed Files Test Runner - Run tests for files that have changed
# Smart test selection based on git changes and dependency analysis

set -euo pipefail

# Script directory
RUNNER_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_ROOT="$(dirname "$RUNNER_DIR")"
SCRIPTS_DIR="$(dirname "$TEST_ROOT")"
PROJECT_ROOT="$(dirname "$SCRIPTS_DIR")"

# Load shared utilities
source "$TEST_ROOT/shared/logging.bash"
source "$TEST_ROOT/shared/utils.bash"
source "$TEST_ROOT/shared/config-simple.bash"

# Configuration
COMMIT_RANGE="${VROOLI_TEST_COMMIT_RANGE:-HEAD~1..HEAD}"
INCLUDE_UNTRACKED="${VROOLI_TEST_INCLUDE_UNTRACKED:-false}"
INCLUDE_DEPENDENCIES="${VROOLI_TEST_INCLUDE_DEPS:-true}"
VERBOSE="${VROOLI_TEST_VERBOSE:-false}"
DRY_RUN="${VROOLI_TEST_DRY_RUN:-false}"

# Test tracking
declare -a CHANGED_FILES=()
declare -a AFFECTED_TESTS=()

#######################################
# Print usage information
#######################################
print_usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Run tests for files that have changed in git.

OPTIONS:
    --range RANGE         Git commit range (default: HEAD~1..HEAD)
    --include-untracked   Include untracked files
    --include-deps        Include dependency-related tests (default: true)
    --no-deps            Exclude dependency-related tests
    --verbose            Enable verbose output
    --dry-run            Show what would be tested without running
    --list-files         List changed files and exit
    --list-tests         List affected tests and exit
    -h, --help           Show this help message

EXAMPLES:
    $0                            # Test changes in last commit
    $0 --range origin/main..HEAD  # Test changes from main branch
    $0 --include-untracked        # Include uncommitted files
    $0 --dry-run                  # Show what would be tested
    $0 --list-files               # Just list changed files

COMMIT RANGE FORMATS:
    HEAD~1..HEAD                  # Last commit (default)
    HEAD~3..HEAD                  # Last 3 commits
    origin/main..HEAD             # Changes from main branch
    abc123..def456               # Between specific commits

EOF
}

#######################################
# Parse command line arguments
#######################################
parse_args() {
    local list_files=false
    local list_tests=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --range)
                COMMIT_RANGE="$2"
                shift 2
                ;;
            --include-untracked)
                INCLUDE_UNTRACKED="true"
                shift
                ;;
            --include-deps)
                INCLUDE_DEPENDENCIES="true"
                shift
                ;;
            --no-deps)
                INCLUDE_DEPENDENCIES="false"
                shift
                ;;
            --verbose|-v)
                VERBOSE="true"
                shift
                ;;
            --dry-run)
                DRY_RUN="true"
                shift
                ;;
            --list-files)
                list_files=true
                shift
                ;;
            --list-tests)
                list_tests=true
                shift
                ;;
            -h|--help)
                print_usage
                exit 0
                ;;
            *)
                vrooli_log_error "Unknown option: $1"
                print_usage
                exit 1
                ;;
        esac
    done
    
    export LIST_FILES_ONLY="$list_files"
    export LIST_TESTS_ONLY="$list_tests"
}

#######################################
# Get list of changed files from git
#######################################
get_changed_files() {
    vrooli_log_info "Analyzing changes in range: $COMMIT_RANGE"
    
    # Get changed files from git
    while IFS= read -r file; do
        # Skip empty lines
        [[ -n "$file" ]] || continue
        
        # Convert to absolute path
        local abs_path="$PROJECT_ROOT/$file"
        if [[ -f "$abs_path" ]]; then
            CHANGED_FILES+=("$abs_path")
        fi
    done < <(git diff --name-only "$COMMIT_RANGE" 2>/dev/null || true)
    
    # Include untracked files if requested
    if [[ "$INCLUDE_UNTRACKED" == "true" ]]; then
        while IFS= read -r file; do
            [[ -n "$file" ]] || continue
            local abs_path="$PROJECT_ROOT/$file"
            if [[ -f "$abs_path" ]]; then
                CHANGED_FILES+=("$abs_path")
            fi
        done < <(git ls-files --others --exclude-standard 2>/dev/null || true)
    fi
    
    # Remove duplicates and sort
    if [[ ${#CHANGED_FILES[@]} -gt 0 ]]; then
        readarray -t CHANGED_FILES < <(printf '%s\n' "${CHANGED_FILES[@]}" | sort -u)
    fi
    
    vrooli_log_info "Found ${#CHANGED_FILES[@]} changed files"
    
    if [[ "$LIST_FILES_ONLY" == "true" ]]; then
        echo "Changed files:"
        for file in "${CHANGED_FILES[@]}"; do
            echo "  $(realpath --relative-to="$PROJECT_ROOT" "$file")"
        done
        exit 0
    fi
    
    if [[ "$VERBOSE" == "true" ]]; then
        for file in "${CHANGED_FILES[@]}"; do
            vrooli_log_debug "Changed: $(realpath --relative-to="$PROJECT_ROOT" "$file")"
        done
    fi
}

#######################################
# Map changed files to affected tests
#######################################
map_affected_tests() {
    vrooli_log_info "Mapping changed files to affected tests..."
    
    local -A test_map=()
    
    for file in "${CHANGED_FILES[@]}"; do
        local rel_path
        rel_path=$(realpath --relative-to="$PROJECT_ROOT" "$file")
        
        # Direct test file mappings
        case "$rel_path" in
            # Test files themselves
            scripts/__test/*.bats|scripts/__test/**/*.bats)
                test_map["$file"]="direct"
                ;;
                
            # Resource test files
            scripts/resources/**/*.bats)
                test_map["$file"]="direct"
                ;;
                
            # Integration test files
            scripts/__test/integration/**/*.sh)
                test_map["$file"]="direct"
                ;;
                
            # Source files that affect specific resource tests
            scripts/resources/*)
                map_resource_tests "$file" test_map
                ;;
                
            # Server source files
            packages/server/src/*)
                map_server_tests "$file" test_map
                ;;
                
            # UI source files
            packages/ui/src/*)
                map_ui_tests "$file" test_map
                ;;
                
            # Shared source files
            packages/shared/src/*)
                map_shared_tests "$file" test_map
                ;;
                
            # Configuration files
            *.yaml|*.yml|*.json|*.env*|*config*)
                map_config_tests "$file" test_map
                ;;
                
            # Docker and deployment files
            *Dockerfile*|docker-compose*|*.k8s.*|scripts/main/*)
                map_deployment_tests "$file" test_map
                ;;
        esac
    done
    
    # Convert map to array
    for test_file in "${!test_map[@]}"; do
        if [[ -f "$test_file" ]]; then
            AFFECTED_TESTS+=("$test_file")
        fi
    done
    
    # Add dependency tests if enabled
    if [[ "$INCLUDE_DEPENDENCIES" == "true" ]]; then
        map_dependency_tests
    fi
    
    # Remove duplicates
    if [[ ${#AFFECTED_TESTS[@]} -gt 0 ]]; then
        readarray -t AFFECTED_TESTS < <(printf '%s\n' "${AFFECTED_TESTS[@]}" | sort -u)
    fi
    
    vrooli_log_info "Found ${#AFFECTED_TESTS[@]} affected tests"
    
    if [[ "$LIST_TESTS_ONLY" == "true" ]]; then
        echo "Affected tests:"
        for test in "${AFFECTED_TESTS[@]}"; do
            echo "  $(realpath --relative-to="$PROJECT_ROOT" "$test")"
        done
        exit 0
    fi
}

#######################################
# Map resource files to their tests
#######################################
map_resource_tests() {
    local file="$1"
    local -n map_ref=$2
    
    # Extract resource path
    local resource_dir
    resource_dir=$(dirname "$file")
    
    # Find tests in the resource directory
    while IFS= read -r test_file; do
        map_ref["$test_file"]="resource"
    done < <(find "$resource_dir" -name "*.bats" -type f 2>/dev/null)
    
    # Find integration tests for this resource
    local resource_name
    resource_name=$(basename "$(dirname "$resource_dir")")
    local integration_test="$TEST_ROOT/integration/services/$resource_name.sh"
    if [[ -f "$integration_test" ]]; then
        map_ref["$integration_test"]="integration"
    fi
}

#######################################
# Map server files to relevant tests
#######################################
map_server_tests() {
    local file="$1"
    local -n map_ref=$2
    
    # Server unit tests
    while IFS= read -r test_file; do
        map_ref["$test_file"]="server"
    done < <(find "$TEST_ROOT/fixtures/tests" -name "*server*.bats" -o -name "*api*.bats" 2>/dev/null)
    
    # API integration tests
    while IFS= read -r test_file; do
        map_ref["$test_file"]="api"
    done < <(find "$TEST_ROOT/integration/scenarios" -name "*api*.sh" -o -name "*server*.sh" 2>/dev/null)
}

#######################################
# Map UI files to relevant tests
#######################################
map_ui_tests() {
    local file="$1"
    local -n map_ref=$2
    
    # UI unit tests
    while IFS= read -r test_file; do
        map_ref["$test_file"]="ui"
    done < <(find "$TEST_ROOT/fixtures/tests" -name "*ui*.bats" -o -name "*frontend*.bats" 2>/dev/null)
}

#######################################
# Map shared files to relevant tests
#######################################
map_shared_tests() {
    local file="$1"
    local -n map_ref=$2
    
    # Shared code affects many tests, be conservative
    while IFS= read -r test_file; do
        map_ref["$test_file"]="shared"
    done < <(find "$TEST_ROOT/fixtures/tests" -name "*.bats" 2>/dev/null)
}

#######################################
# Map configuration files to relevant tests
#######################################
map_config_tests() {
    local file="$1"
    local -n map_ref=$2
    
    # Configuration changes affect core tests
    while IFS= read -r test_file; do
        map_ref["$test_file"]="config"
    done < <(find "$TEST_ROOT/fixtures/tests/core" -name "*.bats" 2>/dev/null)
}

#######################################
# Map deployment files to relevant tests
#######################################
map_deployment_tests() {
    local file="$1"
    local -n map_ref=$2
    
    # Deployment changes affect infrastructure tests
    while IFS= read -r test_file; do
        map_ref["$test_file"]="deployment"
    done < <(find "$TEST_ROOT" -name "*deploy*.bats" -o -name "*docker*.bats" -o -name "*k8s*.bats" 2>/dev/null)
}

#######################################
# Add dependency-based tests
#######################################
map_dependency_tests() {
    vrooli_log_debug "Adding dependency-based tests..."
    
    # If any core files changed, run core tests
    for file in "${CHANGED_FILES[@]}"; do
        local rel_path
        rel_path=$(realpath --relative-to="$PROJECT_ROOT" "$file")
        
        if [[ "$rel_path" =~ (packages/shared|scripts/__test/shared|scripts/__test/fixtures) ]]; then
            # Core infrastructure changes affect all tests
            while IFS= read -r test_file; do
                AFFECTED_TESTS+=("$test_file")
            done < <(find "$TEST_ROOT/fixtures/tests/core" -name "*.bats" 2>/dev/null)
            break
        fi
    done
}

#######################################
# Run the affected tests
#######################################
run_affected_tests() {
    if [[ ${#AFFECTED_TESTS[@]} -eq 0 ]]; then
        vrooli_log_warn "No tests affected by changes"
        exit 0
    fi
    
    if [[ "$DRY_RUN" == "true" ]]; then
        vrooli_log_info "Dry run - would execute ${#AFFECTED_TESTS[@]} tests:"
        for test in "${AFFECTED_TESTS[@]}"; do
            echo "  $(realpath --relative-to="$PROJECT_ROOT" "$test")"
        done
        exit 0
    fi
    
    vrooli_log_header "🎯 Running Affected Tests"
    
    # Categorize tests
    local -a unit_tests=()
    local -a integration_tests=()
    local -a shell_tests=()
    
    for test in "${AFFECTED_TESTS[@]}"; do
        if [[ "$test" =~ \.bats$ ]]; then
            if [[ "$test" =~ scripts/__test/(fixtures|unit)/ ]]; then
                unit_tests+=("$test")
            else
                shell_tests+=("$test")
            fi
        elif [[ "$test" =~ \.sh$ && "$test" =~ integration/ ]]; then
            integration_tests+=("$test")
        fi
    done
    
    local failures=0
    
    # Run unit tests
    if [[ ${#unit_tests[@]} -gt 0 ]]; then
        vrooli_log_info "Running ${#unit_tests[@]} unit tests..."
        if ! "$RUNNER_DIR/run-unit.sh" ${VERBOSE:+--verbose} "${unit_tests[@]}"; then
            ((failures++))
        fi
    fi
    
    # Run integration tests
    if [[ ${#integration_tests[@]} -gt 0 ]]; then
        vrooli_log_info "Running ${#integration_tests[@]} integration tests..."
        
        # Extract service names for integration runner
        local services=()
        for test in "${integration_tests[@]}"; do
            local service_name
            service_name=$(basename "$test" .sh)
            services+=("$service_name")
        done
        
        if ! "$RUNNER_DIR/run-integration.sh" ${VERBOSE:+--verbose} "${services[@]}"; then
            ((failures++))
        fi
    fi
    
    # Run shell tests
    if [[ ${#shell_tests[@]} -gt 0 ]]; then
        vrooli_log_info "Running ${#shell_tests[@]} shell tests..."
        if ! "$TEST_ROOT/shell/core/run-tests.sh" ${VERBOSE:+--verbose} "${shell_tests[@]}"; then
            ((failures++))
        fi
    fi
    
    # Report results
    if [[ $failures -eq 0 ]]; then
        vrooli_log_success "🎉 All affected tests passed!"
        exit 0
    else
        vrooli_log_error "❌ $failures test suite(s) failed"
        exit 1
    fi
}

#######################################
# Main execution
#######################################
main() {
    # Parse arguments
    parse_args "$@"
    
    # Initialize
    vrooli_log_header "🎯 Changed Files Test Runner"
    
    # Check if we're in a git repository
    if ! git rev-parse --git-dir >/dev/null 2>&1; then
        vrooli_log_error "Not in a git repository"
        exit 1
    fi
    
    # Change to project root for git operations
    cd "$PROJECT_ROOT"
    
    # Get changed files
    get_changed_files
    
    if [[ ${#CHANGED_FILES[@]} -eq 0 ]]; then
        vrooli_log_info "No files changed in range: $COMMIT_RANGE"
        exit 0
    fi
    
    # Map to affected tests
    map_affected_tests
    
    # Run the tests
    run_affected_tests
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi