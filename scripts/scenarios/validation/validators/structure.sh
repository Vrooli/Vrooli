#!/bin/bash
# Structure Validator - Validates scenario file and directory structure

set -euo pipefail

# Colors for output (use STRUCT_ prefix to avoid conflicts)
if [[ -z "${STRUCT_RED:-}" ]]; then
    readonly STRUCT_RED='\033[0;31m'
    readonly STRUCT_GREEN='\033[0;32m'
    readonly STRUCT_YELLOW='\033[1;33m'
    readonly STRUCT_BLUE='\033[0;34m'
    readonly STRUCT_NC='\033[0m'
fi

# Validation results
STRUCTURE_ERRORS=0
STRUCTURE_WARNINGS=0

# Print functions
print_check() {
    echo -n "  Checking $1... "
}

print_pass() {
    echo -e "${STRUCT_GREEN}✓${STRUCT_NC}"
}

print_fail() {
    echo -e "${STRUCT_RED}✗${STRUCT_NC} $1"
    STRUCTURE_ERRORS=$((STRUCTURE_ERRORS + 1))
}

print_warn() {
    echo -e "${STRUCT_YELLOW}⚠${STRUCT_NC} $1"
    STRUCTURE_WARNINGS=$((STRUCTURE_WARNINGS + 1))
}

# Check if a file exists and is valid
check_file() {
    local file_path="$1"
    local file_type="${2:-any}"
    local required="${3:-true}"
    
    print_check "$(basename "$file_path")"
    
    if [[ ! -f "$file_path" ]]; then
        if [[ "$required" == "true" ]]; then
            print_fail "File not found"
            return 1
        else
            print_warn "Optional file not found"
            return 0
        fi
    fi
    
    # Validate file type if specified
    case "$file_type" in
        yaml|yml)
            if ! validate_yaml "$file_path"; then
                print_fail "Invalid YAML"
                return 1
            fi
            ;;
        json)
            if ! validate_json "$file_path"; then
                print_fail "Invalid JSON"
                return 1
            fi
            ;;
        sql)
            if ! validate_sql "$file_path"; then
                print_warn "SQL syntax might have issues"
            fi
            ;;
        sh|bash)
            if ! validate_shell "$file_path"; then
                print_warn "Shell script might have issues"
            fi
            ;;
    esac
    
    print_pass
    return 0
}

# Check if a directory exists
check_directory() {
    local dir_path="$1"
    local required="${2:-true}"
    
    print_check "$(basename "$dir_path")/"
    
    if [[ ! -d "$dir_path" ]]; then
        if [[ "$required" == "true" ]]; then
            print_fail "Directory not found"
            return 1
        else
            print_warn "Optional directory not found"
            return 0
        fi
    fi
    
    print_pass
    return 0
}

# Validate YAML syntax
validate_yaml() {
    local file="$1"
    
    # Try python first
    if command -v python3 >/dev/null 2>&1; then
        python3 -c "import yaml; yaml.safe_load(open('$file'))" 2>/dev/null
        return $?
    fi
    
    # Try yq if available
    if command -v yq >/dev/null 2>&1; then
        yq eval '.' "$file" >/dev/null 2>&1
        return $?
    fi
    
    # Basic check - file should not be empty and have valid YAML markers
    if [[ -s "$file" ]] && grep -q '^[a-zA-Z_].*:' "$file"; then
        return 0
    fi
    
    return 1
}

# Validate JSON syntax
validate_json() {
    local file="$1"
    
    # Try jq first
    if command -v jq >/dev/null 2>&1; then
        jq '.' "$file" >/dev/null 2>&1
        return $?
    fi
    
    # Try python
    if command -v python3 >/dev/null 2>&1; then
        python3 -c "import json; json.load(open('$file'))" 2>/dev/null
        return $?
    fi
    
    # Basic check - file should start with { or [
    if [[ -s "$file" ]] && grep -q '^[[:space:]]*[{\[]' "$file"; then
        return 0
    fi
    
    return 1
}

# Validate SQL syntax (basic)
validate_sql() {
    local file="$1"
    
    # Basic SQL validation - check for common syntax patterns
    if ! grep -qiE '(CREATE|INSERT|SELECT|UPDATE|DELETE|ALTER|DROP)' "$file"; then
        return 1
    fi
    
    # Check for unclosed quotes
    local single_quotes=$(grep -o "'" "$file" | wc -l)
    local double_quotes=$(grep -o '"' "$file" | wc -l)
    
    if [[ $((single_quotes % 2)) -ne 0 ]] || [[ $((double_quotes % 2)) -ne 0 ]]; then
        return 1
    fi
    
    return 0
}

# Validate shell script syntax
validate_shell() {
    local file="$1"
    
    # Use shellcheck if available
    if command -v shellcheck >/dev/null 2>&1; then
        shellcheck -S error "$file" >/dev/null 2>&1
        return $?
    fi
    
    # Use bash -n for syntax check
    bash -n "$file" 2>/dev/null
    return $?
}

# Parse required files from config
get_required_files() {
    local config_file="$1"
    
    # Try to extract required files from YAML
    # This is a simplified parser - in production use proper YAML parser
    if [[ -f "$config_file" ]]; then
        # Use awk to parse only the required_files section
        awk '
            /^structure:/ { in_structure=1; next }
            in_structure && /^[[:space:]]*required_files:/ { in_files=1; next }
            in_structure && in_files && /^[[:space:]]*required_dirs:/ { in_files=0; next }
            in_structure && in_files && /^[[:space:]]*[a-z_]+:/ { in_files=0; next }
            in_structure && /^[a-z_]+:/ && !/required_files:/ { in_structure=0; in_files=0; next }
            in_structure && in_files && /^[[:space:]]*-/ { 
                gsub(/^[[:space:]]*-[[:space:]]*/, ""); 
                print 
            }
        ' "$config_file"
    fi
}

# Parse required directories from config
get_required_dirs() {
    local config_file="$1"
    
    if [[ -f "$config_file" ]]; then
        # Use awk to parse only the required_dirs section
        awk '
            /^structure:/ { in_structure=1; next }
            in_structure && /^[[:space:]]*required_dirs:/ { in_dirs=1; next }
            in_structure && in_dirs && /^[[:space:]]*[a-z_]+:/ { in_dirs=0; next }
            in_structure && /^[a-z_]+:/ && !/required_dirs:/ { in_structure=0; in_dirs=0; next }
            in_structure && in_dirs && /^[[:space:]]*-/ { 
                gsub(/^[[:space:]]*-[[:space:]]*/, ""); 
                print 
            }
        ' "$config_file"
    fi
}

# Validate scenario metadata.yaml
validate_metadata() {
    local scenario_dir="$1"
    local metadata_file="$scenario_dir/metadata.yaml"
    
    if [[ ! -f "$metadata_file" ]]; then
        print_fail "metadata.yaml not found"
        return 1
    fi
    
    # Check required fields in metadata
    local scenario_fields=("name" "description" "version")
    local root_fields=("categories")
    
    # Check fields under scenario: section
    for field in "${scenario_fields[@]}"; do
        local field_found=false
        
        # Check if field exists under scenario section
        if grep -A20 "^scenario:" "$metadata_file" 2>/dev/null | grep -q "^[[:space:]]\+${field}:" 2>/dev/null; then
            field_found=true
        fi
        
        if [[ "$field_found" == "false" ]]; then
            print_warn "Missing field in metadata.yaml: scenario.$field"
        fi
    done
    
    # Check fields at root level  
    for field in "${root_fields[@]}"; do
        local field_found=false
        
        if grep -q "^[[:space:]]*${field}:" "$metadata_file" 2>/dev/null; then
            field_found=true
        fi
        
        if [[ "$field_found" == "false" ]]; then
            print_warn "Missing field in metadata.yaml: $field"
        fi
    done
    
    return 0
}

# Validate scenario manifest.yaml
validate_manifest() {
    local scenario_dir="$1"
    local manifest_file="$scenario_dir/manifest.yaml"
    
    if [[ ! -f "$manifest_file" ]]; then
        print_fail "manifest.yaml not found"
        return 1
    fi
    
    # Check for deployment sequence
    if ! grep -q "deployment:" "$manifest_file"; then
        print_warn "No deployment section in manifest.yaml"
    fi
    
    return 0
}

# Main validation function
validate_scenario_structure() {
    local scenario_dir="$1"
    local config_file="${2:-$scenario_dir/scenario-test.yaml}"
    
    echo "  ┌─ Structure Validation ─────────────────────────────┐"
    
    # Core files that every scenario should have
    local core_files=(
        "metadata.yaml"
        "manifest.yaml"
        "README.md"
    )
    
    # Check core files with appropriate types
    check_file "$scenario_dir/metadata.yaml" "yaml"
    check_file "$scenario_dir/manifest.yaml" "yaml"
    check_file "$scenario_dir/README.md" "any"
    
    # Core directories
    local core_dirs=(
        "initialization"
        "deployment"
    )
    
    # Check core directories
    for dir in "${core_dirs[@]}"; do
        check_directory "$scenario_dir/$dir"
    done
    
    # Check for specific initialization subdirectories
    if [[ -d "$scenario_dir/initialization" ]]; then
        # Common initialization subdirectories
        check_directory "$scenario_dir/initialization/database" false
        check_directory "$scenario_dir/initialization/workflows" false
        check_directory "$scenario_dir/initialization/configuration" false
    fi
    
    # Check deployment scripts
    if [[ -d "$scenario_dir/deployment" ]]; then
        check_file "$scenario_dir/deployment/startup.sh" "sh" false
        check_file "$scenario_dir/deployment/monitor.sh" "sh" false
    fi
    
    # Check for UI components if directory exists
    if [[ -d "$scenario_dir/ui" ]]; then
        print_check "ui/"
        print_pass
    fi
    
    # Parse and check additional required files from config
    if [[ -f "$config_file" ]]; then
        local required_files
        readarray -t required_files < <(get_required_files "$config_file")
        
        for file in "${required_files[@]}"; do
            if [[ -n "$file" ]] && [[ ! " ${core_files[@]} " =~ " ${file} " ]]; then
                check_file "$scenario_dir/$file" "any"
            fi
        done
        
        local required_dirs
        readarray -t required_dirs < <(get_required_dirs "$config_file")
        
        for dir in "${required_dirs[@]}"; do
            if [[ -n "$dir" ]] && [[ ! " ${core_dirs[@]} " =~ " ${dir} " ]]; then
                check_directory "$scenario_dir/$dir"
            fi
        done
    fi
    
    # Validate metadata content
    validate_metadata "$scenario_dir"
    
    # Validate manifest content
    validate_manifest "$scenario_dir"
    
    echo "  └────────────────────────────────────────────────────┘"
    
    # Report results
    if [[ $STRUCTURE_ERRORS -gt 0 ]]; then
        echo -e "  ${STRUCT_RED}Structure validation failed with $STRUCTURE_ERRORS error(s)${STRUCT_NC}"
        return 1
    elif [[ $STRUCTURE_WARNINGS -gt 0 ]]; then
        echo -e "  ${STRUCT_YELLOW}Structure validation passed with $STRUCTURE_WARNINGS warning(s)${STRUCT_NC}"
        return 0
    else
        echo -e "  ${STRUCT_GREEN}Structure validation passed${STRUCT_NC}"
        return 0
    fi
}

# Execute test handler for structure type
execute_structure_test() {
    local test_name="$1"
    local test_data="$2"
    
    # For structure tests, test_data is the config file path
    local scenario_dir="$(dirname "$test_data")"
    
    validate_scenario_structure "$scenario_dir" "$test_data"
    return $?
}

# Export functions for use by test runner
export -f validate_scenario_structure
export -f execute_structure_test
