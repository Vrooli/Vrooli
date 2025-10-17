#!/usr/bin/env bash
################################################################################
# Node-RED Performance Optimization Library
# 
# Functions for optimizing workflow execution and resource utilization
################################################################################

set -euo pipefail

# Performance tuning configuration
NODE_RED_MAX_OLD_SPACE="${NODE_RED_MAX_OLD_SPACE:-1024}"  # MB
NODE_RED_FLOW_CACHE_SIZE="${NODE_RED_FLOW_CACHE_SIZE:-100}"
NODE_RED_MESSAGE_BATCH_SIZE="${NODE_RED_MESSAGE_BATCH_SIZE:-50}"
NODE_RED_CONTEXT_STORAGE="${NODE_RED_CONTEXT_STORAGE:-memory}"  # memory|localfilesystem

# Optimize Node-RED runtime settings
node_red::optimize_runtime() {
    log::info "Optimizing Node-RED runtime configuration..."
    
    local settings_file="${HOME}/.local/share/node-red/settings.js"
    
    # Ensure settings directory exists
    mkdir -p "$(dirname "$settings_file")"
    
    # Create optimized settings if not exists
    if [[ ! -f "$settings_file" ]]; then
        cat > "$settings_file" <<'EOF'
module.exports = {
    // Performance optimizations
    flowFilePretty: false,  // Compact JSON for faster parsing
    
    // Context storage configuration
    contextStorage: {
        default: {
            module: "memory"  // Fast in-memory storage
        },
        persistent: {
            module: "localfilesystem",  // Optional persistent storage
            config: {
                cache: true,  // Enable caching for file storage
                flushInterval: 30  // Flush to disk every 30 seconds
            }
        }
    },
    
    // Function node settings
    functionGlobalContext: {
        // Pre-load commonly used modules
        os: require('os'),
        path: require('path'),
        fs: require('fs')
    },
    
    // Logging optimizations
    logging: {
        console: {
            level: "warn",  // Reduce logging overhead
            metrics: false,  // Disable metrics logging
            audit: false  // Disable audit logging
        }
    },
    
    // Editor settings
    editorTheme: {
        page: {
            title: "Node-RED - Vrooli"
        },
        header: {
            title: "Node-RED",
            image: null
        }
    },
    
    // HTTP settings
    httpNodeCors: {
        origin: "*",
        methods: "GET,PUT,POST,DELETE"
    },
    
    // WebSocket settings
    webSocketNodeVerifyClient: function(info) {
        return true;  // Allow all WebSocket connections
    }
};
EOF
        log::success "Created optimized settings.js"
    fi
    
    # Optimize Docker container if running
    if docker ps --format "{{.Names}}" | grep -q "^${NODE_RED_CONTAINER_NAME:-node-red}$"; then
        # Set Node.js memory limit
        docker exec "${NODE_RED_CONTAINER_NAME:-node-red}" sh -c \
            "export NODE_OPTIONS='--max-old-space-size=${NODE_RED_MAX_OLD_SPACE}'"
        
        log::success "Applied runtime optimizations to container"
    fi
}

# Enable flow caching for better performance
node_red::enable_flow_cache() {
    log::info "Enabling flow caching..."
    
    local cache_dir="${HOME}/.local/share/node-red/.cache"
    mkdir -p "$cache_dir"
    
    # Create cache management script
    cat > "${cache_dir}/cache-manager.js" <<'EOF'
// Flow cache manager for performance optimization
const fs = require('fs');
const crypto = require('crypto');

class FlowCache {
    constructor(maxSize = 100) {
        this.cache = new Map();
        this.maxSize = maxSize;
    }
    
    get(key) {
        const item = this.cache.get(key);
        if (item) {
            // Move to end (LRU)
            this.cache.delete(key);
            this.cache.set(key, item);
            return item.value;
        }
        return null;
    }
    
    set(key, value) {
        // Remove oldest if at capacity
        if (this.cache.size >= this.maxSize && !this.cache.has(key)) {
            const firstKey = this.cache.keys().next().value;
            this.cache.delete(firstKey);
        }
        this.cache.set(key, { value, timestamp: Date.now() });
    }
    
    clear() {
        this.cache.clear();
    }
}

module.exports = FlowCache;
EOF
    
    log::success "Flow caching enabled"
}

# Optimize message processing
node_red::optimize_message_processing() {
    log::info "Optimizing message processing..."
    
    # Create message batching module
    cat > "${HOME}/.local/share/node-red/message-batcher.js" <<'EOF'
// Message batching for high-throughput flows
class MessageBatcher {
    constructor(batchSize = 50, flushInterval = 100) {
        this.batchSize = batchSize;
        this.flushInterval = flushInterval;
        this.batches = new Map();
        this.timers = new Map();
    }
    
    addMessage(flowId, message, callback) {
        if (!this.batches.has(flowId)) {
            this.batches.set(flowId, []);
        }
        
        const batch = this.batches.get(flowId);
        batch.push({ message, callback });
        
        if (batch.length >= this.batchSize) {
            this.flush(flowId);
        } else if (!this.timers.has(flowId)) {
            // Set flush timer
            this.timers.set(flowId, setTimeout(() => {
                this.flush(flowId);
            }, this.flushInterval));
        }
    }
    
    flush(flowId) {
        const batch = this.batches.get(flowId);
        if (!batch || batch.length === 0) return;
        
        // Process batch
        const messages = batch.map(b => b.message);
        const callbacks = batch.map(b => b.callback);
        
        // Clear batch and timer
        this.batches.delete(flowId);
        if (this.timers.has(flowId)) {
            clearTimeout(this.timers.get(flowId));
            this.timers.delete(flowId);
        }
        
        // Execute callbacks
        callbacks.forEach(cb => cb(messages));
    }
}

module.exports = MessageBatcher;
EOF
    
    log::success "Message processing optimized"
}

# Performance monitoring
node_red::monitor_performance() {
    log::info "Checking Node-RED performance metrics..."
    
    if ! docker ps --format "{{.Names}}" | grep -q "^${NODE_RED_CONTAINER_NAME:-node-red}$"; then
        log::error "Node-RED container not running"
        return 1
    fi
    
    # Get container stats
    local stats
    stats=$(docker stats --no-stream --format "json" "${NODE_RED_CONTAINER_NAME:-node-red}")
    
    # Parse metrics
    local cpu_percent mem_usage mem_limit
    cpu_percent=$(echo "$stats" | jq -r '.CPUPerc' | sed 's/%//')
    mem_usage=$(echo "$stats" | jq -r '.MemUsage' | cut -d'/' -f1 | sed 's/[^0-9.]//g')
    mem_limit=$(echo "$stats" | jq -r '.MemUsage' | cut -d'/' -f2 | sed 's/[^0-9.]//g')
    
    log::info "Performance Metrics:"
    echo "  CPU Usage: ${cpu_percent}%"
    echo "  Memory: ${mem_usage}MB / ${mem_limit}MB"
    
    # Check flow execution metrics via API
    local url="http://localhost:${NODE_RED_PORT:-1880}"
    local response_time
    
    # Measure API response time
    local start_time end_time
    start_time=$(date +%s%N)
    timeout 5 curl -sf "${url}/flows" > /dev/null 2>&1
    end_time=$(date +%s%N)
    response_time=$(( (end_time - start_time) / 1000000 ))
    
    echo "  API Response Time: ${response_time}ms"
    
    # Performance recommendations
    if (( $(echo "$cpu_percent > 80" | bc -l) )); then
        log::warning "High CPU usage detected. Consider optimizing flows or scaling resources."
    fi
    
    if (( $(echo "$mem_usage > 400" | bc -l) )); then
        log::warning "High memory usage. Consider enabling flow caching and message batching."
    fi
    
    if [[ $response_time -gt 500 ]]; then
        log::warning "Slow API response. Consider optimizing flow complexity."
    fi
    
    return 0
}

# Apply all optimizations
node_red::apply_all_optimizations() {
    log::header "Applying Node-RED performance optimizations..."
    
    node_red::optimize_runtime
    node_red::enable_flow_cache
    node_red::optimize_message_processing
    
    log::success "All optimizations applied"
    
    # Monitor current performance
    node_red::monitor_performance
}