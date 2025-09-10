# JupyterHub Resource

Multi-user server that spawns, manages, and proxies multiple instances of Jupyter notebook servers for collaborative interactive computing.

## Quick Start

```bash
# Install and start JupyterHub
resource-jupyterhub manage install
resource-jupyterhub manage start --wait

# Check status
resource-jupyterhub status

# Access the hub
open http://localhost:8000
```

## Features

- **Multi-User Support**: Isolated notebook environments for each user
- **Authentication**: OAuth/GitHub/Google authentication support
- **Resource Management**: CPU/memory limits per user
- **Persistent Storage**: User notebooks persist across sessions
- **Extension Support**: JupyterLab extensions and kernels

## Configuration

Configuration is managed through environment variables and `config/defaults.sh`:

```bash
# Authentication method (native, oauth, github)
JUPYTERHUB_AUTH_TYPE=native

# Resource limits per user
JUPYTERHUB_CPU_LIMIT=2
JUPYTERHUB_MEM_LIMIT=4G

# Docker image for user servers
JUPYTERHUB_IMAGE=jupyter/scipy-notebook:latest
```

## Usage Examples

### Managing Users

```bash
# List users
resource-jupyterhub content list --type users

# Create a new user
resource-jupyterhub content add --type user --name alice

# Start a user's server
resource-jupyterhub content spawn --user alice
```

### Managing Extensions

```bash
# List available extensions
resource-jupyterhub content list --type extensions

# Install an extension
resource-jupyterhub content add --type extension --name @jupyter-widgets/jupyterlab-manager
```

### Resource Profiles

```bash
# List available profiles
resource-jupyterhub content list --type profiles

# Assign profile to user
resource-jupyterhub content execute --command "assign-profile alice large"
```

## Integration

JupyterHub integrates with:
- **PostgreSQL**: User database and session storage
- **Docker**: Container-based user isolation
- **Redis**: Session caching (optional)
- **Ollama**: AI models within notebooks (optional)

## Troubleshooting

### Hub Won't Start
- Check PostgreSQL is running: `resource-postgres status`
- Verify Docker access: `docker ps`
- Check logs: `resource-jupyterhub logs`

### User Can't Spawn Server
- Check Docker resources: `docker system df`
- Verify user exists: `resource-jupyterhub content list --type users`
- Check spawner logs: `resource-jupyterhub logs --user <username>`

### Authentication Issues
- Verify OAuth credentials in environment
- Check callback URL configuration
- Review auth logs: `resource-jupyterhub logs --component auth`

## Testing

```bash
# Run all tests
resource-jupyterhub test all

# Quick smoke test
resource-jupyterhub test smoke

# Integration tests
resource-jupyterhub test integration
```

## Security Notes

- User servers are isolated in Docker containers
- Authentication required for all access
- Resource limits prevent runaway processes
- Network policies isolate user traffic

## API Access

The JupyterHub API is available at `http://localhost:8001/hub/api`:

```bash
# Get user list (requires admin token)
curl -H "Authorization: token $JUPYTERHUB_API_TOKEN" \
  http://localhost:8001/hub/api/users

# Start a user's server
curl -X POST -H "Authorization: token $JUPYTERHUB_API_TOKEN" \
  http://localhost:8001/hub/api/users/alice/server
```

## Additional Resources

- [JupyterHub Documentation](https://jupyterhub.readthedocs.io)
- [Configuration Reference](docs/configuration.md)
- [API Documentation](docs/api.md)
- [Security Guide](docs/security.md)