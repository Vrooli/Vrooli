#!/bin/bash
# Dependencies validation phase - <30 seconds  
# Validates resource availability and connectivity
set -euo pipefail

# Setup paths and utilities
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"

echo "=== Dependencies Phase (Target: <30s) ==="
start_time=$(date +%s)

error_count=0

# Check required resources from service.json
echo "🔍 Checking resource dependencies..."
if command -v jq >/dev/null 2>&1 && [ -f ".vrooli/service.json" ]; then
    # Get required resources from service.json
    required_resources=$(jq -r '.resources | to_entries[] | select(.value.required == true or .value.enabled == true) | .key' .vrooli/service.json 2>/dev/null || echo "")
    
    if [ -n "$required_resources" ]; then
        while IFS= read -r resource; do
            [ -z "$resource" ] && continue
            
            echo "  🔗 Checking $resource..."
            
            # Try to get resource status using vrooli resource command
            if command -v vrooli >/dev/null 2>&1; then
                if vrooli resource status "$resource" >/dev/null 2>&1; then
                    log::success "   ✅ Resource $resource is available"
                else
                    log::error "   ❌ Resource $resource is not available"
                    ((error_count++))
                fi
            else
                log::warning "   ⚠️  vrooli command not available, skipping $resource check"
            fi
        done <<< "$required_resources"
    else
        log::info "ℹ️  No required resources defined in service.json"
    fi
else
    log::warning "⚠️  Cannot parse service.json, skipping resource checks"
fi

# Check basic connectivity tools
echo "🔍 Checking basic tools..."
required_tools=("curl" "jq")
for tool in "${required_tools[@]}"; do
    if command -v "$tool" >/dev/null 2>&1; then
        log::success "   ✅ Tool $tool available"
    else
        log::error "   ❌ Tool $tool not available"
        ((error_count++))
    fi
done

# Check scenario-specific environment
echo "🔍 Checking scenario environment..."
scenario_name=$(basename "$(pwd)")

# Try to get port information if scenario is configured
if command -v vrooli >/dev/null 2>&1; then
    # Check if we can discover API port
    api_port=$(vrooli scenario port "$scenario_name" API_PORT 2>/dev/null || echo "")
    if [ -n "$api_port" ]; then
        log::success "   ✅ API port discovered: $api_port"
        
        # Test basic connectivity to API if it should be running
        if curl -s --max-time 5 "http://localhost:$api_port/health" >/dev/null 2>&1; then
            log::success "   ✅ API health check passed"
        else
            log::info "   ℹ️  API not responding (may not be started yet)"
        fi
    else
        log::info "   ℹ️  API port not configured or scenario not running"
    fi
    
    # Check if we can discover UI port
    ui_port=$(vrooli scenario port "$scenario_name" UI_PORT 2>/dev/null || echo "")
    if [ -n "$ui_port" ]; then
        log::success "   ✅ UI port discovered: $ui_port"
        
        # Test basic connectivity to UI if it should be running
        if curl -s --max-time 5 "http://localhost:$ui_port" >/dev/null 2>&1; then
            log::success "   ✅ UI health check passed"
        else
            log::info "   ℹ️  UI not responding (may not be started yet)"
        fi
    else
        log::info "   ℹ️  UI port not configured or scenario not running"
    fi
else
    log::warning "   ⚠️  vrooli command not available, skipping port discovery"
fi

# Check for CLI tools if they exist
if [ -d "cli" ]; then
    echo "🔍 Checking CLI dependencies..."
    scenario_cli="cli/$scenario_name"
    
    if [ -f "$scenario_cli" ]; then
        if [ -x "$scenario_cli" ]; then
            # Test basic CLI functionality
            if "$scenario_cli" --help >/dev/null 2>&1 || "$scenario_cli" help >/dev/null 2>&1; then
                log::success "   ✅ CLI help command works"
            else
                log::warning "   ⚠️  CLI help command failed"
            fi
        else
            log::warning "   ⚠️  CLI binary not executable"
        fi
    else
        log::info "   ℹ️  CLI binary not found (may be built during setup)"
    fi
fi

# Check development tools (if in development mode)
if [ "${VROOLI_ENV:-}" = "development" ] || [ "${NODE_ENV:-}" = "development" ]; then
    echo "🔍 Checking development tools..."
    
    # Check Go if we have Go code
    if [ -d "api" ] && [ -f "api/go.mod" ]; then
        if command -v go >/dev/null 2>&1; then
            log::success "   ✅ Go compiler available"
        else
            log::error "   ❌ Go compiler not available"
            ((error_count++))
        fi
    fi
    
    # Check Node.js if we have Node.js code
    if [ -d "ui" ] && [ -f "ui/package.json" ]; then
        if command -v node >/dev/null 2>&1; then
            log::success "   ✅ Node.js runtime available"
        else
            log::error "   ❌ Node.js runtime not available"
            ((error_count++))
        fi
        
        if command -v npm >/dev/null 2>&1; then
            log::success "   ✅ npm package manager available"
        else
            log::error "   ❌ npm package manager not available"
            ((error_count++))
        fi
    fi
fi

# Performance check
end_time=$(date +%s)
duration=$((end_time - start_time))
echo ""

if [ $error_count -eq 0 ]; then
    log::success "✅ Dependencies validation completed successfully in ${duration}s"
else
    log::error "❌ Dependencies validation failed with $error_count errors in ${duration}s"
    echo ""
    log::info "💡 Common fixes:"
    echo "   • Start required resources: vrooli resource start <resource-name>"
    echo "   • Start scenario: vrooli scenario start $scenario_name"
    echo "   • Install missing tools: apt install curl jq (or your package manager)"
    echo "   • Check resource status: vrooli resource status"
fi

if [ $duration -gt 30 ]; then
    log::warning "⚠️  Dependencies phase exceeded 30s target"
fi

# Exit with appropriate code
if [ $error_count -eq 0 ]; then
    exit 0
else
    exit 1
fi