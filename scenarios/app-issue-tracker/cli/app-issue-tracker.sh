#!/bin/bash
################################################################################
# App Issue Tracker CLI - Thin Wrapper Version
# 
# A lightweight CLI that delegates all logic to the scenario's API.
# Port discovery uses ultra-fast file-based lookup.
################################################################################

set -e

# Configuration
SCENARIO_NAME="app-issue-tracker"
API_BASE_PATH="/api/v1"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

################################################################################
# Port Discovery - Ultra-fast file-based lookup
################################################################################
get_api_url() {
    # Use the ultra-fast port command
    local api_port
    api_port=$(vrooli scenario port "${SCENARIO_NAME}" API_PORT 2>/dev/null)
    
    if [[ -z "$api_port" ]]; then
        echo -e "${RED}❌ Error: ${SCENARIO_NAME} is not running${NC}" >&2
        echo "   Start it with: vrooli scenario run ${SCENARIO_NAME}" >&2
        exit 1
    fi
    
    echo "http://localhost:${api_port}"
}

################################################################################
# Helper Functions
################################################################################
usage() {
    echo -e "${CYAN}🐛 App Issue Tracker CLI${NC}"
    echo "File-based issue management with AI investigation and automated fixes"
    echo ""
    echo "Usage: app-issue-tracker [COMMAND] [OPTIONS]"
    echo ""
    echo "Commands:"
    echo -e "  ${GREEN}create${NC}       Create a new issue with detailed context"
    echo -e "  ${GREEN}list${NC}         List issues with powerful filtering options"
    echo -e "  ${GREEN}show${NC}         Show comprehensive issue details"
    echo -e "  ${GREEN}investigate${NC}  Trigger AI-powered issue investigation"
    echo -e "  ${GREEN}search${NC}       Search issues by text or semantic content"
    echo -e "  ${GREEN}fix${NC}          Generate and apply automated fixes"
    echo -e "  ${GREEN}agents${NC}       Manage AI investigation agents"
    echo -e "  ${GREEN}apps${NC}         View app-specific issue statistics"
    echo -e "  ${GREEN}stats${NC}        View comprehensive issue analytics"
    echo -e "  ${GREEN}health${NC}       Check service health and storage status"
    echo ""
    echo "Examples:"
    echo "  app-issue-tracker create --title \"Auth timeout bug\" --type bug --priority critical"
    echo "  app-issue-tracker list --status open --priority high"
    echo "  app-issue-tracker investigate issue-abc123 --agent deep-investigator"
    echo "  app-issue-tracker search \"authentication timeout\""
    echo "  app-issue-tracker fix issue-abc123 --auto-apply"
    echo ""
    echo "Options:"
    echo "  --help, -h       Show help for any command"
    echo "  --json           Output in JSON format (most commands)"
    echo "  --mode           Force operation mode (api|file|auto)"
    echo ""
    echo "For more information: app-issue-tracker <command> --help"
}

# Format JSON output if jq is available
format_json() {
    if command -v jq >/dev/null 2>&1; then
        jq .
    else
        cat
    fi
}

# Make API request
api_request() {
    local method="$1"
    local endpoint="$2"
    local data="${3:-}"
    local api_url
    
    # Get the current API URL
    api_url=$(get_api_url)
    
    local curl_args=(-s)
    
    if [[ "$method" != "GET" ]]; then
        curl_args+=(-X "$method")
    fi
    
    if [[ -n "$data" ]]; then
        curl_args+=(-H "Content-Type: application/json" -d "$data")
    fi
    
    curl "${curl_args[@]}" "${api_url}${endpoint}"
}

# Detect mode (always use API for thin wrapper)
detect_mode() {
    echo "api"
}

################################################################################
# Command Implementations - All delegated to API
################################################################################

# Create issue command
cmd_create() {
    local title=""
    local description=""
    local type="bug"
    local priority="medium"
    local app_id=""
    local error_message=""
    local stack_trace=""
    local tags=""
    local reporter_name=""
    local reporter_email=""
    local environment=""
    local json_output=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --title)
                title="$2"
                shift 2
                ;;
            --description)
                description="$2"
                shift 2
                ;;
            --type)
                type="$2"
                shift 2
                ;;
            --priority)
                priority="$2"
                shift 2
                ;;
            --app-id)
                app_id="$2"
                shift 2
                ;;
            --error)
                error_message="$2"
                shift 2
                ;;
            --stack)
                stack_trace="$2"
                shift 2
                ;;
            --tags)
                tags="$2"
                shift 2
                ;;
            --reporter-name)
                reporter_name="$2"
                shift 2
                ;;
            --reporter-email)
                reporter_email="$2"
                shift 2
                ;;
            --environment)
                environment="$2"
                shift 2
                ;;
            --json)
                json_output=true
                shift
                ;;
            --help|-h)
                echo "Usage: app-issue-tracker create [OPTIONS]"
                echo ""
                echo "Create a new issue with comprehensive details"
                echo ""
                echo "Options:"
                echo "  --title <title>             Issue title (required)"
                echo "  --description <desc>        Detailed description"
                echo "  --type <type>              Issue type (bug/feature/performance/security)"
                echo "  --priority <priority>       Priority (critical/high/medium/low)"
                echo "  --app-id <id>              Application identifier"
                echo "  --error <message>          Error message that occurred"
                echo "  --stack <trace>            Stack trace information"
                echo "  --tags <list>              Comma-separated tags"
                echo "  --reporter-name <name>     Reporter's name"
                echo "  --reporter-email <email>   Reporter's email"
                echo "  --environment <env>        Environment details (JSON)"
                echo "  --json                     Output in JSON format"
                echo ""
                echo "Examples:"
                echo "  app-issue-tracker create --title \"Database timeout\" --type bug --priority high"
                echo "  app-issue-tracker create --title \"Add SSO\" --type feature --app-id auth-service"
                return 0
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Interactive mode if title not provided
    if [[ -z "$title" ]]; then
        read -p "Issue title (required): " title
        [[ -z "$title" ]] && { echo -e "${RED}❌ Title is required${NC}" >&2; return 1; }
        
        read -p "Description: " description
        read -p "Type [bug/feature/performance/security]: " type_input
        [[ -n "$type_input" ]] && type="$type_input"
        
        read -p "Priority [critical/high/medium/low]: " priority_input
        [[ -n "$priority_input" ]] && priority="$priority_input"
        
        read -p "App ID (optional): " app_id
        read -p "Error message (optional): " error_message
        read -p "Tags (comma-separated, optional): " tags
        read -p "Reporter name (optional): " reporter_name
        read -p "Reporter email (optional): " reporter_email
    fi
    
    # Default description to title if empty
    [[ -z "$description" ]] && description="$title"
    
    # Parse environment JSON if provided
    local env_obj="{}"
    if [[ -n "$environment" ]]; then
        # Try to parse as JSON, otherwise create simple object
        if echo "$environment" | jq empty 2>/dev/null; then
            env_obj="$environment"
        else
            env_obj="{\"description\": \"$environment\"}"
        fi
    fi
    
    # Parse tags into array
    local tags_array="[]"
    if [[ -n "$tags" ]]; then
        tags_array=$(echo "$tags" | jq -R 'split(",") | map(gsub("^\\s+|\\s+$";"")) | map(select(. != ""))')
    fi
    
    # Build request JSON
    local request_body=$(jq -n \
        --arg title "$title" \
        --arg description "$description" \
        --arg type "$type" \
        --arg priority "$priority" \
        --arg app_id "$app_id" \
        --arg error_message "$error_message" \
        --arg stack_trace "$stack_trace" \
        --arg reporter_name "$reporter_name" \
        --arg reporter_email "$reporter_email" \
        --argjson tags "$tags_array" \
        --argjson environment "$env_obj" \
        '{
            title: $title,
            description: $description,
            type: $type,
            priority: $priority,
            app_id: $app_id,
            error_message: $error_message,
            stack_trace: $stack_trace,
            tags: $tags,
            reporter_name: $reporter_name,
            reporter_email: $reporter_email,
            environment: $environment
        }')
    
    echo -e "${BLUE}📝 Creating issue...${NC}"
    
    local response
    response=$(api_request "POST" "${API_BASE_PATH}/issues" "$request_body")
    
    if [[ $json_output == true ]]; then
        echo "$response" | format_json
    else
        if echo "$response" | grep -q "success.*true"; then
            echo -e "${GREEN}✅ Issue created successfully${NC}"
            local issue_id
            local filename
            issue_id=$(echo "$response" | jq -r '.data.issue_id // "unknown"' 2>/dev/null)
            filename=$(echo "$response" | jq -r '.data.filename // "unknown"' 2>/dev/null)
            echo "   Issue ID: $issue_id"
            echo "   Filename: $filename"
            echo "   Title: $title"
            echo "   Priority: $priority"
        else
            echo -e "${RED}❌ Failed to create issue${NC}"
            echo "$response" | jq -r '.message // "Unknown error"' 2>/dev/null
        fi
    fi
}

# List issues command
cmd_list() {
    local status=""
    local priority=""
    local type=""
    local limit="20"
    local json_output=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --status)
                status="$2"
                shift 2
                ;;
            --priority)
                priority="$2"
                shift 2
                ;;
            --type)
                type="$2"
                shift 2
                ;;
            --limit)
                limit="$2"
                shift 2
                ;;
            --json)
                json_output=true
                shift
                ;;
            --help|-h)
                echo "Usage: app-issue-tracker list [OPTIONS]"
                echo ""
                echo "List issues with powerful filtering capabilities"
                echo ""
                echo "Options:"
                echo "  --status <status>     Filter by status (open/investigating/in-progress/fixed/closed/failed)"
                echo "  --priority <priority> Filter by priority (critical/high/medium/low)"
                echo "  --type <type>         Filter by type (bug/feature/performance/security)"
                echo "  --limit <num>         Maximum number of results (default: 20)"
                echo "  --json                Output in JSON format"
                echo ""
                echo "Examples:"
                echo "  app-issue-tracker list --status open --priority critical"
                echo "  app-issue-tracker list --type bug --limit 10"
                echo "  app-issue-tracker list --json | jq '.data.issues[] | .title'"
                return 0
                ;;
            *)
                shift
                ;;
        esac
    done
    
    local endpoint="${API_BASE_PATH}/issues"
    local query_params=""
    
    [[ -n "$status" ]] && query_params="${query_params:+$query_params&}status=$status"
    [[ -n "$priority" ]] && query_params="${query_params:+$query_params&}priority=$priority"
    [[ -n "$type" ]] && query_params="${query_params:+$query_params&}type=$type"
    query_params="${query_params:+$query_params&}limit=$limit"
    [[ -n "$query_params" ]] && endpoint="${endpoint}?${query_params}"
    
    echo -e "${BLUE}📋 Fetching issues...${NC}"
    
    local response
    response=$(api_request "GET" "$endpoint")
    
    if [[ $json_output == true ]]; then
        echo "$response" | format_json
    else
        if echo "$response" | grep -q "success.*true"; then
            local count=$(echo "$response" | jq -r '.data.count // 0' 2>/dev/null)
            echo -e "${GREEN}✅ Found $count issues:${NC}"
            echo ""
            
            if [[ $count -gt 0 ]]; then
                echo "$response" | jq -r '.data.issues[]? | 
                    "[\(if .priority == "critical" then "🔥" elif .priority == "high" then "⚠️" elif .priority == "medium" then "📌" else "📝" end)] \(.title)",
                    "  Status: \(.status) | Type: \(if .type == "bug" then "🐛" elif .type == "feature" then "✨" elif .type == "performance" then "⚡" else .type end) | App: \(.app_id)",
                    "  ID: \(.id)",
                    ""' 2>/dev/null
            else
                echo -e "${YELLOW}No issues found matching your criteria${NC}"
                [[ -n "$status" || -n "$priority" || -n "$type" ]] && echo "   Try removing some filters"
            fi
        else
            echo -e "${RED}❌ Failed to fetch issues${NC}"
            echo "$response" | jq -r '.message // "Unknown error"' 2>/dev/null
        fi
    fi
}

# Show issue details
cmd_show() {
    local issue_id="$1"
    local json_output=false
    
    if [[ -z "$issue_id" || "$issue_id" == "--help" || "$issue_id" == "-h" ]]; then
        echo "Usage: app-issue-tracker show <issue-id> [OPTIONS]"
        echo ""
        echo "Show comprehensive details for a specific issue"
        echo ""
        echo "Arguments:"
        echo "  issue-id    Issue ID or filename to display"
        echo ""
        echo "Options:"
        echo "  --json      Output in JSON format"
        echo ""
        echo "Examples:"
        echo "  app-issue-tracker show issue-abc123"
        echo "  app-issue-tracker show 001-auth-timeout-bug.yaml"
        return 0
    fi
    
    shift
    while [[ $# -gt 0 ]]; do
        case $1 in
            --json)
                json_output=true
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    echo -e "${BLUE}📄 Fetching issue details...${NC}"
    echo -e "${YELLOW}⚠️  Note: Direct issue viewing not yet implemented in API mode${NC}"
    echo "   This feature requires either:"
    echo "   1. An API endpoint to get issue by ID"
    echo "   2. File-based access to YAML files"
    echo ""
    echo "   For now, use: app-issue-tracker list --json | jq '.data.issues[] | select(.id == \"$issue_id\")'"
}

# Search issues command
cmd_search() {
    local query=""
    local limit="10"
    local json_output=false
    
    # Parse positional argument for query
    if [[ $# -gt 0 && ! "$1" =~ ^-- ]]; then
        query="$1"
        shift
    fi
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --limit)
                limit="$2"
                shift 2
                ;;
            --json)
                json_output=true
                shift
                ;;
            --help|-h)
                echo "Usage: app-issue-tracker search <query> [OPTIONS]"
                echo ""
                echo "Search through issues using text matching"
                echo ""
                echo "Arguments:"
                echo "  query       Search terms to find in issues"
                echo ""
                echo "Options:"
                echo "  --limit <num>    Maximum number of results (default: 10)"
                echo "  --json           Output in JSON format"
                echo ""
                echo "Examples:"
                echo "  app-issue-tracker search \"authentication timeout\""
                echo "  app-issue-tracker search \"database error\" --limit 5"
                echo "  app-issue-tracker search \"critical bug\" --json"
                return 0
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Interactive mode if query not provided
    if [[ -z "$query" ]]; then
        read -p "Enter search query: " query
        [[ -z "$query" ]] && { echo -e "${RED}❌ Search query is required${NC}" >&2; return 1; }
    fi
    
    local encoded_query
    encoded_query=$(echo "$query" | sed 's/ /%20/g' | sed 's/&/%26/g')
    
    echo -e "${BLUE}🔍 Searching for: $query${NC}"
    
    local response
    response=$(api_request "GET" "${API_BASE_PATH}/issues/search?q=$encoded_query&limit=$limit")
    
    if [[ $json_output == true ]]; then
        echo "$response" | format_json
    else
        if echo "$response" | grep -q "success.*true"; then
            local count=$(echo "$response" | jq -r '.data.count // 0' 2>/dev/null)
            echo -e "${GREEN}✅ Found $count matching issues:${NC}"
            echo ""
            
            if [[ $count -gt 0 ]]; then
                echo "$response" | jq -r '.data.results[]? | 
                    "[\(if .priority == "critical" then "🔥" elif .priority == "high" then "⚠️" elif .priority == "medium" then "📌" else "📝" end)] \(.title)",
                    "  Status: \(.status) | Type: \(.type) | App: \(.app_id)",
                    "  ID: \(.id)",
                    ""' 2>/dev/null
            else
                echo -e "${YELLOW}No issues found matching \"$query\"${NC}"
                echo "   Try using different search terms"
            fi
        else
            echo -e "${RED}❌ Search failed${NC}"
            echo "$response" | jq -r '.message // "Unknown error"' 2>/dev/null
        fi
    fi
}

# Investigate issue command
cmd_investigate() {
    local issue_id="$1"
    local agent_id=""
    local priority="normal"
    local json_output=false
    
    if [[ -z "$issue_id" || "$issue_id" == "--help" || "$issue_id" == "-h" ]]; then
        echo "Usage: app-issue-tracker investigate <issue-id> [OPTIONS]"
        echo ""
        echo "Trigger AI-powered investigation of an issue"
        echo ""
        echo "Arguments:"
        echo "  issue-id    Issue ID to investigate"
        echo ""
        echo "Options:"
        echo "  --agent <id>        Agent to use (default: deep-investigator)"
        echo "  --priority <level>  Investigation priority (normal/high/urgent)"
        echo "  --json              Output in JSON format"
        echo ""
        echo "Examples:"
        echo "  app-issue-tracker investigate issue-abc123"
        echo "  app-issue-tracker investigate issue-def456 --agent auto-fixer --priority high"
        return 0
    fi
    
    shift
    while [[ $# -gt 0 ]]; do
        case $1 in
            --agent)
                agent_id="$2"
                shift 2
                ;;
            --priority)
                priority="$2"
                shift 2
                ;;
            --json)
                json_output=true
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Default agent if not specified
    [[ -z "$agent_id" ]] && agent_id="deep-investigator"
    
    # Build request JSON
    local request_body=$(jq -n \
        --arg issue_id "$issue_id" \
        --arg agent_id "$agent_id" \
        --arg priority "$priority" \
        '{
            issue_id: $issue_id,
            agent_id: $agent_id,
            priority: $priority
        }')
    
    echo -e "${BLUE}🔍 Starting AI investigation...${NC}"
    echo "   Issue ID: $issue_id"
    echo "   Agent: $agent_id"
    echo "   Priority: $priority"
    echo ""
    
    local response
    response=$(api_request "POST" "${API_BASE_PATH}/investigate" "$request_body")
    
    if [[ $json_output == true ]]; then
        echo "$response" | format_json
    else
        if echo "$response" | grep -q "success.*true"; then
            echo -e "${GREEN}✅ Investigation started successfully${NC}"
            local run_id
            local investigation_id
            run_id=$(echo "$response" | jq -r '.data.run_id // "unknown"' 2>/dev/null)
            investigation_id=$(echo "$response" | jq -r '.data.investigation_id // "unknown"' 2>/dev/null)
            echo "   Run ID: $run_id"
            echo "   Investigation ID: $investigation_id"
            echo "   Status: $(echo "$response" | jq -r '.data.status // "unknown"' 2>/dev/null)"
            echo ""
            echo "   The investigation is running in the background."
            echo "   Check the issue status with: app-issue-tracker list --status investigating"
        else
            echo -e "${RED}❌ Failed to start investigation${NC}"
            echo "$response" | jq -r '.message // "Unknown error"' 2>/dev/null
        fi
    fi
}

# Generate fix command
cmd_fix() {
    local issue_id="$1"
    local auto_apply=false
    local backup_enabled=true
    local json_output=false
    
    if [[ -z "$issue_id" || "$issue_id" == "--help" || "$issue_id" == "-h" ]]; then
        echo "Usage: app-issue-tracker fix <issue-id> [OPTIONS]"
        echo ""
        echo "Generate and optionally apply automated fixes"
        echo ""
        echo "Arguments:"
        echo "  issue-id    Issue ID to generate fix for"
        echo ""
        echo "Options:"
        echo "  --auto-apply        Automatically apply the generated fix"
        echo "  --no-backup         Skip backup creation before applying fix"
        echo "  --json              Output in JSON format"
        echo ""
        echo "Examples:"
        echo "  app-issue-tracker fix issue-abc123"
        echo "  app-issue-tracker fix issue-def456 --auto-apply"
        echo "  app-issue-tracker fix issue-ghi789 --auto-apply --no-backup"
        echo ""
        echo "Note: Issue must be investigated before fixes can be generated"
        return 0
    fi
    
    shift
    while [[ $# -gt 0 ]]; do
        case $1 in
            --auto-apply)
                auto_apply=true
                shift
                ;;
            --no-backup)
                backup_enabled=false
                shift
                ;;
            --json)
                json_output=true
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Build request JSON
    local request_body=$(jq -n \
        --arg issue_id "$issue_id" \
        --argjson auto_apply "$auto_apply" \
        --argjson backup_enabled "$backup_enabled" \
        '{
            issue_id: $issue_id,
            auto_apply: $auto_apply,
            backup_enabled: $backup_enabled
        }')
    
    echo -e "${BLUE}🔧 Generating fix...${NC}"
    echo "   Issue ID: $issue_id"
    echo "   Auto-apply: $([ "$auto_apply" == "true" ] && echo "Yes" || echo "No")"
    echo "   Backup enabled: $([ "$backup_enabled" == "true" ] && echo "Yes" || echo "No")"
    echo ""
    
    local response
    response=$(api_request "POST" "${API_BASE_PATH}/generate-fix" "$request_body")
    
    if [[ $json_output == true ]]; then
        echo "$response" | format_json
    else
        if echo "$response" | grep -q "success.*true"; then
            echo -e "${GREEN}✅ Fix generation started successfully${NC}"
            local run_id
            local fix_id
            run_id=$(echo "$response" | jq -r '.data.run_id // "unknown"' 2>/dev/null)
            fix_id=$(echo "$response" | jq -r '.data.fix_id // "unknown"' 2>/dev/null)
            echo "   Run ID: $run_id"
            echo "   Fix ID: $fix_id"
            echo "   Status: $(echo "$response" | jq -r '.data.status // "unknown"' 2>/dev/null)"
            echo ""
            echo "   The fix generation is running in the background."
            echo "   Check the issue status with: app-issue-tracker list --status in-progress"
        else
            echo -e "${RED}❌ Failed to generate fix${NC}"
            echo "$response" | jq -r '.message // "Unknown error"' 2>/dev/null
        fi
    fi
}

# Agents command
cmd_agents() {
    local json_output=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --json)
                json_output=true
                shift
                ;;
            --help|-h)
                echo "Usage: app-issue-tracker agents [OPTIONS]"
                echo ""
                echo "List available AI investigation agents and their capabilities"
                echo ""
                echo "Options:"
                echo "  --json    Output in JSON format"
                return 0
                ;;
            *)
                shift
                ;;
        esac
    done
    
    echo -e "${BLUE}🤖 Fetching available agents...${NC}"
    
    local response
    response=$(api_request "GET" "${API_BASE_PATH}/agents")
    
    if [[ $json_output == true ]]; then
        echo "$response" | format_json
    else
        if echo "$response" | grep -q "success.*true"; then
            local count=$(echo "$response" | jq -r '.data.count // 0' 2>/dev/null)
            echo -e "${GREEN}✅ Found $count agents:${NC}"
            echo ""
            
            if [[ $count -gt 0 ]]; then
                echo "$response" | jq -r '.data.agents[]? | 
                    "🤖 \(.display_name) (\(.id))",
                    "   \(.description)",
                    "   Capabilities: \(.capabilities | join(", "))",
                    "   Success Rate: \(.success_rate)% (\(.successful_runs)/\(.total_runs) runs)",
                    "   Status: \(if .is_active then "✅ Active" else "❌ Inactive" end)",
                    ""' 2>/dev/null
            else
                echo -e "${YELLOW}No agents found${NC}"
            fi
        else
            echo -e "${RED}❌ Failed to fetch agents${NC}"
            echo "$response" | jq -r '.message // "Unknown error"' 2>/dev/null
        fi
    fi
}

# Apps command  
cmd_apps() {
    local json_output=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --json)
                json_output=true
                shift
                ;;
            --help|-h)
                echo "Usage: app-issue-tracker apps [OPTIONS]"
                echo ""
                echo "View app-specific issue statistics and health metrics"
                echo ""
                echo "Options:"
                echo "  --json    Output in JSON format"
                return 0
                ;;
            *)
                shift
                ;;
        esac
    done
    
    echo -e "${BLUE}📱 Fetching app statistics...${NC}"
    
    local response
    response=$(api_request "GET" "${API_BASE_PATH}/apps")
    
    if [[ $json_output == true ]]; then
        echo "$response" | format_json
    else
        if echo "$response" | grep -q "success.*true"; then
            local count=$(echo "$response" | jq -r '.data.count // 0' 2>/dev/null)
            echo -e "${GREEN}✅ Found $count apps with issues:${NC}"
            echo ""
            
            if [[ $count -gt 0 ]]; then
                echo "$response" | jq -r '.data.apps[]? | 
                    "📱 \(.display_name) (\(.id))",
                    "   Type: \(.type) | Status: \(.status)",
                    "   Total Issues: \(.total_issues) | Open: \(.open_issues)",
                    ""' 2>/dev/null
            else
                echo -e "${YELLOW}No apps with issues found${NC}"
            fi
        else
            echo -e "${RED}❌ Failed to fetch app statistics${NC}"
            echo "$response" | jq -r '.message // "Unknown error"' 2>/dev/null
        fi
    fi
}

# Stats command
cmd_stats() {
    local json_output=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --json)
                json_output=true
                shift
                ;;
            --help|-h)
                echo "Usage: app-issue-tracker stats [OPTIONS]"
                echo ""
                echo "View comprehensive issue analytics and metrics"
                echo ""
                echo "Options:"
                echo "  --json    Output in JSON format"
                return 0
                ;;
            *)
                shift
                ;;
        esac
    done
    
    echo -e "${BLUE}📊 Fetching issue statistics...${NC}"
    
    local response
    response=$(api_request "GET" "${API_BASE_PATH}/stats")
    
    if [[ $json_output == true ]]; then
        echo "$response" | format_json
    else
        if echo "$response" | grep -q "success.*true"; then
            echo -e "${GREEN}✅ Issue Tracker Statistics:${NC}"
            echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
            echo ""
            
            if echo "$response" | jq -e '.data.stats' >/dev/null 2>&1; then
                echo "$response" | jq -r '.data.stats | 
                    "📚 Total Issues: \(.total_issues)",
                    "🟢 Open Issues: \(.open_issues)",
                    "🔄 In Progress: \(.in_progress)",
                    "✅ Fixed Today: \(.fixed_today)",
                    "⏱️  Avg Resolution: \(.avg_resolution_hours) hours",
                    ""' 2>/dev/null
                
                # Show top apps if available
                if echo "$response" | jq -e '.data.stats.top_apps' >/dev/null 2>&1; then
                    echo "🏆 Top Apps by Issues:"
                    echo "$response" | jq -r '.data.stats.top_apps[]? | "   \(.app_name): \(.issue_count) issues"' 2>/dev/null
                fi
            else
                echo "$response" | format_json
            fi
        else
            echo -e "${RED}❌ Failed to fetch statistics${NC}"
            echo "$response" | jq -r '.message // "Unknown error"' 2>/dev/null
        fi
    fi
}

# Health check command
cmd_health() {
    local json_output=false
    local verbose=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --json)
                json_output=true
                shift
                ;;
            --verbose)
                verbose=true
                shift
                ;;
            --help|-h)
                echo "Usage: app-issue-tracker health [OPTIONS]"
                echo ""
                echo "Check service health and storage status"
                echo ""
                echo "Options:"
                echo "  --json       Output in JSON format"
                echo "  --verbose    Show detailed health information"
                return 0
                ;;
            *)
                shift
                ;;
        esac
    done
    
    echo -e "${BLUE}🏥 Checking service health...${NC}"
    
    local api_url
    api_url=$(get_api_url)
    
    local response
    response=$(curl -s "${api_url}/health" 2>/dev/null)
    
    if [[ $json_output == true ]]; then
        echo "$response" | format_json
    else
        if echo "$response" | grep -q "success.*true"; then
            echo -e "${GREEN}✅ App Issue Tracker is healthy${NC}"
            echo "   API: ${api_url}"
            
            if [[ "$verbose" == true ]]; then
                echo "   Version: $(echo "$response" | jq -r '.data.version // "unknown"' 2>/dev/null)"
                echo "   Storage: $(echo "$response" | jq -r '.data.storage // "unknown"' 2>/dev/null)"
                echo "   Issues Directory: $(echo "$response" | jq -r '.data.issues_dir // "unknown"' 2>/dev/null)"
                echo "   Mode: API (thin wrapper)"
            fi
        else
            echo -e "${RED}❌ Health check failed${NC}"
            echo "   Response: $response"
            return 1
        fi
    fi
}

################################################################################
# Main Command Router
################################################################################
main() {
    local command="${1:-}"
    
    if [[ -z "$command" ]]; then
        usage
        exit 0
    fi
    
    shift
    
    case "$command" in
        create)
            cmd_create "$@"
            ;;
        list)
            cmd_list "$@"
            ;;
        show)
            cmd_show "$@"
            ;;
        search)
            cmd_search "$@"
            ;;
        investigate)
            cmd_investigate "$@"
            ;;
        fix)
            cmd_fix "$@"
            ;;
        agents)
            cmd_agents "$@"
            ;;
        apps)
            cmd_apps "$@"
            ;;
        stats)
            cmd_stats "$@"
            ;;
        health)
            cmd_health "$@"
            ;;
        --help|-h|help)
            usage
            ;;
        --version|-v|version)
            echo "App Issue Tracker CLI v2.0.0 (thin wrapper)"
            ;;
            
        # Legacy commands for backward compatibility
        move)
            echo -e "${YELLOW}⚠️  'move' command not implemented in thin wrapper mode${NC}"
            echo "   Issues are automatically moved by the API during investigation/fix processes"
            ;;
        priority)
            echo -e "${YELLOW}⚠️  'priority' command not implemented in thin wrapper mode${NC}"
            echo "   Priority changes will be supported in a future API update"
            ;;
        files)
            echo -e "${YELLOW}⚠️  'files' command not available in API mode${NC}"
            echo "   File operations are handled transparently by the API"
            ;;
        config)
            echo -e "${YELLOW}⚠️  'config' command not needed in thin wrapper mode${NC}"
            echo "   Configuration is managed by the scenario service"
            ;;
            
        *)
            echo -e "${RED}❌ Unknown command: $command${NC}" >&2
            usage
            exit 1
            ;;
    esac
}

# Check dependencies
if ! command -v jq &> /dev/null; then
    echo -e "${RED}❌ Error: jq is required but not installed${NC}" >&2
    echo "Install with: apt-get install jq (or your package manager)" >&2
    exit 1
fi

if ! command -v curl &> /dev/null; then
    echo -e "${RED}❌ Error: curl is required but not installed${NC}" >&2
    echo "Install with: apt-get install curl" >&2
    exit 1
fi

# Show current mode (always API for thin wrapper)
echo -e "${GREEN}🌐 Running in API mode (thin wrapper)${NC}" >&2

# Run main function
main "$@"
