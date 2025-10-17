# Performance Tuning Guide

This guide provides comprehensive strategies for optimizing the Qdrant Semantic Knowledge System performance across different workloads and system configurations.

## 🎯 Performance Overview

### Expected Performance Baselines

| Operation | Small Project (<100 items) | Medium Project (100-1K items) | Large Project (1K-10K items) |
|-----------|---------------------------|------------------------------|------------------------------|
| **Single App Search** | <50ms | <100ms | <200ms |
| **Cross-App Search** (5 apps) | <200ms | <500ms | <1s |
| **Cross-App Search** (20 apps) | <500ms | <1.5s | <3s |
| **Full Refresh** | <10s | <60s | <300s |
| **Incremental Refresh** | <5s | <20s | <60s |
| **Embedding Generation** | ~50ms/item | ~50ms/item | ~50ms/item |

### Performance Factors

1. **Qdrant Configuration** - Vector dimensions, distance metrics, storage
2. **Embedding Model** - Model size vs quality tradeoff
3. **Content Volume** - Number of apps, items per app, content size
4. **Hardware Resources** - CPU, RAM, disk I/O, network
5. **Query Patterns** - Search scope, result limits, filtering

## 🚀 Qdrant Optimization

### 1. Vector Configuration

#### Optimal Dimension Settings

```bash
# High quality, slower (default)
DEFAULT_MODEL="mxbai-embed-large"
DEFAULT_DIMENSIONS=1024

# Balanced quality and speed
DEFAULT_MODEL="nomic-embed-text" 
DEFAULT_DIMENSIONS=768

# Fast, lower quality
DEFAULT_MODEL="all-minilm"
DEFAULT_DIMENSIONS=384
```

**Recommendation:** Start with `nomic-embed-text` (768 dims) for best balance.

#### Distance Metric Selection

```bash
# Edit collection creation in cli.sh (internal functions)
create_collection() {
    local collection_name="$1"
    
    # Cosine similarity (default) - best for semantic search
    local distance="Cosine"
    
    # Alternatives:
    # local distance="Euclidean"  # Faster, less semantic accuracy
    # local distance="Dot"        # Fastest, for normalized vectors
    
    curl -X PUT "${QDRANT_URL}/collections/${collection_name}" \
        -H "Content-Type: application/json" \
        -d '{
            "vectors": {
                "size": '"$DEFAULT_DIMENSIONS"',
                "distance": "'"$distance"'"
            },
            "optimizers_config": {
                "default_segment_number": 2,
                "memmap_threshold": 20000
            },
            "hnsw_config": {
                "m": 16,
                "ef_construct": 100
            }
        }'
}
```

### 2. HNSW Index Tuning

#### For Search Speed (Real-time Applications)

```json
{
    "hnsw_config": {
        "m": 32,
        "ef_construct": 200,
        "full_scan_threshold": 10000
    }
}
```

#### For Memory Efficiency (Large Collections)

```json
{
    "hnsw_config": {
        "m": 8,
        "ef_construct": 64,
        "full_scan_threshold": 20000
    }
}
```

#### For Balanced Performance (Recommended)

```json
{
    "hnsw_config": {
        "m": 16,
        "ef_construct": 100,
        "full_scan_threshold": 15000
    }
}
```

### 3. Memory and Storage Optimization

#### Memory-Mapped Storage Configuration

```bash
# Configure memory mapping thresholds
configure_qdrant_storage() {
    # For systems with >16GB RAM
    local memmap_threshold=50000
    
    # For systems with 8-16GB RAM  
    # local memmap_threshold=20000
    
    # For systems with <8GB RAM
    # local memmap_threshold=10000
    
    curl -X PUT "${QDRANT_URL}/collections/${collection_name}" \
        -H "Content-Type: application/json" \
        -d '{
            "optimizers_config": {
                "memmap_threshold": '"$memmap_threshold"',
                "indexing_threshold": 20000,
                "flush_interval_sec": 5
            }
        }'
}
```

#### Disk I/O Optimization

