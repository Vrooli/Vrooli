# CLI Integration

## Purpose
CLI integration provides consistent, scriptable interfaces to resources. Every resource must be accessible via command-line for automation, testing, and manual operation.

## CLI Standards for Resources

### Required Commands
Every resource MUST implement these commands:
```bash
resource-[name] setup      # One-time setup
resource-[name] start      # Start the resource (alias: develop)
resource-[name] stop       # Stop the resource
resource-[name] health     # Check health status
resource-[name] test       # Run tests
resource-[name] help       # Show usage information
```

### Optional but Recommended
```bash
resource-[name] status     # Detailed status information
resource-[name] logs       # View logs
resource-[name] config     # Show/edit configuration
resource-[name] content    # Manage content (add/remove/list)
resource-[name] port       # Show port allocation
resource-[name] version    # Show version information
```

## CLI Implementation Pattern

### Basic Structure
```bash
#!/bin/bash

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_NAME="my-resource"

# Source library functions
source "$SCRIPT_DIR/lib/core.sh"
source "$SCRIPT_DIR/lib/health.sh"
source "$SCRIPT_DIR/lib/lifecycle.sh"

# Main command handler
case "$1" in
    setup)
        shift
        handle_setup "$@"
        ;;
    start|develop)
        shift
        handle_start "$@"
        ;;
    stop)
        shift
        handle_stop "$@"
        ;;
    health)
        shift
        handle_health "$@"
        ;;
    test)
        shift
        handle_test "$@"
        ;;
    status)
        shift
        handle_status "$@"
        ;;
    logs)
        shift
        handle_logs "$@"
        ;;
    content)
        shift
        handle_content "$@"
        ;;
    help|--help|-h|"")
        show_help
        ;;
    *)
        echo "Unknown command: $1"
        show_help
        exit 1
        ;;
esac
```

### Help Function
```bash
show_help() {
    cat <<EOF
Usage: $0 <command> [options]

Commands:
  setup              One-time setup and initialization
  start, develop     Start the resource
  stop               Stop the resource
  health             Check health status
  test               Run tests
  status             Show detailed status
  logs [options]     View logs (-f to follow)
  content <action>   Manage content (add/remove/list)
  help               Show this help message

Options:
  --verbose, -v      Verbose output
  --quiet, -q        Quiet mode (minimal output)
  --json             JSON output format
  --force            Force operation

Examples:
  $0 setup                    # Initial setup
  $0 start                    # Start the resource
  $0 health                   # Check if healthy
  $0 logs -f                  # Follow logs
  $0 content add workflow.json  # Add content
  $0 stop                     # Stop the resource

Environment Variables:
  ${RESOURCE_NAME^^}_PORT     Port to use (default: $DEFAULT_PORT)
  ${RESOURCE_NAME^^}_DATA     Data directory (default: $DATA_DIR)
  ${RESOURCE_NAME^^}_CONFIG   Config file (default: $CONFIG_FILE)

EOF
}
```

## Command Implementations

### Setup Command
```bash
handle_setup() {
    echo "Setting up ${RESOURCE_NAME}..."
    
    # Parse options
    local force=false
    while [[ $# -gt 0 ]]; do
        case $1 in
            --force) force=true; shift ;;
            *) echo "Unknown option: $1"; return 1 ;;
        esac
    done
    
    # Check if already setup
    if [ -f "$DATA_DIR/.setup" ] && [ "$force" != "true" ]; then
        echo "Already setup. Use --force to re-setup."
        return 0
    fi
    
    # Create directories
    mkdir -p "$DATA_DIR"
    mkdir -p "$LOG_DIR"
    mkdir -p "$CONFIG_DIR"
    
    # Initialize configuration
    create_default_config
    
    # Setup database if needed
    setup_database
    
    # Mark as setup
    touch "$DATA_DIR/.setup"
    
    echo "✅ Setup complete"
}
```

