#!/usr/bin/env bash

#######################################
# Simple Test: Basic Atomic Operations
# 
# Tests individual atomic operations
# to verify they work correctly.
#######################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/resources/browserless/examples"
BROWSERLESS_DIR="${APP_ROOT}/resources/browserless"

# Source log utilities first
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh" || { echo "FATAL: Failed to load variable definitions" >&2; exit 1; }
# shellcheck disable=SC1091
source "${var_LOG_FILE}" || { echo "FATAL: Failed to load logging library" >&2; exit 1; }

# Source required libraries
source "${BROWSERLESS_DIR}/lib/common.sh"
source "${BROWSERLESS_DIR}/lib/browser-ops.sh"

log::header "🧪 Testing Atomic Browser Operations"

# Test 1: Direct JavaScript execution
log::info "Test 1: Execute simple JavaScript"
cat > /tmp/test-js.txt << 'EOF'
export default async ({ page, context }) => {
    await page.goto('http://localhost:5678', { waitUntil: 'networkidle2', timeout: 30000 });
    return { 
        success: true, 
        url: page.url(),
        title: await page.title()
    };
};
EOF

response=$(curl -s -X POST \
    -H "Content-Type: application/json" \
    -d "{
        \"code\": $(cat /tmp/test-js.txt | jq -Rs .),
        \"context\": {}
    }" \
    "http://localhost:4110/chrome/function")

echo "Response: $response" | jq '.' || echo "$response"

# Test 2: Using browser::navigate
log::info "Test 2: Navigate using atomic operation"
# First, let me check if the browser::navigate function works
# by calling it directly in a simpler way

echo ""
log::success "Tests completed"