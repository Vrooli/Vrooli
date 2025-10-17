# Prometheus + Grafana Observability Stack

Complete monitoring and observability solution for the Vrooli ecosystem, providing real-time metrics collection, alerting, and visualization.

## Overview

This resource provides:
- **Prometheus**: Time-series metrics database and monitoring system
- **Grafana**: Analytics and visualization platform for metrics
- **Alertmanager**: Alert routing and notification management
- **Exporters**: Metric collectors for various Vrooli resources

## Quick Start

```bash
# Install the monitoring stack
vrooli resource prometheus-grafana manage install

# Start services
vrooli resource prometheus-grafana manage start

# Check status
vrooli resource prometheus-grafana status

# Access interfaces
# Prometheus: http://localhost:9090
# Grafana: http://localhost:3030 (admin/[see credentials])
# Alertmanager: http://localhost:9093
```

## Features

### Core Capabilities
- ✅ Prometheus metrics collection and storage (15-day retention)
- ✅ Grafana visualization with secure authentication
- ✅ Alertmanager for notification routing
- ✅ Node exporter for system metrics
- ✅ Health checks for all services
- ⚠️ Service discovery for Vrooli resources (in progress)
- ⚠️ Pre-configured dashboards (basic setup complete)

### Monitored Resources
- System metrics (CPU, memory, disk, network)
- Docker containers
- PostgreSQL databases
- Redis instances
- N8n workflows
- Custom Vrooli metrics

## Configuration

### Default Ports
- Prometheus: 9090
- Grafana: 3030 (non-standard to avoid conflicts)
- Alertmanager: 9093
- Node Exporter: 9100

### Data Storage
- Prometheus data: `./data/prometheus`
- Grafana data: `./data/grafana`
- Configuration: `./config/`

## Usage Examples

### View Metrics
```bash
# List all monitored targets
vrooli resource prometheus-grafana content list

# Check specific metric
vrooli resource prometheus-grafana content get "up"

# Execute PromQL query
vrooli resource prometheus-grafana content execute "rate(node_cpu_seconds_total[5m])"
```

### Manage Dashboards
```bash
# List available dashboards
vrooli resource prometheus-grafana content list --type dashboard

# Import dashboard
vrooli resource prometheus-grafana content add dashboard ./my-dashboard.json

# Export dashboard
vrooli resource prometheus-grafana content get dashboard "system-overview" > dashboard.json
```

### Configure Alerts
```bash
# Add alert rule
vrooli resource prometheus-grafana content add alert ./alert-rule.yml

# List active alerts
vrooli resource prometheus-grafana content list --type alert

# Silence alert
vrooli resource prometheus-grafana content execute "silence" --alert "HighMemoryUsage" --duration "2h"
```

## Integration

### Adding Custom Metrics
```bash
# Configure custom exporter
cat > custom-exporter.yml << EOF
scrape_configs:
  - job_name: 'my-app'
    static_configs:
      - targets: ['localhost:8080']
EOF

vrooli resource prometheus-grafana content add config ./custom-exporter.yml
```

### Webhook Notifications
```bash
# Configure webhook for alerts
vrooli resource prometheus-grafana content add webhook \
  --url "https://hooks.slack.com/services/..." \
  --channel "#alerts"
```

## Dashboards

### Pre-configured Dashboards
1. **System Overview**: CPU, memory, disk, network metrics
2. **Docker Containers**: Container resource usage and health
3. **PostgreSQL**: Database performance and connections
4. **Redis**: Cache hit rates and memory usage
5. **Vrooli Resources**: Resource-specific metrics

### Creating Custom Dashboards
1. Access Grafana at http://localhost:3030
2. Login with admin credentials
3. Create new dashboard or import from JSON
4. Save and organize in folders

## Alerting

### Default Alert Rules
- High CPU usage (>80% for 5 minutes)
- Low disk space (<10% free)
- Service down (health check failures)
- High memory usage (>90%)
- Database connection pool exhaustion

### Alert Channels
- Email (configured during setup)
- Webhook (for Slack, Discord, etc.)
- PagerDuty (optional)
- Custom integrations via API

## Troubleshooting

### Common Issues

**Services not starting:**
```bash
# Check logs
vrooli resource prometheus-grafana logs

# Verify ports are available
netstat -tlnp | grep -E "9090|3030|9093"

# Reset configuration
vrooli resource prometheus-grafana manage restart --reset
```

**Metrics not appearing:**
```bash
# Check target health
curl http://localhost:9090/api/v1/targets

# Verify exporter is running
vrooli resource prometheus-grafana test integration
```

**Dashboard not loading:**
```bash
# Check Grafana datasource
curl http://localhost:3030/api/datasources

# Restart Grafana
docker restart prometheus-grafana-grafana
```

## Security

- Grafana authentication enabled by default
- Prometheus secured behind reverse proxy in production
- TLS support for remote metric collection
- API keys for programmatic access
- No sensitive data in metrics

## Performance

- Scrape interval: 15 seconds (configurable)
- Retention: 15 days (configurable)
- Query timeout: 30 seconds
- Dashboard cache: 5 minutes

## Dependencies

- Docker and docker-compose
- 2GB RAM minimum
- 10GB disk space for metrics storage
- Network access to monitored resources

## Testing

```bash
# Run smoke tests
vrooli resource prometheus-grafana test smoke

# Run integration tests
vrooli resource prometheus-grafana test integration

# Run all tests
vrooli resource prometheus-grafana test all
```

## API Reference

See [API Documentation](./docs/API.md) for detailed API reference.

## Support

For issues or questions:
- Check [Troubleshooting Guide](./docs/TROUBLESHOOTING.md)
- Review [Configuration Reference](./docs/CONFIGURATION.md)
- Submit issues to Vrooli repository

## License

Part of the Vrooli ecosystem - see main repository for license details.