#!/usr/bin/env bash
# Browserless Mock - Tier 2 (Stateful)
# 
# Provides stateful Browserless browser automation mocking for testing:
# - Browser session management
# - Screenshot and PDF generation
# - Web scraping simulation
# - Function execution
# - Pressure/health monitoring
# - Error injection for resilience testing
#
# Coverage: ~80% of common Browserless operations in 500 lines

# === Configuration ===
declare -gA BROWSERLESS_SESSIONS=()       # Session_id -> "browser|status|started"
declare -gA BROWSERLESS_JOBS=()           # Job_id -> "type|status|result"
declare -gA BROWSERLESS_SCREENSHOTS=()    # Url -> "path|timestamp"
declare -gA BROWSERLESS_CONFIG=(          # Service configuration
    [status]="running"
    [port]="3001"
    [max_browsers]="5"
    [active_browsers]="0"
    [queued]="0"
    [cpu]="0.15"
    [memory]="0.45"
    [error_mode]=""
    [version]="2.0.0"
)

# Debug mode
declare -g BROWSERLESS_DEBUG="${BROWSERLESS_DEBUG:-}"

# === Helper Functions ===
browserless_debug() {
    [[ -n "$BROWSERLESS_DEBUG" ]] && echo "[MOCK:BROWSERLESS] $*" >&2
}

browserless_check_error() {
    case "${BROWSERLESS_CONFIG[error_mode]}" in
        "service_down")
            echo "Error: Browserless service is not running" >&2
            return 1
            ;;
        "browser_limit")
            echo "Error: Browser limit reached" >&2
            return 1
            ;;
        "timeout")
            echo "Error: Operation timed out" >&2
            return 1
            ;;
    esac
    return 0
}

browserless_generate_id() {
    printf "%08x-%04x" $RANDOM $RANDOM
}

# === Main Browserless Command ===
browserless() {
    browserless_debug "browserless called with: $*"
    
    if ! browserless_check_error; then
        return $?
    fi
    
    local command="${1:-help}"
    shift || true
    
    case "$command" in
        screenshot)
            browserless_cmd_screenshot "$@"
            ;;
        pdf)
            browserless_cmd_pdf "$@"
            ;;
        scrape)
            browserless_cmd_scrape "$@"
            ;;
        function)
            browserless_cmd_function "$@"
            ;;
        session)
            browserless_cmd_session "$@"
            ;;
        pressure)
            browserless_cmd_pressure "$@"
            ;;
        status)
            browserless_cmd_status "$@"
            ;;
        start|stop|restart)
            browserless_cmd_service "$command" "$@"
            ;;
        *)
            echo "Browserless CLI - Browser Automation Service"
            echo "Commands:"
            echo "  screenshot - Take website screenshot"
            echo "  pdf        - Generate PDF from URL"
            echo "  scrape     - Scrape website content"
            echo "  function   - Execute browser function"
            echo "  session    - Manage browser sessions"
            echo "  pressure   - Show resource pressure"
            echo "  status     - Show service status"
            echo "  start      - Start service"
            echo "  stop       - Stop service"
            echo "  restart    - Restart service"
            ;;
    esac
}

# === Screenshot Command ===
browserless_cmd_screenshot() {
    local url="${1:-}"
    local output="${2:-screenshot.png}"
    
    [[ -z "$url" ]] && { echo "Error: URL required" >&2; return 1; }
    
    # Check browser availability
    if [[ ${BROWSERLESS_CONFIG[active_browsers]} -ge ${BROWSERLESS_CONFIG[max_browsers]} ]]; then
        echo "Error: No browsers available" >&2
        return 1
    fi
    
    # Increment active browsers
    ((BROWSERLESS_CONFIG[active_browsers]++))
    
    echo "Taking screenshot of: $url"
    echo "Output: $output"
    
    # Store screenshot record
    BROWSERLESS_SCREENSHOTS[$url]="$output|$(date +%s)"
    
    # Simulate processing
    sleep 0.1
    
    # Decrement active browsers
    ((BROWSERLESS_CONFIG[active_browsers]--))
    
    echo "Screenshot saved to: $output"
    echo "Dimensions: 1920x1080"
    echo "Size: 245KB"
}

