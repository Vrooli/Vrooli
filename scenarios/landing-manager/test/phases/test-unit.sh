#!/bin/bash
# Orchestrates language unit tests with coverage thresholds.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
SCENARIO_DIR="${SCENARIO_DIR:-$(cd "${BASH_SOURCE[0]%/*}/../.." && pwd)}"

source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/unit.sh"

# Load scenario environment variables for tests (required for DB access, etc.)
if [ -f "${SCENARIO_DIR}/initialization/configuration/landing-manager.env" ]; then
    set -a
    source "${SCENARIO_DIR}/initialization/configuration/landing-manager.env"
    set +a
fi

testing::unit::validate_all
