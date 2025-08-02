#!/bin/bash
# ====================================================================
# Agent-S2 Health Check
# ====================================================================

check_agent_s2_health() {
    # Agent-S2 doesn't expose a port, check docker health
    local health_status
    health_status=$(docker inspect agent-s2 --format='{{.State.Health.Status}}' 2>/dev/null)
    
    case "$health_status" in
        "healthy")
            echo "healthy"
            ;;
        "starting")
            echo "starting"
            ;;
        *)
            echo "unhealthy"
            ;;
    esac
}