# Resource Function Signature Standard

This document defines the standardized function signatures and patterns that all resource manage.sh files must follow for consistency, maintainability, and interoperability.

## Core Function Requirements

### 1. Required Functions (All Resources)

All resources MUST implement these functions:

```bash
# Main entry point - must handle all argument parsing and routing
{resource}::main()

# Argument parsing - must use consistent args::* pattern  
{resource}::parse_arguments()

# Usage/help display - must follow standard format
{resource}::usage()

# Core lifecycle functions
{resource}::install()
{resource}::uninstall()
{resource}::start()
{resource}::stop()
{resource}::restart()
{resource}::status()

# Information functions
{resource}::logs()
{resource}::info()

# Testing function
{resource}::test()
```

### 2. Standard Function Signatures

#### Core Functions
```bash
# Main entry point
{resource}::main() {
    {resource}::parse_arguments "$@"
    # Route to appropriate action based on parsed arguments
}

# Argument parsing (must use args:: utilities)
{resource}::parse_arguments() {
    args::reset
    args::register_help
    args::register_yes
    
    # Standard action parameter
    args::register \
        --name "action" \
        --flag "a" \
        --desc "Action to perform" \
        --type "value" \
        --options "install|uninstall|start|stop|restart|status|logs|info|test|monitor|diagnose" \
        --default "install"
    
    # Standard parameters
    args::register \
        --name "force" \
        --flag "f" \
        --desc "Force action even if already performed" \
        --type "value" \
        --options "yes|no" \
        --default "no"
        
    args::register \
        --name "lines" \
        --flag "n" \
        --desc "Number of log lines to show" \
        --type "value" \
        --default "50"
    
    # Parse and export
    args::parse "$@"
    export ACTION=$(args::get "action")
    export FORCE=$(args::get "force")
    export LINES=$(args::get "lines")
}

# Usage display (must follow standard format)
{resource}::usage() {
    args::usage "$DESCRIPTION"
    echo
    echo "Standard Actions:"
    echo "  install    - Install and configure the resource"
    echo "  uninstall  - Remove the resource completely"
    echo "  start      - Start the resource service"
    echo "  stop       - Stop the resource service"
    echo "  restart    - Restart the resource service"
    echo "  status     - Show resource status"
    echo "  logs       - Show resource logs"
    echo "  info       - Show resource information"
    echo "  test       - Test resource functionality"
}
```

#### Lifecycle Functions
```bash
# Installation - must be idempotent
{resource}::install() {
    # Return 0 for success, 1 for error, 2 for already installed (when force=no)
}

# Uninstallation - must be safe
{resource}::uninstall() {  
    # Return 0 for success, 1 for error, 2 for not installed
}

# Start service - must be idempotent
{resource}::start() {
    # Return 0 for success, 1 for error, 2 for already running
}

# Stop service - must be safe
{resource}::stop() {
    # Return 0 for success, 1 for error, 2 for not running
}

# Restart service
{resource}::restart() {
    {resource}::stop
    {resource}::start
}

# Status check - must be informative
{resource}::status() {
    # Return 0 if healthy, 1 if unhealthy, 2 if not installed
    # Must print status information
}
```

#### Information Functions
```bash
# Show logs
{resource}::logs() {
    # Show last $LINES lines of logs
    # Return 0 for success, 1 for error
}

# Show information
{resource}::info() {
    # Show resource configuration and details
    # Return 0 for success, 1 for error
}

# Test functionality
{resource}::test() {
    # Run basic functionality tests
    # Return 0 for pass, 1 for fail, 2 for skip
}
```

### 3. Standard Return Codes

All functions must use these standardized return codes:

```bash
# Success codes
0   # Success/OK
2   # Success but no action needed (already installed, not running, etc.)

# Error codes  
1   # General error
3   # Permission/authentication error
4   # Service unavailable/timeout error
5   # Configuration error
6   # Network error
```

### 4. Category-Specific Extensions

Resources MAY implement additional category-specific functions:

#### AI Resources
```bash
{resource}::models()      # List/manage models
{resource}::process()     # Process input data
{resource}::validate()    # Validate model integrity
```

#### Automation Resources  
```bash
{resource}::workflows()   # List/manage workflows
{resource}::execute()     # Execute workflow
{resource}::agents()      # Manage agents (if applicable)
```

#### Storage Resources
```bash
{resource}::create()      # Create database/bucket
{resource}::destroy()     # Destroy database/bucket  
{resource}::connect()     # Test connection
{resource}::migrate()     # Run migrations
{resource}::backup()      # Create backup
{resource}::restore()     # Restore from backup
```

#### Agent Resources
```bash
{resource}::sessions()    # Manage sessions
{resource}::templates()   # Manage templates
{resource}::health_check() # Detailed health check
```

#### Search Resources
```bash
{resource}::search()      # Perform search
{resource}::index()       # Manage search index
{resource}::benchmark()   # Performance testing
```

#### Execution Resources
```bash
{resource}::submit()      # Submit code for execution
{resource}::languages()   # List supported languages
{resource}::monitor()     # Monitor execution queues
```

### 5. Module Loading Pattern

All resources must follow this module loading pattern:

```bash
#!/usr/bin/env bash
set -euo pipefail

# Resource description
DESCRIPTION="Description of the resource"

# Directory setup
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
RESOURCES_DIR="${SCRIPT_DIR}/../.."

# Signal handling
trap 'echo ""; log::info "{resource} operation interrupted by user. Exiting..."; exit 130' INT TERM

# Source common resources (required)
source "${RESOURCES_DIR}/common.sh"
source "${RESOURCES_DIR}/../helpers/utils/args.sh"

# Source configuration modules (required)
source "${SCRIPT_DIR}/config/defaults.sh"
source "${SCRIPT_DIR}/config/messages.sh"

# Export configuration
{resource}::export_config
{resource}::export_messages

# Source library modules (as needed)
source "${SCRIPT_DIR}/lib/common.sh"
source "${SCRIPT_DIR}/lib/install.sh"
source "${SCRIPT_DIR}/lib/status.sh"
# ... other lib modules as needed
```

### 6. Error Handling Standards

```bash
# Standard error handling pattern
{resource}::function_name() {
    # Validate prerequisites
    if ! {resource}::validate_prerequisites; then
        log::error "Prerequisites not met"
        return 1
    fi
    
    # Perform operation with error checking
    if ! some_operation; then
        log::error "Operation failed: specific error message"
        return 1
    fi
    
    # Success
    log::success "Operation completed successfully"
    return 0
}
```

### 7. Configuration Standards

All resources must support these configuration patterns:

```bash
# In config/defaults.sh
{resource}::export_config() {
    export {RESOURCE}_PORT="default_port"
    export {RESOURCE}_VERSION="default_version"
    export {RESOURCE}_DATA_DIR="default_data_dir"
    # ... other defaults
}

# In config/messages.sh  
{resource}::export_messages() {
    export {RESOURCE}_MSG_INSTALL="Installing {resource}..."
    export {RESOURCE}_MSG_SUCCESS="✅ {resource} operation completed"
    export {RESOURCE}_MSG_ERROR="❌ {resource} operation failed"
    # ... other messages
}
```

## Implementation Guidelines

### Migration Strategy
1. **Phase 1**: Update core function signatures (install, start, stop, status)
2. **Phase 2**: Standardize argument parsing and error handling  
3. **Phase 3**: Add missing required functions
4. **Phase 4**: Standardize category-specific functions

### Testing Requirements
- All standardized functions must be tested
- Integration tests must validate standard function behavior
- Error conditions must be tested for proper return codes

### Documentation Requirements
- All functions must have proper documentation comments
- Usage examples must be provided for complex functions
- Return codes must be documented

This standard ensures consistency across all resources while allowing flexibility for category-specific needs.