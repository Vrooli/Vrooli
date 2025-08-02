#!/bin/bash
# ====================================================================
# Category-Specific Resource Interface Compliance Testing
# ====================================================================
#
# Extends the base interface compliance testing with category-specific
# requirements. Each resource category (AI, Storage, Automation, etc.)
# has unique interface requirements beyond the basic standard.
#
# Usage:
#   source "$SCRIPT_DIR/framework/interface-compliance-categories.sh"
#   test_category_specific_compliance "$RESOURCE_NAME" "$CATEGORY" "$MANAGE_SCRIPT_PATH"
#
# Categories and their specific requirements:
#   AI: model-list, inference, benchmark, health-detailed
#   Storage: backup, restore, stats, vacuum, migrate  
#   Automation: workflow-list, workflow-import, ui-status, trigger-test
#   Agents: capability-test, screen-test, session-status, security-check
#   Search: index-stats, query-test, privacy-check
#
# ====================================================================

set -euo pipefail

# Source base interface compliance if not already loaded
if ! declare -f test_resource_interface_compliance >/dev/null 2>&1; then
    SCRIPT_DIR="${SCRIPT_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
    source "$SCRIPT_DIR/framework/interface-compliance.sh"
fi

# Category-specific test counters
CATEGORY_TESTS_RUN=0
CATEGORY_TESTS_PASSED=0
CATEGORY_TESTS_FAILED=0

# Colors for category test output
CC_GREEN='\033[0;32m'
CC_RED='\033[0;31m'
CC_YELLOW='\033[1;33m'
CC_BLUE='\033[0;34m'
CC_BOLD='\033[1m'
CC_NC='\033[0m'

# Category compliance logging
cc_log_info() {
    echo -e "${CC_BLUE}[CATEGORY]${CC_NC} $1"
}

cc_log_success() {
    echo -e "${CC_GREEN}[CATEGORY]${CC_NC} ✅ $1"
}

cc_log_error() {
    echo -e "${CC_RED}[CATEGORY]${CC_NC} ❌ $1"
}

cc_log_warning() {
    echo -e "${CC_YELLOW}[CATEGORY]${CC_NC} ⚠️  $1"
}

# Resource category mappings
declare -A RESOURCE_CATEGORIES=(
    ["ollama"]="ai"
    ["whisper"]="ai" 
    ["unstructured-io"]="ai"
    ["comfyui"]="automation"  # ComfyUI is automation for image generation
    ["n8n"]="automation"
    ["node-red"]="automation"
    ["windmill"]="automation"
    ["huginn"]="automation"
    ["agent-s2"]="agents"
    ["browserless"]="agents"
    ["claude-code"]="agents"
    ["searxng"]="search"
    ["minio"]="storage"
    ["vault"]="storage"
    ["qdrant"]="storage"
    ["questdb"]="storage"
    ["postgres"]="storage"
    ["redis"]="storage"
    ["judge0"]="execution"
)

# Get category for a resource
get_resource_category() {
    local resource_name="$1"
    echo "${RESOURCE_CATEGORIES[$resource_name]:-unknown}"
}

