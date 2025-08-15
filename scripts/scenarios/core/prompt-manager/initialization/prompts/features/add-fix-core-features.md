# Core Vrooli Feature Development and Enhancement

You are tasked with developing or fixing core Vrooli platform features. This includes the shared scripts, the Vrooli CLI, the main lifecycle management system, and foundational utilities that make the entire platform robust and reliable.

## Understanding Core Vrooli Features

**What Core Features Are**: The foundational infrastructure that powers all of Vrooli - the CLI, lifecycle management, shared utilities, configuration systems, and platform services that everything else depends on.

**Why This Matters**: Core features are the foundation that makes everything else possible. Improvements here make the ENTIRE platform more robust, reliable, and capable. A better CLI improves developer experience. Better lifecycle management makes all scenarios more reliable.

## Pre-Implementation Research

**Essential Reading**: Study main CLI at `cli/vrooli` and lifecycle management at `scripts/manage.sh`. Examine shared libraries in `scripts/lib/` and integration patterns throughout resource and scenario systems.

## Core System Architecture

### 1. CLI System (`cli/vrooli`)
The main user interface for all Vrooli operations:

```bash
# Study CLI structure
head -50 cli/vrooli

# Understand command delegation
grep -A 20 "command handlers" cli/vrooli
```

**CLI Components**:
- Command parsing and routing
- Environment detection and setup
- Delegation to specific handlers
- Help system and documentation
- Error handling and logging

### 2. Lifecycle Management (`scripts/manage.sh`)
The core orchestration system:

```bash
# Study lifecycle phases
grep -A 10 "execute_phase" scripts/manage.sh

# Understand service.json integration
grep -A 20 "json::" scripts/manage.sh
```

**Lifecycle Components**:
- Phase execution (setup, develop, build, test, deploy)
- Service.json configuration processing
- Port allocation and management
- Background process management
- Environment variable handling

### 3. Shared Libraries (`scripts/lib/`)
Common utilities used throughout the platform:

```bash
# Study utility categories
ls -la scripts/lib/
tree scripts/lib/utils/
```

**Library Categories**:
- **Utils**: Logging, JSON processing, argument parsing
- **System**: Dependencies, networking, security
- **Service**: Secrets management, configuration
- **Runtimes**: Docker, Kubernetes, language runtimes

## Implementation Phases

### Phase 1: CLI & Interface Development

When adding new CLI commands:

```bash
# Basic CLI command structure
cat > cli/commands/your-command.sh << 'EOF'
#!/usr/bin/env bash
set -euo pipefail

your_command::main() {
    # Handle --help, then execute
}

your_command::execute() {
    # Command logic
}
EOF
```

**Full template**: See any file in `cli/commands/`

### Phase 2: System Integration

**Lifecycle Phase Structure**:
```json
{
  "lifecycle": {
    "your-phase": {
      "steps": [{ "name": "step", "run": "command" }]
    }
  }
}
```
**Examples**: Check `.vrooli/service.json` in any scenario

### Phase 3: Testing & Validation

**Utility Pattern**:
```bash
#!/usr/bin/env bash
set -euo pipefail

your_utility::function_name() {
    # Validate, process, return
}

export -f your_utility::function_name
```
**Reference utilities**: `scripts/lib/utils/*.sh`

## Development Areas

### 1. CLI Enhancements

**Command Development**:
- New command categories (resource, scenario, config, etc.)
- Improved help system and documentation
- Enhanced error handling and user feedback
- Auto-completion support
- Interactive mode improvements

**Testing CLI Commands**:
```bash
# Test CLI functionality
./cli/vrooli --help
./cli/vrooli your-command --help
./cli/vrooli your-command test-args
```

### 2. Lifecycle Management Improvements

**Phase Management**:
- New lifecycle phases for specialized operations
- Better error handling and recovery
- Performance optimization
- Background process management
- Dependency resolution

**Service.json Enhancements**:
- New configuration options
- Better validation and error reporting
- Template and inheritance support
- Environment-specific configurations

### 3. Shared Library Development

**Utility Categories**:
- **Configuration Management**: Enhanced JSON processing, validation
- **Network Utilities**: Better connectivity testing, proxy support
- **Security Features**: Enhanced secret management, encryption
- **Performance Tools**: Monitoring, profiling, optimization
- **Development Tools**: Debugging, testing, validation

### 4. System Integration

**Resource Integration**:
- Better resource discovery and management
- Enhanced injection system
- Improved health checking
- Resource dependency resolution

**Scenario Integration**:
- Enhanced scenario validation
- Better testing framework integration
- Improved deployment automation
- Performance monitoring

## Testing and Validation

### 1. Unit Testing
```bash
# Create test files for utilities
cat > scripts/lib/utils/your-utility.bats << 'EOF'
#!/usr/bin/env bats

# Source the utility
source "${BASH_SOURCE%/*}/your-utility.sh"

@test "your_utility::function_name with valid input" {
    run your_utility::function_name "test-input"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Processing: test-input" ]]
}

@test "your_utility::function_name with empty input" {
    run your_utility::function_name ""
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Input required" ]]
}
EOF
```

### 2. Integration Testing
```bash
# Test CLI integration
./cli/vrooli your-command --dry-run

# Test lifecycle integration
./scripts/manage.sh your-phase --dry-run

# Test with actual scenarios
./scripts/scenarios/framework/scenario-test-runner.sh test-scenario
```

### 3. System Testing
```bash
# Test complete system lifecycle
./scripts/manage.sh setup --target native-linux --dry-run
./scripts/manage.sh develop --dry-run
./scripts/manage.sh test --dry-run
```

## Performance and Reliability

### 1. Performance Optimization
- **Startup Time**: Optimize CLI and lifecycle startup
- **Memory Usage**: Efficient resource management
- **Execution Speed**: Optimize critical paths
- **Caching**: Implement appropriate caching strategies

### 2. Error Handling
```bash
# Pattern: validate → trap cleanup → process → return
your_function() {
    [[ -n "$1" ]] || return 1
    trap 'cleanup' EXIT
    process_data "$1" || return 1
}
```

### 3. Logging and Monitoring
```bash
# Use log::debug, log::info, log::warning, log::error, log::success
# Performance: time_start=$(date +%s.%N); ...; duration=$(($(date +%s.%N) - time_start))
```

## Security and Configuration

### Security and Configuration

```bash
# Secrets: source secrets.sh; secrets::get "name"
# Config: source json.sh; json::get_value ".path" "default"
# Environment: [[ -n "${CI:-}" ]] && env="ci"
```
**Reference implementations**: `/scripts/lib/service/` and `/scripts/lib/utils/`

## Success Criteria

### ✅ Implementation Quality
- [ ] Code follows established patterns and conventions
- [ ] Comprehensive error handling implemented
- [ ] Performance optimized for common use cases
- [ ] Security best practices followed
- [ ] Comprehensive logging and monitoring

### ✅ Integration Testing
- [ ] Works with existing CLI commands
- [ ] Integrates properly with lifecycle system
- [ ] Compatible with resource management
- [ ] Works across different environments
- [ ] Passes all existing tests

### ✅ Documentation and Usability
- [ ] Clear documentation provided
- [ ] Help system updated
- [ ] Examples and usage patterns documented
- [ ] Error messages are helpful and actionable
- [ ] Backward compatibility maintained

### ✅ System Impact
- [ ] Improves overall platform reliability
- [ ] Enhances developer experience
- [ ] Provides measurable value
- [ ] Supports future extensibility
- [ ] No negative performance impact

## Common Patterns and Best Practices

### 1. Function Naming
```bash
# Use namespace prefixes for all functions
utility_name::function_name() { ... }
command_name::subcommand() { ... }
```

### 2. Error Handling
```bash
# Always provide context in errors
command_failed() {
    log::error "Failed to execute command: $1"
    log::error "Working directory: $(pwd)"
    log::error "Environment: $(detect_environment)"
    return 1
}
```

### 3. Resource Cleanup
```bash
# Always clean up resources
cleanup() {
    [[ -n "${temp_dir:-}" ]] && rm -rf "$temp_dir"
    [[ -n "${background_pid:-}" ]] && kill "$background_pid" 2>/dev/null || true
}
trap cleanup EXIT
```

### 4. Configuration Management
```bash
# Use centralized configuration access
get_config() {
    local key="$1"
    local default="$2"
    json::get_value "$key" "$default" 2>/dev/null || echo "$default"
}
```

## File Locations Reference

- **Main CLI**: `cli/vrooli`
- **Lifecycle Management**: `scripts/manage.sh`
- **Shared Libraries**: `scripts/lib/`
- **Configuration Schema**: `.vrooli/schemas/service.schema.json`
- **Testing Framework**: `scripts/__test/`

## Advanced Development

### 1. Plugin Architecture
Design features that can be extended:
- Command plugin system
- Lifecycle hook system
- Configuration extension points
- Resource adapter framework

### 2. Multi-Environment Support
Ensure features work across environments:
- Local development
- Docker containers
- Kubernetes clusters
- CI/CD systems

### 3. Performance Monitoring
Add observability to new features:
- Execution time tracking
- Resource usage monitoring
- Error rate tracking
- User experience metrics

Remember: Core features impact everything in Vrooli. Build with exceptional quality, comprehensive testing, and future extensibility in mind.