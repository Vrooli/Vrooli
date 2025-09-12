# App Monitor

## Purpose
App Monitor provides a centralized dashboard for monitoring and managing all running Vrooli scenarios. It acts as a critical infrastructure component that ensures the health, performance, and availability of deployed scenarios.

## How It Helps Vrooli
- **Self-Monitoring**: Enables Vrooli to track its own application ecosystem
- **Auto-Recovery**: Provides restart capability for unhealthy apps
- **Performance Intelligence**: Uses AI to analyze performance issues and suggest optimizations
- **Cross-Scenario Support**: Other scenarios can query app health status via CLI/API
- **Resource Efficiency**: Identifies resource-hungry apps for optimization

## Dependencies
- **Core Resources**: PostgreSQL (metrics storage), Redis (real-time events)
- **AI**: Ollama (performance analysis)
- **Docker**: Direct Docker API integration for container monitoring
- **Orchestrator**: Optional integration with orchestrator status API

## Environment Variables
```bash
# Required
export API_PORT=21600           # API server port
export UI_PORT=21700            # UI server port

# Optional
export ORCHESTRATOR_STATUS_URL="http://localhost:9500/status"  # Orchestrator status endpoint
export POSTGRES_URL="postgres://..."                           # Database connection
export REDIS_URL="redis://localhost:6379"                     # Redis connection
export API_KEY="your-secret-key"                              # Optional API authentication
```

## Features
- Real-time health monitoring with customizable alert thresholds
- Performance metrics collection (CPU, memory, response times, error rates)
- AI-powered root cause analysis for unhealthy applications
- Restart capability for failed apps
- Direct Docker container monitoring
- Interactive dashboard for visual monitoring
- CLI for scriptable health checks

## API Endpoints

### Health Endpoints
- `GET /health` - API health check
- `GET /api/health` - Alternative health check

### App Management (v1)
- `GET /api/v1/apps` - List all monitored applications
- `GET /api/v1/apps/:id` - Get specific app information
- `POST /api/v1/apps/:id/start` - Start an application
- `POST /api/v1/apps/:id/stop` - Stop an application
- `GET /api/v1/apps/:id/logs` - Get application logs
- `GET /api/v1/apps/:id/logs/lifecycle` - Get lifecycle logs
- `GET /api/v1/apps/:id/logs/background` - Get background logs
- `GET /api/v1/apps/:id/metrics` - Get app performance metrics
- `GET /api/v1/logs/:appName` - Get logs by app name

### System Information
- `GET /api/v1/system/info` - Get system and orchestrator information
- `GET /api/v1/system/metrics` - Get system metrics
- `GET /api/v1/resources` - Get resource status

### Docker Integration
- `GET /api/v1/docker/info` - Docker daemon information
- `GET /api/v1/docker/containers` - List Docker containers

### Real-time Updates
- `GET /ws` - WebSocket endpoint for real-time updates

## CLI Commands
```bash
app-monitor status                  # Check App Monitor service health
app-monitor list                    # List all managed applications
app-monitor start <app-id>          # Start an application
app-monitor stop <app-id>           # Stop an application
app-monitor logs <app-id> [limit]   # Show application logs (default: 50)
app-monitor metrics <app-id> [hrs]  # Show application metrics (default: 24h)
app-monitor docker                  # Show Docker system information
app-monitor help                    # Show help message

# Environment Variables
export API_PORT=21600                                    # Required: API port
export APP_MONITOR_API_URL="http://localhost:$API_PORT" # Optional: API URL override
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