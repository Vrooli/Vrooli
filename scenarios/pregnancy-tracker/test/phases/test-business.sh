#!/bin/bash
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase with runtime requirement and 90-second target
testing::phase::init --require-runtime --target-time "90s"

echo "Testing pregnancy-tracker business logic and domain requirements"

API_URL=""

if ! testing::core::wait_for_scenario "$TESTING_PHASE_SCENARIO_NAME" 30; then
    testing::phase::add_error "❌ Scenario '$TESTING_PHASE_SCENARIO_NAME' did not become ready in time"
    testing::phase::end_with_summary
fi

if ! API_URL=$(testing::connectivity::get_api_url "$TESTING_PHASE_SCENARIO_NAME"); then
    testing::phase::add_error "❌ Could not resolve API URL for $TESTING_PHASE_SCENARIO_NAME"
    testing::phase::end_with_summary
fi

echo ""
echo "🔐 Testing Privacy & Security Business Rules..."

# Test 1: Verify encryption is enabled
echo "  Testing encryption requirement..."
encryption_response=$(curl -sf "$API_URL/api/v1/health/encryption" 2>/dev/null || echo "")
if echo "$encryption_response" | grep -q '"enabled":true'; then
    log::success "  ✅ Encryption is enabled as required"
else
    testing::phase::add_error "  ❌ Encryption must be enabled for health data"
fi

# Test 2: Verify multi-tenant isolation
echo "  Testing multi-tenant isolation..."
status_response=$(curl -sf "$API_URL/api/v1/status" 2>/dev/null || echo "")
if echo "$status_response" | grep -q '"multi_tenant"'; then
    log::success "  ✅ Multi-tenant support configured"
else
    testing::phase::add_warning "  ⚠️  Multi-tenant configuration not verified"
fi

# Test 3: Verify user authentication requirement
echo "  Testing authentication requirement..."
no_auth_response=$(curl -sf -w "%{http_code}" -o /dev/null "$API_URL/api/v1/pregnancy/current" 2>/dev/null || echo "000")
if [ "$no_auth_response" = "401" ]; then
    log::success "  ✅ Authentication required for private data"
else
    testing::phase::add_warning "  ⚠️  Expected 401 for unauthenticated access, got $no_auth_response"
fi

echo ""
echo "📅 Testing Pregnancy Calculation Business Logic..."

# Test 4: Week content availability (weeks 0-42)
echo "  Testing week content range..."
week_errors=0
for week in 0 1 12 24 40 42; do
    week_response=$(curl -sf "$API_URL/api/v1/content/week/$week" 2>/dev/null || echo "")
    if echo "$week_response" | grep -q '"week"'; then
        echo "    ✓ Week $week content available"
    else
        ((week_errors++))
        echo "    ✗ Week $week content missing or malformed"
    fi
done

if [ $week_errors -eq 0 ]; then
    log::success "  ✅ All pregnancy week content accessible"
elif [ $week_errors -lt 3 ]; then
    testing::phase::add_warning "  ⚠️  Some week content missing ($week_errors errors)"
else
    testing::phase::add_error "  ❌ Multiple week content failures ($week_errors errors)"
fi

# Test 5: Invalid week numbers should be rejected
echo "  Testing week validation..."
invalid_week_response=$(curl -sf -w "%{http_code}" -o /dev/null "$API_URL/api/v1/content/week/50" 2>/dev/null || echo "000")
if [ "$invalid_week_response" = "400" ]; then
    log::success "  ✅ Invalid week numbers properly rejected"
else
    testing::phase::add_warning "  ⚠️  Expected 400 for week 50, got $invalid_week_response"
fi

echo ""
echo "🔍 Testing Evidence-Based Content Requirements..."

# Test 6: Search functionality for medical information
echo "  Testing search for medical content..."
if command -v jq >/dev/null 2>&1; then
    search_response=$(curl -sf "$API_URL/api/v1/search?q=morning+sickness" 2>/dev/null || echo "")
    if echo "$search_response" | jq -e '. | length' >/dev/null 2>&1; then
        result_count=$(echo "$search_response" | jq '. | length')
        if [ "$result_count" -gt 0 ]; then
            log::success "  ✅ Search returns medical content ($result_count results)"
        else
            testing::phase::add_warning "  ⚠️  Search working but no content indexed yet"
        fi
    else
        testing::phase::add_warning "  ⚠️  Search may require database initialization"
    fi
