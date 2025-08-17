#!/bin/bash

# Simple validation script for nutrition-tracker
set -e

echo "🔍 Validating nutrition-tracker scenario..."

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

# Check structure
echo "📁 Checking structure..."
required_dirs=(".vrooli" "api" "cli" "initialization" "ui" "test")
for dir in "${required_dirs[@]}"; do
    if [ -d "$dir" ]; then
        echo -e "${GREEN}✓${NC} Found $dir/"
    else
        echo -e "${RED}✗${NC} Missing $dir/"
        exit 1
    fi
done

# Check service.json
echo "⚙️ Checking service.json..."
if [ -f ".vrooli/service.json" ]; then
    if jq empty .vrooli/service.json 2>/dev/null; then
        echo -e "${GREEN}✓${NC} Valid service.json"
    else
        echo -e "${RED}✗${NC} Invalid JSON in service.json"
        exit 1
    fi
else
    echo -e "${RED}✗${NC} Missing service.json"
    exit 1
fi

# Check CLI
echo "🖥️ Checking CLI..."
if [ -f "cli/nutrition-tracker" ]; then
    chmod +x cli/nutrition-tracker
    if ./cli/nutrition-tracker --version >/dev/null 2>&1; then
        echo -e "${GREEN}✓${NC} CLI works"
    else
        echo -e "${RED}✗${NC} CLI failed"
        exit 1
    fi
else
    echo -e "${RED}✗${NC} Missing CLI"
    exit 1
fi

# Check API build
echo "🔨 Checking API build..."
if [ -f "api/main.go" ]; then
    cd api
    if go build -o test-build main.go 2>/dev/null; then
        rm -f test-build
        echo -e "${GREEN}✓${NC} API compiles"
    else
        echo -e "${RED}✗${NC} API compilation failed"
        exit 1
    fi
    cd ..
else
    echo -e "${RED}✗${NC} Missing API main.go"
    exit 1
fi

# Check workflows
echo "📋 Checking n8n workflows..."
workflow_count=$(ls initialization/n8n/*.json 2>/dev/null | wc -l)
if [ "$workflow_count" -gt 0 ]; then
    echo -e "${GREEN}✓${NC} Found $workflow_count workflows"
    for workflow in initialization/n8n/*.json; do
        if jq empty "$workflow" 2>/dev/null; then
            echo -e "  ${GREEN}✓${NC} $(basename $workflow)"
        else
            echo -e "  ${RED}✗${NC} Invalid JSON: $(basename $workflow)"
            exit 1
        fi
    done
else
    echo -e "${RED}✗${NC} No workflows found"
    exit 1
fi

# Check UI
echo "🎨 Checking UI..."
if [ -f "ui/index.html" ] && [ -f "ui/script.js" ] && [ -f "ui/styles.css" ]; then
    echo -e "${GREEN}✓${NC} UI files present"
else
    echo -e "${RED}✗${NC} Missing UI files"
    exit 1
fi

echo -e "\n${GREEN}✅ nutrition-tracker scenario is valid!${NC}"
echo "Ready for conversion with: vrooli scenario convert nutrition-tracker"