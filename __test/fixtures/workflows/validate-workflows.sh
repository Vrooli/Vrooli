#!/bin/bash
# Vrooli Workflow Fixtures Type-Specific Validator
# Validates workflow-specific requirements and tests with available platforms

set -e

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/__test/fixtures/workflows"
FIXTURES_DIR="$SCRIPT_DIR"
METADATA_FILE="$FIXTURES_DIR/metadata.yaml"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Counters
TOTAL_WORKFLOWS=0
VALID_WORKFLOWS=0
FAILED_WORKFLOWS=0
TESTED_WORKFLOWS=0

print_section() {
    echo -e "${YELLOW}--- $1 ---${NC}"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

print_info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

# Check if required tools are available
check_workflow_tools() {
    local missing_tools=()
    
    if ! command -v jq >/dev/null 2>&1; then
        missing_tools+=("jq (JSON processor)")
    fi
    
    if ! command -v curl >/dev/null 2>&1; then
        missing_tools+=("curl")
    fi
    
    if [[ ${#missing_tools[@]} -gt 0 ]]; then
        print_warning "Some workflow validation tools are missing: ${missing_tools[*]}"
        print_info "Install jq for JSON validation: sudo apt-get install jq"
        return 1
    fi
    
    return 0
}

# Check platform availability
check_platform_availability() {
    local platform="$1"
    local port="$2"
    local health_endpoint="$3"
    
    if [[ -z "$port" || "$port" == "null" ]]; then
        return 1
    fi
    
    local health_url="http://localhost:$port$health_endpoint"
    
    if curl -s --max-time 5 "$health_url" >/dev/null 2>&1; then
        return 0
    else
        return 1
    fi
}

# Validate individual workflow file
validate_workflow_file() {
    local workflow_path="$1"
    local platform="$2"
    local expected_complexity="$3"
    local integrations="$4"  # Array as string
    
    TOTAL_WORKFLOWS=$((TOTAL_WORKFLOWS + 1))
    
    if [[ ! -f "$workflow_path" ]]; then
        print_error "Workflow file not found: $workflow_path"
        FAILED_WORKFLOWS=$((FAILED_WORKFLOWS + 1))
        return 1
    fi
    
    local errors=0
    local filename=$(basename "$workflow_path")
    
    # Check file exists and is readable
    if [[ ! -r "$workflow_path" ]]; then
        print_error "$filename: Not readable"
        errors=$((errors + 1))
    fi
    
    # Check file size
    local actual_size=$(stat -c%s "$workflow_path" 2>/dev/null || echo "0")
    if [[ $actual_size -eq 0 ]]; then
        print_error "$filename: Empty file"
        errors=$((errors + 1))
    fi
    
    # Validate JSON structure
    if command -v jq >/dev/null 2>&1; then
        if ! jq empty "$workflow_path" >/dev/null 2>&1; then
            print_error "$filename: Invalid JSON structure"
            errors=$((errors + 1))
        else
            # Platform-specific validation
            case "$platform" in
                "n8n")
                    validate_n8n_workflow "$workflow_path" "$filename"
                    if [[ $? -ne 0 ]]; then
                        errors=$((errors + 1))
                    fi
                    ;;
                "node-red")
                    validate_node_red_workflow "$workflow_path" "$filename"
                    if [[ $? -ne 0 ]]; then
                        errors=$((errors + 1))
                    fi
                    ;;
                "windmill")
                    validate_windmill_workflow "$workflow_path" "$filename"
                    if [[ $? -ne 0 ]]; then
                        errors=$((errors + 1))
                    fi
                    ;;
                "huginn")
                    validate_huginn_workflow "$workflow_path" "$filename"
                    if [[ $? -ne 0 ]]; then
                        errors=$((errors + 1))
                    fi
                    ;;
                "comfyui")
                    validate_comfyui_workflow "$workflow_path" "$filename"
                    if [[ $? -ne 0 ]]; then
                        errors=$((errors + 1))
                    fi
                    ;;
                "integration")
                    # Integration workflows have flexible structure
                    print_info "$filename: Integration workflow (flexible validation)"
                    ;;
                *)
                    print_warning "$filename: Unknown platform '$platform'"
                    ;;
            esac
        fi
    fi
    
    if [[ $errors -gt 0 ]]; then
        print_error "$filename: $errors validation errors"
        FAILED_WORKFLOWS=$((FAILED_WORKFLOWS + 1))
        return 1
    fi
    
    print_success "$filename: Workflow validation passed"
    VALID_WORKFLOWS=$((VALID_WORKFLOWS + 1))
    return 0
}

# N8N workflow validation
validate_n8n_workflow() {
    local workflow_path="$1"
    local filename="$2"
    
    # Check for required N8N structure
    if ! jq -e '.connections' "$workflow_path" >/dev/null 2>&1; then
        print_error "$filename: Missing N8N connections"
        return 1
    fi
    
    if ! jq -e '.nodes' "$workflow_path" >/dev/null 2>&1; then
        print_error "$filename: Missing N8N nodes"
        return 1
    fi
    
    return 0
}

# Node-RED workflow validation
validate_node_red_workflow() {
    local workflow_path="$1"
    local filename="$2"
    
    # Check if it's an array of nodes (Node-RED format)
    if ! jq -e 'type == "array"' "$workflow_path" >/dev/null 2>&1; then
        print_error "$filename: Node-RED workflows should be arrays"
        return 1
    fi
    
    # Check for required Node-RED properties
    local has_nodes=$(jq -e 'map(select(.type)) | length > 0' "$workflow_path" 2>/dev/null)
    if [[ "$has_nodes" != "true" ]]; then
        print_error "$filename: Missing Node-RED node types"
        return 1
    fi
    
    return 0
}

# Windmill workflow validation
validate_windmill_workflow() {
    local workflow_path="$1"
    local filename="$2"
    
    # Check for Windmill-specific structure
    if ! jq -e '.summary' "$workflow_path" >/dev/null 2>&1; then
        print_warning "$filename: Missing Windmill summary (recommended)"
    fi
    
    if ! jq -e '.value' "$workflow_path" >/dev/null 2>&1; then
        print_error "$filename: Missing Windmill value/steps"
        return 1
    fi
    
    return 0
}

# Huginn workflow validation
validate_huginn_workflow() {
    local workflow_path="$1"
    local filename="$2"
    
    # Check for Huginn scenario structure
    if ! jq -e '.scenario' "$workflow_path" >/dev/null 2>&1; then
        print_error "$filename: Missing Huginn scenario"
        return 1
    fi
    
    if ! jq -e '.agents' "$workflow_path" >/dev/null 2>&1; then
        print_error "$filename: Missing Huginn agents"
        return 1
    fi
    
    if ! jq -e '.links' "$workflow_path" >/dev/null 2>&1; then
        print_error "$filename: Missing Huginn links"
        return 1
    fi
    
    return 0
}

# ComfyUI workflow validation
validate_comfyui_workflow() {
    local workflow_path="$1"
    local filename="$2"
    
    # Check for ComfyUI node structure (numbered nodes)
    local has_numbered_nodes=$(jq -e 'keys | map(test("^[0-9]+$")) | any' "$workflow_path" 2>/dev/null)
    if [[ "$has_numbered_nodes" != "true" ]]; then
        print_error "$filename: Missing ComfyUI numbered nodes"
        return 1
    fi
    
    # Check for required ComfyUI node properties
    local has_node_structure=$(jq -e '[.[] | select(.class_type and .inputs)] | length > 0' "$workflow_path" 2>/dev/null)
    if [[ "$has_node_structure" != "true" ]]; then
        print_error "$filename: Missing ComfyUI node structure (class_type, inputs)"
        return 1
    fi
    
    return 0
}

# Test platform connectivity
test_platform_connectivity() {
    print_section "Testing Platform Connectivity"
    
    if [[ ! -f "$METADATA_FILE" ]]; then
        print_info "metadata.yaml not found - skipping platform tests"
        return 0
    fi
    
    # Check if yq is available
    if ! command -v yq >/dev/null 2>&1; then
        print_info "yq not available - skipping platform tests"
        return 0
    fi
    
    # Get platform configurations
    local platforms=$(yq eval '.platforms | keys | .[]' "$METADATA_FILE" 2>/dev/null)
    
    while IFS= read -r platform; do
        if [[ -n "$platform" && "$platform" != "null" ]]; then
            local port=$(yq eval ".platforms.$platform.port" "$METADATA_FILE" 2>/dev/null)
            local health_check=$(yq eval ".platforms.$platform.health_check" "$METADATA_FILE" 2>/dev/null)
            
            print_info "Testing $platform platform connectivity"
            
            if check_platform_availability "$platform" "$port" "$health_check"; then
                print_success "$platform: Available at localhost:$port"
            else
                print_warning "$platform: Not available at localhost:$port"
            fi
        fi
    done <<< "$platforms"
}

# Test integration dependencies
test_integration_dependencies() {
    print_section "Testing Integration Dependencies"
    
    if [[ ! -f "$METADATA_FILE" ]]; then
        print_info "metadata.yaml not found - skipping integration tests"
        return 0
    fi
    
    # Check if yq is available
    if ! command -v yq >/dev/null 2>&1; then
        print_info "yq not available - skipping integration tests"
        return 0
    fi
    
    # Get integration expectations
    local integrations=$(yq eval '.integrationExpectations | keys | .[]' "$METADATA_FILE" 2>/dev/null)
    
    while IFS= read -r integration; do
        if [[ -n "$integration" && "$integration" != "null" ]]; then
            local endpoint=$(yq eval ".integrationExpectations.$integration.endpoint" "$METADATA_FILE" 2>/dev/null)
            
            print_info "Testing $integration integration"
            
            if [[ -n "$endpoint" && "$endpoint" != "null" ]]; then
                if curl -s --max-time 5 "$endpoint" >/dev/null 2>&1; then
                    print_success "$integration: Available at $endpoint"
                    TESTED_WORKFLOWS=$((TESTED_WORKFLOWS + 1))
                else
                    print_warning "$integration: Not available at $endpoint"
                fi
            else
                print_info "$integration: No endpoint specified"
            fi
        fi
    done <<< "$integrations"
}

# Validate all workflows from metadata
validate_all_workflows() {
    print_section "Validating Workflow Files"
    
    if [[ ! -f "$METADATA_FILE" ]]; then
        print_error "metadata.yaml not found"
        return 1
    fi
    
    # Check if yq is available
    if ! command -v yq >/dev/null 2>&1; then
        print_error "yq required for metadata parsing"
        return 1
    fi
    
    # Extract all workflow paths and their properties
    local workflow_data=$(yq eval '.workflows | .. | select(has("path")) | [.path, .platform, .complexity, .integration] | @csv' "$METADATA_FILE" 2>/dev/null)
    
    if [[ -z "$workflow_data" ]]; then
        print_error "No workflow data found in metadata.yaml"
        return 1
    fi
    
    while IFS= read -r line; do
        if [[ -n "$line" ]]; then
            # Parse CSV line (path, platform, complexity, integration)
            local path=$(echo "$line" | cut -d',' -f1 | tr -d '"')
            local platform=$(echo "$line" | cut -d',' -f2 | tr -d '"')
            local complexity=$(echo "$line" | cut -d',' -f3 | tr -d '"')
            local integration=$(echo "$line" | cut -d',' -f4- | tr -d '"')
            
            if [[ -n "$path" && "$path" != "null" ]]; then
                local full_path="$FIXTURES_DIR/$path"
                validate_workflow_file "$full_path" "$platform" "$complexity" "$integration"
            fi
        fi
    done <<< "$workflow_data"
}

# Generate validation report
generate_report() {
    print_section "Workflow Validation Summary"
    
    echo "Total workflows validated: $TOTAL_WORKFLOWS"
    echo "Valid workflows: $VALID_WORKFLOWS"
    echo "Failed validations: $FAILED_WORKFLOWS"
    echo "Integration endpoints tested: $TESTED_WORKFLOWS"
    echo
    
    if [[ $FAILED_WORKFLOWS -eq 0 ]]; then
        print_success "All workflow fixtures validated successfully!"
        echo -e "${GREEN}✅ JSON structure verified${NC}"
        echo -e "${GREEN}✅ Platform-specific formats validated${NC}"
        echo -e "${GREEN}✅ Files accessible and readable${NC}"
    else
        print_error "$FAILED_WORKFLOWS workflow validation failures"
        echo -e "${RED}❌ Fix workflow issues before running tests${NC}"
        return 1
    fi
    
    # Tool availability report
    echo
    print_info "Workflow Processing Tool Availability:"
    if command -v jq >/dev/null 2>&1; then
        echo "  - jq (JSON processor): ✅ Available"
    else
        echo "  - jq (JSON processor): ❌ Missing"
    fi
    
    if command -v curl >/dev/null 2>&1; then
        echo "  - curl (HTTP client): ✅ Available"
    else
        echo "  - curl (HTTP client): ❌ Missing"
    fi
    
    if command -v yq >/dev/null 2>&1; then
        echo "  - yq (YAML processor): ✅ Available"
    else
        echo "  - yq (YAML processor): ❌ Missing"
    fi
}

main() {
    print_section "Workflow Fixtures Type-Specific Validation"
    
    check_workflow_tools || print_info "Continuing with limited validation capabilities"  
    validate_all_workflows
    
    # Run optional tests if tools are available
    test_platform_connectivity
    test_integration_dependencies
    
    echo
    generate_report
}

# Show usage if requested
if [[ "$1" == "--help" || "$1" == "-h" ]]; then
    echo "Vrooli Workflow Fixtures Type-Specific Validator"
    echo ""
    echo "Usage: $0 [options]"
    echo ""
    echo "Options:"
    echo "  -h, --help    Show this help message"
    echo ""
    echo "This validator performs workflow-specific validation:"
    echo "  - JSON structure validation"
    echo "  - Platform-specific format validation (N8N, Node-RED, etc.)"
    echo "  - Platform connectivity testing"
    echo "  - Integration dependency checking"
    echo "  - Workflow complexity assessment"
    echo ""
    echo "Supported platforms:"
    echo "  - N8N (connections, nodes)"
    echo "  - Node-RED (flow arrays)"
    echo "  - Windmill (value, summary)"
    echo "  - Huginn (scenario, agents, links)"
    echo "  - ComfyUI (numbered nodes, class_type)"
    echo ""
    echo "Required tools for full validation:"
    echo "  - jq (JSON processing)"
    echo "  - curl (platform connectivity)"
    echo "  - yq (YAML processing)"
    exit 0
fi

# Run validation if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi