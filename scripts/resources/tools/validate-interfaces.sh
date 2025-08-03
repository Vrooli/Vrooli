#!/bin/bash
# ====================================================================
# Resource Interface Validation Script
# ====================================================================
#
# Validates that all resource manage.sh scripts implement the standard
# interface required by the Vrooli resource ecosystem. This can be used
# as a standalone validation tool or integrated into CI/CD pipelines.
#
# Usage:
#   ./validate-all-interfaces.sh [OPTIONS]
#
# Options:
#   --help              Show help message
#   --verbose           Enable verbose output
#   --resources-dir     Specify resources directory (default: auto-detect)
#   --resource          Test specific resource only
#   --fix               Attempt to fix common issues (future feature)
#   --report            Generate detailed report file
#   --format            Output format: text|json|csv (default: text)
#
# Exit Codes:
#   0 - All resources pass interface compliance
#   1 - One or more resources fail compliance
#   2 - Script error or missing dependencies
#
# Examples:
#   ./validate-all-interfaces.sh                    # Validate all resources
#   ./validate-all-interfaces.sh --resource ollama  # Validate specific resource
#   ./validate-all-interfaces.sh --verbose          # Detailed output
#   ./validate-all-interfaces.sh --format json      # JSON output
#
# ====================================================================

set -euo pipefail

# Script directory and paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCES_DIR="$SCRIPT_DIR"
TESTS_DIR="$SCRIPT_DIR/tests"

# Configuration
VERBOSE=false
SPECIFIC_RESOURCE=""
FIX_ISSUES=false
GENERATE_REPORT=false
OUTPUT_FORMAT="text"
REPORT_FILE=""

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
    cat << EOF
Resource Interface Validation Script

USAGE:
    $0 [OPTIONS]

DESCRIPTION:
    Validates that all resource manage.sh scripts implement the standard
    interface required by the Vrooli resource ecosystem.

OPTIONS:
    --help              Show this help message
    --verbose           Enable verbose output with detailed test results
    --resources-dir     Specify resources directory (default: $RESOURCES_DIR)
    --resource <name>   Test specific resource only
    --fix               Attempt to fix common issues (future feature)
    --report            Generate detailed validation report
    --format <fmt>      Output format: text|json|csv (default: text)

EXIT CODES:
    0 - All resources pass interface compliance
    1 - One or more resources fail compliance  
    2 - Script error or missing dependencies

EXAMPLES:
    $0                                    # Validate all resources
    $0 --resource ollama                  # Validate specific resource
    $0 --verbose                          # Detailed output
    $0 --format json                      # JSON output
    $0 --report --format csv              # Generate CSV report

VALIDATION CHECKS:
    • Required actions: install, start, stop, status, logs
    • Help/usage patterns: --help, -h, --version
    • Error handling: invalid actions, missing arguments
    • Configuration loading and validation
    • Argument patterns: --action, --yes flags

For more information, see: scripts/resources/tests/framework/interface-compliance.sh
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
            --resources-dir)
                RESOURCES_DIR="$2"
                if [[ ! -d "$RESOURCES_DIR" ]]; then
                    log_error "Resources directory not found: $RESOURCES_DIR"
                    exit 2
                fi
                shift 2
                ;;
            --resource)
                SPECIFIC_RESOURCE="$2"
                shift 2
                ;;
            --fix)
                FIX_ISSUES=true
                log_warning "Fix mode not yet implemented"
                shift
                ;;
            --report)
                GENERATE_REPORT=true
                shift
                ;;
            --format)
                OUTPUT_FORMAT="$2"
                if [[ "$OUTPUT_FORMAT" != "text" && "$OUTPUT_FORMAT" != "json" && "$OUTPUT_FORMAT" != "csv" ]]; then
                    log_error "Invalid format: $OUTPUT_FORMAT. Must be text, json, or csv"
                    exit 2
                fi
                shift 2
                ;;
            *)
                log_error "Unknown option: $1"
                show_help
                exit 2
                ;;
        esac
    done
}

