#!/usr/bin/env bash
# =============================================================================
# Infrastructure Health Checks
# Network connectivity, DNS resolution, time synchronization
# =============================================================================

check_network_connectivity() {
    [[ "$CHECK_NETWORK" != "true" ]] && return 0

    log "DEBUG" "Checking network connectivity to $PING_TEST_IP"

    if ping -c 1 -W 2 "$PING_TEST_IP" >/dev/null 2>&1; then
        log "DEBUG" "Network connectivity OK"
        return 0
    else
        log "ERROR" "Network connectivity FAILED - cannot ping $PING_TEST_IP"
        log "WARN" "Cannot auto-fix network connectivity issues - manual intervention required"
        return 1
    fi
}

check_dns_resolution() {
    [[ "$CHECK_DNS" != "true" ]] && return 0

    log "DEBUG" "Checking DNS resolution for $DNS_TEST_DOMAIN"

    if getent hosts "$DNS_TEST_DOMAIN" >/dev/null 2>&1; then
        log "DEBUG" "DNS resolution OK"
        return 0
    else
        log "ERROR" "DNS resolution FAILED for $DNS_TEST_DOMAIN"

        # Try restarting systemd-resolved if available
        if systemctl is-active systemd-resolved >/dev/null 2>&1; then
            log "INFO" "Attempting to restart systemd-resolved"
            if sudo systemctl restart systemd-resolved 2>/dev/null; then
                sleep 2
                if getent hosts "$DNS_TEST_DOMAIN" >/dev/null 2>&1; then
                    log "INFO" "DNS resolution recovered after systemd-resolved restart"
                    return 0
                fi
            fi
        fi

        log "WARN" "DNS issues persist - manual intervention may be required"
        return 1
    fi
}

check_time_sync() {
    [[ "$CHECK_TIME_SYNC" != "true" ]] && return 0

    log "DEBUG" "Checking time synchronization"

    if command -v timedatectl >/dev/null 2>&1; then
        local sync_status
        sync_status=$(timedatectl show -p NTPSynchronized --value 2>/dev/null || echo "unknown")

        if [[ "$sync_status" == "yes" ]]; then
            log "DEBUG" "Time synchronization OK"
            return 0
        else
            log "WARN" "Time synchronization status: $sync_status"

            # Try to enable NTP if disabled
            if timedatectl show -p NTP --value 2>/dev/null | grep -q "no"; then
                log "INFO" "Attempting to enable NTP"
                sudo timedatectl set-ntp true 2>/dev/null || true
            fi

            return 1
        fi
    else
        log "DEBUG" "timedatectl not available, skipping time sync check"
        return 0
    fi
}
