#!/bin/bash
# Resource Generator Initialization Script
# Sets up the environment for resource generation

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(dirname "$SCRIPT_DIR")"
QUEUE_DIR="$SCENARIO_DIR/queue"

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}🚀 Initializing Resource Generator...${NC}"

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# 1. Check required dependencies
echo -e "${YELLOW}Checking dependencies...${NC}"

# Check for resource-claude-code (critical dependency)
if ! command_exists resource-claude-code; then
    echo -e "${RED}❌ Critical dependency missing: resource-claude-code${NC}"
    echo "Please ensure resource-claude-code is installed and in PATH"
    exit 1
fi
echo -e "${GREEN}✅ Found resource-claude-code${NC}"

# Check for Go (for API)
if ! command_exists go; then
    echo -e "${RED}❌ Go is not installed${NC}"
    echo "Please install Go to run the API server"
    exit 1
fi
echo -e "${GREEN}✅ Found Go${NC}"

# Check for Node.js (for UI)
if ! command_exists node; then
    echo -e "${YELLOW}⚠️  Node.js not found - UI will not be available${NC}"
else
    echo -e "${GREEN}✅ Found Node.js${NC}"
fi

# 2. Initialize queue directories
echo -e "${YELLOW}Setting up queue directories...${NC}"
mkdir -p "$QUEUE_DIR"/{pending,in-progress,completed,failed,templates}
echo -e "${GREEN}✅ Queue directories created${NC}"

# 3. Verify prompt files exist
echo -e "${YELLOW}Verifying prompt files...${NC}"
PROMPT_FILE="$SCENARIO_DIR/prompts/main-prompt.md"
if [ ! -f "$PROMPT_FILE" ]; then
    echo -e "${RED}❌ Main prompt file not found: $PROMPT_FILE${NC}"
    exit 1
fi

# Check for shared prompts
SHARED_PROMPTS_DIR="/home/matthalloran8/Vrooli/scripts/shared/prompts"
if [ ! -d "$SHARED_PROMPTS_DIR" ]; then
    echo -e "${YELLOW}⚠️  Shared prompts directory not found: $SHARED_PROMPTS_DIR${NC}"
    echo "Resource generator will use fallback prompts"
else
    echo -e "${GREEN}✅ Found shared prompts library${NC}"
fi

# 4. Build the Go API if needed
echo -e "${YELLOW}Building API server...${NC}"
cd "$SCENARIO_DIR/api"
if [ ! -f "resource-generator-api" ] || [ "main.go" -nt "resource-generator-api" ]; then
    go build -o resource-generator-api main.go
    echo -e "${GREEN}✅ API server built${NC}"
else
    echo -e "${GREEN}✅ API server already up to date${NC}"
fi

# 5. Install Node dependencies if package.json exists
if [ -f "$SCENARIO_DIR/ui/package.json" ]; then
    echo -e "${YELLOW}Installing UI dependencies...${NC}"
    cd "$SCENARIO_DIR/ui"
    if command_exists npm; then
        npm install --silent
        echo -e "${GREEN}✅ UI dependencies installed${NC}"
    else
        echo -e "${YELLOW}⚠️  npm not found - skipping UI setup${NC}"
    fi
fi

# 6. Set up environment variables
echo -e "${YELLOW}Configuring environment...${NC}"
if [ ! -f "$SCENARIO_DIR/.env" ]; then
    cat > "$SCENARIO_DIR/.env" <<EOF
# Resource Generator Environment Configuration
# NOTE: Ports must be provided by the orchestrator
# API_PORT=<set by orchestrator>
# UI_PORT=<set by orchestrator>
QUEUE_DIR=$QUEUE_DIR
RESOURCES_DIR=/home/matthalloran8/Vrooli/resources
API_HOST=localhost
EOF
    echo -e "${GREEN}✅ Environment template created${NC}"
    echo -e "${YELLOW}Note: API_PORT and UI_PORT must be set when starting the scenario${NC}"
else
    echo -e "${GREEN}✅ Environment file already exists${NC}"
fi

# 7. Create sample queue item if none exist
if [ -z "$(ls -A $QUEUE_DIR/pending 2>/dev/null)" ]; then
    echo -e "${YELLOW}Creating sample queue item...${NC}"
    cat > "$QUEUE_DIR/pending/sample.yaml.example" <<'EOF'
# Sample Resource Generation Request
# Rename to .yaml (remove .example) to activate
id: sample-resource-001
title: "Generate Sample Resource"
description: "A sample resource for testing"
type: "resource"
template: "data-processing"
resource_name: "sample-processor"
priority: "low"
requirements:
  features:
    - "Basic data processing"
    - "Health checks"
    - "CLI integration"
  ports:
    main: 25000
status: "pending"
created_by: "initialization"
created_at: "$(date -Iseconds)"
attempt_count: 0
EOF
    echo -e "${GREEN}✅ Sample queue item created (rename to activate)${NC}"
fi

# 8. Verify port availability if provided
if [ -n "$API_PORT" ] || [ -n "$UI_PORT" ]; then
    echo -e "${YELLOW}Checking provided ports...${NC}"
    
    check_port() {
        local port=$1
        local name=$2
        if nc -z localhost $port 2>/dev/null; then
            echo -e "${YELLOW}⚠️  $name port $port is already in use${NC}"
            return 1
        else
            echo -e "${GREEN}✅ $name port $port is available${NC}"
            return 0
        fi
    }
    
    [ -n "$API_PORT" ] && check_port $API_PORT "API"
    [ -n "$UI_PORT" ] && check_port $UI_PORT "UI"
else
    echo -e "${YELLOW}Note: Ports not provided. They must be set when starting the scenario.${NC}"
fi

# 9. Verify startup script exists (should already be in the repo)
if [ -f "$SCENARIO_DIR/start.sh" ]; then
    chmod +x "$SCENARIO_DIR/start.sh"
    echo -e "${GREEN}✅ Startup script is ready${NC}"
else
    echo -e "${RED}❌ Startup script not found at: $SCENARIO_DIR/start.sh${NC}"
    echo "Please ensure the scenario files are complete"
    exit 1
fi

# 10. Final summary
echo ""
echo -e "${GREEN}═══════════════════════════════════════════════════════${NC}"
echo -e "${GREEN}✅ Resource Generator initialization complete!${NC}"
echo -e "${GREEN}═══════════════════════════════════════════════════════${NC}"
echo ""
echo "To start the Resource Generator:"
echo "  API_PORT=<port> UI_PORT=<port> $SCENARIO_DIR/start.sh"
echo "  Or: API_PORT=<port> UI_PORT=<port> vrooli scenario resource-generator start"
echo ""
echo "To add resources to generate:"
echo "  1. Place YAML files in: $QUEUE_DIR/pending/"
echo "  2. Or use the UI (once started with ports)"
echo "  3. Or use CLI: API_PORT=<port> resource-generator queue add <resource-name>"
echo ""
echo "Important: Ports must be provided by the orchestrator when starting."
echo "Queue:    $QUEUE_DIR"
echo ""