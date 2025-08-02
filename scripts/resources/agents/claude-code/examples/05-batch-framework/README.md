# Enhanced Batch Framework Examples

This directory contains examples of using Claude Code's enhanced batch execution framework for complex, multi-stage operations.

## 🚀 Quick Start

### Basic Batch Execution
```bash
# Simple batch with automatic session handling
./manage.sh --action batch-simple \
  --prompt "Fix all code quality issues" \
  --max-turns 100 \
  --allowed-tools "Read,Edit,MultiEdit,Bash(npm test)"
```

### Configuration-Based Workflow
```bash
# Use a JSON configuration file for complex workflows
./manage.sh --action batch-config \
  --config-file templates/batch-config-example.json
```

### Parallel Processing
```bash
# Run multiple tasks in parallel
./manage.sh --action batch-parallel \
  --workers 3 \
  --prompt "Analyze security issues" \
  --max-turns 50
```

## 📋 Configuration File Format

### JSON Configuration
```json
{
  "name": "My Workflow",
  "description": "Description of what this workflow does",
  "defaults": {
    "max_turns": 30,
    "allowed_tools": ["Read", "Edit", "Write"],
    "output_format": "automation"
  },
  "stages": [
    {
      "name": "analysis",
      "prompt": "Analyze the code for issues",
      "tools": ["Read", "Grep"],
      "max_turns": 20,
      "required": false
    },
    {
      "name": "implementation", 
      "prompt": "Fix the issues found",
      "tools": ["Read", "Edit", "MultiEdit"],
      "max_turns": 50,
      "depends_on": ["analysis"],
      "required": true
    }
  ]
}
```

## 🛠️ Available Batch Actions

### `batch-simple`
Clean, straightforward batch execution with automatic session handling and error recovery.

**Features:**
- Automatic session continuity
- Progress tracking
- Error recovery with retry logic
- File modification tracking
- Summary generation

**Usage:**
```bash
./manage.sh --action batch-simple \
  --prompt "Your task description" \
  --max-turns 100 \
  --batch-size 50 \
  --allowed-tools "Read,Edit,Write"
```

### `batch-config`
Configuration-driven workflow execution for complex multi-stage operations.

**Features:**
- JSON/YAML configuration files
- Multi-stage workflows
- Stage dependencies
- Required vs optional stages
- Custom tool sets per stage

**Usage:**
```bash
./manage.sh --action batch-config \
  --config-file path/to/workflow.json
```

### `batch-multi`
Execute multiple different tasks in sequence.

**Features:**
- Sequential task execution
- Individual task tracking
- Cooldown between tasks
- Task-specific configurations

**Usage:**
```bash
# Note: Full multi-task support requires script modification
# This example shows single task - extend for multiple tasks
./manage.sh --action batch-multi \
  --prompt "First task" \
  --max-turns 50
```

### `batch-parallel`
Run multiple tasks concurrently using worker processes.

**Features:**
- Parallel execution
- Worker-based processing
- Task queue management
- Atomic task assignment

**Usage:**
```bash
./manage.sh --action batch-parallel \
  --workers 3 \
  --prompt "Analyze code quality" \
  --max-turns 50
```

## 📊 Output and Results

All batch operations generate structured output including:

- **Session IDs** for continuation
- **Files modified** during execution
- **Progress tracking** and status
- **Error reporting** and recovery
- **Summary reports** in JSON format

### Example Output Directory Structure
```
/tmp/claude_batch_20240115_143022_a1b2c3d4/
├── batch_1.json          # First batch execution results
├── batch_2.json          # Second batch execution results  
├── batch_3.json          # Third batch execution results
└── summary.json          # Overall execution summary
```

### Summary JSON Format
```json
{
  "status": "completed",
  "session_id": "session_abc123def456",
  "total_turns_requested": 100,
  "turns_remaining": 0,
  "batches_executed": 3,
  "failed_batches": 0,
  "files_modified": [
    "src/app.ts",
    "src/utils.ts",
    "README.md"
  ],
  "output_directory": "/tmp/claude_batch_20240115_143022_a1b2c3d4",
  "timestamp": "2024-01-15T14:32:45Z"
}
```

## 🔧 Integration with Automation

### Shell Script Integration
```bash
#!/bin/bash
# automated-maintenance.sh

# Run batch operation and capture results
result_file=$(./manage.sh --action batch-simple \
  --prompt "Perform code maintenance" \
  --max-turns 100 \
  --allowed-tools "Read,Edit,MultiEdit,Bash(npm test)")

# Parse results
status=$(./manage.sh --action extract --input-file "$result_file" --extract-type status)
files_modified=$(./manage.sh --action extract --input-file "$result_file" --extract-type files)

if [[ "$status" == "completed" ]]; then
  echo "✅ Maintenance completed successfully"
  echo "Modified files: $files_modified"
  
  # Commit changes if tests pass
  if npm test; then
    git add -A
    git commit -m "Automated maintenance: $(date)"
  fi
else
  echo "❌ Maintenance failed or incomplete"
  exit 1
fi
```

### n8n Workflow Integration
```yaml
# n8n workflow node configuration
- name: "Claude Code Batch"
  type: "execute-command"
  parameters:
    command: "./manage.sh"
    arguments: 
      - "--action"
      - "batch-simple"
      - "--prompt"
      - "{{ $json.task_description }}"
      - "--max-turns"
      - "{{ $json.max_turns || 50 }}"
      - "--allowed-tools"
      - "{{ $json.allowed_tools || 'Read,Edit,Write' }}"
    options:
      timeout: 3600
      
- name: "Parse Results"
  type: "execute-command"
  parameters:
    command: "./manage.sh"
    arguments:
      - "--action"
      - "parse-result"
      - "--input-file"
      - "{{ $node['Claude Code Batch'].json.output_file }}"
      - "--format"
      - "automation"
```

## 🎯 Best Practices

### 1. Choose the Right Batch Type
- **batch-simple**: Single complex task with many turns
- **batch-config**: Multi-stage workflows with dependencies
- **batch-multi**: Multiple unrelated tasks in sequence
- **batch-parallel**: Multiple similar tasks that can run concurrently

### 2. Set Appropriate Limits
- Start with smaller turn counts and increase as needed
- Use batch sizes of 50 or less for better error recovery
- Set realistic timeouts for complex operations

### 3. Tool Selection
- Be restrictive with tools for safety
- Use `Read,Grep` for analysis-only operations
- Add `Bash(specific command)` for controlled system access
- Include testing tools: `Bash(npm test)`, `Bash(npm run lint)`

### 4. Error Handling
- Make non-critical stages optional (`"required": false`)
- Use progress callbacks for long-running operations
- Monitor output directories for debugging
- Keep backups before running destructive operations

### 5. Resource Management
- Run parallel operations only when system resources allow
- Monitor disk space in output directories
- Clean up temporary files after processing
- Use appropriate cooldown periods between operations

## 🧪 Testing

Test the batch framework with safe operations first:

```bash
# Safe analysis operation
./manage.sh --action batch-simple \
  --prompt "Analyze code structure and document findings" \
  --max-turns 20 \
  --allowed-tools "Read,Grep,Write"

# Test configuration parsing
./manage.sh --action batch-config \
  --config-file templates/batch-config-example.json
```

This enhanced batch framework provides powerful automation capabilities while maintaining the safety and flexibility that makes Claude Code effective for development workflows.