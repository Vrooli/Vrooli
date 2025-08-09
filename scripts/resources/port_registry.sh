#!/usr/bin/env bash
# Port Registry - Central source of truth for all service port assignments
# This file prevents port conflicts between Vrooli services and local resources

# ============================================================================
# LOCAL RESOURCE SERVICES - CAREFULLY CHOSEN TO AVOID CONFLICTS
# ============================================================================
declare -g -A RESOURCE_PORTS=(
    # AI Services (11xxx range)
    ["ollama"]="11434"         # LLM inference engine
    ["whisper"]="8090"         # Speech-to-text service
    ["unstructured-io"]="11450" # Document processing and extraction
    
    # Automation Services (56xx range)
    ["n8n"]="5678"             # Workflow automation
    ["comfyui"]="8188"         # AI-powered image generation workflows
    ["node-red"]="1880"        # Flow-based automation (Node-RED default port)
    ["windmill"]="5681"        # Developer-centric workflow automation
    
    # Storage Services (9xxx range + vector databases)
    ["minio"]="9000"           # Object storage (S3 compatible)
    ["vault"]="8200"           # HashiCorp Vault secret management
    ["qdrant"]="6333"          # Vector database for AI embeddings
    ["questdb"]="9009"         # Time-series database for analytics
    ["postgres"]="5433"        # PostgreSQL instances (5433-5499 range for client instances)
    ["redis"]="6380"           # Redis in-memory data store (6380 to avoid conflict with internal Redis on 6379)
    
    # Agent Services (41xx range - safely above Vrooli range)
    ["browserless"]="4110"     # Browserless.io Chrome service
    ["huginn"]="4111"          # Workflow automation platform
    ["agent-s2"]="4113"        # Agent S2 autonomous computer interaction
    # Note: claude-code is a CLI tool and doesn't use a port
    
    # Search Services (92xx range - moved due to conflict)
    ["searxng"]="9200"         # SearXNG metasearch engine (NOTE: conflicts with debug range)
    
    # Execution Services (23xx range)
    ["judge0"]="2358"          # Code execution sandbox (official Judge0 port)
    
    # Future services can use:
    # - 11xxx for AI services
    # - 56xx for automation (avoiding 5678)
    # - 90xx for storage
    # - 41xx for agents (above Vrooli range)
    # - 81xx for search services
    # - 23xx for execution services
)

# ============================================================================
# RESERVED PORT RANGES
# ============================================================================
declare -g -A RESERVED_RANGES=(
    ["vrooli_core"]="3000-4100"       # Reserved for Vrooli core services
    ["databases"]="5432-5499"         # Database services
    ["cache"]="6379-6399"             # Cache services
    ["debug"]="9200-9299"             # Debug ports
    ["system"]="1-1023"               # System ports (require root)
)

# ============================================================================
# PostgreSQL Instance Configuration
# ============================================================================
# PostgreSQL instances (5433-5499 range for client instances)  
[[ -z "${POSTGRES_DEFAULT_PORT:-}" ]] && readonly POSTGRES_DEFAULT_PORT=5433
[[ -z "${POSTGRES_INSTANCE_PORT_RANGE_START:-}" ]] && readonly POSTGRES_INSTANCE_PORT_RANGE_START=5433
[[ -z "${POSTGRES_INSTANCE_PORT_RANGE_END:-}" ]] && readonly POSTGRES_INSTANCE_PORT_RANGE_END=5499

# ============================================================================
# Helper Functions
# ============================================================================

#######################################
# Get port for a resource service
# Arguments:
#   $1 - resource name
# Returns:
#   Port number or empty string
#######################################
ports::get_resource_port() {
    local resource="$1"
    echo "${RESOURCE_PORTS[$resource]:-}"
}

#######################################
# Validate port assignment
# Arguments:
#   $1 - port number
#   $2 - service name (for error messages)
# Returns:
#   0 if valid, 1 if invalid
#######################################
ports::validate_assignment() {
    local port="$1"
    local service="$2"
    
    # Check if port is a valid number
    if ! [[ "$port" =~ ^[0-9]+$ ]] || [[ "$port" -lt 1 || "$port" -gt 65535 ]]; then
        echo "ERROR: Invalid port number '$port' for $service"
        return 1
    fi
    
    # Check if it's a system port (requires root)
    if [[ "$port" -lt 1024 ]]; then
        echo "WARNING: Port $port for $service requires root privileges"
    fi
    
    return 0
}

#######################################
# Get all ports in use (both Vrooli and resources)
# Returns:
#   Array of all assigned ports
#######################################
ports::get_all_assigned() {
    local -a all_ports=()
    
    # Add resource ports
    for port in "${RESOURCE_PORTS[@]}"; do
        all_ports+=("$port")
    done
    
    printf '%s\n' "${all_ports[@]}" | sort -n | uniq
}

#######################################
# Find next available port in a range
# Arguments:
#   $1 - start of range
#   $2 - end of range
# Returns:
#   Next available port or empty string
#######################################
ports::find_available_in_range() {
    local start="$1"
    local end="$2"
    local -a assigned=($(ports::get_all_assigned))
    
    for ((port=start; port<=end; port++)); do
        local found=0
        for assigned_port in "${assigned[@]}"; do
            if [[ "$port" == "$assigned_port" ]]; then
                found=1
                break
            fi
        done
        
        if [[ "$found" -eq 0 ]]; then
            echo "$port"
            return 0
        fi
    done
    
    return 1
}

#######################################
# Display port registry information
#######################################
ports::show_registry() {
    echo "=== Vrooli Port Registry ==="

    echo
    echo "LOCAL RESOURCES:"
    for resource in "${!RESOURCE_PORTS[@]}"; do
        printf "  %-15s : %s\n" "$resource" "${RESOURCE_PORTS[$resource]}"
    done | sort
    
    echo
    echo "RESERVED RANGES:"
    for range_name in "${!RESERVED_RANGES[@]}"; do
        printf "  %-15s : %s\n" "$range_name" "${RESERVED_RANGES[$range_name]}"
    done | sort
}

#######################################
# Export resource ports as JSON
# Used by TypeScript modules for dynamic loading
#######################################
ports::export_json() {
    local output='{"resource_ports":{'
    local first=true
    
    for resource in "${!RESOURCE_PORTS[@]}"; do
        if [[ "$first" == true ]]; then
            first=false
        else
            output+=","
        fi
        # Ensure port is a valid number
        local port="${RESOURCE_PORTS[$resource]}"
        if [[ "$port" =~ ^[0-9]+$ ]]; then
            output+="\"$resource\":$port"
        fi
    done
    
    output+="}}"
    echo "$output"
}

#######################################
# Show help information
#######################################
ports::show_help() {
    echo "Port Registry - Central source of truth for all service port assignments"
    echo
    echo "Usage: $0 [OPTION]"
    echo
    echo "Options:"
    echo "  --export-json    Export resource ports as JSON (for TypeScript integration)"
    echo "  --help          Show this help message"
    echo "  (no options)    Show port registry information"
}

# If script is run directly, handle CLI arguments
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    case "${1:-}" in
        --export-json)
            ports::export_json
            ;;
        --help)
            ports::show_help
            ;;
        *)
            ports::show_registry
            ;;
    esac
fi