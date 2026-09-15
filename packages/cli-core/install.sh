#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
REPO_ROOT="$(builtin cd "${SCRIPT_DIR}/../.." && builtin pwd)"
# shellcheck source=install/platform.sh
source "${SCRIPT_DIR}/install/platform.sh"

usage() {
    echo "Usage: $0 <module_path> [--name binary-name] [--manifest path] [--install-dir path] [--context-root path] [--freshness-input pattern]"
    echo
    echo "Examples:"
    echo "  $0 scenarios/scenario-completeness-scoring/cli --name scenario-completeness-scoring"
    echo "  CLI_CORE_VERSION=v0.0.1 $0 scenarios/my-cli --install-dir ~/.local/bin"
}

if [[ $# -lt 1 ]]; then
    usage
    exit 1
fi

MODULE_PATH="$1"
shift

NAME=""
MANIFEST=""
INSTALL_DIR="$(vrooli_default_install_dir)"
CONTEXT_ROOT=""
FRESHNESS_INPUTS=()

while [[ $# -gt 0 ]]; do
    case "$1" in
        --name)
            if [[ $# -lt 2 ]]; then
                echo "Missing value for --name"
                exit 1
            fi
            NAME="$2"
            shift 2
            ;;
        --install-dir)
            if [[ $# -lt 2 ]]; then
                echo "Missing value for --install-dir"
                exit 1
            fi
            INSTALL_DIR="$2"
            shift 2
            ;;
        --manifest)
            if [[ $# -lt 2 ]]; then
                echo "Missing value for --manifest"
                exit 1
            fi
            MANIFEST="$2"
            shift 2
            ;;
        --context-root)
            if [[ $# -lt 2 ]]; then
                echo "Missing value for --context-root"
                exit 1
            fi
            CONTEXT_ROOT="$2"
            shift 2
            ;;
        --freshness-input)
            if [[ $# -lt 2 ]]; then
                echo "Missing value for --freshness-input"
                exit 1
            fi
            FRESHNESS_INPUTS+=("$2")
            shift 2
            ;;
        *)
            echo "Unknown argument: $1"
            usage
            exit 1
            ;;
    esac
done

if ! command -v go >/dev/null; then
    echo "Go toolchain is required to build the CLI."
    exit 1
fi

if [[ "${MODULE_PATH}" != /* ]]; then
    MODULE_ABS="${REPO_ROOT}/${MODULE_PATH}"
else
    MODULE_ABS="${MODULE_PATH}"
fi

if [[ -z "${NAME}" ]]; then
    base="$(basename "${MODULE_ABS}")"
    parent="$(basename "$(dirname "${MODULE_ABS}")")"
    if [[ "${base}" == "cli" ]]; then
        NAME="${parent}"
    else
        NAME="${base}"
    fi
fi

if [[ ! -f "${MODULE_ABS}/go.mod" ]]; then
    echo "Module path must contain go.mod: ${MODULE_ABS}"
    exit 1
fi

MANIFEST_ABS=""
if [[ -n "${MANIFEST}" ]]; then
    if [[ "${MANIFEST}" != /* ]]; then
        MANIFEST_ABS="${REPO_ROOT}/${MANIFEST}"
    else
        MANIFEST_ABS="${MANIFEST}"
    fi
    if [[ ! -f "${MANIFEST_ABS}" ]]; then
        echo "Manifest path must contain a file: ${MANIFEST_ABS}"
        exit 1
    fi
fi

if [[ -z "${CONTEXT_ROOT}" && "$(basename "${MODULE_ABS}")" == "cli" ]]; then
    CONTEXT_ROOT="$(dirname "${MODULE_ABS}")"
fi

if [[ ${#FRESHNESS_INPUTS[@]} -eq 0 && -n "${MANIFEST_ABS}" && -n "${CONTEXT_ROOT}" ]]; then
    if command -v jq >/dev/null 2>&1; then
        while IFS= read -r input; do
            if [[ -n "${input}" ]]; then
                FRESHNESS_INPUTS+=("${input}")
            fi
        done < <(jq -r '.cli.freshness.inputs[]? // empty' "${MANIFEST_ABS}")
    fi
    if [[ ${#FRESHNESS_INPUTS[@]} -eq 0 && "${MANIFEST_ABS}" == "${CONTEXT_ROOT}/.vrooli/service.json" ]]; then
        FRESHNESS_INPUTS=("cli/**" ".vrooli/service.json")
    fi
fi

INSTALLER_TARGET="${CLI_CORE_VERSION:+github.com/vrooli/cli-core/cmd/cli-installer@${CLI_CORE_VERSION}}"
INSTALLER_DIR="${REPO_ROOT}"

if [[ -z "${INSTALLER_TARGET}" ]]; then
    INSTALLER_TARGET="./cmd/cli-installer"
    INSTALLER_DIR="${REPO_ROOT}/packages/cli-core"
fi

echo "Building ${NAME} from ${MODULE_ABS}..."
(
    cd "${INSTALLER_DIR}"
    cmd=(
        go run "${INSTALLER_TARGET}"
        --module "${MODULE_ABS}"
        --name "${NAME}"
        --install-dir "${INSTALL_DIR}"
    )
    if [[ -n "${MANIFEST_ABS}" ]]; then
        cmd+=(--manifest "${MANIFEST_ABS}")
    fi
    if [[ -n "${CONTEXT_ROOT}" ]]; then
        cmd+=(--context-root "${CONTEXT_ROOT}")
    fi
    for input in "${FRESHNESS_INPUTS[@]}"; do
        cmd+=(--freshness-input "${input}")
    done
    "${cmd[@]}"
)
