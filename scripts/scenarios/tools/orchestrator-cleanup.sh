#!/usr/bin/env bash
################################################################################
# Vrooli Orchestrator Emergency Cleanup Script
# 
# This script provides emergency cleanup for runaway orchestrator processes.
# Use this when the orchestrator or its child processes are consuming excessive
# CPU and the normal stop command isn't working.
#
# Usage:
#   orchestrator-cleanup.sh [--force]
#
# Options:
#   --force    Skip confirmation prompts
################################################################################

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

# Configuration
ORCHESTRATOR_HOME="${VROOLI_ORCHESTRATOR_HOME:-$HOME/.vrooli/orchestrator}"
PID_FILE="$ORCHESTRATOR_HOME/orchestrator.pid"
FORCE="${1:-}"

echo -e "${CYAN}=== Vrooli Orchestrator Emergency Cleanup ===${NC}"
echo ""

# Function to find and kill processes
cleanup_processes() {
    local killed_count=0
    
    # Find all vrooli-orchestrator processes
    echo -e "${YELLOW}Searching for orchestrator processes...${NC}"
    local orchestrator_pids=$(ps aux | grep -E "vrooli-orchestrator" | grep -v grep | grep -v cleanup | awk '{print $2}')
    
    if [[ -n "$orchestrator_pids" ]]; then
        echo "Found orchestrator processes: $orchestrator_pids"
        for pid in $orchestrator_pids; do
            echo "  Killing PID $pid..."
            kill -9 "$pid" 2>/dev/null || true
            ((killed_count++))
        done
    fi
    
    # Find bash processes with high CPU (>90%)
    echo -e "${YELLOW}Searching for high-CPU bash processes...${NC}"
    local high_cpu_pids=$(ps aux | awk '$3 > 90 && $11 ~ /bash/ {print $2}')
    
    if [[ -n "$high_cpu_pids" ]]; then
        echo "Found high-CPU bash processes: $high_cpu_pids"
        for pid in $high_cpu_pids; do
            # Check if it's related to Vrooli
            local cmd=$(ps -p "$pid" -o args= 2>/dev/null || echo "unknown")
            if [[ "$cmd" == *"vrooli"* ]] || [[ "$cmd" == "bash" ]] || [[ "$FORCE" == "--force" ]]; then
                echo "  Killing PID $pid (CPU > 90%): $cmd"
                kill -9 "$pid" 2>/dev/null || true
                ((killed_count++))
            fi
        done
    fi
    
    # Find orphaned monitor/command processes
    echo -e "${YELLOW}Searching for orphaned worker processes...${NC}"
    local worker_pids=$(ps aux | grep -E "monitor_processes|process_commands" | grep -v grep | awk '{print $2}')
    
    if [[ -n "$worker_pids" ]]; then
        echo "Found orphaned worker processes: $worker_pids"
        for pid in $worker_pids; do
            echo "  Killing PID $pid..."
            kill -9 "$pid" 2>/dev/null || true
            ((killed_count++))
        done
    fi
    
    echo ""
    echo -e "${GREEN}Killed $killed_count processes${NC}"
}

# Function to clean up files
cleanup_files() {
    echo -e "${YELLOW}Cleaning up orchestrator files...${NC}"
    
    # Remove PID file
    if [[ -f "$PID_FILE" ]]; then
        rm -f "$PID_FILE"
        echo "  Removed PID file"
    fi
    
    # Remove command FIFO
    local fifo_file="$ORCHESTRATOR_HOME/commands.fifo"
    if [[ -p "$fifo_file" ]]; then
        rm -f "$fifo_file"
        echo "  Removed command FIFO"
    fi
    
    # Clean up socket files
    if [[ -d "$ORCHESTRATOR_HOME/sockets" ]]; then
        rm -f "$ORCHESTRATOR_HOME/sockets/"*.pid
        rm -f "$ORCHESTRATOR_HOME/sockets/"*.env
        echo "  Cleaned socket directory"
    fi
    
    # Reset process registry (mark all as stopped)
    local registry_file="$ORCHESTRATOR_HOME/processes.json"
    if [[ -f "$registry_file" ]]; then
        # Backup current registry
        cp "$registry_file" "$registry_file.backup.$(date +%Y%m%d_%H%M%S)"
        echo "  Created registry backup"
        
        # Mark all processes as stopped
        jq '.processes |= with_entries(.value.state = "stopped" | .value.pid = null)' \
            "$registry_file" > "$registry_file.tmp"
        mv "$registry_file.tmp" "$registry_file"
        echo "  Reset process states in registry"
    fi
}

# Function to show current status
show_status() {
    echo -e "${CYAN}Current System Status:${NC}"
    echo ""
    
    # CPU usage
    echo "CPU Usage:"
    top -bn1 | head -5
    echo ""
    
    # Orchestrator processes
    echo "Orchestrator-related processes:"
    ps aux | grep -E "vrooli|orchestrator" | grep -v grep | grep -v cleanup || echo "  None found"
    echo ""
    
    # High CPU processes
    echo "High CPU processes (>50%):"
    ps aux | awk '$3 > 50 {print $2, $3"%", $11}' | head -10 || echo "  None found"
}

# Main execution
main() {
    # Show current status
    show_status
    
    # Confirm action if not forced
    if [[ "$FORCE" != "--force" ]]; then
        echo ""
        echo -e "${YELLOW}⚠️  WARNING: This will forcefully kill orchestrator processes${NC}"
        echo "Press Enter to continue or Ctrl+C to cancel..."
        read -r
    fi
    
    # Perform cleanup
    echo ""
    echo -e "${CYAN}Starting cleanup...${NC}"
    echo ""
    
    cleanup_processes
    cleanup_files
    
    echo ""
    echo -e "${GREEN}✅ Cleanup completed${NC}"
    echo ""
    
    # Show status after cleanup
    echo -e "${CYAN}Status after cleanup:${NC}"
    echo ""
    show_status
    
    echo ""
    echo -e "${CYAN}Next steps:${NC}"
    echo "1. Review the orchestrator logs: tail -100 $ORCHESTRATOR_HOME/orchestrator.log"
    echo "2. Check the registry backup: ls -la $ORCHESTRATOR_HOME/processes.json.backup.*"
    echo "3. Start orchestrator fresh: vrooli-orchestrator start"
}

# Run main function
main