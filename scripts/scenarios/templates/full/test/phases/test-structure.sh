#!/bin/bash
# Validates scenario structure using convention-over-configuration.
# Standard structure is tested by default. Use .vrooli/testing.json to define exceptions.
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/structure.sh"

# ONE-LINER: Validate standard structure with config-driven exceptions
testing::structure::validate_all

# Optional: Add custom structure checks here if needed
# Example: testing::phase::check "Custom file exists" test -f custom/file.txt
