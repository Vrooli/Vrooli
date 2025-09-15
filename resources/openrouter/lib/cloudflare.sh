#!/bin/bash
# Cloudflare AI Gateway integration for OpenRouter

# Handle different sourcing contexts
if [[ -n "${BASH_SOURCE[0]}" ]]; then
    APP_ROOT="${APP_ROOT:-$(builtin cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && builtin pwd)}"
else
    APP_ROOT="${APP_ROOT:-$(pwd)}"
fi

# Source dependencies if available
if [[ -f "${APP_ROOT}/scripts/lib/utils/log.sh" ]]; then
    source "${APP_ROOT}/scripts/lib/utils/log.sh"
else
    # Define basic log functions if not available
    log::info() { echo "[INFO] $*"; }
    log::error() { echo "[ERROR] $*" >&2; }
    log::success() { echo "[SUCCESS] $*"; }
    log::warn() { echo "[WARN] $*"; }
fi

if [[ -f "${APP_ROOT}/scripts/lib/utils/format.sh" ]]; then
    source "${APP_ROOT}/scripts/lib/utils/format.sh"
fi

# Cloudflare AI Gateway configuration
export CLOUDFLARE_GATEWAY_ENABLED="${CLOUDFLARE_GATEWAY_ENABLED:-false}"
export CLOUDFLARE_GATEWAY_URL="${CLOUDFLARE_GATEWAY_URL:-}"
export CLOUDFLARE_GATEWAY_ID="${CLOUDFLARE_GATEWAY_ID:-}"
export CLOUDFLARE_ACCOUNT_ID="${CLOUDFLARE_ACCOUNT_ID:-}"
export CLOUDFLARE_API_TOKEN="${CLOUDFLARE_API_TOKEN:-}"

# Check if Cloudflare AI Gateway is configured
openrouter::cloudflare::is_configured() {
    if [[ "$CLOUDFLARE_GATEWAY_ENABLED" != "true" ]]; then
        return 1
    fi
    
    # Check if cloudflare-ai-gateway resource is available
    if ! command -v resource-cloudflare-ai-gateway >/dev/null 2>&1; then
        return 1
    fi
    
    # Check if necessary environment variables are set
    if [[ -z "$CLOUDFLARE_GATEWAY_URL" ]] && [[ -z "$CLOUDFLARE_GATEWAY_ID" ]]; then
        return 1
    fi
    
    return 0
}

# Get the Cloudflare Gateway URL
openrouter::cloudflare::get_gateway_url() {
    local base_url="$1"
    local model="$2"
    
    if ! openrouter::cloudflare::is_configured; then
        echo "$base_url"
        return 0
    fi
    
    # If gateway URL is explicitly set, use it
    if [[ -n "$CLOUDFLARE_GATEWAY_URL" ]]; then
        echo "$CLOUDFLARE_GATEWAY_URL"
        return 0
    fi
    
    # Construct gateway URL from account and gateway IDs
    if [[ -n "$CLOUDFLARE_ACCOUNT_ID" && -n "$CLOUDFLARE_GATEWAY_ID" ]]; then
        echo "https://gateway.ai.cloudflare.com/v1/${CLOUDFLARE_ACCOUNT_ID}/${CLOUDFLARE_GATEWAY_ID}/openrouter"
        return 0
    fi
    
    # Fallback to original URL
    echo "$base_url"
}

