#!/usr/bin/env bash
################################################################################
# Vrooli Firewall Management
# 
# Automatically configures firewall rules to allow Docker containers to reach
# native (non-Docker) services running on the host.
#
# This solves the common Linux Docker issue where containers on custom networks
# cannot reach host services due to iptables/netfilter restrictions.
#
# Usage:
#   firewall.sh [setup|clean|status]
################################################################################

set -euo pipefail

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
FIREWALL_DIR="${APP_ROOT}/scripts/lib/firewall"
VROOLI_ROOT="${VROOLI_ROOT:-$APP_ROOT}"

# Source dependencies
# shellcheck disable=SC1091
source "${VROOLI_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE:-${VROOLI_ROOT}/scripts/lib/utils/log.sh}"
# shellcheck disable=SC1091
source "${VROOLI_ROOT}/scripts/resources/port_registry.sh"

# Configuration
readonly FIREWALL_CHAIN="VROOLI-DOCKER"
readonly DOCKER_USER_CHAIN="DOCKER-USER"
readonly COMMENT_PREFIX="vrooli-resource"

#######################################
# Check if running with sufficient privileges
# Returns: 0 if root/sudo, 1 otherwise
#######################################
firewall::check_privileges() {
    if [[ $EUID -ne 0 ]]; then
        log::error "This script must be run with sudo privileges"
        log::info "Please run: sudo $0 $*"
        return 1
    fi
    return 0
}

#######################################
# Check if iptables is available
# Returns: 0 if available, 1 otherwise
#######################################
firewall::check_iptables() {
    if ! command -v iptables &>/dev/null; then
        log::error "iptables not found. Please install iptables."
        return 1
    fi
    return 0
}

#######################################
# Create custom chain if it doesn't exist
# Returns: 0 on success
#######################################
firewall::create_chain() {
    # Check if chain exists
    if ! iptables -n -L "$FIREWALL_CHAIN" &>/dev/null 2>&1; then
        log::info "Creating firewall chain: $FIREWALL_CHAIN"
        iptables -N "$FIREWALL_CHAIN" 2>/dev/null || true
    fi
    
    # Ensure our chain is referenced in DOCKER-USER
    if ! iptables -C "$DOCKER_USER_CHAIN" -j "$FIREWALL_CHAIN" &>/dev/null 2>&1; then
        log::info "Adding $FIREWALL_CHAIN to $DOCKER_USER_CHAIN"
        iptables -I "$DOCKER_USER_CHAIN" 1 -j "$FIREWALL_CHAIN"
    fi
}

#######################################
# Detect if a service is running natively (not in Docker)
# Arguments:
#   $1 - service name
# Returns: 0 if native, 1 if Docker or not running
#######################################
firewall::is_native_service() {
    local service="$1"
    local port="${RESOURCE_PORTS[$service]:-}"
    
    if [[ -z "$port" ]]; then
        return 1
    fi
    
    # Check if port is in use
    if ! netstat -tln 2>/dev/null | grep -q ":${port}[[:space:]]"; then
        return 1  # Not running
    fi
    
    # Check if a Docker container is using this port
    if docker ps --format "table {{.Ports}}" 2>/dev/null | grep -q ":${port}->"; then
        return 1  # Running in Docker
    fi
    
    # Service is running natively
    return 0
}

#######################################
# Get list of enabled services from service.json
# Returns: Space-separated list of service names
#######################################
firewall::get_enabled_services() {
    local service_file="${VROOLI_ROOT}/.vrooli/service.json"
    
    if [[ ! -f "$service_file" ]]; then
        return
    fi
    
    # Extract enabled services from flattened resources structure
    # The structure is now: .resources.ollama, .resources.postgres, etc.
    # We get the key names (service names) directly
    jq -r '.resources | to_entries[] | select(.value.enabled == true) | .key' "$service_file" 2>/dev/null || true
}

#######################################
# Add firewall rule for a service
# Arguments:
#   $1 - service name
#   $2 - port number
# Returns: 0 on success
#######################################
firewall::add_rule() {
    local service="$1"
    local port="$2"
    local comment="${COMMENT_PREFIX}-${service}"
    
    # Check if rule already exists
    if iptables -C "$FIREWALL_CHAIN" -p tcp --dport "$port" -j ACCEPT -m comment --comment "$comment" &>/dev/null 2>&1; then
        log::debug "Rule already exists for $service on port $port"
        return 0
    fi
    
    # Add the rule
    log::info "Adding firewall rule: Allow Docker containers to reach $service on port $port"
    iptables -A "$FIREWALL_CHAIN" -p tcp --dport "$port" -j ACCEPT -m comment --comment "$comment"
}

#######################################
# Remove firewall rule for a service
# Arguments:
#   $1 - service name
#   $2 - port number
# Returns: 0 on success
#######################################
firewall::remove_rule() {
    local service="$1"
    local port="$2"
    local comment="${COMMENT_PREFIX}-${service}"
    
    # Remove the rule if it exists
    while iptables -C "$FIREWALL_CHAIN" -p tcp --dport "$port" -j ACCEPT -m comment --comment "$comment" &>/dev/null 2>&1; do
        log::info "Removing firewall rule for $service on port $port"
        iptables -D "$FIREWALL_CHAIN" -p tcp --dport "$port" -j ACCEPT -m comment --comment "$comment"
    done
}

