# Vrooli Resource CLI Framework

## Overview

The Vrooli Resource CLI Framework provides a standardized way to create command-line interfaces for individual resources. This framework ensures consistency across all resource CLIs while providing flexibility for resource-specific functionality.

## Architecture

### Command Patterns

Resources support two primary command patterns:

1. **Main CLI Integration**: `vrooli resource <name> <command>`
2. **Resource-Specific CLI**: `resource-<name> <command>`

### Installation

Resource CLIs are automatically installed to `~/.local/bin/` when resources are set up, making them globally accessible.

## Framework Components

### Core Files

- **`cli.sh`**: Main CLI entry point for each resource
- **`lib/cli-command-framework.sh`**: Shared CLI framework utilities
- **`config/messages.sh`**: Resource-specific messages and help text
- **`lib/`**: Resource-specific functionality libraries

### CLI Command Framework

The `cli-command-framework.sh` provides:

- **Command Registration**: Standardized way to register commands
- **Help Generation**: Automatic help text generation
- **Argument Parsing**: Consistent argument handling
- **Error Handling**: Standardized error reporting
- **Format Support**: JSON and text output formatting

## Creating a New Resource CLI

### 1. Basic Structure

```bash
resource-name/
├── cli.sh                    # Main CLI entry point
├── config/
│   ├── defaults.sh           # Default configuration
│   └── messages.sh           # Help messages
├── lib/
│   ├── common.sh             # Common functions
│   ├── install.sh            # Installation logic
│   ├── status.sh             # Status checking
│   └── [other].sh            # Resource-specific functions
└── manage.sh                 # (Deprecated) Legacy management
```

### 2. CLI Entry Point

```bash
#!/usr/bin/env bash
# Resource CLI entry point

set -euo pipefail

# Source framework and utilities
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/cli-command-framework.sh"
source "${RESOURCE_DIR}/lib/common.sh"

# Initialize CLI framework
cli::init "resource-name" "Resource Description"

# Register commands
cli::register_command "status" "Show service status" "resource_status"
cli::register_command "install" "Install resource" "resource_install" "modifies-system"
cli::register_command "start" "Start resource" "resource_start" "modifies-system"
cli::register_command "stop" "Stop resource" "resource_stop" "modifies-system"

# Execute CLI
cli::main "$@"
```

### 3. Command Implementation

```bash
# Example status command
resource_status() {
    local format="${1:-text}"
    
    # Check if service is running
    if pgrep -f "resource-service" >/dev/null; then
        local status="running"
        local port="8080"
    else
        local status="stopped"
        local port=""
    fi
    
    # Format output
    case "$format" in
        json)
            jq -n \
                --arg status "$status" \
                --arg port "$port" \
                '{
                    "status": $status,
                    "port": $port,
                    "healthy": ($status == "running")
                }'
            ;;
        text|*)
            echo "Status: $status"
            [[ -n "$port" ]] && echo "Port: $port"
            ;;
    esac
}
```

## Integration with Main CLI

### Resource Registration

Resources are automatically registered with the main `vrooli` CLI through:

1. **CLI Installation**: Resource CLIs are installed to `~/.local/bin/`
2. **Command Routing**: The main CLI routes `vrooli resource <name>` commands to resource CLIs
3. **Status Integration**: Resource status is aggregated in `vrooli resource status`

### Configuration Management

Resource configuration is managed through:

- **`~/.vrooli/service.json`**: Main service configuration
- **`~/.vrooli/resource-registry/`**: Resource-specific registry files
- **Environment Variables**: Resource-specific configuration

## Best Practices

### 1. Command Design

- **Consistent Naming**: Use consistent command names across resources
- **Clear Help**: Provide clear, actionable help text
- **Error Handling**: Implement proper error handling and reporting
- **Format Support**: Support both text and JSON output formats

### 2. Function Implementation

- **Thin Wrappers**: Keep CLI functions as thin wrappers around lib/ functions
- **Shared Logic**: Use shared functions from `lib/` directory
- **Validation**: Validate inputs and provide helpful error messages
- **Logging**: Use the standard logging framework

### 3. Testing

- **Unit Tests**: Test individual functions in `lib/`
- **Integration Tests**: Test CLI commands end-to-end
- **Validation**: Ensure commands work with the main CLI

## Migration from Legacy Scripts

### From manage.sh

Legacy `manage.sh` scripts should be migrated to the CLI framework:

1. **Extract Functions**: Move logic to `lib/` functions
2. **Create CLI**: Implement `cli.sh` using the framework
3. **Update Documentation**: Update help text and examples
4. **Test Integration**: Ensure compatibility with main CLI

### Benefits of Migration

- **Consistency**: Standardized command patterns
- **Integration**: Better integration with main CLI
- **Maintainability**: Easier to maintain and extend
- **Testing**: Better testing capabilities

## Examples

### Ollama CLI

```bash
# Status check
resource-ollama status

# Model management
resource-ollama list-models
resource-ollama pull-model llama3.2:3b

# AI interaction
resource-ollama generate llama3.2:3b "Write a haiku"
```

### PostgreSQL CLI

```bash
# Status check
resource-postgres status

# Database operations
resource-postgres backup
resource-postgres restore backup.sql
```

## Troubleshooting

### Common Issues

1. **CLI Not Found**: Ensure resource is properly installed and CLI is in PATH
2. **Permission Errors**: Check file permissions and ownership
3. **Configuration Issues**: Verify configuration files and environment variables
4. **Integration Problems**: Check main CLI routing and resource registration

### Debugging

- **Verbose Mode**: Use `--verbose` flag for detailed output
- **Log Files**: Check resource-specific log files
- **Status Commands**: Use status commands to check resource health
- **Configuration**: Verify configuration with `vrooli resource status` 