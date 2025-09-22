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

    if [ -f "api/main.go" ]; then
        # Check for required business functions
        local required_functions=(
            "handleChat"
            "selectModel"
            "processMessage"
            "validateInput"
        )

        local missing_functions=()

        for func in "${required_functions[@]}"; do
            if ! grep -q "func.*$func" api/main.go; then
                missing_functions+=("$func")
            fi
        done

        if [ ${#missing_functions[@]} -eq 0 ]; then
            echo -e "${GREEN}✅ All required business functions present${NC}"
        else
            echo -e "${RED}❌ Missing business functions: ${missing_functions[*]}${NC}"
            FAILED=1
        fi

        # Check for proper error handling
        if grep -q "errors\." api/main.go || grep -q "fmt\.Errorf" api/main.go; then
            echo -e "${GREEN}✅ Error handling implemented${NC}"
        else
            echo -e "${YELLOW}⚠️  Limited error handling detected${NC}"
        fi
    else
        echo -e "${YELLOW}⚠️  No API main.go found${NC}"
    fi
}

# Function to check chatbot-specific logic
test_chatbot_logic() {
    echo -e "${YELLOW}Testing chatbot-specific logic...${NC}"

    if [ -f "api/chatbot.go" ] || [ -f "api/main.go" ]; then
        local file_to_check="api/main.go"
        if [ -f "api/chatbot.go" ]; then
            file_to_check="api/chatbot.go"
        fi

        # Check for conversation management
        if grep -q "conversation\|session\|context" "$file_to_check"; then
            echo -e "${GREEN}✅ Conversation management implemented${NC}"
        else
            echo -e "${RED}❌ No conversation management found${NC}"
            FAILED=1
        fi

        # Check for model orchestration
        if grep -q "model\|orchestrat\|select" "$file_to_check"; then
            echo -e "${GREEN}✅ Model orchestration implemented${NC}"
        else
            echo -e "${RED}❌ No model orchestration found${NC}"
            FAILED=1
        fi
    else
        echo -e "${YELLOW}⚠️  No chatbot logic files found${NC}"
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
        # Check for chat interface
        if [ -f "ui/index.html" ]; then
            if grep -q "chat\|message\|input" ui/index.html; then
                echo -e "${GREEN}✅ Chat interface implemented${NC}"
            else
                echo -e "${RED}❌ Chat interface not found${NC}"
                FAILED=1
            fi
        fi

        # Check for JavaScript functionality
        if [ -f "ui/script.js" ] || find ui -name "*.js" | grep -q .; then
            local js_file="ui/script.js"
            if [ ! -f "$js_file" ]; then
                js_file=$(find ui -name "*.js" | head -1)
            fi

            if [ -n "$js_file" ] && grep -q "WebSocket\|fetch\|send" "$js_file"; then
                echo -e "${GREEN}✅ Frontend communication implemented${NC}"
            else
                echo -e "${RED}❌ Frontend communication not found${NC}"
                FAILED=1
            fi
        else
            echo -e "${YELLOW}⚠️  No JavaScript files found${NC}"
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

            if jq -e '.runtime' .vrooli/service.json > /dev/null 2>&1; then
                echo -e "${GREEN}✅ Runtime configuration present${NC}"
            else
                echo -e "${RED}❌ Runtime configuration missing${NC}"
                FAILED=1
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