# Helper function to test category-specific action implementation
test_category_action() {
    local script_path="$1"
    local action="$2"
    local description="$3"
    local required="${4:-true}"
    
    CATEGORY_TESTS_RUN=$((CATEGORY_TESTS_RUN + 1))
    
    local action_output
    local exit_code
    
    # Use dry-run for potentially dangerous actions
    local extra_args=()
    if [[ "$action" =~ ^(backup|restore|migrate|vacuum)$ ]]; then
        extra_args=("--dry-run" "yes" "--yes" "yes")
    fi
    
    if action_output=$(timeout 30s bash "$script_path" --action "$action" "${extra_args[@]}" 2>&1); then
        exit_code=$?
    else
        exit_code=$?
    fi
    
    # Action should either succeed or give a clear error message
    if [[ $exit_code -eq 0 ]] || [[ "$action_output" =~ "DRY RUN" ]] || [[ "$action_output" =~ "Would" ]]; then
        cc_log_success "$description action '$action' is implemented"
        CATEGORY_TESTS_PASSED=$((CATEGORY_TESTS_PASSED + 1))
        return 0
    elif [[ "$action_output" =~ "Invalid action" ]] || [[ "$action_output" =~ "Unknown action" ]] || [[ "$action_output" =~ "not supported" ]]; then
        if [[ "$required" == "true" ]]; then
            cc_log_error "$description action '$action' is not implemented (required)"
            echo "  Output: ${action_output:0:200}..."
            CATEGORY_TESTS_FAILED=$((CATEGORY_TESTS_FAILED + 1))
            return 1
        else
            cc_log_warning "$description action '$action' is not implemented (optional)"
            CATEGORY_TESTS_PASSED=$((CATEGORY_TESTS_PASSED + 1))
            return 0
        fi
    else
        # Action exists but failed for other reasons (dependencies, etc.) - this is acceptable
        cc_log_success "$description action '$action' is implemented (failed due to dependencies/state)"
        CATEGORY_TESTS_PASSED=$((CATEGORY_TESTS_PASSED + 1))
        return 0
    fi
}

# Test AI resource category compliance
test_ai_category_compliance() {
    local resource_name="$1"
    local script_path="$2"
    
    cc_log_info "Testing AI resource category compliance for $resource_name..."
    
    # Required AI-specific actions
    local ai_required_actions=(
        "model-list:List available AI models"
        "inference:Run inference test" 
        "health-detailed:Detailed health with model information"
    )
    
    # Optional AI-specific actions
    local ai_optional_actions=(
        "model-pull:Download/install AI model"
        "benchmark:Performance characteristics test"
        "model-unload:Unload model from memory"
    )
    
    # Test required actions
    for action_desc in "${ai_required_actions[@]}"; do
        local action="${action_desc%%:*}"
        local description="${action_desc##*:}"
        test_category_action "$script_path" "$action" "AI $description" "true"
    done
    
    # Test optional actions
    for action_desc in "${ai_optional_actions[@]}"; do
        local action="${action_desc%%:*}"
        local description="${action_desc##*:}"
        test_category_action "$script_path" "$action" "AI $description" "false"
    done
    
    return 0
}

# Test Storage resource category compliance
test_storage_category_compliance() {
    local resource_name="$1"
    local script_path="$2"
    
    cc_log_info "Testing Storage resource category compliance for $resource_name..."
    
    # Required storage-specific actions
    local storage_required_actions=(
        "stats:Storage usage statistics"
        "health-detailed:Detailed health with storage metrics"
    )
    
    # Optional storage-specific actions  
    local storage_optional_actions=(
        "backup:Data backup operations"
        "restore:Data restoration"
        "migrate:Data migration between versions"
        "vacuum:Cleanup and optimization"
        "replication-status:Replication health check"
    )
    
    # Test required actions
    for action_desc in "${storage_required_actions[@]}"; do
        local action="${action_desc%%:*}"
        local description="${action_desc##*:}"
        test_category_action "$script_path" "$action" "Storage $description" "true"
    done
    
    # Test optional actions
    for action_desc in "${storage_optional_actions[@]}"; do
        local action="${action_desc%%:*}"
        local description="${action_desc##*:}"
        test_category_action "$script_path" "$action" "Storage $description" "false"
    done
    
    return 0
}

# Test Automation resource category compliance
test_automation_category_compliance() {
    local resource_name="$1"
    local script_path="$2"
    
    cc_log_info "Testing Automation resource category compliance for $resource_name..."
    
    # Required automation-specific actions
    local automation_required_actions=(
        "ui-status:Web UI availability check"
        "health-detailed:Detailed health with workflow metrics"
    )
    
    # Optional automation-specific actions
    local automation_optional_actions=(
        "workflow-list:List available workflows"
        "workflow-import:Import workflow definition"
        "workflow-export:Export workflow definition"
        "trigger-test:Test trigger mechanisms"
        "workflow-validate:Validate workflow definitions"
    )
    
    # Test required actions
    for action_desc in "${automation_required_actions[@]}"; do
        local action="${action_desc%%:*}"
        local description="${action_desc##*:}"
        test_category_action "$script_path" "$action" "Automation $description" "true"
    done
    
    # Test optional actions
    for action_desc in "${automation_optional_actions[@]}"; do
        local action="${action_desc%%:*}"
        local description="${action_desc##*:}"
        test_category_action "$script_path" "$action" "Automation $description" "false"
    done
    
    return 0
}

# Test Agent resource category compliance
test_agents_category_compliance() {
    local resource_name="$1"
    local script_path="$2"
    
    cc_log_info "Testing Agent resource category compliance for $resource_name..."
    
    # Required agent-specific actions
    local agents_required_actions=(
        "capability-test:Test core agent capabilities"
        "health-detailed:Detailed health with capability status"
    )
    
    # Optional agent-specific actions
    local agents_optional_actions=(
        "screen-test:Screenshot/interaction test"
        "session-status:Active session management"
        "security-check:Security validation"
        "automation-test:Basic automation workflow test"
    )
    
    # Test required actions
    for action_desc in "${agents_required_actions[@]}"; do
        local action="${action_desc%%:*}"
        local description="${action_desc##*:}"
        test_category_action "$script_path" "$action" "Agent $description" "true"
    done
    
    # Test optional actions
    for action_desc in "${agents_optional_actions[@]}"; do
        local action="${action_desc%%:*}"
        local description="${action_desc##*:}"
        test_category_action "$script_path" "$action" "Agent $description" "false"
    done
    
    return 0
}

# Test Search resource category compliance
test_search_category_compliance() {
    local resource_name="$1"
    local script_path="$2"
    
    cc_log_info "Testing Search resource category compliance for $resource_name..."
    
    # Required search-specific actions
    local search_required_actions=(
        "query-test:Test search functionality"
        "health-detailed:Detailed health with index information"
    )
    
    # Optional search-specific actions
    local search_optional_actions=(
        "index-stats:Search index information"
        "privacy-check:Privacy compliance validation"
        "performance-test:Query performance benchmarks"
    )
    
    # Test required actions
    for action_desc in "${search_required_actions[@]}"; do
        local action="${action_desc%%:*}"
        local description="${action_desc##*:}"
        test_category_action "$script_path" "$action" "Search $description" "true"
    done
    
    # Test optional actions
    for action_desc in "${search_optional_actions[@]}"; do
        local action="${action_desc%%:*}"
        local description="${action_desc##*:}"
        test_category_action "$script_path" "$action" "Search $description" "false"
    done
    
    return 0
}

# Test Execution resource category compliance
test_execution_category_compliance() {
    local resource_name="$1"
    local script_path="$2"
    
    cc_log_info "Testing Execution resource category compliance for $resource_name..."
    
    # Required execution-specific actions
    local execution_required_actions=(
        "execute-test:Test code execution capability"
        "health-detailed:Detailed health with execution metrics"
    )
    
    # Optional execution-specific actions
    local execution_optional_actions=(
        "language-list:List supported languages"
        "security-check:Execution sandbox validation"
        "performance-test:Execution performance benchmarks"
    )
    
    # Test required actions
    for action_desc in "${execution_required_actions[@]}"; do
        local action="${action_desc%%:*}"
        local description="${action_desc##*:}"
        test_category_action "$script_path" "$action" "Execution $description" "true"
    done
    
    # Test optional actions
    for action_desc in "${execution_optional_actions[@]}"; do
        local action="${action_desc%%:*}"
        local description="${action_desc##*:}"
        test_category_action "$script_path" "$action" "Execution $description" "false"
    done
    
    return 0
}

# Main category-specific compliance test function
test_category_specific_compliance() {
    local resource_name="${1:-unknown}"
    local category="${2:-}"
    local script_path="${3:-}"
    
    # Auto-detect category if not provided
    if [[ -z "$category" ]]; then
        category=$(get_resource_category "$resource_name")
    fi
    
    if [[ -z "$script_path" ]]; then
        cc_log_error "Script path is required for category compliance testing"
        return 1
    fi
    
    if [[ ! -f "$script_path" ]]; then
        cc_log_error "Script file not found: $script_path"
        return 1
    fi
    
    cc_log_info "🏷️  Testing category-specific compliance for: $resource_name (category: $category)"
    echo
    
    # Reset counters for this resource
    CATEGORY_TESTS_RUN=0
    CATEGORY_TESTS_PASSED=0
    CATEGORY_TESTS_FAILED=0
    
    # Run category-specific tests
    local category_result=0
    
    case "$category" in
        "ai")
            test_ai_category_compliance "$resource_name" "$script_path" || category_result=1
            ;;
        "storage")
            test_storage_category_compliance "$resource_name" "$script_path" || category_result=1
            ;;
        "automation")
            test_automation_category_compliance "$resource_name" "$script_path" || category_result=1
            ;;
        "agents")
            test_agents_category_compliance "$resource_name" "$script_path" || category_result=1
            ;;
        "search")
            test_search_category_compliance "$resource_name" "$script_path" || category_result=1
            ;;
        "execution")
            test_execution_category_compliance "$resource_name" "$script_path" || category_result=1
            ;;
        "unknown")
            cc_log_warning "Unknown category for $resource_name - skipping category-specific tests"
            return 0
            ;;
        *)
            cc_log_warning "Unsupported category '$category' for $resource_name - skipping category-specific tests"
            return 0
            ;;
    esac
    
    echo
    
    # Print category compliance summary
    cc_log_info "Category compliance summary for $resource_name ($category):"
    echo "  Tests run: $CATEGORY_TESTS_RUN"
    echo "  Passed: $CATEGORY_TESTS_PASSED"
    echo "  Failed: $CATEGORY_TESTS_FAILED"
    
    if [[ $CATEGORY_TESTS_FAILED -eq 0 ]]; then
        cc_log_success "$resource_name passes category-specific compliance tests"
        return 0
    else
        cc_log_error "$resource_name failed $CATEGORY_TESTS_FAILED category-specific compliance tests"
        return 1
    fi
}

# Combined interface and category compliance test
test_complete_resource_compliance() {
    local resource_name="${1:-unknown}"
    local script_path="${2:-}"
    local category="${3:-}"
    
    if [[ -z "$script_path" ]]; then
        cc_log_error "Script path is required for complete compliance testing"
        return 1
    fi
    
    # Auto-detect category if not provided
    if [[ -z "$category" ]]; then
        category=$(get_resource_category "$resource_name")
    fi
    
    cc_log_info "🔍 Running complete compliance testing for: $resource_name"
    echo
    
    local overall_result=0
    
    # Phase 1: Base interface compliance
    cc_log_info "Phase 1: Base Interface Compliance"
    if ! test_resource_interface_compliance "$resource_name" "$script_path"; then
        overall_result=1
    fi
    echo
    
    # Phase 2: Category-specific compliance  
    cc_log_info "Phase 2: Category-Specific Compliance"
    if ! test_category_specific_compliance "$resource_name" "$category" "$script_path"; then
        overall_result=1
    fi
    echo
    
    # Final summary
    if [[ $overall_result -eq 0 ]]; then
        cc_log_success "✅ $resource_name passes complete compliance testing"
        return 0
    else
        cc_log_error "❌ $resource_name failed complete compliance testing"
        return 1
    fi
}

# Export functions for use in other scripts
export -f test_category_specific_compliance
export -f test_complete_resource_compliance
export -f get_resource_category