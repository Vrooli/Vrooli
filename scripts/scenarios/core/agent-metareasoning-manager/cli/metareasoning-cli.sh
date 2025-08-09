#!/usr/bin/env bash
# Agent Metareasoning Manager CLI
# Command-line interface for managing metareasoning tools and decision support

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Configuration
CLI_VERSION="1.0.0"
CLI_NAME="metareasoning"
DEFAULT_API_BASE="${METAREASONING_API_BASE:-http://localhost:8093/api}"
CONFIG_DIR="${HOME}/.metareasoning"
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
        DEFAULT_FORMAT=$(jq -r '.default_format // "table"' "$CONFIG_FILE" 2>/dev/null || echo "table")
        API_TOKEN=$(jq -r '.api_token // ""' "$CONFIG_FILE" 2>/dev/null || echo "")
    else
        API_BASE="$DEFAULT_API_BASE"
        DEFAULT_FORMAT="table"
        API_TOKEN=""
        create_default_config
    fi
}

# Create default configuration
create_default_config() {
    cat > "$CONFIG_FILE" << EOF
{
  "api_base": "$DEFAULT_API_BASE",
  "default_format": "table",
  "api_token": "",
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
    
    if [[ -n "$API_TOKEN" ]]; then
        curl_args+=("-H" "Authorization: Bearer $API_TOKEN")
    fi
    
    if [[ -n "$data" ]]; then
        curl_args+=("-d" "$data")
    fi
    
    local url="${API_BASE}${endpoint}"
    curl_args+=("$url")
    
    local response
    response=$(curl "${curl_args[@]}" 2>/dev/null) || {
        error "Failed to connect to API at $API_BASE"
        error "Is the metareasoning API server running?"
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
        "yaml")
            echo "$data" | jq -r 'to_entries[] | "\(.key): \(.value)"'
            ;;
        *)
            echo "$data" | jq '.'
            ;;
    esac
}

# Format data as table
format_table() {
    local data="$1"
    
    # Check if this is a prompts list
    if echo "$data" | jq -e '.prompts' > /dev/null 2>&1; then
        echo "$data" | jq -r '
            ["ID", "NAME", "CATEGORY", "DESCRIPTION"] as $headers |
            $headers, 
            (.prompts[] | [
                .id[0:8],
                .name,
                .category,
                .description[0:40]
            ]) |
            @tsv
        ' | column -t -s $'\t'
    # Check if this is a workflows list
    elif echo "$data" | jq -e '.workflows' > /dev/null 2>&1; then
        echo "$data" | jq -r '
            ["NAME", "TYPE", "STATUS", "LAST_RUN"] as $headers |
            $headers,
            (.workflows[] | [
                .name,
                .type,
                .status,
                .last_run // "never"
            ]) |
            @tsv
        ' | column -t -s $'\t'
    else
        echo "$data" | jq '.'
    fi
}

# Show usage information
show_usage() {
    bold "Agent Metareasoning Manager CLI v$CLI_VERSION"
    echo "Command-line interface for managing metareasoning tools and decision support"
    echo ""
    echo "Usage: $CLI_NAME <command> [options] [args]"
    echo ""
    echo "Commands:"
    echo "  prompt <subcommand>       Manage reasoning prompts"
    echo "  workflow <subcommand>     Manage and execute workflows"
    echo "  analyze <type> <input>    Perform analysis (decision, pros-cons, swot, risk)"
    echo "  template <subcommand>     Manage reasoning templates"
    echo "  health                    Check API server health"
    echo "  config                    Manage CLI configuration"
    echo "  help                      Show this help message"
    echo ""
    echo "Prompt Subcommands:"
    echo "  list [--category TYPE]    List available prompts"
    echo "  show <prompt-id>          Show prompt details"
    echo "  create                    Create a new prompt"
    echo "  test <prompt-id>          Test a prompt with sample input"
    echo ""
    echo "Workflow Subcommands:"
    echo "  list [--type TYPE]        List available workflows"
    echo "  run <workflow-name>       Execute a workflow"
    echo "  status <execution-id>     Check workflow execution status"
    echo ""
    echo "Analysis Types:"
    echo "  decision                  Analyze a decision scenario"
    echo "  pros-cons                 Generate pros and cons analysis"
    echo "  swot                      Perform SWOT analysis"
    echo "  risk                      Assess risks"
    echo ""
    echo "Global Options:"
    echo "  --format FORMAT           Output format (json, table, yaml)"
    echo "  --api-base URL           Override API base URL"
    echo ""
    echo "Examples:"
    echo "  $CLI_NAME prompt list"
    echo "  $CLI_NAME workflow run pros-cons-analyzer"
    echo "  $CLI_NAME analyze decision 'Should we migrate to microservices?'"
    echo "  $CLI_NAME health"
    echo ""
}

# Parse command line arguments
parse_args() {
    COMMAND=""
    SUBCOMMAND=""
    ARGS=()
    FORMAT="$DEFAULT_FORMAT"
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --format)
                FORMAT="$2"
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
                elif [[ -z "$SUBCOMMAND" ]] && [[ "$COMMAND" =~ ^(prompt|workflow|template|config)$ ]]; then
                    SUBCOMMAND="$1"
                else
                    ARGS+=("$1")
                fi
                shift
                ;;
        esac
    done
}

# Health check command
cmd_health() {
    log "Checking API server health..."
    
    local response
    response=$(api_request "GET" "/health") || exit 1
    
    local status
    status=$(echo "$response" | jq -r '.status')
    
    case "$status" in
        "healthy")
            success "API server is healthy"
            ;;
        "degraded")
            warn "API server is running but some services are unavailable"
            ;;
        "unhealthy")
            error "API server is unhealthy"
            ;;
        *)
            warn "Unknown health status: $status"
            ;;
    esac
    
    format_output "$response" "$FORMAT"
}

# Prompt management commands
cmd_prompt() {
    case "$SUBCOMMAND" in
        "list")
            log "Fetching prompts..."
            local response
            response=$(api_request "GET" "/prompts") || exit 1
            format_output "$response" "$FORMAT"
            ;;
        "show")
            local prompt_id="${ARGS[0]:-}"
            if [[ -z "$prompt_id" ]]; then
                error "Prompt ID required"
                exit 1
            fi
            
            log "Fetching prompt details..."
            local response
            response=$(api_request "GET" "/prompts/$prompt_id") || exit 1
            format_output "$response" "$FORMAT"
            ;;
        "create"|"test")
            warn "Prompt $SUBCOMMAND command - implementation pending"
            ;;
        *)
            error "Unknown prompt subcommand: $SUBCOMMAND"
            echo "Available: list, show, create, test"
            exit 1
            ;;
    esac
}

# Workflow management commands
cmd_workflow() {
    case "$SUBCOMMAND" in
        "list")
            log "Fetching workflows..."
            local response
            response=$(api_request "GET" "/workflows") || exit 1
            format_output "$response" "$FORMAT"
            ;;
        "run"|"status")
            warn "Workflow $SUBCOMMAND command - implementation pending"
            ;;
        *)
            error "Unknown workflow subcommand: $SUBCOMMAND"
            echo "Available: list, run, status"
            exit 1
            ;;
    esac
}

# Analysis commands
cmd_analyze() {
    local analysis_type="$SUBCOMMAND"
    local input="${ARGS[0]:-}"
    
    if [[ -z "$input" ]]; then
        error "Analysis input required"
        exit 1
    fi
    
    case "$analysis_type" in
        "decision")
            log "Performing decision analysis..."
            ;;
        "pros-cons")
            log "Generating pros-cons analysis..."
            ;;
        "swot")
            log "Performing SWOT analysis..."
            ;;
        "risk")
            log "Assessing risks..."
            ;;
        *)
            error "Unknown analysis type: $analysis_type"
            echo "Available: decision, pros-cons, swot, risk"
            exit 1
            ;;
    esac
    
    local payload
    payload=$(jq -n --arg input "$input" --arg type "$analysis_type" '{
        input: $input,
        type: $type,
        options: {}
    }')
    
    local response
    response=$(api_request "POST" "/analyze/$analysis_type" "$payload") || exit 1
    format_output "$response" "$FORMAT"
}

# Template management commands
cmd_template() {
    case "$SUBCOMMAND" in
        "list")
            log "Fetching templates..."
            local response
            response=$(api_request "GET" "/templates") || exit 1
            format_output "$response" "$FORMAT"
            ;;
        *)
            warn "Template $SUBCOMMAND command - implementation pending"
            ;;
    esac
}

# Configuration management command
cmd_config() {
    case "$SUBCOMMAND" in
        "show")
            if [[ -f "$CONFIG_FILE" ]]; then
                bold "Configuration ($CONFIG_FILE):"
                cat "$CONFIG_FILE" | jq '.'
            else
                warn "No configuration file found"
            fi
            ;;
        "set")
            local key="${ARGS[0]:-}"
            local value="${ARGS[1]:-}"
            
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
            rm -f "$CONFIG_FILE"
            create_default_config
            success "Configuration reset to defaults"
            ;;
        *)
            error "Unknown config subcommand: $SUBCOMMAND"
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
        "prompt")
            cmd_prompt
            ;;
        "workflow")
            cmd_workflow
            ;;
        "analyze")
            cmd_analyze
            ;;
        "template")
            cmd_template
            ;;
        "health")
            cmd_health
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