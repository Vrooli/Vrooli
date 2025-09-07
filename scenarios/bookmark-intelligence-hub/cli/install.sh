#!/bin/bash

# Bookmark Intelligence Hub CLI Installation Script
# This script installs the bookmark-intelligence-hub CLI command globally

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_NAME="bookmark-intelligence-hub"
INSTALL_DIR="$HOME/.vrooli/bin"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "🔖 Installing Bookmark Intelligence Hub CLI..."

# Create install directory if it doesn't exist
if [ ! -d "$INSTALL_DIR" ]; then
    echo "Creating install directory: $INSTALL_DIR"
    mkdir -p "$INSTALL_DIR"
fi

# Check if CLI binary exists
CLI_SOURCE="$SCRIPT_DIR/$CLI_NAME"
if [ ! -f "$CLI_SOURCE" ]; then
    echo -e "${YELLOW}Warning: CLI binary not found at $CLI_SOURCE${NC}"
    echo "Creating a basic shell wrapper instead..."
    
    # Create a basic shell wrapper
    cat > "$CLI_SOURCE" << 'EOF'
#!/bin/bash

# Bookmark Intelligence Hub CLI
# This is a placeholder implementation until the Go CLI is built

CLI_NAME="bookmark-intelligence-hub"
API_BASE_URL="${API_BASE_URL:-http://localhost:15200}"

# Function to make API calls
api_call() {
    local method="$1"
    local endpoint="$2"
    local data="$3"
    
    if [ -n "$data" ]; then
        curl -s -X "$method" \
             -H "Content-Type: application/json" \
             -d "$data" \
             "$API_BASE_URL$endpoint"
    else
        curl -s -X "$method" \
             -H "Content-Type: application/json" \
             "$API_BASE_URL$endpoint"
    fi
}

# Function to display help
show_help() {
    echo "Bookmark Intelligence Hub CLI v1.0.0"
    echo ""
    echo "Usage: $CLI_NAME <command> [options]"
    echo ""
    echo "Commands:"
    echo "  status                 Show operational status and health"
    echo "  profile list           List all profiles"
    echo "  profile create <name>  Create a new profile"
    echo "  sync                   Sync bookmarks for all platforms"
    echo "  sync <platform>        Sync bookmarks for specific platform"
    echo "  categories             List categories"
    echo "  actions                List pending actions"
    echo "  actions approve <id>   Approve a specific action"
    echo "  actions reject <id>    Reject a specific action"
    echo "  platforms              List all platforms"
    echo "  version                Show version information"
    echo "  help                   Show this help message"
    echo ""
    echo "Options:"
    echo "  --json                 Output in JSON format"
    echo "  --verbose              Verbose output"
    echo ""
    echo "Environment Variables:"
    echo "  API_BASE_URL          API base URL (default: http://localhost:15200)"
    echo "  BOOKMARK_API_TOKEN    API authentication token"
    echo ""
    echo "Examples:"
    echo "  $CLI_NAME status"
    echo "  $CLI_NAME profile list --json"
    echo "  $CLI_NAME sync reddit"
    echo "  $CLI_NAME actions approve action-123"
}

# Function to show version
show_version() {
    echo "Bookmark Intelligence Hub CLI v1.0.0"
    echo "API Base URL: $API_BASE_URL"
}

# Function to check status
check_status() {
    local format="human"
    if [ "$1" = "--json" ]; then
        format="json"
    fi
    
    echo "Checking Bookmark Intelligence Hub status..."
    response=$(api_call "GET" "/health")
    
    if [ $? -eq 0 ]; then
        if [ "$format" = "json" ]; then
            echo "$response" | jq '.'
        else
            echo "✅ API is healthy"
            echo "Database: $(echo "$response" | jq -r '.database // "unknown"')"
            echo "Version: $(echo "$response" | jq -r '.version // "unknown"')"
        fi
    else
        echo "❌ API is not responding"
        exit 1
    fi
}

# Function to list profiles
list_profiles() {
    local format="human"
    if [ "$1" = "--json" ]; then
        format="json"
    fi
    
    response=$(api_call "GET" "/api/v1/profiles")
    
    if [ $? -eq 0 ]; then
        if [ "$format" = "json" ]; then
            echo "$response"
        else
            echo "Profiles:"
            echo "$response" | jq -r '.[] | "- \(.name) (ID: \(.id))"'
        fi
    else
        echo "❌ Failed to fetch profiles"
        exit 1
    fi
}

# Function to sync bookmarks
sync_bookmarks() {
    local platform="$1"
    local endpoint="/api/v1/bookmarks/sync"
    
    if [ -n "$platform" ]; then
        endpoint="/api/v1/platforms/$platform/sync"
        echo "Syncing bookmarks for $platform..."
    else
        echo "Syncing bookmarks for all platforms..."
    fi
    
    response=$(api_call "POST" "$endpoint" '{}')
    
    if [ $? -eq 0 ]; then
        echo "✅ Sync completed"
        echo "Processed: $(echo "$response" | jq -r '.processed_count // 0') bookmarks"
    else
        echo "❌ Sync failed"
        exit 1
    fi
}

# Function to list categories
list_categories() {
    response=$(api_call "GET" "/api/v1/categories")
    
    if [ $? -eq 0 ]; then
        echo "Categories:"
        echo "$response" | jq -r '.[] | "- \(.name): \(.count) bookmarks"'
    else
        echo "❌ Failed to fetch categories"
        exit 1
    fi
}

# Function to list actions
list_actions() {
    response=$(api_call "GET" "/api/v1/actions")
    
    if [ $? -eq 0 ]; then
        echo "Pending Actions:"
        echo "$response" | jq -r '.[] | "- \(.title) (ID: \(.id))"'
    else
        echo "❌ Failed to fetch actions"
        exit 1
    fi
}

# Function to approve action
approve_action() {
    local action_id="$1"
    
    if [ -z "$action_id" ]; then
        echo "❌ Action ID is required"
        exit 1
    fi
    
    response=$(api_call "POST" "/api/v1/actions/approve" "{\"action_ids\": [\"$action_id\"]}")
    
    if [ $? -eq 0 ]; then
        echo "✅ Action $action_id approved"
    else
        echo "❌ Failed to approve action"
        exit 1
    fi
}

# Function to list platforms
list_platforms() {
    response=$(api_call "GET" "/api/v1/platforms")
    
    if [ $? -eq 0 ]; then
        echo "Platforms:"
        echo "$response" | jq -r '.[] | "- \(.display_name) (\(.name)): \(if .supported then "✅" else "❌" end)"'
    else
        echo "❌ Failed to fetch platforms"
        exit 1
    fi
}

# Main command processing
case "$1" in
    "status")
        check_status "$2"
        ;;
    "profile")
        case "$2" in
            "list")
                list_profiles "$3"
                ;;
            "create")
                echo "❌ Profile creation not yet implemented"
                exit 1
                ;;
            *)
                echo "❌ Unknown profile command: $2"
                echo "Available: list, create"
                exit 1
                ;;
        esac
        ;;
    "sync")
        sync_bookmarks "$2"
        ;;
    "categories")
        list_categories
        ;;
    "actions")
        case "$2" in
            "")
                list_actions
                ;;
            "approve")
                approve_action "$3"
                ;;
            "reject")
                echo "❌ Action rejection not yet implemented"
                exit 1
                ;;
            *)
                echo "❌ Unknown action command: $2"
                echo "Available: list (default), approve, reject"
                exit 1
                ;;
        esac
        ;;
    "platforms")
        list_platforms
        ;;
    "version"|"--version")
        show_version
        ;;
    "help"|"--help"|"")
        show_help
        ;;
    *)
        echo "❌ Unknown command: $1"
        echo "Run '$CLI_NAME help' for available commands"
        exit 1
        ;;
esac
EOF

    # Make the wrapper executable
    chmod +x "$CLI_SOURCE"
fi

# Install/link the CLI
CLI_DEST="$INSTALL_DIR/$CLI_NAME"

# Remove existing installation if it exists
if [ -L "$CLI_DEST" ] || [ -f "$CLI_DEST" ]; then
    echo "Removing existing installation..."
    rm -f "$CLI_DEST"
fi

# Create symlink
echo "Creating symlink: $CLI_DEST -> $CLI_SOURCE"
ln -s "$CLI_SOURCE" "$CLI_DEST"

# Make executable
chmod +x "$CLI_SOURCE"
chmod +x "$CLI_DEST"

# Add to PATH if not already there
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo ""
    echo -e "${YELLOW}Adding $INSTALL_DIR to PATH...${NC}"
    
    # Add to appropriate shell config file
    SHELL_CONFIG=""
    if [ -f "$HOME/.bashrc" ]; then
        SHELL_CONFIG="$HOME/.bashrc"
    elif [ -f "$HOME/.zshrc" ]; then
        SHELL_CONFIG="$HOME/.zshrc"
    elif [ -f "$HOME/.profile" ]; then
        SHELL_CONFIG="$HOME/.profile"
    fi
    
    if [ -n "$SHELL_CONFIG" ]; then
        echo "" >> "$SHELL_CONFIG"
        echo "# Added by Vrooli Bookmark Intelligence Hub" >> "$SHELL_CONFIG"
        echo "export PATH=\"\$PATH:$INSTALL_DIR\"" >> "$SHELL_CONFIG"
        echo "PATH updated in $SHELL_CONFIG"
        echo -e "${YELLOW}Please run: source $SHELL_CONFIG${NC}"
        echo -e "${YELLOW}Or restart your terminal to use the CLI${NC}"
    else
        echo -e "${YELLOW}Please add $INSTALL_DIR to your PATH manually${NC}"
    fi
fi

echo ""
echo -e "${GREEN}✅ Installation completed!${NC}"
echo ""
echo "Usage:"
echo "  $CLI_NAME --help          # Show help"
echo "  $CLI_NAME status          # Check system status"
echo "  $CLI_NAME profile list    # List profiles"
echo "  $CLI_NAME sync            # Sync all bookmarks"
echo ""
echo "Configuration:"
echo "  Set API_BASE_URL environment variable to use a different API endpoint"
echo "  Default: http://localhost:15200"
echo ""

# Test the installation
if command -v "$CLI_NAME" >/dev/null 2>&1; then
    echo -e "${GREEN}✅ CLI is available in PATH${NC}"
    echo "Testing installation..."
    "$CLI_NAME" version
else
    echo -e "${YELLOW}⚠️  CLI not yet available in PATH${NC}"
    echo "You may need to restart your terminal or source your shell config"
    echo ""
    echo "Direct usage: $CLI_DEST --help"
fi