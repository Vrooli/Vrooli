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
#   ./validate-interfaces.sh [OPTIONS]
#
# Options:
#   --help              Show help message
#   --verbose           Enable verbose output
#   --resources-dir     Specify resources directory (default: auto-detect)
#   --resource          Test specific resource only
#   --level             Validation level: quick|standard|full (default: quick)
#   --fix               Attempt to fix common issues (future feature)
#   --report            Generate detailed report file
#   --format            Output format: text|json|csv|junit (default: text)
#   --cache             Use cached results for performance optimization
#   --parallel          Run validations in parallel
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
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/scripts/resources/tools"
RESOURCES_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
TESTS_DIR="$RESOURCES_DIR/tests"

# Source required utilities
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh"

# Configuration
VERBOSE=false
SPECIFIC_RESOURCE=""
VALIDATION_LEVEL="quick"
FIX_ISSUES=false
GENERATE_REPORT=false
OUTPUT_FORMAT="text"
REPORT_FILE=""
USE_CACHE=false
USE_PARALLEL=false

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
    interface required by the Vrooli resource ecosystem using the Layer 1
    syntax validation system with contract-first validation.

OPTIONS:
    --help              Show this help message
    --verbose           Enable verbose output with detailed test results
    --resources-dir     Specify resources directory (default: $RESOURCES_DIR)
    --resource <name>   Test specific resource only
    --level <level>     Validation level: quick|standard|full (default: quick)
    --fix               Attempt to fix common issues (future feature)
    --report            Generate detailed validation report
    --format <fmt>      Output format: text|json|csv|junit (default: text)
    --cache             Use cached results for performance optimization
    --parallel          Run validations in parallel (future feature)

VALIDATION LEVELS:
    quick    - Layer 1: Syntax validation only (<1s per resource)
    standard - Layer 1+2: Syntax + behavioral testing (future, ~30s)
    full     - Layer 1+2+3: All validation layers (future, ~5min)

EXIT CODES:
    0 - All resources pass interface compliance
    1 - One or more resources fail compliance  
    2 - Script error or missing dependencies

EXAMPLES:
    $0                                    # Quick validation (Layer 1)
    $0 --resource ollama                  # Validate specific resource
    $0 --level standard                   # Standard validation (future)
    $0 --verbose --format json            # Detailed JSON output
    $0 --report --format csv              # Generate CSV report

