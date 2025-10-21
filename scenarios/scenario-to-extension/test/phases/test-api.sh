#!/bin/bash
# Test API functionality for scenario-to-extension
set -euo pipefail

# Get script directory and scenario root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
APP_ROOT="${APP_ROOT:-$(cd "$SCENARIO_DIR/../.." && pwd)}"

# Source logging utilities
source "${APP_ROOT}/scripts/lib/utils/log.sh"

log::info "Testing scenario-to-extension API..."

# Navigate to API directory
cd "$SCENARIO_DIR/api"

# Run Go tests
if go test -v ./...; then
    log::success "All API tests passed"
    exit 0
else
    log::error "API tests failed"
    exit 1
fi
