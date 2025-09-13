#!/bin/bash

# Health monitoring script for AI Chatbot Manager
# Performs comprehensive health checks on all components

set -e

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCENARIO_NAME="ai-chatbot-manager"
CHECK_INTERVAL=5  # Seconds between checks in continuous mode
MAX_RETRIES=3

# Dynamic port discovery
get_ports() {
    API_PORT=$(vrooli scenario port "$SCENARIO_NAME" API_PORT 2>/dev/null)
    UI_PORT=$(vrooli scenario port "$SCENARIO_NAME" UI_PORT 2>/dev/null)
    
    if [[ -z "$API_PORT" ]]; then
        echo -e "${RED}[ERROR]${NC} Scenario is not running"
        echo "Start it with: vrooli scenario run $SCENARIO_NAME"
        exit 1
    fi
    
    API_URL="http://localhost:${API_PORT}"
    UI_URL="http://localhost:${UI_PORT}"
}

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[✓]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[⚠]${NC} $1"
}

log_error() {
    echo -e "${RED}[✗]${NC} $1"
}

# Health check functions
check_api_health() {
    local status
    status=$(curl -s "${API_URL}/health" 2>/dev/null | jq -r '.status // "error"' 2>/dev/null || echo "unreachable")
    
    if [[ "$status" == "healthy" ]]; then
        log_success "API Health: Healthy"
        return 0
    else
        log_error "API Health: $status"
        return 1
    fi
}

check_api_response_time() {
    local response_time
    response_time=$(curl -o /dev/null -s -w '%{time_total}' "${API_URL}/health")
    
    # Convert to milliseconds
    response_time_ms=$(echo "$response_time * 1000" | bc)
    
    if (( $(echo "$response_time_ms < 500" | bc -l) )); then
        log_success "API Response Time: ${response_time_ms}ms (Good)"
    elif (( $(echo "$response_time_ms < 1000" | bc -l) )); then
        log_warning "API Response Time: ${response_time_ms}ms (Acceptable)"
    else
        log_error "API Response Time: ${response_time_ms}ms (Slow)"
    fi
}

check_ui_health() {
    local status_code
    status_code=$(curl -o /dev/null -s -w "%{http_code}" "${UI_URL}")
    
    if [[ "$status_code" == "200" ]]; then
        log_success "UI Health: Responding (HTTP $status_code)"
        return 0
    else
        log_error "UI Health: Not responding (HTTP $status_code)"
        return 1
    fi
}

check_database_connection() {
    # Check if API can connect to database
    local db_status
    db_status=$(curl -s "${API_URL}/health" 2>/dev/null | jq -r '.database // "unknown"' 2>/dev/null || echo "error")
    
    if [[ "$db_status" == "connected" || "$db_status" == "unknown" ]]; then
        log_success "Database Connection: Connected"
        return 0
    else
        log_error "Database Connection: $db_status"
        return 1
    fi
}

check_ollama_connection() {
    # Check if Ollama is accessible
    local ollama_status
    ollama_status=$(curl -s "http://localhost:11434/api/tags" 2>/dev/null | jq -r '.models // empty' 2>/dev/null)
    
    if [[ -n "$ollama_status" ]]; then
        log_success "Ollama Connection: Available"
        return 0
    else
        log_warning "Ollama Connection: Not available (chatbot functionality may be limited)"
        return 1
    fi
}

check_websocket_endpoint() {
    # Check if WebSocket endpoint is accessible (basic check)
    local ws_url="${API_URL}/api/v1/ws/test"
    local response_code
    response_code=$(curl -s -o /dev/null -w "%{http_code}" \
        -H "Connection: Upgrade" \
        -H "Upgrade: websocket" \
        -H "Sec-WebSocket-Version: 13" \
        -H "Sec-WebSocket-Key: $(openssl rand -base64 16)" \
        "$ws_url")
    
    # WebSocket upgrade should return 101 Switching Protocols or 400/404 for invalid path
    if [[ "$response_code" == "101" || "$response_code" == "400" || "$response_code" == "404" ]]; then
        log_success "WebSocket Endpoint: Available"
        return 0
    else
        log_warning "WebSocket Endpoint: Status $response_code"
        return 1
    fi
}

check_api_endpoints() {
    local endpoints=(
        "/api/v1/chatbots"
        "/health"
    )
    
    local failed=0
    for endpoint in "${endpoints[@]}"; do
        local status_code
        status_code=$(curl -o /dev/null -s -w "%{http_code}" "${API_URL}${endpoint}")
        
        if [[ "$status_code" == "200" || "$status_code" == "201" || "$status_code" == "204" ]]; then
            echo -e "  ${GREEN}✓${NC} ${endpoint} (HTTP $status_code)"
        else
            echo -e "  ${RED}✗${NC} ${endpoint} (HTTP $status_code)"
            ((failed++))
        fi
    done
    
    if [[ $failed -eq 0 ]]; then
        log_success "API Endpoints: All responding"
    else
        log_warning "API Endpoints: $failed endpoints not responding"
    fi
}

check_resource_usage() {
    # Get process IDs
    local api_pid ui_pid
    api_pid=$(pgrep -f "ai-chatbot-manager-api" 2>/dev/null | head -1)
    ui_pid=$(pgrep -f "scenarios/ai-chatbot-manager/ui" 2>/dev/null | head -1)
    
    if [[ -n "$api_pid" ]]; then
        # Get CPU and memory usage for API
        local api_stats
        api_stats=$(ps -p "$api_pid" -o %cpu,%mem --no-headers 2>/dev/null || echo "0 0")
        local api_cpu=$(echo "$api_stats" | awk '{print $1}')
        local api_mem=$(echo "$api_stats" | awk '{print $2}')
        
        echo -e "  API Process (PID $api_pid):"
        echo -e "    CPU: ${api_cpu}%"
        echo -e "    Memory: ${api_mem}%"
        
        # Check thresholds
        if (( $(echo "$api_cpu > 80" | bc -l) )); then
            log_warning "API CPU usage high: ${api_cpu}%"
        fi
        if (( $(echo "$api_mem > 10" | bc -l) )); then
            log_warning "API memory usage high: ${api_mem}%"
        fi
    else
        log_warning "API process not found"
    fi
    
    if [[ -n "$ui_pid" ]]; then
        # Get CPU and memory usage for UI
        local ui_stats
        ui_stats=$(ps -p "$ui_pid" -o %cpu,%mem --no-headers 2>/dev/null || echo "0 0")
        local ui_cpu=$(echo "$ui_stats" | awk '{print $1}')
        local ui_mem=$(echo "$ui_stats" | awk '{print $2}')
        
        echo -e "  UI Process (PID $ui_pid):"
        echo -e "    CPU: ${ui_cpu}%"
        echo -e "    Memory: ${ui_mem}%"
    else
        log_warning "UI process not found"
    fi
}

check_port_bindings() {
    log_info "Port Bindings:"
    
    # Check API port
    if netstat -tuln 2>/dev/null | grep -q ":${API_PORT} "; then
        echo -e "  ${GREEN}✓${NC} API Port ${API_PORT} is bound"
    else
        echo -e "  ${RED}✗${NC} API Port ${API_PORT} is not bound"
    fi
    
    # Check UI port
    if netstat -tuln 2>/dev/null | grep -q ":${UI_PORT} "; then
        echo -e "  ${GREEN}✓${NC} UI Port ${UI_PORT} is bound"
    else
        echo -e "  ${RED}✗${NC} UI Port ${UI_PORT} is not bound"
    fi
}

# Main health check function
perform_health_check() {
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}AI Chatbot Manager Health Check${NC}"
    echo -e "${BLUE}Time: $(date '+%Y-%m-%d %H:%M:%S')${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    
    # Get ports
    get_ports
    
    echo ""
    echo -e "${YELLOW}Service URLs:${NC}"
    echo "  API: $API_URL"
    echo "  UI:  $UI_URL"
    
    echo ""
    echo -e "${YELLOW}Health Checks:${NC}"
    
    # Perform all checks
    check_api_health
    check_api_response_time
    check_ui_health
    check_database_connection
    check_ollama_connection
    check_websocket_endpoint
    
    echo ""
    echo -e "${YELLOW}Endpoint Status:${NC}"
    check_api_endpoints
    
    echo ""
    echo -e "${YELLOW}Resource Usage:${NC}"
    check_resource_usage
    
    echo ""
    check_port_bindings
    
    echo ""
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

# Continuous monitoring mode
continuous_monitor() {
    log_info "Starting continuous health monitoring (Ctrl+C to stop)"
    log_info "Check interval: ${CHECK_INTERVAL} seconds"
    echo ""
    
    while true; do
        perform_health_check
        
        # Summary line
        if check_api_health &>/dev/null && check_ui_health &>/dev/null; then
            echo -e "${GREEN}Overall Status: HEALTHY${NC}"
        else
            echo -e "${RED}Overall Status: UNHEALTHY${NC}"
        fi
        
        echo ""
        echo "Next check in ${CHECK_INTERVAL} seconds... (Ctrl+C to stop)"
        sleep "$CHECK_INTERVAL"
        
        # Clear screen for next iteration (optional)
        # clear
    done
}

# Alert mode - returns exit code based on health
alert_mode() {
    perform_health_check
    
    # Return non-zero exit code if any critical check fails
    if ! check_api_health &>/dev/null; then
        exit 1
    fi
    
    if ! check_ui_health &>/dev/null; then
        exit 2
    fi
    
    exit 0
}

# Parse command line arguments
case "${1:-}" in
    --continuous|-c)
        continuous_monitor
        ;;
    --alert|-a)
        alert_mode
        ;;
    --help|-h)
        echo "AI Chatbot Manager Health Monitor"
        echo ""
        echo "Usage: $0 [OPTIONS]"
        echo ""
        echo "Options:"
        echo "  --continuous, -c    Run continuous monitoring"
        echo "  --alert, -a         Run once and exit with status code"
        echo "  --help, -h          Show this help message"
        echo ""
        echo "Default: Run single health check"
        ;;
    *)
        perform_health_check
        ;;
esac