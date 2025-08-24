#!/bin/bash
# ====================================================================
# Find Scenarios Using Resource Tool
# ====================================================================
#
# Discovers which business scenarios use a specific resource by analyzing
# scenario metadata and test files. Supports both individual resources
# and resource combinations.
#
# Usage:
#   ./find-scenarios-using.sh --resource ollama
#   ./find-scenarios-using.sh --resource "ollama,n8n"
#   ./find-scenarios-using.sh --all
#
# ====================================================================

set -euo pipefail

# Configuration
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/scripts/resources/tools"
RESOURCES_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
SCENARIOS_DIR="$(cd "$RESOURCES_DIR/../scenarios" && pwd)"
OUTPUT_FORMAT="text"
SHOW_BUSINESS_VALUE=false

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Usage information
usage() {
    cat << EOF
Find Scenarios Using Resource Tool

Discovers which business scenarios use specific resources by analyzing
scenario metadata, test files, and resource dependencies.

USAGE:
    $0 --resource RESOURCE [OPTIONS]
    $0 --all [OPTIONS]

OPTIONS:
    --resource RESOURCE     Resource name (e.g., ollama, n8n) or comma-separated list
    --all                   Show usage for all resources
    --format FORMAT         Output format: text, json, yaml (default: text)
    --business-value        Include business value information
    --help                  Show this help message

EXAMPLES:
    # Find scenarios using Ollama
    $0 --resource ollama

    # Find scenarios using multiple resources
    $0 --resource "ollama,n8n,minio"

    # Show all resource usage with business value
    $0 --all --business-value

    # JSON output for automation
    $0 --resource ollama --format json
EOF
}

# Parse command line arguments
parse_args() {
    RESOURCE=""
    SHOW_ALL=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --resource)
                RESOURCE="$2"
                shift 2
                ;;
            --all)
                SHOW_ALL=true
                shift
                ;;
            --format)
                OUTPUT_FORMAT="$2"
                shift 2
                ;;
            --business-value)
                SHOW_BUSINESS_VALUE=true
                shift
                ;;
            --help)
                usage
                exit 0
                ;;
            *)
                echo "❌ Unknown option: $1"
                usage
                exit 1
                ;;
        esac
    done
    
    if [[ "$SHOW_ALL" == false && -z "$RESOURCE" ]]; then
        echo "❌ Either --resource or --all must be specified"
        usage
        exit 1
    fi
}

# Extract scenario metadata
extract_scenario_metadata() {
    local scenario_dir="$1"
    local metadata_file="$scenario_dir/metadata.yaml"
    local test_file="$scenario_dir/test.sh"
    
    # Initialize result object
    local scenario_name=$(basename "$scenario_dir")
    local required_resources=()
    local optional_resources=()
    local business_value=""
    local revenue_range=""
    local description=""
    
    # Extract from metadata.yaml if it exists
    if [[ -f "$metadata_file" ]]; then
        # Parse YAML for resources (basic parsing)
        if command -v yq >/dev/null 2>&1; then
            required_resources=($(yq eval '.scenario.resources.required[]? // empty' "$metadata_file" 2>/dev/null || true))
            optional_resources=($(yq eval '.scenario.resources.optional[]? // empty' "$metadata_file" 2>/dev/null || true))
            business_value=$(yq eval '.scenario.business.value_proposition // ""' "$metadata_file" 2>/dev/null || true)
            revenue_range=$(yq eval '.scenario.business.revenue_range // ""' "$metadata_file" 2>/dev/null || true)
            description=$(yq eval '.scenario.description // ""' "$metadata_file" 2>/dev/null || true)
        else
            # Fallback grep-based parsing
            required_resources=($(grep -A 10 "required:" "$metadata_file" 2>/dev/null | grep -E "^\s*-" | sed 's/^\s*-\s*//' | tr -d '"' || true))
            optional_resources=($(grep -A 10 "optional:" "$metadata_file" 2>/dev/null | grep -E "^\s*-" | sed 's/^\s*-\s*//' | tr -d '"' || true))
        fi
    fi
    
    # Extract from test.sh if metadata is incomplete
    if [[ -f "$test_file" ]] && [[ ${#required_resources[@]} -eq 0 ]]; then
        # Look for REQUIRED_RESOURCES array
        if grep -q "REQUIRED_RESOURCES=" "$test_file"; then
            local resources_line=$(grep "REQUIRED_RESOURCES=" "$test_file" | head -1)
            required_resources=($(echo "$resources_line" | sed -n 's/.*REQUIRED_RESOURCES=(\([^)]*\)).*/\1/p' | tr -d '"' | tr ' ' '\n'))
        fi
        
        # Look for @services comment
        if grep -q "# @services:" "$test_file"; then
            local services_line=$(grep "# @services:" "$test_file" | head -1)
            required_resources=($(echo "$services_line" | sed 's/.*@services:\s*//' | tr ',' '\n' | xargs))
        fi
        
        # Look for @optional-services comment
        if grep -q "# @optional-services:" "$test_file"; then
            local optional_line=$(grep "# @optional-services:" "$test_file" | head -1)
            optional_resources=($(echo "$optional_line" | sed 's/.*@optional-services:\s*//' | tr ',' '\n' | xargs))
        fi
        
        # Extract business value from comments
        if [[ -z "$business_value" ]] && grep -q "# @revenue-potential:" "$test_file"; then
            revenue_range=$(grep "# @revenue-potential:" "$test_file" | head -1 | sed 's/.*@revenue-potential:\s*//')
        fi
    fi
    
    # Output result
    cat << EOF
{
    "name": "$scenario_name",
    "description": "$description",
    "required_resources": [$(printf '"%s",' "${required_resources[@]}" | sed 's/,$//')]
    "optional_resources": [$(printf '"%s",' "${optional_resources[@]}" | sed 's/,$//')]
    "business_value": "$business_value",
    "revenue_range": "$revenue_range",
    "path": "$scenario_dir"
}
EOF
}

# Check if scenario uses resource
scenario_uses_resource() {
    local scenario_data="$1"
    local target_resource="$2"
    
    # Extract resources from scenario data
    local required=$(echo "$scenario_data" | jq -r '.required_resources[]?' 2>/dev/null || true)
    local optional=$(echo "$scenario_data" | jq -r '.optional_resources[]?' 2>/dev/null || true)
    local all_resources="$required $optional"
    
    # Check if target resource is in the list
    for resource in $all_resources; do
        if [[ "$resource" == "$target_resource" ]]; then
            return 0
        fi
    done
    return 1
}

# Find scenarios using specific resource
find_scenarios_for_resource() {
    local target_resource="$1"
    local matching_scenarios=()
    
    # Iterate through all scenarios
    for scenario_dir in "$SCENARIOS_DIR"/core/*/; do
        if [[ -d "$scenario_dir" ]]; then
            local scenario_data=$(extract_scenario_metadata "$scenario_dir")
            if scenario_uses_resource "$scenario_data" "$target_resource"; then
                matching_scenarios+=("$scenario_data")
            fi
        fi
    done
    
    # Output results based on format
    case $OUTPUT_FORMAT in
        json)
            printf '{"resource": "%s", "scenarios": [' "$target_resource"
            for i in "${!matching_scenarios[@]}"; do
                echo "${matching_scenarios[i]}"
                if [[ $i -lt $((${#matching_scenarios[@]} - 1)) ]]; then
                    echo ","
                fi
            done
            echo "]}"
            ;;
        text)
            echo -e "${BLUE}📦 Resource: ${CYAN}$target_resource${NC}"
            echo -e "${BLUE}📊 Found ${#matching_scenarios[@]} scenarios using this resource:${NC}"
            echo ""
            
            for scenario_data in "${matching_scenarios[@]}"; do
                local name=$(echo "$scenario_data" | jq -r '.name')
                local description=$(echo "$scenario_data" | jq -r '.description')
                local revenue=$(echo "$scenario_data" | jq -r '.revenue_range')
                
                echo -e "  ${GREEN}✅ $name${NC}"
                if [[ -n "$description" && "$description" != "null" ]]; then
                    echo -e "     ${YELLOW}Description:${NC} $description"
                fi
                if [[ "$SHOW_BUSINESS_VALUE" == true && -n "$revenue" && "$revenue" != "null" ]]; then
                    echo -e "     ${PURPLE}Revenue Potential:${NC} $revenue"
                fi
                echo ""
            done
            ;;
        yaml)
            echo "resource: $target_resource"
            echo "scenario_count: ${#matching_scenarios[@]}"
            echo "scenarios:"
            for scenario_data in "${matching_scenarios[@]}"; do
                local name=$(echo "$scenario_data" | jq -r '.name')
                local description=$(echo "$scenario_data" | jq -r '.description')
                echo "  - name: $name"
                if [[ -n "$description" && "$description" != "null" ]]; then
                    echo "    description: \"$description\""
                fi
            done
            ;;
    esac
}

# Find scenarios for all resources
find_all_resource_usage() {
    # Get list of all resources
    local all_resources=()
    
    # Scan capabilities files
    for capabilities_file in "$RESOURCES_DIR"/*/*/capabilities.yaml; do
        if [[ -f "$capabilities_file" ]]; then
            local resource_name=$(yq eval '.metadata.name' "$capabilities_file" 2>/dev/null || basename "$(dirname "$capabilities_file")")
            all_resources+=("$resource_name")
        fi
    done
    
    # Scan resource directories if no capabilities files found
    if [[ ${#all_resources[@]} -eq 0 ]]; then
        for category_dir in "$RESOURCES_DIR"/*/; do
            if [[ -d "$category_dir" ]]; then
                for resource_dir in "$category_dir"/*/; do
                    if [[ -d "$resource_dir" ]]; then
                        all_resources+=($(basename "$resource_dir"))
                    fi
                done
            fi
        done
    fi
    
    # Find scenarios for each resource
    case $OUTPUT_FORMAT in
        json)
            echo '{"resource_usage": ['
            ;;
        text)
            echo -e "${BLUE}🔍 Resource Usage Analysis${NC}"
            echo -e "${BLUE}Found ${#all_resources[@]} resources${NC}"
            echo ""
            ;;
    esac
    
    for i in "${!all_resources[@]}"; do
        find_scenarios_for_resource "${all_resources[i]}"
        
        if [[ "$OUTPUT_FORMAT" == "json" && $i -lt $((${#all_resources[@]} - 1)) ]]; then
            echo ","
        fi
    done
    
    if [[ "$OUTPUT_FORMAT" == "json" ]]; then
        echo "]}"
    fi
}

# Main execution
main() {
    parse_args "$@"
    
    # Validate scenarios directory
    if [[ ! -d "$SCENARIOS_DIR/core" ]]; then
        echo "❌ Scenarios directory not found: $SCENARIOS_DIR/core"
        exit 1
    fi
    
    if [[ "$SHOW_ALL" == true ]]; then
        find_all_resource_usage
    else
        # Handle comma-separated resources
        IFS=',' read -ra RESOURCES <<< "$RESOURCE"
        for resource in "${RESOURCES[@]}"; do
            resource=$(echo "$resource" | xargs)  # trim whitespace
            find_scenarios_for_resource "$resource"
            if [[ ${#RESOURCES[@]} -gt 1 ]]; then
                echo ""
            fi
        done
    fi
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi