#!/bin/bash
# KiCad Resource CLI

# Get the directory of this script (resolving symlinks)
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    # If this is a symlink, resolve it
    KICAD_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
else
    KICAD_CLI_SCRIPT="${BASH_SOURCE[0]}"
fi
KICAD_CLI_DIR="$(cd "$(dirname "$KICAD_CLI_SCRIPT")" && pwd)"

# Source the library functions
source "${KICAD_CLI_DIR}/lib/common.sh"
source "${KICAD_CLI_DIR}/lib/install.sh"
source "${KICAD_CLI_DIR}/lib/status.sh"
source "${KICAD_CLI_DIR}/lib/inject.sh"
source "${KICAD_CLI_DIR}/lib/test.sh" 2>/dev/null || true

# Main CLI handler
kicad_cli() {
    case "${1:-}" in
        install)
            shift
            kicad::install "$@"
            ;;
        start)
            # KiCad is a desktop application, no start needed
            echo "KiCad is a desktop application - no service to start"
            if kicad::is_installed; then
                echo "✅ KiCad is installed and ready to use"
            else
                echo "❌ KiCad is not installed. Run: vrooli resource install kicad"
            fi
            ;;
        stop)
            # KiCad is a desktop application, no stop needed
            echo "KiCad is a desktop application - no service to stop"
            ;;
        status)
            shift
            kicad_status "$@"
            ;;
        inject)
            shift
            kicad::inject "$@"
            ;;
        export)
            shift
            local project="${1:-}"
            local formats="${2:-}"
            if [[ -z "$project" ]]; then
                echo "Error: Project name required"
                echo "Usage: resource-kicad export <project> [formats]"
                exit 1
            fi
            kicad::export::project "$project" "$formats"
            ;;
        list-projects)
            kicad::init_dirs
            echo "KiCad Projects:"
            if [[ -d "$KICAD_PROJECTS_DIR" ]]; then
                find "$KICAD_PROJECTS_DIR" -name "*.kicad_pro" -o -name "*.pro" 2>/dev/null | while read -r proj; do
                    echo "  - $(basename "$(dirname "$proj")")"
                done
            else
                echo "  No projects found"
            fi
            ;;
        list-libraries)
            kicad::init_dirs
            echo "KiCad Libraries:"
            if [[ -d "$KICAD_LIBRARIES_DIR" ]]; then
                ls -la "$KICAD_LIBRARIES_DIR" 2>/dev/null | grep -E "\.(kicad_sym|kicad_mod|lib)$" | awk '{print "  - " $NF}'
            else
                echo "  No libraries found"
            fi
            ;;
        test)
            shift
            if command -v kicad_run_tests &>/dev/null; then
                kicad_run_tests "$@"
            else
                echo "Running KiCad integration tests..."
                if [[ -f "${KICAD_CLI_DIR}/test/integration.bats" ]]; then
                    timeout 60 bats "${KICAD_CLI_DIR}/test/integration.bats"
                else
                    echo "Test file not found"
                    exit 1
                fi
            fi
            ;;
        examples)
            echo "KiCad Examples:"
            echo ""
            echo "Project Management:"
            echo "  resource-kicad inject /path/to/project.zip          # Import project"
            echo "  resource-kicad list-projects                        # List all projects"
            echo "  resource-kicad export my-board gerber,pdf          # Export to formats"
            echo ""
            echo "Library Management:"
            echo "  resource-kicad inject /path/to/symbols.kicad_sym library  # Import symbols"
            echo "  resource-kicad inject /path/to/footprints library        # Import footprints"
            echo "  resource-kicad list-libraries                            # List libraries"
            echo ""
            if [[ -d "${KICAD_CLI_DIR}/examples" ]]; then
                echo "Example files available in: ${KICAD_CLI_DIR}/examples/"
                ls -la "${KICAD_CLI_DIR}/examples/" 2>/dev/null | grep -v "^total\|^d" | awk '{print "  - " $NF}'
            fi
            ;;
        help|--help)
            kicad_help
            ;;
        *)
            echo "Error: Unknown command: ${1:-}"
            echo
            kicad_help
            exit 1
            ;;
    esac
}

# Help function
kicad_help() {
    cat <<EOF
KiCad CLI - Electronic Design Automation for PCB Design

Usage: $(basename "$0") <command> [options]

Commands:
    install         Install KiCad and dependencies
    start           Check if KiCad is ready (desktop app)
    stop            No-op for desktop application
    status          Show KiCad status and configuration
    inject          Import projects, libraries, or templates
    export          Export project to various formats
    list-projects   List all KiCad projects
    list-libraries  List all KiCad libraries
    test            Run integration tests
    examples        Show usage examples
    help            Show this help message

Examples:
    $(basename "$0") inject circuit.kicad_pcb
    $(basename "$0") export my-board gerber,pdf,step
    $(basename "$0") inject symbols.kicad_sym library

Export Formats:
    gerber  - Manufacturing files for PCB fabrication
    pdf     - PDF documentation
    svg     - Vector graphics
    step    - 3D model for mechanical CAD

For more information, see: scripts/resources/execution/kicad/README.md
EOF
}

# Execute CLI if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    kicad_cli "$@"
fi