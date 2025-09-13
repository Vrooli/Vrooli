#!/bin/bash

# Widget Generation Test for AI Chatbot Manager
# Tests the embeddable widget code generation and functionality

set -e

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Dynamic port discovery
get_api_url() {
    local api_port
    api_port=$(vrooli scenario port ai-chatbot-manager API_PORT 2>/dev/null)
    
    if [[ -z "$api_port" ]]; then
        echo "ERROR: ai-chatbot-manager is not running" >&2
        echo "Start it with: vrooli scenario run ai-chatbot-manager" >&2
        exit 1
    fi
    
    echo "http://localhost:${api_port}"
}

# Get API URL
API_BASE_URL="${API_BASE_URL:-$(get_api_url)}"

log_info() {
    echo -e "${YELLOW}[WIDGET-TEST]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[PASS]${NC} $1"
}

log_error() {
    echo -e "${RED}[FAIL]${NC} $1"
}

log_detail() {
    echo -e "${BLUE}[DETAIL]${NC} $1"
}

# Create a test chatbot and get widget code
create_chatbot_with_widget() {
    log_info "Creating chatbot with custom widget configuration..."
    
    local response
    response=$(curl -s -X POST "$API_BASE_URL/api/v1/chatbots" \
        -H "Content-Type: application/json" \
        -d '{
            "name": "Widget Test Bot",
            "description": "Bot for widget testing",
            "personality": "You are a helpful customer support assistant.",
            "knowledge_base": "This is a test bot for widget functionality testing.",
            "widget_config": {
                "theme": "dark",
                "position": "bottom-left",
                "primaryColor": "#FF5722"
            }
        }')
    
    echo "$response"
}

# Test widget code structure
test_widget_structure() {
    log_info "Testing widget embed code structure..."
    
    local response="$1"
    local widget_code
    widget_code=$(echo "$response" | jq -r '.widget_embed_code // empty')
    
    if [[ -z "$widget_code" ]]; then
        log_error "No widget embed code returned"
        return 1
    fi
    
    # Check for essential widget components
    local checks_passed=0
    local total_checks=0
    
    # Check for script tag
    total_checks=$((total_checks + 1))
    if [[ "$widget_code" =~ "<script>" ]]; then
        log_success "Widget contains script tag"
        checks_passed=$((checks_passed + 1))
    else
        log_error "Widget missing script tag"
    fi
    
    # Check for chatbot ID
    total_checks=$((total_checks + 1))
    if [[ "$widget_code" =~ "CHATBOT_ID" ]]; then
        log_success "Widget contains chatbot ID reference"
        checks_passed=$((checks_passed + 1))
    else
        log_error "Widget missing chatbot ID"
    fi
    
    # Check for WebSocket setup
    total_checks=$((total_checks + 1))
    if [[ "$widget_code" =~ "WebSocket" ]] || [[ "$widget_code" =~ "ws://" ]]; then
        log_success "Widget includes WebSocket functionality"
        checks_passed=$((checks_passed + 1))
    else
        log_error "Widget missing WebSocket setup"
    fi
    
    # Check for API URL configuration
    total_checks=$((total_checks + 1))
    if [[ "$widget_code" =~ "API_URL" ]] || [[ "$widget_code" =~ "localhost" ]]; then
        log_success "Widget includes API URL configuration"
        checks_passed=$((checks_passed + 1))
    else
        log_error "Widget missing API URL"
    fi
    
    # Check for UI elements
    total_checks=$((total_checks + 1))
    if [[ "$widget_code" =~ "createElement" ]]; then
        log_success "Widget creates DOM elements"
        checks_passed=$((checks_passed + 1))
    else
        log_error "Widget missing DOM element creation"
    fi
    
    # Check for chat input handling
    total_checks=$((total_checks + 1))
    if [[ "$widget_code" =~ "input" ]] && [[ "$widget_code" =~ "message" ]]; then
        log_success "Widget includes message input handling"
        checks_passed=$((checks_passed + 1))
    else
        log_error "Widget missing input handling"
    fi
    
    # Check for custom configuration
    total_checks=$((total_checks + 1))
    if [[ "$widget_code" =~ "#FF5722" ]]; then
        log_success "Widget applies custom color configuration"
        checks_passed=$((checks_passed + 1))
    else
        log_error "Widget not using custom configuration"
    fi
    
    # Check for position configuration
    total_checks=$((total_checks + 1))
    if [[ "$widget_code" =~ "bottom-left" ]]; then
        log_success "Widget applies custom position configuration"
        checks_passed=$((checks_passed + 1))
    else
        log_error "Widget not using custom position"
    fi
    
    echo ""
    log_info "Widget structure test: $checks_passed/$total_checks checks passed"
    
    if [[ $checks_passed -eq $total_checks ]]; then
        return 0
    else
        return 1
    fi
}

# Test widget HTML generation
test_widget_html_generation() {
    log_info "Testing widget HTML generation..."
    
    local widget_code="$1"
    
    # Create a test HTML file
    local test_file="/tmp/widget-test-$(date +%s).html"
    
    cat > "$test_file" << EOF
<!DOCTYPE html>
<html>
<head>
    <title>Widget Test Page</title>
</head>
<body>
    <h1>AI Chatbot Widget Test Page</h1>
    <p>This page tests the embedded chatbot widget.</p>
    
    <!-- Embed the widget code -->
    $widget_code
</body>
</html>
EOF
    
    # Check if HTML file was created successfully
    if [[ -f "$test_file" ]]; then
        log_success "Test HTML file created: $test_file"
        
        # Validate HTML structure
        local line_count
        line_count=$(wc -l < "$test_file")
        
        if [[ $line_count -gt 10 ]]; then
            log_success "HTML file contains widget code ($line_count lines)"
        else
            log_error "HTML file seems incomplete ($line_count lines)"
        fi
        
        # Clean up
        rm -f "$test_file"
        return 0
    else
        log_error "Failed to create test HTML file"
        return 1
    fi
}

