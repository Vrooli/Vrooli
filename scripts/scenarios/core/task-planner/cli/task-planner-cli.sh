#!/bin/bash

# Task Planner CLI
# Command-line interface for the AI-powered task management system

set -euo pipefail

# Basic path detection without dependency on var.sh
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Safe remove function (lightweight version)
safe_remove() {
    local file="$1"
    if [[ -f "$file" ]]; then
        rm -f "$file"
    fi
}

# Configuration
CLI_VERSION="1.0.0"
CLI_NAME="task-planner"
DEFAULT_API_BASE="${TASK_PLANNER_API_BASE:-http://localhost:8090/api}"
CONFIG_DIR="${HOME}/.task-planner"
CONFIG_FILE="${CONFIG_DIR}/config.json"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

# Helper functions
log() { echo -e "${BLUE}$*${NC}"; }
success() { echo -e "${GREEN}✅ $*${NC}"; }
warn() { echo -e "${YELLOW}⚠️  $*${NC}"; }
error() { echo -e "${RED}❌ $*${NC}" >&2; }
bold() { echo -e "${BOLD}$*${NC}"; }

# Ensure config directory exists
ensure_config_dir() {
    mkdir -p "$CONFIG_DIR"
}

# Load configuration
load_config() {
    ensure_config_dir
    
    if [[ -f "$CONFIG_FILE" ]]; then
        API_BASE=$(jq -r '.api_base // "'$DEFAULT_API_BASE'"' "$CONFIG_FILE" 2>/dev/null || echo "$DEFAULT_API_BASE")
        DEFAULT_APP_ID=$(jq -r '.default_app_id // null' "$CONFIG_FILE" 2>/dev/null || echo "")
        DEFAULT_FORMAT=$(jq -r '.default_format // "table"' "$CONFIG_FILE" 2>/dev/null || echo "table")
    else
        API_BASE="$DEFAULT_API_BASE"
        DEFAULT_APP_ID=""
        DEFAULT_FORMAT="table"
        create_default_config
    fi
}

# Create default configuration
create_default_config() {
    cat > "$CONFIG_FILE" << EOF
{
  "api_base": "$DEFAULT_API_BASE",
  "default_format": "table",
  "default_app_id": null,
  "created_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
}
EOF
}

# Make API request
api_request() {
    local method="$1"
    local endpoint="$2"
    local data="${3:-}"
    
    local curl_args=("-s" "-X" "$method")
    curl_args+=("-H" "Content-Type: application/json")
    
    if [[ -n "$data" ]]; then
        curl_args+=("-d" "$data")
    fi
    
    local url="${API_BASE}${endpoint}"
    curl_args+=("$url")
    
    local response
    response=$(curl "${curl_args[@]}" 2>/dev/null) || {
        error "Failed to connect to API at $API_BASE"
        return 1
    }
    
    echo "$response"
}

# Format output based on format preference
format_output() {
    local data="$1"
    local format="${2:-$DEFAULT_FORMAT}"
    
    case "$format" in
        "json")
            echo "$data" | jq '.'
            ;;
        "table")
            format_table "$data"
            ;;
        "csv")
            format_csv "$data"
            ;;
        *)
            echo "$data" | jq '.'
            ;;
    esac
}

# Format data as table
format_table() {
    local data="$1"
    
    # Check if this is a task list
    if echo "$data" | jq -e '.tasks' > /dev/null 2>&1; then
        echo "$data" | jq -r '
            ["ID", "TITLE", "STATUS", "PRIORITY", "APP"] as $headers |
            $headers, 
            (.tasks[] | [
                .id[0:8],
                .title[0:40],
                .status,
                .priority,
                .app_name
            ]) |
            @tsv
        ' | column -t -s $'\t'
    # Check if this is an app list
    elif echo "$data" | jq -e '.apps' > /dev/null 2>&1; then
        echo "$data" | jq -r '
            ["ID", "NAME", "DISPLAY_NAME", "TOTAL_TASKS", "COMPLETED"] as $headers |
            $headers,
            (.apps[] | [
                .id[0:8],
                .name,
                .display_name,
                .total_tasks,
                .completed_tasks
            ]) |
            @tsv
        ' | column -t -s $'\t'
    else
        echo "$data" | jq '.'
    fi
}

# Format data as CSV
format_csv() {
    local data="$1"
    
    if echo "$data" | jq -e '.tasks' > /dev/null 2>&1; then
        echo "id,title,status,priority,app_name,created_at"
        echo "$data" | jq -r '.tasks[] | [.id, .title, .status, .priority, .app_name, .created_at] | @csv'
    elif echo "$data" | jq -e '.apps' > /dev/null 2>&1; then
        echo "id,name,display_name,total_tasks,completed_tasks,created_at"
        echo "$data" | jq -r '.apps[] | [.id, .name, .display_name, .total_tasks, .completed_tasks, .created_at] | @csv'
    else
        echo "$data" | jq -r 'to_entries[] | [.key, .value] | @csv'
    fi
}

# Show usage information
show_usage() {
    bold "Task Planner CLI v$CLI_VERSION"
    echo "AI-powered task management from the command line"
    echo ""
    echo "Usage: $CLI_NAME <command> [options] [args]"
    echo ""
    echo "Commands:"
    echo "  parse <text>              Parse unstructured text into tasks"
    echo "  list [options]            List tasks with optional filtering"
    echo "  show <task-id>            Show detailed task information"
    echo "  research <task-id>        Research and plan a task"
    echo "  implement <task-id>       Implement a staged task"
    echo "  status <task-id> <status> Update task status"
    echo "  search <query>            Search tasks"
    echo "  apps                      Manage applications"
    echo "  config                    Manage configuration"
    echo "  help                      Show this help message"
    echo ""
    echo "Global Options:"
    echo "  --format FORMAT           Output format (json, table, csv)"
    echo "  --app-id ID              Filter by app ID"
    echo "  --api-base URL           Override API base URL"
    echo ""
    echo "Examples:"
    echo "  $CLI_NAME parse 'Add login page'"
    echo "  $CLI_NAME list --status=backlog --priority=high"
    echo "  $CLI_NAME research abc123"
    echo "  $CLI_NAME apps list"
    echo ""
}

# Parse command line arguments
parse_args() {
    COMMAND=""
    ARGS=()
    FORMAT="$DEFAULT_FORMAT"
    APP_ID="$DEFAULT_APP_ID"
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --format)
                FORMAT="$2"
                shift 2
                ;;
            --app-id)
                APP_ID="$2"
                shift 2
                ;;
            --api-base)
                API_BASE="$2"
                shift 2
                ;;
            --help|-h)
                show_usage
                exit 0
                ;;
            --version|-v)
                echo "$CLI_VERSION"
                exit 0
                ;;
            -*)
                error "Unknown option: $1"
                exit 1
                ;;
            *)
                if [[ -z "$COMMAND" ]]; then
                    COMMAND="$1"
                else
                    ARGS+=("$1")
                fi
                shift
                ;;
        esac
    done
}

# Parse text command
cmd_parse() {
    local text="${ARGS[0]:-}"
    local file_input=false
    
    # Check for --file option
    if [[ "$text" == "--file" ]]; then
        file_input=true
        local filename="${ARGS[1]:-}"
        if [[ -z "$filename" || ! -f "$filename" ]]; then
            error "File not found or not specified"
            exit 1
        fi
        text=$(cat "$filename")
    fi
    
    if [[ -z "$text" ]]; then
        error "Text input required. Use: $CLI_NAME parse 'your text' or $CLI_NAME parse --file filename.txt"
        exit 1
    fi
    
    if [[ -z "$APP_ID" ]]; then
        error "App ID required. Set with --app-id option or configure default app"
        exit 1
    fi
    
    log "Parsing text into tasks..."
    
    local payload
    payload=$(jq -n \
        --arg app_id "$APP_ID" \
        --arg raw_text "$text" \
        --arg api_token "cli_token" \
        --arg submitted_by "cli" \
        '{
            app_id: $app_id,
            raw_text: $raw_text,
            api_token: $api_token,
            submitted_by: $submitted_by
        }'
    )
    
    local response
    response=$(api_request "POST" "/parse-text" "$payload") || exit 1
    
    if echo "$response" | jq -e '.success' > /dev/null 2>&1; then
        local tasks_created
        tasks_created=$(echo "$response" | jq -r '.tasks_created')
        success "Created $tasks_created tasks"
        
        if echo "$response" | jq -e '.tasks' > /dev/null 2>&1; then
            format_output "$response" "$FORMAT"
        fi
    else
        local error_msg
        error_msg=$(echo "$response" | jq -r '.error // "Unknown error"')
        error "Parsing failed: $error_msg"
        exit 1
    fi
}

# List tasks command
cmd_list() {
    local status=""
    local priority=""
    local tags=""
    local limit=50
    local offset=0
    
    # Parse list-specific options
    local i=0
    while [[ $i -lt ${#ARGS[@]} ]]; do
        case "${ARGS[$i]}" in
            --status)
                status="${ARGS[$((i+1))]}"
                i=$((i+2))
                ;;
            --priority)
                priority="${ARGS[$((i+1))]}"
                i=$((i+2))
                ;;
            --tags)
                tags="${ARGS[$((i+1))]}"
                i=$((i+2))
                ;;
            --limit)
                limit="${ARGS[$((i+1))]}"
                i=$((i+2))
                ;;
            --offset)
                offset="${ARGS[$((i+1))]}"
                i=$((i+2))
                ;;
            *)
                i=$((i+1))
                ;;
        esac
    done
    
    log "Fetching tasks..."
    
    # Build query parameters
    local query_params=""
    [[ -n "$APP_ID" ]] && query_params+="app_id=${APP_ID}&"
    [[ -n "$status" ]] && query_params+="status=${status}&"
    [[ -n "$priority" ]] && query_params+="priority=${priority}&"
    [[ -n "$tags" ]] && query_params+="tags=${tags}&"
    query_params+="limit=${limit}&offset=${offset}"
    
    local response
    response=$(api_request "GET" "/tasks?${query_params}") || exit 1
    
    format_output "$response" "$FORMAT"
}

# Show task details command
cmd_show() {
    local task_id="${ARGS[0]:-}"
    
    if [[ -z "$task_id" ]]; then
        error "Task ID required"
        exit 1
    fi
    
    log "Fetching task details..."
    
    local response
    response=$(api_request "GET" "/tasks/${task_id}") || exit 1
    
    if echo "$response" | jq -e '.error' > /dev/null 2>&1; then
        local error_msg
        error_msg=$(echo "$response" | jq -r '.error')
        error "$error_msg"
        exit 1
    fi
    
    format_output "$response" "$FORMAT"
}

# Research task command
cmd_research() {
    local task_id="${ARGS[0]:-}"
    local force_refresh=false
    
    if [[ -z "$task_id" ]]; then
        error "Task ID required"
        exit 1
    fi
    
    # Check for --force option
    if [[ "${ARGS[1]:-}" == "--force" ]]; then
        force_refresh=true
    fi
    
    log "Researching task..."
    
    local payload
    payload=$(jq -n --argjson force_refresh "$force_refresh" '{force_refresh: $force_refresh}')
    
    local response
    response=$(api_request "POST" "/tasks/${task_id}/research" "$payload") || exit 1
    
    if echo "$response" | jq -e '.success' > /dev/null 2>&1; then
        success "Task research completed"
        format_output "$response" "$FORMAT"
    else
        local error_msg
        error_msg=$(echo "$response" | jq -r '.error // "Unknown error"')
        error "Research failed: $error_msg"
        exit 1
    fi
}

# Implement task command
cmd_implement() {
    local task_id="${ARGS[0]:-}"
    local override_staging=false
    local implementation_notes=""
    
    if [[ -z "$task_id" ]]; then
        error "Task ID required"
        exit 1
    fi
    
    # Parse implementation options
    local i=1
    while [[ $i -lt ${#ARGS[@]} ]]; do
        case "${ARGS[$i]}" in
            --force)
                override_staging=true
                i=$((i+1))
                ;;
            --notes)
                implementation_notes="${ARGS[$((i+1))]}"
                i=$((i+2))
                ;;
            *)
                i=$((i+1))
                ;;
        esac
    done
    
    log "Implementing task..."
    
    local payload
    payload=$(jq -n \
        --argjson override_staging "$override_staging" \
        --arg implementation_notes "$implementation_notes" \
        '{
            override_staging: $override_staging,
            implementation_notes: $implementation_notes
        }'
    )
    
    local response
    response=$(api_request "POST" "/tasks/${task_id}/implement" "$payload") || exit 1
    
    if echo "$response" | jq -e '.success' > /dev/null 2>&1; then
        success "Task implementation completed"
        format_output "$response" "$FORMAT"
    else
        local error_msg
        error_msg=$(echo "$response" | jq -r '.error // "Unknown error"')
        error "Implementation failed: $error_msg"
        exit 1
    fi
}

# Update task status command
cmd_status() {
    local task_id="${ARGS[0]:-}"
    local new_status="${ARGS[1]:-}"
    local reason="${ARGS[2]:-Manual status update via CLI}"
    
    if [[ -z "$task_id" || -z "$new_status" ]]; then
        error "Task ID and new status required"
        exit 1
    fi
    
    log "Updating task status..."
    
    local payload
    payload=$(jq -n \
        --arg to_status "$new_status" \
        --arg reason "$reason" \
        '{
            to_status: $to_status,
            reason: $reason
        }'
    )
    
    local response
    response=$(api_request "PUT" "/tasks/${task_id}/status" "$payload") || exit 1
    
    if echo "$response" | jq -e '.success' > /dev/null 2>&1; then
        success "Task status updated"
        format_output "$response" "$FORMAT"
    else
        local error_msg
        error_msg=$(echo "$response" | jq -r '.error // "Unknown error"')
        error "Status update failed: $error_msg"
        exit 1
    fi
}

# Search tasks command
cmd_search() {
    local query="${ARGS[0]:-}"
    
    if [[ -z "$query" ]]; then
        error "Search query required"
        exit 1
    fi
    
    log "Searching tasks..."
    
    # Simple text search for now
    local query_params="q=$(printf '%s' "$query" | jq -sRr @uri)"
    [[ -n "$APP_ID" ]] && query_params+="&app_id=${APP_ID}"
    
    local response
    response=$(api_request "GET" "/search/text?${query_params}") || {
        # Fallback to regular list with grep-like filtering
        warn "Semantic search not available, using basic filtering"
        response=$(api_request "GET" "/tasks") || exit 1
        response=$(echo "$response" | jq --arg q "$query" '
            .tasks |= map(select(.title | contains($q) or .description | contains($q))) |
            .count = (.tasks | length)
        ')
    }
    
    format_output "$response" "$FORMAT"
}

# Apps management command
cmd_apps() {
    local subcommand="${ARGS[0]:-list}"
    
    case "$subcommand" in
        "list")
            log "Fetching applications..."
            local response
            response=$(api_request "GET" "/apps") || exit 1
            format_output "$response" "$FORMAT"
            ;;
        "create")
            local name="${ARGS[1]:-}"
            local display_name="${ARGS[2]:-$name}"
            
            if [[ -z "$name" ]]; then
                error "App name required: $CLI_NAME apps create <name> [display_name]"
                exit 1
            fi
            
            log "Creating application..."
            local payload
            payload=$(jq -n --arg name "$name" --arg display_name "$display_name" '{
                name: $name,
                display_name: $display_name
            }')
            
            local response
            response=$(api_request "POST" "/apps" "$payload") || exit 1
            
            if echo "$response" | jq -e '.app' > /dev/null 2>&1; then
                success "Application created"
                format_output "$response" "$FORMAT"
            else
                local error_msg
                error_msg=$(echo "$response" | jq -r '.error // "Unknown error"')
                error "Failed to create app: $error_msg"
                exit 1
            fi
            ;;
        "show")
            local app_id="${ARGS[1]:-}"
            if [[ -z "$app_id" ]]; then
                error "App ID required"
                exit 1
            fi
            
            local response
            response=$(api_request "GET" "/apps/${app_id}") || exit 1
            format_output "$response" "$FORMAT"
            ;;
        *)
            error "Unknown apps subcommand: $subcommand"
            echo "Available: list, create, show"
            exit 1
            ;;
    esac
}

# Configuration management command
cmd_config() {
    local subcommand="${ARGS[0]:-show}"
    
    case "$subcommand" in
        "show")
            if [[ -f "$CONFIG_FILE" ]]; then
                bold "Configuration ($CONFIG_FILE):"
                cat "$CONFIG_FILE" | jq '.'
            else
                warn "No configuration file found"
            fi
            ;;
        "set")
            local key="${ARGS[1]:-}"
            local value="${ARGS[2]:-}"
            
            if [[ -z "$key" || -z "$value" ]]; then
                error "Key and value required: $CLI_NAME config set <key> <value>"
                exit 1
            fi
            
            ensure_config_dir
            
            # Update configuration
            local updated_config
            if [[ -f "$CONFIG_FILE" ]]; then
                updated_config=$(cat "$CONFIG_FILE" | jq --arg key "$key" --arg value "$value" '.[$key] = $value')
            else
                updated_config=$(jq -n --arg key "$key" --arg value "$value" '{($key): $value}')
            fi
            
            echo "$updated_config" > "$CONFIG_FILE"
            success "Configuration updated: $key = $value"
            ;;
        "reset")
            safe_remove "$CONFIG_FILE"
            create_default_config
            success "Configuration reset to defaults"
            ;;
        *)
            error "Unknown config subcommand: $subcommand"
            echo "Available: show, set, reset"
            exit 1
            ;;
    esac
}

# Main execution
main() {
    # Check dependencies
    if ! command -v jq &> /dev/null; then
        error "jq is required. Please install it first."
        exit 1
    fi
    
    if ! command -v curl &> /dev/null; then
        error "curl is required. Please install it first."
        exit 1
    fi
    
    # Load configuration
    load_config
    
    # Parse arguments
    parse_args "$@"
    
    # Execute command
    case "$COMMAND" in
        "parse")
            cmd_parse
            ;;
        "list"|"ls")
            cmd_list
            ;;
        "show"|"get")
            cmd_show
            ;;
        "research")
            cmd_research
            ;;
        "implement")
            cmd_implement
            ;;
        "status")
            cmd_status
            ;;
        "search")
            cmd_search
            ;;
        "apps")
            cmd_apps
            ;;
        "config")
            cmd_config
            ;;
        "help"|"")
            show_usage
            ;;
        *)
            error "Unknown command: $COMMAND"
            echo "Use '$CLI_NAME help' for usage information"
            exit 1
            ;;
    esac
}

# Execute main function
main "$@"