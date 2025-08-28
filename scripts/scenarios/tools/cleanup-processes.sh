#!/usr/bin/env bash
################################################################################
# Process Cleanup Tool
# Safely terminates orphaned Vrooli processes and cleans up PID files
################################################################################

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}🧹 Vrooli Process Cleanup Tool${NC}"
echo "================================="

# Track cleanup stats
KILLED_COUNT=0
PID_FILES_CLEANED=0

# Function to safely kill processes by pattern
kill_by_pattern() {
    local pattern="$1"
    local signal="${2:-TERM}"
    local desc="${3:-$pattern}"
    
    local pids=$(pgrep -f "$pattern" 2>/dev/null || true)
    if [[ -n "$pids" ]]; then
        echo -e "${YELLOW}Terminating $desc processes...${NC}"
        for pid in $pids; do
            if kill -$signal "$pid" 2>/dev/null; then
                echo "  ✓ Killed PID $pid"
                ((KILLED_COUNT++)) || true || true
            fi
        done
    fi
}

# Function to clean PID files
clean_pid_files() {
    local pid_patterns=(
        "/tmp/vrooli-*.pid"
        "/tmp/*.pid"
        "/home/matthalloran8/generated-apps/*/.vrooli/*.pid"
        "/home/matthalloran8/Vrooli/.vrooli/*.pid"
    )
    
    echo -e "${YELLOW}Cleaning PID files...${NC}"
    for pattern in "${pid_patterns[@]}"; do
        for pidfile in $pattern; do
            if [[ -f "$pidfile" ]]; then
                # Check if process is still running
                if [[ -r "$pidfile" ]]; then
                    local pid=$(<"$pidfile")
                    if ! kill -0 "$pid" 2>/dev/null; then
                        rm -f "$pidfile"
                        echo "  ✓ Removed stale PID file: $pidfile"
                        ((PID_FILES_CLEANED++)) || true
                    fi
                else
                    # Can't read file, remove it
                    rm -f "$pidfile"
                    echo "  ✓ Removed unreadable PID file: $pidfile"
                    ((PID_FILES_CLEANED++))
                fi
            fi
        done
    done
}

# Function to kill processes holding specific ports
kill_port_holders() {
    local ports=("9500" "9501" "9502" "9503" "9504" "9505")
    
    echo -e "${YELLOW}Checking for processes holding orchestrator ports...${NC}"
    for port in "${ports[@]}"; do
        local pid=$(lsof -ti:$port 2>/dev/null || true)
        if [[ -n "$pid" ]]; then
            echo "  Port $port held by PID $pid"
            if kill -TERM "$pid" 2>/dev/null; then
                echo "  ✓ Killed PID $pid (port $port)"
                ((KILLED_COUNT++)) || true
            fi
        fi
    done
}

echo -e "${YELLOW}Phase 1: Orchestrator Processes${NC}"
echo "---------------------------------"
# Kill orchestrator processes
kill_by_pattern "enhanced_orchestrator\.py" "TERM" "orchestrator"
kill_by_pattern "safe-orchestrator\.py" "TERM" "safe orchestrator"
kill_by_pattern "python.*orchestrator" "TERM" "Python orchestrator"

echo -e "\n${YELLOW}Phase 2: Node.js Processes${NC}"
echo "---------------------------------"
# Kill Node.js UI servers
kill_by_pattern "node.*server\.js" "TERM" "Node.js server"
kill_by_pattern "npm.*start" "TERM" "npm start"
kill_by_pattern "node.*express" "TERM" "Express server"

echo -e "\n${YELLOW}Phase 3: API Binaries${NC}"
echo "---------------------------------"
# Kill Go API binaries for all apps
for app_dir in /home/matthalloran8/generated-apps/*/; do
    if [[ -d "$app_dir" ]]; then
        app_name=$(basename "$app_dir")
        kill_by_pattern "${app_name}-api" "TERM" "$app_name API"
    fi
done

echo -e "\n${YELLOW}Phase 4: Port Cleanup${NC}"
echo "---------------------------------"
kill_port_holders

echo -e "\n${YELLOW}Phase 5: PID File Cleanup${NC}"
echo "---------------------------------"
clean_pid_files

echo -e "\n${YELLOW}Phase 6: Lock File Cleanup${NC}"
echo "---------------------------------"
# Clean lock files
LOCK_FILES_CLEANED=0
for lockfile in /tmp/*.lock /home/matthalloran8/generated-apps/*/.vrooli/*.lock; do
    if [[ -f "$lockfile" ]]; then
        rm -f "$lockfile"
        echo "  ✓ Removed lock file: $lockfile"
        ((LOCK_FILES_CLEANED++)) || true
    fi
done

# Final verification
echo -e "\n${YELLOW}Phase 7: Verification${NC}"
echo "---------------------------------"
REMAINING=$(pgrep -cf "enhanced_orchestrator|node server\.js|npm start" 2>/dev/null || echo "0")

echo -e "\n${GREEN}===== Cleanup Summary =====${NC}"
echo "Processes killed: $KILLED_COUNT"
echo "PID files cleaned: $PID_FILES_CLEANED"
echo "Lock files cleaned: $LOCK_FILES_CLEANED"
echo "Remaining processes: $REMAINING"
# Clean up any extra output from REMAINING
REMAINING=$(echo "$REMAINING" | head -1 | tr -d '\n')

if [[ "$REMAINING" == "0" ]] || [[ "$REMAINING" -eq 0 ]]; then
    echo -e "${GREEN}✅ All processes cleaned successfully!${NC}"
    exit 0
else
    echo -e "${YELLOW}⚠ Some processes may still be running. Run with sudo for complete cleanup.${NC}"
    exit 0
fi