#!/bin/bash
# Airbyte Prometheus Metrics Export

set -euo pipefail

# Resource metadata
RESOURCE_NAME="airbyte"
RESOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DATA_DIR="${RESOURCE_DIR}/data"
METRICS_DIR="${DATA_DIR}/metrics"

# Logging functions
log_info() {
    echo "[INFO] $*" >&2
}

log_error() {
    echo "[ERROR] $*" >&2
}

# Enable metrics export
metrics_enable() {
    log_info "Enabling Prometheus metrics export..."
    
    # Create metrics directory
    mkdir -p "$METRICS_DIR"
    
    # Check deployment method
    if docker ps | grep -q airbyte-abctl-control-plane; then
        # For abctl deployment, patch the server deployment
        log_info "Configuring metrics for Kubernetes deployment..."
        
        # Create ConfigMap for metrics configuration
        cat > "${METRICS_DIR}/metrics-config.yaml" <<'EOF'
apiVersion: v1
kind: ConfigMap
metadata:
  name: airbyte-metrics-config
  namespace: airbyte-abctl
data:
  metrics.properties: |
    # Prometheus metrics configuration
    metrics.enabled=true
    metrics.port=9090
    metrics.path=/metrics
    
    # Metrics to export
    metrics.include.jvm=true
    metrics.include.process=true
    metrics.include.http=true
    metrics.include.database=true
    metrics.include.sync=true
EOF
        
        # Apply configuration
        docker exec airbyte-abctl-control-plane kubectl apply -f - < "${METRICS_DIR}/metrics-config.yaml"
        
        # Patch server deployment to expose metrics port
        docker exec airbyte-abctl-control-plane kubectl -n airbyte-abctl patch deployment airbyte-abctl-server \
            --type='json' -p='[
                {"op": "add", "path": "/spec/template/spec/containers/0/ports/-", "value": {"containerPort": 9090, "name": "metrics"}},
                {"op": "add", "path": "/spec/template/spec/containers/0/env/-", "value": {"name": "METRICS_ENABLED", "value": "true"}}
            ]' 2>/dev/null || log_info "Metrics port may already be configured"
        
        # Create Service for metrics endpoint
        cat > "${METRICS_DIR}/metrics-service.yaml" <<'EOF'
apiVersion: v1
kind: Service
metadata:
  name: airbyte-metrics
  namespace: airbyte-abctl
  labels:
    app: airbyte-metrics
  annotations:
    prometheus.io/scrape: "true"
    prometheus.io/port: "9090"
    prometheus.io/path: "/metrics"
spec:
  selector:
    app.kubernetes.io/name: airbyte-server
  ports:
  - name: metrics
    port: 9090
    targetPort: 9090
    protocol: TCP
EOF
        
        docker exec airbyte-abctl-control-plane kubectl apply -f - < "${METRICS_DIR}/metrics-service.yaml"
        
        log_info "✅ Prometheus metrics enabled in Kubernetes"
        echo ""
        echo "Metrics endpoint will be available at: http://localhost:9090/metrics"
        echo "Note: You may need to restart the server pod for changes to take effect"
        echo "Run: vrooli resource airbyte manage restart"
        
    else
        log_error "Airbyte not running. Start with: vrooli resource airbyte manage start"
        return 1
    fi
    
    # Save metrics configuration
    cat > "${METRICS_DIR}/config.json" <<EOF
{
    "enabled": true,
    "port": 9090,
    "path": "/metrics",
    "scrape_interval": "30s",
    "configured_at": "$(date -Iseconds)"
}
EOF
    
    return 0
}

# Disable metrics export
metrics_disable() {
    log_info "Disabling Prometheus metrics export..."
    
    if docker ps | grep -q airbyte-abctl-control-plane; then
        # Remove metrics service
        docker exec airbyte-abctl-control-plane kubectl -n airbyte-abctl delete service airbyte-metrics 2>/dev/null || true
        
        # Remove metrics configmap
        docker exec airbyte-abctl-control-plane kubectl -n airbyte-abctl delete configmap airbyte-metrics-config 2>/dev/null || true
        
        log_info "✅ Prometheus metrics disabled"
    fi
    
    # Update configuration
    if [[ -f "${METRICS_DIR}/config.json" ]]; then
        jq '.enabled = false' "${METRICS_DIR}/config.json" > "${METRICS_DIR}/config.json.tmp"
        mv "${METRICS_DIR}/config.json.tmp" "${METRICS_DIR}/config.json"
    fi
    
    return 0
}

# Check metrics status
metrics_status() {
    log_info "Checking metrics status..."
    
    if [[ -f "${METRICS_DIR}/config.json" ]]; then
        local enabled
        enabled=$(jq -r '.enabled' "${METRICS_DIR}/config.json" 2>/dev/null || echo "false")
        
        if [[ "$enabled" == "true" ]]; then
            echo "Status: Enabled"
            echo "Configuration:"
            jq '.' "${METRICS_DIR}/config.json"
            
            # Check if metrics endpoint is accessible
            if docker ps | grep -q airbyte-abctl-control-plane; then
                echo ""
                echo "Checking metrics endpoint..."
                
                # Try to get metrics through port-forward
                timeout 5 docker exec airbyte-abctl-control-plane \
                    kubectl -n airbyte-abctl port-forward service/airbyte-metrics 9090:9090 &
                local pf_pid=$!
                
                sleep 2
                if curl -sf http://localhost:9090/metrics | head -5 > /dev/null 2>&1; then
                    echo "✅ Metrics endpoint is accessible"
                else
                    echo "⚠️  Metrics endpoint not accessible (may need restart)"
                fi
                
                kill $pf_pid 2>/dev/null || true
            fi
        else
            echo "Status: Disabled"
        fi
    else
        echo "Status: Not configured"
        echo "Run: vrooli resource airbyte metrics enable"
    fi
    
    return 0
}

# Export metrics to file
metrics_export() {
    local output="${1:-metrics_$(date +%Y%m%d_%H%M%S).txt}"
    
    log_info "Exporting metrics to: ${output}..."
    
    if docker ps | grep -q airbyte-abctl-control-plane; then
        # Port-forward and capture metrics
        timeout 10 docker exec airbyte-abctl-control-plane \
            kubectl -n airbyte-abctl port-forward service/airbyte-metrics 9090:9090 &
        local pf_pid=$!
        
        sleep 2
        
        # Capture metrics
        if curl -sf http://localhost:9090/metrics > "$output" 2>/dev/null; then
            log_info "✅ Metrics exported to: ${output}"
            echo "Lines: $(wc -l < "$output")"
        else
            log_error "Failed to export metrics. Endpoint may not be configured."
            echo "Run: vrooli resource airbyte metrics enable"
        fi
        
        kill $pf_pid 2>/dev/null || true
    else
        log_error "Airbyte not running"
        return 1
    fi
    
    return 0
}

# Configure Prometheus scraping
metrics_configure_prometheus() {
    local prometheus_config="${1:-${METRICS_DIR}/prometheus.yml}"
    
    log_info "Generating Prometheus configuration..."
    
    cat > "$prometheus_config" <<'EOF'
# Prometheus configuration for Airbyte metrics
global:
  scrape_interval: 30s
  evaluation_interval: 30s

scrape_configs:
  - job_name: 'airbyte-server'
    static_configs:
      - targets: ['localhost:9090']
        labels:
          service: 'airbyte'
          component: 'server'
    
  - job_name: 'airbyte-worker'
    static_configs:
      - targets: ['localhost:9091']
        labels:
          service: 'airbyte'
          component: 'worker'
    
  - job_name: 'airbyte-temporal'
    static_configs:
      - targets: ['localhost:9092']
        labels:
          service: 'airbyte'
          component: 'temporal'

  # Kubernetes service discovery (if running in K8s)
  - job_name: 'kubernetes-pods'
    kubernetes_sd_configs:
      - role: pod
        namespaces:
          names: ['airbyte-abctl']
    relabel_configs:
      - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
        action: keep
        regex: true
      - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
        action: replace
        target_label: __metrics_path__
        regex: (.+)
      - source_labels: [__address__, __meta_kubernetes_pod_annotation_prometheus_io_port]
        action: replace
        regex: ([^:]+)(?::\d+)?;(\d+)
        replacement: $1:$2
        target_label: __address__
EOF
    
    log_info "✅ Prometheus configuration generated: ${prometheus_config}"
    echo ""
    echo "To use with Prometheus:"
    echo "  prometheus --config.file=${prometheus_config}"
    
    return 0
}

# Show metrics dashboard
metrics_dashboard() {
    log_info "Airbyte Metrics Dashboard"
    echo "========================="
    
    if docker ps | grep -q airbyte-abctl-control-plane; then
        # Get basic metrics from Kubernetes
        echo ""
        echo "Pod Metrics:"
        docker exec airbyte-abctl-control-plane kubectl -n airbyte-abctl top pods 2>/dev/null || {
            echo "  Metrics server not available. Showing pod status instead:"
            docker exec airbyte-abctl-control-plane kubectl -n airbyte-abctl get pods \
                --no-headers | awk '{print "  " $1 ": " $3}'
        }
        
        echo ""
        echo "Resource Usage:"
        docker exec airbyte-abctl-control-plane kubectl -n airbyte-abctl describe nodes | \
            grep -A5 "Allocated resources:" | tail -5
        
        # Try to get application metrics
        echo ""
        echo "Application Metrics:"
        
        # Get sync job metrics
        local sync_count
        sync_count=$(docker exec airbyte-abctl-control-plane kubectl -n airbyte-abctl exec deploy/airbyte-abctl-server -- \
            curl -s http://localhost:8001/api/v1/jobs/list 2>/dev/null | \
            jq '.jobs | length' 2>/dev/null || echo "0")
        echo "  Total sync jobs: ${sync_count}"
        
        # Get connection count
        local connection_count
        connection_count=$(docker exec airbyte-abctl-control-plane kubectl -n airbyte-abctl exec deploy/airbyte-abctl-server -- \
            curl -s http://localhost:8001/api/v1/connections/list 2>/dev/null | \
            jq '.connections | length' 2>/dev/null || echo "0")
        echo "  Active connections: ${connection_count}"
        
    else
        echo "Airbyte not running"
    fi
    
    return 0
}

# Main metrics command handler
cmd_metrics() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        enable)
            metrics_enable "$@"
            ;;
        disable)
            metrics_disable "$@"
            ;;
        status)
            metrics_status "$@"
            ;;
        export)
            metrics_export "$@"
            ;;
        configure)
            metrics_configure_prometheus "$@"
            ;;
        dashboard)
            metrics_dashboard "$@"
            ;;
        *)
            echo "Usage: vrooli resource airbyte metrics <subcommand> [options]"
            echo ""
            echo "Subcommands:"
            echo "  enable      Enable Prometheus metrics export"
            echo "  disable     Disable metrics export"
            echo "  status      Check metrics configuration status"
            echo "  export [file]     Export current metrics to file"
            echo "  configure [file]  Generate Prometheus configuration"
            echo "  dashboard   Show metrics dashboard"
            return 1
            ;;
    esac
}