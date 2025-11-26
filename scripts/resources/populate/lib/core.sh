#!/usr/bin/env bash
################################################################################
# Population System v2.0 - Core Logic
# Core functions for scenario content population
################################################################################
set -euo pipefail

# Global settings
POPULATION_DRY_RUN="${POPULATION_DRY_RUN:-false}"
POPULATION_PARALLEL="${POPULATION_PARALLEL:-true}"
POPULATION_VERBOSE="${POPULATION_VERBOSE:-false}"

#######################################
# Populate resources with scenario content from path  
# Arguments:
#   $1 - path to scenario directory (e.g., ".", "/path/to/scenario")
#   $@ - additional options
#######################################
populate::add_from_path() {
    local scenario_path="${1:-}"
    shift || true
    
    if [[ -z "$scenario_path" ]]; then
        log::error "Scenario path required"
        echo "Usage: populate.sh /path/to/scenario [options]"
        return 1
    fi
    
    # Convert relative path to absolute
    scenario_path=$(cd "$scenario_path" && pwd) || {
        log::error "Invalid path: $1"
        return 1
    }
    
    # Get scenario name from path
    local scenario_name
    scenario_name=$(basename "$scenario_path")
    
    log::header "📦 Populating resources from path: $scenario_path"
    log::info "Detected scenario: $scenario_name"
    
    # Check if this looks like a scenario directory
    if [[ ! -f "$scenario_path/.vrooli/service.json" ]]; then
        log::error "Not a valid scenario directory (missing .vrooli/service.json): $scenario_path"
        return 1
    fi
    
    # Load scenario configuration directly from path
    local scenario_config
    scenario_config=$(cat "$scenario_path/.vrooli/service.json") || {
        log::error "Failed to load scenario configuration from: $scenario_path/.vrooli/service.json"
        return 1
    }
    
    # Use the same population logic as populate::add but with direct path access
    # Set up environment for path-based processing
    export SCENARIO_PATH="$scenario_path"
    export SCENARIO_NAME="$scenario_name"
    
    # Call the common population logic
    populate::add_internal "$scenario_name" "$scenario_config" "$@"
}

#######################################
# Add content from scenario to resources by name
# Arguments:
#   $1 - scenario name
#   $@ - additional options
#######################################
populate::add() {
    local scenario_name="${1:-}"
    shift || true
    
    if [[ -z "$scenario_name" ]]; then
        log::error "Scenario name required"
        echo "Usage: populate.sh add <scenario> [--dry-run] [--parallel] [--verbose]"
        return 1
    fi
    
    # Load scenario configuration using search paths
    local scenario_config
    scenario_config=$(scenario::load "$scenario_name") || {
        log::error "Failed to load scenario: $scenario_name"
        return 1
    }
    
    # Call the common population logic  
    populate::add_internal "$scenario_name" "$scenario_config" "$@"
}

#######################################
# Internal population logic shared by both add and add_from_path
# Arguments:
#   $1 - scenario name
#   $2 - scenario config JSON
#   $@ - additional options
#######################################
populate::add_internal() {
    local scenario_name="$1"
    local scenario_config="$2"
    shift 2
    
    # Parse options
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --dry-run)
                POPULATION_DRY_RUN=true
                shift
                ;;
            --parallel)
                POPULATION_PARALLEL=true
                shift
                ;;
            --no-parallel)
                POPULATION_PARALLEL=false
                shift
                ;;
            --verbose)
                POPULATION_VERBOSE=true
                shift
                ;;
            *)
                log::error "Unknown option: $1"
                return 1
                ;;
        esac
    done
    
    log::header "📦 Populating resources from scenario: $scenario_name"
    
    # Validate before proceeding
    log::info "Validating scenario structure..."
    if ! validate::scenario "$scenario_config"; then
        log::error "Scenario validation failed"
        return 1
    fi
    log::info "✅ Scenario structure is valid"
    
    if [[ "$POPULATION_DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN MODE] No changes will be made"
    fi
    
    # Process each resource in the scenario (now flat structure)
    log::info "Extracting resources from scenario configuration..."
    local resources
    resources=$(echo "$scenario_config" | jq -r '.dependencies.resources // {} | keys[]' 2>/dev/null || true)
    
    if [[ -z "$resources" ]]; then
        log::warn "No resources defined in scenario - nothing to populate"
        return 0
    fi
    
    # Count resources
    local resource_count
    resource_count=$(echo "$resources" | wc -l)
    log::info "Found $resource_count resource(s) to process: $(echo $resources | tr '\n' ' ')"
    
    local total=0
    local success=0
    local failed=0
    local skipped=0
    
    # Iterate through resources directly (flat structure)
    for resource in $resources; do
        total=$((total + 1))
        log::info "[$total/$resource_count] Processing resource: $resource"
        
        # Extract resource configuration
        local resource_config
        resource_config=$(echo "$scenario_config" | jq -c ".dependencies.resources[\"$resource\"]" 2>/dev/null) || {
            log::error "Failed to extract configuration for resource: $resource"
            failed=$((failed + 1))
            continue
        }
        
        # Check if resource has initialization content
        local has_init
        has_init=$(echo "$resource_config" | jq -r 'has("initialization")' 2>/dev/null || echo "false")
        
        if [[ "$has_init" != "true" ]]; then
            log::info "  → No initialization content for $resource, skipping"
            skipped=$((skipped + 1))
            continue
        fi
        
        # Check if resource is available before trying to populate
        local resource_cli="resource-${resource}"
        if ! command -v "$resource_cli" >/dev/null 2>&1; then
            # Try direct path
            resource_cli="${APP_ROOT}/resources/${resource}/cli.sh"
            if [[ ! -f "$resource_cli" ]]; then
                log::warn "  → Resource CLI not available for $resource, skipping"
                skipped=$((skipped + 1))
                continue
            fi
        fi
        
        # Try to populate the resource
        log::info "  → Adding content to $resource..."
        if content::add_to_resource "$resource" "$resource_config"; then
            success=$((success + 1))
            log::success "  ✅ Successfully populated $resource"
        else
            failed=$((failed + 1))
            log::warn "  ⚠️  Failed to populate $resource (resource may not be running)"
        fi
    done
    
    # Summary
    log::header "📊 Population Summary"
    log::info "Total resources: $total"
    log::info "Successfully populated: $success"
    if [[ $skipped -gt 0 ]]; then
        log::info "Skipped (no content/CLI): $skipped"
    fi
    if [[ $failed -gt 0 ]]; then
        log::warn "Failed to populate: $failed"
        log::info "Note: Resources may not be running yet - this is expected during initial setup"
    fi
    
    # Don't fail the entire setup just because resources aren't ready
    # This is a common scenario during initial setup
    if [[ $success -eq 0 && $total -gt 0 && $skipped -ne $total ]]; then
        log::warn "⚠️  No resources were successfully populated"
        log::info "This is expected if resources are not yet running"
        log::info "Resources will be populated when they become available"
    else
        log::success "✅ Population phase complete!"
    fi
    
    # Always return success unless there's a critical error
    # Resource population is often optional and shouldn't block setup
    return 0
}

#######################################
# Validate scenario configuration
# Arguments:
#   $1 - scenario name
#######################################
populate::validate() {
    local scenario_name="${1:-}"
    
    if [[ -z "$scenario_name" ]]; then
        log::error "Scenario name required"
        echo "Usage: populate.sh validate <scenario>"
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
populate::list() {
    log::header "📋 Available Scenarios"
    scenario::list
}

#######################################
# Show injection status
#######################################
populate::status() {
    log::header "📊 Population Status"
    
    # Check which resources are running
    log::info "Checking resource availability..."
    
    local resources="n8n postgres redis qdrant"
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
