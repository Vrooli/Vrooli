#!/bin/bash

# Wiki.js Status Functions

set -euo pipefail

# Source common functions and format utilities
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
WIKIJS_LIB_DIR="${APP_ROOT}/resources/wikijs/lib"

# Source var.sh first to set up environment variables
source "$WIKIJS_LIB_DIR/../../../../lib/utils/var.sh" 2>/dev/null || true

source "$WIKIJS_LIB_DIR/common.sh"
source "$WIKIJS_LIB_DIR/../../../../lib/utils/format.sh"

# Show Wiki.js status
status_wikijs() {
    local json_mode=false
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --json)
                json_mode=true
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Gather status information
    local installed=$(is_installed && echo "true" || echo "false")
    local running=$(is_running && echo "true" || echo "false")
    local healthy="false"
    local message=""
    local version="unknown"
    local port=$(get_wikijs_port)
    local api_endpoint=$(get_api_endpoint)
    local test_results=""
    
    if [[ "$installed" == "true" ]]; then
        if [[ "$running" == "true" ]]; then
            if check_health; then
                healthy="true"
                # Check if in setup mode
                if timeout 3 curl -s "http://localhost:${port}" 2>/dev/null | grep -q "Wiki.js Setup"; then
                    message="Wiki.js is running and healthy (setup mode - browse to http://localhost:${port} to complete setup)"
                else
                    message="Wiki.js is running and healthy"
                fi
                version=$(get_version)
                
                # Get test results if available
                local test_file="$WIKIJS_DATA_DIR/test_results.json"
                if [[ -f "$test_file" ]]; then
                    test_results=$(cat "$test_file" 2>/dev/null || echo "{}")
                fi
            else
                message="Wiki.js is running but not responding"
            fi
        else
            message="Wiki.js is installed but not running"
        fi
    else
        message="Wiki.js is not installed"
    fi
    
    # Format output
    if [[ "$json_mode" == true ]]; then
        # Build JSON response
        local json_output="{
            \"resource\": \"wikijs\",
            \"installed\": $installed,
            \"running\": $running,
            \"healthy\": $healthy,
            \"message\": \"$message\",
            \"version\": \"$version\",
            \"port\": $port,
            \"api_endpoint\": \"$api_endpoint\""
        
        if [[ -n "$test_results" ]]; then
            json_output="$json_output,
            \"test_results\": $test_results"
        fi
        
        json_output="$json_output
        }"
        
        echo "$json_output"
    else
        # Use format.sh for consistent output
        echo "[HEADER]  📚 Wiki.js Status"
        
        echo "
[INFO]    📊 Basic Status:"
        if [[ "$installed" == "true" ]]; then
            echo "[SUCCESS]    ✅ Installed: Yes"
        else
            echo "[ERROR]      ❌ Installed: No"
        fi
        
        if [[ "$running" == "true" ]]; then
            echo "[SUCCESS]    ✅ Running: Yes"
        else
            echo "[ERROR]      ❌ Running: No"
        fi
        
        if [[ "$healthy" == "true" ]]; then
            echo "[SUCCESS]    ✅ Health: $message"
        else
            echo "[ERROR]      ❌ Health: $message"
        fi
        
        if [[ "$installed" == "true" ]] && [[ "$running" == "true" ]]; then
            echo "\n[INFO]    📊 Configuration"
            echo "[INFO]       Version: $version"
            echo "[INFO]       Port: $port"
            echo "[INFO]       API Endpoint: $api_endpoint"
            
            # Show pages count if available
            if [[ "$healthy" == "true" ]]; then
                local pages_count=$(get_pages_count 2>/dev/null || echo "0")
                echo "[INFO]       Pages: $pages_count"
            fi
            
            # Show test results if available
            if [[ -n "$test_results" ]] && [[ "$test_results" != "{}" ]]; then
                echo "\n[INFO]    📊 Test Results"
                local test_passed=$(echo "$test_results" | jq -r '.passed // 0' 2>/dev/null || echo "0")
                local test_total=$(echo "$test_results" | jq -r '.total // 0' 2>/dev/null || echo "0")
                local test_timestamp=$(echo "$test_results" | jq -r '.timestamp // "unknown"' 2>/dev/null || echo "unknown")
                
                if [[ "$test_passed" == "$test_total" ]] && [[ "$test_total" != "0" ]]; then
                    echo "[SUCCESS]    ✅ Tests: $test_passed/$test_total passed"
                else
                    echo "[WARNING]    ⚠️ Tests: $test_passed/$test_total passed"
                fi
                echo "[INFO]       Last run: $test_timestamp"
            fi
        fi
        
        if [[ "$installed" == "false" ]]; then
            echo "\n[INFO]    📊 Installation"
            echo "[INFO]       To install: vrooli resource wikijs install"
        elif [[ "$running" == "false" ]]; then
            echo "\n[INFO]    📊 Start Service"
            echo "[INFO]       To start: vrooli resource wikijs start"
        fi
    fi
}

# Get count of pages
get_pages_count() {
    local query='{ pages { list { id } } }'
    local response=$(graphql_query "$query" 2>/dev/null || echo "{}")
    echo "$response" | jq -r '.data.pages.list | length' 2>/dev/null || echo "0"
}