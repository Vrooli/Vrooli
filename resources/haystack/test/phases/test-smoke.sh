#!/usr/bin/env bash
################################################################################
# Haystack Smoke Tests - v2.0 Universal Contract Compliant
# 
# Quick validation that Haystack is installed and can start
# Must complete in <30 seconds per universal.yaml
################################################################################

set -euo pipefail

# Setup paths
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
HAYSTACK_LIB_DIR="${APP_ROOT}/resources/haystack/lib"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${HAYSTACK_LIB_DIR}/common.sh"

# Basic smoke test
log::info "Running Haystack smoke test..."

# Check if CLI exists
if [[ ! -x "${APP_ROOT}/resources/haystack/cli.sh" ]]; then
    log::error "Haystack CLI not found"
    exit 1
fi

# Get port
HAYSTACK_PORT=$(haystack::get_port)

# Check if service is running and healthy
if timeout 5 curl -sf "http://localhost:${HAYSTACK_PORT}/health" &>/dev/null; then
    log::success "Haystack is running and healthy"
    exit 0
else
    log::error "Haystack health check failed"
    exit 1
fi