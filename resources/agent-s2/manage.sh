#!/usr/bin/env bash
set -euo pipefail

# Handle Ctrl+C gracefully
trap 'echo ""; log::info "Agent S2 installation interrupted by user. Exiting..."; exit 130' INT TERM

# Agent S2 Autonomous Computer Interaction Setup and Management
# This script handles installation, configuration, and management of Agent S2 using Docker

DESCRIPTION="Install and manage Agent S2 autonomous computer interaction service using Docker"

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
AGENT_S2_SCRIPT_DIR="${APP_ROOT}/resources/agent-s2"

# Source var.sh first to get proper directory variables
# shellcheck disable=SC1091
source "${AGENT_S2_SCRIPT_DIR}/../../../lib/utils/var.sh"

# Source common resources using var_ variables
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_DIR}/common.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/args-cli.sh"

# Source configuration
# shellcheck disable=SC1091
source "${AGENT_S2_SCRIPT_DIR}/config/defaults.sh"
# shellcheck disable=SC1091
source "${AGENT_S2_SCRIPT_DIR}/config/messages.sh"

# Export messages (config will be exported after parsing arguments)
agents2::export_messages

# Source all library modules
# shellcheck disable=SC1091
source "${AGENT_S2_SCRIPT_DIR}/lib/common.sh"
# shellcheck disable=SC1091
source "${AGENT_S2_SCRIPT_DIR}/lib/docker.sh"
# shellcheck disable=SC1091
source "${AGENT_S2_SCRIPT_DIR}/lib/status.sh"
# shellcheck disable=SC1091
source "${AGENT_S2_SCRIPT_DIR}/lib/install.sh"
# shellcheck disable=SC1091
source "${AGENT_S2_SCRIPT_DIR}/lib/api.sh"
# shellcheck disable=SC1091
source "${AGENT_S2_SCRIPT_DIR}/lib/usage.sh"
# shellcheck disable=SC1091
source "${AGENT_S2_SCRIPT_DIR}/lib/modes.sh"
# shellcheck disable=SC1091
source "${AGENT_S2_SCRIPT_DIR}/lib/stealth.sh"

#######################################
# Parse command line arguments
#######################################
agents2::parse_arguments() {
    args::reset
    
    args::register_help
    args::register_yes
    
    args::register \
        --name "action" \
        --flag "a" \
        --desc "Action to perform" \
        --type "value" \
        --options "install|uninstall|start|stop|restart|status|logs|info|usage|mode|switch-mode|test-mode|reset-session-data|reset-session-state|list-sessions|export-session|import-session|configure-stealth|test-stealth|ai-task" \
        --default "install"
    
    args::register \
        --name "force" \
        --flag "f" \
        --desc "Force action even if Agent S2 appears to be already installed/running" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    args::register \
        --name "llm-provider" \
        --desc "LLM provider for Agent S2 (ollama, openai, anthropic)" \
        --type "value" \
        --options "ollama|openai|anthropic" \
        --default "ollama"
    
    args::register \
        --name "llm-model" \
        --desc "LLM model to use" \
        --type "value" \
        --default "llama3.2-vision:11b"
    
    args::register \
        --name "enable-ai" \
        --desc "Enable AI capabilities (vs core automation only)" \
        --type "value" \
        --options "yes|no" \
        --default "yes"
    
    args::register \
        --name "enable-search" \
        --desc "Enable web search integration (requires Perplexica)" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    args::register \
        --name "vnc-password" \
        --desc "Password for VNC access" \
        --type "value" \
        --default "agents2vnc"
    
    args::register \
        --name "enable-host-display" \
        --desc "Enable access to host display (security risk)" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    args::register \
        --name "mode" \
        --desc "Operating mode (sandbox or host)" \
        --type "value" \
        --options "sandbox|host" \
        --default "sandbox"
    
    args::register \
        --name "target-mode" \
        --desc "Target mode for mode switching" \
        --type "value" \
        --options "sandbox|host"
    
    args::register \
        --name "usage-type" \
        --desc "Type of usage example to run" \
        --type "value" \
        --options "screenshot|automation|planning|capabilities|all|help" \
        --default "help"
    
    args::register \
        --name "profile" \
        --desc "Session profile ID for session actions" \
        --type "value" \
        --default ""
    
    args::register \
        --name "output" \
        --desc "Output path for export-session action" \
        --type "value" \
        --default ""
    
    args::register \
        --name "input" \
        --desc "Input path for import-session action" \
        --type "value" \
        --default ""
    
    args::register \
        --name "stealth-enabled" \
        --desc "Enable/disable stealth mode" \
        --type "value" \
        --options "yes|no" \
        --default ""
    
    args::register \
        --name "stealth-feature" \
        --desc "Specific stealth feature to configure" \
        --type "value" \
        --default ""
    
    args::register \
        --name "stealth-url" \
        --desc "URL for stealth mode testing" \
        --type "value" \
        --default "https://bot.sannysoft.com/"
    
    args::register \
        --name "task" \
        --desc "Task description for AI task execution" \
        --type "value" \
        --default ""
    
    args::register \
        --name "allowed-domains" \
        --desc "Comma-separated list of allowed domains for URL navigation" \
        --type "value" \
        --default ""
    
    args::register \
        --name "blocked-domains" \
        --desc "Comma-separated list of blocked domains" \
        --type "value" \
        --default ""
    
    args::register \
        --name "security-profile" \
        --desc "Security profile for URL validation" \
        --type "value" \
        --options "strict|moderate|permissive" \
        --default "moderate"
    
    if args::is_asking_for_help "$@"; then
        agents2::usage
        exit 0
    fi
    
    args::parse "$@"
    
    ACTION=$(args::get "action")
    FORCE=$(args::get "force")
    YES=$(args::get "yes")
    LLM_PROVIDER=$(args::get "llm-provider")
    LLM_MODEL=$(args::get "llm-model")
    ARGS_LLM_PROVIDER=$(args::get "llm-provider")
    ARGS_LLM_MODEL=$(args::get "llm-model")
    VNC_PASSWORD=$(args::get "vnc-password")
    ENABLE_HOST_DISPLAY=$(args::get "enable-host-display")
    ENABLE_AI=$(args::get "enable-ai")
    ENABLE_SEARCH=$(args::get "enable-search")
    ARGS_ENABLE_AI=$(args::get "enable-ai")
    USAGE_TYPE=$(args::get "usage-type")
    MODE=$(args::get "mode")
    TARGET_MODE=$(args::get "target-mode")
    PROFILE_ID=$(args::get "profile")
    OUTPUT_PATH=$(args::get "output")
    INPUT_PATH=$(args::get "input")
    STEALTH_ENABLED=$(args::get "stealth-enabled")
    STEALTH_FEATURE=$(args::get "stealth-feature")
    STEALTH_TEST_URL=$(args::get "stealth-url")
    AI_TASK=$(args::get "task")
    ALLOWED_DOMAINS=$(args::get "allowed-domains")
    BLOCKED_DOMAINS=$(args::get "blocked-domains")
    SECURITY_PROFILE=$(args::get "security-profile")
    export ACTION FORCE YES LLM_PROVIDER LLM_MODEL ARGS_LLM_PROVIDER ARGS_LLM_MODEL VNC_PASSWORD ENABLE_HOST_DISPLAY ENABLE_AI ENABLE_SEARCH ARGS_ENABLE_AI USAGE_TYPE MODE TARGET_MODE PROFILE_ID OUTPUT_PATH INPUT_PATH STEALTH_ENABLED STEALTH_FEATURE STEALTH_TEST_URL AI_TASK ALLOWED_DOMAINS BLOCKED_DOMAINS SECURITY_PROFILE
}

#######################################
# Display usage information
#######################################
agents2::usage() {
    args::usage "$DESCRIPTION"
    echo
    echo "Examples:"
    echo "  $0 --action install                              # Install Agent S2 with default settings"
    echo "  $0 --action install --llm-provider ollama        # Install with Ollama provider (default)"
    echo "  $0 --action install --mode host                  # Install with host mode enabled"
    echo "  $0 --action start --mode sandbox                 # Start in sandbox mode"
    echo "  $0 --action start --mode host                    # Start in host mode"
    echo "  $0 --action switch-mode --target-mode host       # Switch to host mode"
    echo "  $0 --action mode                                 # Show current mode information"
    echo "  $0 --action test-mode                            # Test current mode functionality"
    echo "  $0 --action status                               # Check Agent S2 status"
    echo "  $0 --action logs                                 # View Agent S2 logs"
    echo "  $0 --action usage                                # Show usage examples"
    echo "  $0 --action usage --usage-type screenshot        # Test screenshot API"
    echo "  $0 --action usage --usage-type automation        # Test automation capabilities"
    echo "  $0 --action ai-task --task \"go to reddit\"       # Execute AI task"
    echo "  $0 --action ai-task --task \"search for news\"    # Another AI task example"
    echo "  $0 --action uninstall                           # Remove Agent S2"
    echo
    echo "Security Examples:"
    echo "  $0 --action ai-task --task \"browse news\" --allowed-domains \"reddit.com,news.ycombinator.com\""
    echo "  $0 --action ai-task --task \"research\" --blocked-domains \"facebook.com,twitter.com\""
    echo "  $0 --action ai-task --task \"safe browse\" --security-profile strict"
    echo
    echo "Stealth Mode Examples:"
    echo "  $0 --action configure-stealth --stealth-enabled yes      # Enable stealth mode"
    echo "  $0 --action test-stealth                                # Test stealth effectiveness"
    echo "  $0 --action list-sessions                               # List saved session profiles"
    echo "  $0 --action reset-session-data --profile myprofile      # Reset specific session data"
    echo "  $0 --action export-session --profile myprofile --output session.json"
    echo "  $0 --action import-session --profile newprofile --input session.json"
    echo
    echo "Access Points:"
    echo "  API: http://localhost:4113"
    echo "  VNC: vnc://localhost:5900"
    echo
    echo "Security Notes:"
    echo "  - Agent S2 runs in a secure Docker container with isolated display"
    echo "  - VNC access is password-protected"
    echo "  - Host display access is disabled by default for security"
}

#######################################
# Main execution function
#######################################
agents2::main() {
    agents2::parse_arguments "$@"
    
    # Export configuration after parsing arguments
    agents2::export_config
    
    case "$ACTION" in
        "install")
            agents2::install_service
            ;;
        "uninstall")
            agents2::uninstall_service
            ;;
        "start")
            if [[ -n "$MODE" ]]; then
                agents2::start_in_mode "$MODE"
            else
                agents2::docker_start
            fi
            ;;
        "stop")
            agents2::docker_stop
            ;;
        "restart")
            if [[ -n "$MODE" ]]; then
                agents2::switch_mode "$MODE" true
            else
                agents2::docker_restart
            fi
            ;;
        "status")
            agents2::show_status
            ;;
        "logs")
            agents2::docker_logs
            ;;
        "info")
            agents2::show_info
            ;;
        "usage")
            agents2::run_usage_example "$USAGE_TYPE"
            ;;
        "mode")
            agents2::show_mode_info
            ;;
        "switch-mode")
            if [[ -z "$TARGET_MODE" ]]; then
                log::error "Target mode not specified. Use --target-mode sandbox|host"
                exit 1
            fi
            agents2::switch_mode "$TARGET_MODE" "$FORCE"
            ;;
        "test-mode")
            agents2::test_mode "$MODE"
            ;;
        "reset-session-data")
            agents2::reset_session_data "$PROFILE_ID"
            ;;
        "reset-session-state")
            agents2::reset_session_state
            ;;
        "list-sessions")
            agents2::list_sessions
            ;;
        "export-session")
            agents2::export_session "$PROFILE_ID" "$OUTPUT_PATH"
            ;;
        "import-session")
            agents2::import_session "$PROFILE_ID" "$INPUT_PATH"
            ;;
        "configure-stealth")
            agents2::configure_stealth "$STEALTH_ENABLED" "$STEALTH_FEATURE"
            ;;
        "test-stealth")
            agents2::test_stealth "$STEALTH_TEST_URL"
            ;;
        "ai-task")
            if [[ -z "$AI_TASK" ]]; then
                log::error "Task description is required. Use --task \"your task description\""
                exit 1
            fi
            agents2::execute_ai_task "$AI_TASK"
            ;;
        *)
            log::error "Unknown action: $ACTION"
            agents2::usage
            exit 1
            ;;
    esac
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    agents2::main "$@"
fi