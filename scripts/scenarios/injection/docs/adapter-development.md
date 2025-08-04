# Resource Injection Adapter Development Guide

**Learn how to create injection adapters for new resources in the Vrooli ecosystem.**

## 🎯 Overview

This guide walks through creating injection adapters that enable resources to participate in Vrooli's declarative application deployment system. Adapters follow a standardized interface while allowing resource-specific customization.

## 🏗️ Adapter Architecture

### Core Principles

1. **Co-located**: Adapters live alongside resource management scripts
2. **Standardized Interface**: All adapters implement the same command interface
3. **Self-Contained**: Each adapter manages its own injection logic
4. **Validation First**: All configurations validated before injection
5. **Rollback Support**: Failed operations can be safely reverted

### File Structure

```
scripts/resources/category/resource-name/
├── manage.sh           # Existing resource management
├── inject.sh           # NEW: Injection adapter
├── config/             # Existing configuration
└── lib/                # Existing libraries
```

## 🔧 Implementation Steps

### Step 1: Create Injection Adapter

Create `inject.sh` with the standard interface:

```bash
#!/usr/bin/env bash
set -euo pipefail

# Resource Injection Adapter Template
# Replace RESOURCE_NAME with your actual resource name

DESCRIPTION="Inject data into RESOURCE_NAME"

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
RESOURCES_DIR="${SCRIPT_DIR}/../.."

# Source common utilities
source "${RESOURCES_DIR}/common.sh"

# Resource-specific configuration
readonly RESOURCE_BASE_URL="${RESOURCE_BASE_URL:-http://localhost:PORT}"
readonly RESOURCE_API_BASE="$RESOURCE_BASE_URL/api/v1"

# Operation tracking for rollback
declare -a RESOURCE_ROLLBACK_ACTIONS=()

#######################################
# Display usage information
#######################################
resource_inject::usage() {
    cat << EOF
RESOURCE_NAME Data Injection Adapter

USAGE:
    $0 [OPTIONS] CONFIG_JSON

DESCRIPTION:
    Injects data into RESOURCE_NAME based on scenario configuration.

OPTIONS:
    --validate    Validate the injection configuration
    --inject      Perform the data injection
    --status      Check status of injected data
    --rollback    Rollback injected data
    --help        Show this help message

EOF
}

#######################################
# Validate injection configuration
# Arguments:
#   $1 - configuration JSON
# Returns:
#   0 if valid, 1 if invalid
#######################################
resource_inject::validate_config() {
    local config="$1"
    
    log::info "Validating RESOURCE_NAME injection configuration..."
    
    # Basic JSON validation
    if ! echo "$config" | jq . >/dev/null 2>&1; then
        log::error "Invalid JSON in injection configuration"
        return 1
    fi
    
    # Resource-specific validation logic here
    # Example: Check for required fields, validate file paths, etc.
    
    log::success "RESOURCE_NAME injection configuration is valid"
    return 0
}

#######################################
# Perform data injection
# Arguments:
#   $1 - configuration JSON
# Returns:
#   0 if successful, 1 if failed
#######################################
resource_inject::inject_data() {
    local config="$1"
    
    log::header "🔄 Injecting data into RESOURCE_NAME"
    
    # Check resource accessibility
    if ! resource_inject::check_accessibility; then
        return 1
    fi
    
    # Clear previous rollback actions
    RESOURCE_ROLLBACK_ACTIONS=()
    
    # Implement injection logic here
    # Example: Parse config, call APIs, create files, etc.
    
    log::success "✅ RESOURCE_NAME data injection completed"
    return 0
}

#######################################
# Check injection status
# Arguments:
#   $1 - configuration JSON
# Returns:
#   0 if successful, 1 if failed
#######################################
resource_inject::check_status() {
    local config="$1"
    
    log::header "📊 Checking RESOURCE_NAME injection status"
    
    # Implement status checking logic here
    
    return 0
}

#######################################
# Rollback injection
# Arguments:
#   $1 - configuration JSON (optional)
#######################################
resource_inject::execute_rollback() {
    if [[ ${#RESOURCE_ROLLBACK_ACTIONS[@]} -eq 0 ]]; then
        log::info "No rollback actions to execute"
        return 0
    fi
    
    log::info "Executing rollback actions..."
    
    # Execute rollback actions in reverse order
    for ((i=${#RESOURCE_ROLLBACK_ACTIONS[@]}-1; i>=0; i--)); do
        local action="${RESOURCE_ROLLBACK_ACTIONS[i]}"
        IFS='|' read -r description command <<< "$action"
        
        log::info "Rollback: $description"
        
        if eval "$command"; then
            log::success "Rollback completed: $description"
        else
            log::error "Rollback failed: $description"
        fi
    done
    
    RESOURCE_ROLLBACK_ACTIONS=()
}

#######################################
# Check if resource is accessible
# Returns:
#   0 if accessible, 1 otherwise
#######################################
resource_inject::check_accessibility() {
    if ! system::is_command "curl"; then
        log::error "curl command not available"
        return 1
    fi
    
    if curl -s --max-time 5 "$RESOURCE_BASE_URL/health" >/dev/null 2>&1; then
        log::debug "RESOURCE_NAME is accessible at $RESOURCE_BASE_URL"
        return 0
    fi
    
    log::error "RESOURCE_NAME is not accessible at $RESOURCE_BASE_URL"
    return 1
}

#######################################
# Add rollback action
# Arguments:
#   $1 - description
#   $2 - rollback command
#######################################
resource_inject::add_rollback_action() {
    local description="$1"
    local command="$2"
    
    RESOURCE_ROLLBACK_ACTIONS+=("$description|$command")
    log::debug "Added rollback action: $description"
}

#######################################
# Main execution function
#######################################
resource_inject::main() {
    local action="$1"
    local config="${2:-}"
    
    if [[ -z "$config" ]]; then
        log::error "Configuration JSON required"
        resource_inject::usage
        exit 1
    fi
    
    case "$action" in
        "--validate")
            resource_inject::validate_config "$config"
            ;;
        "--inject")
            resource_inject::inject_data "$config"
            ;;
        "--status")
            resource_inject::check_status "$config"
            ;;
        "--rollback")
            resource_inject::execute_rollback
            ;;
        "--help")
            resource_inject::usage
            ;;
        *)
            log::error "Unknown action: $action"
            resource_inject::usage
            exit 1
            ;;
    esac
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    if [[ $# -eq 0 ]]; then
        resource_inject::usage
        exit 1
    fi
    
    resource_inject::main "$@"
fi
```

### Step 2: Make Adapter Executable

```bash
chmod +x scripts/resources/category/resource-name/inject.sh
```

### Step 3: Extend Resource Management

Add injection support to existing `manage.sh`:

```bash
# In the argument parsing section, add injection-config parameter
args::register \
    --name "injection-config" \
    --desc "JSON configuration for data injection" \
    --type "value" \
    --default ""

# In the action options, add injection actions
--options "install|uninstall|start|stop|restart|status|inject|validate-injection" \

# In the main case statement, add injection actions
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

### Step 4: Update JSON Schema (if needed)

If your resource introduces new injection types, update `../../../.vrooli/schemas/schema.json`:

```json
{
  "definitions": {
    "resourceInjection": {
      "properties": {
        "myNewType": {
          "type": "array",
          "description": "My new injection type",
          "items": {
            "$ref": "#/definitions/myNewTypeInjection"
          }
        }
      }
    },
    "myNewTypeInjection": {
      "type": "object",
      "required": ["name", "file"],
      "properties": {
        "name": {"type": "string"},
        "file": {"type": "string"},
        "enabled": {"type": "boolean", "default": true}
      }
    }
  }
}
```

## 📋 Resource-Specific Implementation Examples

### API-Based Resources (n8n, Windmill, etc.)

```bash
resource_inject::inject_workflows() {
    local workflows_config="$1"
    
    local workflow_count
    workflow_count=$(echo "$workflows_config" | jq 'length')
    
    for ((i=0; i<workflow_count; i++)); do
        local workflow
        workflow=$(echo "$workflows_config" | jq -c ".[$i]")
        
        local name file enabled
        name=$(echo "$workflow" | jq -r '.name')
        file=$(echo "$workflow" | jq -r '.file')
        enabled=$(echo "$workflow" | jq -r '.enabled // true')
        
        log::info "Injecting workflow: $name"
        
        # Import workflow via API
        local response
        if response=$(curl -s -X POST \
            -H "Content-Type: application/json" \
            -d @"$VROOLI_PROJECT_ROOT/$file" \
            "$RESOURCE_API_BASE/workflows/import"); then
            
            local workflow_id
            workflow_id=$(echo "$response" | jq -r '.id')
            
            log::success "Imported workflow: $name (ID: $workflow_id)"
            
            # Add rollback action
            resource_inject::add_rollback_action \
                "Remove workflow: $name" \
                "curl -s -X DELETE '$RESOURCE_API_BASE/workflows/$workflow_id'"
        else
            log::error "Failed to import workflow: $name"
            return 1
        fi
    done
}
```

### File-Based Resources (PostgreSQL, etc.)

```bash
resource_inject::inject_sql() {
    local data_config="$1"
    
    local data_count
    data_count=$(echo "$data_config" | jq 'length')
    
    for ((i=0; i<data_count; i++)); do
        local data_item
        data_item=$(echo "$data_config" | jq -c ".[$i]")
        
        local type file
        type=$(echo "$data_item" | jq -r '.type')
        file=$(echo "$data_item" | jq -r '.file')
        
        log::info "Injecting $type: $file"
        
        local sql_file="$VROOLI_PROJECT_ROOT/$file"
        
        case "$type" in
            "schema")
                if psql -h localhost -p 5432 -U postgres -d mydb -f "$sql_file"; then
                    log::success "Applied schema: $file"
                    
                    # Add rollback action (if possible)
                    resource_inject::add_rollback_action \
                        "Rollback schema: $file" \
                        "echo 'Manual schema rollback required for: $file'"
                else
                    log::error "Failed to apply schema: $file"
                    return 1
                fi
                ;;
            "seed")
                # Similar implementation for seed data
                ;;
        esac
    done
}
```

### Container-Based Resources (MinIO, etc.)

```bash
resource_inject::inject_buckets() {
    local buckets_config="$1"
    
    local bucket_count
    bucket_count=$(echo "$buckets_config" | jq 'length')
    
    for ((i=0; i<bucket_count; i++)); do
        local bucket
        bucket=$(echo "$buckets_config" | jq -c ".[$i]")
        
        local name policy
        name=$(echo "$bucket" | jq -r '.name')
        policy=$(echo "$bucket" | jq -r '.policy // "private"')
        
        log::info "Creating bucket: $name"
        
        # Create bucket via mc command or API
        if mc mb "local/$name" 2>/dev/null; then
            log::success "Created bucket: $name"
            
            # Set policy if specified
            if [[ "$policy" != "private" ]]; then
                mc policy set "$policy" "local/$name"
                log::info "Set bucket policy: $policy"
            fi
            
            # Add rollback action
            resource_inject::add_rollback_action \
                "Remove bucket: $name" \
                "mc rb --force 'local/$name'"
        else
            log::error "Failed to create bucket: $name"
            return 1
        fi
    done
}
```

## 🧪 Testing Your Adapter

### Basic Testing

```bash
# Test validation
./scripts/resources/category/resource-name/inject.sh --validate '{
  "myType": [{"name": "test", "file": "test.json"}]
}'