# Configure Cloudflare AI Gateway
openrouter::cloudflare::configure() {
    local enabled="${1:-false}"
    local gateway_url="${2:-}"
    local gateway_id="${3:-}"
    local account_id="${4:-}"
    
    # Update configuration
    export CLOUDFLARE_GATEWAY_ENABLED="$enabled"
    
    if [[ -n "$gateway_url" ]]; then
        export CLOUDFLARE_GATEWAY_URL="$gateway_url"
    fi
    
    if [[ -n "$gateway_id" ]]; then
        export CLOUDFLARE_GATEWAY_ID="$gateway_id"
    fi
    
    if [[ -n "$account_id" ]]; then
        export CLOUDFLARE_ACCOUNT_ID="$account_id"
    fi
    
    # Try to load API token from vault if available
    if command -v resource-vault >/dev/null 2>&1; then
        local cf_token
        cf_token=$(resource-vault content get --path "resources/cloudflare/api/token" --key "value" --format raw 2>/dev/null || true)
        
        if [[ -n "$cf_token" && "$cf_token" != "No value found"* ]]; then
            export CLOUDFLARE_API_TOKEN="$cf_token"
            log::info "Cloudflare API token loaded from vault"
        fi
    fi
    
    # Save configuration to file
    local config_file="${APP_ROOT}/data/openrouter/cloudflare-config.json"
    mkdir -p "$(dirname "$config_file")"
    
    cat > "$config_file" <<EOF
{
    "enabled": $enabled,
    "gateway_url": "${gateway_url:-null}",
    "gateway_id": "${gateway_id:-null}",
    "account_id": "${account_id:-null}",
    "configured_at": "$(date -Iseconds)"
}
EOF
    
    if [[ "$enabled" == "true" ]]; then
        log::success "Cloudflare AI Gateway integration enabled"
        if openrouter::cloudflare::is_configured; then
            log::info "Gateway URL: $(openrouter::cloudflare::get_gateway_url "$OPENROUTER_API_BASE" "")"
        else
            log::warn "Cloudflare AI Gateway enabled but not fully configured"
        fi
    else
        log::info "Cloudflare AI Gateway integration disabled"
    fi
}

# Load Cloudflare configuration from file
openrouter::cloudflare::load_config() {
    local config_file="${APP_ROOT}/data/openrouter/cloudflare-config.json"
    
    if [[ ! -f "$config_file" ]]; then
        return 0  # Return success even if file doesn't exist
    fi
    
    # Load configuration from file
    local enabled gateway_url gateway_id account_id
    enabled=$(jq -r '.enabled // false' "$config_file" 2>/dev/null || echo "false")
    gateway_url=$(jq -r '.gateway_url // empty' "$config_file" 2>/dev/null || true)
    gateway_id=$(jq -r '.gateway_id // empty' "$config_file" 2>/dev/null || true)
    account_id=$(jq -r '.account_id // empty' "$config_file" 2>/dev/null || true)
    
    # Apply configuration
    export CLOUDFLARE_GATEWAY_ENABLED="$enabled"
    
    if [[ -n "$gateway_url" && "$gateway_url" != "null" ]]; then
        export CLOUDFLARE_GATEWAY_URL="$gateway_url"
    fi
    
    if [[ -n "$gateway_id" && "$gateway_id" != "null" ]]; then
        export CLOUDFLARE_GATEWAY_ID="$gateway_id"
    fi
    
    if [[ -n "$account_id" && "$account_id" != "null" ]]; then
        export CLOUDFLARE_ACCOUNT_ID="$account_id"
    fi
    
    return 0
}

# Show Cloudflare AI Gateway status
openrouter::cloudflare::status() {
    local format="${1:-text}"
    
    # Load configuration
    openrouter::cloudflare::load_config
    
    local status="disabled"
    local gateway_url="Not configured"
    local is_available="false"
    
    if [[ "$CLOUDFLARE_GATEWAY_ENABLED" == "true" ]]; then
        status="enabled"
        
        if openrouter::cloudflare::is_configured; then
            gateway_url=$(openrouter::cloudflare::get_gateway_url "$OPENROUTER_API_BASE" "")
            is_available="true"
        else
            status="enabled (not configured)"
        fi
    fi
    
    if [[ "$format" == "json" ]]; then
        cat <<EOF
{
    "cloudflare_gateway": {
        "status": "$status",
        "enabled": $CLOUDFLARE_GATEWAY_ENABLED,
        "configured": $is_available,
        "gateway_url": "$gateway_url",
        "gateway_id": "${CLOUDFLARE_GATEWAY_ID:-null}",
        "account_id": "${CLOUDFLARE_ACCOUNT_ID:-null}"
    }
}
EOF
    else
        echo "Cloudflare AI Gateway Integration:"
        echo "  Status: $status"
        echo "  Enabled: $CLOUDFLARE_GATEWAY_ENABLED"
        echo "  Configured: $is_available"
        echo "  Gateway URL: $gateway_url"
        
        if [[ -n "$CLOUDFLARE_GATEWAY_ID" ]]; then
            echo "  Gateway ID: $CLOUDFLARE_GATEWAY_ID"
        fi
        
        if [[ -n "$CLOUDFLARE_ACCOUNT_ID" ]]; then
            echo "  Account ID: $CLOUDFLARE_ACCOUNT_ID"
        fi
    fi
}

# Test Cloudflare AI Gateway connection
openrouter::cloudflare::test() {
    if ! openrouter::cloudflare::is_configured; then
        log::error "Cloudflare AI Gateway is not configured"
        return 1
    fi
    
    local gateway_url
    gateway_url=$(openrouter::cloudflare::get_gateway_url "$OPENROUTER_API_BASE" "")
    
    log::info "Testing Cloudflare AI Gateway at: $gateway_url"
    
    # Test basic connectivity
    if timeout 5 curl -sf "$gateway_url/models" >/dev/null 2>&1; then
        log::success "Cloudflare AI Gateway is accessible"
        return 0
    else
        log::error "Failed to connect to Cloudflare AI Gateway"
        return 1
    fi
}

# CLI command handler for cloudflare management
openrouter::cloudflare::cli() {
    local subcommand="${1:-status}"
    shift || true
    
    case "$subcommand" in
        status)
            openrouter::cloudflare::status "$@"
            ;;
        configure)
            local enabled="${1:-false}"
            local gateway_url="${2:-}"
            local gateway_id="${3:-}"
            local account_id="${4:-}"
            
            if [[ "$enabled" == "--help" ]] || [[ "$enabled" == "-h" ]]; then
                echo "Usage: resource-openrouter cloudflare configure <enabled> [gateway_url] [gateway_id] [account_id]"
                echo ""
                echo "Configure Cloudflare AI Gateway integration"
                echo ""
                echo "Arguments:"
                echo "  enabled       true/false to enable/disable integration"
                echo "  gateway_url   (Optional) Full gateway URL"
                echo "  gateway_id    (Optional) Gateway ID for URL construction"
                echo "  account_id    (Optional) Cloudflare account ID"
                echo ""
                echo "Examples:"
                echo "  # Enable with explicit URL"
                echo "  resource-openrouter cloudflare configure true https://gateway.ai.cloudflare.com/v1/abc123/mygateway/openrouter"
                echo ""
                echo "  # Enable with IDs (URL will be constructed)"
                echo "  resource-openrouter cloudflare configure true \"\" mygateway abc123"
                echo ""
                echo "  # Disable"
                echo "  resource-openrouter cloudflare configure false"
                return 0
            fi
            
            openrouter::cloudflare::configure "$enabled" "$gateway_url" "$gateway_id" "$account_id"
            ;;
        test)
            openrouter::cloudflare::test
            ;;
        --help|-h|help)
            echo "Usage: resource-openrouter cloudflare <subcommand> [options]"
            echo ""
            echo "Manage Cloudflare AI Gateway integration"
            echo ""
            echo "Subcommands:"
            echo "  status      Show current Cloudflare AI Gateway configuration"
            echo "  configure   Configure Cloudflare AI Gateway settings"
            echo "  test        Test Cloudflare AI Gateway connectivity"
            echo ""
            echo "Use 'resource-openrouter cloudflare <subcommand> --help' for more information"
            ;;
        *)
            log::error "Unknown cloudflare subcommand: $subcommand"
            echo "Use 'resource-openrouter cloudflare --help' for available commands"
            return 1
            ;;
    esac
}

# Initialize Cloudflare integration on load
openrouter::cloudflare::load_config