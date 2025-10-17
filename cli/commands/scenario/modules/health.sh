#!/usr/bin/env bash
# Scenario Health Check Module
# Handles health validation, diagnostics, and failure analysis

set -euo pipefail

# Collect comprehensive diagnostic data (returns JSON)
scenario::health::collect_all_diagnostic_data() {
    local scenario_name="$1"
    local response="$2"
    local status="$3"
    
    local ui_port api_port
    ui_port=$(echo "$response" | jq -r '.data.allocated_ports.UI_PORT // null' 2>/dev/null)
    api_port=$(echo "$response" | jq -r '.data.allocated_ports.API_PORT // null' 2>/dev/null)
    
    local health_data='{"health_checks": {}, "log_analysis": {}, "responsiveness": {}}'
    
    # Collect health check data for running scenarios
    if [[ "$status" == "running" ]]; then
        health_data=$(scenario::health::collect_health_check_data "$scenario_name" "$ui_port" "$api_port" "$health_data")
        health_data=$(scenario::health::collect_running_diagnostics "$scenario_name" "$health_data")
    fi
    
    # Collect failure diagnostics for stopped scenarios
    if [[ "$status" == "stopped" ]]; then
        health_data=$(scenario::health::collect_failure_diagnostics "$scenario_name" "$health_data")
    fi
    
    echo "$health_data"
}

# Collect health check data for running scenarios
scenario::health::collect_health_check_data() {
    local scenario_name="$1"
    local ui_port="$2"
    local api_port="$3"
    local health_data="$4"
    
    # Check UI health if UI port exists
    if [[ "$ui_port" != "null" ]] && [[ -n "$ui_port" ]]; then
        local ui_health_data
        ui_health_data=$(scenario::health::collect_ui_health_data "$scenario_name" "$ui_port")
        health_data=$(echo "$health_data" | jq --argjson ui_data "$ui_health_data" '.health_checks.ui = $ui_data')
    fi
    
    # Check API health if API port exists
    if [[ "$api_port" != "null" ]] && [[ -n "$api_port" ]]; then
        local api_health_data
        api_health_data=$(scenario::health::collect_api_health_data "$scenario_name" "$api_port")
        health_data=$(echo "$health_data" | jq --argjson api_data "$api_health_data" '.health_checks.api = $api_data')
    fi
    
    echo "$health_data"
}

# Collect running scenario diagnostics (log analysis, responsiveness, etc.)
scenario::health::collect_running_diagnostics() {
    local scenario_name="$1"
    local health_data="$2"
    
    # Collect log analysis data
    local log_analysis_data
    log_analysis_data=$(scenario::health::collect_log_analysis_data "$scenario_name")
    health_data=$(echo "$health_data" | jq --argjson log_data "$log_analysis_data" '.log_analysis = $log_data')
    
    # Collect responsiveness data
    local responsiveness_data
    responsiveness_data=$(scenario::health::collect_responsiveness_data "$scenario_name")
    health_data=$(echo "$health_data" | jq --argjson resp_data "$responsiveness_data" '.responsiveness = $resp_data')
    
    echo "$health_data"
}

# Enhanced Health Checks with Schema Validation (legacy function for text output)
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
    
    # Enhanced diagnostics for running scenarios with issues
    if [[ "$health_issues_found" == "true" ]]; then
        scenario::health::show_diagnostics "$scenario_name"
        scenario::health::show_running_diagnostics "$scenario_name"
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

# ============================================================================
# DATA COLLECTION FUNCTIONS (Return JSON instead of echoing output)
# ============================================================================

# Collect UI health data (returns JSON)
scenario::health::collect_ui_health_data() {
    local scenario_name="$1"
    local ui_port="$2"
    
    local ui_health
    ui_health=$(curl -s --max-time 3 "http://localhost:${ui_port}/health" 2>/dev/null)
    
    local result='{
        "available": false,
        "port": null,
        "status": "unknown",
        "api_connectivity": null,
        "schema_valid": false,
        "response_time": null
    }'
    
    if [[ -n "$ui_health" ]]; then
        local start_time=$(date +%s.%N)
        ui_health=$(curl -s --max-time 3 "http://localhost:${ui_port}/health" 2>/dev/null)
        local end_time=$(date +%s.%N)
        local response_time=$(echo "$end_time - $start_time" | bc -l 2>/dev/null || echo "0")
        
        result=$(echo "$result" | jq \
            --arg port "$ui_port" \
            --arg response_time "$response_time" \
            '.available = true | .port = ($port | tonumber) | .response_time = ($response_time | tonumber)')
        
        # Validate against schema
        if validate_health_response "$ui_health" "ui" >/dev/null 2>&1; then
            local ui_status api_connected api_error api_url latency
            ui_status=$(echo "$ui_health" | jq -r '.status // "unknown"' 2>/dev/null)
            api_connected=$(echo "$ui_health" | jq -r '.api_connectivity.connected // false' 2>/dev/null)
            api_error=$(echo "$ui_health" | jq -r '.api_connectivity.error // null' 2>/dev/null)
            api_url=$(echo "$ui_health" | jq -r '.api_connectivity.api_url // "unknown"' 2>/dev/null)
            latency=$(echo "$ui_health" | jq -r '.api_connectivity.latency_ms // null' 2>/dev/null)
            
            result=$(echo "$result" | jq \
                --arg status "$ui_status" \
                --arg api_connected "$api_connected" \
                --arg api_error "$api_error" \
                --arg api_url "$api_url" \
                --arg latency "$latency" \
                '.schema_valid = true | 
                 .status = $status | 
                 .api_connectivity = {
                     "connected": ($api_connected == "true"),
                     "api_url": $api_url,
                     "error": (if $api_error == "null" then null else $api_error end),
                     "latency_ms": (if $latency == "null" then null else ($latency | tonumber) end)
                 }')
        fi
    fi
    
    echo "$result"
}

# Collect API health data (returns JSON)
scenario::health::collect_api_health_data() {
    local scenario_name="$1"
    local api_port="$2"
    
    local api_health
    api_health=$(curl -s --max-time 3 "http://localhost:${api_port}/health" 2>/dev/null)
    
    local result='{
        "available": false,
        "port": null,
        "status": "unknown",
        "dependencies": {},
        "schema_valid": false,
        "response_time": null
    }'
    
    if [[ -n "$api_health" ]]; then
        local start_time=$(date +%s.%N)
        api_health=$(curl -s --max-time 3 "http://localhost:${api_port}/health" 2>/dev/null)
        local end_time=$(date +%s.%N)
        local response_time=$(echo "$end_time - $start_time" | bc -l 2>/dev/null || echo "0")
        
        result=$(echo "$result" | jq \
            --arg port "$api_port" \
            --arg response_time "$response_time" \
            '.available = true | .port = ($port | tonumber) | .response_time = ($response_time | tonumber)')
        
        # Validate against schema
        if validate_health_response "$api_health" "api" >/dev/null 2>&1; then
            local api_status db_connected
            api_status=$(echo "$api_health" | jq -r '.status // "unknown"' 2>/dev/null)
            db_connected=$(echo "$api_health" | jq -r '.dependencies.database.connected // null' 2>/dev/null)
            
            result=$(echo "$result" | jq \
                --arg status "$api_status" \
                --arg db_connected "$db_connected" \
                '.schema_valid = true | 
                 .status = $status |
                 .dependencies.database = {
                     "connected": (if $db_connected == "null" then null else ($db_connected == "true") end)
                 }')
        fi
    fi
    
    echo "$result"
}

# Collect log analysis data for running scenarios (returns JSON)
scenario::health::collect_log_analysis_data() {
    local scenario_name="$1"
    
    local result='{
        "recent_warnings": [],
        "resource_issues": [],
        "performance_warnings": []
    }'
    
    # Check API logs for warnings
    local recent_api_logs
    recent_api_logs=$(vrooli scenario logs "$scenario_name" --step start-api 2>/dev/null | tail -20)
    if [[ -n "$recent_api_logs" ]]; then
        # Look for warnings
        if echo "$recent_api_logs" | grep -qE "(WARNING|WARN|Error:|error:|timeout|failed|exception)"; then
            local warning=$(echo "$recent_api_logs" | grep -E "(WARNING|WARN|Error:|error:)" | tail -1 | cut -c1-100)
            result=$(echo "$result" | jq \
                --arg component "api" \
                --arg warning "$warning" \
                '.recent_warnings += [{
                    "component": $component,
                    "message": $warning,
                    "timestamp": "recent"
                }]')
        fi
        
        # Look for resource issues
        if echo "$recent_api_logs" | grep -qE "(out of memory|OOM|disk.*full|no space|resource.*exhausted)"; then
            local issue=$(echo "$recent_api_logs" | grep -E "(out of memory|OOM|disk.*full|no space)" | tail -1 | cut -c1-100)
            result=$(echo "$result" | jq \
                --arg component "api" \
                --arg issue "$issue" \
                '.resource_issues += [{
                    "component": $component,
                    "type": "resource_exhaustion",
                    "message": $issue
                }]')
        fi
        
        # Look for connection pool issues
        if echo "$recent_api_logs" | grep -qE "(connection.*pool.*exhausted|too many.*connections|rate.*limit)"; then
            local issue=$(echo "$recent_api_logs" | grep -E "(connection.*pool|rate.*limit)" | tail -1 | cut -c1-100)
            result=$(echo "$result" | jq \
                --arg component "api" \
                --arg issue "$issue" \
                '.performance_warnings += [{
                    "component": $component,
                    "type": "connection_performance",
                    "message": $issue
                }]')
        fi
    fi
    
    # Check UI logs for warnings
    local recent_ui_logs
    recent_ui_logs=$(vrooli scenario logs "$scenario_name" --step start-ui 2>/dev/null | tail -20)
    if [[ -n "$recent_ui_logs" ]]; then
        if echo "$recent_ui_logs" | grep -qE "(WARNING|WARN|Error:|error:|failed|crash)"; then
            local warning=$(echo "$recent_ui_logs" | grep -E "(WARNING|WARN|Error:|error:)" | tail -1 | cut -c1-100)
            result=$(echo "$result" | jq \
                --arg component "ui" \
                --arg warning "$warning" \
                '.recent_warnings += [{
                    "component": $component,
                    "message": $warning,
                    "timestamp": "recent"
                }]')
        fi
    fi
    
    echo "$result"
}

# Collect responsiveness data (returns JSON)
scenario::health::collect_responsiveness_data() {
    local scenario_name="$1"
    
    local result='{
        "api": {"available": false, "response_time": null, "timeout": false},
        "ui": {"available": false, "response_time": null, "timeout": false}
    }'
    
    # Check API responsiveness
    local api_port
    api_port=$(scenario::ports::get "$scenario_name" "API_PORT" 2>/dev/null)
    if [[ -n "$api_port" ]]; then
        local start_time=$(date +%s.%N)
        local api_response
        api_response=$(curl -w "%{http_code}" -s --max-time 10 "http://localhost:${api_port}/health" -o /dev/null 2>/dev/null)
        local end_time=$(date +%s.%N)
        
        if [[ "$api_response" =~ ^[0-9]+$ ]]; then
            local response_time=$(echo "$end_time - $start_time" | bc -l 2>/dev/null || echo "0")
            local timeout_flag=$(echo "$response_time > 10.0" | bc -l 2>/dev/null || echo "0")
            
            result=$(echo "$result" | jq \
                --arg response_time "$response_time" \
                --arg timeout_flag "$timeout_flag" \
                '.api.available = true |
                 .api.response_time = ($response_time | tonumber) |
                 .api.timeout = ($timeout_flag == "1")')
        else
            result=$(echo "$result" | jq '.api.timeout = true')
        fi
    fi
    
    # Check UI responsiveness
    local ui_port
    ui_port=$(scenario::ports::get "$scenario_name" "UI_PORT" 2>/dev/null)
    if [[ -n "$ui_port" ]]; then
        local start_time=$(date +%s.%N)
        local ui_response
        ui_response=$(curl -w "%{http_code}" -s --max-time 10 "http://localhost:${ui_port}/health" -o /dev/null 2>/dev/null)
        local end_time=$(date +%s.%N)
        
        if [[ "$ui_response" =~ ^[0-9]+$ ]]; then
            local response_time=$(echo "$end_time - $start_time" | bc -l 2>/dev/null || echo "0")
            local timeout_flag=$(echo "$response_time > 10.0" | bc -l 2>/dev/null || echo "0")
            
            result=$(echo "$result" | jq \
                --arg response_time "$response_time" \
                --arg timeout_flag "$timeout_flag" \
                '.ui.available = true |
                 .ui.response_time = ($response_time | tonumber) |
                 .ui.timeout = ($timeout_flag == "1")')
        else
            result=$(echo "$result" | jq '.ui.timeout = true')
        fi
    fi
    
    echo "$result"
}

# Collect failure diagnostics for stopped scenarios (returns JSON)
scenario::health::collect_failure_diagnostics() {
    local scenario_name="$1"
    local health_data="$2"
    
    local failure_data='{
        "api_failures": [],
        "ui_failures": [],
        "general_recommendations": []
    }'
    
    # Check API startup logs for common failure patterns
    local api_logs
    api_logs=$(vrooli scenario logs "$scenario_name" --step start-api 2>/dev/null | tail -15)
    if [[ -n "$api_logs" ]]; then
        failure_data=$(scenario::health::analyze_api_failure_patterns "$api_logs" "$failure_data")
    fi
    
    # Check UI startup logs for common failure patterns
    local ui_logs
    ui_logs=$(vrooli scenario logs "$scenario_name" --step start-ui 2>/dev/null | tail -15)
    if [[ -n "$ui_logs" ]]; then
        failure_data=$(scenario::health::analyze_ui_failure_patterns "$ui_logs" "$failure_data")
    fi
    
    # Add general recommendations if no specific failures found
    local has_failures=$(echo "$failure_data" | jq '.api_failures | length > 0 or .ui_failures | length > 0')
    if [[ "$has_failures" == "false" ]]; then
        failure_data=$(echo "$failure_data" | jq '.general_recommendations += [
            "Check detailed logs: vrooli scenario logs '"$scenario_name"'",
            "Verify dependencies: make sure required resources are running",
            "Clean restart: vrooli scenario stop '"$scenario_name"' && vrooli scenario start '"$scenario_name"'"
        ]')
    fi
    
    echo "$health_data" | jq --argjson failure_data "$failure_data" '.log_analysis = $failure_data'
}

# Analyze API failure patterns (returns updated JSON)
scenario::health::analyze_api_failure_patterns() {
    local api_logs="$1"
    local failure_data="$2"
    
    if echo "$api_logs" | grep -q "EADDRINUSE"; then
        local port=$(echo "$api_logs" | grep -oE "port [0-9]+" | head -1 | cut -d' ' -f2)
        [[ -z "$port" ]] && port=$(echo "$api_logs" | grep -oE ":::[0-9]+" | head -1 | cut -d':' -f4)
        failure_data=$(echo "$failure_data" | jq \
            --arg port "${port:-unknown}" \
            '.api_failures += [{
                "type": "port_conflict",
                "message": "Port " + $port + " conflict - check for hardcoded ports in API",
                "recommendation": "Ensure API uses API_PORT environment variable"
            }]')
    elif echo "$api_logs" | grep -qE "(API_PORT.*required|API_PORT.*not set|environment variable.*required|required.*not set)"; then
        failure_data=$(echo "$failure_data" | jq '.api_failures += [{
            "type": "missing_env_var",
            "message": "Missing API_PORT environment variable",
            "recommendation": "Check if scenario is started via lifecycle system"
        }]')
    elif echo "$api_logs" | grep -qE "(database.*connection.*failed|connect.*database.*failed|postgres.*connection.*error)"; then
        failure_data=$(echo "$failure_data" | jq '.api_failures += [{
            "type": "database_connection",
            "message": "Database connection failed",
            "recommendation": "Ensure PostgreSQL resource is running: resource-postgres status"
        }]')
    elif echo "$api_logs" | grep -qE "(go build.*failed|build.*error|compilation.*error)"; then
        failure_data=$(echo "$failure_data" | jq '.api_failures += [{
            "type": "build_error",
            "message": "Build/compilation error in API",
            "recommendation": "Check Go syntax and dependencies in api/ directory"
        }]')
    elif echo "$api_logs" | grep -qE "(Error:|error:|panic:|fatal:|crash)"; then
        local error_line=$(echo "$api_logs" | grep -E "(Error:|error:|panic:|fatal:|crash)" | tail -1 | sed 's/^[[:space:]]*//' | cut -c1-70)
        failure_data=$(echo "$failure_data" | jq --arg error "$error_line" '.api_failures += [{
            "type": "runtime_error",
            "message": $error,
            "recommendation": "Check API logs for detailed error context"
        }]')
    fi
    
    echo "$failure_data"
}

# Analyze UI failure patterns (returns updated JSON)
scenario::health::analyze_ui_failure_patterns() {
    local ui_logs="$1"
    local failure_data="$2"
    
    if echo "$ui_logs" | grep -q "EADDRINUSE"; then
        local port
        port=$(echo "$ui_logs" | grep -oE "port [0-9]+" | head -1 | cut -d' ' -f2) || true
        [[ -z "$port" ]] && port=$(echo "$ui_logs" | grep -oE ":::[0-9]+" | head -1 | cut -d':' -f4) || true
        failure_data=$(echo "$failure_data" | jq \
            --arg port "${port:-3000}" \
            '.ui_failures += [{
                "type": "port_conflict",
                "message": "Port " + $port + " conflict - check PORT vs UI_PORT usage!",
                "recommendation": "UI should use UI_PORT environment variable, not generic PORT"
            }]')
    elif echo "$ui_logs" | grep -qE "(UI_PORT.*required|UI_PORT.*not set|PORT.*required|environment variable.*required)"; then
        failure_data=$(echo "$failure_data" | jq '.ui_failures += [{
            "type": "missing_env_var",
            "message": "Missing UI_PORT environment variable",
            "recommendation": "Check if UI server uses UI_PORT (not generic PORT)"
        }]')
    elif echo "$ui_logs" | grep -qE "(npm.*not found|node.*not found|package.*not found)"; then
        failure_data=$(echo "$failure_data" | jq '.ui_failures += [{
            "type": "dependency_missing",
            "message": "Node.js/npm dependencies missing",
            "recommendation": "Run: cd ui && npm install"
        }]')
    elif echo "$ui_logs" | grep -qE "(Cannot find module|Module not found|import.*failed)"; then
        local module=$(echo "$ui_logs" | grep -oE "(Cannot find module|Module not found).*" | head -1 | cut -c1-50)
        failure_data=$(echo "$failure_data" | jq --arg module "$module" '.ui_failures += [{
            "type": "module_error",
            "message": $module,
            "recommendation": "Check UI dependencies and imports"
        }]')
    elif echo "$ui_logs" | grep -qE "(Error:|error:|crash|fatal)"; then
        local error_line=$(echo "$ui_logs" | grep -E "(Error:|error:|crash|fatal)" | head -1 | sed 's/^[[:space:]]*//' | cut -c1-70)
        failure_data=$(echo "$failure_data" | jq --arg error "$error_line" '.ui_failures += [{
            "type": "runtime_error",
            "message": $error,
            "recommendation": "Check UI logs for detailed error context"
        }]')
    fi
    
    echo "$failure_data"
}

# Display running scenario diagnostics (for text output)
scenario::health::show_running_diagnostics() {
    local scenario_name="$1"
    
    # Collect the diagnostic data
    local diagnostic_data
    diagnostic_data=$(scenario::health::collect_log_analysis_data "$scenario_name")
    
    # Check if there are any warnings to show
    local has_warnings=$(echo "$diagnostic_data" | jq '.recent_warnings | length > 0')
    local has_resource_issues=$(echo "$diagnostic_data" | jq '.resource_issues | length > 0')
    local has_performance_warnings=$(echo "$diagnostic_data" | jq '.performance_warnings | length > 0')
    
    if [[ "$has_warnings" == "true" ]] || [[ "$has_resource_issues" == "true" ]] || [[ "$has_performance_warnings" == "true" ]]; then
        echo ""
        echo "🔍 Running Scenario Analysis:"
        
        # Show recent warnings
        if [[ "$has_warnings" == "true" ]]; then
            echo "  Recent Warnings:"
            echo "$diagnostic_data" | jq -r '.recent_warnings[] | "    ⚠️  " + .component + ": " + .message'
        fi
        
        # Show resource issues
        if [[ "$has_resource_issues" == "true" ]]; then
            echo "  Resource Issues:"
            echo "$diagnostic_data" | jq -r '.resource_issues[] | "    🔴 " + .component + ": " + .message'
        fi
        
        # Show performance warnings
        if [[ "$has_performance_warnings" == "true" ]]; then
            echo "  Performance Warnings:"
            echo "$diagnostic_data" | jq -r '.performance_warnings[] | "    🟡 " + .component + ": " + .message'
        fi
        
        echo "  💡 Check detailed logs: vrooli scenario logs $scenario_name"
    fi
}