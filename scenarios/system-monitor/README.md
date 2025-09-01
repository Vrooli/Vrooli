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
- N8n: Workflow automation
- Ollama: AI analysis (llama3.2:3b)
- Grafana (optional): Advanced visualization

### Shared Workflows
- `ollama`: AI model execution
- `rate-limiter`: API throttling
- `structured-data-extractor`: Data parsing

## Components
- **API**: Go-based REST API on port 8083
- **UI**: Dark-themed JavaScript dashboard on port 3003
- **CLI**: `system-monitor` command for terminal access

## Workflows
1. **threshold-monitor**: Checks system metrics against thresholds every minute
2. **scheduled-reports**: Generates comprehensive daily reports
3. **anomaly-investigator**: AI-powered root cause analysis for anomalies

## UI Style
Dark green "Matrix" aesthetic with real-time updating metrics, animated backgrounds, and cyberpunk-inspired visualizations.

## Usage
```bash
# Start the app
vrooli scenario run system-monitor


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