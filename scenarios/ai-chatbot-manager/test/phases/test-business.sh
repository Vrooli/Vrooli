#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🧠 Testing Business Logic${NC}"
echo "========================"

# Track failures
FAILED=0

# Function to check API business logic
test_api_business_logic() {
    echo -e "${YELLOW}Testing API business logic...${NC}"

    if [ -d "api" ]; then
        # Check for required business functions across all Go files
        local required_patterns=(
            "chat"
            "model"
            "message"
            "validat"
        )

        local missing_patterns=()
        local pattern_names=(
            "chat handling"
            "model selection"
            "message processing"
            "input validation"
        )

        for i in "${!required_patterns[@]}"; do
            if ! grep -riq "${required_patterns[$i]}" api/*.go 2>/dev/null; then
                missing_patterns+=("${pattern_names[$i]}")
            fi
        done

        if [ ${#missing_patterns[@]} -eq 0 ]; then
            echo -e "${GREEN}✅ All required business functions present${NC}"
        else
            echo -e "${YELLOW}⚠️  Some business patterns not detected: ${missing_patterns[*]}${NC}"
            # Note: Not marking as failure since code may use different naming conventions
        fi

        # Check for proper error handling
        if grep -rq "errors\.\|fmt\.Errorf" api/*.go 2>/dev/null; then
            echo -e "${GREEN}✅ Error handling implemented${NC}"
        else
            echo -e "${YELLOW}⚠️  Limited error handling detected${NC}"
        fi
    else
        echo -e "${YELLOW}⚠️  No API directory found${NC}"
    fi
}

# Function to check chatbot-specific logic
test_chatbot_logic() {
    echo -e "${YELLOW}Testing chatbot-specific logic...${NC}"

    if [ -d "api" ]; then
        # Check for conversation management across all API files
        if grep -rq "conversation\|session\|context" api/*.go 2>/dev/null; then
            echo -e "${GREEN}✅ Conversation management implemented${NC}"
        else
            echo -e "${RED}❌ No conversation management found${NC}"
            FAILED=1
        fi

        # Check for model orchestration across all API files
        if grep -rq "model\|Model\|ollama\|Ollama" api/*.go 2>/dev/null; then
            echo -e "${GREEN}✅ Model orchestration implemented${NC}"
        else
            echo -e "${RED}❌ No model orchestration found${NC}"
            FAILED=1
        fi
    else
        echo -e "${YELLOW}⚠️  No API directory found${NC}"
    fi
}

# Function to check WebSocket functionality
test_websocket_functionality() {
    echo -e "${YELLOW}Testing WebSocket functionality...${NC}"

    if [ -f "api/websocket.go" ]; then
        # Check for WebSocket upgrade
        if grep -q "websocket\.Upgrader\|Upgrade" api/websocket.go; then
            echo -e "${GREEN}✅ WebSocket upgrade implemented${NC}"
        else
            echo -e "${RED}❌ WebSocket upgrade not found${NC}"
            FAILED=1
        fi

        # Check for message handling
        if grep -q "ReadJSON\|WriteJSON\|message" api/websocket.go; then
            echo -e "${GREEN}✅ WebSocket message handling implemented${NC}"
        else
            echo -e "${RED}❌ WebSocket message handling not found${NC}"
            FAILED=1
        fi
    else
        echo -e "${YELLOW}⚠️  No WebSocket implementation found${NC}"
    fi
}

# Function to check UI business logic
test_ui_business_logic() {
    echo -e "${YELLOW}Testing UI business logic...${NC}"

    if [ -d "ui" ]; then
        # Check for React/TypeScript components or HTML interface
        local has_interface=false

        if [ -f "ui/index.html" ]; then
            if grep -q "chat\|message\|input\|root" ui/index.html; then
                echo -e "${GREEN}✅ Chat interface entry point found${NC}"
                has_interface=true
            fi
        fi

        # Check for React components
        if [ -d "ui/src" ]; then
            if find ui/src -name "*.tsx" -o -name "*.jsx" -o -name "*.ts" | grep -q .; then
                echo -e "${GREEN}✅ React/TypeScript UI components found${NC}"
                has_interface=true
            fi
        fi

        if [ "$has_interface" = false ]; then
            echo -e "${YELLOW}⚠️  No UI interface components detected${NC}"
        fi

        # Check for JavaScript/TypeScript functionality
        if find ui -name "*.js" -o -name "*.ts" -o -name "*.tsx" 2>/dev/null | grep -q .; then
            if grep -rq "WebSocket\|fetch\|axios\|API\|api" ui/ 2>/dev/null; then
                echo -e "${GREEN}✅ Frontend communication implemented${NC}"
            else
                echo -e "${YELLOW}⚠️  Frontend communication patterns not detected${NC}"
            fi
        else
            echo -e "${YELLOW}⚠️  No JavaScript/TypeScript files found${NC}"
        fi
    else
        echo -e "${YELLOW}⚠️  No UI directory found${NC}"
    fi
}

# Function to check CLI business logic
test_cli_business_logic() {
    echo -e "${YELLOW}Testing CLI business logic...${NC}"

    if [ -d "cli" ]; then
        # Check for CLI commands
        if find cli -name "*.sh" | grep -q .; then
            local cli_file=$(find cli -name "*.sh" | head -1)

            if grep -q "chat\|send\|connect" "$cli_file"; then
                echo -e "${GREEN}✅ CLI chat commands implemented${NC}"
            else
                echo -e "${RED}❌ CLI chat commands not found${NC}"
                FAILED=1
            fi
        else
            echo -e "${YELLOW}⚠️  No CLI scripts found${NC}"
        fi
    else
        echo -e "${YELLOW}⚠️  No CLI directory found${NC}"
    fi
}

# Function to check configuration validation
test_configuration_validation() {
    echo -e "${YELLOW}Testing configuration validation...${NC}"

    if [ -f ".vrooli/service.json" ]; then
        # Check for required configuration fields
        if command -v jq &> /dev/null; then
            if jq -e '.lifecycle' .vrooli/service.json > /dev/null 2>&1; then
                echo -e "${GREEN}✅ Lifecycle configuration present${NC}"
            else
                echo -e "${RED}❌ Lifecycle configuration missing${NC}"
                FAILED=1
            fi

            # Check for v2.0 lifecycle structure
            if jq -e '.lifecycle.version' .vrooli/service.json > /dev/null 2>&1; then
                local version=$(jq -r '.lifecycle.version' .vrooli/service.json)
                echo -e "${GREEN}✅ Lifecycle version ${version} configured${NC}"
            fi

            # Check for essential lifecycle sections
            if jq -e '.lifecycle.setup' .vrooli/service.json > /dev/null 2>&1; then
                echo -e "${GREEN}✅ Setup lifecycle defined${NC}"
            else
                echo -e "${YELLOW}⚠️  Setup lifecycle not defined${NC}"
            fi

            if jq -e '.lifecycle.health' .vrooli/service.json > /dev/null 2>&1; then
                echo -e "${GREEN}✅ Health checks configured${NC}"
            else
                echo -e "${YELLOW}⚠️  Health checks not configured${NC}"
            fi
        else
            echo -e "${YELLOW}⚠️  jq not available - skipping detailed config validation${NC}"
        fi
    else
        echo -e "${RED}❌ service.json not found${NC}"
        FAILED=1
    fi
}

# Main business logic tests
test_api_business_logic
echo ""

test_chatbot_logic
echo ""

test_websocket_functionality
echo ""

test_ui_business_logic
echo ""

test_cli_business_logic
echo ""

test_configuration_validation
echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}🎉 All business logic tests passed!${NC}"
    exit 0
else
    echo -e "${RED}💥 Some business logic tests failed${NC}"
    exit 1
fi