#######################################
# Setup firewall rules for all native services
# Returns: 0 on success
#######################################
firewall::setup() {
    log::header "Setting up Vrooli firewall rules"
    
    # Check prerequisites
    firewall::check_privileges || return 1
    firewall::check_iptables || return 1
    
    # Create our custom chain
    firewall::create_chain
    
    # Get enabled services
    local enabled_services
    enabled_services=$(firewall::get_enabled_services)
    
    if [[ -z "$enabled_services" ]]; then
        log::warn "No enabled services found in service.json"
        return 0
    fi
    
    local rules_added=0
    
    # Process each enabled service
    for service in $enabled_services; do
        local port="${RESOURCE_PORTS[$service]:-}"
        
        if [[ -z "$port" ]]; then
            log::debug "No port defined for service: $service"
            continue
        fi
        
        # Check if service is running natively
        if firewall::is_native_service "$service"; then
            firewall::add_rule "$service" "$port"
            ((rules_added++))
        else
            log::debug "$service is not running natively (port $port)"
            # Remove rule if it exists (service might have been moved to Docker)
            firewall::remove_rule "$service" "$port" 2>/dev/null || true
        fi
    done
    
    if [[ $rules_added -gt 0 ]]; then
        log::success "Added $rules_added firewall rule(s)"
    else
        log::info "No native services require firewall rules"
    fi
    
    # Special case: Always allow Ollama if it's running
    # (since it's commonly run natively for GPU access)
    if firewall::is_native_service "ollama"; then
        firewall::add_rule "ollama" "${RESOURCE_PORTS[ollama]}"
    fi
    
    log::success "Firewall setup complete"
}

#######################################
# Clean all Vrooli firewall rules
# Returns: 0 on success
#######################################
firewall::clean() {
    log::header "Cleaning Vrooli firewall rules"
    
    # Check prerequisites
    firewall::check_privileges || return 1
    firewall::check_iptables || return 1
    
    # Remove reference from DOCKER-USER chain
    if iptables -C "$DOCKER_USER_CHAIN" -j "$FIREWALL_CHAIN" &>/dev/null 2>&1; then
        log::info "Removing $FIREWALL_CHAIN from $DOCKER_USER_CHAIN"
        iptables -D "$DOCKER_USER_CHAIN" -j "$FIREWALL_CHAIN"
    fi
    
    # Flush and delete our chain
    if iptables -n -L "$FIREWALL_CHAIN" &>/dev/null 2>&1; then
        log::info "Removing firewall chain: $FIREWALL_CHAIN"
        iptables -F "$FIREWALL_CHAIN"
        iptables -X "$FIREWALL_CHAIN"
    fi
    
    log::success "Firewall rules cleaned"
}

#######################################
# Show current firewall status
# Returns: 0 on success
#######################################
firewall::status() {
    log::header "Vrooli Firewall Status"
    
    # Check if our chain exists
    if ! iptables -n -L "$FIREWALL_CHAIN" &>/dev/null 2>&1; then
        log::info "Vrooli firewall chain not configured"
        return 0
    fi
    
    # Show rules in our chain
    log::info "Current firewall rules:"
    iptables -n -L "$FIREWALL_CHAIN" -v --line-numbers
    
    # Show which services are allowed
    log::info ""
    log::info "Services with firewall access:"
    iptables -n -L "$FIREWALL_CHAIN" 2>/dev/null | grep "ACCEPT" | while read -r line; do
        if [[ "$line" =~ dpt:([0-9]+) ]]; then
            local port="${BASH_REMATCH[1]}"
            # Find service name for this port
            for service in "${!RESOURCE_PORTS[@]}"; do
                if [[ "${RESOURCE_PORTS[$service]}" == "$port" ]]; then
                    echo "  - $service (port $port)"
                    break
                fi
            done
        fi
    done
}

#######################################
# Main entry point
#######################################
main() {
    local action="${1:-setup}"
    
    case "$action" in
        setup)
            firewall::setup
            ;;
        clean|cleanup|remove)
            firewall::clean
            ;;
        status|show|list)
            firewall::status
            ;;
        help|--help|-h)
            cat << EOF
Vrooli Firewall Management

Usage: $0 [action]

Actions:
  setup    - Configure firewall rules for native services (default)
  clean    - Remove all Vrooli firewall rules
  status   - Show current firewall configuration

This script manages iptables rules to allow Docker containers to reach
native services running on the host. It automatically detects which
services are running natively vs in Docker and configures rules accordingly.

Example:
  sudo $0 setup     # Configure firewall for all native services
  sudo $0 status    # Show current rules
  sudo $0 clean     # Remove all rules

Note: Requires sudo/root privileges to modify iptables.
EOF
            ;;
        *)
            log::error "Unknown action: $action"
            echo "Use '$0 help' for usage information"
            exit 1
            ;;
    esac
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi