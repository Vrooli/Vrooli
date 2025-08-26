#!/usr/bin/env bash
# Detect Deprecated Patterns
# Scans resources for deprecated patterns and suggests v2.0 alternatives
# Usage: ./detect-deprecated.sh [--resource <name>] [--fix-suggestions] [--format <text|json>]

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/log.sh" 2>/dev/null || {
    # Fallback if log.sh not available
    log::info() { echo "[INFO] $*"; }
    log::success() { echo "[✓] $*"; }
    log::error() { echo "[✗] $*"; }
    log::warning() { echo "[⚠] $*"; }
}

# Configuration
RESOURCES_DIR="${APP_ROOT}/resources"
SPECIFIC_RESOURCE=""
SHOW_SUGGESTIONS=false
OUTPUT_FORMAT="text"
VERBOSE=false
DETECTED_ISSUES=()

# Pattern definitions for deprecated features
declare -A DEPRECATED_PATTERNS=(
    # Files
    ["manage.sh"]="Legacy management script → migrate to cli.sh"
    ["manage.bats"]="Legacy test file → migrate to lib/test.sh"
    ["inject.sh"]="Legacy injection script → use 'content add' command"
    
    # Commands
    ["inject_command"]="inject command → 'content add'"
    ["test-api_command"]="test-api command → 'test integration'"
    
    # Functions
    ["main_function"]="main() function → CLI framework dispatch"
    
    # Environment variables
    ["vrooli_resource_env"]="VROOLI_RESOURCE_* → resource-specific prefix"
    
    # Command patterns
    ["action_flag"]="--action flag → direct command"
    ["flat_commands"]="Flat command structure → grouped commands"
)

# Parse arguments
while [[ $# -gt 0 ]]; do
    case "$1" in
        --resource|-r)
            SPECIFIC_RESOURCE="$2"
            shift 2
            ;;
        --fix-suggestions|--suggestions)
            SHOW_SUGGESTIONS=true
            shift
            ;;
        --format|-f)
            OUTPUT_FORMAT="$2"
            shift 2
            ;;
        --verbose|-v)
            VERBOSE=true
            shift
            ;;
        --help|-h)
            cat << EOF
Usage: $0 [OPTIONS]

Detect deprecated patterns in resources and suggest v2.0 alternatives

Options:
  --resource, -r <name>     Scan specific resource (default: all)
  --fix-suggestions         Show fix suggestions for each issue
  --format, -f <format>     Output format: text, json (default: text)
  --verbose, -v             Show detailed output
  --help, -h                Show this help message

Examples:
  # Scan all resources
  $0

  # Scan specific resource with suggestions
  $0 --resource ollama --fix-suggestions

  # Generate JSON report
  $0 --format json > deprecated-patterns.json

EOF
            exit 0
            ;;
        *)
            log::error "Unknown option: $1"
            exit 1
            ;;
    esac
done

#######################################
# Detection Functions
#######################################

# Add issue to detected issues array
add_issue() {
    local resource="$1"
    local file="$2"
    local line="$3"
    local pattern="$4"
    local description="$5"
    local suggestion="$6"
    
    local issue="{\"resource\":\"$resource\",\"file\":\"$file\",\"line\":$line,\"pattern\":\"$pattern\",\"description\":\"$description\",\"suggestion\":\"$suggestion\"}"
    DETECTED_ISSUES+=("$issue")
}

# Check for deprecated files
check_deprecated_files() {
    local resource="$1"
    local resource_dir="${RESOURCES_DIR}/${resource}"
    
    # Check for manage.sh
    if [[ -f "$resource_dir/manage.sh" ]]; then
        add_issue "$resource" "manage.sh" "1" "deprecated_file" \
                  "Legacy management script" \
                  "Migrate functionality to cli.sh using CLI framework"
    fi
    
    # Check for manage.bats
    if [[ -f "$resource_dir/manage.bats" ]]; then
        add_issue "$resource" "manage.bats" "1" "deprecated_test" \
                  "Legacy BATS test file" \
                  "Migrate tests to lib/test.sh"
    fi
    
    # Check for inject.sh
    if [[ -f "$resource_dir/inject.sh" ]]; then
        add_issue "$resource" "inject.sh" "1" "deprecated_inject" \
                  "Legacy injection script" \
                  "Replace with 'content add' command in cli.sh"
    fi
    
    # Check for other deprecated patterns in file structure
    if [[ -d "$resource_dir/scripts" ]] && [[ ! -d "$resource_dir/lib" ]]; then
        add_issue "$resource" "scripts/" "1" "old_structure" \
                  "Old directory structure (scripts/ instead of lib/)" \
                  "Migrate scripts/ to lib/ for v2.0 compliance"
    fi
}

# Check for deprecated command patterns
check_deprecated_commands() {
    local resource="$1"
    local cli_file="${RESOURCES_DIR}/${resource}/cli.sh"
    
    if [[ ! -f "$cli_file" ]]; then
        return 0
    fi
    
    local line_num=0
    while IFS= read -r line; do
        ((line_num++))
        
        # Check for deprecated inject command
        if echo "$line" | grep -q 'cli::register_command.*"inject"'; then
            add_issue "$resource" "cli.sh" "$line_num" "deprecated_inject_cmd" \
                      "Deprecated 'inject' command" \
                      "Replace with 'content add' command"
        fi
        
        # Check for test-api command
        if echo "$line" | grep -q 'cli::register_command.*"test-api"'; then
            add_issue "$resource" "cli.sh" "$line_num" "deprecated_test_api" \
                      "Deprecated 'test-api' command" \
                      "Replace with 'test integration' command"
        fi
        
        # Check for --action flag pattern
        if echo "$line" | grep -q '\--action'; then
            add_issue "$resource" "cli.sh" "$line_num" "deprecated_action_flag" \
                      "Deprecated --action flag pattern" \
                      "Use direct commands instead of --action flag"
        fi
        
        # Check for main() function
        if echo "$line" | grep -q '^main()'; then
            add_issue "$resource" "cli.sh" "$line_num" "deprecated_main_func" \
                      "Deprecated main() function" \
                      "Use CLI framework dispatch instead"
        fi
        
        # Check if using only flat commands (no command groups)
        if echo "$line" | grep -q 'cli::register_command.*"start"\|"stop"\|"install"' && \
           ! grep -q 'cli::register_command_group\|cli::register_subcommand' "$cli_file"; then
            # Only report this once per file
            if ! echo "${DETECTED_ISSUES[*]}" | grep -q "flat_commands.*$resource"; then
                add_issue "$resource" "cli.sh" "$line_num" "flat_commands" \
                          "Using flat command structure (v1.0)" \
                          "Migrate to grouped commands: manage/test/content"
            fi
        fi
        
    done < "$cli_file"
}

# Check for deprecated environment variables
check_deprecated_env_vars() {
    local resource="$1"
    local resource_dir="${RESOURCES_DIR}/${resource}"
    
    # Check all shell files for deprecated environment variables
    while IFS= read -r -d '' file; do
        local relative_file="${file#$resource_dir/}"
        local line_num=0
        
        while IFS= read -r line; do
            ((line_num++))
            
            # Check for VROOLI_RESOURCE_ pattern
            if echo "$line" | grep -q 'VROOLI_RESOURCE_'; then
                add_issue "$resource" "$relative_file" "$line_num" "deprecated_env_var" \
                          "Deprecated VROOLI_RESOURCE_* environment variable" \
                          "Use resource-specific prefix (e.g., OLLAMA_*, POSTGRES_*)"
            fi
            
        done < "$file"
        
    done < <(find "$resource_dir" -name "*.sh" -type f -print0 2>/dev/null || true)
}

