# Sudo Override for Claude Code

## 🎯 Overview

The Claude Code resource now includes optional sudo override functionality that allows Claude Code to execute system commands with elevated privileges during automated resource management operations.

## ⚠️ Security Warning

**This feature should be used with extreme caution!** Enabling sudo override:
- Allows Claude Code to execute system commands with root privileges
- Can potentially modify system configuration
- Should only be used in controlled, trusted environments
- Requires proper sudo configuration on the system

## 🔧 Configuration

### **1. Command Line Arguments**

```bash
# Enable sudo override
--sudo-override yes

# Specify allowed commands (comma-separated)
--sudo-commands "systemctl,service,docker,apt-get,chown,chmod"

# Provide sudo password (use environment variable instead for security)
--sudo-password "your-password"
```

### **2. Environment Variables**

```bash
# Enable sudo override
export SUDO_OVERRIDE="yes"

# Specify allowed commands
export SUDO_COMMANDS="systemctl,service,docker,apt-get,chown,chmod,mkdir,rm,cp,mv"

# Provide sudo password (more secure than command line)
export SUDO_PASSWORD="your-password"
```

### **3. Default Allowed Commands**

When no specific commands are provided, the following commands are allowed by default:

- **Service Management**: `systemctl`, `service`
- **Package Management**: `apt-get`, `apt`, `npm`, `pip`, `brew`, `snap`
- **Container Management**: `docker`, `podman`
- **File Operations**: `chown`, `chmod`, `mkdir`, `rm`, `cp`, `mv`
- **System Monitoring**: `lsof`, `netstat`, `ps`, `kill`, `pkill`
- **Version Control**: `git`

## 🚀 Usage Examples

### **Basic Usage**

```bash
# Run Claude Code with sudo override
resource-claude-code content run \
  --prompt "Install and configure PostgreSQL service" \
  --sudo-override yes \
  --dangerously-skip-permissions yes
```

### **With Specific Commands**

```bash
# Only allow specific commands
resource-claude-code content run \
  --prompt "Start Docker service and check status" \
  --sudo-override yes \
  --sudo-commands "systemctl,docker,ps" \
  --dangerously-skip-permissions yes
```

### **With Environment Variables**

```bash
# Set environment variables
export SUDO_OVERRIDE="yes"
export SUDO_COMMANDS="systemctl,service,docker,apt-get"
export SUDO_PASSWORD="your-password"

# Run Claude Code
resource-claude-code content run \
  --prompt "Install and start Redis service" \
  --dangerously-skip-permissions yes
```

### **Testing Sudo Override**

```bash
# Test sudo override functionality
resource-claude-code test sudo \
  --sudo-override yes
```

## 🔒 Security Features

### **1. Temporary Sudoers Files**

The system creates temporary sudoers files that are:
- Automatically cleaned up when the script exits
- Limited to specific commands only
- Generated with unique process IDs to avoid conflicts

### **2. Command Path Validation**

Commands are validated to ensure they point to legitimate system binaries:
- Full paths are resolved using `command -v`
- Defaults to `/usr/bin/` if command not found
- Prevents execution of arbitrary scripts

### **3. Automatic Cleanup**

Sudo override configuration is automatically cleaned up:
- When the script exits normally
- When the script is interrupted (SIGINT, SIGTERM)
- Temporary files are removed from `/etc/sudoers.d/`

### **4. Password Security**

Sudo passwords can be provided via:
- Environment variable `SUDO_PASSWORD` (recommended)
- Command line argument `--sudo-password` (less secure)
- Interactive prompt (if no password provided)

## 🧪 Testing

### **Test Sudo Access**

```bash
# Test basic sudo functionality
resource-claude-code test sudo

# Test with specific configuration
resource-claude-code test sudo \
  --sudo-override yes \
  --sudo-commands "systemctl,docker"
```

### **Test Specific Commands**

```bash
# Test systemctl access
sudo -n systemctl status docker

# Test docker access
sudo -n docker ps

# Test package management
sudo -n apt-get update --dry-run
```

## 🔧 Integration with Resource Management

### **Vrooli Resource Improvement Loop**

```bash
# Set environment for automated resource management
export SUDO_OVERRIDE="yes"
export SUDO_COMMANDS="systemctl,service,docker,apt-get,chown,chmod"
export CLAUDE_DANGEROUSLY_SKIP_PERMISSIONS="yes"

# Run the resource improvement loop
./auto/tasks/resource-improvement/task.sh
```

### **Automated Resource Setup**

```bash
# Example: Automated PostgreSQL setup
resource-claude-code content run \
  --prompt "Install PostgreSQL, create user, configure service, and start it" \
  --sudo-override yes \
  --sudo-commands "apt-get,systemctl,chown,chmod,mkdir" \
  --dangerously-skip-permissions yes \
  --timeout 1800
```

## 🚨 Troubleshooting

### **Common Issues**

#### **1. "Sudo is not available"**
```bash
# Check if sudo is installed
which sudo

# Install sudo if needed (Ubuntu/Debian)
apt-get install sudo

# Install sudo if needed (CentOS/RHEL)
yum install sudo
```

#### **2. "User does not have sudo privileges"**
```bash
# Check sudo group membership
groups $USER

# Add user to sudo group (Ubuntu/Debian)
sudo usermod -aG sudo $USER

# Add user to sudo group (CentOS/RHEL)
sudo usermod -aG wheel $USER
```

#### **3. "Failed to install temporary sudoers file"**
```bash
# Check if /etc/sudoers.d/ exists
ls -la /etc/sudoers.d/

# Create directory if missing
sudo mkdir -p /etc/sudoers.d/
sudo chmod 755 /etc/sudoers.d/
```

#### **4. "Invalid sudo password"**
```bash
# Test sudo password manually
echo "your-password" | sudo -S echo "Password works"

# Check password via environment variable
export SUDO_PASSWORD="your-password"
echo "$SUDO_PASSWORD" | sudo -S echo "Environment password works"
```

### **Manual Cleanup**

If automatic cleanup fails, manually remove temporary sudoers files:

```bash
# List temporary sudoers files
sudo ls -la /etc/sudoers.d/claude-code-temp-*

# Remove specific file
sudo rm -f /etc/sudoers.d/claude-code-temp-12345

# Remove all temporary files (use with caution)
sudo rm -f /etc/sudoers.d/claude-code-temp-*
```

## 📋 Best Practices

### **1. Security**
- Use environment variables for passwords, not command line arguments
- Limit sudo commands to only what's necessary
- Test in a sandbox environment first
- Monitor sudo usage with logging

### **2. Automation**
- Use specific command lists rather than broad permissions
- Set appropriate timeouts for long-running operations
- Implement proper error handling and rollback procedures
- Log all sudo operations for audit purposes

### **3. Integration**
- Integrate with existing Vrooli resource management workflows
- Use the sandbox for testing dangerous operations
- Implement staged approval for critical system changes
- Have rollback procedures ready

## 🔗 Related Documentation

- [Claude Code Configuration Guide](CONFIGURATION.md)
- [Claude Code Security Best Practices](SECURITY.md)
- [Vrooli Resource Management](auto/tasks/resource-improvement/prompts/resource-improvement-loop.md)
- [Sudo Configuration Guide](https://gist.github.com/eriknomitch/99ad47b6b2d6c293c5412a3f7e5a9db6) 