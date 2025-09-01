# Design Patterns

## Overview

This document captures recurring design patterns in the Vrooli system. These patterns represent proven solutions to common problems and should be applied consistently across the codebase.

## Core Patterns

### 1. Unified Service Pattern

<!-- EMBED:UNIFIED_SERVICE:START -->
**Problem**: Multiple components implementing similar functionality with slight variations, leading to code duplication.

**Solution**: Create a unified service that abstracts common functionality and provides a consistent interface.

**Implementation**:
```bash
# Instead of duplicate implementations
component1::process() { 
    # 85 lines of similar code
}
component2::process() {
    # 85 lines of similar code  
}

# Use unified service
unified::service::process() {
    # Single implementation
}
component1::process() {
    unified::service::process "component1" "$@"
}
```

**Example**: Qdrant embedding service unified 5 extractors, eliminating ~425 lines of duplicate code.

**Benefits**:
- Single source of truth
- Consistent behavior
- Easier maintenance
- Simplified testing
<!-- EMBED:UNIFIED_SERVICE:END -->

### 2. Resource Adapter Pattern

<!-- EMBED:RESOURCE_ADAPTER:START -->
**Problem**: Resources have different interfaces but need to work together seamlessly.

**Solution**: Create adapters that translate between resource interfaces.

**Structure**:
```
resources/
├── resource-a/
│   ├── cli.sh           # Resource interface
│   └── adapters/
│       ├── resource-b.sh # Adapter to resource B
│       └── resource-c.sh # Adapter to resource C
```

**Implementation**:
```bash
# Adapter pattern
resource_a::adapt_to_b() {
    local data="$1"
    # Transform data from A format to B format
    resource_b::process "$(transform_data "$data")"
}
```

**Benefits**:
- Resources remain independent
- Easy to add new integrations
- Clear separation of concerns
<!-- EMBED:RESOURCE_ADAPTER:END -->

### 3. Event-Driven Communication Pattern

<!-- EMBED:EVENT_DRIVEN:START -->
**Problem**: Tight coupling between components makes the system fragile and hard to modify.

**Solution**: Use event bus for asynchronous, loosely-coupled communication.

**Implementation**:
```python
# Publisher
def publish_event(event_type, payload):
    redis_client.publish(f"events:{event_type}", json.dumps(payload))

# Subscriber
def subscribe_to_events(event_type, handler):
    pubsub = redis_client.pubsub()
    pubsub.subscribe(f"events:{event_type}")
    for message in pubsub.listen():
        handler(json.loads(message['data']))
```

**Event Schema**:
```json
{
    "event_id": "uuid",
    "event_type": "resource.started",
    "timestamp": "2025-01-01T00:00:00Z",
    "source": "resource-manager",
    "payload": {},
    "correlation_id": "request-uuid"
}
```

**Benefits**:
- Components can evolve independently
- Easy to add new event handlers
- Natural async processing
- Event replay capability
<!-- EMBED:EVENT_DRIVEN:END -->

### 4. Progressive Enhancement Pattern

<!-- EMBED:PROGRESSIVE_ENHANCEMENT:START -->
**Problem**: Complex features are hard to implement all at once and risky to deploy.

**Solution**: Build features incrementally, with each phase adding value.

**Phases**:
1. **Basic Functionality**: Minimal viable feature
2. **Enhanced Performance**: Optimization and caching
3. **Advanced Features**: Additional capabilities
4. **Intelligence Layer**: AI/ML enhancements

**Example**: Embedding System Evolution
```
Phase 1: Basic sequential processing (works but slow)
Phase 2: Parallel processing (25x faster)
Phase 3: Incremental updates (only changed files)
Phase 4: Predictive caching (anticipate needs)
```

**Benefits**:
- Early value delivery
- Reduced risk
- User feedback incorporation
- Gradual complexity introduction
<!-- EMBED:PROGRESSIVE_ENHANCEMENT:END -->

### 5. Fail-Safe Degradation Pattern

<!-- EMBED:FAIL_SAFE:START -->
**Problem**: Component failures can cascade and bring down the entire system.

**Solution**: Design components to degrade gracefully when dependencies fail.

**Implementation**:
```python
def get_embedding(content):
    try:
        # Try primary method
        return ollama.embed(content)
    except OllamaUnavailable:
        try:
            # Fallback to cached embedding
            return cache.get_embedding(content)
        except CacheMiss:
            # Return basic keyword extraction
            return extract_keywords(content)
```

**Degradation Levels**:
1. Full functionality with all resources
2. Reduced functionality with cache
3. Basic functionality with local processing
4. Safe mode with essential features only

**Benefits**:
- System remains operational during failures
- Users experience degraded service, not outage
- Time to diagnose and fix issues
<!-- EMBED:FAIL_SAFE:END -->

### 6. Batch Processing Pattern

<!-- EMBED:BATCH_PROCESSING:START -->
**Problem**: Processing items individually is inefficient and slow.

**Solution**: Group items into batches for processing.

**Implementation**:
```bash
# Batch processor
process_batch() {
    local batch_size=50
    local items=()
    
    while IFS= read -r item; do
        items+=("$item")
        
        if [[ ${#items[@]} -ge $batch_size ]]; then
            process_items "${items[@]}"
            items=()
        fi
    done
    
    # Process remaining items
    if [[ ${#items[@]} -gt 0 ]]; then
        process_items "${items[@]}"
    fi
}
```

**Optimal Batch Sizes**:
- Database operations: 100-1000 items
- API calls: 10-50 items
- File I/O: 50-100 items
- Embedding generation: 20-50 items

**Benefits**:
- Reduced overhead
- Better resource utilization
- Improved throughput
- Amortized connection costs
<!-- EMBED:BATCH_PROCESSING:END -->

### 7. Scenario-Driven Development Pattern

<!-- EMBED:SCENARIO_DRIVEN:START -->
**Problem**: Building features without clear use cases leads to over-engineering.

**Solution**: Every feature starts with a concrete scenario that provides business value.

**Structure**:
```
scenarios/
├── {scenario-name}/
│   ├── prd.md           # Product requirements
│   ├── init/            # Initial workflows
│   ├── data/            # Test data
│   └── validation/      # Success criteria
```

**Process**:
1. Define scenario with clear business value
2. Create PRD with requirements
3. Build minimal implementation
4. Validate against criteria
5. Scenario becomes reusable capability

**Benefits**:
- Clear success criteria
- Business value focus
- Natural test cases
- Reusable components
<!-- EMBED:SCENARIO_DRIVEN:END -->

### 8. Multi-Phase Refactoring Pattern

<!-- EMBED:MULTI_PHASE:START -->
**Problem**: Large refactoring efforts are risky and can break existing functionality.

**Solution**: Break refactoring into phases, each independently valuable and testable.

**Phase Structure**:
```
Phase 1: Extract (identify and isolate target code)
Phase 2: Abstract (create common interfaces)
Phase 3: Consolidate (merge duplicate code)
Phase 4: Optimize (improve performance)
Phase 5: Enhance (add new capabilities)
```

**Example**: Embedding System Refactor
- Phase 1: Create unified service ✅
- Phase 2: Generic content parser (pending)
- Phase 3: Unified parallel processing (pending)
- Phase 4: Centralized configuration (pending)

**Benefits**:
- Each phase is independently valuable
- Can stop at any phase
- Easier testing and validation
- Reduced risk
<!-- EMBED:MULTI_PHASE:END -->

### 9. Resource Lifecycle Pattern

<!-- EMBED:RESOURCE_LIFECYCLE:START -->
**Problem**: Resources need consistent management across install, start, stop, and cleanup.

**Solution**: Standardized lifecycle interface for all resources.

**Interface**:
```bash
resource::install()    # One-time setup
resource::configure()  # Apply configuration
resource::start()      # Start the resource
resource::stop()       # Stop gracefully
resource::health()     # Check health status
resource::cleanup()    # Remove all traces
```

**State Machine**:
```
UNINSTALLED -> INSTALLED -> CONFIGURED -> RUNNING
     ↑             ↓            ↓           ↓
     └─────────────┴────────────┴──── STOPPED
```

**Benefits**:
- Predictable behavior
- Easy automation
- Consistent error handling
- Clear state transitions
<!-- EMBED:RESOURCE_LIFECYCLE:END -->

### 10. Parallel Worker Pool Pattern

