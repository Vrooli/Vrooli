#!/bin/bash

# Ensure required resources are running for document-manager scenario
set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${GREEN}Checking required resources for document-manager...${NC}"

# Function to check and start resource if needed
check_and_start_resource() {
    local resource=$1
    local required=$2
    
    echo -n "Checking $resource... "
    
    # Check if resource is running
    status=$(vrooli resource status "$resource" | grep -oP "Running: \K(true|false)")
    
    if [[ "$status" == "true" ]]; then
        echo -e "${GREEN}✓ Running${NC}"
    else
        if [[ "$required" == "true" ]]; then
            echo -e "${YELLOW}Starting...${NC}"
            vrooli resource start "$resource" > /dev/null 2>&1
            
            # Wait for resource to be ready
            sleep 3
            
            # Verify it started
            status=$(vrooli resource status "$resource" | grep -oP "Running: \K(true|false)")
            if [[ "$status" == "true" ]]; then
                echo -e "  ${GREEN}✓ Started successfully${NC}"
            else
                echo -e "  ${RED}✗ Failed to start${NC}"
                exit 1
            fi
        else
            echo -e "${YELLOW}⚠ Not running (optional)${NC}"
        fi
    fi
}

# Check each required resource
check_and_start_resource "postgres" "true"
check_and_start_resource "qdrant" "true"
check_and_start_resource "redis" "true"
check_and_start_resource "n8n" "true"
check_and_start_resource "ollama" "true"
check_and_start_resource "unstructured-io" "false"  # Optional

echo -e "${GREEN}All required resources are running!${NC}"