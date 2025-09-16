#!/bin/bash
# Basic validation that Eclipse Ditto resource is properly scaffolded

echo "Eclipse Ditto Resource Validation"
echo "=================================="

failed=0

# Test 1: CLI executable
echo -n "1. CLI is executable... "
if [[ -x "/home/matthalloran8/Vrooli/resources/eclipse-ditto/cli.sh" ]]; then
    echo "✓"
else
    echo "✗"
    failed=$((failed + 1))
fi

# Test 2: Required files exist
echo -n "2. Required files... "
required_files=(
    "cli.sh"
    "lib/core.sh"
    "lib/test.sh"
    "config/defaults.sh"
    "config/schema.json"
    "config/runtime.json"
    "test/run-tests.sh"
    "PRD.md"
    "README.md"
)
missing=0
for file in "${required_files[@]}"; do
    if [[ ! -f "/home/matthalloran8/Vrooli/resources/eclipse-ditto/$file" ]]; then
        missing=$((missing + 1))
    fi
done
if [[ $missing -eq 0 ]]; then
    echo "✓"
else
    echo "✗ Missing $missing files"
    failed=$((failed + 1))
fi

# Test 3: Port allocation
echo -n "3. Port registered... "
source /home/matthalloran8/Vrooli/scripts/resources/port_registry.sh
if [[ "${RESOURCE_PORTS[eclipse-ditto]}" == "8089" ]]; then
    echo "✓"
else
    echo "✗"
    failed=$((failed + 1))
fi

# Test 4: Help command works
echo -n "4. Help command... "
if /home/matthalloran8/Vrooli/resources/eclipse-ditto/cli.sh help &>/dev/null; then
    echo "✓"
else
    echo "✗"
    failed=$((failed + 1))
fi

# Test 5: Info command works
echo -n "5. Info command... "
if /home/matthalloran8/Vrooli/resources/eclipse-ditto/cli.sh info &>/dev/null; then
    echo "✓"
else
    echo "✗"
    failed=$((failed + 1))
fi

echo ""
if [[ $failed -eq 0 ]]; then
    echo "✅ All validation checks passed"
    exit 0
else
    echo "❌ Validation failed: $failed checks failed"
    exit 1
fi