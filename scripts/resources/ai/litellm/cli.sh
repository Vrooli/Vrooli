#!/bin/bash
# LiteLLM CLI interface

# Get the real script directory (resolving symlinks)
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    LITELLM_CLI_DIR="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
else
    LITELLM_CLI_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
fi

# Source dependencies
source "${LITELLM_CLI_DIR}/lib/core.sh"
source "${LITELLM_CLI_DIR}/lib/status.sh"
source "${LITELLM_CLI_DIR}/lib/install.sh"
source "${LITELLM_CLI_DIR}/lib/docker.sh"
source "${LITELLM_CLI_DIR}/lib/content.sh"

# Main CLI handler
litellm::cli() {
    local command="${1:-help}"
    shift
    
    case "$command" in
        status)
            # Pass all arguments directly to status function
            litellm::status "$@"
            ;;
        status-detailed)
            litellm::status::detailed "$@"
            ;;
        install)
            local verbose="false"
            local force="false"
            
            # Parse install arguments
            while [[ $# -gt 0 ]]; do
                case "$1" in
                    --verbose|-v)
                        verbose="true"
                        shift
                        ;;
                    --force|-f)
                        force="true"
                        shift
                        ;;
                    *)
                        shift
                        ;;
                esac
            done
            
            litellm::install "$verbose" "$force"
            ;;
        uninstall)
            local verbose="false"
            local keep_data="false"
            local force="false"
            
            # Parse uninstall arguments
            while [[ $# -gt 0 ]]; do
                case "$1" in
                    --verbose|-v)
                        verbose="true"
                        shift
                        ;;
                    --keep-data)
                        keep_data="true"
                        shift
                        ;;
                    --force|-f)
                        force="true"
                        shift
                        ;;
                    *)
                        shift
                        ;;
                esac
            done
            
            litellm::uninstall "$verbose" "$keep_data" "$force"
            ;;
        start)
            local verbose="false"
            local wait="true"
            
            # Parse start arguments
            while [[ $# -gt 0 ]]; do
                case "$1" in
                    --verbose|-v)
                        verbose="true"
                        shift
                        ;;
                    --no-wait)
                        wait="false"
                        shift
                        ;;
                    *)
                        shift
                        ;;
                esac
            done
            
            litellm::start "$verbose" "$wait"
            ;;
        stop)
            local verbose="false"
            local force="false"
            
            # Parse stop arguments
            while [[ $# -gt 0 ]]; do
                case "$1" in
                    --verbose|-v)
                        verbose="true"
                        shift
                        ;;
                    --force|-f)
                        force="true"
                        shift
                        ;;
                    *)
                        shift
                        ;;
                esac
            done
            
            litellm::stop "$verbose" "$force"
            ;;
        restart)
            local verbose="false"
            
            # Parse restart arguments
            while [[ $# -gt 0 ]]; do
                case "$1" in
                    --verbose|-v)
                        verbose="true"
                        shift
                        ;;
                    *)
                        shift
                        ;;
                esac
            done
            
            litellm::restart "$verbose"
            ;;
        test|test-connection)
            local verbose="false"
            local timeout="$LITELLM_HEALTH_CHECK_TIMEOUT"
            
            # Parse test arguments
            while [[ $# -gt 0 ]]; do
                case "$1" in
                    --verbose|-v)
                        verbose="true"
                        shift
                        ;;
                    --timeout|-t)
                        timeout="$2"
                        shift 2
                        ;;
                    *)
                        shift
                        ;;
                esac
            done
            
            if litellm::test_connection "$timeout" "$verbose"; then
                echo "✓ LiteLLM proxy is accessible"
                return 0
            else
                echo "✗ Failed to connect to LiteLLM proxy"
                return 1
            fi
            ;;
        test-model)
            local model="${1:-$LITELLM_DEFAULT_MODEL}"
            local verbose="false"
            local timeout="$LITELLM_TIMEOUT"
            shift
            
            # Parse test-model arguments
            while [[ $# -gt 0 ]]; do
                case "$1" in
                    --verbose|-v)
                        verbose="true"
                        shift
                        ;;
                    --timeout|-t)
                        timeout="$2"
                        shift 2
                        ;;
                    *)
                        shift
                        ;;
                esac
            done
            
            if litellm::test_model "$model" "$timeout" "$verbose"; then
                echo "✓ Model '$model' is working"
                return 0
            else
                echo "✗ Model '$model' test failed"
                return 1
            fi
            ;;
        list-models)
            local verbose="false"
            local timeout="$LITELLM_TIMEOUT"
            
            # Parse list-models arguments
            while [[ $# -gt 0 ]]; do
                case "$1" in
                    --verbose|-v)
                        verbose="true"
                        shift
                        ;;
                    --timeout|-t)
                        timeout="$2"
                        shift 2
                        ;;
                    *)
                        shift
                        ;;
                esac
            done
            
            litellm::list_models "$timeout" "$verbose"
            ;;
        proxy-status)
            local timeout="$LITELLM_TIMEOUT"
            
            # Parse proxy-status arguments
            while [[ $# -gt 0 ]]; do
                case "$1" in
                    --timeout|-t)
                        timeout="$2"
                        shift 2
                        ;;
                    *)
                        shift
                        ;;
                esac
            done
            
            litellm::get_status "$timeout" | jq '.' 2>/dev/null || litellm::get_status "$timeout"
            ;;
        logs)
            local lines="${1:-50}"
            local follow="false"
            shift
            
            # Parse logs arguments
            while [[ $# -gt 0 ]]; do
                case "$1" in
                    --follow|-f)
                        follow="true"
                        shift
                        ;;
                    --lines|-n)
                        lines="$2"
                        shift 2
                        ;;
                    *)
                        shift
                        ;;
                esac
            done
            
            litellm::logs "$lines" "$follow"
            ;;
        exec)
            litellm::exec "$*"
            ;;
        stats)
            litellm::stats
            ;;
        inspect)
            litellm::inspect
            ;;
        update)
            local verbose="false"
            
            # Parse update arguments
            while [[ $# -gt 0 ]]; do
                case "$1" in
                    --verbose|-v)
                        verbose="true"
                        shift
                        ;;
                    *)
                        shift
                        ;;
                esac
            done
            
            litellm::update "$verbose"
            ;;
        upgrade)
            local verbose="false"
            
            # Parse upgrade arguments
            while [[ $# -gt 0 ]]; do
                case "$1" in
                    --verbose|-v)
                        verbose="true"
                        shift
                        ;;
                    *)
                        shift
                        ;;
                esac
            done
            
            litellm::upgrade "$verbose"
            ;;
        backup)
            local backup_dir=""
            local verbose="false"
            
            # Parse backup arguments
            while [[ $# -gt 0 ]]; do
                case "$1" in
                    --dir|-d)
                        backup_dir="$2"
                        shift 2
                        ;;
                    --verbose|-v)
                        verbose="true"
                        shift
                        ;;
                    *)
                        shift
                        ;;
                esac
            done
            
            litellm::backup "$backup_dir" "$verbose"
            ;;
        reset)
            local verbose="false"
            local keep_data="true"
            
            # Parse reset arguments
            while [[ $# -gt 0 ]]; do
                case "$1" in
                    --verbose|-v)
                        verbose="true"
                        shift
                        ;;
                    --purge-data)
                        keep_data="false"
                        shift
                        ;;
                    *)
                        shift
                        ;;
                esac
            done
            
            litellm::reset "$verbose" "$keep_data"
            ;;
        validate)
            local verbose="false"
            
            # Parse validate arguments
            while [[ $# -gt 0 ]]; do
                case "$1" in
                    --verbose|-v)
                        verbose="true"
                        shift
                        ;;
                    *)
                        shift
                        ;;
                esac
            done
            
            litellm::validate "$verbose"
            ;;
        content)
            # Handle content management subcommands
            local subcommand="${1:-help}"
            shift
            
            case "$subcommand" in
                add)
                    litellm::content::add "$@"
                    ;;
                list)
                    litellm::content::list "$@"
                    ;;
                get)
                    litellm::content::get "$@"
                    ;;
                remove)
                    litellm::content::remove "$@"
                    ;;
                execute)
                    litellm::content::execute "$@"
                    ;;
                help)
                    cat <<EOF
