#!/usr/bin/env bash
# Vrooli Resource Interface Validation - Pre-commit Hook
# 
# This pre-commit hook validates resource interface compliance before allowing commits.
# It ensures that any changes to manage.sh scripts maintain interface standards.
#
# Installation:
#   1. Copy this file to .git/hooks/pre-commit
#   2. Make it executable: chmod +x .git/hooks/pre-commit
#   3. Customize configuration below as needed
#
# Or use with pre-commit framework:
#   1. Add to .pre-commit-config.yaml (see bottom of file)
#   2. Run: pre-commit install

set -euo pipefail

# =============================================================================
# Configuration
# =============================================================================

# Validation level (quick, standard, full)
VALIDATION_LEVEL="quick"

# Enable parallel processing for faster validation
USE_PARALLEL="true"

# Output format (text, json, junit)
OUTPUT_FORMAT="text"

# Enable verbose output for debugging
VERBOSE_OUTPUT="false"

# Fail commit on validation warnings (not just errors)
STRICT_MODE="false"

# Skip validation if commit message contains this string
SKIP_VALIDATION_MARKER="[skip-resource-validation]"

# Only validate resources that have changes (more efficient)
VALIDATE_CHANGED_ONLY="true"

# Maximum time to allow for validation (seconds)
VALIDATION_TIMEOUT="30"

# =============================================================================
# Helper Functions
# =============================================================================

# Color output functions
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
BOLD='\033[1m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1" >&2
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} ✅ $1" >&2
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} ⚠️  $1" >&2
}

log_error() {
    echo -e "${RED}[ERROR]${NC} ❌ $1" >&2
}

# Check if we're in a git repository
check_git_repo() {
    if ! git rev-parse --git-dir &>/dev/null; then
        log_error "Not in a git repository"
        exit 1
    fi
}

# Check if validation framework exists
check_validation_framework() {
    local resources_dir="scripts/resources"
    local validation_script="$resources_dir/tools/validate-interfaces.sh"
    
    if [[ ! -f "$validation_script" ]]; then
        log_error "Validation framework not found: $validation_script"
        log_error "Please ensure you're running from the project root directory"
        exit 1
    fi
    
    if [[ ! -x "$validation_script" ]]; then
        log_error "Validation script is not executable: $validation_script"
        log_error "Run: chmod +x $validation_script"
        exit 1
    fi
}

# Get commit message to check for skip marker
get_commit_message() {
    # Try to get commit message from various sources
    local commit_msg=""
    
    # Check if there's a commit message file
    if [[ -f ".git/COMMIT_EDITMSG" ]]; then
        commit_msg=$(head -1 ".git/COMMIT_EDITMSG" 2>/dev/null || echo "")
    fi
    
    # Fallback to checking staged changes
    if [[ -z "$commit_msg" ]]; then
        commit_msg=$(git log --format=%B -n 1 HEAD 2>/dev/null || echo "")
    fi
    
    echo "$commit_msg"
}

# Check if validation should be skipped
should_skip_validation() {
    local commit_msg
    commit_msg=$(get_commit_message)
    
    if [[ "$commit_msg" == *"$SKIP_VALIDATION_MARKER"* ]]; then
        log_info "Skipping resource validation (found: $SKIP_VALIDATION_MARKER)"
        return 0
    fi
    
    return 1
}

# Get list of changed manage.sh files
get_changed_resources() {
    local changed_files
    changed_files=$(git diff --cached --name-only --diff-filter=AM | grep 'scripts/resources/.*/manage\.sh$' || true)
    
    if [[ -z "$changed_files" ]]; then
        return 1
    fi
    
    # Extract resource names from file paths
    local resources=()
    while IFS= read -r file; do
        if [[ -n "$file" ]]; then
            local resource_dir
            resource_dir=$(dirname "$file")
            local resource_name
            resource_name=$(basename "$resource_dir")
            resources+=("$resource_name")
        fi
    done <<< "$changed_files"
    
    # Remove duplicates and print
    printf '%s\n' "${resources[@]}" | sort -u
}

# Run validation for specific resources
validate_specific_resources() {
    local resources=("$@")
    local validation_script="scripts/resources/tools/validate-interfaces.sh"
    local validation_args=()
    
    # Build validation arguments
    validation_args+=("--level" "$VALIDATION_LEVEL")
    validation_args+=("--format" "$OUTPUT_FORMAT")
    
    if [[ "$USE_PARALLEL" == "true" ]]; then
        validation_args+=("--parallel")
    fi
    
    if [[ "$VERBOSE_OUTPUT" == "true" ]]; then
        validation_args+=("--verbose")
    fi
    
    # Validate each resource
    local failed_resources=()
    local total_resources=${#resources[@]}
    
    log_info "Validating $total_resources changed resource(s)..."
    
    for resource in "${resources[@]}"; do
        log_info "Validating resource: $resource"
        
        if timeout "$VALIDATION_TIMEOUT" "$validation_script" --resource "$resource" "${validation_args[@]}" >&2; then
            log_success "Resource $resource passed validation"
        else
            local exit_code=$?
            if [[ $exit_code -eq 124 ]]; then
                log_error "Validation timeout for resource: $resource"
            else
                log_error "Validation failed for resource: $resource"
            fi
            failed_resources+=("$resource")
        fi
    done
    
    # Report results
    if [[ ${#failed_resources[@]} -eq 0 ]]; then
        log_success "All $total_resources resource(s) passed validation!"
        return 0
    else
        log_error "${#failed_resources[@]} resource(s) failed validation:"
        for resource in "${failed_resources[@]}"; do
            echo "  • $resource" >&2
        done
        return 1
    fi
}

# Run validation for all resources
validate_all_resources() {
    local validation_script="scripts/resources/tools/validate-interfaces.sh"
    local validation_args=()
    
    # Build validation arguments
    validation_args+=("--level" "$VALIDATION_LEVEL")
    validation_args+=("--format" "$OUTPUT_FORMAT")
    
    if [[ "$USE_PARALLEL" == "true" ]]; then
        validation_args+=("--parallel")
    fi
    
    if [[ "$VERBOSE_OUTPUT" == "true" ]]; then
        validation_args+=("--verbose")
    fi
    
    log_info "Running full resource interface validation..."
    
    if timeout "$VALIDATION_TIMEOUT" "$validation_script" "${validation_args[@]}" >&2; then
        log_success "All resources passed validation!"
        return 0
    else
        local exit_code=$?
        if [[ $exit_code -eq 124 ]]; then
            log_error "Validation timeout (>${VALIDATION_TIMEOUT}s)"
        else
            log_error "Resource validation failed"
        fi
        return 1
    fi
}

# Main validation function
run_validation() {
    if [[ "$VALIDATE_CHANGED_ONLY" == "true" ]]; then
        # Get list of changed resources
        local changed_resources
        if changed_resources=$(get_changed_resources); then
            local resources_array
            mapfile -t resources_array <<< "$changed_resources"
            
            if [[ ${#resources_array[@]} -gt 0 ]]; then
                validate_specific_resources "${resources_array[@]}"
            else
                log_info "No resource changes detected in this commit"
                return 0
            fi
        else
            log_info "No resource changes detected in this commit"
            return 0
        fi
    else
        # Validate all resources
        validate_all_resources
    fi
}

# =============================================================================
# Main Execution
# =============================================================================

main() {
    # Check prerequisites
    check_git_repo
    check_validation_framework
    
    # Check if validation should be skipped
    if should_skip_validation; then
        exit 0
    fi
    
    # Show configuration
    if [[ "$VERBOSE_OUTPUT" == "true" ]]; then
        log_info "Pre-commit validation configuration:"
        echo "  Validation level: $VALIDATION_LEVEL" >&2
        echo "  Parallel processing: $USE_PARALLEL" >&2
        echo "  Output format: $OUTPUT_FORMAT" >&2
        echo "  Validate changed only: $VALIDATE_CHANGED_ONLY" >&2
        echo "  Strict mode: $STRICT_MODE" >&2
        echo "  Timeout: ${VALIDATION_TIMEOUT}s" >&2
        echo >&2
    fi
    
    # Run validation
    if run_validation; then
        log_success "Resource interface validation passed! ✨"
        exit 0
    else
        echo >&2
        log_error "Resource interface validation failed!"
        echo >&2
        echo "To skip this validation, add '$SKIP_VALIDATION_MARKER' to your commit message." >&2
        echo "To fix issues, review the validation requirements in:" >&2
        echo "  • scripts/resources/contracts/v1.0/ (interface contracts)" >&2
        echo "  • scripts/resources/tests/framework/ (validation framework)" >&2
        echo >&2
        echo "For help, run:" >&2
        echo "  scripts/resources/tools/validate-interfaces.sh --help" >&2
        echo >&2
        exit 1
    fi
}

# Run main function
main "$@"

# =============================================================================
# Pre-commit Framework Integration
# =============================================================================

# To use with pre-commit framework, add this to .pre-commit-config.yaml:

# repos:
# - repo: local
#   hooks:
#   - id: resource-interface-validation
#     name: Resource Interface Validation
#     entry: scripts/resources/examples/pre-commit-hook.sh
#     language: script
#     files: 'scripts/resources/.*/manage\.sh$'
#     pass_filenames: false
#     stages: [commit]
#     verbose: true

# Then run:
# pre-commit install
# pre-commit run --all-files  # Test on all files
# pre-commit run resource-interface-validation  # Test specific hook

# =============================================================================
# Advanced Configuration Examples
# =============================================================================

# Environment-specific configuration:
# You can override configuration using environment variables:

# export RESOURCE_VALIDATION_LEVEL="standard"
# export RESOURCE_VALIDATION_PARALLEL="false"
# export RESOURCE_VALIDATION_STRICT="true"
# export RESOURCE_VALIDATION_TIMEOUT="60"

# Example usage in different scenarios:

# 1. Development (fast feedback):
#    VALIDATION_LEVEL="quick"
#    VALIDATE_CHANGED_ONLY="true"
#    USE_PARALLEL="true"

# 2. CI/CD (comprehensive):
#    VALIDATION_LEVEL="standard"
#    VALIDATE_CHANGED_ONLY="false"
#    STRICT_MODE="true"

# 3. Release (thorough):
#    VALIDATION_LEVEL="full"
#    VALIDATE_CHANGED_ONLY="false"
#    STRICT_MODE="true"
#    VALIDATION_TIMEOUT="300"

# =============================================================================
# Troubleshooting
# =============================================================================

# Common issues and solutions:

# 1. "Permission denied" error:
#    chmod +x .git/hooks/pre-commit
#    chmod +x scripts/resources/tools/validate-interfaces.sh

# 2. "Validation framework not found" error:
#    - Ensure you're in the project root directory
#    - Check that scripts/resources/tools/validate-interfaces.sh exists

# 3. Validation timeout:
#    - Increase VALIDATION_TIMEOUT value
#    - Enable parallel processing: USE_PARALLEL="true"
#    - Use quicker validation level: VALIDATION_LEVEL="quick"

# 4. Too many false positives:
#    - Set STRICT_MODE="false"
#    - Use VALIDATION_LEVEL="quick" during development
#    - Add [skip-resource-validation] to commit message for emergency commits

# 5. Performance issues:
#    - Enable parallel processing: USE_PARALLEL="true"
#    - Use VALIDATE_CHANGED_ONLY="true" for incremental validation
#    - Reduce VALIDATION_TIMEOUT for faster feedback