# Claude Code Sandbox Testing Guide

This guide walks through testing the Claude Code sandbox to ensure it's working correctly.

## 🧪 Testing Checklist

### 1. Prerequisites Check

```bash
# Verify Docker is installed and running
docker --version
docker ps

# Check Docker permissions
docker run hello-world

# Verify Node.js for Claude Code
node --version  # Should be 18+
```

### 2. Build Sandbox Image

```bash
cd resources/claude-code/sandbox
./claude-sandbox.sh build
```

Expected output:
- ✅ "Building Claude Code sandbox image..."
- ✅ "Sandbox image built successfully"

### 3. Set Up Authentication

```bash
# This will create ~/.claude-sandbox directory
./claude-sandbox.sh setup
```

Expected behavior:
1. Creates `~/.claude-sandbox` directory
2. Prompts for Claude login
3. Opens browser for authentication
4. Saves credentials to sandbox config

**Verify:**
```bash
ls -la ~/.claude-sandbox/
# Should see .credentials.json and other config files
```

### 4. Test Interactive Session

```bash
./claude-sandbox.sh interactive
# or simply
./claude-sandbox.sh
```

Expected behavior:
- Container starts with sandbox warning
- Claude prompt appears
- Can interact with Claude normally
- Exit with Ctrl+D or 'exit'

### 5. Test File Access Restrictions

```bash
# In sandbox, try to access outside workspace
./claude-sandbox.sh run -p "List files in /etc" --print
```

Expected: Should only see test-files content, not system files

```bash
# Test with actual test files
./claude-sandbox.sh run -p "Analyze example.js" --print
```

Expected: Should analyze the example.js file in test-files/

### 6. Test Resource Limits

```bash
# Check resource constraints are applied
docker stats claude-code-sandbox
```

Expected limits:
- CPU: ≤ 200% (2 cores)
- Memory: ≤ 2GB

### 7. Test Network Isolation

```bash
# From within sandbox
./claude-sandbox.sh run -p "Try to access google.com" --print
```

Expected: Network requests should fail (if internal network is enabled)

### 8. Test Management Script Integration

```bash
# Test via manage.sh
./manage.sh --action sandbox
./manage.sh --action sandbox --sandbox-command run --prompt "Hello"
./manage.sh --action sandbox --sandbox-command stop
```

### 9. Test Safety Features

```bash
# Test with dangerous permissions flag
./claude-sandbox.sh run \
  --dangerously-skip-permissions \
  --prompt "Create a file named test.txt with 'Hello'" \
  --print
```

Expected: File created only in test-files/, not elsewhere

### 10. Test Cleanup

```bash
# Stop container
./claude-sandbox.sh stop

# Verify stopped
docker ps -a | grep claude-code-sandbox

# Clean up
./claude-sandbox.sh cleanup
```

## 🔍 Verification Tests

### Security Verification

1. **User Isolation**:
```bash
docker exec claude-code-sandbox whoami
# Expected: claude (not root)
```

2. **Filesystem Isolation**:
```bash
docker exec claude-code-sandbox ls /
# Should not see host filesystem
```

3. **Capability Restrictions**:
```bash
docker inspect claude-code-sandbox | grep -A5 CapDrop
# Should show ALL capabilities dropped
```

### Functionality Verification

1. **Claude Tools Available**:
```bash
./manage.sh --action sandbox --prompt "What tools do you have available?"
```

2. **File Operations**:
```bash
# Create test file
echo "test content" > test-files/verify.txt

# Test read
./manage.sh --action sandbox --prompt "Read verify.txt"

# Test edit
./manage.sh --action sandbox --prompt "Add a line to verify.txt"
```

3. **Code Analysis**:
```bash
./manage.sh --action sandbox --prompt "Analyze example.js for bugs"
```

## 📊 Performance Testing

### Response Time Test
```bash
time ./manage.sh --action sandbox \
  --sandbox-command run \
  --prompt "Say hello" \
  --print
```

### Concurrent Operations
```bash
# Run multiple sandbox instances
for i in {1..3}; do
  ./claude-sandbox.sh run --prompt "Count to $i" &
done
wait
```

## 🐛 Troubleshooting Tests

### Authentication Issues
```bash
# Test credential mounting
docker run --rm \
  -v ~/.claude-sandbox:/home/claude/.claude:ro \
  vrooli/claude-code-sandbox:latest \
  ls -la /home/claude/.claude/
```

### Container Issues
```bash
# Debug mode
docker compose -f docker/docker-compose.yml logs
docker compose -f docker/docker-compose.yml exec claude-sandbox /bin/bash
```

### Permission Issues
```bash
# Test file permissions
ls -la ~/.claude-sandbox/
stat ~/.claude-sandbox/.credentials.json
```

## ✅ Success Criteria

The sandbox is working correctly if:

1. ✅ Container builds without errors
2. ✅ Authentication persists between sessions
3. ✅ Claude can only access test-files/
4. ✅ Resource limits are enforced
5. ✅ Network isolation works (if enabled)
6. ✅ Runs as non-root user
7. ✅ Integration with manage.sh works
8. ✅ Can execute Claude commands
9. ✅ Cleanup removes all artifacts
10. ✅ No access to host filesystem

## 🎯 Test Scenarios

### Scenario 1: Code Review
```bash
cp ~/projects/sample.js test-files/
./manage.sh --action sandbox --prompt "Review sample.js for best practices"
```

### Scenario 2: Bug Fixing
```bash
./manage.sh --action sandbox --prompt "Fix all bugs in example.js"
```

### Scenario 3: Batch Processing
```bash
./manage.sh --action sandbox --sandbox-command batch \
  --prompt "Add comments to all functions" \
  --max-turns 20
```

### Scenario 4: Security Audit
```bash
./manage.sh --action sandbox --prompt "Security audit test-patterns.py"
```

## 📝 Test Report Template

After testing, document results:

```markdown
## Claude Code Sandbox Test Report

**Date**: [DATE]
**Tester**: [NAME]
**Version**: [SANDBOX VERSION]

### Test Results

| Test | Status | Notes |
|------|--------|-------|
| Docker Build | ✅/❌ | |
| Authentication | ✅/❌ | |
| File Isolation | ✅/❌ | |
| Resource Limits | ✅/❌ | |
| Network Isolation | ✅/❌ | |
| User Permissions | ✅/❌ | |
| Tool Integration | ✅/❌ | |
| Cleanup | ✅/❌ | |

### Issues Found
1. [Issue description]
2. [Issue description]

### Recommendations
1. [Recommendation]
2. [Recommendation]
```

## 🚀 Next Steps

After successful testing:
1. Document any custom configurations
2. Create project-specific test files
3. Integrate into development workflow
4. Share findings with team
5. Consider automation opportunities