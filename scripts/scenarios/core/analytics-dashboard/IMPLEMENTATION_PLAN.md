# Resource Monitoring Dashboard - Implementation Plan

## Overview
Transform the generic analytics-dashboard scenario into a **Resource Monitoring & Management Platform** that auto-discovers and monitors Vrooli's local resources with real-time alerts and management capabilities.

## Core Requirements
1. **Auto-discovery**: Find locally running resources (configurable from UI)
2. **Status UI**: View resource status with stop/restart options
3. **Alert System**: Vault-connected n8n workflow for SMS alerts with throttling
4. **Event Storage**: QuestDB integration for time-series events
5. **Documentation**: In-app documentation page

## Architecture

### Resources Used
- **PostgreSQL**: Resource configurations, alert rules, discovered services
- **QuestDB**: Comprehensive time-series storage with resource metrics, alert events, performance data, API metrics, and aggregated analytics
- **n8n**: Monitoring workflows and alert notifications
- **Node-RED**: Real-time metric collection and streaming
- **Windmill**: Interactive UI for monitoring and management
- **Vault**: Secure credential storage for notifications
- **Redis**: Alert throttling cache and real-time status

### Key Workflows

1. **Auto-Discovery Flow**
   - Scans localhost ports using port-registry.sh ranges
   - Attempts health checks on known endpoints
   - Stores discovered resources in PostgreSQL
   - Updates UI resource registry

2. **Continuous Monitoring Loop**
   - Node-RED polls resources every 30 seconds
   - Writes comprehensive metrics to QuestDB (availability, response times, performance data)
   - Stores aggregated analytics (5min, hourly, daily summaries)
   - Publishes status to Redis
   - Triggers n8n for threshold violations

3. **Alert Notification Workflow**
   - Checks Redis for throttling
   - Retrieves credentials from Vault
   - Sends SMS/email based on severity
   - Implements 15-minute cooldown

4. **Resource Management Actions**
   - UI-triggered restart/stop commands
   - Sandboxed execution with audit logging
   - Real-time status updates

5. **Configuration Management**
   - UI forms update PostgreSQL
   - Live reload of alert rules
   - No restart required

## File Structure

```
analytics-dashboard/
├── .vrooli/
│   └── service.json                     # Resource monitoring configuration
├── initialization/
│   ├── automation/
│   │   ├── n8n/
│   │   │   ├── resource-monitor.json    # Main monitoring workflow
│   │   │   ├── auto-discovery.json      # Resource discovery
│   │   │   └── alert-handler.json       # Alert processing
│   │   └── node-red/
│   │       └── metrics-collector.json   # Real-time metrics
│   ├── configuration/
│   │   ├── monitor-config.json          # Monitoring thresholds
│   │   ├── resource-registry.json       # Discovered resources
│   │   ├── alert-rules.json             # Alert conditions
│   │   └── vault-mapping.json           # Vault secret paths
│   ├── storage/
│   │   ├── postgres/
│   │   │   ├── schema.sql               # Resource registry tables
│   │   │   └── seed.sql                 # Default rules
│   │   └── questdb/
│   │       └── tables.sql               # Time-series tables
│   └── ui/
│       └── windmill/
│           ├── dashboard-app.json       # Main dashboard
│           ├── config-panel.json        # Configuration UI
│           └── docs-page.json           # Documentation
├── deployment/
│   ├── startup.sh                      # Initialize monitoring
│   └── discover.sh                     # Run auto-discovery
├── docs/
│   └── README.md                       # User documentation
└── test.sh                             # Integration tests
```

## Implementation Steps

### Phase 1: Core Infrastructure
1. ✅ Store implementation plan
2. Update service.json with monitoring focus
3. Create PostgreSQL schema for resource registry
4. Create QuestDB tables for metrics

### Phase 2: Monitoring Workflows
5. Create n8n resource monitoring workflow
6. Create n8n auto-discovery workflow
7. Create n8n alert handler with throttling
8. Create Node-RED metrics collector

### Phase 3: User Interface
9. Create Windmill dashboard UI
10. Create configuration panel
11. Create documentation page

### Phase 4: Configuration & Deployment
12. Create configuration files
13. Create deployment scripts
14. Create test script

### Phase 5: Documentation
15. Create comprehensive README

## Success Criteria
- Auto-discovers all running Vrooli resources
- Monitors resource health every 30 seconds
- Sends throttled SMS alerts for critical issues
- Provides one-click restart/stop capabilities
- Stores configurable retention of metrics in QuestDB (default 30 days)
- Configurable alert rules via UI
- Complete documentation accessible in-app

## Security Considerations
- All credentials stored in Vault
- Role-based access for management actions
- Sandboxed command execution
- Comprehensive audit logging
- No plaintext secrets in configs

## Performance Targets
- Sub-second dashboard updates
- <100ms health check response
- Support for 50+ monitored resources
- Configurable metric retention with multi-level aggregation (5min/hourly/daily)
- 99.9% monitoring uptime