#!/usr/bin/env bash
# Validates domain-specific assets: seed data, workflows, and security sandbox wiring.
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "240s"

schema_file="initialization/postgres/schema.sql"
seed_file="initialization/postgres/seed.sql"

if testing::phase::check "Database schema present" test -f "$schema_file"; then
  for table in injection_techniques agent_configurations test_results tournaments tournament_results; do
    testing::phase::check "Schema defines table: ${table}" grep -q "$table" "$schema_file"
  done
  testing::phase::check "Schema defines robustness scoring" grep -qi "robustness" "$schema_file"
fi

if testing::phase::check "Seed data present" test -f "$seed_file"; then
  testing::phase::check "Seed injections populated" grep -q "INSERT INTO injection_techniques" "$seed_file"
fi

for workflow in initialization/n8n/security-sandbox.json initialization/n8n/injection-tester.json; do
  if testing::phase::check "Workflow exists: ${workflow}" test -f "$workflow"; then
    testing::phase::check "Workflow JSON valid: ${workflow}" jq empty "$workflow"
  fi
done

testing::phase::check "Security sandbox script" bash "$TESTING_PHASE_SCENARIO_DIR/test/test-security-sandbox.sh"

testing::phase::end_with_summary "Business logic validation completed"
