# Collision Avoidance

## Purpose
Multiple agents working simultaneously can conflict. Collision avoidance ensures agents work harmoniously, maximizing throughput while preventing conflicts.

## Types of Collisions

### Resource Collisions
```markdown
## Resource Collision Types

1. **File Collisions**
   - Two agents editing same file
   - Conflicting changes to shared config
   - Simultaneous port allocations

2. **Database Collisions**
   - Concurrent schema changes
   - Conflicting data updates
   - Lock contention

3. **Port Collisions**
   - Multiple services claiming same port
   - Dynamic allocation conflicts
   - Port range exhaustion

4. **Queue Collisions**
   - Multiple agents taking same item
   - Race conditions in selection
   - Stale lock files

5. **API Collisions**
   - Rate limit exhaustion
   - Conflicting webhook registrations
   - State synchronization issues
```

## Collision Prevention Mechanisms

### 1. File-Level Locking
```bash
# Acquire file lock before editing
acquire_file_lock() {
    local file=$1
    local lock_file="${file}.lock"
    local agent_id=${AGENT_ID:-$USER}
    local timeout=300  # 5 minutes
    
    # Try to acquire lock
    local start_time=$(date +%s)
    while [ -f "$lock_file" ]; do
        local lock_age=$(($(date +%s) - $(stat -c %Y "$lock_file" 2>/dev/null || echo 0)))
        
        # Break stale locks (older than timeout)
        if [ $lock_age -gt $timeout ]; then
            echo "Breaking stale lock on $file"
            rm -f "$lock_file"
            break
        fi
        
        # Check timeout
        if [ $(($(date +%s) - start_time)) -gt $timeout ]; then
            echo "Failed to acquire lock on $file"
            return 1
        fi
        
        sleep 1
    done
    
    # Create lock
    echo "$agent_id:$(date +%s)" > "$lock_file"
    return 0
}

# Release file lock
release_file_lock() {
    local file=$1
    local lock_file="${file}.lock"
    rm -f "$lock_file"
}

# Use locks in practice
edit_with_lock() {
    local file=$1
    
    if acquire_file_lock "$file"; then
        # Do the edit
        edit_file "$file"
        release_file_lock "$file"
    else
        echo "Could not acquire lock, skipping edit"
        return 1
    fi
}
```

### 2. Queue Item Reservation
```python
def reserve_queue_item(agent_id):
    """
    Atomically reserve a queue item for processing
    """
    import fcntl
    import yaml
    from pathlib import Path
    
    pending_dir = Path("queue/pending")
    in_progress_dir = Path("queue/in-progress")
    
    # Get sorted list of pending items
    pending_items = sorted(pending_dir.glob("*.yaml"))
    
    for item_path in pending_items:
        lock_path = item_path.with_suffix('.yaml.lock')
        
        try:
            # Try to acquire exclusive lock
            with open(lock_path, 'w') as lock_file:
                fcntl.flock(lock_file.fileno(), fcntl.LOCK_EX | fcntl.LOCK_NB)
                
                # We got the lock, check if item still exists
                if item_path.exists():
                    # Load item
                    with open(item_path) as f:
                        item = yaml.safe_load(f)
                    
                    # Check cooldown
                    if is_cooldown_expired(item):
                        # Reserve it
                        item['metadata']['assigned_to'] = agent_id
                        item['metadata']['reserved_at'] = datetime.now().isoformat()
                        
                        # Move to in-progress
                        new_path = in_progress_dir / item_path.name
                        item_path.rename(new_path)
                        
                        # Save updated item
                        with open(new_path, 'w') as f:
                            yaml.dump(item, f)
                        
                        return item
                
                # Release lock if we didn't take the item
                fcntl.flock(lock_file.fileno(), fcntl.LOCK_UN)
                
        except IOError:
            # Lock is held by another process, try next item
            continue
    
    return None  # No items available
```

### 3. Port Allocation Registry
```bash
# Central port registry
PORT_REGISTRY="/tmp/vrooli-port-registry.json"

# Allocate port atomically
allocate_port() {
    local service=$1
    local preferred_port=$2
    local port_range_start=${3:-40000}
    local port_range_end=${4:-49999}
    
    # Lock registry for atomic operation
    (
        flock -x 200
        
        # Read current allocations
        if [ -f "$PORT_REGISTRY" ]; then
            allocations=$(cat "$PORT_REGISTRY")
        else
            allocations="{}"
        fi
        
        # Check if preferred port is available
        if [ -n "$preferred_port" ]; then
            allocated=$(echo "$allocations" | jq -r ".[\"$preferred_port\"]")
            if [ "$allocated" == "null" ]; then
                # Allocate preferred port
                echo "$allocations" | jq ".[\"$preferred_port\"] = \"$service\"" > "$PORT_REGISTRY"
                echo "$preferred_port"
                return 0
            fi
        fi
        
        # Find next available port
        for port in $(seq $port_range_start $port_range_end); do
            allocated=$(echo "$allocations" | jq -r ".[\"$port\"]")
            if [ "$allocated" == "null" ] && ! nc -z localhost $port 2>/dev/null; then
                # Allocate this port
                echo "$allocations" | jq ".[\"$port\"] = \"$service\"" > "$PORT_REGISTRY"
                echo "$port"
                return 0
            fi
        done
        
        return 1  # No ports available
    ) 200>"${PORT_REGISTRY}.lock"
}

# Release port
release_port() {
    local port=$1
    
    (
        flock -x 200
        
        if [ -f "$PORT_REGISTRY" ]; then
            cat "$PORT_REGISTRY" | jq "del(.[\"$port\"])" > "$PORT_REGISTRY"
        fi
    ) 200>"${PORT_REGISTRY}.lock"
}
```

### 4. Cooldown Periods
```python
def calculate_cooldown(max_workers=3, iteration_hours=4):
    """
    Calculate cooldown period to prevent collisions
    """
    import random
    from datetime import datetime, timedelta
    
    # Base cooldown: time for all workers to complete
    base_cooldown_hours = max_workers * iteration_hours
    
    # Add jitter to prevent thundering herd
    jitter_minutes = random.randint(0, 60)
    
    cooldown_delta = timedelta(
        hours=base_cooldown_hours,
        minutes=jitter_minutes
    )
    
    cooldown_until = datetime.now() + cooldown_delta
    
    return cooldown_until.isoformat()

def respect_cooldown(item):
    """
    Check if item is available after cooldown
    """
    from datetime import datetime
    
    cooldown_until = item.get('metadata', {}).get('cooldown_until')
    
    if not cooldown_until:
        return True  # No cooldown set
    
    cooldown_time = datetime.fromisoformat(cooldown_until)
    now = datetime.now()
    
    return now > cooldown_time
```

## Coordination Strategies

### Strategy 1: Work Area Segregation
```python
def assign_work_areas(num_agents=3):
    """
    Assign different work areas to each agent
    """
    work_areas = {
        0: {
            'focus': 'scenarios',
            'priority': 'high-impact',
            'categories': ['business', 'productivity']
        },
        1: {
            'focus': 'resources',
            'priority': 'stability',
            'categories': ['infrastructure', 'storage']
        },
        2: {
            'focus': 'integration',
            'priority': 'cross-cutting',
            'categories': ['automation', 'monitoring']
        }
    }
    
    # Agents work primarily in their area
    # But can help others if their queue is empty
    return work_areas

def select_based_on_area(agent_id, available_items):
    """
    Select items based on assigned work area
    """
    areas = assign_work_areas()
    my_area = areas[agent_id % len(areas)]
    
    # Prioritize items in my focus area
    prioritized = []
    others = []
    
    for item in available_items:
        if item['type'] == my_area['focus']:
            prioritized.append(item)
        else:
            others.append(item)
    
    # Return prioritized items first, then others
    return prioritized + others
```

### Strategy 2: Time-Based Coordination
```bash
# Stagger agent start times
stagger_start() {
    local agent_id=$1
    local total_agents=$2
    local iteration_minutes=$((4 * 60))  # 4 hours
    
    # Calculate stagger offset
    local offset=$((iteration_minutes / total_agents * agent_id))
    
    echo "Agent $agent_id waiting $offset minutes to stagger start"
    sleep $(($offset * 60))
}

# Time-sliced work windows
get_work_window() {
    local agent_id=$1
    local hour=$(date +%H)
    
    # Assign 2-hour windows
    case $agent_id in
        0) echo "00-08" ;;
        1) echo "08-16" ;;
        2) echo "16-24" ;;
    esac
}
```

