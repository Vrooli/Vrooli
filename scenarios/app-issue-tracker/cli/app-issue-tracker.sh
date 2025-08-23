#!/usr/bin/env bash
# App Issue Tracker CLI - Modern wrapper for API
# Lightweight CLI that interfaces with the Go API backend

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Configuration
API_URL="${ISSUE_TRACKER_API_URL:-http://localhost:8090/api}"
API_TOKEN="${ISSUE_TRACKER_API_TOKEN:-}"
CONFIG_FILE="${HOME}/.config/vrooli/app-issue-tracker-config.json"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

log_success() {
    echo -e "${GREEN}✓${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

log_error() {
    echo -e "${RED}✗${NC} $1"
}

show_help() {
    cat << EOF
App Issue Tracker CLI

Usage: app-issue-tracker <command> [options]

Commands:
  create       Create a new issue
  list         List issues with filters
  get          Get details of a specific issue
  investigate  Trigger AI investigation for an issue
  search       Search issues semantically
  stats        Show statistics and metrics
  config       Configure API settings

Options:
  -h, --help   Show this help message

Examples:
  # Create a new issue
  app-issue-tracker create --title "Bug in auth service" --type bug --priority high

  # List open issues
  app-issue-tracker list --status open --priority high

  # Get issue details
  app-issue-tracker get ISSUE-ID

  # Trigger investigation
  app-issue-tracker investigate ISSUE-ID --agent deep-investigator

  # Search for similar issues
  app-issue-tracker search "authentication timeout error"

  # Configure API token
  app-issue-tracker config --token YOUR_API_TOKEN

EOF
}

# Load configuration
load_config() {
    if [ -f "$CONFIG_FILE" ]; then
        API_TOKEN=$(jq -r '.api_token // empty' "$CONFIG_FILE" 2>/dev/null || echo "")
        API_URL=$(jq -r '.api_url // empty' "$CONFIG_FILE" 2>/dev/null || echo "$API_URL")
    fi
}

# Save configuration
save_config() {
    local token=$1
    local url=$2
    
    mkdir -p "$(dirname "$CONFIG_FILE")"
    cat > "$CONFIG_FILE" << EOF
{
  "api_token": "$token",
  "api_url": "$url"
}
EOF
    log_success "Configuration saved to $CONFIG_FILE"
}

# API request helper
api_request() {
    local method=$1
    local endpoint=$2
    local data=${3:-}
    
    local auth_header=""
    if [ -n "$API_TOKEN" ]; then
        auth_header="-H \"X-API-Token: $API_TOKEN\""
    fi
    
    local response
    if [ -n "$data" ]; then
        response=$(curl -s -X "$method" \
            "$API_URL/$endpoint" \
            -H "Content-Type: application/json" \
            $auth_header \
            -d "$data")
    else
        response=$(curl -s -X "$method" \
            "$API_URL/$endpoint" \
            -H "Content-Type: application/json" \
            $auth_header)
    fi
    
    echo "$response"
}

# Create issue command
create_issue() {
    local title=""
    local description=""
    local type="bug"
    local priority="medium"
    local error_message=""
    local stack_trace=""
    local tags=""
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --title) title="$2"; shift 2 ;;
            --description) description="$2"; shift 2 ;;
            --type) type="$2"; shift 2 ;;
            --priority) priority="$2"; shift 2 ;;
            --error) error_message="$2"; shift 2 ;;
            --stack) stack_trace="$2"; shift 2 ;;
            --tags) tags="$2"; shift 2 ;;
            *) shift ;;
        esac
    done
    
    if [ -z "$title" ]; then
        log_error "Title is required"
        echo "Usage: app-issue-tracker create --title \"Issue title\" [options]"
        exit 1
    fi
    
    if [ -z "$description" ]; then
        description="$title"
    fi
    
    # Build JSON payload
    local json_data
    json_data=$(jq -n \
        --arg title "$title" \
        --arg desc "$description" \
        --arg type "$type" \
        --arg priority "$priority" \
        --arg error "$error_message" \
        --arg stack "$stack_trace" \
        --arg tags "$tags" \
        '{
            title: $title,
            description: $desc,
            type: $type,
            priority: $priority,
            error_message: $error,
            stack_trace: $stack,
            tags: ($tags | split(",") | map(select(. != "")))
        }')
    
    log_info "Creating issue..."
    response=$(api_request "POST" "issues" "$json_data")
    
    if [ "$(echo "$response" | jq -r '.success')" = "true" ]; then
        issue_id=$(echo "$response" | jq -r '.data.issue_id')
        log_success "Issue created successfully"
        echo "Issue ID: $issue_id"
    else
        log_error "Failed to create issue"
        echo "$response" | jq -r '.message // "Unknown error"'
        exit 1
    fi
}

# List issues command
list_issues() {
    local status=""
    local priority=""
    local type=""
    local limit="20"
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --status) status="$2"; shift 2 ;;
            --priority) priority="$2"; shift 2 ;;
            --type) type="$2"; shift 2 ;;
            --limit) limit="$2"; shift 2 ;;
            *) shift ;;
        esac
    done
    
    # Build query parameters
    local query_params=""
    [ -n "$status" ] && query_params="${query_params}&status=$status"
    [ -n "$priority" ] && query_params="${query_params}&priority=$priority"
    [ -n "$type" ] && query_params="${query_params}&type=$type"
    query_params="${query_params}&limit=$limit"
    
    # Remove leading &
    query_params="${query_params#&}"
    
    log_info "Fetching issues..."
    response=$(api_request "GET" "issues?$query_params")
    
    if [ "$(echo "$response" | jq -r '.success')" = "true" ]; then
        count=$(echo "$response" | jq -r '.data.count')
        log_info "Found $count issues"
        echo ""
        
        echo "$response" | jq -r '.data.issues[] | 
            "[\(.priority | ascii_upcase)] \(.title)\n  Status: \(.status) | Type: \(.type) | Created: \(.created_at)\n  ID: \(.id)\n"'
    else
        log_error "Failed to fetch issues"
        echo "$response" | jq -r '.message // "Unknown error"'
        exit 1
    fi
}

# Search issues command
search_issues() {
    local query="$*"
    
    if [ -z "$query" ]; then
        log_error "Search query is required"
        echo "Usage: app-issue-tracker search \"your search query\""
        exit 1
    fi
    
    log_info "Searching for: $query"
    
    local encoded_query
    encoded_query=$(echo "$query" | jq -sRr @uri)
    response=$(api_request "GET" "issues/search?q=$encoded_query")
    
    if [ "$(echo "$response" | jq -r '.success')" = "true" ]; then
        count=$(echo "$response" | jq -r '.data.count')
        log_info "Found $count matching issues"
        echo ""
        
        echo "$response" | jq -r '.data.results[] | 
            "[\(.similarity | . * 100 | floor)% match] \(.title)\n  Status: \(.status) | Type: \(.type)\n  ID: \(.id)\n"'
    else
        log_error "Search failed"
        echo "$response" | jq -r '.message // "Unknown error"'
        exit 1
    fi
}

# Trigger investigation
investigate_issue() {
    local issue_id=$1
    shift
    
    local agent_id=""
    local priority="normal"
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --agent) agent_id="$2"; shift 2 ;;
            --priority) priority="$2"; shift 2 ;;
            *) shift ;;
        esac
    done
    
    if [ -z "$issue_id" ]; then
        log_error "Issue ID is required"
        echo "Usage: app-issue-tracker investigate ISSUE-ID [--agent AGENT-ID]"
        exit 1
    fi
    
    local json_data
    json_data=$(jq -n \
        --arg id "$issue_id" \
        --arg agent "$agent_id" \
        --arg priority "$priority" \
        '{
            issue_id: $id,
            agent_id: (if $agent == "" then null else $agent end),
            priority: $priority
        }')
    
    log_info "Triggering investigation..."
    response=$(api_request "POST" "investigate" "$json_data")
    
    if [ "$(echo "$response" | jq -r '.success')" = "true" ]; then
        log_success "Investigation started"
        echo "$response" | jq -r '.data | "Run ID: \(.run_id)\nInvestigation ID: \(.investigation_id)"'
    else
        log_error "Failed to trigger investigation"
        echo "$response" | jq -r '.message // "Unknown error"'
        exit 1
    fi
}

# Show statistics
show_stats() {
    log_info "Fetching statistics..."
    response=$(api_request "GET" "stats")
    
    if [ "$(echo "$response" | jq -r '.success')" = "true" ]; then
        log_info "Issue Tracker Statistics"
        echo ""
        echo "$response" | jq -r '.data.stats | 
            "Total Issues: \(.total_issues)\n" +
            "Open Issues: \(.open_issues)\n" +
            "In Progress: \(.in_progress)\n" +
            "Fixed Today: \(.fixed_today)\n" +
            "Average Resolution Time: \(.avg_resolution_hours) hours\n" +
            "\nTop Applications by Issues:\n" +
            (.top_apps[] | "  - \(.app_name): \(.issue_count) issues")'
    else
        log_error "Failed to fetch statistics"
        exit 1
    fi
}

# Configure command
configure() {
    local token=""
    local url=""
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --token) token="$2"; shift 2 ;;
            --url) url="$2"; shift 2 ;;
            *) shift ;;
        esac
    done
    
    if [ -z "$token" ]; then
        log_error "API token is required"
        echo "Usage: app-issue-tracker config --token YOUR_API_TOKEN [--url API_URL]"
        exit 1
    fi
    
    [ -z "$url" ] && url="$API_URL"
    
    save_config "$token" "$url"
}

# Main command router
main() {
    load_config
    
    case "${1:-}" in
        create)
            shift
            create_issue "$@"
            ;;
        list)
            shift
            list_issues "$@"
            ;;
        search)
            shift
            search_issues "$@"
            ;;
        investigate)
            shift
            investigate_issue "$@"
            ;;
        stats)
            show_stats
            ;;
        config)
            shift
            configure "$@"
            ;;
        -h|--help|help)
            show_help
            ;;
        "")
            show_help
            ;;
        *)
            log_error "Unknown command: $1"
            echo "Run 'app-issue-tracker --help' for usage information"
            exit 1
            ;;
    esac
}

# Check dependencies
if ! command -v jq &> /dev/null; then
    log_error "jq is required but not installed"
    echo "Install with: apt-get install jq (or your package manager)"
    exit 1
fi

if ! command -v curl &> /dev/null; then
    log_error "curl is required but not installed"
    echo "Install with: apt-get install curl"
    exit 1
fi

# Run main function
main "$@"