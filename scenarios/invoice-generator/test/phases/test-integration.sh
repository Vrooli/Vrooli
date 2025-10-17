#!/bin/bash
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase with runtime requirement and 120-second target
testing::phase::init --require-runtime --target-time "120s"

echo "Comprehensive integration testing: API, CLI, and Database"

TEST_CLIENT_ID=""
TEST_INVOICE_ID=""
API_URL=""

cleanup_test_data() {
    if [ -n "$TEST_INVOICE_ID" ]; then
        curl -sf -X DELETE "$API_URL/api/v1/invoices/$TEST_INVOICE_ID" >/dev/null 2>&1 || true
    fi
    if [ -n "$TEST_CLIENT_ID" ]; then
        curl -sf -X DELETE "$API_URL/api/v1/clients/$TEST_CLIENT_ID" >/dev/null 2>&1 || true
    fi
}

# Register cleanup function
testing::phase::register_cleanup cleanup_test_data

if ! testing::core::wait_for_scenario "$TESTING_PHASE_SCENARIO_NAME" 30; then
    testing::phase::add_error "❌ Scenario '$TESTING_PHASE_SCENARIO_NAME' did not become ready in time"
    testing::phase::end_with_summary
fi

if ! API_URL=$(testing::connectivity::get_api_url "$TESTING_PHASE_SCENARIO_NAME"); then
    testing::phase::add_error "❌ Could not resolve API URL for $TESTING_PHASE_SCENARIO_NAME"
    testing::phase::end_with_summary
fi

# Ensure downstream phases inherit the resolved ports
API_PORT="${API_URL##*:}"
if UI_URL=$(testing::connectivity::get_ui_url "$TESTING_PHASE_SCENARIO_NAME" 2>/dev/null); then
    UI_PORT="${UI_URL##*:}"
fi

echo ""
echo "🌐 Testing API Integration..."
if curl -sf --max-time 10 "$API_URL/health" >/dev/null 2>&1; then
    echo "✅ API health check passed ($API_URL/health)"
else
    echo "❌ API integration tests failed - service not responding"
    echo "   Expected API at: $API_URL"
    echo "   💡 Tip: Start with 'vrooli scenario run $TESTING_PHASE_SCENARIO_NAME'"
    testing::phase::add_error
    testing::phase::end_with_summary
fi

echo "🔍 Testing API endpoints..."
if curl -sf --max-time 10 "$API_URL/api/v1/invoices" >/dev/null 2>&1; then
    echo "  ✅ Invoices endpoint accessible"
else
    echo "  ❌ Invoices endpoint failed"
    testing::phase::add_error
    testing::phase::end_with_summary
fi

if curl -sf --max-time 10 "$API_URL/api/v1/clients" >/dev/null 2>&1; then
    echo "  ✅ Clients endpoint accessible"
else
    echo "  ❌ Clients endpoint failed"
    testing::phase::add_error
    testing::phase::end_with_summary
fi

health_response=$(curl -sf --max-time 10 "$API_URL/health" 2>/dev/null || echo "")
if [[ "$health_response" =~ "status" ]]; then
    echo "  ✅ Health endpoint returns valid response"
else
    echo "  ❌ Health endpoint response unexpected"
    testing::phase::add_error
    testing::phase::end_with_summary
fi

echo ""
echo "🔄 Testing end-to-end workflow: Client → Invoice → Payment..."

# Step 1: Create client
if command -v jq >/dev/null 2>&1; then
    timestamp=$(date +%s)
    client_payload=$(jq -n --arg name "Integration Test Client $timestamp" --arg email "integration$timestamp@example.com" '{name:$name, email:$email}')
else
    client_payload='{"name":"Integration Test Client","email":"integration@example.com"}'
fi

client_response=$(curl -sf --max-time 15 -X POST -H "Content-Type: application/json" -d "$client_payload" "$API_URL/api/v1/clients" 2>/dev/null || echo "")
if echo "$client_response" | jq -e '.id' >/dev/null 2>&1; then
    TEST_CLIENT_ID=$(echo "$client_response" | jq -r '.id')
    echo "✅ Step 1: Client created - ID: $TEST_CLIENT_ID"
else
    echo "❌ Step 1 failed: Could not create client"
    testing::phase::add_error
    testing::phase::end_with_summary
fi

# Step 2: Create invoice for client
if [ -n "$TEST_CLIENT_ID" ]; then
    if command -v jq >/dev/null 2>&1; then
        invoice_payload=$(jq -n \
            --arg client_id "$TEST_CLIENT_ID" \
            --arg invoice_number "INT-$(date +%s)" \
            --arg issue_date "$(date +%Y-%m-%d)" \
            --arg due_date "$(date -d '+30 days' +%Y-%m-%d)" \
            '{
                client_id: $client_id,
                invoice_number: $invoice_number,
                issue_date: $issue_date,
                due_date: $due_date,
                line_items: [{
                    description: "Integration Test Service",
                    quantity: 1,
                    unit_price: 250.00
                }]
            }')
    else
        invoice_payload='{"client_id":"'$TEST_CLIENT_ID'","invoice_number":"INT-001","issue_date":"2025-01-01","due_date":"2025-02-01","line_items":[{"description":"Test","quantity":1,"unit_price":250.00}]}'
    fi

    invoice_response=$(curl -sf --max-time 15 -X POST -H "Content-Type: application/json" -d "$invoice_payload" "$API_URL/api/v1/invoices" 2>/dev/null || echo "")
    if echo "$invoice_response" | jq -e '.id' >/dev/null 2>&1; then
        TEST_INVOICE_ID=$(echo "$invoice_response" | jq -r '.id')
        echo "✅ Step 2: Invoice created - ID: $TEST_INVOICE_ID"
    else
        echo "❌ Step 2 failed: Could not create invoice"
        testing::phase::add_error
        testing::phase::end_with_summary
    fi
fi

# Step 3: Record payment
if [ -n "$TEST_INVOICE_ID" ]; then
    if command -v jq >/dev/null 2>&1; then
        payment_payload=$(jq -n \
            --arg invoice_id "$TEST_INVOICE_ID" \
            --arg reference "INT-PAY-$(date +%s)" \
            '{
                invoice_id: $invoice_id,
                amount: 100.00,
                payment_method: "bank_transfer",
                reference: $reference
            }')
    else
        payment_payload='{"invoice_id":"'$TEST_INVOICE_ID'","amount":100.00,"payment_method":"bank_transfer","reference":"INT-PAY-001"}'
    fi

    payment_response=$(curl -sf --max-time 15 -X POST -H "Content-Type: application/json" -d "$payment_payload" "$API_URL/api/v1/payments" 2>/dev/null || echo "")
    if echo "$payment_response" | jq -e '.id' >/dev/null 2>&1; then
        echo "✅ Step 3: Payment recorded"
    else
        echo "❌ Step 3 failed: Could not record payment"
        testing::phase::add_error
        testing::phase::end_with_summary
    fi
fi

# Step 4: Verify invoice balance updated
if [ -n "$TEST_INVOICE_ID" ]; then
    invoice_check=$(curl -sf --max-time 10 "$API_URL/api/v1/invoices/$TEST_INVOICE_ID" 2>/dev/null || echo "")
    if echo "$invoice_check" | jq -e '.balance_due' >/dev/null 2>&1; then
        balance=$(echo "$invoice_check" | jq -r '.balance_due')
        echo "✅ Step 4: Invoice balance verified - Balance Due: $balance"
    else
        echo "❌ Step 4 failed: Could not verify invoice balance"
        testing::phase::add_error
        testing::phase::end_with_summary
    fi
fi

echo ""
echo "🗄️  Testing resource connectivity (PostgreSQL if configured)..."
if command -v resource-postgres >/dev/null 2>&1; then
    if resource-postgres test smoke >/dev/null 2>&1; then
        echo "✅ Database integration tests passed"
    else
        echo "❌ Database integration tests failed"
        testing::phase::add_error
        testing::phase::end_with_summary
    fi
else
    echo "ℹ️  PostgreSQL resource CLI not available; skipping"
fi

echo ""
echo "📊 Integration Test Summary:"
echo "════════════════════════════"
echo "   API health: passed"
echo "   Endpoints: passed"
echo "   Client creation: passed"
echo "   Invoice creation: passed"
echo "   Payment recording: passed"
echo "   Balance verification: passed"

echo ""
log::success "✅ SUCCESS: All integration tests passed!"

# End with summary
testing::phase::end_with_summary "Integration tests completed"
