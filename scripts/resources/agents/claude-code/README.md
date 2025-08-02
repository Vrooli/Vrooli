# Claude Code - Anthropic's Official CLI for Claude

Claude Code is a powerful command-line interface that brings Claude's capabilities directly to your terminal. It provides deep codebase awareness and the ability to edit files and run commands directly in your development environment.

## Quick Reference
- **Category**: Agents
- **Type**: CLI Tool
- **Installation**: npm package (`@anthropic-ai/claude-code`)
- **API Docs**: [Complete API Reference](docs/API.md)
- **Status**: Functional (requires TTY for some operations)

## When to Use
- **AI pair programming** and code review assistance
- **Automated code analysis** and refactoring  
- **Development workflow** assistance and automation
- **Code documentation** generation and maintenance

**Alternative**: Direct IDE integration, manual code review, other AI development tools

## Important Notes
- **TTY Requirements**: Some operations require interactive terminal access
- **Authentication**: Requires Claude Pro or Max subscription and login
- **Automation**: Use `--print` mode and health-check actions for non-interactive environments

## 🚀 Quick Start

```bash
# Install Claude Code
./manage.sh --action install

# Check installation
./manage.sh --action status

# Start interactive session
claude

# Run single prompt
./manage.sh --action run --prompt "Explain this code"

# Batch processing
./manage.sh --action batch --prompt "Update documentation" --max-turns 20
```

## Prerequisites
- **Node.js 18 or newer** - Required for npm installation
- **Claude Pro or Max subscription** - For full functionality
- **npm** - Node package manager (comes with Node.js)

## 📚 Documentation

- 📖 [**Complete API Reference**](docs/API.md) - CLI commands, management script API, advanced usage patterns
- ⚙️ [**Configuration Guide**](docs/CONFIGURATION.md) - Installation, authentication, performance optimization
- 🔧 [**Troubleshooting**](docs/TROUBLESHOOTING.md) - Common issues, diagnostics, and solutions
- 📂 [**Examples**](examples/README.md) - Practical usage examples and workflows

## Core Features

### Deep Codebase Awareness
- Understands your entire project context
- Recognizes frameworks and libraries
- Follows coding conventions
- Suggests best practices

### Direct File Editing
- Makes contextual changes
- Preserves formatting
- Creates new files when needed
- Integrates with version control

### Command Execution
- Run tests and builds
- Install dependencies
- Execute scripts
- Monitor logs

### Composable Interface
```bash
# Monitor logs with AI analysis
tail -f server.log | claude -p "Summarize any errors"

# Analyze code patterns
find . -name "*.js" | claude -p "Find security issues"

# Generate documentation
claude -p "Document all functions" < utils.js > docs.md
```

## Service Management

```bash
# Installation and setup
./manage.sh --action install
./manage.sh --action status
./manage.sh --action info

# Running prompts
./manage.sh --action run --prompt "Your task"
./manage.sh --action batch --prompt "Complex task" --max-turns 50

# Session management
./manage.sh --action session
./manage.sh --action session --session-id abc123

# Configuration
./manage.sh --action settings
./manage.sh --action logs
```

## Subscription Plans

### Pro Plan ($20/month)
- Light coding work
- Small repositories
- Basic usage limits

### Max Plan ($100/month)
- Heavy usage capabilities
- ~50-200 prompts every 5 hours in Claude Code
- Priority access during high demand

## Integration Examples

### With Version Control
```bash
# Review staged changes
git diff --cached | claude -p "Review staged changes"

# Analyze commit history
git log --oneline -10 | claude -p "Summarize recent changes"
```

### With CI/CD
```bash
# Pre-commit code review
./manage.sh --action run --prompt "Review for security issues" --allowed-tools "Read,Grep"

# Automated testing
./manage.sh --action run --prompt "Generate unit tests" --allowed-tools "Read,Write"
```

### With Other AI Resources
```bash
# Analysis pipeline with Ollama
ANALYSIS=$(./manage.sh --action run --prompt "Analyze code structure")
echo "$ANALYSIS" | curl -X POST http://localhost:11434/api/generate
```

## Security Features

- **Permission System**: Requires confirmation for potentially dangerous operations
- **Tool Restrictions**: Limit capabilities with `--allowed-tools`
- **Safe Defaults**: Conservative settings for file modifications
- **Encrypted Communication**: All communication with Claude is encrypted

### ⚠️ Dangerous Operations
The `--dangerously-skip-permissions` flag disables all safety checks:
- Only use in isolated test environments
- Never use in production
- Always combine with `--allowed-tools` restrictions

## Enterprise Options

- **Amazon Bedrock**: Use with existing AWS infrastructure
- **Google Vertex AI**: Integrate with GCP services
- **Direct API**: API access for custom integrations

## Best Practices

1. **Start in project root** - Claude Code works best from your project's root directory
2. **Use .gitignore** - Claude respects .gitignore patterns automatically
3. **Be specific** - Clear, specific prompts yield better results
4. **Review changes** - Always review code changes before committing
5. **Use version control** - Commit working code before major changes

## Automatic Updates

Claude Code automatically updates itself to ensure you have:
- Latest features and capabilities
- Security patches and fixes
- Performance improvements
- Bug fixes

Check for updates manually: `claude doctor`

For detailed usage instructions, configuration options, and troubleshooting, see the documentation links above.