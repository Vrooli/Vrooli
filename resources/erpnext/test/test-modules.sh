#!/usr/bin/env bash
# Test script for e-commerce and manufacturing modules

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"
source "${APP_ROOT}/resources/erpnext/config/defaults.sh"

log::info "Testing E-commerce and Manufacturing Modules"

# Test e-commerce help
log::info "Testing e-commerce help..."
if vrooli resource erpnext ecommerce --help &>/dev/null; then
    log::success "E-commerce help works"
else
    log::error "E-commerce help failed"
fi

# Test manufacturing help  
log::info "Testing manufacturing help..."
if vrooli resource erpnext manufacturing --help &>/dev/null; then
    log::success "Manufacturing help works"
else
    log::error "Manufacturing help failed"
fi

# Test API ping (doesn't require auth)
log::info "Testing API connectivity..."
if timeout 2 curl -sf http://localhost:8020/api/method/ping -H "Host: vrooli.local" | grep -q "pong"; then
    log::success "API connectivity works"
else
    log::error "API connectivity failed"
fi

log::info "Module test complete"