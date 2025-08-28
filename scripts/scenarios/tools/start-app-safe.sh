#!/usr/bin/env bash
################################################################################
# Safe App Starter with PID Tracking
# Ensures proper PID management and prevents orphaned processes
################################################################################

set -euo pipefail

APP_NAME="${1:-}"
PHASE="${2:-develop}"

if [[ -z "$APP_NAME" ]]; then
    echo "Usage: $0 <app-name> [phase]"
    echo "Example: $0 document-manager develop"
    exit 1
fi

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${YELLOW}🚀 Safe App Starter${NC}"
echo "========================="

# Step 1: Setup PID directories
echo -e "\n${YELLOW}Step 1: Setting up PID directories...${NC}"
bash /home/matthalloran8/Vrooli/scripts/scenarios/tools/ensure-pid-dirs.sh

# Step 2: Kill any existing instance
echo -e "\n${YELLOW}Step 2: Checking for existing instances...${NC}"

# Kill orchestrator if running (it might conflict)
if pgrep -f "enhanced_orchestrator.py" >/dev/null 2>&1; then
    echo "  Found orchestrator running, stopping..."
    pkill -f "enhanced_orchestrator.py"
    sleep 2
fi

# Check if app is already running
APP_DIR="/home/matthalloran8/generated-apps/$APP_NAME"
if [[ ! -d "$APP_DIR" ]]; then
    echo -e "${RED}Error: App directory not found: $APP_DIR${NC}"
    exit 1
fi

# Stop app if running
if pgrep -f "${APP_NAME}-api" >/dev/null 2>&1 || pgrep -f "$APP_DIR.*server" >/dev/null 2>&1; then
    echo "  Found $APP_NAME processes, stopping..."
    pkill -f "${APP_NAME}-api" 2>/dev/null || true
    pkill -f "$APP_DIR.*server" 2>/dev/null || true
    sleep 2
fi

# Step 3: Clean environment
echo -e "\n${YELLOW}Step 3: Cleaning environment...${NC}"
# Remove old PID files for this app
rm -f "/tmp/vrooli-apps/${APP_NAME}.pid" 2>/dev/null || true
rm -f "/tmp/vrooli-api-pids/${APP_NAME}.pid" 2>/dev/null || true
rm -f "/tmp/vrooli-ui-pids/${APP_NAME}.pid" 2>/dev/null || true
echo "  ✓ Cleaned old PID files"

# Step 4: Set environment
echo -e "\n${YELLOW}Step 4: Setting environment...${NC}"
export VROOLI_ROOT="/home/matthalloran8/Vrooli"
export APP_ROOT="$APP_DIR"
export PM_HOME="$HOME/.vrooli/processes"
export PM_LOG_DIR="$HOME/.vrooli/logs"

# Create process manager directories
mkdir -p "$PM_HOME"
mkdir -p "$PM_LOG_DIR"

echo "  VROOLI_ROOT=$VROOLI_ROOT"
echo "  APP_ROOT=$APP_ROOT"
echo "  PM_HOME=$PM_HOME"

# Step 5: Create PID tracking wrapper for the app
echo -e "\n${YELLOW}Step 5: Starting app with PID tracking...${NC}"

# Create a wrapper script that will track PIDs
WRAPPER_SCRIPT="/tmp/start-${APP_NAME}-wrapper.sh"
cat > "$WRAPPER_SCRIPT" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

APP_NAME="$1"
APP_DIR="$2"
PHASE="$3"

# Function to track PID
track_pid() {
    local name="$1"
    local pid="$2"
    local type="$3"  # api or ui
    
    if [[ "$type" == "api" ]]; then
        echo "$pid" > "/tmp/vrooli-api-pids/${name}.pid"
    elif [[ "$type" == "ui" ]]; then
        echo "$pid" > "/tmp/vrooli-ui-pids/${name}.pid"
    fi
    echo "$pid" > "/tmp/vrooli-apps/${name}.pid"
}

# Export tracking function
export -f track_pid

# Run the manage script with PID tracking hooks
cd "$APP_DIR"
export VROOLI_ROOT="/home/matthalloran8/Vrooli"

# Override background process starting to add PID tracking
export PROCESS_STARTED_HOOK='
if [[ -n "${bg_pid:-}" ]]; then
    if [[ "$name" == *"api"* ]]; then
        track_pid "'$APP_NAME'" "$bg_pid" "api"
    elif [[ "$name" == *"ui"* ]] || [[ "$name" == *"server"* ]]; then
        track_pid "'$APP_NAME'" "$bg_pid" "ui"
    fi
fi
'

# Start the app
exec scripts/manage.sh "$PHASE" 2>&1
EOF

chmod +x "$WRAPPER_SCRIPT"

# Execute the wrapper
echo "Starting $APP_NAME in $PHASE mode..."
if timeout 60 bash "$WRAPPER_SCRIPT" "$APP_NAME" "$APP_DIR" "$PHASE"; then
    echo -e "\n${GREEN}✅ App started successfully!${NC}"
    
    # Show running processes
    echo -e "\n${YELLOW}Running processes:${NC}"
    ps aux | grep -E "${APP_NAME}-api|$APP_DIR.*server" | grep -v grep || true
    
    # Show PID files
    echo -e "\n${YELLOW}PID files created:${NC}"
    ls -la /tmp/vrooli-*pids/${APP_NAME}.pid 2>/dev/null || echo "  No PID files found"
    
    # Show logs location
    echo -e "\n${YELLOW}Logs:${NC}"
    echo "  Process logs: $PM_LOG_DIR/"
    echo "  App logs: $APP_DIR/logs/"
    
    echo -e "\n${GREEN}App is running!${NC}"
    echo "Stop with: bash /home/matthalloran8/Vrooli/scripts/scenarios/tools/cleanup-processes.sh"
else
    echo -e "\n${RED}Failed to start app${NC}"
    exit 1
fi

# Clean up wrapper
rm -f "$WRAPPER_SCRIPT"