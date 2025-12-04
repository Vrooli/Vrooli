# Docker Engine Check (infra-docker)

Verifies that the Docker daemon is running and responsive.

## Overview

| Property | Value |
|----------|-------|
| Check ID | `infra-docker` |
| Category | Infrastructure |
| Interval | 60 seconds |
| Platforms | All (where Docker is installed) |

## What It Monitors

This check runs `docker info` to verify:

- Docker daemon is running
- Current user can communicate with the daemon
- Docker is responsive (not hung)

## Status Meanings

| Status | Meaning |
|--------|---------|
| **OK** | Docker daemon is running and responsive |
| **Warning** | Docker is installed but daemon isn't running |
| **Critical** | Docker command failed or timed out |

## Why It Matters

Docker is required for:
- Running containerized resources (PostgreSQL, Redis, etc.)
- Building and running scenarios with container dependencies
- Isolated development environments
- Container-based testing

Most Vrooli resources run as Docker containers, making Docker availability critical.

## Common Failure Causes

### 1. Docker Daemon Not Running
```bash
# Check daemon status
sudo systemctl status docker

# Start daemon
sudo systemctl start docker

# Enable auto-start
sudo systemctl enable docker
```

### 2. Permission Issues
```bash
# Check if user is in docker group
groups $USER | grep docker

# Add user to docker group (requires re-login)
sudo usermod -aG docker $USER
newgrp docker
```

### 3. Docker Socket Issues
```bash
# Check socket permissions
ls -la /var/run/docker.sock

# Socket should be owned by root:docker with 660 permissions
sudo chown root:docker /var/run/docker.sock
sudo chmod 660 /var/run/docker.sock
```

### 4. Disk Space Issues
```bash
# Docker needs disk space for images/containers
df -h /var/lib/docker

# Clean up if needed
docker system prune -a
```

### 5. Resource Exhaustion
```bash
# Check for hung containers consuming resources
docker ps -a
docker stats --no-stream

# Force restart daemon
sudo systemctl restart docker
```

## Troubleshooting Steps

1. **Check daemon status**
   ```bash
   sudo systemctl status docker
   docker info
   ```

2. **Check logs**
   ```bash
   sudo journalctl -u docker --since "10 minutes ago"
   ```

3. **Test basic functionality**
   ```bash
   docker run --rm hello-world
   ```

4. **Check disk space**
   ```bash
   docker system df
   df -h /var/lib/docker
   ```

5. **Restart daemon**
   ```bash
   sudo systemctl restart docker
   ```

## Configuration

No special configuration required. The check uses the default Docker socket location.

## Related Checks

- **resource-postgres**: Runs as Docker container
- **resource-redis**: Runs as Docker container
- **resource-qdrant**: Runs as Docker container
- All resource checks depend on Docker being available

## Auto-Heal Actions

When this check fails, autoheal may attempt:
1. Start Docker daemon via systemctl
2. Restart Docker daemon if hung
3. Clean up disk space if near capacity
4. Alert administrators for manual intervention

## Docker Alternatives

On systems without Docker, you may use:
- **Podman**: Drop-in replacement, check uses `podman info`
- **containerd**: Lower-level runtime, check uses `ctr version`

---

*Back to [Check Catalog](../check-catalog.md)*
