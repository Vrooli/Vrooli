#!/usr/bin/env bash
################################################################################
# Test script to verify orchestrator doesn't create runaway processes
# This simulates rapid start/stop cycles that previously caused CPU overload
################################################################################

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ORCHESTRATOR="$SCRIPT_DIR/vrooli-orchestrator.sh"

echo -e "${CYAN}=== Orchestrator Stress Test ===${NC}"
echo "This test will start/stop the orchestrator multiple times"
echo "and verify no runaway processes are created."
echo ""

# Function to check for high CPU processes
check_cpu() {
    local high_cpu_count=$(ps aux | awk '$3 > 90 && $11 ~ /bash/ {print $2}' | wc -l)
    if [[ $high_cpu_count -gt 0 ]]; then
        echo -e "${RED}❌ Found $high_cpu_count high-CPU bash processes!${NC}"
        ps aux | awk '$3 > 90 && $11 ~ /bash/' | head -5
        return 1
    else
        echo -e "${GREEN}✓ No high-CPU processes found${NC}"
        return 0
    fi
}

# Function to count orchestrator processes
count_orchestrators() {
    ps aux | grep -E "vrooli-orchestrator.*start" | grep -v grep | wc -l
}

# Clean up before test
echo -e "${YELLOW}Cleaning up any existing processes...${NC}"
"$SCRIPT_DIR/orchestrator-cleanup.sh" --force >/dev/null 2>&1 || true
sleep 2

# Initial CPU check
echo -e "\n${CYAN}Initial system check:${NC}"
check_cpu || true

# Test rapid start/stop cycles
echo -e "\n${CYAN}Starting rapid cycle test...${NC}"
for i in {1..5}; do
    echo -e "\n${YELLOW}Cycle $i/5${NC}"
    
    # Start orchestrator
    echo "  Starting orchestrator..."
    "$ORCHESTRATOR" start
    sleep 2
    
    # Check status
    orchestrator_count=$(count_orchestrators)
    echo "  Orchestrator instances running: $orchestrator_count"
    
    if [[ $orchestrator_count -gt 1 ]]; then
        echo -e "  ${RED}⚠️  Multiple orchestrator instances detected!${NC}"
    fi
    
    # Stop orchestrator
    echo "  Stopping orchestrator..."
    "$ORCHESTRATOR" stop
    sleep 2
    
    # Check for runaway processes
    echo "  Checking for runaway processes..."
    if ! check_cpu; then
        echo -e "${RED}Test FAILED at cycle $i${NC}"
        echo "Runaway processes detected!"
        exit 1
    fi
done

# Test concurrent start attempts
echo -e "\n${CYAN}Testing concurrent start prevention...${NC}"
"$ORCHESTRATOR" start
sleep 1

# Try to start again (should fail)
echo "Attempting to start second instance..."
if "$ORCHESTRATOR" start 2>&1 | grep -q "already running"; then
    echo -e "${GREEN}✓ Second instance correctly prevented${NC}"
else
    echo -e "${RED}❌ Second instance was not prevented!${NC}"
fi

# Clean up
"$ORCHESTRATOR" stop
sleep 2

# Final check
echo -e "\n${CYAN}Final system check:${NC}"
if check_cpu; then
    echo -e "\n${GREEN}🎉 TEST PASSED!${NC}"
    echo "Orchestrator can be safely started/stopped without creating runaway processes."
else
    echo -e "\n${RED}❌ TEST FAILED!${NC}"
    echo "Runaway processes were created during the test."
    exit 1
fi

# Show final process count
echo -e "\n${CYAN}Final stats:${NC}"
echo "Orchestrator processes: $(count_orchestrators)"
echo "High-CPU bash processes: $(ps aux | awk '$3 > 90 && $11 ~ /bash/' | wc -l)"
echo "Total bash processes: $(pgrep bash | wc -l)"