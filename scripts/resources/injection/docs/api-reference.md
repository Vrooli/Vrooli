# Resource Injection API Reference

**Complete specification of interfaces, configuration formats, and command-line APIs.**

## 🎯 Overview

This document provides comprehensive API documentation for the Resource Data Injection System, including command-line interfaces, configuration schemas, and adapter specifications.

## 🎛️ Central Engine API

### Command Line Interface

```bash
./scripts/resources/injection/engine.sh [OPTIONS]
```

#### Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `--action` | enum | `inject` | Action to perform: `inject`, `validate`, `status`, `rollback`, `list-scenarios` |
| `--scenario` | string | - | Scenario name to inject |
| `--config-file` | path | `~/.vrooli/scenarios.json` | Path to scenarios configuration file |
| `--all-active` | boolean | `no` | Process all active scenarios |
| `--force` | boolean | `no` | Force injection even if validation fails |
| `--dry-run` | boolean | `no` | Show what would be injected without doing it |
| `--yes` | boolean | `no` | Auto-confirm prompts |
| `--help` | - | - | Show help message |

#### Examples

```bash
# Inject specific scenario
./scripts/resources/injection/engine.sh --action inject --scenario vrooli-core

# Inject all active scenarios
./scripts/resources/injection/engine.sh --action inject --all-active yes

# Validate scenario configuration
./scripts/resources/injection/engine.sh --action validate --scenario test-app

# List available scenarios
./scripts/resources/injection/engine.sh --action list-scenarios

# Dry run injection
./scripts/resources/injection/engine.sh --action inject --scenario test-app --dry-run yes
```

#### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Configuration validation failed |
| 3 | Resource injection failed |
| 4 | Dependency resolution failed |

## 📋 Schema Validator API

### Command Line Interface

```bash
./scripts/resources/injection/schema-validator.sh [OPTIONS]
```

#### Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `--action` | enum | `validate` | Action: `validate`, `init`, `check-schema` |
| `--config-file` | path | `~/.vrooli/scenarios.json` | Configuration file path |
| `--verbose` | boolean | `no` | Enable verbose output |
| `--help` | - | - | Show help message |

#### Examples

```bash
# Validate configuration file
./scripts/resources/injection/schema-validator.sh --action validate

# Initialize new configuration
./scripts/resources/injection/schema-validator.sh --action init --config-file ./new-config.json

# Check schema validity
./scripts/resources/injection/schema-validator.sh --action check-schema
```

## 🔌 Resource Adapter API

### Standard Interface

All resource adapters must implement this interface:

```bash
./scripts/resources/category/resource/inject.sh [ACTION] [CONFIG_JSON]
```

#### Actions

| Action | Description | Required Config | Returns |
|--------|-------------|----------------|---------|
| `--validate` | Validate injection configuration | JSON config | 0=valid, 1=invalid |
| `--inject` | Perform data injection | JSON config | 0=success, 1=failed |
| `--status` | Check injection status | JSON config | 0=healthy, 1=issues |
| `--rollback` | Rollback injection | JSON config (optional) | 0=success, 1=failed |
| `--help` | Show help message | - | 0 |

#### Examples

```bash
# Validate n8n configuration
./scripts/resources/automation/n8n/inject.sh --validate '{
  "workflows": [{"name": "test", "file": "test.json"}]
}'

# Inject n8n workflows
./scripts/resources/automation/n8n/inject.sh --inject '{
  "workflows": [
    {
      "name": "user-onboarding",
      "file": "workflows/onboarding.json",
      "enabled": true,
      "tags": ["users"]
    }
  ]
}'

# Check injection status
./scripts/resources/automation/n8n/inject.sh --status '{
  "workflows": [{"name": "user-onboarding"}]
}'
```

## 📄 Configuration Schema

### Root Configuration

```typescript
interface ScenariosConfiguration {
  version: string;                    // Schema version (e.g., "1.0.0")
  scenarios: Record<string, Scenario>; // Available scenarios
  active?: string[];                  // Active scenario names
}
```

### Scenario Definition

```typescript
interface Scenario {
  description: string;           // Human-readable description
  version: string;              // Scenario version (semantic versioning)
  dependencies?: string[];      // Other scenarios to inject first
  resources: Record<string, ResourceInjectionConfig>;
}
```

### Resource Injection Configuration

```typescript
interface ResourceInjectionConfig {
  workflows?: WorkflowInjection[];      // n8n workflows
  scripts?: ScriptInjection[];          // Windmill scripts
  apps?: AppInjection[];                // Windmill applications
  flows?: FlowInjection[];              // Node-RED flows
  data?: DataInjection[];               // Database/file data
  buckets?: BucketInjection[];          // Storage buckets
  models?: ModelInjection[];            // AI models
  credentials?: CredentialInjection[];  // Authentication configs
  configurations?: ConfigurationInjection[]; // Custom configurations
}
```

## 🔧 Resource-Specific Configurations

### n8n Workflows

```typescript
interface WorkflowInjection {
  name: string;           // Workflow name
  file: string;           // Path to workflow JSON file (relative to Vrooli root)
  enabled?: boolean;      // Whether to activate workflow (default: true)
  tags?: string[];        // Tags for organization
}

interface CredentialInjection {
  name: string;           // Credential name
  type: 'httpAuth' | 'apiKey' | 'oauth2' | 'basic' | 'bearer';
  config: {
    url?: string;         // Target URL
    authType?: string;    // Authentication type
    [key: string]: any;   // Additional config
  };
}
```

#### Example

```json
{
  "n8n": {
    "workflows": [
      {
        "name": "user-registration",
        "file": "scenarios/saas/workflows/user-registration.json",
        "enabled": true,
        "tags": ["users", "auth"]
      }
    ],
    "credentials": [
      {
        "name": "api-auth",
        "type": "httpAuth",
        "config": {
          "url": "http://localhost:3000/api",
          "authType": "bearer"
        }
      }
    ]
  }
}
```

### Windmill Scripts & Apps

```typescript
interface ScriptInjection {
  path: string;           // Script path in Windmill (e.g., "f/myapp/script")
  file: string;           // Path to script file
  summary?: string;       // Script description
  schema?: string;        // Path to JSON schema file
  language?: 'typescript' | 'python' | 'bash' | 'javascript' | 'go' | 'rust';
}

interface AppInjection {
  path: string;           // App path in Windmill
  file: string;           // Path to app configuration JSON
  summary?: string;       // App description
}
```

#### Example

```json
{
  "windmill": {
    "scripts": [
      {
        "path": "f/ecommerce/inventory_sync",
        "file": "scenarios/ecommerce/scripts/inventory_sync.ts",
        "summary": "Synchronize inventory across platforms",
        "language": "typescript"
      }
    ],
    "apps": [
      {
        "path": "f/ecommerce/dashboard",
        "file": "scenarios/ecommerce/apps/dashboard.json",
        "summary": "E-commerce management dashboard"
      }
    ]
  }
}
```

### Node-RED Flows

```typescript
interface FlowInjection {
  name: string;           // Flow name
  file: string;           // Path to flow JSON file
  deploy?: boolean;       // Whether to deploy automatically (default: true)
}
```

#### Example

```json
{
  "node-red": {
    "flows": [
      {
        "name": "data-ingestion",
        "file": "scenarios/analytics/flows/data-ingestion.json",
        "deploy": true
      }
    ]
  }
}
```

### Database Data

```typescript
interface DataInjection {
  type: 'schema' | 'seed' | 'migration' | 'file' | 'directory';
  file: string;           // Path to data file
  bucket?: string;        // Target bucket (for file storage)
  source?: string;        // Source path (for directory injection)
  recursive?: boolean;    // Process directories recursively
}
```

#### Example

```json
{
  "postgres": {
    "data": [
      {
        "type": "schema",
        "file": "scenarios/ecommerce/sql/schema.sql"
      },
      {
        "type": "seed",
        "file": "scenarios/ecommerce/sql/sample-products.sql"
      }
    ]
  }
}
```

### Storage Buckets

```typescript
interface BucketInjection {
  name: string;                        // Bucket name
  policy?: 'private' | 'public-read' | 'public-read-write';
  versioning?: boolean;                // Enable versioning
}
```

#### Example

```json
{
  "minio": {
    "buckets": [
      {
        "name": "product-images",
        "policy": "public-read"
      },
      {
        "name": "user-uploads",
        "policy": "private",
        "versioning": true
      }
    ],
    "data": [
      {
        "type": "directory",
        "bucket": "product-images",
        "source": "scenarios/ecommerce/assets/",
        "recursive": true
      }
    ]
  }
}
```

### AI Models

```typescript
interface ModelInjection {
  name: string;           // Model name
  source?: string;        // Model source (URL, file, registry)
  type?: 'huggingface' | 'ollama' | 'gguf' | 'safetensors' | 'pytorch';
  config?: object;        // Model-specific configuration
}
```

#### Example

```json
{
  "ollama": {
    "models": [
      {
        "name": "customer-support",
        "source": "modelfiles/customer-support.modelfile",
        "type": "ollama"
      }
    ]
  }
}
```

## 🔄 Integration APIs

### Resource Management Integration

Resource `manage.sh` scripts should support injection actions:

