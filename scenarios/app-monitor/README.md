# App Monitor

## Purpose
App Monitor provides a centralized dashboard for monitoring and managing all running Vrooli scenarios. It acts as a critical infrastructure component that ensures the health, performance, and availability of deployed scenarios.

## How It Helps Vrooli
- **Self-Monitoring**: Enables Vrooli to track its own application ecosystem
- **Auto-Recovery**: Automatically restarts unhealthy apps using shared workflows
- **Performance Intelligence**: Uses AI to analyze performance issues and suggest optimizations
- **Cross-Scenario Support**: Other scenarios can query app health status via CLI/API
- **Resource Efficiency**: Identifies resource-hungry apps for optimization

## Dependencies
- **Core Resources**: PostgreSQL (metrics storage), Redis (real-time events)
- **Automation**: n8n (health checks, auto-restart), Node-RED (Docker monitoring), Windmill (dashboards)
- **AI**: Ollama (performance analysis via shared workflow)
- **Shared Workflows**: `ollama.json`, `cache-manager.json`, `rate-limiter.json`

## Features
- Real-time health monitoring with customizable alert thresholds
- Performance metrics collection (CPU, memory, response times, error rates)
- AI-powered root cause analysis for unhealthy applications
- Automatic restart capability for failed apps
- Docker container monitoring via Node-RED
- Interactive Windmill dashboards for visual monitoring
- CLI for scriptable health checks

## API Endpoints
- `GET /health` - API health check
- `GET /api/apps` - List all monitored applications
- `GET /api/apps/:id/health` - Get specific app health status
- `POST /api/apps/:id/restart` - Trigger app restart
- `GET /api/docker/info` - Docker daemon information
- `GET /api/metrics/:id` - Get app performance metrics

## CLI Commands
```bash
app-monitor list              # List all monitored apps
app-monitor health <app-id>   # Check specific app health
app-monitor restart <app-id>  # Restart an application
app-monitor metrics <app-id>  # Get performance metrics
app-monitor analyze <app-id>  # Run AI performance analysis
```

## UI Style
Clean, professional monitoring dashboard with real-time updates. Dark theme with status-based color coding (green=healthy, yellow=degraded, red=unhealthy). Includes charts for metrics visualization and a grid layout for multiple app monitoring.

## Integration Points
- **system-monitor**: Can be queried for overall system health
- **app-debugger**: Provides health data for debugging
- **deployment-manager**: Checks app health before deployments
- **agent-dashboard**: Includes app health in agent status views

## Database Schema
- `apps` - Application registry
- `app_health_status` - Current health states
- `app_metrics` - Performance metrics time series
- `app_incidents` - Historical incident tracking
- `app_recommendations` - AI-generated optimization suggestions