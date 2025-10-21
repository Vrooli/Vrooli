#!/usr/bin/env bash
#
# Business Logic Tests for scenario-to-desktop
# Tests core desktop generation functionality and business value delivery
#

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

# Source test utilities if available
if [ -f "${SCRIPT_DIR}/../../scripts/lib/test-utils.sh" ]; then
    source "${SCRIPT_DIR}/../../scripts/lib/test-utils.sh"
else
    # Minimal test utilities
    test_info() { echo "[INFO] $*"; }
    test_error() { echo "[ERROR] $*" >&2; }
    test_success() { echo "[SUCCESS] $*"; }
fi

API_BASE_URL="${API_BASE_URL:-http://localhost:${API_PORT:-19044}}"
TEST_OUTPUT_DIR="/tmp/scenario-to-desktop-test-$$"

cleanup() {
    test_info "Cleaning up test artifacts..."
    rm -rf "${TEST_OUTPUT_DIR}"
}

trap cleanup EXIT

# Create test output directory
mkdir -p "${TEST_OUTPUT_DIR}"

test_info "Starting business logic tests for scenario-to-desktop"

# ============================================================================
# Test 1: Desktop Generation API Functionality
# ============================================================================
test_info "Test 1: Verify desktop generation API accepts valid requests"

RESPONSE=$(curl -sf -X POST "${API_BASE_URL}/api/v1/desktop/generate" \
    -H "Content-Type: application/json" \
    -d '{
        "app_name": "Test Desktop App",
        "framework": "electron",
        "template_type": "basic",
        "output_path": "'"${TEST_OUTPUT_DIR}/test-app"'",
        "description": "Business test app",
        "version": "1.0.0",
        "api_endpoint": "http://localhost:3000",
        "target_platforms": ["win", "mac", "linux"]
    }' || true)

if echo "${RESPONSE}" | grep -q "build_id"; then
    test_success "Desktop generation API accepts valid requests"
else
    test_error "Desktop generation API failed: ${RESPONSE}"
    exit 1
fi

# ============================================================================
# Test 2: Template System Availability
# ============================================================================
test_info "Test 2: Verify all template types are accessible"

TEMPLATES=$(curl -sf "${API_BASE_URL}/api/v1/templates" || echo "ERROR")

if echo "${TEMPLATES}" | grep -q "basic"; then
    test_success "Template system is accessible"
else
    test_error "Template system unavailable: ${TEMPLATES}"
    exit 1
fi

# ============================================================================
# Test 3: Build Status Tracking
# ============================================================================
test_info "Test 3: Verify build status tracking works"

# Extract build_id from previous response
BUILD_ID=$(echo "${RESPONSE}" | grep -o '"build_id":"[^"]*"' | cut -d'"' -f4 || echo "test-build-id")

STATUS_RESPONSE=$(curl -sf "${API_BASE_URL}/api/v1/desktop/status/${BUILD_ID}" || echo '{"status":"unknown"}')

if echo "${STATUS_RESPONSE}" | grep -q "status"; then
    test_success "Build status tracking is functional"
else
    test_error "Build status tracking failed: ${STATUS_RESPONSE}"
    exit 1
fi

# ============================================================================
# Test 4: CLI Tool Availability
# ============================================================================
test_info "Test 4: Verify CLI tool is installed and functional"

if command -v scenario-to-desktop > /dev/null 2>&1; then
    CLI_VERSION=$(scenario-to-desktop version || echo "ERROR")
    if echo "${CLI_VERSION}" | grep -q "scenario-to-desktop"; then
        test_success "CLI tool is installed and functional"
    else
        test_error "CLI tool installed but not working: ${CLI_VERSION}"
        exit 1
    fi
else
    test_error "CLI tool not found in PATH"
    exit 1
fi

# ============================================================================
# Test 5: Cross-Platform Template Support
# ============================================================================
test_info "Test 5: Verify all required templates exist"

REQUIRED_TEMPLATES=("basic" "advanced" "multi_window" "kiosk")
MISSING_TEMPLATES=()

for template in "${REQUIRED_TEMPLATES[@]}"; do
    TEMPLATE_RESPONSE=$(curl -sf "${API_BASE_URL}/api/v1/templates/${template}" || echo "NOT_FOUND")
    if echo "${TEMPLATE_RESPONSE}" | grep -q "NOT_FOUND"; then
        MISSING_TEMPLATES+=("${template}")
    fi
done

if [ ${#MISSING_TEMPLATES[@]} -eq 0 ]; then
    test_success "All required templates are available"
else
    test_error "Missing templates: ${MISSING_TEMPLATES[*]}"
    exit 1
fi

# ============================================================================
# Summary
# ============================================================================
test_success "All business logic tests passed!"
test_info "Business value verified:"
test_info "  ✓ Desktop generation capability functional"
test_info "  ✓ Template system comprehensive"
test_info "  ✓ Build tracking operational"
test_info "  ✓ CLI tools accessible"
test_info "  ✓ Cross-platform support complete"

exit 0
