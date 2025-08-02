#!/bin/bash
# ====================================================================
# Basic Assertion Functions
# ====================================================================
#
# Core assertion functions for equality, string operations, and basic
# value checks.
#
# Functions:
#   - assert_equals()          - Assert two values are equal
#   - assert_not_equals()      - Assert two values are not equal
#   - assert_contains()        - Assert string contains substring
#   - assert_not_contains()    - Assert string does not contain substring
#   - assert_empty()           - Assert string is empty
#   - assert_not_empty()       - Assert string is not empty
#   - assert_greater_than()    - Assert number is greater than
#   - assert_greater_than_or_equal() - Assert number is greater than or equal
#
# ====================================================================

# Test assertion counter (shared across all assertion modules)
TEST_ASSERTIONS=${TEST_ASSERTIONS:-0}
FAILED_ASSERTIONS=${FAILED_ASSERTIONS:-0}
PASSED_ASSERTIONS=${PASSED_ASSERTIONS:-0}

# Colors for assertion output
ASSERT_GREEN='\033[0;32m'
ASSERT_RED='\033[0;31m'
ASSERT_YELLOW='\033[1;33m'
ASSERT_NC='\033[0m'

# Assert two values are equal
assert_equals() {
    local actual="$1"
    local expected="$2"
    local message="${3:-Equality assertion}"
    
    TEST_ASSERTIONS=$((TEST_ASSERTIONS + 1))
    
    if [[ "$actual" == "$expected" ]]; then
        echo -e "${ASSERT_GREEN}✓${ASSERT_NC} $message"
        PASSED_ASSERTIONS=$((PASSED_ASSERTIONS + 1))
        return 0
    else
        echo -e "${ASSERT_RED}✗${ASSERT_NC} $message"
        echo "  Expected: '$expected'"
        echo "  Actual:   '$actual'"
        FAILED_ASSERTIONS=$((FAILED_ASSERTIONS + 1))
        return 1
    fi
}

# Assert two values are not equal
assert_not_equals() {
    local actual="$1"
    local unexpected="$2"
    local message="${3:-Inequality assertion}"
    
    TEST_ASSERTIONS=$((TEST_ASSERTIONS + 1))
    
    if [[ "$actual" != "$unexpected" ]]; then
        echo -e "${ASSERT_GREEN}✓${ASSERT_NC} $message"
        PASSED_ASSERTIONS=$((PASSED_ASSERTIONS + 1))
        return 0
    else
        echo -e "${ASSERT_RED}✗${ASSERT_NC} $message"
        echo "  Should not equal: '$unexpected'"
        echo "  Actual:          '$actual'"
        FAILED_ASSERTIONS=$((FAILED_ASSERTIONS + 1))
        return 1
    fi
}

# Assert string contains substring
assert_contains() {
    local haystack="$1"
    local needle="$2"
    local message="${3:-Contains assertion}"
    
    TEST_ASSERTIONS=$((TEST_ASSERTIONS + 1))
    
    if [[ "$haystack" == *"$needle"* ]]; then
        echo -e "${ASSERT_GREEN}✓${ASSERT_NC} $message"
        PASSED_ASSERTIONS=$((PASSED_ASSERTIONS + 1))
        return 0
    else
        echo -e "${ASSERT_RED}✗${ASSERT_NC} $message"
        echo "  String: '$haystack'"
        echo "  Should contain: '$needle'"
        FAILED_ASSERTIONS=$((FAILED_ASSERTIONS + 1))
        return 1
    fi
}

# Assert string does not contain substring
assert_not_contains() {
    local haystack="$1"
    local needle="$2"
    local message="${3:-Does not contain assertion}"
    
    TEST_ASSERTIONS=$((TEST_ASSERTIONS + 1))
    
    if [[ "$haystack" != *"$needle"* ]]; then
        echo -e "${ASSERT_GREEN}✓${ASSERT_NC} $message"
        PASSED_ASSERTIONS=$((PASSED_ASSERTIONS + 1))
        return 0
    else
        echo -e "${ASSERT_RED}✗${ASSERT_NC} $message"
        echo "  String: '$haystack'"
        echo "  Should not contain: '$needle'"
        FAILED_ASSERTIONS=$((FAILED_ASSERTIONS + 1))
        return 1
    fi
}

# Assert string is empty
assert_empty() {
    local value="$1"
    local message="${2:-Empty assertion}"
    
    TEST_ASSERTIONS=$((TEST_ASSERTIONS + 1))
    
    if [[ -z "$value" ]]; then
        echo -e "${ASSERT_GREEN}✓${ASSERT_NC} $message"
        PASSED_ASSERTIONS=$((PASSED_ASSERTIONS + 1))
        return 0
    else
        echo -e "${ASSERT_RED}✗${ASSERT_NC} $message"
        echo "  Expected empty, got: '$value'"
        FAILED_ASSERTIONS=$((FAILED_ASSERTIONS + 1))
        return 1
    fi
}

# Assert string is not empty
assert_not_empty() {
    local value="$1"
    local message="${2:-Not empty assertion}"
    
    TEST_ASSERTIONS=$((TEST_ASSERTIONS + 1))
    
    if [[ -n "$value" ]]; then
        echo -e "${ASSERT_GREEN}✓${ASSERT_NC} $message"
        PASSED_ASSERTIONS=$((PASSED_ASSERTIONS + 1))
        return 0
    else
        echo -e "${ASSERT_RED}✗${ASSERT_NC} $message"
        echo "  Expected non-empty value, got empty string"
        FAILED_ASSERTIONS=$((FAILED_ASSERTIONS + 1))
        return 1
    fi
}

# Assert number is greater than
assert_greater_than() {
    local actual="$1"
    local expected="$2"
    local message="${3:-Greater than assertion}"
    
    TEST_ASSERTIONS=$((TEST_ASSERTIONS + 1))
    
    if [[ "$actual" -gt "$expected" ]]; then
        echo -e "${ASSERT_GREEN}✓${ASSERT_NC} $message"
        PASSED_ASSERTIONS=$((PASSED_ASSERTIONS + 1))
        return 0
    else
        echo -e "${ASSERT_RED}✗${ASSERT_NC} $message"
        echo "  Expected: > $expected, got: $actual"
        FAILED_ASSERTIONS=$((FAILED_ASSERTIONS + 1))
        return 1
    fi
}

# Assert number is greater than or equal
assert_greater_than_or_equal() {
    local actual="$1"
    local expected="$2"
    local message="${3:-Greater than or equal assertion}"
    
    TEST_ASSERTIONS=$((TEST_ASSERTIONS + 1))
    
    if [[ "$actual" -ge "$expected" ]]; then
        echo -e "${ASSERT_GREEN}✓${ASSERT_NC} $message"
        PASSED_ASSERTIONS=$((PASSED_ASSERTIONS + 1))
        return 0
    else
        echo -e "${ASSERT_RED}✗${ASSERT_NC} $message"
        echo "  Expected: >= $expected, got: $actual"
        FAILED_ASSERTIONS=$((FAILED_ASSERTIONS + 1))
        return 1
    fi
}