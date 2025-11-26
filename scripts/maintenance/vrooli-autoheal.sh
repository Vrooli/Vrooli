#!/usr/bin/env bash
set -eo pipefail

# =============================================================================
# Vrooli Autoheal - Modular Edition
# Comprehensive monitoring and auto-recovery for Vrooli infrastructure
# =============================================================================

# Determine script directory for sourcing modules
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
AUTOHEAL_DIR="${SCRIPT_DIR}/autoheal"

# Trap handlers for cleanup on exit/error
trap 'cleanup_on_exit' EXIT
trap 'cleanup_on_error $? $LINENO' ERR

cleanup_on_exit() {
    cleanup_lock 2>/dev/null || true
}

cleanup_on_error() {
    local exit_code=$1
    local line_num=$2
    log "ERROR" "Script failed at line $line_num with exit code $exit_code" 2>/dev/null || true
    cleanup_lock 2>/dev/null || true
}

# Source all modules in dependency order
source "${AUTOHEAL_DIR}/config.sh"
source "${AUTOHEAL_DIR}/utils.sh"
source "${AUTOHEAL_DIR}/recovery/cleanup.sh"
source "${AUTOHEAL_DIR}/checks/infrastructure.sh"
source "${AUTOHEAL_DIR}/checks/services.sh"
source "${AUTOHEAL_DIR}/checks/system.sh"
source "${AUTOHEAL_DIR}/checks/api.sh"
source "${AUTOHEAL_DIR}/checks/resources.sh"
source "${AUTOHEAL_DIR}/checks/scenarios.sh"

# =============================================================================
# MAIN EXECUTION - Orchestrates all checks in priority order
# =============================================================================

main() {
    ensure_prereqs

    LOCK_METHOD=""
    if ! acquire_lock; then
        return 0
    fi

    log_section "🔧 Vrooli Autoheal - Starting Run"
    log "INFO" "Grace period: ${GRACE_SECONDS}s | Infrastructure: $ENABLE_INFRASTRUCTURE_CHECKS | Services: $ENABLE_SERVICE_CHECKS | System: $ENABLE_SYSTEM_CHECKS"
    sleep "$GRACE_SECONDS"

    # Phase 1: Infrastructure Checks (network, DNS, time)
    if [[ "$ENABLE_INFRASTRUCTURE_CHECKS" == "true" ]]; then
        log_section "Phase 1: Infrastructure Health"
        check_network_connectivity || true
        check_dns_resolution || true
        check_time_sync || true
    fi

    # Phase 2: Critical Service Checks (cloudflared, display, docker)
    if [[ "$ENABLE_SERVICE_CHECKS" == "true" ]]; then
        log_section "Phase 2: Critical Services"
        check_cloudflared_service || true
        check_cloudflared_tunnel_connectivity || true
        check_display_manager || true
        check_systemd_resolved || true
        check_docker_daemon || true
    fi

    # Phase 3: System Resource Checks (disk, swap, zombies)
    if [[ "$ENABLE_SYSTEM_CHECKS" == "true" ]]; then
        log_section "Phase 3: System Resources"
        check_disk_space || true
        check_inode_usage || true
        check_swap_usage || true
        check_zombie_processes || true
        check_port_exhaustion || true
        check_certificate_expiration || true
    fi

    # Phase 4: API Health Check (before resource/scenario checks)
    log_section "Phase 4: Vrooli API Health"
    check_api_health || true

    # Phase 5: Resource Checks
    log_section "Phase 5: Resource Health"
    local resources=()
    while IFS= read -r item; do
        resources+=("$item")
    done < <(parse_list "$RESOURCE_LIST")

    local item
    for item in "${resources[@]}"; do
        [[ -z "$item" ]] && continue
        check_resource "$item" || true
    done

    # Phase 6: Scenario Checks
    log_section "Phase 6: Scenario Health"
    local scenarios=()
    while IFS= read -r item; do
        scenarios+=("$item")
    done < <(parse_list "$SCENARIO_LIST")

    for item in "${scenarios[@]}"; do
        [[ -z "$item" ]] && continue
        check_scenario "$item" || true
    done

    log_section "✅ Autoheal Run Complete"
    cleanup_lock
}

main "$@"
