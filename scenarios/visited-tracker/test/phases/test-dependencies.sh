#!/bin/bash
# Simplified Dependencies validation phase - <30 seconds
# Validates required resources using resource CLI commands
set -uo pipefail

echo "=== Dependencies Phase (Target: <30s) ==="
start_time=$(date +%s)

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

error_count=0
warning_count=0

# Check PostgreSQL (required) using resource CLI
echo "🔍 Checking PostgreSQL (required)..."
if resource-postgres test smoke >/dev/null 2>&1; then
    echo -e "${GREEN}✅ PostgreSQL is running and ready${NC}"
else
    echo -e "${RED}❌ PostgreSQL smoke test failed${NC}"
    echo "   Start with: vrooli resource start postgres"
    ((error_count++))
fi

# Check Redis (optional) using resource CLI
echo "🔍 Checking Redis (optional)..."
if resource-redis test smoke >/dev/null 2>&1; then
    echo -e "${GREEN}✅ Redis is running and responsive${NC}"
else
    echo -e "${YELLOW}⚠️  Redis smoke test failed (optional dependency)${NC}"
    echo "   Start with: vrooli resource start redis"
    ((warning_count++))
fi

# Check Go environment
echo "🔍 Checking Go environment..."
if go version >/dev/null 2>&1; then
    go_version=$(go version | grep -oE 'go[0-9]+\.[0-9]+' | head -1)
    echo -e "${GREEN}✅ Go is available: $go_version${NC}"
else
    echo -e "${RED}❌ Go is not installed${NC}"
    ((error_count++))
fi

# Check Node.js environment
echo "🔍 Checking Node.js environment..."
if node --version >/dev/null 2>&1; then
    node_version=$(node --version)
    echo -e "${GREEN}✅ Node.js is available: $node_version${NC}"
else
    echo -e "${RED}❌ Node.js is not installed${NC}"
    ((error_count++))
fi

# Check essential tools
echo "🔍 Checking essential tools..."
essential_tools=("jq" "curl")

for tool in "${essential_tools[@]}"; do
    if "$tool" --version >/dev/null 2>&1; then
        echo -e "${GREEN}✅ $tool is available${NC}"
    else
        echo -e "${RED}❌ $tool is not available${NC}"
        ((error_count++))
    fi
done

# Performance check
end_time=$(date +%s)
duration=$((end_time - start_time))
echo ""

# Results
if [ $error_count -eq 0 ]; then
    if [ $warning_count -eq 0 ]; then
        echo -e "${GREEN}✅ Dependencies validation completed successfully in ${duration}s${NC}"
    else
        echo -e "${GREEN}✅ Dependencies validation completed with $warning_count warnings in ${duration}s${NC}"
    fi
else
    echo -e "${RED}❌ Dependencies validation failed with $error_count errors and $warning_count warnings in ${duration}s${NC}"
fi

if [ $duration -gt 30 ]; then
    echo -e "${YELLOW}⚠️  Dependencies phase exceeded 30s target${NC}"
fi

# Exit with appropriate code
if [ $error_count -eq 0 ]; then
    exit 0
else
    exit 1
fi