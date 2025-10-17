#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🔍 Testing Dependencies${NC}"
echo "======================="

# Track failures
FAILED=0

# Function to check command availability
check_command() {
    local cmd=$1
    local description=$2

    if command -v "$cmd" &> /dev/null; then
        echo -e "${GREEN}✅ $description found: $(which $cmd)${NC}"
        return 0
    else
        echo -e "${RED}❌ $description not found${NC}"
        return 1
    fi
}

# Function to check Go module dependencies
check_go_dependencies() {
    echo -e "${YELLOW}Checking Go dependencies...${NC}"

    if [ -f "api/go.mod" ]; then
        cd api
        if go mod download && go mod verify; then
            echo -e "${GREEN}✅ Go module dependencies verified${NC}"
        else
            echo -e "${RED}❌ Go module dependencies failed${NC}"
            FAILED=1
        fi
        cd ..
    else
        echo -e "${YELLOW}⚠️  No Go module found${NC}"
    fi
}

# Function to check Node.js dependencies
check_node_dependencies() {
    echo -e "${YELLOW}Checking Node.js dependencies...${NC}"

    if [ -f "ui/package.json" ]; then
        cd ui
        if npm install --dry-run > /dev/null 2>&1; then
            echo -e "${GREEN}✅ Node.js dependencies can be installed${NC}"
        else
            echo -e "${RED}❌ Node.js dependencies check failed${NC}"
            FAILED=1
        fi
        cd ..
    else
        echo -e "${YELLOW}⚠️  No Node.js package.json found${NC}"
    fi
}

# Function to check required files
check_required_files() {
    echo -e "${YELLOW}Checking required files...${NC}"

    local required_files=(
        ".vrooli/service.json"
        "Makefile"
        "README.md"
    )

    local missing_files=()

    for file in "${required_files[@]}"; do
        if [ ! -f "$file" ]; then
            missing_files+=("$file")
        fi
    done

    if [ ${#missing_files[@]} -eq 0 ]; then
        echo -e "${GREEN}✅ All required files present${NC}"
    else
        echo -e "${RED}❌ Missing required files: ${missing_files[*]}${NC}"
        FAILED=1
    fi
}

# Function to check service.json validity
check_service_json() {
    echo -e "${YELLOW}Checking service.json validity...${NC}"

    if command -v jq &> /dev/null; then
        if jq . .vrooli/service.json > /dev/null 2>&1; then
            echo -e "${GREEN}✅ service.json is valid JSON${NC}"
        else
            echo -e "${RED}❌ service.json is invalid JSON${NC}"
            FAILED=1
        fi
    else
        echo -e "${YELLOW}⚠️  jq not available - skipping JSON validation${NC}"
    fi
}

# Main dependency checks
echo "Checking system dependencies..."
check_command "go" "Go" || FAILED=1
check_command "node" "Node.js" || FAILED=1
check_command "npm" "npm" || FAILED=1

echo ""
check_go_dependencies

echo ""
check_node_dependencies

echo ""
check_required_files

echo ""
check_service_json

echo ""
if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}🎉 All dependency checks passed!${NC}"
    exit 0
else
    echo -e "${RED}💥 Some dependency checks failed${NC}"
    exit 1
fi