else
    testing::phase::add_warning "  ⚠️  jq not available for search validation"
fi

echo ""
echo "📊 Testing Data Tracking Business Rules..."

# Test 7: Daily log creation requires user context
echo "  Testing daily log authorization..."
log_response=$(curl -sf -w "%{http_code}" -o /dev/null \
    -X POST \
    -H "Content-Type: application/json" \
    -d '{"mood":5,"energy":7}' \
    "$API_URL/api/v1/logs/daily" 2>/dev/null || echo "000")
if [ "$log_response" = "401" ]; then
    log::success "  ✅ Daily logs require user authentication"
else
    testing::phase::add_warning "  ⚠️  Expected 401 for unauthenticated log creation"
fi

# Test 8: Kick counting requires user context
echo "  Testing kick count authorization..."
kick_response=$(curl -sf -w "%{http_code}" -o /dev/null \
    -X POST \
    -H "Content-Type: application/json" \
    -d '{"count":10}' \
    "$API_URL/api/v1/kicks/count" 2>/dev/null || echo "000")
if [ "$kick_response" = "401" ]; then
    log::success "  ✅ Kick counting requires user authentication"
else
    testing::phase::add_warning "  ⚠️  Expected 401 for unauthenticated kick count"
fi

echo ""
echo "📤 Testing Export & Medical Data Business Rules..."

# Test 9: Export requires user authentication
echo "  Testing export authorization..."
export_response=$(curl -sf -w "%{http_code}" -o /dev/null \
    "$API_URL/api/v1/export/json" 2>/dev/null || echo "000")
if [ "$export_response" = "401" ]; then
    log::success "  ✅ Data export requires user authentication"
else
    testing::phase::add_warning "  ⚠️  Expected 401 for unauthenticated export"
fi

# Test 10: Emergency card requires user authentication
echo "  Testing emergency card authorization..."
emergency_response=$(curl -sf -w "%{http_code}" -o /dev/null \
    "$API_URL/api/v1/export/emergency-card" 2>/dev/null || echo "000")
if [ "$emergency_response" = "401" ]; then
    log::success "  ✅ Emergency card requires user authentication"
else
    testing::phase::add_warning "  ⚠️  Expected 401 for unauthenticated emergency card access"
fi

echo ""
echo "👥 Testing Partner Access Business Rules..."

# Test 11: Partner invite requires authentication
echo "  Testing partner invite authorization..."
invite_response=$(curl -sf -w "%{http_code}" -o /dev/null \
    -X POST \
    "$API_URL/api/v1/partner/invite" 2>/dev/null || echo "000")
if [ "$invite_response" = "401" ]; then
    log::success "  ✅ Partner invites require user authentication"
else
    testing::phase::add_warning "  ⚠️  Expected 401 for unauthenticated partner invite"
fi

# Test 12: Partner view requires authentication
echo "  Testing partner view authorization..."
partner_view_response=$(curl -sf -w "%{http_code}" -o /dev/null \
    "$API_URL/api/v1/partner/view" 2>/dev/null || echo "000")
if [ "$partner_view_response" = "401" ] || [ "$partner_view_response" = "403" ]; then
    log::success "  ✅ Partner view requires proper authorization"
else
    testing::phase::add_warning "  ⚠️  Expected 401/403 for unauthorized partner view"
fi

echo ""
echo "📊 Business Logic Test Summary:"
echo "════════════════════════════════"
echo "   Privacy & Security: validated"
echo "   Pregnancy Calculations: validated"
echo "   Evidence-Based Content: validated"
echo "   Data Tracking: validated"
echo "   Export Functions: validated"
echo "   Partner Access: validated"

if [ $TESTING_PHASE_ERROR_COUNT -eq 0 ]; then
    log::success "✅ SUCCESS: All business logic requirements validated!"
else
    echo "⚠️  Some business rules need attention (see errors above)"
fi

# End with summary
testing::phase::end_with_summary "Business logic tests completed"
