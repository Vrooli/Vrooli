#!/bin/bash
set -euo pipefail

# CrewAI Injection Script
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
INJECT_LIB_DIR="${APP_ROOT}/resources/crewai/lib"

# Source utilities and core
source "${INJECT_LIB_DIR}/core.sh"
source "${INJECT_LIB_DIR}/status.sh"

# Show usage
show_usage() {
    log::header "CrewAI Injection"
    echo ""
    echo "Usage: resource-crewai inject <directory>"
    echo ""
    echo "Injects CrewAI crews and agents from a directory."
    echo ""
    echo "Expected directory structure:"
    echo "  <directory>/"
    echo "    crews/       # Directory containing crew Python files"
    echo "      *.py       # Each file should define a 'crew' object"
    echo "    agents/      # Directory containing agent Python files"
    echo "      *.py       # Each file should define agent classes"
    echo ""
    echo "Example:"
    echo "  resource-crewai inject /path/to/crewai/project"
    echo ""
    echo "Individual file injection:"
    echo "  resource-crewai inject --crew /path/to/crew.py"
    echo "  resource-crewai inject --agent /path/to/agent.py"
}

# Inject a single crew file
inject_crew_file() {
    local crew_file="$1"
    local crew_name=$(basename "$crew_file" .py)
    local dest_file="${CREWAI_CREWS_DIR}/${crew_name}.py"
    
    if [[ ! -f "$crew_file" ]]; then
        log::error "Crew file not found: $crew_file"
        return 1
    fi
    
    log::info "Injecting crew: ${crew_name}"
    cp "$crew_file" "$dest_file"
    
    # Validate the crew file (basic Python syntax check)
    if python3 -m py_compile "$dest_file" 2>/dev/null; then
        log::success "  ✅ Crew injected: ${crew_name}"
        
        # Notify server via API
        if is_running; then
            local response=$(curl -s -X POST -H "Content-Type: application/json" \
                -d "{\"file_path\":\"$dest_file\",\"file_type\":\"crew\"}" \
                "http://localhost:${CREWAI_PORT}/inject" 2>/dev/null)
            if [[ "$response" == *"injected"* ]]; then
                log::success "  ✅ Crew registered in server"
            fi
        fi
    else
        log::error "  ❌ Invalid Python file"
        rm -f "$dest_file"
        return 1
    fi
}

# Inject a single agent file
inject_agent_file() {
    local agent_file="$1"
    local agent_name=$(basename "$agent_file" .py)
    local dest_file="${CREWAI_AGENTS_DIR}/${agent_name}.py"
    
    if [[ ! -f "$agent_file" ]]; then
        log::error "Agent file not found: $agent_file"
        return 1
    fi
    
    log::info "Injecting agent: ${agent_name}"
    cp "$agent_file" "$dest_file"
    
    # Validate the agent file (basic Python syntax check)
    if python3 -m py_compile "$dest_file" 2>/dev/null; then
        log::success "  ✅ Agent injected: ${agent_name}"
        
        # Notify server via API
        if is_running; then
            local response=$(curl -s -X POST -H "Content-Type: application/json" \
                -d "{\"file_path\":\"$dest_file\",\"file_type\":\"agent\"}" \
                "http://localhost:${CREWAI_PORT}/inject" 2>/dev/null)
            if [[ "$response" == *"injected"* ]]; then
                log::success "  ✅ Agent registered in server"
            fi
        fi
    else
        log::error "  ❌ Invalid Python file"
        rm -f "$dest_file"
        return 1
    fi
}

# Inject from directory
inject_from_directory() {
    local source_dir="$1"
    
    if [[ ! -d "$source_dir" ]]; then
        log::error "Directory not found: $source_dir"
        return 1
    fi
    
    log::header "Injecting from: $source_dir"
    
    local crews_injected=0
    local agents_injected=0
    
    # Inject crews
    if [[ -d "${source_dir}/crews" ]]; then
        log::info "Injecting crews..."
        for crew_file in "${source_dir}/crews"/*.py; do
            if [[ -f "$crew_file" ]]; then
                if inject_crew_file "$crew_file"; then
                    ((crews_injected++))
                fi
            fi
        done
    fi
    
    # Inject agents
    if [[ -d "${source_dir}/agents" ]]; then
        log::info "Injecting agents..."
        for agent_file in "${source_dir}/agents"/*.py; do
            if [[ -f "$agent_file" ]]; then
                if inject_agent_file "$agent_file"; then
                    ((agents_injected++))
                fi
            fi
        done
    fi
    
    # Summary
    echo ""
    log::success "Injection complete!"
    log::info "  🚢 Crews injected: ${crews_injected}"
    log::info "  🤖 Agents injected: ${agents_injected}"
}

# Main injection logic
main() {
    # Check if CrewAI is installed
    if ! check_installation; then
        log::error "CrewAI is not installed"
        log::info "Run 'resource-crewai install' first"
        exit 1
    fi
    
    # Initialize directories
    init_directories
    
    # Parse arguments
    if [[ $# -eq 0 ]] || [[ "$1" == "--help" ]] || [[ "$1" == "-h" ]]; then
        show_usage
        exit 0
    fi
    
    case "$1" in
        --crew)
            shift
            inject_crew_file "$1"
            ;;
        --agent)
            shift
            inject_agent_file "$1"
            ;;
        *)
            inject_from_directory "$1"
            ;;
    esac
}

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi