#!/usr/bin/env bash
set -euo pipefail

# Update Repository Placeholders Script
# Replaces REPOSITORY_URL_PLACEHOLDER and related placeholders in scenario templates
# with actual repository information from service.json

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"

# Source utilities
if [[ -f "${PROJECT_ROOT}/scripts/helpers/utils/log.sh" ]]; then
    # shellcheck disable=SC1091
    source "${PROJECT_ROOT}/scripts/helpers/utils/log.sh"
else
    log::info() { echo "[INFO] $1"; }
    log::success() { echo "[SUCCESS] $1"; }
    log::warning() { echo "[WARNING] $1"; }
    log::error() { echo "[ERROR] $1" >&2; }
fi

if [[ -f "${PROJECT_ROOT}/scripts/helpers/utils/repository.sh" ]]; then
    # shellcheck disable=SC1091
    source "${PROJECT_ROOT}/scripts/helpers/utils/repository.sh"
else
    log::error "Repository helper not found: ${PROJECT_ROOT}/scripts/helpers/utils/repository.sh"
    exit 1
fi

#######################################
# Update repository placeholders in a single file
# Arguments:
#   $1 - File path to update
# Returns:
#   0 on success, 1 on failure
#######################################
update_file_placeholders() {
    local file_path="$1"
    
    if [[ ! -f "$file_path" ]]; then
        log::error "File not found: $file_path"
        return 1
    fi
    
    # Check if file contains any repository placeholders
    if ! grep -q "REPOSITORY_URL_PLACEHOLDER\|ISSUES_URL_PLACEHOLDER\|HOMEPAGE_URL_PLACEHOLDER" "$file_path"; then
        return 0  # No placeholders to replace
    fi
    
    # Get repository information
    local repo_url repo_branch
    repo_url=$(repository::get_url 2>/dev/null || echo "")
    repo_branch=$(repository::get_branch 2>/dev/null || echo "main")
    
    if [[ -z "$repo_url" ]]; then
        log::warning "No repository URL found in service.json, skipping: $file_path"
        return 0
    fi
    
    log::info "Updating repository placeholders in: $file_path"
    
    # Create backup
    cp "$file_path" "${file_path}.backup"
    
    # Replace placeholders
    sed -i "s|REPOSITORY_URL_PLACEHOLDER|${repo_url}|g" "$file_path"
    sed -i "s|ISSUES_URL_PLACEHOLDER|${repo_url}/issues|g" "$file_path"
    sed -i "s|HOMEPAGE_URL_PLACEHOLDER|${repo_url}|g" "$file_path"
    
    # Log what was changed
    local changes
    changes=$(diff "${file_path}.backup" "$file_path" | grep -c "^<\|^>" || echo "0")
    if [[ "$changes" -gt 0 ]]; then
        log::success "Updated $((changes / 2)) placeholder(s) in: $file_path"
        
        # Show what was changed if verbose
        if [[ "${VERBOSE:-false}" == "true" ]]; then
            echo "Changes:"
            diff "${file_path}.backup" "$file_path" | grep -E "^<|^>" | head -6
        fi
    fi
    
    # Remove backup on success
    rm "${file_path}.backup"
    
    return 0
}

#######################################
# Update repository placeholders in all scenario templates
# Returns:
#   0 on success
#######################################
update_all_templates() {
    local templates_dir="${PROJECT_ROOT}/scripts/scenarios/templates"
    local updated_count=0
    local total_count=0
    
    log::info "Scanning for repository placeholders in scenario templates..."
    
    # Find all service.json files in templates
    while IFS= read -r -d '' template_file; do
        ((total_count++))
        if update_file_placeholders "$template_file"; then
            ((updated_count++))
        fi
    done < <(find "$templates_dir" -name "service.json" -print0)
    
    # Also check other JSON files that might contain placeholders
    while IFS= read -r -d '' json_file; do
        if [[ "$json_file" != *"/service.json" ]]; then
            ((total_count++))
            if update_file_placeholders "$json_file"; then
                ((updated_count++))
            fi
        fi
    done < <(find "$templates_dir" -name "*.json" -print0)
    
    log::success "Processed $total_count template files, updated $updated_count files"
    
    return 0
}

