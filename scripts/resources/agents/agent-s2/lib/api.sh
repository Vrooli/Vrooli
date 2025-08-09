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
    if ! response=$(agents2::api_request "POST" "/screenshot?format=png&quality=95" ''); then
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

#######################################
# Execute AI task using natural language
# Arguments:
#   $1 - Task description
# Returns: 0 if successful, 1 if failed
#######################################
agents2::execute_ai_task() {
    local task="${1:-}"
    
    if [[ -z "$task" ]]; then
        log::error "Task description is required"
        return 1
    fi
    
    log::info "Executing AI task: $task"
    
    # Check browser health before starting task
    log::info "Checking browser state..."
    local cleanup_response
    if cleanup_response=$(agents2::api_request "POST" "/browser/cleanup" 2>/dev/null); then
        if system::is_command "jq"; then
            local processes_killed=$(echo "$cleanup_response" | jq -r '.processes_killed // 0')
            local locks_cleared=$(echo "$cleanup_response" | jq -r '.locks_cleared // false')
            if [[ "$processes_killed" -gt 0 ]] || [[ "$locks_cleared" == "true" ]]; then
                log::info "Browser cleanup performed: $processes_killed processes killed, locks cleared: $locks_cleared"
                sleep 2  # Give time for cleanup to settle
            fi
        fi
    fi
    
    # Show what we're sending to the AI
    log::info "=== AI REQUEST DEBUG ==="
    log::info "Task: $task"
    log::info "Security Config:"
    log::info "  - Allowed domains: ${ALLOWED_DOMAINS:-<none>}"
    log::info "  - Blocked domains: ${BLOCKED_DOMAINS:-<none>}"
    log::info "  - Security profile: ${SECURITY_PROFILE:-<default>}"
    log::info "======================="
    
    # Build request JSON with security parameters
    local request_json
    local jq_args=()
    jq_args+=(--arg task "$task")
    
    # Always add security parameters (empty string if not set)
    jq_args+=(--arg allowed_domains "${ALLOWED_DOMAINS:-}")
    jq_args+=(--arg blocked_domains "${BLOCKED_DOMAINS:-}")
    jq_args+=(--arg security_profile "${SECURITY_PROFILE:-}")
    
    # Build JSON with conditional fields
    if [[ -n "${ALLOWED_DOMAINS:-}" ]] || [[ -n "${BLOCKED_DOMAINS:-}" ]] || [[ -n "${SECURITY_PROFILE:-}" ]]; then
        request_json=$(jq -n "${jq_args[@]}" '{
            task: $task,
            screenshot_before: true,
            screenshot_after: true,
            security_config: ({} 
                | if ($allowed_domains != "") then .allowed_domains = ($allowed_domains | split(",")) else . end
                | if ($blocked_domains != "") then .blocked_domains = ($blocked_domains | split(",")) else . end
                | if ($security_profile != "") then .security_profile = $security_profile else . end
            )
        }')
    else
        # No security parameters, build simpler JSON
        request_json=$(jq -n \
            --arg task "$task" \
            '{
                task: $task,
                screenshot_before: true,
                screenshot_after: true
            }')
    fi
    
    # Execute AI action
    local response
    if response=$(agents2::api_request "POST" "/ai/action" "$request_json"); then
        log::success "AI task completed"
        
        # Pretty print response with jq if available
        if system::is_command "jq"; then
            # First show the full response
            echo "$response" | jq .
            
            # Extract and display key information
            local success task_status actions_count
            success=$(echo "$response" | jq -r '.success // false')
            task_status=$(echo "$response" | jq -r 'if .success then "completed" else "failed" end')
            actions_count=$(echo "$response" | jq -r '.actions_taken | length // 0')
            
            echo
            log::info "Task Status: $task_status"
            log::info "Actions Executed: $actions_count"
            
            # Display debug info if available in response
            if echo "$response" | jq -e '.debug_info' >/dev/null 2>&1; then
                echo
                log::info "=== AI PROMPTS SENT TO MODEL ==="
                echo "$response" | jq -r '.debug_info.from_handler | "System Prompt:\n\(.system_prompt)\n\nUser Prompt:\n\(.user_prompt)\n\nScreenshot Included: \(.screenshot_included)\n\nContext:\n\(.context | tostring)"'
                log::info "==============================="
            fi
            
            if [[ "$success" != "true" ]]; then
                local error_msg
                error_msg=$(echo "$response" | jq -r '.error // "Unknown error"')
                log::error "Task failed: $error_msg"
                return 1
            fi
        else
            echo "$response"
        fi
        
        return 0
    else
        log::error "Failed to execute AI task"
        return 1
    fi
}

#######################################
# Wrapper functions for test compatibility
# These provide the agent_s2:: interface expected by tests
#######################################

#######################################
# Browser automation wrapper
# Arguments:
#   $1 - URL
#   $2 - Action (e.g., "click button")
# Returns: 0 if successful, 1 if failed
#######################################
agent_s2::automate_browser() {
    local url="${1:-}"
    local action="${2:-}"
    
    if [[ -z "$url" ]]; then
        echo "URL is required" >&2
        return 1
    fi
    
    if [[ -z "$action" ]]; then
        echo "Action is required" >&2
        return 1
    fi
    
    # Use the specific API endpoint expected by tests
    local request_json
    request_json=$(jq -n \
        --arg url "$url" \
        --arg action "$action" \
        '{url: $url, action: $action}')
    
    agents2::api_request "POST" "/api/automate" "$request_json"
}

#######################################
# Screenshot capture wrapper
# Arguments:
#   $1 - URL (optional)
# Returns: 0 if successful, 1 if failed
#######################################
agent_s2::capture_screenshot() {
    local url="${1:-}"
    
    if [[ -z "$url" ]]; then
        echo "URL is required" >&2
        return 1
    fi
    
    # Use the specific API endpoint expected by tests
    local request_json
    request_json=$(jq -n --arg url "$url" '{url: $url}')
    
    agents2::api_request "POST" "/api/screenshot" "$request_json"
}

#######################################
# Data extraction wrapper
# Arguments:
#   $1 - URL (optional)
#   $2 - Selector or description
# Returns: 0 if successful, 1 if failed
#######################################
agent_s2::extract_data() {
    local url="${1:-}"
    local selector="${2:-}"
    
    if [[ -z "$url" ]]; then
        echo "URL is required" >&2
        return 1
    fi
    
    if [[ -z "$selector" ]]; then
        echo "Selector is required" >&2
        return 1
    fi
    
    # Use the specific API endpoint expected by tests
    local request_json
    request_json=$(jq -n \
        --arg url "$url" \
        --arg selector "$selector" \
        '{url: $url, selector: $selector}')
    
    agents2::api_request "POST" "/api/extract" "$request_json"
}

#######################################
# Health check wrapper
# Returns: 0 if healthy, 1 if not
#######################################
agent_s2::health_check() {
    local response
    if response=$(agents2::api_request "GET" "/api/health" 2>/dev/null); then
        echo "$response"
        return 0
    else
        echo "Agent-S2 API is not responding" >&2
        return 1
    fi
}

#######################################
# Session creation wrapper
# Returns: 0 if successful, 1 if failed
#######################################
agent_s2::create_session() {
    agents2::api_request "POST" "/api/session/create" '{}'
}

#######################################
# Session destruction wrapper
# Arguments:
#   $1 - Session ID
# Returns: 0 if successful, 1 if failed
#######################################
agent_s2::destroy_session() {
    local session_id="${1:-}"
    
    if [[ -z "$session_id" ]]; then
        echo "Session ID is required" >&2
        return 1
    fi
    
    # Use the specific API endpoint expected by tests
    local request_json
    request_json=$(jq -n --arg session_id "$session_id" '{session_id: $session_id}')
    
    agents2::api_request "POST" "/api/session/destroy" "$request_json"
}

#######################################
# Batch automation wrapper
# Arguments:
#   $1 - JSON array of actions
# Returns: 0 if successful, 1 if failed
#######################################
agent_s2::batch_automate() {
    local url="${1:-}"
    local actions="${2:-}"
    
    if [[ -z "$url" ]]; then
        echo "URL is required" >&2
        return 1
    fi
    
    if [[ -z "$actions" ]]; then
        echo "Actions array is required" >&2
        return 1
    fi
    
    # Use the specific API endpoint expected by tests
    local request_json
    request_json=$(jq -n \
        --arg url "$url" \
        --argjson actions "$actions" \
        '{url: $url, actions: $actions}')
    
    agents2::api_request "POST" "/api/batch" "$request_json"
}

# Export functions for subshell availability
export -f agents2::api_request
export -f agents2::test_screenshot
export -f agents2::test_mouse_automation
export -f agents2::test_keyboard_automation
export -f agents2::test_automation_sequence
export -f agents2::test_async_task
export -f agents2::api_test_menu
export -f agents2::execute_ai_task
export -f agent_s2::automate_browser
export -f agent_s2::capture_screenshot
export -f agent_s2::extract_data
export -f agent_s2::health_check
export -f agent_s2::create_session
export -f agent_s2::destroy_session
export -f agent_s2::batch_automate