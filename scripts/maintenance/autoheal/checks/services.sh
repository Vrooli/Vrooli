#!/usr/bin/env bash
# =============================================================================
# Critical Service Checks
# Cloudflare tunnel, display manager, Docker daemon, systemd-resolved
# =============================================================================

# =============================================================================
# CLOUDFLARE TUNNEL CHECKS
# =============================================================================

check_cloudflared_service() {
    [[ "$CHECK_CLOUDFLARED" != "true" ]] && return 0

    log "DEBUG" "Checking cloudflared service status"

    if ! systemctl is-active "$CLOUDFLARED_SERVICE" >/dev/null 2>&1; then
        log "ERROR" "Cloudflared service is not running"
        restart_cloudflared
        return $?
    fi

    log "DEBUG" "Cloudflared service is running"
    return 0
}

check_cloudflared_tunnel_connectivity() {
    [[ "$CHECK_CLOUDFLARED" != "true" ]] && return 0

    log "DEBUG" "Checking Cloudflare tunnel connectivity"

    # Test local port connectivity (tunnel should be able to reach this)
    if ! curl -sf --max-time 5 "http://localhost:${CLOUDFLARED_TEST_PORT}/health" >/dev/null 2>&1; then
        log "WARN" "Cannot reach test port ${CLOUDFLARED_TEST_PORT} - target may be down"
        return 0  # Not a tunnel issue if the target is down
    fi

    # If external tunnel URL is provided, test end-to-end
    if [[ -n "$CLOUDFLARED_TUNNEL_URL" ]]; then
        log "DEBUG" "Testing external tunnel access: $CLOUDFLARED_TUNNEL_URL"

        if curl -sf --max-time 10 "${CLOUDFLARED_TUNNEL_URL}/health" >/dev/null 2>&1; then
            log "DEBUG" "Cloudflare tunnel end-to-end connectivity OK"
            return 0
        else
            log "ERROR" "Cloudflare tunnel end-to-end connectivity FAILED"
            log "INFO" "Local service is up but tunnel is not forwarding traffic"
            restart_cloudflared
            return $?
        fi
    fi

    # Check for connection errors in recent logs
    local recent_errors
    recent_errors=$(journalctl -u "$CLOUDFLARED_SERVICE" --since "5 minutes ago" 2>/dev/null | grep -c "ERR" || true)

    if [[ "$recent_errors" -gt 10 ]]; then
        log "WARN" "Cloudflared has $recent_errors errors in last 5 minutes"
        log "INFO" "Restarting cloudflared due to high error count"
        restart_cloudflared
        return $?
    fi

    return 0
}

restart_cloudflared() {
    log "INFO" "Restarting cloudflared service"

    if sudo systemctl restart "$CLOUDFLARED_SERVICE" 2>/dev/null; then
        log "INFO" "Cloudflared service restarted"

        # Wait and verify
        sleep "$VERIFY_DELAY"

        if systemctl is-active "$CLOUDFLARED_SERVICE" >/dev/null 2>&1; then
            log "INFO" "Cloudflared service is healthy after restart"
            return 0
        else
            log "ERROR" "Cloudflared service failed to start"
            return 1
        fi
    else
        log "ERROR" "Failed to restart cloudflared service (permission denied or service not found)"
        return 1
    fi
}

# =============================================================================
# DISPLAY MANAGER CHECKS
# =============================================================================

check_display_manager() {
    [[ "$CHECK_DISPLAY" != "true" ]] && return 0

    log "DEBUG" "Checking display manager ($DISPLAY_MANAGER) status"

    # Check if display manager service exists
    if ! systemctl list-unit-files | grep -q "^${DISPLAY_MANAGER}.service"; then
        log "DEBUG" "Display manager $DISPLAY_MANAGER not installed, skipping"
        return 0
    fi

    if ! systemctl is-active "$DISPLAY_MANAGER" >/dev/null 2>&1; then
        log "ERROR" "Display manager $DISPLAY_MANAGER is not running"

        if [[ "$DISPLAY_AUTO_RESTART" == "true" ]]; then
            log "WARN" "Auto-restarting display manager (this will disconnect all desktop sessions!)"
            restart_display_manager
            return $?
        else
            log "WARN" "Display manager is down but auto-restart is disabled (set VROOLI_AUTOHEAL_DISPLAY_RESTART=true to enable)"
            return 1
        fi
    fi

    log "DEBUG" "Display manager is running"

    # Optional: Test if X11 display is responsive
    if [[ -n "${DISPLAY:-}" ]] && command -v xdpyinfo >/dev/null 2>&1; then
        if timeout 3 xdpyinfo >/dev/null 2>&1; then
            log "DEBUG" "X11 display $DISPLAY is responsive"
        else
            log "WARN" "X11 display $DISPLAY is not responding"
        fi
    fi

    return 0
}

restart_display_manager() {
    log "INFO" "Restarting display manager $DISPLAY_MANAGER"
    log "WARN" "This will disconnect all active desktop sessions!"

    if sudo systemctl restart "$DISPLAY_MANAGER" 2>/dev/null; then
        log "INFO" "Display manager restarted"
        sleep 5  # Give it time to start

        if systemctl is-active "$DISPLAY_MANAGER" >/dev/null 2>&1; then
            log "INFO" "Display manager is healthy after restart"
            return 0
        else
            log "ERROR" "Display manager failed to start"
            return 1
        fi
    else
        log "ERROR" "Failed to restart display manager"
        return 1
    fi
}

# =============================================================================
# SYSTEMD-RESOLVED CHECK
# =============================================================================

check_systemd_resolved() {
    [[ "$CHECK_SYSTEMD_RESOLVED" != "true" ]] && return 0

    log "DEBUG" "Checking systemd-resolved status"

    if ! systemctl is-active systemd-resolved >/dev/null 2>&1; then
        log "WARN" "systemd-resolved is not running"

        log "INFO" "Attempting to start systemd-resolved"
        if sudo systemctl start systemd-resolved 2>/dev/null; then
            log "INFO" "systemd-resolved started successfully"
            return 0
        else
            log "ERROR" "Failed to start systemd-resolved"
            return 1
        fi
    fi

    log "DEBUG" "systemd-resolved is running"
    return 0
}

# =============================================================================
# DOCKER DAEMON CHECK
# =============================================================================

check_docker_daemon() {
    [[ "$CHECK_DOCKER_DAEMON" != "true" ]] && return 0

    log "DEBUG" "Checking Docker daemon health"

    # Basic docker command test
    if ! docker info >/dev/null 2>&1; then
        log "ERROR" "Docker daemon is not responding"

        # Check if docker service is running
        if ! systemctl is-active docker >/dev/null 2>&1; then
            log "ERROR" "Docker service is not running"
            log "INFO" "Attempting to start docker service"

            if sudo systemctl start docker 2>/dev/null; then
                sleep 5
                if docker info >/dev/null 2>&1; then
                    log "INFO" "Docker daemon recovered"
                    return 0
                fi
            fi

            log "ERROR" "Failed to recover Docker daemon"
            return 1
        fi

        # Service is running but not responding - may need restart
        log "INFO" "Docker service is running but not responding - attempting restart"
        if sudo systemctl restart docker 2>/dev/null; then
            sleep 10  # Docker takes time to restart
            if docker info >/dev/null 2>&1; then
                log "INFO" "Docker daemon recovered after restart"
                return 0
            fi
        fi

        log "ERROR" "Docker daemon restart failed"
        return 1
    fi

    log "DEBUG" "Docker daemon is healthy"
    return 0
}
