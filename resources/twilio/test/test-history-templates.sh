#!/usr/bin/env bash
set -euo pipefail

# Test script for Twilio message history and templates
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"

# Test message history
test_message_history() {
    log::info "Testing message history..."
    
    # Check history stats
    if vrooli resource twilio content history-stats &>/dev/null; then
        log::success "  ✅ History stats working"
    else
        log::error "  ❌ History stats failed"
        return 1
    fi
    
    # Export history (even if empty)
    local export_file="/tmp/twilio_test_export.csv"
    if vrooli resource twilio content history-export "$export_file" &>/dev/null; then
        log::success "  ✅ History export working"
        rm -f "$export_file"
    else
        log::error "  ❌ History export failed"
        return 1
    fi
    
    # List history
    if vrooli resource twilio content history &>/dev/null; then
        log::success "  ✅ History list working"
    else
        log::error "  ❌ History list failed"
        return 1
    fi
    
    return 0
}

# Test message templates
test_message_templates() {
    log::info "Testing message templates..."
    
    # Create a test template
    local template_name="test_template_$(date +%s)"
    if vrooli resource twilio content template-create \
        "$template_name" \
        "Test message for {{name}} with code {{code}}" \
        "Test template" &>/dev/null; then
        log::success "  ✅ Template creation working"
    else
        log::error "  ❌ Template creation failed"
        return 1
    fi
    
    # List templates
    if vrooli resource twilio content template-list &>/dev/null; then
        log::success "  ✅ Template listing working"
    else
        log::error "  ❌ Template listing failed"
        return 1
    fi
    
    # Get specific template
    if vrooli resource twilio content template-get "$template_name" &>/dev/null; then
        log::success "  ✅ Template retrieval working"
    else
        log::error "  ❌ Template retrieval failed"
        return 1
    fi
    
    # Update template
    if vrooli resource twilio content template-update \
        "$template_name" \
        "Updated message for {{name}}" \
        "Updated description" &>/dev/null; then
        log::success "  ✅ Template update working"
    else
        log::error "  ❌ Template update failed"
        return 1
    fi
    
    # Delete template
    if vrooli resource twilio content template-delete "$template_name" &>/dev/null; then
        log::success "  ✅ Template deletion working"
    else
        log::error "  ❌ Template deletion failed"
        return 1
    fi
    
    return 0
}

# Test template sending in test mode
test_template_sending() {
    log::info "Testing template sending (test mode)..."
    
    # Set test credentials
    export TWILIO_ACCOUNT_SID="AC_test_templates"
    export TWILIO_AUTH_TOKEN="test_token"
    
    # Create a template for testing
    local template_name="send_test_$(date +%s)"
    vrooli resource twilio content template-create \
        "$template_name" \
        "Hello {{name}}, your order {{order_id}} is ready!" \
        "Order notification" &>/dev/null
    
    # Test sending with template (should work in test mode)
    if vrooli resource twilio content template-send \
        "$template_name" \
        "+1234567890" \
        "+0987654321" \
        "name=John" \
        "order_id=12345" 2>&1 | grep -q "Test mode detected"; then
        log::success "  ✅ Template sending (test mode) working"
    else
        log::error "  ❌ Template sending (test mode) failed"
        return 1
    fi
    
    # Clean up
    vrooli resource twilio content template-delete "$template_name" &>/dev/null
    
    return 0
}

# Run all tests
main() {
    log::header "🧪 Twilio History & Templates Integration Tests"
    
    local errors=0
    
    test_message_history || ((errors++))
    test_message_templates || ((errors++))
    test_template_sending || ((errors++))
    
    if [[ $errors -eq 0 ]]; then
        log::success "✅ All history & template tests passed!"
        return 0
    else
        log::error "❌ $errors test(s) failed"
        return 1
    fi
}

# Only run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi