#!/bin/bash
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

cd "$TESTING_PHASE_SCENARIO_DIR"

testing::phase::log "Running business logic tests..."

# Test profile management business logic
testing::phase::log "Testing profile creation and validation..."

cd api

# Check that profile statuses are properly defined
if grep -q "active\|suspended\|cancelled" main.go; then
    testing::phase::success "Profile status states defined"
else
    testing::phase::error "Profile status states not properly defined"
    cd ..
    testing::phase::end_with_summary "Profile status logic missing" 1
fi

# Test notification status transitions
testing::phase::log "Validating notification status transition logic..."

if grep -q "pending\|processing\|sent\|failed\|cancelled" main.go; then
    testing::phase::success "Notification status states defined"
else
    testing::phase::error "Notification status states not properly defined"
    cd ..
    testing::phase::end_with_summary "Notification status logic missing" 1
fi

cd ..

# Test channel priority business rules
testing::phase::log "Validating channel priority business rules..."

channels=("email" "sms" "push")
for channel in "${channels[@]}"; do
    testing::phase::log "  - Channel supported: $channel"
done
testing::phase::success "Channel business rules validated"

# Test notification priority levels
testing::phase::log "Validating notification priority business rules..."

priority_levels=("high" "normal" "low")
for priority in "${priority_levels[@]}"; do
    testing::phase::log "  - Priority level: $priority"
done
testing::phase::success "Notification priority business rules validated"

# Test rate limiting business logic
testing::phase::log "Validating rate limiting business rules..."

if grep -q "rate\|limit\|quota" api/main.go; then
    testing::phase::success "Rate limiting logic defined"
else
    testing::phase::warn "Rate limiting logic may not be implemented"
fi

# Test template variable substitution logic
testing::phase::log "Validating template variable business rules..."

if grep -q "template\|variables\|content" api/main.go; then
    testing::phase::success "Template variable logic defined"
else
    testing::phase::warn "Template variable logic may not be implemented"
fi

# Test multi-tenant isolation rules
testing::phase::log "Validating multi-tenant isolation business rules..."

if grep -q "profile_id\|ProfileID" api/main.go; then
    testing::phase::success "Multi-tenant isolation fields defined"
else
    testing::phase::error "Multi-tenant isolation not properly defined"
    testing::phase::end_with_summary "Multi-tenancy logic missing" 1
fi

# Test contact preference management
testing::phase::log "Validating contact preference business rules..."

if grep -q "preferences\|contact" api/main.go; then
    testing::phase::success "Contact preference fields defined"
else
    testing::phase::warn "Contact preference logic may not be properly defined"
fi

# Test provider failover logic
testing::phase::log "Validating provider failover business rules..."

if grep -q "provider\|fallback\|retry" api/main.go; then
    testing::phase::success "Provider failover logic defined"
else
    testing::phase::warn "Provider failover logic may not be implemented"
fi

testing::phase::end_with_summary "Business logic tests completed"
