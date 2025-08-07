#!/usr/bin/env bash
set -euo pipefail

# Port Abstraction Validation Tool
# Checks scenarios for hardcoded port references that should be service references

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"

# Source utilities
# shellcheck disable=SC1091
source "$PROJECT_ROOT/scripts/helpers/utils/log.sh"

#######################################
# Configuration
#######################################

# File patterns to check
FILE_PATTERNS=(
    "*.json"
    "*.yaml" 
    "*.yml"
    "*.ts"
    "*.js"
    "*.sh"
)

# Ports that should be abstracted (from port-registry.sh)
PORTS_TO_ABSTRACT=(
    "11434"  # ollama
    "5678"   # n8n
    "5681"   # windmill
    "9200"   # searxng
    "6333"   # qdrant
    "9000"   # minio
    "8200"   # vault
    "5433"   # postgres  
    "6380"   # redis
    "4110"   # browserless
    "8090"   # whisper
    "8188"   # comfyui
    "11450"  # unstructured-io
    "9009"   # questdb
    "4113"   # agent-s2
    "2358"   # judge0
)

# Show help
show_help() {
    cat << 'EOF'
Port Abstraction Validation Tool

Validates that scenarios use service references instead of hardcoded ports.

USAGE:
    validate-port-abstraction.sh [path] [options]

ARGUMENTS:
    path    Path to validate (scenario directory or project root)
            Default: current directory

OPTIONS:
    --verbose       Show detailed output for each violation
    --summary-only  Show only summary statistics
    --fix-suggestions  Show specific fix suggestions for violations
    --help          Show this help message

EXIT CODES:
    0    No hardcoded ports found
    1    Hardcoded ports found that need migration
    2    Script error

EXAMPLES:
    # Validate single scenario
    validate-port-abstraction.sh scripts/scenarios/core/research-assistant

    # Validate all scenarios with suggestions
    validate-port-abstraction.sh scripts/scenarios/core/ --fix-suggestions

    # Quick summary of project status
    validate-port-abstraction.sh --summary-only

EOF
}

#######################################
# Core validation functions
#######################################

# Check if file should be validated
should_validate_file() {
    local file="$1"
    local filename
    filename="$(basename "$file")"
    
    # Skip if file doesn't exist or isn't regular file
    [[ ! -f "$file" ]] && return 1
    
    # Skip backup and temp files
    [[ "$filename" =~ \.(backup|bak|orig|tmp)$ ]] && return 1
    
    # Skip git and system files
    [[ "$filename" =~ ^\.git ]] && return 1
    
    # Check file patterns
    for pattern in "${FILE_PATTERNS[@]}"; do
        if [[ "$filename" == $pattern ]]; then
            return 0
        fi
    done
    
    return 1
}

# Find hardcoded ports in a file
find_hardcoded_ports() {
    local file="$1"
    local verbose="${2:-false}"
    local violations=0
    
    # Check for localhost:PORT patterns for ports that should be abstracted
    for port in "${PORTS_TO_ABSTRACT[@]}"; do
        local matches
        mapfile -t matches < <(grep -n "localhost:$port" "$file" 2>/dev/null || true)
        
        for match in "${matches[@]}"; do
            if [[ -n "$match" ]]; then
                ((violations++))
                
                if [[ "$verbose" == "true" ]]; then
                    echo "    Line ${match%%:*}: ${match#*:}"
                    echo "      → Should use: \${service.RESOURCE.url}"
                fi
            fi
        done
    done
    
    echo "$violations"
}

# Get service name suggestion for a port
get_service_suggestion() {
    local port="$1"
    
    case "$port" in
        "11434") echo "ollama" ;;
        "5678")  echo "n8n" ;;
        "5681")  echo "windmill" ;;
        "9200")  echo "searxng" ;;
        "6333")  echo "qdrant" ;;
        "9000")  echo "minio" ;;
        "8200")  echo "vault" ;;
        "5433")  echo "postgres" ;;
        "6380")  echo "redis" ;;
        "4110")  echo "browserless" ;;
        "8090")  echo "whisper" ;;
        "8188")  echo "comfyui" ;;
        "11450") echo "unstructured-io" ;;
        "9009")  echo "questdb" ;;
        "4113")  echo "agent-s2" ;;
        "2358")  echo "judge0" ;;
        *)       echo "unknown" ;;
    esac
}

# Validate a single file
validate_file() {
    local file="$1"
    local verbose="${2:-false}"
    local show_suggestions="${3:-false}"
    
    local violation_count
    violation_count=$(find_hardcoded_ports "$file" "$verbose")
    
    if [[ "$violation_count" -gt 0 ]]; then
        local relative_path
        relative_path="$(realpath --relative-to="$PWD" "$file")"
        echo "❌ $relative_path: $violation_count hardcoded port(s)"
        
        if [[ "$show_suggestions" == "true" ]]; then
            echo "   Fix suggestions:"
            for port in "${PORTS_TO_ABSTRACT[@]}"; do
                if grep -q "localhost:$port" "$file" 2>/dev/null; then
                    local service
                    service=$(get_service_suggestion "$port")
                    echo "     localhost:$port → \${service.$service.url}"
                fi
            done
        fi
        
        return 1
    fi
    
    return 0
}

# Validate a directory or file
validate_path() {
    local path="$1"
    local verbose="${2:-false}"
    local show_suggestions="${3:-false}"
    local summary_only="${4:-false}"
    
    local total_files=0
    local files_with_violations=0
    local total_violations=0
    
    if [[ -f "$path" ]]; then
        # Single file
        if should_validate_file "$path"; then
            ((total_files++))
            if ! validate_file "$path" "$verbose" "$show_suggestions"; then
                ((files_with_violations++))
                local file_violations
                file_violations=$(find_hardcoded_ports "$path")
                ((total_violations += file_violations))
            fi
        fi
    elif [[ -d "$path" ]]; then
        # Directory
        [[ "$summary_only" != "true" ]] && log::info "Validating: $path"
        
        while IFS= read -r file; do
            if should_validate_file "$file"; then
                ((total_files++))
                
                local file_violations
                file_violations=$(find_hardcoded_ports "$file")
                
                if [[ "$file_violations" -gt 0 ]]; then
                    ((files_with_violations++))
                    ((total_violations += file_violations))
                    
                    if [[ "$summary_only" != "true" ]]; then
                        validate_file "$file" "$verbose" "$show_suggestions"
                    fi
                fi
            fi
        done < <(find "$path" -type f)
    else
        log::error "Path not found: $path"
        return 2
    fi
    
    # Summary
    echo ""
    log::info "Validation Summary:"
    log::info "  Files checked: $total_files"
    log::info "  Files with violations: $files_with_violations"
    log::info "  Total hardcoded port references: $total_violations"
    
    if [[ "$total_violations" -eq 0 ]]; then
        log::success "✅ No hardcoded ports found - all scenarios use service references!"
        return 0
    else
        log::warn "⚠️  Found $total_violations hardcoded port references in $files_with_violations files"
        log::info "💡 Run migration tool: scripts/scenarios/tools/migrate-ports-simple.sh"
        return 1
    fi
}

#######################################
# Main script
#######################################

main() {
    local path="$(pwd)"
    local verbose=false
    local show_suggestions=false
    local summary_only=false
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --verbose)
                verbose=true
                shift
                ;;
            --fix-suggestions)
                show_suggestions=true
                shift
                ;;
            --summary-only)
                summary_only=true
                shift
                ;;
            --help|-h)
                show_help
                exit 0
                ;;
            -*)
                log::error "Unknown option: $1"
                exit 2
                ;;
            *)
                path="$1"
                shift
                ;;
        esac
    done
    
    # Validate path exists
    if [[ ! -e "$path" ]]; then
        log::error "Path does not exist: $path"
        exit 2
    fi
    
    # Run validation
    validate_path "$path" "$verbose" "$show_suggestions" "$summary_only"
}

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi