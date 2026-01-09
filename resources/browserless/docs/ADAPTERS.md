# Browserless Adapter Pattern

## Overview

The Browserless Adapter Pattern enables browserless to provide UI automation interfaces for other resources when their APIs are unavailable, broken, or limited. This creates a resilient fallback system where resources can leverage browserless's browser automation capabilities as alternative interfaces.

## Architecture

```
browserless/
├── adapters/
│   ├── common.sh        # Adapter framework utilities
│   ├── registry.sh      # Adapter discovery and registration
│   └── vault/          # HashiCorp Vault adapter
│       ├── api.sh      # Main adapter interface
│       └── manifest.json # Adapter metadata
```

## Usage

### The "for" Pattern

The adapter pattern introduces a new CLI syntax using the "for" keyword:

```bash
resource-browserless for <target-resource> <command> [options]
```

### Examples

#### Vault Adapter
```bash
# Add secrets via UI when API is unavailable
resource-browserless for vault add-secret secret/myapp username=admin

# List secrets through browser automation
resource-browserless for vault list-secrets secret/
```

## Creating New Adapters

### 1. Directory Structure

Create a new directory under `adapters/`:

```bash
browserless/adapters/your-resource/
├── api.sh          # Main adapter interface (required)
├── manifest.json   # Adapter metadata (recommended)
└── [other-files]   # Additional implementation files
```

### 2. Implement api.sh

Your `api.sh` must implement:

```bash
#!/usr/bin/env bash

# Source adapter framework
source "$(dirname "$(dirname "${BASH_SOURCE[0]}")")/common.sh"

# Initialize function
your_resource::init() {
    adapter::init "your_resource"
    # Load configuration
    adapter::load_target_config "your_resource"
    return 0
}

# Command dispatcher
your_resource::dispatch() {
    local command="${1:-}"
    shift || true
    
    your_resource::init
    
    if ! adapter::check_browserless_health; then
        return 1
    fi
    
    case "$command" in
        your-command)
            your_resource::your_command "$@"
            ;;
        help|--help|-h|"")
            your_resource::show_help
            ;;
        *)
            log::error "Unknown command: $command"
            return 1
            ;;
    esac
}

# Export dispatcher
export -f your_resource::dispatch
export -f your_resource::init
```

### 3. Create manifest.json

Document your adapter's capabilities:

```json
{
  "name": "your-resource",
  "version": "1.0.0",
  "description": "UI automation adapter for your resource",
  "provider": "browserless",
  "capabilities": [
    "capability-1",
    "capability-2"
  ],
  "requirements": {
    "browserless": "running",
    "target_resource": "your-resource"
  },
  "environment": {
    "YOUR_RESOURCE_URL": {
      "description": "Resource URL",
      "default": "http://localhost:8080",
      "required": false
    }
  },
  "fallback_priority": 5,
  "use_cases": [
    "Use case 1",
    "Use case 2"
  ]
}
```

## Adapter Framework Functions

The adapter framework provides these utilities:

### Core Functions

- `adapter::init(resource_name)` - Initialize adapter framework
- `adapter::check_browserless_health()` - Verify browserless is running
- `adapter::load_target_config(resource)` - Load resource configuration
- `adapter::execute_browser_function(code, timeout, persistent, session_id)` - Execute browser automation

### Registry Functions

- `adapter::register_capability(resource, capability, priority, description)` - Register adapter capability
- `adapter::list()` - List all available adapters
- `registry::find_adapter(resource, capability)` - Find best adapter for capability
- `registry::discover()` - Auto-discover all adapters

## Benefits

### 1. **Resilience**
Resources continue functioning even when APIs fail, providing uninterrupted service.

### 2. **Cost Optimization**
Switch between expensive APIs and browser automation based on usage patterns.

### 3. **Feature Completeness**
Access UI-only features that aren't exposed through APIs.

### 4. **Debugging**
Visual debugging of automation flows through browser automation.

### 5. **Emergency Access**
Bypass rate limits, authentication issues, or API outages.

## Best Practices

### 1. **Session Management**
Use persistent sessions to maintain authentication state:
```bash
resource-browserless for vault add-secret secret/myapp username=admin \
  --session "my-persistent-session"
```

### 2. **Error Handling**
Always check browserless health before operations:
```bash
if ! adapter::check_browserless_health; then
    log::error "Browserless not available"
    return 1
fi
```

### 3. **Timeout Configuration**
Set appropriate timeouts for long-running operations:
```bash
export VAULT_UI_TIMEOUT=120000  # 2 minutes
resource-browserless for vault add-secret secret/myapp username=admin
```

### 4. **Credential Security**
Use environment variables for sensitive data:
```bash
export VAULT_USER="user@example.com"
export VAULT_PASSWORD="secure-password"
resource-browserless for vault add-secret secret/myapp username=admin password="$VAULT_PASSWORD"
```

## Backward Compatibility

Legacy commands are maintained but will show deprecation warnings:

```bash
# Old syntax (deprecated)
resource-browserless for vault add-secret secret/myapp username=admin

# New syntax (recommended)
resource-browserless for vault add-secret secret/myapp username=admin
```

## Troubleshooting

### Adapter Not Found
```bash
# List available adapters
resource-browserless for --help

# Check adapter directory
ls resources/browserless/adapters/
```

### Browserless Not Running
```bash
# Start browserless
vrooli resource start browserless

# Check status
resource-browserless status
```

### Authentication Issues
```bash
# Set credentials
export VAULT_USER="your-email"
export VAULT_PASSWORD="your-password"
```

## Future Adapters

Planned adapters for the ecosystem:

- **grafana** - Dashboard automation and monitoring
- **jenkins** - CI/CD pipeline management
- **gitlab** - Repository and CI/CD operations
- **comfyui** - Image generation workflow management
- **nocodb** - Database UI operations

## Contributing

To contribute a new adapter:

1. Create adapter directory structure
2. Implement required functions in `api.sh`
3. Add comprehensive `manifest.json`
4. Write tests for your adapter
5. Document usage and examples
6. Submit pull request

## Architecture Benefits

The adapter pattern creates a **capability mesh** where:

- Resources provide services to each other
- Failures trigger automatic fallbacks
- Costs optimize dynamically
- Capabilities compound over time

This transforms Vrooli from a collection of resources into an **antifragile system** that gets stronger under stress.
