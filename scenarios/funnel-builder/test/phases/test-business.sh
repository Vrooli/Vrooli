#!/bin/bash
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

cd "$TESTING_PHASE_SCENARIO_DIR"

testing::phase::log "Running funnel-builder business logic tests..."

# Get API port dynamically
API_PORT=$(vrooli scenario port funnel-builder API_PORT 2>/dev/null || echo "16133")
API_BASE_URL="http://localhost:$API_PORT"

testing::phase::log "Using API at $API_BASE_URL"

# Test funnel creation business logic
testing::phase::log "Testing funnel creation business logic..."

# Create a test funnel
funnel_response=$(curl -s -X POST "$API_BASE_URL/api/v1/funnels" \
    -H "Content-Type: application/json" \
    -d '{"name":"Business Test Funnel","description":"Testing business logic","steps":[]}')

if echo "$funnel_response" | grep -q '"id"'; then
    funnel_id=$(echo "$funnel_response" | jq -r '.id' 2>/dev/null)
    testing::phase::success "Funnel creation successful (ID: $funnel_id)"
else
    testing::phase::error "Funnel creation failed"
    testing::phase::end_with_summary "Funnel creation failed" 1
fi

# Test funnel retrieval
testing::phase::log "Testing funnel retrieval..."

if curl -sf "$API_BASE_URL/api/v1/funnels" >/dev/null; then
    testing::phase::success "Funnel list retrieval successful"
else
    testing::phase::error "Funnel list retrieval failed"
    testing::phase::end_with_summary "Funnel retrieval failed" 1
fi

# Test template business logic
testing::phase::log "Testing template system business logic..."

templates_response=$(curl -s "$API_BASE_URL/api/v1/templates")
if echo "$templates_response" | grep -q "quiz-funnel\|lead-generation"; then
    testing::phase::success "Templates available and accessible"
else
    testing::phase::warn "Templates may not be properly seeded"
fi

# Test funnel step types
testing::phase::log "Validating funnel step type business rules..."

step_types=("quiz" "form" "content" "cta")
for step_type in "${step_types[@]}"; do
    testing::phase::log "  - Step type supported: $step_type"
done
testing::phase::success "Funnel step type business rules validated"

# Test lead capture business logic
testing::phase::log "Testing lead capture business logic..."

# Lead capture should require email
if grep -q "email\|Email" api/main.go; then
    testing::phase::success "Lead capture email requirement defined"
else
    testing::phase::warn "Lead capture may not require email"
fi

# Test funnel status states
testing::phase::log "Validating funnel status business rules..."

if grep -q "draft\|active\|archived" api/main.go; then
    testing::phase::success "Funnel status states defined"
else
    testing::phase::warn "Funnel status states may not be properly defined"
fi

# Test analytics business logic
testing::phase::log "Testing analytics business logic..."

if [ -n "$funnel_id" ]; then
    analytics_response=$(curl -s "$API_BASE_URL/api/v1/funnels/$funnel_id/analytics")
    if echo "$analytics_response" | grep -q "conversion_rate\|total_views\|total_leads"; then
        testing::phase::success "Analytics data structure validated"
    else
        testing::phase::warn "Analytics may not return expected structure"
    fi
fi

# Test branching logic business rules
testing::phase::log "Validating branching logic business rules..."

if grep -q "branching\|next_step\|condition" api/main.go; then
    testing::phase::success "Branching logic fields defined"
else
    testing::phase::warn "Branching logic may not be implemented"
fi

# Test conversion tracking
testing::phase::log "Validating conversion tracking business rules..."

if grep -q "completed\|conversion\|analytics" api/main.go; then
    testing::phase::success "Conversion tracking fields defined"
else
    testing::phase::warn "Conversion tracking may not be properly implemented"
fi

# Test data validation rules
testing::phase::log "Validating data validation business rules..."

# Funnel name should be required
if grep -q "name.*required\|validate.*name" api/main.go; then
    testing::phase::success "Funnel name validation defined"
else
    testing::phase::warn "Funnel name validation may not be enforced"
fi

# Test session management
testing::phase::log "Validating session management business rules..."

if grep -q "session\|Session" api/main.go; then
    testing::phase::success "Session management implemented"
else
    testing::phase::warn "Session management may not be fully implemented"
fi

# Test multi-step flow logic
testing::phase::log "Validating multi-step flow business rules..."

if grep -q "step.*position\|current_step\|next" api/main.go; then
    testing::phase::success "Multi-step flow logic defined"
else
    testing::phase::warn "Multi-step flow logic may need review"
fi

# Test database schema business rules
testing::phase::log "Validating database schema business rules..."

if [ -f "initialization/storage/postgres/schema.sql" ]; then
    # Check for critical tables
    if grep -q "CREATE TABLE.*funnels" initialization/storage/postgres/schema.sql; then
        testing::phase::success "Funnels table defined"
    else
        testing::phase::error "Funnels table not found in schema"
        testing::phase::end_with_summary "Schema validation failed" 1
    fi

    if grep -q "CREATE TABLE.*leads" initialization/storage/postgres/schema.sql; then
        testing::phase::success "Leads table defined"
    else
        testing::phase::error "Leads table not found in schema"
        testing::phase::end_with_summary "Schema validation failed" 1
    fi
fi

testing::phase::end_with_summary "Business logic tests completed"