<!-- EMBED:WORKER_POOL:START -->
**Problem**: Processing large datasets sequentially is too slow.

**Solution**: Use worker pool with controlled concurrency.

**Implementation**:
```python
import asyncio
from asyncio import Semaphore

class WorkerPool:
    def __init__(self, max_workers=16):
        self.semaphore = Semaphore(max_workers)
        self.results = []
    
    async def process_item(self, item):
        async with self.semaphore:
            result = await heavy_operation(item)
            self.results.append(result)
    
    async def process_all(self, items):
        tasks = [self.process_item(item) for item in items]
        await asyncio.gather(*tasks)
        return self.results
```

**Worker Sizing**:
- CPU-bound: workers = CPU cores
- I/O-bound: workers = 2-3x CPU cores  
- Memory-bound: workers = available_memory / task_memory

**Benefits**:
- Controlled resource usage
- Prevents system overload
- Optimal throughput
- Easy to monitor
<!-- EMBED:WORKER_POOL:END -->

## Testing Patterns

### 1. Mock Service Pattern

<!-- EMBED:MOCK_SERVICE:START -->
**Problem**: Testing components that depend on external services.

**Solution**: Create mock services that simulate real behavior.

**Implementation**:
```bash
# Mock service implementation
mock_ollama() {
    case "$1" in
        embed)
            echo '{"embedding": [0.1, 0.2, 0.3]}'
            ;;
        generate)
            echo '{"response": "Mocked response"}'
            ;;
    esac
}

# Test setup
setup() {
    export OLLAMA_COMMAND="mock_ollama"
}
```

**Benefits**:
- Tests run without dependencies
- Predictable test behavior
- Fast test execution
- Edge case testing
<!-- EMBED:MOCK_SERVICE:END -->

### 2. Fixture Pattern

<!-- EMBED:FIXTURE:START -->
**Problem**: Tests need consistent, realistic test data.

**Solution**: Create reusable fixtures that represent common scenarios.

**Structure**:
```
test/
├── fixtures/
│   ├── valid/
│   │   ├── workflow.json
│   │   └── scenario.yaml
│   └── invalid/
│       ├── malformed.json
│       └── missing-fields.yaml
```

**Usage**:
```bash
setup_fixtures() {
    cp -r "$FIXTURE_DIR/valid" "$TEST_DIR/"
}

@test "process valid workflow" {
    run process_workflow "$TEST_DIR/valid/workflow.json"
    [ "$status" -eq 0 ]
}
```

**Benefits**:
- Consistent test data
- Realistic scenarios
- Easy to maintain
- Shareable across tests
<!-- EMBED:FIXTURE:END -->

## Performance Patterns

### 1. Caching Pattern

<!-- EMBED:CACHING:START -->
**Problem**: Repeated expensive operations slow down the system.

**Solution**: Multi-level caching with appropriate invalidation.

**Levels**:
1. **In-Memory Cache**: Hot data, microsecond access
2. **Redis Cache**: Shared data, millisecond access
3. **Disk Cache**: Large data, second access
4. **Computed Cache**: Pre-computed results

**Implementation**:
```python
def get_with_cache(key, compute_fn, ttl=3600):
    # L1: Memory cache
    if key in memory_cache:
        return memory_cache[key]
    
    # L2: Redis cache
    cached = redis.get(key)
    if cached:
        memory_cache[key] = cached
        return cached
    
    # L3: Compute and cache
    result = compute_fn()
    redis.setex(key, ttl, result)
    memory_cache[key] = result
    return result
```

**Benefits**:
- Dramatic performance improvement
- Reduced resource usage
- Better scalability
<!-- EMBED:CACHING:END -->

### 2. Streaming Pattern

<!-- EMBED:STREAMING:START -->
**Problem**: Loading large files into memory causes OOM errors.

**Solution**: Process data in streams/chunks.

**Implementation**:
```python
def process_large_file(filepath):
    with open(filepath, 'r') as f:
        for chunk in iter(lambda: f.read(8192), ''):
            process_chunk(chunk)
            yield get_results()
```

**Benefits**:
- Constant memory usage
- Can process unlimited data
- Early result availability
- Better responsiveness
<!-- EMBED:STREAMING:END -->

## Security Patterns

### 1. Least Privilege Pattern

<!-- EMBED:LEAST_PRIVILEGE:START -->
**Problem**: Components with excessive permissions are security risks.

**Solution**: Grant minimum necessary permissions.

**Implementation**:
- Each resource runs with dedicated user
- Resources can only access their data directories
- Network access restricted to required ports
- File system access limited to specific paths

**Benefits**:
- Limits damage from compromised components
- Easier audit and compliance
- Clear permission boundaries
<!-- EMBED:LEAST_PRIVILEGE:END -->

### 2. Secret Management Pattern

<!-- EMBED:SECRET_MANAGEMENT:START -->
**Problem**: Hardcoded secrets are security vulnerabilities.

**Solution**: Centralized secret management with Vault.

**Implementation**:
```python
def get_secret(path):
    # Never hardcode secrets
    # secret = "hardcoded_password"  # WRONG!
    
    # Use vault instead
    return vault_client.read(f"secret/data/{path}")["data"]
```

**Benefits**:
- Secrets never in code
- Audit trail
- Rotation capability
- Access control
<!-- EMBED:SECRET_MANAGEMENT:END -->

## Anti-Patterns to Avoid

### 1. God Object Anti-Pattern

<!-- EMBED:GOD_OBJECT:START -->
**Problem**: Single object/file that does everything.

**Why it's bad**:
- Hard to test
- Difficult to maintain
- High coupling
- Poor reusability

**Solution**: Split into focused components with single responsibilities.
<!-- EMBED:GOD_OBJECT:END -->

### 2. Copy-Paste Programming

<!-- EMBED:COPY_PASTE:START -->
**Problem**: Duplicating code instead of creating reusable functions.

**Why it's bad**:
- Maintenance nightmare
- Inconsistent behavior
- Bug propagation
- Larger codebase

**Solution**: Extract common functionality into shared libraries.
<!-- EMBED:COPY_PASTE:END -->

### 3. Premature Optimization

<!-- EMBED:PREMATURE_OPT:START -->
**Problem**: Optimizing before understanding bottlenecks.

**Why it's bad**:
- Wasted effort
- Complex code
- May optimize wrong thing
- Harder to maintain

**Solution**: Profile first, optimize bottlenecks only.
<!-- EMBED:PREMATURE_OPT:END -->

### 4. Silent Failure

<!-- EMBED:SILENT_FAILURE:START -->
**Problem**: Swallowing errors without logging or handling.

**Why it's bad**:
- Impossible to debug
- Data corruption
- Cascading failures
- Poor user experience

**Solution**: Always log errors, fail fast and loud.
<!-- EMBED:SILENT_FAILURE:END -->

## Pattern Application Guidelines

### When to Apply Patterns

1. **Identify recurring problems** - If you solve the same problem twice, use a pattern
2. **Consider complexity** - Don't over-engineer simple problems
3. **Team familiarity** - Use patterns the team understands
4. **Performance requirements** - Some patterns have overhead
5. **Maintenance burden** - Patterns should simplify, not complicate

### Pattern Selection Matrix

| Problem Type | Recommended Pattern | Complexity |
|-------------|-------------------|------------|
| Code duplication | Unified Service | Low |
| Slow processing | Parallel Workers | Medium |
| External dependencies | Mock Service | Low |
| System coupling | Event-Driven | Medium |
| Resource management | Lifecycle | Low |
| Performance issues | Caching | Medium |
| Large data | Streaming | Medium |
| Complex refactoring | Multi-Phase | High |

## Evolution of Patterns

Patterns evolve as the system grows:

1. **Emergence**: Problem identified repeatedly
2. **Formalization**: Pattern documented and named
3. **Adoption**: Team starts using pattern
4. **Refinement**: Pattern improved based on usage
5. **Standardization**: Pattern becomes standard practice

## Conclusion

These patterns represent collective learning from the Vrooli project. They should be:

1. **Applied consistently** - Same problem, same solution
2. **Documented clearly** - Others need to understand
3. **Evolved carefully** - Patterns can improve
4. **Retired when obsolete** - Don't keep bad patterns

Remember: Patterns are tools, not rules. Use judgment to determine when and how to apply them. The goal is to build maintainable, scalable, and robust systems that deliver value.