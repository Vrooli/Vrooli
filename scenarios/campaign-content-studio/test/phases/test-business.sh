#!/bin/bash
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s"

cd "$TESTING_PHASE_SCENARIO_DIR"

# Ensure key configuration files exist and contain expected markers
if [ -f "initialization/configuration/app-config.json" ]; then
  testing::phase::check "App config contains campaign settings" grep -q "campaign" initialization/configuration/app-config.json
else
  testing::phase::add_warning "App configuration missing"
  testing::phase::add_test skipped
fi

if [ -f "initialization/configuration/content-templates.json" ]; then
  testing::phase::check "Content templates include blog entry" grep -q "blog" initialization/configuration/content-templates.json
else
  testing::phase::add_warning "Content templates missing"
  testing::phase::add_test skipped
fi

if [ -f "initialization/storage/seed.sql" ]; then
  testing::phase::check "Seed data inserts campaigns" grep -q "INSERT INTO campaigns" initialization/storage/seed.sql
  testing::phase::check "Seed data inserts documents" grep -q "INSERT INTO documents" initialization/storage/seed.sql
else
  testing::phase::add_warning "Seed SQL file missing"
  testing::phase::add_test skipped
fi

# Validate automation assets are present for orchestration
for workflow in initialization/automation/n8n/campaign-management.json \
                initialization/automation/n8n/content-generation.json; do
  testing::phase::check "Automation workflow available: ${workflow}" test -f "$workflow"
done

# Confirm CLI assets exist to deliver business workflows
if [ -f "cli/campaign-content-studio.bats" ]; then
  testing::phase::check "CLI BATS suite present" test -s "cli/campaign-content-studio.bats"
else
  testing::phase::add_warning "CLI BATS suite missing"
  testing::phase::add_test skipped
fi

# Ensure n8n workflow catalog is referenced in documentation
if [ -f "README.md" ]; then
  testing::phase::check "README documents automation workflows" grep -qi "n8n" README.md
fi

testing::phase::end_with_summary "Business logic validations completed"
