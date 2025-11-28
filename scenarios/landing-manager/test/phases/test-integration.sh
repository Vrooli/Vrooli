#!/bin/bash
# Runs Browser Automation Studio workflow automations from requirements registry.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
SCENARIO_DIR="${SCENARIO_DIR:-$(cd "${BASH_SOURCE[0]%/*}/../.." && pwd)}"

source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/integration.sh"

# Load scenario environment variables for tests (required for DB access, etc.)
if [ -f "${SCENARIO_DIR}/initialization/configuration/landing-manager.env" ]; then
    set -a
    source "${SCENARIO_DIR}/initialization/configuration/landing-manager.env"
    set +a
fi

# Landing-manager workflows involve scenario generation, lifecycle management, and promotion
# which can take 60-120+ seconds. Increase the default 45s timeout to accommodate these.
export TESTING_PLAYBOOKS_WORKFLOW_TIMEOUT=180

testing::integration::validate_all
