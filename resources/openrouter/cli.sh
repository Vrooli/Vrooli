#!/bin/bash
# OpenRouter CLI interface

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    OPENROUTER_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    # Recalculate APP_ROOT from resolved symlink location
    APP_ROOT="$(builtin cd "${OPENROUTER_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
OPENROUTER_CLI_DIR="${APP_ROOT}/resources/openrouter"

# Source dependencies
source "${OPENROUTER_CLI_DIR}/lib/core.sh"
source "${OPENROUTER_CLI_DIR}/lib/status.sh"
source "${OPENROUTER_CLI_DIR}/lib/install.sh"
source "${OPENROUTER_CLI_DIR}/lib/inject.sh"
source "${OPENROUTER_CLI_DIR}/lib/configure.sh"
source "${OPENROUTER_CLI_DIR}/lib/content.sh"

# Main CLI handler
openrouter::cli() {
    local command="${1:-help}"
    shift
    
    case "$command" in
        status)
            # Pass all arguments directly to status function
            openrouter::status "$@"
            ;;
        install)
            openrouter::install "$@"
            ;;
        uninstall)
            openrouter::uninstall "$@"
            ;;
        start)
            # OpenRouter is an API service, always "running"
            echo "OpenRouter is an API service (no start needed)"
            return 0
            ;;
        stop)
            # OpenRouter is an API service, can't be stopped
            echo "OpenRouter is an API service (cannot be stopped)"
            return 0
            ;;
        test|test-connection)
            if openrouter::test_connection; then
                echo "✓ OpenRouter API is accessible"
                return 0
            else
                echo "✗ Failed to connect to OpenRouter API"
                return 1
            fi
            ;;
        list-models)
            openrouter::list_models
            ;;
        usage|credits)
            openrouter::get_usage | jq '.' 2>/dev/null || openrouter::get_usage
            ;;
        inject)
            openrouter::inject "$@"
            ;;
        configure)
            openrouter::configure "$@"
            ;;
        show-config)
            openrouter::show_config
            ;;
        content)
            openrouter::content "$@"
            ;;
        run-tests)
            # Run integration tests and save results
            local test_dir="${OPENROUTER_CLI_DIR}/test"
            local result_dir="${var_ROOT_DIR}/data/test-results"
            mkdir -p "$result_dir"
            
            if [[ -f "${test_dir}/integration.bats" ]]; then
                echo "Running OpenRouter integration tests..."
                local timestamp=$(date -Iseconds)
                local test_output
                test_output=$(timeout 30 bats "${test_dir}/integration.bats" 2>&1)
                local exit_code=$?
                
                # Parse test results
                local test_status="failed"
                if [[ $exit_code -eq 0 ]]; then
                    test_status="passed"
                fi
                
                # Save results
                cat > "${result_dir}/openrouter-test.json" <<EOF
{
    "resource": "openrouter",
    "status": "$test_status",
    "timestamp": "$timestamp",
    "exit_code": $exit_code,
    "tests_run": $(echo "$test_output" | grep -E '^1\.\.[0-9]+$' | cut -d. -f3 || echo "0"),
    "output": $(echo "$test_output" | jq -Rs . 2>/dev/null || echo '""')
}
EOF
                echo "$test_output"
                return $exit_code
            else
                echo "No tests found for OpenRouter"
                return 1
            fi
            ;;
        help|--help|-h)
            cat <<EOF
OpenRouter Resource CLI

Usage: $0 <command> [options]

Commands:
  status [--verbose] [--format json|text]  Check OpenRouter status
  install [--verbose]                      Install/configure OpenRouter
  uninstall [--verbose]                    Remove OpenRouter configuration
  configure [--api-key KEY] [--vault|--file] Set up API key
  show-config                               Show current configuration
  start                                     No-op (API service)
  stop                                      No-op (API service)
  test|test-connection                     Test API connectivity
  list-models                               List available models
  usage|credits                             Show usage and credits
  inject <target> [data]                   Inject config to other resources
  content <add|list|get|remove|execute>    Manage prompts and configurations
  run-tests                                 Run integration tests
  help                                      Show this help message

Examples:
  $0 status
  $0 install --verbose
  $0 list-models
  $0 content add --file prompt.txt --type prompt --name my-prompt
  $0 inject n8n
EOF
            ;;
        *)
            echo "Unknown command: $command"
            echo "Run '$0 help' for usage information"
            return 1
            ;;
    esac
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    set +e  # Disable strict error handling for CLI
    openrouter::cli "$@"
fi
