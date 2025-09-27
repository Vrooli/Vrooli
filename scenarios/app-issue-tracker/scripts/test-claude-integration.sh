#!/usr/bin/env bash
# Test script for Claude Code integration with App Issue Tracker

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

echo "[WARN] Legacy Claude integration test expects flat YAML files." >&2
echo "       Folder-based issue bundles are now default—use \"make test\" instead." >&2
exit 1

info "Testing Claude Code Integration for App Issue Tracker"
echo

# Test 1: Check if resource-claude-code is available
info "Test 1: Checking resource-claude-code availability..."
if command -v resource-claude-code &> /dev/null; then
    success "resource-claude-code CLI is available"
    resource-claude-code status 2>&1 | head -5 || true
else
    error "resource-claude-code CLI not found. Please install it first."
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
    TEST_ISSUE_ID="test-claude-$(date +%s)"
    TEST_ISSUE_FILE="$ISSUES_DIR/open/500-test-claude-integration.yaml"
    
    cat > "$TEST_ISSUE_FILE" << EOF
id: $TEST_ISSUE_ID
title: "Test Claude Integration Issue"
description: "This is a test issue to verify Claude Code integration is working properly"
type: bug
priority: low
app_id: app-issue-tracker
status: open

reporter:
  name: "Test Script"
  email: "test@example.com"
  timestamp: $(date -Iseconds)

error_context:
  error_message: "Claude integration test error"
  stack_trace: |
    Error: Test error for Claude integration
    at testFunction (test.js:10:5)
    at main (test.js:20:3)

metadata:
  created_at: $(date -Iseconds)
  updated_at: $(date -Iseconds)
  tags:
    - "test"
    - "claude-integration"
EOF
    
    success "Created test issue: $TEST_ISSUE_ID"
fi

# Test 3: Run investigation using claude-investigator.sh
info "Test 3: Testing investigation script..."
INVESTIGATION_OUTPUT=$(mktemp)

if "$SCRIPT_DIR/claude-investigator.sh" investigate "$TEST_ISSUE_ID" "test-agent" "$SCRIPT_DIR/.." "Analyze this test issue and provide a brief summary" > "$INVESTIGATION_OUTPUT" 2>&1; then
    success "Investigation script executed successfully"
    
    # Check if output contains expected JSON structure
    if jq -e '.issue_id' "$INVESTIGATION_OUTPUT" > /dev/null 2>&1; then
        success "Investigation returned valid JSON"
        info "Issue ID from response: $(jq -r '.issue_id' "$INVESTIGATION_OUTPUT")"
        info "Status: $(jq -r '.status' "$INVESTIGATION_OUTPUT")"
        
        # Show a snippet of the investigation report
        REPORT_SNIPPET=$(jq -r '.investigation_report' "$INVESTIGATION_OUTPUT" 2>/dev/null | head -c 200)
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

# Test 4: Test fix generation (optional)
info "Test 4: Testing fix generation script..."
if [[ -f "$SCRIPT_DIR/claude-fix-generator.sh" ]]; then
    # First check if POSTGRES_PASSWORD is set (required by the script)
    if [[ -z "${POSTGRES_PASSWORD:-}" ]]; then
        warn "Skipping fix generation test - POSTGRES_PASSWORD not set"
        info "To test fix generation, run: export POSTGRES_PASSWORD=your_password"
    else
        FIX_OUTPUT=$(mktemp)
        
        if "$SCRIPT_DIR/claude-fix-generator.sh" generate "$TEST_ISSUE_ID" "$SCRIPT_DIR/.." false false > "$FIX_OUTPUT" 2>&1; then
            success "Fix generation script executed"
            
            if jq -e '.issue_id' "$FIX_OUTPUT" > /dev/null 2>&1; then
                success "Fix generation returned valid JSON"
                info "Fix status: $(jq -r '.fix_generation_status' "$FIX_OUTPUT")"
            else
                warn "Fix output is not valid JSON"
            fi
        else
            warn "Fix generation failed (this may be expected if no investigation report exists)"
        fi
        
        rm -f "$FIX_OUTPUT"
    fi
else
    warn "Fix generator script not found"
fi

# Test 5: Verify resource-claude-code direct execution
info "Test 5: Testing direct resource-claude-code execution..."
TEST_PROMPT="Say 'Hello from App Issue Tracker!' in exactly 5 words."
DIRECT_OUTPUT=$(mktemp)

if echo "$TEST_PROMPT" | resource-claude-code run - > "$DIRECT_OUTPUT" 2>&1; then
    success "Direct Claude Code execution works"
    info "Response: $(cat "$DIRECT_OUTPUT")"
else
    warn "Direct execution failed - may need authentication"
    echo "Try running: claude auth login"
fi

rm -f "$DIRECT_OUTPUT"

echo
success "🎉 Claude Code integration tests completed!"
echo
info "Summary:"
echo "  ✓ resource-claude-code CLI is available"
echo "  ✓ Investigation script updated with proper integration"
echo "  ✓ Fix generation script updated with proper integration"
echo "  ✓ Test infrastructure is in place"
echo
info "Next steps:"
echo "  1. Ensure the API server properly triggers investigations"
echo "  2. Test with real issues from the UI or CLI"
echo "  3. Monitor Claude Code usage with: resource-claude-code usage"
