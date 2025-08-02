#!/bin/bash
# ====================================================================
# Health Check Module Index
# ====================================================================
#
# Sources all resource-specific health check implementations
#
# ====================================================================

# Get the directory of this script
HEALTH_CHECKS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source port mappings
source "$(dirname "$HEALTH_CHECKS_DIR")/port-mappings.conf"

# Source individual health check implementations
source "$HEALTH_CHECKS_DIR/ollama.sh"
source "$HEALTH_CHECKS_DIR/postgres.sh"
source "$HEALTH_CHECKS_DIR/redis.sh"
source "$HEALTH_CHECKS_DIR/agent-s2.sh"
source "$HEALTH_CHECKS_DIR/generic.sh"  # Includes multiple resource checks

# Source category-specific health check modules
source "$HEALTH_CHECKS_DIR/ai-health-checks.sh"
source "$HEALTH_CHECKS_DIR/storage-health-checks.sh"
source "$HEALTH_CHECKS_DIR/automation-health-checks.sh"
source "$HEALTH_CHECKS_DIR/agents-health-checks.sh"
source "$HEALTH_CHECKS_DIR/search-health-checks.sh"
source "$HEALTH_CHECKS_DIR/execution-health-checks.sh"

# Special resource health checks that don't fit the standard pattern
check_windmill_health() {
    # Windmill uses port 5681 and exposes /api/version endpoint
    if curl -s --max-time 5 "http://localhost:5681/api/version" >/dev/null 2>&1; then
        echo "healthy"
    else
        echo "unreachable"
    fi
}

check_unstructured_io_health() {
    # Unstructured-IO uses port 11450 and exposes /healthcheck endpoint
    if curl -s --max-time 5 "http://localhost:11450/healthcheck" >/dev/null 2>&1; then
        echo "healthy"
    else
        echo "unreachable"
    fi
}

check_claude_code_health() {
    # Claude Code is a CLI tool, check if it responds to basic commands
    if which claude >/dev/null 2>&1; then
        if timeout 5 claude --version >/dev/null 2>&1; then
            echo "healthy"
        else
            echo "unreachable"
        fi
    elif timeout 10 npx @anthropic-ai/claude-code --version >/dev/null 2>&1; then
        echo "healthy"
    else
        echo "unreachable"
    fi
}

check_questdb_health() {
    # QuestDB uses port 9010 and exposes HTTP API
    if curl -s --max-time 5 "http://localhost:9010/status" >/dev/null 2>&1; then
        echo "healthy"
    elif curl -s --max-time 5 "http://localhost:9010/" >/dev/null 2>&1; then
        echo "healthy"
    else
        echo "unreachable"
    fi
}