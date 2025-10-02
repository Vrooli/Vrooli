#!/usr/bin/env bash
# Test script for Codex integration with App Issue Tracker

set -euo pipefail

# Colors for output
RED='\033[1;31m'
GREEN='\033[1;32m'
YELLOW='\033[1;33m'
CYAN='\033[1;36m'
NC='\033[0m'

info() { echo -e "${CYAN}ℹ️  $1${NC}"; }
success() { echo -e "${GREEN}✅ $1${NC}"; }
error() { echo -e "${RED}❌ $1${NC}"; }
warn() { echo -e "${YELLOW}⚠️  $1${NC}"; }

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ISSUES_DIR="${SCRIPT_DIR}/../data/issues"

echo "[WARN] Legacy Codex integration test expects flat YAML files." >&2
echo "       Folder-based issue bundles are now default—use \"make test\" instead." >&2
exit 1

info "Testing Codex Integration for App Issue Tracker"
echo

# Test 1: Check if resource-codex is available
info "Test 1: Checking resource-codex availability..."
if command -v resource-codex &> /dev/null; then
    success "resource-codex CLI is available"
    resource-codex status 2>&1 | head -5 || true
else
    error "resource-codex CLI not found. Please install it first."
    exit 1
fi

# Test 2: Find a test issue to investigate
info "Test 2: Finding test issue..."
TEST_ISSUE_FILE=""
TEST_ISSUE_ID=""

# Look for an existing test issue in the open folder
for file in "$ISSUES_DIR/open"/*.yaml; do
    if [[ -f "$file" ]]; then
        TEST_ISSUE_FILE="$file"
        TEST_ISSUE_ID=$(grep "^id:" "$file" | sed 's/id: *//' | head -1)
        break
    fi
done

if [[ -n "$TEST_ISSUE_ID" ]]; then
    success "Found test issue: $TEST_ISSUE_ID"
    info "Issue file: $TEST_ISSUE_FILE"
else
    warn "No open issues found. Creating a test issue..."
    
    # Create a test issue
    TEST_ISSUE_ID="test-codex-$(date +%s)"
    TEST_ISSUE_FILE="$ISSUES_DIR/open/500-test-codex-integration.yaml"
    
    cat > "$TEST_ISSUE_FILE" << EOF
id: $TEST_ISSUE_ID
title: "Test Codex Integration Issue"
description: "This is a test issue to verify Codex integration is working properly"
type: bug
priority: low
app_id: app-issue-tracker
status: open

reporter:
  name: "Test Script"
  email: "test@example.com"
  timestamp: $(date -Iseconds)

error_context:
  error_message: "Codex integration test error"
  stack_trace: |
    Error: Test error for Codex integration
    at testFunction (test.js:10:5)
    at main (test.js:20:3)

metadata:
  created_at: $(date -Iseconds)
  updated_at: $(date -Iseconds)
  tags:
    - "test"
    - "codex-integration"
EOF
    
    success "Created test issue: $TEST_ISSUE_ID"
fi

# Test 3: Run unified investigation (includes fix recommendations)
info "Test 3: Testing unified investigation workflow..."
INVESTIGATION_OUTPUT=$(mktemp)

if "$SCRIPT_DIR/claude-investigator.sh" resolve "$TEST_ISSUE_ID" "test-agent" "$SCRIPT_DIR/.." "Analyze this test issue and provide a brief summary" > "$INVESTIGATION_OUTPUT" 2>&1; then
    success "Investigation executed successfully"

    # Check if output contains expected JSON structure
    if jq -e '.issue_id' "$INVESTIGATION_OUTPUT" > /dev/null 2>&1; then
        success "Investigation returned valid JSON"
        info "Issue ID from response: $(jq -r '.issue_id' "$INVESTIGATION_OUTPUT")"
        info "Status: $(jq -r '.status' "$INVESTIGATION_OUTPUT")"

        # Show a snippet of the investigation report (includes fixes)
        REPORT_SNIPPET=$(jq -r '.investigation.report' "$INVESTIGATION_OUTPUT" 2>/dev/null | head -c 200)
        if [[ -n "$REPORT_SNIPPET" ]]; then
            info "Report snippet: ${REPORT_SNIPPET}..."
        fi
    else
        warn "Investigation output is not valid JSON"
        echo "Output: $(head -n 10 "$INVESTIGATION_OUTPUT")"
    fi
else
    error "Investigation script failed"
    echo "Error output:"
    cat "$INVESTIGATION_OUTPUT"
fi

rm -f "$INVESTIGATION_OUTPUT"

# Test 4: Verify resource-codex direct execution
info "Test 4: Testing direct resource-codex execution..."
TEST_PROMPT="Say 'Hello from App Issue Tracker!' in exactly 5 words."
DIRECT_OUTPUT=$(mktemp)

if resource-codex content execute --context text --operation analyze "$TEST_PROMPT" > "$DIRECT_OUTPUT" 2>&1; then
    success "Direct Codex execution works"
    info "Response: $(cat "$DIRECT_OUTPUT")"
else
    warn "Direct execution failed - may need authentication"
    echo "Try running: resource-codex manage configure-cli"
fi

rm -f "$DIRECT_OUTPUT"

echo
success "🎉 Codex integration tests completed!"
echo
info "Summary:"
echo "  ✓ resource-codex or resource-claude-code CLI is available"
echo "  ✓ Unified investigation script (claude-investigator.sh) tested"
echo "  ✓ Fix generation via unified script tested"
echo "  ✓ Test infrastructure is in place"
echo
info "Next steps:"
echo "  1. Ensure the API server properly triggers investigations"
echo "  2. Test with real issues from the UI or CLI"
echo "  3. Monitor Codex status with: resource-codex status"
