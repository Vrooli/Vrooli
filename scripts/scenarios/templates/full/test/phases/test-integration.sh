#!/bin/bash
# Integration test phase - <120 seconds
# Tests API endpoints, database operations, and resource integrations
set -euo pipefail

# Setup paths and utilities
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"

echo "=== Integration Tests Phase ==="
start_time=$(date +%s)

error_count=0
test_count=0
scenario_name=$(basename "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)")

# Source connectivity testing module
source "$APP_ROOT/scripts/lib/testing/shell/connectivity.sh"

# Get dynamic URLs using the testing library
API_BASE_URL=$(testing::connectivity::get_api_url "$scenario_name" 2>/dev/null || echo "")
UI_BASE_URL=$(testing::connectivity::get_ui_url "$scenario_name" 2>/dev/null || echo "")

# Test API endpoints (if API exists and is running)
if [ -n "$API_BASE_URL" ] && [ -d "api" ]; then
    echo "🧪 Testing API integration..."
    echo "  Using API base URL: $API_BASE_URL"
    
    # Test health endpoint
    echo "  Testing health endpoint..."
    if curl -s --max-time 10 "$API_BASE_URL/health" >/dev/null 2>&1; then
        log::success "   ✅ Health endpoint responding"
        test_count=$((test_count + 1))
    else
        log::error "   ❌ Health endpoint not responding"
        error_count=$((error_count + 1))
    fi
    
    # Test API root/version endpoint
    echo "  Testing API root endpoint..."
    if curl -s --max-time 10 "$API_BASE_URL/api/v1" >/dev/null 2>&1; then
        log::success "   ✅ API root endpoint responding"
        test_count=$((test_count + 1))
    else
        log::warning "   ⚠️  API root endpoint not available (may be expected)"
    fi
    
    # Test common CRUD operations (generic template)
    # NOTE: Actual scenarios should customize these tests
    echo "  Testing core API functionality..."
    
    # Example: Test listing endpoint (customize per scenario)
    # if curl -s --max-time 10 "$API_BASE_URL/api/v1/items" | jq -e '.items' >/dev/null 2>&1; then
    #     log::success "   ✅ List endpoint responding with expected format"
    #     test_count=$((test_count + 1))
    # else
    #     log::warning "   ⚠️  List endpoint test needs customization"
    # fi
    
    log::info "   💡 Customize API tests in test/phases/test-integration.sh"
else
    log::info "ℹ️  API not available or not running, skipping API tests"
fi

# Test UI accessibility (if UI exists and is running)
if [ -n "$UI_BASE_URL" ] && [ -d "ui" ]; then
    echo "🧪 Testing UI integration..."
    echo "  Using UI base URL: $UI_BASE_URL"
    
    # Test UI loads
    echo "  Testing UI accessibility..."
    if curl -s --max-time 10 "$UI_BASE_URL" >/dev/null 2>&1; then
        log::success "   ✅ UI accessible"
        test_count=$((test_count + 1))
    else
        log::error "   ❌ UI not accessible"
        error_count=$((error_count + 1))
    fi
    
    # Test for basic HTML structure
    echo "  Testing UI content..."
    ui_content=$(curl -s --max-time 10 "$UI_BASE_URL" 2>/dev/null || echo "")
    if echo "$ui_content" | grep -q "<html\|<title\|<body" 2>/dev/null; then
        log::success "   ✅ UI returns valid HTML"
        test_count=$((test_count + 1))
    else
        log::warning "   ⚠️  UI content format unexpected"
    fi
else
    log::info "ℹ️  UI not available or not running, skipping UI tests"
fi

