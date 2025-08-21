#!/usr/bin/env bash
################################################################################
# Claude Code Resource CLI
# 
# Lightweight CLI interface for Claude Code using the CLI Command Framework
#
# Usage:
#   resource-claude-code <command> [options]
#
################################################################################

set -euo pipefail

# Get script directory (resolving symlinks for installed CLI)
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    CLAUDE_CODE_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
else
    CLAUDE_CODE_CLI_SCRIPT="${BASH_SOURCE[0]}"
fi
CLAUDE_CODE_CLI_DIR="$(cd "$(dirname "$CLAUDE_CODE_CLI_SCRIPT")" && pwd)"

# Source standard variables
# shellcheck disable=SC1091
source "${CLAUDE_CODE_CLI_DIR}/../../../lib/utils/var.sh"

# Source utilities using var_ variables
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"

# Source the CLI Command Framework
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/cli-command-framework.sh"

# Source Claude Code configuration
# shellcheck disable=SC1091
source "${CLAUDE_CODE_CLI_DIR}/config/defaults.sh" 2>/dev/null || true

# Source Claude Code libraries
for lib in common status install session session-enhanced mcp templates settings automation execute batch error-handling; do
    lib_file="${CLAUDE_CODE_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize CLI framework
cli::init "claude-code" "Claude Code AI development assistant"

# Override help to provide Claude Code-specific examples
cli::register_command "help" "Show this help message with Claude Code examples" "claude_code_show_help"

# Register additional Claude Code-specific commands
cli::register_command "inject" "Inject templates/prompts into Claude Code" "claude_code_inject" "modifies-system"
cli::register_command "run" "Run a prompt with Claude Code" "claude_code_run" "modifies-system"
cli::register_command "session" "Session management" "claude_code_session"
cli::register_command "mcp" "MCP server management" "claude_code_mcp" "modifies-system"
cli::register_command "template" "Template management" "claude_code_template" "modifies-system"
cli::register_command "batch" "Batch processing" "claude_code_batch" "modifies-system"
cli::register_command "settings" "Settings management" "claude_code_settings" "modifies-system"
cli::register_command "usage" "Show usage statistics and limits" "claude_code_usage"
cli::register_command "set-tier" "Set subscription tier for accurate limits" "claude_code_set_tier" "modifies-system"
cli::register_command "reset-usage" "Reset usage counters (for testing)" "claude_code_reset_usage" "modifies-system"
cli::register_command "test-rate-limit" "Test rate limit detection (diagnostic)" "claude_code_test_rate_limit"
cli::register_command "for" "Adapter management (e.g., litellm)" "claude_code_for" "modifies-system"
cli::register_command "uninstall" "Uninstall Claude Code (requires --force)" "claude_code_uninstall" "modifies-system"

################################################################################
# Resource-specific command implementations
################################################################################

# Inject templates/prompts/sessions into Claude Code
claude_code_inject() {
    local file="${1:-}"
    
    if [[ -z "$file" ]]; then
        log::error "File path required for injection"
        echo "Usage: resource-claude-code inject <file.json>"
        echo ""
        echo "Examples:"
        echo "  resource-claude-code inject templates.json"
        echo "  resource-claude-code inject shared:initialization/agents/claude-code/templates.json"
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

# Validate Claude Code configuration
claude_code_validate() {
    if command -v claude_code::test &>/dev/null; then
        claude_code::test
    else
        log::header "Validating Claude Code"
        if command -v claude-code &>/dev/null; then
            log::success "Claude Code is installed"
        else
            log::error "Claude Code not installed"
            return 1
        fi
    fi
}

# Show Claude Code status
claude_code_status() {
    if command -v claude_code::status &>/dev/null; then
        claude_code::status
    else
        log::header "Claude Code Status"
        if command -v claude-code &>/dev/null; then
            echo "Installation: ✅ Installed"
            claude-code --version 2>/dev/null || echo "Version: Unknown"
        else
            echo "Installation: ❌ Not installed"
        fi
    fi
}

# Start Claude Code (session)
claude_code_start() {
    if command -v claude_code::session &>/dev/null; then
        claude_code::session
    else
        log::error "Claude Code session start not available"
        return 1
    fi
}

# Stop Claude Code (not applicable, but kept for consistency)
claude_code_stop() {
    log::info "Claude Code runs on-demand, no daemon to stop"
    return 0
}

# Install Claude Code
claude_code_install() {
    if command -v claude_code::install &>/dev/null; then
        claude_code::install
    else
        log::error "claude_code::install not available"
        return 1
    fi
}

# Uninstall Claude Code
claude_code_uninstall() {
    FORCE="${FORCE:-false}"
    
    if [[ "$FORCE" != "true" ]]; then
        echo "⚠️  This will remove Claude Code and all its data. Use --force to confirm."
        return 1
    fi
    
    if command -v claude_code::uninstall &>/dev/null; then
        claude_code::uninstall
    else
        npm uninstall -g @anthropic/claude-code 2>/dev/null || true
        log::success "Claude Code uninstalled"
    fi
}

# Run a command with Claude Code
claude_code_run() {
    local prompt="${*:-}"
    
    if [[ -z "$prompt" ]]; then
        log::error "Prompt required"
        echo "Usage: resource-claude-code run <prompt>"
        echo ""
        echo "Examples:"
        echo "  resource-claude-code run \"Write a hello world program\""
        echo "  resource-claude-code run \"Fix the bug in main.py\""
        return 1
    fi
    
    # Set environment variables for claude_code::run function
    export PROMPT="$prompt"
    export MAX_TURNS="${MAX_TURNS:-5}"
    export TIMEOUT="${TIMEOUT:-600}"
    export OUTPUT_FORMAT="${OUTPUT_FORMAT:-text}"
    # Default to essential tools for autonomous operation
    export ALLOWED_TOOLS="${ALLOWED_TOOLS:-Read,Write,Edit,Bash,LS,Glob,Grep}"
    export SKIP_PERMISSIONS="${SKIP_PERMISSIONS:-yes}"
    
    # Always use non-interactive mode for autonomous platform
    export CLAUDE_NON_INTERACTIVE="true"
    
    if command -v claude_code::run &>/dev/null; then
        claude_code::run
    else
        # Fall back to direct claude command with non-interactive mode
        echo "$prompt" | claude --print --max-turns "${MAX_TURNS:-5}"
    fi
}

# Session management
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
            echo "Usage: resource-claude-code session <action>"
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

# MCP server management
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
            echo "Usage: resource-claude-code mcp <action>"
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

# Template management
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
            echo "Usage: resource-claude-code template <action>"
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

# Batch processing
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
                echo "Usage: resource-claude-code batch simple <file|prompt> [total_turns] [batch_size] [allowed_tools] [output_dir]"
                echo ""
                echo "Examples:"
                echo "  resource-claude-code batch simple tasks.txt"
                echo "  resource-claude-code batch simple \"Fix all bugs\" 10 25"
                return 1
            fi
            
            local prompt
            if [[ -f "$file_or_prompt" ]]; then
                # Read from file
                prompt=$(cat "$file_or_prompt")
                if [[ -z "$prompt" ]]; then
                    log::error "File is empty: $file_or_prompt"
                    return 1
                fi
                log::info "Reading prompts from file: $file_or_prompt"
            else
                # Use as direct prompt
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
            echo "Usage: resource-claude-code batch <type>"
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

# Settings management
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
            echo "Usage: resource-claude-code settings <action>"
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

# Show usage statistics and limits
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
        
        # Current usage
        local hourly=$(echo "$usage_json" | jq -r '.current_hour_requests')
        local daily=$(echo "$usage_json" | jq -r '.current_day_requests')
        local weekly=$(echo "$usage_json" | jq -r '.current_week_requests')
        local last_5h=$(echo "$usage_json" | jq -r '.last_5_hours')
        
        # Subscription tier and limits
        local tier=$(echo "$usage_json" | jq -r '.subscription_tier')
        local tier_key="${tier:-free}"
        [[ "$tier_key" == "unknown" ]] && tier_key="free"
        
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
        
        # Check for recent rate limit hits
        local last_rate_limit=$(echo "$usage_json" | jq -r '.last_rate_limit // null')
        if [[ "$last_rate_limit" != "null" ]]; then
            local limit_timestamp=$(echo "$last_rate_limit" | jq -r '.timestamp // ""')
            local limit_type=$(echo "$last_rate_limit" | jq -r '.limit_type // "unknown"')
            if [[ -n "$limit_timestamp" ]]; then
                log::warn "⚠️  Last rate limit hit: $limit_timestamp (type: $limit_type)"
            fi
        fi
        echo ""
        
        echo "Current Usage:"
        echo "  This hour:    $hourly requests"
        echo "  Last 5 hours: $last_5h / $limit_5h (rolling window)"
        echo "  Today:        $daily / $limit_daily"
        echo "  This week:    $weekly / $limit_weekly"
        echo ""
        
        # Calculate percentages
        local pct_5h=0
        local pct_daily=0
        local pct_weekly=0
        [[ $limit_5h -gt 0 ]] && pct_5h=$((last_5h * 100 / limit_5h))
        [[ $limit_daily -gt 0 ]] && pct_daily=$((daily * 100 / limit_daily))
        [[ $limit_weekly -gt 0 ]] && pct_weekly=$((weekly * 100 / limit_weekly))
        
        echo "Usage Percentages:"
        echo "  5-hour:  ${pct_5h}%"
        echo "  Daily:   ${pct_daily}%"
        echo "  Weekly:  ${pct_weekly}%"
        echo ""
        
        # Time until reset
        echo "Time Until Reset:"
        echo "  5-hour:  $(claude_code::time_until_reset 5_hour)"
        echo "  Daily:   $(claude_code::time_until_reset daily)"
        echo "  Weekly:  $(claude_code::time_until_reset weekly)"
        echo ""
        
        # Last rate limit encounter
        local last_limit=$(echo "$usage_json" | jq -r '.last_rate_limit')
        if [[ "$last_limit" != "null" ]]; then
            local limit_time=$(echo "$last_limit" | jq -r '.timestamp')
            local limit_type=$(echo "$last_limit" | jq -r '.limit_type')
            echo "Last Rate Limit:"
            echo "  Time: $limit_time"
            echo "  Type: $limit_type"
            echo ""
        fi
        
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
        
        # Show limit source information
        echo ""
        local limits_source=$(echo "$usage_json" | jq -r '.limits_source // null')
        if [[ "$limits_source" != "null" ]]; then
            local last_verified=$(echo "$limits_source" | jq -r '.last_verified // "unknown"')
            local accuracy_note=$(echo "$limits_source" | jq -r '.accuracy_note // ""')
            
            echo "ℹ️  Limit Information:"
            echo "  Based on: Claude Code weekly hour allocations"
            echo "    • Free: Limited trial → ~30 requests/5hr"
            echo "    • Pro (\$20/mo): 40-80 hours/week → ~45 requests/5hr"
            echo "    • Teams (\$100/mo): 140-280 hours/week → ~225 requests/5hr"
            echo "    • Enterprise (\$200/mo): 240-480 hours/week → ~900 requests/5hr"
            echo ""
            echo "  Sources:"
            echo "    • Anthropic Help Center (support.anthropic.com)"
            echo "    • TechCrunch rate limits article (2024-08-28)"
            echo "  Last verified: $last_verified"
            echo ""
            if [[ -n "$accuracy_note" && "$accuracy_note" != "null" ]]; then
                echo "  ⚠️  $accuracy_note"
            fi
            echo "  📝 5-hour limit uses rolling window, not fixed reset times"
            
            # Check if limits might be outdated (>30 days)
            if [[ "$last_verified" != "unknown" && "$last_verified" != "null" ]]; then
                # Try to parse the date safely
                local days_since_verified
                if date -d "$last_verified" &>/dev/null; then
                    days_since_verified=$(( ($(date +%s) - $(date -d "$last_verified" +%s)) / 86400 ))
                    
                    # Only show warning if calculation seems valid (not negative, not huge)
                    if [[ $days_since_verified -gt 30 && $days_since_verified -lt 365 ]]; then
                        log::warn "  ⚠️  Limits last verified ${days_since_verified} days ago - may be outdated"
                        echo "  Check latest at: https://support.anthropic.com/en/articles/11145838"
                    elif [[ $days_since_verified -lt 0 || $days_since_verified -gt 365 ]]; then
                        # Date parsing issue, just show the date
                        echo "  Last verified: $last_verified"
                    fi
                else
                    # Couldn't parse date, just show it
                    echo "  Verification date: $last_verified"
                fi
            fi
        fi
    fi
}

# Set subscription tier for accurate limit tracking
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
        echo ""
        echo "Note: You can also use the old names: max_100, max_200"
        return 1
    fi
    
    claude_code::set_subscription_tier "$tier"
}

# Reset usage counters (mainly for testing)
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

# Test rate limit detection (diagnostic tool)
claude_code_test_rate_limit() {
    local test_input="${1:-}"
    
    log::header "🧪 Testing Rate Limit Detection"
    echo ""
    
    # If no input provided, use the example JSON from the user
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

# Adapter management (for command routing)
claude_code_for() {
    local target="${1:-}"
    local action="${2:-}"
    shift 2 || true
    
    if [[ -z "$target" || -z "$action" ]]; then
        log::error "Usage: resource-claude-code for <target> <action> [options]"
        echo ""
        echo "Available targets:"
        echo "  litellm - LiteLLM backend adapter"
        echo ""
        echo "LiteLLM actions:"
        echo "  connect    - Connect to LiteLLM backend"
        echo "  disconnect - Disconnect from LiteLLM"
        echo "  status     - Check connection status"
        echo "  config     - Manage configuration"
        echo "  test       - Test LiteLLM connection"
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
                    litellm::display_status
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
                                log::error "Usage: resource-claude-code for litellm config set <setting> <value>"
                                return 1
                            fi
                            # Remove 'set-' prefix if present
                            local setting="${1#set-}"
                            litellm::config_set "$setting" "$2"
                            ;;
                        get)
                            if [[ $# -lt 1 ]]; then
                                log::error "Usage: resource-claude-code for litellm config get <setting>"
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

# Custom help function with Claude Code-specific examples
claude_code_show_help() {
    # Show standard framework help first
    cli::_handle_help
    
    # Add Claude Code-specific examples
    echo ""
    echo "🚀 Claude Code AI Development Assistant Examples:"
    echo ""
    echo "Code Generation & Assistance:"
    echo "  resource-claude-code run \"Write a hello world program\"     # Generate code"
    echo "  resource-claude-code run \"Fix the bug in main.py\"          # Debug code"
    echo "  resource-claude-code run \"Add unit tests for utils.js\"     # Write tests"
    echo ""
    echo "Session Management:"
    echo "  resource-claude-code session list                           # List all sessions"
    echo "  resource-claude-code session resume session-123             # Resume session"
    echo "  resource-claude-code session view session-123               # View session"
    echo "  resource-claude-code session analytics                      # Show analytics"
    echo ""
    echo "Template System:"
    echo "  resource-claude-code template list                          # List templates"
    echo "  resource-claude-code template run react-component           # Run template"
    echo "  resource-claude-code template create my-template            # Create template"
    echo ""
    echo "Batch Processing:"
    echo "  resource-claude-code batch simple tasks.txt                 # Process file"
    echo "  resource-claude-code batch parallel task1.txt task2.txt     # Parallel"
    echo ""
    echo "MCP & Settings:"
    echo "  resource-claude-code mcp status                             # Check MCP servers"
    echo "  resource-claude-code settings set model claude-3-opus       # Set model"
    echo "  resource-claude-code inject shared:templates/claude.json    # Import templates"
    echo ""
    echo "Usage Tracking & Rate Limits:"
    echo "  resource-claude-code usage                                  # Show usage stats"
    echo "  resource-claude-code usage json                             # JSON output"
    echo "  resource-claude-code set-tier pro                           # Set subscription tier"
    echo "  resource-claude-code reset-usage daily                      # Reset counters"
    echo ""
    echo "LiteLLM Adapter (Fallback on Rate Limits):"
    echo "  resource-claude-code for litellm connect                    # Use LiteLLM backend"
    echo "  resource-claude-code for litellm disconnect                 # Back to native Claude"
    echo "  resource-claude-code for litellm status                     # Check connection"
    echo "  resource-claude-code for litellm config show                # View configuration"
    echo "  resource-claude-code for litellm config set auto-fallback on # Enable auto-fallback"
    echo ""
    echo "AI Capabilities:"
    echo "  • Advanced code generation and debugging"
    echo "  • Context-aware file editing and refactoring"
    echo "  • Multi-turn conversations with memory"
    echo "  • Template-based workflow automation"
    echo ""
    echo "Integration: CLI tool for direct AI assistance"
    echo "Documentation: https://docs.anthropic.com/claude/docs"
}

################################################################################
# Main execution - dispatch to framework
################################################################################

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi