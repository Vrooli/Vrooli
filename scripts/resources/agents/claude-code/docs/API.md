# Claude Code API Reference

Claude Code provides both interactive CLI commands and management script API for programmatic usage.

## 🔌 CLI Commands

### Basic Commands

```bash
# Start interactive session
claude

# Get help
claude --help

# Check version
claude --version

# Run diagnostics
claude doctor

# Pipe input to Claude
echo "Explain this code" | claude

# Use with file monitoring
tail -f app.log | claude -p "Alert me if you see errors"
```

### Command Line Options

```bash
# Run with specific prompt
claude -p "Your prompt here"

# Pipe input with prompt
cat file.js | claude -p "Review this code"

# Chain commands
find . -name "*.js" | claude -p "Find security issues"
```

### Interactive Features

- **Tab**: Command completion
- **↑**: Command history
- **/**: Available slash commands
- **Ctrl+C**: Interrupt operations

## 🛠️ Management Script API

The management script provides programmatic access to Claude Code functionality.

### Installation & Management

```bash
# Install Claude Code
./manage.sh --action install

# Check status
./manage.sh --action status

# Get detailed information
./manage.sh --action info

# Uninstall
./manage.sh --action uninstall

# Force reinstall
./manage.sh --action install --force yes
```

### Running Single Prompts

```bash
# Basic prompt execution
./manage.sh --action run --prompt "Explain this Python function"

# With max turns limit
./manage.sh --action run --prompt "Write unit tests" --max-turns 10

# With specific tools allowed
./manage.sh --action run --prompt "Fix this bug" --allowed-tools "Edit,Bash"

# With JSON output format
./manage.sh --action run --prompt "Analyze code" --output-format stream-json

# Skip permission checks (USE WITH EXTREME CAUTION)
./manage.sh --action run --prompt "System task" --dangerously-skip-permissions yes
```

#### Available Tools

| Tool | Purpose | Example Use |
|------|---------|-------------|
| `Edit` | Modify existing files | Code fixes, refactoring |
| `Read` | Read file contents | Code analysis, review |
| `Write` | Create new files | Documentation, new modules |
| `Bash` | Execute shell commands | Running tests, builds |
| `Grep` | Search file contents | Finding patterns, code search |

#### Output Formats

- `default` - Standard interactive format
- `stream-json` - JSON-formatted streaming output
- `plain` - Plain text without formatting

### Batch Processing

```bash
# Basic batch mode
./manage.sh --action batch --prompt "Refactor all test files" --max-turns 50 --timeout 3600

# Batch with specific tools
./manage.sh --action batch --prompt "Update documentation" \
  --allowed-tools "Edit,Read,Write" \
  --max-turns 30

# Batch with permission checks disabled (DANGEROUS)
./manage.sh --action batch --prompt "System maintenance" \
  --dangerously-skip-permissions yes \
  --max-turns 20
```

#### Batch Parameters

| Parameter | Description | Default | Example |
|-----------|-------------|---------|---------|
| `--max-turns` | Maximum conversation turns | 10 | `--max-turns 50` |
| `--timeout` | Timeout in seconds | 1800 | `--timeout 3600` |
| `--allowed-tools` | Comma-separated tool list | All tools | `--allowed-tools "Edit,Read"` |
| `--output-format` | Output format | default | `--output-format stream-json` |

### Session Management

```bash
# List recent sessions
./manage.sh --action session

# Resume a specific session
./manage.sh --action session --session-id abc123def456

# Resume with additional turns
./manage.sh --action session --session-id abc123def456 --max-turns 20
```

### Settings & Configuration

```bash
# View current settings
./manage.sh --action settings

# View logs
./manage.sh --action logs
```

## 🔍 Advanced Usage Patterns

### Complex Refactoring

```bash
./manage.sh --action batch \
  --prompt "Refactor all API endpoints to use async/await" \
  --allowed-tools "Edit,Read,Grep,Bash" \
  --max-turns 100 \
  --timeout 7200 \
  --output-format stream-json
```

### Code Review Automation

```bash
./manage.sh --action run \
  --prompt "Review this code for security issues and best practices" \
  --allowed-tools "Read,Grep" \
  --max-turns 10
```

### Documentation Generation

```bash
./manage.sh --action batch \
  --prompt "Generate comprehensive documentation for all public APIs" \
  --allowed-tools "Read,Write,Edit" \
  --max-turns 50
```

### System Administration (Dangerous)

```bash
# EXTREME CAUTION: Only use in isolated environments
./manage.sh --action batch \
  --prompt "Optimize system configuration files" \
  --dangerously-skip-permissions yes \
  --max-turns 50 \
  --timeout 3600
```

## 🔄 Composability with Unix Tools

Claude Code works seamlessly with Unix pipes and standard tools:

```bash
# Monitor and analyze logs
tail -f server.log | claude -p "Summarize any errors"

# Code analysis pipeline
find . -name "*.js" | claude -p "Find security issues"

# Documentation generation
claude -p "Document all functions" < utils.js > docs.md

# Test result analysis
npm test 2>&1 | claude -p "Explain any failing tests"

# Git integration
git diff HEAD~1 | claude -p "Summarize changes"
```

## 📊 Status and Health Checks

### Diagnostic Commands

```bash
# Full system diagnostics
claude doctor

# Installation status
./manage.sh --action status

# Version information
claude --version

# Configuration check  
./manage.sh --action settings
```

### Exit Codes

| Exit Code | Meaning |
|-----------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Authentication error |
| 3 | Network error |
| 4 | Permission denied |
| 5 | Subscription limit reached |

## 🔐 Security Considerations

### Permission System

By default, Claude Code requires confirmation for potentially dangerous operations:
- File modifications
- Command execution
- System-level changes

### Dangerous Permissions Override

**⚠️ WARNING**: The `--dangerously-skip-permissions` flag disables ALL safety checks.

**Safe Usage**:
```bash
# Restrict tools even when skipping permissions
./manage.sh --action run --prompt "Task" \
  --dangerously-skip-permissions yes \
  --allowed-tools "Edit,Read"
```

**Never Use In**:
- Production environments
- Systems with sensitive data
- Untrusted code repositories
- Shared development environments

### Best Practices

1. **Tool Restriction**: Always use `--allowed-tools` to limit capabilities
2. **Turn Limits**: Set reasonable `--max-turns` to prevent runaway operations
3. **Timeouts**: Use `--timeout` for batch operations
4. **Review Changes**: Always review modifications before committing
5. **Version Control**: Work in version-controlled environments

## 🚀 Integration Examples

### With CI/CD

```bash
# Code review in CI pipeline
./manage.sh --action run \
  --prompt "Review changes for security and best practices" \
  --allowed-tools "Read,Grep" \
  --max-turns 5 \
  --timeout 300
```

### With Development Workflow

```bash
# Pre-commit hook
git diff --cached | claude -p "Review staged changes"

# Automated testing
./manage.sh --action run \
  --prompt "Generate unit tests for modified files" \
  --allowed-tools "Read,Write" \
  --max-turns 20
```

### With Other AI Resources

```bash
# Analysis pipeline with Ollama
ANALYSIS=$(./manage.sh --action run --prompt "Analyze code structure")
echo "$ANALYSIS" | curl -X POST http://localhost:11434/api/generate
```

This API reference provides comprehensive coverage of Claude Code's capabilities for both interactive and programmatic usage.