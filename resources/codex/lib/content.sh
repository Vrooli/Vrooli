#!/usr/bin/env bash
# Codex Content Management Functions

# Set script directory for sourcing
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
CODEX_CONTENT_DIR="${APP_ROOT}/resources/codex/lib"

# Source required utilities
# shellcheck disable=SC1091
source "${CODEX_CONTENT_DIR}/common.sh"
# shellcheck disable=SC1091
source "${CODEX_CONTENT_DIR}/core.sh" 2>/dev/null || true

#######################################
# Add content (Python script) to Codex
# Arguments:
#   file_path - Path to the Python script to add
#   [name] - Optional name for the script
# Returns:
#   0 on success, 1 on failure
#######################################
codex::content::add() {
    local file_path="${1:-}"
    local name="${2:-}"

    if [[ -z "${file_path}" ]]; then
        log::error "No file specified"
        echo "Usage: content add <file_path> [name]"
        return 1
    fi

    if [[ ! -f "${file_path}" ]]; then
        log::error "File not found: ${file_path}"
        return 1
    fi

    # Create directories if needed
    mkdir -p "${CODEX_SCRIPTS_DIR}" "${CODEX_INJECTED_DIR}" "${CODEX_OUTPUT_DIR}"

    # Determine target name
    local basename
    basename=$(basename "${file_path}")
    if [[ -n "${name}" ]]; then
        # Use provided name but keep extension
        basename="${name}"
        # Add appropriate extension if missing
        if [[ ! "${basename}" =~ \.(py|js|go|sh|sql|java|cpp|c|rs)$ ]]; then
            # Try to detect from original file
            local ext="${file_path##*.}"
            if [[ -n "$ext" ]]; then
                basename="${basename}.${ext}"
            fi
        fi
    fi

    local target="${CODEX_SCRIPTS_DIR}/${basename}"

    log::info "Adding content: ${basename}"

    # Copy to scripts directory
    if cp "${file_path}" "${target}"; then
        # Also copy to injected directory for tracking
        cp "${file_path}" "${CODEX_INJECTED_DIR}/${basename}"

        log::success "Content added successfully: ${basename}"
        log::info "Content location: ${target}"
    else
        log::error "Failed to add content"
        return 1
    fi

    return 0
}

#######################################
# List all content (Python scripts)
# Arguments:
#   --format [text|json] - Output format
# Returns:
#   0 on success
#######################################
codex::content::list() {
    local format="text"

    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --format)
                format="${2:-text}"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done

    if [[ ! -d "${CODEX_SCRIPTS_DIR}" ]]; then
        if [[ "$format" == "json" ]]; then
            echo "[]"
        else
            log::info "No scripts directory found"
        fi
        return 0
    fi

    local scripts
    scripts=$(find "${CODEX_SCRIPTS_DIR}" -name "*.py" -type f 2>/dev/null | sort)

    if [[ "$format" == "json" ]]; then
        echo "["
        local first=true
        while IFS= read -r script; do
            [[ -z "$script" ]] && continue
            if [[ "$first" == "true" ]]; then
                first=false
            else
                echo ","
            fi
            local basename size modified
            basename=$(basename "${script}")
            size=$(stat -c%s "${script}" 2>/dev/null || echo "0")
            modified=$(stat -c%y "${script}" 2>/dev/null | cut -d. -f1 || echo "unknown")
            echo -n "  {\"name\": \"${basename}\", \"size\": ${size}, \"modified\": \"${modified}\"}"
        done <<< "${scripts}"
        echo ""
        echo "]"
    else
        log::header "Codex Scripts"

        local count=0
        while IFS= read -r script; do
            [[ -z "$script" ]] && continue
            ((count++))
            local basename size modified
            basename=$(basename "${script}")
            size=$(stat -c%s "${script}" 2>/dev/null || echo "0")
            modified=$(stat -c%y "${script}" 2>/dev/null | cut -d. -f1 || echo "unknown")
            printf "  %2d. %-30s %6s bytes  %s\n" "${count}" "${basename}" "${size}" "${modified}"
        done <<< "${scripts}"

        if [[ ${count} -eq 0 ]]; then
            log::info "No scripts found"
        else
            echo ""
            log::info "Total: ${count} script(s)"
        fi
    fi

    return 0
}

#######################################
# Get content (show script details)
# Arguments:
#   name - Name of the script to get
# Returns:
#   0 on success, 1 if not found
#######################################
codex::content::get() {
    local name="${1:-}"

    if [[ -z "${name}" ]]; then
        log::error "No script name specified"
        echo "Usage: content get <script_name>"
        return 1
    fi

    # Add .py extension if not present
    if [[ ! "${name}" =~ \.py$ ]]; then
        name="${name}.py"
    fi

    local script_path="${CODEX_SCRIPTS_DIR}/${name}"

    if [[ ! -f "${script_path}" ]]; then
        log::error "Script not found: ${name}"
        return 1
    fi

    log::header "Script: ${name}"

    # Show metadata
    local size modified
    size=$(stat -c%s "${script_path}" 2>/dev/null || echo "0")
    modified=$(stat -c%y "${script_path}" 2>/dev/null | cut -d. -f1 || echo "unknown")

    echo "Location: ${script_path}"
    echo "Size: ${size} bytes"
    echo "Modified: ${modified}"
    echo ""
    echo "Content:"
    echo "--------"
    cat "${script_path}"

    return 0
}

#######################################
# Remove content (delete script)
# Arguments:
#   name - Name of the script to remove
# Returns:
#   0 on success, 1 if not found
#######################################
codex::content::remove() {
    local name="${1:-}"

    if [[ -z "${name}" ]]; then
        log::error "No script name specified"
        echo "Usage: content remove <script_name>"
        return 1
    fi

    # Add .py extension if not present
    if [[ ! "${name}" =~ \.py$ ]]; then
        name="${name}.py"
    fi

    local script_path="${CODEX_SCRIPTS_DIR}/${name}"
    local injected_path="${CODEX_INJECTED_DIR}/${name}"

    if [[ ! -f "${script_path}" ]]; then
        log::error "Script not found: ${name}"
        return 1
    fi

    log::info "Removing script: ${name}"

    # Remove from both locations
    rm -f "${script_path}"
    rm -f "${injected_path}"

    log::success "Script removed successfully: ${name}"
    return 0
}

#######################################
# Show usage information for content execute command
#######################################
codex::content::show_usage() {
    echo "Usage: content execute [OPTIONS] <prompt_or_file>"
    echo
    echo "Execute prompts or analyze files using Codex AI."
    echo
    echo "ARGUMENTS:"
    echo "  <prompt_or_file>     Text prompt or path to file for processing"
    echo
    echo "OPTIONS:"
    echo "  --allowed-tools TOOLS    Comma-separated list of allowed tools"
    echo "  --max-turns NUMBER      Maximum conversation turns (default: 10)"
    echo "  --timeout SECONDS       Timeout in seconds (default: 1800)"
    echo "  --skip-permissions      Skip confirmation prompts (DANGEROUS)"
    echo "  --operation TYPE        Operation type: generate, review, test, explain"
    echo "  -h, --help             Show this help message"
    echo
    echo "EXAMPLES:"
    echo "  content execute 'Fix the auth bug'"
    echo "  content execute 'Review main.py' --operation review"
    echo "  content execute script.py --operation test"
}

#######################################
# Execute content (run script with Codex)
# Arguments:
#   All arguments parsed as flags and positional parameters
# Returns:
#   0 on success, 1 on failure
#######################################
codex::content::execute() {
    # Parse command line arguments
    local input=""
    local operation="generate"
    local allowed_tools=""
    local max_turns=""
    local timeout=""
    local skip_permissions=false

    # Parse flags and positional arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --allowed-tools)
                allowed_tools="$2"
                shift 2
                ;;
            --max-turns)
                max_turns="$2"
                shift 2
                ;;
            --timeout)
                timeout="$2"
                shift 2
                ;;
            --skip-permissions)
                skip_permissions=true
                shift
                ;;
            --operation)
                operation="$2"
                shift 2
                ;;
            --help|-h)
                codex::content::show_usage
                return 0
                ;;
            -*)
                log::error "Unknown option: $1"
                codex::content::show_usage
                return 1
                ;;
            *)
                if [[ -z "$input" ]]; then
                    input="$1"
                fi
                shift
                ;;
        esac
    done

    if [[ -z "${input}" ]]; then
        log::error "No input specified"
        codex::content::show_usage
        return 1
    fi

    # Set permission overrides from command line
    if [[ -n "$allowed_tools" ]]; then
        export CODEX_ALLOWED_TOOLS="$allowed_tools"
    fi

    if [[ -n "$max_turns" ]]; then
        export CODEX_MAX_TURNS="$max_turns"
    fi

    if [[ -n "$timeout" ]]; then
        export CODEX_TIMEOUT="$timeout"
    fi

    if [[ "$skip_permissions" == true ]]; then
        export CODEX_SKIP_CONFIRMATIONS="true"
        export CODEX_SKIP_PERMISSIONS="true"
        export CODEX_ALLOWED_TOOLS="*"
        export CODEX_CLI_MODE="yolo"
        export CODEX_CLI_SANDBOX="danger-full-access"
        log::warn "WARNING: Permission checks are disabled!"
    fi

    # Check if input is a file or a prompt
    local request_content
    if [[ -f "${input}" ]]; then
        # It's a file - read content and create appropriate request
        request_content=$(cat "${input}")
        case "$operation" in
            review)
                request_content="Review this code and provide feedback:\n\n$request_content"
                ;;
            test)
                request_content="Create comprehensive tests for this code:\n\n$request_content"
                ;;
            explain)
                request_content="Explain what this code does:\n\n$request_content"
                ;;
            *)
                request_content="Complete or improve this code:\n\n$request_content"
                ;;
        esac
    elif [[ -f "${CODEX_SCRIPTS_DIR}/${input}" ]]; then
        # Check in scripts directory
        request_content=$(cat "${CODEX_SCRIPTS_DIR}/${input}")
        case "$operation" in
            review)
                request_content="Review this code and provide feedback:\n\n$request_content"
                ;;
            test)
                request_content="Create comprehensive tests for this code:\n\n$request_content"
                ;;
            explain)
                request_content="Explain what this code does:\n\n$request_content"
                ;;
            *)
                request_content="Complete or improve this code:\n\n$request_content"
                ;;
        esac
    else
        # It's a direct prompt
        request_content="$input"
    fi

    # Execute using available backend
    if type -t codex::smart_execute &>/dev/null; then
        codex::smart_execute "$request_content"
    elif type -t codex::generate_code &>/dev/null; then
        codex::generate_code "$request_content"
    else
        log::error "No execution system available"
        return 1
    fi
}

# Main entry point
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    case "$1" in
        add)
            shift
            codex::content::add "$@"
            ;;
        list)
            shift
            codex::content::list "$@"
            ;;
        get)
            shift
            codex::content::get "$@"
            ;;
        remove)
            shift
            codex::content::remove "$@"
            ;;
        execute)
            shift
            codex::content::execute "$@"
            ;;
        *)
            echo "Usage: $0 {add|list|get|remove|execute} [args...]"
            echo ""
            echo "Commands:"
            echo "  add <file> [name]           - Add a script to Codex"
            echo "  list [--format json]        - List all scripts"
            echo "  get <name>                  - Show script details"
            echo "  remove <name>               - Remove a script"
            echo "  execute <input> [options]   - Execute with Codex"
            exit 1
            ;;
    esac
fi
