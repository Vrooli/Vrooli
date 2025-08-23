#!/usr/bin/env bash
set -euo pipefail

#############################################################################
# Pre-flight Safety Check for Vrooli Development Environment
# 
# This script performs comprehensive safety checks BEFORE starting the
# orchestrator to prevent fork bombs and system overload.
#############################################################################

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "========================================="
echo "Vrooli Pre-flight Safety Check"
echo "========================================="
echo

# Configuration
# Adjust for systems with many kernel threads (32-core system has ~400 kernel threads)
# Count only user processes, excluding kernel threads
# Your system normally runs ~300 user processes (postgres, resque, etc.)
MAX_USER_PROCESSES=500  # Increased for server with background services
MIN_MEMORY_MB=1000
MAX_APPS_TO_START=10
CRITICAL_PROCESS_LIMIT=1000  # Only panic if we hit this total

# Check 1: Process count (excluding kernel threads)
# Kernel threads show up as [name] in ps output
TOTAL_PROCESSES=$(ps aux | wc -l)
KERNEL_THREADS=$(ps aux | awk '$11 ~ /^\[.*\]$/' | wc -l)
USER_PROCESSES=$((TOTAL_PROCESSES - KERNEL_THREADS))

echo "System process breakdown:"
echo "  Total processes: $TOTAL_PROCESSES"
echo "  Kernel threads: $KERNEL_THREADS"
echo "  User processes: $USER_PROCESSES"

# Check for critical overload first
if [[ $TOTAL_PROCESSES -gt $CRITICAL_PROCESS_LIMIT ]]; then
    echo -e "${RED}CRITICAL: System overload detected ($TOTAL_PROCESSES > $CRITICAL_PROCESS_LIMIT)${NC}"
    echo "This indicates a serious problem like a fork bomb."
    
    # Show what's using resources
    echo
    echo "Top user-space processes:"
    ps aux | grep -vE "\[.*\]" | awk '{print $11}' | grep -v "^$" | sed 's|/.*||' | sort | uniq -c | sort -rn | head -10
    
    exit 1
elif [[ $USER_PROCESSES -gt $MAX_USER_PROCESSES ]]; then
    echo -e "${YELLOW}WARNING: High user process count ($USER_PROCESSES > $MAX_USER_PROCESSES)${NC}"
    echo "This may impact performance but is not critical."
    echo "Consider closing some applications if you experience issues."
    echo
    # Don't exit - just warn
fi

# Check 2: Available memory
if command -v free >/dev/null 2>&1; then
    AVAILABLE_MEM=$(free -m | awk 'NR==2{print $7}')
    echo "Available memory: ${AVAILABLE_MEM}MB"
    
    if [[ $AVAILABLE_MEM -lt $MIN_MEMORY_MB ]]; then
        echo -e "${RED}CRITICAL: Insufficient memory (${AVAILABLE_MEM}MB < ${MIN_MEMORY_MB}MB)${NC}"
        echo "Starting apps now could crash your system."
        exit 1
    fi
fi

# Check 3: Count generated apps
GENERATED_APPS_DIR="${HOME}/generated-apps"
if [[ -d "$GENERATED_APPS_DIR" ]]; then
    APP_COUNT=$(find "$GENERATED_APPS_DIR" -maxdepth 1 -type d ! -name ".*" ! -name "generated-apps" | wc -l)
    echo "Generated apps found: $APP_COUNT"
    
    if [[ $APP_COUNT -gt $MAX_APPS_TO_START ]]; then
        echo -e "${YELLOW}WARNING: Many apps found ($APP_COUNT)${NC}"
        echo "Only the first $MAX_APPS_TO_START apps will be started to prevent overload."
        
        # Create a file to limit apps
        echo "$MAX_APPS_TO_START" > /tmp/vrooli-max-apps-limit
    fi
fi

# Check 4: Kill any zombie orchestrators
if pgrep -f "app_orchestrator" >/dev/null 2>&1; then
    echo -e "${YELLOW}Found existing orchestrator processes, cleaning up...${NC}"
    pkill -9 -f "app_orchestrator" 2>/dev/null || true
    sleep 1
fi

# Check 5: Clean up stale lock files
echo "Cleaning up stale locks..."
rm -f /tmp/vrooli-orchestrator.lock /tmp/vrooli-orchestrator.pid 2>/dev/null || true
rm -rf /tmp/vrooli-apps/*.pid 2>/dev/null || true

# Check 6: CPU load
if command -v uptime >/dev/null 2>&1; then
    LOAD_AVG=$(uptime | awk -F'load average:' '{print $2}' | awk '{print $1}' | tr -d ',')
    LOAD_INT=$(echo "$LOAD_AVG" | cut -d. -f1)
    echo "CPU load average: $LOAD_AVG"
    
    if [[ $LOAD_INT -gt 10 ]]; then
        echo -e "${YELLOW}WARNING: High CPU load detected${NC}"
        echo "System is already under heavy load. Proceed with caution."
        read -p "Continue anyway? (y/n): " response
        if [[ "$response" != "y" ]]; then
            echo "Aborted."
            exit 1
        fi
    fi
fi

echo
echo -e "${GREEN}✅ All pre-flight checks passed!${NC}"
echo "System is ready to start development environment."
echo

# Create marker file to indicate checks passed
echo "$(date +%s)" > /tmp/vrooli-preflight-passed

exit 0