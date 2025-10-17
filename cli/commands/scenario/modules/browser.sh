#!/usr/bin/env bash
# Scenario Browser Operations Module
# Handles opening scenarios in web browsers

set -euo pipefail

# Open scenario in browser if it has a UI
scenario::browser::open() {
    local scenario_name="${1:-}"
    local port_name="UI_PORT"
    local print_url=false
    
    # Parse options
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --port)
                port_name="${2:-}"
                if [[ -z "$port_name" ]]; then
                    log::error "--port requires a port name"
                    return 1
                fi
                shift 2
                ;;
            --print-url)
                print_url=true
                shift
                ;;
            *)
                if [[ -z "$scenario_name" ]]; then
                    scenario_name="$1"
                fi
                shift
                ;;
        esac
    done
    
    if [[ -z "$scenario_name" ]]; then
        log::error "Scenario name required"
        echo "Usage: vrooli scenario open <scenario-name> [options]"
        echo ""
        echo "Options:"
        echo "  --port <name>    Open specific port (default: UI_PORT)"
        echo "  --print-url      Print URL instead of opening browser"
        echo ""
        echo "Examples:"
        echo "  vrooli scenario open app-monitor"
        echo "  vrooli scenario open app-monitor --port API_PORT"
        echo "  vrooli scenario open app-monitor --print-url"
        return 1
    fi
    
    # Use the ports module to get the port
    local port_value
    local port_result=0
    
    # Use the ports module to get the step name
    local step_name
    step_name=$(scenario::ports::map_port_to_step "$port_name")
    
    # Direct file read - extremely fast
    local process_file="$HOME/.vrooli/processes/scenarios/${scenario_name}/${step_name}.json"
    
    if [[ ! -f "$process_file" ]]; then
        # Process not running or doesn't exist
        port_result=1
    else
        # Extract port directly from JSON file
        port_value=$(jq -r '.port // empty' "$process_file" 2>/dev/null)
        
        if [[ -z "$port_value" ]] || [[ "$port_value" == "null" ]]; then
            # Port not found in process file
            port_result=1
        fi
    fi
    
    if [[ $port_result -ne 0 ]] || [[ -z "$port_value" ]]; then
        case "$port_name" in
            UI_PORT)
                log::error "No UI port found for scenario '$scenario_name'"
                log::info "This scenario may not have a web interface, or it's not running"
                ;;
            API_PORT)
                log::error "No API port found for scenario '$scenario_name'"
                ;;
            *)
                log::error "No port '$port_name' found for scenario '$scenario_name'"
                ;;
        esac
        log::info "Start the scenario first: vrooli scenario start $scenario_name"
        return 1
    fi
    
    local url="http://localhost:$port_value"
    
    if [[ "$print_url" == "true" ]]; then
        echo "$url"
        return 0
    fi
    
    log::info "Opening $scenario_name at $url"
    
    # Detect platform and open browser
    scenario::browser::open_url "$url"
}

# Open URL in platform-appropriate browser
scenario::browser::open_url() {
    local url="$1"
    
    case "$(uname -s)" in
        Linux*)
            if command -v xdg-open >/dev/null 2>&1; then
                xdg-open "$url" >/dev/null 2>&1 &
            elif command -v firefox >/dev/null 2>&1; then
                firefox "$url" >/dev/null 2>&1 &
            elif command -v google-chrome >/dev/null 2>&1; then
                google-chrome "$url" >/dev/null 2>&1 &
            elif command -v chromium >/dev/null 2>&1; then
                chromium "$url" >/dev/null 2>&1 &
            else
                log::warn "No browser found. Please open manually: $url"
                return 1
            fi
            ;;
        Darwin*)
            open "$url"
            ;;
        CYGWIN*|MINGW32*|MSYS*|MINGW*)
            start "$url"
            ;;
        *)
            log::warn "Unknown platform. Please open manually: $url"
            echo "$url"
            return 1
            ;;
    esac
}