# === PDF Command ===
browserless_cmd_pdf() {
    local url="${1:-}"
    local output="${2:-document.pdf}"
    
    [[ -z "$url" ]] && { echo "Error: URL required" >&2; return 1; }
    
    # Check browser availability
    if [[ ${BROWSERLESS_CONFIG[active_browsers]} -ge ${BROWSERLESS_CONFIG[max_browsers]} ]]; then
        echo "Error: No browsers available" >&2
        return 1
    fi
    
    # Increment active browsers
    ((BROWSERLESS_CONFIG[active_browsers]++))
    
    echo "Generating PDF from: $url"
    echo "Output: $output"
    
    # Simulate processing
    sleep 0.1
    
    # Decrement active browsers
    ((BROWSERLESS_CONFIG[active_browsers]--))
    
    echo "PDF saved to: $output"
    echo "Pages: 3"
    echo "Size: 512KB"
}

# === Scrape Command ===
browserless_cmd_scrape() {
    local url="${1:-}"
    local selector="${2:-body}"
    
    [[ -z "$url" ]] && { echo "Error: URL required" >&2; return 1; }
    
    local job_id=$(browserless_generate_id)
    BROWSERLESS_JOBS[$job_id]="scrape|running|"
    
    echo "Scraping: $url"
    echo "Selector: $selector"
    echo "Job ID: $job_id"
    
    # Simulate scraping
    BROWSERLESS_JOBS[$job_id]="scrape|completed|success"
    
    echo ""
    echo "Scraped content:"
    echo "<div class=\"content\">"
    echo "  <h1>Sample Page Title</h1>"
    echo "  <p>This is sample scraped content from $url</p>"
    echo "  <ul>"
    echo "    <li>Item 1</li>"
    echo "    <li>Item 2</li>"
    echo "  </ul>"
    echo "</div>"
}

# === Function Command ===
browserless_cmd_function() {
    local code="${1:-}"
    
    [[ -z "$code" ]] && { echo "Error: function code required" >&2; return 1; }
    
    local job_id=$(browserless_generate_id)
    BROWSERLESS_JOBS[$job_id]="function|running|"
    
    echo "Executing browser function..."
    echo "Job ID: $job_id"
    
    # Simulate function execution
    BROWSERLESS_JOBS[$job_id]="function|completed|success"
    
    echo "Function result:"
    echo '{"success":true,"data":"Function executed successfully","timing":125}'
}

# === Session Management ===
browserless_cmd_session() {
    local action="${1:-list}"
    shift || true
    
    case "$action" in
        list)
            echo "Browser sessions:"
            if [[ ${#BROWSERLESS_SESSIONS[@]} -eq 0 ]]; then
                echo "  (none)"
            else
                for session_id in "${!BROWSERLESS_SESSIONS[@]}"; do
                    local data="${BROWSERLESS_SESSIONS[$session_id]}"
                    IFS='|' read -r browser status started <<< "$data"
                    echo "  $session_id - Browser: $browser, Status: $status"
                done
            fi
            ;;
        create)
            if [[ ${BROWSERLESS_CONFIG[active_browsers]} -ge ${BROWSERLESS_CONFIG[max_browsers]} ]]; then
                echo "Error: Browser limit reached" >&2
                return 1
            fi
            
            local session_id=$(browserless_generate_id)
            BROWSERLESS_SESSIONS[$session_id]="chrome|active|$(date +%s)"
            ((BROWSERLESS_CONFIG[active_browsers]++))
            
            browserless_debug "Created session: $session_id"
            echo "Session created: $session_id"
            ;;
        close)
            local session_id="${1:-}"
            [[ -z "$session_id" ]] && { echo "Error: session ID required" >&2; return 1; }
            
            if [[ -n "${BROWSERLESS_SESSIONS[$session_id]}" ]]; then
                unset BROWSERLESS_SESSIONS[$session_id]
                ((BROWSERLESS_CONFIG[active_browsers]--))
                echo "Session closed: $session_id"
            else
                echo "Error: session not found: $session_id" >&2
                return 1
            fi
            ;;
        *)
            echo "Usage: browserless session {list|create|close} [args]"
            return 1
            ;;
    esac
}

# === Pressure Command ===
browserless_cmd_pressure() {
    echo "Browser Pool Pressure"
    echo "===================="
    echo "Running: ${BROWSERLESS_CONFIG[active_browsers]}"
    echo "Queued: ${BROWSERLESS_CONFIG[queued]}"
    echo "Max Concurrent: ${BROWSERLESS_CONFIG[max_browsers]}"
    echo "Available: $((${BROWSERLESS_CONFIG[max_browsers]} - ${BROWSERLESS_CONFIG[active_browsers]}))"
    echo "CPU: ${BROWSERLESS_CONFIG[cpu]}"
    echo "Memory: ${BROWSERLESS_CONFIG[memory]}"
    echo "Recently Rejected: 0"
}

# === Status Command ===
browserless_cmd_status() {
    echo "Browserless Status"
    echo "=================="
    echo "Service: ${BROWSERLESS_CONFIG[status]}"
    echo "Port: ${BROWSERLESS_CONFIG[port]}"
    echo "Version: ${BROWSERLESS_CONFIG[version]}"
    echo ""
    echo "Browsers:"
    echo "  Active: ${BROWSERLESS_CONFIG[active_browsers]}/${BROWSERLESS_CONFIG[max_browsers]}"
    echo "  Sessions: ${#BROWSERLESS_SESSIONS[@]}"
    echo ""
    echo "Jobs: ${#BROWSERLESS_JOBS[@]}"
    echo "Screenshots: ${#BROWSERLESS_SCREENSHOTS[@]}"
}

# === Service Management ===
browserless_cmd_service() {
    local action="$1"
    
    case "$action" in
        start)
            if [[ "${BROWSERLESS_CONFIG[status]}" == "running" ]]; then
                echo "Browserless is already running"
            else
                BROWSERLESS_CONFIG[status]="running"
                echo "Browserless started on port ${BROWSERLESS_CONFIG[port]}"
            fi
            ;;
        stop)
            BROWSERLESS_CONFIG[status]="stopped"
            BROWSERLESS_CONFIG[active_browsers]="0"
            BROWSERLESS_SESSIONS=()
            echo "Browserless stopped"
            ;;
        restart)
            BROWSERLESS_CONFIG[status]="stopped"
            BROWSERLESS_CONFIG[active_browsers]="0"
            BROWSERLESS_SESSIONS=()
            BROWSERLESS_CONFIG[status]="running"
            echo "Browserless restarted"
            ;;
    esac
}

# === HTTP API Mock (via curl interceptor) ===
curl() {
    browserless_debug "curl called with: $*"
    
    local url="" method="GET" data=""
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -X) method="$2"; shift 2 ;;
            -d|--data) data="$2"; shift 2 ;;
            http*) url="$1"; shift ;;
            *) shift ;;
        esac
    done
    
    # Check if this is a Browserless API call
    if [[ "$url" =~ localhost:3001 || "$url" =~ 127.0.0.1:3001 ]]; then
        browserless_handle_api "$method" "$url" "$data"
        return $?
    fi
    
    echo "curl: Not a Browserless endpoint"
    return 0
}

browserless_handle_api() {
    local method="$1" url="$2" data="$3"
    
    case "$url" in
        */screenshot)
            if [[ "$method" == "POST" ]]; then
                echo '{"success":true,"data":"data:image/png;base64,iVBORw0KGgoAAAANS..."}'
            else
                echo '{"error":"Method not allowed"}'
            fi
            ;;
        */pdf)
            if [[ "$method" == "POST" ]]; then
                echo '{"success":true,"data":"data:application/pdf;base64,JVBERi0xLj..."}'
            else
                echo '{"error":"Method not allowed"}'
            fi
            ;;
        */scrape)
            if [[ "$method" == "POST" ]]; then
                echo '{"data":[{"html":"<h1>Title</h1>","text":"Title"}]}'
            else
                echo '{"error":"Method not allowed"}'
            fi
            ;;
        */function)
            if [[ "$method" == "POST" ]]; then
                echo '{"data":"Function result","type":"string"}'
            else
                echo '{"error":"Method not allowed"}'
            fi
            ;;
        */pressure)
            echo '{"running":'${BROWSERLESS_CONFIG[active_browsers]}',"queued":'${BROWSERLESS_CONFIG[queued]}',"maxConcurrent":'${BROWSERLESS_CONFIG[max_browsers]}'}'
            ;;
        */health)
            if [[ "${BROWSERLESS_CONFIG[status]}" == "running" ]]; then
                echo '{"status":"ok","version":"'${BROWSERLESS_CONFIG[version]}'"}'
            else
                echo '{"status":"error","message":"Service not running"}'
            fi
            ;;
        *)
            echo '{"status":"ok","version":"'${BROWSERLESS_CONFIG[version]}'"}'
            ;;
    esac
}

# === Mock Control Functions ===
browserless_mock_reset() {
    browserless_debug "Resetting mock state"
    
    BROWSERLESS_SESSIONS=()
    BROWSERLESS_JOBS=()
    BROWSERLESS_SCREENSHOTS=()
    BROWSERLESS_CONFIG[error_mode]=""
    BROWSERLESS_CONFIG[status]="running"
    BROWSERLESS_CONFIG[active_browsers]="0"
    BROWSERLESS_CONFIG[queued]="0"
}

browserless_mock_set_error() {
    BROWSERLESS_CONFIG[error_mode]="$1"
    browserless_debug "Set error mode: $1"
}

browserless_mock_dump_state() {
    echo "=== Browserless Mock State ==="
    echo "Status: ${BROWSERLESS_CONFIG[status]}"
    echo "Port: ${BROWSERLESS_CONFIG[port]}"
    echo "Active Browsers: ${BROWSERLESS_CONFIG[active_browsers]}/${BROWSERLESS_CONFIG[max_browsers]}"
    echo "Sessions: ${#BROWSERLESS_SESSIONS[@]}"
    echo "Jobs: ${#BROWSERLESS_JOBS[@]}"
    echo "Screenshots: ${#BROWSERLESS_SCREENSHOTS[@]}"
    echo "Error Mode: ${BROWSERLESS_CONFIG[error_mode]:-none}"
    echo "=========================="
}

# === Convention-based Test Functions ===
test_browserless_connection() {
    browserless_debug "Testing connection..."
    
    local result
    result=$(curl -s http://localhost:3001/health 2>&1)
    
    if [[ "$result" =~ "status" ]]; then
        browserless_debug "Connection test passed"
        return 0
    else
        browserless_debug "Connection test failed"
        return 1
    fi
}

test_browserless_health() {
    browserless_debug "Testing health..."
    
    test_browserless_connection || return 1
    
    curl -s http://localhost:3001/pressure >/dev/null 2>&1 || return 1
    browserless session create >/dev/null 2>&1 || return 1
    
    browserless_debug "Health test passed"
    return 0
}

test_browserless_basic() {
    browserless_debug "Testing basic operations..."
    
    browserless screenshot "https://example.com" test.png >/dev/null 2>&1 || return 1
    browserless pdf "https://example.com" test.pdf >/dev/null 2>&1 || return 1
    browserless scrape "https://example.com" "h1" >/dev/null 2>&1 || return 1
    
    browserless_debug "Basic test passed"
    return 0
}

# === Export Functions ===
export -f browserless curl
export -f test_browserless_connection test_browserless_health test_browserless_basic
export -f browserless_mock_reset browserless_mock_set_error browserless_mock_dump_state
export -f browserless_debug browserless_check_error

# Initialize
browserless_mock_reset
browserless_debug "Browserless Tier 2 mock initialized"
# Ensure we return success when sourced
return 0 2>/dev/null || true