```bash
# Configure Qdrant storage for SSD vs HDD
setup_qdrant_storage() {
    # For SSD storage (recommended)
    export QDRANT_STORAGE_CONFIG='{
        "storage": {
            "optimizers": {
                "deleted_threshold": 0.2,
                "vacuum_min_vector_number": 1000,
                "default_segment_number": 2
            }
        }
    }'
    
    # For HDD storage (reduce random I/O)
    # export QDRANT_STORAGE_CONFIG='{
    #     "storage": {
    #         "optimizers": {
    #             "deleted_threshold": 0.4,
    #             "vacuum_min_vector_number": 5000,
    #             "default_segment_number": 1
    #         }
    #     }
    # }'
}
```

## ⚡ Embedding Generation Optimization

### 1. Model Selection Strategy

#### Performance vs Quality Matrix

| Model | Dimensions | Speed | Quality | Use Case |
|-------|------------|-------|---------|----------|
| `all-minilm` | 384 | Fastest | Good | Development, prototyping |
| `nomic-embed-text` | 768 | Fast | Very Good | Production (recommended) |
| `mxbai-embed-large` | 1024 | Slow | Excellent | Critical applications |

#### Dynamic Model Selection

```bash
# Choose model based on content type and criticality
get_optimal_model() {
    local content_type="$1"
    local app_criticality="$2"
    
    case "$content_type" in
        "knowledge"|"scenarios")
            # High-value content, use best model
            echo "mxbai-embed-large"
            ;;
        "workflows"|"code")
            # Balanced content, use balanced model
            echo "nomic-embed-text"
            ;;
        "resources")
            # Lower priority, use fast model
            echo "all-minilm"
            ;;
    esac
}
```

### 2. Batch Processing Optimization

#### Optimal Batch Sizes

```bash
# Configure batch sizes based on available memory
configure_batch_processing() {
    local available_memory_gb="$1"
    
    if [[ $available_memory_gb -ge 16 ]]; then
        export EMBEDDING_BATCH_SIZE=20
        export PARALLEL_EXTRACTORS=4
    elif [[ $available_memory_gb -ge 8 ]]; then
        export EMBEDDING_BATCH_SIZE=10
        export PARALLEL_EXTRACTORS=2
    else
        export EMBEDDING_BATCH_SIZE=5
        export PARALLEL_EXTRACTORS=1
    fi
}
```

#### Parallel Processing

```bash
# Process extractors in parallel for faster refresh
parallel_embedding_refresh() {
    local app_id="$1"
    
    # Create named pipes for coordination
    mkfifo /tmp/extract_workflows /tmp/extract_scenarios /tmp/extract_docs \
           /tmp/extract_code /tmp/extract_resources
    
    # Start extractors in parallel
    {
        qdrant::extract::workflows_batch "$PWD" "/tmp/workflows.txt"
        echo "workflows_done" > /tmp/extract_workflows
    } &
    
    {
        qdrant::extract::scenarios_batch "$PWD" "/tmp/scenarios.txt"
        echo "scenarios_done" > /tmp/extract_scenarios
    } &
    
    {
        qdrant::extract::docs_batch "$PWD" "/tmp/docs.txt"
        echo "docs_done" > /tmp/extract_docs
    } &
    
    {
        qdrant::extract::code_batch "$PWD" "/tmp/code.txt"
        echo "code_done" > /tmp/extract_code
    } &
    
    {
        qdrant::extract::resources_batch "$PWD" "/tmp/resources.txt"
        echo "resources_done" > /tmp/extract_resources
    } &
    
    # Wait for all extractors to complete
    wait
    
    # Process embeddings in parallel
    for content_type in workflows scenarios docs code resources; do
        {
            process_content_embeddings "$app_id" "$content_type" "/tmp/${content_type}.txt"
        } &
    done
    
    wait
    
    # Cleanup
    rm -f /tmp/extract_* /tmp/*.txt
}
```

### 3. Incremental Processing

#### Content Change Detection

```bash
# Only re-embed changed content
incremental_embedding_update() {
    local app_id="$1"
    
    # Get last embedding timestamp
    local last_update=$(get_last_embedding_timestamp "$app_id")
    
    # Find files modified since last update
    local changed_files=$(find . -newer "$last_update" -type f \
        \( -name "*.json" -o -name "*.md" -o -name "*.sh" -o -name "*.ts" \))
    
    if [[ -z "$changed_files" ]]; then
        log::info "No content changes detected, skipping embedding update"
        return 0
    fi
    
    log::info "Processing $(echo "$changed_files" | wc -l) changed files"
    
    # Process only changed content
    for file in $changed_files; do
        local content_type=$(detect_content_type "$file")
        if [[ -n "$content_type" ]]; then
            process_single_file_embedding "$app_id" "$content_type" "$file"
        fi
    done
}
```

## 🔍 Search Performance Optimization

### 1. Query Optimization

#### Efficient Search Parameters

```bash
# Optimize search parameters for different use cases
optimized_search() {
    local query="$1"
    local use_case="$2"
    
    case "$use_case" in
        "interactive"|"autocomplete")
            # Fast response for user interfaces
            local limit=5
            local ef=50
            local exact=false
            ;;
        "discovery"|"exploration")
            # Broader results for exploration
            local limit=20
            local ef=100
            local exact=false
            ;;
        "precise"|"reference")
            # High accuracy for critical searches
            local limit=10
            local ef=200
            local exact=true
            ;;
    esac
    
    search_with_params "$query" "$limit" "$ef" "$exact"
}
```

#### Search Parameter Tuning

```bash
# Configure search parameters in Qdrant
search_with_params() {
    local query="$1"
    local limit="$2"
    local ef="$3"     # Higher = more accurate, slower
    local exact="$4"  # true = exact search, false = approximate
    
    curl -X POST "${QDRANT_URL}/collections/${collection}/points/search" \
        -H "Content-Type: application/json" \
        -d '{
            "vector": '"$(get_query_vector "$query")"',
            "limit": '"$limit"',
            "params": {
                "hnsw_ef": '"$ef"',
                "exact": '"$exact"'
            },
            "score_threshold": 0.7
        }'
}
```

### 2. Result Caching

#### Query Result Caching

```bash
# Cache search results for frequently accessed queries
search_with_cache() {
    local query="$1"
    local cache_key="search:$(echo -n "$query" | sha256sum | cut -d' ' -f1)"
    
    # Check cache first
    local cached_result=$(redis-cli GET "$cache_key" 2>/dev/null)
    if [[ -n "$cached_result" ]]; then
        log::debug "Cache hit for query: $query"
        echo "$cached_result"
        return 0
    fi
    
    # Perform search
    local result=$(perform_search "$query")
    
    # Cache result for 5 minutes
    redis-cli SETEX "$cache_key" 300 "$result" >/dev/null 2>&1
    
    echo "$result"
}
```

#### Vector Caching

```bash
# Cache generated query vectors
get_cached_query_vector() {
    local query="$1"
    local vector_cache_key="vector:$(echo -n "$query" | sha256sum | cut -d' ' -f1)"
    
    # Check if vector is cached
    local cached_vector=$(redis-cli GET "$vector_cache_key" 2>/dev/null)
    if [[ -n "$cached_vector" ]]; then
        echo "$cached_vector"
        return 0
    fi
    
    # Generate new vector
    local vector=$(generate_embedding "$query")
    
    # Cache vector for 1 hour
    redis-cli SETEX "$vector_cache_key" 3600 "$vector" >/dev/null 2>&1
    
    echo "$vector"
}
```

### 3. Multi-App Search Optimization

#### Parallel Collection Search

```bash
# Search multiple collections in parallel
parallel_multi_app_search() {
    local query="$1"
    local apps=($2)
    local max_parallel=5
    
    # Create temporary directory for results
    local temp_dir=$(mktemp -d)
    
    # Search apps in parallel batches
    local batch_size=$max_parallel
    for ((i=0; i<${#apps[@]}; i+=batch_size)); do
        local batch=("${apps[@]:i:batch_size}")
        
        for app in "${batch[@]}"; do
            {
                search_single_app "$query" "$app" > "$temp_dir/result_$app.json"
            } &
        done
        
        # Wait for batch to complete
        wait
    done
    
    # Combine and rank results
    combine_search_results "$temp_dir"/*.json
    
    # Cleanup
    rm -rf "$temp_dir"
}
```

#### Smart App Filtering

```bash
# Filter apps based on relevance before searching
intelligent_app_filtering() {
    local query="$1"
    local all_apps=($2)
    
    # Get app metadata and relevance scores
    local relevant_apps=()
    for app in "${all_apps[@]}"; do
        local app_score=$(calculate_app_relevance "$query" "$app")
        if (( $(echo "$app_score > 0.3" | bc -l) )); then
            relevant_apps+=("$app")
        fi
    done
    
    log::info "Filtered from ${#all_apps[@]} to ${#relevant_apps[@]} apps"
    echo "${relevant_apps[@]}"
}
```

## 🖥️ System-Level Optimization

### 1. Resource Allocation

#### Memory Allocation Guidelines

```bash
# Calculate optimal memory allocation
calculate_memory_allocation() {
    local total_memory_gb="$1"
    local num_apps="$2"
    local avg_items_per_app="$3"
    
    # Estimate memory needs
    local qdrant_memory=$((total_memory_gb * 60 / 100))      # 60% for Qdrant
    local system_memory=$((total_memory_gb * 20 / 100))      # 20% for system
    local embedding_memory=$((total_memory_gb * 15 / 100))   # 15% for embedding generation
    local cache_memory=$((total_memory_gb * 5 / 100))        # 5% for caching
    
    cat << EOF
Memory Allocation Recommendations:
- Qdrant: ${qdrant_memory}GB
- System: ${system_memory}GB  
- Embedding Generation: ${embedding_memory}GB
- Caching: ${cache_memory}GB

Collection Estimates:
- Total vectors: $((num_apps * avg_items_per_app))
- Memory per vector (1024 dims): ~4KB
- Total vector memory: $((num_apps * avg_items_per_app * 4 / 1024))MB
EOF
}
```

#### CPU Optimization

```bash
# Configure CPU usage for embedding generation
optimize_cpu_usage() {
    local cpu_cores=$(nproc)
    
    # Use 75% of available cores for parallel processing
    local max_parallel_jobs=$((cpu_cores * 3 / 4))
    
    export EMBEDDING_PARALLEL_JOBS="$max_parallel_jobs"
    export OMP_NUM_THREADS="$max_parallel_jobs"
    
    log::info "Configured for $max_parallel_jobs parallel embedding jobs"
}
```

### 2. Storage Optimization

#### SSD vs HDD Configuration

```bash
# Optimize for different storage types
configure_storage_optimization() {
    local storage_type="$1"  # "ssd" or "hdd"
    
    case "$storage_type" in
        "ssd")
            # Optimize for SSD - more random I/O friendly
            export QDRANT_CONFIG='{
                "storage": {
                    "on_disk_payload": true,
                    "optimizers": {
                        "deleted_threshold": 0.2,
                        "vacuum_min_vector_number": 1000,
                        "default_segment_number": 4
                    }
                }
            }'
            ;;
        "hdd")
            # Optimize for HDD - sequential I/O friendly
            export QDRANT_CONFIG='{
                "storage": {
                    "on_disk_payload": true,
                    "optimizers": {
                        "deleted_threshold": 0.4,
                        "vacuum_min_vector_number": 5000,
                        "default_segment_number": 1
                    }
                }
            }'
            ;;
    esac
}
```

### 3. Network Optimization

#### Connection Pooling

```bash
# Optimize network connections to Qdrant
setup_connection_optimization() {
    # Connection pooling for multiple requests
    export QDRANT_CONNECTION_POOL_SIZE=10
    export QDRANT_REQUEST_TIMEOUT=30
    export QDRANT_MAX_RETRIES=3
    
    # TCP optimization
    echo 'net.core.rmem_max = 16777216' >> /etc/sysctl.conf
    echo 'net.core.wmem_max = 16777216' >> /etc/sysctl.conf
    echo 'net.ipv4.tcp_rmem = 4096 16384 16777216' >> /etc/sysctl.conf
    echo 'net.ipv4.tcp_wmem = 4096 16384 16777216' >> /etc/sysctl.conf
    
    sysctl -p
}
```

## 📊 Performance Monitoring

### 1. Metrics Collection

#### Embedding Performance Metrics

```bash
# Monitor embedding generation performance
monitor_embedding_performance() {
    local start_time=$(date +%s.%N)
    local items_processed=0
    
    # Your embedding operations here
    # ...
    
    local end_time=$(date +%s.%N)
    local duration=$(echo "$end_time - $start_time" | bc)
    local items_per_second=$(echo "scale=2; $items_processed / $duration" | bc)
    
    # Log metrics
    log::info "Embedding Performance: ${items_processed} items in ${duration}s (${items_per_second} items/sec)"
    
    # Send to monitoring system
    curl -X POST "http://localhost:9090/metrics" \
        -d "embedding_duration_seconds $duration"
    curl -X POST "http://localhost:9090/metrics" \
        -d "embedding_items_per_second $items_per_second"
}
```

#### Search Performance Metrics

```bash
# Monitor search performance
monitor_search_performance() {
    local query="$1"
    local start_time=$(date +%s.%N)
    
    # Perform search
    local results=$(search_operation "$query")
    
    local end_time=$(date +%s.%N)
    local duration=$(echo "$end_time - $start_time" | bc)
    local result_count=$(echo "$results" | jq length)
    
    # Log metrics
    log::info "Search Performance: query='$query' duration=${duration}s results=$result_count"
    
    # Track slow queries
    if (( $(echo "$duration > 1.0" | bc -l) )); then
        log::warn "Slow query detected: '$query' took ${duration}s"
    fi
    
    echo "$results"
}
```

### 2. Performance Alerting

#### Automated Performance Alerts

```bash
# Set up performance monitoring and alerting
setup_performance_alerts() {
    # Create monitoring script
    cat > /usr/local/bin/qdrant-monitor.sh << 'EOF'
#!/usr/bin/env bash

# Check Qdrant health
qdrant_health=$(curl -s http://localhost:6333/ | jq -r '.status')
if [[ "$qdrant_health" != "ok" ]]; then
    echo "ALERT: Qdrant health check failed: $qdrant_health"
fi

# Check search performance
test_query="test query"
search_time=$(time (curl -s -X POST "http://localhost:6333/collections/test/points/search" \
    -H "Content-Type: application/json" \
    -d '{"vector": [0.1, 0.2, 0.3], "limit": 5}' > /dev/null) 2>&1 | grep real | awk '{print $2}')

if [[ $(echo "$search_time" | sed 's/[^0-9.]//g') > 1.0 ]]; then
    echo "ALERT: Search performance degraded: ${search_time}"
fi

# Check memory usage
memory_usage=$(free | grep Mem | awk '{printf "%.1f", $3/$2 * 100.0}')
if [[ $(echo "$memory_usage > 85" | bc -l) ]]; then
    echo "ALERT: High memory usage: ${memory_usage}%"
fi
EOF

    chmod +x /usr/local/bin/qdrant-monitor.sh
    
    # Add to crontab for regular monitoring
    (crontab -l 2>/dev/null; echo "*/5 * * * * /usr/local/bin/qdrant-monitor.sh") | crontab -
}
```

## 🔧 Performance Troubleshooting

### 1. Common Performance Issues

#### Slow Search Responses

```bash
# Diagnose and fix slow search performance
diagnose_slow_search() {
    local query="$1"
    
    echo "=== Search Performance Diagnosis ==="
    
    # Check Qdrant health
    local health=$(curl -s http://localhost:6333/)
    echo "Qdrant Health: $health"
    
    # Check collection info
    local collection_info=$(curl -s "http://localhost:6333/collections")
    echo "Collections: $(echo "$collection_info" | jq '.result[].name')"
    
    # Check vector count
    local vector_counts=$(echo "$collection_info" | jq '.result[] | "\(.name): \(.vectors_count) vectors"')
    echo "Vector Counts: $vector_counts"
    
    # Test query performance
    echo "Testing query performance..."
    time search_single_app "$query" "test-app"
    
    # Recommendations
    echo "=== Performance Recommendations ==="
    echo "1. Consider reducing ef parameter for faster search"
    echo "2. Use exact=false for approximate search"
    echo "3. Implement result caching for frequent queries"
    echo "4. Filter by content type to reduce search scope"
}
```

#### High Memory Usage

```bash
# Diagnose memory issues
diagnose_memory_usage() {
    echo "=== Memory Usage Analysis ==="
    
    # System memory
    free -h
    
    # Qdrant memory usage
    local qdrant_pid=$(pgrep qdrant)
    if [[ -n "$qdrant_pid" ]]; then
        echo "Qdrant Memory Usage:"
        ps -p "$qdrant_pid" -o pid,ppid,cmd,%mem,%cpu --sort=-%mem
    fi
    
    # Collection sizes
    echo "Collection Memory Estimates:"
    curl -s "http://localhost:6333/collections" | jq -r '.result[] | 
        "\(.name): \(.vectors_count) vectors ≈ \((.vectors_count * 4 / 1024) | floor)MB"'
    
    echo "=== Optimization Suggestions ==="
    echo "1. Consider using smaller embedding dimensions"
    echo "2. Implement collection archiving for old data"
    echo "3. Increase memmap_threshold to use disk storage"
    echo "4. Consider horizontal scaling with multiple Qdrant instances"
}
```

### 2. Performance Tuning Recipes

#### Recipe 1: Fast Development Setup

```bash
# Optimize for development speed over accuracy
setup_fast_development() {
    # Use fast, smaller model
    export DEFAULT_MODEL="all-minilm"
    export DEFAULT_DIMENSIONS=384
    
    # Reduce search accuracy for speed
    export DEFAULT_SEARCH_EF=50
    export DEFAULT_SEARCH_EXACT=false
    
    # Smaller batch sizes for faster feedback
    export EMBEDDING_BATCH_SIZE=5
    
    # Enable aggressive caching
    export ENABLE_SEARCH_CACHE=true
    export CACHE_TTL=300
    
    log::info "Configured for fast development (reduced accuracy)"
}
```

#### Recipe 2: Production Accuracy Setup

```bash
# Optimize for production accuracy and reliability
setup_production_accuracy() {
    # Use high-quality model
    export DEFAULT_MODEL="mxbai-embed-large"
    export DEFAULT_DIMENSIONS=1024
    
    # Higher search accuracy
    export DEFAULT_SEARCH_EF=200
    export DEFAULT_SEARCH_EXACT=false
    
    # Larger batch sizes for efficiency
    export EMBEDDING_BATCH_SIZE=20
    
    # Conservative caching
    export ENABLE_SEARCH_CACHE=true
    export CACHE_TTL=600
    
    # Enable monitoring
    export ENABLE_PERFORMANCE_MONITORING=true
    
    log::info "Configured for production accuracy"
}
```

#### Recipe 3: Large Scale Setup

```bash
# Optimize for large scale deployments
setup_large_scale() {
    # Balanced model for scale
    export DEFAULT_MODEL="nomic-embed-text"
    export DEFAULT_DIMENSIONS=768
    
    # Optimize for memory efficiency
    export QDRANT_MEMMAP_THRESHOLD=20000
    export QDRANT_DEFAULT_SEGMENT_NUMBER=2
    
    # Aggressive parallel processing
    export EMBEDDING_PARALLEL_JOBS=8
    export PARALLEL_EXTRACTORS=4
    
    # Extended caching
    export ENABLE_SEARCH_CACHE=true
    export CACHE_TTL=1800
    
    # Enable comprehensive monitoring
    export ENABLE_PERFORMANCE_MONITORING=true
    export ENABLE_MEMORY_MONITORING=true
    
    log::info "Configured for large scale deployment"
}
```

## 📈 Performance Benchmarking

### 1. Benchmark Script

```bash
#!/usr/bin/env bash
# Performance benchmark script

benchmark_system_performance() {
    local test_queries=(
        "send email notifications"
        "user authentication"
        "database optimization"
        "error handling patterns"
        "webhook processing"
    )
    
    echo "=== Semantic Search Performance Benchmark ==="
    echo "System: $(uname -a)"
    echo "Memory: $(free -h | grep Mem | awk '{print $2}')"
    echo "CPU: $(lscpu | grep 'Model name' | sed 's/Model name:\s*//')"
    echo "Timestamp: $(date)"
    echo
    
    # Single app search benchmark
    echo "=== Single App Search ==="
    local single_app_times=()
    for query in "${test_queries[@]}"; do
        local start_time=$(date +%s.%N)
        search_single_app "$query" "test-app" > /dev/null
        local end_time=$(date +%s.%N)
        local duration=$(echo "$end_time - $start_time" | bc)
        single_app_times+=("$duration")
        echo "Query: '$query' - ${duration}s"
    done
    
    # Multi-app search benchmark
    echo "=== Multi-App Search ==="
    local multi_app_times=()
    for query in "${test_queries[@]}"; do
        local start_time=$(date +%s.%N)
        search_all_apps "$query" > /dev/null
        local end_time=$(date +%s.%N)
        local duration=$(echo "$end_time - $start_time" | bc)
        multi_app_times+=("$duration")
        echo "Query: '$query' - ${duration}s"
    done
    
    # Calculate averages
    local single_avg=$(printf '%s\n' "${single_app_times[@]}" | awk '{sum+=$1} END {print sum/NR}')
    local multi_avg=$(printf '%s\n' "${multi_app_times[@]}" | awk '{sum+=$1} END {print sum/NR}')
    
    echo
    echo "=== Performance Summary ==="
    echo "Single App Search Average: ${single_avg}s"
    echo "Multi-App Search Average: ${multi_avg}s"
    echo
    
    # Performance rating
    if (( $(echo "$single_avg < 0.1" | bc -l) )); then
        echo "Single App Performance: EXCELLENT"
    elif (( $(echo "$single_avg < 0.2" | bc -l) )); then
        echo "Single App Performance: GOOD"
    else
        echo "Single App Performance: NEEDS OPTIMIZATION"
    fi
    
    if (( $(echo "$multi_avg < 0.5" | bc -l) )); then
        echo "Multi-App Performance: EXCELLENT"
    elif (( $(echo "$multi_avg < 1.0" | bc -l) )); then
        echo "Multi-App Performance: GOOD"
    else
        echo "Multi-App Performance: NEEDS OPTIMIZATION"
    fi
}
```

---

## 📋 Performance Checklist

### Pre-Deployment Performance Checklist

- [ ] **Model Selection**: Chosen appropriate embedding model for use case
- [ ] **Qdrant Configuration**: Optimized HNSW parameters and storage settings
- [ ] **Memory Allocation**: Configured appropriate memory limits and mapping
- [ ] **Batch Processing**: Set optimal batch sizes for available resources
- [ ] **Caching Strategy**: Implemented result and vector caching
- [ ] **Monitoring Setup**: Configured performance metrics and alerting
- [ ] **Benchmark Testing**: Ran performance benchmarks on target hardware
- [ ] **Resource Limits**: Set appropriate CPU and memory limits
- [ ] **Storage Optimization**: Configured for SSD/HDD as appropriate
- [ ] **Network Tuning**: Optimized connection pooling and timeouts

### Performance Maintenance Tasks

- [ ] **Weekly**: Review performance metrics and identify trends
- [ ] **Monthly**: Run benchmark tests and compare with baselines
- [ ] **Quarterly**: Review and optimize Qdrant configuration
- [ ] **As Needed**: Tune parameters based on usage patterns
- [ ] **Before Scaling**: Re-benchmark with increased load

---

*Remember: Performance optimization is an iterative process. Start with the recommended settings, measure actual performance, and tune based on your specific workload patterns.*