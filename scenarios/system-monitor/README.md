# System Monitor

## Purpose
Real-time server monitoring with AI-driven anomaly investigation and automated reporting.

## Features
- **Real-time Metrics**: CPU, memory, disk, network monitoring
- **AI Anomaly Detection**: Automatic investigation of system anomalies using Ollama
- **Automated Reports**: Scheduled daily HTML reports with insights
- **Threshold Monitoring**: Configurable warning/critical thresholds
- **Dark Cyberpunk UI**: Matrix-inspired monitoring dashboard

## Dependencies
### Resources
- PostgreSQL: Metrics, thresholds, investigations storage
- QuestDB: Time-series performance data
- Redis: Real-time alerts and caching
- Node-RED: Workflow automation
- Ollama: AI analysis (llama3.2:3b)
- Grafana (optional): Advanced visualization

### Shared Workflows
- `metric-collector`: Node-RED flow for system telemetry
- `anomaly-detector`: Node-RED flow for trigger evaluation

## Components
- **API**: Go-based REST API (port assigned by lifecycle)
- **UI**: Dark-themed React dashboard (port assigned by lifecycle)
- **CLI**: `system-monitor` command for terminal access

## Workflows
1. **metric-collector**: Node-RED flow that gathers metrics and writes to storage
2. **anomaly-detector**: Node-RED flow that evaluates triggers and opens investigations
3. **scheduled-monitor.sh**: Automation script for periodic checks and reports

## UI Style
Dark green "Matrix" aesthetic with real-time updating metrics, animated backgrounds, and cyberpunk-inspired visualizations.

## Usage
```bash
# Start the app
make start

# Or with the lifecycle CLI
vrooli scenario start system-monitor


# CLI commands
system-monitor status
system-monitor metrics --last 1h
system-monitor investigate --anomaly-id <id>
```

## Integration Points
Other scenarios can leverage system-monitor for:
- Performance tracking
- Anomaly detection
- System health checks
- Resource usage monitoring
