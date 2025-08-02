# Claude Code Automation Integration Guide

This guide explains how to integrate Claude Code with external automation tools, orchestration systems, and workflows for maximum flexibility and power.

## 🎯 Philosophy: Separation of Concerns

**Claude Code's Role**: AI-powered coding tool that excels at understanding and modifying code  
**Orchestration Layer**: External tools (n8n, workflows, scripts) handle multi-resource coordination

This separation allows:
- Claude Code to focus on what it does best
- Orchestration tools to handle complex workflows
- Better maintainability and testing
- Reusable components across different automation systems

## 🔄 Integration Patterns

### Pattern 1: Sequential Processing
Claude Code processes code, external tools handle results

```bash
# Step 1: Claude Code analyzes
claude -p "Analyze code quality" --output-format stream-json > analysis.json

# Step 2: External tool processes results  
cat analysis.json | jq '.content[0].text' | external-processor

# Step 3: Claude Code implements fixes
claude -p "Fix issues based on analysis" --resume session-id
```

### Pattern 2: Resource Orchestration
External tools coordinate multiple resources, Claude Code handles code tasks

```yaml
# n8n workflow example
nodes:
  1. HTTP Trigger (receive code change request)
  2. Whisper Node (transcribe voice request if audio)
  3. Claude Code Node (analyze and implement changes)  
  4. MinIO Node (store artifacts)
  5. Notification Node (alert team)
```

### Pattern 3: Pipeline Processing
Claude Code as one stage in larger automation pipeline

```bash
# Pipeline: Source → AI Analysis → Implementation → Validation → Deploy
source_files | ai_analyzer | claude_implementer | validator | deployer
```

## 🛠️ Automation-Friendly Features

### Structured Output Formats

**Current**: Stream JSON (requires parsing)
```bash
claude -p "Fix bugs" --output-format stream-json
# Returns: complex streaming JSON that's hard to parse
```

**Enhanced** (to be implemented):
```bash
# Simple structured output
claude -p "Fix bugs" --output-format automation
# Returns: {"status": "completed", "files_modified": ["src/app.ts"], "summary": "Fixed 3 bugs"}

# Result extraction
claude result --session abc123 --extract files-changed
# Returns: src/app.ts\nsrc/utils.ts

claude result --session abc123 --extract summary  
# Returns: Fixed authentication bug and updated error handling
```

### Predictable Exit Codes

**Current**: Complex exit code interpretation required
```bash
# Complex logic needed to determine if operation succeeded
if grep -q "error_max_turns\|success" output.json; then
    # Success
else  
    # Failure
fi
```

**Enhanced** (to be implemented):
```bash
# Predictable exit codes
claude -p "Fix bugs" --exit-code-mode simple
echo $?  # 0=success, 1=error, 2=max_turns_reached, 3=permission_denied
```

### Session State Management

**Current**: Manual session ID extraction
```bash
session_id=$(tail -1 output.json | jq -r '.sessionId')
```

**Enhanced** (to be implemented):
```bash
# Automatic session management
claude session --create "bug-fix-workflow" 
claude session --status "bug-fix-workflow"  # active|completed|failed
claude session --export "bug-fix-workflow" --format diff
```

## 🔧 Integration with Specific Tools

### n8n Workflows

**Claude Code Node Configuration**:
```yaml
name: "Claude Code Task"
type: "claude-code"
parameters:
  action: "run"
  prompt: "{{ $json.task_description }}"
  allowed_tools: ["Read", "Edit", "Write"]
  max_turns: 20
  output_format: "automation"
  timeout: 1800
```

**Example Workflow**: Voice-to-Code Pipeline
```yaml
nodes:
  - name: "Voice Input"
    type: "webhook"
    
  - name: "Transcribe Audio"  
    type: "whisper"
    parameters:
      audio_file: "{{ $node['Voice Input'].json.audio_url }}"
      
  - name: "Implement Request"
    type: "claude-code"
    parameters:
      prompt: "Implement: {{ $node['Transcribe Audio'].json.transcript }}"
      allowed_tools: ["Read", "Edit", "Write", "Bash(npm test)"]
      
  - name: "Store Results"
    type: "minio"
    parameters:
      bucket: "code-artifacts"
      files: "{{ $node['Implement Request'].json.files_modified }}"
      
  - name: "Notify Team"
    type: "slack"
    parameters:
      message: "Code implemented: {{ $node['Implement Request'].json.summary }}"
```

### GitHub Actions Integration

**Workflow File**: `.github/workflows/ai-code-review.yml`
```yaml
name: AI Code Review
on: [pull_request]

jobs:
  ai-review:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Install Claude Code
        run: npm install -g @anthropic-ai/claude-code
        
      - name: AI Code Review
        run: |
          ./scripts/resources/agents/claude-code/manage.sh \
            --action run \
            --prompt "Review this PR for security and best practices" \
            --allowed-tools "Read,Grep,WebSearch" \
            --max-turns 10 \
            --output-format automation > review.json
            
      - name: Post Review Comment
        uses: actions/github-script@v7
        with:
          script: |
            const fs = require('fs');
            const review = JSON.parse(fs.readFileSync('review.json'));
            github.rest.issues.createComment({
              issue_number: context.issue.number,
              owner: context.repo.owner,
              repo: context.repo.repo,
              body: `## AI Code Review\n\n${review.summary}`
            });
```

### Shell Script Orchestration

**Example**: Automated Maintenance Pipeline
```bash
#!/bin/bash
# maintenance-pipeline.sh

set -euo pipefail

# Configuration
CLAUDE_SCRIPT="./scripts/resources/agents/claude-code/manage.sh"
LOG_DIR="logs/maintenance/$(date +%Y-%m-%d)"
mkdir -p "$LOG_DIR"

# Stage 1: Code Analysis
echo "🔍 Analyzing code quality..."
$CLAUDE_SCRIPT \
  --action run \
  --prompt "Analyze entire codebase for quality issues" \
  --allowed-tools "Read,Grep,Glob" \
  --max-turns 20 \
  --output-format automation > "$LOG_DIR/analysis.json"

# Extract issues found
issues_count=$(jq -r '.metrics.issues_found // 0' "$LOG_DIR/analysis.json")
if [[ $issues_count -gt 0 ]]; then
  echo "⚠️  Found $issues_count issues, proceeding with fixes..."
  
  # Stage 2: Implement Fixes
  $CLAUDE_SCRIPT \
    --action batch \
    --prompt "Fix the issues identified in the analysis" \
    --allowed-tools "Read,Edit,MultiEdit,Bash(npm test)" \
    --max-turns 50 \
    --output-format automation > "$LOG_DIR/fixes.json"
    
  # Stage 3: Validate Changes
  if npm test; then
    echo "✅ Tests passed, committing changes..."
    git add -A
    git commit -m "Automated maintenance: fixed $issues_count issues"
  else
    echo "❌ Tests failed, reverting changes..."
    git checkout -- .
    exit 1
  fi
else
  echo "✅ No issues found, maintenance complete"
fi
```

### Docker Integration

**Dockerfile** for containerized Claude Code:
```dockerfile
FROM node:18-alpine

# Install Claude Code
RUN npm install -g @anthropic-ai/claude-code

# Copy management scripts
COPY scripts/resources/agents/claude-code /claude-code
WORKDIR /workspace

# Entry point for automation
ENTRYPOINT ["/claude-code/manage.sh"]
```

**Usage in Docker Compose**:
```yaml
version: '3.8'
services:
  claude-code:
    build: .
    volumes:
      - ./src:/workspace/src
      - ./logs:/workspace/logs
    environment:
      - ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}
    command: >
      --action batch
      --prompt "Refactor legacy code for better maintainability"
      --allowed-tools "Read,Edit,MultiEdit"
      --max-turns 100
      --output-format automation
```

## 📊 Monitoring & Observability

### Structured Logging for Automation

**Log Format** (to be implemented):
```json
{
  "timestamp": "2024-01-15T10:30:00Z",
  "session_id": "abc123",
  "action": "run",
  "status": "completed",
  "duration_ms": 45000,
  "turns_used": 15,
  "max_turns": 20,
  "files_modified": ["src/app.ts", "src/utils.ts"],
  "tools_used": ["Read", "Edit", "Bash"],
  "prompt_tokens": 1500,
  "completion_tokens": 800,
  "cost_estimate": "$0.05"
}
```

### Health Checks for Automation

```bash
# Service health check
claude-code --health-check --json
# Returns: {"status": "healthy", "api_connected": true, "version": "1.0.58"}

# Capability check
claude-code --capabilities --json  
# Returns: {"tools": ["Read", "Edit", ...], "max_turns": 200, "subscriptions": ["pro"]}
```

### Metrics Collection

**Prometheus Metrics** (future enhancement):
```bash
# Counter: Total operations
claude_code_operations_total{action="run",status="success"} 142

# Histogram: Operation duration  
claude_code_duration_seconds{action="batch"} 45.2

# Gauge: Active sessions
claude_code_active_sessions 3
```

## 🔧 Configuration Management

### Environment-Based Configuration

```bash
# .env for automation environments
CLAUDE_CODE_DEFAULT_TOOLS="Read,Edit,Write"
CLAUDE_CODE_DEFAULT_MAX_TURNS=20
CLAUDE_CODE_DEFAULT_TIMEOUT=1800
CLAUDE_CODE_OUTPUT_FORMAT=automation
CLAUDE_CODE_LOG_LEVEL=info
```

### Workflow Configuration Files

**claude-workflow.yaml**:
```yaml
name: "Code Maintenance Workflow"
description: "Automated code quality improvements"

defaults:
  max_turns: 30
  timeout: 1800
  allowed_tools: ["Read", "Edit", "MultiEdit", "Bash(npm test)"]
  output_format: "automation"

stages:
  - name: "analyze"
    prompt: "Analyze code for quality issues"
    tools: ["Read", "Grep", "Glob"]
    max_turns: 10
    
  - name: "implement"
    prompt: "Fix issues found in analysis stage"
    depends_on: ["analyze"]
    max_turns: 50
    
  - name: "validate"
    prompt: "Run tests and verify fixes"
    tools: ["Bash(npm test)", "Read"]
    max_turns: 5
    required: true
```

## 🔄 Best Practices for Automation

### 1. Error Handling
```bash
# Robust error handling in automation scripts
if ! claude_output=$(claude -p "Task" --output-format automation 2>&1); then
  echo "Claude Code failed: $claude_output"
  # Handle failure (retry, alert, rollback)
  exit 1
fi

# Parse structured output safely
if ! status=$(echo "$claude_output" | jq -r '.status' 2>/dev/null); then
  echo "Failed to parse Claude Code output"
  exit 1
fi
```

### 2. Resource Management
```bash
# Check resource availability before starting
check_resources() {
  if ! claude --version &>/dev/null; then
    echo "Claude Code not available"
    return 1
  fi
  
  if ! curl -s "https://api.anthropic.com/health" &>/dev/null; then
    echo "Anthropic API not reachable"
    return 1
  fi
  
  return 0
}
```

### 3. Timeout and Retry Logic
```bash
# Retry with exponential backoff
retry_with_backoff() {
  local max_attempts=3
  local delay=1
  
  for i in $(seq 1 $max_attempts); do
    if claude -p "$1" --timeout 300; then
      return 0
    fi
    
    echo "Attempt $i failed, retrying in ${delay}s..."
    sleep $delay
    delay=$((delay * 2))
  done
  
  return 1
}
```

### 4. State Management
```bash
# Save and restore workflow state
save_workflow_state() {
  local state_file="$1"
  cat > "$state_file" <<EOF
{
  "session_id": "$session_id",
  "stage": "$current_stage", 
  "timestamp": "$(date -Iseconds)",
  "files_modified": $(printf '%s\n' "${modified_files[@]}" | jq -R . | jq -s .)
}
EOF
}

restore_workflow_state() {
  local state_file="$1"
  if [[ -f "$state_file" ]]; then
    session_id=$(jq -r '.session_id' "$state_file")
    current_stage=$(jq -r '.stage' "$state_file")
    # Resume from saved state
  fi
}
```

## 🚀 Advanced Integration Patterns

### Multi-Stage Workflows with State Passing

```bash
# Stage 1: Analysis
analysis_result=$(claude -p "Analyze code" --output-format automation)
analysis_summary=$(echo "$analysis_result" | jq -r '.summary')

# Stage 2: Implementation (using analysis results)
implementation_prompt="Based on this analysis: $analysis_summary, implement the fixes"
claude -p "$implementation_prompt" --output-format automation
```

### Parallel Processing with Resource Coordination

```bash
# Parallel code analysis across multiple directories
analyze_directory() {
  local dir="$1"
  claude -p "Analyze code quality in $dir" \
    --allowed-tools "Read,Grep" \
    --output-format automation > "analysis_${dir//\//_}.json" &
}

# Start parallel analyses
for dir in src tests docs; do
  analyze_directory "$dir"
done

# Wait for all to complete
wait

# Consolidate results
jq -s 'map(select(.status == "completed")) | {total_issues: map(.metrics.issues_found // 0) | add}' analysis_*.json
```

### Event-Driven Automation

```bash
# File watcher integration
inotifywait -m -e modify src/ | while read -r directory event filename; do
  echo "File changed: $directory$filename"
  
  # Automatic code review on changes
  claude -p "Review changes in $directory$filename for issues" \
    --allowed-tools "Read,Grep" \
    --max-turns 5 \
    --output-format automation > "review_$(date +%s).json"
done
```

This automation integration guide enables powerful orchestration of Claude Code with external tools while maintaining clean separation of concerns and robust error handling.