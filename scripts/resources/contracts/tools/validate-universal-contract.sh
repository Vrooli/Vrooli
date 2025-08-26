#!/usr/bin/env bash
# Validate Universal Contract v2.0
# Checks resources for compliance with the v2.0 Universal Contract specification
# Usage: ./validate-universal-contract.sh [--resource <name>] [--layer <1|2|3>] [--verbose] [--format <text|json>]

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
CONTRACT_FILE="${APP_ROOT}/scripts/resources/contracts/v2.0/universal.yaml"
RESOURCES_DIR="${APP_ROOT}/resources"
VALIDATION_LAYER=1
OUTPUT_FORMAT="text"
SPECIFIC_RESOURCE=""
VERBOSE=false
SCORE_TOTAL=0
SCORE_EARNED=0
VALIDATION_RESULTS=()

# Parse arguments
while [[ $# -gt 0 ]]; do
    case "$1" in
        --resource|-r)
            SPECIFIC_RESOURCE="$2"
            shift 2
            ;;
        --layer|-l)
            VALIDATION_LAYER="$2"
            shift 2
            ;;
        --verbose|-v)
            VERBOSE=true
            shift
            ;;
        --format|-f)
            OUTPUT_FORMAT="$2"
            shift 2
            ;;
        --help|-h)
            cat << EOF
Usage: $0 [OPTIONS]

Validate resources against the v2.0 Universal Contract

Options:
  --resource, -r <name>  Validate specific resource (default: all)
  --layer, -l <1|2|3>    Validation layer (default: 1)
                         1: Syntax and structure
                         2: Behavioral testing
                         3: Integration testing
  --format, -f <format>  Output format: text, json (default: text)
  --verbose, -v          Show detailed output
  --help, -h             Show this help message

Examples:
  # Validate all resources with Layer 1
  $0

  # Validate specific resource with Layer 2
  $0 --resource ollama --layer 2 --verbose

  # Generate JSON report
  $0 --format json > compliance-report.json

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
# Validation Functions
#######################################

# Check if file exists
check_file_exists() {
    local resource="$1"
    local file="$2"
    local required="$3"
    local points="${4:-5}"
    
    local path="${RESOURCES_DIR}/${resource}/${file}"
    
    if [[ -f "$path" ]]; then
        [[ "$VERBOSE" == "true" ]] && log::success "  ✓ $file exists"
        ((SCORE_EARNED += points))
        return 0
    elif [[ "$required" == "true" ]]; then
        log::error "  ✗ Missing required file: $file"
        return 1
    else
        [[ "$VERBOSE" == "true" ]] && log::warning "  ⚠ Optional file missing: $file"
        return 2
    fi
    ((SCORE_TOTAL += points))
}

# Check command registration
check_command_registered() {
    local resource="$1"
    local command="$2"
    local cli_file="${RESOURCES_DIR}/${resource}/cli.sh"
    
    if [[ ! -f "$cli_file" ]]; then
        return 1
    fi
    
    # Check for v2.0 style registration (grouped commands)
    if echo "$command" | grep -q "::"; then
        # It's a subcommand (e.g., manage::install)
        local group="${command%%::*}"
        local subcmd="${command##*::}"
        
        if grep -q "cli::register_subcommand.*\"$group\".*\"$subcmd\"" "$cli_file" 2>/dev/null; then
            return 0
        fi
    fi
    
    # Check for v1.0 style registration (flat commands)
    if grep -q "cli::register_command.*\"$command\"" "$cli_file" 2>/dev/null; then
        return 0
    fi
    
    return 1
}

# Validate Layer 1: Syntax and Structure
validate_layer1() {
    local resource="$1"
    log::info "Layer 1: Syntax and Structure Validation"
    
    # Check required files
    check_file_exists "$resource" "cli.sh" "true" 20
    check_file_exists "$resource" "lib/core.sh" "true" 10
    check_file_exists "$resource" "lib/test.sh" "true" 10
    check_file_exists "$resource" "config/defaults.sh" "true" 10
    
    # Check optional files
    check_file_exists "$resource" "lib/install.sh" "false" 5
    check_file_exists "$resource" "lib/status.sh" "false" 5
    check_file_exists "$resource" "lib/content.sh" "false" 5
    
    # Check for deprecated files
    local deprecated_score=0
    if [[ -f "${RESOURCES_DIR}/${resource}/manage.sh" ]]; then
        log::warning "  ⚠ Deprecated file found: manage.sh"
        deprecated_score=-10
    fi
    if [[ -f "${RESOURCES_DIR}/${resource}/inject.sh" ]]; then
        log::warning "  ⚠ Deprecated file found: inject.sh"
        deprecated_score=$((deprecated_score - 5))
    fi
    ((SCORE_EARNED += deprecated_score))
    
    # Check command structure
    log::info "  Checking command structure..."
    local cli_file="${RESOURCES_DIR}/${resource}/cli.sh"
    local command_score=0
    
    if [[ -f "$cli_file" ]]; then
        # Check for v2.0 command groups
        if grep -q "cli::register_command_group" "$cli_file" 2>/dev/null; then
            log::success "    ✓ Uses v2.0 command groups"
            command_score=20
        elif grep -q "cli::register_command.*manage\|test\|content" "$cli_file" 2>/dev/null; then
            log::warning "    ⚠ Has some v2.0 commands but not using groups"
            command_score=10
        else
            log::warning "    ⚠ Using v1.0 flat command structure"
            command_score=5
        fi
        
        # Check for required commands (v2.0 style)
        local required_commands=(
            "manage::install"
            "manage::start"
            "manage::stop"
            "test::smoke"
            "content::list"
        )
        
        local found_commands=0
        for cmd in "${required_commands[@]}"; do
            if check_command_registered "$resource" "$cmd"; then
                ((found_commands++))
            fi
        done
        
        if [[ $found_commands -eq ${#required_commands[@]} ]]; then
            log::success "    ✓ All required v2.0 commands registered"
            command_score=$((command_score + 15))
        elif [[ $found_commands -gt 0 ]]; then
            log::warning "    ⚠ Some v2.0 commands missing ($found_commands/${#required_commands[@]})"
            command_score=$((command_score + 5))
        else
            # Check for v1.0 fallback
            if grep -q "cli::register_command.*\"start\"\|\"stop\"\|\"install\"" "$cli_file" 2>/dev/null; then
                log::info "    ℹ Using v1.0 command structure"
            else
                log::error "    ✗ Missing required commands"
            fi
        fi
    fi
    
    ((SCORE_EARNED += command_score))
    ((SCORE_TOTAL += 35))
}

# Validate Layer 2: Behavioral Testing
validate_layer2() {
    local resource="$1"
    log::info "Layer 2: Behavioral Testing"
    
    # Skip if Layer 1 failed
    if [[ ! -f "${RESOURCES_DIR}/${resource}/cli.sh" ]]; then
        log::error "  Cannot run Layer 2 - cli.sh not found"
        return 1
    fi
    
    local cli_cmd="resource-${resource}"
    
    # Check if CLI is installed
    if ! command -v "$cli_cmd" &>/dev/null; then
        # Try direct execution
        cli_cmd="${RESOURCES_DIR}/${resource}/cli.sh"
        if [[ ! -x "$cli_cmd" ]]; then
            log::error "  CLI not executable"
            return 1
        fi
    fi
    
    # Test help command
    log::info "  Testing help command..."
    if $cli_cmd help >/dev/null 2>&1; then
        log::success "    ✓ Help command works"
        ((SCORE_EARNED += 5))
    else
        log::error "    ✗ Help command failed"
    fi
    ((SCORE_TOTAL += 5))
    
    # Test status command (with both v1 and v2 syntax)
    log::info "  Testing status command..."
    if $cli_cmd manage status >/dev/null 2>&1 || $cli_cmd status >/dev/null 2>&1; then
        log::success "    ✓ Status command works"
        ((SCORE_EARNED += 5))
    else
        log::warning "    ⚠ Status command not available"
    fi
    ((SCORE_TOTAL += 5))
    
    # Test dry-run capability
    log::info "  Testing dry-run mode..."
    if $cli_cmd manage start --dry-run 2>&1 | grep -q "DRY RUN\|dry.run\|Would execute"; then
        log::success "    ✓ Dry-run mode works"
        ((SCORE_EARNED += 5))
    elif $cli_cmd start --dry-run 2>&1 | grep -q "DRY RUN\|dry.run\|Would execute"; then
        log::success "    ✓ Dry-run mode works (v1.0)"
        ((SCORE_EARNED += 3))
    else
        log::warning "    ⚠ Dry-run mode not detected"
    fi
    ((SCORE_TOTAL += 5))
}

# Validate Layer 3: Integration Testing
validate_layer3() {
    local resource="$1"
    log::info "Layer 3: Integration Testing"
    log::warning "  Layer 3 validation not yet implemented"
    
    # TODO: Implement full lifecycle testing
    # - Install resource
    # - Start resource
    # - Add content
    # - Verify content
    # - Stop resource
    # - Uninstall resource
}

# Calculate compliance percentage
calculate_score() {
    local percentage=0
    if [[ $SCORE_TOTAL -gt 0 ]]; then
        percentage=$((SCORE_EARNED * 100 / SCORE_TOTAL))
    fi
    echo "$percentage"
}

# Get compliance status
get_compliance_status() {
    local percentage="$1"
    
    if [[ $percentage -ge 90 ]]; then
        echo "✅ Fully Compliant"
    elif [[ $percentage -ge 70 ]]; then
        echo "🔶 Mostly Compliant"
    elif [[ $percentage -ge 50 ]]; then
        echo "⚠️  Partially Compliant"
    elif [[ $percentage -ge 25 ]]; then
        echo "⚠️  Limited Compliance"
    else
        echo "❌ Non-Compliant"
    fi
}

# Validate a single resource
validate_resource() {
    local resource="$1"
    
    # Reset scores
    SCORE_TOTAL=0
    SCORE_EARNED=0
    
    echo
    log::info "=== Validating: $resource ==="
    
    # Check if resource directory exists
    if [[ ! -d "${RESOURCES_DIR}/${resource}" ]]; then
        log::error "Resource directory not found: $resource"
        return 1
    fi
    
    # Run validation based on layer
    case "$VALIDATION_LAYER" in
        1)
            validate_layer1 "$resource"
            ;;
        2)
            validate_layer1 "$resource"
            validate_layer2 "$resource"
            ;;
        3)
            validate_layer1 "$resource"
            validate_layer2 "$resource"
            validate_layer3 "$resource"
            ;;
        *)
            log::error "Invalid validation layer: $VALIDATION_LAYER"
            return 1
            ;;
    esac
    
    # Calculate and display score
    local percentage=$(calculate_score)
    local status=$(get_compliance_status "$percentage")
    
    echo
    log::info "Compliance Score: ${percentage}% (${SCORE_EARNED}/${SCORE_TOTAL} points)"
    log::info "Status: $status"
    
    # Store result
    VALIDATION_RESULTS+=("{\"resource\":\"$resource\",\"score\":$percentage,\"points_earned\":$SCORE_EARNED,\"points_total\":$SCORE_TOTAL,\"status\":\"$status\"}")
    
    return 0
}

# Output JSON report
output_json_report() {
    echo "{"
    echo "  \"timestamp\": \"$(date -Iseconds)\","
    echo "  \"layer\": $VALIDATION_LAYER,"
    echo "  \"results\": ["
    
    local first=true
    for result in "${VALIDATION_RESULTS[@]}"; do
        if [[ "$first" == "true" ]]; then
            first=false
        else
            echo ","
        fi
        echo -n "    $result"
    done
    
    echo ""
    echo "  ],"
    echo "  \"summary\": {"
    echo "    \"total_resources\": ${#VALIDATION_RESULTS[@]},"
    echo "    \"average_score\": $(calculate_average_score)"
    echo "  }"
    echo "}"
}

# Calculate average score
calculate_average_score() {
    local total=0
    local count=0
    
    for result in "${VALIDATION_RESULTS[@]}"; do
        local score=$(echo "$result" | grep -o '"score":[0-9]*' | cut -d: -f2)
        total=$((total + score))
        count=$((count + 1))
    done
    
    if [[ $count -gt 0 ]]; then
        echo $((total / count))
    else
        echo 0
    fi
}

#######################################
# Main
#######################################

main() {
    log::info "Universal Contract v2.0 Validation Tool"
    log::info "Validation Layer: $VALIDATION_LAYER"
    
    # Determine which resources to validate
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
    
    # Validate each resource
    for resource in "${resources[@]}"; do
        validate_resource "$resource" || true
    done
    
    # Output results
    if [[ "$OUTPUT_FORMAT" == "json" ]]; then
        output_json_report
    else
        echo
        echo "========================================"
        log::info "Validation Summary"
        echo "========================================"
        log::info "Total Resources Validated: ${#VALIDATION_RESULTS[@]}"
        log::info "Average Compliance Score: $(calculate_average_score)%"
        
        # List non-compliant resources
        echo
        log::info "Resources Needing Attention:"
        for result in "${VALIDATION_RESULTS[@]}"; do
            local resource=$(echo "$result" | grep -o '"resource":"[^"]*"' | cut -d'"' -f4)
            local score=$(echo "$result" | grep -o '"score":[0-9]*' | cut -d: -f2)
            if [[ $score -lt 70 ]]; then
                printf "  %-20s %3d%% compliance\n" "$resource" "$score"
            fi
        done
    fi
}

# Run main
main