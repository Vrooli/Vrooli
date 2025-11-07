#!/bin/bash
# Validates scenario structure using convention-over-configuration.
# Standard structure is tested by default. Use .vrooli/testing.json to define exceptions.
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/structure.sh"

testing::phase::init --target-time "30s"

# ONE-LINER: Validate standard structure with config-driven exceptions
testing::structure::validate_all --scenario "$TESTING_PHASE_SCENARIO_NAME"

# Optional: Add custom structure checks here if needed
# Example: testing::phase::check "Custom file exists" test -f custom/file.txt

testing::phase::end_with_summary "Structure validation completed"