# Test CLI integration (if CLI exists)
if [ -d "cli" ]; then
    echo "🧪 Testing CLI integration..."
    cli_binary="cli/$scenario_name"
    
    if [ -f "$cli_binary" ] && [ -x "$cli_binary" ]; then
        # Test CLI help functionality
        echo "  Testing CLI help..."
        if "$cli_binary" --help >/dev/null 2>&1 || "$cli_binary" help >/dev/null 2>&1; then
            log::success "   ✅ CLI help command works"
            test_count=$((test_count + 1))
        else
            log::error "   ❌ CLI help command failed"
            error_count=$((error_count + 1))
        fi
        
        # Test CLI version (if supported)
        echo "  Testing CLI version..."
        if "$cli_binary" --version >/dev/null 2>&1 || "$cli_binary" version >/dev/null 2>&1; then
            log::success "   ✅ CLI version command works"
            test_count=$((test_count + 1))
        else
            log::info "   ℹ️  CLI version command not available"
        fi
        
        # Run BATS tests if they exist
        if [ -f "cli/${scenario_name}.bats" ] && command -v bats >/dev/null 2>&1; then
            echo "  Running CLI BATS tests..."
            if bats "cli/${scenario_name}.bats" >/dev/null 2>&1; then
                log::success "   ✅ CLI BATS tests passed"
                test_count=$((test_count + 1))
            else
                log::error "   ❌ CLI BATS tests failed"
                error_count=$((error_count + 1))
            fi
        else
            log::info "   ℹ️  No BATS tests found or BATS not available"
        fi
    else
        log::info "   ℹ️  CLI binary not found or not executable"
    fi
else
    log::info "ℹ️  No CLI directory found, skipping CLI tests"
fi

# Test database integration (if database is used)
echo "🧪 Testing database integration..."
if command -v jq >/dev/null 2>&1 && [ -f ".vrooli/service.json" ]; then
    # Check if scenario uses PostgreSQL
    uses_postgres=$(jq -r '.resources | has("postgres") and (.postgres.required == true or .postgres.enabled == true)' .vrooli/service.json 2>/dev/null || echo "false")
    
    if [ "$uses_postgres" = "true" ]; then
        echo "  Testing PostgreSQL connection..."
        
        # Try to test database connection through the API if possible
        if [ -n "$API_PORT" ]; then
            # Generic database health check through API
            # NOTE: Actual scenarios should customize this
            log::info "   💡 Add database-specific tests via API"
        else
            log::info "   ℹ️  API not available for database testing"
        fi
    else
        log::info "   ℹ️  PostgreSQL not configured, skipping database tests"
    fi
else
    log::info "   ℹ️  Cannot parse service configuration, skipping database tests"
fi

# Test resource integrations
echo "🧪 Testing resource integrations..."
if command -v jq >/dev/null 2>&1 && [ -f ".vrooli/service.json" ]; then
    # Get enabled resources
    enabled_resources=$(jq -r '.resources | to_entries[] | select(.value.required == true or .value.enabled == true) | .key' .vrooli/service.json 2>/dev/null || echo "")
    
    if [ -n "$enabled_resources" ]; then
        while IFS= read -r resource; do
            [ -z "$resource" ] && continue
            
            echo "  Testing $resource integration..."
            # NOTE: Actual scenarios should customize resource-specific tests
            log::info "   💡 Add $resource-specific integration tests"
        done <<< "$enabled_resources"
    else
        log::info "   ℹ️  No resources configured for testing"
    fi
fi

# Performance check
end_time=$(date +%s)
duration=$((end_time - start_time))
echo ""

if [ $error_count -eq 0 ]; then
    log::success "✅ Integration tests completed successfully in ${duration}s"
    log::success "   Tests run: $test_count, Errors: $error_count"
else
    log::error "❌ Integration tests failed with $error_count errors in ${duration}s"
    log::error "   Tests run: $test_count, Errors: $error_count"
    echo ""
    log::info "💡 Troubleshooting tips:"
    echo "   • Ensure scenario is running: vrooli scenario start $scenario_name"
    echo "   • Check service logs: vrooli scenario logs $scenario_name"
    echo "   • Verify resource health: vrooli resource status"
    echo "   • Customize integration tests for your scenario's specific endpoints"
fi

if [ $duration -gt 120 ]; then
    log::warning "⚠️  Integration phase exceeded 120s target"
fi

# Exit with appropriate code
if [ $error_count -eq 0 ]; then
    exit 0
else
    exit 1
fi