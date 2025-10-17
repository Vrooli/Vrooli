#!/usr/bin/env bash

#######################################
# Browserless Actions - CLI Interface
# 
# Provides command-line interface for atomic browser operations.
# Each action is designed for single-purpose agent tasks without
# requiring full workflow files.
#######################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
ACTIONS_DIR="${APP_ROOT}/resources/browserless/lib"

# Source required libraries
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh" || { echo "FATAL: Failed to load variable definitions" >&2; exit 1; }
# shellcheck disable=SC1091
source "${var_LOG_FILE}" || { echo "FATAL: Failed to load logging library" >&2; exit 1; }
source "$ACTIONS_DIR/browser-ops.sh"
source "$ACTIONS_DIR/session-manager.sh"

# Global variables for universal options
OUTPUT_PATH=""
TIMEOUT_MS="30000"
WAIT_MS="2000"
SESSION_NAME=""
FULL_PAGE="false"
VIEWPORT_WIDTH="1920"
VIEWPORT_HEIGHT="1080"

#######################################
# Smart URL preprocessing - adds protocol if missing
# Automatically adds http:// for localhost URLs and https:// for others
# Arguments:
#   $1 - Raw URL input
# Returns:
#   Properly formatted URL with protocol
#######################################
actions::preprocess_url() {
    local raw_url="$1"
    
    # If URL already has protocol, return as-is
    if [[ "$raw_url" =~ ^https?:// ]]; then
        echo "$raw_url"
        return 0
    fi
    
    # Check if it's a localhost URL (localhost, 127.0.0.1, or starts with localhost:)
    if [[ "$raw_url" =~ ^(localhost|127\.0\.0\.1)(:|$) ]] || [[ "$raw_url" =~ ^localhost: ]]; then
        echo "http://$raw_url"
    else
        # For all other URLs, default to https://
        echo "https://$raw_url"
    fi
}

#######################################
# Parse universal options from arguments
# Sets global variables for common options
#######################################
actions::parse_universal_options() {
    # Reset to defaults (globals are already declared)
    OUTPUT_PATH=""
    TIMEOUT_MS="30000"
    WAIT_MS="2000"
    SESSION_NAME=""
    FULL_PAGE="false"
    VIEWPORT_WIDTH="1920"
    VIEWPORT_HEIGHT="1080"
    
    local remaining_args=()
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --output)
                OUTPUT_PATH="$2"
                shift 2
                ;;
            --timeout)
                if [[ ! "$2" =~ ^[0-9]+$ ]] || [[ "$2" -lt 1000 ]] || [[ "$2" -gt 300000 ]]; then
                    echo "Warning: Invalid timeout value '$2', using default 30000ms" >&2
                    TIMEOUT_MS="30000"
                else
                    TIMEOUT_MS="$2"
                fi
                shift 2
                ;;
            --wait-ms)
                if [[ ! "$2" =~ ^[0-9]+$ ]] || [[ "$2" -lt 0 ]] || [[ "$2" -gt 60000 ]]; then
                    echo "Warning: Invalid wait-ms value '$2', using default 2000ms" >&2
                    WAIT_MS="2000"
                else
                    WAIT_MS="$2"
                fi
                shift 2
                ;;
            --session)
                SESSION_NAME="$2"
                shift 2
                ;;
            --fullpage)
                FULL_PAGE="true"
                shift
                ;;
            --mobile)
                VIEWPORT_WIDTH="390"
                VIEWPORT_HEIGHT="844"
                shift
                ;;
            *)
                # Collect non-option arguments
                remaining_args+=("$1")
                shift
                ;;
        esac
    done
    
    # Return the remaining arguments
    echo "${remaining_args[@]}"
}

#######################################
# Create temporary session for one-off commands
#######################################
actions::create_temp_session() {
    if [[ -n "$SESSION_NAME" ]]; then
        echo "$SESSION_NAME"
    else
        local temp_session
        temp_session="temp_$(date +%s)_$$"
        session::create "$temp_session" >/dev/null 2>&1
        echo "$temp_session"
    fi
}

#######################################
# Cleanup temporary session
#######################################
actions::cleanup_temp_session() {
    local session_id="$1"
    
    if [[ "$session_id" =~ ^temp_ ]] && [[ -z "$SESSION_NAME" ]]; then
        session::destroy "$session_id" >/dev/null 2>&1
    fi
}

