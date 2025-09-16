# Agent Framework Integration Template

This template ensures consistent integration of the agent framework across all resources.

## 1. Agent Configuration File Template

Create `/resources/<resource-name>/config/agents.conf`:

```bash
#!/usr/bin/env bash
################################################################################
# <Resource Name> Agent Configuration
# 
# Plugin configuration for the unified agent management system
################################################################################

# Resource identification
RESOURCE_NAME="<resource-name>"

# Registry configuration
REGISTRY_FILE="${APP_ROOT}/.vrooli/<resource-name>-agents.json"

# Logging configuration
LOG_DIR="${APP_ROOT}/.vrooli/logs/resources/<resource-name>"

# Agent identification
AGENT_ID_PREFIX="<resource-name>-agent"

# Log search patterns for parent process detection
SEARCH_PATTERNS="<resource-name>|agent|task"

# Health monitoring configuration (optional)
HEALTH_CHECK_ENABLED="true"         # Enable for long-running AI operations
HEALTH_CHECK_INTERVAL="30"          # Seconds between health checks
HEALTH_CHECK_TYPE="basic"           # basic|enhanced|deep

# Metrics collection configuration (optional)
METRICS_ENABLED="true"              # Enable metrics collection
METRICS_INTERVAL="60"               # Seconds between metric collections

# Monitor process configuration
MONITOR_PIDFILE="${APP_ROOT}/.vrooli/<resource-name>.monitor.pid"

# Resource-specific settings (optional overrides)
# CUSTOM_CLEANUP_HOOK=""
# CUSTOM_VALIDATION=""
# CUSTOM_POST_REGISTER=""
```

## 2. CLI Integration Checklist

### Required Integration Steps:

#### Step 1: Source Agent Framework
Add to your `cli.sh` after other sourcing:

```bash
# Source agent management (load config and manager directly)
if [[ -f "${APP_ROOT}/resources/<resource-name>/config/agents.conf" ]]; then
    source "${APP_ROOT}/resources/<resource-name>/config/agents.conf"
    source "${APP_ROOT}/scripts/resources/agents/agent-manager.sh"
fi
```

#### Step 2: Add Agent Management Command
```bash
# Agent management commands
# Create wrapper for agents command that delegates to manager
<resource_name>::agents::command() {
    if type -t agent_manager::load_config &>/dev/null; then
        "${APP_ROOT}/scripts/resources/agents/agent-manager.sh" --config="<resource-name>" "$@"
    else
        log::error "Agent management not available"
        return 1
    fi
}
export -f <resource_name>::agents::command

cli::register_command "agents" "Manage running <Resource Name> agents" "<resource_name>::agents::command"
```

#### Step 3: Add Cleanup Function
```bash
################################################################################
# Agent cleanup function
################################################################################

#######################################
# Setup agent cleanup on signals
# Arguments:
#   $1 - Agent ID
#######################################
<resource_name>::setup_agent_cleanup() {
    local agent_id="$1"
    
    # Export the agent ID so trap can access it
    export <RESOURCE_NAME>_CURRENT_AGENT_ID="$agent_id"
    
    # Cleanup function that uses the exported variable
    <resource_name>::agent_cleanup() {
        if [[ -n "${<RESOURCE_NAME>_CURRENT_AGENT_ID:-}" ]] && type -t agent_manager::unregister &>/dev/null; then
            agent_manager::unregister "${<RESOURCE_NAME>_CURRENT_AGENT_ID}" >/dev/null 2>&1
        fi
        exit 0
    }
    
    # Register cleanup for common signals
    trap '<resource_name>::agent_cleanup' EXIT SIGTERM SIGINT
}
```

#### Step 4: Integrate Metrics in Operation Functions

For each operation function that executes work:

```bash
<resource_name>_operation() {
    # ... argument parsing ...
    
    # Register agent if agent management is available
    local agent_id=""
    if type -t agent_manager::register &>/dev/null; then
        agent_id=$(agent_manager::generate_id)
        local command_string="resource-<resource-name> <operation> $*"
        if agent_manager::register "$agent_id" $$ "$command_string"; then
            log::debug "Registered agent: $agent_id"
            
            # Set up signal handler for cleanup
            <resource_name>::setup_agent_cleanup "$agent_id"
            
            # Metrics will be tracked during actual operation execution
        fi
    fi
    
    # ... validation logic ...
    
    # Execute operation with metrics tracking
    local result=0
    local start_time=$(date +%s%3N)  # Milliseconds
    
    # Track operation start metrics
    if [[ -n "$agent_id" ]] && type -t agents::metrics::increment &>/dev/null; then
        agents::metrics::increment "${REGISTRY_FILE:-${APP_ROOT}/.vrooli/<resource-name>-agents.json}" "$agent_id" "requests" 1
    fi
    
    # ACTUAL OPERATION EXECUTION HERE
    <actual_operation_command>
    result=$?
    
    # Track operation completion metrics
    if [[ -n "$agent_id" ]] && type -t agents::metrics::histogram &>/dev/null; then
        local end_time=$(date +%s%3N)
        local duration=$((end_time - start_time))
        agents::metrics::histogram "${REGISTRY_FILE:-${APP_ROOT}/.vrooli/<resource-name>-agents.json}" "$agent_id" "request_duration_ms" "$duration"
        
        # Track success/error
        if [[ $result -eq 0 ]]; then
            log::debug "<Resource Name> operation completed successfully"
        else
            type -t agents::metrics::increment &>/dev/null && \
                agents::metrics::increment "${REGISTRY_FILE:-${APP_ROOT}/.vrooli/<resource-name>-agents.json}" "$agent_id" "errors" 1
        fi
        
        # Update process metrics
        if type -t agents::metrics::gauge &>/dev/null; then
            # Get current process memory usage (in MB)
            local memory_kb=$(ps -o rss= -p $$ 2>/dev/null | awk '{print $1}' || echo "0")
            local memory_mb=$((memory_kb / 1024))
            agents::metrics::gauge "${REGISTRY_FILE:-${APP_ROOT}/.vrooli/<resource-name>-agents.json}" "$agent_id" "memory_mb" "$memory_mb"
        fi
    fi
    
    # Unregister agent on completion
    if [[ -n "$agent_id" ]] && type -t agent_manager::unregister &>/dev/null; then
        agent_manager::unregister "$agent_id" >/dev/null 2>&1
    fi
    
    return $result
}
```

## 3. Health Monitoring Guidelines

### When to Enable Health Monitoring:
- ✅ **AI inference operations** (LLM calls, image generation, etc.)
- ✅ **Long-running agents** (AutoGPT, CrewAI, AutoGen)
- ✅ **Proxy services** (LiteLLM, API gateways)
- ✅ **Complex processing** (data analysis, file processing)
- ❌ **Simple utilities** (format converters, validators)
- ❌ **Infrastructure services** (databases, message queues)

### Health Check Intervals:
- **30 seconds**: AI operations, agents (default)
- **60 seconds**: Proxy services, APIs
- **120 seconds**: Background processing

## 4. Metrics Collection Standards

### Required Metrics:
- **requests**: Count of operations started
- **errors**: Count of failed operations  
- **request_duration_ms**: Histogram of operation timing
- **memory_mb**: Current process memory usage

### Optional Metrics:
- **active_connections**: For services with connections
- **cpu_ticks**: CPU usage sampling
- **restarts**: Count of process restarts

## 5. Testing Integration

After integration, verify with:

```bash
# Test agent registration
resource-<resource-name> <operation> <test-args>

# Check agent registry
resource-<resource-name> agents list

# View metrics
resource-<resource-name> agents metrics <agent-id>

# Test monitoring dashboard
resource-<resource-name> agents monitor
```

## 6. Validation Checklist

- [ ] Agent config file created with proper naming
- [ ] CLI sources agent framework correctly
- [ ] Agent command wrapper implemented
- [ ] Cleanup function added with proper signal handling
- [ ] Metrics tracking added to all operation functions
- [ ] Health monitoring enabled for appropriate resource types
- [ ] Registry file path uses resource name
- [ ] Search patterns include resource-specific terms
- [ ] Integration tested with actual operations

## 7. Common Pitfalls

- ❌ **Don't track metrics at registration** - track during actual operations
- ❌ **Don't forget cleanup handlers** - processes will leak without them
- ❌ **Don't enable health monitoring for simple utilities** - creates unnecessary overhead
- ❌ **Don't use hardcoded registry paths** - use REGISTRY_FILE variable
- ❌ **Don't skip error tracking** - failed operations need metrics too

## 8. Support

For questions or issues with agent framework integration:
1. Check existing implementations in `ollama` and `claude-code` resources
2. Review the agent framework documentation in `/scripts/resources/agents/`
3. Test with the monitoring dashboard to verify metrics are working