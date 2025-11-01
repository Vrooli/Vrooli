#!/bin/bash
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

cd "$TESTING_PHASE_SCENARIO_DIR"

# Verify CLI install script is executable
if [ -f "cli/install.sh" ]; then
  testing::phase::check "CLI installer executable" test -x "cli/install.sh"
else
  testing::phase::add_warning "CLI installer missing"
  testing::phase::add_test skipped
fi

# Validate Go dependencies when toolchain is available
if [ -f "api/go.mod" ]; then
  if command -v go >/dev/null 2>&1; then
    testing::phase::check "Go toolchain resolves module" bash -c 'cd api && go env GOMOD >/dev/null'
  else
    testing::phase::add_warning "Go toolchain not available; skipping Go dependency check"
    testing::phase::add_test skipped
  fi
fi

# Validate UI dependency manifest without installing packages
if [ -f "ui/package.json" ]; then
  if command -v node >/dev/null 2>&1; then
    testing::phase::check "UI manifest parses" bash -c 'cd ui && node -e "require(\"./package.json\").name" >/dev/null'
  else
    testing::phase::add_warning "Node.js not available; skipping UI manifest check"
    testing::phase::add_test skipped
  fi
fi

# Ensure automation workflows that power scenario integrations are present
for workflow in initialization/automation/n8n/campaign-management.json \
                initialization/automation/n8n/content-generation.json \
                initialization/automation/n8n/document-processing.json \
                initialization/automation/n8n/search-retrieval.json; do
  testing::phase::check "Workflow available: ${workflow}" test -f "$workflow"
done

# Ensure storage seeds exist
for seed in initialization/storage/postgres/schema.sql initialization/storage/postgres/seed.sql; do
  testing::phase::check "Storage seed present: ${seed}" test -f "$seed"
done

testing::phase::end_with_summary "Dependency validation completed"