# Test with invalid config
./scripts/resources/category/resource-name/inject.sh --validate '{
  "invalid": "config"
}'

# Test injection (ensure resource is running)
./scripts/resources/category/resource-name/inject.sh --inject '{
  "myType": [{"name": "test", "file": "test.json", "enabled": true}]
}'
```

### Integration Testing

```bash
# Test via resource management
./scripts/resources/category/resource-name/manage.sh \
  --action validate-injection \
  --injection-config '{"myType": [...]}'

# Test via main orchestrator
./scripts/resources/index.sh \
  --action inject \
  --scenario test-scenario
```

### Schema Testing

```bash
# Validate your schema changes
./scripts/resources/_injection/schema-validator.sh --action check-schema

# Test with example scenario
./scripts/resources/_injection/schema-validator.sh \
  --action validate \
  --config-file ./test-scenario.json
```

## 🛡️ Best Practices

### Validation

- **Validate early**: Check configuration before making any changes
- **File existence**: Verify all referenced files exist
- **JSON validation**: Use `jq` to validate JSON structures
- **Resource accessibility**: Check if resource is running before injection

### Error Handling

- **Descriptive errors**: Provide clear error messages with remediation steps
- **Graceful failures**: Don't crash on individual item failures
- **Rollback support**: Always provide rollback actions for successful operations

### Security

- **Input validation**: Sanitize all inputs before using them
- **Path traversal**: Validate file paths to prevent directory traversal
- **Command injection**: Use parameterized commands, avoid `eval` with user input
- **Credential handling**: Never log sensitive information

### Performance

- **Batch operations**: Group similar operations when possible
- **Progress indication**: Show progress for long-running operations
- **Concurrent operations**: Consider parallel processing for independent operations

## 📖 Common Patterns

### Configuration Parsing

```bash
# Parse array of items
local items_config="$1"
local item_count
item_count=$(echo "$items_config" | jq 'length')

for ((i=0; i<item_count; i++)); do
    local item
    item=$(echo "$items_config" | jq -c ".[$i]")
    
    # Extract fields
    local name file enabled
    name=$(echo "$item" | jq -r '.name')
    file=$(echo "$item" | jq -r '.file')
    enabled=$(echo "$item" | jq -r '.enabled // true')
    
    # Process item...
done
```

### File Path Resolution

```bash
# Resolve file path relative to Vrooli root
local file_path="$1"
local resolved_path="$VROOLI_PROJECT_ROOT/$file_path"

if [[ ! -f "$resolved_path" ]]; then
    log::error "File not found: $resolved_path"
    return 1
fi
```

### API Error Handling

```bash
# Make API call with error handling
local response
local http_code

response=$(curl -s -w "%{http_code}" \
    -X POST \
    -H "Content-Type: application/json" \
    -d "$data" \
    "$API_ENDPOINT")

http_code="${response: -3}"
response="${response%???}"

case "$http_code" in
    200|201|202)
        log::success "API call successful"
        ;;
    400)
        log::error "Bad request: $(echo "$response" | jq -r '.message // "Unknown error"')"
        return 1
        ;;
    *)
        log::error "API call failed with status: $http_code"
        return 1
        ;;
esac
```

## 🚀 Advanced Features

### Conditional Injection

```bash
# Only inject if condition is met
resource_inject::inject_conditionally() {
    local config="$1"
    
    # Check if resource has feature X
    if resource_inject::has_feature "advanced-workflows"; then
        resource_inject::inject_advanced_workflows "$config"
    else
        log::info "Resource doesn't support advanced workflows, skipping"
    fi
}
```

### Dependency Management

```bash
# Ensure dependencies are satisfied
resource_inject::check_dependencies() {
    local required_resources=("postgres" "redis")
    
    for resource in "${required_resources[@]}"; do
        if ! resources::is_service_running "$(resources::get_default_port "$resource")"; then
            log::error "Required resource not running: $resource"
            return 1
        fi
    done
}
```

### Progress Tracking

```bash
# Show progress for long operations
resource_inject::inject_with_progress() {
    local items="$1"
    local total_count
    total_count=$(echo "$items" | jq 'length')
    
    for ((i=0; i<total_count; i++)); do
        local progress=$((i * 100 / total_count))
        log::info "Progress: $progress% ($((i+1))/$total_count)"
        
        # Process item...
    done
}
```

## 🔧 Troubleshooting

### Common Issues

1. **Resource not accessible**: Ensure resource is running and API is available
2. **File not found**: Check file paths are relative to Vrooli root
3. **Permission denied**: Verify script has execution permissions
4. **JSON parsing errors**: Validate JSON syntax before processing

### Debugging

```bash
# Enable debug logging
export LOG_LEVEL=debug

# Test individual functions
resource_inject::validate_config "$test_config"

# Check resource accessibility
resource_inject::check_accessibility
```

### Testing with curl

```bash
# Test resource API directly
curl -v http://localhost:PORT/api/endpoint

# Test with sample data
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{"test": "data"}' \
  http://localhost:PORT/api/endpoint
```

## 📚 Reference

- **[Main README](../README.md)**: System overview and usage
- **[API Reference](api-reference.md)**: Complete interface specification
- **[Schema Documentation](../config/schema.json)**: Configuration format specification
- **[Example Adapters](../../../automation/n8n/inject.sh)**: Working implementation examples

---

**Creating injection adapters enables resources to participate in Vrooli's declarative application deployment ecosystem.**