#######################################
# Take a screenshot of a URL
# Usage: browserless screenshot <url> [options]
#######################################
actions::screenshot() {
    # Parse options first (this sets global variables)
    local remaining_args_array=()
    local url_param=""
    local scenario_name=""
    local scenario_path=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --url)
                url_param="$2"
                shift 2
                ;;
            --scenario)
                scenario_name="$2"
                shift 2
                ;;
            --path)
                scenario_path="$2"
                shift 2
                ;;
            --output)
                OUTPUT_PATH="$2"
                shift 2
                ;;
            --timeout)
                if [[ ! "$2" =~ ^[0-9]+$ ]] || [[ "$2" -lt 1000 ]] || [[ "$2" -gt 300000 ]]; then
                    echo "Warning: Invalid timeout value '$2', using default 30000ms" >&2
                    TIMEOUT_MS="30000"
                else
                    TIMEOUT_MS="$2"
                fi
                shift 2
                ;;
            --wait-ms)
                if [[ ! "$2" =~ ^[0-9]+$ ]] || [[ "$2" -lt 0 ]] || [[ "$2" -gt 60000 ]]; then
                    echo "Warning: Invalid wait-ms value '$2', using default 2000ms" >&2
                    WAIT_MS="2000"
                else
                    WAIT_MS="$2"
                fi
                shift 2
                ;;
            --session)
                SESSION_NAME="$2"
                shift 2
                ;;
            --fullpage)
                FULL_PAGE="true"
                shift
                ;;
            --mobile)
                VIEWPORT_WIDTH="390"
                VIEWPORT_HEIGHT="844"
                shift
                ;;
            --help|-h)
                echo "Usage: browserless screenshot [URL] [OPTIONS]"
                echo ""
                echo "Arguments:"
                echo "  URL                    Target URL to screenshot (can also use --url)"
                echo ""
                echo "Options:"
                echo "  --url URL              Target URL (alternative to positional argument)"
                echo "  --output FILE          Output file path (default: screenshot-TIMESTAMP.png)"
                echo "  --fullpage             Capture full page instead of viewport"
                echo "  --mobile               Use mobile viewport (390x844)"
                echo "  --timeout MS           Timeout in milliseconds (default: 30000)"
                echo "  --wait-ms MS           Wait time after load (default: 2000)"
                echo "  --session NAME         Use persistent session"
                echo "  --scenario NAME        Target scenario (auto-resolves http://localhost:<UI_PORT>)"
                echo "  --path RELATIVE_PATH   Optional path when using --scenario (e.g. /dashboard)"
                echo "  --help, -h             Show this help message"
                echo ""
                echo "Examples:"
                echo "  browserless screenshot https://example.com"
                echo "  browserless screenshot --url https://example.com --output page.png"
                echo "  browserless screenshot https://example.com --fullpage --mobile"
                echo "  browserless screenshot --scenario ecosystem-manager --path /dashboard"
                return 0
                ;;
            *)
                remaining_args_array+=("$1")
                shift
                ;;
        esac
    done
    
    if [[ -n "$scenario_path" && -z "$scenario_name" ]]; then
        echo "Error: --path option requires --scenario" >&2
        return 1
    fi

    # Determine URL from either --url flag or positional argument
    local url=""
    if [[ -n "$scenario_name" ]]; then
        if [[ -n "$url_param" ]]; then
            echo "Error: --url cannot be used together with --scenario" >&2
            return 1
        fi

        if [[ ${#remaining_args_array[@]} -gt 0 ]]; then
            echo "Error: Positional URL arguments are not allowed when using --scenario. Use --path for relative routes." >&2
            return 1
        fi

        if [[ "$scenario_path" == http://* || "$scenario_path" == https://* ]]; then
            echo "Error: --path expects a relative value (e.g. /dashboard)." >&2
            return 1
        fi

        if ! command -v vrooli >/dev/null 2>&1; then
            echo "Error: vrooli CLI is required for --scenario usage" >&2
            return 1
        fi

        local scenario_port
        if ! scenario_port=$(vrooli scenario port "$scenario_name" UI_PORT 2>/dev/null); then
            echo "Error: Failed to resolve UI port for scenario '$scenario_name'." >&2
            echo "       • Confirm the scenario name is correct." >&2
            echo "       • Run 'vrooli scenario status $scenario_name' to verify it is running and healthy." >&2
            return 1
        fi

        if [[ -z "$scenario_port" ]] || ! [[ "$scenario_port" =~ ^[0-9]+$ ]]; then
            echo "Error: Invalid UI port for scenario '$scenario_name': ${scenario_port:-"<empty>"}." >&2
            echo "       Run 'vrooli scenario status $scenario_name' to verify it is running and exposing a UI port." >&2
            return 1
        fi

        local trimmed_path="${scenario_path}"
        if [[ -n "$trimmed_path" ]]; then
            trimmed_path="${trimmed_path#/}"
            if [[ -n "$trimmed_path" ]]; then
                url="http://localhost:${scenario_port}/${trimmed_path}"
            else
                url="http://localhost:${scenario_port}/"
            fi
        else
            url="http://localhost:${scenario_port}"
        fi
    else
        if [[ -n "$url_param" ]]; then
            url="$url_param"
        elif [[ ${#remaining_args_array[@]} -gt 0 ]]; then
            url="${remaining_args_array[0]}"
        fi
    fi
    
    # Validate URL is provided
    if [[ -z "$url" ]]; then
        echo "Error: URL required" >&2
        echo "" >&2
        echo "Usage: browserless screenshot [URL] [OPTIONS]" >&2
        echo "   or: browserless screenshot --url URL [OPTIONS]" >&2
        echo "" >&2
        echo "Examples:" >&2
        echo "  browserless screenshot https://example.com --output page.png" >&2
        echo "  browserless screenshot example.com --output page.png" >&2
        echo "  browserless screenshot localhost:3000 --output page.png" >&2
        echo "  browserless screenshot --url https://example.com --fullpage" >&2
        echo "" >&2
        echo "Use --help for full documentation" >&2
        return 1
    fi
    
    # Smart URL preprocessing - automatically add protocol if missing
    url=$(actions::preprocess_url "$url")
    log::debug "Preprocessed URL: $url"
    
    # Set default output if not specified
    if [[ -z "$OUTPUT_PATH" ]]; then
        OUTPUT_PATH="screenshot-$(date +%s).png"
    fi
    
    # Ensure output directory exists (only if path contains a directory)
    if [[ "$OUTPUT_PATH" == */* ]]; then
        mkdir -p "${OUTPUT_PATH%/*}"
    fi
    
    log::info "📸 Taking screenshot of $url"
    
    # Use the browserless screenshot API directly
    local browserless_port="${BROWSERLESS_PORT:-4110}"
    
    # Build JSON payload using more reliable method
    local json_payload
    json_payload=$(cat <<EOF
{
    "url": "$url"
}
EOF
    )
    
    # Add fullPage option if specified
    if [[ "$FULL_PAGE" == "true" ]]; then
        json_payload=$(echo "$json_payload" | jq '. + {"options": {"fullPage": true}}')
    fi
    
    # Always add viewport to ensure consistent sizing (browserless server may have different defaults)
    if [[ -n "$VIEWPORT_WIDTH" ]] && [[ -n "$VIEWPORT_HEIGHT" ]]; then
        json_payload=$(echo "$json_payload" | jq --argjson width "$VIEWPORT_WIDTH" --argjson height "$VIEWPORT_HEIGHT" \
            '. + {"viewport": {"width": $width, "height": $height}}')
    fi
    
    # Make the API call
    local http_status
    http_status=$(curl -s -X POST \
        "http://localhost:${browserless_port}/chrome/screenshot" \
        -H "Content-Type: application/json" \
        -d "$json_payload" \
        -o "$OUTPUT_PATH" \
        -w "%{http_code}" 2>/dev/null)
    
    if [[ "$http_status" == "200" ]]; then
        echo "Screenshot saved: $OUTPUT_PATH"
        echo "URL: $url"
        return 0
    else
        echo "Error: Failed to take screenshot (HTTP $http_status)" >&2
        return 1
    fi
}

#######################################
# Navigate to a URL and return basic info
# Usage: browserless navigate <url> [options]
#######################################
actions::navigate() {
    # Reset defaults
    OUTPUT_PATH=""
    TIMEOUT_MS="30000"
    WAIT_MS="2000"
    SESSION_NAME=""
    FULL_PAGE="false"
    VIEWPORT_WIDTH="1920"
    VIEWPORT_HEIGHT="1080"
    
    # Parse options directly
    local remaining_args_array=()
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --output)
                OUTPUT_PATH="$2"
                shift 2
                ;;
            --timeout)
                if [[ ! "$2" =~ ^[0-9]+$ ]] || [[ "$2" -lt 1000 ]] || [[ "$2" -gt 300000 ]]; then
                    echo "Warning: Invalid timeout value '$2', using default 30000ms" >&2
                    TIMEOUT_MS="30000"
                else
                    TIMEOUT_MS="$2"
                fi
                shift 2
                ;;
            --wait-ms)
                if [[ ! "$2" =~ ^[0-9]+$ ]] || [[ "$2" -lt 0 ]] || [[ "$2" -gt 60000 ]]; then
                    echo "Warning: Invalid wait-ms value '$2', using default 2000ms" >&2
                    WAIT_MS="2000"
                else
                    WAIT_MS="$2"
                fi
                shift 2
                ;;
            --session)
                SESSION_NAME="$2"
                shift 2
                ;;
            --fullpage)
                FULL_PAGE="true"
                shift
                ;;
            --mobile)
                VIEWPORT_WIDTH="390"
                VIEWPORT_HEIGHT="844"
                shift
                ;;
            *)
                remaining_args_array+=("$1")
                shift
                ;;
        esac
    done
    
    local url="${remaining_args_array[0]:-}"
    if [[ -z "$url" ]]; then
        echo "Error: URL required" >&2
        echo "Usage: browserless navigate <url> [--output result.json] [--wait-ms 2000]" >&2
        echo "" >&2
        echo "Examples:" >&2
        echo "  browserless navigate https://example.com" >&2
        echo "  browserless navigate example.com --output result.json" >&2
        echo "  browserless navigate localhost:3000" >&2
        return 1
    fi
    
    # Smart URL preprocessing - automatically add protocol if missing
    url=$(actions::preprocess_url "$url")
    log::debug "Preprocessed URL: $url"
    
    local session_id
    session_id=$(actions::create_temp_session)
    
    log::info "🌐 Navigating to $url"
    
    # Wait initial period before navigation
    browser::wait "$WAIT_MS" "$session_id"
    
    local result
    if result=$(browser::navigate "$url" "$session_id"); then
        local success
        success=$(echo "$result" | jq -r '.success // false')
        if [[ "$success" == "true" ]]; then
            local clean_result
            clean_result=$(echo "$result" | jq '{
                success: .success,
                url: .url,
                title: .title,
                timestamp: (now | strftime("%Y-%m-%dT%H:%M:%SZ"))
            }')
            
            if [[ -n "$OUTPUT_PATH" ]]; then
                mkdir -p "${OUTPUT_PATH%/*}"
                echo "$clean_result" > "$OUTPUT_PATH"
                echo "Navigation result saved: $OUTPUT_PATH"
            fi
            
            echo "Successfully navigated to: $(echo "$clean_result" | jq -r '.url')"
            echo "Page title: $(echo "$clean_result" | jq -r '.title')"
        else
            local error
            error=$(echo "$result" | jq -r '.error // "Unknown error"')
            echo "Error: $error" >&2
            actions::cleanup_temp_session "$session_id"
            return 1
        fi
    else
        echo "Error: Navigation failed" >&2
        actions::cleanup_temp_session "$session_id"
        return 1
    fi
    
    actions::cleanup_temp_session "$session_id"
    return 0
}

#######################################
# Check if a URL loads successfully
# Usage: browserless health-check <url> [options]
#######################################
actions::health_check() {
    # Reset defaults
    OUTPUT_PATH=""
    TIMEOUT_MS="30000"
    WAIT_MS="2000"
    SESSION_NAME=""
    FULL_PAGE="false"
    VIEWPORT_WIDTH="1920"
    VIEWPORT_HEIGHT="1080"
    
    # Parse options directly
    local remaining_args_array=()
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --output)
                OUTPUT_PATH="$2"
                shift 2
                ;;
            --timeout)
                if [[ ! "$2" =~ ^[0-9]+$ ]] || [[ "$2" -lt 1000 ]] || [[ "$2" -gt 300000 ]]; then
                    echo "Warning: Invalid timeout value '$2', using default 30000ms" >&2
                    TIMEOUT_MS="30000"
                else
                    TIMEOUT_MS="$2"
                fi
                shift 2
                ;;
            --wait-ms)
                if [[ ! "$2" =~ ^[0-9]+$ ]] || [[ "$2" -lt 0 ]] || [[ "$2" -gt 60000 ]]; then
                    echo "Warning: Invalid wait-ms value '$2', using default 2000ms" >&2
                    WAIT_MS="2000"
                else
                    WAIT_MS="$2"
                fi
                shift 2
                ;;
            --session)
                SESSION_NAME="$2"
                shift 2
                ;;
            --fullpage)
                FULL_PAGE="true"
                shift
                ;;
            --mobile)
                VIEWPORT_WIDTH="390"
                VIEWPORT_HEIGHT="844"
                shift
                ;;
            *)
                remaining_args_array+=("$1")
                shift
                ;;
        esac
    done
    
    local url="${remaining_args_array[0]:-}"
    local expected_text="${remaining_args_array[1]:-}"
    
    if [[ -z "$url" ]]; then
        echo "Error: URL required" >&2
        echo "Usage: browserless health-check <url> [expected-text] [--timeout 10000]" >&2
        echo "" >&2
        echo "Examples:" >&2
        echo "  browserless health-check https://example.com" >&2
        echo "  browserless health-check example.com" >&2
        echo "  browserless health-check localhost:3000" >&2
        return 1
    fi
    
    # Smart URL preprocessing - automatically add protocol if missing
    url=$(actions::preprocess_url "$url")
    log::debug "Preprocessed URL: $url"
    
    local session_id
    session_id=$(actions::create_temp_session)
    
    log::info "🏥 Health checking $url" >&2
    
    local result
    if result=$(browser::navigate "$url" "$session_id"); then
        local success
        success=$(echo "$result" | jq -r '.success // false')
        if [[ "$success" == "true" ]]; then
            local healthy=true
            local health_result
            health_result=$(echo "$result" | jq '{
                url: .url,
                title: .title,
                healthy: true,
                timestamp: (now | strftime("%Y-%m-%dT%H:%M:%SZ"))
            }')
            
            # Check for expected text if provided
            if [[ -n "$expected_text" ]]; then
                local page_content
                if page_content=$(browser::evaluate "return document.body.textContent;" "$session_id"); then
                    local content_success=$(echo "$page_content" | jq -r '.success // false')
                    if [[ "$content_success" == "true" ]]; then
                        local content=$(echo "$page_content" | jq -r '.result // ""')
                        if [[ "$content" == *"$expected_text"* ]]; then
                            health_result=$(echo "$health_result" | jq --arg text "$expected_text" '.expected_text_found = true | .expected_text = $text')
                        else
                            healthy=false
                            health_result=$(echo "$health_result" | jq --arg text "$expected_text" '.healthy = false | .expected_text_found = false | .expected_text = $text')
                        fi
                    else
                        healthy=false
                        health_result=$(echo "$health_result" | jq '.healthy = false | .error = "Could not read page content"')
                    fi
                fi
            fi
            
            if [[ -n "$OUTPUT_PATH" ]]; then
                mkdir -p "${OUTPUT_PATH%/*}"
                echo "$health_result" > "$OUTPUT_PATH"
                echo "Health check result saved: $OUTPUT_PATH"
            fi
            
            if [[ "$healthy" == "true" ]]; then
                echo "✅ Health check passed"
                echo "URL: $(echo "$health_result" | jq -r '.url')"
                echo "Title: $(echo "$health_result" | jq -r '.title')"
                if [[ -n "$expected_text" ]]; then
                    echo "Expected text found: $expected_text"
                fi
            else
                echo "❌ Health check failed"
                if [[ -n "$expected_text" ]]; then
                    echo "Expected text not found: $expected_text"
                fi
                actions::cleanup_temp_session "$session_id"
                return 1
            fi
        else
            local error
            error=$(echo "$result" | jq -r '.error // "Unknown error"')
            echo "❌ Health check failed: $error" >&2
            actions::cleanup_temp_session "$session_id"
            return 1
        fi
    else
        echo "❌ Health check failed: Could not navigate to URL" >&2
        actions::cleanup_temp_session "$session_id"
        return 1
    fi
    
    actions::cleanup_temp_session "$session_id"
    return 0
}

#######################################
# Check if an element exists on a page
# Usage: browserless element-exists <url> --selector "button.login" [options]
#######################################
actions::element_exists() {
    # Reset defaults
    OUTPUT_PATH=""
    TIMEOUT_MS="30000"
    WAIT_MS="2000"
    SESSION_NAME=""
    FULL_PAGE="false"
    VIEWPORT_WIDTH="1920"
    VIEWPORT_HEIGHT="1080"
    
    local selector=""
    local remaining_args_array=()
    
    # Parse options directly
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --selector)
                selector="$2"
                shift 2
                ;;
            --output)
                OUTPUT_PATH="$2"
                shift 2
                ;;
            --timeout)
                if [[ ! "$2" =~ ^[0-9]+$ ]] || [[ "$2" -lt 1000 ]] || [[ "$2" -gt 300000 ]]; then
                    echo "Warning: Invalid timeout value '$2', using default 30000ms" >&2
                    TIMEOUT_MS="30000"
                else
                    TIMEOUT_MS="$2"
                fi
                shift 2
                ;;
            --wait-ms)
                if [[ ! "$2" =~ ^[0-9]+$ ]] || [[ "$2" -lt 0 ]] || [[ "$2" -gt 60000 ]]; then
                    echo "Warning: Invalid wait-ms value '$2', using default 2000ms" >&2
                    WAIT_MS="2000"
                else
                    WAIT_MS="$2"
                fi
                shift 2
                ;;
            --session)
                SESSION_NAME="$2"
                shift 2
                ;;
            --fullpage)
                FULL_PAGE="true"
                shift
                ;;
            --mobile)
                VIEWPORT_WIDTH="390"
                VIEWPORT_HEIGHT="844"
                shift
                ;;
            *)
                remaining_args_array+=("$1")
                shift
                ;;
        esac
    done
    
    local url="${remaining_args_array[0]:-}"
    
    if [[ -z "$url" ]] || [[ -z "$selector" ]]; then
        echo "Error: URL and selector required" >&2
        echo "Usage: browserless element-exists <url> --selector \"button.login\" [--timeout 5000]" >&2
        return 1
    fi
    
    local session_id
    session_id=$(actions::create_temp_session)
    
    log::info "🔍 Checking if element exists: $selector"
    
    # Navigate first
    local nav_result
    if nav_result=$(browser::navigate "$url" "$session_id"); then
        local nav_success=$(echo "$nav_result" | jq -r '.success // false')
        if [[ "$nav_success" == "true" ]]; then
            # Wait for page to stabilize
            browser::wait "$WAIT_MS" "$session_id"
            
            # Check if element exists
            local exists
            if exists=$(browser::element_exists "$selector" "$session_id"); then
                if [[ "$exists" == "true" ]]; then
                    echo "✅ Element exists: $selector"
                    actions::cleanup_temp_session "$session_id"
                    return 0
                else
                    echo "❌ Element not found: $selector" >&2
                    actions::cleanup_temp_session "$session_id"
                    return 1
                fi
            else
                echo "❌ Error checking element: $selector" >&2
                actions::cleanup_temp_session "$session_id"
                return 1
            fi
        else
            local error=$(echo "$nav_result" | jq -r '.error // "Unknown error"')
            echo "Error: Navigation failed - $error" >&2
            actions::cleanup_temp_session "$session_id"
            return 1
        fi
    else
        echo "Error: Could not navigate to URL" >&2
        actions::cleanup_temp_session "$session_id"
        return 1
    fi
}

#######################################
# Extract text content from a page
# Usage: browserless extract-text <url> --selector "h1" [options]
#######################################
actions::extract_text() {
    # Reset defaults
    OUTPUT_PATH=""
    TIMEOUT_MS="30000"
    WAIT_MS="2000"
    SESSION_NAME=""
    FULL_PAGE="false"
    VIEWPORT_WIDTH="1920"
    VIEWPORT_HEIGHT="1080"
    
    local selector=""
    local remaining_args_array=()
    
    # Parse options directly
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --selector)
                selector="$2"
                shift 2
                ;;
            --output)
                OUTPUT_PATH="$2"
                shift 2
                ;;
            --timeout)
                if [[ ! "$2" =~ ^[0-9]+$ ]] || [[ "$2" -lt 1000 ]] || [[ "$2" -gt 300000 ]]; then
                    echo "Warning: Invalid timeout value '$2', using default 30000ms" >&2
                    TIMEOUT_MS="30000"
                else
                    TIMEOUT_MS="$2"
                fi
                shift 2
                ;;
            --wait-ms)
                if [[ ! "$2" =~ ^[0-9]+$ ]] || [[ "$2" -lt 0 ]] || [[ "$2" -gt 60000 ]]; then
                    echo "Warning: Invalid wait-ms value '$2', using default 2000ms" >&2
                    WAIT_MS="2000"
                else
                    WAIT_MS="$2"
                fi
                shift 2
                ;;
            --session)
                SESSION_NAME="$2"
                shift 2
                ;;
            --fullpage)
                FULL_PAGE="true"
                shift
                ;;
            --mobile)
                VIEWPORT_WIDTH="390"
                VIEWPORT_HEIGHT="844"
                shift
                ;;
            *)
                remaining_args_array+=("$1")
                shift
                ;;
        esac
    done
    
    local url="${remaining_args_array[0]:-}"
    
    if [[ -z "$url" ]] || [[ -z "$selector" ]]; then
        echo "Error: URL and selector required" >&2
        echo "Usage: browserless extract-text <url> --selector \"h1\" [--output title.txt]" >&2
        return 1
    fi
    
    local session_id
    session_id=$(actions::create_temp_session)
    
    log::info "📄 Extracting text from: $selector"
    
    # Navigate first
    local nav_result
    if nav_result=$(browser::navigate "$url" "$session_id"); then
        local nav_success=$(echo "$nav_result" | jq -r '.success // false')
        if [[ "$nav_success" == "true" ]]; then
            # Wait for page to stabilize
            browser::wait "$WAIT_MS" "$session_id"
            
            # Extract text
            local text
            if text=$(browser::get_text "$selector" "$session_id"); then
                if [[ -n "$OUTPUT_PATH" ]]; then
                    mkdir -p "${OUTPUT_PATH%/*}"
                    echo "$text" > "$OUTPUT_PATH"
                    echo "Text saved: $OUTPUT_PATH"
                fi
                
                echo "Extracted text: $text"
                actions::cleanup_temp_session "$session_id"
                return 0
            else
                echo "Error: Could not extract text from selector: $selector" >&2
                actions::cleanup_temp_session "$session_id"
                return 1
            fi
        else
            local error=$(echo "$nav_result" | jq -r '.error // "Unknown error"')
            echo "Error: Navigation failed - $error" >&2
            actions::cleanup_temp_session "$session_id"
            return 1
        fi
    else
        echo "Error: Could not navigate to URL" >&2
        actions::cleanup_temp_session "$session_id"
        return 1
    fi
}

#######################################
# Extract structured data using custom JavaScript
# Usage: browserless extract <url> --script "return {title: document.title}" [options]
#######################################
actions::extract() {
    # Reset defaults
    OUTPUT_PATH=""
    TIMEOUT_MS="30000"
    WAIT_MS="2000"
    SESSION_NAME=""
    FULL_PAGE="false"
    VIEWPORT_WIDTH="1920"
    VIEWPORT_HEIGHT="1080"
    
    local script=""
    local remaining_args_array=()
    
    # Parse options directly
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --script)
                script="$2"
                shift 2
                ;;
            --output)
                OUTPUT_PATH="$2"
                shift 2
                ;;
            --timeout)
                if [[ ! "$2" =~ ^[0-9]+$ ]] || [[ "$2" -lt 1000 ]] || [[ "$2" -gt 300000 ]]; then
                    echo "Warning: Invalid timeout value '$2', using default 30000ms" >&2
                    TIMEOUT_MS="30000"
                else
                    TIMEOUT_MS="$2"
                fi
                shift 2
                ;;
            --wait-ms)
                if [[ ! "$2" =~ ^[0-9]+$ ]] || [[ "$2" -lt 0 ]] || [[ "$2" -gt 60000 ]]; then
                    echo "Warning: Invalid wait-ms value '$2', using default 2000ms" >&2
                    WAIT_MS="2000"
                else
                    WAIT_MS="$2"
                fi
                shift 2
                ;;
            --session)
                SESSION_NAME="$2"
                shift 2
                ;;
            --fullpage)
                FULL_PAGE="true"
                shift
                ;;
            --mobile)
                VIEWPORT_WIDTH="390"
                VIEWPORT_HEIGHT="844"
                shift
                ;;
            *)
                remaining_args_array+=("$1")
                shift
                ;;
        esac
    done
    
    local url="${remaining_args_array[0]:-}"
    
    if [[ -z "$url" ]] || [[ -z "$script" ]]; then
        echo "Error: URL and script required" >&2
        echo "Usage: browserless extract <url> --script \"return {title: document.title}\" [--output data.json]" >&2
        return 1
    fi
    
    local session_id
    session_id=$(actions::create_temp_session)
    
    log::info "⚙️  Extracting data with custom script"
    
    # Navigate first
    local nav_result
    if nav_result=$(browser::navigate "$url" "$session_id"); then
        local nav_success=$(echo "$nav_result" | jq -r '.success // false')
        if [[ "$nav_success" == "true" ]]; then
            # Wait for page to stabilize
            browser::wait "$WAIT_MS" "$session_id"
            
            # Execute custom script
            local result
            if result=$(browser::evaluate "$script" "$session_id"); then
                local eval_success=$(echo "$result" | jq -r '.success // false')
                if [[ "$eval_success" == "true" ]]; then
                    local extracted_data=$(echo "$result" | jq -r '.result')
                    
                    if [[ -n "$OUTPUT_PATH" ]]; then
                        mkdir -p "${OUTPUT_PATH%/*}"
                        echo "$extracted_data" > "$OUTPUT_PATH"
                        echo "Data saved: $OUTPUT_PATH"
                    fi
                    
                    echo "Extracted data:"
                    echo "$extracted_data"
                    actions::cleanup_temp_session "$session_id"
                    return 0
                else
                    local error
            error=$(echo "$result" | jq -r '.error // "Unknown error"')
                    echo "Error: Script execution failed - $error" >&2
                    actions::cleanup_temp_session "$session_id"
                    return 1
                fi
            else
                echo "Error: Could not execute script" >&2
                actions::cleanup_temp_session "$session_id"
                return 1
            fi
        else
            local error=$(echo "$nav_result" | jq -r '.error // "Unknown error"')
            echo "Error: Navigation failed - $error" >&2
            actions::cleanup_temp_session "$session_id"
            return 1
        fi
    else
        echo "Error: Could not navigate to URL" >&2
        actions::cleanup_temp_session "$session_id"
        return 1
    fi
}

#######################################
# Perform basic interactions (fill forms, click buttons)
# Usage: browserless interact <url> --fill "input[name=email]:admin@example.com" --click "button[type=submit]" [options]
#######################################
actions::interact() {
    # Reset defaults
    OUTPUT_PATH=""
    TIMEOUT_MS="30000"
    WAIT_MS="2000"
    SESSION_NAME=""
    FULL_PAGE="false"
    VIEWPORT_WIDTH="1920"
    VIEWPORT_HEIGHT="1080"
    
    local interactions=()
    local remaining_args_array=()
    
    # Parse options directly
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --fill)
                interactions+=("fill:$2")
                shift 2
                ;;
            --click)
                interactions+=("click:$2")
                shift 2
                ;;
            --wait-for)
                interactions+=("wait-for:$2")
                shift 2
                ;;
            --output)
                OUTPUT_PATH="$2"
                shift 2
                ;;
            --timeout)
                if [[ ! "$2" =~ ^[0-9]+$ ]] || [[ "$2" -lt 1000 ]] || [[ "$2" -gt 300000 ]]; then
                    echo "Warning: Invalid timeout value '$2', using default 30000ms" >&2
                    TIMEOUT_MS="30000"
                else
                    TIMEOUT_MS="$2"
                fi
                shift 2
                ;;
            --wait-ms)
                if [[ ! "$2" =~ ^[0-9]+$ ]] || [[ "$2" -lt 0 ]] || [[ "$2" -gt 60000 ]]; then
                    echo "Warning: Invalid wait-ms value '$2', using default 2000ms" >&2
                    WAIT_MS="2000"
                else
                    WAIT_MS="$2"
                fi
                shift 2
                ;;
            --session)
                SESSION_NAME="$2"
                shift 2
                ;;
            --fullpage)
                FULL_PAGE="true"
                shift
                ;;
            --mobile)
                VIEWPORT_WIDTH="390"
                VIEWPORT_HEIGHT="844"
                shift
                ;;
            *)
                remaining_args_array+=("$1")
                shift
                ;;
        esac
    done
    
    local url="${remaining_args_array[0]:-}"
    
    if [[ -z "$url" ]]; then
        echo "Error: URL required" >&2
        echo "Usage: browserless interact <url> --fill \"input[name=email]:value\" --click \"button[type=submit]\" [--screenshot result.png]" >&2
        return 1
    fi
    
    if [[ ${#interactions[@]} -eq 0 ]]; then
        echo "Error: At least one interaction required" >&2
        echo "Available interactions: --fill, --click, --wait-for" >&2
        return 1
    fi
    
    local session_id
    session_id=$(actions::create_temp_session)
    
    log::info "🤖 Performing interactions on $url"
    
    # Navigate first
    local nav_result
    if nav_result=$(browser::navigate "$url" "$session_id"); then
        local nav_success=$(echo "$nav_result" | jq -r '.success // false')
        if [[ "$nav_success" == "true" ]]; then
            # Wait for page to stabilize
            browser::wait "$WAIT_MS" "$session_id"
            
            # Execute interactions in order
            for interaction in "${interactions[@]}"; do
                local action="${interaction%%:*}"
                local param="${interaction#*:}"
                
                case "$action" in
                    fill)
                        # Format: "selector:value"
                        local selector="${param%%:*}"
                        local value="${param#*:}"
                        log::info "✍️  Filling $selector with: $value"
                        if ! browser::fill "$selector" "$value" "$session_id"; then
                            echo "Error: Failed to fill $selector" >&2
                            actions::cleanup_temp_session "$session_id"
                            return 1
                        fi
                        ;;
                    click)
                        log::info "👆 Clicking: $param"
                        if ! browser::click "$param" "$session_id"; then
                            echo "Error: Failed to click $param" >&2
                            actions::cleanup_temp_session "$session_id"
                            return 1
                        fi
                        ;;
                    wait-for)
                        log::info "⏳ Waiting for element: $param"
                        if ! browser::wait_for_element "$param" "$TIMEOUT_MS" "$session_id"; then
                            echo "Error: Element did not appear: $param" >&2
                            actions::cleanup_temp_session "$session_id"
                            return 1
                        fi
                        ;;
                esac
                
                # Small wait between interactions
                browser::wait "500" "$session_id"
            done
            
            # Take final screenshot if output path provided
            if [[ -n "$OUTPUT_PATH" ]]; then
                mkdir -p "${OUTPUT_PATH%/*}"
                if browser::screenshot "$OUTPUT_PATH" "$session_id" "$FULL_PAGE"; then
                    echo "Final screenshot saved: $OUTPUT_PATH"
                else
                    echo "Warning: Could not save screenshot" >&2
                fi
            fi
            
            echo "✅ All interactions completed successfully"
            actions::cleanup_temp_session "$session_id"
            return 0
        else
            local error=$(echo "$nav_result" | jq -r '.error // "Unknown error"')
            echo "Error: Navigation failed - $error" >&2
            actions::cleanup_temp_session "$session_id"
            return 1
        fi
    else
        echo "Error: Could not navigate to URL" >&2
        actions::cleanup_temp_session "$session_id"
        return 1
    fi
}

#######################################
# Capture console logs from a page
# Usage: browserless console <url> [--filter error] [options]
#######################################
actions::console() {
    # Reset defaults
    OUTPUT_PATH=""
    TIMEOUT_MS="30000"
    WAIT_MS="2000"
    SESSION_NAME=""
    FULL_PAGE="false"
    VIEWPORT_WIDTH="1920"
    VIEWPORT_HEIGHT="1080"
    
    local filter=""
    local remaining_args_array=()
    
    # Parse options directly
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --filter)
                filter="$2"
                shift 2
                ;;
            --output)
                OUTPUT_PATH="$2"
                shift 2
                ;;
            --timeout)
                if [[ ! "$2" =~ ^[0-9]+$ ]] || [[ "$2" -lt 1000 ]] || [[ "$2" -gt 300000 ]]; then
                    echo "Warning: Invalid timeout value '$2', using default 30000ms" >&2
                    TIMEOUT_MS="30000"
                else
                    TIMEOUT_MS="$2"
                fi
                shift 2
                ;;
            --wait-ms)
                if [[ ! "$2" =~ ^[0-9]+$ ]] || [[ "$2" -lt 0 ]] || [[ "$2" -gt 60000 ]]; then
                    echo "Warning: Invalid wait-ms value '$2', using default 2000ms" >&2
                    WAIT_MS="2000"
                else
                    WAIT_MS="$2"
                fi
                shift 2
                ;;
            --session)
                SESSION_NAME="$2"
                shift 2
                ;;
            --fullpage)
                FULL_PAGE="true"
                shift
                ;;
            --mobile)
                VIEWPORT_WIDTH="390"
                VIEWPORT_HEIGHT="844"
                shift
                ;;
            *)
                remaining_args_array+=("$1")
                shift
                ;;
        esac
    done
    
    local url="${remaining_args_array[0]:-}"
    
    if [[ -z "$url" ]]; then
        echo "Error: URL required" >&2
        echo "Usage: browserless console <url> [--filter error] [--output console.json]" >&2
        return 1
    fi
    
    local session_id
    session_id=$(actions::create_temp_session)
    
    log::info "🖥️  Capturing console logs from $url"
    
    # Use browserless function API with direct console event handling
    local browserless_port="${BROWSERLESS_PORT:-4110}"
    
    # Create the JavaScript function for console capture using ES6 export default format
    local wrapped_code="export default async ({ page, context }) => {
        try {
            const logs = [];
            
            // Set up console event listener before navigation
            page.on('console', msg => {
                logs.push({
                    level: msg.type(),
                    message: msg.text(),
                    timestamp: new Date().toISOString()
                });
            });
            
            // Navigate to the URL
            await page.goto('$url', { 
                waitUntil: 'networkidle2',
                timeout: 30000 
            });
            
            // Wait for any additional console activity
            await new Promise(resolve => setTimeout(resolve, $WAIT_MS));
            
            return {
                success: true,
                logs: logs,
                url: page.url(),
                title: await page.title()
            };
        } catch (error) {
            return { 
                success: false, 
                error: error.message,
                stack: error.stack 
            };
        }
    };"
    
    # Execute via browserless v2 API
    local result
    result=$(curl -s -X POST \
        -H "Content-Type: application/javascript" \
        -d "$wrapped_code" \
        "http://localhost:${browserless_port}/chrome/function" 2>/dev/null)
    
    local success
    success=$(echo "$result" | jq -r '.success // false')
    if [[ "$success" == "true" ]]; then
        local logs=$(echo "$result" | jq '.logs')
        
        # Apply filter if specified
        if [[ -n "$filter" ]]; then
            logs=$(echo "$logs" | jq --arg filter "$filter" '[.[] | select(.level == $filter)]')
        fi
        
        local filtered_result
        filtered_result=$(echo "$result" | jq --argjson logs "$logs" '.logs = $logs')
        
        if [[ -n "$OUTPUT_PATH" ]]; then
            mkdir -p "${OUTPUT_PATH%/*}"
            echo "$filtered_result" > "$OUTPUT_PATH"
            echo "Console logs saved: $OUTPUT_PATH"
        fi
        
        local log_count=$(echo "$logs" | jq 'length')
        echo "Captured $log_count console messages"
        if [[ -n "$filter" ]]; then
            echo "Filter: $filter"
        fi
        echo "$logs" | jq '.'
    else
        local error=$(echo "$result" | jq -r '.error // "Unknown error"')
        echo "Error: Console capture failed - $error" >&2
        actions::cleanup_temp_session "$session_id"
        return 1
    fi
    
    actions::cleanup_temp_session "$session_id"
    return 0
}

#######################################
# Get page performance metrics
# Usage: browserless performance <url> [options]
#######################################
actions::performance() {
    # Reset defaults
    OUTPUT_PATH=""
    TIMEOUT_MS="30000"
    WAIT_MS="2000"
    SESSION_NAME=""
    FULL_PAGE="false"
    VIEWPORT_WIDTH="1920"
    VIEWPORT_HEIGHT="1080"
    
    # Parse options directly
    local remaining_args_array=()
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --output)
                OUTPUT_PATH="$2"
                shift 2
                ;;
            --timeout)
                if [[ ! "$2" =~ ^[0-9]+$ ]] || [[ "$2" -lt 1000 ]] || [[ "$2" -gt 300000 ]]; then
                    echo "Warning: Invalid timeout value '$2', using default 30000ms" >&2
                    TIMEOUT_MS="30000"
                else
                    TIMEOUT_MS="$2"
                fi
                shift 2
                ;;
            --wait-ms)
                if [[ ! "$2" =~ ^[0-9]+$ ]] || [[ "$2" -lt 0 ]] || [[ "$2" -gt 60000 ]]; then
                    echo "Warning: Invalid wait-ms value '$2', using default 2000ms" >&2
                    WAIT_MS="2000"
                else
                    WAIT_MS="$2"
                fi
                shift 2
                ;;
            --session)
                SESSION_NAME="$2"
                shift 2
                ;;
            --fullpage)
                FULL_PAGE="true"
                shift
                ;;
            --mobile)
                VIEWPORT_WIDTH="390"
                VIEWPORT_HEIGHT="844"
                shift
                ;;
            *)
                remaining_args_array+=("$1")
                shift
                ;;
        esac
    done
    
    local url="${remaining_args_array[0]:-}"
    if [[ -z "$url" ]]; then
        echo "Error: URL required" >&2
        echo "Usage: browserless performance <url> [--output metrics.json]" >&2
        return 1
    fi
    
    local session_id
    session_id=$(actions::create_temp_session)
    
    log::info "⚡ Measuring performance for $url"
    
    local result
    result=$(browser::execute_js "
        const startTime = performance.now();
        
        // Navigate to the URL
        await page.goto('$url', { 
            waitUntil: 'networkidle2',
            timeout: 30000 
        });
        
        // Get performance metrics
        const metrics = await page.evaluate(() => {
            const perf = performance;
            const navigation = perf.getEntriesByType('navigation')[0];
            
            return {
                // Navigation timing
                dns_lookup: navigation ? navigation.domainLookupEnd - navigation.domainLookupStart : 0,
                tcp_connect: navigation ? navigation.connectEnd - navigation.connectStart : 0,
                request_response: navigation ? navigation.responseEnd - navigation.requestStart : 0,
                dom_interactive: navigation ? navigation.domInteractive - navigation.fetchStart : 0,
                dom_complete: navigation ? navigation.domComplete - navigation.fetchStart : 0,
                load_event: navigation ? navigation.loadEventEnd - navigation.loadEventStart : 0,
                
                // Page metrics
                total_load_time: navigation ? navigation.loadEventEnd - navigation.fetchStart : 0,
                first_paint: null,
                first_contentful_paint: null,
                
                // Resource counts
                resource_count: perf.getEntriesByType('resource').length,
                
                // Memory usage (if available)
                memory_used: navigator.deviceMemory || null,
                memory_limit: navigator.deviceMemory ? navigator.deviceMemory * 1024 * 1024 * 1024 : null,
                
                // Connection info
                connection_type: navigator.connection ? navigator.connection.effectiveType : null,
                connection_downlink: navigator.connection ? navigator.connection.downlink : null
            };
        });
        
        // Try to get paint timing if available
        const paintMetrics = await page.evaluate(() => {
            const paintEntries = performance.getEntriesByType('paint');
            const result = {};
            paintEntries.forEach(entry => {
                if (entry.name === 'first-paint') {
                    result.first_paint = entry.startTime;
                } else if (entry.name === 'first-contentful-paint') {
                    result.first_contentful_paint = entry.startTime;
                }
            });
            return result;
        });
        
        const totalTime = performance.now() - startTime;
        
        return {
            success: true,
            metrics: {
                ...metrics,
                ...paintMetrics,
                total_measurement_time: totalTime
            },
            url: page.url(),
            title: await page.title(),
            timestamp: new Date().toISOString()
        };
    " "$session_id")
    
    local success
    success=$(echo "$result" | jq -r '.success // false')
    if [[ "$success" == "true" ]]; then
        if [[ -n "$OUTPUT_PATH" ]]; then
            mkdir -p "${OUTPUT_PATH%/*}"
            echo "$result" > "$OUTPUT_PATH"
            echo "Performance metrics saved: $OUTPUT_PATH"
        fi
        
        local metrics=$(echo "$result" | jq '.metrics')
        echo "Performance Metrics:"
        echo "  Total Load Time: $(echo "$metrics" | jq -r '.total_load_time // "N/A"')ms"
        echo "  DOM Interactive: $(echo "$metrics" | jq -r '.dom_interactive // "N/A"')ms"
        echo "  DOM Complete: $(echo "$metrics" | jq -r '.dom_complete // "N/A"')ms"
        echo "  First Paint: $(echo "$metrics" | jq -r '.first_paint // "N/A"')ms"
        echo "  First Contentful Paint: $(echo "$metrics" | jq -r '.first_contentful_paint // "N/A"')ms"
        echo "  Resource Count: $(echo "$metrics" | jq -r '.resource_count // "N/A"')"
        echo "  Connection Type: $(echo "$metrics" | jq -r '.connection_type // "N/A"')"
    else
        local error=$(echo "$result" | jq -r '.error // "Unknown error"')
        echo "Error: Performance measurement failed - $error" >&2
        actions::cleanup_temp_session "$session_id"
        return 1
    fi
    
    actions::cleanup_temp_session "$session_id"
    return 0
}

#######################################
# Extract form data and input fields
# Usage: browserless extract-forms <url> [options]
#######################################
actions::extract_forms() {
    # Reset defaults
    OUTPUT_PATH=""
    TIMEOUT_MS="30000"
    WAIT_MS="2000"
    SESSION_NAME=""
    FULL_PAGE="false"
    VIEWPORT_WIDTH="1920"
    VIEWPORT_HEIGHT="1080"
    
    # Parse options directly
    local remaining_args_array=()
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --output)
                OUTPUT_PATH="$2"
                shift 2
                ;;
            --timeout)
                if [[ ! "$2" =~ ^[0-9]+$ ]] || [[ "$2" -lt 1000 ]] || [[ "$2" -gt 300000 ]]; then
                    echo "Warning: Invalid timeout value '$2', using default 30000ms" >&2
                    TIMEOUT_MS="30000"
                else
                    TIMEOUT_MS="$2"
                fi
                shift 2
                ;;
            --wait-ms)
                if [[ ! "$2" =~ ^[0-9]+$ ]] || [[ "$2" -lt 0 ]] || [[ "$2" -gt 60000 ]]; then
                    echo "Warning: Invalid wait-ms value '$2', using default 2000ms" >&2
                    WAIT_MS="2000"
                else
                    WAIT_MS="$2"
                fi
                shift 2
                ;;
            --session)
                SESSION_NAME="$2"
                shift 2
                ;;
            --fullpage)
                FULL_PAGE="true"
                shift
                ;;
            --mobile)
                VIEWPORT_WIDTH="390"
                VIEWPORT_HEIGHT="844"
                shift
                ;;
            *)
                remaining_args_array+=("$1")
                shift
                ;;
        esac
    done
    
    local url="${remaining_args_array[0]:-}"
    if [[ -z "$url" ]]; then
        echo "Error: URL required" >&2
        echo "Usage: browserless extract-forms <url> [--output forms.json]" >&2
        return 1
    fi
    
    local session_id
    session_id=$(actions::create_temp_session)
    
    log::info "📝 Extracting forms from $url"
    
    # Navigate first
    local nav_result
    if nav_result=$(browser::navigate "$url" "$session_id"); then
        local nav_success=$(echo "$nav_result" | jq -r '.success // false')
        if [[ "$nav_success" == "true" ]]; then
            # Wait for page to stabilize
            browser::wait "$WAIT_MS" "$session_id"
            
            # Extract form data
            local result
            if result=$(browser::evaluate "
                const forms = Array.from(document.querySelectorAll('form')).map((form, index) => {
                    const inputs = Array.from(form.querySelectorAll('input, select, textarea')).map(input => ({
                        type: input.type || input.tagName.toLowerCase(),
                        name: input.name || '',
                        id: input.id || '',
                        placeholder: input.placeholder || '',
                        required: input.required || false,
                        value: input.value || '',
                        className: input.className || ''
                    }));
                    
                    return {
                        index: index,
                        action: form.action || '',
                        method: form.method || 'get',
                        id: form.id || '',
                        className: form.className || '',
                        inputs: inputs,
                        inputCount: inputs.length
                    };
                });
                
                return {
                    forms: forms,
                    formCount: forms.length,
                    totalInputs: forms.reduce((sum, form) => sum + form.inputCount, 0)
                };
            " "$session_id"); then
                local eval_success=$(echo "$result" | jq -r '.success // false')
                if [[ "$eval_success" == "true" ]]; then
                    local forms_data=$(echo "$result" | jq '.result')
                    
                    if [[ -n "$OUTPUT_PATH" ]]; then
                        mkdir -p "${OUTPUT_PATH%/*}"
                        echo "$forms_data" | jq '.' > "$OUTPUT_PATH"
                        echo "Forms data saved: $OUTPUT_PATH"
                    fi
                    
                    local form_count=$(echo "$forms_data" | jq -r '.formCount')
                    local total_inputs=$(echo "$forms_data" | jq -r '.totalInputs')
                    echo "Found $form_count forms with $total_inputs total inputs"
                    echo "$forms_data" | jq '.'
                    actions::cleanup_temp_session "$session_id"
                    return 0
                else
                    local error
            error=$(echo "$result" | jq -r '.error // "Unknown error"')
                    echo "Error: Form extraction failed - $error" >&2
                    actions::cleanup_temp_session "$session_id"
                    return 1
                fi
            else
                echo "Error: Could not extract forms" >&2
                actions::cleanup_temp_session "$session_id"
                return 1
            fi
        else
            local error=$(echo "$nav_result" | jq -r '.error // "Unknown error"')
            echo "Error: Navigation failed - $error" >&2
            actions::cleanup_temp_session "$session_id"
            return 1
        fi
    else
        echo "Error: Could not navigate to URL" >&2
        actions::cleanup_temp_session "$session_id"
        return 1
    fi
}

#######################################
# Extract interactive elements with metadata and selector options
# Usage: browserless extract-elements --url <url> --output <file> --screenshot <file> [options]
#######################################
actions::extract_elements() {
    # Reset defaults
    OUTPUT_PATH=""
    local screenshot_path=""
    TIMEOUT_MS="30000"
    WAIT_MS="3000"  # Longer wait for dynamic content
    SESSION_NAME=""
    FULL_PAGE="false"
    VIEWPORT_WIDTH="1920"
    VIEWPORT_HEIGHT="1080"
    
    local url_param=""
    local remaining_args_array=()
    
    # Parse options directly
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --url)
                url_param="$2"
                shift 2
                ;;
            --output)
                OUTPUT_PATH="$2"
                shift 2
                ;;
            --screenshot)
                screenshot_path="$2"
                shift 2
                ;;
            --timeout)
                if [[ ! "$2" =~ ^[0-9]+$ ]] || [[ "$2" -lt 1000 ]] || [[ "$2" -gt 300000 ]]; then
                    echo "Warning: Invalid timeout value '$2', using default 30000ms" >&2
                    TIMEOUT_MS="30000"
                else
                    TIMEOUT_MS="$2"
                fi
                shift 2
                ;;
            --wait-ms)
                if [[ ! "$2" =~ ^[0-9]+$ ]] || [[ "$2" -lt 0 ]] || [[ "$2" -gt 60000 ]]; then
                    echo "Warning: Invalid wait-ms value '$2', using default 3000ms" >&2
                    WAIT_MS="3000"
                else
                    WAIT_MS="$2"
                fi
                shift 2
                ;;
            --session)
                SESSION_NAME="$2"
                shift 2
                ;;
            --fullpage)
                FULL_PAGE="true"
                shift
                ;;
            --mobile)
                VIEWPORT_WIDTH="390"
                VIEWPORT_HEIGHT="844"
                shift
                ;;
            --help|-h)
                echo "Usage: browserless extract-elements --url URL --output FILE --screenshot FILE [OPTIONS]"
                echo ""
                echo "Arguments:"
                echo "  --url URL              Target URL to analyze (required)"
                echo "  --output FILE          Output file for element data (required)"
                echo "  --screenshot FILE      Output file for screenshot (required)"
                echo ""
                echo "Options:"
                echo "  --timeout MS           Timeout in milliseconds (default: 30000)"
                echo "  --wait-ms MS           Wait time after load (default: 3000)"
                echo "  --session NAME         Use persistent session"
                echo "  --fullpage             Capture full page screenshot"
                echo "  --mobile               Use mobile viewport (390x844)"
                echo "  --help, -h             Show this help message"
                echo ""
                echo "Example:"
                echo "  browserless extract-elements --url https://example.com --output elements.json --screenshot page.png"
                return 0
                ;;
            *)
                remaining_args_array+=("$1")
                shift
                ;;
        esac
    done
    
    # Validate required parameters
    if [[ -z "$url_param" ]]; then
        echo "Error: --url is required" >&2
        echo "Usage: browserless extract-elements --url URL --output FILE --screenshot FILE" >&2
        return 1
    fi
    
    if [[ -z "$OUTPUT_PATH" ]]; then
        echo "Error: --output is required" >&2
        echo "Usage: browserless extract-elements --url URL --output FILE --screenshot FILE" >&2
        return 1
    fi
    
    if [[ -z "$screenshot_path" ]]; then
        echo "Error: --screenshot is required" >&2
        echo "Usage: browserless extract-elements --url URL --output FILE --screenshot FILE" >&2
        return 1
    fi
    
    # Smart URL preprocessing
    local url
    url=$(actions::preprocess_url "$url_param")
    log::debug "Preprocessed URL: $url"
    
    # Ensure output directories exist
    if [[ "$OUTPUT_PATH" == */* ]]; then
        mkdir -p "${OUTPUT_PATH%/*}"
    fi
    if [[ "$screenshot_path" == */* ]]; then
        mkdir -p "${screenshot_path%/*}"
    fi
    
    log::info "🔍 Extracting elements from $url"
    
    # Use browserless function API for comprehensive element extraction
    local browserless_port="${BROWSERLESS_PORT:-4110}"
    
    # Create comprehensive JavaScript function for element extraction
    local wrapped_code="export default async ({ page, context }) => {
        try {
            // Set viewport
            await page.setViewport({ width: $VIEWPORT_WIDTH, height: $VIEWPORT_HEIGHT });
            
            // Navigate to the URL
            await page.goto('$url', { 
                waitUntil: 'networkidle2',
                timeout: $TIMEOUT_MS 
            });
            
            // Wait for page to stabilize
            await new Promise(resolve => setTimeout(resolve, $WAIT_MS));
            
            // Wait for document ready
            await page.waitForFunction(() => document.readyState === 'complete', { timeout: 5000 }).catch(() => {});
            
            // Extract page context information
            const pageContext = await page.evaluate(() => {
                const forms = document.querySelectorAll('form');
                const buttons = document.querySelectorAll('button, input[type=\"button\"], input[type=\"submit\"], [role=\"button\"]');
                const links = document.querySelectorAll('a[href]');
                const passwordInputs = document.querySelectorAll('input[type=\"password\"]');
                const searchInputs = document.querySelectorAll('input[placeholder*=\"search\" i], input[name*=\"search\" i], input[id*=\"search\" i]');
                
                return {
                    title: document.title || '',
                    url: window.location.href,
                    hasLogin: passwordInputs.length > 0,
                    hasSearch: searchInputs.length > 0,
                    formCount: forms.length,
                    buttonCount: buttons.length,
                    linkCount: links.length,
                    viewport: {
                        width: window.innerWidth,
                        height: window.innerHeight
                    }
                };
            });
            
            // Extract interactive elements with comprehensive metadata
            const elements = await page.evaluate(() => {
                // Helper function to generate multiple selector options
                function generateSelectors(element) {
                    const selectors = [];
                    
                    // ID selector (highest robustness)
                    if (element.id) {
                        selectors.push({
                            selector: '#' + element.id,
                            type: 'id',
                            robustness: 0.95,
                            fallback: false
                        });
                    }
                    
                    // Data attribute selectors (high robustness)
                    const dataAttrs = ['data-testid', 'data-test', 'data-qa', 'data-automation', 'data-cy'];
                    for (const attr of dataAttrs) {
                        if (element.hasAttribute(attr)) {
                            selectors.push({
                                selector: '[' + attr + '=\"' + element.getAttribute(attr) + '\"]',
                                type: 'data-attr',
                                robustness: 0.9,
                                fallback: false
                            });
                        }
                    }
                    
                    // Name attribute selector (medium-high robustness)
                    if (element.name) {
                        selectors.push({
                            selector: '[name=\"' + element.name + '\"]',
                            type: 'name',
                            robustness: 0.8,
                            fallback: false
                        });
                    }
                    
                    // Semantic class selectors (medium robustness)
                    if (element.className) {
                        const classes = element.className.split(' ').filter(c => c && !c.match(/^[a-z0-9]{6,}$/));
                        if (classes.length > 0) {
                            selectors.push({
                                selector: '.' + classes[0],
                                type: 'class',
                                robustness: 0.6,
                                fallback: false
                            });
                        }
                    }
                    
                    // Text-based selector for buttons and links (medium robustness)
                    const text = element.textContent?.trim();
                    if (text && text.length < 50 && (element.tagName === 'BUTTON' || element.tagName === 'A')) {
                        selectors.push({
                            selector: element.tagName.toLowerCase() + ':contains(\"' + text + '\")',
                            type: 'text',
                            robustness: 0.7,
                            fallback: false
                        });
                    }
                    
                    // Tag + type selector for inputs (medium robustness)
                    if (element.tagName === 'INPUT' && element.type) {
                        selectors.push({
                            selector: 'input[type=\"' + element.type + '\"]',
                            type: 'type',
                            robustness: 0.5,
                            fallback: true
                        });
                    }
                    
                    // CSS path fallback (low robustness)
                    function getPath(el) {
                        if (el.id) return '#' + el.id;
                        if (el === document.body) return 'body';
                        
                        let path = [];
                        while (el && el.nodeType === Node.ELEMENT_NODE) {
                            let selector = el.nodeName.toLowerCase();
                            if (el.id) {
                                selector += '#' + el.id;
                                path.unshift(selector);
                                break;
                            } else {
                                let sibling = el;
                                let nth = 1;
                                while (sibling.previousElementSibling) {
                                    sibling = sibling.previousElementSibling;
                                    if (sibling.nodeName.toLowerCase() === selector) nth++;
                                }
                                if (nth !== 1) selector += ':nth-of-type(' + nth + ')';
                            }
                            path.unshift(selector);
                            el = el.parentNode;
                        }
                        return path.join(' > ');
                    }
                    
                    selectors.push({
                        selector: getPath(element),
                        type: 'path',
                        robustness: 0.3,
                        fallback: true
                    });
                    
                    return selectors;
                }
                
                // Helper function to classify element type
                function classifyElement(element) {
                    const tagName = element.tagName.toLowerCase();
                    const type = element.type?.toLowerCase();
                    const role = element.getAttribute('role')?.toLowerCase();
                    
                    if (tagName === 'button' || type === 'button' || type === 'submit' || role === 'button') {
                        return 'button';
                    } else if (tagName === 'input') {
                        if (type === 'text' || type === 'email' || type === 'password' || !type) {
                            return 'input';
                        } else if (type === 'checkbox' || type === 'radio') {
                            return 'checkbox';
                        } else {
                            return 'input';
                        }
                    } else if (tagName === 'select') {
                        return 'select';
                    } else if (tagName === 'textarea') {
                        return 'textarea';
                    } else if (tagName === 'a') {
                        return 'link';
                    } else if (tagName === 'form') {
                        return 'form';
                    } else {
                        return 'element';
                    }
                }
                
                // Helper function to determine category
                function categorizeElement(element, text) {
                    const tagName = element.tagName.toLowerCase();
                    const type = element.type?.toLowerCase();
                    const classes = element.className.toLowerCase();
                    const textLower = text.toLowerCase();
                    
                    // Authentication patterns
                    if (type === 'password' || textLower.includes('login') || textLower.includes('sign in') || 
                        textLower.includes('register') || textLower.includes('sign up') || classes.includes('auth')) {
                        return 'authentication';
                    }
                    
                    // Navigation patterns
                    if (tagName === 'a' || textLower.includes('menu') || textLower.includes('nav') || 
                        classes.includes('menu') || classes.includes('nav')) {
                        return 'navigation';
                    }
                    
                    // Data entry patterns
                    if (tagName === 'input' || tagName === 'textarea' || tagName === 'select' || 
                        textLower.includes('search') || type === 'email' || type === 'text') {
                        return 'data-entry';
                    }
                    
                    // Action patterns
                    if (textLower.includes('submit') || textLower.includes('save') || textLower.includes('delete') || 
                        textLower.includes('edit') || type === 'submit') {
                        return 'actions';
                    }
                    
                    // Content patterns
                    if (textLower.includes('read more') || textLower.includes('expand') || textLower.includes('filter')) {
                        return 'content';
                    }
                    
                    return 'general';
                }
                
                // Select interactive elements
                const interactiveSelectors = [
                    'button',
                    'input',
                    'select',
                    'textarea',
                    'a[href]',
                    '[onclick]',
                    '[role=\"button\"]',
                    '[role=\"link\"]',
                    '[role=\"menuitem\"]',
                    '[tabindex]',
                    'form'
                ];
                
                const allElements = document.querySelectorAll(interactiveSelectors.join(', '));
                
                return Array.from(allElements).map((element, index) => {
                    const rect = element.getBoundingClientRect();
                    const text = element.textContent?.trim() || '';
                    const placeholder = element.placeholder || '';
                    const ariaLabel = element.getAttribute('aria-label') || '';
                    const title = element.title || '';
                    
                    // Calculate confidence score based on various factors
                    let confidence = 0.5; // Base confidence
                    
                    // Boost confidence for common interactive patterns
                    if (element.tagName === 'BUTTON' || element.type === 'submit') confidence += 0.3;
                    if (element.tagName === 'A' && element.href) confidence += 0.2;
                    if (element.id || element.hasAttribute('data-testid')) confidence += 0.2;
                    if (text.length > 0 && text.length < 100) confidence += 0.1;
                    if (rect.width > 0 && rect.height > 0) confidence += 0.1;
                    
                    // Reduce confidence for hidden or very small elements
                    if (rect.width < 10 || rect.height < 10) confidence -= 0.3;
                    if (getComputedStyle(element).visibility === 'hidden') confidence -= 0.5;
                    if (getComputedStyle(element).display === 'none') confidence -= 0.5;
                    
                    confidence = Math.max(0, Math.min(1, confidence));
                    
                    return {
                        text: text,
                        tagName: element.tagName,
                        type: classifyElement(element),
                        selectors: generateSelectors(element),
                        boundingBox: {
                            x: rect.x,
                            y: rect.y,
                            width: rect.width,
                            height: rect.height
                        },
                        confidence: Math.round(confidence * 100) / 100,
                        category: categorizeElement(element, text),
                        attributes: {
                            id: element.id || '',
                            className: element.className || '',
                            name: element.name || '',
                            type: element.type || '',
                            placeholder: placeholder,
                            'aria-label': ariaLabel,
                            title: title,
                            href: element.href || '',
                            role: element.getAttribute('role') || ''
                        }
                    };
                }).filter(el => el.confidence > 0.1); // Filter out very low confidence elements
            });
            
            // Take screenshot
            const screenshot = await page.screenshot({ 
                encoding: 'base64',
                fullPage: ${FULL_PAGE:-false}
            });
            
            return {
                success: true,
                elements: elements,
                pageContext: pageContext,
                screenshot: screenshot,
                elementCount: elements.length,
                timestamp: new Date().toISOString()
            };
        } catch (error) {
            return { 
                success: false, 
                error: error.message,
                stack: error.stack 
            };
        }
    };"
    
    # Execute via browserless API
    local result
    result=$(curl -s -X POST \
        -H "Content-Type: application/javascript" \
        -d "$wrapped_code" \
        "http://localhost:${browserless_port}/chrome/function" 2>/dev/null)
    
    local success
    success=$(echo "$result" | jq -r '.success // false')
    if [[ "$success" == "true" ]]; then
        # Extract and save screenshot
        local screenshot_data=$(echo "$result" | jq -r '.screenshot')
        echo "$screenshot_data" | base64 -d > "$screenshot_path"
        
        # Create final output with elements and context
        local elements_output
        elements_output=$(echo "$result" | jq '{
            elements: .elements,
            pageContext: .pageContext,
            elementCount: .elementCount,
            timestamp: .timestamp
        }')
        
        # Save elements data
        echo "$elements_output" > "$OUTPUT_PATH"
        
        local element_count=$(echo "$result" | jq -r '.elementCount')
        echo "✅ Extracted $element_count interactive elements"
        echo "Elements data saved: $OUTPUT_PATH"
        echo "Screenshot saved: $screenshot_path"
        
        return 0
    else
        local error=$(echo "$result" | jq -r '.error // "Unknown error"')
        echo "Error: Element extraction failed - $error" >&2
        return 1
    fi
}

#######################################
# Dispatch function for CLI routing
#######################################
actions::dispatch() {
    # Validate at least one argument provided
    if [[ $# -eq 0 ]]; then
        echo "Error: No action specified" >&2
        echo "Usage: browserless <action> [arguments] [options]" >&2
        echo "" >&2
        echo "Available actions:" >&2
        echo "  screenshot     - Take screenshots of URLs" >&2
        echo "  navigate       - Navigate to URLs and get basic info" >&2
        echo "  health-check   - Check if URLs load successfully" >&2
        echo "  element-exists - Check if elements exist on pages" >&2
        echo "  extract-text   - Extract text content from elements" >&2
        echo "  extract        - Extract structured data with custom scripts" >&2
        echo "  extract-forms  - Extract form data and input fields" >&2
        echo "  extract-elements - Extract interactive elements with metadata" >&2
        echo "  interact       - Perform form fills, clicks, and wait operations" >&2
        echo "  console        - Capture console logs from pages" >&2
        echo "  performance    - Measure page performance metrics" >&2
        return 1
    fi
    
    local action="$1"
    shift
    
    # Validate action is not empty
    if [[ -z "$action" ]]; then
        echo "Error: Action cannot be empty" >&2
        return 1
    fi
    
    # Validate action is alphanumeric with hyphens only
    if ! [[ "$action" =~ ^[a-zA-Z0-9-]+$ ]]; then
        echo "Error: Invalid action format: $action" >&2
        echo "Actions must contain only letters, numbers, and hyphens" >&2
        return 1
    fi
    
    case "$action" in
        screenshot)
            actions::screenshot "$@"
            ;;
        navigate)
            actions::navigate "$@"
            ;;
        health-check)
            actions::health_check "$@"
            ;;
        element-exists)
            actions::element_exists "$@"
            ;;
        extract-text)
            actions::extract_text "$@"
            ;;
        extract)
            actions::extract "$@"
            ;;
        extract-forms)
            actions::extract_forms "$@"
            ;;
        extract-elements)
            actions::extract_elements "$@"
            ;;
        interact)
            actions::interact "$@"
            ;;
        console)
            actions::console "$@"
            ;;
        performance)
            actions::performance "$@"
            ;;
        *)
            echo "Error: Unknown action: $action" >&2
            echo "" >&2
            echo "Available actions:" >&2
            echo "  screenshot     - Take screenshots of URLs" >&2
            echo "  navigate       - Navigate to URLs and get basic info" >&2
            echo "  health-check   - Check if URLs load successfully" >&2
            echo "  element-exists - Check if elements exist on pages" >&2
            echo "  extract-text   - Extract text content from elements" >&2
            echo "  extract        - Extract structured data with custom scripts" >&2
            echo "  extract-forms  - Extract form data and input fields" >&2
            echo "  extract-elements - Extract interactive elements with metadata" >&2
            echo "  interact       - Perform form fills, clicks, and wait operations" >&2
            echo "  console        - Capture console logs from pages" >&2
            echo "  performance    - Measure page performance metrics" >&2
            echo "" >&2
            echo "Universal options:" >&2
            echo "  --output <path>    - Save result to file" >&2
            echo "  --timeout <ms>     - Max wait time (default: 30000)" >&2
            echo "  --wait-ms <ms>     - Initial wait before action (default: 2000)" >&2
            echo "  --session <name>   - Use persistent session" >&2
            echo "  --fullpage         - Full page screenshots" >&2
            echo "  --mobile           - Use mobile viewport" >&2
            return 1
            ;;
    esac
}

# Export the dispatch function
export -f actions::dispatch
