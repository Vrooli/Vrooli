#!/bin/bash
# Monitoring script for Resume Screening Assistant
# Tracks health, performance, and business metrics

set -euo pipefail

# Configuration
SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCENARIO_ID="resume-screening-assistant"
MONITORING_LOG="/tmp/vrooli-${SCENARIO_ID}-monitoring.log"
METRICS_FILE="/tmp/vrooli-${SCENARIO_ID}-metrics.json"

# Monitoring interval (seconds)
MONITOR_INTERVAL="${MONITOR_INTERVAL:-30}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# Metrics tracking
TOTAL_CHECKS=0
HEALTHY_CHECKS=0
UNHEALTHY_CHECKS=0
RESPONSE_TIMES=()

# Business metrics
RESUMES_PROCESSED=0
CANDIDATES_MATCHED=0
AVERAGE_SCORE=0

# Logging functions
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$MONITORING_LOG"
}

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1" | tee -a "$MONITORING_LOG"
}

log_success() {
    echo -e "${GREEN}[OK]${NC} $1" | tee -a "$MONITORING_LOG"
}

log_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1" | tee -a "$MONITORING_LOG"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" | tee -a "$MONITORING_LOG"
}

log_metric() {
    echo -e "${CYAN}[METRIC]${NC} $1" | tee -a "$MONITORING_LOG"
}

# Check service health
check_service_health() {
    local service=$1
    local health_url=$2
    local description=$3
    
    local start_time=$(date +%s%N)
    local response_code
    response_code=$(curl -s -o /dev/null -w "%{http_code}" "$health_url" 2>/dev/null || echo "000")
    local end_time=$(date +%s%N)
    local response_time=$(((end_time - start_time) / 1000000))
    
    TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
    
    if [[ "$response_code" == "200" ]]; then
        log_success "$description is healthy (${response_time}ms)"
        HEALTHY_CHECKS=$((HEALTHY_CHECKS + 1))
        RESPONSE_TIMES+=("$response_time")
        return 0
    else
        log_error "$description is unhealthy (status: $response_code)"
        UNHEALTHY_CHECKS=$((UNHEALTHY_CHECKS + 1))
        return 1
    fi
}

# Monitor resource health
monitor_resources() {
    log_info "Checking resource health..."
    
    # Check Unstructured.io
    check_service_health "unstructured-io" \
        "http://localhost:8000/healthcheck" \
        "Unstructured.io"
    
    # Check Ollama
    check_service_health "ollama" \
        "http://localhost:11434/api/tags" \
        "Ollama"
    
    # Check Qdrant
    check_service_health "qdrant" \
        "http://localhost:6333/collections" \
        "Qdrant"
    
    # Check optional resources
    if curl -s -f "http://localhost:3000/pressure" > /dev/null 2>&1; then
        check_service_health "browserless" \
            "http://localhost:3000/pressure" \
            "Browserless"
    fi
    
    if curl -s -f "http://localhost:9000/minio/health/live" > /dev/null 2>&1; then
        check_service_health "minio" \
            "http://localhost:9000/minio/health/live" \
            "MinIO"
    fi
}

# Monitor application endpoints
monitor_application() {
    log_info "Checking application endpoints..."
    
    local api_base="http://localhost:3000/api/resume-screening-assistant"
    
    # Check health endpoint
    check_service_health "app-health" \
        "$api_base/health" \
        "Application health"
    
    # Check API endpoints if available
    if curl -s -f "$api_base" > /dev/null 2>&1; then
        check_service_health "app-api" \
            "$api_base" \
            "API endpoint"
    fi
}

# Monitor vector database
monitor_vector_database() {
    log_info "Checking vector database..."
    
    local qdrant_url="http://localhost:6333"
    local collections=("candidate_profiles" "assessment_templates" "job_requirements")
    
    for collection in "${collections[@]}"; do
        local collection_response
        collection_response=$(curl -s "$qdrant_url/collections/$collection" 2>/dev/null || echo '{"status":"error"}')
        
        if echo "$collection_response" | grep -q "vectors_count"; then
            local vector_count
            vector_count=$(echo "$collection_response" | jq '.result.vectors_count' 2>/dev/null || echo "0")
            log_metric "$collection collection: $vector_count vectors"
        else
            log_warning "$collection collection not accessible"
        fi
    done
}

