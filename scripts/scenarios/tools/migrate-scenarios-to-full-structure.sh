#!/bin/bash
# Scenario Migration Script
# Upgrades incomplete scenarios to the full SCENARIO_TEMPLATE structure

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIOS_DIR="$SCRIPT_DIR/../core"
TEMPLATE_DIR="$SCRIPT_DIR/../templates/full"
BACKUP_DIR="$SCRIPT_DIR/../.backups/$(date +%Y%m%d_%H%M%S)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# State
DRY_RUN=false
VERBOSE=false
FORCE=false

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_verbose() {
    if [[ "$VERBOSE" == "true" ]]; then
        echo -e "${CYAN}[VERBOSE]${NC} $1"
    fi
}

show_help() {
    cat << EOF
Usage: $0 [options] [scenario-name]

Upgrade scenarios to the full SCENARIO_TEMPLATE structure.

ARGUMENTS:
  scenario-name     Migrate specific scenario (optional, migrates all if not specified)

OPTIONS:
  -d, --dry-run     Show what would be done without executing
  -v, --verbose     Enable verbose logging
  -f, --force       Overwrite existing files (use with caution)
  -h, --help        Show this help message

EXAMPLES:
  # Migrate all incomplete scenarios
  $0

  # Migrate specific scenario
  $0 multi-modal-ai-assistant

  # Dry run to see what would happen
  $0 --dry-run --verbose

EOF
}

# Parse arguments
parse_arguments() {
    local target_scenario=""
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            -d|--dry-run)
                DRY_RUN=true
                shift
                ;;
            -v|--verbose)
                VERBOSE=true
                shift
                ;;
            -f|--force)
                FORCE=true
                shift
                ;;
            -h|--help)
                show_help
                exit 0
                ;;
            -*)
                log_error "Unknown option: $1"
                show_help
                exit 1
                ;;
            *)
                if [[ -z "$target_scenario" ]]; then
                    target_scenario="$1"
                else
                    log_error "Multiple scenario names provided"
                    exit 1
                fi
                shift
                ;;
        esac
    done
    
    echo "$target_scenario"
}

# Check if scenario needs migration
needs_migration() {
    local scenario_dir="$1"
    local missing_files=()
    
    # Required files for full structure
    local required_files=(
        "manifest.yaml"
        "deployment/startup.sh"
        "deployment/validate.sh"
        "deployment/monitor.sh"
        "initialization/database/schema.sql"
        "initialization/configuration/app-config.json"
    )
    
    for file in "${required_files[@]}"; do
        if [[ ! -f "$scenario_dir/$file" ]]; then
            missing_files+=("$file")
        fi
    done
    
    if [[ ${#missing_files[@]} -gt 0 ]]; then
        log_verbose "Missing files in $(basename "$scenario_dir"): ${missing_files[*]}"
        return 0  # Needs migration
    else
        return 1  # Already complete
    fi
}

# Extract scenario info from metadata.yaml
extract_scenario_info() {
    local metadata_file="$1"
    local info_type="$2"
    
    case "$info_type" in
        "id")
            grep "^[[:space:]]*id:" "$metadata_file" | awk -F': ' '{print $2}' | tr -d '"' | xargs
            ;;
        "name")
            grep "^[[:space:]]*name:" "$metadata_file" | awk -F': ' '{print $2}' | tr -d '"' | xargs
            ;;
        "description")
            grep "^[[:space:]]*description:" "$metadata_file" | awk -F': ' '{print $2}' | tr -d '"' | xargs
            ;;
    esac
}

# Create backup of scenario
backup_scenario() {
    local scenario_dir="$1"
    local scenario_name="$(basename "$scenario_dir")"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "Would backup: $scenario_dir -> $BACKUP_DIR/$scenario_name"
        return 0
    fi
    
    mkdir -p "$BACKUP_DIR"
    cp -r "$scenario_dir" "$BACKUP_DIR/"
    log_success "Backed up $scenario_name"
}

# Copy template file with substitutions
copy_template_file() {
    local source_file="$1"
    local dest_file="$2"
    local scenario_id="$3"
    local scenario_name="$4"
    local scenario_description="$5"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "Would copy: $source_file -> $dest_file"
        return 0
    fi
    
    # Create destination directory
    mkdir -p "$(dirname "$dest_file")"
    
    # Copy file with basic substitutions
    sed -e "s/scenario-name-here/$scenario_id/g" \
        -e "s/Human Readable Scenario Name/$scenario_name/g" \
        -e "s/Brief description of what this scenario tests and demonstrates/$scenario_description/g" \
        -e "s/{{ scenario.id }}/$scenario_id/g" \
        -e "s/{{ scenario.name }}/$scenario_name/g" \
        -e "s/{{ scenario.version }}/1.0.0/g" \
        "$source_file" > "$dest_file"
}

# Migrate single scenario
migrate_scenario() {
    local scenario_dir="$1"
    local scenario_name="$(basename "$scenario_dir")"
    
    log_info "Migrating scenario: $scenario_name"
    
    # Check if metadata.yaml exists
    if [[ ! -f "$scenario_dir/metadata.yaml" ]]; then
        log_error "metadata.yaml not found in $scenario_name, skipping"
        return 1
    fi
    
    # Extract scenario information
    local scenario_id scenario_name_human scenario_description
    scenario_id=$(extract_scenario_info "$scenario_dir/metadata.yaml" "id")
    scenario_name_human=$(extract_scenario_info "$scenario_dir/metadata.yaml" "name")
    scenario_description=$(extract_scenario_info "$scenario_dir/metadata.yaml" "description")
    
    log_verbose "ID: $scenario_id"
    log_verbose "Name: $scenario_name_human"
    log_verbose "Description: $scenario_description"
    
    # Create backup
    backup_scenario "$scenario_dir"
    
    # Copy missing structure components
    local template_files=(
        "manifest.yaml"
        "deployment/startup.sh"
        "deployment/validate.sh"
        "deployment/monitor.sh"
        "initialization/database/schema.sql"
        "initialization/database/seed.sql"
        "initialization/configuration/app-config.json"
        "initialization/configuration/resource-urls.json"
        "initialization/configuration/feature-flags.json"
        "initialization/workflows/triggers.yaml"
        "initialization/workflows/n8n/main-workflow.json"
        "initialization/ui/windmill-app.json"
    )
    
    for template_file in "${template_files[@]}"; do
        local source_path="$TEMPLATE_DIR/$template_file"
        local dest_path="$scenario_dir/$template_file"
        
        # Skip if file already exists and force is not set
        if [[ -f "$dest_path" && "$FORCE" != "true" ]]; then
            log_verbose "Skipping existing file: $template_file"
            continue
        fi
        
        # Skip if source doesn't exist
        if [[ ! -f "$source_path" ]]; then
            log_warning "Template file not found: $template_file"
            continue
        fi
        
        copy_template_file "$source_path" "$dest_path" "$scenario_id" "$scenario_name_human" "$scenario_description"
        log_verbose "Copied: $template_file"
    done
    
    # Create empty directories that might be needed
    local directories=(
        "initialization/storage"
        "initialization/vectors"
        "initialization/workflows/windmill"
        "initialization/workflows/node-red"
        "ui/scripts"
    )
    
    for dir in "${directories[@]}"; do
        local dir_path="$scenario_dir/$dir"
        if [[ ! -d "$dir_path" ]]; then
            if [[ "$DRY_RUN" == "true" ]]; then
                log_info "Would create directory: $dir"
            else
                mkdir -p "$dir_path"
                log_verbose "Created directory: $dir"
            fi
        fi
    done
    
    # Make scripts executable
    local scripts=(
        "deployment/startup.sh"
        "deployment/validate.sh"
        "deployment/monitor.sh"
        "test.sh"
    )
    
    for script in "${scripts[@]}"; do
        local script_path="$scenario_dir/$script"
        if [[ -f "$script_path" ]]; then
            if [[ "$DRY_RUN" == "true" ]]; then
                log_info "Would make executable: $script"
            else
                chmod +x "$script_path"
                log_verbose "Made executable: $script"
            fi
        fi
    done
    
    log_success "Migration completed for: $scenario_name"
}

# Main migration function
migrate_scenarios() {
    local target_scenario="$1"
    
    log_info "Starting scenario migration process..."
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log_warning "DRY RUN MODE - No actual changes will be made"
    fi
    
    # Validate template directory exists
    if [[ ! -d "$TEMPLATE_DIR" ]]; then
        log_error "Template directory not found: $TEMPLATE_DIR"
        exit 1
    fi
    
    # Find scenarios to migrate
    local scenarios_to_migrate=()
    
    if [[ -n "$target_scenario" ]]; then
        # Migrate specific scenario
        local scenario_path="$SCENARIOS_DIR/$target_scenario"
        if [[ ! -d "$scenario_path" ]]; then
            log_error "Scenario not found: $target_scenario"
            exit 1
        fi
        scenarios_to_migrate=("$scenario_path")
    else
        # Find all scenarios that need migration
        for scenario_dir in "$SCENARIOS_DIR"/*; do
            if [[ -d "$scenario_dir" && "$(basename "$scenario_dir")" != "." && "$(basename "$scenario_dir")" != ".." ]]; then
                if needs_migration "$scenario_dir"; then
                    scenarios_to_migrate+=("$scenario_dir")
                fi
            fi
        done
    fi
    
    if [[ ${#scenarios_to_migrate[@]} -eq 0 ]]; then
        log_info "No scenarios need migration. All scenarios are already complete!"
        exit 0
    fi
    
    log_info "Found ${#scenarios_to_migrate[@]} scenarios that need migration:"
    for scenario_dir in "${scenarios_to_migrate[@]}"; do
        echo "  - $(basename "$scenario_dir")"
    done
    
    echo ""
    
    # Migrate each scenario
    local success_count=0
    local error_count=0
    
    for scenario_dir in "${scenarios_to_migrate[@]}"; do
        if migrate_scenario "$scenario_dir"; then
            ((success_count++))
        else
            ((error_count++))
        fi
        echo ""
    done
    
    # Summary
    log_info "Migration Summary:"
    log_success "Successfully migrated: $success_count scenarios"
    if [[ $error_count -gt 0 ]]; then
        log_error "Failed migrations: $error_count scenarios"
    fi
    
    if [[ "$DRY_RUN" != "true" && "$success_count" -gt 0 ]]; then
        log_info "Backups created in: $BACKUP_DIR"
        log_warning "Remember to test migrated scenarios with: ./test.sh"
    fi
}

# Entry point
main() {
    local target_scenario
    target_scenario=$(parse_arguments "$@")
    
    echo -e "${CYAN}"
    echo "╔══════════════════════════════════════════════════════════════╗"
    echo "║               Scenario Migration Tool                       ║"
    echo "║            Upgrade to Full Template Structure               ║"
    echo "╚══════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
    
    migrate_scenarios "$target_scenario"
}

main "$@"