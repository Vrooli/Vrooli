#!/usr/bin/env bash
# ====================================================================
# Resource Interface Compliance Fixing Tool
# ====================================================================
#
# Automatically fixes common interface compliance issues in resource manage.sh scripts
# by generating missing required actions and standardizing function signatures.
#
# Usage:
#   ./fix-interface-compliance.sh [OPTIONS]
#
# Options:
#   --help              Show help message
#   --verbose           Enable verbose output
#   --resource NAME     Fix specific resource only
#   --batch             Enable batch processing mode
#   --resources LIST    Comma-separated list of resources for batch processing
#   --category NAME     Resource category for batch processing (ai, automation, storage, etc.)
#   --all               Process all available resources
#   --actions LIST      Comma-separated list of actions to generate
#   --dry-run           Show what would be changed without applying
#   --apply             Apply the fixes (default: dry-run only)
#   --backup            Create backup before applying (default: true)
#   --force             Apply even with warnings (default: false)
#   --jobs N            Number of parallel jobs for batch processing (default: 3)
#
# Examples:
#   # Single resource processing
#   ./fix-interface-compliance.sh --resource whisper --dry-run
#   ./fix-interface-compliance.sh --resource whisper --actions install,start,stop,status,logs --apply
#   
#   # Batch processing
#   ./fix-interface-compliance.sh --batch --resources "whisper,n8n,comfyui" --apply
#   ./fix-interface-compliance.sh --batch --category automation --dry-run
#   ./fix-interface-compliance.sh --batch --all --apply --jobs 5
#
# ====================================================================

set -euo pipefail

# Script directory and paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCES_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
TOOLS_DIR="$SCRIPT_DIR"
CONTRACTS_DIR="$RESOURCES_DIR/contracts"

# Source var.sh for directory variables
# shellcheck disable=SC1091
source "$SCRIPT_DIR/../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# Configuration
VERBOSE=false
SPECIFIC_RESOURCE=""
SPECIFIC_ACTIONS=""
RESOURCE_CATEGORY=""
DRY_RUN=true
CREATE_BACKUP=true
FORCE_APPLY=false
FIX_PERMISSIONS=true
FIX_PATTERNS=true

# Batch processing configuration
BATCH_MODE=false
BATCH_RESOURCES=""
BATCH_CATEGORY=""
BATCH_ALL=false
MAX_PARALLEL_JOBS=3

# Colors for output
if [[ -t 1 ]]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[1;33m'
    BLUE='\033[0;34m'
    BOLD='\033[1m'
    NC='\033[0m'
else
    RED='' GREEN='' YELLOW='' BLUE='' BOLD='' NC=''
fi

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} ✅ $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} ❌ $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} ⚠️  $1"
}

log_verbose() {
    if [[ "$VERBOSE" == "true" ]]; then
        echo -e "${BLUE}[VERBOSE]${NC} 🔍 $1"
    fi
}

# Show usage information
show_help() {
    cat << 'EOF'
Resource Interface Compliance Fixing Tool

USAGE:
    ./fix-interface-compliance.sh [OPTIONS]

DESCRIPTION:
    Automatically fixes common interface compliance issues in resource manage.sh
    scripts by generating missing required actions and standardizing patterns.

OPTIONS:
    --help              Show this help message
    --verbose           Enable verbose output with detailed information
    
    # Single Resource Mode:
    --resource <name>   Fix specific resource only (e.g., whisper, n8n)
    
    # Batch Processing Mode:
    --batch             Enable batch processing mode
    --resources <list>  Comma-separated list of resources to fix (requires --batch)
    --category <cat>    Fix all resources in category (ai, automation, storage, agents, search, execution)
    --all               Fix all available resources (requires --batch)
    --jobs <N>          Number of parallel jobs for batch processing (default: 3)
    
    # General Options:
    --actions <list>    Comma-separated actions to generate (install,start,stop,status,logs)
    --dry-run           Show what would be changed without applying (default)
    --apply             Apply the fixes (overrides dry-run)
    --backup            Create backup before applying (default: true)
    --no-backup         Skip creating backup files
    --force             Apply even with warnings (default: false)
    --fix-permissions   Fix script execute permissions (default: true)
    --no-permissions    Skip permission fixing
    --fix-patterns      Fix quoted action patterns (default: true)
    --no-patterns       Skip pattern fixing

EXAMPLES:
    # Single Resource Processing:
    ./fix-interface-compliance.sh --resource whisper --dry-run
    ./fix-interface-compliance.sh --resource whisper --actions install,start,stop,status,logs --apply
    ./fix-interface-compliance.sh --resource n8n --apply
    
    # Batch Processing:
    ./fix-interface-compliance.sh --batch --resources "whisper,n8n,comfyui" --dry-run
    ./fix-interface-compliance.sh --batch --category automation --apply
    ./fix-interface-compliance.sh --batch --all --apply --jobs 5
    ./fix-interface-compliance.sh --batch --category ai --apply --verbose

SAFETY FEATURES:
    • Dry-run mode by default - shows changes without applying
    • Automatic backups with timestamp
    • Validation of generated code before applying
    • Detailed diff output showing all changes

EXIT CODES:
    0 - Success (fixes applied or dry-run completed)
    1 - Fixes needed but not applied (dry-run mode)
    2 - Error in script execution
    3 - Resource validation failed

EOF
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --help|-h)
                show_help
                exit 0
                ;;
            --verbose|-v)
                VERBOSE=true
                shift
                ;;
            --resource)
                SPECIFIC_RESOURCE="$2"
                shift 2
                ;;
            --batch)
                BATCH_MODE=true
                shift
                ;;
            --resources)
                BATCH_RESOURCES="$2"
                shift 2
                ;;
            --category)
                if [[ "$BATCH_MODE" == "true" ]]; then
                    BATCH_CATEGORY="$2"
                else
                    RESOURCE_CATEGORY="$2"
                fi
                shift 2
                ;;
            --all)
                BATCH_ALL=true
                shift
                ;;
            --jobs)
                MAX_PARALLEL_JOBS="$2"
                shift 2
                ;;
            --actions)
                SPECIFIC_ACTIONS="$2"
                shift 2
                ;;
            --dry-run)
                DRY_RUN=true
                shift
                ;;
            --apply)
                DRY_RUN=false
                shift
                ;;
            --backup)
                CREATE_BACKUP=true
                shift
                ;;
            --no-backup)
                CREATE_BACKUP=false
                shift
                ;;
            --force)
                FORCE_APPLY=true
                shift
                ;;
            --fix-permissions)
                FIX_PERMISSIONS=true
                shift
                ;;
            --no-permissions)
                FIX_PERMISSIONS=false
                shift
                ;;
            --fix-patterns)
                FIX_PATTERNS=true
                shift
                ;;
            --no-patterns)
                FIX_PATTERNS=false
                shift
                ;;
            *)
                log_error "Unknown option: $1"
                show_help
                exit 2
                ;;
        esac
    done
    
    # Validate argument combinations
    if [[ "$BATCH_MODE" == "true" && -n "$SPECIFIC_RESOURCE" ]]; then
        log_error "Cannot use both --batch and --resource together"
        log_error "Use --batch with --resources, --category, or --all"
        exit 2
    fi
    
    if [[ -n "$BATCH_RESOURCES" && "$BATCH_MODE" != "true" ]]; then
        log_error "--resources requires --batch mode"
        exit 2
    fi
    
    if [[ -n "$BATCH_CATEGORY" && "$BATCH_MODE" != "true" ]]; then
        # If --category is used without --batch, treat as single resource category override
        RESOURCE_CATEGORY="$BATCH_CATEGORY"
        BATCH_CATEGORY=""
    fi
    
    if [[ "$BATCH_ALL" == "true" && "$BATCH_MODE" != "true" ]]; then
        log_error "--all requires --batch mode"
        exit 2
    fi
}

# Validate prerequisites
validate_prerequisites() {
    log_verbose "Validating prerequisites..."
    
    # Check if contracts directory exists
    if [[ ! -d "$CONTRACTS_DIR" ]]; then
        log_error "Contracts directory not found: $CONTRACTS_DIR"
        log_error "Please ensure you're running from the correct directory"
        exit 2
    fi
    
    # Check required tools
    local missing_tools=()
    for tool in bash grep sed awk; do
        if ! command -v "$tool" &> /dev/null; then
            missing_tools+=("$tool")
        fi
    done
    
    if [[ ${#missing_tools[@]} -gt 0 ]]; then
        log_error "Missing required tools: ${missing_tools[*]}"
        exit 2
    fi
    
    log_verbose "Prerequisites validation complete"
}

# Detect resource category from path
detect_resource_category() {
    local resource_path="$1"
    
    # Extract category from path (e.g., scripts/resources/ai/whisper -> ai)
    if [[ "$resource_path" =~ scripts/resources/([^/]+)/([^/]+) ]]; then
        echo "${BASH_REMATCH[1]}"
    else
        echo "unknown"
    fi
}

# Find resource directory
find_resource_dir() {
    local resource_name="$1"
    
    local resource_dir
    resource_dir=$(find "$RESOURCES_DIR" -name "$resource_name" -type d -not -path "*/tests/*" 2>/dev/null | head -1)
    
    if [[ -n "$resource_dir" && -d "$resource_dir" && -f "$resource_dir/manage.sh" ]]; then
        echo "$resource_dir"
        return 0
    else
        log_error "Resource directory not found for: $resource_name"
        return 1
    fi
}

# Find all available resources
find_all_resources() {
    log_verbose "Finding all available resources..." >&2
    
    local resources=()
    
    # Find all directories with manage.sh files
    while IFS= read -r -d '' manage_script; do
        local resource_dir
        resource_dir=$(dirname "$manage_script")
        local resource_name
        resource_name=$(basename "$resource_dir")
        
        # Skip test directories and invalid names
        if [[ "$resource_dir" != */tests/* && "$resource_name" =~ ^[a-z][a-z0-9-]*$ ]]; then
            resources+=("$resource_name")
            log_verbose "  Found resource: $resource_name" >&2
        fi
    done < <(find "$RESOURCES_DIR" -name "manage.sh" -type f -print0 2>/dev/null)
    
    # Sort and return unique resources
    printf '%s\n' "${resources[@]}" | sort -u
}

# Find resources by category
find_resources_by_category() {
    local target_category="$1"
    
    log_verbose "Finding resources in category: $target_category" >&2
    
    local resources=()
    
    # Find all directories with manage.sh files in the target category
    local category_dir="$RESOURCES_DIR/$target_category"
    
    if [[ ! -d "$category_dir" ]]; then
        log_error "Category directory not found: $category_dir"
        return 1
    fi
    
    while IFS= read -r -d '' manage_script; do
        local resource_dir
        resource_dir=$(dirname "$manage_script")
        local resource_name
        resource_name=$(basename "$resource_dir")
        
        # Validate resource name format
        if [[ "$resource_name" =~ ^[a-z][a-z0-9-]*$ ]]; then
            resources+=("$resource_name")
            log_verbose "  Found resource in $target_category: $resource_name" >&2
        fi
    done < <(find "$category_dir" -name "manage.sh" -type f -print0 2>/dev/null)
    
    if [[ ${#resources[@]} -eq 0 ]]; then
        log_warning "No resources found in category: $target_category"
        return 1
    fi
    
    # Sort and return resources
    printf '%s\n' "${resources[@]}" | sort -u
}

# Process resources in batch with simplified execution
process_batch_resources() {
    local resources_array=("$@")
    
    if [[ ${#resources_array[@]} -eq 0 ]]; then
        log_error "No resources provided for batch processing"
        return 1
    fi
    
    local total_resources=${#resources_array[@]}
    local successful_fixes=0
    local failed_fixes=0
    local no_fixes_needed=0
    
    log_info "Processing ${#resources_array[@]} resources in batch mode"
    echo
    
    # Process resources sequentially for now (parallel processing was causing hangs)
    local count=0
    for resource in "${resources_array[@]}"; do
        ((count++))
        log_info "[$count/$total_resources] Processing: $resource"
        echo "----------------------------------------"
        
        # Process single resource
        if analyze_and_fix_resource "$resource"; then
            ((no_fixes_needed++))
            log_verbose "$resource: No fixes needed" >&2
        else
            local exit_code=$?
            if [[ $exit_code -eq 1 && "$DRY_RUN" == "true" ]]; then
                # In dry-run mode, exit code 1 means fixes are needed
                ((successful_fixes++))
                log_verbose "$resource: Fixes needed (dry-run)" >&2
            else
                ((failed_fixes++))
                log_verbose "$resource: Failed to process" >&2
            fi
        fi
        
        echo
    done
    
    # Display batch summary
    echo "========================================"
    log_info "BATCH PROCESSING SUMMARY"
    echo "========================================"
    log_info "Total resources processed: $total_resources"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "Resources needing fixes: $successful_fixes"
        log_info "Resources with no issues: $no_fixes_needed"
    else
        log_success "Successfully fixed: $successful_fixes"
        log_info "No fixes needed: $no_fixes_needed"
    fi
    
    if [[ $failed_fixes -gt 0 ]]; then
        log_error "Failed to process: $failed_fixes"
    fi
    
    echo
    
    # Return appropriate exit code
    if [[ $failed_fixes -gt 0 ]]; then
        return 1
    elif [[ "$DRY_RUN" == "true" && $successful_fixes -gt 0 ]]; then
        return 1  # Indicate fixes are needed
    else
        return 0
    fi
}

# Get required actions from contracts
get_required_actions() {
    local category="$1"
    local core_contract="$CONTRACTS_DIR/v1.0/core.yaml"
    local category_contract="$CONTRACTS_DIR/v1.0/$category.yaml"
    
    # Use stderr for verbose messages to keep stdout clean for return value
    log_verbose "Reading required actions from contracts..." >&2
    log_verbose "Core contract: $core_contract" >&2
    log_verbose "Category contract: $category_contract" >&2
    
    # Extract core required actions
    local core_actions=""
    if [[ -f "$core_contract" ]]; then
        core_actions=$(grep -A 50 "required_actions:" "$core_contract" | grep "^  [a-z]" | sed 's/:.*$//' | sed 's/^  //' | tr '\n' ' ')
    fi
    
    log_verbose "Core actions found: $core_actions" >&2
    echo "$core_actions" | tr ' ' '\n' | grep -v '^$' | sort -u
}

# Extract implemented actions from manage.sh script
extract_implemented_actions() {
    local script_path="$1"
    
    log_verbose "Extracting implemented actions from: $script_path" >&2
    
    # Look for case statements with action patterns
    local actions=""
    
    # Pattern 1: case "$ACTION" in "action"|'action')
    actions+=$(grep -A 50 'case.*ACTION.*in' "$script_path" | grep -E '^\s*"[a-z-]+"\)|^\s*'\''[a-z-]+'\''.*\)' | sed -E 's/.*"([^"]+)".*/\1/;s/.*'\''([^'\'']+)'\''.*/\1/' | tr '\n' ' ')
    
    # Pattern 2: case statements without quotes
    actions+=$(grep -A 50 'case.*ACTION.*in' "$script_path" | grep -E '^\s*[a-z-]+\)' | sed -E 's/^\s*([a-z-]+)\).*/\1/' | tr '\n' ' ')
    
    log_verbose "Raw extracted actions: $actions" >&2
    
    # Clean and deduplicate
    echo "$actions" | tr ' ' '\n' | grep -E '^[a-z-]+$' | sort -u
}

# Check if required functions exist for actions
check_missing_functions() {
    local script_path="$1"
    local resource_name="$2"
    local required_actions="$3"
    
    log_verbose "Checking for missing functions in: $script_path" >&2
    
    local missing_functions=()
    
    while IFS= read -r action; do
        [[ -z "$action" ]] && continue
        
        local expected_function="${resource_name}::${action}"
        log_verbose "Checking for function: $expected_function" >&2
        
        # Check if the expected function exists
        if ! grep -q "^${expected_function}()" "$script_path"; then
            missing_functions+=("$action")
            log_verbose "Missing function: $expected_function" >&2
        else
            log_verbose "Found function: $expected_function" >&2
        fi
    done <<< "$required_actions"
    
    # Return missing functions as actions that need function stubs
    printf '%s\n' "${missing_functions[@]}"
}

# Generate function stub for missing action
generate_action_stub() {
    local resource_name="$1"
    local action="$2"
    local category="$3"
    
    log_verbose "Generating stub for action: $action"
    
    local function_name="${resource_name}::${action}"
    local description=""
    local example_body=""
    
    case "$action" in
        "install")
            description="Install the ${resource_name} resource"
            example_body='    log::info "Installing '"$resource_name"'..."
    
    # TODO: Add installation logic here
    # Example patterns:
    # - Check if already installed: if resource_installed; then return 2; fi
    # - Install dependencies: install_dependencies
    # - Download/setup service: setup_service
    # - Configure service: configure_service
    # - Start service: '"$resource_name"'::start
    
    log::success "'"$resource_name"' installation completed"'
            ;;
        "start")
            description="Start the ${resource_name} service"
            example_body='    log::info "Starting '"$resource_name"'..."
    
    # TODO: Add service start logic here
    # Example patterns:
    # - Check if already running: if is_running; then return 2; fi
    # - Start service container/process: start_service
    # - Wait for health check: wait_for_health
    
    log::success "'"$resource_name"' service started"'
            ;;
        "stop")
            description="Stop the ${resource_name} service"
            example_body='    log::info "Stopping '"$resource_name"'..."
    
    # TODO: Add service stop logic here
    # Example patterns:
    # - Check if running: if ! is_running; then return 2; fi
    # - Stop service gracefully: stop_service
    # - Force stop if needed: if FORCE=yes; then force_stop; fi
    
    log::success "'"$resource_name"' service stopped"'
            ;;
        "status")
            description="Show ${resource_name} service status"
            example_body='    log::info "Checking '"$resource_name"' status..."
    
    # TODO: Add status check logic here
    # Example patterns:
    # - Check if service is running: if is_running; then
    # - Check service health: if is_healthy; then
    # - Show detailed status information
    # - Return appropriate exit code (0=healthy, 1=unhealthy, 2=not running)
    
    log::info "'"$resource_name"' status check completed"'
            ;;
        "logs")
            description="Show ${resource_name} service logs"
            example_body='    log::info "Displaying '"$resource_name"' logs..."
    
    local lines="${LINES:-50}"
    
    # TODO: Add log display logic here
    # Example patterns:
    # - Show container logs: docker logs --tail "$lines" container_name
    # - Show service logs: tail -n "$lines" /path/to/logfile
    # - Format logs appropriately
    
    log::info "'"$resource_name"' logs displayed"'
            ;;
        *)
            description="Handle ${action} action for ${resource_name}"
            example_body='    log::info "Handling '"$action"' action for '"$resource_name"'..."
    
    # TODO: Implement '"$action"' action logic here
    
    log::success "'"$action"' action completed"'
            ;;
    esac
    
    cat << EOF

#######################################
# ${description}
# Arguments:
#   None
# Returns:
#   0 - Success
#   1 - Error
#   2 - Already in desired state (skip)
#######################################
${function_name}() {
${example_body}
}
EOF
}

# Generate case statement entry for action
generate_case_entry() {
    local resource_name="$1"
    local action="$2"
    
    cat << EOF
        "$action")
            ${resource_name}::${action}
            ;;
EOF
}

# Fix script permissions
fix_script_permissions() {
    local script_path="$1"
    
    log_verbose "Checking script permissions for: $script_path" >&2
    
    if [[ ! -x "$script_path" ]]; then
        if [[ "$DRY_RUN" == "true" ]]; then
            echo "PERMISSION FIX: Make script executable (chmod +x)"
            return 1
        else
            log_info "Making script executable: $script_path"
            chmod +x "$script_path"
            log_success "Fixed permissions for: $script_path"
            return 0
        fi
    else
        log_verbose "Script is already executable: $script_path" >&2
        return 0
    fi
}

# Fix quoted action patterns
fix_action_patterns() {
    local script_path="$1"
    
    log_verbose "Checking action patterns in: $script_path" >&2
    
    # Count quoted patterns in case statements
    local quoted_count
    quoted_count=$(grep -c '^\s*"[a-z-]*")' "$script_path" 2>/dev/null || echo "0")
    quoted_count=${quoted_count//[^0-9]/}  # Remove any non-numeric characters
    quoted_count=${quoted_count:-0}        # Default to 0 if empty
    
    if [[ $quoted_count -gt 0 ]]; then
        if [[ "$DRY_RUN" == "true" ]]; then
            echo "PATTERN FIX: Remove quotes from $quoted_count action patterns"
            echo "  Example: \"install\") -> install)"
            return 1
        else
            log_info "Fixing $quoted_count quoted action patterns in: $script_path"
            
            # Create a sed script to fix quoted patterns
            local temp_file="/tmp/fix_patterns_$$"
            
            # Use sed to remove quotes from action patterns in case statements
            sed -E 's/^(\s*)"([a-z-]+)"\)/\1\2)/' "$script_path" > "$temp_file"
            
            # Validate the result
            if bash -n "$temp_file"; then
                mv "$temp_file" "$script_path"
                log_success "Fixed action patterns in: $script_path"
                return 0
            else
                log_error "Pattern fixing created syntax errors, reverting"
                trash::safe_remove "$temp_file" --temp
                return 1
            fi
        fi
    else
        log_verbose "No quoted action patterns found" >&2
        return 0
    fi
}

# Add missing logs function if case statement references it but function doesn't exist
add_missing_logs_function() {
    local script_path="$1"
    local resource_name="$2"
    
    log_verbose "Checking for missing logs function in: $script_path" >&2
    
    # Check if case statement has logs action
    if grep -q '^\s*logs)\s*$' "$script_path"; then
        # Check if logs function exists
        if ! grep -q "^${resource_name}::logs()" "$script_path"; then
            log_verbose "Case statement has logs action but function missing" >&2
            
            if [[ "$DRY_RUN" == "true" ]]; then
                echo "LOGS FIX: Add missing ${resource_name}::logs() function"
                return 1
            else
                log_info "Adding missing logs function to: $script_path"
                
                # Find insertion point (before main function)
                local main_function_line
                main_function_line=$(grep -n "${resource_name}::main()" "$script_path" | head -1 | cut -d: -f1)
                
                if [[ -n "$main_function_line" ]]; then
                    local insert_line=$((main_function_line - 1))
                    local temp_file="/tmp/add_logs_$$"
                    
                    # Insert logs function before main
                    head -n "$insert_line" "$script_path" > "$temp_file"
                    cat >> "$temp_file" << EOF

#######################################
# Show ${resource_name} service logs
# Arguments:
#   None
# Returns:
#   0 - Success
#   1 - Error
#######################################
${resource_name}::logs() {
    log::info "Displaying ${resource_name} logs..."
    
    # TODO: Add log display logic here
    # Example patterns:
    # - Show container logs: docker logs --tail "\$lines" container_name
    # - Show service logs: tail -n "\$lines" /path/to/logfile
    # - Format logs appropriately
    
    log::info "${resource_name} logs displayed"
}
EOF
                    tail -n +"$main_function_line" "$script_path" >> "$temp_file"
                    
                    # Validate and apply
                    if bash -n "$temp_file"; then
                        mv "$temp_file" "$script_path"
                        log_success "Added missing logs function to: $script_path"
                        return 0
                    else
                        log_error "Adding logs function created syntax errors"
                        trash::safe_remove "$temp_file" --temp
                        return 1
                    fi
                else
                    log_warning "Could not find main function to insert logs function"
                    return 1
                fi
            fi
        else
            log_verbose "logs function already exists" >&2
            return 0
        fi
    else
        log_verbose "No logs action in case statement" >&2
        return 0
    fi
}

# Analyze resource and generate fixes
analyze_and_fix_resource() {
    local resource_name="$1"
    
    log_info "Analyzing resource: $resource_name"
    
    # Find resource directory
    local resource_dir
    if ! resource_dir=$(find_resource_dir "$resource_name"); then
        return 1
    fi
    
    local manage_script="$resource_dir/manage.sh"
    log_verbose "Resource directory: $resource_dir"
    log_verbose "Manage script: $manage_script"
    
    # Detect or use provided category
    local category="$RESOURCE_CATEGORY"
    if [[ -z "$category" ]]; then
        category=$(detect_resource_category "$resource_dir")
        log_verbose "Auto-detected category: $category"
    fi
    
    # Get required actions
    local required_actions
    if [[ -n "$SPECIFIC_ACTIONS" ]]; then
        required_actions=$(echo "$SPECIFIC_ACTIONS" | tr ',' '\n')
        log_verbose "Using specified actions: $SPECIFIC_ACTIONS"
    else
        required_actions=$(get_required_actions "$category")
        log_verbose "Using contract-based actions for category: $category"
    fi
    
    # Get implemented actions  
    local implemented_actions
    implemented_actions=$(extract_implemented_actions "$manage_script")
    
    log_verbose "Required actions:"
    echo "$required_actions" | while read -r action; do
        [[ -n "$action" ]] && log_verbose "  - $action"
    done
    
    log_verbose "Implemented actions:"
    echo "$implemented_actions" | while read -r action; do
        [[ -n "$action" ]] && log_verbose "  - $action"
    done
    
    # Comprehensive compliance checks
    local fixes_needed=()
    local fixes_applied=0
    local fixes_failed=0
    
    # Check and fix permissions
    if [[ "$FIX_PERMISSIONS" == "true" ]]; then
        if ! fix_script_permissions "$manage_script"; then
            fixes_needed+=("permissions")
        else
            if [[ "$DRY_RUN" == "false" ]]; then
                ((fixes_applied++))
            fi
        fi
    fi
    
    # Check and fix action patterns
    if [[ "$FIX_PATTERNS" == "true" ]]; then
        if ! fix_action_patterns "$manage_script"; then
            fixes_needed+=("patterns")
        else
            if [[ "$DRY_RUN" == "false" ]]; then
                ((fixes_applied++))
            fi
        fi
    fi
    
    # Check and add missing logs function
    if ! add_missing_logs_function "$manage_script" "$resource_name"; then
        fixes_needed+=("logs-function")
    else
        if [[ "$DRY_RUN" == "false" ]]; then
            ((fixes_applied++))
        fi
    fi
    
    # Check for missing functions (this is the real validation)
    local missing_actions
    missing_actions=$(check_missing_functions "$manage_script" "$resource_name" "$required_actions")
    
    # Convert to array for easier handling
    local missing_actions_array=()
    while IFS= read -r action; do
        [[ -n "$action" ]] && missing_actions_array+=("$action")
    done <<< "$missing_actions"
    
    # Check if any fixes are needed
    local total_fixes=$((${#fixes_needed[@]} + ${#missing_actions_array[@]}))
    
    if [[ $total_fixes -eq 0 ]]; then
        log_success "No compliance issues found for $resource_name"
        return 0
    fi
    
    if [[ ${#missing_actions_array[@]} -gt 0 ]]; then
        log_info "Missing functions for $resource_name: ${missing_actions_array[*]}"
        fixes_needed+=("missing-functions")
    fi
    
    # Generate fixes
    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "DRY RUN - Showing all fixes that would be applied to $manage_script:"
        echo
        
        # Show all needed fixes
        for fix_type in "${fixes_needed[@]}"; do
            case "$fix_type" in
                "permissions")
                    echo "# ========== PERMISSION FIX =========="
                    fix_script_permissions "$manage_script"
                    echo
                    ;;
                "patterns")
                    echo "# ========== PATTERN FIXES =========="
                    fix_action_patterns "$manage_script"
                    echo
                    ;;
                "logs-function")
                    echo "# ========== LOGS FUNCTION FIX =========="
                    add_missing_logs_function "$manage_script" "$resource_name"
                    echo
                    ;;
                "missing-functions")
                    echo "# ========== MISSING FUNCTION STUBS =========="
                    for action in "${missing_actions_array[@]}"; do
                        echo "# Function stub for: $action"
                        generate_action_stub "$resource_name" "$action" "$category"
                        echo
                    done
                    
                    echo "# ========== CASE STATEMENT ENTRIES =========="
                    echo "# Add these entries to the case statement in main function (if not already present):"
                    for action in "${missing_actions_array[@]}"; do
                        generate_case_entry "$resource_name" "$action"
                    done
                    ;;
            esac
        done
        
        echo
        log_warning "This was a dry run. Use --apply to actually make changes."
        return 1
        
    else
        # Apply fixes
        log_info "Applying $total_fixes fix(es) to $resource_name..."
        
        # Create backup if needed and not already done
        if [[ "$CREATE_BACKUP" == "true" && $total_fixes -gt 0 ]]; then
            local backup_path="${manage_script}.backup.$(date +%Y%m%d_%H%M%S)"
            cp "$manage_script" "$backup_path"
            log_success "Backup created: $backup_path"
        fi
        
        # Apply function stub fixes if needed
        if [[ ${#missing_actions_array[@]} -gt 0 ]]; then
            apply_fixes "$resource_name" "$manage_script" "${missing_actions_array[@]}"
            local stub_result=$?
            if [[ $stub_result -eq 0 ]]; then
                ((fixes_applied++))
            else
                ((fixes_failed++))
            fi
        fi
        
        # Report results
        if [[ $fixes_failed -eq 0 ]]; then
            log_success "Successfully applied all $total_fixes fix(es) to $resource_name"
            log_info "Fixed: ${fixes_needed[*]}"
            return 0
        else
            log_error "Failed to apply $fixes_failed fix(es) to $resource_name"
            return 1
        fi
    fi
}

# Apply fixes to the manage.sh file
apply_fixes() {
    local resource_name="$1"
    local script_path="$2"
    shift 2
    local missing_actions=("$@")
    
    log_info "Applying function stub fixes to $script_path..."
    
    # Generate temporary file with fixes
    local temp_file="/tmp/manage_${resource_name}_fixed_$$"
    local additions_file="/tmp/manage_${resource_name}_additions_$$"
    
    # Generate function stubs
    {
        echo
        echo "# ========================================"
        echo "# GENERATED FUNCTION STUBS"
        echo "# Generated by fix-interface-compliance.sh on $(date)"
        echo "# ========================================"
        
        for action in "${missing_actions[@]}"; do
            generate_action_stub "$resource_name" "$action" "$(detect_resource_category "$(dirname "$script_path")")"
        done
    } > "$additions_file"
    
    # Find insertion point (before main function or at end)
    local main_function_line
    main_function_line=$(grep -n "${resource_name}::main()" "$script_path" | head -1 | cut -d: -f1)
    
    if [[ -n "$main_function_line" ]]; then
        # Insert before main function
        local insert_line=$((main_function_line - 1))
        head -n "$insert_line" "$script_path" > "$temp_file"
        cat "$additions_file" >> "$temp_file"
        tail -n +"$main_function_line" "$script_path" >> "$temp_file"
        log_verbose "Inserted function stubs before main function (line $main_function_line)"
    else
        # Append at end of file
        cp "$script_path" "$temp_file"
        cat "$additions_file" >> "$temp_file"
        log_verbose "Appended function stubs at end of file"
    fi
    
    # Validate generated script
    if ! bash -n "$temp_file"; then
        log_error "Generated script has syntax errors!"
        trash::safe_remove "$temp_file" --temp
        trash::safe_remove "$additions_file" --temp
        return 1
    fi
    
    # Apply changes
    mv "$temp_file" "$script_path"
    trash::safe_remove "$additions_file" --temp
    
    log_success "Applied fixes to $script_path"
    log_info "Generated function stubs for: ${missing_actions[*]}"
    log_info "Note: Case statement entries already exist - only the missing functions were added."
    
    return 0
}

# Main execution function
main() {
    parse_args "$@"
    validate_prerequisites
    
    log_info "Resource Interface Compliance Fixing Tool"
    echo "========================================"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "Running in DRY-RUN mode - no changes will be applied"
    else
        log_warning "Running in APPLY mode - changes will be made to files"
        if [[ "$CREATE_BACKUP" == "true" ]]; then
            log_info "Backups will be created automatically"
        fi
    fi
    echo
    
    # Handle batch vs single resource processing
    if [[ "$BATCH_MODE" == "true" ]]; then
        # Simplified batch processing
        log_info "Batch mode enabled"
        
        local resources_to_process=()
        
        if [[ -n "$BATCH_RESOURCES" ]]; then
            log_info "Processing specified resources: $BATCH_RESOURCES"
            IFS=',' read -ra resources_to_process <<< "$BATCH_RESOURCES"
        elif [[ -n "$BATCH_CATEGORY" ]]; then
            log_info "Processing category: $BATCH_CATEGORY"
            local category_resources
            if ! category_resources=$(find_resources_by_category "$BATCH_CATEGORY"); then
                log_error "No resources found in category: $BATCH_CATEGORY"
                exit 2
            fi
            while IFS= read -r resource; do
                [[ -n "$resource" ]] && resources_to_process+=("$resource")
            done <<< "$category_resources"
        elif [[ "$BATCH_ALL" == "true" ]]; then
            log_info "Processing all resources"
            local all_resources
            if ! all_resources=$(find_all_resources); then
                log_error "No resources found"
                exit 2
            fi
            while IFS= read -r resource; do
                [[ -n "$resource" ]] && resources_to_process+=("$resource")
            done <<< "$all_resources"
        else
            log_error "Batch mode requires --resources, --category, or --all"
            exit 2
        fi
        
        if [[ ${#resources_to_process[@]} -eq 0 ]]; then
            log_error "No resources found to process"
            exit 2
        fi
        
        # Process resources
        if ! process_batch_resources "${resources_to_process[@]}"; then
            if [[ "$DRY_RUN" == "true" ]]; then
                log_info "Some resources need fixes. Use --apply to apply them."
                exit 1
            else
                log_error "Some resources failed to process."
                exit 1
            fi
        fi
        
    elif [[ -n "$SPECIFIC_RESOURCE" ]]; then
        # Single resource processing
        log_info "Single resource mode: $SPECIFIC_RESOURCE"
        echo
        
        if ! analyze_and_fix_resource "$SPECIFIC_RESOURCE"; then
            if [[ "$DRY_RUN" == "true" ]]; then
                log_info "Use --apply to apply the fixes shown above"
                exit 1
            else
                exit 1
            fi
        fi
        
    else
        log_error "No resource specified. Use either:"
        log_info "  Single resource: $0 --resource NAME"
        log_info "  Batch processing: $0 --batch [--resources LIST | --category NAME | --all]"
        log_info "Examples:"
        log_info "  $0 --resource whisper --dry-run"
        log_info "  $0 --batch --category automation --apply"
        exit 2
    fi
    
    log_success "Compliance fixing completed successfully!"
}

# Execute main function only if not being sourced
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi