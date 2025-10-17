#!/usr/bin/env bash
################################################################################
# Keycloak Theme Customization Tests
################################################################################

set -euo pipefail

# Determine script location
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
RESOURCE_DIR="$(cd "${TEST_DIR}/.." && pwd)"
APP_ROOT="$(cd "${RESOURCE_DIR}/../.." && pwd)"

# Source dependencies
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${RESOURCE_DIR}/config/defaults.sh"

log::info "Starting Keycloak theme customization tests..."

# Test theme creation
log::info "Testing theme creation..."
TEST_THEME="test-theme-$$"
if "${RESOURCE_DIR}/cli.sh" theme create "$TEST_THEME" > /dev/null 2>&1; then
    log::success "Theme creation successful"
else
    log::error "Theme creation failed"
    exit 1
fi

# Test theme listing
log::info "Testing theme listing..."
if "${RESOURCE_DIR}/cli.sh" theme list 2>&1 | grep -q "$TEST_THEME"; then
    log::success "Theme appears in list"
else
    log::error "Theme not found in list"
    exit 1
fi

# Test theme deployment
log::info "Testing theme deployment..."
if "${RESOURCE_DIR}/cli.sh" theme deploy "$TEST_THEME" > /dev/null 2>&1; then
    log::success "Theme deployment successful"
else
    log::error "Theme deployment failed"
    exit 1
fi

# Test theme customization
log::info "Testing theme customization..."
if "${RESOURCE_DIR}/cli.sh" theme customize "$TEST_THEME" primary-color "#ff0000" > /dev/null 2>&1; then
    log::success "Theme customization successful"
else
    log::error "Theme customization failed"
    exit 1
fi

# Test theme export
log::info "Testing theme export..."
EXPORT_FILE="/tmp/${TEST_THEME}-export.tar.gz"
if "${RESOURCE_DIR}/cli.sh" theme export "$TEST_THEME" "$EXPORT_FILE" > /dev/null 2>&1; then
    if [[ -f "$EXPORT_FILE" ]]; then
        log::success "Theme export successful"
        rm -f "$EXPORT_FILE"
    else
        log::error "Export file not created"
        exit 1
    fi
else
    log::error "Theme export failed"
    exit 1
fi

# Clean up test theme
log::info "Cleaning up test theme..."
if "${RESOURCE_DIR}/cli.sh" theme remove "$TEST_THEME" > /dev/null 2>&1; then
    log::success "Theme cleanup successful"
else
    log::warning "Theme cleanup failed (non-critical)"
fi

log::success "All theme tests passed"