#!/bin/bash

# LlamaIndex CLI Interface
# Direct wrapper for LlamaIndex resource management

# Resolve symlinks to get the actual script directory
LLAMAINDEX_CLI_SCRIPT="${BASH_SOURCE[0]}"
while [[ -L "$LLAMAINDEX_CLI_SCRIPT" ]]; do
    LLAMAINDEX_CLI_DIR="$(cd "$(dirname "$LLAMAINDEX_CLI_SCRIPT")" && pwd)"
    LLAMAINDEX_CLI_SCRIPT="$(readlink "$LLAMAINDEX_CLI_SCRIPT")"
    [[ "$LLAMAINDEX_CLI_SCRIPT" != /* ]] && LLAMAINDEX_CLI_SCRIPT="$LLAMAINDEX_CLI_DIR/$LLAMAINDEX_CLI_SCRIPT"
done
LLAMAINDEX_CLI_DIR="$(cd "$(dirname "$LLAMAINDEX_CLI_SCRIPT")" && pwd)"
LLAMAINDEX_LIB_DIR="$LLAMAINDEX_CLI_DIR/lib"

# Source core functions
source "$LLAMAINDEX_LIB_DIR/core.sh" || exit 1

# Main CLI handler
main() {
    local command="${1:-status}"
    shift || true
    
    case "$command" in
        status)
            # Handle --format flag
            local format="plain"
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
                    *)
                        format="$1"
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