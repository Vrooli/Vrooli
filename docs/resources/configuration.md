# Vrooli Resource Configuration

## Overview

Vrooli resources use a multi-layered configuration system to manage settings, credentials, and operational parameters.

## Configuration Files

### Primary Configuration

**Repo `.vrooli/service.json`**
- Canonical project-level configuration checked into the repository
- Contains project resource enablement, lifecycle configuration, and service metadata
- This is the source of truth for the project definition

**Runtime `~/.vrooli/` state**
- User-machine runtime state, caches, archives, and generated operational files
- Not the canonical source of truth for repo-owned configuration

**Scenario `.vrooli/service.json`**
- Canonical scenario-level configuration
- Defines scenario metadata, lifecycle, and dependency declarations
- Used by the scenario system for orchestration

## Dependency Contract

Dependencies are declared as flat keyed maps.

- `dependencies.resources` is keyed by canonical resource name
- `dependencies.scenarios` is keyed by canonical scenario name
- Do not use `required` / `optional` arrays
- Do not use CLI aliases like `resource-claude-code` as config keys
- Put scenario dependencies only under `dependencies.scenarios`, never under `dependencies.resources`

Each dependency answers two separate questions:

- `required`: whether the dependency is functionally necessary
- `startup_policy`: how lifecycle orchestration should behave during startup

Supported `startup_policy` values:

- `must_start`: attempt startup and fail scenario startup if the dependency cannot start
- `try_start`: attempt startup and continue in degraded mode if the dependency cannot start
- `ignore`: do not auto-start; use only if already available

Recommended defaults:

- `required: true` => `startup_policy: "must_start"`
- `required: false` => `startup_policy: "try_start"`
- Use `ignore` for operator-managed or ambient dependencies

Example:

```json
{
  "dependencies": {
    "resources": {
      "postgres": {
        "enabled": true,
        "required": true,
        "startup_policy": "must_start",
        "description": "Primary relational store"
      },
      "qdrant": {
        "enabled": true,
        "required": false,
        "startup_policy": "try_start",
        "description": "Semantic search when available",
        "degraded_behavior": "Search falls back to lexical matching"
      }
    },
    "scenarios": {
      "prompt-manager": {
        "enabled": true,
        "required": false,
        "startup_policy": "try_start",
        "description": "Prompt lookup and skill guidance"
      }
    }
  }
}
```

## Resource-Specific Configuration

### Environment Variables

Resources can be configured through environment variables:

```bash
# Ollama configuration
export OLLAMA_HOST="localhost"
export OLLAMA_PORT="11434"
export OLLAMA_MODELS_PATH="/opt/ollama/models"

# PostgreSQL configuration
export POSTGRES_HOST="localhost"
export POSTGRES_PORT="5432"
export POSTGRES_DB="vrooli"
export POSTGRES_USER="vrooli"
export POSTGRES_PASSWORD="secret"

# Redis configuration
export REDIS_HOST="localhost"
export REDIS_PORT="6379"
export REDIS_DB="0"
```

### Configuration Files

Resources may have their own configuration files:

- **Ollama**: `~/.ollama/config.json`
- **PostgreSQL**: `~/.vrooli/postgres/postgresql.conf`
- **Redis**: `data/resources/redis/config/redis.conf`
- **n8n**: `~/.vrooli/n8n/.n8n/config.json`

## Configuration Management

### CLI Commands

```bash
# View configuration
vrooli resource status

# Check specific resource configuration
vrooli resource status <name>

# Update configuration (resource-specific)
resource-<name> config set <key> <value>
```

### Configuration Validation

```bash
# Validate all resource configurations
./scripts/resources/tools/validate-universal-contract.sh --resource <name>

# Fix configuration issues
./scripts/resources/tools/validate-dependency-contract.sh
```

## Security Considerations

### Credential Management

- **Vault Integration**: Use Vault for secure credential storage
- **Environment Variables**: Use environment variables for sensitive data
- **File Permissions**: Ensure configuration files have appropriate permissions
- **Secret Rotation**: Implement regular secret rotation

### Access Control

- **User Permissions**: Limit access to configuration files
- **Network Security**: Use appropriate network security measures
- **Audit Logging**: Log configuration changes and access

## Troubleshooting

### Common Configuration Issues

1. **Missing Configuration**: Ensure all required configuration is present
2. **Permission Errors**: Check file and directory permissions
3. **Network Issues**: Verify network connectivity and firewall settings
4. **Resource Conflicts**: Check for port conflicts and resource limits

### Debugging Configuration

```bash
# Check configuration status
vrooli resource status --verbose

# View resource logs
vrooli resource <name> logs

# Test resource connectivity
resource-<name> test
```

## Best Practices

### Configuration Design

- **Default Values**: Provide sensible defaults for all settings
- **Validation**: Validate configuration values on startup
- **Documentation**: Document all configuration options
- **Migration**: Provide migration paths for configuration changes

### Operational Practices

- **Backup**: Regularly backup configuration files
- **Version Control**: Use version control for configuration templates
- **Testing**: Test configuration changes in non-production environments
- **Monitoring**: Monitor configuration-related errors and issues 
