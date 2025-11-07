#!/bin/bash
# Business-layer validation: ensure core workflow capabilities are wired correctly
# Uses .vrooli/endpoints.json as source of truth for API endpoints and CLI commands

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/business.sh"

testing::phase::init --target-time "90s"

# Validate API endpoints from .vrooli/endpoints.json
testing::business::validate_endpoints

# Validate CLI commands from .vrooli/endpoints.json
testing::business::validate_cli_commands

# Validate WebSocket endpoints from .vrooli/endpoints.json
testing::business::validate_websockets

testing::phase::end_with_summary "Business logic validation completed"