# Test widget JavaScript validity
test_widget_javascript() {
    log_info "Testing widget JavaScript validity..."
    
    local widget_code="$1"
    
    # Extract JavaScript from widget code
    local js_code
    js_code=$(echo "$widget_code" | sed -n '/<script>/,/<\/script>/p' | sed '1d;$d')
    
    if [[ -z "$js_code" ]]; then
        log_error "Could not extract JavaScript from widget code"
        return 1
    fi
    
    # Check for common JavaScript syntax elements
    local checks_passed=0
    
    # Check for function definitions
    if [[ "$js_code" =~ "function" ]]; then
        log_success "Widget contains function definitions"
        checks_passed=$((checks_passed + 1))
    fi
    
    # Check for event handlers
    if [[ "$js_code" =~ "onclick" ]] || [[ "$js_code" =~ "addEventListener" ]]; then
        log_success "Widget includes event handlers"
        checks_passed=$((checks_passed + 1))
    fi
    
    # Check for error handling
    if [[ "$js_code" =~ "try" ]] || [[ "$js_code" =~ "catch" ]]; then
        log_success "Widget includes error handling"
        checks_passed=$((checks_passed + 1))
    fi
    
    # Check for IIFE (Immediately Invoked Function Expression)
    if [[ "$js_code" =~ "\(function\(\)" ]]; then
        log_success "Widget uses IIFE pattern for encapsulation"
        checks_passed=$((checks_passed + 1))
    fi
    
    if [[ $checks_passed -ge 3 ]]; then
        log_info "JavaScript validation: $checks_passed/4 checks passed"
        return 0
    else
        log_error "JavaScript validation failed: only $checks_passed/4 checks passed"
        return 1
    fi
}

# Test multiple widget configurations
test_multiple_configurations() {
    log_info "Testing multiple widget configurations..."
    
    local configs=(
        '{"theme":"light","position":"top-right","primaryColor":"#4CAF50"}'
        '{"theme":"dark","position":"bottom-right","primaryColor":"#2196F3"}'
        '{"theme":"light","position":"top-left","primaryColor":"#9C27B0"}'
    )
    
    local success_count=0
    
    for config in "${configs[@]}"; do
        local response
        response=$(curl -s -X POST "$API_BASE_URL/api/v1/chatbots" \
            -H "Content-Type: application/json" \
            -d "{
                \"name\": \"Config Test Bot $(date +%s)\",
                \"personality\": \"Test bot\",
                \"widget_config\": $config
            }")
        
        local widget_code
        widget_code=$(echo "$response" | jq -r '.widget_embed_code // empty')
        
        if [[ -n "$widget_code" ]]; then
            # Extract color from config
            local expected_color
            expected_color=$(echo "$config" | jq -r '.primaryColor')
            
            if [[ "$widget_code" =~ "$expected_color" ]]; then
                log_success "Widget correctly applied configuration: $config"
                success_count=$((success_count + 1))
            else
                log_error "Widget did not apply configuration correctly"
            fi
        fi
    done
    
    if [[ $success_count -eq ${#configs[@]} ]]; then
        log_success "All ${#configs[@]} configuration tests passed"
        return 0
    else
        log_error "Only $success_count/${#configs[@]} configuration tests passed"
        return 1
    fi
}

# Main test execution
main() {
    echo "========================================="
    echo "AI Chatbot Manager Widget Generation Tests"
    echo "========================================="
    echo "API URL: $API_BASE_URL"
    echo ""
    
    # Check dependencies
    if ! command -v jq >/dev/null 2>&1; then
        log_error "jq is required but not installed"
        exit 1
    fi
    
    # Create test chatbot and get widget
    log_info "Creating test chatbot..."
    local response
    response=$(create_chatbot_with_widget)
    
    local chatbot_id
    chatbot_id=$(echo "$response" | jq -r '.chatbot.id // empty')
    
    if [[ -z "$chatbot_id" ]]; then
        log_error "Failed to create test chatbot"
        echo "$response"
        exit 1
    fi
    
    log_success "Created test chatbot: $chatbot_id"
    
    local widget_code
    widget_code=$(echo "$response" | jq -r '.widget_embed_code // empty')
    
    if [[ -z "$widget_code" ]]; then
        log_error "No widget code generated"
        exit 1
    fi
    
    log_success "Widget embed code generated (${#widget_code} characters)"
    echo ""
    
    # Run tests
    local failed=0
    
    # Test widget structure
    if ! test_widget_structure "$response"; then
        failed=$((failed + 1))
    fi
    echo ""
    
    # Test HTML generation
    if ! test_widget_html_generation "$widget_code"; then
        failed=$((failed + 1))
    fi
    echo ""
    
    # Test JavaScript validity
    if ! test_widget_javascript "$widget_code"; then
        failed=$((failed + 1))
    fi
    echo ""
    
    # Test multiple configurations
    if ! test_multiple_configurations; then
        failed=$((failed + 1))
    fi
    
    echo ""
    echo "========================================="
    
    if [[ $failed -eq 0 ]]; then
        log_success "All widget generation tests passed!"
        echo ""
        echo "Example widget embed code (first 500 chars):"
        echo "${widget_code:0:500}..."
        echo ""
        echo "To test the widget in a browser:"
        echo "1. Create an HTML file with the widget code"
        echo "2. Open the file in a web browser"
        echo "3. The chat widget should appear on the page"
        exit 0
    else
        log_error "$failed widget tests failed"
        exit 1
    fi
}

# Run main function
main "$@"