#######################################
# Update repository placeholders in specific scenario
# Arguments:
#   $1 - Scenario name or path
# Returns:
#   0 on success, 1 on failure
#######################################
update_scenario_templates() {
    local scenario="$1"
    local scenario_path
    
    # Determine if input is a path or scenario name
    if [[ -d "$scenario" ]]; then
        scenario_path="$scenario"
    elif [[ -d "${PROJECT_ROOT}/scripts/scenarios/core/$scenario" ]]; then
        scenario_path="${PROJECT_ROOT}/scripts/scenarios/core/$scenario"
    else
        log::error "Scenario not found: $scenario"
        return 1
    fi
    
    log::info "Updating repository placeholders in scenario: $scenario_path"
    
    local updated_count=0
    local total_count=0
    
    # Find all JSON files in the scenario
    while IFS= read -r -d '' json_file; do
        ((total_count++))
        if update_file_placeholders "$json_file"; then
            ((updated_count++))
        fi
    done < <(find "$scenario_path" -name "*.json" -print0)
    
    log::success "Processed $total_count files in scenario, updated $updated_count files"
    
    return 0
}

#######################################
# Show help message
#######################################
show_help() {
    cat <<EOF
Update Repository Placeholders Script

Replaces repository placeholders in scenario templates with actual repository
information from service.json configuration.

Usage:
    $0 [options] [scenario]

Options:
    --all, -a       Update all scenario templates (default)
    --scenario, -s  Update specific scenario (provide name or path)
    --verbose, -v   Show detailed output
    --dry-run       Show what would be done without making changes
    --help, -h      Show this help message

Examples:
    $0                              # Update all templates
    $0 --scenario ai-assistant      # Update specific scenario
    $0 --verbose --dry-run          # Show what would be changed

Placeholders replaced:
    REPOSITORY_URL_PLACEHOLDER  -> Repository URL from service.json
    ISSUES_URL_PLACEHOLDER      -> Repository URL + /issues
    HOMEPAGE_URL_PLACEHOLDER    -> Repository URL

EOF
}

#######################################
# Main function
#######################################
main() {
    local action="all"
    local scenario=""
    local dry_run=false
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --all|-a)
                action="all"
                shift
                ;;
            --scenario|-s)
                action="scenario"
                scenario="${2:-}"
                if [[ -z "$scenario" ]]; then
                    log::error "Scenario name required with --scenario"
                    exit 1
                fi
                shift 2
                ;;
            --verbose|-v)
                export VERBOSE=true
                shift
                ;;
            --dry-run)
                dry_run=true
                shift
                ;;
            --help|-h)
                show_help
                exit 0
                ;;
            *)
                # Assume it's a scenario name if no flag
                if [[ -z "$scenario" && "$action" == "all" ]]; then
                    action="scenario"
                    scenario="$1"
                else
                    log::error "Unknown option: $1"
                    show_help
                    exit 1
                fi
                shift
                ;;
        esac
    done
    
    # Check if repository configuration exists
    if ! repository::get_url >/dev/null 2>&1; then
        log::error "No repository configuration found in service.json"
        log::info "Please ensure .vrooli/service.json contains repository information"
        exit 1
    fi
    
    # Show repository information
    local repo_url repo_branch
    repo_url=$(repository::get_url)
    repo_branch=$(repository::get_branch)
    log::info "Repository URL: $repo_url"
    log::info "Repository branch: $repo_branch"
    
    if [[ "$dry_run" == "true" ]]; then
        log::info "DRY RUN MODE - No files will be modified"
        # In dry run, we would need to implement preview logic
        log::info "Would replace placeholders with repository information"
        return 0
    fi
    
    # Execute the requested action
    case "$action" in
        "all")
            update_all_templates
            ;;
        "scenario")
            update_scenario_templates "$scenario"
            ;;
        *)
            log::error "Unknown action: $action"
            exit 1
            ;;
    esac
    
    log::success "Repository placeholder update completed successfully!"
}

# Only run main if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi