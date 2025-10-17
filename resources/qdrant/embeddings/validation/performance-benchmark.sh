#!/usr/bin/env bash
# Performance Benchmarking for Qdrant Embeddings
# Tests and measures embedding system performance

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
EMBEDDINGS_DIR="${APP_ROOT}/resources/qdrant/embeddings"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"

# Performance configuration
BENCHMARK_DIR="/tmp/embedding-benchmarks-$$"
trap "rm -rf $BENCHMARK_DIR" EXIT

# Default test parameters
DEFAULT_SAMPLE_SIZE=100
DEFAULT_PARALLEL_WORKERS=4
DEFAULT_EMBEDDING_MODEL="${QDRANT_EMBEDDING_MODEL:-mxbai-embed-large}"

#######################################
# Create benchmark test data
# Arguments:
#   $1 - Sample size (number of test items)
#   $2 - Content type (code|docs|scenarios)
# Returns: Path to test data file
#######################################
create_test_data() {
    local sample_size="$1"
    local content_type="$2"
    
    mkdir -p "$BENCHMARK_DIR"
    local test_file="$BENCHMARK_DIR/test_${content_type}.jsonl"
    
    log::info "Generating $sample_size test samples for $content_type..."
    
    for i in $(seq 1 "$sample_size"); do
        case "$content_type" in
            code)
                local content="function testFunction${i}() { 
    // This is a test function number $i
    console.log('Processing item $i');
    return { id: $i, status: 'processed' };
}"
                local metadata='{"filename":"test'${i}'.js","language":"JavaScript","lines":5,"functions":1,"content_type":"code","extractor":"benchmark"}'
                ;;
            docs)
                local content="# Test Document $i

This is a test documentation file number $i. It contains important information about the system architecture and implementation details.

## Overview
This document describes the functionality of component $i in the system.

## Key Features
- Feature A for component $i
- Feature B with enhanced capabilities
- Integration with external systems"
                local metadata='{"filename":"test'${i}'.md","sections":3,"content_type":"documentation","extractor":"benchmark"}'
                ;;
            scenarios)
                local content="# Test Scenario $i PRD

## Business Objective
Create a test application that demonstrates functionality $i.

## Success Criteria
- Feature $i is implemented and functional
- User can interact with component $i
- System processes requests efficiently

## Technical Requirements
- API endpoint for feature $i
- Database integration
- User interface components"
                local metadata='{"scenario_id":"test-scenario-'${i}'","type":"web_app","complexity":"medium","content_type":"scenario","extractor":"benchmark"}'
                ;;
        esac
        
        jq -n --arg content "$content" --argjson metadata "$metadata" '{content: $content, metadata: $metadata}' >> "$test_file"
    done
    
    log::success "Created test data: $test_file"
    echo "$test_file"
}

#######################################
# Benchmark single embedding generation
# Arguments:
#   $1 - Text content
# Returns: Time in milliseconds
#######################################
benchmark_single_embedding() {
    local content="$1"
    local start_time=$(date +%s%3N)
    
    # Generate embedding via Ollama API
    local embedding_response=$(curl -s -X POST "http://localhost:11434/api/embeddings" \
        -H "Content-Type: application/json" \
        -d "$(jq -n --arg model "$DEFAULT_EMBEDDING_MODEL" --arg prompt "$content" '{model: $model, prompt: $prompt}')")
    
    local end_time=$(date +%s%3N)
    local duration=$((end_time - start_time))
    
    # Validate response
    if echo "$embedding_response" | jq -e '.embedding' >/dev/null 2>&1; then
        echo "$duration"
    else
        log::error "Embedding generation failed for benchmark"
        echo "-1"
    fi
}

#######################################
# Benchmark batch embedding generation
# Arguments:
#   $1 - Test data file (JSONL)
#   $2 - Batch size
# Returns: Performance metrics
#######################################
benchmark_batch_processing() {
    local test_file="$1"
    local batch_size="${2:-10}"
    
    log::info "Benchmarking batch processing with batch size $batch_size..."
    
    local total_items=$(wc -l < "$test_file")
    local start_time=$(date +%s)
    local processed_count=0
    local failed_count=0
    local total_embedding_time=0
    
    # Process in batches
    local batch_num=1
    while IFS= read -r json_line; do
        local content=$(echo "$json_line" | jq -r '.content')
        local embedding_time=$(benchmark_single_embedding "$content")
        
        if [[ "$embedding_time" -gt 0 ]]; then
            ((processed_count++))
            total_embedding_time=$((total_embedding_time + embedding_time))
        else
            ((failed_count++))
        fi
        
        # Progress reporting
        if [[ $((processed_count % batch_size)) -eq 0 ]]; then
            log::info "Processed batch $batch_num ($(( processed_count + failed_count ))/$total_items items)"
            ((batch_num++))
        fi
        
    done < "$test_file"
    
    local end_time=$(date +%s)
    local total_time=$((end_time - start_time))
    
    # Calculate metrics
    local avg_embedding_time=0
    if [[ $processed_count -gt 0 ]]; then
        avg_embedding_time=$((total_embedding_time / processed_count))
    fi
    
    local throughput=0
    if [[ $total_time -gt 0 ]]; then
        throughput=$((processed_count / total_time))
    fi
    
    # Report results
    cat << EOF
{
    "total_items": $total_items,
    "processed_count": $processed_count,
    "failed_count": $failed_count,
    "total_time_seconds": $total_time,
    "avg_embedding_time_ms": $avg_embedding_time,
    "throughput_per_second": $throughput,
    "success_rate": $(echo "scale=2; $processed_count * 100 / $total_items" | bc)
}
EOF
}

#######################################
# Benchmark parallel processing
# Arguments:
#   $1 - Test data file
#   $2 - Number of parallel workers
# Returns: Performance metrics
#######################################
benchmark_parallel_processing() {
    local test_file="$1"
    local workers="${2:-$DEFAULT_PARALLEL_WORKERS}"
    
    log::info "Benchmarking parallel processing with $workers workers..."
    
    local total_items=$(wc -l < "$test_file")
    local chunk_size=$((total_items / workers + 1))
    
    # Split data into chunks
    split -l "$chunk_size" "$test_file" "$BENCHMARK_DIR/chunk_"
    
    local start_time=$(date +%s)
    local pids=()
    local result_files=()
    
    # Start parallel workers
    for chunk_file in "$BENCHMARK_DIR"/chunk_*; do
        local result_file="$chunk_file.results"
        result_files+=("$result_file")
        
        {
            local chunk_processed=0
            local chunk_failed=0
            local chunk_time=0
            
            while IFS= read -r json_line; do
                local content=$(echo "$json_line" | jq -r '.content')
                local embedding_time=$(benchmark_single_embedding "$content")
                
                if [[ "$embedding_time" -gt 0 ]]; then
                    ((chunk_processed++))
                    chunk_time=$((chunk_time + embedding_time))
                else
                    ((chunk_failed++))
                fi
            done < "$chunk_file"
            
            echo "$chunk_processed $chunk_failed $chunk_time" > "$result_file"
        } &
        
        pids+=($!)
    done
    
    # Wait for all workers
    local failed_workers=0
    for pid in "${pids[@]}"; do
        if ! wait "$pid"; then
            ((failed_workers++))
        fi
    done
    
    local end_time=$(date +%s)
    local total_time=$((end_time - start_time))
    
    # Aggregate results
    local total_processed=0
    local total_failed=0
    local total_embedding_time=0
    
    for result_file in "${result_files[@]}"; do
        if [[ -f "$result_file" ]]; then
            local chunk_results=$(cat "$result_file")
            local processed=$(echo "$chunk_results" | cut -d' ' -f1)
            local failed=$(echo "$chunk_results" | cut -d' ' -f2)
            local time=$(echo "$chunk_results" | cut -d' ' -f3)
            
            total_processed=$((total_processed + processed))
            total_failed=$((total_failed + failed))
            total_embedding_time=$((total_embedding_time + time))
        fi
    done
    
    # Calculate metrics
    local avg_embedding_time=0
    if [[ $total_processed -gt 0 ]]; then
        avg_embedding_time=$((total_embedding_time / total_processed))
    fi
    
    local throughput=0
    if [[ $total_time -gt 0 ]]; then
        throughput=$((total_processed / total_time))
    fi
    
    local efficiency=$((total_processed * 100 / (workers * total_time)))
    
    # Report results
    cat << EOF
{
    "workers": $workers,
    "total_items": $total_items,
    "processed_count": $total_processed,
    "failed_count": $total_failed,
    "failed_workers": $failed_workers,
    "total_time_seconds": $total_time,
    "avg_embedding_time_ms": $avg_embedding_time,
    "throughput_per_second": $throughput,
    "efficiency_percent": $efficiency,
    "success_rate": $(echo "scale=2; $total_processed * 100 / $total_items" | bc)
}
EOF
}

#######################################
# Benchmark memory usage during processing
# Arguments:
#   $1 - Test data file
# Returns: Memory usage metrics
#######################################
benchmark_memory_usage() {
    local test_file="$1"
    
    log::info "Benchmarking memory usage..."
    
    local memory_log="$BENCHMARK_DIR/memory.log"
    local start_memory=$(free | awk '/Mem:/ {print $3}')
    
    # Start memory monitoring
    {
        while true; do
            local current_memory=$(free | awk '/Mem:/ {print $3}')
            echo "$(date +%s) $current_memory" >> "$memory_log"
            sleep 1
        done
    } &
    local monitor_pid=$!
    
    # Run embedding processing
    local processing_start=$(date +%s)
    local processed=0
    
    while IFS= read -r json_line && [[ $processed -lt 50 ]]; do  # Limit for memory test
        local content=$(echo "$json_line" | jq -r '.content')
        benchmark_single_embedding "$content" >/dev/null
        ((processed++))
    done < "$test_file"
    
    local processing_end=$(date +%s)
    
    # Stop memory monitoring
    kill $monitor_pid 2>/dev/null || true
    
    # Analyze memory usage
    local peak_memory=$(sort -k2 -nr "$memory_log" | head -1 | cut -d' ' -f2)
    local end_memory=$(free | awk '/Mem:/ {print $3}')
    
    local memory_increase=$((peak_memory - start_memory))
    local memory_increase_mb=$((memory_increase / 1024))
    
    cat << EOF
{
    "processed_items": $processed,
    "processing_time_seconds": $((processing_end - processing_start)),
    "start_memory_kb": $start_memory,
    "peak_memory_kb": $peak_memory,
    "end_memory_kb": $end_memory,
    "memory_increase_mb": $memory_increase_mb,
    "memory_per_item_kb": $((memory_increase / processed))
}
EOF
}

#######################################
# Run comprehensive performance benchmark
#######################################
run_comprehensive_benchmark() {
    local sample_size="${1:-$DEFAULT_SAMPLE_SIZE}"
    
    log::info "=== Comprehensive Performance Benchmark ==="
    log::info "Sample size: $sample_size items"
    log::info "Embedding model: $DEFAULT_EMBEDDING_MODEL"
    
    # Create test data for different content types
    local code_file=$(create_test_data "$sample_size" "code")
    local docs_file=$(create_test_data "$sample_size" "docs")
    
    # Single embedding benchmark
    log::info "Running single embedding benchmark..."
    local single_time=$(benchmark_single_embedding "This is a test sentence for performance benchmarking.")
    log::info "Single embedding time: ${single_time}ms"
    
    echo "=== PERFORMANCE BENCHMARK RESULTS ===" > "$BENCHMARK_DIR/results.txt"
    echo "Timestamp: $(date)" >> "$BENCHMARK_DIR/results.txt"
    echo "Model: $DEFAULT_EMBEDDING_MODEL" >> "$BENCHMARK_DIR/results.txt"
    echo "Sample size: $sample_size" >> "$BENCHMARK_DIR/results.txt"
    echo >> "$BENCHMARK_DIR/results.txt"
    
    # Batch processing benchmarks
    for content_type in "code" "docs"; do
        log::info "Benchmarking $content_type batch processing..."
        local file_var="${content_type}_file"
        local test_file="${!file_var}"
        
        local batch_results=$(benchmark_batch_processing "$test_file" 10)
        echo "=== ${content_type^^} BATCH PROCESSING ===" >> "$BENCHMARK_DIR/results.txt"
        echo "$batch_results" | jq . >> "$BENCHMARK_DIR/results.txt"
        echo >> "$BENCHMARK_DIR/results.txt"
        
        # Report key metrics
        local throughput=$(echo "$batch_results" | jq -r '.throughput_per_second')
        local success_rate=$(echo "$batch_results" | jq -r '.success_rate')
        log::info "$content_type batch: ${throughput} items/sec, ${success_rate}% success rate"
    done
    
    # Parallel processing benchmark
    for workers in 1 2 4; do
        log::info "Benchmarking parallel processing with $workers workers..."
        local parallel_results=$(benchmark_parallel_processing "$code_file" "$workers")
        
        echo "=== PARALLEL PROCESSING ($workers workers) ===" >> "$BENCHMARK_DIR/results.txt"
        echo "$parallel_results" | jq . >> "$BENCHMARK_DIR/results.txt"
        echo >> "$BENCHMARK_DIR/results.txt"
        
        local throughput=$(echo "$parallel_results" | jq -r '.throughput_per_second')
        local efficiency=$(echo "$parallel_results" | jq -r '.efficiency_percent')
        log::info "Parallel ($workers workers): ${throughput} items/sec, ${efficiency}% efficiency"
    done
    
    # Memory usage benchmark
    log::info "Benchmarking memory usage..."
    local memory_results=$(benchmark_memory_usage "$code_file")
    
    echo "=== MEMORY USAGE ===" >> "$BENCHMARK_DIR/results.txt"
    echo "$memory_results" | jq . >> "$BENCHMARK_DIR/results.txt"
    
    local memory_per_item=$(echo "$memory_results" | jq -r '.memory_per_item_kb')
    log::info "Memory usage: ${memory_per_item}KB per item"
    
    # Copy results to embeddings directory
    cp "$BENCHMARK_DIR/results.txt" "$EMBEDDINGS_DIR/validation/benchmark-results-$(date +%Y%m%d-%H%M%S).txt"
    
    log::success "Benchmark complete! Results saved to validation directory"
    log::info "View results: cat $EMBEDDINGS_DIR/validation/benchmark-results-*.txt"
}

#######################################
# Main function
#######################################
main() {
    local benchmark_type="${1:-comprehensive}"
    local sample_size="${2:-$DEFAULT_SAMPLE_SIZE}"
    
    case "$benchmark_type" in
        single)
            local time=$(benchmark_single_embedding "Test content for benchmarking")
            echo "Single embedding time: ${time}ms"
            ;;
        batch)
            local test_file=$(create_test_data "$sample_size" "code")
            benchmark_batch_processing "$test_file" 10
            ;;
        parallel)
            local workers="${3:-$DEFAULT_PARALLEL_WORKERS}"
            local test_file=$(create_test_data "$sample_size" "code")
            benchmark_parallel_processing "$test_file" "$workers"
            ;;
        memory)
            local test_file=$(create_test_data "$sample_size" "code")
            benchmark_memory_usage "$test_file"
            ;;
        comprehensive)
            run_comprehensive_benchmark "$sample_size"
            ;;
        *)
            log::error "Unknown benchmark type: $benchmark_type"
            log::info "Usage: $0 <single|batch|parallel|memory|comprehensive> [sample_size] [workers]"
            return 1
            ;;
    esac
}

# Run main if called directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi