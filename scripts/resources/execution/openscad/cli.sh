#!/usr/bin/env bash
set -euo pipefail

# Get the directory of this script
OPENSCAD_CLI_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Main CLI handler
main() {
    local command="${1:-}"
    shift || true
    
    case "${command}" in
        start)
            source "${OPENSCAD_CLI_DIR}/lib/common.sh"
            source "${OPENSCAD_CLI_DIR}/../../../lib/utils/log.sh"
            source "${OPENSCAD_CLI_DIR}/lib/install.sh"
            source "${OPENSCAD_CLI_DIR}/lib/lifecycle.sh"
            openscad::start "$@"
            ;;
        stop)
            source "${OPENSCAD_CLI_DIR}/lib/common.sh"
            source "${OPENSCAD_CLI_DIR}/../../../lib/utils/log.sh"
            source "${OPENSCAD_CLI_DIR}/lib/lifecycle.sh"
            openscad::stop "$@"
            ;;
        restart)
            source "${OPENSCAD_CLI_DIR}/lib/common.sh"
            source "${OPENSCAD_CLI_DIR}/../../../lib/utils/log.sh"
            source "${OPENSCAD_CLI_DIR}/lib/install.sh"
            source "${OPENSCAD_CLI_DIR}/lib/lifecycle.sh"
            openscad::restart "$@"
            ;;
        status)
            source "${OPENSCAD_CLI_DIR}/lib/common.sh"
            source "${OPENSCAD_CLI_DIR}/../../../lib/utils/log.sh"
            source "${OPENSCAD_CLI_DIR}/lib/status.sh"
            openscad::status "$@"
            ;;
        install)
            source "${OPENSCAD_CLI_DIR}/lib/common.sh"
            source "${OPENSCAD_CLI_DIR}/../../../lib/utils/log.sh"
            source "${OPENSCAD_CLI_DIR}/lib/install.sh"
            openscad::install "$@"
            ;;
        uninstall)
            source "${OPENSCAD_CLI_DIR}/lib/common.sh"
            source "${OPENSCAD_CLI_DIR}/../../../lib/utils/log.sh"
            source "${OPENSCAD_CLI_DIR}/lib/install.sh"
            openscad::uninstall "$@"
            ;;
        inject)
            source "${OPENSCAD_CLI_DIR}/lib/common.sh"
            source "${OPENSCAD_CLI_DIR}/../../../lib/utils/log.sh"
            source "${OPENSCAD_CLI_DIR}/lib/inject.sh"
            source "${OPENSCAD_CLI_DIR}/lib/lifecycle.sh"
            openscad::inject "$@"
            ;;
        list|list-injected)
            source "${OPENSCAD_CLI_DIR}/lib/common.sh"
            source "${OPENSCAD_CLI_DIR}/../../../lib/utils/log.sh"
            source "${OPENSCAD_CLI_DIR}/lib/inject.sh"
            openscad::list_injected "$@"
            ;;
        render)
            source "${OPENSCAD_CLI_DIR}/lib/common.sh"
            source "${OPENSCAD_CLI_DIR}/../../../lib/utils/log.sh"
            source "${OPENSCAD_CLI_DIR}/lib/inject.sh"
            source "${OPENSCAD_CLI_DIR}/lib/lifecycle.sh"
            openscad::render "$@"
            ;;
        clear|clear-data)
            source "${OPENSCAD_CLI_DIR}/lib/common.sh"
            source "${OPENSCAD_CLI_DIR}/../../../lib/utils/log.sh"
            source "${OPENSCAD_CLI_DIR}/lib/inject.sh"
            openscad::clear_data "$@"
            ;;
        help|--help|-h|"")
            cat << EOF
OpenSCAD Resource CLI

Usage: resource-openscad <command> [options]

Commands:
    start           Start OpenSCAD service
    stop            Stop OpenSCAD service
    restart         Restart OpenSCAD service
    status [format] Check OpenSCAD status (format: text|json)
    install         Install OpenSCAD
    uninstall       Uninstall OpenSCAD
    inject <file>   Inject OpenSCAD script (.scad)
    list            List injected scripts and outputs
    render <script> [format] [name]  Render script to format (stl,off,png,etc)
    clear           Clear all OpenSCAD data
    help            Show this help message

Examples:
    resource-openscad install
    resource-openscad start
    resource-openscad status json
    resource-openscad inject cube.scad
    resource-openscad render cube.scad stl my-cube
    resource-openscad list

Output formats: stl, off, amf, 3mf, dxf, svg, png
Default directories:
    Scripts: ~/.openscad/scripts
    Outputs: ~/.openscad/output

OpenSCAD is a programmatic 3D CAD modeler using script-based solid modeling.
EOF
            ;;
        *)
            echo "Unknown command: ${command}"
            echo "Run 'resource-openscad help' for usage information"
            exit 1
            ;;
    esac
}

main "$@"