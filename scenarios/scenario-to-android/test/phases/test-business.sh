#!/usr/bin/env bash
# Scenario to Android - Business Logic Tests
#
# Tests the core business value and functionality of scenario-to-android.
# Validates that the scenario delivers on its PRD requirements.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Helper functions
pass() {
    echo -e "${GREEN}✓${NC} $1"
    TESTS_PASSED=$((TESTS_PASSED + 1))
    TESTS_RUN=$((TESTS_RUN + 1))
}

fail() {
    echo -e "${RED}✗${NC} $1"
    TESTS_FAILED=$((TESTS_FAILED + 1))
    TESTS_RUN=$((TESTS_RUN + 1))
}

test_header() {
    echo -e "\n${BLUE}━━━ $1${NC}"
}

echo -e "${BLUE}🎯 Business Logic Tests${NC}\n"

# ============================================================================
# Test 1: Android Templates Exist
# ============================================================================
test_header "Android Templates"

if [[ -d "$SCENARIO_DIR/initialization/templates/android" ]]; then
    TEMPLATE_COUNT=$(find "$SCENARIO_DIR/initialization/templates/android" -type f 2>/dev/null | wc -l)
    if [[ $TEMPLATE_COUNT -gt 0 ]]; then
        pass "Android templates exist ($TEMPLATE_COUNT files)"
    else
        fail "Android templates directory is empty"
    fi
else
    fail "Android templates directory missing"
fi

# ============================================================================
# Test 2: CLI Tool Functionality
# ============================================================================
test_header "CLI Tool"

# Check if CLI binary exists (may not be in PATH until terminal restart)
CLI_BINARY="${HOME}/.vrooli/bin/scenario-to-android"
if [[ -f "$CLI_BINARY" ]]; then
    pass "CLI binary installed at $CLI_BINARY"

    # Test using direct path if not in PATH
    CLI_CMD="$CLI_BINARY"
    if ! command -v scenario-to-android >/dev/null 2>&1; then
        echo -e "${YELLOW}⚠${NC}  CLI not in PATH yet (restart terminal to add to PATH)"
    else
        CLI_CMD="scenario-to-android"
        pass "CLI tool is accessible in PATH"
    fi

    # Test help command
    if "$CLI_CMD" help &>/dev/null; then
        pass "CLI help command works"
    else
        fail "CLI help command failed"
    fi

    # Test status command
    if "$CLI_CMD" status &>/dev/null; then
        pass "CLI status command works"
    else
        fail "CLI status command failed"
    fi
else
    fail "CLI binary not installed at expected location: $CLI_BINARY"
fi

# ============================================================================
# Test 3: API Endpoints (if running)
# ============================================================================
test_header "API Endpoints"

# API_PORT must be set by environment
if [[ -z "${API_PORT}" ]]; then
    warn "API_PORT not set, skipping API endpoint tests"
    echo "Skipped"
    exit 0
fi

# Check if API is running
if curl -sf "http://localhost:$API_PORT/health" &>/dev/null; then
    pass "API health endpoint responds"

    # Test status endpoint
    STATUS_JSON=$(curl -sf "http://localhost:$API_PORT/api/v1/status" 2>/dev/null)
    if [[ -n "$STATUS_JSON" ]]; then
        pass "API status endpoint responds"

        # Validate status response structure
        if echo "$STATUS_JSON" | grep -q '"build_system"'; then
            pass "API status includes build_system info"
        else
            fail "API status missing build_system field"
        fi

        # Check for required fields
        if echo "$STATUS_JSON" | grep -q '"ready"'; then
            pass "API status includes readiness check"
        else
            fail "API status missing ready field"
        fi
    else
        fail "API status endpoint not responding or returned empty response"
    fi
else
    echo -e "${YELLOW}⚠${NC}  API not running - skipping API tests"
fi

# ============================================================================
# Test 4: Build Configuration
# ============================================================================
test_header "Build Configuration"

if [[ -f "$SCENARIO_DIR/.vrooli/service.json" ]]; then
    SERVICE_JSON="$SCENARIO_DIR/.vrooli/service.json"

    # Check for android_config
    if grep -q '"android_config"' "$SERVICE_JSON"; then
        pass "Android configuration present in service.json"

        # Validate key fields
        if grep -q '"min_sdk"' "$SERVICE_JSON"; then
            pass "Minimum SDK version configured"
        else
            fail "Missing min_sdk configuration"
        fi

        if grep -q '"target_sdk"' "$SERVICE_JSON"; then
            pass "Target SDK version configured"
        else
            fail "Missing target_sdk configuration"
        fi
    else
        fail "Android configuration missing from service.json"
    fi
else
    fail "service.json not found"
fi

# ============================================================================
# Test 5: Core Business Value Verification
# ============================================================================
test_header "Business Value Verification"

# Verify the scenario can theoretically convert scenarios to Android
# (Full integration test would require Android SDK)

# Check for conversion script
if [[ -f "$SCENARIO_DIR/cli/convert.sh" ]]; then
    pass "Conversion script exists"
else
    fail "Conversion script missing"
fi

# Check for signing capability
if grep -q "keystore" "$SCENARIO_DIR/cli/convert.sh" 2>/dev/null; then
    pass "APK signing capability present in conversion script"
else
    echo -e "${YELLOW}⚠${NC}  APK signing capability not detected"
fi

# ============================================================================
# Test Summary
# ============================================================================
echo -e "\n${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}Business Logic Test Results:${NC}"
echo -e "  Tests run: $TESTS_RUN"
echo -e "  ${GREEN}Passed: $TESTS_PASSED${NC}"
echo -e "  ${RED}Failed: $TESTS_FAILED${NC}"

if [[ $TESTS_FAILED -gt 0 ]]; then
    echo -e "${RED}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    exit 1
else
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    exit 0
fi
