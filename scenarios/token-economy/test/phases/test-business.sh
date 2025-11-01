#!/bin/bash
# Validate core business invariants and domain workflows
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "90s"

# Run focused Go tests that encode business rules
if [ -f "api/business_test.go" ]; then
  testing::phase::check "Go business suite" \
    bash -c 'cd api && go test -timeout 90s -run "Test(.*Logic|.*Validation|.*Constraints)"'
else
  testing::phase::add_warning "api/business_test.go missing; skipping business suite"
  testing::phase::add_test skipped
fi

# Spot-check PRD-aligned workflows existence
workflows=(
  "createTokenHandler"
  "mintTokenHandler"
  "transferTokenHandler"
  "createWalletHandler"
  "getBalanceHandler"
)
for workflow in "${workflows[@]}"; do
  if testing::phase::check "Implementation contains ${workflow}" \
       bash -c "grep -q 'func ${workflow}' api/main.go"; then
    :
  else
    testing::phase::add_warning "Missing handler ${workflow}"
  fi
done

# Validate schema invariants supporting business rules
if [ -f "initialization/storage/postgres/schema.sql" ]; then
  testing::phase::check "Balances constrained non-negative" \
    bash -c "grep -qi 'CHECK (.*balance.*>= 0' initialization/storage/postgres/schema.sql"

  testing::phase::check "Transactions table present" \
    bash -c "grep -qi 'CREATE TABLE.*transactions' initialization/storage/postgres/schema.sql"

  testing::phase::check "Household isolation columns exist" \
    bash -c "grep -qi 'household_id' initialization/storage/postgres/schema.sql"
else
  testing::phase::add_warning "Schema file missing for invariant checks"
  testing::phase::add_test skipped
fi

# Optional cross-check: ensure documentation references business decisions
if [ -f "PRD.md" ]; then
  testing::phase::check "PRD documents token supply policy" \
    bash -c 'grep -iq "supply" PRD.md'
else
  testing::phase::add_warning "PRD.md missing; cannot cross-check documentation"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Business validation completed"
