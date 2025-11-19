#!/bin/bash
# Runs Browser Automation Studio workflow automations from requirements registry.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/integration.sh"

SCENARIO_DIR="$(cd "${BASH_SOURCE[0]%/*}/../.." && pwd)"
node "${SCENARIO_DIR}/test/playbooks/scripts/build-registry.mjs"

testing::integration::validate_all
