#!/bin/bash

# Test Genie Structural Validation
# Ensures required directories, files, and lifecycle assets exist

set -euo pipefail

SCENARIO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

PASS=0
FAIL=0

check_path() {
    local type="$1"
    local target="$2"
    local description="$3"

    if [[ "$type" == "dir" && -d "${SCENARIO_ROOT}/${target}" ]]; then
        echo -e "${GREEN}✅ ${description}${NC}"
        PASS=$((PASS + 1))
    elif [[ "$type" == "file" && -f "${SCENARIO_ROOT}/${target}" ]]; then
        echo -e "${GREEN}✅ ${description}${NC}"
        PASS=$((PASS + 1))
    else
        echo -e "${RED}❌ ${description}${NC}"
        echo -e "   Missing: ${target}"
        FAIL=$((FAIL + 1))
    fi
}

echo -e "${BLUE}📁 Verifying Test Genie structure${NC}"

# Core directories
check_path "dir" "api" "API source directory present"
check_path "dir" "cli" "CLI directory present"
check_path "dir" "ui" "UI directory present"
check_path "dir" "prompts" "Prompts directory present"
check_path "dir" "initialization" "Initialization assets present"
check_path "dir" "test/phases" "Phase-based test directory present"

# Required files
check_path "file" "Makefile" "Lifecycle Makefile present"
check_path "file" "PRD.md" "PRD document available"
check_path "file" "README.md" "README document available"
check_path "file" ".vrooli/service.json" "Lifecycle service.json configured"
check_path "file" "scenario-test.yaml" "Scenario test definition present"
check_path "file" "test/run-all-tests.sh" "Master test runner available"

# Key implementation artifacts
check_path "file" "api/main.go" "Go API entrypoint present"
check_path "file" "cli/test-genie" "CLI executable present"
check_path "file" "ui/server.js" "UI server entrypoint present"

echo ""
if [[ $FAIL -eq 0 ]]; then
    echo -e "${GREEN}🎉 Structural validation passed (${PASS} checks)${NC}"
    exit 0
else
    echo -e "${YELLOW}⚠️  Structural validation found ${FAIL} issue(s)${NC}"
    exit 1
fi
