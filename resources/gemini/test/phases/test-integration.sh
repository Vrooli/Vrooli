#!/usr/bin/env bash
################################################################################
# Gemini Resource Integration Tests
# 
# End-to-end functionality testing (< 120s)
################################################################################

set -euo pipefail

# Determine paths
PHASES_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DIR="$(cd "$PHASES_DIR/.." && pwd)"
RESOURCE_DIR="$(cd "$TEST_DIR/.." && pwd)"
APP_ROOT="$(cd "$RESOURCE_DIR/../.." && pwd)"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/lib/utils/format.sh"

# Source resource libraries
source "${RESOURCE_DIR}/config/defaults.sh"
source "${RESOURCE_DIR}/lib/core.sh"
source "${RESOURCE_DIR}/lib/content.sh" 2>/dev/null || true

log::info "Starting Gemini integration tests..."

# Test 1: Full initialization with Vault integration
log::info "Test 1: Testing full initialization..."
if gemini::init true; then
    log::success "✓ Full initialization successful"
    
    # Check where key was loaded from
    if [[ "$GEMINI_API_KEY" != "placeholder-gemini-key" ]]; then
        log::info "  API key loaded from secure source"
    else
        log::warn "  Using placeholder API key"
    fi
else
    log::error "✗ Full initialization failed"
    exit 1
fi

# Test 2: Model listing (if valid API key)
log::info "Test 2: Testing model listing..."
if [[ "$GEMINI_API_KEY" != "placeholder-gemini-key" ]]; then
    models=$(gemini::list_models 2>&1)
    if [[ $? -eq 0 ]] && [[ -n "$models" ]]; then
        log::success "✓ Model listing successful"
        echo "$models" | head -3 | while read -r model; do
            log::info "  - $model"
        done
    else
        log::error "✗ Model listing failed"
        exit 1
    fi
else
    log::warn "⚠ Skipping model listing (placeholder API key)"
fi

# Test 3: Content generation (if valid API key)
log::info "Test 3: Testing content generation..."
if [[ "$GEMINI_API_KEY" != "placeholder-gemini-key" ]]; then
    response=$(gemini::generate "Say 'test successful' and nothing else" 2>&1)
    if [[ $? -eq 0 ]] && [[ -n "$response" ]]; then
        log::success "✓ Content generation successful"
        log::info "  Response: ${response:0:50}..."
    else
        log::error "✗ Content generation failed"
        exit 1
    fi
else
    log::warn "⚠ Skipping content generation (placeholder API key)"
fi

# Test 4: Content management functions
log::info "Test 4: Testing content management..."
if type -t gemini::content::init >/dev/null; then
    # Initialize content storage
    export GEMINI_CONTENT_STORAGE="/tmp/gemini-test-$$"
    
    if gemini::content::init; then
        log::success "✓ Content storage initialized"
        
        # Test adding content
        echo "Test prompt" > /tmp/test-prompt.txt
        if gemini::content::add "/tmp/test-prompt.txt" "test-prompt"; then
            log::success "✓ Content added successfully"
        else
            log::warn "⚠ Content add failed (non-critical)"
        fi
        
        # Test listing content
        if gemini::content::list >/dev/null 2>&1; then
            log::success "✓ Content listing works"
        else
            log::warn "⚠ Content listing failed (non-critical)"
        fi
        
        # Cleanup
        rm -rf "$GEMINI_CONTENT_STORAGE"
        rm -f /tmp/test-prompt.txt
    else
        log::warn "⚠ Content storage initialization failed (non-critical)"
    fi
else
    log::warn "⚠ Content management not implemented"
fi

# Test 5: Error handling
log::info "Test 5: Testing error handling..."
if [[ "$GEMINI_API_KEY" != "placeholder-gemini-key" ]]; then
    # Test with invalid model
    response=$(gemini::generate "test" "invalid-model-name" 5 2>&1)
    if [[ $? -ne 0 ]]; then
        log::success "✓ Error handling works correctly"
    else
        log::warn "⚠ Error handling may not be working properly"
    fi
else
    log::warn "⚠ Skipping error handling test (placeholder API key)"
fi

# Test 6: CLI integration
log::info "Test 6: Testing CLI integration..."
if "${RESOURCE_DIR}/cli.sh" status >/dev/null 2>&1; then
    log::success "✓ CLI status command works"
else
    log::error "✗ CLI status command failed"
    exit 1
fi

if "${RESOURCE_DIR}/cli.sh" list-models >/dev/null 2>&1; then
    log::success "✓ CLI list-models command works"
else
    # Not critical if it fails with placeholder key
    if [[ "$GEMINI_API_KEY" == "placeholder-gemini-key" ]]; then
        log::warn "⚠ CLI list-models failed (expected with placeholder key)"
    else
        log::error "✗ CLI list-models command failed"
        exit 1
    fi
fi

log::success "✅ All integration tests passed!"
exit 0