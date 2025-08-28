#!/usr/bin/env bash
# CrewAI Content Management Functions

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"

# Content management functions
crewai::content::add() {
    log::info "Use 'content inject' to add crews and agents to CrewAI"
    echo "Usage: resource-crewai content inject <directory>"
    echo "       resource-crewai content inject --crew <file.py>"
    echo "       resource-crewai content inject --agent <file.py>"
}

crewai::content::list() {
    list_crews
    echo ""
    list_agents
}

crewai::content::get() {
    local name="${1:-}"
    if [[ -z "$name" ]]; then
        log::error "Please specify crew/agent name"
        return 1
    fi
    
    if [[ -f "${CREWAI_CREWS_DIR}/${name}.py" ]]; then
        cat "${CREWAI_CREWS_DIR}/${name}.py"
    elif [[ -f "${CREWAI_AGENTS_DIR}/${name}.py" ]]; then
        cat "${CREWAI_AGENTS_DIR}/${name}.py"
    else
        log::error "Crew/agent not found: $name"
        return 1
    fi
}

crewai::content::remove() {
    local name="${1:-}"
    if [[ -z "$name" ]]; then
        log::error "Please specify crew/agent name to remove"
        return 1
    fi
    
    local removed=false
    if [[ -f "${CREWAI_CREWS_DIR}/${name}.py" ]]; then
        rm -f "${CREWAI_CREWS_DIR}/${name}.py"
        log::success "Removed crew: $name"
        removed=true
    fi
    if [[ -f "${CREWAI_AGENTS_DIR}/${name}.py" ]]; then
        rm -f "${CREWAI_AGENTS_DIR}/${name}.py"
        log::success "Removed agent: $name"
        removed=true
    fi
    
    if [[ "$removed" == "false" ]]; then
        log::error "Crew/agent not found: $name"
        return 1
    fi
}

crewai::content::execute() {
    log::info "CrewAI execution through API endpoints"
    if is_running; then
        curl -s "http://localhost:${CREWAI_PORT}/health" | python3 -m json.tool 2>/dev/null || echo "Server running on port ${CREWAI_PORT}"
    else
        log::error "CrewAI server is not running. Start it with: resource-crewai manage start"
        return 1
    fi
}

crewai::content::inject() {
    # Delegate to the existing inject script
    "${APP_ROOT}/resources/crewai/lib/inject.sh" "$@"
}