```bash
# Add to argument parsing
args::register \
    --name "injection-config" \
    --desc "JSON configuration for data injection" \
    --type "value" \
    --default ""

# Add to action options
--options "install|uninstall|start|stop|restart|status|inject|validate-injection"

# Add to case statement
"inject")
    if [[ -z "$INJECTION_CONFIG" ]]; then
        log::error "Injection configuration required"
        exit 1
    fi
    "${SCRIPT_DIR}/inject.sh" --inject "$INJECTION_CONFIG"
    ;;
"validate-injection")
    if [[ -z "$INJECTION_CONFIG" ]]; then
        log::error "Injection configuration required" 
        exit 1
    fi
    "${SCRIPT_DIR}/inject.sh" --validate "$INJECTION_CONFIG"
    ;;
```

### Main Resource Orchestrator Integration

The main orchestrator supports injection actions:

```bash
# Inject specific scenario
./scripts/resources/index.sh --action inject --scenario SCENARIO_NAME

# Inject all active scenarios
./scripts/resources/index.sh --action inject-all

# With custom config file
./scripts/resources/index.sh --action inject --scenario test --scenarios-config ./test.json
```

## 🛠️ Adapter Implementation Requirements

### Required Functions

Every adapter must implement these functions:

```bash
# Validation
resource_inject::validate_config(config_json) -> exit_code

# Injection
resource_inject::inject_data(config_json) -> exit_code

# Status checking
resource_inject::check_status(config_json) -> exit_code

# Rollback
resource_inject::execute_rollback() -> exit_code

# Accessibility check
resource_inject::check_accessibility() -> exit_code

# Usage information
resource_inject::usage() -> void
```

### Required Variables

```bash
# Resource configuration
readonly RESOURCE_BASE_URL="${RESOURCE_BASE_URL:-http://localhost:PORT}"
readonly RESOURCE_API_BASE="$RESOURCE_BASE_URL/api/v1"

# Rollback tracking
declare -a RESOURCE_ROLLBACK_ACTIONS=()
```

### Error Handling Standards

```bash
# Log levels
log::error "Critical error message"
log::warn "Warning message"
log::info "Informational message"
log::success "Success message"
log::debug "Debug information"

# Exit codes
return 0  # Success
return 1  # General error
return 2  # Validation error
return 3  # Network/connectivity error
```

## 📊 Response Formats

### JSON Schema Validation Response

```json
{
  "valid": true,
  "errors": [],
  "warnings": ["Optional warning message"]
}
```

### Injection Status Response

```json
{
  "status": "healthy|degraded|failed",
  "items": [
    {
      "name": "item-name",
      "type": "workflow|script|data",
      "status": "present|missing|error",
      "details": "Additional information"
    }
  ],
  "summary": {
    "total": 5,
    "healthy": 4,
    "failed": 1
  }
}
```

## 🔐 Security Considerations

### Input Validation

- Validate all JSON inputs with `jq`
- Sanitize file paths to prevent directory traversal
- Validate URLs and network endpoints
- Check file permissions before reading

### Credential Handling

- Never log sensitive information
- Use secure storage for credentials
- Encrypt credentials at rest
- Validate credential formats

### Command Injection Prevention

```bash
# Safe command construction
local safe_name
safe_name=$(printf '%q' "$user_input")

# Avoid direct command interpolation
# BAD: eval "command $user_input"
# GOOD: command "$safe_name"
```

## 🧪 Testing APIs

### Unit Testing

```bash
# Test configuration validation
test_validate_config() {
    local result
    result=$(resource_inject::validate_config '{"valid": "config"}')
    assertEquals 0 $?
}

# Test error handling
test_invalid_config() {
    local result
    result=$(resource_inject::validate_config '{"invalid": config}')
    assertEquals 1 $?
}
```

### Integration Testing

```bash
# Test with running resource
test_injection_with_running_resource() {
    # Start resource
    resource::start
    
    # Test injection
    local result
    result=$(resource_inject::inject_data "$test_config")
    assertEquals 0 $?
    
    # Verify injection
    resource_inject::check_status "$test_config"
    assertEquals 0 $?
}
```

## 📚 Reference Implementation

See the n8n adapter for a complete reference implementation:

- **Implementation**: `scripts/resources/automation/n8n/inject.sh`
- **Integration**: `scripts/resources/automation/n8n/manage.sh`
- **Testing**: Test with running n8n instance

## 🔗 Related Documentation

- **[Main README](../README.md)**: System overview and usage
- **[Adapter Development Guide](adapter-development.md)**: Creating new adapters
- **[Troubleshooting Guide](troubleshooting.md)**: Common issues and solutions
- **[Configuration Schema](../../../../.vrooli/schemas/schema.json)**: JSON schema specification

---

**This API reference provides the complete specification for implementing and using the Resource Data Injection System.**