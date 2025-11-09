#!/bin/bash
# {{CLI_NAME}} - Command-line interface for {{SCENARIO_DISPLAY_NAME}}
# This is a thin wrapper around the API, following the agent-metareasoning-manager pattern

set -euo pipefail

# Configuration
readonly CLI_NAME="{{CLI_NAME}}"
readonly CLI_VERSION="1.0.0"
readonly CONFIG_DIR="$HOME/.${CLI_NAME}"
readonly CONFIG_FILE="$CONFIG_DIR/config.json"
readonly DEFAULT_TOKEN="{{CLI_TOKEN}}"
readonly SCENARIO_ID="{{SCENARIO_ID}}"
CONFIG_API_BASE=""
API_BASE=""

# Colors for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m' # No Color

# Initialize configuration
init_config() {
    if [[ ! -d "$CONFIG_DIR" ]]; then
        mkdir -p "$CONFIG_DIR"
    fi
    
    if [[ ! -f "$CONFIG_FILE" ]]; then
        cat > "$CONFIG_FILE" <<EOF
{
    "api_base": "",
    "api_token": "$DEFAULT_TOKEN",
    "output_format": "table",
    "created_at": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
}
EOF
        echo -e "${GREEN}✓${NC} Configuration initialized at $CONFIG_FILE"
        echo "API base will be auto-detected from the running scenario unless you override it via '$CLI_NAME configure api_base <url>'."
    fi
}

# Load configuration
load_config() {
    if [[ -f "$CONFIG_FILE" ]]; then
        CONFIG_API_BASE=$(jq -r '.api_base // ""' "$CONFIG_FILE" 2>/dev/null || echo "")
        API_TOKEN=$(jq -r '.api_token // "'$DEFAULT_TOKEN'"' "$CONFIG_FILE" 2>/dev/null || echo "$DEFAULT_TOKEN")
        OUTPUT_FORMAT=$(jq -r '.output_format // "table"' "$CONFIG_FILE" 2>/dev/null || echo "table")
    else
        CONFIG_API_BASE=""
        API_TOKEN="$DEFAULT_TOKEN"
        OUTPUT_FORMAT="table"
    fi
}

detect_api_port() {
    if command -v vrooli >/dev/null 2>&1; then
        vrooli scenario port "$SCENARIO_ID" API_PORT 2>/dev/null || true
    fi
}

detect_api_base() {
    if [[ -n "${API_BASE_URL:-}" ]]; then
        echo "${API_BASE_URL%/}"
        return 0
    fi

    if [[ -n "${API_PORT:-}" ]]; then
        echo "http://${API_HOST:-localhost}:${API_PORT}/api/v1"
        return 0
    fi

    local detected_port
    detected_port="$(detect_api_port)"
    if [[ -n "$detected_port" ]]; then
        echo "http://localhost:${detected_port}/api/v1"
        return 0
    fi

    return 1
}

normalize_api_base() {
    local value="$1"
    if [[ -z "$value" || "$value" == "null" ]]; then
        return 1
    fi
    echo "${value%/}"
}

resolve_api_base() {
    local candidate

    if candidate=$(normalize_api_base "$1" 2>/dev/null); then
        echo "$candidate"
        return 0
    fi

    local detected
    if detected=$(detect_api_base); then
        echo "$detected"
        return 0
    fi

    return 1
}

# Make API request
api_request() {
    local method="$1"
    local endpoint="$2"
    local data="${3:-}"
    
    if [[ -z "${API_BASE:-}" ]]; then
        echo -e "${RED}✗${NC} API base URL is not configured. Run '$CLI_NAME configure api_base <url>' or start the scenario via the lifecycle so the CLI can auto-detect it." >&2
        return 1
    fi

    local url="${API_BASE}${endpoint}"
    local curl_opts=(-s -X "$method" -H "Authorization: Bearer $API_TOKEN")
    
    if [[ -n "$data" ]]; then
        curl_opts+=(-H "Content-Type: application/json" -d "$data")
    fi
    
    response=$(curl "${curl_opts[@]}" "$url" 2>/dev/null)
    exit_code=$?
    
    if [[ $exit_code -ne 0 ]]; then
        echo -e "${RED}✗${NC} Failed to connect to API at $url" >&2
        return 1
    fi
    
    echo "$response"
}

# Format output
format_output() {
    local data="$1"
    local format="${2:-$OUTPUT_FORMAT}"
    
    case "$format" in
        json)
            echo "$data" | jq '.' 2>/dev/null || echo "$data"
            ;;
        table)
            # Convert JSON to simple table format
            echo "$data" | jq -r '
                if type == "array" then
                    .[] | [.id, .name, .status // .description // ""] | @tsv
                elif type == "object" then
                    to_entries | .[] | [.key, .value] | @tsv
                else
                    .
                end
            ' 2>/dev/null || echo "$data"
            ;;
        raw)
            echo "$data"
            ;;
        *)
            echo "$data"
            ;;
    esac
}

# Command: health
cmd_health() {
    echo -e "${BLUE}Checking system health...${NC}"
    
    response=$(api_request GET "/health")
    if [[ $? -eq 0 ]]; then
        status=$(echo "$response" | jq -r '.status // "unknown"' 2>/dev/null)
        
        if [[ "$status" == "healthy" ]]; then
            echo -e "${GREEN}✓${NC} System is healthy"
        else
            echo -e "${YELLOW}⚠${NC} System status: $status"
        fi
        
        echo "$response" | jq -r '
            "  API: \(.service // "Unknown")",
            "  Version: \(.version // "Unknown")",
            "  Database: \(.database // "Unknown")"
        ' 2>/dev/null
    else
        echo -e "${RED}✗${NC} API is not responding"
        return 1
    fi
}

# Command: list resources
cmd_list() {
    local resource="${1:-resources}"
    
    echo -e "${BLUE}Listing ${resource}...${NC}"
    
    response=$(api_request GET "/api/v1/${resource}")
    if [[ $? -eq 0 ]]; then
        count=$(echo "$response" | jq '. | length' 2>/dev/null || echo "0")
        echo -e "${GREEN}✓${NC} Found $count ${resource}"
        echo
        format_output "$response"
    else
        echo -e "${RED}✗${NC} Failed to list ${resource}"
        return 1
    fi
}

# Command: get resource
cmd_get() {
    local resource="${1:-resources}"
    local id="$2"
    
    if [[ -z "$id" ]]; then
        echo -e "${RED}✗${NC} ID is required" >&2
        return 1
    fi
    
    echo -e "${BLUE}Getting ${resource} ${id}...${NC}"
    
    response=$(api_request GET "/api/v1/${resource}/${id}")
    if [[ $? -eq 0 ]]; then
        echo -e "${GREEN}✓${NC} Retrieved ${resource}"
        echo
        format_output "$response"
    else
        echo -e "${RED}✗${NC} Failed to get ${resource}"
        return 1
    fi
}

# Command: create resource
cmd_create() {
    local resource="${1:-resources}"
    shift
    
    # Build JSON from remaining arguments
    local json_data="{}"
    while [[ $# -gt 0 ]]; do
        key="$1"
        value="${2:-}"
        json_data=$(echo "$json_data" | jq --arg k "$key" --arg v "$value" '. + {($k): $v}')
        shift 2 || break
    done
    
    echo -e "${BLUE}Creating ${resource}...${NC}"
    
    response=$(api_request POST "/api/v1/${resource}" "$json_data")
    if [[ $? -eq 0 ]]; then
        id=$(echo "$response" | jq -r '.data.id // .id // ""' 2>/dev/null)
        echo -e "${GREEN}✓${NC} Created ${resource} ${id}"
        echo
        format_output "$response"
    else
        echo -e "${RED}✗${NC} Failed to create ${resource}"
        return 1
    fi
}

# Command: update resource
cmd_update() {
    local resource="${1:-resources}"
    local id="$2"
    shift 2
    
    if [[ -z "$id" ]]; then
        echo -e "${RED}✗${NC} ID is required" >&2
        return 1
    fi
    
    # Build JSON from remaining arguments
    local json_data="{}"
    while [[ $# -gt 0 ]]; do
        key="$1"
        value="${2:-}"
        json_data=$(echo "$json_data" | jq --arg k "$key" --arg v "$value" '. + {($k): $v}')
        shift 2 || break
    done
    
    echo -e "${BLUE}Updating ${resource} ${id}...${NC}"
    
    response=$(api_request PUT "/api/v1/${resource}/${id}" "$json_data")
    if [[ $? -eq 0 ]]; then
        echo -e "${GREEN}✓${NC} Updated ${resource} ${id}"
        echo
        format_output "$response"
    else
        echo -e "${RED}✗${NC} Failed to update ${resource}"
        return 1
    fi
}

# Command: delete resource
cmd_delete() {
    local resource="${1:-resources}"
    local id="$2"
    
    if [[ -z "$id" ]]; then
        echo -e "${RED}✗${NC} ID is required" >&2
        return 1
    fi
    
    echo -e "${BLUE}Deleting ${resource} ${id}...${NC}"
    
    response=$(api_request DELETE "/api/v1/${resource}/${id}")
    if [[ $? -eq 0 ]]; then
        echo -e "${GREEN}✓${NC} Deleted ${resource} ${id}"
    else
        echo -e "${RED}✗${NC} Failed to delete ${resource}"
        return 1
    fi
}

# Command: execute workflow
cmd_execute() {
    local workflow_id="${1:-}"
    shift
    
    if [[ -z "$workflow_id" ]]; then
        echo -e "${RED}✗${NC} Workflow ID is required" >&2
        return 1
    fi
    
    # Build input data from remaining arguments
    local input_data="{\"workflow_id\": \"$workflow_id\""
    if [[ $# -gt 0 ]]; then
        local user_input="$*"
        input_data="${input_data}, \"input\": \"$user_input\""
    fi
    input_data="${input_data}}"
    
    echo -e "${BLUE}Executing workflow ${workflow_id}...${NC}"
    
    response=$(api_request POST "/api/v1/execute" "$input_data")
    if [[ $? -eq 0 ]]; then
        execution_id=$(echo "$response" | jq -r '.data.execution_id // .execution_id // ""' 2>/dev/null)
        echo -e "${GREEN}✓${NC} Started execution ${execution_id}"
        echo
        format_output "$response"
    else
        echo -e "${RED}✗${NC} Failed to execute workflow"
        return 1
    fi
}

# Command: configure
cmd_configure() {
    local key="${1:-}"
    local value="${2:-}"
    
    if [[ -z "$key" ]]; then
        echo "Current configuration:"
        cat "$CONFIG_FILE" | jq '.' 2>/dev/null || cat "$CONFIG_FILE"
        return 0
    fi
    
    case "$key" in
        api|api_base)
            jq --arg v "$value" '.api_base = $v' "$CONFIG_FILE" > "${CONFIG_FILE}.tmp" && mv "${CONFIG_FILE}.tmp" "$CONFIG_FILE"
            echo -e "${GREEN}✓${NC} API base set to: $value"
            CONFIG_API_BASE="$value"
            ;;
        token|api_token)
            jq --arg v "$value" '.api_token = $v' "$CONFIG_FILE" > "${CONFIG_FILE}.tmp" && mv "${CONFIG_FILE}.tmp" "$CONFIG_FILE"
            echo -e "${GREEN}✓${NC} API token updated"
            API_TOKEN="$value"
            ;;
        format|output_format)
            jq --arg v "$value" '.output_format = $v' "$CONFIG_FILE" > "${CONFIG_FILE}.tmp" && mv "${CONFIG_FILE}.tmp" "$CONFIG_FILE"
            echo -e "${GREEN}✓${NC} Output format set to: $value"
            OUTPUT_FORMAT="$value"
            ;;
        *)
            echo -e "${RED}✗${NC} Unknown configuration key: $key"
            echo "Valid keys: api_base, api_token, output_format"
            return 1
            ;;
    esac
}

# Command: version
cmd_version() {
    echo "$CLI_NAME version $CLI_VERSION"
    if [[ -n "${API_BASE:-}" ]]; then
        echo "API endpoint: $API_BASE"
    else
        echo "API endpoint: (auto-detect when scenario is running)"
    fi
}

# Command: help
cmd_help() {
    cat <<EOF
$CLI_NAME - Command-line interface for {{SCENARIO_DISPLAY_NAME}}

Usage: $CLI_NAME <command> [options]

Commands:
    health              Check system health
    list [resource]     List resources
    get <resource> <id> Get specific resource
    create <resource> [key value...]  Create new resource
    update <resource> <id> [key value...]  Update resource
    delete <resource> <id>  Delete resource
    execute <workflow_id> [input...]  Execute workflow
    configure [key] [value]  View or update configuration
    version             Show version information
    help                Show this help message

Configuration:
    The CLI stores its configuration in: $CONFIG_FILE
    By default it auto-detects the API base URL from the running scenario via `vrooli scenario port $SCENARIO_ID API_PORT`.
    
    Configure API endpoint manually (only needed for remote deployments):
        $CLI_NAME configure api_base http://localhost:<api-port>/api/v1
    
    Configure API token:
        $CLI_NAME configure api_token your-token-here
    
    Configure output format (json|table|raw):
        $CLI_NAME configure output_format json

Examples:
    # Check health
    $CLI_NAME health
    
    # List all resources
    $CLI_NAME list resources
    
    # Get specific resource
    $CLI_NAME get resources abc123
    
    # Create new resource
    $CLI_NAME create resources name "My Resource" description "A test resource"
    
    # Execute workflow
    $CLI_NAME execute workflow-1 "Process this data"
    
    # Update configuration
    $CLI_NAME configure api_base https://api.example.com/v1

Environment Variables:
    ${CLI_NAME^^}_API_BASE    Override API base URL
    ${CLI_NAME^^}_API_TOKEN   Override API token
    ${CLI_NAME^^}_FORMAT      Override output format
    API_BASE_URL              Force API base (used before auto-detection)
    API_PORT                  Lifecycle-provided API port (used for auto-detection)

For more information, visit: https://github.com/Vrooli/Vrooli
EOF
}

# Main entry point
main() {
    # Initialize configuration on first run
    init_config
    
    # Load configuration
    load_config
    
    # Parse command early so we know whether API access is required
    local command="${1:-help}"
    shift || true

    local env_prefix="$(echo "$CLI_NAME" | tr '[:lower:]' '[:upper:]')"
    local env_api_base_var="${env_prefix}_API_BASE"
    local env_api_token_var="${env_prefix}_API_TOKEN"
    local env_format_var="${env_prefix}_FORMAT"

    local env_api_base="${!env_api_base_var:-}"
    local env_api_token="${!env_api_token_var:-}"
    local env_format="${!env_format_var:-}"

    if [[ -n "$env_api_token" ]]; then
        API_TOKEN="$env_api_token"
    fi

    if [[ -n "$env_format" ]]; then
        OUTPUT_FORMAT="$env_format"
    fi

    local requires_api=1
    case "$command" in
        help|--help|-h|version|--version|-v|configure|config)
            requires_api=0
            ;;
    esac

    if [[ -n "$env_api_base" ]]; then
        API_BASE="${env_api_base%/}"
    fi

    if (( requires_api )); then
        if [[ -z "${API_BASE:-}" ]]; then
            if ! API_BASE=$(resolve_api_base "$CONFIG_API_BASE"); then
                echo -e "${RED}✗${NC} Unable to determine API base URL. Ensure the scenario is running via 'vrooli scenario run $SCENARIO_ID' or configure it manually with '$CLI_NAME configure api_base <url>'."
                exit 1
            fi
        fi
    fi
    
    case "$command" in
        health)
            cmd_health "$@"
            ;;
        list|ls)
            cmd_list "$@"
            ;;
        get)
            cmd_get "$@"
            ;;
        create|add)
            cmd_create "$@"
            ;;
        update|edit)
            cmd_update "$@"
            ;;
        delete|rm|remove)
            cmd_delete "$@"
            ;;
        execute|exec|run)
            cmd_execute "$@"
            ;;
        configure|config)
            cmd_configure "$@"
            ;;
        version|--version|-v)
            cmd_version
            ;;
        help|--help|-h)
            cmd_help
            ;;
        *)
            echo -e "${RED}✗${NC} Unknown command: $command"
            echo "Run '$CLI_NAME help' for usage information"
            exit 1
            ;;
    esac
}

# Run main function with all arguments
main "$@"
