# Vrooli Resource Configuration

## Overview

Vrooli resources use a multi-layered configuration system to manage settings, credentials, and operational parameters.

## Configuration Files

### Primary Configuration

**`~/.vrooli/service.json`**
- Main service configuration file
- Contains resource enablement, basic settings, and service definitions
- Managed by the main CLI and resource management system

**`~/.vrooli/resource-registry/`**
- Directory containing resource-specific registry files
- Each resource has a JSON file defining its CLI interface and capabilities
- Used by the main CLI for command routing and discovery

**`~/.vrooli/scenarios.json`**
- Scenario-specific configuration
- Defines which scenarios are available and their resource requirements
- Used by the scenario system for resource orchestration

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
vrooli resource <name> status

# Update configuration (resource-specific)
resource-<name> config set <key> <value>
```

### Configuration Validation

```bash
# Validate all resource configurations
./tools/validate-interfaces.sh

# Fix configuration issues
./tools/fix-interface-compliance.sh
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