#!/bin/bash

# REST API wrapper for Mail-in-a-Box email management

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
MAILINABOX_API_LIB_DIR="${APP_ROOT}/resources/mail-in-a-box/lib"

# Source dependencies
source "$MAILINABOX_API_LIB_DIR/core.sh"
source "$MAILINABOX_API_LIB_DIR/content.sh"
source "${APP_ROOT}/scripts/lib/utils/format.sh"

# API endpoint handler
mailinabox_api_handler() {
    local method="$1"
    local endpoint="$2"
    shift 2
    local params="$@"
    
    case "$endpoint" in
        "/health")
            mailinabox_api_health
            ;;
        "/accounts")
            case "$method" in
                "GET")
                    mailinabox_api_list_accounts
                    ;;
                "POST")
                    mailinabox_api_create_account "$params"
                    ;;
                "DELETE")
                    mailinabox_api_delete_account "$params"
                    ;;
                *)
                    echo '{"error": "Method not allowed"}'
                    return 1
                    ;;
            esac
            ;;
        "/aliases")
            case "$method" in
                "GET")
                    mailinabox_api_list_aliases
                    ;;
                "POST")
                    mailinabox_api_create_alias "$params"
                    ;;
                "DELETE")
                    mailinabox_api_delete_alias "$params"
                    ;;
                *)
                    echo '{"error": "Method not allowed"}'
                    return 1
                    ;;
            esac
            ;;
        "/stats")
            mailinabox_api_get_stats
            ;;
        *)
            echo '{"error": "Endpoint not found"}'
            return 1
            ;;
    esac
}

# Health check endpoint
mailinabox_api_health() {
    local health=$(mailinabox_get_health)
    local status="unhealthy"
    [[ "$health" == "healthy" ]] && status="healthy"
    
    cat <<EOF
{
    "status": "$status",
    "service": "mail-in-a-box",
    "smtp": $(mailinabox_is_running && echo "true" || echo "false"),
    "webmail": $(docker inspect mailinabox-webmail &>/dev/null && echo "true" || echo "false")
}
EOF
}

# List email accounts
mailinabox_api_list_accounts() {
    if ! mailinabox_is_running; then
        echo '{"error": "Service not running"}'
        return 1
    fi
    
    local accounts=$(mailinabox_list_accounts | grep -E "^[^@]+@[^@]+$" | jq -R . | jq -s .)
    echo "{\"accounts\": ${accounts:-[]}}"
}

# Create email account
mailinabox_api_create_account() {
    local email="$1"
    local password="$2"
    
    if [[ -z "$email" ]] || [[ -z "$password" ]]; then
        echo '{"error": "Email and password required"}'
        return 1
    fi
    
    if mailinabox_add_content "$email" "$password"; then
        echo "{\"success\": true, \"email\": \"$email\"}"
    else
        echo '{"error": "Failed to create account"}'
        return 1
    fi
}

# Delete email account
mailinabox_api_delete_account() {
    local email="$1"
    
    if [[ -z "$email" ]]; then
        echo '{"error": "Email required"}'
        return 1
    fi
    
    if mailinabox_remove_content "$email"; then
        echo "{\"success\": true, \"email\": \"$email\"}"
    else
        echo '{"error": "Failed to delete account"}'
        return 1
    fi
}

# List email aliases
mailinabox_api_list_aliases() {
    if ! mailinabox_is_running; then
        echo '{"error": "Service not running"}'
        return 1
    fi
    
    # For now, return empty array as aliases are not fully implemented
    echo '{"aliases": []}'
}

# Create email alias
mailinabox_api_create_alias() {
    local alias="$1"
    local target="$2"
    
    if [[ -z "$alias" ]] || [[ -z "$target" ]]; then
        echo '{"error": "Alias and target email required"}'
        return 1
    fi
    
    # Call the content.sh function for alias creation
    if docker exec "$MAILINABOX_CONTAINER_NAME" addalias "$alias" "$target" 2>/dev/null; then
        echo "{\"success\": true, \"alias\": \"$alias\", \"target\": \"$target\"}"
    else
        echo '{"error": "Failed to create alias"}'
        return 1
    fi
}

# Delete email alias
mailinabox_api_delete_alias() {
    local alias="$1"
    
    if [[ -z "$alias" ]]; then
        echo '{"error": "Alias required"}'
        return 1
    fi
    
    if docker exec "$MAILINABOX_CONTAINER_NAME" delalias "$alias" 2>/dev/null; then
        echo "{\"success\": true, \"alias\": \"$alias\"}"
    else
        echo '{"error": "Failed to delete alias"}'
        return 1
    fi
}

# Get email statistics
mailinabox_api_get_stats() {
    if ! mailinabox_is_running; then
        echo '{"error": "Service not running"}'
        return 1
    fi
    
    # Get basic stats from postfix logs
    local sent=$(docker exec "$MAILINABOX_CONTAINER_NAME" grep -c "status=sent" /var/log/mail/mail.log 2>/dev/null || echo "0")
    local received=$(docker exec "$MAILINABOX_CONTAINER_NAME" grep -c "postfix/smtpd" /var/log/mail/mail.log 2>/dev/null || echo "0")
    local queued=$(docker exec "$MAILINABOX_CONTAINER_NAME" postqueue -p 2>/dev/null | grep -c "^[A-F0-9]" || echo "0")
    
    cat <<EOF
{
    "stats": {
        "sent_today": $sent,
        "received_today": $received,
        "queued": $queued,
        "accounts": $(mailinabox_list_content | grep -c "@" || echo "0")
    }
}
EOF
}

# Simple HTTP server for API (for testing)
mailinabox_api_serve() {
    local port="${1:-8543}"
    
    format::bold_green "Starting Mail-in-a-Box API server on port $port..."
    
    while true; do
        { echo -ne "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\n\r\n"; mailinabox_api_handler "GET" "/health"; } | nc -l -p "$port" -q 1
    done
}