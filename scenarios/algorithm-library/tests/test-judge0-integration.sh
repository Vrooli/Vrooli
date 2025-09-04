#!/bin/bash

# Test Judge0 integration for Algorithm Library

set -e

echo "Testing Judge0 integration..."

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

# Check if Judge0 is running
echo -n "Checking Judge0 availability... "
if resource-judge0 status 2>/dev/null | grep -q "Installed: Yes"; then
    echo -e "${GREEN}✓${NC}"
else
    echo -e "${RED}✗${NC}"
    echo "Judge0 is not installed. Run: resource-judge0 manage install"
    exit 1
fi

# Test simple Python code execution via Judge0
echo -n "Testing Python execution... "
PYTHON_CODE='print("Hello from Judge0")'
RESPONSE=$(curl -s -X POST http://localhost:2358/submissions \
    -H "Content-Type: application/json" \
    -d "{
        \"source_code\": \"$PYTHON_CODE\",
        \"language_id\": 71
    }" 2>/dev/null || echo "FAILED")

if echo "$RESPONSE" | grep -q "token"; then
    echo -e "${GREEN}✓${NC}"
else
    echo -e "${RED}✗${NC}"
    echo "Failed to submit to Judge0"
fi

# Test JavaScript execution
echo -n "Testing JavaScript execution... "
JS_CODE='console.log("Hello from Judge0")'
RESPONSE=$(curl -s -X POST http://localhost:2358/submissions \
    -H "Content-Type: application/json" \
    -d "{
        \"source_code\": \"$JS_CODE\",
        \"language_id\": 63
    }" 2>/dev/null || echo "FAILED")

if echo "$RESPONSE" | grep -q "token"; then
    echo -e "${GREEN}✓${NC}"
else
    echo -e "${RED}✗${NC}"
    echo "Failed to submit JavaScript to Judge0"
fi

echo ""
echo "Judge0 integration test complete!"