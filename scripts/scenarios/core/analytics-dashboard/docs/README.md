# Resource Monitoring & Management Platform

A comprehensive infrastructure monitoring solution for Vrooli's local resource ecosystem, providing real-time health monitoring, automated alerting, and management capabilities.

## Overview

This platform automatically discovers and monitors Vrooli's 20+ local resources, providing:

- **Auto-discovery**: Automatically finds running resources on startup
- **Real-time monitoring**: 30-second health checks with historical metrics
- **Smart alerting**: SMS/email notifications with intelligent throttling
- **Management interface**: One-click restart/stop capabilities via UI
- **Historical analysis**: QuestDB-powered metrics for trend analysis

## Architecture

### Core Components

1. **PostgreSQL** - Resource registry and configuration storage
2. **QuestDB** - High-performance time-series metrics storage  
3. **n8n** - Workflow automation for monitoring and alerting
4. **Node-RED** - Real-time metrics collection engine
5. **Windmill** - Interactive management dashboard
6. **Vault** - Secure credential storage for notifications
7. **Redis** - Alert throttling and real-time status cache

### Data Flow

```
Resources → Node-RED (collect) → QuestDB (store) → Windmill (display)
           ↓
       n8n (monitor) → Vault (creds) → SMS/Email (alerts)
           ↓
       Redis (throttle)
```

## Quick Start

### Prerequisites

Required services must be running:
- PostgreSQL (port 5433)
- Redis (port 6380)
- QuestDB (port 9009)
- Vault (port 8200)
- n8n (port 5678)
- Windmill (port 5681)

### Installation

1. **Deploy the scenario**:
   ```bash
   cd /path/to/vrooli
   ./scripts/scenarios/injection/engine.sh analytics-dashboard
   ```

2. **Initialize the platform**:
   ```bash
   cd scripts/scenarios/core/analytics-dashboard
   ./deployment/startup.sh
   ```

3. **Verify deployment**:
   ```bash
   ./test.sh
   ```

### Access Points

- **Dashboard**: http://localhost:5681/f/monitoring/dashboard
- **Configuration**: http://localhost:5681/f/monitoring/config
- **n8n Workflows**: http://localhost:5678
- **QuestDB Console**: http://localhost:9009

## Configuration

### Notification Setup

1. **Configure Vault secrets**:
   ```bash
   # Twilio for SMS
   vault kv put secret/monitoring/twilio \
     account_sid=your_sid \
     auth_token=your_token \
     from_number=+1234567890 \
     to_number=+0987654321

   # SMTP for email  
   vault kv put secret/monitoring/smtp \
     host=smtp.gmail.com \
     port=587 \
     username=alerts@yourcompany.com \
     password=your_password \
     from_email=alerts@yourcompany.com \
     to_email=ops@yourcompany.com
   ```

2. **Customize thresholds**:
   Edit `initialization/configuration/monitor-config.json`:
   ```json
   {
     "thresholds": {
       "responseTime": {"warning": 1000, "critical": 5000},
       "availability": {"warning": 95, "critical": 90}
     }
   }
   ```

### Alert Rules

Modify `initialization/configuration/alert-rules.json` to customize:

- **Response time thresholds**
- **Availability targets** 
- **Notification channels**
- **Cooldown periods**

## Features

### Auto-Discovery

- Scans known service ports on startup
- Attempts health checks on discovered services
- Updates resource registry automatically
- Manual discovery via webhook: `POST /webhook/discover-resources`

### Real-time Monitoring

- Health checks every 30 seconds
- Response time tracking
- Availability calculations
- Error rate monitoring
- Historical trend analysis

### Smart Alerting

- **Severity levels**: Info, Warning, Critical
- **Throttling**: Configurable cooldown periods
- **Channels**: SMS (critical), Email (all), Slack (optional)
- **Templates**: Customizable message formats

### Management Actions

Available for each resource type:

- **Restart**: `systemctl restart` or `docker restart`
- **Stop/Start**: Service control
- **View Logs**: Recent log entries
- **Health Check**: Manual health verification
- **View Metrics**: Real-time performance data

## Monitoring Capabilities

### Tracked Metrics

- **Availability**: Uptime percentage
- **Response Time**: HTTP endpoint response times
- **Error Rate**: Failed requests percentage
- **Resource Usage**: CPU, Memory, Disk (where available)
- **Queue Sizes**: For automation/execution services

### Aggregations

- **5-minute**: Rolling averages for dashboards
- **Hourly**: Trend analysis and capacity planning
- **Daily**: Historical reporting and SLA tracking

## Alert Examples

### SMS (Critical)
```
🚨 CRITICAL: postgres is DOWN - Immediate action required\!
```

### Email (Warning)
```
Subject: [WARNING] Resource Alert: redis

Resource: redis
Status: Degraded
Response Time: 1,250ms (threshold: 1,000ms)
Time: 2024-01-15 14:30:22

View Dashboard: http://localhost:5681/f/monitoring/dashboard
```

## Troubleshooting

### Common Issues

1. **Discovery not working**:
   ```bash
   # Check n8n webhook
   curl -X POST http://localhost:5678/webhook/discover-resources
   ```

2. **Alerts not sending**:
   ```bash
   # Verify Vault secrets
   vault kv get secret/monitoring/twilio
   vault kv get secret/monitoring/smtp
   ```

3. **Metrics not collecting**:
   ```bash
   # Check Node-RED flows
   curl http://localhost:1880/flows
   ```

4. **Database connection issues**:
   ```bash
   # Test PostgreSQL
   psql -h localhost -p 5433 -U vrooli -d vrooli -c "SELECT 1"
   
   # Test QuestDB
   curl http://localhost:9009/status
   ```

### Log Locations

- **Startup logs**: `deployment/startup.log`
- **n8n execution logs**: n8n web interface
- **Node-RED logs**: Node-RED debug panel
- **System logs**: Check individual service logs

## API Reference

### Endpoints

- `GET /api/metrics/current` - Current resource status
- `POST /webhook/discover-resources` - Trigger discovery
- `POST /webhook/alert-handler` - Process alerts
- `GET /api/resources` - List all resources
- `POST /api/resources/{id}/action` - Execute resource action

### Webhook Payloads

**Alert Handler**:
```json
{
  "resource_name": "postgres",
  "severity": "critical", 
  "message": "Service is down",
  "metric_type": "availability",
  "metric_value": 0,
  "threshold_value": 1
}
```

## Development

### Adding New Resources

1. Add to `known_services` in monitor-config.json
2. Define resource actions in PostgreSQL seed data
3. Add health check endpoint mapping
4. Update discovery logic in n8n workflow

### Custom Metrics

1. Add metric definition to `metric_configurations` table
2. Update Node-RED collection flow
3. Create QuestDB table columns if needed
4. Add dashboard widget in Windmill

## Performance

### Targets

- **Initialization**: < 3 minutes
- **Health checks**: < 100ms response
- **Alert delivery**: < 30 seconds
- **Dashboard updates**: < 1 second
- **Data retention**: 30 days detailed, 1 year aggregated

### Resource Usage

- **CPU**: ~100-500m (0.1-0.5 cores)
- **Memory**: ~256-512MB
- **Storage**: ~1-10GB (depending on retention)
- **Network**: Minimal (internal only)

## Security

- All credentials stored in Vault
- Role-based access control
- Audit logging for all actions
- No plaintext secrets in configuration
- Sandboxed command execution

## Contributing

1. Follow existing patterns for workflows and configurations
2. Test thoroughly with `./test.sh`
3. Document any new features or configuration options
4. Update alert rules and thresholds as needed

## Support

- **Issues**: Check logs and run validation scripts
- **Configuration**: Review configuration files in `initialization/`
- **Custom needs**: Modify workflows and alert rules
- **Performance**: Adjust monitoring intervals and retention policies
EOF < /dev/null
