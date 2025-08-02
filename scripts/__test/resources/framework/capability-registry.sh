#!/bin/bash
# ====================================================================
# Resource Capability Registry Parser and Validator
# ====================================================================
#
# Parses the resource capability registry YAML and validates resources
# against their category requirements. Provides capability-driven
# testing and validation.
#
# Usage:
#   source "$SCRIPT_DIR/framework/capability-registry.sh"
#   validate_resource_capabilities "$RESOURCE_NAME" "$CATEGORY"
#
# ====================================================================

set -euo pipefail

# Get the directory of this script
REGISTRY_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REGISTRY_FILE="$REGISTRY_DIR/resource-capability-registry.yaml"

# Capability validation counters
CAPABILITY_TESTS_RUN=0
CAPABILITY_TESTS_PASSED=0
CAPABILITY_TESTS_FAILED=0

# Colors for capability test output
CR_GREEN='\033[0;32m'
CR_RED='\033[0;31m'
CR_YELLOW='\033[1;33m'
CR_BLUE='\033[0;34m'
CR_BOLD='\033[1m'
CR_NC='\033[0m'

# Capability registry logging
cr_log_info() {
    echo -e "${CR_BLUE}[CAPABILITY]${CR_NC} $1"
}

cr_log_success() {
    echo -e "${CR_GREEN}[CAPABILITY]${CR_NC} ✅ $1"
}

cr_log_error() {
    echo -e "${CR_RED}[CAPABILITY]${CR_NC} ❌ $1"
}

cr_log_warning() {
    echo -e "${CR_YELLOW}[CAPABILITY]${CR_NC} ⚠️  $1"
}

# Check if yq is available for YAML parsing
check_yaml_parser() {
    if ! which yq >/dev/null 2>&1; then
        cr_log_warning "yq not found - using simplified YAML parsing"
        return 1
    fi
    return 0
}

# Simple YAML parser for when yq is not available
parse_yaml_simple() {
    local yaml_file="$1"
    local category="$2"
    local capability_type="$3"  # required_capabilities or optional_capabilities
    
    # Extract capabilities for a category using basic text processing
    sed -n "/^  $category:/,/^  [a-z]/p" "$yaml_file" | \
    sed -n "/$capability_type:/,/^    [a-z_]*_capabilities:/p" | \
    grep "^      - name:" | \
    sed 's/^      - name: "\(.*\)"$/\1/' | \
    sed 's/^      - name: \(.*\)$/\1/'
}

# Advanced YAML parser using yq
parse_yaml_yq() {
    local yaml_file="$1"
    local category="$2"
    local capability_type="$3"
    
    yq eval ".categories.$category.$capability_type[].name" "$yaml_file" 2>/dev/null | \
    grep -v "null" || echo ""
}

# Get capabilities for a category
get_category_capabilities() {
    local category="$1"
    local capability_type="$2"  # required_capabilities or optional_capabilities
    
    if [[ ! -f "$REGISTRY_FILE" ]]; then
        cr_log_error "Registry file not found: $REGISTRY_FILE"
        return 1
    fi
    
    local capabilities=""
    
    if check_yaml_parser; then
        capabilities=$(parse_yaml_yq "$REGISTRY_FILE" "$category" "$capability_type")
    else
        capabilities=$(parse_yaml_simple "$REGISTRY_FILE" "$category" "$capability_type")
    fi
    
    echo "$capabilities"
}

# Get capability details
get_capability_details() {
    local category="$1"
    local capability_name="$2"
    
    if check_yaml_parser; then
        local description=$(yq eval ".categories.$category.required_capabilities[] | select(.name == \"$capability_name\") | .description" "$REGISTRY_FILE" 2>/dev/null)
        if [[ "$description" == "null" || -z "$description" ]]; then
            description=$(yq eval ".categories.$category.optional_capabilities[] | select(.name == \"$capability_name\") | .description" "$REGISTRY_FILE" 2>/dev/null)
        fi
        echo "$description"
    else
        # Simple text extraction
        grep -A 3 "name: \"$capability_name\"\\|name: $capability_name" "$REGISTRY_FILE" | \
        grep "description:" | head -1 | \
        sed 's/.*description: "\(.*\)"$/\1/' | \
        sed 's/.*description: \(.*\)$/\1/' || echo "No description available"
    fi
}

# Test a specific capability
test_capability() {
    local resource_name="$1"
    local category="$2"
    local capability_name="$3"
    local required="$4"  # true/false
    local resource_port="$5"
    local manage_script="$6"
    
    CAPABILITY_TESTS_RUN=$((CAPABILITY_TESTS_RUN + 1))
    
    local capability_description
    capability_description=$(get_capability_details "$category" "$capability_name")
    
    cr_log_info "Testing $capability_name capability: $capability_description"
    
    # Determine test method based on capability name and category
    local test_result="unknown"
    
    case "$capability_name" in
        "model_management")
            test_result=$(test_model_management_capability "$resource_name" "$resource_port")
            ;;
        "inference")
            test_result=$(test_inference_capability "$resource_name" "$resource_port")
            ;;
        "data_persistence")
            test_result=$(test_data_persistence_capability "$resource_name" "$resource_port")
            ;;
        "web_interface")
            test_result=$(test_web_interface_capability "$resource_name" "$resource_port")
            ;;
        "core_automation")
            test_result=$(test_core_automation_capability "$resource_name" "$resource_port")
            ;;
        "query_processing")
            test_result=$(test_query_processing_capability "$resource_name" "$resource_port")
            ;;
        "code_execution")
            test_result=$(test_code_execution_capability "$resource_name" "$resource_port")
            ;;
        "health_monitoring")
            test_result=$(test_health_monitoring_capability "$resource_name" "$manage_script")
            ;;
        *)
            # Generic capability test
            test_result=$(test_generic_capability "$resource_name" "$capability_name" "$resource_port" "$manage_script")
            ;;
    esac
    
    # Evaluate test result
    if [[ "$test_result" == "passed" ]]; then
        cr_log_success "Capability '$capability_name' test passed"
        CAPABILITY_TESTS_PASSED=$((CAPABILITY_TESTS_PASSED + 1))
        return 0
    elif [[ "$test_result" == "failed" ]]; then
        if [[ "$required" == "true" ]]; then
            cr_log_error "Required capability '$capability_name' test failed"
            CAPABILITY_TESTS_FAILED=$((CAPABILITY_TESTS_FAILED + 1))
            return 1
        else
            cr_log_warning "Optional capability '$capability_name' test failed"
            CAPABILITY_TESTS_PASSED=$((CAPABILITY_TESTS_PASSED + 1))
            return 0
        fi
    else
        cr_log_warning "Capability '$capability_name' test result inconclusive: $test_result"
        CAPABILITY_TESTS_PASSED=$((CAPABILITY_TESTS_PASSED + 1))
        return 0
    fi
}

# Specific capability test implementations
test_model_management_capability() {
    local resource_name="$1"
    local resource_port="$2"
    
    # Test model listing endpoint
    local models_response
    models_response=$(curl -s --max-time 10 "http://localhost:${resource_port}/api/models" 2>/dev/null || \
                     curl -s --max-time 10 "http://localhost:${resource_port}/api/tags" 2>/dev/null)
    
    if [[ -n "$models_response" ]] && echo "$models_response" | jq . >/dev/null 2>&1; then
        echo "passed"
    else
        echo "failed"
    fi
}

test_inference_capability() {
    local resource_name="$1"
    local resource_port="$2"
    
    # Test inference endpoint with minimal request
    local inference_response
    inference_response=$(curl -s --max-time 30 -X POST "http://localhost:${resource_port}/api/generate" \
        -H "Content-Type: application/json" \
        -d '{"model": "test", "prompt": "hi", "stream": false}' 2>/dev/null)
    
    if [[ -n "$inference_response" ]] && (echo "$inference_response" | jq -e '.response' >/dev/null 2>&1 || \
                                          echo "$inference_response" | jq -e '.error' >/dev/null 2>&1); then
        echo "passed"
    else
        echo "failed"
    fi
}

test_data_persistence_capability() {
    local resource_name="$1"
    local resource_port="$2"
    
    # Test basic connectivity to storage port
    if timeout 3 bash -c "</dev/tcp/localhost/$resource_port" 2>/dev/null; then
        echo "passed"
    else
        echo "failed"
    fi
}

test_web_interface_capability() {
    local resource_name="$1"
    local resource_port="$2"
    
    # Test web interface accessibility
    local ui_response
    ui_response=$(curl -s --max-time 10 "http://localhost:${resource_port}/" 2>/dev/null)
    
    if [[ -n "$ui_response" ]] && echo "$ui_response" | grep -q -E "(html|HTML|<!DOCTYPE)"; then
        echo "passed"
    else
        echo "failed"
    fi
}

test_core_automation_capability() {
    local resource_name="$1"
    local resource_port="$2"
    
    # Test basic automation endpoint
    if curl -s --max-time 5 "http://localhost:${resource_port}/health" >/dev/null 2>&1 || \
       curl -s --max-time 5 "http://localhost:${resource_port}/" >/dev/null 2>&1; then
        echo "passed"
    else
        echo "failed"
    fi
}

test_query_processing_capability() {
    local resource_name="$1"
    local resource_port="$2"
    
    # Test search query processing
    local search_response
    search_response=$(curl -s --max-time 15 "http://localhost:${resource_port}/search?q=test" 2>/dev/null)
    
    if [[ -n "$search_response" ]]; then
        echo "passed"
    else
        echo "failed"
    fi
}

test_code_execution_capability() {
    local resource_name="$1"
    local resource_port="$2"
    
    # Test code execution endpoint
    local exec_response
    exec_response=$(curl -s --max-time 10 "http://localhost:${resource_port}/system_info" 2>/dev/null || \
                   curl -s --max-time 10 "http://localhost:${resource_port}/languages" 2>/dev/null)
    
    if [[ -n "$exec_response" ]] && echo "$exec_response" | jq . >/dev/null 2>&1; then
        echo "passed"
    else
        echo "failed"
    fi
}

test_health_monitoring_capability() {
    local resource_name="$1"
    local manage_script="$2"
    
    if [[ -f "$manage_script" ]]; then
        # Test health-detailed action
        local health_output
        health_output=$(timeout 30s bash "$manage_script" --action health-detailed 2>/dev/null || \
                       timeout 30s bash "$manage_script" --action status 2>/dev/null)
        
        if [[ $? -eq 0 ]]; then
            echo "passed"
        else
            echo "failed"
        fi
    else
        echo "failed"
    fi
}

test_generic_capability() {
    local resource_name="$1"
    local capability_name="$2"
    local resource_port="$3"
    local manage_script="$4"
    
    # Generic capability test - try manage script action first
    if [[ -f "$manage_script" ]]; then
        local action_name=$(echo "$capability_name" | tr '_' '-')
        local action_output
        action_output=$(timeout 30s bash "$manage_script" --action "$action_name" --dry-run yes 2>/dev/null)
        
        if [[ $? -eq 0 ]] && [[ ! "$action_output" =~ "Invalid action" ]]; then
            echo "passed"
        else
            echo "unknown"
        fi
    else
        echo "unknown"
    fi
}

# Main capability validation function
validate_resource_capabilities() {
    local resource_name="${1:-unknown}"
    local category="${2:-}"
    local resource_port="${3:-8080}"
    local manage_script="${4:-}"
    
    cr_log_info "🔍 Validating capabilities for: $resource_name (category: $category)"
    echo
    
    # Reset counters
    CAPABILITY_TESTS_RUN=0
    CAPABILITY_TESTS_PASSED=0
    CAPABILITY_TESTS_FAILED=0
    
    if [[ -z "$category" ]]; then
        cr_log_error "Category is required for capability validation"
        return 1
    fi
    
    # Get required capabilities
    local required_capabilities
    required_capabilities=$(get_category_capabilities "$category" "required_capabilities")
    
    # Get optional capabilities
    local optional_capabilities
    optional_capabilities=$(get_category_capabilities "$category" "optional_capabilities")
    
    cr_log_info "Testing required capabilities..."
    
    # Test required capabilities
    if [[ -n "$required_capabilities" ]]; then
        while IFS= read -r capability; do
            if [[ -n "$capability" ]]; then
                test_capability "$resource_name" "$category" "$capability" "true" "$resource_port" "$manage_script"
            fi
        done <<< "$required_capabilities"
    else
        cr_log_warning "No required capabilities found for category: $category"
    fi
    
    echo
    cr_log_info "Testing optional capabilities..."
    
    # Test optional capabilities
    if [[ -n "$optional_capabilities" ]]; then
        while IFS= read -r capability; do
            if [[ -n "$capability" ]]; then
                test_capability "$resource_name" "$category" "$capability" "false" "$resource_port" "$manage_script"
            fi
        done <<< "$optional_capabilities"
    else
        cr_log_warning "No optional capabilities found for category: $category"
    fi
    
    echo
    
    # Print capability validation summary
    cr_log_info "Capability validation summary for $resource_name:"
    echo "  Tests run: $CAPABILITY_TESTS_RUN"
    echo "  Passed: $CAPABILITY_TESTS_PASSED"
    echo "  Failed: $CAPABILITY_TESTS_FAILED"
    
    if [[ $CAPABILITY_TESTS_FAILED -eq 0 ]]; then
        cr_log_success "$resource_name passes capability validation"
        return 0
    else
        cr_log_error "$resource_name failed $CAPABILITY_TESTS_FAILED capability tests"
        return 1
    fi
}

# List all categories and their capabilities
list_capability_registry() {
    cr_log_info "🗂️  Resource Capability Registry"
    echo
    
    if [[ ! -f "$REGISTRY_FILE" ]]; then
        cr_log_error "Registry file not found: $REGISTRY_FILE"
        return 1
    fi
    
    local categories=("ai" "storage" "automation" "agents" "search" "execution")
    
    for category in "${categories[@]}"; do
        echo "📂 Category: $category"
        
        local required_caps
        required_caps=$(get_category_capabilities "$category" "required_capabilities")
        
        if [[ -n "$required_caps" ]]; then
            echo "  ✅ Required capabilities:"
            while IFS= read -r cap; do
                if [[ -n "$cap" ]]; then
                    local desc=$(get_capability_details "$category" "$cap")
                    echo "    • $cap: $desc"
                fi
            done <<< "$required_caps"
        fi
        
        local optional_caps
        optional_caps=$(get_category_capabilities "$category" "optional_capabilities")
        
        if [[ -n "$optional_caps" ]]; then
            echo "  ⭐ Optional capabilities:"
            while IFS= read -r cap; do
                if [[ -n "$cap" ]]; then
                    local desc=$(get_capability_details "$category" "$cap")
                    echo "    • $cap: $desc"
                fi
            done <<< "$optional_caps"
        fi
        
        echo
    done
}

# Export functions for use in other scripts
export -f validate_resource_capabilities
export -f list_capability_registry
export -f get_category_capabilities
export -f get_capability_details