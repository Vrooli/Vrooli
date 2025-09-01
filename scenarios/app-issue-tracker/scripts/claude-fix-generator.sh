#!/usr/bin/env bash

# Claude Code Fix Generator Script
# Generates, tests, and applies automated fixes for investigated issues

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="${SCRIPT_DIR}/.."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
ISSUES_DIR="${ISSUES_DIR:-/home/matthalloran8/Vrooli/scenarios/app-issue-tracker/issues}"
API_BASE_URL="http://localhost:8090/api"

# Logging functions
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

# Generate fix for an issue
generate_fix() {
    local issue_id="$1"
    local project_path="${2:-}"
    local auto_apply="${3:-false}"
    local backup_enabled="${4:-true}"
    
    log "Starting fix generation for issue: $issue_id"
    
    # Create fix workspace
    local workspace_dir="/tmp/claude-fix-${issue_id}"
    mkdir -p "$workspace_dir"
    
    # Get issue details from YAML files
    local investigation_report
    local issue_title
    local issue_description
    local error_message
    local stack_trace
    local issue_file=""
    
    # Search for issue file across all folders
    for folder in open investigating in-progress fixed closed failed; do
        for file in "$ISSUES_DIR/$folder"/*.yaml; do
            if [[ -f "$file" ]] && grep -q "id: $issue_id" "$file" 2>/dev/null; then
                issue_file="$file"
                break 2
            fi
        done
    done
    
    if [[ -z "$issue_file" || ! -f "$issue_file" ]]; then
        error "Issue file not found for ID: $issue_id"
        error "Searched in: $ISSUES_DIR"
        return 1
    fi
    
    log "Found issue file: $issue_file"
    
    # Extract data from YAML using grep and sed (portable approach)
    issue_title=$(grep "^title:" "$issue_file" | sed 's/title: *"\?\(.*\)"\?/\1/' | sed 's/"$//')
    issue_description=$(grep "^description:" "$issue_file" | sed 's/description: *"\?\(.*\)"\?/\1/' | sed 's/"$//')
    
    # Extract multi-line investigation report
    investigation_report=$(awk '/^investigation:/, /^fix:/' "$issue_file" | awk '/report:/{flag=1; next} /[a-z_]+:/{flag=0} flag' | sed 's/^  *//')
    
    # Extract error context
    error_message=$(awk '/^error_context:/, /^reproduction:/' "$issue_file" | grep "error_message:" | sed 's/.*error_message: *"\?\(.*\)"\?/\1/' | sed 's/"$//')
    stack_trace=$(awk '/^error_context:/, /^reproduction:/' "$issue_file" | awk '/stack_trace:/{flag=1; next} /[a-z_]+:/{flag=0} flag' | sed 's/^  *//')
    
    if [[ -n "$issue_title" ]]; then
        log "Retrieved issue data from file"
    else
        error "Failed to parse issue data from: $issue_file"
        return 1
    fi
    
    if [[ -z "$investigation_report" || "$investigation_report" == "" ]]; then
        error "No investigation report found for issue: $issue_id"
        error "Please run investigation first before generating fixes"
        return 1
    fi
    
    # Create fix generation prompt
    local fix_prompt_file="${workspace_dir}/fix-generation-prompt.md"
    
    cat > "$fix_prompt_file" << EOF
# Automated Fix Generation Request

You are an expert software engineer tasked with generating automated fixes for issues.

## Issue Information
- **Issue ID**: $issue_id
- **Title**: $issue_title
- **Description**: $issue_description

## Investigation Report
$investigation_report

## Error Details
$(if [[ -n "$error_message" ]]; then echo "**Error Message**: $error_message"; fi)
$(if [[ -n "$stack_trace" ]]; then echo "**Stack Trace**: $stack_trace"; fi)

## Task Requirements
Based on the investigation report above, please:

1. **Generate Specific Code Fixes**: Create actual code changes to resolve the identified issue
2. **Create Test Cases**: Write comprehensive tests to verify the fix works correctly
3. **Provide Implementation Plan**: Give step-by-step instructions for applying the fix safely
4. **Include Rollback Strategy**: Explain how to safely revert if the fix causes issues
5. **Risk Assessment**: Identify potential risks and mitigation strategies

## Output Format
Please structure your response as follows:

### 1. Fix Summary
Brief description of the proposed solution and what it changes.

### 2. Code Changes
Provide specific code modifications using diff format or clear before/after examples.
Include full file paths and line numbers where possible.

### 3. New Files
If any new files need to be created, provide their complete content.

### 4. Test Cases
Write comprehensive test cases that verify:
- The fix resolves the original issue
- No regressions are introduced
- Edge cases are handled properly

### 5. Implementation Steps
Detailed step-by-step instructions for:
1. Backing up current state
2. Applying the code changes
3. Running tests
4. Verifying the fix
5. Monitoring for issues

### 6. Rollback Plan
Clear instructions for reverting the changes if needed:
1. How to detect if rollback is needed
2. Steps to revert all changes
3. How to verify successful rollback

### 7. Risk Assessment
- **High Risk Changes**: List any changes that could significantly impact the system
- **Dependencies**: Identify what other components might be affected
- **Testing Strategy**: Recommend additional testing beyond the provided test cases
- **Monitoring**: Suggest metrics or logs to watch after deployment

### 8. Verification Checklist
A checklist to confirm the fix is working correctly:
- [ ] Original issue symptoms are resolved
- [ ] All existing tests still pass
- [ ] New tests pass
- [ ] Performance is not degraded
- [ ] No new errors in logs
- [ ] Dependent systems are not affected

## Context Information
- Project Path: $project_path
- Auto-apply Mode: $auto_apply
- Backup Enabled: $backup_enabled
- Generated: $(date -Iseconds)

Please provide a comprehensive fix that is safe, well-tested, and thoroughly documented.
EOF

    log "Generated fix prompt: $fix_prompt_file"
    
    # Check if claude-code is available
    if ! command -v claude-code &> /dev/null; then
        error "claude-code CLI not found. Please install Claude Code."
        return 1
    fi
    
    # Run Claude Code fix generation
    log "Executing Claude Code fix generation..."
    local fix_output="${workspace_dir}/fix-report.md"
    local claude_exit_code=0
    
    # Navigate to project path if specified
    local original_dir=$(pwd)
    if [[ -n "$project_path" && -d "$project_path" ]]; then
        log "Changing to project directory: $project_path"
        cd "$project_path"
    fi
    
    # Execute Claude Code with the fix prompt
    if claude-code --file "$fix_prompt_file" > "$fix_output" 2>&1; then
        success "Claude Code fix generation completed successfully"
    else
        claude_exit_code=$?
        error "Claude Code fix generation failed with exit code: $claude_exit_code"
        cd "$original_dir"
        return $claude_exit_code
    fi
    
    cd "$original_dir"
    
    # Parse the fix report
    local fix_content
    if [[ -f "$fix_output" ]]; then
        fix_content=$(cat "$fix_output")
        log "Fix report generated (${#fix_content} characters)"
    else
        error "Fix output file not found"
        return 1
    fi
    
    # Extract structured data from the fix report
    local fix_summary=""
    local implementation_steps=""
    local rollback_plan=""
    local risk_level="medium"
    
    # Parse fix summary
    if echo "$fix_content" | grep -q "### 1. Fix Summary"; then
        fix_summary=$(echo "$fix_content" | sed -n '/### 1. Fix Summary/,/### 2/p' | head -n -1 | tail -n +2)
    fi
    
    # Parse implementation steps
    if echo "$fix_content" | grep -q "### 5. Implementation Steps"; then
        implementation_steps=$(echo "$fix_content" | sed -n '/### 5. Implementation Steps/,/### 6/p' | head -n -1 | tail -n +2)
    fi
    
    # Parse rollback plan
    if echo "$fix_content" | grep -q "### 6. Rollback Plan"; then
        rollback_plan=$(echo "$fix_content" | sed -n '/### 6. Rollback Plan/,/### 7/p' | head -n -1 | tail -n +2)
    fi
    
    # Determine risk level from content
    if echo "$fix_content" | grep -qi "high risk"; then
        risk_level="high"
    elif echo "$fix_content" | grep -qi "low risk"; then
        risk_level="low"
    fi
    
    # Create backup if enabled
    local backup_path=""
    if [[ "$backup_enabled" == "true" && -n "$project_path" && -d "$project_path" ]]; then
        backup_path="${workspace_dir}/backup-$(date +%Y%m%d-%H%M%S).tar.gz"
        log "Creating backup: $backup_path"
        tar -czf "$backup_path" -C "$(dirname "$project_path")" "$(basename "$project_path")" 2>/dev/null || true
        if [[ -f "$backup_path" ]]; then
            success "Backup created: $backup_path"
        else
            warn "Backup creation failed"
            backup_path=""
        fi
    fi
    
    # Extract code changes for potential auto-application
    local has_code_changes=false
    if echo "$fix_content" | grep -q "### 2. Code Changes"; then
        has_code_changes=true
        # Extract and save code changes
        echo "$fix_content" | sed -n '/### 2. Code Changes/,/### 3/p' | head -n -1 | tail -n +2 > "${workspace_dir}/code-changes.md"
    fi
    
    # Apply fixes automatically if requested and safe
    local auto_apply_result="not_attempted"
    if [[ "$auto_apply" == "true" && "$has_code_changes" == "true" && "$risk_level" != "high" ]]; then
        log "Auto-applying fixes (risk level: $risk_level)..."
        if apply_code_changes "$workspace_dir/code-changes.md" "$project_path"; then
            auto_apply_result="success"
            success "Fixes applied automatically"
        else
            auto_apply_result="failed"
            warn "Auto-apply failed, manual intervention required"
        fi
    elif [[ "$risk_level" == "high" ]]; then
        warn "Skipping auto-apply due to high risk assessment"
        auto_apply_result="skipped_high_risk"
    fi
    
    # Update issue status in database
    update_issue_status "$issue_id" "fix_generated" "$fix_content"
    
    # Generate JSON output
    local json_output=$(cat << EOF
{
  "issue_id": "$issue_id",
  "fix_generation_status": "completed",
  "fix_report": $(echo "$fix_content" | jq -R -s '.'),
  "fix_summary": $(echo "${fix_summary:-No summary available}" | jq -R -s '.'),
  "implementation_steps": $(echo "${implementation_steps:-No steps provided}" | jq -R -s '.'),
  "rollback_plan": $(echo "${rollback_plan:-No rollback plan provided}" | jq -R -s '.'),
  "risk_level": "$risk_level",
  "has_code_changes": $has_code_changes,
  "auto_apply_enabled": $auto_apply,
  "auto_apply_result": "$auto_apply_result",
  "backup_path": $(echo "${backup_path:-}" | jq -R -s '.'),
  "workspace_path": "$workspace_dir",
  "fix_generated_at": "$(date -Iseconds)",
  "project_path": $(echo "${project_path:-}" | jq -R -s '.')
}
EOF
)
    
    # Output the result
    echo "$json_output"
    
    success "Fix generation completed for issue: $issue_id"
    log "Workspace preserved at: $workspace_dir"
    
    return 0
}

# Apply code changes from a markdown file
apply_code_changes() {
    local changes_file="$1"
    local project_path="$2"
    
    if [[ ! -f "$changes_file" ]]; then
        error "Changes file not found: $changes_file"
        return 1
    fi
    
    if [[ -z "$project_path" || ! -d "$project_path" ]]; then
        error "Invalid project path: $project_path"
        return 1
    fi
    
    log "Applying code changes from: $changes_file"
    
    # This is a simplified implementation
    # In a real implementation, you would parse the markdown and apply specific changes
    # For now, we'll just log that changes would be applied
    warn "Auto-apply is not yet fully implemented"
    warn "Please review the changes in: $changes_file"
    warn "And apply them manually to: $project_path"
    
    return 1
}

# Update issue status in database
update_issue_status() {
    local issue_id="$1"
    local status="$2"
    local fix_report="$3"
    
    if command -v psql &> /dev/null; then
        log "Updating issue status to: $status"
        
        # Escape the fix report for SQL
        local escaped_report=$(echo "$fix_report" | sed "s/'/''/g")
        
        PGPASSWORD=postgres psql -h localhost -p 5432 -U postgres -d issue_tracker -c \
            "UPDATE issues SET status = '$status', suggested_fix = '$escaped_report', updated_at = CURRENT_TIMESTAMP WHERE id = '$issue_id';" \
            >/dev/null 2>&1
        
        if [[ $? -eq 0 ]]; then
            success "Issue status updated successfully"
        else
            warn "Failed to update issue status in database"
        fi
    else
        warn "Cannot update database - psql not available"
    fi
}

# Test generated fixes
test_fixes() {
    local issue_id="$1"
    local project_path="${2:-}"
    
    log "Testing fixes for issue: $issue_id"
    
    if [[ -z "$project_path" || ! -d "$project_path" ]]; then
        warn "No project path specified or path doesn't exist, skipping tests"
        return 0
    fi
    
    local original_dir=$(pwd)
    cd "$project_path"
    
    # Look for common test commands
    local test_command=""
    
    if [[ -f "package.json" ]] && grep -q "test" package.json; then
        test_command="npm test"
    elif [[ -f "Makefile" ]] && grep -q "test" Makefile; then
        test_command="make test"
    elif [[ -f "pytest.ini" ]] || [[ -f "setup.py" ]]; then
        test_command="python -m pytest"
    elif [[ -f "go.mod" ]]; then
        test_command="go test ./..."
    elif [[ -f "Cargo.toml" ]]; then
        test_command="cargo test"
    else
        warn "No test framework detected, skipping automated testing"
        cd "$original_dir"
        return 0
    fi
    
    log "Running tests with command: $test_command"
    
    if eval "$test_command"; then
        success "All tests passed"
        cd "$original_dir"
        return 0
    else
        error "Tests failed"
        cd "$original_dir"
        return 1
    fi
}

# Validate fix completeness
validate_fix() {
    local fix_report_file="$1"
    
    log "Validating fix completeness..."
    
    if [[ ! -f "$fix_report_file" ]]; then
        error "Fix report file not found: $fix_report_file"
        return 1
    fi
    
    local content=$(cat "$fix_report_file")
    local score=0
    local max_score=8
    
    # Check for required sections
    if echo "$content" | grep -q "### 1. Fix Summary"; then
        ((score++))
    else
        warn "Missing: Fix Summary"
    fi
    
    if echo "$content" | grep -q "### 2. Code Changes"; then
        ((score++))
    else
        warn "Missing: Code Changes"
    fi
    
    if echo "$content" | grep -q "### 4. Test Cases"; then
        ((score++))
    else
        warn "Missing: Test Cases"
    fi
    
    if echo "$content" | grep -q "### 5. Implementation Steps"; then
        ((score++))
    else
        warn "Missing: Implementation Steps"
    fi
    
    if echo "$content" | grep -q "### 6. Rollback Plan"; then
        ((score++))
    else
        warn "Missing: Rollback Plan"
    fi
    
    if echo "$content" | grep -q "### 7. Risk Assessment"; then
        ((score++))
    else
        warn "Missing: Risk Assessment"
    fi
    
    if echo "$content" | grep -q "### 8. Verification Checklist"; then
        ((score++))
    else
        warn "Missing: Verification Checklist"
    fi
    
    # Check for code examples
    if echo "$content" | grep -qE '```|`[^`]+`'; then
        ((score++))
    else
        warn "Missing: Code examples or formatting"
    fi
    
    local percentage=$((score * 100 / max_score))
    log "Fix completeness: $score/$max_score ($percentage%)"
    
    if [[ $score -ge 6 ]]; then
        success "Fix validation passed"
        return 0
    else
        warn "Fix validation failed - missing required sections"
        return 1
    fi
}

# Main function
main() {
    local command="${1:-}"
    shift || true
    
    case "$command" in
        generate)
            local issue_id="${1:-}"
            local project_path="${2:-}"
            local auto_apply="${3:-false}"
            local backup_enabled="${4:-true}"
            
            if [[ -z "$issue_id" ]]; then
                error "Issue ID required"
                echo "Usage: $0 generate <issue_id> [project_path] [auto_apply] [backup_enabled]"
                exit 1
            fi
            
            generate_fix "$issue_id" "$project_path" "$auto_apply" "$backup_enabled"
            ;;
            
        test)
            local issue_id="${1:-}"
            local project_path="${2:-}"
            
            if [[ -z "$issue_id" ]]; then
                error "Issue ID required"
                exit 1
            fi
            
            test_fixes "$issue_id" "$project_path"
            ;;
            
        validate)
            local fix_report_file="${1:-}"
            
            if [[ -z "$fix_report_file" ]]; then
                error "Fix report file required"
                exit 1
            fi
            
            validate_fix "$fix_report_file"
            ;;
            
        help|--help|-h)
            cat << EOF
Claude Code Fix Generator

Usage: $0 <command> [options]

Commands:
  generate <issue_id> [project_path] [auto_apply] [backup_enabled]
    Generate fixes for an investigated issue
    
  test <issue_id> [project_path]  
    Run tests to verify fixes work correctly
    
  validate <fix_report_file>
    Validate completeness of a fix report
    
  help
    Show this help message

Examples:
  $0 generate "issue-123" "/path/to/project" true true
  $0 test "issue-123" "/path/to/project"
  $0 validate "/tmp/claude-fix-issue-123/fix-report.md"

Environment Variables:
  POSTGRES_URL    - PostgreSQL connection string
  API_BASE_URL    - App Issue Tracker API base URL
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