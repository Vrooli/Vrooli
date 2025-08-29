#!/usr/bin/env bash

# Claude Code Investigation Script
# Interfaces between N8N workflows and Claude Code CLI for issue investigation

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIG_DIR="${SCRIPT_DIR}/../initialization/configuration"
CLAUDE_CONFIG="${CONFIG_DIR}/claude-config.json"

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

# Main investigation function
investigate_issue() {
    local issue_id="$1"
    local agent_id="$2"
    local project_path="${3:-}"
    local prompt_template="${4:-}"
    
    log "Starting investigation for issue: $issue_id"
    
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
    
    # Check if claude-code is available
    if ! command -v claude-code &> /dev/null; then
        error "claude-code CLI not found. Please install Claude Code."
        return 1
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
    
    # Execute Claude Code with the investigation prompt
    if claude-code --file "$prompt_file" > "$investigation_output" 2>&1; then
        success "Claude Code investigation completed successfully"
    else
        claude_exit_code=$?
        error "Claude Code investigation failed with exit code: $claude_exit_code"
        cd "$original_dir"
        return $claude_exit_code
    fi
    
    cd "$original_dir"
    
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
    
    # Parse root cause (look for specific section)
    if echo "$report_content" | grep -q "### Root Cause Analysis"; then
        root_cause=$(echo "$report_content" | sed -n '/### Root Cause Analysis/,/###/p' | head -n -1 | tail -n +2 | tr '\n' ' ')
    fi
    
    # Parse suggested fix
    if echo "$report_content" | grep -q "### Recommended Solutions"; then
        suggested_fix=$(echo "$report_content" | sed -n '/### Recommended Solutions/,/###/p' | head -n -1 | tail -n +2 | tr '\n' ' ')
    fi
    
    # Parse confidence score
    if echo "$report_content" | grep -qE "confidence.*[0-9]+"; then
        confidence_score=$(echo "$report_content" | grep -oE "confidence.*([0-9]+)" | grep -oE "[0-9]+" | head -1)
    fi
    
    # Parse affected files
    if echo "$report_content" | grep -q "### Affected Components"; then
        affected_files=$(echo "$report_content" | sed -n '/### Affected Components/,/###/p' | head -n -1 | tail -n +2 | grep -oE '[a-zA-Z0-9/_.-]+\.(js|ts|py|go|java|cpp|c|h)' | head -10 | tr '\n' ',' | sed 's/,$//')
    fi
    
    # Generate JSON output for N8N
    local json_output=$(cat << EOF
{
  "issue_id": "$issue_id",
  "agent_id": "$agent_id",
  "investigation_report": $(echo "$report_content" | jq -R -s '.'),
  "root_cause": $(echo "${root_cause:-Unknown}" | jq -R -s '.'),
  "suggested_fix": $(echo "${suggested_fix:-No specific fix identified}" | jq -R -s '.'),
  "confidence_score": ${confidence_score:-5},
  "affected_files": [$(if [[ -n "$affected_files" ]]; then echo "$affected_files" | sed 's/,/","/g' | sed 's/^/"/' | sed 's/$/"/' | sed 's/,"$/"/'; fi)],
  "investigation_completed_at": "$(date -Iseconds)",
  "status": "completed",
  "workspace_path": "$workspace_dir"
}
EOF
)
    
    # Output the result
    echo "$json_output"
    
    # Clean up workspace (optional - keep for debugging)
    # rm -rf "$workspace_dir"
    
    success "Investigation completed for issue: $issue_id"
    return 0
}

# Generate investigation fix
generate_fix() {
    local issue_id="$1"
    local investigation_report="$2"
    local project_path="${3:-}"
    
    log "Generating fix for issue: $issue_id"
    
    # Create fix workspace
    local workspace_dir="/tmp/claude-fix-${issue_id}"
    mkdir -p "$workspace_dir"
    
    # Create fix generation prompt
    local fix_prompt="${workspace_dir}/fix-prompt.md"
    
    cat > "$fix_prompt" << EOF
# Fix Generation Request

Based on the investigation report below, please generate actual code fixes for this issue.

## Investigation Report
$investigation_report

## Task
1. **Analyze the investigation findings**
2. **Generate specific code changes** to fix the identified issue
3. **Create test cases** to verify the fix
4. **Provide implementation steps**

## Expected Output Format
Please provide:

### Code Changes
Specific files and their modifications using diff format or clear before/after examples.

### Test Cases
Unit tests or integration tests to verify the fix works.

### Implementation Steps
Step-by-step instructions for applying the fix.

### Rollback Plan
How to safely revert if the fix causes issues.

### Verification
How to confirm the fix resolves the original problem.

Please generate the fix now.
EOF

    # Execute Claude Code for fix generation
    local fix_output="${workspace_dir}/fix-report.md"
    local original_dir=$(pwd)
    
    if [[ -n "$project_path" && -d "$project_path" ]]; then
        cd "$project_path"
    fi
    
    if claude-code --file "$fix_prompt" > "$fix_output" 2>&1; then
        success "Fix generation completed successfully"
    else
        error "Fix generation failed"
        cd "$original_dir"
        return 1
    fi
    
    cd "$original_dir"
    
    # Read the fix content
    local fix_content
    if [[ -f "$fix_output" ]]; then
        fix_content=$(cat "$fix_output")
    else
        error "Fix output file not found"
        return 1
    fi
    
    # Generate JSON output
    local fix_json=$(cat << EOF
{
  "issue_id": "$issue_id",
  "fix_report": $(echo "$fix_content" | jq -R -s '.'),
  "fix_generated_at": "$(date -Iseconds)",
  "status": "completed",
  "workspace_path": "$workspace_dir"
}
EOF
)
    
    echo "$fix_json"
    success "Fix generation completed for issue: $issue_id"
    return 0
}

# Main function
main() {
    local command="${1:-}"
    shift || true
    
    case "$command" in
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