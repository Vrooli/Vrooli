#!/bin/bash

# Structure Test Phase
# Validates Email Triage scenario structure and required files

set -euo pipefail

echo "📁 Testing Email Triage structure..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

SCENARIO_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
FAILURES=0

# Required files and directories
REQUIRED_FILES=(
    ".vrooli/service.json"
    "PRD.md"
    "README.md"
    "Makefile"
    "api/main.go"
    "api/go.mod"
    "cli/email-triage"
    "initialization/postgres/schema.sql"
)

REQUIRED_DIRS=(
    "api"
    "api/handlers"
    "api/services"
    "api/models"
    "cli"
    "initialization"
    "initialization/postgres"
    "ui"
    "test"
    "test/phases"
)

# Check required files
echo "  Checking required files..."
for file in "${REQUIRED_FILES[@]}"; do
    if [[ -f "$SCENARIO_DIR/$file" ]]; then
        echo -e "    ${GREEN}✓${NC} $file"
    else
        echo -e "    ${RED}✗${NC} $file"
        ((FAILURES++))
    fi
done

# Check required directories
echo "  Checking required directories..."
for dir in "${REQUIRED_DIRS[@]}"; do
    if [[ -d "$SCENARIO_DIR/$dir" ]]; then
        echo -e "    ${GREEN}✓${NC} $dir/"
    else
        echo -e "    ${RED}✗${NC} $dir/"
        ((FAILURES++))
    fi
done

# Check if API binary is built
echo "  Checking API binary..."
if [[ -f "$SCENARIO_DIR/api/email-triage-api" ]]; then
    echo -e "    ${GREEN}✓${NC} API binary exists"
else
    echo -e "    ${RED}✗${NC} API binary missing (run 'make build')"
    ((FAILURES++))
fi

# Check if CLI is installed
echo "  Checking CLI installation..."
if command -v email-triage >/dev/null 2>&1; then
    echo -e "    ${GREEN}✓${NC} CLI is installed"
else
    echo -e "    ${RED}✗${NC} CLI not installed"
    ((FAILURES++))
fi

# Check service.json validity
echo "  Validating service.json..."
if jq -e '.service.name == "email-triage"' "$SCENARIO_DIR/.vrooli/service.json" >/dev/null 2>&1; then
    echo -e "    ${GREEN}✓${NC} Valid service.json"
else
    echo -e "    ${RED}✗${NC} Invalid service.json"
    ((FAILURES++))
fi

# Summary
if [[ $FAILURES -eq 0 ]]; then
    echo -e "\n${GREEN}✅ Structure validation passed${NC}"
    exit 0
else
    echo -e "\n${RED}❌ $FAILURES structure issue(s) found${NC}"
    exit 1
fi