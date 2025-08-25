#!/bin/bash

# LlamaIndex CLI Interface
# Direct wrapper for LlamaIndex resource management

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    LLAMAINDEX_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    # Recalculate APP_ROOT from resolved symlink location
    APP_ROOT="$(builtin cd "${LLAMAINDEX_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
LLAMAINDEX_CLI_DIR="${APP_ROOT}/resources/llamaindex"
LLAMAINDEX_LIB_DIR="$LLAMAINDEX_CLI_DIR/lib"

# Source core functions
source "$LLAMAINDEX_LIB_DIR/core.sh" || exit 1

# Main CLI handler
main() {
    local command="${1:-status}"
    shift || true
    
    case "$command" in
        status)
            # Handle --format and --fast flags
            local format="plain"
            local fast_mode="false"
            while [[ $# -gt 0 ]]; do
                case "$1" in
                    --format)
                        format="${2:-plain}"
                        shift 2 || shift
                        ;;
                    --format=*)
                        format="${1#*=}"
                        shift
                        ;;
                    --fast)
                        fast_mode="true"
                        shift
                        ;;
                    *)
                        # Only treat as format if it's not a flag
                        if [[ "$1" != --* ]]; then
                            format="$1"
                        fi
                        shift
                        ;;
                esac
            done
            llamaindex::get_status "$format"
            ;;
        start)
            llamaindex::start
            ;;
        stop)
            llamaindex::stop
            ;;
        restart)
            llamaindex::restart
            ;;
        install)
            llamaindex::install
            ;;
        uninstall)
            llamaindex::uninstall
            ;;
        index)
            llamaindex::index_documents "$@"
            ;;
        query)
            llamaindex::query "$@"
            ;;
        list-indices)
            llamaindex::list_indices
            ;;
        help|--help|-h)
            cat <<EOF
LlamaIndex CLI - RAG and Document Processing

Usage: resource-llamaindex <command> [options]

Commands:
  status         Show status (default)
  start          Start LlamaIndex service
  stop           Stop LlamaIndex service
  restart        Restart service
  install        Install LlamaIndex
  uninstall      Uninstall LlamaIndex
  index <path>   Index documents from path
  query <text>   Query indexed documents
  list-indices   List available indices
  help           Show this help

Examples:
  resource-llamaindex status
  resource-llamaindex index /path/to/docs
  resource-llamaindex query "What is Vrooli?"
EOF
            ;;
        *)
            log::error "Unknown command: $command"
            echo "Use 'resource-llamaindex help' for usage information"
            return 1
            ;;
    esac
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
