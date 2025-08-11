#!/usr/bin/env bash
set -euo pipefail

# Resource Data Injection Engine - Simplified but Compatible Version
# Maintains interface compatibility while removing unnecessary complexity

SCRIPTS_SCENARIOS_INJECTION_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${SCRIPTS_SCENARIOS_INJECTION_DIR}/../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"

DEFAULT_SCENARIOS_DIR="${var_SCRIPTS_SCENARIOS_DIR}/core"

# Parse arguments
ACTION="inject"
SCENARIO_NAME=""
CONFIG_FILE=""
ALL_ACTIVE="no"
DRY_RUN="no"
FORCE="no"
INITIALIZATION_DIR=""

#######################################
# Display usage
#######################################
usage() {
    cat << EOF
Resource Data Injection Engine

USAGE:
    $0 [OPTIONS] [INITIALIZATION_DIRECTORY]

OPTIONS:
    -a, --action ACTION        Action to perform (default: inject)
                              Options: inject, validate, list-scenarios
    -s, --scenario NAME        Scenario name to process
        --config-file PATH     Config file (ignored - for compatibility)
        --all-active          Process all scenarios (yes/no, default: no)
        --dry-run             Show what would be done (yes/no, default: no)
        --force               Force injection (yes/no, default: no)
    -h, --help                Show this help

ARGUMENTS:
    INITIALIZATION_DIRECTORY   Directory containing resource initialization files
                              Structure: category/resource/files
                              Example: initialization/automation/comfyui/

EXAMPLES:
    # Inject from initialization directory
    $0 initialization/
    
    # Inject specific scenario
    $0 --action inject --scenario scenario-name
    
    # Inject all scenarios
    $0 --action inject --all-active yes
    
    # Validate scenario
    $0 --action validate --scenario scenario-name
    
    # List available scenarios
    $0 --action list-scenarios

EOF
}

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -a|--action)
            ACTION="$2"
            shift 2
            ;;
        -s|--scenario)
            SCENARIO_NAME="$2"
            shift 2
            ;;
        --config-file)
            CONFIG_FILE="$2"  # Accept but ignore
            shift 2
            ;;
        --all-active)
            ALL_ACTIVE="$2"
            shift 2
            ;;
        --dry-run)
            DRY_RUN="$2"
            shift 2
            ;;
        --force|--yes|-y)
            FORCE="yes"
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            # If it's a directory path, treat it as initialization directory
            if [[ -d "$1" ]] || [[ "$1" =~ ^[^-] ]]; then
                INITIALIZATION_DIR="$1"
                shift
            else
                log::error "Unknown option: $1"
                usage
                exit 1
            fi
            ;;
    esac
done

#######################################
# Resolve secrets
#######################################
resolve_secret() {
    local key="$1"
    
    # Try environment variable
    local value="${!key:-}"
    if [[ -n "$value" ]]; then
        echo "$value"
        return 0
    fi
    
    # Try .vrooli/secrets.json
    if [[ -f ".vrooli/secrets.json" ]]; then
        value=$(jq -r ".$key // empty" .vrooli/secrets.json 2>/dev/null || true)
        if [[ -n "$value" && "$value" != "null" ]]; then
            echo "$value"
            return 0
        fi
    fi
    
    echo ""
}

#######################################
# Substitute secrets in JSON
#######################################
substitute_secrets() {
    local json="$1"
    local result="$json"
    
    while [[ "$result" =~ \{\{([^}]+)\}\} ]]; do
        local placeholder="${BASH_REMATCH[0]}"
        local secret_name="${BASH_REMATCH[1]}"
        local secret_value
        
        secret_value=$(resolve_secret "$secret_name")
        if [[ -n "$secret_value" ]]; then
            secret_value=$(echo "$secret_value" | sed 's/\\/\\\\/g; s/"/\\"/g')
            result="${result//$placeholder/$secret_value}"
        else
            log::warn "Secret not found: $secret_name"
            result="${result//$placeholder/}"
        fi
    done
    
    echo "$result"
}

#######################################
# Find inject script for resource
#######################################
find_inject_script() {
    local resource="$1"
    find "${var_SCRIPTS_RESOURCES_DIR}" -name "inject.sh" -path "*/${resource}/lib/*" 2>/dev/null | head -1
}

#######################################
# Inject resource data
#######################################
inject_resource() {
    local resource_name="$1"
    local resource_config="$2"
    local scenario_dir="$3"
    
    # Check if enabled
    local enabled
    enabled=$(echo "$resource_config" | jq -r '.enabled // true')
    if [[ "$enabled" != "true" ]]; then
        log::debug "Resource $resource_name is disabled"
        return 0
    fi
    
    # Find inject script
    local inject_script
    inject_script=$(find_inject_script "$resource_name")
    
    if [[ -z "$inject_script" ]] || [[ ! -x "$inject_script" ]]; then
        log::debug "No injection for: $resource_name"
        return 0
    fi
    
    # Get initialization data
    local init_data
    init_data=$(echo "$resource_config" | jq -c '.initialization // {}')
    
    if [[ "$init_data" == "{}" ]]; then
        return 0
    fi
    
    # Fix relative paths
    init_data=$(echo "$init_data" | jq --arg dir "$scenario_dir" '
        walk(if type == "object" and has("file") then 
            .file = ($dir + "/" + .file) 
        else . end)
    ')
    
    log::info "Injecting $resource_name..."
    
    if [[ "$DRY_RUN" == "yes" ]]; then
        log::info "[DRY RUN] Would inject $resource_name"
        return 0
    fi
    
    if "$inject_script" --inject "$init_data" 2>/dev/null; then
        log::success "✓ $resource_name"
        return 0
    else
        log::error "✗ $resource_name"
        return 1
    fi
}

#######################################
# Process scenario directory
#######################################
process_scenario() {
    local scenario_dir="$1"
    local service_json="$scenario_dir/.vrooli/service.json"
    
    if [[ ! -f "$service_json" ]]; then
        log::error "No service.json: $service_json"
        return 1
    fi
    
    # Read and process service.json
    local service_config
    service_config=$(cat "$service_json")
    service_config=$(substitute_secrets "$service_config")
    
    local scenario_name
    scenario_name=$(echo "$service_config" | jq -r '.service.name // "unknown"')
    
    log::header "📦 $scenario_name"
    
    # Process all resources
    local failed=0
    for category in automation data storage ai; do
        local resources
        resources=$(echo "$service_config" | jq -r --arg cat "$category" '.resources[$cat] // {} | keys[]' 2>/dev/null || true)
        
        for resource in $resources; do
            local resource_config
            resource_config=$(echo "$service_config" | jq -c --arg cat "$category" --arg res "$resource" '.resources[$cat][$res]')
            
            if ! inject_resource "$resource" "$resource_config" "$scenario_dir"; then
                ((failed++))
            fi
        done
    done
    
    if [[ $failed -eq 0 ]]; then
        log::success "✅ Completed: $scenario_name"
        return 0
    else
        log::error "❌ Failed: $scenario_name ($failed errors)"
        return 1
    fi
}

#######################################
# Process initialization directory
#######################################
process_initialization_directory() {
    local init_dir="$1"
    
    if [[ ! -d "$init_dir" ]]; then
        log::error "Initialization directory not found: $init_dir"
        return 1
    fi
    
    log::header "📁 Processing initialization directory: $init_dir"
    
    # Look for service.json in the parent directory
    local service_json
    if [[ -f "${init_dir}/../.vrooli/service.json" ]]; then
        service_json="${init_dir}/../.vrooli/service.json"
    elif [[ -f "${init_dir}/.vrooli/service.json" ]]; then
        service_json="${init_dir}/.vrooli/service.json"
    elif [[ -f "${PWD}/.vrooli/service.json" ]]; then
        service_json="${PWD}/.vrooli/service.json"
    else
        log::info "No service.json found, proceeding with directory-based injection"
        return process_directory_based_injection "$init_dir"
    fi
    
    # If we have a service.json, process it normally using the directory containing it
    local parent_dir="$(dirname "$(dirname "$service_json")")"
    process_scenario "$parent_dir"
}

#######################################
# Process directory-based injection (fallback when no service.json)
#######################################
process_directory_based_injection() {
    local init_dir="$1"
    
    log::info "Scanning for resource data in: $init_dir"
    
    local failed=0
    local processed=0
    
    # Walk through the directory structure looking for category/resource/files
    for category_dir in "$init_dir"/*/; do
        if [[ ! -d "$category_dir" ]]; then
            continue
        fi
        
        local category="$(basename "$category_dir")"
        log::info "Processing category: $category"
        
        for resource_dir in "$category_dir"/*/; do
            if [[ ! -d "$resource_dir" ]]; then
                continue
            fi
            
            local resource="$(basename "$resource_dir")"
            log::info "Processing resource: $resource"
            
            # Find inject script
            local inject_script
            inject_script=$(find_inject_script "$resource")
            
            if [[ -z "$inject_script" ]] || [[ ! -x "$inject_script" ]]; then
                log::debug "No injection script for: $resource"
                continue
            fi
            
            # Look for data files in the resource directory
            local data_files=()
            while IFS= read -r -d '' file; do
                data_files+=("$file")
            done < <(find "$resource_dir" -type f -print0 2>/dev/null)
            
            if [[ ${#data_files[@]} -eq 0 ]]; then
                log::debug "No data files found in: $resource_dir"
                continue
            fi
            
            # Create basic initialization structure for each file
            for file in "${data_files[@]}"; do
                local relative_path="${file#$PWD/}"
                local filename="$(basename "$file")"
                local extension="${filename##*.}"
                
                # Determine type based on file extension
                local file_type="workflow"
                case "$extension" in
                    "json") file_type="workflow" ;;
                    "sql") file_type="script" ;;
                    "sh"|"bash") file_type="script" ;;
                    *) file_type="data" ;;
                esac
                
                # Create JSON for this file
                local init_data="{\"workflows\":[{\"file\":\"$relative_path\",\"type\":\"$file_type\",\"enabled\":true}]}"
                
                log::info "Injecting $resource from $filename..."
                
                if [[ "$DRY_RUN" == "yes" ]]; then
                    log::info "[DRY RUN] Would inject $resource with $filename"
                    ((processed++))
                    continue
                fi
                
                if "$inject_script" --inject "$init_data" 2>/dev/null; then
                    log::success "✓ $resource ($filename)"
                    ((processed++))
                else
                    log::error "✗ $resource ($filename)"
                    ((failed++))
                fi
            done
        done
    done
    
    if [[ $processed -eq 0 && $failed -eq 0 ]]; then
        log::info "No processable resource data found"
        return 0
    elif [[ $failed -eq 0 ]]; then
        log::success "✅ Processed $processed items from initialization directory"
        return 0
    else
        log::error "❌ Failed: $failed errors, $processed successful"
        return 1
    fi
}

#######################################
# Validate scenario
#######################################
validate_scenario() {
    local scenario_dir="$1"
    local service_json="$scenario_dir/.vrooli/service.json"
    
    if [[ ! -f "$service_json" ]]; then
        log::error "Missing service.json"
        return 1
    fi
    
    if ! jq . "$service_json" >/dev/null 2>&1; then
        log::error "Invalid JSON"
        return 1
    fi
    
    log::success "Valid scenario"
    return 0
}

#######################################
# List scenarios
#######################################
list_scenarios() {
    log::header "📋 Available Scenarios"
    
    for dir in "${DEFAULT_SCENARIOS_DIR}"/*/; do
        if [[ -d "$dir" ]] && [[ -f "$dir/.vrooli/service.json" ]]; then
            local name
            name=$(jq -r '.service.name // "unknown"' "$dir/.vrooli/service.json" 2>/dev/null || echo "unknown")
            local desc
            desc=$(jq -r '.service.description // ""' "$dir/.vrooli/service.json" 2>/dev/null || echo "")
            log::info "• $name - $desc"
        fi
    done
}

#######################################
# Main
#######################################
main() {
    case "$ACTION" in
        "inject")
            if [[ -n "$INITIALIZATION_DIR" ]]; then
                # Process initialization directory
                process_initialization_directory "$INITIALIZATION_DIR"
            elif [[ "$ALL_ACTIVE" == "yes" ]]; then
                # Process all scenarios
                for dir in "${DEFAULT_SCENARIOS_DIR}"/*/; do
                    if [[ -d "$dir" ]]; then
                        process_scenario "$dir"
                    fi
                done
            elif [[ -n "$SCENARIO_NAME" ]]; then
                # Find scenario by name
                local scenario_dir="${DEFAULT_SCENARIOS_DIR}/${SCENARIO_NAME}"
                if [[ ! -d "$scenario_dir" ]]; then
                    log::error "Scenario not found: $SCENARIO_NAME"
                    exit 1
                fi
                process_scenario "$scenario_dir"
            else
                log::error "Specify --scenario, --all-active, or provide initialization directory"
                exit 1
            fi
            ;;
        "validate")
            if [[ -n "$SCENARIO_NAME" ]]; then
                local scenario_dir="${DEFAULT_SCENARIOS_DIR}/${SCENARIO_NAME}"
                if [[ ! -d "$scenario_dir" ]]; then
                    log::error "Scenario not found: $SCENARIO_NAME"
                    exit 1
                fi
                validate_scenario "$scenario_dir"
            else
                log::error "Specify --scenario"
                exit 1
            fi
            ;;
        "list-scenarios")
            list_scenarios
            ;;
        *)
            log::error "Unknown action: $ACTION"
            exit 1
            ;;
    esac
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main
fi