#!/bin/bash
# ====================================================================
# Utility Assertion Functions
# ====================================================================
#
# Utility functions for test management, resource requirements, and
# reporting.
#
# Functions:
#   - require_resource()       - Assert resource is available
#   - require_resources()      - Require multiple resources
#   - require_tools()          - Assert required tools are available
#   - skip_test()              - Utility function to skip test
#   - print_assertion_summary() - Print assertion summary
#   - log_step()               - Log test step
#   - assert_with_context()    - Assert with detailed error context
#
# ====================================================================

# Require resource to be available (skip test if not)
require_resource() {
    local resource="$1"
    local message="${2:-Resource requirement}"
    
    # Check if resource is in healthy resources list
    local resource_available=false
    
    # First try to use HEALTHY_RESOURCES array if available
    if [[ -n "${HEALTHY_RESOURCES:-}" ]]; then
        for healthy_resource in "${HEALTHY_RESOURCES[@]}"; do
            if [[ "$healthy_resource" == "$resource" ]]; then
                resource_available=true
                break
            fi
        done
    # Fallback to checking HEALTHY_RESOURCES_STR string
    elif [[ -n "${HEALTHY_RESOURCES_STR:-}" ]]; then
        if [[ " $HEALTHY_RESOURCES_STR " == *" $resource "* ]]; then
            resource_available=true
        fi
    fi
    
    if [[ "$resource_available" == "true" ]]; then
        echo -e "${ASSERT_GREEN}✓${ASSERT_NC} Resource available: $resource"
        return 0
    else
        echo -e "${ASSERT_YELLOW}⚠${ASSERT_NC} Skipping test - required resource not available: $resource"
        exit 77  # Standard exit code for skipped tests
    fi
}

# Require multiple resources
require_resources() {
    local required_resources=("$@")
    
    for resource in "${required_resources[@]}"; do
        require_resource "$resource"
    done
}

# Require tools to be available
require_tools() {
    local tools=("$@")
    local missing_tools=()
    
    for tool in "${tools[@]}"; do
        if ! command -v "$tool" >/dev/null 2>&1; then
            missing_tools+=("$tool")
        fi
    done
    
    if [[ ${#missing_tools[@]} -gt 0 ]]; then
        echo -e "${ASSERT_YELLOW}⚠${ASSERT_NC} Skipping test - missing required tools: ${missing_tools[*]}"
        exit 77  # Standard exit code for skipped tests
    fi
    
    echo -e "${ASSERT_GREEN}✓${ASSERT_NC} All required tools available: ${tools[*]}"
}

# Utility function to skip test
skip_test() {
    local reason="${1:-Test skipped}"
    echo -e "${ASSERT_YELLOW}⚠${ASSERT_NC} $reason"
    exit 77  # Standard exit code for skipped tests
}

# Print assertion summary
print_assertion_summary() {
    echo
    echo "Assertion Summary:"
    echo "  Total assertions: $TEST_ASSERTIONS"
    echo "  Passed assertions: $PASSED_ASSERTIONS"
    echo "  Failed assertions: $FAILED_ASSERTIONS"
    echo "  Success rate: $(awk -v total="$TEST_ASSERTIONS" -v failed="$FAILED_ASSERTIONS" 'BEGIN {printf "%.1f", (total - failed) / (total == 0 ? 1 : total) * 100}')%"
    
    if [[ $FAILED_ASSERTIONS -gt 0 ]]; then
        return 1
    fi
    return 0
}

# Log test step
log_step() {
    local step_num="$1"
    local description="$2"
    echo -e "${ASSERT_YELLOW}[$step_num]${ASSERT_NC} $description"
}

# Assert with detailed error context
assert_with_context() {
    local assertion_result="$1"
    local context_info="$2"
    local message="$3"
    
    if [[ "$assertion_result" == "0" ]]; then
        return 0
    else
        echo -e "${ASSERT_RED}✗${ASSERT_NC} $message"
        echo "  Context: $context_info"
        FAILED_ASSERTIONS=$((FAILED_ASSERTIONS + 1))
        return 1
    fi
}