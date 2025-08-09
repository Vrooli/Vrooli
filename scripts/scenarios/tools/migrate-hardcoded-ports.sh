#!/usr/bin/env bash
set -euo pipefail

# Port Abstraction Migration Tool
# Converts hardcoded localhost:PORT references to service references
# Usage: ./migrate-hardcoded-ports.sh [scenario-path] [options]

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"

# Source required utilities
# shellcheck disable=SC1091
source "$PROJECT_ROOT/scripts/lib/utils/log.sh"
# shellcheck disable=SC1091
source "$PROJECT_ROOT/scripts/resources/port_registry.sh"

#######################################
# Configuration and constants
#######################################

# Port mappings from port_registry.sh to service names
declare -A PORT_TO_SERVICE_MAP=()
for service in "${!RESOURCE_PORTS[@]}"; do
    port="${RESOURCE_PORTS[$service]}"
    PORT_TO_SERVICE_MAP["$port"]="$service"
done

# Common hardcoded port patterns to replace
declare -A PORT_PATTERNS=(
    ["11434"]="ollama"
    ["5678"]="n8n" 
    ["5681"]="windmill"
    ["9200"]="searxng"
    ["6333"]="qdrant"
    ["9000"]="minio"
    ["8200"]="vault"
    ["5433"]="postgres"
    ["6380"]="redis"
    ["4110"]="browserless"
    ["8090"]="whisper"
    ["8188"]="comfyui"
    ["11450"]="unstructured-io"
    ["9009"]="questdb"
    ["4113"]="agent-s2"
    ["2358"]="judge0"
)

#######################################
# Script options and help
#######################################

SCENARIO_PATH=""
DRY_RUN=false
VERBOSE=false
FORCE=false
VALIDATE_ONLY=false

show_help() {
    cat << 'EOF'
Port Abstraction Migration Tool

Converts hardcoded localhost:PORT references to service references (${service.NAME.url})
across all files in a scenario directory.

USAGE:
    migrate-hardcoded-ports.sh <scenario-path> [options]

ARGUMENTS:
    scenario-path    Path to scenario directory to migrate

OPTIONS:
    --dry-run       Show what would be changed without making changes
    --verbose       Show detailed output during migration
    --force         Overwrite files even if they contain mixed patterns
    --validate      Only validate current state, don't migrate
    --help          Show this help message

EXAMPLES:
    # Dry run to see what would be migrated
    migrate-hardcoded-ports.sh scripts/scenarios/core/research-assistant --dry-run

    # Migrate with verbose output
    migrate-hardcoded-ports.sh scripts/scenarios/core/research-assistant --verbose

    # Validate current state of scenario
    migrate-hardcoded-ports.sh scripts/scenarios/core/research-assistant --validate

SUPPORTED PATTERNS:
    localhost:11434 → \${service.ollama.url}
    localhost:5678  → \${service.n8n.url}
    localhost:5681  → \${service.windmill.url}
    localhost:9200  → \${service.searxng.url}
    ... and 10+ more services

FILES PROCESSED:
    - .vrooli/service.json (deployment/monitoring sections)
    - initialization/**/*.json (N8N workflows, Windmill apps, config files)
    - **/*.yaml (trigger configurations)
    - **/*.ts (TypeScript automation scripts)
    - **/*.sh (shell scripts with URLs)

EOF
}

#######################################
# Utility functions
#######################################

# Check if a file should be processed
should_process_file() {
    local file="$1"
    local filename
    filename="$(basename "$file")"
    
    # Skip if file doesn't exist
    [[ ! -f "$file" ]] && return 1
    
    # Skip backup files
    [[ "$filename" =~ \.(backup|bak|orig)$ ]] && return 1
    
    # Skip git files
    [[ "$filename" =~ ^\.git ]] && return 1
    
    # Process text files that commonly contain URLs
    if [[ "$filename" =~ \.(json|yaml|yml|ts|js|sh|md)$ ]]; then
        return 0
    fi
    
    # Check if file is text
    if file -b --mime-type "$file" | grep -q "text/"; then
        return 0
    fi
    
    return 1
}

# Count hardcoded port references in a file
count_hardcoded_ports() {
    local file="$1"
    local count=0
    
    # Count localhost:XXXX patterns
    while IFS= read -r line; do
        # Match localhost:PORT patterns
        local matches
        matches=$(echo "$line" | grep -o 'localhost:[0-9]\{4,5\}' | wc -l)
        ((count += matches))
    done < "$file"
    
    echo "$count"
}

# Get service name from port number
get_service_from_port() {
    local port="$1"
    echo "${PORT_PATTERNS[$port]:-unknown}"
}

# Migrate hardcoded ports in a single file
migrate_file() {
    local file="$1"
    local changed=false
    local backup_file="${file}.migration-backup"
    
    log::info "Processing: $(realpath --relative-to="$PROJECT_ROOT" "$file")"
    
    # Create backup
    cp "$file" "$backup_file"
    
    # Read file content
    local content
    content=$(cat "$file")
    local original_content="$content"
    
    # Apply migrations for each known port pattern
    for port in "${!PORT_PATTERNS[@]}"; do
        local service="${PORT_PATTERNS[$port]}"
        
        # Replace localhost:PORT with ${service.NAME.url}
        local old_pattern="localhost:$port"
        local new_pattern="\${service.$service.url}"
        
        if echo "$content" | grep -q "$old_pattern"; then
            content="${content//$old_pattern/$new_pattern}"
            changed=true
            
            [[ "$VERBOSE" == "true" ]] && log::info "  $old_pattern → $new_pattern"
        fi
        
        # Also handle http://localhost:PORT patterns
        old_pattern="http://localhost:$port"
        new_pattern="\${service.$service.url}"
        
        if echo "$content" | grep -q "$old_pattern"; then
            content="${content//$old_pattern/$new_pattern}"
            changed=true
            
            [[ "$VERBOSE" == "true" ]] && log::info "  $old_pattern → $new_pattern"
        fi
        
        # Handle https://localhost:PORT patterns (less common)
        old_pattern="https://localhost:$port"
        new_pattern="\${service.$service.url}"
        
        if echo "$content" | grep -q "$old_pattern"; then
            # For HTTPS, we need to be more careful - might need manual review
            log::warn "Found HTTPS pattern in $file - review manually: $old_pattern"
        fi
    done
    
    # Write back changes or show diff
    if [[ "$changed" == "true" ]]; then
        if [[ "$DRY_RUN" == "true" ]]; then
            log::info "  [DRY RUN] Would update file with service references"
            # Show a few example changes
            local changes
            changes=$(diff -u "$backup_file" <(echo "$content") | head -20 || true)
            [[ -n "$changes" ]] && echo "$changes"
        else
            echo "$content" > "$file"
            log::success "  ✓ Updated with service references"
        fi
    else
        [[ "$VERBOSE" == "true" ]] && log::info "  No hardcoded ports found"
    fi
    
    # Remove backup if no changes or if dry run
    if [[ "$DRY_RUN" == "true" ]] || [[ "$changed" == "false" ]]; then
        rm -f "$backup_file"
    else
        [[ "$VERBOSE" == "true" ]] && log::info "  Backup created: $backup_file"
    fi
    
    echo "$changed"
}

# Validate a scenario directory
validate_scenario() {
    local scenario_path="$1"
    local total_files=0
    local files_with_ports=0
    local total_port_references=0
    
    log::info "Validating scenario: $(basename "$scenario_path")"
    
    # Find all files to process (using simpler approach)
    while IFS= read -r file; do
        if should_process_file "$file"; then
            ((total_files++))
            
            local port_count
            port_count=$(count_hardcoded_ports "$file")
            
            if [[ "$port_count" -gt 0 ]]; then
                ((files_with_ports++))
                ((total_port_references += port_count))
                
                if [[ "$VERBOSE" == "true" ]]; then
                    local relative_path
                    relative_path="$(realpath --relative-to="$scenario_path" "$file")"
                    log::warn "  $relative_path: $port_count hardcoded port(s)"
                fi
            fi
        fi
    done < <(find "$scenario_path" -type f)
    
    # Summary
    echo ""
    log::info "Validation Summary:"
    log::info "  Total files processed: $total_files"
    log::info "  Files with hardcoded ports: $files_with_ports"
    log::info "  Total hardcoded port references: $total_port_references"
    
    if [[ "$files_with_ports" -eq 0 ]]; then
        log::success "✅ No hardcoded ports found - scenario is fully abstracted!"
        return 0
    else
        log::warn "⚠️  Found hardcoded ports that should be migrated"
        return 1
    fi
}

# Migrate entire scenario directory
migrate_scenario() {
    local scenario_path="$1"
    local total_files=0
    local migrated_files=0
    
    log::info "Migrating scenario: $(basename "$scenario_path")"
    [[ "$DRY_RUN" == "true" ]] && log::info "[DRY RUN MODE - No files will be modified]"
    
    # Find and process all files (using simpler approach)
    while IFS= read -r file; do
        if should_process_file "$file"; then
            ((total_files++))
            
            local file_changed
            file_changed=$(migrate_file "$file")
            
            if [[ "$file_changed" == "true" ]]; then
                ((migrated_files++))
            fi
        fi
    done < <(find "$scenario_path" -type f)
    
    # Summary
    echo ""
    log::info "Migration Summary:"
    log::info "  Total files processed: $total_files"
    log::info "  Files migrated: $migrated_files"
    
    if [[ "$migrated_files" -eq 0 ]]; then
        log::success "✅ No migration needed - scenario already uses service references!"
    else
        if [[ "$DRY_RUN" == "true" ]]; then
            log::info "🔍 Dry run complete - run without --dry-run to apply changes"
        else
            log::success "✅ Migration complete! $migrated_files files updated"
            log::info "💡 Backup files created with .migration-backup extension"
            log::info "🧪 Run tests to verify the migration worked correctly"
        fi
    fi
}

#######################################
# Main script logic
#######################################

main() {
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --dry-run)
                DRY_RUN=true
                shift
                ;;
            --verbose)
                VERBOSE=true
                shift
                ;;
            --force)
                FORCE=true
                shift
                ;;
            --validate)
                VALIDATE_ONLY=true
                shift
                ;;
            --help|-h)
                show_help
                exit 0
                ;;
            -*)
                log::error "Unknown option: $1"
                show_help
                exit 1
                ;;
            *)
                if [[ -z "$SCENARIO_PATH" ]]; then
                    SCENARIO_PATH="$1"
                else
                    log::error "Multiple paths provided. Only one scenario path is allowed."
                    exit 1
                fi
                shift
                ;;
        esac
    done
    
    # Validate required arguments
    if [[ -z "$SCENARIO_PATH" ]]; then
        log::error "Scenario path is required"
        show_help
        exit 1
    fi
    
    # Validate scenario path exists
    if [[ ! -d "$SCENARIO_PATH" ]]; then
        log::error "Scenario directory does not exist: $SCENARIO_PATH"
        exit 1
    fi
    
    # Convert to absolute path
    SCENARIO_PATH="$(realpath "$SCENARIO_PATH")"
    
    # Show configuration
    [[ "$VERBOSE" == "true" ]] && {
        log::info "Configuration:"
        log::info "  Scenario: $SCENARIO_PATH"
        log::info "  Dry run: $DRY_RUN"
        log::info "  Force: $FORCE"
        log::info "  Validate only: $VALIDATE_ONLY"
    }
    
    # Execute requested action
    if [[ "$VALIDATE_ONLY" == "true" ]]; then
        validate_scenario "$SCENARIO_PATH"
    else
        migrate_scenario "$SCENARIO_PATH"
    fi
}

# Run main function if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi