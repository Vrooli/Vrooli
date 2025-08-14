#!/usr/bin/env bash
################################################################################
# Claude Code Resource CLI
# 
# Lightweight CLI interface for Claude Code that delegates to existing lib functions.
#
# Usage:
#   resource-claude-code <command> [options]
#
################################################################################

set -euo pipefail

# Get script directory (resolving symlinks)
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    # If this is a symlink, resolve it
    CLAUDE_CODE_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
else
    CLAUDE_CODE_CLI_SCRIPT="${BASH_SOURCE[0]}"
fi
CLAUDE_CODE_CLI_DIR="$(cd "$(dirname "$CLAUDE_CODE_CLI_SCRIPT")" && pwd)"
VROOLI_ROOT="${VROOLI_ROOT:-$(cd "$CLAUDE_CODE_CLI_DIR/../../../.." && pwd)}"
export VROOLI_ROOT
export RESOURCE_DIR="$CLAUDE_CODE_CLI_DIR"
export CLAUDE_CODE_SCRIPT_DIR="$CLAUDE_CODE_CLI_DIR"  # For compatibility with existing libs

# Source utilities first
# shellcheck disable=SC1091
source "${VROOLI_ROOT}/scripts/lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LOG_FILE:-${VROOLI_ROOT}/scripts/lib/utils/log.sh}" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE:-${VROOLI_ROOT}/scripts/resources/common.sh}" 2>/dev/null || true

# Source the CLI template
# shellcheck disable=SC1091
source "${VROOLI_ROOT}/scripts/lib/resources/cli/resource-cli-template.sh"

# Initialize with resource name (before sourcing configs that may set it as readonly)
resource_cli::init "claude-code"

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

################################################################################
# Delegate to existing claude-code functions
################################################################################

# Inject templates/prompts/sessions into Claude Code
resource_cli::inject() {
    local file="${1:-}"
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ -z "$file" ]]; then
        log::error "File path required for injection"
        echo "Usage: resource-claude-code inject <file.json>"
        return 1
    fi
    
    # Handle shared: prefix
    if [[ "$file" == shared:* ]]; then
        file="${VROOLI_ROOT}/${file#shared:}"
    fi
    
    if [[ ! -f "$file" ]]; then
        log::error "File not found: $file"
        return 1
    fi
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would inject: $file"
        return 0
    fi
    
    # Export log functions for inject script
    export -f log::header log::info log::error log::success log::warn \
              log::echo_color log::get_color_code log::initialize_color \
              log::initialize_reset log::subheader log::warning log::prompt \
              log::debug 2>/dev/null || true
    
    # Use the inject script with proper environment
    VROOLI_ROOT="${VROOLI_ROOT}" \
    RESOURCE_DIR="${RESOURCE_DIR}" \
    "${CLAUDE_CODE_CLI_DIR}/inject.sh" --inject "$file"
}

# Validate Claude Code configuration
resource_cli::validate() {
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
resource_cli::status() {
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
resource_cli::start() {
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would start Claude Code session"
        return 0
    fi
    
    if command -v claude_code::session &>/dev/null; then
        claude_code::session
    else
        log::error "Claude Code session start not available"
        return 1
    fi
}

# Stop Claude Code (not applicable, but kept for consistency)
resource_cli::stop() {
    log::info "Claude Code runs on-demand, no daemon to stop"
    return 0
}

# Install Claude Code
resource_cli::install() {
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would install Claude Code"
        return 0
    fi
    
    if command -v claude_code::install &>/dev/null; then
        claude_code::install
    else
        log::error "claude_code::install not available"
        return 1
    fi
}

# Uninstall Claude Code
resource_cli::uninstall() {
    FORCE="${FORCE:-false}"
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$FORCE" != "true" ]]; then
        echo "⚠️  This will remove Claude Code and all its data. Use --force to confirm."
        return 1
    fi
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would uninstall Claude Code"
        return 0
    fi
    
    if command -v claude_code::uninstall &>/dev/null; then
        claude_code::uninstall
    else
        npm uninstall -g @anthropic/claude-code 2>/dev/null || true
        log::success "Claude Code uninstalled"
    fi
}

################################################################################
# Claude Code-specific commands
################################################################################

# Run a command with Claude Code
claude_code_run() {
    local prompt="${*:-}"
    
    if [[ -z "$prompt" ]]; then
        log::error "Prompt required"
        echo "Usage: resource-claude-code run <prompt>"
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
            echo "Available: list, resume, delete, view, analytics"
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
            echo "Available: register, unregister, status, test"
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
            echo "Available: list, load, run, create, info"
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
            echo "Available: simple, config, multi, parallel"
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
            echo "Available: show, get, set, reset, tips"
            return 1
            ;;
    esac
}

# Show help
resource_cli::show_help() {
    cat << EOF
🚀 Claude Code Resource CLI

USAGE:
    resource-claude-code <command> [options]

CORE COMMANDS:
    inject <file>           Inject templates/prompts into Claude Code
    validate                Validate Claude Code installation
    status                  Show Claude Code status
    start                   Start a Claude Code session
    stop                    (No-op, Claude Code runs on-demand)
    install                 Install Claude Code
    uninstall               Uninstall Claude Code (requires --force)
    
CLAUDE CODE COMMANDS:
    run <prompt>            Run a prompt with Claude Code
    
    session <action>        Session management:
      list                    List all sessions
      resume <id>             Resume a session
      delete <id>             Delete a session
      view <id>               View session details
      analytics               Show session analytics
    
    mcp <action>            MCP server management:
      register                Register MCP server
      unregister              Unregister MCP server
      status                  Show MCP status
      test                    Test MCP connection
    
    template <action>       Template management:
      list                    List templates
      load <name>             Load a template
      run <name>              Run a template
      create <name>           Create a template
      info <name>             Show template info
    
    batch <type>            Batch processing:
      simple <file>           Run simple batch
      config <file>           Run with config file
      multi <files...>        Process multiple files
      parallel <files...>     Process in parallel
    
    settings <action>       Settings management:
      show                    Show current settings
      get <key>               Get a setting
      set <key> <value>       Set a setting
      reset                   Reset to defaults
      tips                    Show configuration tips

OPTIONS:
    --verbose, -v           Show detailed output
    --dry-run               Preview actions without executing
    --force                 Force operation (skip confirmations)

EXAMPLES:
    resource-claude-code status
    resource-claude-code run "Write a hello world program"
    resource-claude-code session list
    resource-claude-code template run react-component
    resource-claude-code batch simple tasks.txt
    resource-claude-code mcp status
    resource-claude-code settings set model claude-3-opus

For more information: https://docs.vrooli.com/resources/claude-code
EOF
}

# Main command router
resource_cli::main() {
    # Parse common options first
    local remaining_args
    remaining_args=$(resource_cli::parse_options "$@")
    set -- $remaining_args
    
    local command="${1:-help}"
    shift || true
    
    case "$command" in
        # Standard resource commands
        inject|validate|status|start|stop|install|uninstall)
            resource_cli::$command "$@"
            ;;
            
        # Claude Code-specific commands
        run)
            claude_code_run "$@"
            ;;
        session)
            claude_code_session "$@"
            ;;
        mcp)
            claude_code_mcp "$@"
            ;;
        template)
            claude_code_template "$@"
            ;;
        batch)
            claude_code_batch "$@"
            ;;
        settings)
            claude_code_settings "$@"
            ;;
            
        help|--help|-h)
            resource_cli::show_help
            ;;
        *)
            log::error "Unknown command: $command"
            echo ""
            resource_cli::show_help
            exit 1
            ;;
    esac
}

# Run main if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    resource_cli::main "$@"
fi