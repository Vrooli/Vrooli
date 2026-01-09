#!/bin/bash
# ====================================================================
# Find Resources for Scenario Tool
# ====================================================================
#
# Discovers which resources are required or recommended for a specific
# business scenario by analyzing scenario metadata and dependencies.
#
# Usage:
#   ./find-resources-for.sh --scenario research-assistant
#   ./find-resources-for.sh --scenario research-assistant --show-alternatives
#   ./find-resources-for.sh --all
#
# ====================================================================

set -euo pipefail

# Configuration
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/scripts/resources/tools"
RESOURCES_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
SCENARIOS_DIR="$(cd "$RESOURCES_DIR/../scenarios" && pwd)"
OUTPUT_FORMAT="text"
SHOW_ALTERNATIVES=false
SHOW_BUSINESS_INFO=false

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
Find Resources for Scenario Tool

Discovers which resources are required, recommended, or optional for
specific business scenarios, including alternatives and business value analysis.

USAGE:
    $0 --scenario SCENARIO [OPTIONS]
    $0 --all [OPTIONS]

OPTIONS:
    --scenario SCENARIO     Scenario name (e.g., research-assistant)
    --all                   Analyze all scenarios
    --show-alternatives     Include alternative resource suggestions
    --business-info         Include business value and cost information
    --format FORMAT         Output format: text, json, yaml (default: text)
    --help                  Show this help message

EXAMPLES:
    # Basic resource analysis
    $0 --scenario research-assistant

    # Detailed analysis with alternatives
    $0 --scenario research-assistant --show-alternatives --business-info

    # Analyze all scenarios
    $0 --all --format json

    # Get deployment planning information
    $0 --scenario research-assistant --business-info
EOF
}

# Parse command line arguments
parse_args() {
    SCENARIO=""
    SHOW_ALL=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --scenario)
                SCENARIO="$2"
                shift 2
                ;;
            --all)
                SHOW_ALL=true
                shift
                ;;
            --show-alternatives)
                SHOW_ALTERNATIVES=true
                shift
                ;;
            --business-info)
                SHOW_BUSINESS_INFO=true
                shift
                ;;
            --format)
                OUTPUT_FORMAT="$2"
                shift 2
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
    
    if [[ "$SHOW_ALL" == false && -z "$SCENARIO" ]]; then
        echo "❌ Either --scenario or --all must be specified"
        usage
        exit 1
    fi
}

# Load resource capability data
load_resource_capabilities() {
    local resource_name="$1"
    
    # Try to find capabilities file
    local capabilities_file=""
    for category_dir in "$RESOURCES_DIR"/*/; do
        local potential_file="$category_dir/$resource_name/capabilities.yaml"
        if [[ -f "$potential_file" ]]; then
            capabilities_file="$potential_file"
            break
        fi
    done
    
    if [[ -z "$capabilities_file" ]]; then
        # Return minimal data if no capabilities file found
        cat << EOF
{
    "name": "$resource_name",
    "category": "unknown",
    "business": {"complexity_score": 5},
    "technical": {"ports": []}
}
EOF
        return
    fi
    
    # Convert YAML to JSON (basic conversion)
    if command -v yq >/dev/null 2>&1; then
        yq eval -o=json "$capabilities_file"
    else
        # Fallback: create basic JSON structure
        cat << EOF
{
    "metadata": {"name": "$resource_name"},
    "spec": {
        "business": {"complexity_score": 5},
        "technical": {"ports": []}
    }
}
EOF
    fi
}

# Extract scenario analysis
analyze_scenario() {
    local scenario_dir="$1"
    local scenario_name=$(basename "$scenario_dir")
    
    # Check if scenario exists
    if [[ ! -d "$scenario_dir" ]]; then
        echo "❌ Scenario not found: $scenario_name"
        return 1
    fi
    
    local metadata_file="$scenario_dir/metadata.yaml"
    local test_file="$scenario_dir/test.sh"
    local manifest_file="$scenario_dir/manifest.yaml"
    
    # Initialize arrays
    local required_resources=()
    local optional_resources=()
    local description=""
    local business_value=""
    local revenue_range=""
    local complexity=""
    local implementation_time=""
    
    # Extract from metadata.yaml
    if [[ -f "$metadata_file" ]]; then
        if command -v yq >/dev/null 2>&1; then
            required_resources=($(yq eval '.scenario.dependencies.resources.required[]? // empty' "$metadata_file" 2>/dev/null | tr '\n' ' '))
            optional_resources=($(yq eval '.scenario.dependencies.resources.optional[]? // empty' "$metadata_file" 2>/dev/null | tr '\n' ' '))
            description=$(yq eval '.scenario.description // ""' "$metadata_file" 2>/dev/null)
            business_value=$(yq eval '.scenario.business.value_proposition // ""' "$metadata_file" 2>/dev/null)
            revenue_range=$(yq eval '.scenario.business.revenue_range // ""' "$metadata_file" 2>/dev/null)
            complexity=$(yq eval '.scenario.complexity // ""' "$metadata_file" 2>/dev/null)
        fi
    fi
    
    # Extract from test.sh if metadata incomplete
    if [[ -f "$test_file" && ${#required_resources[@]} -eq 0 ]]; then
        # Parse REQUIRED_RESOURCES array
        if grep -q "REQUIRED_RESOURCES=" "$test_file"; then
            local resources_line=$(grep "REQUIRED_RESOURCES=" "$test_file" | head -1)
            required_resources=($(echo "$resources_line" | sed -n 's/.*REQUIRED_RESOURCES=(\([^)]*\)).*/\1/p' | tr -d '"' | tr ',' ' '))
        fi
        
        # Parse @services comment
        if grep -q "# @services:" "$test_file"; then
            local services_line=$(grep "# @services:" "$test_file" | head -1)
            required_resources=($(echo "$services_line" | sed 's/.*@services:\s*//' | tr ',' ' '))
        fi
        
        # Parse @optional-services comment
        if grep -q "# @optional-services:" "$test_file"; then
            local optional_line=$(grep "# @optional-services:" "$test_file" | head -1)
            optional_resources=($(echo "$optional_line" | sed 's/.*@optional-services:\s*//' | tr ',' ' '))
        fi
        
        # Extract other metadata from comments
        if grep -q "# @revenue-potential:" "$test_file"; then
            revenue_range=$(grep "# @revenue-potential:" "$test_file" | head -1 | sed 's/.*@revenue-potential:\s*//')
        fi
        
        if grep -q "# @complexity:" "$test_file"; then
            complexity=$(grep "# @complexity:" "$test_file" | head -1 | sed 's/.*@complexity:\s*//')
        fi
        
        if grep -q "# @duration:" "$test_file"; then
            implementation_time=$(grep "# @duration:" "$test_file" | head -1 | sed 's/.*@duration:\s*//')
        fi
    fi
    
    # Analyze resource capabilities and suggest alternatives
    local resource_analysis=()
    local total_complexity=0
    local estimated_ports=()
    
    for resource in "${required_resources[@]}" "${optional_resources[@]}"; do
        if [[ -n "$resource" ]]; then
            local capabilities=$(load_resource_capabilities "$resource")
            resource_analysis+=("$capabilities")
            
            # Extract complexity and ports
            local complexity_score=$(echo "$capabilities" | jq -r '.spec.business.complexity_score // 5' 2>/dev/null || echo "5")
            local ports=($(echo "$capabilities" | jq -r '.spec.technical.ports[]? // empty' 2>/dev/null || true))
            
            total_complexity=$((total_complexity + complexity_score))
            estimated_ports+=("${ports[@]}")
        fi
    done
    
    # Calculate deployment information
    local avg_complexity=$((total_complexity / (${#required_resources[@]} + ${#optional_resources[@]} + 1)))
    local unique_ports=($(printf '%s\n' "${estimated_ports[@]}" | sort -u))
    
    # Generate alternatives if requested
    local alternatives=()
    if [[ "$SHOW_ALTERNATIVES" == true ]]; then
        alternatives=($(find_alternative_resources "${required_resources[@]}"))
    fi
    
    # Output result
    cat << EOF
{
    "scenario": "$scenario_name",
    "description": "$description",
    "complexity": "$complexity",
    "business": {
        "value_proposition": "$business_value",
        "revenue_range": "$revenue_range",
        "implementation_time": "$implementation_time",
        "avg_complexity_score": $avg_complexity
    },
    "resources": {
        "required": [$(printf '"%s",' "${required_resources[@]}" | sed 's/,$//')]
        "optional": [$(printf '"%s",' "${optional_resources[@]}" | sed 's/,$//')]
        "alternatives": [$(printf '"%s",' "${alternatives[@]}" | sed 's/,$//')]
    },
    "deployment": {
        "estimated_ports": [$(printf '%s,' "${unique_ports[@]}" | sed 's/,$//')]
        "total_complexity": $total_complexity,
        "resource_count": $((${#required_resources[@]} + ${#optional_resources[@]}))
    },
    "path": "$scenario_dir"
}
EOF
}

# Find alternative resources based on capabilities
find_alternative_resources() {
    local target_resources=("$@")
    local alternatives=()
    
    # Simple alternative mapping (could be enhanced with capability analysis)
    for resource in "${target_resources[@]}"; do
        case "$resource" in
            "ollama")
                alternatives+=("openai-api" "anthropic-api")
                ;;
            "agent-s2")
                alternatives+=("browserless" "selenium")
                ;;
            "whisper")
                alternatives+=("google-speech-api" "azure-speech")
                ;;
            "minio")
                alternatives+=("aws-s3" "local-filesystem")
                ;;
            "postgres")
                alternatives+=("mysql" "sqlite" "mongodb")
                ;;
        esac
    done
    
    printf '%s\n' "${alternatives[@]}" | sort -u
}

# Display scenario analysis
display_scenario_analysis() {
    local scenario_data="$1"
    local scenario_name=$(echo "$scenario_data" | jq -r '.scenario')
    
    case $OUTPUT_FORMAT in
        json)
            echo "$scenario_data"
            ;;
        yaml)
            echo "$scenario_data" | yq eval -P
            ;;
        text)
            echo -e "${BLUE}📋 Scenario: ${CYAN}$scenario_name${NC}"
            
            local description=$(echo "$scenario_data" | jq -r '.description // ""')
            if [[ -n "$description" && "$description" != "null" ]]; then
                echo -e "${YELLOW}Description:${NC} $description"
            fi
            
            echo ""
            echo -e "${GREEN}🔧 Required Resources:${NC}"
            local required=($(echo "$scenario_data" | jq -r '.dependencies.resources.required[]? // empty'))
            for resource in "${required[@]}"; do
                echo -e "  ${GREEN}✅ $resource${NC}"
            done
            
            local optional=($(echo "$scenario_data" | jq -r '.dependencies.resources.optional[]? // empty'))
            if [[ ${#optional[@]} -gt 0 ]]; then
                echo ""
                echo -e "${YELLOW}⚙️ Optional Resources:${NC}"
                for resource in "${optional[@]}"; do
                    echo -e "  ${YELLOW}🔧 $resource${NC}"
                done
            fi
            
            if [[ "$SHOW_ALTERNATIVES" == true ]]; then
                local alternatives=($(echo "$scenario_data" | jq -r '.dependencies.resources.alternatives[]? // empty'))
                if [[ ${#alternatives[@]} -gt 0 ]]; then
                    echo ""
                    echo -e "${PURPLE}🔄 Alternative Resources:${NC}"
                    for alt in "${alternatives[@]}"; do
                        echo -e "  ${PURPLE}🔀 $alt${NC}"
                    done
                fi
            fi
            
            if [[ "$SHOW_BUSINESS_INFO" == true ]]; then
                echo ""
                echo -e "${BLUE}💼 Business Information:${NC}"
                local revenue=$(echo "$scenario_data" | jq -r '.business.revenue_range // ""')
                local complexity=$(echo "$scenario_data" | jq -r '.business.avg_complexity_score // 0')
                local resource_count=$(echo "$scenario_data" | jq -r '.deployment.resource_count // 0')
                
                if [[ -n "$revenue" && "$revenue" != "null" ]]; then
                    echo -e "  ${GREEN}💰 Revenue Potential:${NC} $revenue"
                fi
                echo -e "  ${YELLOW}📊 Complexity Score:${NC} $complexity/10"
                echo -e "  ${CYAN}🔧 Resource Count:${NC} $resource_count"
                
                local ports=($(echo "$scenario_data" | jq -r '.deployment.estimated_ports[]? // empty'))
                if [[ ${#ports[@]} -gt 0 ]]; then
                    echo -e "  ${PURPLE}🌐 Estimated Ports:${NC} $(IFS=', '; echo "${ports[*]}")"
                fi
            fi
            
            echo ""
            ;;
    esac
}

# Analyze all scenarios
analyze_all_scenarios() {
    local scenarios=()
    
    # Find all scenario directories
    for scenario_dir in "$SCENARIOS_DIR"/*/; do
        if [[ -d "$scenario_dir" ]]; then
            scenarios+=("$scenario_dir")
        fi
    done
    
    case $OUTPUT_FORMAT in
        json)
            echo '{"scenarios": ['
            ;;
        text)
            echo -e "${BLUE}🔍 Scenario Resource Analysis${NC}"
            echo -e "${BLUE}Found ${#scenarios[@]} scenarios${NC}"
            echo ""
            ;;
    esac
    
    for i in "${!scenarios[@]}"; do
        local analysis=$(analyze_scenario "${scenarios[i]}")
        display_scenario_analysis "$analysis"
        
        if [[ "$OUTPUT_FORMAT" == "json" && $i -lt $((${#scenarios[@]} - 1)) ]]; then
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
        analyze_all_scenarios
    else
        local scenario_dir="$SCENARIOS_DIR/$SCENARIO"
        local analysis=$(analyze_scenario "$scenario_dir")
        display_scenario_analysis "$analysis"
    fi
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
