#\!/usr/bin/env bash
# Codex CLI Interface

set -euo pipefail

# Get the actual directory of this script (resolving symlinks)
SCRIPT_PATH="${BASH_SOURCE[0]}"
if [[ -L "$SCRIPT_PATH" ]]; then
    SCRIPT_PATH="$(readlink -f "$SCRIPT_PATH")"
fi
CODEX_CLI_DIR="$(cd "$(dirname "$SCRIPT_PATH")" && pwd)"

# Source libraries
# shellcheck disable=SC1091
source "${CODEX_CLI_DIR}/lib/common.sh"
# shellcheck disable=SC1091
source "${CODEX_CLI_DIR}/lib/status.sh"
# shellcheck disable=SC1091
source "${CODEX_CLI_DIR}/lib/inject.sh" 2>/dev/null || true

#######################################
# Display help message
#######################################
codex::help() {
    cat <<EOF
Codex Resource CLI

Usage: $(basename "$0") <command> [options]

Commands:
  status [--format json]    Check Codex status
  start                     Start Codex service (mark as running)
  stop                      Stop Codex service (mark as stopped)
  inject <file>            Inject a Python script for Codex processing
  list                     List injected scripts
  run <script>             Run a specific script with Codex
  help                     Show this help message

Options:
  --format json            Output in JSON format (for status command)

Examples:
  $(basename "$0") status
  $(basename "$0") start
  $(basename "$0") inject my_script.py
  $(basename "$0") list
  $(basename "$0") run generate_function.py
EOF
}

#######################################
# Start Codex service
#######################################
codex::start() {
    log::info "Starting Codex service..."
    
    if ! codex::is_configured; then
        log::error "Codex not configured. Please set OPENAI_API_KEY or configure Vault"
        return 1
    fi
    
    codex::save_status "running"
    log::success "Codex service marked as running"
    
    # Verify API is accessible
    if codex::is_available; then
        log::success "Codex API is accessible"
    else
        log::warn "Codex API not responding. Check your API key and network connection"
    fi
}

#######################################
# Stop Codex service
#######################################
codex::stop() {
    log::info "Stopping Codex service..."
    codex::save_status "stopped"
    log::success "Codex service marked as stopped"
}

#######################################
# List injected scripts
#######################################
codex::list() {
    log::header "Injected Codex Scripts"
    
    if [[ ! -d "${CODEX_SCRIPTS_DIR}" ]]; then
        log::info "No scripts directory found"
        return 0
    fi
    
    local count=0
    while IFS= read -r script; do
        ((count++))
        local basename
        basename=$(basename "${script}")
        echo "  ${count}. ${basename}"
    done < <(find "${CODEX_SCRIPTS_DIR}" -name "*.py" -type f 2>/dev/null | sort)
    
    if [[ ${count} -eq 0 ]]; then
        log::info "No scripts found"
    else
        log::info "Total: ${count} script(s)"
    fi
}

#######################################
# Main command dispatcher
#######################################
main() {
    local command="${1:-help}"
    shift || true
    
    case "${command}" in
        status)
            codex::status "$@"
            ;;
        start)
            codex::start "$@"
            ;;
        stop)
            codex::stop "$@"
            ;;
        inject)
            if type -t codex::inject &>/dev/null; then
                codex::inject "$@"
            else
                log::error "Inject command not yet implemented"
                exit 1
            fi
            ;;
        list)
            codex::list "$@"
            ;;
        run)
            if type -t codex::run &>/dev/null; then
                codex::run "$@"
            else
                log::error "Run command not yet implemented"
                exit 1
            fi
            ;;
        help|--help|-h)
            codex::help
            ;;
        *)
            log::error "Unknown command: ${command}"
            codex::help
            exit 1
            ;;
    esac
}

# Run main function
main "$@"
