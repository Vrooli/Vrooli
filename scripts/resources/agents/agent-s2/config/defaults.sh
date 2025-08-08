#!/usr/bin/env bash
# Agent-S2 Resource Configuration Defaults
# This file contains all configuration constants and defaults for the Agent-S2 resource

# Source shared secrets management library
# Use the same project root detection method as the secrets library
_agents2_defaults_detect_project_root() {
    local current_dir
    current_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    
    # Walk up directory tree looking for .vrooli directory
    while [[ "$current_dir" != "/" ]]; do
        if [[ -d "$current_dir/.vrooli" ]]; then
            echo "$current_dir"
            return 0
        fi
        current_dir="$(dirname "$current_dir")"
    done
    
    # Fallback: assume we're in scripts and go up to project root
    echo "/home/matthalloran8/Vrooli"
}

PROJECT_ROOT="$(_agents2_defaults_detect_project_root)"
# shellcheck disable=SC1091
source "$PROJECT_ROOT/scripts/lib/service/secrets.sh"
# Agent S2 Configuration Defaults
# All configuration constants and default values

#######################################
# Export configuration constants
# Idempotent - safe to call multiple times
#######################################
agents2::export_config() {
    # Service configuration (only set if not already defined)
    if [[ -z "${AGENTS2_PORT:-}" ]]; then
        readonly AGENTS2_PORT="${AGENTS2_CUSTOM_PORT:-$(resources::get_default_port "agent-s2")}"
    fi
    if [[ -z "${AGENTS2_BASE_URL:-}" ]]; then
        readonly AGENTS2_BASE_URL="http://localhost:${AGENTS2_PORT}"
    fi
    if [[ -z "${AGENTS2_VNC_PORT:-}" ]]; then
        readonly AGENTS2_VNC_PORT="${AGENTS2_CUSTOM_VNC_PORT:-5900}"
    fi
    if [[ -z "${AGENTS2_VNC_URL:-}" ]]; then
        readonly AGENTS2_VNC_URL="vnc://localhost:${AGENTS2_VNC_PORT}"
    fi
    if [[ -z "${AGENTS2_CONTAINER_NAME:-}" ]]; then
        readonly AGENTS2_CONTAINER_NAME="agent-s2"
    fi
    if [[ -z "${AGENTS2_DATA_DIR:-}" ]]; then
        readonly AGENTS2_DATA_DIR="${HOME}/.agent-s2"
    fi
    if [[ -z "${AGENTS2_IMAGE_NAME:-}" ]]; then
        readonly AGENTS2_IMAGE_NAME="agent-s2:latest"
    fi

    # LLM configuration (only set if not already defined)
    if [[ -z "${AGENTS2_LLM_PROVIDER:-}" ]]; then
        # Intelligently detect provider based on available API keys
        local detected_provider="ollama"  # Default to Ollama
        
        # Check if API keys are provided via environment or arguments
        if [[ -n "${OPENAI_API_KEY:-}" ]] || [[ -n "${ARGS_OPENAI_API_KEY:-}" ]]; then
            detected_provider="openai"
        elif [[ -n "${ANTHROPIC_API_KEY:-}" ]] || [[ -n "${ARGS_ANTHROPIC_API_KEY:-}" ]]; then
            detected_provider="anthropic"
        fi
        
        # Use explicit provider if specified, otherwise use detected
        readonly AGENTS2_LLM_PROVIDER="${LLM_PROVIDER:-${ARGS_LLM_PROVIDER:-$detected_provider}}"
    fi
    if [[ -z "${AGENTS2_LLM_MODEL:-}" ]]; then
        # Set model defaults based on provider
        local default_model
        case "${AGENTS2_LLM_PROVIDER}" in
            "ollama")
                default_model="llama3.2-vision:11b"
                ;;
            "openai")
                default_model="gpt-4o"
                ;;
            "anthropic")
                default_model="claude-3-7-sonnet-20250219"
                ;;
            *)
                default_model="llama3.2-vision:11b"
                ;;
        esac
        readonly AGENTS2_LLM_MODEL="${LLM_MODEL:-${ARGS_LLM_MODEL:-$default_model}}"
    fi
    if [[ -z "${AGENTS2_OPENAI_API_KEY:-}" ]]; then
        readonly AGENTS2_OPENAI_API_KEY="${OPENAI_API_KEY:-}"
    fi
    if [[ -z "${AGENTS2_ANTHROPIC_API_KEY:-}" ]]; then
        readonly AGENTS2_ANTHROPIC_API_KEY="${ANTHROPIC_API_KEY:-}"
    fi
    if [[ -z "${AGENTS2_OLLAMA_BASE_URL:-}" ]]; then
        readonly AGENTS2_OLLAMA_BASE_URL="${OLLAMA_BASE_URL:-http://host.docker.internal:11434}"
    fi
    if [[ -z "${AGENTS2_ENABLE_AI:-}" ]]; then
        local enable_ai="false"
        [[ "${ENABLE_AI:-${ARGS_ENABLE_AI:-yes}}" == "yes" ]] && enable_ai="true"
        readonly AGENTS2_ENABLE_AI="$enable_ai"
    fi
    if [[ -z "${AGENTS2_ENABLE_SEARCH:-}" ]]; then
        local enable_search="false"
        [[ "${ENABLE_SEARCH:-${ARGS_ENABLE_SEARCH:-no}}" == "yes" ]] && enable_search="true"
        readonly AGENTS2_ENABLE_SEARCH="$enable_search"
    fi

    # Display configuration (only set if not already defined)
    if [[ -z "${AGENTS2_DISPLAY:-}" ]]; then
        readonly AGENTS2_DISPLAY=":99"
    fi
    if [[ -z "${AGENTS2_SCREEN_RESOLUTION:-}" ]]; then
        readonly AGENTS2_SCREEN_RESOLUTION="1920x1080x24"
    fi
    if [[ -z "${AGENTS2_VNC_PASSWORD:-}" ]]; then
        readonly AGENTS2_VNC_PASSWORD="${VNC_PASSWORD:-agents2vnc}"
    fi
    if [[ -z "${AGENTS2_ENABLE_HOST_DISPLAY:-}" ]]; then
        readonly AGENTS2_ENABLE_HOST_DISPLAY="${ENABLE_HOST_DISPLAY:-no}"
    fi

    # Network configuration (only set if not already defined)
    if [[ -z "${AGENTS2_NETWORK_NAME:-}" ]]; then
        readonly AGENTS2_NETWORK_NAME="agent-s2-network"
    fi

    # Security configuration (only set if not already defined)
    if [[ -z "${AGENTS2_SECURITY_OPT:-}" ]]; then
        readonly AGENTS2_SECURITY_OPT="seccomp=unconfined"
    fi
    if [[ -z "${AGENTS2_USER:-}" ]]; then
        readonly AGENTS2_USER="agents2"
    fi
    if [[ -z "${AGENTS2_USER_ID:-}" ]]; then
        readonly AGENTS2_USER_ID="1000"
    fi
    if [[ -z "${AGENTS2_GROUP_ID:-}" ]]; then
        readonly AGENTS2_GROUP_ID="1000"
    fi

    # Health check configuration (only set if not already defined)
    if [[ -z "${AGENTS2_HEALTH_CHECK_INTERVAL:-}" ]]; then
        readonly AGENTS2_HEALTH_CHECK_INTERVAL=30
    fi
    if [[ -z "${AGENTS2_HEALTH_CHECK_TIMEOUT:-}" ]]; then
        readonly AGENTS2_HEALTH_CHECK_TIMEOUT=10
    fi
    if [[ -z "${AGENTS2_HEALTH_CHECK_RETRIES:-}" ]]; then
        readonly AGENTS2_HEALTH_CHECK_RETRIES=3
    fi
    if [[ -z "${AGENTS2_API_TIMEOUT:-}" ]]; then
        readonly AGENTS2_API_TIMEOUT=120
    fi

    # Wait timeouts (only set if not already defined)
    if [[ -z "${AGENTS2_STARTUP_MAX_WAIT:-}" ]]; then
        readonly AGENTS2_STARTUP_MAX_WAIT=120
    fi
    if [[ -z "${AGENTS2_STARTUP_WAIT_INTERVAL:-}" ]]; then
        readonly AGENTS2_STARTUP_WAIT_INTERVAL=5
    fi
    if [[ -z "${AGENTS2_INITIALIZATION_WAIT:-}" ]]; then
        readonly AGENTS2_INITIALIZATION_WAIT=15
    fi

    # Resource limits (only set if not already defined)
    if [[ -z "${AGENTS2_MEMORY_LIMIT:-}" ]]; then
        readonly AGENTS2_MEMORY_LIMIT="4g"
    fi
    if [[ -z "${AGENTS2_CPU_LIMIT:-}" ]]; then
        readonly AGENTS2_CPU_LIMIT="2.0"
    fi
    if [[ -z "${AGENTS2_SHM_SIZE:-}" ]]; then
        readonly AGENTS2_SHM_SIZE="2gb"
    fi

    # Export for global access
    export AGENTS2_PORT AGENTS2_BASE_URL AGENTS2_VNC_PORT AGENTS2_VNC_URL
    export AGENTS2_CONTAINER_NAME AGENTS2_DATA_DIR AGENTS2_IMAGE_NAME
    export AGENTS2_LLM_PROVIDER AGENTS2_LLM_MODEL AGENTS2_API_KEY
    export AGENTS2_OPENAI_API_KEY AGENTS2_ANTHROPIC_API_KEY AGENTS2_OLLAMA_BASE_URL
    export AGENTS2_DISPLAY AGENTS2_SCREEN_RESOLUTION AGENTS2_VNC_PASSWORD
    export AGENTS2_ENABLE_HOST_DISPLAY AGENTS2_NETWORK_NAME
    export AGENTS2_SECURITY_OPT AGENTS2_USER AGENTS2_USER_ID AGENTS2_GROUP_ID
    export AGENTS2_HEALTH_CHECK_INTERVAL AGENTS2_HEALTH_CHECK_TIMEOUT
    export AGENTS2_HEALTH_CHECK_RETRIES AGENTS2_API_TIMEOUT
    export AGENTS2_STARTUP_MAX_WAIT AGENTS2_STARTUP_WAIT_INTERVAL
    export AGENTS2_INITIALIZATION_WAIT
    export AGENTS2_MEMORY_LIMIT AGENTS2_CPU_LIMIT AGENTS2_SHM_SIZE
}

#######################################
# Load environment variables from service.json
# This function looks for agent-s2 environment configuration
# and exports the variables for docker-compose
#######################################
agents2::load_environment_from_config() {
    local config_file
    config_file="$(secrets::get_project_config_file)"
    
    # Check if config file exists and has agent-s2 configuration
    if [[ -f "$config_file" ]] && command -v jq >/dev/null 2>&1; then
        # Check if agent-s2 has environment configuration
        if jq -e '.services.agents."agent-s2".environment' "$config_file" >/dev/null 2>&1; then
            log::info "Loading Agent-S2 environment variables from service.json"
            
            # Extract and export each environment variable
            while IFS="=" read -r key value; do
                if [[ -n "$key" && -n "$value" ]]; then
                    # Remove quotes from value if present
                    value=$(echo "$value" | sed 's/^"//;s/"$//')
                    export "$key=$value"
                    log::debug "Loaded environment variable: $key"
                fi
            done < <(jq -r '.services.agents."agent-s2".environment | to_entries[] | "\(.key)=\(.value)"' "$config_file" 2>/dev/null)
        else
            log::debug "No environment configuration found for agent-s2 in service.json"
        fi
    else
        log::debug "Resources configuration file not found or jq not available"
    fi
}

# Export function for subshell availability
export -f agents2::export_config
export -f agents2::load_environment_from_config