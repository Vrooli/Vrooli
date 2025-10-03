#!/usr/bin/env bash

# AI Agent Investigation Script
# Unified interface for Codex and Claude Code for issue investigation

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIG_DIR="${SCRIPT_DIR}/../initialization/configuration"
AGENT_SETTINGS="${CONFIG_DIR}/agent-settings.json"
CODEX_CONFIG="${CONFIG_DIR}/codex-config.json"
PROMPTS_DIR="${SCRIPT_DIR}/../prompts"
DEFAULT_PROMPT_FILE="${PROMPTS_DIR}/unified-resolver.md"

# Agent backend (dynamically determined)
AGENT_PROVIDER=""
AGENT_CLI_COMMAND=""
AGENT_OPERATION_CMD=""

STATUSES=(open active completed failed archived investigating "in-progress" fixed closed)

ISSUE_DIR=""
ISSUE_METADATA_PATH=""
INVESTIGATION_JSON=""
INVESTIGATION_REPORT=""
INVESTIGATION_ROOT_CAUSE=""
INVESTIGATION_SUGGESTED_FIX=""
INVESTIGATION_CONFIDENCE=0
INVESTIGATION_AFFECTED_JSON="[]"
INVESTIGATION_STARTED_AT=""
INVESTIGATION_COMPLETED_AT=""
FIX_JSON=""
FIX_REPORT=""
FIX_SUMMARY=""
FIX_IMPLEMENTATION=""
FIX_TEST_PLAN=""
FIX_ROLLBACK=""
FIX_STATUS="skipped"
FIX_ERROR=""

# Colors for output (only used in TTY mode)
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Detect if stderr is a TTY (for color output)
USE_COLORS=false
if [[ -t 2 ]]; then
    USE_COLORS=true
fi

# Logging functions with conditional color support
log() {
    if [[ "$USE_COLORS" == "true" ]]; then
        echo -e "${BLUE}[$(date '+%Y-%m-%d %H:%M:%S')]${NC} $*" >&2
    else
        echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" >&2
    fi
}

error() {
    if [[ "$USE_COLORS" == "true" ]]; then
        echo -e "${RED}[ERROR]${NC} $*" >&2
    else
        echo "[ERROR] $*" >&2
    fi
}

success() {
    if [[ "$USE_COLORS" == "true" ]]; then
        echo -e "${GREEN}[SUCCESS]${NC} $*" >&2
    else
        echo "[SUCCESS] $*" >&2
    fi
}

warn() {
    if [[ "$USE_COLORS" == "true" ]]; then
        echo -e "${YELLOW}[WARN]${NC} $*" >&2
    else
        echo "[WARN] $*" >&2
    fi
}

DEFAULT_PROMPT_TEMPLATE="You are an elite software engineer responsible for full-cycle incident and improvements management on assigned scenarios or resources. You combine first-principles diagnostics with disciplined execution that preserves platform integrity and accelerates future agents."

load_default_prompt_template() {
    if [[ -f "$DEFAULT_PROMPT_FILE" ]]; then
        DEFAULT_PROMPT_TEMPLATE="$(cat "$DEFAULT_PROMPT_FILE")"
    else
        warn "Prompt file not found at $DEFAULT_PROMPT_FILE; using built-in fallback instructions"
    fi
}

load_default_prompt_template

# Load agent backend configuration
detect_agent_backend() {
    local preferred_provider=""
    local auto_fallback="true"
    local fallback_order=()

    # Try to load from agent-settings.json if available
    if [[ -f "$AGENT_SETTINGS" ]] && command -v jq >/dev/null 2>&1; then
        preferred_provider=$(jq -r '.agent_backend.provider // "codex"' "$AGENT_SETTINGS")
        auto_fallback=$(jq -r '.agent_backend.auto_fallback // true' "$AGENT_SETTINGS")

        # Read fallback order
        readarray -t fallback_order < <(jq -r '.agent_backend.fallback_order[]?' "$AGENT_SETTINGS")
    else
        # Default fallback order
        preferred_provider="codex"
        fallback_order=("codex" "claude-code")
    fi

    # Ensure preferred provider is tried first by reordering fallback array
    if [[ -n "$preferred_provider" ]]; then
        # Remove preferred provider from fallback list if it exists
        local temp_fallback=()
        for p in "${fallback_order[@]}"; do
            if [[ "$p" != "$preferred_provider" ]]; then
                temp_fallback+=("$p")
            fi
        done
        # Put preferred provider at the front
        fallback_order=("$preferred_provider" "${temp_fallback[@]}")
    fi

    # Try each provider in order
    for provider in "${fallback_order[@]}"; do
        local cli_cmd=""
        local operation_cmd=""

        # Try to read provider config from agent-settings.json
        if [[ -f "$AGENT_SETTINGS" ]] && command -v jq >/dev/null 2>&1; then
            cli_cmd=$(jq -r ".providers[\"$provider\"].cli_command // empty" "$AGENT_SETTINGS")
            operation_cmd=$(jq -r ".providers[\"$provider\"].operations.investigate.command // empty" "$AGENT_SETTINGS")
        fi

        # Fallback to defaults if not found in config
        if [[ -z "$cli_cmd" ]]; then
            case "$provider" in
                codex)
                    cli_cmd="resource-codex"
                    operation_cmd="run -"
                    ;;
                claude-code)
                    cli_cmd="resource-claude-code"
                    operation_cmd="run -"
                    ;;
                *)
                    continue
                    ;;
            esac
        fi

        # Check if CLI is available (don't check status - these are on-demand tools, not persistent services)
        if command -v "$cli_cmd" >/dev/null 2>&1; then
            AGENT_PROVIDER="$provider"
            AGENT_CLI_COMMAND="$cli_cmd"
            AGENT_OPERATION_CMD="$operation_cmd"
            log "Using agent backend: $provider ($cli_cmd)"
            return 0
        else
            warn "Agent backend '$provider' CLI not found: $cli_cmd"
        fi

        if [[ "$auto_fallback" != "true" ]]; then
            break
        fi
    done

    error "No available agent backend found. Tried: ${fallback_order[*]}"
    return 1
}

# Initialize agent backend
if ! detect_agent_backend; then
    error "Failed to initialize agent backend"
    exit 1
fi

# Load provider-specific config if available
if [[ "$AGENT_PROVIDER" == "codex" && -f "$CODEX_CONFIG" ]]; then
    export CODEX_CONFIG
fi

ensure_command() {
    local cmd="$1"
    if ! command -v "$cmd" >/dev/null 2>&1; then
        error "Required command '$cmd' not found in PATH"
        return 1
    fi
    return 0
}

find_issue_directory() {
    local root="$1"
    local issue_id="$2"

    for status in "${STATUSES[@]}"; do
        local candidate="${root%/}/${status}/${issue_id}"
        if [[ -d "$candidate" && -f "$candidate/metadata.yaml" ]]; then
            echo "$candidate"
            return 0
        fi
    done
    return 1
}

store_artifact() {
    local source_file="$1"
    local filename="$2"

    if [[ -z "$ISSUE_DIR" || ! -d "$ISSUE_DIR" ]]; then
        warn "Cannot store artifact because issue directory is unknown"
        return 0
    fi

    if [[ ! -f "$source_file" ]]; then
        warn "Artifact source '$source_file' missing"
        return 0
    fi

    local artifacts_dir="${ISSUE_DIR}/artifacts"
    mkdir -p "$artifacts_dir"
    cp "$source_file" "${artifacts_dir}/${filename}"
}

render_prompt_template() {
    local template="$1"
    local issue_title="$2"
    local issue_description="$3"
    local issue_type="$4"
    local issue_priority="$5"
    local app_name="$6"
    local error_message="$7"
    local stack_trace="$8"
    local affected_files="$9"
    local issue_metadata="${10}"
    local issue_artifacts="${11}"

    local fallback="(not provided)"

    [[ -z "${issue_title// }" ]] && issue_title="$fallback"
    [[ -z "${issue_description// }" ]] && issue_description="$fallback"
    [[ -z "${issue_type// }" ]] && issue_type="$fallback"
    [[ -z "${issue_priority// }" ]] && issue_priority="$fallback"
    [[ -z "${app_name// }" ]] && app_name="$fallback"
    [[ -z "${error_message// }" ]] && error_message="$fallback"
    [[ -z "${stack_trace// }" ]] && stack_trace="$fallback"
    [[ -z "${affected_files// }" ]] && affected_files="$fallback"
    [[ -z "${issue_metadata// }" ]] && issue_metadata="$fallback"
    [[ -z "${issue_artifacts// }" ]] && issue_artifacts="$fallback"

    template="${template//{{issue_title}}/$issue_title}"
    template="${template//{{issue_description}}/$issue_description}"
    template="${template//{{issue_type}}/$issue_type}"
    template="${template//{{issue_priority}}/$issue_priority}"
    template="${template//{{app_name}}/$app_name}"
    template="${template//{{error_message}}/$error_message}"
    template="${template//{{stack_trace}}/$stack_trace}"
    template="${template//{{affected_files}}/$affected_files}"
    template="${template//{{issue_metadata}}/$issue_metadata}"
    template="${template//{{issue_artifacts}}/$issue_artifacts}"

    echo "$template"
}

resolve_workflow() {
    local issue_id="$1"
    local agent_id="$2"
    local project_path="${3:-}"
    local prompt_template="${4:-$DEFAULT_PROMPT_TEMPLATE}"

    if [[ -z "$issue_id" ]]; then
        error "Issue ID required"
        exit 1
    fi

    ensure_command jq || exit 1

    local issues_root="${ISSUES_DIR:-}"
    if [[ -z "$issues_root" ]]; then
        if [[ -n "$project_path" ]]; then
            issues_root="${project_path%/}/data/issues"
        else
            issues_root="${SCRIPT_DIR}/../data/issues"
        fi
    fi

    if [[ -d "$issues_root" ]]; then
        if ISSUE_DIR=$(find_issue_directory "$issues_root" "$issue_id"); then
            ISSUE_METADATA_PATH="${ISSUE_DIR}/metadata.yaml"
        else
            warn "Issue directory not found in $issues_root; artifacts will not be stored"
            ISSUE_DIR=""
            ISSUE_METADATA_PATH=""
        fi
    else
        warn "Issues directory '$issues_root' not found; artifacts will not be stored"
        ISSUE_DIR=""
        ISSUE_METADATA_PATH=""
    fi

    # Run investigation (don't fail on non-zero exit, check result instead)
    # CRITICAL: Capture JSON output separately from logs (logs go to stderr via log() function)
    local investigation_json
    investigation_json=$(investigate_issue "$issue_id" "$agent_id" "$project_path" "$prompt_template" true)
    local invest_exit_code=$?

    # Extract max_turns_exceeded and error from investigation JSON
    local max_turns_exceeded=$(echo "$investigation_json" | jq -r '.max_turns_exceeded // false' 2>/dev/null || echo "false")
    local invest_error=$(echo "$investigation_json" | jq -r '.error // ""' 2>/dev/null || echo "")

    # Determine final status
    local run_status="completed"
    if [[ $invest_exit_code -ne 0 ]] && [[ "$max_turns_exceeded" != "true" ]]; then
        # Real failure (not just max turns)
        run_status="failed"
    fi

    local output=$(jq -n \
        --arg issue_id "$issue_id" \
        --arg agent_id "$agent_id" \
        --arg status "$run_status" \
        --arg error "$invest_error" \
        --argjson investigation "$investigation_json" \
        '{
            issue_id: $issue_id,
            agent_id: $agent_id,
            status: $status,
            error: (if $error != "" then $error else null end),
            investigation: $investigation
        }')

    echo "$output"
}

# Main investigation function
investigate_issue() {
    local issue_id="$1"
    local agent_id="$2"
    local project_path="${3:-}"
    local prompt_template="${4:-$DEFAULT_PROMPT_TEMPLATE}"
    local emit_json="${5:-true}"

    log "Starting investigation for issue: $issue_id"

    local started_at="$(date -Iseconds)"
    INVESTIGATION_STARTED_AT="$started_at"

    local title=""
    local description=""
    local issue_type=""
    local priority=""
    local app_id=""
    local error_message=""
    local stack_trace=""
    local affected_files=""
    local metadata_content=""
    local artifact_paths=""

    if [[ -n "$ISSUE_METADATA_PATH" && -f "$ISSUE_METADATA_PATH" ]]; then
        title=$(grep "^title:" "$ISSUE_METADATA_PATH" | head -1 | sed 's/title: *"\?\(.*\)"\?/\1/' | sed 's/"$//')
        description=$(grep "^description:" "$ISSUE_METADATA_PATH" | head -1 | sed 's/description: *"\?\(.*\)"\?/\1/' | sed 's/"$//')
        issue_type=$(grep "^type:" "$ISSUE_METADATA_PATH" | head -1 | sed 's/type: *"\?\(.*\)"\?/\1/' | sed 's/"$//')
        priority=$(grep "^priority:" "$ISSUE_METADATA_PATH" | head -1 | sed 's/priority: *"\?\(.*\)"\?/\1/' | sed 's/"$//')
        app_id=$(grep "^app_id:" "$ISSUE_METADATA_PATH" | head -1 | sed 's/app_id: *"\?\(.*\)"\?/\1/' | sed 's/"$//')
        metadata_content=$(cat "$ISSUE_METADATA_PATH")

        if grep -q "error_message:" "$ISSUE_METADATA_PATH"; then
            error_message=$(awk '/^error_context:/, /^[^[:space:]]/ {print}' "$ISSUE_METADATA_PATH" | grep "error_message:" | head -1 | sed 's/.*error_message: *"\?\(.*\)"\?/\1/' | sed 's/"$//')
        fi

        if grep -q "stack_trace:" "$ISSUE_METADATA_PATH"; then
            stack_trace=$(awk '/^error_context:/,/^[^[:space:]]/ {print}' "$ISSUE_METADATA_PATH" | awk '/stack_trace:/{flag=1; next} /^[^[:space:]]/{flag=0} flag' | sed 's/^[[:space:]]*//')
        fi

        if grep -q "affected_files:" "$ISSUE_METADATA_PATH"; then
            affected_files=$(sed -n '/affected_files:/,/^[[:space:]]*[a-z_]*:/p' "$ISSUE_METADATA_PATH" | grep ' - ' | sed 's/.*- "\(.*\)"/\1/' | tr '\n' ', ' | sed 's/, $//')
        fi
    fi

    if [[ -n "$ISSUE_DIR" ]]; then
        local artifacts_dir="${ISSUE_DIR}/artifacts"
        if [[ -d "$artifacts_dir" ]]; then
            while IFS= read -r artifact; do
                local relative_path="${artifact#"$ISSUE_DIR/"}"
                artifact_paths+="- $relative_path"$'\n'
            done < <(find "$artifacts_dir" -type f | sort)
            artifact_paths="${artifact_paths%$'\n'}"
        fi
    fi

    # Render the unified-resolver.md template with issue-specific data
    prompt_template=$(render_prompt_template "$prompt_template" "$title" "$description" "$issue_type" "$priority" "$app_id" "$error_message" "$stack_trace" "$affected_files" "$metadata_content" "$artifact_paths")

    # Create investigation workspace
    local workspace_dir="/tmp/codex-investigation-${issue_id}"
    mkdir -p "$workspace_dir"

    # Create investigation prompt file (unified-resolver.md already has all the structure)
    local prompt_file="${workspace_dir}/investigation-prompt.md"
    printf '%s\n' "$prompt_template" > "$prompt_file"

    log "Created investigation prompt: $prompt_file"

    # Verify agent backend is available
    if [[ -z "$AGENT_CLI_COMMAND" ]]; then
        error "No agent backend initialized. Please ensure resource-codex or resource-claude-code is available."
        return 1
    fi

    # Run investigation using configured backend
    log "Executing investigation with $AGENT_PROVIDER..."
    local investigation_output="${workspace_dir}/investigation-report.md"
    local agent_exit_code=0

    # Navigate to project path if specified
    local original_dir=$(pwd)
    if [[ -n "$project_path" && -d "$project_path" ]]; then
        log "Changing to project directory: $project_path"
        cd "$project_path"
    fi

    # Execute agent with the investigation prompt via stdin (matches ecosystem-manager pattern)
    local prompt_payload
    prompt_payload="$(cat "$prompt_file")"

    # Load settings from agent-settings.json if available (similar to ecosystem-manager)
    local max_turns=80
    local allowed_tools="Read,Write,Edit,Bash,LS,Glob,Grep"
    local skip_permissions="yes"
    local timeout_seconds=600

    if [[ -f "$AGENT_SETTINGS" ]] && command -v jq >/dev/null 2>&1; then
        # Try to load provider-specific settings
        local provider_max_turns=$(jq -r ".providers[\"$AGENT_PROVIDER\"].operations.investigate.max_turns // empty" "$AGENT_SETTINGS" 2>/dev/null)
        local provider_allowed_tools=$(jq -r ".providers[\"$AGENT_PROVIDER\"].operations.investigate.allowed_tools // empty" "$AGENT_SETTINGS" 2>/dev/null)
        local provider_timeout=$(jq -r ".providers[\"$AGENT_PROVIDER\"].operations.investigate.timeout_seconds // empty" "$AGENT_SETTINGS" 2>/dev/null)

        # Use provider-specific settings if available
        [[ -n "$provider_max_turns" ]] && max_turns="$provider_max_turns"
        [[ -n "$provider_allowed_tools" ]] && allowed_tools="$provider_allowed_tools"
        [[ -n "$provider_timeout" ]] && timeout_seconds="$provider_timeout"
    fi

    # Set environment variables for resource-claude-code (ecosystem-manager pattern)
    # Go uses: cmd.Env = append(os.Environ(), "MAX_TURNS=...", ...)
    # Bash equivalent: export the vars then call the command
    export MAX_TURNS="$max_turns"
    export ALLOWED_TOOLS="$allowed_tools"
    export TIMEOUT="$timeout_seconds"
    export SKIP_PERMISSIONS="$skip_permissions"

    log "Agent execution settings: MAX_TURNS=$MAX_TURNS, ALLOWED_TOOLS=$ALLOWED_TOOLS, TIMEOUT=${TIMEOUT}s"

    # Execute agent and capture exit code (disable errexit for this command)
    # CRITICAL: Pipe prompt via stdin with '-' argument (ecosystem-manager pattern)
    set +e
    $AGENT_CLI_COMMAND $AGENT_OPERATION_CMD < "$prompt_file" > "$investigation_output" 2>&1
    agent_exit_code=$?
    set -e

    if [[ $agent_exit_code -eq 0 ]]; then
        success "$AGENT_PROVIDER investigation completed successfully"
    else
        error "$AGENT_PROVIDER investigation failed with exit code: $agent_exit_code"
        # Don't fail yet - check if it's a max turns error below
    fi

    cd "$original_dir"

    local completed_at="$(date -Iseconds)"
    INVESTIGATION_COMPLETED_AT="$completed_at"

    # Read the investigation report (no parsing - just store the full AI response)
    local report_content
    if [[ -f "$investigation_output" ]]; then
        report_content=$(cat "$investigation_output")
        log "Investigation report generated (${#report_content} characters)"
    else
        error "Investigation output file not found"
        return 1
    fi

    INVESTIGATION_REPORT="$report_content"
    INVESTIGATION_COMPLETED_AT="${INVESTIGATION_COMPLETED_AT:-$(date -Iseconds)}"

    # CRITICAL: resource-claude-code exit codes cannot be trusted (ecosystem-manager lesson)
    # The agent may exit non-zero even when it successfully completes its work:
    # - Test commands that fail during investigation
    # - Permission warnings on non-critical files
    # - Shell error handling for edge cases
    # - Timeout signals after agent finishes useful work
    #
    # Solution: Check output QUALITY, not exit codes. If the agent produced a substantial,
    # structured report, treat as success regardless of exit code.

    local has_substantial_output="false"

    # Check for structured sections from unified-resolver.md prompt (NO size gate - ecosystem-manager pattern)
    local lower_output=$(echo "$report_content" | tr '[:upper:]' '[:lower:]')
    if echo "$lower_output" | grep -q "investigation summary" || \
       echo "$lower_output" | grep -q "root cause" || \
       echo "$lower_output" | grep -q "remediation" || \
       echo "$lower_output" | grep -q "validation plan" || \
       echo "$lower_output" | grep -q "confidence assessment" || \
       echo "$lower_output" | grep -q "implementation steps"; then
        has_substantial_output="true"
        if [[ $agent_exit_code -ne 0 ]]; then
            warn "Agent exited with code $agent_exit_code but produced structured report - treating as success"
        fi
    # Fallback: very large output is probably valid even without specific markers
    elif [[ ${#report_content} -gt 2000 ]]; then
        has_substantial_output="true"
        warn "Agent produced large output (${#report_content} chars) without specific markers - treating as success"
    fi

    # Detect max turns exceeded (ecosystem-manager pattern)
    local max_turns_exceeded="false"
    local error_message=""
    local timeout_exceeded="false"

    if [[ $agent_exit_code -ne 0 ]]; then
        local lower_output=$(echo "$report_content" | tr '[:upper:]' '[:lower:]')

        # Exit code 143 = SIGTERM (128 + 15), usually from timeout
        # Exit code 124 = timeout command's own timeout exit code
        if [[ $agent_exit_code -eq 143 ]] || [[ $agent_exit_code -eq 124 ]]; then
            timeout_exceeded="true"
            # Check if we have substantial output despite timeout
            if [[ ${#report_content} -gt 1000 ]] || [[ "$has_substantial_output" == "true" ]]; then
                warn "Agent hit timeout limit but produced substantial output (${#report_content} chars)"
                error_message="Agent execution exceeded timeout but completed useful work. Consider increasing timeout in Settings for complex tasks."
                # Treat as success since we have structured output
                agent_exit_code=0
            else
                error_message="Agent execution timed out without producing sufficient output"
            fi
        elif echo "$lower_output" | grep -q "max turns" && echo "$lower_output" | grep -q "reached"; then
            max_turns_exceeded="true"
            error_message="Claude reached the configured MAX_TURNS limit. The output may contain partial results. Consider increasing max_turns in Settings or simplifying the task."
            warn "$error_message"
        else
            error_message="Agent execution failed with exit code $agent_exit_code"
        fi
    fi

    local json_output=$(jq -n \
        --arg report "$INVESTIGATION_REPORT" \
        --arg started "$INVESTIGATION_STARTED_AT" \
        --arg completed "$INVESTIGATION_COMPLETED_AT" \
        --arg workspace "$workspace_dir" \
        --arg max_turns "$max_turns_exceeded" \
        --arg error "$error_message" \
        --argjson exit_code "$agent_exit_code" \
        '{
            report: $report,
            started_at: $started,
            completed_at: $completed,
            status: "analysis_completed",
            workspace_path: $workspace,
            max_turns_exceeded: ($max_turns == "true"),
            error: (if $error != "" then $error else null end),
            exit_code: $exit_code
        }')

    INVESTIGATION_JSON="$json_output"
    store_artifact "$investigation_output" "investigation-report.md"

    if [[ "$emit_json" == "true" ]]; then
        echo "$json_output"
    fi

    # CRITICAL: Check output quality, not exit codes (ecosystem-manager pattern)
    # resource-claude-code exit codes are unreliable - check if investigation succeeded
    if [[ $agent_exit_code -eq 0 ]] || [[ "$has_substantial_output" == "true" ]]; then
        success "Investigation completed for issue: $issue_id"
        return 0
    else
        # Only fail if BOTH non-zero exit AND no substantial output
        error "Investigation failed: no structured report and non-zero exit code ($agent_exit_code)"
        return $agent_exit_code
    fi
}

# Main function
main() {
    local command="${1:-}"
    shift || true

    case "$command" in
        resolve)
            local issue_id="${1:-}"
            local agent_id="${2:-}"
            local project_path="${3:-}"
            local prompt_template="${4:-Perform a full investigation and provide code-level remediation guidance.}"

            if [[ -z "$issue_id" ]]; then
                error "Issue ID required"
                exit 1
            fi

            resolve_workflow "$issue_id" "$agent_id" "$project_path" "$prompt_template"
            ;;

        help|--help|-h)
            cat << EOF
AI Agent Investigation Script

Usage: $0 resolve <issue_id> <agent_id> [project_path] [prompt_template]

Supports: resource-codex, resource-claude-code (auto-detected)
Current backend: ${AGENT_PROVIDER:-not initialized}

Arguments:
  issue_id         - Required: Issue identifier
  agent_id         - Required: Agent identifier
  project_path     - Optional: Path to project directory
  prompt_template  - Optional: Custom prompt template (default: unified-resolver.md)

Note: unified-resolver.md includes both investigation and fix recommendations in a single AI call.

Examples:
  $0 resolve "issue-123" "unified-resolver" "/path/to/project"
  $0 resolve "issue-456" "unified-resolver" "/path/to/project" "Custom prompt"
EOF
            ;;

        *)
            error "Unknown command: $command"
            echo "Usage: $0 resolve <issue_id> <agent_id> [project_path] [prompt_template]"
            echo "Run '$0 help' for more information"
            exit 1
            ;;
    esac
}

# Run main function
main "$@"