### Start Command
```bash
handle_start() {
    echo "Starting ${RESOURCE_NAME}..."
    
    # Check if already running
    if is_running; then
        echo "Already running (PID: $(get_pid))"
        return 0
    fi
    
    # Ensure setup was done
    if [ ! -f "$DATA_DIR/.setup" ]; then
        echo "Not setup yet. Running setup first..."
        handle_setup || return 1
    fi
    
    # Start the service
    start_service
    
    # Wait for ready
    if wait_for_ready; then
        echo "✅ ${RESOURCE_NAME} started successfully"
        echo "   Port: $(get_port)"
        echo "   PID: $(get_pid)"
        echo "   Logs: $LOG_FILE"
    else
        echo "❌ Failed to start"
        return 1
    fi
}
```

### Content Management
```bash
handle_content() {
    local action=$1
    shift
    
    case "$action" in
        add)
            handle_content_add "$@"
            ;;
        remove|rm|delete|del)
            handle_content_remove "$@"
            ;;
        list|ls)
            handle_content_list "$@"
            ;;
        *)
            echo "Usage: $0 content {add|remove|list} [options]"
            return 1
            ;;
    esac
}

handle_content_add() {
    local file=$1
    
    if [ -z "$file" ]; then
        echo "Usage: $0 content add <file>"
        return 1
    fi
    
    if [ ! -f "$file" ]; then
        echo "File not found: $file"
        return 1
    fi
    
    # Determine content type
    local type=$(determine_content_type "$file")
    
    # Validate content
    if ! validate_content "$file" "$type"; then
        echo "Invalid content format"
        return 1
    fi
    
    # Add to resource
    cp "$file" "$DATA_DIR/content/"
    
    # Reload if running
    if is_running; then
        reload_content
    fi
    
    echo "✅ Added $type: $(basename "$file")"
}

handle_content_list() {
    local format="${1:---table}"
    
    if [ "$format" = "--json" ]; then
        list_content_json
    else
        echo "Content in ${RESOURCE_NAME}:"
        echo "------------------------"
        ls -la "$DATA_DIR/content/" 2>/dev/null | grep -v "^total\|^d" | awk '{print $NF}' | while read -r file; do
            [ -n "$file" ] && [ "$file" != "." ] && [ "$file" != ".." ] && echo "  - $file"
        done
    fi
}
```

### Status Command
```bash
handle_status() {
    local format="${1:---human}"
    
    # Gather status information
    local running=$(is_running && echo "true" || echo "false")
    local pid=$(get_pid)
    local port=$(get_port)
    local health=$(check_health >/dev/null 2>&1 && echo "healthy" || echo "unhealthy")
    local uptime=$(get_uptime)
    local memory=$(get_memory_usage)
    local cpu=$(get_cpu_usage)
    
    if [ "$format" = "--json" ]; then
        cat <<EOF
{
  "resource": "${RESOURCE_NAME}",
  "running": ${running},
  "pid": ${pid:-null},
  "port": ${port:-null},
  "health": "${health}",
  "uptime": ${uptime:-null},
  "memory_mb": ${memory:-null},
  "cpu_percent": ${cpu:-null}
}
EOF
    else
        echo "${RESOURCE_NAME} Status"
        echo "==================="
        echo "Running:  ${running}"
        [ -n "$pid" ] && echo "PID:      ${pid}"
        [ -n "$port" ] && echo "Port:     ${port}"
        echo "Health:   ${health}"
        [ -n "$uptime" ] && echo "Uptime:   ${uptime}"
        [ -n "$memory" ] && echo "Memory:   ${memory} MB"
        [ -n "$cpu" ] && echo "CPU:      ${cpu}%"
    fi
}
```

## Output Formatting

### Verbose Mode
```bash
# Global verbose flag
VERBOSE=${VERBOSE:-false}

verbose() {
    if [ "$VERBOSE" = "true" ]; then
        echo "[VERBOSE] $*" >&2
    fi
}

# Parse global options
while [[ $# -gt 0 ]] && [[ "$1" =~ ^- ]]; do
    case $1 in
        --verbose|-v) VERBOSE=true; shift ;;
        --quiet|-q) QUIET=true; shift ;;
        --json) JSON_OUTPUT=true; shift ;;
        --) shift; break ;;
        *) break ;;
    esac
done
```

### JSON Output
```bash
# JSON output helper
output_json() {
    local status=$1
    local message=$2
    local data=$3
    
    if [ -n "$data" ]; then
        echo "{\"status\":\"$status\",\"message\":\"$message\",\"data\":$data}"
    else
        echo "{\"status\":\"$status\",\"message\":\"$message\"}"
    fi
}

# Use in commands
if [ "$JSON_OUTPUT" = "true" ]; then
    output_json "success" "Resource started" "{\"pid\":$pid,\"port\":$port}"
else
    echo "✅ Resource started (PID: $pid, Port: $port)"
fi
```

## Error Handling

### Consistent Error Messages
```bash
error() {
    echo "ERROR: $*" >&2
}

warning() {
    echo "WARNING: $*" >&2
}

info() {
    echo "INFO: $*"
}

# Exit codes
EXIT_SUCCESS=0
EXIT_GENERAL_ERROR=1
EXIT_MISUSE=2
EXIT_CANT_EXECUTE=126
EXIT_NOT_FOUND=127

# Use in functions
handle_invalid_command() {
    error "Invalid command: $1"
    show_help
    exit $EXIT_MISUSE
}
```

## Integration with Vrooli CLI

### Registration Pattern
```bash
# In install.sh
register_with_vrooli() {
    local cli_path="$SCRIPT_DIR/cli.sh"
    local symlink="/usr/local/bin/resource-${RESOURCE_NAME}"
    
    # Create symlink
    sudo ln -sf "$cli_path" "$symlink"
    
    # Register with Vrooli
    vrooli resource register "${RESOURCE_NAME}" "$cli_path"
    
    echo "✅ Registered as 'resource-${RESOURCE_NAME}'"
    echo "   Also available via: vrooli resource ${RESOURCE_NAME}"
}
```

### Subcommand Pattern
```bash
# When called via vrooli resource [name] [command]
if [ "$0" = "vrooli" ]; then
    RESOURCE_NAME=$1
    shift
    COMMAND=$1
    shift
else
    # Direct invocation
    COMMAND=$1
    shift
fi

case "$COMMAND" in
    # ... command handling ...
esac
```

## Testing CLI Commands

### BATS Tests
```bash
#!/usr/bin/env bats

@test "CLI shows help" {
    run ./cli.sh help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Usage:" ]]
}

@test "CLI starts resource" {
    run ./cli.sh start
    [ "$status" -eq 0 ]
    [[ "$output" =~ "started successfully" ]]
}

@test "CLI checks health" {
    ./cli.sh start
    run ./cli.sh health
    [ "$status" -eq 0 ]
    [[ "$output" =~ "healthy" ]]
    ./cli.sh stop
}

@test "CLI handles invalid command" {
    run ./cli.sh invalid-command
    [ "$status" -ne 0 ]
    [[ "$output" =~ "Unknown command" ]]
}

@test "CLI supports JSON output" {
    ./cli.sh start
    run ./cli.sh status --json
    [ "$status" -eq 0 ]
    echo "$output" | jq .resource
    ./cli.sh stop
}
```

## Best Practices

### DO's
✅ **Consistent command names** across all resources
✅ **Clear help text** with examples
✅ **Meaningful exit codes** for scripting
✅ **Support for common flags** (--verbose, --json)
✅ **Idempotent operations** where possible
✅ **Fast response** for status checks

### DON'Ts
❌ **Don't require sudo** unless absolutely necessary
❌ **Don't hide errors** - report clearly
❌ **Don't block indefinitely** - use timeouts
❌ **Don't assume environment** - check dependencies
❌ **Don't break backward compatibility** without notice

## Common CLI Issues

### Issue: Command Not Found
```bash
# Solution: Ensure proper PATH
export PATH="$PATH:/usr/local/bin"
# Or use full path
/path/to/resource-name command
```

### Issue: Permission Denied
```bash
# Solution: Make executable
chmod +x cli.sh
# Or check file ownership
ls -l cli.sh
```

### Issue: Environment Not Set
```bash
# Solution: Source environment
source /etc/vrooli/env
# Or set inline
RESOURCE_PORT=8080 ./cli.sh start
```

## Remember

**Consistency is key** - Same commands across all resources

**Scriptability matters** - Exit codes and parseable output

**Help should help** - Clear documentation and examples

**Fast feedback** - Quick response to commands

**Graceful handling** - Errors should be informative

Good CLI integration makes resources easy to use, automate, and debug. It's the primary interface for both humans and scripts.