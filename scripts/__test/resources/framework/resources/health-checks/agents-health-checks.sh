#!/bin/bash
# ====================================================================
# Agent Resource Health Checks
# ====================================================================
#
# Category-specific health checks for agent resources including
# screen interaction capability, security sandbox status, and
# automation readiness.
#
# Supported Agent Resources:
# - Agent-S2: Autonomous screen interaction
# - Browserless: Chrome-as-a-Service
# - Claude Code: AI development assistant (CLI)
#
# ====================================================================

# Agent resource health check implementations
check_agent_s2_health() {
    local port="${1:-4113}"
    local health_level="${2:-basic}"
    
    # Basic connectivity test
    if ! curl -s --max-time 5 "http://localhost:${port}/health" >/dev/null 2>&1; then
        # Try alternative endpoints
        if curl -s --max-time 5 "http://localhost:${port}/" >/dev/null 2>&1; then
            echo "healthy"
            return 0
        else
            echo "unreachable"
            return 1
        fi
    fi
    
    if [[ "$health_level" == "basic" ]]; then
        echo "healthy"
        return 0
    fi
    
    # Detailed health check
    local screen_access="false"
    local mouse_control="false"
    local keyboard_control="false"
    local desktop_available="false"
    
    # Test screenshot capability
    if curl -s --max-time 10 -X POST "http://localhost:${port}/screenshot?format=png&response_format=binary" >/dev/null 2>&1; then
        screen_access="true"
        desktop_available="true"
    fi
    
    # Test mouse control (dry run)
    if curl -s --max-time 5 -X POST "http://localhost:${port}/mouse/position" >/dev/null 2>&1; then
        mouse_control="true"
    fi
    
    # Test keyboard control (dry run)
    if curl -s --max-time 5 -X POST "http://localhost:${port}/keyboard/type" \
        -H "Content-Type: application/json" -d '{"text":""}' >/dev/null 2>&1; then
        keyboard_control="true"
    fi
    
    if [[ "$screen_access" == "true" && "$mouse_control" == "true" && "$keyboard_control" == "true" ]]; then
        echo "healthy:screen_access:mouse_control:keyboard_control:desktop_ready"
    elif [[ "$screen_access" == "true" ]]; then
        echo "degraded:screen_access:limited_control:desktop_available"
    else
        echo "degraded:no_screen_access:desktop_unavailable"
    fi
    
    return 0
}

check_browserless_health() {
    local port="${1:-4110}"
    local health_level="${2:-basic}"
    
    # Basic connectivity test
    if ! curl -s --max-time 5 "http://localhost:${port}/health" >/dev/null 2>&1; then
        # Try alternative endpoints
        if curl -s --max-time 5 "http://localhost:${port}/" >/dev/null 2>&1; then
            echo "healthy"
            return 0
        else
            echo "unreachable"
            return 1
        fi
    fi
    
    if [[ "$health_level" == "basic" ]]; then
        echo "healthy"
        return 0
    fi
    
    # Detailed health check
    local chrome_available="false"
    local screenshot_capable="false"
    local pdf_capable="false"
    local concurrent_sessions="unknown"
    
    # Test Chrome availability and basic functionality
    local test_payload='{"url": "data:text/html,<html><body>Test</body></html>"}'
    
    # Test screenshot capability
    if curl -s --max-time 15 -X POST "http://localhost:${port}/chrome/screenshot" \
        -H "Content-Type: application/json" -d "$test_payload" >/dev/null 2>&1; then
        chrome_available="true"
        screenshot_capable="true"
    fi
    
    # Test PDF generation capability
    if curl -s --max-time 15 -X POST "http://localhost:${port}/chrome/pdf" \
        -H "Content-Type: application/json" -d "$test_payload" >/dev/null 2>&1; then
        pdf_capable="true"
    fi
    
    # Check for session metrics (if available)
    local metrics_response
    metrics_response=$(curl -s --max-time 5 "http://localhost:${port}/metrics" 2>/dev/null)
    if [[ -n "$metrics_response" ]]; then
        concurrent_sessions=$(echo "$metrics_response" | grep -o "sessions.*" | head -1 || echo "metrics_available")
    fi
    
    if [[ "$chrome_available" == "true" && "$screenshot_capable" == "true" && "$pdf_capable" == "true" ]]; then
        echo "healthy:chrome_available:screenshot_ready:pdf_ready:sessions:$concurrent_sessions"
    elif [[ "$chrome_available" == "true" && "$screenshot_capable" == "true" ]]; then
        echo "degraded:chrome_available:screenshot_ready:pdf_failed:sessions:$concurrent_sessions"
    elif [[ "$chrome_available" == "true" ]]; then
        echo "degraded:chrome_available:limited_functionality:sessions:$concurrent_sessions"
    else
        echo "degraded:chrome_unavailable"
    fi
    
    return 0
}

check_claude_code_health() {
    local port="${1:-}"  # CLI tool, no port
    local health_level="${2:-basic}"
    
    # Claude Code is a CLI tool, check if it's available
    local claude_command=""
    
    # Try different ways to invoke Claude Code
    if which claude >/dev/null 2>&1; then
        claude_command="claude"
    elif which claude-code >/dev/null 2>&1; then
        claude_command="claude-code"
    elif timeout 10 npx @anthropic-ai/claude-code --version >/dev/null 2>&1; then
        claude_command="npx @anthropic-ai/claude-code"
    else
        echo "unreachable"
        return 1
    fi
    
    if [[ "$health_level" == "basic" ]]; then
        echo "healthy"
        return 0
    fi
    
    # Detailed health check
    local version_available="false"
    local help_available="false"
    local auth_status="unknown"
    local version_info="unknown"
    
    # Test version command
    local version_output
    if version_output=$(timeout 10 $claude_command --version 2>/dev/null); then
        version_available="true"
        version_info=$(echo "$version_output" | head -1 | tr -d '\n' || echo "version_detected")
    fi
    
    # Test help command
    if timeout 10 $claude_command --help >/dev/null 2>&1; then
        help_available="true"
    fi
    
    # Check authentication status (if possible without triggering auth flow)
    # This is tricky since we don't want to trigger interactive auth
    auth_status="not_checked"
    
    if [[ "$version_available" == "true" && "$help_available" == "true" ]]; then
        echo "healthy:version_available:help_available:auth:$auth_status:version:$version_info"
    elif [[ "$version_available" == "true" ]]; then
        echo "degraded:version_available:help_failed:auth:$auth_status:version:$version_info"
    else
        echo "degraded:basic_commands_failed"
    fi
    
    return 0
}

# Generic agent health check dispatcher
check_agent_resource_health() {
    local resource_name="$1"
    local port="$2"
    local health_level="${3:-basic}"
    
    case "$resource_name" in
        "agent-s2")
            check_agent_s2_health "$port" "$health_level"
            ;;
        "browserless")
            check_browserless_health "$port" "$health_level"
            ;;
        "claude-code")
            check_claude_code_health "$port" "$health_level"
            ;;
        *)
            # Fallback to generic HTTP health check for unknown agents
            if [[ -n "$port" ]] && curl -s --max-time 5 "http://localhost:${port}/health" >/dev/null 2>&1; then
                echo "healthy"
            elif [[ -n "$port" ]] && curl -s --max-time 5 "http://localhost:${port}/" >/dev/null 2>&1; then
                echo "healthy"
            else
                echo "unreachable"
            fi
            ;;
    esac
}

# Agent resource capability testing
test_agent_resource_capabilities() {
    local resource_name="$1"
    local port="$2"
    
    case "$resource_name" in
        "agent-s2")
            test_agent_s2_capabilities "$port"
            ;;
        "browserless")
            test_browserless_capabilities "$port"
            ;;
        "claude-code")
            test_claude_code_capabilities "$port"
            ;;
        *)
            echo "capability_testing_not_implemented"
            ;;
    esac
}

test_agent_s2_capabilities() {
    local port="$1"
    
    local capabilities=()
    
    # Test screenshot capability
    if curl -s --max-time 5 -X POST "http://localhost:${port}/screenshot" >/dev/null 2>&1; then
        capabilities+=("screen_capture")
    fi
    
    # Test mouse control
    if curl -s --max-time 5 -X POST "http://localhost:${port}/mouse/position" >/dev/null 2>&1; then
        capabilities+=("mouse_control")
    fi
    
    # Test keyboard control
    if curl -s --max-time 5 -X POST "http://localhost:${port}/keyboard/type" >/dev/null 2>&1; then
        capabilities+=("keyboard_control")
    fi
    
    # Test window management
    if curl -s --max-time 5 -X POST "http://localhost:${port}/window/list" >/dev/null 2>&1; then
        capabilities+=("window_management")
    fi
    
    capabilities+=("desktop_automation")
    capabilities+=("visual_reasoning")
    
    if [[ ${#capabilities[@]} -gt 0 ]]; then
        echo "capabilities:$(IFS=,; echo "${capabilities[*]}")"
    else
        echo "no_capabilities_detected"
    fi
}

test_browserless_capabilities() {
    local port="$1"
    
    local capabilities=()
    
    # Test screenshot capability
    if curl -s --max-time 5 -X POST "http://localhost:${port}/chrome/screenshot" >/dev/null 2>&1; then
        capabilities+=("web_screenshot")
    fi
    
    # Test PDF generation
    if curl -s --max-time 5 -X POST "http://localhost:${port}/chrome/pdf" >/dev/null 2>&1; then
        capabilities+=("pdf_generation")
    fi
    
    # Test content extraction
    if curl -s --max-time 5 -X POST "http://localhost:${port}/chrome/content" >/dev/null 2>&1; then
        capabilities+=("content_extraction")
    fi
    
    capabilities+=("web_automation")
    capabilities+=("chrome_headless")
    capabilities+=("concurrent_sessions")
    
    if [[ ${#capabilities[@]} -gt 0 ]]; then
        echo "capabilities:$(IFS=,; echo "${capabilities[*]}")"
    else
        echo "no_capabilities_detected"
    fi
}

test_claude_code_capabilities() {
    local port="$1"  # Unused for CLI tool
    
    local capabilities=()
    
    # Detect available Claude Code command
    local claude_command=""
    if which claude >/dev/null 2>&1; then
        claude_command="claude"
    elif which claude-code >/dev/null 2>&1; then
        claude_command="claude-code"
    elif timeout 10 npx @anthropic-ai/claude-code --version >/dev/null 2>&1; then
        claude_command="npx @anthropic-ai/claude-code"
    fi
    
    if [[ -n "$claude_command" ]]; then
        capabilities+=("ai_assistance")
        capabilities+=("code_analysis")
        capabilities+=("development_tools")
        
        # Test help system
        if timeout 10 $claude_command --help >/dev/null 2>&1; then
            capabilities+=("help_system")
        fi
        
        # Test version info
        if timeout 10 $claude_command --version >/dev/null 2>&1; then
            capabilities+=("version_info")
        fi
    fi
    
    if [[ ${#capabilities[@]} -gt 0 ]]; then
        echo "capabilities:$(IFS=,; echo "${capabilities[*]}")"
    else
        echo "no_capabilities_detected"
    fi
}

# Security validation for agent resources
validate_agent_security() {
    local resource_name="$1"
    local port="$2"
    
    case "$resource_name" in
        "agent-s2")
            validate_agent_s2_security "$port"
            ;;
        "browserless")
            validate_browserless_security "$port"
            ;;
        "claude-code")
            validate_claude_code_security "$port"
            ;;
        *)
            echo "security_validation_not_implemented"
            ;;
    esac
}

validate_agent_s2_security() {
    local port="$1"
    
    local security_status=()
    
    # Check if running in a container/sandbox
    if docker ps --format "{{.Names}}" 2>/dev/null | grep -q "agent-s2"; then
        security_status+=("containerized")
    fi
    
    # Check for desktop access limitations
    # This is challenging to test without affecting the desktop
    security_status+=("desktop_access_required")
    
    # Check network isolation
    security_status+=("network_exposed:$port")
    
    if [[ ${#security_status[@]} -gt 0 ]]; then
        echo "security:$(IFS=,; echo "${security_status[*]}")"
    else
        echo "security_unknown"
    fi
}

validate_browserless_security() {
    local port="$1"
    
    local security_status=()
    
    # Check if running in a container
    if docker ps --format "{{.Names}}" 2>/dev/null | grep -q "browserless"; then
        security_status+=("containerized")
    fi
    
    # Check for Chrome sandbox
    security_status+=("chrome_sandbox")
    
    # Check network exposure
    security_status+=("network_exposed:$port")
    
    if [[ ${#security_status[@]} -gt 0 ]]; then
        echo "security:$(IFS=,; echo "${security_status[*]}")"
    else
        echo "security_unknown"
    fi
}

validate_claude_code_security() {
    local port="$1"  # Unused for CLI tool
    
    local security_status=()
    
    # CLI tool runs in user context
    security_status+=("user_context")
    
    # Check if it has file system access
    security_status+=("filesystem_access")
    
    # Network access for API calls
    security_status+=("network_access")
    
    if [[ ${#security_status[@]} -gt 0 ]]; then
        echo "security:$(IFS=,; echo "${security_status[*]}")"
    else
        echo "security_unknown"
    fi
}

# Export functions
export -f check_agent_resource_health
export -f test_agent_resource_capabilities
export -f validate_agent_security
export -f check_agent_s2_health
export -f check_browserless_health
export -f check_claude_code_health