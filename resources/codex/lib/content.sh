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
# shellcheck disable=SC1091
source "${CODEX_CONTENT_DIR}/orchestrator.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${CODEX_CONTENT_DIR}/tools/registry.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${CODEX_CONTENT_DIR}/workspace/manager.sh" 2>/dev/null || true

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
# Execute content (run script with Codex)
# Arguments:
#   input - Prompt, file path, or script name
#   operation - Operation type (generate, review, test, explain, complete)
#   context - Execution context (optional: cli, sandbox, text)
# Returns:
#   0 on success, 1 on failure
#######################################
codex::content::execute() {
    local input="${1:-}"
    local operation="${2:-generate}"
    local context="${3:-auto}"
    
    if [[ -z "${input}" ]]; then
        log::error "No input specified"
        echo "Usage: content execute <prompt_or_file> [operation] [context]"
        echo "Operations: generate, review, test, explain, complete, analyze"
        echo "Contexts: auto, cli, direct, sandbox, text"
        return 1
    fi
    
    # Determine capability based on operation
    local capability
    case "$operation" in
        generate)
            # For generate operation, analyze the actual request to determine capability
            if type -t orchestrator::analyze_request &>/dev/null; then
                capability=$(orchestrator::analyze_request "$input")
            else
                capability="text-generation"
            fi
            ;;
        complete|explain)
            capability="text-generation"
            ;;
        review|test|analyze)
            capability="function-calling"
            ;;
        *)
            capability="text-generation"
            ;;
    esac
    
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
            analyze)
                # Use code analysis tool
                capability="function-calling"
                request_content="Analyze this code for issues and improvements:\n\n$request_content"
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
            analyze)
                capability="function-calling"
                request_content="Analyze this code for issues and improvements:\n\n$request_content"
                ;;
            *)
                request_content="Complete or improve this code:\n\n$request_content"
                ;;
        esac
    else
        # It's a direct prompt
        request_content="$input"
    fi
    
    # Use new orchestrator system
    if type -t orchestrator::execute &>/dev/null; then
        log::info "Executing via orchestrator (capability: $capability, context: $context)"
        
        # Create workspace if using function calling with sandbox context
        local workspace_id=""
        if [[ "$capability" == "function-calling" || "$operation" == "analyze" ]] && [[ "$context" == "sandbox" ]]; then
            if type -t workspace_manager::create &>/dev/null; then
                local workspace_result
                workspace_result=$(workspace_manager::create "" "moderate" '{"description": "Content execution workspace", "auto_cleanup": true}')
                
                if [[ $(echo "$workspace_result" | jq -r '.success // false') == "true" ]]; then
                    workspace_id=$(echo "$workspace_result" | jq -r '.workspace_id')
                    log::debug "Created workspace: $workspace_id"
                fi
            fi
        fi
        
        # Execute via orchestrator
        local result
        if [[ "$context" == "auto" ]]; then
            result=$(orchestrator::execute "$request_content")
        else
            result=$(orchestrator::execute "$request_content" "$context")
        fi
        
        # Output result
        echo "$result"
        
        # Cleanup workspace
        if [[ -n "$workspace_id" ]] && type -t workspace_manager::delete &>/dev/null; then
            workspace_manager::delete "$workspace_id" "true" >/dev/null 2>&1
        fi
        
        return 0
    else
        # Fallback to legacy system
        log::warn "Orchestrator not available, falling back to legacy system"
        
        if type -t codex::smart_execute &>/dev/null; then
            codex::smart_execute "$request_content"
        elif type -t codex::generate_code &>/dev/null; then
            codex::generate_code "$request_content"
        else
            log::error "No execution system available"
            return 1
        fi
    fi
}

#######################################
# Analyze content using the new code analysis tools
# Arguments:
#   name_or_path - Name of script or file path
#   analysis_type - Type of analysis (syntax, style, complexity, security, all)
# Returns:
#   0 on success, 1 on failure
#######################################
codex::content::analyze() {
    local input="${1:-}"
    local analysis_type="${2:-all}"
    
    if [[ -z "${input}" ]]; then
        log::error "No input specified"
        echo "Usage: content analyze <script_name_or_path> [analysis_type]"
        echo "Analysis types: syntax, style, complexity, security, all"
        return 1
    fi
    
    # Determine file path
    local file_path
    if [[ -f "${input}" ]]; then
        file_path="${input}"
    elif [[ -f "${CODEX_SCRIPTS_DIR}/${input}" ]]; then
        file_path="${CODEX_SCRIPTS_DIR}/${input}"
    else
        log::error "File not found: ${input}"
        return 1
    fi
    
    # Read file content
    local code_content
    code_content=$(cat "$file_path")
    
    # Use tools registry to analyze code
    if type -t tool_registry::execute_tool &>/dev/null; then
        local language
        language=$(basename "$file_path" | sed 's/.*\.//')
        
        # Map common extensions to language names
        case "$language" in
            py) language="python" ;;
            js) language="javascript" ;;
            ts) language="typescript" ;;
            go) language="go" ;;
            sh) language="bash" ;;
        esac
        
        local tool_args
        tool_args=$(jq -n \
            --arg code "$code_content" \
            --arg lang "$language" \
            --arg type "$analysis_type" \
            '{code: $code, language: $lang, analysis_type: $type}')
        
        local result
        result=$(tool_registry::execute_tool "analyze_code" "$tool_args" "sandbox")
        
        if [[ $(echo "$result" | jq -r '.success // false') == "true" ]]; then
            # Format and display results nicely
            echo "Code Analysis Results for: $(basename "$file_path")"
            echo "================================================"
            echo "$result" | jq -r 'if .analysis_type == "comprehensive" then
                "Comprehensive Analysis:\n" +
                "- Syntax errors: " + (.syntax.error_count | tostring) + "\n" +
                "- Style score: " + (.style.style_score | tostring) + "\n" +
                "- Complexity: " + .complexity.complexity_rating + "\n" +
                "- Security risk: " + .security.risk_level + "\n"
            else
                "Analysis Type: " + .analysis_type + "\n" +
                (if .error_count then "Errors: " + (.error_count | tostring) + "\n" else "" end) +
                (if .issue_count then "Issues: " + (.issue_count | tostring) + "\n" else "" end) +
                (if .vulnerability_count then "Vulnerabilities: " + (.vulnerability_count | tostring) + "\n" else "" end)
            end'
        else
            log::error "Code analysis failed: $(echo "$result" | jq -r '.error // "Unknown error"')"
            return 1
        fi
    else
        log::error "Code analysis tools not available"
        return 1
    fi
}

#######################################
# Create a new workspace for content development
# Arguments:
#   workspace_name - Optional name for the workspace
#   security_level - Security level (strict, moderate, relaxed)
# Returns:
#   0 on success, 1 on failure
#######################################
codex::content::create_workspace() {
    local workspace_name="${1:-content-dev-$(date +%s)}"
    local security_level="${2:-moderate}"
    
    if type -t workspace_manager::create &>/dev/null; then
        local options
        options=$(jq -n \
            --arg desc "Content development workspace" \
            --argjson cleanup true \
            --argjson backup false \
            --argjson monitoring false \
            '{description: $desc, auto_cleanup: $cleanup, backup_on_delete: $backup, monitoring: $monitoring}')
        
        local result
        result=$(workspace_manager::create "$workspace_name" "$security_level" "$options")
        
        if [[ $(echo "$result" | jq -r '.success // false') == "true" ]]; then
            local workspace_id workspace_dir
            workspace_id=$(echo "$result" | jq -r '.workspace_id')
            workspace_dir=$(echo "$result" | jq -r '.workspace_dir')
            
            log::success "Workspace created: $workspace_id"
            log::info "Workspace directory: $workspace_dir"
            log::info "Data directory: $workspace_dir/data"
            
            # Export workspace info for easy access
            export CODEX_CURRENT_WORKSPACE="$workspace_id"
            export CODEX_CURRENT_WORKSPACE_DIR="$workspace_dir"
            
            echo "Workspace ID: $workspace_id"
            echo "Export these variables to use:"
            echo "export CODEX_CURRENT_WORKSPACE=\"$workspace_id\""
            echo "export CODEX_CURRENT_WORKSPACE_DIR=\"$workspace_dir\""
        else
            log::error "Failed to create workspace: $(echo "$result" | jq -r '.error // "Unknown error"')"
            return 1
        fi
    else
        log::error "Workspace management not available"
        return 1
    fi
}

#######################################
# List available workspaces
# Arguments:
#   filter - Filter workspaces (active, expired, all)
# Returns:
#   0 on success
#######################################
codex::content::list_workspaces() {
    local filter="${1:-active}"
    
    if type -t workspace_manager::list &>/dev/null; then
        local result
        result=$(workspace_manager::list "$filter")
        
        local count
        count=$(echo "$result" | jq 'length')
        
        if [[ $count -gt 0 ]]; then
            log::header "Content Workspaces ($filter)"
            echo "$result" | jq -r '.[] | "  " + .workspace_id + " (" + .security_level + ") - " + (.created_at // "unknown")'
            echo ""
            log::info "Total: $count workspace(s)"
        else
            log::info "No $filter workspaces found"
        fi
    else
        log::error "Workspace management not available"
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
        analyze)
            shift
            codex::content::analyze "$@"
            ;;
        workspace)
            case "$2" in
                create)
                    shift 2
                    codex::content::create_workspace "$@"
                    ;;
                list)
                    shift 2
                    codex::content::list_workspaces "$@"
                    ;;
                *)
                    echo "Usage: $0 workspace {create|list} [args...]"
                    exit 1
                    ;;
            esac
            ;;
        *)
            echo "Usage: $0 {add|list|get|remove|execute|analyze|workspace} [args...]"
            echo ""
            echo "Commands:"
            echo "  add <file> [name]           - Add a script to Codex"
            echo "  list [--format json]        - List all scripts"
            echo "  get <name>                  - Show script details"
            echo "  remove <name>               - Remove a script"
            echo "  execute <input> [op] [ctx]  - Execute with Codex"
            echo "  analyze <script> [type]     - Analyze code quality"
            echo "  workspace create [name] [sec] - Create development workspace"
            echo "  workspace list [filter]     - List workspaces"
            exit 1
            ;;
    esac
fi