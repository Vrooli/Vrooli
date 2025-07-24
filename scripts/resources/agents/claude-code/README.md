# Claude Code - Anthropic's Official CLI for Claude

Claude Code is a powerful command-line interface that brings Claude's capabilities directly to your terminal. It provides deep codebase awareness and the ability to edit files and run commands directly in your development environment.

## Overview

Claude Code embeds Claude Opus 4—the same model Anthropic's researchers and engineers use—right in your terminal. It offers:

- **Deep codebase awareness** - Understands your entire project context
- **Direct file editing** - Can modify files in your project
- **Command execution** - Runs commands and scripts
- **Composable interface** - Works with pipes and shell scripts
- **Automatic updates** - Keeps itself up to date

## Prerequisites

- **Node.js 18 or newer** - Required for npm installation
- **npm** - Node package manager (comes with Node.js)
- **Claude Pro or Max subscription** - For full functionality

## Installation

### Quick Install

```bash
# Using the resource manager
./scripts/resources/index.sh --action install --resources claude-code

# Or directly with the manage script
./scripts/resources/agents/claude-code/manage.sh --action install
```

### Manual Install

If you prefer to install manually:

```bash
npm install -g @anthropic-ai/claude-code
```

### Verify Installation

```bash
# Check installation status
./scripts/resources/agents/claude-code/manage.sh --action status

# Or directly
claude --version
claude doctor
```

## Usage

### Getting Started

1. **Navigate to your project**:
   ```bash
   cd /path/to/your/project
   ```

2. **Start Claude Code**:
   ```bash
   claude
   ```

3. **Login** (first time only):
   - You'll be prompted to authenticate
   - Use your Claude account credentials

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

### Pro Tips

- Use **Tab** for command completion
- Press **↑** for command history
- Type **/** to see available slash commands
- Use **Ctrl+C** to interrupt long-running operations

## Subscription Plans

### Pro Plan ($20/month)
- Ideal for light coding work
- Suitable for small repositories
- Basic usage limits

### Max Plan ($100/month)
- Heavy usage capabilities
- ~225 messages every 5 hours in web/mobile apps
- ~50-200 prompts every 5 hours in Claude Code
- Priority access during high demand

## Features

### File Editing
Claude Code can directly edit files in your project:
- Understands file structure
- Makes contextual changes
- Preserves formatting
- Creates new files when needed

### Command Execution
Execute commands directly:
- Run tests
- Build projects
- Install dependencies
- Check logs

### Codebase Understanding
Deep awareness of your project:
- Understands project structure
- Recognizes frameworks and libraries
- Follows coding conventions
- Suggests best practices

### Composability
Works with Unix pipes and scripts:
```bash
# Monitor logs
tail -f server.log | claude -p "Summarize any errors"

# Analyze code
find . -name "*.js" | claude -p "Find security issues"

# Generate documentation
claude -p "Document all functions" < utils.js > docs.md
```

## Management Script

The management script provides convenient commands for both installation and usage:

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

### Running Claude Code

```bash
# Run a single prompt
./manage.sh --action run --prompt "Explain this Python function"

# Run with specific max turns
./manage.sh --action run --prompt "Write unit tests" --max-turns 10

# Run with allowed tools
./manage.sh --action run --prompt "Fix this bug" --allowed-tools "Edit,Bash"

# Run with JSON output
./manage.sh --action run --prompt "Analyze code" --output-format stream-json
```

### Batch Processing

```bash
# Run in batch mode with extended timeout
./manage.sh --action batch --prompt "Refactor all test files" --max-turns 50 --timeout 3600

# Batch with specific tools
./manage.sh --action batch --prompt "Update documentation" \
  --allowed-tools "Edit,Read,Write" \
  --max-turns 30
```

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

### Advanced Usage Examples

```bash
# Complex refactoring task
./manage.sh --action batch \
  --prompt "Refactor all API endpoints to use async/await" \
  --allowed-tools "Edit,Read,Grep,Bash" \
  --max-turns 100 \
  --timeout 7200 \
  --output-format stream-json

# Code review automation
./manage.sh --action run \
  --prompt "Review this code for security issues and best practices" \
  --allowed-tools "Read,Grep" \
  --max-turns 10

# Documentation generation
./manage.sh --action batch \
  --prompt "Generate comprehensive documentation for all public APIs" \
  --allowed-tools "Read,Write,Edit" \
  --max-turns 50
```

## Configuration

Claude Code stores configuration in your home directory:
- Credentials are securely stored
- Settings persist across sessions
- Automatic authentication refresh

## Troubleshooting

### Installation Issues

**Node.js version too old**:
```bash
# Check Node.js version
node --version

# Update Node.js from https://nodejs.org/
```

**npm not found**:
```bash
# npm should come with Node.js
# Reinstall Node.js if missing
```

**Permission errors**:
```bash
# Use npm with proper permissions
npm install -g @anthropic-ai/claude-code

# Or fix npm permissions
npm config set prefix ~/.npm-global
echo 'export PATH=~/.npm-global/bin:$PATH' >> ~/.bashrc
source ~/.bashrc
```

### Authentication Issues

**Login problems**:
- Ensure you have a valid Claude Pro or Max subscription
- Check your internet connection
- Try logging out and back in
- Run `claude doctor` for diagnostics

**Credential errors**:
- Credentials are stored securely on your system
- Clear credentials and re-login if needed
- Check account status at claude.ai

### Usage Issues

**Command not found**:
```bash
# Ensure Claude is in PATH
which claude

# Reinstall if needed
npm install -g @anthropic-ai/claude-code
```

**Slow responses**:
- Check internet connection
- Verify subscription limits haven't been reached
- Try during off-peak hours
- Consider upgrading to Max plan

## Security

- **Credentials**: Stored securely on your local system
- **File access**: Only accesses files you explicitly work with
- **Command execution**: Requires confirmation for potentially dangerous operations
- **Network**: All communication is encrypted

## Enterprise Options

For enterprise deployments:
- **Amazon Bedrock**: Use with existing AWS infrastructure
- **Google Vertex AI**: Integrate with GCP services
- **API Access**: Direct API integration available

## Best Practices

1. **Start in project root**: Claude Code works best when started from your project's root directory
2. **Use .gitignore**: Claude respects .gitignore patterns
3. **Be specific**: Clear, specific prompts yield better results
4. **Review changes**: Always review code changes before committing
5. **Use version control**: Commit working code before major changes

## Updates

Claude Code automatically updates itself to ensure you have:
- Latest features
- Security patches
- Performance improvements
- Bug fixes

To manually check for updates:
```bash
claude doctor
```

## Support

- **Documentation**: https://docs.anthropic.com/claude-code
- **Help Center**: https://support.anthropic.com
- **Community**: Join the Claude community for tips and discussions
- **Feedback**: Report issues at https://github.com/anthropics/claude-code/issues

## Integration with Vrooli

When installed through the Vrooli resource manager:
- Automatically configured in `~/.vrooli/resources.local.json`
- Available as a CLI tool for development tasks
- Can be used alongside other AI resources
- No port allocation needed (CLI tool only)

## Uninstallation

To remove Claude Code:

```bash
# Using resource manager
./scripts/resources/index.sh --action uninstall --resources claude-code

# Or directly
./scripts/resources/agents/claude-code/manage.sh --action uninstall

# Manual uninstall
npm uninstall -g @anthropic-ai/claude-code
```

---

Claude Code brings the power of Claude directly to your development workflow, making it an essential tool for modern software development.