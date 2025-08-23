# Claude Code Troubleshooting Guide

This guide covers common issues and their solutions when using Claude Code.

## 🚨 Installation Issues

### Node.js Version Problems

**Symptoms**: Error messages about incompatible Node.js version

**Diagnosis**:
```bash
# Check current Node.js version
node --version

# Check npm version
npm --version
```

**Solution**:
```bash
# Update Node.js from https://nodejs.org/
# Minimum requirement: Node.js 18

# Verify after update
node --version  # Should be 18.x or higher
```

### npm Not Found

**Symptoms**: `npm: command not found`

**Solution**:
```bash
# npm should come with Node.js
# Reinstall Node.js from https://nodejs.org/

# Verify npm installation
which npm
npm --version
```

### Permission Errors

**Symptoms**: EACCES permission denied errors during installation

**Solution**:
```bash
# Configure npm to use a different directory
npm config set prefix ~/.npm-global
echo 'export PATH=~/.npm-global/bin:$PATH' >> ~/.bashrc
source ~/.bashrc

# Reinstall Claude Code
npm install -g @anthropic-ai/claude-code

# Alternative: Fix npm permissions
sudo chown -R $(whoami) $(npm config get prefix)/{lib/node_modules,bin,share}
```

### Installation Verification

```bash
# Check if Claude Code is properly installed
which claude
claude --version

# Run diagnostics
claude doctor

# Check with resource manager
./manage.sh --action status
```

## 🔐 Authentication Issues

### Login Problems

**Symptoms**: Cannot authenticate or login fails

**Diagnosis Steps**:
```bash
# Check authentication status
claude doctor

# Verify internet connectivity
ping claude.ai

# Check subscription status at claude.ai
```

**Solutions**:

1. **Subscription Check**:
   - Ensure you have Claude Pro or Max subscription
   - Verify subscription is active at claude.ai
   - Check payment status

2. **Network Issues**:
   ```bash
   # Test connectivity
   curl -I https://claude.ai
   
   # Check firewall/proxy settings
   # Ensure port 443 (HTTPS) is accessible
   ```

3. **Clear and Re-authenticate**:
   ```bash
   # Start fresh authentication
   claude
   # Follow login prompts
   ```

### Credential Errors

**Symptoms**: Invalid credentials or authentication expired

**Solution**:
```bash
# Authentication is handled automatically
# Simply run claude and follow prompts if re-authentication is needed
claude

# Check status after re-authentication
claude doctor
```

### Subscription Limit Issues

**Symptoms**: "Usage limit reached" or similar errors

**Solutions**:
- Wait for usage window to reset (5-hour windows)
- Upgrade to Max plan for higher limits
- Use during off-peak hours
- Check usage at claude.ai

## 💻 Command Execution Issues

### Command Not Found

**Symptoms**: `claude: command not found`

**Diagnosis**:
```bash
# Check if claude is in PATH
which claude
echo $PATH
```

**Solutions**:
```bash
# Add to PATH if needed
echo 'export PATH=$PATH:/usr/local/bin' >> ~/.bashrc
source ~/.bashrc

# Reinstall if necessary
npm install -g @anthropic-ai/claude-code

# Use full path as temporary fix
/usr/local/bin/claude --version
```

### Slow Response Times

**Symptoms**: Long delays between prompts and responses

**Diagnosis**:
```bash
# Test response time
time claude -p "Hello"

# Check network connectivity
ping claude.ai

# Monitor system resources
top -p $(pgrep -f claude)
```

**Solutions**:
1. **Network Optimization**:
   - Check internet connection stability
   - Try using wired connection instead of WiFi
   - Test during different times of day

2. **Subscription Tier**:
   - Upgrade to Max plan for priority access
   - Check if usage limits are affecting performance

3. **System Resources**:
   ```bash
   # Free up memory
   free -h
   
   # Close unnecessary applications
   # Restart if needed
   ```

### Permission Denied Errors

**Symptoms**: Cannot execute commands or access files

**Solutions**:
```bash
# Check file permissions
ls -la

# Ensure you're in correct directory
pwd

# For system operations, use appropriate permissions
# Or use --dangerously-skip-permissions (with extreme caution)
```

## 🔧 Management Script Issues

### Script Not Executable

**Symptoms**: Permission denied when running `./manage.sh`

**Solution**:
```bash
# Make script executable
chmod +x ./manage.sh

# Or run with bash
bash ./manage.sh --action status
```

### Invalid Arguments

**Symptoms**: "Unknown option" or invalid argument errors

**Solution**:
```bash
# Check available options
./manage.sh --help

# Common valid actions
./manage.sh --action install
./manage.sh --action status
./manage.sh --action run --prompt "test"
```

### Timeout Issues

**Symptoms**: Operations timeout before completion

**Solutions**:
```bash
# Increase timeout for batch operations
./manage.sh --action batch --prompt "Long task" --timeout 3600

# Use smaller max-turns for complex tasks
./manage.sh --action batch --prompt "Task" --max-turns 20

# Break large tasks into smaller ones
./manage.sh --action run --prompt "Subtask 1" --max-turns 5
```

## 📁 File and Project Issues

### File Access Problems

**Symptoms**: Cannot read or modify files

**Diagnosis**:
```bash
# Check file permissions
ls -la filename

# Check directory permissions
ls -la .

# Verify file exists
file filename
```

**Solutions**:
```bash
# Fix file permissions
chmod 644 filename

# Fix directory permissions
chmod 755 .

# Ensure you're in the correct directory
pwd
ls -la
```

### Large Project Performance

**Symptoms**: Slow performance in large codebases

**Solutions**:
```bash
# Use .gitignore to exclude unnecessary files
echo "node_modules/" >> .gitignore
echo "dist/" >> .gitignore

# Limit tools to reduce overhead
./manage.sh --action run --prompt "Task" --allowed-tools "Read,Edit"

# Work in smaller subdirectories
cd specific-module/
claude -p "Focus on this module"
```

### Git Integration Issues

**Symptoms**: Problems with version control integration

**Solutions**:
```bash
# Verify git repository
git status

# Ensure .gitignore is properly configured
cat .gitignore

# Claude respects .gitignore automatically
# Add files to ignore if needed
echo ".claude-temp" >> .gitignore
```

## 🌐 Network and Connectivity Issues

### Connection Timeouts

**Symptoms**: "Connection timeout" or "Network error" messages

**Diagnosis**:
```bash
# Test basic connectivity
ping claude.ai

# Test HTTPS connectivity
curl -I https://claude.ai

# Check DNS resolution
nslookup claude.ai
```

**Solutions**:
1. **Network Check**:
   ```bash
   # Restart network interface
   sudo systemctl restart network-manager
   
   # Or restart networking
   sudo service networking restart
   ```

2. **Proxy/Firewall**:
   ```bash
   # Check proxy settings
   echo $http_proxy $https_proxy
   
   # Temporarily disable proxy if needed
   unset http_proxy https_proxy
   ```

3. **DNS Issues**:
   ```bash
   # Use different DNS servers
   echo "nameserver 8.8.8.8" | sudo tee /etc/resolv.conf
   ```

### SSL/TLS Errors

**Symptoms**: Certificate or SSL handshake errors

**Solutions**:
```bash
# Update certificates
sudo apt-get update && sudo apt-get install ca-certificates

# For macOS
brew install ca-certificates

# Test SSL connection
openssl s_client -connect claude.ai:443
```

## 🔄 Session and State Issues

### Session Corruption

**Symptoms**: Unexpected behavior or errors in ongoing sessions

**Solutions**:
```bash
# Start fresh session
claude  # Exit current session first

# Clear temporary files if needed
rm -rf /tmp/claude-*

# Check session status
./manage.sh --action session
```

### Memory Issues

**Symptoms**: Out of memory errors or system slowdown

**Solutions**:
```bash
# Check memory usage
free -h
ps aux | grep claude

# Restart Claude Code
pkill -f claude
claude

# Reduce concurrent operations
# Use smaller batch sizes
./manage.sh --action batch --prompt "Task" --max-turns 10
```

## 🔍 Debugging and Diagnostics

### Enable Debug Mode

```bash
# Set debug environment variable
export CLAUDE_DEBUG=true

# Run with debug output
claude -p "Test prompt"

# Check debug logs
./manage.sh --action logs
```

### Comprehensive Diagnostics

```bash
# Full system check
claude doctor

# Resource manager status
./manage.sh --action status

# System information
uname -a
node --version
npm --version
which claude
```

### Log Analysis

```bash
# View recent logs
./manage.sh --action logs

# Monitor logs in real-time
tail -f ~/.config/claude/logs/claude.log

# Search for specific errors
grep -i error ~/.config/claude/logs/claude.log
```

## ⚠️ Dangerous Operations Troubleshooting

### Permission Override Issues

**When using `--dangerously-skip-permissions`**:

**Problems**:
- Unintended file modifications
- System instability
- Security vulnerabilities

**Solutions**:
```bash
# Revert to safe mode immediately
./manage.sh --action run --prompt "Task" --allowed-tools "Read"

# Use version control to revert changes
git checkout -- affected-files

# Limit tools even with dangerous flag
./manage.sh --action run --prompt "Task" \
  --dangerously-skip-permissions yes \
  --allowed-tools "Edit,Read"
```

## 📞 Getting Additional Help

### Collect Diagnostic Information

```bash
# Create diagnostic report
{
  echo "=== System Info ==="
  uname -a
  echo "=== Node.js Version ==="
  node --version
  echo "=== npm Version ==="
  npm --version
  echo "=== Claude Status ==="
  claude doctor
  echo "=== Resource Manager Status ==="
  ./manage.sh --action status
  echo "=== Recent Logs ==="
  tail -50 ~/.config/claude/logs/claude.log 2>/dev/null || echo "No logs found"
} > claude-diagnostic-report.txt
```

### Support Resources

- **Official Documentation**: https://docs.anthropic.com/claude-code
- **Help Center**: https://support.anthropic.com
- **Community Forums**: Claude community discussions
- **GitHub Issues**: https://github.com/anthropics/claude-code/issues

### Quick Fix Summary

| Problem | Quick Fix |
|---------|-----------|
| Command not found | `npm install -g @anthropic-ai/claude-code` |
| Permission denied | `chmod +x ./manage.sh` or fix npm permissions |
| Authentication failed | `claude` and follow login prompts |
| Slow responses | Check internet connection, try off-peak hours |
| File access issues | Check file/directory permissions with `ls -la` |
| Timeout errors | Increase `--timeout` or reduce `--max-turns` |
| Memory issues | Restart claude, reduce batch size |
| Network errors | Check connectivity with `ping claude.ai` |

Most issues can be resolved by running `claude doctor` and following the diagnostic recommendations.