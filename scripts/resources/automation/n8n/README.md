# n8n Automation Platform for Vrooli

This directory contains the n8n setup with enhanced host system access for the Vrooli project.

## Overview

n8n is a workflow automation platform that allows you to connect various services and automate tasks. This custom setup provides:

- **Host Command Access**: Run system commands directly from n8n workflows
- **Docker Integration**: Control other Docker containers from within n8n
- **Workspace Access**: Direct access to the Vrooli project files
- **Enhanced Tools**: Pre-installed utilities for common automation tasks

## Features

### Custom Docker Image
- Based on official n8n image with additional tools
- Seamless host command execution (no absolute paths needed)
- Pre-installed: bash, git, curl, wget, jq, python3, docker-cli
- Smart PATH handling for transparent host binary access

### Volume Mounts
- `/host/usr/bin`, `/host/bin`: Read-only access to system binaries
- `/host/home`: Read-write access to user home directory
- `/workspace`: Direct access to Vrooli project
- `/var/run/docker.sock`: Docker control capabilities

### Security Note
This setup prioritizes functionality over isolation. It's designed for development and trusted environments where n8n needs full system access.

## Installation

### Quick Start
```bash
# Install with custom image (recommended)
./n8n.sh --action install --build-image yes

# Install with standard n8n image
./n8n.sh --action install
```

### Installation Options
```bash
./n8n.sh --action install \
  --build-image yes \           # Build custom image with host access
  --basic-auth yes \            # Enable authentication (default)
  --username admin \            # Set username (default: admin)
  --password YourPassword \     # Set password (auto-generated if not specified)
  --database postgres \         # Use PostgreSQL (default: sqlite)
  --webhook-url https://...     # Set external webhook URL
```

## Usage

### Management Commands
```bash
# Check status
./n8n.sh --action status

# View logs
./n8n.sh --action logs

# Stop n8n
./n8n.sh --action stop

# Start n8n
./n8n.sh --action start

# Restart n8n
./n8n.sh --action restart

# Reset admin password
./n8n.sh --action reset-password

# Uninstall n8n
./n8n.sh --action uninstall
```

### Using Execute Command Node

With the custom image, you can run commands naturally in the Execute Command node:

```bash
# System commands work without paths
tput bel                      # Terminal bell
ls ~/Vrooli                   # List Vrooli directory
whoami                        # Current user

# Docker commands
docker ps                     # List containers
docker logs container-name    # View container logs

# Python scripts
python3 /workspace/script.py  # Run Python scripts

# Git operations
cd /workspace && git status   # Check git status

# Direct file access
cat /host/home/.bashrc       # Read host files
echo "test" > /workspace/output.txt  # Write to workspace
```

### Workflow Examples

#### 1. System Monitoring
```javascript
// Execute Command node
ps aux | grep node
df -h
free -m
```

#### 2. Docker Management
```javascript
// Execute Command node
docker stats --no-stream
docker restart my-container
```

#### 3. File Processing
```javascript
// Execute Command node
find /workspace -name "*.log" -mtime -1
grep -r "ERROR" /workspace/logs/
```

## Architecture

### Directory Structure
```
n8n/
├── n8n.sh                   # Main management script
├── Dockerfile              # Custom n8n image definition
├── docker-entrypoint.sh    # PATH setup and command handling
├── README.md              # This file
└── config/                # Future configuration files
```

### How It Works

1. **Custom Entrypoint**: The `docker-entrypoint.sh` script:
   - Adds host directories to PATH
   - Sets up command fallback handling
   - Preserves n8n's original functionality

2. **Smart Command Resolution**:
   - Container binaries are preferred
   - Falls back to host binaries if not found
   - Special wrappers for compatibility (e.g., tput)

3. **Volume Strategy**:
   - Read-only mounts for system directories
   - Read-write for workspace and home
   - Conditional mounts based on availability

## Troubleshooting

### Command Not Found
- Verify the command exists on the host: `which command-name`
- Check if the directory is mounted: `ls /host/usr/bin/`
- Try using the full path: `/host/usr/bin/command-name`

### Permission Denied
- Some commands may require specific permissions
- Check file ownership and permissions
- Consider using sudo in the host system if needed

### Container Won't Start
- Check logs: `docker logs n8n`
- Verify ports are available: `ss -tlnp | grep 5678`
- Ensure Docker socket is accessible

### Build Failures
- Check Docker is running: `docker info`
- Verify Dockerfile syntax
- Ensure all files in this directory are present

## Development

### Building Custom Image Manually
```bash
cd scripts/resources/automation/n8n
docker build -t n8n-vrooli:latest .
```

### Testing Commands
```bash
# Test command resolution
docker exec n8n which tput
docker exec n8n tput bel

# Test workspace access
docker exec n8n ls /workspace

# Test Docker access
docker exec n8n docker ps
```

### Extending the Image
Edit `Dockerfile` to add more tools:
```dockerfile
RUN apk add --no-cache \
    your-package-here
```

Then rebuild: `./n8n.sh --action install --build-image yes --force yes`

## Best Practices

1. **Security**: Only use in trusted environments
2. **Commands**: Test commands in Docker first before using in workflows
3. **Paths**: Use `/workspace` for Vrooli files, `/host/home` for user files
4. **Cleanup**: Regularly clean execution data to save space
5. **Backups**: Export workflows regularly

## FAQ

**Q: Why custom image instead of standard n8n?**
A: Standard n8n in Docker can't access host commands. Our custom image bridges this gap.

**Q: Is this secure?**
A: This setup prioritizes functionality over isolation. Use only in trusted environments.

**Q: Can I use this in production?**
A: Recommend standard n8n for production unless you specifically need host access.

**Q: How do I update n8n?**
A: Rebuild the custom image: `./n8n.sh --action install --build-image yes --force yes`

## Support

For issues related to:
- This custom setup: Check this README and n8n.sh script
- n8n itself: Visit https://docs.n8n.io
- Vrooli integration: See Vrooli documentation