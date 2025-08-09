#!/usr/bin/env bash
# Vrooli Issue Tracker CLI
# Command-line interface for reporting and managing issues

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_EXIT_CODES_FILE}"

# Configuration
TRACKER_API_URL="${ISSUE_TRACKER_URL:-http://localhost:8091/api}"
TRACKER_API_TOKEN="${ISSUE_TRACKER_TOKEN:-}"
CONFIG_FILE="${var_VROOLI_CONFIG_DIR}/.vrooli-tracker-config.json"

# Colors are provided by log.sh utility functions

# Helper functions
vrooli_tracker::show_help() {
    cat << EOF
Vrooli Issue Tracker CLI

Usage: vrooli-tracker <command> [options]

Commands:
  create       Create a new issue
  list         List issues with filters
  get          Get details of a specific issue
  update       Update issue status or details
  investigate  Trigger AI investigation for an issue
  search       Search issues semantically
  stats        Show statistics and metrics
  config       Configure API settings

Options:
  -h, --help   Show this help message

Examples:
  # Create a new issue
  vrooli-tracker create --title "Bug in auth service" --type bug --priority high

  # List open issues
  vrooli-tracker list --status open --priority high

  # Get issue details
  vrooli-tracker get ISSUE-ID

  # Trigger investigation
  vrooli-tracker investigate ISSUE-ID --agent deep-investigator

  # Search for similar issues
  vrooli-tracker search "authentication timeout error"

  # Configure API token
  vrooli-tracker config --token YOUR_API_TOKEN

EOF
}

# Load configuration
vrooli_tracker::load_config() {
    if [ -f "$CONFIG_FILE" ]; then
        TRACKER_API_TOKEN=$(jq -r '.api_token // empty' "$CONFIG_FILE" 2>/dev/null || echo "")
        TRACKER_API_URL=$(jq -r '.api_url // empty' "$CONFIG_FILE" 2>/dev/null || echo "$TRACKER_API_URL")
    fi
    
    if [ -z "$TRACKER_API_TOKEN" ]; then
        log::warning "No API token configured. Run 'vrooli-tracker config --token YOUR_TOKEN'"
    fi
}

# Save configuration
vrooli_tracker::save_config() {
    mkdir -p "$(dirname "$CONFIG_FILE")"
    cat > "$CONFIG_FILE" << EOF
{
  "api_token": "$1",
  "api_url": "$2",
  "app_name": "$3"
}
EOF
    log::success "Configuration saved to $CONFIG_FILE"
}

# API request helper
vrooli_tracker::api_request() {
    local method=$1
    local endpoint=$2
    local data=$3
    
    local auth_header=""
    if [ -n "$TRACKER_API_TOKEN" ]; then
        auth_header="-H \"X-API-Token: $TRACKER_API_TOKEN\""
    fi
    
    local response
    if [ -n "$data" ]; then
        response=$(curl -s -X "$method" \
            "$TRACKER_API_URL/$endpoint" \
            -H "Content-Type: application/json" \
            $auth_header \
            -d "$data")
    else
        response=$(curl -s -X "$method" \
            "$TRACKER_API_URL/$endpoint" \
            -H "Content-Type: application/json" \
            $auth_header)
    fi
    
    echo "$response"
}

# Create issue command
vrooli_tracker::create_issue() {
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
        log::error "Title is required"
        echo "Usage: vrooli-tracker create --title \"Issue title\" [options]"
        exit 1
    fi
    
    if [ -z "$description" ]; then
        description="$title"
    fi
    
    # Build JSON payload
    local json_data=$(jq -n \
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
            errorMessage: $error,
            stackTrace: $stack,
            tags: ($tags | split(","))
        }')
    
    echo "Creating issue..."
    response=$(vrooli_tracker::api_request "POST" "issues" "$json_data")
    
    if [ "$(echo "$response" | jq -r '.success')" = "true" ]; then
        issue_id=$(echo "$response" | jq -r '.issueId')
        log::success "✓ Issue created successfully"
        echo "Issue ID: $issue_id"
        
        # Check for similar issues
        similar=$(echo "$response" | jq -r '.similarIssues // empty')
        if [ -n "$similar" ] && [ "$similar" != "null" ]; then
            log::warning "Similar issues found:"
            echo "$similar" | jq -r '.[] | "  - \(.title) (similarity: \(.similarity))"'
        fi
    else
        log::error "✗ Failed to create issue"
        echo "$response" | jq -r '.message'
        exit 1
    fi
}

# List issues command
vrooli_tracker::list_issues() {
    local status=""
    local priority=""
    local type=""
    local limit="20"
    local search=""
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --status) status="$2"; shift 2 ;;
            --priority) priority="$2"; shift 2 ;;
            --type) type="$2"; shift 2 ;;
            --limit) limit="$2"; shift 2 ;;
            --search) search="$2"; shift 2 ;;
            *) shift ;;
        esac
    done
    
    # Build query parameters
    local query_params=""
    [ -n "$status" ] && query_params="${query_params}&statuses[]=$status"
    [ -n "$priority" ] && query_params="${query_params}&priorities[]=$priority"
    [ -n "$type" ] && query_params="${query_params}&types[]=$type"
    [ -n "$search" ] && query_params="${query_params}&search=$(echo "$search" | jq -sRr @uri)"
    query_params="${query_params}&pageSize=$limit"
    
    # Remove leading &
    query_params="${query_params#&}"
    
    echo "Fetching issues..."
    response=$(vrooli_tracker::api_request "GET" "issues?$query_params")
    
    if [ "$(echo "$response" | jq -r '.success')" = "true" ]; then
        total=$(echo "$response" | jq -r '.totalCount')
        log::info "Found $total issues"
        echo ""
        
        echo "$response" | jq -r '.issues[] | 
            "[\(.priority | ascii_upcase)] \(.title)\n  App: \(.app_name) | Status: \(.status) | Created: \(.created_at)\n  ID: \(.id)\n"'
    else
        log::error "✗ Failed to fetch issues"
        echo "$response" | jq -r '.message'
        exit 1
    fi
}

# Get issue details
vrooli_tracker::get_issue() {
    local issue_id=$1
    
    if [ -z "$issue_id" ]; then
        log::error "Issue ID is required"
        echo "Usage: vrooli-tracker get ISSUE-ID"
        exit 1
    fi
    
    echo "Fetching issue details..."
    response=$(vrooli_tracker::api_request "GET" "issues/$issue_id")
    
    if [ "$(echo "$response" | jq -r '.success')" = "true" ]; then
        echo "$response" | jq -r '.issue | 
            "Title: \(.title)\n" +
            "Status: \(.status)\n" +
            "Priority: \(.priority)\n" +
            "Type: \(.type)\n" +
            "Application: \(.app_name)\n" +
            "Created: \(.created_at)\n" +
            "Reporter: \(.reporter_name // "Unknown")\n" +
            "\nDescription:\n\(.description)\n" +
            if .investigation_report then "\nInvestigation Report:\n\(.investigation_report)\n" else "" end +
            if .root_cause then "Root Cause: \(.root_cause)\n" else "" end +
            if .suggested_fix then "Suggested Fix:\n\(.suggested_fix)\n" else "" end'
    else
        log::error "✗ Failed to fetch issue"
        echo "$response" | jq -r '.message'
        exit 1
    fi
}

# Trigger investigation
vrooli_tracker::investigate_issue() {
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
        log::error "Issue ID is required"
        echo "Usage: vrooli-tracker investigate ISSUE-ID [--agent AGENT-ID]"
        exit 1
    fi
    
    local json_data=$(jq -n \
        --arg id "$issue_id" \
        --arg agent "$agent_id" \
        --arg priority "$priority" \
        '{
            issueId: $id,
            agentId: (if $agent == "" then null else $agent end),
            priority: $priority
        }')
    
    echo "Triggering investigation..."
    response=$(vrooli_tracker::api_request "POST" "investigate" "$json_data")
    
    if [ "$(echo "$response" | jq -r '.success')" = "true" ]; then
        log::success "✓ Investigation started"
        echo "$response" | jq -r '"Run ID: \(.runId)\nInvestigation ID: \(.investigationId)"'
    else
        log::error "✗ Failed to trigger investigation"
        echo "$response" | jq -r '.message'
        exit 1
    fi
}

# Search issues
vrooli_tracker::search_issues() {
    local query="$*"
    
    if [ -z "$query" ]; then
        log::error "Search query is required"
        echo "Usage: vrooli-tracker search \"your search query\""
        exit 1
    fi
    
    echo "Searching for: $query"
    
    local encoded_query=$(echo "$query" | jq -sRr @uri)
    response=$(vrooli_tracker::api_request "GET" "issues/search?q=$encoded_query")
    
    if [ "$(echo "$response" | jq -r '.success')" = "true" ]; then
        count=$(echo "$response" | jq -r '.results | length')
        log::info "Found $count matching issues"
        echo ""
        
        echo "$response" | jq -r '.results[] | 
            "[\(.similarity | . * 100 | floor)% match] \(.title)\n  App: \(.app_name) | Status: \(.status)\n  ID: \(.id)\n"'
    else
        log::error "✗ Search failed"
        echo "$response" | jq -r '.message'
        exit 1
    fi
}

# Show statistics
vrooli_tracker::show_stats() {
    echo "Fetching statistics..."
    response=$(vrooli_tracker::api_request "GET" "stats")
    
    if [ "$(echo "$response" | jq -r '.success')" = "true" ]; then
        log::info "Issue Tracker Statistics"
        echo ""
        echo "$response" | jq -r '.stats | 
            "Total Issues: \(.total_issues)\n" +
            "Open Issues: \(.open_issues)\n" +
            "In Progress: \(.in_progress)\n" +
            "Fixed Today: \(.fixed_today)\n" +
            "Average Resolution Time: \(.avg_resolution_hours) hours\n" +
            "\nTop Applications by Issues:\n" +
            (.top_apps[] | "  - \(.app_name): \(.issue_count) issues")'
    else
        log::error "✗ Failed to fetch statistics"
        exit 1
    fi
}

# Configure command
vrooli_tracker::configure() {
    local token=""
    local url=""
    local app_name=""
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --token) token="$2"; shift 2 ;;
            --url) url="$2"; shift 2 ;;
            --app) app_name="$2"; shift 2 ;;
            *) shift ;;
        esac
    done
    
    if [ -z "$token" ]; then
        log::error "API token is required"
        echo "Usage: vrooli-tracker config --token YOUR_API_TOKEN [--url API_URL]"
        exit 1
    fi
    
    [ -z "$url" ] && url="$TRACKER_API_URL"
    [ -z "$app_name" ] && app_name="unknown"
    
    vrooli_tracker::save_config "$token" "$url" "$app_name"
}

# Main command router
vrooli_tracker::main() {
    vrooli_tracker::load_config
    
    case "$1" in
        create)
            shift
            vrooli_tracker::create_issue "$@"
            ;;
        list)
            shift
            vrooli_tracker::list_issues "$@"
            ;;
        get)
            shift
            vrooli_tracker::get_issue "$@"
            ;;
        investigate)
            shift
            vrooli_tracker::investigate_issue "$@"
            ;;
        search)
            shift
            vrooli_tracker::search_issues "$@"
            ;;
        stats)
            vrooli_tracker::show_stats
            ;;
        config)
            shift
            vrooli_tracker::configure "$@"
            ;;
        -h|--help|help)
            vrooli_tracker::show_help
            ;;
        *)
            log::error "Unknown command: $1"
            echo "Run 'vrooli-tracker --help' for usage information"
            exit 1
            ;;
    esac
}

# Check dependencies
if ! command -v jq &> /dev/null; then
    log::error "jq is required but not installed"
    echo "Install with: apt-get install jq (or your package manager)"
    exit 1
fi

if ! command -v curl &> /dev/null; then
    log::error "curl is required but not installed"
    echo "Install with: apt-get install curl"
    exit 1
fi

# Run main function
vrooli_tracker::main "$@"