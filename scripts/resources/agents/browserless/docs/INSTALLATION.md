[← Back to README](../README.md) | [Documentation Index](./README.md)

# Browserless Installation Guide

This guide covers the installation and setup of Browserless for the Vrooli project.

## Prerequisites

Before installing Browserless, ensure you have:

- Docker installed and running
- User in docker group or sudo access
- 2GB+ RAM available
- 1GB+ free disk space
- Port 4110 available (or custom port)

### Verify Docker Setup

```bash
# Check Docker is installed
docker --version

# Check Docker daemon is running
docker ps

# Check user has Docker access
docker run hello-world
```

## Installation Methods

### Quick Install (Recommended)

The simplest way to install Browserless:

```bash
# Install with default settings
./manage.sh --action install
```

This will:
- Pull the latest Browserless Docker image
- Create container with default configuration
- Start the service on port 4110
- Configure health checks
- Update Vrooli resource configuration

### Custom Installation

Install with specific settings:

```bash
./manage.sh --action install \
  --port 4110 \              # Service port (default: 4110)
  --max-browsers 5 \         # Maximum concurrent sessions (default: 5)
  --timeout 30000 \          # Default timeout in ms (default: 30000)
  --headless yes             # Run in headless mode (default: yes)
```

### Installation Options

| Option | Description | Default |
|--------|-------------|---------|
| `--port` | Port to expose Browserless service | 4110 |
| `--max-browsers` | Maximum concurrent browser sessions | 5 |
| `--timeout` | Default timeout in milliseconds | 30000 |
| `--headless` | Run Chrome in headless mode | yes |
| `--enable-debugger` | Enable Chrome DevTools debugger | false |
| `--shared-memory` | Shared memory size (e.g., "2gb") | 2gb |

## Post-Installation Verification

### 1. Check Service Status

```bash
./manage.sh --action status
```

Expected output:
```
✅ browserless is running on port 4110
   Container: browserless
   Health: healthy
   Uptime: 2 minutes
```

### 2. Test Basic Functionality

```bash
# Test pressure endpoint
curl http://localhost:4110/pressure

# Expected response:
{
  "isAvailable": true,
  "queued": 0,
  "running": 0,
  "maxConcurrent": 5
}
```

### 3. Run Usage Examples

```bash
# Test screenshot capture
./manage.sh --action usage --usage-type screenshot

# Test PDF generation
./manage.sh --action usage --usage-type pdf
```

## Installation Troubleshooting

### Port Already in Use

If port 4110 is already in use:

```bash
# Check what's using the port
sudo lsof -i :4110

# Install on different port
./manage.sh --action install --port 4111
```

### Docker Permission Denied

If you get permission errors:

```bash
# Add user to docker group
sudo usermod -aG docker $USER

# Log out and back in, or run:
newgrp docker
```

### Insufficient Resources

If installation fails due to resources:

```bash
# Check available memory
free -h

# Check disk space
df -h

# Install with lower resource usage
./manage.sh --action install --max-browsers 2
```

## Updating Browserless

To update to the latest version:

```bash
# Stop current instance
./manage.sh --action stop

# Pull latest image
docker pull ghcr.io/browserless/chrome:latest

# Restart service
./manage.sh --action restart
```

## Uninstallation

To completely remove Browserless:

```bash
# Remove service and container
./manage.sh --action uninstall

# This will:
# - Stop and remove the container
# - Clean up configuration
# - Remove from Vrooli resources
```

## Configuration After Installation

Browserless is automatically configured in `~/.vrooli/resources.local.json`. See [Configuration Guide](./CONFIGURATION.md) for customization options.

## Next Steps

- Review [Configuration Guide](./CONFIGURATION.md) for advanced settings
- Check [Usage Guide](./USAGE.md) for common workflows
- See [API Reference](./API.md) for endpoint documentation

---
**See also:** [Configuration](./CONFIGURATION.md) | [Usage Guide](./USAGE.md) | [Troubleshooting](./TROUBLESHOOTING.md)