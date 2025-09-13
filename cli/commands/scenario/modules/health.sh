#!/usr/bin/env bash
# Scenario Health Check Module
# Handles health validation, diagnostics, and failure analysis

set -euo pipefail

# Enhanced Health Checks with Schema Validation
scenario::health::check_scenario() {
    local scenario_name="$1"
    local response="$2"
    local status="$3"
    
    # Only perform health checks for running scenarios
    if [[ "$status" != "running" ]] || [[ ! -f "$HEALTH_VALIDATOR" ]]; then
        return 0
    fi
    
    local ui_port api_port
    ui_port=$(echo "$response" | jq -r '.data.allocated_ports.UI_PORT // null' 2>/dev/null)
    api_port=$(echo "$response" | jq -r '.data.allocated_ports.API_PORT // null' 2>/dev/null)
    
    local health_issues_found=false
    
    # Check UI health if UI port exists
    if [[ "$ui_port" != "null" ]] && [[ -n "$ui_port" ]]; then
        health_issues_found=$(scenario::health::check_ui_health "$scenario_name" "$ui_port" "$health_issues_found")
    fi
    
    # Check API health if API port exists
    if [[ "$api_port" != "null" ]] && [[ -n "$api_port" ]]; then
        health_issues_found=$(scenario::health::check_api_health "$scenario_name" "$api_port" "$health_issues_found" "$ui_port")
    fi
    
    # Provide diagnostics if health issues found
    if [[ "$health_issues_found" == "true" ]]; then
        scenario::health::show_diagnostics "$scenario_name"
    fi
}

# Check UI health with schema validation
scenario::health::check_ui_health() {
    local scenario_name="$1"
    local ui_port="$2"
    local health_issues_found="$3"
    
    local ui_health
    ui_health=$(curl -s --max-time 3 "http://localhost:${ui_port}/health" 2>/dev/null)
    
    if [[ -n "$ui_health" ]]; then
        echo ""
        echo "Health Checks:"
        echo "  UI Service (port $ui_port):"
        
        # Validate against schema
        if validate_health_response "$ui_health" "ui" >/dev/null 2>&1; then
            local ui_status api_connected api_error api_url
            ui_status=$(echo "$ui_health" | jq -r '.status // "unknown"' 2>/dev/null)
            api_connected=$(echo "$ui_health" | jq -r '.api_connectivity.connected // false' 2>/dev/null)
            api_error=$(echo "$ui_health" | jq -r '.api_connectivity.error // null' 2>/dev/null)
            api_url=$(echo "$ui_health" | jq -r '.api_connectivity.api_url // "unknown"' 2>/dev/null)
            
            # Show UI status
            case "$ui_status" in
                healthy)
                    echo "    Status:      ✅ $ui_status"
                    ;;
                degraded)
                    echo "    Status:      ⚠️  $ui_status"
                    health_issues_found=true
                    ;;
                unhealthy)
                    echo "    Status:      ❌ $ui_status"
                    health_issues_found=true
                    ;;
                *)
                    echo "    Status:      ❓ $ui_status"
                    ;;
            esac
            
            # Show API connectivity
            if [[ "$api_connected" == "true" ]]; then
                echo "    API Link:    ✅ Connected to $api_url"
                local latency
                latency=$(echo "$ui_health" | jq -r '.api_connectivity.latency_ms // null' 2>/dev/null)
                [[ "$latency" != "null" ]] && echo "    Latency:     ${latency}ms"
            elif [[ "$api_connected" == "false" ]]; then
                echo "    API Link:    ❌ DISCONNECTED from $api_url"
                health_issues_found=true
                if [[ "$api_error" != "null" ]] && [[ -n "$api_error" ]]; then
                    echo "    Error:       $api_error"
                fi
            fi
        else
            echo "    Status:      ⚠️  Invalid health response (not compliant with schema)"
            echo "    💡 UI health endpoint must include 'api_connectivity' field"
            echo "       See: ${SCENARIO_CMD_DIR}/schemas/health-ui.schema.json"
            health_issues_found=true
        fi
    else
        echo ""
        echo "Health Checks:"
        echo "  UI Service:    ⚠️  Health endpoint not responding"
        health_issues_found=true
    fi
    
    echo "$health_issues_found"
}

# Check API health with schema validation
scenario::health::check_api_health() {
    local scenario_name="$1"
    local api_port="$2"
    local health_issues_found="$3"
    local ui_port="$4"
    
    local api_health
    api_health=$(curl -s --max-time 3 "http://localhost:${api_port}/health" 2>/dev/null)
    
    if [[ -n "$api_health" ]]; then
        if [[ "$ui_port" == "null" ]]; then
            echo ""
            echo "Health Checks:"
        fi
        echo "  API Service (port $api_port):"
        
        # Validate against schema
        if validate_health_response "$api_health" "api" >/dev/null 2>&1; then
            local api_status
            api_status=$(echo "$api_health" | jq -r '.status // "unknown"' 2>/dev/null)
            
            case "$api_status" in
                healthy)
                    echo "    Status:      ✅ $api_status"
                    ;;
                degraded)
                    echo "    Status:      ⚠️  $api_status"
                    health_issues_found=true
                    ;;
                unhealthy)
                    echo "    Status:      ❌ $api_status"
                    health_issues_found=true
                    ;;
                *)
                    echo "    Status:      ❓ $api_status"
                    ;;
            esac
            
            # Show dependencies if available
            local db_connected
            db_connected=$(echo "$api_health" | jq -r '.dependencies.database.connected // null' 2>/dev/null)
            if [[ "$db_connected" != "null" ]]; then
                if [[ "$db_connected" == "true" ]]; then
                    echo "    Database:    ✅ Connected"
                else
                    echo "    Database:    ❌ Disconnected"
                    health_issues_found=true
                fi
            fi
        else
            echo "    Status:      ⚠️  Invalid health response"
            health_issues_found=true
        fi
    else
        if [[ "$ui_port" == "null" ]]; then
            echo ""
            echo "Health Checks:"
        fi
        echo "  API Service:   ⚠️  Health endpoint not responding"
        health_issues_found=true
    fi
    
    echo "$health_issues_found"
}

# Show health diagnostics
scenario::health::show_diagnostics() {
    local scenario_name="$1"
    
    echo ""
    echo "💡 Diagnostics:"
    if [[ "${api_connected:-}" == "false" ]]; then
        echo "  • UI cannot reach API - check API logs and configuration"
        echo "  • Verify API_URL environment variable in UI matches API port"
    fi
    echo "  • Check logs: vrooli scenario logs $scenario_name"
    echo "  • Validate health endpoints comply with schemas in:"
    echo "    ${SCENARIO_CMD_DIR}/schemas/"
}

# Automatic failure diagnosis for stopped scenarios
scenario::health::diagnose_failure() {
    local scenario_name="$1"
    local json_output="$2"
    
    if [[ "$json_output" == "true" ]]; then
        return 0
    fi
    
    local failure_found=false
    
    # Check API startup logs for common failure patterns
    local api_logs
    api_logs=$(vrooli scenario logs "$scenario_name" --step start-api 2>/dev/null | tail -15)
    if [[ -n "$api_logs" ]]; then
        failure_found=$(scenario::health::diagnose_api_failure "$api_logs" "$failure_found")
    fi
    
    # Check UI startup logs for common failure patterns
    local ui_logs
    ui_logs=$(vrooli scenario logs "$scenario_name" --step start-ui 2>/dev/null | tail -15)
    if [[ -n "$ui_logs" ]]; then
        failure_found=$(scenario::health::diagnose_ui_failure "$ui_logs" "$failure_found")
    fi
    
    # Generic advice if no specific failure patterns found
    if [[ "$failure_found" == "false" ]]; then
        echo ""
        echo "💡 Troubleshooting tips:"
        echo "   • Check detailed logs: vrooli scenario logs $scenario_name"
        echo "   • Verify dependencies: make sure required resources are running"
        echo "   • Clean restart: vrooli scenario stop $scenario_name && vrooli scenario start $scenario_name"
    fi
}

