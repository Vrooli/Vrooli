# Claude Code Configuration Guide

This guide covers installation, configuration, and optimization of Claude Code for development workflows.

## 📋 Prerequisites

### System Requirements

- **Node.js 18 or newer** - Required for npm installation
- **npm** - Node package manager (comes with Node.js)
- **Claude Pro or Max subscription** - For full functionality

### Subscription Plans

#### Pro Plan ($20/month)
- Ideal for light coding work
- Suitable for small repositories
- Basic usage limits

#### Max Plan ($100/month)
- Heavy usage capabilities
- ~225 messages every 5 hours in web/mobile apps
- ~50-200 prompts every 5 hours in Claude Code
- Priority access during high demand

## 🚀 Installation Methods

### Resource Manager Installation (Recommended)

```bash
# Using the resource manager
./scripts/resources/index.sh --action install --resources claude-code

# Or directly with the manage script
./scripts/resources/agents/claude-code/manage.sh --action install
```

### Manual Installation

```bash
# Global npm installation
npm install -g @anthropic-ai/claude-code

# Verify installation
claude --version
claude doctor
```

### Alternative Installation Methods

#### Using npm with proper permissions
```bash
# Configure npm global directory
npm config set prefix ~/.npm-global
echo 'export PATH=~/.npm-global/bin:$PATH' >> ~/.bashrc
source ~/.bashrc

# Install Claude Code
npm install -g @anthropic-ai/claude-code
```

#### System-wide installation (requires sudo)
```bash
sudo npm install -g @anthropic-ai/claude-code
```

## ⚙️ Configuration Storage

### Configuration Locations

- **Main Configuration**: Stored in user home directory
- **Credentials**: Securely stored using system keychain
- **Session Data**: Temporary session information
- **Logs**: Application logs and debugging information

### Vrooli Integration

When installed through Vrooli resource manager:

```json
{
  "agents": {
    "claude-code": {
      "enabled": true,
      "type": "cli",
      "installation": {
        "method": "npm-global",
        "package": "@anthropic-ai/claude-code"
      },
      "status": "available"
    }
  }
}
```

## 🔐 Authentication Setup

### Initial Authentication

1. **Start Claude Code**:
   ```bash
   claude
   ```

2. **Follow Login Prompts**:
   - You'll be redirected to authenticate via web browser
   - Use your Claude Pro or Max account credentials
   - Authentication tokens are stored securely

3. **Verify Authentication**:
   ```bash
   claude doctor
   ```

### Authentication Troubleshooting

#### Login Problems
- Ensure you have a valid Claude Pro or Max subscription
- Check your internet connection
- Try logging out and back in
- Verify account status at claude.ai

#### Credential Management
```bash
# Check authentication status
claude doctor

# Clear credentials and re-login if needed
# (Credentials are cleared automatically during re-authentication)
```

## 🎛️ Advanced Configuration

### Environment Variables

```bash
# Set custom configuration directory
export CLAUDE_CONFIG_DIR="$HOME/.config/claude"

# Enable debug logging
export CLAUDE_DEBUG=true

# Set custom timeout (seconds)
export CLAUDE_TIMEOUT=120

# Disable automatic updates
export CLAUDE_NO_AUTO_UPDATE=true
```

### Project-Specific Configuration

#### .clauderc File
Create a `.clauderc` file in your project root:

```json
{
  "defaultTools": ["Edit", "Read", "Bash"],
  "maxTurns": 10,
  "timeout": 1800,
  "outputFormat": "default",
  "safetyChecks": true
}
```

#### .gitignore Integration
Claude Code respects `.gitignore` patterns automatically.

### MCP (Model Context Protocol) Templates

The resource includes MCP configuration templates:

#### Basic Local Configuration (`templates/mcp-local.json`)
```json
{
  "mcpServers": {
    "filesystem": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-filesystem", "/path/to/directory"]
    }
  }
}
```

#### Advanced Configuration (`templates/mcp-advanced.json`)
```json
{
  "mcpServers": {
    "filesystem": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-filesystem", "/path/to/directory"]
    },
    "database": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-sqlite", "/path/to/database.db"]
    }
  }
}
```

#### Project-Specific Template (`templates/mcp-project.json`)
```json
{
  "mcpServers": {
    "project": {
      "command": "node",
      "args": ["./scripts/mcp-server.js"],
      "env": {
        "PROJECT_ROOT": "."
      }
    }
  }
}
```

## 🔧 Performance Optimization

### Response Time Optimization

```bash
# Use specific tools to reduce overhead
./manage.sh --action run --prompt "Task" --allowed-tools "Read,Edit"

# Set reasonable turn limits
./manage.sh --action run --prompt "Task" --max-turns 5

# Use shorter timeouts for quick tasks
./manage.sh --action run --prompt "Task" --timeout 300
```

### Memory Usage

```bash
# Monitor memory usage
top -p $(pgrep -f claude)

# Clear session history periodically
claude doctor  # Includes cache cleanup
```

### Network Optimization

- **Stable Connection**: Ensure reliable internet connectivity
- **Off-Peak Usage**: Use during lower-demand periods
- **Subscription Tier**: Consider Max plan for priority access

## 🔒 Security Configuration

### Permission System

Default security settings:
- **File Access**: Prompts before modifying files
- **Command Execution**: Requires confirmation for shell commands
- **System Operations**: Blocks dangerous system-level operations

### Safe Configuration

```bash
# Recommended safe settings
./manage.sh --action run \
  --prompt "Your task" \
  --allowed-tools "Read,Edit" \
  --max-turns 10 \
  --timeout 1800
```

### Enterprise Security

For enterprise environments:

```bash
# Restricted tool access
export CLAUDE_ALLOWED_TOOLS="Read,Edit"

# Disable dangerous operations
export CLAUDE_DISABLE_BASH=true

# Enable audit logging
export CLAUDE_AUDIT_LOG=true
export CLAUDE_AUDIT_PATH="/var/log/claude/"
```

## 🏢 Enterprise Integration

### Amazon Bedrock

For AWS environments:
```bash
# Configure Bedrock endpoint
export CLAUDE_ENDPOINT="bedrock"
export AWS_REGION="us-east-1"
```

### Google Vertex AI

For GCP environments:
```bash
# Configure Vertex AI endpoint
export CLAUDE_ENDPOINT="vertex"
export GOOGLE_CLOUD_PROJECT="your-project-id"
```

### Direct API Integration

```bash
# Use direct API access
export CLAUDE_API_KEY="your-api-key"
export CLAUDE_ENDPOINT="api"
```

## 🔄 Update Management

### Automatic Updates

Claude Code automatically updates by default:
- Latest features
- Security patches
- Performance improvements
- Bug fixes

### Manual Update Control

```bash
# Disable automatic updates
export CLAUDE_NO_AUTO_UPDATE=true

# Manual update check
claude doctor

# Force update check
npm update -g @anthropic-ai/claude-code
```

## 🐛 Troubleshooting Configuration

### Installation Issues

#### Node.js Version Problems
```bash
# Check Node.js version
node --version

# Update Node.js from https://nodejs.org/
# Minimum requirement: Node.js 18
```

#### Permission Errors
```bash
# Fix npm permissions
npm config set prefix ~/.npm-global
echo 'export PATH=~/.npm-global/bin:$PATH' >> ~/.bashrc
source ~/.bashrc

# Reinstall with proper permissions
npm install -g @anthropic-ai/claude-code
```

#### Command Not Found
```bash
# Check if Claude is in PATH
which claude

# Add to PATH if needed
echo 'export PATH=$PATH:/usr/local/bin' >> ~/.bashrc
source ~/.bashrc
```

### Configuration Validation

```bash
# Full system diagnostics
claude doctor

# Check configuration status
./manage.sh --action settings

# Verify installation
./manage.sh --action status
```

### Reset Configuration

```bash
# Reset to defaults (requires re-authentication)
rm -rf ~/.config/claude  # or your custom config directory

# Reinstall if needed
./manage.sh --action install --force yes
```

## 📊 Monitoring and Logging

### Enable Debug Logging

```bash
# Environment variable
export CLAUDE_DEBUG=true

# Check debug logs
./manage.sh --action logs
```

### Performance Monitoring

```bash
# Monitor resource usage
./manage.sh --action status --verbose

# Check response times
time claude -p "Simple test prompt"
```

### Usage Analytics

Track your usage patterns:
- Turn count per session
- Tool usage frequency
- Response time metrics
- Error rates

This configuration guide ensures optimal Claude Code setup for your development environment and workflow requirements.