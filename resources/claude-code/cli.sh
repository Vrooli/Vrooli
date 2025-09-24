#!/usr/bin/env bash
################################################################################
# Claude Code Resource CLI - v2.0 Universal Contract Compliant
# 
# AI development assistant with session management, batch processing, and MCP support
#
# Usage:
#   resource-claude-code <command> [options]
#   resource-claude-code <group> <subcommand> [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    CLAUDE_CODE_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    # Recalculate APP_ROOT from resolved symlink location
    APP_ROOT="$(builtin cd "${CLAUDE_CODE_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
CLAUDE_CODE_CLI_DIR="${APP_ROOT}/resources/claude-code"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"

# Source agent management (load config and manager directly)
if [[ -f "${APP_ROOT}/resources/claude-code/config/agents.conf" ]]; then
    source "${APP_ROOT}/resources/claude-code/config/agents.conf"
    source "${APP_ROOT}/scripts/resources/agents/agent-manager.sh"
fi
# shellcheck disable=SC1091
source "${CLAUDE_CODE_CLI_DIR}/config/defaults.sh"

# Source Claude Code libraries in dependency order
# Load common.sh first as it contains core functions used by other modules
if [[ -f "${CLAUDE_CODE_CLI_DIR}/lib/common.sh" ]]; then
    # shellcheck disable=SC1091
    source "${CLAUDE_CODE_CLI_DIR}/lib/common.sh"
else
    log::error "Critical: common.sh library not found"
    exit 1
fi

# Load other libraries in dependency order (common.sh is already loaded)
for lib in status install session session-enhanced mcp templates settings automation execute batch error-handling content agents; do
    lib_file="${CLAUDE_CODE_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        if ! source "$lib_file"; then
            log::warn "Failed to load library: $lib_file"
            # Continue loading other libraries, but warn about failures
        fi
    else
        log::debug "Optional library not found: $lib_file"
    fi
done

# Initialize CLI framework in v2.0 mode (auto-creates manage/test/content groups)
cli::init "claude-code" "Claude Code AI development assistant" "v2"

# Override default handlers to point directly to claude-code implementations
CLI_COMMAND_HANDLERS["manage::install"]="claude_code::install"
CLI_COMMAND_HANDLERS["manage::uninstall"]="claude_code::uninstall"
CLI_COMMAND_HANDLERS["manage::start"]="claude_code::start"
CLI_COMMAND_HANDLERS["manage::stop"]="claude_code::stop"
CLI_COMMAND_HANDLERS["manage::restart"]="claude_code::restart"
CLI_COMMAND_HANDLERS["test::smoke"]="claude_code::test"
CLI_COMMAND_HANDLERS["test::integration"]="claude_code::test"
CLI_COMMAND_HANDLERS["test::all"]="claude_code::test"

# Override content handlers for Claude Code-specific functionality  
CLI_COMMAND_HANDLERS["content::add"]="claude_code::content::add"
CLI_COMMAND_HANDLERS["content::list"]="claude_code::content::list"
CLI_COMMAND_HANDLERS["content::get"]="claude_code::content::get"
CLI_COMMAND_HANDLERS["content::remove"]="claude_code::content::remove"

# Add optional content subcommands for business operations
cli::register_subcommand "content" "execute" "Execute template or prompt" "claude_code::content::execute"
cli::register_subcommand "content" "inject" "Inject templates/prompts (legacy)" "claude_code_inject"

# Add manage subcommands for Claude Code-specific management
cli::register_subcommand "manage" "session" "Session management" "claude_code_session"
cli::register_subcommand "manage" "mcp" "MCP server management" "claude_code_mcp"
cli::register_subcommand "manage" "template" "Template management" "claude_code_template"  
cli::register_subcommand "manage" "batch" "Batch processing" "claude_code_batch"
cli::register_subcommand "manage" "settings" "Settings management" "claude_code_settings"
cli::register_subcommand "manage" "for" "Adapter management (litellm)" "claude_code_for"

# TOP-LEVEL CUSTOM COMMANDS - Essential for auto/ folder functionality
cli::register_command "run" "Run prompt with Claude Code (CRITICAL for auto/)" "claude_code_run" "modifies-system"
cli::register_command "usage" "Show usage statistics and limits" "claude_code_usage"
cli::register_command "set-tier" "Set subscription tier for accurate limits" "claude_code_set_tier" "modifies-system"
cli::register_command "reset-usage" "Reset usage counters (testing)" "claude_code_reset_usage" "modifies-system" 
cli::register_command "test-rate-limit" "Test rate limit detection (diagnostic)" "claude_code_test_rate_limit"

# Information commands
cli::register_command "status" "Show detailed resource status" "claude_code::status"
cli::register_command "logs" "Show Claude Code logs" "claude_code::logs"

# Agent management commands
# Create wrapper for agents command that delegates to manager
claude_code::agents::command() {
    if type -t agent_manager::load_config &>/dev/null; then
        "${APP_ROOT}/scripts/resources/agents/agent-manager.sh" --config="claude-code" "$@"
    else
        log::error "Agent management not available"
        return 1
    fi
}
export -f claude_code::agents::command

cli::register_command "agents" "Manage running Claude Code agents" "claude_code::agents::command"

################################################################################
# Agent cleanup function
################################################################################

#######################################
# Setup agent cleanup on signals
# Arguments:
#   $1 - Agent ID
#######################################
claude_code::kill_children() {
    local signal="$1"
    local child

    [[ "$signal" != SIG* ]] && signal="SIG${signal}"

    if [[ -r "/proc/$$/task/$$/children" ]]; then
        while read -r child; do
            [[ -n "$child" ]] || continue
            kill "-$signal" "$child" 2>/dev/null || kill "-$signal" "-$child" 2>/dev/null || true
        done < "/proc/$$/task/$$/children"
    else
        for child in $(pgrep -P $$ 2>/dev/null); do
            kill "-$signal" "$child" 2>/dev/null || kill "-$signal" "-$child" 2>/dev/null || true
        done
    fi
}

claude_code::setup_agent_cleanup() {
    local agent_id="$1"

    # Export the agent ID so trap can access it
    export CLAUDE_CODE_CURRENT_AGENT_ID="$agent_id"

    # Ensure we track the last known exit status
    export CLAUDE_CODE_LAST_EXIT=${CLAUDE_CODE_LAST_EXIT:-1}

    claude_code::agent_cleanup() {
        local signal="${1:-EXIT}"
        local exit_code=${CLAUDE_CODE_LAST_EXIT:-$?}

        trap - EXIT SIGTERM SIGINT

        # Forward the signal to child processes when invoked via signal traps
        if [[ "$signal" != "EXIT" ]]; then
            claude_code::kill_children "$signal"

            local sig_name="$signal"
            [[ "$sig_name" != SIG* ]] && sig_name="SIG${sig_name}"
            local sig_number
            sig_number=$(kill -l "$sig_name" 2>/dev/null || echo 0)
            if [[ "$sig_number" =~ ^[0-9]+$ && "$sig_number" -gt 0 ]]; then
                exit_code=$((128 + sig_number))
            fi
        else
            claude_code::kill_children SIGTERM
        fi

        if [[ -n "${CLAUDE_CODE_CURRENT_AGENT_ID:-}" ]] && type -t agent_manager::unregister &>/dev/null; then
            agent_manager::unregister "${CLAUDE_CODE_CURRENT_AGENT_ID}" >/dev/null 2>&1 || true
        fi

        # Allow any remaining children to exit without hanging forever
        local cleanup_deadline=$((SECONDS + 5))
        while true; do
            local child_pids
            child_pids=$(pgrep -P $$ 2>/dev/null || true)
            if [[ -z "$child_pids" ]]; then
                break
            fi

            if (( SECONDS >= cleanup_deadline )); then
                for child in $child_pids; do
                    kill -KILL "$child" 2>/dev/null || true
                done
                break
            fi

            sleep 0.1
        done

        wait 2>/dev/null || true

        unset CLAUDE_CODE_CURRENT_AGENT_ID
        exit "$exit_code"
    }

    # Register cleanup handlers
    trap 'claude_code::agent_cleanup EXIT' EXIT
    trap 'claude_code::agent_cleanup SIGINT' SIGINT
    trap 'claude_code::agent_cleanup SIGTERM' SIGTERM
}

################################################################################
# Preserved wrapper functions for backward compatibility and complex operations
################################################################################

# Session management wrapper
claude_code_session() {
    local action="${1:-list}"
    shift || true
    
    case "$action" in
        list)
            if command -v claude_code::session_list &>/dev/null; then
                claude_code::session_list "$@"
            else
                claude-code session list "$@"
            fi
            ;;
        resume)
            if command -v claude_code::session_resume &>/dev/null; then
                claude_code::session_resume "$@"
            else
                claude-code session resume "$@"
            fi
            ;;
        delete)
            if command -v claude_code::session_delete &>/dev/null; then
                claude_code::session_delete "$@"
            else
                claude-code session delete "$@"
            fi
            ;;
        view)
            if command -v claude_code::session_view &>/dev/null; then
                claude_code::session_view "$@"
            else
                claude-code session view "$@"
            fi
            ;;
        analytics)
            if command -v claude_code::session_analytics &>/dev/null; then
                claude_code::session_analytics "$@"
            else
                log::error "Session analytics not available"
                return 1
            fi
            ;;
        *)
            log::error "Unknown session action: $action"
            echo "Usage: resource-claude-code manage session <action>"
            echo ""
            echo "Available actions:"
            echo "  list        List all sessions"
            echo "  resume <id> Resume a session" 
            echo "  delete <id> Delete a session"
            echo "  view <id>   View session details"
            echo "  analytics   Show session analytics"
            return 1
            ;;
    esac
}

# MCP server management wrapper
claude_code_mcp() {
    local action="${1:-status}"
    shift || true
    
    case "$action" in
        register)
            if command -v claude_code::register_mcp &>/dev/null; then
                claude_code::register_mcp "$@"
            else
                log::error "MCP registration not available"
                return 1
            fi
            ;;
        unregister)
            if command -v claude_code::unregister_mcp &>/dev/null; then
                claude_code::unregister_mcp "$@"
            else
                log::error "MCP unregistration not available"
                return 1
            fi
            ;;
        status)
            if command -v claude_code::mcp_status &>/dev/null; then
                claude_code::mcp_status "$@"
            else
                log::error "MCP status not available"
                return 1
            fi
            ;;
        test)
            if command -v claude_code::mcp_test &>/dev/null; then
                claude_code::mcp_test "$@"
            else
                log::error "MCP test not available"
                return 1
            fi
            ;;
        *)
            log::error "Unknown MCP action: $action"
            echo "Usage: resource-claude-code manage mcp <action>"
            echo ""
            echo "Available actions:"
            echo "  register   Register MCP server"
            echo "  unregister Unregister MCP server"
            echo "  status     Show MCP status"
            echo "  test       Test MCP connection"
            return 1
            ;;
    esac
}

# Template management wrapper
claude_code_template() {
    local action="${1:-list}"
    shift || true
    
    case "$action" in
        list)
            if command -v claude_code::templates_list &>/dev/null; then
                claude_code::templates_list "$@"
            else
                log::error "Template listing not available"
                return 1
            fi
            ;;
        load)
            if command -v claude_code::template_load &>/dev/null; then
                claude_code::template_load "$@"
            else
                log::error "Template loading not available"
                return 1
            fi
            ;;
        run)
            if command -v claude_code::template_run &>/dev/null; then
                claude_code::template_run "$@"
            else
                log::error "Template run not available"
                return 1
            fi
            ;;
        create)
            if command -v claude_code::template_create &>/dev/null; then
                claude_code::template_create "$@"
            else
                log::error "Template creation not available"
                return 1
            fi
            ;;
        info)
            if command -v claude_code::template_info &>/dev/null; then
                claude_code::template_info "$@"
            else
                log::error "Template info not available"
                return 1
            fi
            ;;
        *)
            log::error "Unknown template action: $action"
            echo "Usage: resource-claude-code manage template <action>"
            echo ""
            echo "Available actions:"
            echo "  list          List templates"
            echo "  load <name>   Load a template"
            echo "  run <name>    Run a template"
            echo "  create <name> Create a template"
            echo "  info <name>   Show template info"
            return 1
            ;;
    esac
}

# Batch processing wrapper
claude_code_batch() {
    local type="${1:-simple}"
    shift || true
    
    case "$type" in
        simple)
            # Handle file input properly for simple batch processing
            local file_or_prompt="${1:-}"
            local total_turns="${2:-5}"
            local batch_size="${3:-50}"
            local allowed_tools="${4:-Read,Edit,Write,Bash}"
            local output_dir="${5:-}"
            
            if [[ -z "$file_or_prompt" ]]; then
                log::error "File path or prompt required for batch processing"
                echo "Usage: resource-claude-code manage batch simple <file|prompt> [total_turns] [batch_size] [allowed_tools] [output_dir]"
                return 1
            fi
            
            local prompt
            if [[ -f "$file_or_prompt" ]]; then
                prompt=$(cat "$file_or_prompt")
                if [[ -z "$prompt" ]]; then
                    log::error "File is empty: $file_or_prompt"
                    return 1
                fi
                log::info "Reading prompts from file: $file_or_prompt"
            else
                prompt="$file_or_prompt"
            fi
            
            # Always use non-interactive mode
            export CLAUDE_NON_INTERACTIVE="true"
            
            if command -v claude_code::batch_simple &>/dev/null; then
                claude_code::batch_simple "$prompt" "$total_turns" "$batch_size" "$allowed_tools" "$output_dir"
            else
                log::error "Simple batch not available"
                return 1
            fi
            ;;
        config)
            if command -v claude_code::batch_config &>/dev/null; then
                export CLAUDE_NON_INTERACTIVE="true"
                claude_code::batch_config "$@"
            else
                log::error "Config batch not available"
                return 1
            fi
            ;;
        multi)
            if command -v claude_code::batch_multi &>/dev/null; then
                export CLAUDE_NON_INTERACTIVE="true"
                claude_code::batch_multi "$@"
            else
                log::error "Multi batch not available"
                return 1
            fi
            ;;
        parallel)
            if command -v claude_code::batch_parallel &>/dev/null; then
                export CLAUDE_NON_INTERACTIVE="true"
                claude_code::batch_parallel "$@"
            else
                log::error "Parallel batch not available"
                return 1
            fi
            ;;
        *)
            log::error "Unknown batch type: $type"
            echo "Usage: resource-claude-code manage batch <type>"
            echo ""
            echo "Available types:"
            echo "  simple    Run simple batch"
            echo "  config    Run with config file"
            echo "  multi     Process multiple files"
            echo "  parallel  Process in parallel"
            return 1
            ;;
    esac
}

# Settings management wrapper
claude_code_settings() {
    local action="${1:-show}"
    shift || true
    
    case "$action" in
        show)
            if command -v claude_code::settings &>/dev/null; then
                claude_code::settings
            else
                claude-code settings
            fi
            ;;
        get)
            if command -v claude_code::settings_get &>/dev/null; then
                claude_code::settings_get "$@"
            else
                log::error "Settings get not available"
                return 1
            fi
            ;;
        set)
            if command -v claude_code::settings_set &>/dev/null; then
                claude_code::settings_set "$@"
            else
                log::error "Settings set not available"
                return 1
            fi
            ;;
        reset)
            if command -v claude_code::settings_reset &>/dev/null; then
                claude_code::settings_reset "$@"
            else
                log::error "Settings reset not available"
                return 1
            fi
            ;;
        tips)
            if command -v claude_code::settings_tips &>/dev/null; then
                claude_code::settings_tips
            else
                log::error "Settings tips not available"
                return 1
            fi
            ;;
        *)
            log::error "Unknown settings action: $action"
            echo "Usage: resource-claude-code manage settings <action>"
            echo ""
            echo "Available actions:"
            echo "  show         Show current settings"
            echo "  get <key>    Get a setting"
            echo "  set <key>    Set a setting"
            echo "  reset        Reset to defaults"
            echo "  tips         Show configuration tips"
            return 1
            ;;
    esac
}

# Adapter management wrapper (for litellm)
claude_code_for() {
    local target="${1:-}"
    local action="${2:-}"
    shift 2 || true
    
    if [[ -z "$target" || -z "$action" ]]; then
        log::error "Usage: resource-claude-code manage for <target> <action> [options]"
        echo ""
        echo "Available targets:"
        echo "  litellm - LiteLLM backend adapter"
        return 1
    fi
    
    case "$target" in
        litellm)
            local adapter_dir="${CLAUDE_CODE_CLI_DIR}/adapters/litellm"
            
            if [[ ! -d "$adapter_dir" ]]; then
                log::error "LiteLLM adapter not found"
                return 1
            fi
            
            case "$action" in
                connect)
                    # shellcheck disable=SC1090
                    source "${adapter_dir}/connect.sh"
                    litellm::connect "$@"
                    ;;
                disconnect)
                    # shellcheck disable=SC1090
                    source "${adapter_dir}/disconnect.sh"
                    litellm::disconnect "$@"
                    ;;
                status)
                    # shellcheck disable=SC1090
                    source "${adapter_dir}/status.sh"
                    local format="${1:-text}"
                    case "$format" in
                        json|--json)
                            litellm::get_full_status
                            ;;
                        text|--text|*)
                            litellm::display_status
                            ;;
                    esac
                    ;;
                config)
                    # shellcheck disable=SC1090
                    source "${adapter_dir}/config.sh"
                    local config_action="${1:-show}"
                    shift || true
                    case "$config_action" in
                        show)
                            litellm::config_show
                            ;;
                        set|set-*)
                            if [[ $# -lt 2 ]]; then
                                log::error "Usage: resource-claude-code manage for litellm config set <setting> <value>"
                                return 1
                            fi
                            local setting="${1#set-}"
                            litellm::config_set "$setting" "$2"
                            ;;
                        get)
                            if [[ $# -lt 1 ]]; then
                                log::error "Usage: resource-claude-code manage for litellm config get <setting>"
                                return 1
                            fi
                            litellm::config_get "$1"
                            ;;
                        *)
                            log::error "Unknown config action: $config_action"
                            echo "Available actions: show, set, get"
                            return 1
                            ;;
                    esac
                    ;;
                test)
                    # shellcheck disable=SC1090
                    source "${adapter_dir}/status.sh"
                    if litellm::test_connection "${1:-test}"; then
                        log::success "✅ LiteLLM connection test successful"
                    else
                        log::error "❌ LiteLLM connection test failed"
                        return 1
                    fi
                    ;;
                *)
                    log::error "Unknown action for litellm: $action"
                    echo "Available actions: connect, disconnect, status, config, test"
                    return 1
                    ;;
            esac
            ;;
        *)
            log::error "Unknown target: $target"
            echo "Available targets: litellm"
            return 1
            ;;
    esac
}

# Run command - CRITICAL for auto/ folder functionality
claude_code_run() {
    local agent_tag=""
    local prompt=""
    
    # Parse arguments properly
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --tag)
                agent_tag="$2"
                shift 2
                ;;
            -)
                prompt="-"
                shift
                break
                ;;
            *)
                prompt="$*"
                break
                ;;
        esac
    done
    
    # Check if prompt is "-" to read from stdin
    if [[ "$prompt" == "-" ]]; then
        # Read entire stdin into prompt
        prompt=$(cat)
        if [[ -z "$prompt" ]]; then
            log::error "No prompt provided via stdin"
            return 1
        fi
    elif [[ -z "$prompt" ]]; then
        log::error "Prompt required"
        echo "Usage: resource-claude-code run [--tag <tag>] <prompt>"
        echo "  or: echo \"prompt\" | resource-claude-code run [--tag <tag>] -"
        echo ""
        echo "Examples:"
        echo "  resource-claude-code run \"Write a hello world program\""
        echo "  resource-claude-code run --tag my-task-123 \"Fix the bug in main.py\""
        echo "  echo \"test prompt\" | resource-claude-code run --tag test-agent -"
        return 1
    fi
    
    # Register agent if agent management is available
    local agent_id=""
    local command_string=""
    if type -t agent_manager::register &>/dev/null; then
        # Use tag as agent_id if provided, otherwise generate
        if [[ -n "$agent_tag" ]]; then
            agent_id="$agent_tag"
            command_string="resource-claude-code run (via ecosystem-manager)"
        else
            agent_id=$(agent_manager::generate_id)
            command_string="resource-claude-code run"
        fi
        
        if agent_manager::register "$agent_id" $$ "$command_string"; then
            log::debug "Registered agent: $agent_id"
            
            # Set up signal handler for cleanup
            claude_code::setup_agent_cleanup "$agent_id"
            
            # Metrics will be tracked during actual operation execution
        fi
    fi
    
    # For large prompts, use a temp file to avoid environment variable size limits
    local prompt_file
    prompt_file=$(mktemp)
    echo "$prompt" > "$prompt_file"
    
    # Set environment variables for claude_code::run function (except prompt)
    export PROMPT_FILE="$prompt_file"  # Pass file path instead of content
    export MAX_TURNS="${MAX_TURNS:-30}"
    export TIMEOUT="${TIMEOUT:-600}"
    export OUTPUT_FORMAT="${OUTPUT_FORMAT:-text}"
    export ALLOWED_TOOLS="${ALLOWED_TOOLS:-Read,Write,Edit,Bash,LS,Glob,Grep}"
    export SKIP_PERMISSIONS="${SKIP_PERMISSIONS:-yes}"
    
    # Always use non-interactive mode for autonomous platform
    export CLAUDE_NON_INTERACTIVE="true"

    local result=0
    local start_time=$(date +%s%3N)  # Milliseconds

    # Default last-exit status in case of early termination
    export CLAUDE_CODE_LAST_EXIT=1
    
    # Track operation start metrics
    if [[ -n "$agent_id" ]] && type -t agents::metrics::increment &>/dev/null; then
        agents::metrics::increment "${REGISTRY_FILE:-${APP_ROOT}/.vrooli/claude-code-agents.json}" "$agent_id" "requests" 1
    fi
    
    if command -v claude_code::run &>/dev/null; then
        claude_code::run
        result=$?
    else
        # Fall back to direct claude command with non-interactive mode
        echo "$prompt" | claude --print --max-turns "${MAX_TURNS:-5}"
        result=$?
    fi
    export CLAUDE_CODE_LAST_EXIT=$result
    
    # Track operation completion metrics
    if [[ -n "$agent_id" ]] && type -t agents::metrics::histogram &>/dev/null; then
        local end_time=$(date +%s%3N)
        local duration=$((end_time - start_time))
        agents::metrics::histogram "${REGISTRY_FILE:-${APP_ROOT}/.vrooli/claude-code-agents.json}" "$agent_id" "request_duration_ms" "$duration"
        
        # Track success/error
        if [[ $result -eq 0 ]]; then
            log::debug "Claude Code operation completed successfully"
        else
            type -t agents::metrics::increment &>/dev/null && \
                agents::metrics::increment "${REGISTRY_FILE:-${APP_ROOT}/.vrooli/claude-code-agents.json}" "$agent_id" "errors" 1
        fi
        
        # Update process metrics
        if type -t agents::metrics::gauge &>/dev/null; then
            # Get current process memory usage (in MB)
            local memory_kb=$(ps -o rss= -p $$ 2>/dev/null | awk '{print $1}' || echo "0")
            local memory_mb=$((memory_kb / 1024))
            agents::metrics::gauge "${REGISTRY_FILE:-${APP_ROOT}/.vrooli/claude-code-agents.json}" "$agent_id" "memory_mb" "$memory_mb"
        fi
    fi
    
    # Clean up
    rm -f "$prompt_file"
    
    # Unregister agent on completion
    if [[ -n "$agent_id" ]] && type -t agent_manager::unregister &>/dev/null; then
        agent_manager::unregister "$agent_id" >/dev/null 2>&1
    fi
    
    return $result
}

# Usage statistics
claude_code_usage() {
    local format="${1:-text}"
    
    # Check usage limits
    claude_code::check_usage_limits
    local usage_status=$?
    
    # Get usage data
    local usage_json=$(claude_code::get_usage)
    
    if [[ "$format" == "json" ]]; then
        echo "$usage_json"
    else
        log::header "📊 Claude Code Usage Statistics"
        
        # Calculate current usage from timestamped data
        local current_hour=$(date +%Y%m%d%H)
        local current_day=$(date +%Y%m%d)
        local current_week=$(date +%Y%U)
        
        local hourly=$(echo "$usage_json" | jq -r ".hourly_requests.\"$current_hour\" // 0")
        local daily=$(echo "$usage_json" | jq -r ".daily_requests.\"$current_day\" // 0")
        local weekly=$(echo "$usage_json" | jq -r ".weekly_requests.\"$current_week\" // 0")
        
        # Calculate last 5 hours usage
        local last_5h=0
        for i in {0..4}; do
            local hour_key=$(date -d "-$i hours" +%Y%m%d%H)
            local hour_count=$(echo "$usage_json" | jq -r ".hourly_requests.\"$hour_key\" // 0")
            last_5h=$((last_5h + hour_count))
        done
        
        # Subscription tier and limits
        local tier=$(echo "$usage_json" | jq -r '.subscription_tier')
        local tier_key="${tier:-free}"
        [[ "$tier_key" == "unknown" ]] && tier_key="free"
        
        # Map max subscription to max_100 (default) unless specified otherwise
        if [[ "$tier_key" == "max" ]]; then
            tier_key="max_100"
        fi
        
        local limit_5h=$(echo "$usage_json" | jq -r ".estimated_limits.${tier_key}.\"5_hour\"")
        local limit_daily=$(echo "$usage_json" | jq -r ".estimated_limits.${tier_key}.daily")
        local limit_weekly=$(echo "$usage_json" | jq -r ".estimated_limits.${tier_key}.weekly")
        
        echo ""
        if [[ "$tier" == "unknown" ]]; then
            log::warn "Subscription Tier: Unknown (using Free tier limits)"
            echo "  💡 Set your tier for accurate limits: resource-claude-code set-tier <tier>"
            echo "     Options: free, pro, teams, enterprise"
        else
            log::info "Subscription Tier: $tier"
        fi
        echo ""
        
        echo "Current Usage:"
        echo "  This hour:    $hourly requests"
        echo "  Last 5 hours: $last_5h / $limit_5h (rolling window)"
        echo "  Today:        $daily / $limit_daily"
        echo "  This week:    $weekly / $limit_weekly"
        echo ""
        
        # Status indicator
        case $usage_status in
            0)
                log::success "✅ Usage is within safe limits"
                ;;
            1)
                log::warn "⚠️  High usage detected - approaching limits"
                ;;
            2)
                log::error "❌ Critical usage - rate limits imminent"
                log::info "Consider waiting or switching to LiteLLM fallback"
                ;;
        esac
    fi
}

# Set subscription tier
claude_code_set_tier() {
    local tier="${1:-}"
    
    if [[ -z "$tier" ]]; then
        log::error "Subscription tier required"
        echo ""
        echo "Usage: resource-claude-code set-tier <tier>"
        echo ""
        echo "Available tiers:"
        echo "  free       - Free tier (limited trial)"
        echo "  pro        - Pro tier (\$20/month)"
        echo "  teams      - Teams tier (\$100/month)"
        echo "  enterprise - Enterprise tier (\$200/month)"
        return 1
    fi
    
    claude_code::set_subscription_tier "$tier"
}

# Reset usage counters
claude_code_reset_usage() {
    local type="${1:-}"
    
    if [[ -z "$type" ]]; then
        log::error "Reset type required"
        echo ""
        echo "Usage: resource-claude-code reset-usage <type>"
        echo ""
        echo "Available types:"
        echo "  hourly   - Reset hourly counters"
        echo "  daily    - Reset daily counters"
        echo "  weekly   - Reset weekly counters"
        echo "  all      - Reset all counters"
        return 1
    fi
    
    log::warn "⚠️  This will reset usage tracking counters"
    read -p "Are you sure? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log::info "Reset cancelled"
        return 0
    fi
    
    claude_code::reset_usage_counters "$type"
}

# Test rate limit detection
claude_code_test_rate_limit() {
    local test_input="${1:-}"
    
    log::header "🧪 Testing Rate Limit Detection"
    echo ""
    
    # If no input provided, use the example JSON
    if [[ -z "$test_input" ]]; then
        test_input='{"type":"result","subtype":"success","is_error":true,"duration_ms":393,"duration_api_ms":0,"num_turns":1,"result":"Claude AI usage limit reached|1755806400","session_id":"71b47827-9527-47f3-8614-34249f877747","total_cost_usd":0,"usage":{"input_tokens":0,"cache_creation_input_tokens":0,"cache_read_input_tokens":0,"output_tokens":0,"server_tool_use":{"web_search_requests":0}},"service_tier":"standard"}'
        log::info "Using sample rate limit JSON response for testing"
    fi
    
    # Test the detection function
    local rate_info=$(claude_code::detect_rate_limit "$test_input" "1")
    local detected=$(echo "$rate_info" | jq -r '.detected')
    
    if [[ "$detected" == "true" ]]; then
        log::success "✅ Rate limit detected successfully!"
        echo ""
        echo "Detection results:"
        echo "$rate_info" | jq '.'
        
        # Test recording the rate limit
        log::info "Testing rate limit recording..."
        claude_code::record_rate_limit "$rate_info"
        
        # Show updated usage
        echo ""
        log::info "Updated usage statistics:"
        local usage_json=$(claude_code::get_usage)
        echo "  Last 5 hours: $(echo "$usage_json" | jq -r '.last_5_hours') requests"
        echo "  Daily: $(echo "$usage_json" | jq -r '.current_day_requests') requests"
        echo "  Weekly: $(echo "$usage_json" | jq -r '.current_week_requests') requests"
        
        # Check if last rate limit was recorded
        local last_rate_limit=$(echo "$usage_json" | jq -r '.last_rate_limit // null')
        if [[ "$last_rate_limit" != "null" ]]; then
            log::success "✅ Rate limit was properly recorded"
        else
            log::warn "⚠️  Rate limit was not recorded in usage tracking"
        fi
    else
        log::error "❌ Rate limit was not detected"
        echo "Detection results:"
        echo "$rate_info" | jq '.'
    fi
    
    echo ""
    log::info "You can test with custom input:"
    echo "  resource-claude-code test-rate-limit '<json_or_text_output>'"
}

# Inject wrapper (legacy compatibility)
claude_code_inject() {
    local file="${1:-}"
    
    if [[ -z "$file" ]]; then
        log::error "File path required for injection"
        echo "Usage: resource-claude-code content inject <file.json>"
        return 1
    fi
    
    # Handle shared: prefix
    if [[ "$file" == shared:* ]]; then
        file="${var_ROOT_DIR}/${file#shared:}"
    fi
    
    if [[ ! -f "$file" ]]; then
        log::error "File not found: $file"
        return 1
    fi
    
    # Export log functions for inject script
    export -f log::header log::info log::error log::success log::warn \
              log::echo_color log::get_color_code log::initialize_color \
              log::initialize_reset log::subheader log::warning log::prompt \
              log::debug 2>/dev/null || true
    
    # Use the inject script with proper environment
    VROOLI_ROOT="${var_ROOT_DIR}" \
    RESOURCE_DIR="${RESOURCE_DIR}" \
    "${CLAUDE_CODE_CLI_DIR}/inject.sh" --inject "$file"
}

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi
