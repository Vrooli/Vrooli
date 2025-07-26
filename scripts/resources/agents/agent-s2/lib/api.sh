#!/usr/bin/env bash
# Agent S2 API Functions
# API interaction and testing utilities

#######################################
# Make API request to Agent S2
# Arguments:
#   $1 - HTTP method (GET, POST, etc.)
#   $2 - Endpoint path
#   $3 - Request body (optional, for POST)
# Returns: 0 if successful, 1 if failed
# Outputs: Response body
#######################################
agents2::api_request() {
    local method="$1"
    local endpoint="$2"
    local body="${3:-}"
    local url="${AGENTS2_BASE_URL}${endpoint}"
    
    local curl_args=(
        -s
        -X "$method"
        -H "Content-Type: application/json"
        --max-time "$AGENTS2_API_TIMEOUT"
    )
    
    if [[ -n "$body" ]]; then
        curl_args+=(-d "$body")
    fi
    
    if [[ -n "${AGENTS2_API_KEY:-}" ]]; then
        curl_args+=(-H "Authorization: Bearer $AGENTS2_API_KEY")
    fi
    
    local response
    local status_code
    
    # Make request and capture both response and status
    response=$(curl -w '\n%{http_code}' "${curl_args[@]}" "$url" 2>/dev/null)
    status_code=$(echo "$response" | tail -n1)
    response=$(echo "$response" | sed '$d')
    
    # Check status code
    if [[ "$status_code" =~ ^2[0-9][0-9]$ ]]; then
        echo "$response"
        return 0
    else
        echo "$response" >&2
        return 1
    fi
}

#######################################
# Test screenshot API
# Arguments:
#   $1 - Output file path (optional)
# Returns: 0 if successful, 1 if failed
#######################################
agents2::test_screenshot() {
    local output_file="${1:-screenshot_test.png}"
    
    log::info "Testing screenshot API..."
    
    local response
    if ! response=$(agents2::api_request "POST" "/screenshot" '{"format": "png", "quality": 95}'); then
        log::error "Screenshot API request failed"
        return 1
    fi
    
    # Extract base64 data using jq if available
    if system::is_command "jq"; then
        local image_data
        image_data=$(echo "$response" | jq -r '.data // empty' 2>/dev/null)
        
        if [[ -n "$image_data" ]]; then
            # Remove data URL prefix and decode
            echo "$image_data" | sed 's/^data:image\/[^;]*;base64,//' | base64 -d > "$output_file"
            log::success "Screenshot saved to: $output_file"
            
            # Show image info if possible
            if system::is_command "file"; then
                file "$output_file"
            fi
            return 0
        fi
    fi
    
    log::warn "Could not extract screenshot data"
    echo "$response"
    return 1
}

#######################################
# Test mouse automation
# Returns: 0 if successful, 1 if failed
#######################################
agents2::test_mouse_automation() {
    log::info "Testing mouse automation..."
    
    # Get current position
    log::info "Getting current mouse position..."
    local pos_response
    if pos_response=$(agents2::api_request "GET" "/mouse/position"); then
        echo "Current position: $pos_response"
    fi
    
    # Move mouse
    log::info "Moving mouse to center of screen..."
    local move_request='{"task_type": "mouse_move", "parameters": {"x": 960, "y": 540}}'
    local move_response
    if move_response=$(agents2::api_request "POST" "/execute" "$move_request"); then
        echo "Move response: $move_response"
        return 0
    else
        log::error "Mouse move failed"
        return 1
    fi
}

#######################################
# Test keyboard automation
# Returns: 0 if successful, 1 if failed
#######################################
agents2::test_keyboard_automation() {
    log::info "Testing keyboard automation..."
    
    # Type text
    local type_request='{"task_type": "type_text", "parameters": {"text": "Hello from Agent S2!", "interval": 0.1}}'
    local type_response
    if type_response=$(agents2::api_request "POST" "/execute" "$type_request"); then
        log::success "Text typed successfully"
        echo "Response: $type_response"
        return 0
    else
        log::error "Text typing failed"
        return 1
    fi
}

#######################################
# Test automation sequence
# Returns: 0 if successful, 1 if failed
#######################################
agents2::test_automation_sequence() {
    log::info "Testing automation sequence..."
    
    local sequence_request
    sequence_request=$(cat <<'EOF'
{
    "task_type": "automation_sequence",
    "parameters": {
        "steps": [
            {"type": "mouse_move", "parameters": {"x": 100, "y": 100}},
            {"type": "wait", "parameters": {"seconds": 1}},
            {"type": "click", "parameters": {"x": 100, "y": 100}},
            {"type": "wait", "parameters": {"seconds": 0.5}},
            {"type": "type_text", "parameters": {"text": "Agent S2 Test"}},
            {"type": "key_press", "parameters": {"keys": ["Return"]}}
        ]
    }
}
EOF
)
    
    local sequence_response
    if sequence_response=$(agents2::api_request "POST" "/execute" "$sequence_request"); then
        log::success "Automation sequence completed"
        echo "$sequence_response" | jq . 2>/dev/null || echo "$sequence_response"
        return 0
    else
        log::error "Automation sequence failed"
        return 1
    fi
}

#######################################
# Test async task execution
# Returns: 0 if successful, 1 if failed
#######################################
agents2::test_async_task() {
    log::info "Testing async task execution..."
    
    # Submit async task
    local async_request='{"task_type": "screenshot", "parameters": {}, "async_execution": true}'
    local submit_response
    if ! submit_response=$(agents2::api_request "POST" "/execute" "$async_request"); then
        log::error "Failed to submit async task"
        return 1
    fi
    
    # Extract task ID
    local task_id
    if system::is_command "jq"; then
        task_id=$(echo "$submit_response" | jq -r '.task_id // empty')
    fi
    
    if [[ -z "$task_id" ]]; then
        log::error "Could not extract task ID"
        echo "$submit_response"
        return 1
    fi
    
    log::info "Task submitted with ID: $task_id"
    
    # Poll for completion
    local max_attempts=10
    local attempt=1
    
    while [[ $attempt -le $max_attempts ]]; do
        sleep 1
        local status_response
        if status_response=$(agents2::api_request "GET" "/tasks/$task_id"); then
            local status
            if system::is_command "jq"; then
                status=$(echo "$status_response" | jq -r '.status // "unknown"')
            else
                status="unknown"
            fi
            
            log::info "Task status: $status"
            
            if [[ "$status" == "completed" || "$status" == "failed" ]]; then
                echo "$status_response" | jq . 2>/dev/null || echo "$status_response"
                return 0
            fi
        fi
        ((attempt++))
    done
    
    log::error "Task did not complete within timeout"
    return 1
}

#######################################
# Interactive API test menu
#######################################
agents2::api_test_menu() {
    while true; do
        echo
        log::header "Agent S2 API Test Menu"
        echo "1. Test Health Check"
        echo "2. Get Capabilities"
        echo "3. Take Screenshot"
        echo "4. Test Mouse Automation"
        echo "5. Test Keyboard Automation"
        echo "6. Test Automation Sequence"
        echo "7. Test Async Task"
        echo "8. Custom API Request"
        echo "9. Back to Main Menu"
        echo
        read -p "Select option (1-9): " choice
        
        case "$choice" in
            1)
                log::info "Testing health check..."
                agents2::api_request "GET" "/health" | jq . 2>/dev/null || agents2::api_request "GET" "/health"
                ;;
            2)
                log::info "Getting capabilities..."
                agents2::api_request "GET" "/capabilities" | jq . 2>/dev/null || agents2::api_request "GET" "/capabilities"
                ;;
            3)
                agents2::test_screenshot
                ;;
            4)
                agents2::test_mouse_automation
                ;;
            5)
                agents2::test_keyboard_automation
                ;;
            6)
                agents2::test_automation_sequence
                ;;
            7)
                agents2::test_async_task
                ;;
            8)
                read -p "Enter endpoint path (e.g., /health): " endpoint
                read -p "Enter HTTP method (GET/POST): " method
                if [[ "$method" == "POST" ]]; then
                    read -p "Enter JSON body (or press Enter for none): " body
                fi
                agents2::api_request "$method" "$endpoint" "$body" | jq . 2>/dev/null || agents2::api_request "$method" "$endpoint" "$body"
                ;;
            9)
                break
                ;;
            *)
                log::error "Invalid option"
                ;;
        esac
        
        echo
        read -p "Press Enter to continue..."
    done
}