LiteLLM Content Management

Usage: $0 content <subcommand> [options]

Subcommands:
  add --type <type> [--file <file>] [--name <name>] [--data <data>]
                                Add content (config, provider, example)
  list [--type <type>] [--format json|text]
                                List content
  get --type <type> --name <name>
                                Get content by name
  remove --type <type> --name <name>
                                Remove content by name
  execute --type <type> --name <name>
                                Apply/execute content

Content Types:
  config      Configuration files (YAML)
  provider    Provider configurations (JSON)
  example     Example files

Examples:
  $0 content add --type config --file my-config.yaml --name custom
  $0 content list --type provider
  $0 content execute --type config --name custom
EOF
                    ;;
                *)
                    echo "Unknown content subcommand: $subcommand"
                    echo "Run '$0 content help' for usage information"
                    return 1
                    ;;
            esac
            ;;
        run-tests)
            # Run integration tests and save results
            local test_dir="${LITELLM_CLI_DIR}/test"
            local result_dir="${var_ROOT_DIR}/data/test-results"
            mkdir -p "$result_dir"
            
            if [[ -f "${test_dir}/integration.bats" ]]; then
                echo "Running LiteLLM integration tests..."
                local timestamp=$(date -Iseconds)
                local test_output
                test_output=$(timeout 60 bats "${test_dir}/integration.bats" 2>&1)
                local exit_code=$?
                
                # Parse test results
                local test_status="failed"
                if [[ $exit_code -eq 0 ]]; then
                    test_status="passed"
                fi
                
                # Save results
                cat > "${result_dir}/litellm-test.json" <<EOF
{
    "resource": "litellm",
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
                echo "No tests found for LiteLLM"
                return 1
            fi
            ;;
        help|--help|-h)
            cat <<EOF
LiteLLM Resource CLI

Usage: $0 <command> [options]

Core Commands:
  status [--verbose] [--format json|text]  Check LiteLLM status
  status-detailed                          Show detailed status information
  install [--verbose] [--force]           Install LiteLLM proxy server
  uninstall [--verbose] [--keep-data] [--force] Remove LiteLLM
  start [--verbose] [--no-wait]           Start LiteLLM proxy server
  stop [--verbose] [--force]              Stop LiteLLM proxy server
  restart [--verbose]                     Restart LiteLLM proxy server

Testing Commands:
  test [--verbose] [--timeout N]          Test proxy connectivity
  test-model [model] [--verbose] [--timeout N] Test specific model
  list-models [--verbose] [--timeout N]   List available models
  proxy-status [--timeout N]              Get proxy status and metrics
  run-tests                               Run integration tests

Management Commands:
  logs [lines] [--follow]                 Show container logs
  exec <command>                          Execute command in container
  stats                                   Show resource usage statistics
  inspect                                 Inspect container configuration
  update [--verbose]                      Update LiteLLM image
  upgrade [--verbose]                     Upgrade with backup
  backup [--dir path] [--verbose]         Create backup
  reset [--verbose] [--purge-data]        Reset to defaults
  validate [--verbose]                    Validate installation

Content Management:
  content add --type <type> [options]     Add content (config/provider/example)
  content list [--type <type>]            List content
  content get --type <type> --name <name> Get content
  content remove --type <type> --name <name> Remove content
  content execute --type <type> --name <name> Apply/execute content

Configuration:
  Config File: ${LITELLM_CONFIG_FILE}
  Environment: ${LITELLM_ENV_FILE}
  Data Directory: ${LITELLM_DATA_DIR}
  Service URL: ${LITELLM_API_BASE}

Examples:
  $0 install --verbose
  $0 status --format json
  $0 test-model gpt-3.5-turbo
  $0 content add --type config --file config.yaml
  $0 logs --follow

For content management help: $0 content help
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
    litellm::cli "$@"
fi