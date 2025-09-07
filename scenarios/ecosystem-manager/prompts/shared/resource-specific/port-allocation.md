# Port Allocation

## Purpose
Proper port allocation prevents conflicts and ensures resources can run simultaneously. Every resource needs predictable, non-conflicting ports.

## Port Allocation Ranges

### Standard Ranges
```markdown
## Vrooli Port Allocation Strategy

20000-24999: Core Infrastructure
- 20000-20999: Databases (PostgreSQL, MongoDB, etc.)
- 21000-21999: Cache/Queue (Redis, RabbitMQ, etc.)
- 22000-22999: Search/Analytics (Elasticsearch, QuestDB, etc.)
- 23000-23999: Storage (MinIO, SeaweedFS, etc.)
- 24000-24999: Reserved for future core

25000-29999: Supporting Services
- 25000-25999: Monitoring (Prometheus, Grafana, etc.)
- 26000-26999: Security (Vault, Keycloak, etc.)
- 27000-27999: CI/CD (Jenkins, GitLab, etc.)
- 28000-28999: Documentation (Wiki, Docs, etc.)
- 29000-29999: Reserved for future support

30000-34999: Application Scenarios
- 30000-30999: Business scenarios
- 31000-31999: Productivity scenarios
- 32000-32999: AI/ML scenarios
- 33000-33999: Communication scenarios
- 34000-34999: Reserved for future scenarios

35000-39999: Development Tools
- 35000-35999: IDEs and editors
- 36000-36999: Debuggers and profilers
- 37000-37999: Build tools
- 38000-38999: Testing tools
- 39000-39999: Reserved for future tools

40000-49999: Dynamic Allocation Pool
- Used for temporary services
- Ephemeral containers
- Testing instances
- User-created services
```

## Port Allocation Process

### 1. Check Registry
```bash
# Check if port is already allocated
check_port_allocation() {
    local port=$1
    local registry="/scripts/resources/port-registry.sh"
    
    # Check registry
    allocated=$($registry check $port)
    if [ "$allocated" != "available" ]; then
        echo "Port $port already allocated to: $allocated"
        return 1
    fi
    
    # Check if port is actually in use
    if nc -z localhost $port 2>/dev/null; then
        echo "Port $port is in use but not in registry!"
        return 1
    fi
    
    return 0
}
```

### 2. Allocate Port
```bash
# Allocate port for resource
allocate_port() {
    local resource=$1
    local preferred_port=$2
    local range_start=$3
    local range_end=$4
    
    # Try preferred port first
    if [ -n "$preferred_port" ]; then
        if check_port_allocation $preferred_port; then
            register_port $resource $preferred_port
            echo $preferred_port
            return 0
        fi
    fi
    
    # Find next available in range
    for port in $(seq $range_start $range_end); do
        if check_port_allocation $port; then
            register_port $resource $port
            echo $port
            return 0
        fi
    done
    
    echo "No available ports in range $range_start-$range_end"
    return 1
}
```

### 3. Register Allocation
```bash
# Register port in registry
register_port() {
    local resource=$1
    local port=$2
    local registry="/scripts/resources/port-registry.sh"
    
    # Add to registry
    $registry register $resource $port
    
    # Update service.json
    update_service_json $resource $port
    
    # Document in README
    update_readme_port $resource $port
    
    echo "Port $port registered for $resource"
}
```

## Configuration in service.json

### Single Port Resource
```json
{
  "name": "my-resource",
  "ports": {
    "main": 25000
  }
}
```

### Multi-Port Resource
```json
{
  "name": "complex-resource",
  "ports": {
    "api": 25000,
    "ui": 25001,
    "admin": 25002,
    "metrics": 25003
  }
}
```

### Dynamic Port Resource
```json
{
  "name": "dynamic-resource",
  "ports": {
    "main": "${DYNAMIC_PORT:-40000}"
  },
  "port_allocation": {
    "strategy": "dynamic",
    "range_start": 40000,
    "range_end": 49999
  }
}
```

## Environment Variable Configuration

### Standard Pattern
```bash
# In resource startup script
export RESOURCE_PORT=${RESOURCE_PORT:-25000}
export RESOURCE_API_PORT=${RESOURCE_API_PORT:-25001}
export RESOURCE_UI_PORT=${RESOURCE_UI_PORT:-25002}

# Use in configuration
config.port = process.env.RESOURCE_PORT || 25000;
```

### Docker Compose Pattern
```yaml
services:
  my-resource:
    ports:
      - "${RESOURCE_PORT:-25000}:8080"
    environment:
      - PORT=${RESOURCE_PORT:-25000}
```

## Conflict Resolution

### When Conflicts Occur
```bash
# Handle port conflicts
handle_port_conflict() {
    local resource=$1
    local conflicted_port=$2
    
    echo "Port conflict detected for $resource on port $conflicted_port"
    
    # Find alternative port
    local new_port=$(find_alternative_port $resource)
    
    if [ -n "$new_port" ]; then
        echo "Reassigning $resource to port $new_port"
        update_port_allocation $resource $new_port
    else
        echo "ERROR: No alternative ports available for $resource"
        return 1
    fi
}
```

### Prevention Strategies
1. **Always check registry before starting**
2. **Use consistent port ranges**
3. **Document all port usage**
4. **Update registry immediately**
5. **Release ports on shutdown**

## Port Management Commands

### CLI Integration
```bash
# Resource port commands
case "$1" in
    port)
        case "$2" in
            show)
                show_port_allocation $RESOURCE_NAME
                ;;
            check)
                check_port_availability $3
                ;;
            release)
                release_port $RESOURCE_NAME
                ;;
            *)
                echo "Usage: $0 port {show|check|release}"
                ;;
        esac
        ;;
esac
```

## Best Practices

### DO's
✅ **Use designated ranges** - Stay within allocated range
✅ **Check before binding** - Always verify availability
✅ **Register immediately** - Update registry on allocation
✅ **Document clearly** - Note ports in README and service.json
✅ **Release on shutdown** - Free ports when stopping

### DON'Ts
❌ **Don't hardcode without checking** - Always verify availability
❌ **Don't use random ports** - Stay within designated ranges
❌ **Don't forget to register** - Update registry immediately
❌ **Don't use well-known ports** - Avoid <10000
❌ **Don't skip conflict handling** - Always handle gracefully

## Common Port Allocation Issues

### Issue: Port Already in Use
```bash
# Solution: Find who's using it
lsof -i :$PORT
# Kill if orphaned
kill $(lsof -t -i :$PORT)
# Or find alternative port
```

### Issue: Registry Out of Sync
```bash
# Solution: Reconcile registry with reality
scripts/resources/port-registry.sh reconcile
```

### Issue: Dynamic Ports Exhausted
```bash
# Solution: Clean up unused allocations
scripts/resources/port-registry.sh cleanup-dynamic
```

## Remember

**Ports are limited resources** - Use them wisely

**Conflicts break everything** - Prevention is critical

**Registry is source of truth** - Keep it updated

**Ranges prevent chaos** - Respect boundaries

**Documentation prevents confusion** - Always document

Good port allocation is invisible when done right - everything just works without conflicts.