# Diagnose API startup failures
scenario::health::diagnose_api_failure() {
    local api_logs="$1"
    local failure_found="$2"
    
    if echo "$api_logs" | grep -q "EADDRINUSE"; then
        local port=$(echo "$api_logs" | grep -oE "port [0-9]+" | head -1 | cut -d' ' -f2)
        [[ -z "$port" ]] && port=$(echo "$api_logs" | grep -oE ":::[0-9]+" | head -1 | cut -d':' -f4)
        echo ""
        echo "🔍 API Startup Failure:"
        echo "   Port ${port:-'unknown'} conflict - check for hardcoded ports in API"
        echo "   💡 Ensure API uses API_PORT environment variable"
        failure_found=true
    elif echo "$api_logs" | grep -qE "(API_PORT.*required|API_PORT.*not set|environment variable.*required|required.*not set)"; then
        echo ""
        echo "🔍 API Startup Failure:"
        echo "   Missing API_PORT environment variable"
        echo "   💡 Check if scenario is started via lifecycle system"
        failure_found=true
    elif echo "$api_logs" | grep -qE "(database.*connection.*failed|connect.*database.*failed|postgres.*connection.*error)"; then
        echo ""
        echo "🔍 API Startup Failure:"
        echo "   Database connection failed"
        echo "   💡 Ensure PostgreSQL resource is running: resource-postgres status"
        failure_found=true
    elif echo "$api_logs" | grep -qE "(go build.*failed|build.*error|compilation.*error)"; then
        echo ""
        echo "🔍 API Startup Failure:"
        echo "   Build/compilation error in API"
        echo "   💡 Check Go syntax and dependencies in api/ directory"
        failure_found=true
    elif echo "$api_logs" | grep -qE "(Error:|error:|panic:|fatal:|crash)"; then
        local error_line=$(echo "$api_logs" | grep -E "(Error:|error:|panic:|fatal:|crash)" | tail -1 | sed 's/^[[:space:]]*//' | cut -c1-70)
        echo ""
        echo "🔍 API Startup Failure:"
        echo "   $error_line"
        failure_found=true
    fi
    
    echo "$failure_found"
}

# Diagnose UI startup failures
scenario::health::diagnose_ui_failure() {
    local ui_logs="$1"
    local failure_found="$2"
    
    if echo "$ui_logs" | grep -q "EADDRINUSE"; then
        local port
        port=$(echo "$ui_logs" | grep -oE "port [0-9]+" | head -1 | cut -d' ' -f2) || true
        [[ -z "$port" ]] && port=$(echo "$ui_logs" | grep -oE ":::[0-9]+" | head -1 | cut -d':' -f4) || true
        echo ""
        echo "🔍 UI Startup Failure:"
        echo "   Port ${port:-'3000'} conflict - check PORT vs UI_PORT usage!"
        echo "   💡 UI should use UI_PORT environment variable, not generic PORT"
        failure_found=true
    elif echo "$ui_logs" | grep -qE "(UI_PORT.*required|UI_PORT.*not set|PORT.*required|environment variable.*required)"; then
        echo ""
        echo "🔍 UI Startup Failure:"
        echo "   Missing UI_PORT environment variable"
        echo "   💡 Check if UI server uses UI_PORT (not generic PORT)"
        failure_found=true
    elif echo "$ui_logs" | grep -qE "(npm.*not found|node.*not found|package.*not found)"; then
        echo ""
        echo "🔍 UI Startup Failure:"
        echo "   Node.js/npm dependencies missing"
        echo "   💡 Run: cd ui && npm install"
        failure_found=true
    elif echo "$ui_logs" | grep -qE "(Cannot find module|Module not found|import.*failed)"; then
        local module=$(echo "$ui_logs" | grep -oE "(Cannot find module|Module not found).*" | head -1 | cut -c1-50)
        echo ""
        echo "🔍 UI Startup Failure:"
        echo "   $module"
        echo "   💡 Check UI dependencies and imports"
        failure_found=true
    elif echo "$ui_logs" | grep -qE "(Error:|error:|crash|fatal)"; then
        local error_line=$(echo "$ui_logs" | grep -E "(Error:|error:|crash|fatal)" | head -1 | sed 's/^[[:space:]]*//' | cut -c1-70)
        echo ""
        echo "🔍 UI Startup Failure:"
        echo "   $error_line"
        failure_found=true
    fi
    
    echo "$failure_found"
}