# Claude Code Testing Sandbox Requirements

## ⚠️ Important: Integration Tests Run in Sandbox Mode

Claude Code is unique among Vrooli resources because it can actively modify the filesystem and execute commands. To prevent unintended changes during testing, **all Claude Code integration tests MUST run in sandbox mode**.

## 🔒 Why Sandbox Mode is Required

Claude Code differs from other resources:
- **Other resources** (Ollama, PostgreSQL, Redis) provide services without filesystem access
- **Claude Code** can:
  - Execute arbitrary prompts that modify files
  - Run system commands
  - Delete or overwrite existing files
  - Access sensitive data

Without sandboxing, integration tests could accidentally:
- Modify test framework files
- Delete important project files
- Execute unintended commands
- Compromise system security

## 🚀 How Sandbox Mode Works

The integration test automatically:
1. **Detects test environment** using environment variables
2. **Forces sandbox mode** for all Claude Code operations
3. **Skips dangerous operations** like direct CLI calls
4. **Uses safe alternatives** for status checks

### Environment Detection
```bash
# Test runs in sandbox if ANY of these are true:
- VROOLI_TEST=true
- CI=true
- GITHUB_ACTIONS=true
- FORCE_SANDBOX=true
- Always (default for safety)
```

### Sandbox Features
- ✅ Complete filesystem isolation (only test-files/ accessible)
- ✅ Docker container with resource limits
- ✅ Non-root execution
- ✅ Network isolation (optional)
- ✅ Separate authentication credentials

## 📋 Test Modifications for Sandbox

### 1. Status Checks
```bash
# Instead of:
claude-code --version

# Use:
manage.sh --action test-safe
```

### 2. Availability Tests
```bash
# Instead of:
which claude-code

# Check:
- Management script exists
- Sandbox script exists
- Docker is available
```

### 3. Safe Test Actions
New `test-safe` action provides:
- Installation verification
- Configuration checks
- Sandbox availability
- NO prompt execution

## 🧪 Running Tests

### Standard Test Execution
```bash
# From project root
./__test/resources/run.sh --resource claude-code

# Test automatically uses sandbox
```

### Manual Test with Sandbox
```bash
# Force sandbox mode
FORCE_SANDBOX=true ./__test/resources/single/agents/claude-code.test.sh
```

### Verify Sandbox Usage
Look for this message in test output:
```
🔒 Claude Code integration tests ALWAYS run in sandbox mode
   This prevents accidental file system modifications during testing
```

## 🔧 Sandbox Setup Requirements

### Prerequisites
1. **Docker installed and running**
   ```bash
   docker --version
   docker ps
   ```

2. **Claude Code installed**
   ```bash
   npm install -g @anthropic-ai/claude-code
   ```

3. **Sandbox built**
   ```bash
   cd resources/claude-code/sandbox
   ./claude-sandbox.sh build
   ```

### First-Time Setup
```bash
# Build sandbox image
./resources/claude-code/sandbox/claude-sandbox.sh build

# Optional: Set up sandbox authentication
./resources/claude-code/sandbox/claude-sandbox.sh setup
```

## 🚨 Troubleshooting

### "Docker not found"
- Install Docker
- Ensure Docker daemon is running
- Add user to docker group: `sudo usermod -aG docker $USER`

### "Sandbox script not found"
- Run tests from Vrooli project root
- Ensure full project is cloned
- Check file permissions

### "Authentication required"
- Sandbox auth is optional for basic tests
- Run `./claude-sandbox.sh setup` if needed
- Uses separate credentials from main Claude

## 📝 Writing New Tests

When adding Claude Code tests:

### DO ✅
- Use `manage.sh --action test-safe` for status checks
- Test management script functionality
- Verify configuration and setup
- Use sandbox for any prompt execution

### DON'T ❌
- Call `claude-code` CLI directly
- Execute prompts without sandbox
- Modify files outside test environment
- Skip sandbox safety checks

## 🔍 Test Coverage

Current safe test coverage:
- ✅ Installation verification
- ✅ Configuration detection
- ✅ Management script functions
- ✅ MCP registration status
- ✅ Error handling
- ✅ Help functionality

Sandbox-only tests:
- 🔒 Prompt execution
- 🔒 File modifications
- 🔒 Command execution
- 🔒 Batch operations

## 🤝 Contributing

When modifying Claude Code tests:
1. Always maintain sandbox safety
2. Document any new test actions
3. Verify tests work in CI environment
4. Never bypass sandbox without explicit approval

---

**Remember**: The sandbox protects both the test environment and developers' systems. It's not optional - it's a critical safety feature for testing AI code generation tools.