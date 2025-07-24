#!/usr/bin/env bash
# Port Registry - Central source of truth for all service port assignments
# This file prevents port conflicts between Vrooli services and local resources

# ============================================================================
# VROOLI CORE SERVICES - DO NOT USE THESE PORTS FOR RESOURCES
# ============================================================================
declare -g -A VROOLI_PORTS=(
    ["ui"]="3000"              # Frontend web application
    ["api"]="5329"             # Backend API server
    ["jobs"]="4001"            # Background jobs service
    ["postgres"]="5432"        # PostgreSQL database
    ["redis"]="6379"           # Redis cache
    ["adminer"]="8080"         # Database admin UI
    ["nsfw"]="8000"            # NSFW detector service (internal)
    ["debug_server"]="9229"    # Server debugger
    ["debug_jobs"]="9230"      # Jobs debugger
)

# ============================================================================
# LOCAL RESOURCE SERVICES - CAREFULLY CHOSEN TO AVOID CONFLICTS
# ============================================================================
declare -g -A RESOURCE_PORTS=(
    # AI Services (11xxx range)
    ["ollama"]="11434"         # LLM inference engine
    ["localai"]="11435"        # Alternative AI service (changed from 8080)
    
    # Automation Services (56xx range)
    ["n8n"]="5678"             # Workflow automation
    ["node-red"]="5680"        # Flow-based automation (changed from 1880)
    
    # Storage Services (9xxx range)
    ["minio"]="9000"           # Object storage (S3 compatible)
    ["ipfs"]="9001"            # Distributed storage (changed from 5001)
    
    # Agent Services (41xx range - safely above Vrooli range)
    ["browserless"]="4110"     # Browserless.io Chrome service
    
    # Future services can use:
    # - 11xxx for AI services
    # - 56xx for automation (avoiding 5678)
    # - 90xx for storage
    # - 41xx for agents (above Vrooli range)
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
# Helper Functions
# ============================================================================

#######################################
# Get port for a Vrooli service
# Arguments:
#   $1 - service name
# Returns:
#   Port number or empty string
#######################################
ports::get_vrooli_port() {
    local service="$1"
    echo "${VROOLI_PORTS[$service]:-}"
}

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
# Check if port is reserved by Vrooli
# Arguments:
#   $1 - port number
# Returns:
#   0 if reserved, 1 if available
#######################################
ports::is_vrooli_reserved() {
    local port="$1"
    
    # Check exact matches
    for vrooli_port in "${VROOLI_PORTS[@]}"; do
        if [[ "$port" == "$vrooli_port" ]]; then
            return 0
        fi
    done
    
    # Check reserved ranges
    if [[ "$port" -ge 3000 && "$port" -le 4100 ]]; then
        return 0  # In Vrooli core range
    fi
    
    if [[ "$port" -ge 9200 && "$port" -le 9299 ]]; then
        return 0  # In debug range
    fi
    
    return 1
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
    
    # Check if it conflicts with Vrooli
    if ports::is_vrooli_reserved "$port"; then
        echo "ERROR: Port $port for $service conflicts with Vrooli services"
        return 1
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
    
    # Add Vrooli ports
    for port in "${VROOLI_PORTS[@]}"; do
        all_ports+=("$port")
    done
    
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
        
        if [[ "$found" -eq 0 ]] && ! ports::is_vrooli_reserved "$port"; then
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
    echo "VROOLI CORE SERVICES:"
    for service in "${!VROOLI_PORTS[@]}"; do
        printf "  %-15s : %s\n" "$service" "${VROOLI_PORTS[$service]}"
    done | sort
    
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

# If script is run directly, show the registry
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    ports::show_registry
fi