#!/bin/bash
# VirusTotal Resource CLI Interface
# Implements v2.0 universal contract for threat intelligence operations

set -euo pipefail

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_NAME="virustotal"

# Source the core library
source "${SCRIPT_DIR}/lib/core.sh"

# Main CLI entrypoint
main() {
    local command="${1:-}"
    shift || true
    
    case "${command}" in
        help)
            show_help
            ;;
        info)
            show_info "$@"
            ;;
        manage)
            handle_manage "$@"
            ;;
        test)
            handle_test "$@"
            ;;
        content)
            handle_content "$@"
            ;;
        status)
            show_status "$@"
            ;;
        logs)
            show_logs "$@"
            ;;
        credentials)
            show_credentials "$@"
            ;;
        *)
            echo "Error: Unknown command '${command}'"
            show_help
            exit 1
            ;;
    esac
}

# Show help information
show_help() {
    cat << EOF
VirusTotal Resource - Multi-engine threat intelligence and malware scanning

Usage: resource-${RESOURCE_NAME} <command> [options]

Commands:
  help                Show this help message
  info               Show resource configuration and runtime information
  manage            Lifecycle management (install/start/stop/restart/uninstall)
  test              Run validation tests (smoke/integration/unit/all)
  content           Manage scan operations (add/list/get/remove/execute)
  status            Show service status and API quota
  logs              View service logs
  credentials       Display API configuration

Examples:
  # Install and start the service
  resource-${RESOURCE_NAME} manage install
  resource-${RESOURCE_NAME} manage start

  # Submit a file for scanning
  resource-${RESOURCE_NAME} content add --file /path/to/file.exe

  # Get scan results
  resource-${RESOURCE_NAME} content get --hash SHA256_HASH

  # Check service health
  resource-${RESOURCE_NAME} test smoke

Configuration:
  API Key: Set VIRUSTOTAL_API_KEY environment variable
  Port: ${VIRUSTOTAL_PORT:-8290} (configurable via VIRUSTOTAL_PORT)
  
For more information, see README.md
EOF
}

# Run main function
main "$@"