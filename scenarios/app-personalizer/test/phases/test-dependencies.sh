#!/bin/bash
# Dependencies validation using unified helper
# Validates runtimes, package managers, resources, and connectivity
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/dependencies.sh"

testing::phase::init --target-time "60s"

# ONE-LINER: Validate all dependencies automatically
# This helper:
# - Detects tech stack from service.json and file structure
# - Validates language runtimes (Go, Node.js, Python)
# - Checks package managers and dependency resolution
# - Tests resource health (postgres, redis, ollama, etc.)
# - Validates runtime connectivity (if scenario is running)
#
# All powered by `vrooli scenario status --json` with fallbacks to file detection

testing::dependencies::validate_all \
  --scenario "$TESTING_PHASE_SCENARIO_NAME"

testing::phase::end_with_summary "Dependency validation completed"
