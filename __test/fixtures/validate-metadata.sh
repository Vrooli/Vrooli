#!/bin/bash
# Vrooli Fixture Metadata Central Validator
# Validates all fixture metadata files follow the common schema format

set -e

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/__test/fixtures"
FIXTURES_DIR="$SCRIPT_DIR"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Counters
TOTAL_COLLECTIONS=0
VALID_COLLECTIONS=0
FAILED_COLLECTIONS=0
TOTAL_FIXTURES=0
VALID_FIXTURES=0

# Expected fixture collection types
EXPECTED_TYPES=("images" "audio" "documents" "workflows")

print_header() {
    echo -e "${BLUE}============================================${NC}"
    echo -e "${BLUE} Vrooli Fixture Metadata Central Validator${NC}"
    echo -e "${BLUE}============================================${NC}"
    echo
}

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

# Check if yq is available for YAML parsing
check_yaml_parser() {
    if ! command -v yq >/dev/null 2>&1; then
        print_error "yq (YAML parser) is required but not installed"
        print_info "Install with: sudo snap install yq, brew install yq, or pip install yq"
        exit 1
    fi
}

# Validate a single metadata file's schema
validate_metadata_schema() {
    local metadata_file="$1"
    local collection_dir="$2"
    local collection_name
    collection_name=$(basename "$collection_dir")
    
    TOTAL_COLLECTIONS=$((TOTAL_COLLECTIONS + 1))
    
    # Check file exists
    if [[ ! -f "$metadata_file" ]]; then
        print_error "$collection_name: metadata.yaml not found"
        FAILED_COLLECTIONS=$((FAILED_COLLECTIONS + 1))
        return 1
    fi
    
    # Check YAML syntax
    if ! yq eval '.' "$metadata_file" >/dev/null 2>&1; then
        print_error "$collection_name: Invalid YAML syntax"
        FAILED_COLLECTIONS=$((FAILED_COLLECTIONS + 1))
        return 1
    fi
    
    # Validate required top-level fields
    local version
    version=$(yq eval '.version' "$metadata_file" 2>/dev/null)
    local type=$(yq eval '.type' "$metadata_file" 2>/dev/null)
    local description=$(yq eval '.description' "$metadata_file" 2>/dev/null)
    local schema_section=$(yq eval '.schema' "$metadata_file" 2>/dev/null)
    local test_suites=$(yq eval '.testSuites' "$metadata_file" 2>/dev/null)
    
    local errors=0
    
    # Validate version
    if [[ "$version" != "1.0" ]]; then
        print_error "$collection_name: Invalid or missing version (expected: 1.0, got: $version)"
        errors=$((errors + 1))
    fi
    
    # Validate type
    local valid_type=false
    for expected_type in "${EXPECTED_TYPES[@]}"; do
        if [[ "$type" == "$expected_type" ]]; then
            valid_type=true
            break
        fi
    done
    
    if [[ "$valid_type" != true ]]; then
        print_error "$collection_name: Invalid type '$type' (expected: ${EXPECTED_TYPES[*]})"
        errors=$((errors + 1))
    fi
    
    # Validate type matches directory name
    if [[ "$type" != "$collection_name" ]]; then
        print_error "$collection_name: Type '$type' doesn't match directory name '$collection_name'"
        errors=$((errors + 1))
    fi
    
    # Validate description
    if [[ "$description" == "null" || -z "$description" ]]; then
        print_error "$collection_name: Missing description field"
        errors=$((errors + 1))
    fi
    
    # Validate schema section
    if [[ "$schema_section" == "null" ]]; then
        print_error "$collection_name: Missing schema section"
        errors=$((errors + 1))
    else
        # Check for commonProperties and typeProperties
        local common_props=$(yq eval '.schema.commonProperties' "$metadata_file" 2>/dev/null)
        local type_props=$(yq eval '.schema.typeProperties' "$metadata_file" 2>/dev/null)
        
        if [[ "$common_props" == "null" ]]; then
            print_error "$collection_name: Missing schema.commonProperties"
            errors=$((errors + 1))
        else
            # Validate required common properties
            local has_path=$(yq eval '.schema.commonProperties | contains(["path"])' "$metadata_file" 2>/dev/null)
            local has_tags=$(yq eval '.schema.commonProperties | contains(["tags"])' "$metadata_file" 2>/dev/null)
            local has_testdata=$(yq eval '.schema.commonProperties | contains(["testData"])' "$metadata_file" 2>/dev/null)
            
            if [[ "$has_path" != "true" ]]; then
                print_error "$collection_name: schema.commonProperties missing required 'path'"
                errors=$((errors + 1))
            fi
            if [[ "$has_tags" != "true" ]]; then
                print_error "$collection_name: schema.commonProperties missing required 'tags'"
                errors=$((errors + 1))
            fi
            if [[ "$has_testdata" != "true" ]]; then
                print_error "$collection_name: schema.commonProperties missing required 'testData'"
                errors=$((errors + 1))
            fi
        fi
        
        if [[ "$type_props" == "null" ]]; then
            print_warning "$collection_name: Missing schema.typeProperties (optional but recommended)"
        fi
    fi
    
    # Validate main collection section exists
    local collection_section=$(yq eval ".$type" "$metadata_file" 2>/dev/null)
    if [[ "$collection_section" == "null" ]]; then
        print_error "$collection_name: Missing main collection section '$type'"
        errors=$((errors + 1))
    fi
    
    # Validate testSuites section
    if [[ "$test_suites" == "null" ]]; then
        print_error "$collection_name: Missing testSuites section"
        errors=$((errors + 1))
    fi
    
    if [[ $errors -gt 0 ]]; then
        print_error "$collection_name: $errors schema validation errors"
        FAILED_COLLECTIONS=$((FAILED_COLLECTIONS + 1))
        return 1
    fi
    
    print_success "$collection_name: Schema validation passed"
    VALID_COLLECTIONS=$((VALID_COLLECTIONS + 1))
    return 0
}

# Validate fixture paths exist
validate_fixture_paths() {
    local metadata_file="$1"
    local collection_dir="$2" 
    local collection_name=$(basename "$collection_dir")
    
    print_info "$collection_name: Validating fixture paths..."
    
    local fixture_count=0
    local valid_paths=0
    local errors=0
    
    # Get the collection type to know which section to check
    local type=$(yq eval '.type' "$metadata_file" 2>/dev/null)
    
    # Extract all fixture paths from the collection
    local fixture_paths
    if [[ "$type" == "workflows" ]]; then
        # Workflows are organized by platform
        fixture_paths=$(yq eval ".$type | .. | select(has(\"path\")) | .path" "$metadata_file" 2>/dev/null | grep -v "null" || true)
    else
        # Other types have nested categorization
        fixture_paths=$(yq eval ".$type | .. | select(has(\"path\")) | .path" "$metadata_file" 2>/dev/null | grep -v "null" || true)
    fi
    
    if [[ -n "$fixture_paths" ]]; then
        while IFS= read -r path; do
            if [[ -n "$path" && "$path" != "null" ]]; then
                fixture_count=$((fixture_count + 1))
                local full_path="$collection_dir/$path"
                
                if [[ -f "$full_path" ]]; then
                    valid_paths=$((valid_paths + 1))
                else
                    print_error "$collection_name: Fixture not found: $path"
                    errors=$((errors + 1))
                fi
            fi
        done <<< "$fixture_paths"
    fi
    
    TOTAL_FIXTURES=$((TOTAL_FIXTURES + fixture_count))
    VALID_FIXTURES=$((VALID_FIXTURES + valid_paths))
    
    if [[ $errors -gt 0 ]]; then
        print_error "$collection_name: $errors missing fixture files"
        return 1
    fi
    
    print_success "$collection_name: All $fixture_count fixture paths validated"
    return 0
}

# Validate test suite references
validate_test_suites() {
    local metadata_file="$1"
    local collection_dir="$2"
    local collection_name=$(basename "$collection_dir")
    
    print_info "$collection_name: Validating test suite references..."
    
    local errors=0
    local suite_count=0
    
    # Get all test suite fixture references
    local suite_fixtures=$(yq eval '.testSuites | .. | select(type == "!!str")' "$metadata_file" 2>/dev/null | grep -v "null" || true)
    
    if [[ -n "$suite_fixtures" ]]; then
        while IFS= read -r fixture_path; do
            if [[ -n "$fixture_path" && "$fixture_path" != "null" ]]; then
                suite_count=$((suite_count + 1))
                local full_path="$collection_dir/$fixture_path"
                
                if [[ ! -f "$full_path" ]]; then
                    print_error "$collection_name: Test suite references missing fixture: $fixture_path"
                    errors=$((errors + 1))
                fi
            fi
        done <<< "$suite_fixtures"
    fi
    
    if [[ $errors -gt 0 ]]; then
        print_error "$collection_name: $errors invalid test suite references"
        return 1
    fi
    
    if [[ $suite_count -gt 0 ]]; then
        print_success "$collection_name: All $suite_count test suite references validated"
    else
        print_warning "$collection_name: No test suite fixtures found"
    fi
    
    return 0
}

# Run type-specific validator if available
run_type_validator() {
    local collection_dir="$1"
    local collection_name=$(basename "$collection_dir")
    local type_validator="$collection_dir/validate-$collection_name.sh"
    
    if [[ -f "$type_validator" && -x "$type_validator" ]]; then
        print_info "$collection_name: Running type-specific validator..."
        
        if "$type_validator" >/dev/null 2>&1; then
            print_success "$collection_name: Type-specific validation passed"
            return 0
        else
            print_error "$collection_name: Type-specific validation failed"
            return 1
        fi
    else
        print_warning "$collection_name: No type-specific validator found (validate-$collection_name.sh)"
        return 0
    fi
}

# Validate a single collection
validate_collection() {
    local collection_dir="$1"
    local collection_name=$(basename "$collection_dir")
    local metadata_file="$collection_dir/metadata.yaml"
    
    print_section "Validating $collection_name Collection"
    
    local validation_failed=false
    
    # Schema validation
    if ! validate_metadata_schema "$metadata_file" "$collection_dir"; then
        validation_failed=true
    fi
    
    # Path validation (only if schema passed)
    if [[ "$validation_failed" != true ]]; then
        if ! validate_fixture_paths "$metadata_file" "$collection_dir"; then
            validation_failed=true
        fi
        
        # Test suite validation
        if ! validate_test_suites "$metadata_file" "$collection_dir"; then
            validation_failed=true
        fi
        
        # Type-specific validation
        if ! run_type_validator "$collection_dir"; then
            validation_failed=true
        fi
    fi
    
    if [[ "$validation_failed" == true ]]; then
        print_error "$collection_name: Collection validation failed"
        return 1
    fi
    
    print_success "$collection_name: Collection validation passed"
    return 0
}

# Discover and validate all fixture collections
validate_all_collections() {
    print_section "Discovering Fixture Collections"
    
    local collections_found=()
    
    # Find all directories with metadata.yaml files
    for dir in "$FIXTURES_DIR"/*; do
        if [[ -d "$dir" && -f "$dir/metadata.yaml" ]]; then
            local collection_name=$(basename "$dir")
            collections_found+=("$collection_name")
            print_info "Found collection: $collection_name"
        fi
    done
    
    if [[ ${#collections_found[@]} -eq 0 ]]; then
        print_error "No fixture collections found with metadata.yaml files"
        return 1
    fi
    
    echo
    
    # Validate each collection
    local overall_success=true
    for collection in "${collections_found[@]}"; do
        if ! validate_collection "$FIXTURES_DIR/$collection"; then
            overall_success=false
        fi
        echo
    done
    
    return $([[ "$overall_success" == true ]])
}

# Generate validation report
generate_report() {
    print_section "Validation Summary"
    
    echo "Collections examined: $TOTAL_COLLECTIONS"
    echo "Collections valid: $VALID_COLLECTIONS"  
    echo "Collections failed: $FAILED_COLLECTIONS"
    echo "Total fixtures validated: $TOTAL_FIXTURES"
    echo "Valid fixture paths: $VALID_FIXTURES"
    echo
    
    if [[ $FAILED_COLLECTIONS -eq 0 ]]; then
        print_success "All fixture collections validated successfully!"
        echo -e "${GREEN}✅ Metadata schema compliance verified${NC}"
        echo -e "${GREEN}✅ All fixture paths validated${NC}"
        echo -e "${GREEN}✅ Test suite integrity confirmed${NC}"
    else
        print_error "$FAILED_COLLECTIONS collection(s) failed validation"
        echo -e "${RED}❌ Metadata schema issues detected${NC}"
        echo -e "${RED}❌ Fix errors before running tests${NC}"
        exit 1
    fi
    
    # Schema compliance statistics
    echo
    print_info "Schema Compliance Statistics:"
    local compliance_rate=0
    if [[ $TOTAL_COLLECTIONS -gt 0 ]]; then
        compliance_rate=$((VALID_COLLECTIONS * 100 / TOTAL_COLLECTIONS))
    fi
    echo "  - Schema compliance rate: ${compliance_rate}%"
    echo "  - Average fixtures per collection: $((TOTAL_FIXTURES / TOTAL_COLLECTIONS))"
    
    # Type-specific validator availability
    echo
    print_info "Type-Specific Validator Availability:"
    for expected_type in "${EXPECTED_TYPES[@]}"; do
        local validator_path="$FIXTURES_DIR/$expected_type/validate-$expected_type.sh"
        if [[ -f "$validator_path" && -x "$validator_path" ]]; then
            echo "  - $expected_type: ✅ Available"
        else
            echo "  - $expected_type: ❌ Missing"
        fi
    done
}

main() {
    print_header
    
    # Check prerequisites
    check_yaml_parser
    
    # Run validation
    if validate_all_collections; then
        generate_report
    else
        generate_report
        exit 1
    fi
}

# Show usage if requested
if [[ "$1" == "--help" || "$1" == "-h" ]]; then
    echo "Vrooli Fixture Metadata Central Validator"
    echo ""
    echo "Usage: $0 [options]"
    echo ""
    echo "Options:"
    echo "  -h, --help    Show this help message"
    echo ""
    echo "This validator checks that all fixture collections follow the"
    echo "standardized metadata schema defined in METADATA_SCHEMA.md"
    echo ""
    echo "Validation includes:"
    echo "  - Schema compliance (required fields, types, structure)"
    echo "  - Path validation (all fixture files exist)"
    echo "  - Test suite integrity (all references are valid)"
    echo "  - Type-specific validation (if validator scripts exist)"
    exit 0
fi

# Run validation if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi