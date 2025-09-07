#!/usr/bin/env bash
# Test script to verify Claude Code integration

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

info "Testing Claude Code Integration for Scenario Generator V1"
echo

# Test 1: Check if resource-claude-code is available
info "Test 1: Checking resource-claude-code availability..."
if command -v resource-claude-code &> /dev/null; then
    success "resource-claude-code CLI is available"
else
    error "resource-claude-code CLI not found. Please install it first."
    exit 1
fi

# Test 2: Test basic command execution
info "Test 2: Testing basic Claude Code command..."
if resource-claude-code status &> /dev/null; then
    success "Claude Code status check passed"
else
    warn "Claude Code status check failed - may need authentication"
fi

# Test 3: Test the execute command with simple prompt
info "Test 3: Testing execute command..."
TEST_PROMPT="Say 'Hello, Scenario Generator!' in one line."
if echo "$TEST_PROMPT" | resource-claude-code execute - &> /dev/null; then
    success "Claude Code execute command works"
else
    error "Claude Code execute command failed"
    warn "You may need to run: claude auth login"
    exit 1
fi

# Test 4: Test Go API compilation
info "Test 4: Checking Go API compilation..."
cd api
if go build -o test-binary .; then
    success "Go API compiles successfully"
    rm -f test-binary
else
    error "Go API compilation failed"
    exit 1
fi
cd ..

# Test 5: Verify pipeline package imports
info "Test 5: Verifying pipeline package..."
if go list ./api/pipeline &> /dev/null; then
    success "Pipeline package is valid"
else
    error "Pipeline package has issues"
    exit 1
fi

echo
success "🎉 All Claude Code integration tests passed!"
info "The scenario generator is ready to use resource-claude-code for AI generation."
echo
info "Next steps:"
echo "  1. Start the API server: cd api && go run ."
echo "  2. Start the UI: cd ui && node server.js"
echo "  3. Access the dashboard at http://localhost:\${UI_PORT} (port varies by configuration)"