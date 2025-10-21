#!/bin/bash
# Structure validation phase for tech-tree-designer
# Verifies scenario follows Vrooli v2.0 contract standards

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "30s"

cd "$TESTING_PHASE_SCENARIO_DIR"

# Check required files exist
testing::phase::log "Validating required files..."

REQUIRED_FILES=(
    "PRD.md"
    "README.md"
    "Makefile"
    ".vrooli/service.json"
    "api/main.go"
    "api/go.mod"
    "ui/package.json"
    "ui/index.html"
    "cli/install.sh"
)

for file in "${REQUIRED_FILES[@]}"; do
    if [ ! -f "$file" ]; then
        testing::phase::error "Missing required file: $file"
        testing::phase::end_with_summary "Structure validation failed" 1
    fi
done

# Check Makefile has required targets
testing::phase::log "Validating Makefile targets..."

REQUIRED_TARGETS=(
    "help"
    "start"
    "run"
    "stop"
    "test"
    "logs"
    "status"
    "clean"
    "fmt"
    "lint"
)

for target in "${REQUIRED_TARGETS[@]}"; do
    if ! grep -q "^${target}:" Makefile; then
        testing::phase::error "Missing required Makefile target: $target"
        testing::phase::end_with_summary "Makefile validation failed" 1
    fi
done

# Check service.json structure
testing::phase::log "Validating service.json structure..."
if ! jq -e '.service.name' .vrooli/service.json &>/dev/null; then
    testing::phase::error "Invalid service.json: missing service.name"
    testing::phase::end_with_summary "service.json validation failed" 1
fi

if ! jq -e '.lifecycle.health' .vrooli/service.json &>/dev/null; then
    testing::phase::error "Invalid service.json: missing lifecycle.health"
    testing::phase::end_with_summary "service.json validation failed" 1
fi

# Check test phase structure
testing::phase::log "Validating test phase structure..."
if [ ! -d test/phases ]; then
    testing::phase::error "Missing test/phases directory"
    testing::phase::end_with_summary "Test structure validation failed" 1
fi

testing::phase::log "All structure checks passed"
testing::phase::end_with_summary "Structure validated" 0
