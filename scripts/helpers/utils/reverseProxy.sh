#!/usr/bin/env bash
# Utilities for managing the Caddy reverse proxy service.
set -euo pipefail

UTILS_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${UTILS_DIR}/log.sh"
# shellcheck disable=SC1091
source "${UTILS_DIR}/flow.sh"
# shellcheck disable=SC1091
source "${UTILS_DIR}/system.sh"

CADDY_CONFIG_DIR="/etc/caddy"
CADDY_MAIN_CONFIG_FILE="${CADDY_CONFIG_DIR}/Caddyfile"
CADDY_SNIPPETS_DIR="${CADDY_CONFIG_DIR}/conf.d"
APP_CADDY_SNIPPET="${CADDY_SNIPPETS_DIR}/vrooli.caddy"

# Ensures the main Caddyfile imports snippets from conf.d
ensure_caddy_import_configured() {
    # Ensure the snippets directory exists so import works
    flow::maybe_run_sudo mkdir -p "$CADDY_SNIPPETS_DIR"
    if [ ! -f "$CADDY_MAIN_CONFIG_FILE" ]; then
        log::info "Main Caddyfile ($CADDY_MAIN_CONFIG_FILE) does not exist. Creating with import directive."
        echo -e "import ${CADDY_SNIPPETS_DIR}/*.caddy" | flow::maybe_run_sudo tee "$CADDY_MAIN_CONFIG_FILE" > /dev/null
        return
    fi

    if ! flow::maybe_run_sudo grep -q "import ${CADDY_SNIPPETS_DIR}/\*.caddy" "$CADDY_MAIN_CONFIG_FILE"; then
        log::info "Adding import directive to $CADDY_MAIN_CONFIG_FILE"
        echo -e "\nimport ${CADDY_SNIPPETS_DIR}/*.caddy" | flow::maybe_run_sudo tee -a "$CADDY_MAIN_CONFIG_FILE" > /dev/null
    else
        log::info "Caddyfile import directive already present."
    fi
}

# Generates a Caddyfile snippet for the application
# Usage: generate_caddy_snippet <domain> <target_port>
generate_caddy_snippet() {
    local domain_list="${1:?Domain list argument required}"
    local target_port_server="${2:?Target server port argument required}"
    local target_port_ui="${3:?Target UI port argument required}"

    # Ensure the snippets directory exists
    flow::maybe_run_sudo mkdir -p "$CADDY_SNIPPETS_DIR"

    log::info "Generating Caddy snippet for ${domain_list}: API -> localhost:${target_port_server}, UI -> localhost:${target_port_ui}"
    local snippet_content
    snippet_content=$(cat <<EOF
${domain_list} {
    # Route API requests to the backend server
    route /api/* {
        reverse_proxy localhost:${target_port_server}
    }

    # Route all other requests to the UI dev server
    route {
        reverse_proxy localhost:${target_port_ui}
    }

    # Optional: Add more directives here, like logging, headers, etc.
    # log {
    #   output file /var/log/caddy/${domain_list}.log
    # }
}
EOF
)

    echo "$snippet_content" | flow::maybe_run_sudo tee "$APP_CADDY_SNIPPET" > /dev/null
    log::success "Generated Caddy snippet: $APP_CADDY_SNIPPET"
}

# Removes any auto_https off directive so that Caddy can manage TLS automatically
ensure_auto_https_enabled() {
    if flow::maybe_run_sudo grep -q "auto_https off" "$CADDY_MAIN_CONFIG_FILE"; then
        log::info "Enabling automatic HTTPS by removing 'auto_https off' directive"
        flow::maybe_run_sudo sed -i '/auto_https off/d' "$CADDY_MAIN_CONFIG_FILE"
    fi
}

# Starts or reloads the Caddy service
start_or_reload_caddy() {
    ensure_caddy_import_configured
    ensure_auto_https_enabled

    log::info "Attempting to start/reload Caddy service..."
    if systemctl is-active --quiet caddy; then
        log::info "Caddy is active, attempting reload."
        # Validate config before reloading
        if flow::maybe_run_sudo caddy validate --config "$CADDY_MAIN_CONFIG_FILE" --adapter caddyfile; then
            flow::maybe_run_sudo systemctl reload caddy
            log::success "Caddy reloaded successfully."
        else
            flow::exit_with_error "Caddy configuration validation failed. Check $APP_CADDY_SNIPPET and $CADDY_MAIN_CONFIG_FILE." "$ERROR_CONFIGURATION"
        fi
    else
        log::info "Caddy is not active, attempting start."
        flow::maybe_run_sudo systemctl start caddy
        if systemctl is-active --quiet caddy; then
            log::success "Caddy started successfully."
        else
            flow::exit_with_error "Failed to start Caddy service." "$ERROR_SERVICE_START_FAILED"
        fi
    fi
}

# Stops the Caddy service
stop_caddy_proxy() {
    log::info "Stopping Caddy service..."
    if systemctl is-active --quiet caddy; then
        flow::maybe_run_sudo systemctl stop caddy
        log::success "Caddy stopped."
    else
        log::info "Caddy service was not running."
    fi
}

# Checks the status of the Caddy service
check_caddy_status() {
    if systemctl is-active --quiet caddy; then
        log::info "Caddy service is active."
        return 0
    else
        log::info "Caddy service is inactive."
        return 1
    fi
}

start_reverse_proxy() {
    local domain="${1:-}"
    local port_server="${2:-}"
    local port_ui="${3:-}"

    if [[ -z "$domain" ]]; then
        flow::exit_with_error "Domain is required" "$ERROR_USAGE"
    fi
    if [[ -z "$port_server" || -z "$port_ui" ]]; then
        flow::exit_with_error "Both Server and UI Ports are required" "$ERROR_USAGE"
    fi

    log::header "Starting reverse proxy for $domain (API: $port_server, UI: $port_ui)"
    ensure_caddy_import_configured
    generate_caddy_snippet "$domain" "$port_server" "$port_ui"
    start_or_reload_caddy
    log::success "Reverse proxy started for $domain (API: $port_server, UI: $port_ui)"
}

stop_reverse_proxy() {
    log::header "Stopping reverse proxy"
    stop_caddy_proxy
    log::success "Reverse proxy stopped"
}

check_reverse_proxy_status() {
    check_caddy_status
}