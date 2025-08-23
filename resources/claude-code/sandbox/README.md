# Claude Code Sandbox Environment

A secure, isolated Docker container environment for testing Claude Code capabilities without risking your main system or codebase.

## 🎯 Purpose

The Claude Code sandbox provides:
- **Complete isolation** - Test claude-code without access to your main filesystem
- **Subscription authentication** - Uses your Claude subscription with separate credentials
- **Resource limits** - Prevents runaway processes from affecting your system
- **Network isolation** - Optional network access control for security
- **Non-root execution** - Runs as unprivileged user for additional security

## 🚀 Quick Start

### 1. Initial Setup

First time setup creates isolated authentication for the sandbox:

```bash
# From the claude-code resource directory
./sandbox/claude-sandbox.sh setup
```

This will:
- Create `~/.claude-sandbox` directory for isolated credentials
- Authenticate using your Claude subscription
- Store credentials separately from your main Claude installation

### 2. Run Interactive Session

Start an interactive Claude session in the sandbox:

```bash
./sandbox/claude-sandbox.sh
# or
./manage.sh --action sandbox
```

### 3. Run Specific Commands

Execute Claude commands in the sandbox:

```bash
# Run a single prompt
./sandbox/claude-sandbox.sh run -p "Analyze this test code" --print

# Use with management script
./manage.sh --action sandbox --sandbox-command run --prompt "Fix bugs in test.js"
```

## 📁 Directory Structure

```
sandbox/
├── docker/
│   ├── Dockerfile          # Secure container definition
│   └── docker-compose.yml  # Container orchestration
├── test-files/            # Your test code goes here
├── docs/                  # Additional documentation
├── claude-sandbox.sh      # Main wrapper script
└── README.md             # This file
```

## 🔒 Security Features

### Filesystem Isolation
- Container can only access `/workspace` (mapped to `test-files/`)
- Credentials mounted read-only
- Temporary filesystem for claude operations

### Process Isolation
- Runs as non-root user (uid 1001)
- Dropped capabilities (only essential ones retained)
- No new privileges flag set

### Network Isolation
- Internal bridge network by default
- No external internet access unless explicitly enabled
- Protects against accidental data exfiltration

### Resource Limits
- CPU: 2 cores max (0.5 reserved)
- Memory: 2GB max (512MB reserved)
- Temporary storage: 100MB

## 🛠️ Usage Examples

### Basic Testing

1. **Add test files**:
```bash
# Copy files you want to test
cp ~/myproject/example.js ./sandbox/test-files/
```

2. **Run analysis**:
```bash
./manage.sh --action sandbox --prompt "Analyze example.js for security issues"
```

### Advanced Workflows

1. **Batch processing in sandbox**:
```bash
./sandbox/claude-sandbox.sh run \
  --batch \
  --prompt "Refactor all test files" \
  --max-turns 50
```

2. **Safe experimentation**:
```bash
# Test with dangerous permissions in isolated environment
./sandbox/claude-sandbox.sh run \
  --dangerously-skip-permissions \
  --prompt "Experimental code generation"
```

## 🔧 Configuration

### Environment Variables

- `CLAUDE_SANDBOX_CONFIG`: Directory for sandbox credentials (default: `~/.claude-sandbox`)
- `CLAUDE_CONFIG_DIR`: Set by container to use sandbox credentials

### Docker Compose Options

Edit `docker/docker-compose.yml` to:
- Enable external network access (uncomment network section)
- Adjust resource limits
- Mount additional directories (use with caution)

## 🚨 Important Notes

### Authentication
- Sandbox uses your Claude subscription but with **separate credentials**
- First run requires interactive authentication
- Credentials persist between sessions in `~/.claude-sandbox`

### Limitations
- No access to main filesystem
- Network restrictions may affect some operations
- Limited to resources defined in container

### Best Practices
1. Only copy necessary files to `test-files/`
2. Review generated code before using in production
3. Clean up test files regularly
4. Monitor resource usage for long-running tasks

## 🔍 Troubleshooting

### Authentication Issues
```bash
# Re-authenticate if needed
./sandbox/claude-sandbox.sh setup
```

### Container Issues
```bash
# Check container status
docker ps -a | grep claude-sandbox

# View container logs
docker logs claude-code-sandbox

# Stop and clean up
./sandbox/claude-sandbox.sh stop
./sandbox/claude-sandbox.sh cleanup
```

### Permission Errors
- Ensure Docker daemon is running
- Check user has Docker permissions: `docker ps`
- Verify file permissions in test-files directory

## 🎯 Use Cases

### Safe Testing
- Test Claude Code on untrusted code
- Experiment with `--dangerously-skip-permissions`
- Try new features without risk

### Development
- Prototype AI-assisted workflows
- Test automation scripts
- Validate Claude Code capabilities

### Security Analysis
- Analyze suspicious code safely
- Test security tools integration
- Audit AI-generated code

## 📚 Advanced Topics

### Custom Docker Images
Build with additional tools:
```dockerfile
FROM vrooli/claude-code-sandbox:latest
RUN apk add --no-cache python3 py3-pip
```

### Integration with CI/CD
```yaml
# Example GitHub Action
- name: Claude Code Security Check
  run: |
    ./sandbox/claude-sandbox.sh run \
      --prompt "Security audit PR changes" \
      --allowed-tools "Read,Grep"
```

### Multi-Container Workflows
Combine with other Vrooli resources:
```bash
# Start sandbox with other services
docker compose -f docker/docker-compose.yml \
  -f ../../storage/postgres/docker-compose.yml \
  up
```

## 🤝 Contributing

To improve the sandbox:
1. Test thoroughly in isolated environment
2. Document security implications
3. Follow Vrooli resource standards
4. Submit PR with test results

## 📋 Checklist

- [ ] Authenticated sandbox (run `setup`)
- [ ] Added test files to `test-files/`
- [ ] Tested basic commands
- [ ] Reviewed security settings
- [ ] Understood limitations

Remember: The sandbox is for **testing only**. Always review and validate any code before using in production environments.