LAYER 1 VALIDATION CHECKS:
    • Contract compliance: Resources must implement actions defined in contracts
    • Required core actions: install, start, stop, status, logs
    • Category-specific actions: Based on resource category (ai, automation, etc.)
    • Help patterns: --help, -h, --version support
    • Error handling: Proper bash error handling patterns
    • File structure: Required config/ and lib/ directories
    • Argument patterns: Consistent --action and flag usage

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
                shift
                ;;
            --report)
                GENERATE_REPORT=true
                shift
                ;;
            --format)
                OUTPUT_FORMAT="$2"
                if [[ "$OUTPUT_FORMAT" != "text" && "$OUTPUT_FORMAT" != "json" && "$OUTPUT_FORMAT" != "csv" && "$OUTPUT_FORMAT" != "junit" ]]; then
                    log_error "Invalid format: $OUTPUT_FORMAT. Must be text, json, csv, or junit"
                    exit 2
                fi
                shift 2
                ;;
            --level)
                VALIDATION_LEVEL="$2"
                if [[ "$VALIDATION_LEVEL" != "quick" && "$VALIDATION_LEVEL" != "standard" && "$VALIDATION_LEVEL" != "full" ]]; then
                    log_error "Invalid validation level: $VALIDATION_LEVEL. Must be quick, standard, or full"
                    exit 2
                fi
                shift 2
                ;;
            --cache)
                USE_CACHE=true
                shift
                ;;
            --parallel)
                USE_PARALLEL=true
                shift
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
    
    # Check if Layer 1 validation framework exists
    local syntax_validator_path="$TESTS_DIR/framework/validators/syntax.sh"
    if [[ ! -f "$syntax_validator_path" ]]; then
        log_error "Layer 1 syntax validator not found: $syntax_validator_path"
        log_error "Please ensure you're running from the correct directory"
        exit 2
    fi
    
    # Check if contract parser exists
    local contract_parser_path="$TESTS_DIR/framework/parsers/contract-parser.sh"
    if [[ ! -f "$contract_parser_path" ]]; then
        log_error "Contract parser not found: $contract_parser_path"
        log_error "Please ensure the Layer 1 validation system is properly installed"
        exit 2
    fi
    
    # Check if contracts directory exists
    local contracts_dir="$RESOURCES_DIR/contracts"
    if [[ ! -d "$contracts_dir" ]]; then
        log_error "Contracts directory not found: $contracts_dir"
        log_error "Please ensure the Layer 1 validation system is properly installed"
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
    log_info "Validation level: $VALIDATION_LEVEL"
    log_info "Output format: $OUTPUT_FORMAT"
    
    # Source the Layer 1 validation framework
    source "$TESTS_DIR/framework/validators/syntax.sh"
    
    # Source reporters based on output format
    case "$OUTPUT_FORMAT" in
        "text")
            source "$TESTS_DIR/framework/reporters/text-reporter.sh"
            text_reporter_init
            ;;
        "junit")
            source "$TESTS_DIR/framework/reporters/junit-reporter.sh"
            junit_reporter_init "Vrooli.ResourceValidation.Layer1"
            ;;
    esac
    
    # Initialize the syntax validator
    if ! syntax_validator_init "$RESOURCES_DIR/contracts"; then
        log_error "Failed to initialize Layer 1 validation system"
        exit 2
    fi
    
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
    
    # Validate each script (with optional parallel execution)
    if [[ "$USE_PARALLEL" == "true" ]]; then
        log_info "Running validations in parallel..."
        
        # Parallel processing with background jobs
        local job_pids=()
        local temp_results_dir="/tmp/vrooli_validation_$$"
        mkdir -p "$temp_results_dir"
        
        # Launch validation jobs in parallel
        for script_path in "${scripts[@]}"; do
            local resource_name
            resource_name=$(get_resource_name "$script_path")
            
            # Launch validation in background
            (
                # Detect resource category
                local resource_category
                resource_category=$(detect_resource_category "$(dirname "$script_path")")
                
                # Capture validation output
                local validation_output
                local validation_result
                local result_file="$temp_results_dir/${resource_name}.result"
                
                # Choose validation level
                case "$VALIDATION_LEVEL" in
                    "quick")
                        # Layer 1 only: Syntax validation
                        if validation_output=$(validate_resource_syntax "$resource_name" "$resource_category" "$script_path" "$USE_CACHE" 2>&1); then
                            validation_result="passed"
                            echo "passed" > "$result_file"
                            echo "Layer 1 syntax validation passed" >> "$result_file"
                        else
                            validation_result="failed"
                            echo "failed" > "$result_file"
                            echo "Layer 1 syntax validation failed" >> "$result_file"
                        fi
                        ;;
                    "standard"|"full")
                        # Future: Layer 1 + 2 or all layers
                        if validation_output=$(validate_resource_syntax "$resource_name" "$resource_category" "$script_path" "$USE_CACHE" 2>&1); then
                            validation_result="passed"
                            echo "passed" > "$result_file"
                            echo "Layer 1 syntax validation passed (higher layers not implemented)" >> "$result_file"
                        else
                            validation_result="failed"
                            echo "failed" > "$result_file"
                            echo "Layer 1 syntax validation failed" >> "$result_file"
                        fi
                        ;;
                esac
                
                # Store additional result data
                echo "$validation_output" >> "$result_file"
            ) &
            
            job_pids+=($!)
        done
        
        # Wait for all jobs to complete
        log_info "Waiting for ${#job_pids[@]} parallel validation jobs to complete..."
        for pid in "${job_pids[@]}"; do
            wait "$pid"
        done
        
        # Collect results from parallel jobs
        for script_path in "${scripts[@]}"; do
            local resource_name
            resource_name=$(get_resource_name "$script_path")
            local result_file="$temp_results_dir/${resource_name}.result"
            
            if [[ -f "$result_file" ]]; then
                local result_status
                result_status=$(head -1 "$result_file")
                local validation_output
                validation_output=$(tail -n +3 "$result_file")
                
                if [[ "$result_status" == "passed" ]]; then
                    passed_resources=$((passed_resources + 1))
                    log_success "$resource_name passes validation"
                    validation_result="passed"
                else
                    failed_resources=$((failed_resources + 1))
                    failed_resource_names+=("$resource_name")
                    log_error "$resource_name fails validation"
                    validation_result="failed"
                fi
                
                # Store report data
                if [[ "$OUTPUT_FORMAT" == "json" ]]; then
                    json_entries+=($(generate_json_entry "$resource_name" "$validation_result" "${validation_output//\"/\\\"}"))
                elif [[ "$OUTPUT_FORMAT" == "csv" ]]; then
                    csv_entries+=($(generate_csv_entry "$resource_name" "$validation_result" "${validation_output//\"/\\\"}"))
                fi
                
                # Show detailed output if verbose
                if [[ "$VERBOSE" == "true" && -n "$validation_output" ]]; then
                    echo "Detailed output for $resource_name:"
                    echo "$validation_output" | sed 's/^/  /'
                fi
            else
                log_error "No result file found for $resource_name"
                failed_resources=$((failed_resources + 1))
                failed_resource_names+=("$resource_name")
            fi
        done
        
        # Cleanup temp directory
        trash::safe_remove "$temp_results_dir" --no-confirm
        
    else
        # Sequential processing (original implementation)
        for script_path in "${scripts[@]}"; do
            local resource_name
            resource_name=$(get_resource_name "$script_path")
            
            log_info "Validating $resource_name..."
            log_verbose "Script path: $script_path"
            
            # Detect resource category
            local resource_category
            resource_category=$(detect_resource_category "$(dirname "$script_path")")
            log_verbose "Detected category: $resource_category"
            
            # Capture validation output
            local validation_output
            local validation_result
            
            # Choose validation level
            case "$VALIDATION_LEVEL" in
                "quick")
                    # Layer 1 only: Syntax validation
                    if validation_output=$(validate_resource_syntax "$resource_name" "$resource_category" "$script_path" "$USE_CACHE" 2>&1); then
                        validation_result="passed"
                        passed_resources=$((passed_resources + 1))
                        log_success "$resource_name passes Layer 1 syntax validation"
                    else
                        validation_result="failed"
                        failed_resources=$((failed_resources + 1))
                        failed_resource_names+=("$resource_name")
                        log_error "$resource_name fails Layer 1 syntax validation"
                    fi
                    ;;
                "standard")
                    # Layer 1 + 2: Syntax + Behavioral (future implementation)
                    log_warning "Standard validation (Layer 1+2) not yet implemented. Using Layer 1 only."
                    if validation_output=$(validate_resource_syntax "$resource_name" "$resource_category" "$script_path" "$USE_CACHE" 2>&1); then
                        validation_result="passed"
                        passed_resources=$((passed_resources + 1))
                        log_success "$resource_name passes Layer 1 syntax validation"
                    else
                        validation_result="failed"
                        failed_resources=$((failed_resources + 1))
                        failed_resource_names+=("$resource_name")
                        log_error "$resource_name fails Layer 1 syntax validation"
                    fi
                    ;;
                "full")
                    # Layer 1 + 2 + 3: All layers (future implementation)
                    log_warning "Full validation (Layer 1+2+3) not yet implemented. Using Layer 1 only."
                    if validation_output=$(validate_resource_syntax "$resource_name" "$resource_category" "$script_path" "$USE_CACHE" 2>&1); then
                        validation_result="passed"
                        passed_resources=$((passed_resources + 1))
                        log_success "$resource_name passes Layer 1 syntax validation"
                    else
                        validation_result="failed"
                        failed_resources=$((failed_resources + 1))
                        failed_resource_names+=("$resource_name")
                        log_error "$resource_name fails Layer 1 syntax validation"
                    fi
                    ;;
            esac
            
            # Store report data and send to appropriate reporter
            local duration_ms=500  # Default duration, will be improved with timing
            local cached_result="false"
            if [[ "$validation_output" =~ "CACHED" ]]; then
                cached_result="true"
            fi
            
            case "$OUTPUT_FORMAT" in
                "json")
                    json_entries+=("$(generate_json_entry "$resource_name" "$validation_result" "${validation_output//\"/\\\"}")")
                    ;;
                "csv")
                    csv_entries+=("$(generate_csv_entry "$resource_name" "$validation_result" "${validation_output//\"/\\\"}")")
                    ;;
                "text")
                    text_report_resource_result "$resource_name" "$validation_result" "$validation_output" "$duration_ms" "$cached_result"
                    ;;
                "junit")
                    junit_report_resource_result "$resource_name" "$validation_result" "$validation_output" "$duration_ms" "$cached_result"
                    ;;
            esac
            
            # Show detailed output if verbose
            if [[ "$VERBOSE" == "true" && -n "$validation_output" ]]; then
                echo "Detailed output:"
                echo "$validation_output" | sed 's/^/  /'
            fi
            
            echo "========================================"
        done
    fi
    
    # Calculate total duration (approximation for now)
    local total_duration_s=5  # Placeholder - will be improved with actual timing
    
    # Generate summary using appropriate reporter
    case "$OUTPUT_FORMAT" in
        "text")
            # Get cache stats if available
            local cache_stats
            cache_stats=$(cache_get_stats 2>/dev/null || echo "")
            
            text_report_summary "$total_resources" "$passed_resources" "$failed_resources" "$total_duration_s"
            if [[ -n "$cache_stats" ]]; then
                text_report_cache_stats "$cache_stats"
            fi
            text_report_completion "$failed_resources"
            ;;
        "junit")
            # Get cache stats for JUnit properties
            local cache_stats
            cache_stats=$(cache_get_stats 2>/dev/null || echo "")
            
            # Finalize JUnit XML output
            junit_report_finalize "$cache_stats"
            ;;
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
        
        # Apply fixes if requested
        if [[ "$FIX_ISSUES" == "true" ]]; then
            log_info "🔧 Attempting to fix interface compliance issues..."
            echo
            
            local fix_tool="$SCRIPT_DIR/fix-interface-compliance.sh"
            if [[ ! -x "$fix_tool" ]]; then
                log_error "Fix tool not found or not executable: $fix_tool"
                log_info "💡 To fix issues manually, review the validation requirements in:"
                log_info "   scripts/resources/contracts/v1.0/ (contract specifications)"
                log_info "   scripts/resources/tests/framework/ (validation framework)"
            else
                local fixes_applied=0
                local fixes_failed=0
                
                for resource in "${failed_resource_names[@]}"; do
                    log_info "Attempting to fix: $resource"
                    
                    # Run fix tool for this resource
                    if "$fix_tool" --resource "$resource" --apply --backup; then
                        log_success "✅ Applied fixes for $resource"
                        ((fixes_applied++))
                    else
                        log_error "❌ Failed to fix $resource"
                        ((fixes_failed++))
                    fi
                    echo
                done
                
                # Report fix results
                if [[ $fixes_applied -gt 0 ]]; then
                    log_success "🎉 Applied fixes to $fixes_applied resource(s)"
                    
                    if [[ $fixes_failed -eq 0 ]]; then
                        log_info "✨ All failed resources have been fixed!"
                        log_info "   Note: You may need to manually add case statement entries for new actions"
                        log_info "   Re-run validation to verify the fixes: $0 --resource <name>"
                    else
                        log_warning "⚠️  $fixes_failed resource(s) could not be automatically fixed"
                        log_info "   Manual intervention may be required for these resources"
                    fi
                else
                    log_warning "⚠️  No resources could be automatically fixed"
                    log_info "💡 To fix issues manually, review the validation requirements in:"
                    log_info "   scripts/resources/contracts/v1.0/ (contract specifications)"
                    log_info "   scripts/resources/tests/framework/ (validation framework)"
                fi
            fi
            echo
        else
            log_info "💡 To automatically fix issues, add --fix flag:"
            log_info "   $0 --fix $(if [[ -n \"$SPECIFIC_RESOURCE\" ]]; then echo \"--resource $SPECIFIC_RESOURCE\"; fi)"
            log_info ""
            log_info "   Or review validation requirements manually:"
            log_info "   scripts/resources/contracts/v1.0/ (contract specifications)"
            log_info "   scripts/resources/tests/framework/ (validation framework)"
        fi
        
        # Cleanup validation system
        syntax_validator_cleanup
        return 1
    else
        echo
        log_success "All resources pass Layer 1 syntax validation! 🎉"
        
        # Cleanup validation system
        syntax_validator_cleanup
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