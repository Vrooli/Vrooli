#!/bin/bash

# SimPy Resource - Status Functions
set -euo pipefail

# Get the script directory
SIMPY_STATUS_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source common functions
source "$SIMPY_STATUS_LIB_DIR/common.sh"

# Main status function
status::main() {
    local format="${1:-text}"
    
    # Collect status data
    local installed running health healthy status version message
    local simulations outputs
    
    if simpy::is_installed; then
        installed="true"
        version=$(simpy::get_version)
        
        if simpy::is_healthy; then
            running="true"
            health="healthy"
            healthy="true"
            status="running"
            message="SimPy is installed and functional"
        else
            running="false"
            health="unhealthy"
            healthy="false"
            status="stopped"
            message="SimPy is installed but not functioning properly"
        fi
    else
        installed="false"
        running="false"
        health="unhealthy"
        healthy="false"
        status="stopped"
        version="not_installed"
        message="SimPy is not installed"
    fi
    
    simulations=$(simpy::get_simulation_count)
    outputs=$(simpy::get_output_count)
    
    # Output based on format
    if [[ "$format" == "--json" ]] || [[ "$format" == "json" ]]; then
        # JSON output
        format::key_value json \
            name "$SIMPY_RESOURCE_NAME" \
            category "$SIMPY_CATEGORY" \
            description "$SIMPY_DESCRIPTION" \
            installed "$installed" \
            running "$running" \
            health "$health" \
            healthy "$healthy" \
            status "$status" \
            version "$version" \
            message "$message" \
            simulations "$simulations" \
            outputs "$outputs" \
            container_name "$SIMPY_CONTAINER_NAME" \
            image_name "$SIMPY_IMAGE_NAME" \
            simulations_dir "$SIMPY_SIMULATIONS_DIR" \
            outputs_dir "$SIMPY_OUTPUTS_DIR"
    else
        # Text output
        log::header "SimPy Status Report"
        echo
        echo "Description: $SIMPY_DESCRIPTION"
        echo "Category: $SIMPY_CATEGORY"
        echo
        echo "Basic Status:"
        echo "  $([ "$installed" == "true" ] && echo "✅" || echo "❌") Installed: $([ "$installed" == "true" ] && echo "Yes" || echo "No")"
        echo "  $([ "$running" == "true" ] && echo "✅" || echo "❌") Running: $([ "$running" == "true" ] && echo "Yes" || echo "No")"
        echo "  $([ "$healthy" == "true" ] && echo "✅" || echo "❌") Health: $health"
        echo
        echo "Container Info:"
        echo "  📦 Name: $SIMPY_CONTAINER_NAME"
        echo "  🖼️  Image: $SIMPY_IMAGE_NAME"
        echo "  📊 Version: $version"
        echo
        echo "Metrics:"
        echo "  📝 Simulations: $simulations"
        echo "  📊 Outputs: $outputs"
        echo
        echo "Directories:"
        echo "  📁 Simulations: $SIMPY_SIMULATIONS_DIR"
        echo "  📁 Outputs: $SIMPY_OUTPUTS_DIR"
        echo
        echo "Status Message:"
        echo "  $([ "$healthy" == "true" ] && echo "✅" || echo "❌") $message"
    fi
}

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    status::main "$@"
fi