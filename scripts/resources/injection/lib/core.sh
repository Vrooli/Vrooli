#!/usr/bin/env bash
################################################################################
# Injection System v2.0 - Core Logic
# Core functions for scenario content injection
################################################################################
set -euo pipefail

# Global settings
INJECTION_DRY_RUN="${INJECTION_DRY_RUN:-false}"
INJECTION_PARALLEL="${INJECTION_PARALLEL:-true}"
INJECTION_VERBOSE="${INJECTION_VERBOSE:-false}"

#######################################
# Add content from scenario to resources
# Arguments:
#   $1 - scenario name
#   $@ - additional options
#######################################
inject::add() {
    local scenario_name="${1:-}"
    shift || true
    
    # Parse options
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --dry-run)
                INJECTION_DRY_RUN=true
                shift
                ;;
            --parallel)
                INJECTION_PARALLEL=true
                shift
                ;;
            --no-parallel)
                INJECTION_PARALLEL=false
                shift
                ;;
            --verbose)
                INJECTION_VERBOSE=true
                shift
                ;;
            *)
                log::error "Unknown option: $1"
                return 1
                ;;
        esac
    done
    
    if [[ -z "$scenario_name" ]]; then
        log::error "Scenario name required"
        echo "Usage: inject.sh add <scenario> [--dry-run] [--parallel] [--verbose]"
        return 1
    fi
    
    log::header "📦 Injecting content from scenario: $scenario_name"
    
    # Load scenario configuration
    local scenario_config
    scenario_config=$(scenario::load "$scenario_name") || {
        log::error "Failed to load scenario: $scenario_name"
        return 1
    }
    
    # Validate before proceeding
    if ! validate::scenario "$scenario_config"; then
        log::error "Scenario validation failed"
        return 1
    fi
    
    if [[ "$INJECTION_DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN MODE] No changes will be made"
    fi
    
    # Process each resource in the scenario
    local resources
    resources=$(echo "$scenario_config" | jq -r '.resources | keys[]' 2>/dev/null || true)
    
    if [[ -z "$resources" ]]; then
        log::warn "No resources defined in scenario"
        return 0
    fi
    
    local total=0
    local success=0
    local failed=0
    
    for resource in $resources; do
        ((total++))
        log::info "Processing resource: $resource"
        
        local resource_config
        resource_config=$(echo "$scenario_config" | jq -c ".resources[\"$resource\"]")
        
        if content::add_to_resource "$resource" "$resource_config"; then
            ((success++))
            log::success "✅ $resource"
        else
            ((failed++))
            log::error "❌ $resource"
        fi
    done
    
    # Summary
    log::header "📊 Injection Summary"
    log::info "Total resources: $total"
    log::info "Successful: $success"
    if [[ $failed -gt 0 ]]; then
        log::warn "Failed: $failed"
        return 1
    fi
    
    log::success "✅ Injection complete!"
    return 0
}

#######################################
# Validate scenario configuration
# Arguments:
#   $1 - scenario name
#######################################
inject::validate() {
    local scenario_name="${1:-}"
    
    if [[ -z "$scenario_name" ]]; then
        log::error "Scenario name required"
        echo "Usage: inject.sh validate <scenario>"
        return 1
    fi
    
    log::header "🔍 Validating scenario: $scenario_name"
    
    # Load scenario
    local scenario_config
    scenario_config=$(scenario::load "$scenario_name") || {
        log::error "Failed to load scenario: $scenario_name"
        return 1
    }
    
    # Run validation
    if validate::scenario "$scenario_config"; then
        log::success "✅ Scenario is valid"
        return 0
    else
        log::error "❌ Scenario validation failed"
        return 1
    fi
}

#######################################
# List available scenarios
#######################################
inject::list() {
    log::header "📋 Available Scenarios"
    scenario::list
}

#######################################
# Show injection status
#######################################
inject::status() {
    log::header "📊 Injection Status"
    
    # Check which resources are running
    log::info "Checking resource availability..."
    
    local resources="n8n postgres redis qdrant windmill"
    for resource in $resources; do
        if command -v "resource-$resource" >/dev/null 2>&1; then
            if "resource-$resource" status >/dev/null 2>&1; then
                log::success "✅ $resource is running"
            else
                log::info "💤 $resource is not running"
            fi
        else
            log::info "❓ $resource CLI not installed"
        fi
    done
}