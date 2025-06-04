#!/usr/bin/env bash
set -euo pipefail
DESCRIPTION="Sets up the firewall to safely allow traffic to the server"

SETUP_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${SETUP_DIR}/../utils/env.sh"
# shellcheck disable=SC1091
source "${SETUP_DIR}/../utils/flow.sh"
# shellcheck disable=SC1091
source "${SETUP_DIR}/../utils/log.sh"
# shellcheck disable=SC1091
source "${SETUP_DIR}/../utils/system.sh"

# Check if host has internet access. Exits with error if no access.
firewall::setup() {
    # Use argument or environment variable
    local environment="${1:-$ENVIRONMENT}"
    if [[ -z "$environment" ]]; then
        flow::exit_with_error "Environment is required to setup firewall" "$ERROR_USAGE"
    fi

    if ! flow::can_run_sudo; then
        log::warning "Skipping firewall setup due to sudo mode"
        return
    fi
    # Skip firewall setup if ufw isn't available
    if ! system::is_command "ufw"; then
        log::warning "ufw not found; skipping firewall setup"
        return
    fi

    log::header "🔥🧱 Setting up firewall in $environment environment..."
    
    # Track if any changes were made
    local changes_made=false

    # Cache verbose status for initial checks
    local status_verbose
    status_verbose=$(sudo ufw status verbose)

    # 1) Enable UFW only if not already active
    if echo "$status_verbose" | grep -q "^Status: active"; then
        log::info "UFW already active"
    else
        log::info "Enabling UFW"
        sudo ufw --force enable
        changes_made=true
        # Update status after enabling
        status_verbose=$(sudo ufw status verbose)
    fi

    # 2) Apply default policies only if they differ
    local defaults
    defaults=$(echo "$status_verbose" | grep "^Default:")
    if echo "$defaults" | grep -q "deny (incoming).*allow (outgoing)"; then
        log::info "Default policies already set"
    else
        log::info "Setting default policies"
        sudo ufw default allow outgoing
        sudo ufw default deny incoming
        changes_made=true
    fi

    # 3) Only open required ports using a loop to minimize status calls
    local ports=("80/tcp" "443/tcp" "22/tcp")
    if env::in_development "$environment"; then
        ports+=("${PORT_UI:-3000}/tcp" "${PORT_SERVER:-5329}/tcp")
    fi

    # Cache plain status for rule checks
    local status_plain
    status_plain=$(sudo ufw status)

    for port_proto in "${ports[@]}"; do
        if echo "$status_plain" | grep -qw "$port_proto"; then
            log::info "Rule for $port_proto already exists"
        else
            log::info "Allowing $port_proto"
            sudo ufw allow "$port_proto"
            changes_made=true
        fi
    done

    # 4) Reload sysctl and UFW only if changes were made
    if $changes_made; then
        log::info "Reloading sysctl and UFW"
        sudo sysctl -p >/dev/null 2>&1
        sudo ufw reload >/dev/null 2>&1
    else
        log::info "No firewall changes needed; skipping reload"
    fi

    log::success "Firewall setup complete"
}

# If this script is run directly, invoke its main function.
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    firewall::setup "$@"
fi