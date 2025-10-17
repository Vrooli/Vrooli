#!/bin/bash
# Validation script for accessibility-compliance-hub scenario
#
# Purpose: Run comprehensive quality checks before committing changes
#
# Usage:
#   make validate              # Recommended way
#   bash scripts/validate.sh   # Direct execution
#
# Checks performed:
#   - No compiled binaries present
#   - Test artifacts properly gitignored
#   - Configuration files valid (service.json)
#   - Required files exist
#   - CLI is executable
#   - Go code compiles successfully
#   - All tests pass
#   - .gitignore coverage complete
#   - Makefile has required targets
#
# Exit codes:
#   0 - All checks passed
#   1 - One or more checks failed
#
# Author: Vrooli AI
# Last Updated: 2025-10-05

set -euo pipefail

# Colors
readonly GREEN='\033[0;32m'
readonly RED='\033[0;31m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m'

# Get scenario root
SCENARIO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$SCENARIO_ROOT"

echo -e "${BLUE}🔍 Validating accessibility-compliance-hub scenario...${NC}\n"

# Track validation results
CHECKS_PASSED=0
CHECKS_FAILED=0
WARNINGS=0

check_pass() {
    echo -e "${GREEN}✓${NC} $1"
    ((CHECKS_PASSED++)) || true
}

check_fail() {
    echo -e "${RED}✗${NC} $1"
    ((CHECKS_FAILED++)) || true
}

check_warn() {
    echo -e "${YELLOW}⚠${NC} $1"
    ((WARNINGS++)) || true
}

# 1. Check for compiled binaries that shouldn't be committed
echo -e "${YELLOW}Checking for uncommitted binaries...${NC}"
if [ -f "api/accessibility-compliance-hub-api" ]; then
    check_fail "Compiled binary found: api/accessibility-compliance-hub-api (run 'make clean')"
else
    check_pass "No compiled API binaries found"
fi

# 2. Check for test artifacts
echo -e "\n${YELLOW}Checking for test artifacts...${NC}"
if [ -f "api/coverage.html" ]; then
    check_warn "Test artifact found: api/coverage.html (automatically regenerated, but gitignored)"
else
    check_pass "No stray test artifacts (coverage.html)"
fi

# 3. Validate service.json
echo -e "\n${YELLOW}Validating configuration files...${NC}"
if jq empty .vrooli/service.json 2>/dev/null; then
    check_pass "service.json is valid JSON"
else
    check_fail "service.json is invalid JSON"
fi

# 4. Check required files exist
echo -e "\n${YELLOW}Checking required files...${NC}"
required_files=(
    ".vrooli/service.json"
    "PRD.md"
    "README.md"
    "PROBLEMS.md"
    "Makefile"
    "api/main.go"
    "api/go.mod"
    "cli/accessibility-compliance-hub"
    "cli/install.sh"
    "initialization/storage/postgres/schema.sql"
    "initialization/automation/n8n/accessibility-audit.json"
)

for file in "${required_files[@]}"; do
    if [ -f "$file" ]; then
        check_pass "$file exists"
    else
        check_fail "$file is missing"
    fi
done

# 5. Check CLI is executable
echo -e "\n${YELLOW}Checking file permissions...${NC}"
if [ -x "cli/accessibility-compliance-hub" ]; then
    check_pass "CLI is executable"
else
    check_fail "CLI is not executable (run: chmod +x cli/accessibility-compliance-hub)"
fi

# 6. Verify Go code compiles
echo -e "\n${YELLOW}Checking Go code compilation...${NC}"
if cd api && go build -o /dev/null main.go 2>/dev/null; then
    check_pass "Go code compiles successfully"
    cd "$SCENARIO_ROOT"
else
    check_fail "Go code fails to compile"
    cd "$SCENARIO_ROOT"
fi

# 7. Run Go tests
echo -e "\n${YELLOW}Running Go tests...${NC}"
if cd api && go test -short ./... >/dev/null 2>&1; then
    check_pass "Go tests pass"
    cd "$SCENARIO_ROOT"
else
    check_fail "Go tests fail"
    cd "$SCENARIO_ROOT"
fi

# 8. Check for proper .gitignore coverage
echo -e "\n${YELLOW}Validating .gitignore coverage...${NC}"
if grep -q "api/.*-api" .gitignore && grep -q "coverage" .gitignore; then
    check_pass ".gitignore protects binaries and test artifacts"
else
    check_fail ".gitignore incomplete"
fi

# 9. Validate Makefile targets
echo -e "\n${YELLOW}Checking Makefile targets...${NC}"
required_targets=("help" "start" "stop" "test" "logs" "status" "clean")
for target in "${required_targets[@]}"; do
    if grep -q "^${target}:" Makefile; then
        check_pass "Makefile has '${target}' target"
    else
        check_fail "Makefile missing '${target}' target"
    fi
done

# Print summary
echo -e "\n${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}Validation Summary${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}Passed:${NC}  $CHECKS_PASSED"
echo -e "${RED}Failed:${NC}  $CHECKS_FAILED"
echo -e "${YELLOW}Warnings:${NC} $WARNINGS"

if [ $CHECKS_FAILED -eq 0 ]; then
    echo -e "\n${GREEN}✓ All validation checks passed!${NC}"
    if [ $WARNINGS -gt 0 ]; then
        echo -e "${YELLOW}⚠ ${WARNINGS} warning(s) - review recommended but not blocking${NC}"
    fi
    exit 0
else
    echo -e "\n${RED}✗ ${CHECKS_FAILED} validation check(s) failed!${NC}"
    echo -e "${YELLOW}Fix the issues above before committing.${NC}"
    exit 1
fi
