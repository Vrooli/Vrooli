#!/usr/bin/env bash
# Validates that all UI user flow requirements have integration test coverage
# Usage: ./scripts/validate-user-flows.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
REQUIREMENTS_DIR="$SCENARIO_ROOT/requirements"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

MISSING_TESTS=()
MISSING_FILES=()
INCORRECT_TYPE=()
TOTAL_CHECKED=0
TOTAL_VALID=0

echo "====================================================================="
echo "Vrooli Ascension - User Flow Validation"
echo "====================================================================="
echo ""

# Check if jq is available
if ! command -v jq >/dev/null 2>&1; then
    echo -e "${RED}❌ Error: jq is required but not installed${NC}"
    exit 1
fi

# Check ui/user-flows.json exists
if [ ! -f "$REQUIREMENTS_DIR/ui/user-flows.json" ]; then
    echo -e "${RED}❌ Error: requirements/ui/user-flows.json not found${NC}"
    exit 1
fi

echo "Checking UI user flow requirements..."
echo ""

# Process each requirement in ui/user-flows.json
while IFS= read -r req_entry; do
    req_id=$(echo "$req_entry" | jq -r '.id')

    TOTAL_CHECKED=$((TOTAL_CHECKED + 1))

    # Get integration phase validations
    validations=$(echo "$req_entry" | jq -c '[.validation[]? | select(.phase == "integration")]')
    validation_count=$(echo "$validations" | jq 'length')

    if [ "$validation_count" -eq 0 ]; then
        MISSING_TESTS+=("$req_id: No integration test defined")
        echo -e "${RED}❌ $req_id${NC} - No integration validation"
        continue
    fi

    # Check each validation
    has_valid_test=false
    while IFS= read -r validation; do
        val_type=$(echo "$validation" | jq -r '.type // "unknown"')
        val_ref=$(echo "$validation" | jq -r '.ref // ""')

        # Check validation type
        if [ "$val_type" != "automation" ]; then
            INCORRECT_TYPE+=("$req_id: validation type '$val_type' should be 'automation'")
            echo -e "${YELLOW}⚠️  $req_id${NC} - Validation type is '$val_type' (should be 'automation')"
            continue
        fi

        # Check if file exists
        if [ -z "$val_ref" ]; then
            MISSING_TESTS+=("$req_id: Validation has no ref field")
            echo -e "${RED}❌ $req_id${NC} - Validation missing ref field"
            continue
        fi

        playbook_path="$SCENARIO_ROOT/$val_ref"
        if [ ! -f "$playbook_path" ]; then
            MISSING_FILES+=("$req_id: Playbook not found at $val_ref")
            echo -e "${RED}❌ $req_id${NC} - Playbook missing: $val_ref"
            continue
        fi

        has_valid_test=true
        echo -e "${GREEN}✅ $req_id${NC} - $val_ref"

    done < <(echo "$validations" | jq -c '.[]')

    if [ "$has_valid_test" = true ]; then
        TOTAL_VALID=$((TOTAL_VALID + 1))
    fi

done < <(jq -c '.requirements[]' "$REQUIREMENTS_DIR/ui/user-flows.json")

echo ""
echo "====================================================================="
echo "Validation Summary"
echo "====================================================================="
echo ""
echo "Total requirements checked: $TOTAL_CHECKED"
echo -e "Valid test coverage:        ${GREEN}$TOTAL_VALID${NC}"
echo ""

if [ ${#MISSING_TESTS[@]} -gt 0 ]; then
    echo -e "${RED}Missing Tests (${#MISSING_TESTS[@]}):${NC}"
    for item in "${MISSING_TESTS[@]}"; do
        echo "  - $item"
    done
    echo ""
fi

if [ ${#MISSING_FILES[@]} -gt 0 ]; then
    echo -e "${RED}Missing Playbook Files (${#MISSING_FILES[@]}):${NC}"
    for item in "${MISSING_FILES[@]}"; do
        echo "  - $item"
    done
    echo ""
fi

if [ ${#INCORRECT_TYPE[@]} -gt 0 ]; then
    echo -e "${YELLOW}Incorrect Validation Types (${#INCORRECT_TYPE[@]}):${NC}"
    for item in "${INCORRECT_TYPE[@]}"; do
        echo "  - $item"
    done
    echo ""
    echo "Note: Integration phase tests require validation type 'automation', not 'test'"
    echo ""
fi

# Check if requirements/ui/user-flows.json is imported in index.json
if ! grep -q '"ui/user-flows.json"' "$REQUIREMENTS_DIR/index.json"; then
    echo -e "${YELLOW}⚠️  Warning: ui/user-flows.json is not imported in requirements/index.json${NC}"
    echo "   Add '\"ui/user-flows.json\"' to the imports array"
    echo ""
fi

# Exit with error if any issues found
TOTAL_ISSUES=$((${#MISSING_TESTS[@]} + ${#MISSING_FILES[@]} + ${#INCORRECT_TYPE[@]}))

if [ $TOTAL_ISSUES -gt 0 ]; then
    echo -e "${RED}❌ Validation failed with $TOTAL_ISSUES issue(s)${NC}"
    exit 1
else
    echo -e "${GREEN}✅ All UI user flows have valid integration test coverage!${NC}"
    exit 0
fi
