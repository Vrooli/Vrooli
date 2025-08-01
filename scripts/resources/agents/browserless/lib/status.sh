#!/usr/bin/env bash
# Browserless Status and Health Checks
# Service monitoring and health verification

#######################################
# Check if Browserless API is responsive
# Returns: 0 if responsive, 1 otherwise
#######################################
browserless::is_healthy() {
    if system::is_command "curl"; then
        # Try multiple times as the service takes time to initialize
        local attempts=0
        while [ $attempts -lt $BROWSERLESS_HEALTH_CHECK_MAX_ATTEMPTS ]; do
            # Browserless uses /pressure endpoint for health checks
            if curl -f -s --max-time $BROWSERLESS_API_TIMEOUT "$BROWSERLESS_BASE_URL/pressure" >/dev/null 2>&1; then
                return 0
            fi
            attempts=$((attempts + 1))
            sleep $BROWSERLESS_HEALTH_CHECK_INTERVAL
        done
    fi
    return 1
}

#######################################
# Get Browserless pressure/status information
# Returns: JSON response or empty string
#######################################
browserless::get_pressure() {
    if system::is_command "curl"; then
        curl -s --max-time $BROWSERLESS_API_TIMEOUT "$BROWSERLESS_BASE_URL/pressure" 2>/dev/null || echo "{}"
    else
        echo "{}"
    fi
}

#######################################
# Show detailed service status
#######################################
browserless::show_detailed_status() {
    local pressure
    pressure=$(browserless::get_pressure)
    
    if [[ -n "$pressure" && "$pressure" != "{}" ]]; then
        echo
        log::info "Browser Session Details:"
        if command -v jq &> /dev/null; then
            echo "$pressure" | jq -r "
                \"  Running browsers: \(.running // 0)/\(.maxConcurrent // \"N/A\")\",
                \"  Queued requests: \(.queued // 0)\",
                \"  Recently rejected: \(.recentlyRejected // 0)\",
                \"  Available: \(.isAvailable // false)\",
                \"  CPU usage: \((.cpu // 0) * 100 | round)%\",
                \"  Memory usage: \((.memory // 0) * 100 | round)%\"
            " 2>/dev/null || {
                echo "  Status: Available"
                echo "  Raw response: $pressure"
            }
        else
            echo "  Status: Available"
            echo "  Raw response: $pressure"
        fi
    else
        echo
        log::info "Browser Session Details:"
        echo "  Status: Service not responding"
    fi
}

#######################################
# Show comprehensive Browserless status
#######################################
browserless::show_status() {
    log::header "📊 Browserless Status"
    
    # Check Docker
    if ! system::is_command "docker"; then
        log::error "${MSG_DOCKER_NOT_FOUND}"
        return 1
    fi
    
    if ! docker info >/dev/null 2>&1; then
        log::error "${MSG_DOCKER_NOT_RUNNING}"
        return 1
    fi
    
    # Check container status
    if browserless::container_exists; then
        if browserless::is_running; then
            log::success "${MSG_RUNNING}"
            
            # Show resource usage
            browserless::show_resource_usage
            
            # Check health
            if browserless::is_healthy; then
                log::success "${MSG_HEALTHY}"
            else
                log::warn "${MSG_HEALTH_CHECK_FAILED}"
            fi
            
            # Show detailed browser session info
            browserless::show_detailed_status
            
            # Additional details
            echo
            log::info "Browserless Details:"
            log::info "  Web UI: $BROWSERLESS_BASE_URL"
            log::info "  Container: $BROWSERLESS_CONTAINER_NAME"
            log::info "  Max Browsers: $BROWSERLESS_MAX_BROWSERS"
            log::info "  Headless Mode: $BROWSERLESS_HEADLESS"
            log::info "  Timeout: ${BROWSERLESS_TIMEOUT}ms"
            
            # Show management commands
            echo
            log::info "Management Commands:"
            log::info "  View logs: $0 --action logs"
            log::info "  Restart: $0 --action restart"
            log::info "  Test API: $0 --action usage"
        else
            log::warn "${MSG_EXISTS_NOT_RUNNING}"
            log::info "Start with: $0 --action start"
        fi
    else
        log::error "${MSG_NOT_INSTALLED}"
        log::info "Install with: $0 --action install"
    fi
}

#######################################
# Wait for Browserless to become healthy
# Arguments:
#   $1 - Max wait time in seconds (optional, defaults to 60)
# Returns: 0 if healthy, 1 if timeout
#######################################
browserless::wait_for_healthy() {
    local max_wait="${1:-60}"
    local wait_time=0
    local check_interval=3
    
    log::info "${MSG_WAITING_INIT}"
    
    while [ $wait_time -lt $max_wait ]; do
        if browserless::is_healthy; then
            log::success "✅ Browserless is healthy and responsive"
            return 0
        fi
        sleep $check_interval
        wait_time=$((wait_time + check_interval))
        echo -n "."
    done
    echo
    
    log::warn "${MSG_STARTED_NOT_HEALTHY}"
    return 1
}

#######################################
# Show service information and endpoints
#######################################
browserless::show_info() {
    cat << EOF
=== Browserless Resource Information ===

ID: browserless
Category: agents
Display Name: Browserless
Description: Browser automation powered by Chrome/Chromium

Service Details:
- Container Name: $BROWSERLESS_CONTAINER_NAME
- Service Port: $BROWSERLESS_PORT
- Service URL: $BROWSERLESS_BASE_URL
- Docker Image: $BROWSERLESS_IMAGE
- Data Directory: $BROWSERLESS_DATA_DIR

Endpoints:
- Status Check: $BROWSERLESS_BASE_URL/pressure
- Screenshot: POST $BROWSERLESS_BASE_URL/chrome/screenshot
- PDF: POST $BROWSERLESS_BASE_URL/chrome/pdf
- Content: POST $BROWSERLESS_BASE_URL/chrome/content
- Function: POST $BROWSERLESS_BASE_URL/chrome/function
- Scrape: POST $BROWSERLESS_BASE_URL/chrome/scrape

Configuration:
- Max Browsers: ${BROWSERLESS_MAX_BROWSERS}
- Headless Mode: ${BROWSERLESS_HEADLESS}
- Timeout: ${BROWSERLESS_TIMEOUT}ms

Browserless Features:
- High-performance browser automation
- Screenshot generation
- PDF generation
- Web scraping
- JavaScript execution
- Form automation
- Performance monitoring
- Network interception

Example Usage:
# Take a screenshot
curl -X POST $BROWSERLESS_BASE_URL/chrome/screenshot \\
  -H "Content-Type: application/json" \\
  -d '{"url": "https://example.com"}' \\
  --output screenshot.png

# Generate PDF
curl -X POST $BROWSERLESS_BASE_URL/chrome/pdf \\
  -H "Content-Type: application/json" \\
  -d '{"url": "https://example.com"}' \\
  --output document.pdf

# Scrape webpage
curl -X POST $BROWSERLESS_BASE_URL/chrome/content \\
  -H "Content-Type: application/json" \\
  -d '{"url": "https://example.com"}'

For more information, visit: https://www.browserless.io/docs/
EOF
}

# Export functions for subshell availability
export -f browserless::is_healthy
export -f browserless::get_pressure
export -f browserless::show_detailed_status
export -f browserless::show_status
export -f browserless::wait_for_healthy
export -f browserless::show_info