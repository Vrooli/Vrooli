#!/usr/bin/env bash
# Core functionality for Cline resource (v2.0 contract compliance)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Source common dependencies
source "${SCRIPT_DIR}/common.sh"
source "${SCRIPT_DIR}/config.sh"
source "${RESOURCE_DIR}/config/defaults.sh"

# Lifecycle Management
cline::install() {
    source "${SCRIPT_DIR}/install.sh"
    cline::install::main "$@"
}

cline::start() {
    source "${SCRIPT_DIR}/start.sh"
    cline::start::main "$@"
}

cline::stop() {
    source "${SCRIPT_DIR}/stop.sh"
    cline::stop::main "$@"
}

cline::restart() {
    cline::stop "$@"
    sleep 2
    cline::start "$@"
}

cline::uninstall() {
    log::info "Uninstalling Cline..."
    
    # Stop if running
    cline::stop || true
    
    # Remove configuration directory
    if [[ -d "${CLINE_CONFIG_DIR}" ]]; then
        rm -rf "${CLINE_CONFIG_DIR}"
        log::success "Configuration removed"
    fi
    
    # Remove cache directory
    if [[ -d "${CLINE_CACHE_DIR}" ]]; then
        rm -rf "${CLINE_CACHE_DIR}"
        log::success "Cache removed"
    fi
    
    log::success "Cline uninstalled"
}

# Status Management
cline::status() {
    source "${SCRIPT_DIR}/status.sh"
    cline::status::main "$@"
}

cline::logs() {
    source "${SCRIPT_DIR}/logs.sh"
    cline::logs::main "$@"
}

# Content Management
cline::content() {
    source "${SCRIPT_DIR}/content.sh"
    cline::content::main "$@"
}

# Integration Management
cline::integrate() {
    source "${SCRIPT_DIR}/integrate.sh"
    cline::integrate::main "$@"
}

# Cache Management
cline::cache() {
    source "${SCRIPT_DIR}/cache.sh"
    cline::cache::main "$@"
}

# Testing
cline::test() {
    source "${SCRIPT_DIR}/test.sh"
    cline::test::main "$@"
}

# Health Check
cline::health_check() {
    local timeout="${1:-5}"
    
    # Check configuration exists
    if [[ ! -d "${CLINE_CONFIG_DIR}" ]]; then
        return 1
    fi
    
    # Check provider is accessible
    local provider=$(cline::config::get_provider 2>/dev/null || echo "")
    if [[ -z "$provider" ]]; then
        return 1
    fi
    
    # Provider-specific health check
    case "$provider" in
        ollama)
            timeout "$timeout" curl -sf http://localhost:11434/api/version &>/dev/null || return 1
            ;;
        openrouter)
            # OpenRouter doesn't need a health check
            return 0
            ;;
        *)
            return 1
            ;;
    esac
    
    return 0
}

# Info Command
cline::info() {
    cat <<EOF
{
  "name": "cline",
  "version": "$(cline::get_version 2>/dev/null || echo "unknown")",
  "description": "AI coding assistant for VS Code",
  "category": "agents",
  "ports": {},
  "dependencies": ["ollama"],
  "capabilities": [
    "ai-assistant",
    "code-completion",
    "multi-provider",
    "caching",
    "integrations"
  ],
  "runtime": $(cat "${RESOURCE_DIR}/config/runtime.json" 2>/dev/null || echo '{}')
}
EOF
}

# Version Management
cline::get_version() {
    if command -v code &>/dev/null; then
        code --list-extensions | grep -i "saoudrizwan.claude-dev" | cut -d'@' -f2 || echo "not installed"
    else
        echo "not installed"
    fi
}

# Main entry point for direct execution
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    command="${1:-}"
    shift || true
    
    case "$command" in
        install)
            cline::install "$@"
            ;;
        start)
            cline::start "$@"
            ;;
        stop)
            cline::stop "$@"
            ;;
        restart)
            cline::restart "$@"
            ;;
        uninstall)
            cline::uninstall "$@"
            ;;
        status)
            cline::status "$@"
            ;;
        logs)
            cline::logs "$@"
            ;;
        content)
            cline::content "$@"
            ;;
        integrate)
            cline::integrate "$@"
            ;;
        cache)
            cline::cache "$@"
            ;;
        test)
            cline::test "$@"
            ;;
        health)
            cline::health_check "$@"
            ;;
        info)
            cline::info
            ;;
        version)
            cline::get_version
            ;;
        *)
            echo "Usage: $0 {install|start|stop|restart|uninstall|status|logs|content|integrate|cache|test|health|info|version}"
            exit 1
            ;;
    esac
fi