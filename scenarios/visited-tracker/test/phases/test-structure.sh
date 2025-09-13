#!/bin/bash
# Structure validation phase - <15 seconds
# Validates required files, configuration, and directory structure
set -euo pipefail

echo "=== Structure Phase (Target: <15s) ==="
start_time=$(date +%s)

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

error_count=0

# Check required files
echo "🔍 Checking required files..."
required_files=(
    ".vrooli/service.json"
    "README.md"
    "PRD.md"
    "api/main.go"
    "api/go.mod"
    "cli/visited-tracker"
    "cli/install.sh"
    "ui/index.html"
    "ui/package.json"
    "ui/server.js"
    "scenario-test.yaml"
)

missing_files=()
for file in "${required_files[@]}"; do
    if [ ! -f "$file" ]; then
        missing_files+=("$file")
        ((error_count++))
    fi
done

if [ ${#missing_files[@]} -gt 0 ]; then
    echo -e "${RED}❌ Missing required files:${NC}"
    printf "   - %s\n" "${missing_files[@]}"
else
    echo -e "${GREEN}✅ All required files present${NC}"
fi

# Check required directories
echo "🔍 Checking directory structure..."
required_dirs=("api" "cli" "ui" "data" "test")
missing_dirs=()
for dir in "${required_dirs[@]}"; do
    if [ ! -d "$dir" ]; then
        missing_dirs+=("$dir")
        ((error_count++))
    fi
done

if [ ${#missing_dirs[@]} -gt 0 ]; then
    echo -e "${RED}❌ Missing required directories:${NC}"
    printf "   - %s\n" "${missing_dirs[@]}"
else
    echo -e "${GREEN}✅ All required directories present${NC}"
fi

# Validate service.json schema
echo "🔍 Validating service.json..."
if command -v jq >/dev/null 2>&1; then
    if ! jq empty < .vrooli/service.json >/dev/null 2>&1; then
        echo -e "${RED}❌ Invalid JSON in service.json${NC}"
        ((error_count++))
    else
        echo -e "${GREEN}✅ service.json is valid JSON${NC}"
        
        # Check required fields
        required_fields=("service.name" "service.version" "ports" "lifecycle")
        for field in "${required_fields[@]}"; do
            if ! jq -e ".$field" < .vrooli/service.json >/dev/null 2>&1; then
                echo -e "${RED}❌ Missing required field in service.json: $field${NC}"
                ((error_count++))
            fi
        done
        
        if [ $error_count -eq 0 ]; then
            service_name=$(jq -r '.service.name' .vrooli/service.json)
            if [ "$service_name" = "visited-tracker" ]; then
                echo -e "${GREEN}✅ service.json contains correct service name${NC}"
            else
                echo -e "${RED}❌ Incorrect service name in service.json: $service_name${NC}"
                ((error_count++))
            fi
        fi
    fi
else
    echo -e "${YELLOW}⚠️  jq not available, skipping JSON validation${NC}"
fi

# Check Go module structure
echo "🔍 Validating Go module..."
if [ -f "api/go.mod" ]; then
    if grep -q "module " api/go.mod; then
        echo -e "${GREEN}✅ Go module properly defined${NC}"
    else
        echo -e "${RED}❌ Invalid go.mod structure${NC}"
        ((error_count++))
    fi
else
    echo -e "${RED}❌ go.mod missing${NC}"
    ((error_count++))
fi

# Check Node.js package.json structure
echo "🔍 Validating Node.js package..."
if [ -f "ui/package.json" ]; then
    if command -v jq >/dev/null 2>&1; then
        if jq -e '.name' ui/package.json >/dev/null 2>&1; then
            package_name=$(jq -r '.name' ui/package.json)
            echo -e "${GREEN}✅ Node.js package properly defined: $package_name${NC}"
        else
            echo -e "${RED}❌ Invalid package.json structure${NC}"
            ((error_count++))
        fi
    fi
else
    echo -e "${RED}❌ ui/package.json missing${NC}"
    ((error_count++))
fi

# Check CLI binary exists and is executable
echo "🔍 Validating CLI binary..."
if [ -f "cli/visited-tracker" ]; then
    if [ -x "cli/visited-tracker" ]; then
        echo -e "${GREEN}✅ CLI binary exists and is executable${NC}"
    else
        echo -e "${RED}❌ CLI binary is not executable${NC}"
        ((error_count++))
    fi
else
    echo -e "${RED}❌ CLI binary missing${NC}"
    ((error_count++))
fi

# Check scenario-test.yaml structure
echo "🔍 Validating scenario-test.yaml..."
if [ -f "scenario-test.yaml" ]; then
    if grep -q "version:" scenario-test.yaml && grep -q "scenario:" scenario-test.yaml; then
        echo -e "${GREEN}✅ scenario-test.yaml structure valid${NC}"
    else
        echo -e "${RED}❌ scenario-test.yaml missing required fields${NC}"
        ((error_count++))
    fi
else
    echo -e "${RED}❌ scenario-test.yaml missing${NC}"
    ((error_count++))
fi

# Check data directory structure
echo "🔍 Checking data directory..."
if [ -d "data" ]; then
    if [ -d "data/campaigns" ]; then
        echo -e "${GREEN}✅ Data directory structure correct${NC}"
    else
        echo -e "${YELLOW}⚠️  data/campaigns directory missing (will be created on setup)${NC}"
    fi
else
    echo -e "${YELLOW}⚠️  data directory missing (will be created on setup)${NC}"
fi

# Performance check
end_time=$(date +%s)
duration=$((end_time - start_time))
echo ""
if [ $error_count -eq 0 ]; then
    echo -e "${GREEN}✅ Structure validation completed successfully in ${duration}s${NC}"
else
    echo -e "${RED}❌ Structure validation failed with $error_count errors in ${duration}s${NC}"
fi

if [ $duration -gt 15 ]; then
    echo -e "${YELLOW}⚠️  Structure phase exceeded 15s target${NC}"
fi

# Exit with appropriate code
if [ $error_count -eq 0 ]; then
    exit 0
else
    exit 1
fi