# Validate prerequisites
validate_prerequisites() {
    log_verbose "Validating prerequisites..."
    
    # Check if interface compliance framework exists
    local framework_path="$TESTS_DIR/framework/interface-compliance.sh"
    if [[ ! -f "$framework_path" ]]; then
        log_error "Interface compliance framework not found: $framework_path"
        log_error "Please ensure you're running from the correct directory"
        exit 2
    fi
    
    # Check required tools
    local missing_tools=()
    for tool in bash timeout; do
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

# Find all manage.sh scripts
find_all_manage_scripts() {
    if [[ -n "$SPECIFIC_RESOURCE" ]]; then
        log_verbose "Looking for manage.sh script for specific resource: $SPECIFIC_RESOURCE"
        
        # Try to find the specific resource
        local found_script
        log_verbose "Searching for: find \"$RESOURCES_DIR\" -name \"manage.sh\" -path \"*/$SPECIFIC_RESOURCE/*\" -type f"
        found_script=$(find "$RESOURCES_DIR" -name "manage.sh" -path "*/$SPECIFIC_RESOURCE/*" -type f 2>/dev/null | head -1)
        log_verbose "Find result: '$found_script'"
        
        if [[ -n "$found_script" && -f "$found_script" ]]; then
            log_verbose "Found script for $SPECIFIC_RESOURCE: $found_script"
            printf '%s\0' "$found_script"
        else
            log_error "No manage.sh script found for resource: $SPECIFIC_RESOURCE"
            log_verbose "Search was: find \"$RESOURCES_DIR\" -name \"manage.sh\" -path \"*/$SPECIFIC_RESOURCE/*\" -type f"
            log_verbose "Directory listing around $SPECIFIC_RESOURCE:"
            find "$RESOURCES_DIR" -name "*$SPECIFIC_RESOURCE*" -type d 2>/dev/null | head -5 | while read dir; do
                log_verbose "  Found directory: $dir"
                if [[ -f "$dir/manage.sh" ]]; then
                    log_verbose "    -> Contains manage.sh: $dir/manage.sh"
                fi
            done
            exit 1
        fi
    else
        log_verbose "Searching for all manage.sh scripts in: $RESOURCES_DIR"
        
        # Find all manage.sh scripts, excluding tests directory
        find "$RESOURCES_DIR" -name "manage.sh" -type f -not -path "*/tests/*" -print0 2>/dev/null
    fi
}

# Extract resource name from script path
get_resource_name() {
    local script_path="$1"
    local resource_dir
    resource_dir=$(dirname "$script_path")
    basename "$resource_dir"
}

# Generate JSON report entry
generate_json_entry() {
    local resource="$1"
    local status="$2"
    local details="$3"
    
    cat << EOF
    {
        "resource": "$resource",
        "status": "$status",
        "details": "$details",
        "timestamp": "$(date -Iseconds)"
    }
EOF
}

# Generate CSV report entry
generate_csv_entry() {
    local resource="$1"
    local status="$2"
    local details="$3"
    local timestamp="$(date -Iseconds)"
    
    echo "\"$resource\",\"$status\",\"$details\",\"$timestamp\""
}

# Main validation function
run_validation() {
    log_info "Starting resource interface validation..."
    log_info "Resources directory: $RESOURCES_DIR"
    log_info "Output format: $OUTPUT_FORMAT"
    
    # Source the interface compliance framework
    source "$TESTS_DIR/framework/interface-compliance.sh"
    
    # Find all scripts to validate
    local scripts=()
    if [[ -n "$SPECIFIC_RESOURCE" ]]; then
        log_verbose "Looking for manage.sh script for specific resource: $SPECIFIC_RESOURCE"
        
        # Try to find the specific resource
        local found_script
        log_verbose "Searching for: find \"$RESOURCES_DIR\" -name \"manage.sh\" -path \"*/$SPECIFIC_RESOURCE/*\" -type f"
        found_script=$(find "$RESOURCES_DIR" -name "manage.sh" -path "*/$SPECIFIC_RESOURCE/*" -type f 2>/dev/null | head -1)
        log_verbose "Find result: '$found_script'"
        
        if [[ -n "$found_script" && -f "$found_script" ]]; then
            log_verbose "Found script for $SPECIFIC_RESOURCE: $found_script"
            scripts+=("$found_script")
        else
            log_error "No manage.sh script found for resource: $SPECIFIC_RESOURCE"
            log_verbose "Search was: find \"$RESOURCES_DIR\" -name \"manage.sh\" -path \"*/$SPECIFIC_RESOURCE/*\" -type f"
            log_verbose "Directory listing around $SPECIFIC_RESOURCE:"
            find "$RESOURCES_DIR" -name "*$SPECIFIC_RESOURCE*" -type d 2>/dev/null | head -5 | while read dir; do
                log_verbose "  Found directory: $dir"
                if [[ -f "$dir/manage.sh" ]]; then
                    log_verbose "    -> Contains manage.sh: $dir/manage.sh"
                fi
            done
            exit 1
        fi
    else
        log_verbose "Searching for all manage.sh scripts in: $RESOURCES_DIR"
        
        # Find all manage.sh scripts, excluding tests directory
        while IFS= read -r -d '' script_path; do
            scripts+=("$script_path")
            log_verbose "Found script: $script_path"
        done < <(find "$RESOURCES_DIR" -name "manage.sh" -type f -not -path "*/tests/*" -print0 2>/dev/null)
    fi
    
    log_info "Found ${#scripts[@]} manage.sh script(s) to validate"
    echo
    
    # Validation results
    local total_resources=${#scripts[@]}
    local passed_resources=0
    local failed_resources=0
    local failed_resource_names=()
    
    # Report data (for JSON/CSV output)
    local json_entries=()
    local csv_entries=()
    
    # Initialize CSV header if needed
    if [[ "$OUTPUT_FORMAT" == "csv" ]]; then
        csv_entries+=("Resource,Status,Details,Timestamp")
    fi
    
    # Validate each script
    for script_path in "${scripts[@]}"; do
        local resource_name
        resource_name=$(get_resource_name "$script_path")
        
        log_info "Validating $resource_name..."
        log_verbose "Script path: $script_path"
        
        # Capture validation output
        local validation_output
        local validation_result
        
        if validation_output=$(test_resource_interface_compliance "$resource_name" "$script_path" 2>&1); then
            validation_result="passed"
            passed_resources=$((passed_resources + 1))
            log_success "$resource_name passes interface compliance"
        else
            validation_result="failed"
            failed_resources=$((failed_resources + 1))
            failed_resource_names+=("$resource_name")
            log_error "$resource_name fails interface compliance"
        fi
        
        # Store report data
        if [[ "$OUTPUT_FORMAT" == "json" ]]; then
            json_entries+=("$(generate_json_entry "$resource_name" "$validation_result" "${validation_output//\"/\\\"}")")
        elif [[ "$OUTPUT_FORMAT" == "csv" ]]; then
            csv_entries+=("$(generate_csv_entry "$resource_name" "$validation_result" "${validation_output//\"/\\\"}")")
        fi
        
        # Show detailed output if verbose
        if [[ "$VERBOSE" == "true" && -n "$validation_output" ]]; then
            echo "Detailed output:"
            echo "$validation_output" | sed 's/^/  /'
        fi
        
        echo "========================================"
    done
    
    # Generate summary
    log_info "Validation Summary:"
    echo "  Total resources: $total_resources"
    echo "  Passed: $passed_resources"
    echo "  Failed: $failed_resources"
    
    # Output results in requested format
    case "$OUTPUT_FORMAT" in
        "json")
            echo
            echo "{"
            echo "  \"summary\": {"
            echo "    \"total\": $total_resources,"
            echo "    \"passed\": $passed_resources,"
            echo "    \"failed\": $failed_resources,"
            echo "    \"timestamp\": \"$(date -Iseconds)\""
            echo "  },"
            echo "  \"results\": ["
            
            local first=true
            for entry in "${json_entries[@]}"; do
                if [[ "$first" == "true" ]]; then
                    first=false
                else
                    echo ","
                fi
                echo "$entry"
            done
            
            echo
            echo "  ]"
            echo "}"
            ;;
            
        "csv")
            echo
            for entry in "${csv_entries[@]}"; do
                echo "$entry"
            done
            ;;
    esac
    
    # Generate report file if requested
    if [[ "$GENERATE_REPORT" == "true" ]]; then
        local timestamp=$(date +%Y%m%d_%H%M%S)
        REPORT_FILE="interface_validation_report_${timestamp}.${OUTPUT_FORMAT}"
        
        log_info "Generating report: $REPORT_FILE"
        
        case "$OUTPUT_FORMAT" in
            "json")
                {
                    echo "{"
                    echo "  \"summary\": {"
                    echo "    \"total\": $total_resources,"
                    echo "    \"passed\": $passed_resources,"
                    echo "    \"failed\": $failed_resources,"
                    echo "    \"timestamp\": \"$(date -Iseconds)\""
                    echo "  },"
                    echo "  \"results\": ["
                    
                    local first=true
                    for entry in "${json_entries[@]}"; do
                        if [[ "$first" == "true" ]]; then
                            first=false
                        else
                            echo ","
                        fi
                        echo "$entry"
                    done
                    
                    echo
                    echo "  ]"
                    echo "}"
                } > "$REPORT_FILE"
                ;;
                
            "csv")
                {
                    for entry in "${csv_entries[@]}"; do
                        echo "$entry"
                    done
                } > "$REPORT_FILE"
                ;;
                
            "text")
                {
                    echo "Resource Interface Validation Report"
                    echo "Generated: $(date)"
                    echo "========================================"
                    echo
                    echo "Summary:"
                    echo "  Total resources: $total_resources"
                    echo "  Passed: $passed_resources"
                    echo "  Failed: $failed_resources"
                    echo
                    
                    if [[ $failed_resources -gt 0 ]]; then
                        echo "Failed resources:"
                        for resource in "${failed_resource_names[@]}"; do
                            echo "  • $resource"
                        done
                        echo
                    fi
                } > "$REPORT_FILE"
                ;;
        esac
        
        log_success "Report generated: $REPORT_FILE"
    fi
    
    # Final result
    if [[ $failed_resources -gt 0 ]]; then
        echo
        log_error "Interface validation failed for $failed_resources resource(s):"
        for resource in "${failed_resource_names[@]}"; do
            echo "  • $resource"
        done
        echo
        log_info "💡 To fix issues, review the interface compliance requirements in:"
        log_info "   scripts/resources/tests/framework/interface-compliance.sh"
        return 1
    else
        echo
        log_success "All resources pass interface compliance validation! 🎉"
        return 0
    fi
}

# Main execution
main() {
    parse_args "$@"
    validate_prerequisites
    run_validation
}

# Run main function
main "$@"