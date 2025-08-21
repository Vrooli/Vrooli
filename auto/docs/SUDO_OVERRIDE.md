# Auto/ Sudo Override System

## 🎯 Overview

The auto/ system now includes integrated sudo override functionality that allows Claude Code to execute system commands with elevated privileges during automated resource management loops. This system requires **only one password entry** at the beginning and maintains sudo access throughout the entire loop session.

## ⚠️ Security Warning

**This feature should be used with extreme caution!** Enabling sudo override:
- Allows Claude Code to execute system commands with root privileges
- Can potentially modify system configuration
- Should only be used in controlled, trusted environments
- Requires proper sudo configuration on the system

## 🚀 Quick Start

### **1. Initialize Sudo Override (One-Time Setup)**

```bash
# Initialize with default commands
./auto/manage-resource-loop.sh sudo-init

# Initialize with custom commands
./auto/manage-resource-loop.sh sudo-init "systemctl,docker,apt-get,chown,chmod"
```

This will:
- Prompt for your sudo password once
- Store it securely for the session
- Test sudo access
- Configure allowed commands

### **2. Test Sudo Override**

```bash
# Test that sudo override is working
./auto/manage-resource-loop.sh sudo-test
```

### **3. Start Resource Improvement Loop with Sudo**

```bash
# Start the loop - sudo override will be automatically enabled
./auto/manage-resource-loop.sh start
```

## 🔧 How It Works

### **1. One-Time Password Entry**

When you run `sudo-init`, the system:
1. Prompts for your sudo password
2. Validates the password with sudo
3. Stores it securely in `auto/data/sudo-override/password`
4. Creates configuration in `auto/data/sudo-override/config`

### **2. Automatic Integration**

The auto/ system automatically:
1. **Loads sudo configuration** when the loop starts
2. **Exports environment variables** for Claude Code
3. **Enables sudo override** for all iterations
4. **Maintains access** throughout the entire loop session

### **3. Secure Storage**

- Password is stored in `auto/data/sudo-override/password` with 600 permissions
- Configuration is stored in `auto/data/sudo-override/config`
- Files are automatically cleaned up when you run `sudo-cleanup`

## 📋 Available Commands

### **Sudo Override Management**

```bash
# Initialize sudo override (one-time setup)
./auto/manage-resource-loop.sh sudo-init [commands]

# Test sudo override functionality
./auto/manage-resource-loop.sh sudo-test

# Check sudo override status
./auto/manage-resource-loop.sh sudo-status

# Remove sudo override configuration
./auto/manage-resource-loop.sh sudo-cleanup
```

### **Loop Management (Enhanced)**

```bash
# Start loop with sudo override
./auto/manage-resource-loop.sh start

# Stop loop
./auto/manage-resource-loop.sh stop

# Check status
./auto/manage-resource-loop.sh status

# View logs
./auto/manage-resource-loop.sh logs [-f]
```

## 🔒 Security Features

### **1. Secure Password Storage**
- Password stored with 600 permissions (user-only access)
- Stored in isolated directory `auto/data/sudo-override/`
- Automatically cleaned up with `sudo-cleanup`

### **2. Command Whitelisting**
- Only specified commands are allowed with sudo
- Default commands: `systemctl,service,docker,podman,apt-get,apt,chown,chmod,mkdir,rm,cp,mv,lsof,netstat,ps,kill,pkill,npm,pip,brew,snap,git`
- Custom commands can be specified during initialization

### **3. Automatic Cleanup**
- Configuration is automatically cleaned up when loop stops
- Manual cleanup available with `sudo-cleanup`
- Password is never logged or displayed

### **4. Session Isolation**
- Sudo access is limited to the current session
- Configuration is not persisted across reboots
- Each session requires re-initialization

## 🎯 Integration with Resource Management

### **Automatic Sudo Access**

When the resource improvement loop starts:

1. **Sudo override is automatically initialized** if available
2. **Environment variables are set** for Claude Code:
   - `SUDO_OVERRIDE="yes"`
   - `SUDO_COMMANDS="systemctl,docker,apt-get,..."`
   - `SUDO_PASSWORD="[stored-password]"`
3. **Claude Code can execute sudo commands** without prompting

### **Example Resource Management Tasks**

With sudo override enabled, Claude Code can:

```bash
# Install and configure services
sudo systemctl start postgresql
sudo systemctl enable redis

# Install packages
sudo apt-get update
sudo apt-get install -y docker.io

# Manage file permissions
sudo chown -R $USER:$USER /opt/myapp
sudo chmod 755 /opt/myapp/scripts

# Manage Docker containers
sudo docker run -d --name myapp nginx
sudo docker ps
```

## 🧪 Testing Examples

### **Test Sudo Override Setup**

```bash
# 1. Initialize sudo override
./auto/manage-resource-loop.sh sudo-init

# 2. Test functionality
./auto/manage-resource-loop.sh sudo-test

# 3. Check status
./auto/manage-resource-loop.sh sudo-status
```

### **Test Resource Management with Sudo**

```bash
# Start the resource improvement loop
./auto/manage-resource-loop.sh start

# The loop will automatically:
# - Load sudo override configuration
# - Enable sudo access for Claude Code
# - Allow system commands in resource management
```

### **Manual Testing**

```bash
# Test specific sudo commands
sudo -n systemctl status docker
sudo -n docker ps
sudo -n apt-get update --dry-run
```

## 🚨 Troubleshooting

### **Common Issues**

#### **1. "Sudo override not available"**
```bash
# Check if sudo override script exists
ls -la auto/lib/sudo-override.sh

# Check if script is executable
chmod +x auto/lib/sudo-override.sh
```

#### **2. "Failed to initialize sudo override"**
```bash
# Check sudo access manually
sudo -n echo "test"

# If password required, initialize with password
./auto/manage-resource-loop.sh sudo-init
```

#### **3. "Sudo override test failed"**
```bash
# Check stored configuration
cat auto/data/sudo-override/config

# Re-initialize if needed
./auto/manage-resource-loop.sh sudo-cleanup
./auto/manage-resource-loop.sh sudo-init
```

#### **4. "Permission denied" in loop**
```bash
# Check if sudo override is loaded
./auto/manage-resource-loop.sh sudo-status

# Re-initialize if needed
./auto/manage-resource-loop.sh sudo-init
```

### **Debugging Commands**

```bash
# Check sudo override status
./auto/manage-resource-loop.sh sudo-status

# Test sudo override functionality
./auto/manage-resource-loop.sh sudo-test

# View loop logs for sudo-related messages
./auto/manage-resource-loop.sh logs | grep -i sudo
```

## 📋 Best Practices

### **1. Security**
- Only initialize sudo override in trusted environments
- Use specific command lists rather than broad permissions
- Regularly review and clean up sudo override configuration
- Monitor loop logs for unexpected sudo usage

### **2. Automation**
- Initialize sudo override before starting long-running loops
- Test sudo override functionality before production use
- Have rollback procedures ready for automated system changes
- Use the sandbox for testing dangerous operations

### **3. Maintenance**
- Clean up sudo override configuration when not needed
- Monitor system logs for sudo usage
- Regularly update allowed command lists
- Review and audit sudo access patterns

## 🔗 Related Documentation

- [Claude Code Sudo Override](../scripts/resources/agents/claude-code/docs/SUDO_OVERRIDE.md)
- [Resource Improvement Loop](../tasks/resource-improvement/prompts/resource-improvement-loop.md)
- [Auto/ System Overview](README.md)
- [Loop Core Documentation](../lib/loop.sh)

## 🎉 Summary

The auto/ sudo override system provides:

1. **One-time password entry** - No repeated prompts during loops
2. **Automatic integration** - Seamless sudo access for Claude Code
3. **Secure storage** - Password stored securely for session duration
4. **Command whitelisting** - Only specified commands allowed
5. **Easy management** - Simple commands for setup, testing, and cleanup

This enables fully automated resource management loops with sudo access while maintaining security and control. 