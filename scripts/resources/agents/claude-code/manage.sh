#!/usr/bin/env bash
set -euo pipefail

# Claude Code management script
# This script manages the Claude Code CLI installation

DESCRIPTION="Manages Claude Code CLI installation and configuration"

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
RESOURCES_DIR="${SCRIPT_DIR}/../.."

# Source common resources
# shellcheck disable=SC1091
source "${RESOURCES_DIR}/common.sh"
# shellcheck disable=SC1091
source "${RESOURCES_DIR}/../helpers/utils/args.sh"

# Source configuration
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/config/defaults.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/config/messages.sh"

# Export configuration
claude_code::export_config

# Source all library modules
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/common.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/install.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/status.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/execute.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/session.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/settings.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/mcp.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/automation.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/batch.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/error-handling.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/templates.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/session-enhanced.sh"

#######################################
# Parse command line arguments
#######################################
claude_code::parse_arguments() {
    args::reset
    
    args::register_help
    args::register_yes
    
    args::register \
        --name "action" \
        --flag "a" \
        --desc "Action to perform" \
        --type "value" \
        --options "install|uninstall|status|info|run|batch|session|settings|logs|register-mcp|unregister-mcp|mcp-status|mcp-test|parse-result|extract|session-manage|health-check|run-automation|batch-automation|batch-simple|batch-config|batch-multi|batch-parallel|error-report|error-validate|safe-execute|template-list|template-load|template-run|template-create|template-info|template-validate|session-extract|session-analytics|session-recover|session-cleanup|session-list-enhanced|sandbox|help" \
        --default "help"
    
    args::register \
        --name "force" \
        --flag "f" \
        --desc "Force action even if already installed" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    args::register \
        --name "prompt" \
        --flag "p" \
        --desc "Prompt to send to Claude" \
        --type "value" \
        --default ""
    
    args::register \
        --name "max-turns" \
        --desc "Maximum number of turns for Claude" \
        --type "value" \
        --default "$DEFAULT_MAX_TURNS"
    
    args::register \
        --name "session-id" \
        --desc "Session ID to resume or manage" \
        --type "value" \
        --default ""
    
    args::register \
        --name "allowed-tools" \
        --desc "Comma-separated list of allowed tools" \
        --type "value" \
        --default ""
    
    args::register \
        --name "timeout" \
        --desc "Timeout in seconds for Claude execution" \
        --type "value" \
        --default "$DEFAULT_TIMEOUT"
    
    args::register \
        --name "output-format" \
        --desc "Output format (stream-json, text)" \
        --type "value" \
        --options "stream-json|text" \
        --default "$DEFAULT_OUTPUT_FORMAT"
    
    args::register \
        --name "dangerously-skip-permissions" \
        --desc "Skip permission checks (USE WITH EXTREME CAUTION)" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    # MCP-specific arguments
    args::register \
        --name "scope" \
        --desc "MCP registration scope (local, project, user, auto)" \
        --type "value" \
        --options "local|project|user|auto" \
        --default "$DEFAULT_MCP_SCOPE"
    
    args::register \
        --name "api-key" \
        --desc "API key for MCP authentication" \
        --type "value" \
        --default ""
    
    args::register \
        --name "server-url" \
        --desc "Vrooli server URL for MCP registration" \
        --type "value" \
        --default ""
    
    args::register \
        --name "format" \
        --desc "Output format for status commands (text, json)" \
        --type "value" \
        --options "text|json" \
        --default "text"
    
    # Automation-specific arguments
    args::register \
        --name "input-file" \
        --desc "Input file path for automation commands" \
        --type "value" \
        --default ""
    
    args::register \
        --name "extract-type" \
        --desc "Type of information to extract (status|files|summary|session|errors|turns)" \
        --type "value" \
        --options "status|files|summary|session|errors|turns" \
        --default "status"
    
    args::register \
        --name "check-type" \
        --desc "Health check type (basic|full|capabilities)" \
        --type "value" \
        --options "basic|full|capabilities" \
        --default "basic"
    
    args::register \
        --name "batch-size" \
        --desc "Batch size for batch automation" \
        --type "value" \
        --default "50"
    
    args::register \
        --name "config-file" \
        --desc "Configuration file for batch-config action" \
        --type "value" \
        --default ""
    
    args::register \
        --name "workers" \
        --desc "Number of parallel workers for batch-parallel" \
        --type "value" \
        --default "2"
    
    # Error handling arguments
    args::register \
        --name "exit-code" \
        --desc "Exit code for error reporting" \
        --type "value" \
        --default ""
    
    args::register \
        --name "error-context" \
        --desc "Context information for error reporting" \
        --type "value" \
        --default ""
    
    args::register \
        --name "max-retries" \
        --desc "Maximum number of retries for safe execution" \
        --type "value" \
        --default "3"
    
    # Template system arguments
    args::register \
        --name "template-name" \
        --desc "Name of template to use" \
        --type "value" \
        --default ""
    
    args::register \
        --name "template-vars" \
        --desc "Template variables in format 'key1=value1,key2=value2'" \
        --type "value" \
        --default ""
    
    args::register \
        --name "template-description" \
        --desc "Description for new template creation" \
        --type "value" \
        --default ""
    
    # Enhanced session management arguments
    args::register \
        --name "output-file" \
        --desc "Path to session output file for result extraction" \
        --type "value" \
        --default ""
    
    args::register \
        --name "analysis-period" \
        --desc "Time period for analytics (day|week|month|all)" \
        --type "value" \
        --options "day|week|month|all" \
        --default "week"
    
    args::register \
        --name "recovery-strategy" \
        --desc "Session recovery strategy (auto|clean|force|continue)" \
        --type "value" \
        --options "auto|clean|force|continue" \
        --default "auto"
    
    args::register \
        --name "cleanup-strategy" \
        --desc "Session cleanup strategy (old|failed|all|size)" \
        --type "value" \
        --options "old|failed|all|size" \
        --default "old"
    
    args::register \
        --name "threshold" \
        --desc "Threshold for cleanup (days for old, MB for size)" \
        --type "value" \
        --default "30"
    
    args::register \
        --name "filter" \
        --desc "Session filter (all|active|completed|error|recent)" \
        --type "value" \
        --options "all|active|completed|error|recent" \
        --default "recent"
    
    args::register \
        --name "sort-by" \
        --desc "Sort sessions by (date|turns|files|duration)" \
        --type "value" \
        --options "date|turns|files|duration" \
        --default "date"
    
    args::register \
        --name "limit" \
        --desc "Maximum number of sessions to show" \
        --type "value" \
        --default "20"
    
    # Sandbox arguments
    args::register \
        --name "sandbox-command" \
        --desc "Sandbox command (setup|build|run|interactive|exec|stop|cleanup)" \
        --type "value" \
        --options "setup|build|run|interactive|exec|stop|cleanup" \
        --default "interactive"
    
    if args::is_asking_for_help "$@"; then
        claude_code::usage
        exit 0
    fi
    
    args::parse "$@"
    
    export ACTION=$(args::get "action")
    export FORCE=$(args::get "force")
    export YES=$(args::get "yes")
    export PROMPT=$(args::get "prompt")
    export MAX_TURNS=$(args::get "max-turns")
    export SESSION_ID=$(args::get "session-id")
    export ALLOWED_TOOLS=$(args::get "allowed-tools")
    export TIMEOUT=$(args::get "timeout")
    export OUTPUT_FORMAT=$(args::get "output-format")
    export SKIP_PERMISSIONS=$(args::get "dangerously-skip-permissions")
    
    # MCP-specific variables
    export MCP_SCOPE=$(args::get "scope")
    export MCP_API_KEY=$(args::get "api-key")
    export MCP_SERVER_URL=$(args::get "server-url")
    export MCP_FORMAT=$(args::get "format")
    
    # Automation-specific variables
    export INPUT_FILE=$(args::get "input-file")
    export EXTRACT_TYPE=$(args::get "extract-type")
    export CHECK_TYPE=$(args::get "check-type")
    export BATCH_SIZE=$(args::get "batch-size")
    export CONFIG_FILE=$(args::get "config-file")
    export WORKERS=$(args::get "workers")
    
    # Error handling variables
    export EXIT_CODE=$(args::get "exit-code")
    export ERROR_CONTEXT=$(args::get "error-context")
    export MAX_RETRIES=$(args::get "max-retries")
    
    # Template system variables
    export TEMPLATE_NAME=$(args::get "template-name")
    export TEMPLATE_VARS=$(args::get "template-vars")
    export TEMPLATE_DESCRIPTION=$(args::get "template-description")
    
    # Enhanced session management variables
    export OUTPUT_FILE=$(args::get "output-file")
    export ANALYSIS_PERIOD=$(args::get "analysis-period")
    export RECOVERY_STRATEGY=$(args::get "recovery-strategy")
    export CLEANUP_STRATEGY=$(args::get "cleanup-strategy")
    export THRESHOLD=$(args::get "threshold")
    export FILTER=$(args::get "filter")
    export SORT_BY=$(args::get "sort-by")
    export LIMIT=$(args::get "limit")
    
    # Sandbox variables
    export SANDBOX_COMMAND=$(args::get "sandbox-command")
}

#######################################
# Main execution function
#######################################
claude_code::main() {
    claude_code::parse_arguments "$@"
    
    case "$ACTION" in
        "install")
            claude_code::install
            ;;
        "uninstall")
            claude_code::uninstall
            ;;
        "status")
            claude_code::status
            ;;
        "info")
            claude_code::info
            ;;
        "run")
            claude_code::run
            ;;
        "batch")
            claude_code::batch
            ;;
        "session")
            claude_code::session
            ;;
        "settings")
            claude_code::settings
            ;;
        "logs")
            claude_code::logs
            ;;
        "register-mcp")
            claude_code::register_mcp
            ;;
        "unregister-mcp")
            claude_code::unregister_mcp
            ;;
        "mcp-status")
            claude_code::mcp_status
            ;;
        "mcp-test")
            claude_code::mcp_test
            ;;
        "parse-result")
            claude_code::parse_result "$INPUT_FILE" "$MCP_FORMAT"
            ;;
        "extract")
            claude_code::extract "$INPUT_FILE" "$EXTRACT_TYPE"
            ;;
        "session-manage")
            claude_code::session_manage "$EXTRACT_TYPE" "$SESSION_ID" "$MCP_FORMAT"
            ;;
        "health-check")
            claude_code::health_check "$CHECK_TYPE" "$MCP_FORMAT"
            ;;
        "run-automation")
            local output_file
            output_file=$(claude_code::run_automation "$PROMPT" "$ALLOWED_TOOLS" "$MAX_TURNS" "$INPUT_FILE" 2>&1 | tail -1)
            claude_code::parse_result "$output_file" "$MCP_FORMAT"
            ;;
        "batch-automation")
            claude_code::batch_automation "$PROMPT" "$MAX_TURNS" "$BATCH_SIZE"
            ;;
        "batch-simple")
            claude_code::batch_simple "$PROMPT" "$MAX_TURNS" "$BATCH_SIZE" "$ALLOWED_TOOLS"
            ;;
        "batch-config")
            if [[ -z "$CONFIG_FILE" ]]; then
                log::error "Configuration file required for batch-config action"
                log::info "Use --config-file path/to/config.json"
                exit 1
            fi
            claude_code::batch_config "$CONFIG_FILE"
            ;;
        "batch-multi")
            # For batch-multi, we need multiple prompt/turn pairs
            # This is a simplified implementation - real usage would pass multiple args
            if [[ -z "$PROMPT" ]]; then
                log::error "At least one prompt required for batch-multi"
                exit 1
            fi
            claude_code::batch_multi "$PROMPT" "$MAX_TURNS"
            ;;
        "batch-parallel")
            if [[ -z "$PROMPT" ]]; then
                log::error "At least one prompt required for batch-parallel"
                exit 1
            fi
            claude_code::batch_parallel "$WORKERS" "$PROMPT" "$MAX_TURNS"
            ;;
        "error-report")
            if [[ -z "$EXIT_CODE" ]]; then
                log::error "Exit code required for error-report action"
                log::info "Use --exit-code <code>"
                exit 1
            fi
            claude_code::error_report "${ERROR_CONTEXT:-claude_code}" "$EXIT_CODE"
            ;;
        "error-validate")
            claude_code::error_validate_config
            ;;
        "sandbox")
            # Run claude-code in sandboxed environment
            export SANDBOX_COMMAND="${SANDBOX_COMMAND:-interactive}"
            "${SCRIPT_DIR}/sandbox/claude-sandbox.sh" "${SANDBOX_COMMAND}" "${PROMPT:-}"
            ;;
        "safe-execute")
            if [[ -z "$PROMPT" ]]; then
                log::error "Prompt required for safe-execute action"
                exit 1
            fi
            claude_code::safe_execute "$PROMPT" "$ALLOWED_TOOLS" "$MAX_TURNS" "$MAX_RETRIES"
            ;;
        "template-list")
            claude_code::templates_list "${MCP_FORMAT:-text}"
            ;;
        "template-load")
            if [[ -z "$TEMPLATE_NAME" ]]; then
                log::error "Template name required for template-load action"
                log::info "Use --template-name <name>"
                exit 1
            fi
            # Parse template variables from comma-separated format
            local template_args=()
            if [[ -n "$TEMPLATE_VARS" ]]; then
                IFS=',' read -ra var_pairs <<< "$TEMPLATE_VARS"
                for pair in "${var_pairs[@]}"; do
                    template_args+=("$pair")
                done
            fi
            claude_code::template_load "$TEMPLATE_NAME" "${template_args[@]}"
            ;;
        "template-run")
            if [[ -z "$TEMPLATE_NAME" ]]; then
                log::error "Template name required for template-run action"
                log::info "Use --template-name <name>"
                exit 1
            fi
            # Parse template variables from comma-separated format
            local template_args=()
            if [[ -n "$TEMPLATE_VARS" ]]; then
                IFS=',' read -ra var_pairs <<< "$TEMPLATE_VARS"
                for pair in "${var_pairs[@]}"; do
                    template_args+=("$pair")
                done
            fi
            claude_code::template_run "$TEMPLATE_NAME" "$MAX_TURNS" "$ALLOWED_TOOLS" "${template_args[@]}"
            ;;
        "template-create")
            if [[ -z "$TEMPLATE_NAME" ]]; then
                log::error "Template name required for template-create action"
                log::info "Use --template-name <name>"
                exit 1
            fi
            claude_code::template_create "$TEMPLATE_NAME" "${TEMPLATE_DESCRIPTION:-Custom template}"
            ;;
        "template-info")
            if [[ -z "$TEMPLATE_NAME" ]]; then
                log::error "Template name required for template-info action"
                log::info "Use --template-name <name>"
                exit 1
            fi
            claude_code::template_info "$TEMPLATE_NAME"
            ;;
        "template-validate")
            if [[ -z "$TEMPLATE_NAME" ]]; then
                log::error "Template name required for template-validate action"
                log::info "Use --template-name <name>"
                exit 1
            fi
            claude_code::template_validate "$TEMPLATE_NAME"
            ;;
        "session-extract")
            if [[ -z "$OUTPUT_FILE" ]]; then
                log::error "Output file required for session-extract action"
                log::info "Use --output-file <path>"
                exit 1
            fi
            claude_code::session_extract_results "$OUTPUT_FILE" "$SESSION_ID" "${MCP_FORMAT:-json}"
            ;;
        "session-analytics")
            claude_code::session_analytics "$ANALYSIS_PERIOD" "${MCP_FORMAT:-text}"
            ;;
        "session-recover")
            if [[ -z "$SESSION_ID" ]]; then
                log::error "Session ID required for session-recover action"
                log::info "Use --session-id <id>"
                exit 1
            fi
            claude_code::session_recover "$SESSION_ID" "$RECOVERY_STRATEGY" "$PROMPT"
            ;;
        "session-cleanup")
            claude_code::session_cleanup "$CLEANUP_STRATEGY" "$THRESHOLD"
            ;;
        "session-list-enhanced")
            claude_code::session_list_enhanced "$FILTER" "$SORT_BY" "${MCP_FORMAT:-text}" "$LIMIT"
            ;;
        "help"|*)
            claude_code::usage
            ;;
    esac
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    claude_code::main "$@"
fi