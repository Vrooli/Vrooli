#!/bin/bash
# ====================================================================
# Generate Resource Capability Map Tool
# ====================================================================
#
# Generates comprehensive capability maps showing resource relationships,
# business value, and optimal combinations for scenario generation.
#
# Usage:
#   ./generate-capability-map.sh --output capability-map.json
#   ./generate-capability-map.sh --format matrix --category ai
#   ./generate-capability-map.sh --business-focus
#
# ====================================================================

set -euo pipefail

# Configuration
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/scripts/resources/tools"
RESOURCES_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
SCENARIOS_DIR="$(cd "$RESOURCES_DIR/../scenarios" && pwd)"
OUTPUT_FILE=""
OUTPUT_FORMAT="json"
CATEGORY_FILTER=""
BUSINESS_FOCUS=false

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
Generate Resource Capability Map Tool

Creates comprehensive maps of resource capabilities, relationships, and
business value for AI-driven scenario generation and deployment planning.

USAGE:
    $0 [OPTIONS]

OPTIONS:
    --output FILE           Output file (default: stdout)
    --format FORMAT         Output format: json, yaml, matrix, markdown (default: json)
    --category CATEGORY     Filter by resource category (ai, automation, agents, storage, search)
    --business-focus        Include detailed business intelligence
    --help                  Show this help message

OUTPUT FORMATS:
    json        Complete capability data for automation
    yaml        Human-readable YAML format
    matrix      Relationship matrix showing resource combinations
    markdown    Documentation-ready markdown tables

EXAMPLES:
    # Complete capability map
    $0 --output resources-map.json

    # AI resources only with business data
    $0 --category ai --business-focus --format yaml

    # Resource relationship matrix
    $0 --format matrix --output resource-matrix.md

    # Business-focused documentation
    $0 --business-focus --format markdown
EOF
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --output)
                OUTPUT_FILE="$2"
                shift 2
                ;;
            --format)
                OUTPUT_FORMAT="$2"
                shift 2
                ;;
            --category)
                CATEGORY_FILTER="$2"
                shift 2
                ;;
            --business-focus)
                BUSINESS_FOCUS=true
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
}

# Load all resource capabilities
load_all_capabilities() {
    local capabilities=()
    
    # Find all capability files
    for category_dir in "$RESOURCES_DIR"/*/; do
        if [[ -d "$category_dir" ]]; then
            local category=$(basename "$category_dir")
            
            # Skip if category filter is set and doesn't match
            if [[ -n "$CATEGORY_FILTER" && "$category" != "$CATEGORY_FILTER" ]]; then
                continue
            fi
            
            for resource_dir in "$category_dir"/*/; do
                if [[ -d "$resource_dir" ]]; then
                    local resource_name=$(basename "$resource_dir")
                    local capabilities_file="$resource_dir/capabilities.yaml"
                    
                    if [[ -f "$capabilities_file" ]]; then
                        # Load capability data
                        local capability_data=""
                        if command -v yq >/dev/null 2>&1; then
                            capability_data=$(yq eval -o=json "$capabilities_file")
                        else
                            # Fallback: create basic structure
                            capability_data=$(cat << EOF
{
    "metadata": {"name": "$resource_name", "category": "$category"},
    "spec": {
        "capabilities": {"primary": [], "secondary": []},
        "business": {"revenue_range": {}, "complexity_score": 5},
        "scenarios": {"primary": [], "secondary": []},
        "technical": {"ports": []}
    }
}
EOF
)
                        fi
                        capabilities+=("$capability_data")
                    else
                        # Create minimal entry for resources without capabilities.yaml
                        local minimal_data=$(cat << EOF
{
    "metadata": {"name": "$resource_name", "category": "$category"},
    "spec": {
        "capabilities": {"primary": ["$category-functionality"]},
        "business": {"complexity_score": 5},
        "scenarios": {"primary": []},
        "technical": {"ports": []}
    },
    "status": "no_capabilities_file"
}
EOF
)
                        capabilities+=("$minimal_data")
                    fi
                fi
            done
        fi
    done
    
    printf '%s\n' "${capabilities[@]}"
}

# Generate resource relationship matrix
generate_relationship_matrix() {
    local capabilities=("$@")
    local matrix=()
    
    # Create header
    local resources=()
    for capability in "${capabilities[@]}"; do
        local name=$(echo "$capability" | jq -r '.metadata.name')
        resources+=("$name")
    done
    
    # Generate matrix data
    for i in "${!capabilities[@]}"; do
        local resource_a=$(echo "${capabilities[i]}" | jq -r '.metadata.name')
        local row=("$resource_a")
        
        for j in "${!capabilities[@]}"; do
            local resource_b=$(echo "${capabilities[j]}" | jq -r '.metadata.name')
            
            if [[ $i -eq $j ]]; then
                row+=("●")  # Self
            else
                # Check for compatibility
                local compatibility=$(check_resource_compatibility "${capabilities[i]}" "${capabilities[j]}")
                row+=("$compatibility")
            fi
        done
        
        matrix+=("$(IFS='|'; echo "${row[*]}")")
    done
    
    # Output matrix
    case $OUTPUT_FORMAT in
        matrix|markdown)
            echo "# Resource Compatibility Matrix"
            echo ""
            echo "| Resource |$(printf ' %s |' "${resources[@]}")"
            echo "|----------|$(printf '%*s|' ${#resources[@]} | tr ' ' '-')"
            
            for row in "${matrix[@]}"; do
                echo "| $(echo "$row" | tr '|' ' | ') |"
            done
            ;;
        json)
            echo '{"matrix": {'
            echo '"headers": ['"$(printf '"%s",' "${resources[@]}" | sed 's/,$//')"'],'
            echo '"rows": ['
            for i in "${!matrix[@]}"; do
                local row_data=($(echo "${matrix[i]}" | tr '|' ' '))
                echo '  ["'"$(IFS='","'; echo "${row_data[*]}")"'"]'
                if [[ $i -lt $((${#matrix[@]} - 1)) ]]; then
                    echo ","
                fi
            done
            echo "]}"
            ;;
    esac
}

# Check compatibility between two resources
check_resource_compatibility() {
    local resource_a="$1"
    local resource_b="$2"
    
    local name_a=$(echo "$resource_a" | jq -r '.metadata.name')
    local name_b=$(echo "$resource_b" | jq -r '.metadata.name')
    
    # Check for explicit optimal combinations
    local combinations_a=($(echo "$resource_a" | jq -r '.spec.scenarios.optimal_combinations[][]? // empty' 2>/dev/null | tr '\n' ' '))
    for combo in "${combinations_a[@]}"; do
        if [[ "$combo" == "$name_b" ]]; then
            return "★"  # Optimal
        fi
    done
    
    # Check for same category (good compatibility)
    local cat_a=$(echo "$resource_a" | jq -r '.metadata.category')
    local cat_b=$(echo "$resource_b" | jq -r '.metadata.category')
    
    # Check for complementary categories
    case "$cat_a-$cat_b" in
        "ai-automation"|"automation-ai"|"ai-storage"|"storage-ai"|"automation-storage"|"storage-automation")
            echo "◐"  # Good compatibility
            ;;
        "agents-ai"|"ai-agents"|"agents-automation"|"automation-agents")
            echo "◑"  # Very good compatibility
            ;;
        *)
            echo "○"  # Basic compatibility
            ;;
    esac
}

# Generate business intelligence report
generate_business_report() {
    local capabilities=("$@")
    
    case $OUTPUT_FORMAT in
        markdown)
            echo "# Resource Business Intelligence Report"
            echo ""
            echo "## Revenue Potential by Category"
            echo ""
            echo "| Resource | Category | Revenue Range | Complexity | Target Industries |"
            echo "|----------|----------|---------------|------------|------------------|"
            
            for capability in "${capabilities[@]}"; do
                local name=$(echo "$capability" | jq -r '.metadata.name')
                local category=$(echo "$capability" | jq -r '.metadata.category')
                local revenue=$(echo "$capability" | jq -r '.spec.business.revenue_range // {}' | jq -r 'if type == "object" then "\(.min // "N/A")-\(.max // "N/A") \(.currency // "")" else . end' 2>/dev/null || echo "N/A")
                local complexity=$(echo "$capability" | jq -r '.spec.business.complexity_score // "N/A"')
                local industries=$(echo "$capability" | jq -r '.spec.business.target_industries[]? // empty' 2>/dev/null | head -3 | tr '\n' ', ' | sed 's/,$//')
                
                echo "| $name | $category | $revenue | $complexity/10 | $industries |"
            done
            
            echo ""
            echo "## Scenario Integration Summary"
            echo ""
            
            # Group by scenarios
            local all_scenarios=()
            for capability in "${capabilities[@]}"; do
                local scenarios=($(echo "$capability" | jq -r '.spec.scenarios.primary[]? // empty' 2>/dev/null))
                for scenario in "${scenarios[@]}"; do
                    if [[ ! " ${all_scenarios[@]} " =~ " $scenario " ]]; then
                        all_scenarios+=("$scenario")
                    fi
                done
            done
            
            echo "| Scenario | Resources | Business Value |"
            echo "|----------|-----------|----------------|"
            
            for scenario in "${all_scenarios[@]}"; do
                local scenario_resources=()
                local scenario_revenue=""
                
                for capability in "${capabilities[@]}"; do
                    local name=$(echo "$capability" | jq -r '.metadata.name')
                    local scenarios=($(echo "$capability" | jq -r '.spec.scenarios.primary[]? // empty' 2>/dev/null))
                    for s in "${scenarios[@]}"; do
                        if [[ "$s" == "$scenario" ]]; then
                            scenario_resources+=("$name")
                            if [[ -z "$scenario_revenue" ]]; then
                                scenario_revenue=$(echo "$capability" | jq -r '.spec.business.revenue_range // {}' | jq -r 'if type == "object" then "\(.min // "N/A")-\(.max // "N/A") \(.currency // "")" else . end' 2>/dev/null || echo "N/A")
                            fi
                            break
                        fi
                    done
                done
                
                echo "| $scenario | $(IFS=', '; echo "${scenario_resources[*]}") | $scenario_revenue |"
            done
            ;;
        json)
            echo '{'
            echo '"business_intelligence": {'
            echo '"resources": ['
            
            for i in "${!capabilities[@]}"; do
                local capability="${capabilities[i]}"
                local name=$(echo "$capability" | jq -r '.metadata.name')
                local business=$(echo "$capability" | jq '.spec.business // {}')
                
                echo "  {"
                echo "    \"name\": \"$name\","
                echo "    \"business\": $business"
                echo "  }"
                
                if [[ $i -lt $((${#capabilities[@]} - 1)) ]]; then
                    echo ","
                fi
            done
            
            echo "],"
            echo '"summary": {'
            
            # Calculate totals
            local total_resources=${#capabilities[@]}
            local avg_complexity=0
            local complexity_sum=0
            
            for capability in "${capabilities[@]}"; do
                local complexity=$(echo "$capability" | jq -r '.spec.business.complexity_score // 5')
                complexity_sum=$((complexity_sum + complexity))
            done
            
            if [[ $total_resources -gt 0 ]]; then
                avg_complexity=$((complexity_sum / total_resources))
            fi
            
            echo "  \"total_resources\": $total_resources,"
            echo "  \"average_complexity\": $avg_complexity,"
            echo "  \"categories\": $(for capability in "${capabilities[@]}"; do echo "$capability" | jq -r '.metadata.category'; done | sort | uniq -c | wc -l)"
            echo "}"
            echo "}"
            echo "}"
            ;;
    esac
}

# Generate complete capability map
generate_complete_map() {
    local capabilities=("$@")
    
    case $OUTPUT_FORMAT in
        json)
            echo '{'
            echo '"metadata": {'
            echo '  "generated_at": "'$(date -Iseconds)'",'
            echo '  "total_resources": '${#capabilities[@]}','
            echo '  "category_filter": "'${CATEGORY_FILTER:-all}'"'
            echo '},'
            echo '"capabilities": ['
            
            for i in "${!capabilities[@]}"; do
                echo "${capabilities[i]}"
                if [[ $i -lt $((${#capabilities[@]} - 1)) ]]; then
                    echo ","
                fi
            done
            
            echo "]"
            echo "}"
            ;;
        yaml)
            echo "metadata:"
            echo "  generated_at: $(date -Iseconds)"
            echo "  total_resources: ${#capabilities[@]}"
            echo "  category_filter: ${CATEGORY_FILTER:-all}"
            echo ""
            echo "capabilities:"
            
            for capability in "${capabilities[@]}"; do
                echo "$capability" | yq eval -P | sed 's/^/  /'
                echo ""
            done
            ;;
    esac
}

# Main execution
main() {
    parse_args "$@"
    
    echo "🔍 Loading resource capabilities..." >&2
    local capabilities=($(load_all_capabilities))
    
    echo "📊 Found ${#capabilities[@]} resources" >&2
    if [[ -n "$CATEGORY_FILTER" ]]; then
        echo "🏷️  Filtered by category: $CATEGORY_FILTER" >&2
    fi
    
    # Generate output based on format
    local output=""
    case $OUTPUT_FORMAT in
        matrix)
            output=$(generate_relationship_matrix "${capabilities[@]}")
            ;;
        markdown)
            if [[ "$BUSINESS_FOCUS" == true ]]; then
                output=$(generate_business_report "${capabilities[@]}")
            else
                output=$(generate_relationship_matrix "${capabilities[@]}")
            fi
            ;;
        json|yaml)
            if [[ "$BUSINESS_FOCUS" == true ]]; then
                output=$(generate_business_report "${capabilities[@]}")
            else
                output=$(generate_complete_map "${capabilities[@]}")
            fi
            ;;
        *)
            echo "❌ Unknown output format: $OUTPUT_FORMAT"
            exit 1
            ;;
    esac
    
    # Output to file or stdout
    if [[ -n "$OUTPUT_FILE" ]]; then
        echo "$output" > "$OUTPUT_FILE"
        echo "✅ Capability map generated: $OUTPUT_FILE" >&2
    else
        echo "$output"
    fi
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi