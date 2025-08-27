#!/usr/bin/env bash
#############################################################################
# Setup script for Python App Orchestrator
# Creates virtual environment and installs dependencies
#############################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
ORCHESTRATOR_DIR="${APP_ROOT}/scripts/scenarios/tools/orchestrator"
VENV_DIR="${ORCHESTRATOR_DIR}/venv"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}Setting up Python App Orchestrator...${NC}"

# Check Python availability
if ! command -v python3 &>/dev/null; then
    echo -e "${RED}Error: Python 3 is required but not found${NC}"
    echo "Please install Python 3.8 or later"
    exit 1
fi

# Get Python version
PYTHON_VERSION=$(python3 -c 'import sys; print(f"{sys.version_info.major}.{sys.version_info.minor}")')
echo -e "${BLUE}Found Python ${PYTHON_VERSION}${NC}"

# Check minimum version (3.8+)
if ! python3 -c 'import sys; exit(0 if sys.version_info >= (3, 8) else 1)' 2>/dev/null; then
    echo -e "${RED}Error: Python 3.8+ is required${NC}"
    exit 1
fi

# Try to create virtual environment
if [[ ! -d "$VENV_DIR" ]]; then
    echo -e "${BLUE}Creating virtual environment...${NC}"
    if python3 -m venv "$VENV_DIR" 2>/dev/null; then
        echo -e "${GREEN}✓ Virtual environment created${NC}"
        USE_VENV=true
    else
        echo -e "${YELLOW}Warning: Could not create virtual environment${NC}"
        echo -e "${YELLOW}This usually means python3-venv is not installed${NC}"
        echo -e "${YELLOW}To install: sudo apt-get install python3-venv${NC}"
        echo ""
        echo -e "${BLUE}Checking if required packages are available system-wide...${NC}"
        
        # Check if we can run without venv (system packages)
        CAN_RUN_WITHOUT_VENV=true
        for pkg in colorama psutil jsonschema; do
            if ! python3 -c "import $pkg" 2>/dev/null; then
                echo -e "${YELLOW}  Missing: $pkg${NC}"
                CAN_RUN_WITHOUT_VENV=false
            else
                echo -e "${GREEN}  Found: $pkg${NC}"
            fi
        done
        
        if [[ "$CAN_RUN_WITHOUT_VENV" == "true" ]]; then
            echo -e "${GREEN}✓ Can run with system Python packages${NC}"
            USE_VENV=false
        else
            echo -e "${YELLOW}Missing optional packages - will use minimal Python orchestrator${NC}"
            echo -e "${BLUE}The minimal orchestrator uses only Python standard library${NC}"
            echo ""
            echo "To enable full features (optional):"
            echo "  1. Install python3-venv: sudo apt-get install python3-venv"
            echo "  2. Install packages globally: pip3 install --user colorama psutil jsonschema"
            echo ""
            echo -e "${GREEN}Proceeding with minimal orchestrator...${NC}"
            USE_VENV=false
        fi
    fi
else
    echo -e "${YELLOW}Virtual environment already exists${NC}"
    USE_VENV=true
fi

# Activate virtual environment if we're using it
if [[ "$USE_VENV" == "true" ]]; then
    source "${VENV_DIR}/bin/activate"
fi

# Upgrade pip
echo -e "${BLUE}Upgrading pip...${NC}"
pip install --quiet --upgrade pip

# Install dependencies
echo -e "${BLUE}Installing dependencies...${NC}"
if [[ -f "${ORCHESTRATOR_DIR}/requirements.txt" ]]; then
    pip install --quiet -r "${ORCHESTRATOR_DIR}/requirements.txt"
    echo -e "${GREEN}✓ Dependencies installed${NC}"
else
    echo -e "${YELLOW}No requirements.txt found, skipping dependency installation${NC}"
fi

# Make orchestrator executable
chmod +x "${ORCHESTRATOR_DIR}/app_orchestrator.py"
chmod +x "${ORCHESTRATOR_DIR}/run.sh"

echo -e "${GREEN}✅ Python App Orchestrator setup complete!${NC}"
echo ""
echo "To run the orchestrator directly:"
echo "  ${ORCHESTRATOR_DIR}/run.sh [--verbose] [--fast]"
echo ""
echo "Or use it via vrooli develop"