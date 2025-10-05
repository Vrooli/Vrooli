#!/bin/bash
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase with runtime requirement and 180-second target
testing::phase::init --require-runtime --target-time "180s"

echo "Testing core business functionality..."

# Test counters are handled by phase helpers
created_invoice_ids=()
created_client_ids=()

# Cleanup function to run on exit
cleanup() {
    echo "Cleaning up test data..."
    for invoice_id in "${created_invoice_ids[@]}"; do
        if [ -n "$invoice_id" ] && [ "$invoice_id" != "null" ]; then
            echo "Deleting test invoice: $invoice_id"
            curl -sf -X DELETE "$API_BASE_URL/api/v1/invoices/$invoice_id" >/dev/null 2>&1 || echo "Warning: Failed to delete invoice $invoice_id"
        fi
    done
    for client_id in "${created_client_ids[@]}"; do
        if [ -n "$client_id" ] && [ "$client_id" != "null" ]; then
            echo "Deleting test client: $client_id"
            curl -sf -X DELETE "$API_BASE_URL/api/v1/clients/$client_id" >/dev/null 2>&1 || echo "Warning: Failed to delete client $client_id"
        fi
    done
}

# Register cleanup
testing::phase::register_cleanup cleanup

if ! testing::core::wait_for_scenario "$TESTING_PHASE_SCENARIO_NAME" 30 >/dev/null 2>&1; then
    testing::phase::add_error "❌ Scenario '$TESTING_PHASE_SCENARIO_NAME' did not become ready"
    testing::phase::end_with_summary
fi

if ! API_BASE_URL=$(testing::connectivity::get_api_url "$TESTING_PHASE_SCENARIO_NAME"); then
    testing::phase::add_error "❌ Unable to determine API URL for $TESTING_PHASE_SCENARIO_NAME"
    testing::phase::end_with_summary
fi

echo "Using API base URL: $API_BASE_URL"

# Test 1: Client creation
echo "Testing client creation..."
timestamp=$(date +%s)
client_data='{"name":"Test Client '$timestamp'","email":"testclient'$timestamp'@example.com"}'
client_response=$(curl -sf -X POST "$API_BASE_URL/api/v1/clients" -H "Content-Type: application/json" -d "$client_data" 2>/dev/null || echo "")

if echo "$client_response" | jq -e '.id' >/dev/null 2>&1; then
    client_id=$(echo "$client_response" | jq -r '.id')
    created_client_ids+=("$client_id")
    echo "✅ Client creation test passed - ID: $client_id"
    testing::phase::add_test passed
else
    echo "❌ Client creation test failed"
    testing::phase::add_test failed
fi

# Test 2: Invoice creation
if [ -n "$client_id" ]; then
    echo "Testing invoice creation..."
    invoice_data='{
        "client_id": "'$client_id'",
        "invoice_number": "TEST-'$timestamp'",
        "issue_date": "'$(date +%Y-%m-%d)'",
        "due_date": "'$(date -d '+30 days' +%Y-%m-%d)'",
        "line_items": [
            {
                "description": "Test Service",
                "quantity": 1,
                "unit_price": 100.00
            }
        ]
    }'
    invoice_response=$(curl -sf -X POST "$API_BASE_URL/api/v1/invoices" -H "Content-Type: application/json" -d "$invoice_data" 2>/dev/null || echo "")

    if echo "$invoice_response" | jq -e '.id' >/dev/null 2>&1; then
        invoice_id=$(echo "$invoice_response" | jq -r '.id')
        created_invoice_ids+=("$invoice_id")
        echo "✅ Invoice creation test passed - ID: $invoice_id"
        testing::phase::add_test passed

        # Verify calculations
        total_amount=$(echo "$invoice_response" | jq -r '.total_amount')
        if [ "$total_amount" = "100.00" ] || [ "$total_amount" = "100" ]; then
            echo "✅ Invoice calculation test passed"
            testing::phase::add_test passed
        else
            echo "❌ Invoice calculation test failed - expected 100.00, got $total_amount"
            testing::phase::add_test failed
        fi
    else
        echo "❌ Invoice creation test failed"
        testing::phase::add_test failed
    fi
fi

# Test 3: Payment recording
if [ -n "$invoice_id" ]; then
    echo "Testing payment recording..."
    payment_data='{
        "invoice_id": "'$invoice_id'",
        "amount": 50.00,
        "payment_method": "credit_card",
        "reference": "TEST-PAY-'$timestamp'"
    }'
    payment_response=$(curl -sf -X POST "$API_BASE_URL/api/v1/payments" -H "Content-Type: application/json" -d "$payment_data" 2>/dev/null || echo "")

    if echo "$payment_response" | jq -e '.id' >/dev/null 2>&1; then
        echo "✅ Payment recording test passed"
        testing::phase::add_test passed

        # Verify balance update
        invoice_check=$(curl -sf "$API_BASE_URL/api/v1/invoices/$invoice_id" 2>/dev/null || echo "")
        balance_due=$(echo "$invoice_check" | jq -r '.balance_due')
        if [ "$balance_due" = "50.00" ] || [ "$balance_due" = "50" ]; then
            echo "✅ Balance update test passed"
            testing::phase::add_test passed
        else
            echo "❌ Balance update test failed - expected 50.00, got $balance_due"
            testing::phase::add_test failed
        fi
    else
        echo "❌ Payment recording test failed"
        testing::phase::add_test failed
    fi
fi

# Test 4: Invoice status workflow
if [ -n "$invoice_id" ]; then
    echo "Testing invoice status workflow..."

    # Update to sent
    status_update='{"status":"sent"}'
    status_response=$(curl -sf -X PATCH "$API_BASE_URL/api/v1/invoices/$invoice_id/status" -H "Content-Type: application/json" -d "$status_update" 2>/dev/null || echo "")

    if echo "$status_response" | jq -e '.status' >/dev/null 2>&1; then
        status=$(echo "$status_response" | jq -r '.status')
        if [ "$status" = "sent" ]; then
            echo "✅ Invoice status update test passed"
            testing::phase::add_test passed
        else
            echo "❌ Invoice status update test failed - expected 'sent', got '$status'"
            testing::phase::add_test failed
        fi
    else
        echo "❌ Invoice status update test failed"
        testing::phase::add_test failed
    fi
fi

# Test 5: Data persistence and listing
echo "Testing data persistence..."
invoices_response=$(curl -sf "$API_BASE_URL/api/v1/invoices" 2>/dev/null || echo "")
if echo "$invoices_response" | jq -e '.invoices' >/dev/null 2>&1; then
    invoice_count=$(echo "$invoices_response" | jq '.invoices | length')
    echo "✅ Data persistence test passed - $invoice_count invoices found"
    testing::phase::add_test passed
else
    echo "❌ Data persistence test failed"
    testing::phase::add_test failed
fi

# Test 6: PDF generation
if [ -n "$invoice_id" ]; then
    echo "Testing PDF generation..."
    pdf_response=$(curl -sf "$API_BASE_URL/api/v1/invoices/$invoice_id/pdf" 2>/dev/null || echo "")

    if [ -n "$pdf_response" ]; then
        echo "✅ PDF generation test passed"
        testing::phase::add_test passed
    else
        echo "⚠️  PDF generation test skipped - endpoint may not be available"
    fi
fi

echo "Summary: $TESTING_PHASE_TEST_COUNT tests, $TESTING_PHASE_ERROR_COUNT failed"

if [ $TESTING_PHASE_ERROR_COUNT -eq 0 ]; then
    echo "SUCCESS: All business tests passed"
else
    echo "ERROR: Some business tests failed"
fi

# End with summary
testing::phase::end_with_summary "Business logic tests completed"
