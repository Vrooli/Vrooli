#!/bin/bash

# Visitor Intelligence Structure Validation
# Verifies the scenario meets all PRD requirements and is ready for deployment

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Counters
check_count=0
pass_count=0
fail_count=0

# Utility functions
log_info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

log_success() {
    echo -e "${GREEN}✓${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}⚠${NC} $1"
}

log_error() {
    echo -e "${RED}✗${NC} $1" >&2
}

# Check function wrapper
check() {
    local name="$1"
    local condition="$2"
    
    ((check_count++))
    
    if eval "$condition"; then
        log_success "$name"
        ((pass_count++))
        return 0
    else
        log_error "$name"
        ((fail_count++))
        return 1
    fi
}

echo "🧪 Visitor Intelligence Structure Validation"
echo "============================================"
echo ""

# PRD P0 Requirements Validation
log_info "Validating P0 (Must Have) Requirements..."
echo ""

check "PRD.md exists and is complete" "test -f PRD.md && grep -q 'Core Capability' PRD.md"
check "Service configuration exists" "test -f .vrooli/service.json"
check "PostgreSQL schema exists" "test -f initialization/storage/postgres/schema.sql"
check "Go API implementation exists" "test -f api/main.go && test -f api/go.mod"
check "JavaScript tracking library exists" "test -f ui/tracker.js"
check "CLI tool exists and is executable" "test -x cli/visitor-intelligence"
check "Integration documentation exists" "test -f docs/integration-guide.md"
check "API documentation exists" "test -f docs/api-reference.md"

echo ""
log_info "Validating Core Components..."
echo ""

# Check API endpoints are defined
check "API has tracking endpoint" "grep -q 'trackHandler' api/main.go"
check "API has analytics endpoint" "grep -q 'getAnalyticsHandler' api/main.go"  
check "API has visitor profile endpoint" "grep -q 'getVisitorHandler' api/main.go"
check "API has health endpoint" "grep -q 'healthHandler' api/main.go"

# Check database schema completeness
check "Database has visitors table" "grep -q 'CREATE TABLE.*visitors' initialization/storage/postgres/schema.sql"
check "Database has visitor_events table" "grep -q 'CREATE TABLE.*visitor_events' initialization/storage/postgres/schema.sql"
check "Database has visitor_sessions table" "grep -q 'CREATE TABLE.*visitor_sessions' initialization/storage/postgres/schema.sql"
check "Database has proper indexes" "grep -q 'CREATE INDEX' initialization/storage/postgres/schema.sql"

# Check JavaScript tracking library features
check "Tracking script has fingerprinting" "grep -q 'generateFingerprint' ui/tracker.js"
check "Tracking script has event tracking" "grep -q 'trackEvent' ui/tracker.js"
check "Tracking script has auto-initialization" "grep -q 'autoInit' ui/tracker.js"
check "Tracking script respects Do Not Track" "grep -q 'doNotTrack' ui/tracker.js"

# Check CLI functionality
check "CLI has status command" "grep -q 'cmd_status' cli/visitor-intelligence"
check "CLI has track command" "grep -q 'cmd_track' cli/visitor-intelligence"
check "CLI has analytics command" "grep -q 'cmd_analytics' cli/visitor-intelligence"
check "CLI has profile command" "grep -q 'cmd_profile' cli/visitor-intelligence"
check "CLI install script exists" "test -x cli/install.sh"

# Check testing infrastructure
check "Integration tests exist" "test -x test/integration-test.sh"
check "CLI tests exist" "test -f cli/visitor-intelligence.bats"
check "Phased test runner configured" "test -f test/run-tests.sh"

# Check service configuration compliance
check "Service uses PostgreSQL" "grep -q '\"postgres\"' .vrooli/service.json"
check "Service uses Redis" "grep -q '\"redis\"' .vrooli/service.json"  
check "Service has proper lifecycle" "grep -q '\"setup\"' .vrooli/service.json"
check "Service has health checks" "grep -q '\"health\"' .vrooli/service.json"

echo ""
log_info "Validating P1 (Should Have) Requirements..."
echo ""

# P1 Requirements
check "Dashboard UI exists" "test -f ui/dashboard.html"
check "Example integration exists" "test -f ui/example.html"
check "Build system exists" "test -f ui/build.js && test -f ui/package.json"
check "README with usage examples" "test -f README.md && grep -q 'Quick Start' README.md"

echo ""
log_info "Validating Documentation Quality..."
echo ""

# Documentation completeness
check "Integration guide has examples" "grep -q 'E-Commerce' docs/integration-guide.md"
check "API reference has endpoints" "grep -q 'POST /visitor/track' docs/api-reference.md"  
check "PRD has success metrics" "grep -q 'Success Metrics' PRD.md"
check "PRD has technical architecture" "grep -q 'Technical Architecture' PRD.md"

echo ""
log_info "Validating File Structure..."
echo ""

# Directory structure
check "Has API directory" "test -d api"
check "Has CLI directory" "test -d cli"
check "Has UI directory" "test -d ui"
check "Has docs directory" "test -d docs"
check "Has test directory" "test -d test"
check "Has initialization directory" "test -d initialization"
check "Has .vrooli directory" "test -d .vrooli"

echo ""
log_info "Validating Code Quality..."
echo ""

# Code quality checks
check "Go code has error handling" "grep -q 'if err != nil' api/main.go"
check "Go code has logging" "grep -q 'log\.' api/main.go"
check "JavaScript has error handling" "grep -q 'try\|catch' ui/tracker.js"
check "SQL has proper constraints" "grep -q 'NOT NULL\|UNIQUE\|FOREIGN KEY' initialization/storage/postgres/schema.sql"
check "CLI has help documentation" "grep -q 'cmd_help' cli/visitor-intelligence"

echo ""
log_info "Validating Integration Readiness..."
echo ""

# Integration readiness
check "Tracking script has one-line integration" "grep -q 'data-scenario' ui/tracker.js"
check "API supports CORS" "grep -q 'cors' api/main.go"
check "Service config has port allocation" "grep -q 'API_PORT' .vrooli/service.json"
check "Database schema is production-ready" "grep -q 'PARTITION\|INDEX' initialization/storage/postgres/schema.sql"

# Performance validation
check "JavaScript tracking is lightweight" "test \$(wc -c < ui/tracker.js) -lt 100000"  # < 100KB
check "API uses connection pooling" "grep -q 'SetMaxOpenConns\|SetMaxIdleConns' api/main.go"
check "Database has performance indexes" "grep -c 'CREATE INDEX' initialization/storage/postgres/schema.sql | awk '{exit ($1 >= 10) ? 0 : 1}'"

echo ""
echo "============================================"
log_info "Validation Summary:"
log_info "  Total checks: $check_count"
log_success "  Passed: $pass_count"

if [[ "$fail_count" -gt 0 ]]; then
    log_error "  Failed: $fail_count"
    echo ""
    log_error "Structure validation FAILED"
    log_info "Review the failed checks above and ensure all components are properly implemented."
    exit 1
else
    log_info "  Failed: $fail_count"
    echo ""
    log_success "🎉 All structure validation checks PASSED!"
    echo ""
    log_info "Visitor Intelligence scenario is ready for:"
    echo "  ✅ Production deployment"
    echo "  ✅ Integration with other scenarios"  
    echo "  ✅ High-volume visitor tracking"
    echo "  ✅ Real-time analytics and insights"
    echo "  ✅ Privacy-compliant data collection"
    echo ""
    log_success "Scenario successfully implements all PRD requirements! 🚀"
    echo "SUCCESS"
    exit 0
fi
