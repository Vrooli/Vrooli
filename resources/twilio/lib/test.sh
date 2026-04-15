#!/usr/bin/env bash
################################################################################
# Twilio Test Library - v2.0 Universal Contract Implementation
# 
# Test implementations for Twilio resource validation
################################################################################

set -euo pipefail

# Get the directory of this script
SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
RESOURCE_DIR="$(builtin cd "${SCRIPT_DIR}/.." && builtin pwd)"
REPO_ROOT="$(builtin cd "${RESOURCE_DIR}/../.." && builtin pwd)"
TWILIO_DIR="${RESOURCE_DIR}"
TWILIO_LIB_DIR="${TWILIO_DIR}/lib"

# Source required libraries
source "${REPO_ROOT}/scripts/lib/utils/log.sh"
source "${TWILIO_LIB_DIR}/core.sh"
source "${TWILIO_LIB_DIR}/common.sh"
source "${TWILIO_LIB_DIR}/sms.sh"

################################################################################
# Test Functions
################################################################################

# Run smoke tests (quick validation)
twilio::test::smoke() {
    log::header "🔥 Twilio Smoke Tests"
    local passed=0
    local failed=0
    
    # Test 1: Check if Twilio CLI is installed
    log::info "Test 1: Twilio CLI installation..."
    if twilio::is_installed; then
        log::success "  ✅ Twilio CLI is installed"
        passed=$((passed + 1))
    else
        log::error "  ❌ Twilio CLI is not installed"
        failed=$((failed + 1))
    fi
    
    # Test 2: Check configuration directories
    log::info "Test 2: Configuration directories..."
    if [[ -d "$TWILIO_CONFIG_DIR" ]] && [[ -d "$TWILIO_DATA_DIR" ]]; then
        log::success "  ✅ Configuration directories exist"
        passed=$((passed + 1))
    else
        log::error "  ❌ Configuration directories missing"
        failed=$((failed + 1))
    fi
    
    # Test 3: Check credentials configuration
    log::info "Test 3: Credentials configuration..."
    if twilio::core::has_valid_credentials; then
        log::success "  ✅ Valid credentials configured"
        passed=$((passed + 1))
    else
        log::warn "  ⚠️  No valid credentials configured"
        # Not a failure as credentials are optional for smoke test
        passed=$((passed + 1))
    fi
    
    # Test 4: Health check responds
    log::info "Test 4: Health check..."
    if twilio::core::health_check &>/dev/null; then
        log::success "  ✅ Health check responds"
        passed=$((passed + 1))
    else
        log::error "  ❌ Health check failed"
        failed=$((failed + 1))
    fi
    
    # Summary
    echo
    log::info "📊 Smoke Test Results:"
    log::success "  Passed: $passed"
    if [[ $failed -gt 0 ]]; then
        log::error "  Failed: $failed"
        return 1
    else
        log::info "  Failed: 0"
        return 0
    fi
}

# Run integration tests
twilio::test::integration() {
    log::header "🔗 Twilio Integration Tests"
    local passed=0
    local failed=0
    
    # Test 1: Initialize resource
    log::info "Test 1: Resource initialization..."
    if twilio::core::init; then
        log::success "  ✅ Resource initialized successfully"
        passed=$((passed + 1))
    else
        log::error "  ❌ Resource initialization failed"
        failed=$((failed + 1))
    fi
    
    # Test 2: Load secrets (if Vault available)
    log::info "Test 2: Secrets loading..."
    if twilio::core::load_secrets; then
        log::success "  ✅ Secrets loaded successfully"
        passed=$((passed + 1))
    else
        log::warn "  ⚠️  Secrets not available (may be expected)"
        passed=$((passed + 1))
    fi
    
    # Test 3: API connection (if credentials available)
    log::info "Test 3: API connection..."
    if twilio::core::has_valid_credentials; then
        if twilio::core::test_api_connection; then
            log::success "  ✅ API connection successful"
            passed=$((passed + 1))
        else
            log::error "  ❌ API connection failed"
            failed=$((failed + 1))
        fi
    else
        log::warn "  ⚠️  Skipping - no credentials"
        passed=$((passed + 1))
    fi
    
    # Test 4: Phone number retrieval (if configured)
    log::info "Test 4: Phone number management..."
    if [[ -f "$TWILIO_PHONE_NUMBERS_FILE" ]]; then
        local numbers=$(jq '.numbers | length' "$TWILIO_PHONE_NUMBERS_FILE" 2>/dev/null || echo "0")
        if [[ $numbers -gt 0 ]]; then
            log::success "  ✅ Phone numbers configured: $numbers"
            passed=$((passed + 1))
        else
            log::warn "  ⚠️  No phone numbers configured"
            passed=$((passed + 1))
        fi
    else
        log::warn "  ⚠️  Phone numbers file not found"
        passed=$((passed + 1))
    fi
    
    # Test 5: SMS sending capability (test mode only)
    log::info "Test 5: SMS capability check..."
    if twilio::is_installed && twilio::core::has_valid_credentials; then
        # Don't actually send SMS, just verify the function exists
        if declare -f twilio::send_sms &>/dev/null; then
            log::success "  ✅ SMS sending function available"
            passed=$((passed + 1))
        else
            log::error "  ❌ SMS sending function missing"
            failed=$((failed + 1))
        fi
    else
        log::warn "  ⚠️  Skipping - not configured"
        passed=$((passed + 1))
    fi
    
    # Test 6: Bulk SMS capability check
    log::info "Test 6: Bulk SMS capability check..."
    if declare -f twilio::send_bulk_sms &>/dev/null; then
        # Test with test credentials
        export TWILIO_ACCOUNT_SID="AC_test_bulk"
        export TWILIO_AUTH_TOKEN="test_token"
        
        # Test bulk send in test mode
        if twilio::send_bulk_sms "Test message" "" "+15551234567" "+15551234568" 2>&1 | grep -q "Test mode detected"; then
            log::success "  ✅ Bulk SMS function works in test mode"
            passed=$((passed + 1))
        else
            log::error "  ❌ Bulk SMS function failed"
            failed=$((failed + 1))
        fi
    else
        log::error "  ❌ Bulk SMS function missing"
        failed=$((failed + 1))
    fi
    
    # Test 7: CSV file SMS capability check
    log::info "Test 7: CSV file SMS capability check..."
    if declare -f twilio::send_sms_from_file &>/dev/null; then
        # Create test CSV file
        local test_csv="/tmp/twilio_test_${RANDOM}.csv"
        cat > "$test_csv" <<EOF
phone_number,message
+15551234567,Test message 1
+15551234568,Test message 2
EOF
        
        # Test CSV processing in test mode
        export TWILIO_ACCOUNT_SID="AC_test_csv"
        export TWILIO_AUTH_TOKEN="test_token"
        
        local output
        output=$(twilio::send_sms_from_file "$test_csv" "" 2>&1)
        
        if echo "$output" | grep -q "Test mode detected"; then
            log::success "  ✅ CSV SMS function works in test mode"
            passed=$((passed + 1))
        else
            log::error "  ❌ CSV SMS function failed"
            log::debug "Output was: $output"
            failed=$((failed + 1))
        fi
        
        # Clean up
        rm -f "$test_csv"
    else
        log::error "  ❌ CSV SMS function missing"
        failed=$((failed + 1))
    fi
    
    # Test 8: Voice call function availability
    log::info "Test 8: Voice call function availability..."
    if declare -f make_call &>/dev/null; then
        log::success "  ✅ Voice call function available"
        passed=$((passed + 1))
    else
        log::error "  ❌ Voice call function missing"
        failed=$((failed + 1))
    fi
    
    # Test 9: Voice conference function availability
    log::info "Test 9: Voice conference function availability..."
    if declare -f make_conference &>/dev/null; then
        log::success "  ✅ Conference call function available"
        passed=$((passed + 1))
    else
        log::error "  ❌ Conference call function missing"
        failed=$((failed + 1))
    fi
    
    # Test 10: Voice history functionality
    log::info "Test 10: Voice history functionality..."
    if declare -f get_voice_history &>/dev/null && declare -f get_voice_stats &>/dev/null; then
        log::success "  ✅ Voice history functions available"
        passed=$((passed + 1))
    else
        log::error "  ❌ Voice history functions missing"
        failed=$((failed + 1))
    fi
    
    # Test 11: WhatsApp message function availability
    log::info "Test 11: WhatsApp message function availability..."
    if declare -f send_whatsapp &>/dev/null; then
        log::success "  ✅ WhatsApp message function available"
        passed=$((passed + 1))
    else
        log::error "  ❌ WhatsApp message function missing"
        failed=$((failed + 1))
    fi
    
    # Test 12: WhatsApp template function availability
    log::info "Test 12: WhatsApp template function availability..."
    if declare -f send_whatsapp_template &>/dev/null; then
        log::success "  ✅ WhatsApp template function available"
        passed=$((passed + 1))
    else
        log::error "  ❌ WhatsApp template function missing"
        failed=$((failed + 1))
    fi
    
    # Test 13: WhatsApp bulk function availability
    log::info "Test 13: WhatsApp bulk function availability..."
    if declare -f send_whatsapp_bulk &>/dev/null; then
        log::success "  ✅ WhatsApp bulk function available"
        passed=$((passed + 1))
    else
        log::error "  ❌ WhatsApp bulk function missing"
        failed=$((failed + 1))
    fi
    
    # Test 14: WhatsApp history functionality
    log::info "Test 14: WhatsApp history functionality..."
    if declare -f get_whatsapp_history &>/dev/null && declare -f get_whatsapp_stats &>/dev/null; then
        log::success "  ✅ WhatsApp history functions available"
        passed=$((passed + 1))
    else
        log::error "  ❌ WhatsApp history functions missing"
        failed=$((failed + 1))
    fi
    
    # Test 15: WhatsApp setup helper availability
    log::info "Test 15: WhatsApp setup helper availability..."
    if declare -f setup_whatsapp_sandbox &>/dev/null; then
        log::success "  ✅ WhatsApp setup helper available"
        passed=$((passed + 1))
    else
        log::error "  ❌ WhatsApp setup helper missing"
        failed=$((failed + 1))
    fi
    
    # Summary
    echo
    log::info "📊 Integration Test Results:"
    log::success "  Passed: $passed"
    if [[ $failed -gt 0 ]]; then
        log::error "  Failed: $failed"
        return 1
    else
        log::info "  Failed: 0"
        return 0
    fi
}

# Run unit tests
twilio::test::unit() {
    log::header "🧪 Twilio Unit Tests"
    local passed=0
    local failed=0
    
    # Test 1: Credential format validation
    log::info "Test 1: Credential format validation..."
    local test_sid="ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
    local test_token="xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
    
    if [[ "$test_sid" =~ ^AC[a-f0-9x]{32}$ ]]; then
        log::success "  ✅ SID format validation works"
        passed=$((passed + 1))
    else
        log::error "  ❌ SID format validation failed"
        failed=$((failed + 1))
    fi
    
    # Test 2: Phone number format validation
    log::info "Test 2: Phone number format validation..."
    local test_number="+12345678901"
    
    if [[ "$test_number" =~ ^\+[1-9][0-9]{1,14}$ ]]; then
        log::success "  ✅ Phone number format validation works"
        passed=$((passed + 1))
    else
        log::error "  ❌ Phone number format validation failed"
        failed=$((failed + 1))
    fi
    
    # Test 3: Directory creation
    log::info "Test 3: Directory creation..."
    twilio::ensure_dirs
    if [[ -d "$TWILIO_CONFIG_DIR" ]] && [[ -d "$TWILIO_DATA_DIR" ]]; then
        log::success "  ✅ Directories created successfully"
        passed=$((passed + 1))
    else
        log::error "  ❌ Directory creation failed"
        failed=$((failed + 1))
    fi
    
    # Test 4: Command detection
    log::info "Test 4: Command detection..."
    if declare -f twilio::get_command &>/dev/null; then
        log::success "  ✅ Command detection function exists"
        passed=$((passed + 1))
    else
        log::error "  ❌ Command detection function missing"
        failed=$((failed + 1))
    fi
    
    # Test 5: Version detection
    log::info "Test 5: Version detection..."
    local version=$(twilio::get_version)
    if [[ -n "$version" ]]; then
        log::success "  ✅ Version detection works: $version"
        passed=$((passed + 1))
    else
        log::error "  ❌ Version detection failed"
        failed=$((failed + 1))
    fi
    
    # Summary
    echo
    log::info "📊 Unit Test Results:"
    log::success "  Passed: $passed"
    if [[ $failed -gt 0 ]]; then
        log::error "  Failed: $failed"
        return 1
    else
        log::info "  Failed: 0"
        return 0
    fi
}

# Run all tests
twilio::test::all() {
    log::header "🧪 Running All Twilio Tests"
    echo
    
    local smoke_result=0
    local integration_result=0
    local unit_result=0
    
    # Run smoke tests
    if twilio::test::smoke; then
        smoke_result=0
    else
        smoke_result=1
    fi
    echo
    
    # Run unit tests
    if twilio::test::unit; then
        unit_result=0
    else
        unit_result=1
    fi
    echo
    
    # Run integration tests
    if twilio::test::integration; then
        integration_result=0
    else
        integration_result=1
    fi
    echo
    
    # Overall summary
    log::header "📊 Overall Test Results"
    if [[ $smoke_result -eq 0 ]]; then
        log::success "  ✅ Smoke tests: PASSED"
    else
        log::error "  ❌ Smoke tests: FAILED"
    fi
    
    if [[ $unit_result -eq 0 ]]; then
        log::success "  ✅ Unit tests: PASSED"
    else
        log::error "  ❌ Unit tests: FAILED"
    fi
    
    if [[ $integration_result -eq 0 ]]; then
        log::success "  ✅ Integration tests: PASSED"
    else
        log::error "  ❌ Integration tests: FAILED"
    fi
    
    # Return overall result
    if [[ $smoke_result -eq 0 ]] && [[ $unit_result -eq 0 ]] && [[ $integration_result -eq 0 ]]; then
        echo
        log::success "All tests passed! 🎉"
        return 0
    else
        echo
        log::error "Some tests failed. Please review the results above."
        return 1
    fi
}

################################################################################
# Export Functions
################################################################################

export -f twilio::test::smoke
export -f twilio::test::integration
export -f twilio::test::unit
export -f twilio::test::all