# Check for v1.0 contract patterns
check_v1_contract_patterns() {
    local resource="$1"
    local resource_dir="${RESOURCES_DIR}/${resource}"
    
    # Check if resource has any reference to old contract files
    if grep -r "v1.0\|category-based" "$resource_dir" >/dev/null 2>&1; then
        add_issue "$resource" "various" "0" "v1_contract_ref" \
                  "References to v1.0 contracts" \
                  "Update to reference v2.0 Universal Contract"
    fi
    
    # Check for category-based organization
    local has_ai_features=false
    local has_storage_features=false
    
    if [[ -f "$resource_dir/lib/ai.sh" ]] || [[ -f "$resource_dir/lib/llm.sh" ]]; then
        has_ai_features=true
    fi
    
    if [[ -f "$resource_dir/lib/storage.sh" ]] || [[ -f "$resource_dir/lib/database.sh" ]]; then
        has_storage_features=true
    fi
    
    if [[ "$has_ai_features" == "true" ]] || [[ "$has_storage_features" == "true" ]]; then
        add_issue "$resource" "lib/" "0" "category_organization" \
                  "Category-based file organization (ai.sh, storage.sh)" \
                  "v2.0 uses universal structure with core.sh, test.sh"
    fi
}

# Scan a single resource
scan_resource() {
    local resource="$1"
    
    [[ "$VERBOSE" == "true" ]] && log::info "Scanning: $resource"
    
    # Check if resource directory exists
    if [[ ! -d "${RESOURCES_DIR}/${resource}" ]]; then
        log::error "Resource directory not found: $resource"
        return 1
    fi
    
    # Run all checks
    check_deprecated_files "$resource"
    check_deprecated_commands "$resource"
    check_deprecated_env_vars "$resource"
    check_v1_contract_patterns "$resource"
}

# Get severity level for a pattern
get_severity() {
    local pattern="$1"
    
    case "$pattern" in
        deprecated_file|deprecated_inject|deprecated_main_func)
            echo "HIGH"
            ;;
        flat_commands|deprecated_test_api|deprecated_inject_cmd)
            echo "MEDIUM"
            ;;
        deprecated_env_var|deprecated_action_flag)
            echo "LOW"
            ;;
        *)
            echo "INFO"
            ;;
    esac
}

# Generate fix suggestion
generate_fix_suggestion() {
    local resource="$1"
    local pattern="$2"
    local file="$3"
    
    case "$pattern" in
        deprecated_file)
            echo "1. Review functionality in $file"
            echo "2. Migrate functions to appropriate lib/ files"
            echo "3. Update cli.sh to use CLI framework"
            echo "4. Remove $file after migration"
            ;;
        flat_commands)
            echo "1. Add command groups: cli::register_command_group 'manage' '...'"
            echo "2. Convert flat commands to subcommands:"
            echo "   'start' → 'manage start'"
            echo "   'install' → 'manage install'"
            echo "3. Update help text and examples"
            ;;
        deprecated_inject_cmd)
            echo "1. Remove 'inject' command registration"
            echo "2. Implement 'content add' command"
            echo "3. Update documentation and examples"
            ;;
        deprecated_main_func)
            echo "1. Remove main() function"
            echo "2. Use cli::dispatch at end of script"
            echo "3. Register commands with cli::register_command"
            ;;
        *)
            echo "See v2.0 Universal Contract documentation for guidance"
            ;;
    esac
}

# Output text report
output_text_report() {
    local total_issues=${#DETECTED_ISSUES[@]}
    
    echo
    echo "========================================"
    echo "Deprecated Pattern Detection Report"
    echo "========================================"
    echo "Scan Date: $(date)"
    echo "Total Issues Found: $total_issues"
    echo
    
    if [[ $total_issues -eq 0 ]]; then
        log::success "No deprecated patterns found!"
        return 0
    fi
    
    # Group issues by resource
    declare -A resource_issues=()
    for issue in "${DETECTED_ISSUES[@]}"; do
        local resource=$(echo "$issue" | grep -o '"resource":"[^"]*"' | cut -d'"' -f4)
        resource_issues["$resource"]=1
    done
    
    # Report by resource
    for resource in $(printf '%s\n' "${!resource_issues[@]}" | sort); do
        echo "Resource: $resource"
        echo "----------------------------------------"
        
        for issue in "${DETECTED_ISSUES[@]}"; do
            local issue_resource=$(echo "$issue" | grep -o '"resource":"[^"]*"' | cut -d'"' -f4)
            if [[ "$issue_resource" == "$resource" ]]; then
                local file=$(echo "$issue" | grep -o '"file":"[^"]*"' | cut -d'"' -f4)
                local line=$(echo "$issue" | grep -o '"line":[0-9]*' | cut -d: -f2)
                local pattern=$(echo "$issue" | grep -o '"pattern":"[^"]*"' | cut -d'"' -f4)
                local description=$(echo "$issue" | grep -o '"description":"[^"]*"' | cut -d'"' -f4)
                local severity=$(get_severity "$pattern")
                
                printf "  [%s] %s:%s - %s\n" "$severity" "$file" "$line" "$description"
                
                if [[ "$SHOW_SUGGESTIONS" == "true" ]]; then
                    echo "    Fix:"
                    generate_fix_suggestion "$resource" "$pattern" "$file" | sed 's/^/      /'
                    echo
                fi
            fi
        done
        echo
    done
    
    # Summary by severity
    declare -A severity_counts=()
    for issue in "${DETECTED_ISSUES[@]}"; do
        local pattern=$(echo "$issue" | grep -o '"pattern":"[^"]*"' | cut -d'"' -f4)
        local severity=$(get_severity "$pattern")
        severity_counts["$severity"]=$((${severity_counts["$severity"]:-0} + 1))
    done
    
    echo "Summary by Severity:"
    for severity in HIGH MEDIUM LOW INFO; do
        if [[ -n "${severity_counts[$severity]:-}" ]]; then
            printf "  %-8s %d issues\n" "$severity:" "${severity_counts[$severity]}"
        fi
    done
}

# Output JSON report
output_json_report() {
    echo "{"
    echo "  \"timestamp\": \"$(date -Iseconds)\","
    echo "  \"total_issues\": ${#DETECTED_ISSUES[@]},"
    echo "  \"issues\": ["
    
    local first=true
    for issue in "${DETECTED_ISSUES[@]}"; do
        if [[ "$first" == "true" ]]; then
            first=false
        else
            echo ","
        fi
        
        # Add severity to issue
        local pattern=$(echo "$issue" | grep -o '"pattern":"[^"]*"' | cut -d'"' -f4)
        local severity=$(get_severity "$pattern")
        local enhanced_issue="${issue%\}},\"severity\":\"$severity\"}"
        
        echo -n "    $enhanced_issue"
    done
    
    echo ""
    echo "  ]"
    echo "}"
}

#######################################
# Main
#######################################

main() {
    log::info "Deprecated Pattern Detection Tool"
    echo "Scanning for v1.0 patterns that need v2.0 migration..."
    
    # Determine which resources to scan
    local resources=()
    if [[ -n "$SPECIFIC_RESOURCE" ]]; then
        resources=("$SPECIFIC_RESOURCE")
    else
        # Find all resources
        for dir in "${RESOURCES_DIR}"/*; do
            if [[ -d "$dir" ]]; then
                resources+=("$(basename "$dir")")
            fi
        done
    fi
    
    # Scan each resource
    for resource in "${resources[@]}"; do
        scan_resource "$resource" || true
    done
    
    # Output results
    if [[ "$OUTPUT_FORMAT" == "json" ]]; then
        output_json_report
    else
        output_text_report
    fi
}

# Make executable if called directly
chmod +x "${BASH_SOURCE[0]}" 2>/dev/null || true

# Run main
main