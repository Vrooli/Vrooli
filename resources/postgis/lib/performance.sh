#!/bin/bash
# PostGIS Performance Optimization Functions

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/*}/../.." && builtin pwd)}"
POSTGIS_PERF_LIB_DIR="${APP_ROOT}/resources/postgis/lib"

# Source dependencies
source "${POSTGIS_PERF_LIB_DIR}/common.sh"

# PostGIS configuration
POSTGIS_CONTAINER="postgis-main"

#######################################
# Analyze spatial indexes and provide recommendations
# Returns: Index analysis and recommendations
#######################################
postgis::performance::analyze_indexes() {
    log::header "PostGIS Index Analysis"
    
    local container="${POSTGIS_CONTAINER}"
    if ! docker ps --format "{{.Names}}" | grep -q "^${container}$"; then
        log::error "PostGIS container not running"
        return 1
    fi
    
    # Check for missing spatial indexes
    log::info "Checking for tables without spatial indexes..."
    docker exec "${container}" psql -U vrooli -d spatial <<'EOF'
SELECT 
    t.schemaname,
    t.tablename,
    a.attname as geometry_column,
    'CREATE INDEX idx_' || t.tablename || '_' || a.attname || ' ON ' || 
    t.schemaname || '.' || t.tablename || ' USING GIST(' || a.attname || ');' as create_index_sql
FROM pg_tables t
JOIN pg_attribute a ON a.attrelid = (t.schemaname || '.' || t.tablename)::regclass
JOIN pg_type ty ON ty.oid = a.atttypid
WHERE t.schemaname = 'public'
AND ty.typname IN ('geometry', 'geography')
AND NOT EXISTS (
    SELECT 1 FROM pg_indexes i 
    WHERE i.schemaname = t.schemaname 
    AND i.tablename = t.tablename 
    AND i.indexdef LIKE '%' || a.attname || '%'
    AND i.indexdef LIKE '%gist%'
);
EOF
    
    # Analyze index usage
    log::info "Analyzing index usage statistics..."
    docker exec "${container}" psql -U vrooli -d spatial <<'EOF'
SELECT 
    schemaname,
    tablename,
    indexname,
    idx_scan as index_scans,
    idx_tup_read as tuples_read,
    idx_tup_fetch as tuples_fetched,
    pg_size_pretty(pg_relation_size(indexrelid)) as index_size,
    CASE 
        WHEN idx_scan = 0 THEN 'UNUSED - Consider dropping'
        WHEN idx_scan < 100 THEN 'LOW USAGE'
        WHEN idx_scan < 1000 THEN 'MODERATE USAGE'
        ELSE 'HIGH USAGE'
    END as usage_category
FROM pg_stat_user_indexes
WHERE schemaname = 'public'
ORDER BY idx_scan DESC;
EOF
    
    return 0
}

#######################################
# Optimize query performance with EXPLAIN ANALYZE
# Args: SQL query to analyze
# Returns: Query plan with recommendations
#######################################
postgis::performance::analyze_query() {
    local query="$1"
    
    if [[ -z "$query" ]]; then
        log::error "Usage: postgis performance analyze-query \"SELECT ...\""
        return 1
    fi
    
    local container="${POSTGIS_CONTAINER}"
    if ! docker ps --format "{{.Names}}" | grep -q "^${container}$"; then
        log::error "PostGIS container not running"
        return 1
    fi
    
    log::header "Query Performance Analysis"
    
    # Run EXPLAIN ANALYZE
    log::info "Analyzing query execution plan..."
    docker exec "${container}" psql -U vrooli -d spatial -c "EXPLAIN (ANALYZE, BUFFERS, VERBOSE) $query" 2>&1 || {
        log::error "Query analysis failed"
        return 1
    }
    
    # Provide recommendations based on common patterns
    log::info "Performance recommendations:"
    echo "1. Ensure spatial columns have GIST indexes"
    echo "2. Use ST_DWithin() instead of ST_Distance() for proximity searches"
    echo "3. Simplify complex geometries with ST_Simplify() when appropriate"
    echo "4. Use && operator for bounding box checks before expensive operations"
    echo "5. Consider clustering tables by spatial index for better cache locality"
    
    return 0
}

#######################################
# Optimize database configuration for spatial workloads
# Returns: Configuration recommendations
#######################################
postgis::performance::tune_config() {
    log::header "PostGIS Configuration Tuning"
    
    local container="${POSTGIS_CONTAINER}"
    if ! docker ps --format "{{.Names}}" | grep -q "^${container}$"; then
        log::error "PostGIS container not running"
        return 1
    fi
    
    # Get current configuration
    log::info "Current PostGIS configuration..."
    docker exec "${container}" psql -U vrooli -d spatial <<'EOF'
SELECT name, setting, unit, category, short_desc
FROM pg_settings
WHERE name IN (
    'shared_buffers',
    'work_mem',
    'maintenance_work_mem',
    'effective_cache_size',
    'max_connections',
    'checkpoint_segments',
    'checkpoint_completion_target',
    'wal_buffers',
    'default_statistics_target',
    'random_page_cost',
    'effective_io_concurrency'
)
ORDER BY category, name;
EOF
    
    # Apply optimizations for spatial workloads
    log::info "Applying spatial workload optimizations..."
    
    # These are recommendations, actual values depend on available memory
    cat <<'EOF'

Recommended configuration for spatial workloads:

shared_buffers = 256MB          # 25% of RAM for dedicated server
work_mem = 16MB                 # Memory for sorts/joins per operation
maintenance_work_mem = 128MB    # Memory for VACUUM, CREATE INDEX
effective_cache_size = 1GB      # OS cache estimate
random_page_cost = 2.0          # SSD optimization (4.0 for HDD)

For production workloads, also consider:
- Enable parallel query execution
- Tune autovacuum for frequently updated spatial tables
- Use table partitioning for large spatial datasets
- Consider PostGIS topology for complex spatial relationships
EOF
    
    return 0
}

#######################################
# Create optimized spatial indexes
# Args: table_name geometry_column
# Returns: 0 on success
#######################################
postgis::performance::create_spatial_index() {
    local table_name="$1"
    local geom_column="${2:-geom}"
    
    if [[ -z "$table_name" ]]; then
        log::error "Usage: postgis performance create-index <table_name> [geometry_column]"
        return 1
    fi
    
    local container="${POSTGIS_CONTAINER}"
    if ! docker ps --format "{{.Names}}" | grep -q "^${container}$"; then
        log::error "PostGIS container not running"
        return 1
    fi
    
    log::info "Creating spatial index on $table_name.$geom_column..."
    
    # Create GIST index
    docker exec "${container}" psql -U vrooli -d spatial <<EOF
-- Create spatial index
CREATE INDEX IF NOT EXISTS idx_${table_name}_${geom_column} 
ON ${table_name} USING GIST(${geom_column});

-- Update statistics
ANALYZE ${table_name};

-- Cluster table by spatial index for better performance
CLUSTER ${table_name} USING idx_${table_name}_${geom_column};

-- Show index info
SELECT 
    indexname,
    indexdef,
    pg_size_pretty(pg_relation_size(indexname::regclass)) as size
FROM pg_indexes 
WHERE tablename = '${table_name}'
AND indexname LIKE '%${geom_column}%';
EOF
    
    if [[ $? -eq 0 ]]; then
        log::success "Spatial index created successfully"
        return 0
    else
        log::error "Failed to create spatial index"
        return 1
    fi
}

#######################################
# Vacuum and analyze spatial tables
# Returns: 0 on success
#######################################
postgis::performance::vacuum_analyze() {
    log::header "PostGIS Vacuum and Analyze"
    
    local container="${POSTGIS_CONTAINER}"
    if ! docker ps --format "{{.Names}}" | grep -q "^${container}$"; then
        log::error "PostGIS container not running"
        return 1
    fi
    
    log::info "Running VACUUM ANALYZE on spatial tables..."
    
    # Get list of tables with geometry columns
    local tables=$(docker exec "${container}" psql -U vrooli -d spatial -t -c "
        SELECT DISTINCT f_table_name 
        FROM geometry_columns 
        WHERE f_table_schema = 'public';" 2>/dev/null)
    
    for table in $tables; do
        if [[ -n "$table" ]]; then
            log::info "Vacuuming table: $table"
            docker exec "${container}" psql -U vrooli -d spatial -c "VACUUM ANALYZE $table;" 2>/dev/null
        fi
    done
    
    log::success "Vacuum and analyze completed"
    return 0
}

#######################################
# Show query performance statistics
# Returns: Performance stats
#######################################
postgis::performance::show_stats() {
    log::header "PostGIS Performance Statistics"
    
    local container="${POSTGIS_CONTAINER}"
    if ! docker ps --format "{{.Names}}" | grep -q "^${container}$"; then
        log::error "PostGIS container not running"
        return 1
    fi
    
    # Database statistics
    log::info "Database statistics..."
    docker exec "${container}" psql -U vrooli -d spatial <<'EOF'
SELECT 
    'Database Size' as metric,
    pg_size_pretty(pg_database_size('spatial')) as value
UNION ALL
SELECT 
    'Total Tables',
    COUNT(*)::text
FROM pg_tables WHERE schemaname = 'public'
UNION ALL
SELECT 
    'Spatial Tables',
    COUNT(DISTINCT f_table_name)::text
FROM geometry_columns WHERE f_table_schema = 'public'
UNION ALL
SELECT 
    'Total Indexes',
    COUNT(*)::text
FROM pg_indexes WHERE schemaname = 'public'
UNION ALL
SELECT 
    'Cache Hit Ratio',
    ROUND(100.0 * sum(blks_hit) / NULLIF(sum(blks_hit) + sum(blks_read), 0), 2)::text || '%'
FROM pg_stat_database WHERE datname = 'spatial';
EOF
    
    # Slow queries
    log::info "Slowest queries (if pg_stat_statements enabled)..."
    docker exec "${container}" psql -U vrooli -d spatial -c "
        SELECT 
            SUBSTRING(query, 1, 60) as query_preview,
            ROUND(mean_exec_time::numeric, 2) as avg_ms,
            calls
        FROM pg_stat_statements
        WHERE query NOT LIKE '%pg_stat%'
        ORDER BY mean_exec_time DESC
        LIMIT 5;" 2>/dev/null || echo "pg_stat_statements not enabled"
    
    return 0
}

# Export functions
export -f postgis::performance::analyze_indexes
export -f postgis::performance::analyze_query
export -f postgis::performance::tune_config
export -f postgis::performance::create_spatial_index
export -f postgis::performance::vacuum_analyze
export -f postgis::performance::show_stats