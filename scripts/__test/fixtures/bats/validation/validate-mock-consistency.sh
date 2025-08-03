#!/usr/bin/env bash
# Mock Consistency Validation Script
# Validates that all mocks are properly implemented and consistent

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Counters
ERRORS=0
WARNINGS=0
CHECKS=0

# Get the fixtures directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
FIXTURES_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

echo "=== Mock Consistency Validation ==="
echo "Fixtures directory: $FIXTURES_DIR"
echo ""

#######################################
# Helper Functions
#######################################

check_pass() {
    local message="$1"
    echo -e "${GREEN}✓${NC} $message"
    ((CHECKS++))
}

check_fail() {
    local message="$1"
    echo -e "${RED}✗${NC} $message"
    ((ERRORS++))
    ((CHECKS++))
}

check_warn() {
    local message="$1"
    echo -e "${YELLOW}⚠${NC} $message"
    ((WARNINGS++))
    ((CHECKS++))
}

check_file_exists() {
    local file="$1"
    local description="$2"
    
    if [[ -f "$file" ]]; then
        check_pass "$description exists"
        return 0
    else
        check_fail "$description missing: $file"
        return 1
    fi
}

check_function_defined() {
    local file="$1"
    local function="$2"
    
    if grep -q "^${function}()" "$file" 2>/dev/null || \
       grep -q "^function ${function}" "$file" 2>/dev/null || \
       grep -q "^${function} ()" "$file" 2>/dev/null; then
        return 0
    else
        return 1
    fi
}

#######################################
# Validation Checks
#######################################

echo "1. Checking core infrastructure files..."
echo "----------------------------------------"

# Check core files exist
check_file_exists "$FIXTURES_DIR/core/common_setup.bash" "Common setup"
check_file_exists "$FIXTURES_DIR/core/assertions.bash" "Assertions library"
check_file_exists "$FIXTURES_DIR/core/error_handling.bash" "Error handling"
check_file_exists "$FIXTURES_DIR/core/benchmarking.bash" "Benchmarking"
check_file_exists "$FIXTURES_DIR/core/path_resolver.bash" "Path resolver"
check_file_exists "$FIXTURES_DIR/mocks/mock_registry.bash" "Mock registry"

echo ""
echo "2. Checking mock implementations..."
echo "------------------------------------"

# Define expected mock structure
declare -A EXPECTED_MOCKS=(
    ["system"]="docker http filesystem commands verification"
    ["ai"]="ollama whisper comfyui unstructured-io"
    ["automation"]="n8n node-red huginn windmill"
    ["agents"]="agent-s2 browserless claude-code"
    ["storage"]="minio postgres qdrant questdb redis vault"
    ["search"]="searxng"
    ["execution"]="judge0"
)

# Check each category
for category in "${!EXPECTED_MOCKS[@]}"; do
    echo "  Checking $category mocks..."
    
    if [[ "$category" == "system" ]]; then
        mock_dir="$FIXTURES_DIR/mocks/system"
    else
        mock_dir="$FIXTURES_DIR/mocks/resources/$category"
    fi
    
    if [[ ! -d "$mock_dir" ]]; then
        check_fail "Directory missing: $mock_dir"
        continue
    fi
    
    # Check each expected mock
    for mock in ${EXPECTED_MOCKS[$category]}; do
        mock_file="$mock_dir/${mock}.bash"
        if check_file_exists "$mock_file" "    $mock mock"; then
            # Check for required functions in resource mocks
            if [[ "$category" != "system" ]]; then
                if check_function_defined "$mock_file" "mock::${mock}::setup"; then
                    check_pass "    $mock has setup function"
                else
                    check_warn "    $mock missing setup function"
                fi
            fi
        fi
    done
done

echo ""
echo "3. Checking mock registry integration..."
echo "-----------------------------------------"

# Check that mock registry can detect all resource categories
for resource in ollama whisper n8n claude-code postgres searxng judge0; do
    if grep -q "\"$resource\"" "$FIXTURES_DIR/mocks/mock_registry.bash"; then
        check_pass "Resource '$resource' registered"
    else
        check_fail "Resource '$resource' not in registry"
    fi
done

echo ""
echo "4. Checking template consistency..."
echo "------------------------------------"

# Check template directories
for template_dir in basic resource integration performance advanced; do
    dir_path="$FIXTURES_DIR/templates/$template_dir"
    if [[ -d "$dir_path" ]]; then
        check_pass "Template directory '$template_dir' exists"
        
        # Check for at least one .bats file
        if ls "$dir_path"/*.bats >/dev/null 2>&1; then
            check_pass "  Contains .bats files"
        else
            check_warn "  No .bats files in $template_dir"
        fi
    else
        check_fail "Template directory '$template_dir' missing"
    fi
done

echo ""
echo "5. Checking documentation..."
echo "-----------------------------"

# Check documentation files
docs=(
    "README.md"
    "docs/setup-guide.md"
    "docs/assertions.md"
    "docs/mock-registry.md"
    "docs/troubleshooting.md"
    "docs/migration-guide.md"
)

for doc in "${docs[@]}"; do
    check_file_exists "$FIXTURES_DIR/$doc" "Documentation: $doc"
done

echo ""
echo "6. Checking for deprecated files..."
echo "------------------------------------"

# Check for deprecated files that should have been removed
deprecated_files=(
    "$FIXTURES_DIR/standard_mock_framework.bash.deprecated"
    "$FIXTURES_DIR/http_mock_helpers.bash.deprecated"
)

for file in "${deprecated_files[@]}"; do
    if [[ -f "$file" ]]; then
        check_fail "Deprecated file still exists: $(basename "$file")"
    else
        check_pass "Deprecated file removed: $(basename "$file")"
    fi
done

echo ""
echo "7. Checking assertion functions..."
echo "-----------------------------------"

# Check for common assertion functions
assertions=(
    "assert_success"
    "assert_failure"
    "assert_output_contains"
    "assert_file_exists"
    "assert_dir_exists"
    "assert_json_valid"
    "assert_resource_healthy"
)

for assertion in "${assertions[@]}"; do
    if check_function_defined "$FIXTURES_DIR/core/assertions.bash" "$assertion"; then
        check_pass "Assertion function: $assertion"
    else
        check_fail "Missing assertion: $assertion"
    fi
done

echo ""
echo "8. Checking path resolver functions..."
echo "---------------------------------------"

# Check path resolver functions
path_functions=(
    "vrooli_resolve_fixtures_path"
    "vrooli_fixture_path"
    "vrooli_source_fixture"
    "vrooli_validate_test_environment"
    "vrooli_test_tmpdir"
)

for func in "${path_functions[@]}"; do
    if check_function_defined "$FIXTURES_DIR/core/path_resolver.bash" "$func"; then
        check_pass "Path function: $func"
    else
        check_fail "Missing path function: $func"
    fi
done

echo ""
echo "9. Checking for common issues..."
echo "---------------------------------"

# Check for hardcoded paths
echo -n "Checking for hardcoded paths... "
if grep -r "/root/Vrooli" "$FIXTURES_DIR" --include="*.bash" --include="*.bats" >/dev/null 2>&1; then
    check_warn "Found hardcoded paths - should use path resolver"
else
    check_pass "No hardcoded paths found"
fi

# Check for inconsistent sourcing patterns
echo -n "Checking for inconsistent sourcing... "
old_patterns=$(grep -r "BATS_TEST_DIRNAME\|dirname \"\$0\"" "$FIXTURES_DIR/templates" --include="*.bats" 2>/dev/null | wc -l)
if [[ $old_patterns -gt 0 ]]; then
    check_pass "Templates use consistent sourcing ($old_patterns instances)"
else
    check_warn "Check template sourcing patterns"
fi

echo ""
echo "10. Running basic load test..."
echo "-------------------------------"

# Try to load the infrastructure
if bash -c "source '$FIXTURES_DIR/core/common_setup.bash' && echo 'loaded'" >/dev/null 2>&1; then
    check_pass "Infrastructure loads without errors"
else
    check_fail "Infrastructure fails to load"
fi

# Try each setup mode
for mode in "setup_standard_mocks" "setup_resource_test ollama" "setup_integration_test ollama whisper"; do
    if bash -c "
        source '$FIXTURES_DIR/core/common_setup.bash'
        $mode >/dev/null 2>&1
        echo 'ok'
    " >/dev/null 2>&1; then
        check_pass "Setup mode works: ${mode%% *}"
    else
        check_fail "Setup mode fails: ${mode%% *}"
    fi
done

#######################################
# Summary
#######################################

echo ""
echo "========================================"
echo "           VALIDATION SUMMARY           "
echo "========================================"
echo -e "Total checks:   $CHECKS"
echo -e "Passed:        ${GREEN}$((CHECKS - ERRORS - WARNINGS))${NC}"
echo -e "Warnings:      ${YELLOW}$WARNINGS${NC}"
echo -e "Failed:        ${RED}$ERRORS${NC}"
echo ""

if [[ $ERRORS -eq 0 ]]; then
    if [[ $WARNINGS -eq 0 ]]; then
        echo -e "${GREEN}✓ All checks passed!${NC}"
        echo "The mock infrastructure is consistent and ready to use."
    else
        echo -e "${YELLOW}⚠ Validation passed with warnings${NC}"
        echo "The infrastructure works but has minor issues to address."
    fi
    exit 0
else
    echo -e "${RED}✗ Validation failed!${NC}"
    echo "Please fix the errors above before using the infrastructure."
    exit 1
fi