#!/usr/bin/env bash
# App Issue Tracker CLI v2 - File-Based Issue Management
# Supports both API mode and direct file operations

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/scenarios/app-issue-tracker/cli"
ISSUES_DIR="${APP_ROOT}/scenarios/app-issue-tracker/issues"

# Configuration
API_URL="${ISSUE_TRACKER_API_URL:-http://localhost:8090/api}"
API_TOKEN="${ISSUE_TRACKER_API_TOKEN:-}"
CONFIG_FILE="${HOME}/.config/vrooli/app-issue-tracker-config.json"
MODE="${APP_ISSUE_TRACKER_MODE:-auto}"  # auto|api|file

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[1;36m'
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

# Determine operation mode
detect_mode() {
    if [[ "$MODE" == "file" ]]; then
        echo "file"
        return
    fi
    
    if [[ "$MODE" == "api" ]]; then
        echo "api"
        return
    fi
    
    # Auto-detect: try API first, fallback to file
    if curl -s "$API_URL/../health" >/dev/null 2>&1; then
        echo "api"
    else
        echo "file"
    fi
}

show_help() {
    cat << EOF
${CYAN}🐛 App Issue Tracker CLI v2${NC}
${CYAN}File-Based Issue Management${NC}

Usage: app-issue-tracker <command> [options]

Commands:
  ${GREEN}create${NC}       Create a new issue
  ${GREEN}list${NC}         List issues with filters  
  ${GREEN}show${NC}         Show full details of an issue
  ${GREEN}move${NC}         Move issue between status folders
  ${GREEN}investigate${NC}  Trigger AI investigation for an issue
  ${GREEN}search${NC}       Search issues by text or semantically
  ${GREEN}priority${NC}     Change issue priority
  ${GREEN}stats${NC}        Show statistics and metrics
  ${GREEN}config${NC}       Configure CLI settings
  ${GREEN}files${NC}        Direct file management commands

Options:
  -h, --help   Show this help message
  --mode       Force mode: auto|api|file (default: auto)

Examples:
  # Create a new issue
  app-issue-tracker create --title "Auth timeout bug" --type bug --priority critical

  # List open critical issues
  app-issue-tracker list --status open --priority critical

  # Show full issue details
  app-issue-tracker show 001-auth-timeout-bug.yaml

  # Move issue to investigation
  app-issue-tracker move 001-auth-timeout-bug.yaml investigating

  # Start investigation
  app-issue-tracker investigate issue-abc123 --agent deep-investigator

  # Search for issues
  app-issue-tracker search "authentication timeout"

  # Change priority
  app-issue-tracker priority 500-minor-bug.yaml critical

  # Direct file operations
  app-issue-tracker files status           # Show file counts
  app-issue-tracker files add              # Interactive add
  app-issue-tracker files archive 30       # Archive old issues

Mode Selection:
  API Mode:    Uses HTTP API (requires API server running)
  File Mode:   Direct file operations (works offline)
  Auto Mode:   Tries API first, falls back to file operations

EOF
}

# Load configuration
load_config() {
    if [ -f "$CONFIG_FILE" ]; then
        API_TOKEN=$(jq -r '.api_token // empty' "$CONFIG_FILE" 2>/dev/null || echo "")
        API_URL=$(jq -r '.api_url // empty' "$CONFIG_FILE" 2>/dev/null || echo "$API_URL")
        MODE=$(jq -r '.mode // "auto"' "$CONFIG_FILE" 2>/dev/null || echo "auto")
    fi
}

# Save configuration  
save_config() {
    local token=$1
    local url=$2
    local mode=$3
    
    mkdir -p "${CONFIG_FILE%/*}"
    cat > "$CONFIG_FILE" << EOF
{
  "api_token": "$token",
  "api_url": "$url",
  "mode": "$mode"
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

# File operations for direct mode

file_create_issue() {
    local title="" description="" type="bug" priority="medium"
    local error_message="" stack_trace="" tags=""
    local reporter_name="" reporter_email=""
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --title) title="$2"; shift 2 ;;
            --description) description="$2"; shift 2 ;;
            --type) type="$2"; shift 2 ;;
            --priority) priority="$2"; shift 2 ;;
            --error) error_message="$2"; shift 2 ;;
            --stack) stack_trace="$2"; shift 2 ;;
            --tags) tags="$2"; shift 2 ;;
            --reporter-name) reporter_name="$2"; shift 2 ;;
            --reporter-email) reporter_email="$2"; shift 2 ;;
            *) shift ;;
        esac
    done
    
    if [[ -z "$title" ]]; then
        log_error "Title is required"
        exit 1
    fi
    
    # Generate priority number and filename
    local priority_num
    case "$priority" in
        critical) priority_num=001 ;;
        high) priority_num=100 ;;
        medium) priority_num=200 ;;
        low) priority_num=500 ;;
        *) priority_num=200 ;;
    esac
    
    # Create safe filename
    local safe_title=$(echo "$title" | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9-]/-/g' | sed 's/--*/-/g')
    local filename="${priority_num}-${safe_title}.yaml"
    local filepath="$ISSUES_DIR/open/$filename"
    
    # Check if file exists, find next available number
    local counter=$priority_num
    while [[ -f "$filepath" ]]; do
        ((counter++))
        filename=$(printf "%03d-%s.yaml" "$counter" "$safe_title")
        filepath="$ISSUES_DIR/open/$filename"
    done
    
    # Generate issue ID
    local issue_id="issue-$(date +%Y%m%d-%H%M%S)-$(echo "$safe_title" | head -c 10)"
    
    # Select template
    local template_file="$ISSUES_DIR/templates/bug-template.yaml"
    case "$type" in
        feature) template_file="$ISSUES_DIR/templates/feature-template.yaml" ;;
        performance) template_file="$ISSUES_DIR/templates/performance-template.yaml" ;;
    esac
    
    if [[ ! -f "$template_file" ]]; then
        log_error "Template not found: $template_file"
        exit 1
    fi
    
    # Create issue from template
    cp "$template_file" "$filepath"
    
    # Update with actual values
    local timestamp=$(date -Iseconds)
    sed -i "s/unique-.*-id/$issue_id/g" "$filepath"
    sed -i "s/title: \".*\"/title: \"$title\"/g" "$filepath" 
    sed -i "s/description: \".*\"/description: \"${description:-$title}\"/g" "$filepath"
    sed -i "s/priority: .*/priority: $priority/g" "$filepath"
    sed -i "s/type: .*/type: $type/g" "$filepath"
    sed -i "s/YYYY-MM-DDTHH:MM:SSZ/$timestamp/g" "$filepath"
    
    # Update reporter info
    [[ -n "$reporter_name" ]] && sed -i "s/Reporter Name/$reporter_name/g" "$filepath"
    [[ -n "$reporter_email" ]] && sed -i "s/reporter@domain.com/$reporter_email/g" "$filepath"
    
    # Add error context
    if [[ -n "$error_message" ]]; then
        sed -i "s/error_message: \".*\"/error_message: \"$error_message\"/g" "$filepath"
    fi
    
    # Add tags
    if [[ -n "$tags" ]]; then
        local tag_array=$(echo "$tags" | tr ',' '\n' | sed 's/^/    - "/' | sed 's/$/"/')
        sed -i "/tags:/,/labels:/{/tags:/!{/labels:/!d;};/tags:/a\\$tag_array" "$filepath"
    fi
    
    log_success "Issue created: $filename"
    echo "Location: $filepath"
    echo "ID: $issue_id"
}

file_list_issues() {
    local status="${1:-open}"
    
    if [[ ! -d "$ISSUES_DIR/$status" ]]; then
        log_error "Invalid status folder: $status"
        exit 1
    fi
    
    echo -e "${CYAN}📋 Issues in $status:${NC}"
    echo
    
    local count=0
    for file in "$ISSUES_DIR/$status"/*.yaml; do
        if [[ -f "$file" ]]; then
            local filename=$(basename "$file")
            local title=$(grep "^title:" "$file" | sed 's/title: *"\?\(.*\)"\?/\1/' | sed 's/"$//')
            local priority=$(grep "^priority:" "$file" | sed 's/priority: *//')
            local type=$(grep "^type:" "$file" | sed 's/type: *//')
            local app_id=$(grep "^app_id:" "$file" | sed 's/app_id: *"\?\(.*\)"\?/\1/' | sed 's/"$//')
            local created=$(grep "created_at:" "$file" | sed 's/.*created_at: *"\?\(.*\)"\?/\1/' | sed 's/"$//' | head -1)
            
            # Color by priority
            case "$priority" in
                critical) priority_color="${RED}" ;;
                high) priority_color="${YELLOW}" ;;
                medium) priority_color="${BLUE}" ;;
                *) priority_color="${GREEN}" ;;
            esac
            
            # Type icon
            case "$type" in
                bug) type_icon="🐛" ;;
                feature) type_icon="✨" ;;
                performance) type_icon="⚡" ;;
                security) type_icon="🔒" ;;
                *) type_icon="📝" ;;
            esac
            
            echo -e "${priority_color}●${NC} $filename"
            echo -e "  $type_icon ${title}"
            echo -e "  App: ${app_id} | Priority: ${priority_color}${priority}${NC} | Created: ${created}"
            echo
            ((count++))
        fi
    done
    
    if [[ $count -eq 0 ]]; then
        log_info "No issues found in $status/"
    else
        log_info "Found $count issues in $status"
    fi
}

# Unified create command (tries API first, falls back to file)
create_issue() {
    local mode=$(detect_mode)
    
    if [[ "$mode" == "api" ]]; then
        create_issue_api "$@"
    else
        log_warning "API not available, using file mode"
        file_create_issue "$@"
    fi
}

create_issue_api() {
    local title="" description="" type="bug" priority="medium"
    local error_message="" stack_trace="" tags=""
    
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
    
    [[ -z "$description" ]] && description="$title"
    
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
    
    log_info "Creating issue via API..."
    local response=$(api_request "POST" "issues" "$json_data")
    
    if [ "$(echo "$response" | jq -r '.success')" = "true" ]; then
        local issue_id=$(echo "$response" | jq -r '.data.issue_id')
        local filename=$(echo "$response" | jq -r '.data.filename')
        log_success "Issue created successfully"
        echo "Issue ID: $issue_id"
        echo "File: $filename"
    else
        log_error "Failed to create issue via API"
        echo "$response" | jq -r '.message // "Unknown error"'
        log_warning "Falling back to file mode..."
        file_create_issue "$@"
    fi
}

# Unified list command
list_issues() {
    local mode=$(detect_mode)
    
    if [[ "$mode" == "api" ]]; then
        list_issues_api "$@"
    else
        file_list_issues "$@"
    fi
}

list_issues_api() {
    local status="" priority="" type="" limit="20"
    
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
    query_params="${query_params#&}"
    
    log_info "Fetching issues via API..."
    local response=$(api_request "GET" "issues?$query_params")
    
    if [ "$(echo "$response" | jq -r '.success')" = "true" ]; then
        local count=$(echo "$response" | jq -r '.data.count')
        log_success "Found $count issues"
        echo ""
        
        echo "$response" | jq -r '.data.issues[] | 
            "[\(.priority | ascii_upcase)] \(.title)\n  Status: \(.status) | Type: \(.type) | App: \(.app_id)\n  ID: \(.id)\n"'
    else
        log_error "API request failed, trying file mode..."
        file_list_issues "$status"
    fi
}

# Show issue details (file mode only for now)
show_issue() {
    local filename="$1"
    if [[ -z "$filename" ]]; then
        log_error "Filename required"
        echo "Usage: app-issue-tracker show <filename>"
        exit 1
    fi
    
    # Find file in any folder
    local found="" file_path=""
    for folder in open investigating in-progress fixed closed failed; do
        local test_path="$ISSUES_DIR/$folder/$filename"
        if [[ -f "$test_path" ]]; then
            found="$folder"
            file_path="$test_path"
            break
        fi
    done
    
    if [[ -z "$found" ]]; then
        log_error "Issue file not found: $filename"
        exit 1
    fi
    
    echo -e "${CYAN}📄 Issue Details: $filename (Status: $found)${NC}"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    # Pretty print YAML
    if command -v bat &> /dev/null; then
        bat "$file_path" --style=header,grid --language=yaml
    elif command -v pygmentize &> /dev/null; then
        pygmentize -l yaml "$file_path"
    else
        cat "$file_path"
    fi
}

# Move issue between folders
move_issue() {
    local filename="$1"
    local to_folder="$2"
    
    if [[ -z "$filename" || -z "$to_folder" ]]; then
        log_error "Filename and destination folder required"
        echo "Usage: app-issue-tracker move <filename> <to-folder>"
        exit 1
    fi
    
    # Use direct file management
    "$ISSUES_DIR/manage.sh" move "$filename" "$to_folder"
}

# Change issue priority  
change_priority() {
    local filename="$1"
    local new_priority="$2"
    
    if [[ -z "$filename" || -z "$new_priority" ]]; then
        log_error "Filename and new priority required"
        echo "Usage: app-issue-tracker priority <filename> <critical|high|medium|low>"
        exit 1
    fi
    
    # Use direct file management
    "$ISSUES_DIR/manage.sh" priority "$filename" "$new_priority"
}

# Search issues
search_issues() {
    local query="$*"
    local mode=$(detect_mode)
    
    if [[ -z "$query" ]]; then
        log_error "Search query required"
        exit 1
    fi
    
    if [[ "$mode" == "api" ]]; then
        log_info "Searching via API..."
        local encoded_query=$(echo "$query" | jq -sRr @uri)
        local response=$(api_request "GET" "issues/search?q=$encoded_query")
        
        if [ "$(echo "$response" | jq -r '.success')" = "true" ]; then
            local count=$(echo "$response" | jq -r '.data.count')
            log_success "Found $count matching issues"
            echo ""
            echo "$response" | jq -r '.data.results[] | 
                "[\(.priority | ascii_upcase)] \(.title)\n  Status: \(.status) | Type: \(.type) | App: \(.app_id)\n  ID: \(.id)\n"'
        else
            log_warning "API search failed, trying file search..."
            "$ISSUES_DIR/manage.sh" search "$query"
        fi
    else
        log_info "Searching files..."
        "$ISSUES_DIR/manage.sh" search "$query"
    fi
}

# Trigger investigation
investigate_issue() {
    local issue_id="$1"
    shift
    local mode=$(detect_mode)
    
    if [[ -z "$issue_id" ]]; then
        log_error "Issue ID or filename required"
        echo "Usage: app-issue-tracker investigate <issue-id> [--agent AGENT-ID]"
        exit 1
    fi
    
    # If it looks like a filename, try to extract issue ID
    if [[ "$issue_id" =~ \.yaml$ ]]; then
        # Find file and extract ID
        local found_file=""
        for folder in open investigating in-progress; do
            local test_path="$ISSUES_DIR/$folder/$issue_id"
            if [[ -f "$test_path" ]]; then
                found_file="$test_path"
                break
            fi
        done
        
        if [[ -n "$found_file" ]]; then
            local extracted_id=$(grep "^id:" "$found_file" | sed 's/id: *//')
            log_info "Extracted issue ID: $extracted_id"
            issue_id="$extracted_id"
            
            # Move file to investigating folder if not already there
            if [[ "$folder" == "open" ]]; then
                log_info "Moving issue to investigating folder..."
                move_issue "$(basename "$found_file")" investigating
            fi
        else
            log_error "Issue file not found: $issue_id"
            exit 1
        fi
    fi
    
    if [[ "$mode" == "api" ]]; then
        local agent_id="" priority="normal"
        
        while [[ $# -gt 0 ]]; do
            case $1 in
                --agent) agent_id="$2"; shift 2 ;;
                --priority) priority="$2"; shift 2 ;;
                *) shift ;;
            esac
        done
        
        local json_data=$(jq -n \
            --arg id "$issue_id" \
            --arg agent "$agent_id" \
            --arg priority "$priority" \
            '{
                issue_id: $id,
                agent_id: (if $agent == "" then "deep-investigator" else $agent end),
                priority: $priority
            }')
        
        log_info "Triggering investigation via API..."
        local response=$(api_request "POST" "investigate" "$json_data")
        
        if [ "$(echo "$response" | jq -r '.success')" = "true" ]; then
            log_success "Investigation started"
            echo "$response" | jq -r '.data | "Run ID: \(.run_id)\nInvestigation ID: \(.investigation_id)"'
        else
            log_error "Investigation failed"
            echo "$response" | jq -r '.message // "Unknown error"'
        fi
    else
        log_info "Using Claude Code investigation script directly..."
        local script_path="${APP_ROOT}/scenarios/app-issue-tracker/scripts/claude-investigator.sh"
        if [[ -f "$script_path" ]]; then
            "$script_path" investigate "$issue_id" "deep-investigator" "$(pwd)" "Investigate this issue thoroughly"
        else
            log_error "Claude investigator script not found at: $script_path"
        fi
    fi
}

# File management commands
file_commands() {
    local subcommand="${1:-status}"
    shift || true
    
    case "$subcommand" in
        status)
            "$ISSUES_DIR/manage.sh" status
            ;;
        add)
            "$ISSUES_DIR/manage.sh" add
            ;;
        archive)
            "$ISSUES_DIR/manage.sh" archive "$@"
            ;;
        *)
            echo "File commands: status, add, archive"
            echo "For full file management, use: $ISSUES_DIR/manage.sh"
            ;;
    esac
}

# Show statistics
show_stats() {
    local mode=$(detect_mode)
    
    if [[ "$mode" == "api" ]]; then
        log_info "Fetching statistics via API..."
        local response=$(api_request "GET" "stats")
        
        if [ "$(echo "$response" | jq -r '.success')" = "true" ]; then
            echo -e "${CYAN}📊 Issue Tracker Statistics${NC}"
            echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
            echo "$response" | jq -r '.data.stats | 
                "Total Issues: \(.total_issues)\n" +
                "Open Issues: \(.open_issues)\n" +
                "In Progress: \(.in_progress)\n" +
                "Fixed Today: \(.fixed_today)\n"'
        else
            log_warning "API stats failed, using file counts..."
            "$ISSUES_DIR/manage.sh" status
        fi
    else
        "$ISSUES_DIR/manage.sh" status
    fi
}

# Configure CLI
configure() {
    local token="" url="" mode="auto"
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --token) token="$2"; shift 2 ;;
            --url) url="$2"; shift 2 ;;
            --mode) mode="$2"; shift 2 ;;
            *) shift ;;
        esac
    done
    
    [[ -z "$url" ]] && url="$API_URL"
    
    save_config "$token" "$url" "$mode"
    echo "Mode: $mode"
    echo "API URL: $url"
}

# Main command router
main() {
    load_config
    
    # Check for mode override
    if [[ "${1:-}" == "--mode" ]]; then
        MODE="$2"
        shift 2
    fi
    
    case "${1:-}" in
        create)
            shift
            create_issue "$@"
            ;;
        list)
            shift
            list_issues "$@"
            ;;
        show)
            shift
            show_issue "$@"
            ;;
        move)
            shift
            move_issue "$@"
            ;;
        priority)
            shift
            change_priority "$@"
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
        files)
            shift
            file_commands "$@"
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

# Show current mode
current_mode=$(detect_mode)
if [[ "$current_mode" == "file" ]]; then
    echo -e "${YELLOW}📁 Running in file mode (API not available)${NC}" >&2
else
    echo -e "${GREEN}🌐 Running in API mode${NC}" >&2
fi

# Run main function
main "$@"