#\!/usr/bin/env bash
#
# Browserless installation functions

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/resources/browserless/lib"
source "${APP_ROOT}/scripts/lib/utils/format.sh"
source "$SCRIPT_DIR/common.sh"

function install_browserless() {
    format_section "📦 Installing Browserless"
    
    # Create directories
    ensure_directories
    format_info "Created data directories"
    
    # Pull Docker image
    format_info "Pulling Browserless image..."
    if docker pull "$BROWSERLESS_IMAGE"; then
        format_success "Successfully pulled $BROWSERLESS_IMAGE"
    else
        format_error "Failed to pull Docker image"
        return 1
    fi
    
    # Register CLI with install-resource-cli.sh
    format_info "Registering Browserless CLI..."
    if "$SCRIPT_DIR/../../../lib/resources/install-resource-cli.sh" \
        --name browserless \
        --cli-path "$SCRIPT_DIR/../cli.sh"; then
        format_success "CLI registered successfully"
    else
        format_warning "Failed to register CLI"
    fi
    
    # Start the service
    format_info "Starting Browserless..."
    source "$SCRIPT_DIR/start.sh"
    if start_browserless; then
        format_success "Browserless installed and started successfully"
    else
        format_error "Failed to start Browserless"
        return 1
    fi
    
    # Run initial health check
    sleep 5
    if [[ "$(check_health)" == "healthy" ]]; then
        format_success "Health check passed"
    else
        format_warning "Service started but health check failed"
    fi
    
    format_info "Access Browserless at: http://localhost:${BROWSERLESS_PORT}"
    format_info "API Documentation: http://localhost:${BROWSERLESS_PORT}/docs"
}
