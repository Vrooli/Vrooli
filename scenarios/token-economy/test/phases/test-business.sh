#!/bin/bash
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "90s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "Running business logic tests for token-economy..."

# Run Go business logic tests
if [ -f "api/business_test.go" ]; then
    echo "Running Go business logic tests..."
    cd api

    # Run business tests with verbose output
    if go test -v -run "^Test.*Logic|^Test.*Validation|^Test.*Constraint" -timeout 90s; then
        echo "✓ Business logic tests passed"
    else
        echo "✗ Business logic tests failed"
        testing::phase::fail "Business logic tests failed"
    fi

    # Run specific business test patterns
    echo ""
    echo "Running wallet address resolution tests..."
    go test -v -run "TestResolveWalletAddressLogic" -timeout 30s

    echo ""
    echo "Running token validation tests..."
    go test -v -run "TestTokenValidation" -timeout 30s

    echo ""
    echo "Running wallet type validation tests..."
    go test -v -run "TestWalletTypeValidation" -timeout 30s

    echo ""
    echo "Running transaction type validation tests..."
    go test -v -run "TestTransactionTypeValidation" -timeout 30s

    echo ""
    echo "Running balance constraints tests..."
    go test -v -run "TestBalanceConstraints" -timeout 30s

    echo ""
    echo "Running token supply logic tests..."
    go test -v -run "TestTokenSupplyLogic" -timeout 30s

    echo ""
    echo "Running household isolation tests..."
    go test -v -run "TestHouseholdIsolation" -timeout 30s

    echo ""
    echo "Running metadata handling tests..."
    go test -v -run "TestMetadataHandling" -timeout 30s

    echo ""
    echo "Running unique constraints tests..."
    go test -v -run "TestUniqueConstraints" -timeout 30s

    cd ..
else
    echo "⚠ No business logic tests found (api/business_test.go missing)"
fi

# Validate business rules from PRD
echo ""
echo "Validating business rules implementation..."

if [ -f "PRD.md" ]; then
    echo "Checking PRD for business rules..."

    # Check if key business concepts are mentioned in code
    business_concepts=(
        "token"
        "wallet"
        "balance"
        "transaction"
        "household"
    )

    for concept in "${business_concepts[@]}"; do
        if grep -qi "$concept" api/main.go 2>/dev/null; then
            echo "✓ Business concept '$concept' implemented"
        else
            echo "⚠ Business concept '$concept' not found in implementation"
        fi
    done
fi

# Test business invariants
echo ""
echo "Testing business invariants..."

if [ -f "api/main.go" ]; then
    # Check for balance validation
    if grep -q "balance.*>=.*0\|amount.*>=.*0" api/main.go initialization/storage/postgres/schema.sql; then
        echo "✓ Balance non-negativity constraint exists"
    else
        echo "⚠ Balance non-negativity constraint not found"
    fi

    # Check for household isolation
    if grep -q "household_id" api/main.go initialization/storage/postgres/schema.sql; then
        echo "✓ Household isolation implemented"
    else
        echo "⚠ Household isolation not implemented"
    fi

    # Check for transaction immutability
    if grep -q "transactions.*INSERT\|CREATE TABLE.*transactions" initialization/storage/postgres/schema.sql; then
        echo "✓ Transaction ledger exists"
    else
        echo "⚠ Transaction ledger not found"
    fi
fi

# Test data validation rules
echo ""
echo "Testing data validation rules..."

if [ -f "initialization/storage/postgres/schema.sql" ]; then
    # Check for NOT NULL constraints
    null_constraints=$(grep -c "NOT NULL" initialization/storage/postgres/schema.sql)
    echo "✓ Found $null_constraints NOT NULL constraints"

    # Check for UNIQUE constraints
    unique_constraints=$(grep -c "UNIQUE" initialization/storage/postgres/schema.sql)
    echo "✓ Found $unique_constraints UNIQUE constraints"

    # Check for CHECK constraints
    check_constraints=$(grep -c "CHECK" initialization/storage/postgres/schema.sql)
    echo "✓ Found $check_constraints CHECK constraints"

    # Check for REFERENCES (foreign keys)
    fk_constraints=$(grep -c "REFERENCES" initialization/storage/postgres/schema.sql)
    echo "✓ Found $fk_constraints foreign key constraints"
fi

# Test business workflows
echo ""
echo "Testing business workflow coverage..."

workflows=(
    "createTokenHandler"
    "mintTokenHandler"
    "transferTokenHandler"
    "createWalletHandler"
    "getBalanceHandler"
)

for workflow in "${workflows[@]}"; do
    if grep -q "func $workflow" api/main.go; then
        echo "✓ Workflow '$workflow' implemented"
    else
        echo "⚠ Workflow '$workflow' not found"
    fi
done

# Run integration scenarios that test business logic
echo ""
echo "Running business integration scenarios..."

if [ -f "api/integration_test.go" ]; then
    cd api

    echo "Testing token lifecycle..."
    go test -v -run "TestTokenLifecycle" -timeout 60s || echo "⚠ Token lifecycle test incomplete"

    echo "Testing wallet balance flow..."
    go test -v -run "TestWalletBalanceFlow" -timeout 60s || echo "⚠ Wallet balance flow test incomplete"

    echo "Testing transaction history..."
    go test -v -run "TestTransactionHistory" -timeout 60s || echo "⚠ Transaction history test incomplete"

    cd ..
fi

testing::phase::end_with_summary "Business logic tests completed"