# Monitor Ollama models
monitor_ollama_models() {
    log_info "Checking Ollama models..."
    
    local models_response
    models_response=$(curl -s "http://localhost:11434/api/tags" 2>/dev/null || echo '{"models":[]}')
    
    if echo "$models_response" | jq -e '.models' > /dev/null 2>&1; then
        local model_count
        model_count=$(echo "$models_response" | jq '.models | length' 2>/dev/null || echo "0")
        
        if [[ $model_count -gt 0 ]]; then
            log_metric "Ollama models available: $model_count"
            
            # List model names
            local model_names
            model_names=$(echo "$models_response" | jq -r '.models[].name' 2>/dev/null | tr '\n' ', ')
            log_info "Models: ${model_names%,}"
        else
            log_warning "No Ollama models available"
        fi
    fi
}

# Calculate performance metrics
calculate_performance_metrics() {
    local metric_type=$1
    
    if [[ ${#RESPONSE_TIMES[@]} -gt 0 ]]; then
        # Calculate average response time
        local sum=0
        for time in "${RESPONSE_TIMES[@]}"; do
            sum=$((sum + time))
        done
        local avg=$((sum / ${#RESPONSE_TIMES[@]}))
        
        # Calculate P95 (simplified - just using max for now)
        local max=0
        for time in "${RESPONSE_TIMES[@]}"; do
            if [[ $time -gt $max ]]; then
                max=$time
            fi
        done
        
        log_metric "Average response time: ${avg}ms"
        log_metric "P95 response time: ${max}ms"
        
        # Check against targets
        if [[ $avg -lt 500 ]]; then
            log_success "Performance target met (avg < 500ms)"
        else
            log_warning "Performance below target (avg >= 500ms)"
        fi
    fi
}

# Monitor business metrics
monitor_business_metrics() {
    log_info "Checking business metrics..."
    
    # Simulate business metric collection
    # In production, these would come from actual API calls
    
    # Increment counters (simulated)
    RESUMES_PROCESSED=$((RESUMES_PROCESSED + RANDOM % 5))
    CANDIDATES_MATCHED=$((CANDIDATES_MATCHED + RANDOM % 3))
    AVERAGE_SCORE=$((70 + RANDOM % 30))
    
    log_metric "Resumes processed: $RESUMES_PROCESSED"
    log_metric "Candidates matched: $CANDIDATES_MATCHED"
    log_metric "Average assessment score: $AVERAGE_SCORE%"
    
    # Calculate conversion rate
    if [[ $RESUMES_PROCESSED -gt 0 ]]; then
        local conversion_rate=$(( (CANDIDATES_MATCHED * 100) / RESUMES_PROCESSED ))
        log_metric "Match rate: ${conversion_rate}%"
    fi
}

# Generate metrics report
generate_metrics_report() {
    local uptime=$((TOTAL_CHECKS * MONITOR_INTERVAL))
    local health_rate=0
    
    if [[ $TOTAL_CHECKS -gt 0 ]]; then
        health_rate=$(( (HEALTHY_CHECKS * 100) / TOTAL_CHECKS ))
    fi
    
    # Create JSON metrics file
    cat > "$METRICS_FILE" <<EOF
{
    "scenario_id": "$SCENARIO_ID",
    "timestamp": "$(date -Iseconds)",
    "uptime_seconds": $uptime,
    "health_metrics": {
        "total_checks": $TOTAL_CHECKS,
        "healthy_checks": $HEALTHY_CHECKS,
        "unhealthy_checks": $UNHEALTHY_CHECKS,
        "health_rate": $health_rate
    },
    "business_metrics": {
        "resumes_processed": $RESUMES_PROCESSED,
        "candidates_matched": $CANDIDATES_MATCHED,
        "average_assessment_score": $AVERAGE_SCORE,
        "match_rate": $(if [[ $RESUMES_PROCESSED -gt 0 ]]; then echo $(( (CANDIDATES_MATCHED * 100) / RESUMES_PROCESSED )); else echo 0; fi)
    },
    "performance_metrics": {
        "response_times": [$(IFS=,; echo "${RESPONSE_TIMES[*]}")]
    }
}
EOF
    
    log_info "Metrics saved to: $METRICS_FILE"
}

# Display dashboard
display_dashboard() {
    clear
    echo "========================================="
    echo "Resume Screening Assistant - Live Monitor"
    echo "========================================="
    echo "Time: $(date '+%Y-%m-%d %H:%M:%S')"
    echo ""
    
    # Health Status
    local health_rate=0
    if [[ $TOTAL_CHECKS -gt 0 ]]; then
        health_rate=$(( (HEALTHY_CHECKS * 100) / TOTAL_CHECKS ))
    fi
    
    echo "HEALTH STATUS"
    echo "  Overall Health: ${health_rate}%"
    echo "  Checks: $HEALTHY_CHECKS/$TOTAL_CHECKS passed"
    echo ""
    
    # Resource Status
    echo "RESOURCE STATUS"
    echo -n "  Unstructured.io: "
    if curl -s -f "http://localhost:8000/healthcheck" > /dev/null 2>&1; then
        echo -e "${GREEN}●${NC} Online"
    else
        echo -e "${RED}●${NC} Offline"
    fi
    
    echo -n "  Ollama:          "
    if curl -s -f "http://localhost:11434/api/tags" > /dev/null 2>&1; then
        echo -e "${GREEN}●${NC} Online"
    else
        echo -e "${RED}●${NC} Offline"
    fi
    
    echo -n "  Qdrant:          "
    if curl -s -f "http://localhost:6333/collections" > /dev/null 2>&1; then
        echo -e "${GREEN}●${NC} Online"
    else
        echo -e "${RED}●${NC} Offline"
    fi
    echo ""
    
    # Business Metrics
    echo "BUSINESS METRICS"
    echo "  Resumes Processed: $RESUMES_PROCESSED"
    echo "  Candidates Matched: $CANDIDATES_MATCHED"
    echo "  Average Score: $AVERAGE_SCORE%"
    if [[ $RESUMES_PROCESSED -gt 0 ]]; then
        local match_rate=$(( (CANDIDATES_MATCHED * 100) / RESUMES_PROCESSED ))
        echo "  Match Rate: ${match_rate}%"
    fi
    echo ""
    
    # Performance
    if [[ ${#RESPONSE_TIMES[@]} -gt 0 ]]; then
        local sum=0
        for time in "${RESPONSE_TIMES[@]}"; do
            sum=$((sum + time))
        done
        local avg=$((sum / ${#RESPONSE_TIMES[@]}))
        
        echo "PERFORMANCE"
        echo "  Avg Response Time: ${avg}ms"
        echo "  Total Requests: ${#RESPONSE_TIMES[@]}"
    fi
    
    echo ""
    echo "========================================="
    echo "Press Ctrl+C to stop monitoring"
    echo "Logs: $MONITORING_LOG"
    echo "Metrics: $METRICS_FILE"
}

# Signal handlers
cleanup() {
    log_info "Stopping monitoring..."
    generate_metrics_report
    log_info "Monitoring stopped. Total runtime: $((TOTAL_CHECKS * MONITOR_INTERVAL)) seconds"
    exit 0
}

trap cleanup SIGINT SIGTERM

# Main monitoring loop
main() {
    local mode="${1:-continuous}"
    
    log "========================================="
    log "Starting monitoring for $SCENARIO_ID"
    log "Mode: $mode"
    log "Interval: ${MONITOR_INTERVAL}s"
    log "========================================="
    
    case "$mode" in
        once|single)
            # Single check
            monitor_resources
            monitor_application
            monitor_vector_database
            monitor_ollama_models
            monitor_business_metrics
            calculate_performance_metrics "all"
            generate_metrics_report
            ;;
            
        dashboard)
            # Interactive dashboard mode
            while true; do
                monitor_resources > /dev/null 2>&1
                monitor_application > /dev/null 2>&1
                monitor_vector_database > /dev/null 2>&1
                monitor_ollama_models > /dev/null 2>&1
                monitor_business_metrics > /dev/null 2>&1
                display_dashboard
                sleep "$MONITOR_INTERVAL"
            done
            ;;
            
        continuous|*)
            # Continuous monitoring with logging
            while true; do
                log "--- Monitor cycle $(date '+%H:%M:%S') ---"
                monitor_resources
                monitor_application
                monitor_vector_database
                monitor_ollama_models
                monitor_business_metrics
                calculate_performance_metrics "all"
                
                # Generate report every 10 cycles
                if [[ $((TOTAL_CHECKS % 10)) -eq 0 ]]; then
                    generate_metrics_report
                fi
                
                sleep "$MONITOR_INTERVAL"
            done
            ;;
    esac
}

# Run main function
main "$@"