### Strategy 3: Dependency-Based Coordination
```python
def coordinate_by_dependencies():
    """
    Agents work on non-conflicting dependency chains
    """
    # Build dependency graph
    dep_graph = build_dependency_graph()
    
    # Find independent work streams
    independent_streams = find_independent_paths(dep_graph)
    
    # Assign streams to agents
    assignments = {}
    for i, stream in enumerate(independent_streams):
        agent_id = i % get_num_agents()
        if agent_id not in assignments:
            assignments[agent_id] = []
        assignments[agent_id].extend(stream)
    
    return assignments
```

## Conflict Resolution

### When Collisions Occur
```python
def resolve_collision(collision_type, agents_involved):
    """
    Resolve different types of collisions
    """
    if collision_type == 'file_edit':
        # Last write wins, but preserve both versions
        return merge_changes_safely(agents_involved)
    
    elif collision_type == 'queue_item':
        # First to move to in-progress wins
        return honor_first_mover(agents_involved)
    
    elif collision_type == 'port_allocation':
        # Registry determines owner
        return check_port_registry(agents_involved)
    
    elif collision_type == 'api_rate_limit':
        # Implement backoff
        return exponential_backoff(agents_involved)
    
    else:
        # Generic resolution: priority based on agent ID
        return prioritize_by_agent_id(agents_involved)
```

### Collision Recovery
```bash
# Detect and recover from collisions
detect_collisions() {
    # Check for multiple agents on same item
    for item in queue/in-progress/*.yaml; do
        if [ -f "$item" ]; then
            assigned_to=$(yq '.metadata.assigned_to' "$item")
            lock_owner=$(cat "${item}.lock" 2>/dev/null | cut -d: -f1)
            
            if [ "$assigned_to" != "$lock_owner" ]; then
                echo "Collision detected on $item"
                resolve_queue_collision "$item"
            fi
        fi
    done
    
    # Check for port conflicts
    for port in $(netstat -tlnp 2>/dev/null | grep LISTEN | awk '{print $4}' | cut -d: -f2); do
        registry_owner=$(jq -r ".[\"$port\"]" "$PORT_REGISTRY" 2>/dev/null)
        actual_owner=$(lsof -i :$port -t 2>/dev/null | xargs ps -p 2>/dev/null | tail -1 | awk '{print $NF}')
        
        if [ -n "$registry_owner" ] && [ "$registry_owner" != "$actual_owner" ]; then
            echo "Port collision on $port"
            resolve_port_collision "$port"
        fi
    done
}
```

## Monitoring and Metrics

### Collision Metrics
```python
def track_collision_metrics():
    """
    Monitor collision rates and patterns
    """
    metrics = {
        'file_collisions': count_file_lock_conflicts(),
        'queue_collisions': count_queue_conflicts(),
        'port_collisions': count_port_conflicts(),
        'total_collisions': 0,
        'collision_rate': 0,
        'affected_agents': set(),
        'hotspot_files': find_frequently_locked_files(),
        'hotspot_items': find_frequently_contested_items()
    }
    
    metrics['total_collisions'] = sum([
        metrics['file_collisions'],
        metrics['queue_collisions'],
        metrics['port_collisions']
    ])
    
    total_operations = count_total_operations()
    metrics['collision_rate'] = metrics['total_collisions'] / total_operations
    
    # Alert if collision rate is high
    if metrics['collision_rate'] > 0.1:  # >10% collision rate
        alert_high_collision_rate(metrics)
        adjust_cooldown_periods()
        increase_work_area_segregation()
    
    return metrics
```

## Best Practices

### DO's
✅ **Always use locks** for shared resource access
✅ **Respect cooldowns** to prevent thundering herd
✅ **Check before claiming** - verify resource availability
✅ **Release quickly** - hold locks minimum time
✅ **Fail gracefully** - handle lock timeouts properly
✅ **Monitor patterns** - track collision hotspots

### DON'Ts
❌ **Don't ignore locks** - always check lock files
❌ **Don't hold indefinitely** - set lock timeouts
❌ **Don't retry immediately** - use exponential backoff
❌ **Don't force break locks** - investigate first
❌ **Don't assume exclusivity** - always verify

## Remember for Collision Avoidance

**Prevention is better than resolution** - Design to avoid collisions

**Locks are agreements** - Respect them universally

**Cooldowns create harmony** - Time separation prevents conflicts

**Monitor and adjust** - Track patterns and adapt

**Fail gracefully** - When collisions occur, handle cleanly

Good collision avoidance is invisible - the system just works smoothly with multiple agents collaborating effectively without stepping on each other.