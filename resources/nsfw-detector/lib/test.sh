#!/usr/bin/env bash
# Test library functions for NSFW Detector resource

set -euo pipefail

# Resource configuration
readonly RESOURCE_NAME="nsfw-detector"
readonly RESOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
readonly TEST_DIR="${RESOURCE_DIR}/test"
readonly PHASES_DIR="${TEST_DIR}/phases"

# Colors for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly NC='\033[0m'

# Test execution wrapper
run_test_phase() {
    local phase="$1"
    local script="${PHASES_DIR}/test-${phase}.sh"
    
    if [[ ! -f "$script" ]]; then
        echo -e "${YELLOW}Warning: $phase test script not found${NC}"
        return 2
    fi
    
    if [[ ! -x "$script" ]]; then
        chmod +x "$script"
    fi
    
    echo -e "${YELLOW}Running $phase tests...${NC}"
    
    if "$script"; then
        echo -e "${GREEN}✓ $phase tests passed${NC}"
        return 0
    else
        echo -e "${RED}✗ $phase tests failed${NC}"
        return 1
    fi
}

# Smoke test wrapper
test_smoke() {
    run_test_phase "smoke"
}

# Unit test wrapper
test_unit() {
    run_test_phase "unit"
}

# Integration test wrapper
test_integration() {
    run_test_phase "integration"
}

# All tests wrapper
test_all() {
    local failed=0
    
    run_test_phase "smoke" || ((failed++))
    run_test_phase "unit" || ((failed++))
    run_test_phase "integration" || ((failed++))
    
    if [[ $failed -eq 0 ]]; then
        echo -e "${GREEN}All test phases passed!${NC}"
        return 0
    else
        echo -e "${RED}$failed test phase(s) failed${NC}"
        return 1
    fi
}

# Test helper functions
assert_equals() {
    local expected="$1"
    local actual="$2"
    local message="${3:-Assertion failed}"
    
    if [[ "$expected" != "$actual" ]]; then
        echo -e "${RED}✗ $message${NC}"
        echo "  Expected: $expected"
        echo "  Actual: $actual"
        return 1
    fi
    return 0
}

assert_contains() {
    local haystack="$1"
    local needle="$2"
    local message="${3:-Assertion failed}"
    
    if [[ "$haystack" != *"$needle"* ]]; then
        echo -e "${RED}✗ $message${NC}"
        echo "  String does not contain: $needle"
        return 1
    fi
    return 0
}

assert_file_exists() {
    local file="$1"
    local message="${2:-File not found}"
    
    if [[ ! -f "$file" ]]; then
        echo -e "${RED}✗ $message: $file${NC}"
        return 1
    fi
    return 0
}

assert_command_succeeds() {
    local command="$1"
    local message="${2:-Command failed}"
    
    if ! eval "$command" > /dev/null 2>&1; then
        echo -e "${RED}✗ $message: $command${NC}"
        return 1
    fi
    return 0
}

# Export functions for use in test scripts
export -f run_test_phase
export -f test_smoke
export -f test_unit
export -f test_integration
export -f test_all
export -f assert_equals
export -f assert_contains
export -f assert_file_exists
export -f assert_command_succeeds