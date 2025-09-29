#!/usr/bin/env bash

# File-Based Claude Code Investigation Script
# Reads YAML issue files, investigates with Claude, writes results back

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ISSUES_DIR="${SCRIPT_DIR}/../data/issues"

echo "[WARN] The file-based investigator script requires updates for the new folder-per-issue storage." >&2
echo "       Please trigger investigations through the API/CLI for now." >&2
exit 1

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log() { echo -e "${BLUE}[$(date '+%Y-%m-%d %H:%M:%S')]${NC} $*" >&2; }
error() { echo -e "${RED}[ERROR]${NC} $*" >&2; }
success() { echo -e "${GREEN}[SUCCESS]${NC} $*" >&2; }
warn() { echo -e "${YELLOW}[WARN]${NC} $*" >&2; }

# Find issue file by ID across all folders
find_issue_file() {
    local issue_id="$1"
    
    for folder in open active completed failed archived investigating in-progress fixed closed; do
        for file in "$ISSUES_DIR/$folder"/*.yaml; do
            if [[ -f "$file" ]]; then
                local file_issue_id=$(grep "^id:" "$file" | sed 's/id: *//')
                if [[ "$file_issue_id" == "$issue_id" ]]; then
                    echo "$file"
                    return 0
                fi
            fi
        done
    done
    
    return 1
}

# Main investigation function
investigate_issue() {
    local issue_id="$1"
    local agent_id="${2:-unified-resolver}"
    local project_path="${3:-$(pwd)}"
    local custom_prompt="${4:-}"
    
    log "Starting file-based investigation for issue: $issue_id"
    
    # Find issue file
    local issue_file
    if ! issue_file=$(find_issue_file "$issue_id"); then
        error "Issue file not found for ID: $issue_id"
        return 1
    fi
    
    log "Found issue file: $issue_file"
    
    # Read issue data
    local title=$(grep "^title:" "$issue_file" | sed 's/title: *"\?\(.*\)"\?/\1/' | sed 's/"$//')
    local description=$(grep "^description:" "$issue_file" | sed 's/description: *"\?\(.*\)"\?/\1/' | sed 's/"$//')
    local type=$(grep "^type:" "$issue_file" | sed 's/type: *//')
    local priority=$(grep "^priority:" "$issue_file" | sed 's/priority: *//')
    local app_id=$(grep "^app_id:" "$issue_file" | sed 's/app_id: *"\?\(.*\)"\?/\1/' | sed 's/"$//')
    local error_message=$(grep "error_message:" "$issue_file" | head -1 | sed 's/.*error_message: *"\?\(.*\)"\?/\1/' | sed 's/"$//')
    
    # Extract stack trace (multiline)
    local stack_trace=""
    if grep -q "stack_trace: |" "$issue_file"; then
        stack_trace=$(sed -n '/stack_trace: |/,/^[[:space:]]*[a-z_]*:/p' "$issue_file" | sed '1d;$d' | sed 's/^[[:space:]]*//')
    fi
    
    # Extract affected files
    local affected_files=""
    if grep -q "affected_files:" "$issue_file"; then
        affected_files=$(sed -n '/affected_files:/,/^[[:space:]]*[a-z_]*:/p' "$issue_file" | grep "- \"" | sed 's/.*- "\(.*\)"/\1/' | tr '\n' ',' | sed 's/,$//')
    fi
    
    # Create investigation workspace
    local workspace_dir="/tmp/file-investigation-${issue_id}"
    mkdir -p "$workspace_dir"
    
    # Create comprehensive investigation prompt
    local prompt_file="${workspace_dir}/investigation-prompt.md"
    
    cat > "$prompt_file" << EOF
# File-Based Issue Investigation

## Issue Information
- **ID:** $issue_id
- **Title:** $title
- **Type:** $type
- **Priority:** $priority
- **Application:** $app_id

## Description
$description

$(if [[ -n "$error_message" ]]; then echo "## Error Message"; echo "$error_message"; echo; fi)

$(if [[ -n "$stack_trace" ]]; then echo "## Stack Trace"; echo '```'; echo "$stack_trace"; echo '```'; echo; fi)

$(if [[ -n "$affected_files" ]]; then echo "## Affected Files"; echo "$affected_files" | tr ',' '\n' | sed 's/^/- /'; echo; fi)

## Investigation Request

${custom_prompt:-Please perform a thorough investigation of this issue:}

1. **Analyze the codebase** at: $project_path
2. **Identify the root cause** of this $type
3. **Examine affected files** and related components
4. **Check for similar patterns** in the codebase
5. **Provide actionable recommendations** with implementation details

## Required Output Format

Please structure your response as follows:

### Investigation Summary
Brief overview of your findings and approach.

### Root Cause Analysis
Detailed explanation of what is causing the issue.

### Affected Components  
List of files, functions, or systems impacted.

### Recommended Solutions
Prioritized list of potential fixes with specific implementation steps.

### Testing Strategy
How to verify the fix works and prevent regression.

### Related Issues
Any similar issues or patterns found in the codebase.

### Confidence Level
Rate your confidence in this analysis (1-10) and explain uncertainties.

## Project Context
- Project Path: $project_path
- Investigation Agent: $agent_id
- Timestamp: $(date -Iseconds)

Please begin your investigation now.
EOF

    log "Created investigation prompt: $prompt_file"
    
    # Check if claude-code is available
    if ! command -v claude-code &> /dev/null; then
        error "claude-code CLI not found. Please install Claude Code."
        return 1
    fi
    
    # Execute Claude Code investigation
    log "Executing Claude Code investigation..."
    local investigation_output="${workspace_dir}/investigation-report.md"
    local original_dir=$(pwd)
    
    # Navigate to project path
    if [[ -n "$project_path" && -d "$project_path" ]]; then
        log "Investigating in directory: $project_path"
        cd "$project_path"
    fi
    
    # Run Claude Code
    local claude_exit_code=0
    if claude-code --file "$prompt_file" > "$investigation_output" 2>&1; then
        success "Claude Code investigation completed"
    else
        claude_exit_code=$?
        error "Claude Code investigation failed with exit code: $claude_exit_code"
        cd "$original_dir"
        return $claude_exit_code
    fi
    
    cd "$original_dir"
    
    # Read investigation results
    local report_content=""
    if [[ -f "$investigation_output" ]]; then
        report_content=$(cat "$investigation_output")
        log "Investigation report generated (${#report_content} characters)"
    else
        error "Investigation output file not found"
        return 1
    fi
    
    # Parse structured data from report
    local root_cause=""
    local suggested_fix=""
    local confidence_score=""
    local investigation_summary=""
    
    # Extract sections using more robust parsing
    if echo "$report_content" | grep -q "### Root Cause Analysis"; then
        root_cause=$(echo "$report_content" | sed -n '/### Root Cause Analysis/,/###/p' | sed '1d;$d' | tr '\n' ' ' | sed 's/^[[:space:]]*//' | sed 's/[[:space:]]*$//')
    fi
    
    if echo "$report_content" | grep -q "### Recommended Solutions"; then
        suggested_fix=$(echo "$report_content" | sed -n '/### Recommended Solutions/,/###/p' | sed '1d;$d' | tr '\n' ' ' | sed 's/^[[:space:]]*//' | sed 's/[[:space:]]*$//')
    fi
    
    if echo "$report_content" | grep -q "### Investigation Summary"; then
        investigation_summary=$(echo "$report_content" | sed -n '/### Investigation Summary/,/###/p' | sed '1d;$d' | tr '\n' ' ' | sed 's/^[[:space:]]*//' | sed 's/[[:space:]]*$//')
    fi
    
    # Extract confidence score
    if echo "$report_content" | grep -qiE "confidence.*[0-9]+"; then
        confidence_score=$(echo "$report_content" | grep -oiE "confidence[^0-9]*([0-9]+)" | grep -oE "[0-9]+" | head -1)
    fi
    
    # Update the issue file directly
    local current_folder=$(dirname "$issue_file")
    current_folder=$(basename "$current_folder")
    
    # Create temporary updated file
    local temp_file="${workspace_dir}/updated-issue.yaml"
    cp "$issue_file" "$temp_file"
    
    # Update investigation section
    local now=$(date -Iseconds)
    
    # Update investigation fields
    sed -i "s/agent_id: \"\"/agent_id: \"$agent_id\"/g" "$temp_file"
    sed -i "s/completed_at: \"\"/completed_at: \"$now\"/g" "$temp_file"
    sed -i "s/updated_at: \".*\"/updated_at: \"$now\"/g" "$temp_file"
    
    # Update investigation report (escape for sed)
    local escaped_report=$(echo "$report_content" | sed 's/\\/\\\\/g' | sed 's/"/\\"/g' | sed 's/$/\\n/' | tr -d '\n' | sed 's/\\n$//')
    sed -i "s/report: \"\"/report: \"$escaped_report\"/g" "$temp_file"
    
    # Update root cause
    if [[ -n "$root_cause" ]]; then
        local escaped_cause=$(echo "$root_cause" | sed 's/\\/\\\\/g' | sed 's/"/\\"/g')
        sed -i "s/root_cause: \"\"/root_cause: \"$escaped_cause\"/g" "$temp_file"
    fi
    
    # Update suggested fix
    if [[ -n "$suggested_fix" ]]; then
        local escaped_fix=$(echo "$suggested_fix" | sed 's/\\/\\\\/g' | sed 's/"/\\"/g')
        sed -i "s/suggested_fix: \"\"/suggested_fix: \"$escaped_fix\"/g" "$temp_file"
    fi
    
    # Update confidence score
    if [[ -n "$confidence_score" ]]; then
        sed -i "s/confidence_score: null/confidence_score: $confidence_score/g" "$temp_file"
    fi
    
    # Move back to original location with updates
    mv "$temp_file" "$issue_file"
    
    # Generate JSON output for API compatibility
    local json_output=$(cat << EOF
{
  "issue_id": "$issue_id",
  "agent_id": "$agent_id",
  "investigation_report": $(echo "$report_content" | jq -R -s '.'),
  "root_cause": $(echo "${root_cause:-Unknown root cause}" | jq -R -s '.'),
  "suggested_fix": $(echo "${suggested_fix:-No specific fix identified}" | jq -R -s '.'),
  "confidence_score": ${confidence_score:-5},
  "affected_files": [$(if [[ -n "$affected_files" ]]; then echo "$affected_files" | sed 's/,/","/g' | sed 's/^/"/' | sed 's/$/"/'; fi)],
  "investigation_completed_at": "$now",
  "status": "completed",
  "workspace_path": "$workspace_dir",
  "file_path": "$issue_file",
  "mode": "file-based"
}
EOF
)
    
    # Output JSON result
    echo "$json_output"
    
    success "File-based investigation completed for issue: $issue_id"
    success "Updated issue file: $(basename "$issue_file")"
    
    return 0
}

# Generate fix from investigation
generate_fix() {
    local issue_id="$1"
    local project_path="${2:-$(pwd)}"
    
    log "Generating fix for issue: $issue_id"
    
    # Find issue file
    local issue_file
    if ! issue_file=$(find_issue_file "$issue_id"); then
        error "Issue file not found for ID: $issue_id"
        return 1
    fi
    
    # Read investigation data
    local investigation_report=""
    if grep -q "report:" "$issue_file"; then
        investigation_report=$(sed -n '/report: |/,/^[[:space:]]*[a-z_]*:/p' "$issue_file" | sed '1d;$d' | sed 's/^[[:space:]]*//')
    fi
    
    if [[ -z "$investigation_report" ]]; then
        error "No investigation report found. Run investigation first."
        return 1
    fi
    
    # Create fix workspace
    local workspace_dir="/tmp/file-fix-${issue_id}"
    mkdir -p "$workspace_dir"
    
    # Create fix generation prompt
    local fix_prompt="${workspace_dir}/fix-prompt.md"
    
    cat > "$fix_prompt" << EOF
# Fix Generation Request

Generate specific code fixes based on the investigation report below.

## Investigation Report
$investigation_report

## Requirements
1. **Generate specific code changes** to fix the identified issue
2. **Create test cases** to verify the fix works
3. **Provide implementation steps** for applying the fix
4. **Include rollback plan** in case of issues

## Expected Output Format

### Code Changes
Specific files and their modifications using diff format or clear before/after examples.

### Test Cases  
Unit tests or integration tests to verify the fix works.

### Implementation Steps
Step-by-step instructions for applying the fix safely.

### Rollback Plan
How to safely revert if the fix causes problems.

### Verification
How to confirm the fix resolves the original problem.

## Context
- Issue ID: $issue_id
- Project Path: $project_path
- Timestamp: $(date -Iseconds)

Please generate the fix now.
EOF

    # Execute Claude Code for fix generation
    local original_dir=$(pwd)
    if [[ -n "$project_path" && -d "$project_path" ]]; then
        cd "$project_path"
    fi
    
    local fix_output="${workspace_dir}/fix-report.md"
    if claude-code --file "$fix_prompt" > "$fix_output" 2>&1; then
        success "Fix generation completed"
    else
        error "Fix generation failed"
        cd "$original_dir"
        return 1
    fi
    
    cd "$original_dir"
    
    # Read fix content
    local fix_content=""
    if [[ -f "$fix_output" ]]; then
        fix_content=$(cat "$fix_output")
    else
        error "Fix output file not found"
        return 1
    fi
    
    # Update issue file with fix information
    local now=$(date -Iseconds)
    local escaped_fix=$(echo "$fix_content" | sed 's/\\/\\\\/g' | sed 's/"/\\"/g' | sed 's/$/\\n/' | tr -d '\n' | sed 's/\\n$//')
    
    # Update fix section in YAML
    sed -i "/implementation_plan: \"\"/c\\  implementation_plan: \"$escaped_fix\"" "$issue_file"
    sed -i "s/updated_at: \".*\"/updated_at: \"$now\"/g" "$issue_file"
    
    # Generate JSON output
    local json_output=$(cat << EOF
{
  "issue_id": "$issue_id",
  "fix_report": $(echo "$fix_content" | jq -R -s '.'),
  "fix_generated_at": "$now",
  "status": "completed",
  "workspace_path": "$workspace_dir",
  "file_path": "$issue_file"
}
EOF
)
    
    echo "$json_output"
    success "Fix generation completed for issue: $issue_id"
    return 0
}

# Move issue between folders
move_issue() {
    local issue_id="$1"
    local to_folder="$2"
    
    log "Moving issue $issue_id to $to_folder"
    
    # Find current file
    local issue_file
    if ! issue_file=$(find_issue_file "$issue_id"); then
        error "Issue file not found for ID: $issue_id"
        return 1
    fi
    
    local current_folder=$(basename "$(dirname "$issue_file")")
    local filename=$(basename "$issue_file")
    
    if [[ "$current_folder" == "$to_folder" ]]; then
        warn "Issue already in $to_folder folder"
        return 0
    fi
    
    # Update status in file
    local now=$(date -Iseconds)
    sed -i "s/status: $current_folder/status: $to_folder/g" "$issue_file"
    sed -i "s/updated_at: \".*\"/updated_at: \"$now\"/g" "$issue_file"
    
    # Add status-specific timestamps
    case "$to_folder" in
        investigating)
            sed -i "/investigation:/,/fix:/{/started_at: \"\"/c\\  started_at: \"$now\"}" "$issue_file"
            ;;
        in-progress)
            if grep -q "implementation:" "$issue_file"; then
                sed -i "/implementation:/,/metadata:/{/started_at: \"\"/c\\  started_at: \"$now\"}" "$issue_file"
            fi
            ;;
        fixed)
            sed -i "s/resolved_at: \"\"/resolved_at: \"$now\"/g" "$issue_file"
            ;;
    esac
    
    # Move file
    local new_path="$ISSUES_DIR/$to_folder/$filename"
    mv "$issue_file" "$new_path"
    
    success "Moved $filename from $current_folder to $to_folder"
    echo "New location: $new_path"
}

# Main function
main() {
    local command="${1:-}"
    shift || true
    
    case "$command" in
        investigate)
            local issue_id="${1:-}"
            local agent_id="${2:-unified-resolver}"
            local project_path="${3:-$(pwd)}"
            local prompt_template="${4:-}"
            
            if [[ -z "$issue_id" ]]; then
                error "Issue ID required"
                exit 1
            fi
            
            investigate_issue "$issue_id" "$agent_id" "$project_path" "$prompt_template"
            ;;
            
        generate-fix)
            local issue_id="${1:-}"
            local project_path="${2:-$(pwd)}"
            
            if [[ -z "$issue_id" ]]; then
                error "Issue ID required"
                exit 1
            fi
            
            generate_fix "$issue_id" "$project_path"
            ;;
            
        move)
            local issue_id="${1:-}"
            local to_folder="${2:-}"
            
            if [[ -z "$issue_id" || -z "$to_folder" ]]; then
                error "Issue ID and destination folder required"
                exit 1
            fi
            
            move_issue "$issue_id" "$to_folder"
            ;;
            
        help|--help|-h)
            cat << EOF
File-Based Claude Code Investigation Script

Usage: $0 <command> [options]

Commands:
  investigate <issue_id> [agent_id] [project_path] [prompt]
    Run Claude Code investigation for an issue
    
  generate-fix <issue_id> [project_path] 
    Generate code fixes based on investigation
    
  move <issue_id> <folder>
    Move issue to different status folder
    
  help
    Show this help message

Examples:
  $0 investigate "issue-123" "unified-resolver" "/path/to/project"
  $0 generate-fix "issue-123" "/path/to/project"
  $0 move "issue-123" "active"

Folder Options: open, active, completed, failed, archived
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
