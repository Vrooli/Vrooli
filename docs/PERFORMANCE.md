# Performance Documentation

## Overview

This document tracks performance characteristics, optimizations, and bottlenecks in the Vrooli system. It serves as a guide for understanding system behavior under load and optimizing performance.

## Current Performance Metrics

### System-Wide Metrics

<!-- EMBED:SYSTEM_METRICS:START -->
| Metric | Current Value | Target | Status |
|--------|--------------|--------|--------|
| **System Startup** | ~30 seconds | <20 seconds | ⚠️ Needs Improvement |
| **Resource Response** | <100ms | <50ms | ✅ Acceptable |
| **API Latency (p50)** | 45ms | <50ms | ✅ Good |
| **API Latency (p99)** | 250ms | <200ms | ⚠️ Needs Improvement |
| **Memory Usage (Idle)** | 2.1GB | <2GB | ⚠️ Slightly High |
| **Memory Usage (Peak)** | 8.5GB | <8GB | ⚠️ At Limit |
<!-- EMBED:SYSTEM_METRICS:END -->

### Embedding System Performance

<!-- EMBED:EMBEDDING_METRICS:START -->
| Operation | Current Performance | After Optimization | Improvement |
|-----------|-------------------|-------------------|-------------|
| **Single Embedding** | 250ms | 50ms | 5x faster |
| **Batch Processing** | 10 items/sec | 50 items/sec | 5x faster |
| **Parallel Processing** | 50 items/sec | 100 items/sec | 2x faster |
| **Full Refresh (224 items)** | 7669 seconds | 300 seconds | 25x faster |
| **Incremental Update** | N/A | 10 seconds | New Feature |
| **Search Latency** | 200ms | 50ms | 4x faster |
<!-- EMBED:EMBEDDING_METRICS:END -->

### Resource Performance

<!-- EMBED:RESOURCE_METRICS:START -->
| Resource | Startup Time | Memory Usage | CPU Usage | Status |
|----------|-------------|--------------|-----------|---------|
| **PostgreSQL** | 3s | 500MB | 2% | ✅ Optimal |
| **Redis** | <1s | 100MB | 1% | ✅ Optimal |
| **Qdrant** | 5s | 800MB | 5% | ✅ Good |
| **N8n** | 15s | 600MB | 3% | ⚠️ Slow Start |
| **Ollama** | 20s | 2GB | 10% | ⚠️ Heavy |
<!-- EMBED:RESOURCE_METRICS:END -->

## Performance Bottlenecks

### 1. Embedding Generation

<!-- EMBED:EMBEDDING_BOTTLENECK:START -->
**Issue**: Sequential embedding generation creates significant delays.

**Root Cause**:
- Ollama API calls are synchronous
- No connection pooling
- Large text chunks not optimized
- No caching of identical content

**Impact**:
- 2+ hour refresh times for moderate codebases
- Blocks development workflow
- High memory usage during processing

**Optimization Applied**:
- Implemented 16-worker parallel processing
- Added batch processing with size 50
- Unified embedding service reduces overhead
- File-level instead of function-level for code

**Results**:
- 25x performance improvement
- Memory usage controlled with monitoring
- Still room for improvement with caching
<!-- EMBED:EMBEDDING_BOTTLENECK:END -->

### 2. System Startup Time

<!-- EMBED:STARTUP_BOTTLENECK:START -->
**Issue**: 30-second startup time impacts developer experience.

**Root Cause**:
- Sequential resource initialization
- Ollama model loading (20s)
- N8n initialization (15s)
- No parallel startup

**Optimization Strategy**:
```python
# Parallel startup implementation
async def start_resources_parallel():
    tasks = []
    for resource in resources:
        if resource.can_parallel_start:
            tasks.append(start_resource_async(resource))
    await asyncio.gather(*tasks)
```

**Expected Improvement**: 
- Reduce to <15 seconds with parallel start
- Lazy load Ollama models
- Precompile Python bytecode
<!-- EMBED:STARTUP_BOTTLENECK:END -->

### 3. Memory Usage

<!-- EMBED:MEMORY_BOTTLENECK:START -->
**Issue**: Peak memory usage reaches system limits.

**Root Cause**:
- Loading entire files into memory
- Ollama model persistence (2GB)
- No garbage collection tuning
- Memory leaks in long-running processes

**Optimizations**:
- Stream processing for large files
- Explicit garbage collection after batch operations
- Memory monitoring with automatic throttling
- Process recycling for long-running tasks

**Code Example**:
```python
# Memory-efficient file processing
def process_large_file(filepath):
    with open(filepath, 'r') as f:
        for chunk in iter(lambda: f.read(8192), ''):
            process_chunk(chunk)
            gc.collect()  # Force garbage collection
```
<!-- EMBED:MEMORY_BOTTLENECK:END -->

### 4. Database Query Performance

<!-- EMBED:DATABASE_BOTTLENECK:START -->
**Issue**: Complex queries cause API latency spikes.

**Root Cause**:
- Missing database indexes
- N+1 query problems
- No query result caching
- Inefficient JOIN operations

**Optimizations Applied**:
```sql
-- Added indexes for common queries
CREATE INDEX idx_embeddings_app_id ON embeddings(app_id);
CREATE INDEX idx_embeddings_type ON embeddings(content_type);
CREATE INDEX idx_workflows_status ON workflows(status);

-- Optimized query with proper indexing
SELECT w.*, COUNT(e.id) as execution_count
FROM workflows w
LEFT JOIN executions e ON w.id = e.workflow_id
WHERE w.app_id = $1 AND w.status = 'active'
GROUP BY w.id;
```

**Results**:
- 80% reduction in query time
- Eliminated N+1 problems
- Added Redis caching for hot queries
<!-- EMBED:DATABASE_BOTTLENECK:END -->

## Optimization Techniques

### 1. Parallel Processing

<!-- EMBED:PARALLEL_TECHNIQUE:START -->
**Technique**: Use multi-processing for CPU-bound tasks.

**Implementation**:
```bash
# Parallel processing with controlled workers
MAX_WORKERS=16
export -f process_item

# Using xargs for parallel execution
find . -name "*.json" | \
  xargs -P $MAX_WORKERS -I {} bash -c 'process_item "$@"' _ {}

# Using GNU parallel
find . -name "*.json" | \
  parallel -j $MAX_WORKERS process_item {}
```

**Best Practices**:
- Set workers = CPU cores for CPU-bound tasks
- Set workers = 2-3x cores for I/O-bound tasks
- Monitor memory per worker
- Implement worker pooling
<!-- EMBED:PARALLEL_TECHNIQUE:END -->

### 2. Caching Strategy

<!-- EMBED:CACHING_TECHNIQUE:START -->
**Multi-Level Caching**:

1. **Application Cache** (In-Memory)
   - Hot data (<1MB)
   - TTL: 60 seconds
   - LRU eviction

2. **Redis Cache** (Distributed)
   - Shared across processes
   - TTL: 3600 seconds
   - Selective invalidation

3. **Disk Cache** (Persistent)
   - Large computed results
   - TTL: 24 hours
   - Content-addressed storage

**Implementation**:
```python
@cache_decorator(ttl=3600, cache_type='redis')
def expensive_computation(params):
    # Computation here
    return result
```
<!-- EMBED:CACHING_TECHNIQUE:END -->

### 3. Lazy Loading

<!-- EMBED:LAZY_LOADING:START -->
**Technique**: Load resources only when needed.

**Implementation**:
```python
class ResourceManager:
    def __init__(self):
        self._resources = {}
    
    def get_resource(self, name):
        if name not in self._resources:
            self._resources[name] = self._load_resource(name)
        return self._resources[name]
    
    def _load_resource(self, name):
        # Expensive loading operation
        return load_resource_from_disk(name)
```

**Benefits**:
- Reduced startup time
- Lower memory usage
- Better cache locality
<!-- EMBED:LAZY_LOADING:END -->

### 4. Batch Operations

<!-- EMBED:BATCH_TECHNIQUE:START -->
**Technique**: Process multiple items in single operations.

**Database Batching**:
```python
# Instead of multiple inserts
for item in items:
    db.execute("INSERT INTO table VALUES (?)", item)

# Use batch insert
db.executemany("INSERT INTO table VALUES (?)", items)
```

**API Batching**:
```python
# Batch API calls
def process_batch(items, batch_size=50):
    for i in range(0, len(items), batch_size):
        batch = items[i:i+batch_size]
        api.process_batch(batch)
```

**Benefits**:
- Reduced network overhead
- Better throughput
- Amortized connection costs
<!-- EMBED:BATCH_TECHNIQUE:END -->

## Performance Monitoring

### Metrics Collection

<!-- EMBED:METRICS_COLLECTION:START -->
**System Metrics**:
```bash
# CPU and Memory monitoring
vmstat 1 10  # Every second for 10 seconds
iostat -x 1  # I/O statistics
netstat -i   # Network statistics

# Process-specific monitoring
ps aux | grep vrooli
top -p $(pgrep -f vrooli)
```

**Application Metrics**:
```python
import time
import psutil
import logging

class PerformanceMonitor:
    def __init__(self):
        self.metrics = {}
    
    def measure(self, operation):
        def decorator(func):
            def wrapper(*args, **kwargs):
                start_time = time.time()
                start_memory = psutil.Process().memory_info().rss
                
                result = func(*args, **kwargs)
                
                duration = time.time() - start_time
                memory_delta = psutil.Process().memory_info().rss - start_memory
                
                self.record_metric(operation, duration, memory_delta)
                return result
            return wrapper
        return decorator
    
    def record_metric(self, operation, duration, memory):
        logging.info(f"Performance: {operation} took {duration:.2f}s, {memory/1024/1024:.1f}MB")
```
<!-- EMBED:METRICS_COLLECTION:END -->

### Performance Testing

<!-- EMBED:PERFORMANCE_TESTING:START -->
**Load Testing**:
```bash
# Stress test embedding system
vrooli test performance --component embeddings --load high

# API load testing with Apache Bench
ab -n 1000 -c 10 http://localhost:8080/api/search

# Memory leak detection
valgrind --leak-check=full vrooli develop
```

**Benchmark Suite**:
```python
# benchmarks/test_performance.py
import pytest
import time

@pytest.mark.benchmark
def test_embedding_generation_speed(benchmark):
    result = benchmark(generate_embedding, "test content")
    assert result is not None

@pytest.mark.benchmark  
def test_search_latency(benchmark):
    result = benchmark(search_embeddings, "query")
    assert len(result) > 0
```
<!-- EMBED:PERFORMANCE_TESTING:END -->

## Performance Tuning Guide

### System-Level Tuning

<!-- EMBED:SYSTEM_TUNING:START -->
**Kernel Parameters**:
```bash
# Increase file descriptors
ulimit -n 65536

# Tune network stack
sysctl -w net.core.somaxconn=1024
sysctl -w net.ipv4.tcp_max_syn_backlog=2048

# Memory management
sysctl -w vm.swappiness=10
sysctl -w vm.dirty_ratio=15
```

**Database Tuning**:
```sql
-- PostgreSQL configuration
shared_buffers = 2GB
effective_cache_size = 6GB
work_mem = 32MB
maintenance_work_mem = 512MB
```
<!-- EMBED:SYSTEM_TUNING:END -->

### Application-Level Tuning

<!-- EMBED:APP_TUNING:START -->
**Python Optimizations**:
```python
# Use slots for classes
class OptimizedClass:
    __slots__ = ['attribute1', 'attribute2']

# Use generators for large datasets
def process_large_dataset():
    for item in read_items():
        yield process(item)

# Profile code to find bottlenecks
import cProfile
cProfile.run('main()', 'profile_stats')
```

**Node.js Optimizations**:
```javascript
// Increase heap size
node --max-old-space-size=4096 app.js

// Use clustering for multi-core
const cluster = require('cluster');
const numCPUs = require('os').cpus().length;

if (cluster.isMaster) {
    for (let i = 0; i < numCPUs; i++) {
        cluster.fork();
    }
} else {
    startServer();
}
```
<!-- EMBED:APP_TUNING:END -->

## Performance Roadmap

### Short Term (1 month)

<!-- EMBED:PERF_ROADMAP_SHORT:START -->
1. **Implement embedding cache** - 50% reduction in redundant processing
2. **Parallel resource startup** - Reduce startup to <15 seconds
3. **Query optimization** - Index all foreign keys
4. **Memory profiling** - Identify and fix leaks
5. **Connection pooling** - Implement for all resources
<!-- EMBED:PERF_ROADMAP_SHORT:END -->

### Medium Term (3 months)

<!-- EMBED:PERF_ROADMAP_MEDIUM:START -->
1. **Distributed processing** - Scale across multiple machines
2. **Advanced caching** - Predictive cache warming
3. **Database sharding** - Horizontal scaling for large datasets
4. **CDN integration** - Static asset optimization
5. **Service mesh** - Advanced load balancing
<!-- EMBED:PERF_ROADMAP_MEDIUM:END -->

### Long Term (6+ months)

<!-- EMBED:PERF_ROADMAP_LONG:START -->
1. **GPU acceleration** - For embedding generation
2. **Edge computing** - Process at the edge
3. **Quantum-resistant** - Future-proof cryptography
4. **AI-driven optimization** - Self-tuning system
5. **Zero-copy architecture** - Eliminate data duplication
<!-- EMBED:PERF_ROADMAP_LONG:END -->

## Performance Best Practices

### Do's

<!-- EMBED:PERF_DOS:START -->
✅ **Profile before optimizing** - Measure first
✅ **Cache aggressively** - But invalidate correctly
✅ **Batch operations** - Reduce overhead
✅ **Use appropriate data structures** - Right tool for the job
✅ **Monitor continuously** - Catch regressions early
✅ **Document optimizations** - Explain why, not just what
✅ **Test performance** - Automated performance tests
✅ **Consider trade-offs** - Performance vs maintainability
<!-- EMBED:PERF_DOS:END -->

### Don'ts

<!-- EMBED:PERF_DONTS:START -->
❌ **Premature optimization** - Focus on bottlenecks
❌ **Micro-optimizations** - Unless proven necessary
❌ **Ignore memory** - Memory leaks compound
❌ **Sequential when parallel possible** - Use all cores
❌ **Synchronous I/O** - Use async where appropriate
❌ **Large transactions** - Break into smaller chunks
❌ **Unbounded operations** - Always set limits
❌ **Ignore database indexes** - Critical for performance
<!-- EMBED:PERF_DONTS:END -->

## Conclusion

Performance optimization is an ongoing process. The Vrooli system has made significant improvements, particularly in the embedding system (25x faster), but there's always room for enhancement. Focus on:

1. **Measure everything** - You can't improve what you don't measure
2. **Optimize bottlenecks** - Focus on the 20% that gives 80% improvement
3. **Monitor continuously** - Catch regressions before production
4. **Document changes** - Future developers need context
5. **Balance trade-offs** - Performance vs maintainability vs cost

The goal is not maximum performance at all costs, but optimal performance that meets user needs while maintaining system reliability and developer productivity.