#!/usr/bin/env bash

# Claude Code Investigation Script
# Direct interface to Claude Code CLI for issue investigation

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIG_DIR="${SCRIPT_DIR}/../initialization/configuration"
CLAUDE_CONFIG="${CONFIG_DIR}/claude-config.json"

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

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging function
log() {
    echo -e "${BLUE}[$(date '+%Y-%m-%d %H:%M:%S')]${NC} $*" >&2
}

error() {
    echo -e "${RED}[ERROR]${NC} $*" >&2
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $*" >&2
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $*" >&2
}

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

extract_section() {
    local heading="$1"
    local content="$2"

    printf '%s\n' "$content" | awk -v heading="$heading" '
        $0 ~ heading {flag=1; next}
        /^### / && flag {exit}
        flag {print}
    '
}

resolve_workflow() {
    local issue_id="$1"
    local agent_id="$2"
    local project_path="${3:-}"
    local prompt_template="${4:-Perform a full investigation and provide code-level remediation guidance.}"
    local auto_flag="${5:-true}"

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

    if ! investigate_issue "$issue_id" "$agent_id" "$project_path" "$prompt_template" false; then
        error "Investigation failed"
        exit 1
    fi

    local auto_resolve="true"
    if [[ "$auto_flag" =~ ^(false|0|no|analysis)$ ]]; then
        auto_resolve="false"
    fi

    local run_status="analysis_complete"
    local error_message=""

    if [[ "$auto_resolve" == "true" ]]; then
        if generate_fix "$issue_id" "$INVESTIGATION_REPORT" "$project_path" false; then
            run_status="completed"
        else
            run_status="failed"
            error_message="${FIX_ERROR:-Fix generation failed}"
            if [[ -z "$FIX_JSON" ]]; then
                FIX_JSON=$(jq -n --arg report "$FIX_REPORT" --arg err "$error_message" '{status: "failed", report: ($report // ""), error: $err}')
            fi
        fi
    else
        FIX_STATUS="skipped"
        FIX_JSON=$(jq -n '{status: "skipped"}')
    fi

    if [[ "$FIX_STATUS" == "generated" ]]; then
        : # FIX_JSON already populated by generate_fix
    elif [[ -z "$FIX_JSON" ]]; then
        FIX_JSON=$(jq -n --arg status "$FIX_STATUS" '{status: $status}')
    fi

    local investigation_json="${INVESTIGATION_JSON:-"{}"}"
    local fix_json="${FIX_JSON:-"{}"}"

    local output=$(jq -n \
        --arg issue_id "$issue_id" \
        --arg agent_id "$agent_id" \
        --arg status "$run_status" \
        --arg error "$error_message" \
        --arg auto "$auto_resolve" \
        --argjson investigation "$investigation_json" \
        --argjson fix "$fix_json" \
        '{
            issue_id: $issue_id,
            agent_id: $agent_id,
            status: $status,
            auto_resolve: ($auto == "true"),
            error: (if ($error | length) > 0 then $error else null end),
            investigation: $investigation,
            fix: $fix
        }')

    echo "$output"
}

# Main investigation function
investigate_issue() {
    local issue_id="$1"
    local agent_id="$2"
    local project_path="${3:-}"
    local prompt_template="${4:-}"
    local emit_json="${5:-true}"

    log "Starting investigation for issue: $issue_id"

    local started_at="$(date -Iseconds)"
    INVESTIGATION_STARTED_AT="$started_at"
    
    # Create investigation workspace
    local workspace_dir="/tmp/claude-investigation-${issue_id}"
    mkdir -p "$workspace_dir"
    
    # Create investigation prompt file
    local prompt_file="${workspace_dir}/investigation-prompt.md"
    
    cat > "$prompt_file" << EOF
# Issue Investigation Request

You are a senior software engineer investigating a reported issue. Please analyze the codebase and provide a comprehensive investigation report.

## Task
$prompt_template

## Investigation Steps
1. **Analyze the codebase** at the specified path
2. **Identify the root cause** of the issue
3. **Examine related files** and dependencies
4. **Check for similar patterns** in the codebase
5. **Provide actionable recommendations**

## Expected Output Format
Please structure your response as follows:

### Investigation Summary
Brief overview of the issue and investigation approach.

### Root Cause Analysis
Detailed explanation of what is causing the issue.

### Affected Components
List of files, functions, or systems impacted.

### Recommended Solutions
Prioritized list of potential fixes with implementation details.

### Testing Strategy
How to verify the fix and prevent regression.

### Related Issues
Any similar issues or patterns to watch for.

### Confidence Level
Rate your confidence in the analysis (1-10) and explain any uncertainties.

## Context
- Issue ID: $issue_id
- Agent ID: $agent_id
- Project Path: $project_path
- Timestamp: $(date -Iseconds)

Please begin your investigation now.
EOF

    log "Created investigation prompt: $prompt_file"
    
    # Check if resource-claude-code is available
    if ! command -v resource-claude-code &> /dev/null; then
        error "resource-claude-code CLI not found. Please ensure the resource is installed."
        return 1
    fi
    
    # Check resource status
    log "Checking Claude Code resource status..."
    if ! resource-claude-code status &> /dev/null; then
        warn "Claude Code resource may not be running. Attempting to start..."
        resource-claude-code start &> /dev/null || true
    fi
    
    # Run Claude Code investigation
    log "Executing Claude Code investigation..."
    local investigation_output="${workspace_dir}/investigation-report.md"
    local claude_exit_code=0
    
    # Navigate to project path if specified
    local original_dir=$(pwd)
    if [[ -n "$project_path" && -d "$project_path" ]]; then
        log "Changing to project directory: $project_path"
        cd "$project_path"
    fi
    
    # Execute Claude Code with the investigation prompt using resource-claude-code run
    if cat "$prompt_file" | resource-claude-code run - > "$investigation_output" 2>&1; then
        success "Claude Code investigation completed successfully"
    else
        claude_exit_code=$?
        error "Claude Code investigation failed with exit code: $claude_exit_code"
        # Try to show the error content for debugging
        if [[ -f "$investigation_output" ]]; then
            warn "Error output: $(head -n 20 "$investigation_output")"
        fi
        cd "$original_dir"
        return $claude_exit_code
    fi
    
    cd "$original_dir"

    local completed_at="$(date -Iseconds)"
    INVESTIGATION_COMPLETED_AT="$completed_at"
    
    # Parse the investigation report
    local report_content
    if [[ -f "$investigation_output" ]]; then
        report_content=$(cat "$investigation_output")
        log "Investigation report generated (${#report_content} characters)"
    else
        error "Investigation output file not found"
        return 1
    fi
    
    # Extract structured data from the report
    local root_cause=""
    local suggested_fix=""
    local confidence_score=""
    local affected_files=""

    if echo "$report_content" | grep -q "### Root Cause Analysis"; then
        root_cause=$(echo "$report_content" | sed -n '/### Root Cause Analysis/,/###/p' | head -n -1 | tail -n +2 | tr '\n' ' ')
    fi

    if echo "$report_content" | grep -q "### Recommended Solutions"; then
        suggested_fix=$(echo "$report_content" | sed -n '/### Recommended Solutions/,/###/p' | head -n -1 | tail -n +2 | tr '\n' ' ')
    fi

    if echo "$report_content" | grep -qE "confidence.*[0-9]+"; then
        confidence_score=$(echo "$report_content" | grep -oE "confidence.*([0-9]+)" | grep -oE "[0-9]+" | head -1)
    fi

    if echo "$report_content" | grep -q "### Affected Components"; then
        affected_files=$(echo "$report_content" | sed -n '/### Affected Components/,/###/p' | head -n -1 | tail -n +2 | grep -oE '[a-zA-Z0-9/_.-]+\.(js|ts|py|go|java|cpp|c|h)' | head -10 | tr '\n' ',' | sed 's/,$//')
    fi

    local affected_json="[]"
    if [[ -n "$affected_files" ]]; then
        affected_json="[$(echo "$affected_files" | sed 's/,/","/g' | sed 's/^/"/' | sed 's/$/"/' | sed 's/,"$/"/') ]"
    fi

    local confidence_value=${confidence_score:-5}

    INVESTIGATION_REPORT="$report_content"
    INVESTIGATION_ROOT_CAUSE="${root_cause:-Unknown}"
    INVESTIGATION_SUGGESTED_FIX="${suggested_fix:-No specific fix identified}"
    INVESTIGATION_CONFIDENCE=$confidence_value
    INVESTIGATION_AFFECTED_JSON="$affected_json"
    INVESTIGATION_COMPLETED_AT="${INVESTIGATION_COMPLETED_AT:-$(date -Iseconds)}"

    local json_output=$(jq -n \
        --arg issue_id "$issue_id" \
        --arg agent_id "$agent_id" \
        --arg report "$INVESTIGATION_REPORT" \
        --arg root "$INVESTIGATION_ROOT_CAUSE" \
        --arg suggested "$INVESTIGATION_SUGGESTED_FIX" \
        --arg started "$INVESTIGATION_STARTED_AT" \
        --arg completed "$INVESTIGATION_COMPLETED_AT" \
        --arg workspace "$workspace_dir" \
        --argjson affected "$INVESTIGATION_AFFECTED_JSON" \
        --argjson confidence "$confidence_value" \
        '{
            issue_id: $issue_id,
            agent_id: $agent_id,
            investigation_report: $report,
            root_cause: $root,
            suggested_fix: $suggested,
            confidence_score: $confidence,
            affected_files: $affected,
            investigation_started_at: $started,
            investigation_completed_at: $completed,
            status: "analysis_completed",
            workspace_path: $workspace
        }')

    INVESTIGATION_JSON="$json_output"
    store_artifact "$investigation_output" "investigation-report.md"

    if [[ "$emit_json" == "true" ]]; then
        echo "$json_output"
    fi

    success "Investigation completed for issue: $issue_id"
    return 0
}

# Generate investigation fix
generate_fix() {
    local issue_id="$1"
    local investigation_report="$2"
    local project_path="${3:-}"
    local emit_json="${4:-true}"

    log "Generating fix for issue: $issue_id"

    local workspace_dir="/tmp/claude-fix-${issue_id}"
    mkdir -p "$workspace_dir"

    local fix_prompt="${workspace_dir}/fix-prompt.md"

    cat > "$fix_prompt" << EOF
# Unified Fix Generation Request

Based on the investigation summary below, deliver concrete remediation steps.

## Investigation Summary
$investigation_report

## Tasks
1. Summarize the proposed fix in plain language.
2. Outline implementation steps with key files and functions.
3. Recommend automated and manual tests to validate the change.
4. Provide a rollback plan if the change must be reverted.

## Output Structure
Produce markdown sections with the following headings:
- ### Summary
- ### Implementation Steps
- ### Test Plan
- ### Rollback Plan
- ### Notes (optional)

Be precise and reference relevant files and functions where possible.
EOF

    local fix_output="${workspace_dir}/fix-report.md"
    local original_dir=$(pwd)

    if [[ -n "$project_path" && -d "$project_path" ]]; then
        cd "$project_path"
    fi

    if ! command -v resource-claude-code &> /dev/null; then
        error "resource-claude-code CLI not found"
        cd "$original_dir"
        return 1
    fi

    if ! cat "$fix_prompt" | resource-claude-code run - > "$fix_output" 2>&1; then
        FIX_STATUS="failed"
        FIX_ERROR="Claude Code fix generation failed"
        if [[ -f "$fix_output" ]]; then
            warn "Error output: $(head -n 20 "$fix_output")"
        fi
        cd "$original_dir"
        return 1
    fi

    cd "$original_dir"

    if [[ ! -f "$fix_output" ]]; then
        FIX_STATUS="failed"
        FIX_ERROR="Fix output file not found"
        return 1
    fi

    local fix_content
    fix_content=$(cat "$fix_output")

    local summary=$(extract_section '### Summary' "$fix_content")
    if [[ -z "$summary" ]]; then
        summary=$(head -n 40 "$fix_output")
    fi

    local implementation=$(extract_section '### Implementation Steps' "$fix_content")
    local test_plan=$(extract_section '### Test Plan' "$fix_content")
    local rollback_plan=$(extract_section '### Rollback Plan' "$fix_content")

    FIX_REPORT="$fix_content"
    FIX_SUMMARY="${summary:-Proposed fix details not provided.}"
    FIX_IMPLEMENTATION="${implementation:-Implementation steps not provided.}"
    FIX_TEST_PLAN="${test_plan:-Test plan not provided.}"
    FIX_ROLLBACK="${rollback_plan:-Rollback plan not provided.}"
    FIX_STATUS="generated"
    FIX_ERROR=""

    local fix_json=$(jq -n \
        --arg issue_id "$issue_id" \
        --arg report "$FIX_REPORT" \
        --arg summary "$FIX_SUMMARY" \
        --arg implementation "$FIX_IMPLEMENTATION" \
        --arg test_plan "$FIX_TEST_PLAN" \
        --arg rollback "$FIX_ROLLBACK" \
        --arg workspace "$workspace_dir" \
        '{
            issue_id: $issue_id,
            fix_report: $report,
            summary: $summary,
            implementation_plan: $implementation,
            test_plan: $test_plan,
            rollback_plan: $rollback,
            status: "generated",
            workspace_path: $workspace,
            fix_generated_at: (now | todateiso8601)
        }')

    FIX_JSON="$fix_json"
    store_artifact "$fix_output" "fix-report.md"

    if [[ "$emit_json" == "true" ]]; then
        echo "$fix_json"
    fi

    success "Fix generation completed for issue: $issue_id"
    return 0
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
            local auto_flag="${5:-true}"

            if [[ -z "$issue_id" ]]; then
                error "Issue ID required"
                exit 1
            fi

            resolve_workflow "$issue_id" "$agent_id" "$project_path" "$prompt_template" "$auto_flag"
            ;;

        investigate)
            local issue_id="${1:-}"
            local agent_id="${2:-}"
            local project_path="${3:-}"
            local prompt_template="${4:-Investigate this issue and provide a detailed analysis.}"
            
            if [[ -z "$issue_id" ]]; then
                error "Issue ID required"
                exit 1
            fi
            
            investigate_issue "$issue_id" "$agent_id" "$project_path" "$prompt_template"
            ;;
            
        generate-fix)
            local issue_id="${1:-}"
            local investigation_report="${2:-}"
            local project_path="${3:-}"
            
            if [[ -z "$issue_id" ]]; then
                error "Issue ID required"
                exit 1
            fi
            
            generate_fix "$issue_id" "$investigation_report" "$project_path"
            ;;
            
        help|--help|-h)
            cat << EOF
Claude Code Investigation Script

Usage: $0 <command> [options]

Commands:
  resolve <issue_id> <agent_id> [project_path] [prompt_template] [auto_resolve]
    Run unified investigation + resolution (auto_resolve defaults to true)

  investigate <issue_id> <agent_id> [project_path] [prompt_template]
    Run Claude Code investigation for an issue
    
  generate-fix <issue_id> <investigation_report> [project_path]
    Generate code fixes based on investigation
    
  help
    Show this help message

Examples:
  $0 investigate "issue-123" "agent-001" "/path/to/project" "Analyze login error"
  $0 generate-fix "issue-123" "Root cause: null pointer exception..."
EOF
            ;;
            
        *)
            error "Unknown command: $command"
            echo "Use '$0 help' for usage information"
            exit 1
            ;;
    esac
}

# Run main function
main "$@"
