# OBS Studio Docker Deployment Guide

This guide covers deploying OBS Studio using Docker containers with specific version tags for production reliability.

## Overview

The Docker deployment provides:
- Isolated OBS Studio environment
- Version-specific deployments (no 'latest' tags)
- VNC/NoVNC remote access
- Health monitoring
- Automated supervisor management
- GPU acceleration support

## Quick Start

### Build and Run

```bash
# Build Docker image with specific version
vrooli resource obs-studio docker build

# Run OBS Studio container
vrooli resource obs-studio docker run

# Check container status
vrooli resource obs-studio docker status
```

## Docker Commands

### Build Image

Build OBS Studio Docker image with specific version:

```bash
vrooli resource obs-studio docker build [version]
```

Default version: 30.2.3

### Run Container

Start OBS Studio in Docker:

```bash
vrooli resource obs-studio docker run [version]
```

This will:
- Create container named `vrooli-obs-studio`
- Map WebSocket port (4455)
- Map VNC port (5900)
- Map NoVNC web port (6080)
- Mount configuration and recording directories
- Enable GPU acceleration if available

### Stop Container

Stop running container:

```bash
vrooli resource obs-studio docker stop
```

### Remove Container

Remove stopped container:

```bash
vrooli resource obs-studio docker remove
```

### Container Status

Check container health and resource usage:

```bash
vrooli resource obs-studio docker status
```

Output includes:
- Running/Stopped state
- Health check status
- CPU and memory usage

### Container Logs

View container logs:

```bash
vrooli resource obs-studio docker logs [lines]
```

Default: Last 50 lines

### Cleanup

Remove all Docker resources:

```bash
vrooli resource obs-studio docker cleanup
```

This removes:
- Container
- Docker images
- Build cache

## Access Methods

### WebSocket API

```
ws://localhost:4455
```

Use for programmatic control via obs-websocket protocol.

### VNC Access

```
vnc://localhost:5900
Password: obspass123 (configurable via OBS_VNC_PASSWORD)
```

Use any VNC client for direct desktop access.

### Web VNC (NoVNC)

```
http://localhost:6080
```

Browser-based VNC access, no client needed.

## Configuration

### Environment Variables

```bash
export OBS_DOCKER_VERSION="30.2.3"     # OBS version to install
export OBS_CONTAINER_NAME="custom-obs"  # Container name
export OBS_WEBSOCKET_PORT="4455"        # WebSocket port
export OBS_VNC_PORT="5900"              # VNC port
export OBS_NOVNC_PORT="6080"            # Web VNC port
export OBS_VNC_PASSWORD="secure123"     # VNC password
```

### Volume Mounts

The container mounts these directories:
- `~/.vrooli/obs-studio/config` → `/home/obs/.config/obs-studio`
- `~/.vrooli/obs-studio/recordings` → `/home/obs/recordings`
- `~/.vrooli/obs-studio/scenes` → `/home/obs/scenes`

### GPU Acceleration

GPU access is enabled by default with:
- `--device=/dev/dri:/dev/dri` - Direct rendering infrastructure
- `--shm-size=2g` - Shared memory for video processing

## Version Management

### Specific Versions Only

This implementation follows best practices:
- Default version: 30.2.3
- No 'latest' tag usage in production
- Explicit version specification
- Tagged images: `vrooli/obs-studio:30.2.3`

### Upgrading Versions

```bash
# Build new version
vrooli resource obs-studio docker build 30.3.0

# Stop old container
vrooli resource obs-studio docker stop

# Remove old container
vrooli resource obs-studio docker remove

# Run new version
vrooli resource obs-studio docker run 30.3.0
```

## Health Monitoring

The container includes health checks:

### Internal Health Check

Monitors:
- Xvfb (virtual display)
- x11vnc (VNC server)
- OBS process
- WebSocket port

### Docker Health Status

```bash
docker inspect --format='{{.State.Health.Status}}' vrooli-obs-studio
```

States:
- `starting` - Initial startup
- `healthy` - All services running
- `unhealthy` - Service failure detected

## Troubleshooting

### Container Won't Start

```bash
# Check logs
vrooli resource obs-studio docker logs 100

# Verify port availability
netstat -tulpn | grep -E "4455|5900|6080"

# Check Docker daemon
docker info
```

### VNC Connection Failed

```bash
# Verify VNC is running
docker exec vrooli-obs-studio pgrep x11vnc

# Check VNC logs
docker exec vrooli-obs-studio cat /home/obs/x11vnc.log
```

### WebSocket Connection Failed

```bash
# Check OBS is running
docker exec vrooli-obs-studio pgrep obs

# Test WebSocket port
nc -zv localhost 4455
```

### Performance Issues

```bash
# Check resource usage
vrooli resource obs-studio docker status

# Increase shared memory
docker run --shm-size=4g ...

# Check GPU access
docker exec vrooli-obs-studio ls -la /dev/dri
```

## Security Considerations

### Network Isolation

- Ports are only exposed to localhost by default
- Use SSH tunneling for remote access
- Configure firewall rules as needed

### VNC Security

- Change default VNC password
- Use SSH tunneling for VNC over network
- Consider disabling VNC if not needed

### Container Security

- Runs as non-root user (obs)
- Minimal base image (Ubuntu 22.04)
- Regular security updates via rebuilds

## Production Deployment

### Recommended Settings

```bash
# Production environment variables
export OBS_DOCKER_VERSION="30.2.3"
export OBS_VNC_PASSWORD="$(openssl rand -base64 32)"
export OBS_WEBSOCKET_PORT="4455"

# Run with restart policy
vrooli resource obs-studio docker run

# The container uses --restart unless-stopped by default
```

### Monitoring

```bash
# Set up monitoring loop
while true; do
    vrooli resource obs-studio docker status
    sleep 60
done
```

### Backup

```bash
# Backup configuration
tar -czf obs-backup.tar.gz ~/.vrooli/obs-studio/config

# Backup scenes
tar -czf obs-scenes.tar.gz ~/.vrooli/obs-studio/scenes
```

## Integration Examples

### Python WebSocket Control

```python
import obsws_python as obs

# Connect to Docker container
client = obs.ReqClient(host='localhost', port=4455)

# Get version
version = client.get_version()
print(f"OBS Version: {version.obs_version}")

# Start recording
client.start_recording()
```

### Node.js Integration

```javascript
const OBSWebSocket = require('obs-websocket-js');
const obs = new OBSWebSocket();

// Connect to Docker container
obs.connect('ws://localhost:4455')
  .then(() => {
    console.log('Connected to OBS in Docker');
    return obs.call('StartRecording');
  })
  .catch(err => console.error('Error:', err));
```

## Summary

The Docker deployment provides a reliable, version-controlled way to run OBS Studio with:
- Specific version tags (no 'latest')
- Remote access via VNC/NoVNC
- Health monitoring
- GPU acceleration
- Production-ready configuration

This implementation satisfies the P2 requirement for Docker support with proper version management.