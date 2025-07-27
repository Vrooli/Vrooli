# Windmill Workflow Automation Platform

Windmill is a developer-centric workflow automation platform that allows you to build workflows and UIs using code (TypeScript, Python, Go, Bash) instead of drag-and-drop interfaces.

## Overview

Windmill combines the power of:
- **Code-first approach**: Write workflows in TypeScript, Python, Go, or Bash
- **Built-in IDE**: Web-based editor with autocomplete and debugging
- **Scalable execution**: Multi-worker architecture for high performance
- **REST API**: Complete API for automation and integration
- **Real-time collaboration**: Multi-user support with version control

## Quick Start

### Installation

```bash
# Install with default settings (3 workers, internal database)
./manage.sh --action install

# Install with custom configuration
./manage.sh --action install --workers 5 --external-db yes --db-url "postgresql://user:pass@host:port/db"

# Install without Language Server Protocol (saves memory)
./manage.sh --action install --no-lsp
```

### Management Commands

```bash
# Service management
./manage.sh --action start              # Start services
./manage.sh --action stop               # Stop services
./manage.sh --action restart            # Restart services
./manage.sh --action status             # Show detailed status

# Worker management
./manage.sh --action scale-workers --workers 10     # Scale to 10 workers
./manage.sh --action restart-workers               # Restart all workers

# Monitoring and logs
./manage.sh --action logs                          # Show all logs
./manage.sh --action logs --service worker --follow # Follow worker logs

# Data management
./manage.sh --action backup                        # Create backup
./manage.sh --action restore --backup-path /path/to/backup.tar.gz

# Cleanup
./manage.sh --action uninstall                     # Complete removal
```

## Architecture

### Services

- **Windmill Server** (Port 5681): Web interface + REST API
- **PostgreSQL Database**: Persistent storage for all data
- **Worker Containers**: Scalable script execution (default: 3 workers)
- **Native Worker**: System command execution (bash, system tools)
- **Language Server**: IDE features and autocomplete (optional)
- **Multiplayer Service**: Real-time collaboration (Enterprise, optional)

### Storage

- **Docker Volumes**: Persistent storage for database and application data
- **Configuration**: Environment file at `docker/.env`
- **Backups**: Automated backup to `~/.windmill-backup/`

## Configuration

### Environment Variables

Key configuration options in `docker/.env`:

```bash
# Core settings
WINDMILL_SERVER_PORT=5681
WINDMILL_WORKER_REPLICAS=3
WINDMILL_WORKER_MEMORY_LIMIT=2048M

# Database (internal)
WINDMILL_DB_PASSWORD=your-secure-password

# Database (external)
WINDMILL_DB_EXTERNAL=yes
WINDMILL_DATABASE_URL=postgresql://user:pass@host:port/windmill

# Security
WINDMILL_SUPERADMIN_EMAIL=admin@windmill.dev
WINDMILL_SUPERADMIN_PASSWORD=changeme
WINDMILL_JWT_SECRET=your-jwt-secret-32-chars-minimum

# Features
WINDMILL_ENABLE_LSP=yes
WINDMILL_ENABLE_MULTIPLAYER=no
```

### Worker Scaling

Workers can be scaled dynamically based on workload:

- **Development**: 1-3 workers
- **Small production**: 3-10 workers
- **Large production**: 10+ workers
- **Rule of thumb**: 1 worker per CPU core

Each worker requires ~2GB RAM and supports multiple concurrent executions.

## Core Concepts: Scripts vs Flows vs Apps

Windmill provides three main ways to build automation and user interfaces:

### Scripts
**What they are**: Individual functions written in TypeScript, Python, Go, or Bash that perform specific tasks.

**When to use**:
- Single-purpose functions (e.g., send email, process data, call API)
- Building blocks for larger workflows
- Quick automation tasks
- Webhook handlers
- Scheduled jobs

**Example use cases**:
- Data transformation function
- API integration endpoint
- Database query executor
- File processing utility

### Flows
**What they are**: Visual workflows that chain multiple scripts together with control logic, branching, and error handling.

**When to use**:
- Complex multi-step processes
- Business logic requiring conditions and loops
- Orchestrating multiple API calls
- ETL pipelines
- Approval workflows

**Key features**:
- Visual flow editor
- Conditional branching (if/else)
- For loops and while loops
- Parallel execution
- Error handling and retries
- Input/output mapping between steps

**Example use cases**:
- User onboarding workflow (create account → send email → setup defaults)
- Data pipeline (extract → transform → validate → load)
- Approval process (submit → review → approve/reject → notify)
- Integration workflow (fetch from API A → process → send to API B)

### Apps
**What they are**: Low-code UI applications built with drag-and-drop components that can execute scripts and flows.

**When to use**:
- Creating user interfaces for non-technical users
- Building admin panels and dashboards
- Interactive forms and data entry
- Monitoring and reporting interfaces
- Internal tools

**Key features**:
- Drag-and-drop UI builder
- Pre-built components (forms, tables, charts)
- Data binding to scripts/flows
- Responsive design
- Real-time updates
- User authentication

**Example use cases**:
- Admin dashboard for user management
- Data entry form with validation
- Report viewer with export options
- Monitoring dashboard with live metrics
- Internal tool for customer support

### Choosing Between Scripts, Flows, and Apps

| Use Case | Best Choice | Why |
|----------|-------------|-----|
| Simple API call | Script | Single action, reusable |
| Multi-step process | Flow | Visual orchestration, error handling |
| User interface needed | App | Interactive UI with backend logic |
| Scheduled task | Script or Flow | Depends on complexity |
| Webhook handler | Script | Direct execution, fast response |
| ETL pipeline | Flow | Multiple steps, data transformation |
| Admin panel | App | UI for non-technical users |
| Microservice | Script | Single responsibility, API endpoint |

### Combining Scripts, Flows, and Apps

Windmill's power comes from combining these concepts:

1. **Scripts as Building Blocks**: Write reusable scripts for common tasks
2. **Flows for Orchestration**: Chain scripts together with business logic
3. **Apps for User Interaction**: Create UIs that trigger flows and display results

**Example Architecture**:
```
App (Customer Dashboard)
  ├── Flow: Load Customer Data
  │   ├── Script: Authenticate User
  │   ├── Script: Fetch from Database
  │   └── Script: Format for Display
  └── Flow: Update Customer
      ├── Script: Validate Input
      ├── Script: Update Database
      └── Script: Send Notification
```

### Programmatic App Management

Windmill provides API endpoints for programmatic app creation and management:

```bash
# List available app examples
./manage.sh --action list-apps

# Prepare app for manual import (includes JSON and instructions)
./manage.sh --action prepare-app --app-name admin-dashboard

# Deploy app via API (recommended)
./manage.sh --action deploy-app --app-name admin-dashboard --workspace demo

# Check API availability
./manage.sh --action check-app-api
```

**Available App Examples**:
- `admin-dashboard`: User management interface with metrics
- `data-entry-form`: Multi-step onboarding form with validation  
- `monitoring-dashboard`: Real-time system monitoring

**Note**: Some Windmill versions may have workspace constraint issues that prevent API deployment. If this occurs, the script will provide guidance for manual import through the UI.

## Usage

### Web Interface

1. **Access Windmill**: http://localhost:5681
2. **Login**: Use configured superadmin credentials
3. **Create Workspace**: Set up your first workspace
4. **Choose Your Tool**:
   - **Scripts**: For individual functions
   - **Flows**: For multi-step workflows
   - **Apps**: For user interfaces

### Supported Languages

#### TypeScript (Recommended)
```typescript
export function main(name: string): string {
    return `Hello ${name} from Windmill!`;
}
```

#### Python
```python
def main(name: str) -> str:
    return f"Hello {name} from Windmill!"
```

#### Go
```go
package main

func Main(name string) string {
    return fmt.Sprintf("Hello %s from Windmill!", name)
}
```

#### Bash
```bash
#!/bin/bash
name="$1"
echo "Hello $name from Windmill!"
```

### API Usage

Create an API token in the web interface (User Settings → Tokens):

```bash
# List workspaces
curl -H "Authorization: Bearer $TOKEN" \
     http://localhost:5681/api/w/list

# Execute script
curl -X POST \
     -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json" \
     -d '{"args":{"name":"World"}}' \
     http://localhost:5681/api/w/workspace/jobs/run/f/hello
```

## System Requirements

### Minimum Requirements
- **Memory**: 4GB RAM (2GB for workers + 1GB server + 1GB database)
- **Disk**: 5GB available space
- **CPU**: 2 cores (recommended: 1 worker per core)
- **Docker**: Docker Engine + Docker Compose v2

### Recommended Requirements
- **Memory**: 8GB+ RAM for production workloads
- **Disk**: 20GB+ for larger workspaces and job history
- **CPU**: 4+ cores for optimal performance
- **Network**: Stable internet connection for Hub scripts

## Features

### Workflow Capabilities
- **Triggers**: Webhook, Schedule, Manual execution
- **Flow Control**: Conditional logic, loops, parallel execution
- **Error Handling**: Try/catch blocks, retry policies
- **State Management**: Variables, resources, secrets

### Developer Experience
- **IDE Integration**: Built-in VS Code-like editor
- **IntelliSense**: Auto-completion for TypeScript/Python
- **Debugging**: Step-by-step execution, breakpoints
- **Version Control**: Git integration (Enterprise)
- **Testing**: Built-in test runner and validation

### Enterprise Features
- **SAML/OAuth**: Enterprise authentication
- **RBAC**: Role-based access control
- **Audit Logs**: Comprehensive activity tracking
- **High Availability**: Multi-region deployment
- **Premium Support**: Dedicated support channels

## Troubleshooting

### Common Issues

#### Services won't start
```bash
# Check system resources
./manage.sh --action status

# Check Docker status
docker ps -a
docker stats

# Check logs
./manage.sh --action logs
```

#### Port conflicts
```bash
# Use custom port
WINDMILL_CUSTOM_PORT=5682 ./manage.sh --action install

# Check port usage
sudo lsof -i :5681
```

#### Memory issues
```bash
# Reduce worker count
./manage.sh --action scale-workers --workers 1

# Check memory usage
docker stats
free -h
```

#### Database connectivity
```bash
# Check database status
./manage.sh --action logs --service db

# Test database connection
./manage.sh --action status
```

### Performance Optimization

1. **Worker Scaling**: Scale workers based on CPU cores and workload
2. **Memory Limits**: Adjust worker memory limits based on script requirements
3. **Database**: Use external PostgreSQL for production
4. **Monitoring**: Monitor resource usage with `docker stats`

### Getting Help

- **Documentation**: https://docs.windmill.dev
- **GitHub Issues**: https://github.com/windmill-labs/windmill/issues
- **Discord Community**: https://discord.gg/V7PM2YHsPB
- **Commercial Support**: https://windmill.dev/pricing

## Advanced Configuration

### External Database

For production deployments, use external PostgreSQL:

```bash
# Install with external database
./manage.sh --action install \
  --external-db yes \
  --db-url "postgresql://windmill:password@db.example.com:5432/windmill"
```

### Custom Docker Images

Override the default Windmill image:

```bash
# Set custom image
export WINDMILL_IMAGE="ghcr.io/windmill-labs/windmill:1.100.0"
./manage.sh --action install
```

### Resource Limits

Customize resource limits in `docker/.env`:

```bash
# Increase worker memory
WINDMILL_WORKER_MEMORY_LIMIT=4096M

# Scale workers
WINDMILL_WORKER_REPLICAS=8
```

### Security Hardening

1. **Change default passwords** immediately after installation
2. **Use strong JWT secrets** (32+ characters)
3. **Enable HTTPS** with reverse proxy (Caddy, nginx)
4. **Regular backups** with encryption
5. **Update images** regularly for security patches

## Development

### Local Development

For development and testing:

```bash
# Install in development mode
./manage.sh --action install --workers 1 --no-multiplayer

# Enable debug logging
WINDMILL_LOG_LEVEL=debug ./manage.sh --action restart
```

### Custom Scripts

Place example scripts in `examples/` directory:

- `examples/basic-typescript-script.ts`
- `examples/python-data-processing.py`
- `examples/webhook-trigger.json`
- `examples/api-integration.json`

Import these via the Windmill web interface after creating a workspace.

## License

Windmill is available under multiple licenses:
- **Community Edition**: AGPLv3 (open source)
- **Enterprise Edition**: Commercial license with additional features

See https://windmill.dev/pricing for details.

## Contributing

This Windmill resource implementation is part of the Vrooli project. For contributions:

1. Follow existing code patterns in other resource implementations
2. Add comprehensive BATS tests for new functionality
3. Update documentation for any configuration changes
4. Test thoroughly with different configurations

## Changelog

### v1.0.0 (Initial Implementation)
- Complete Docker Compose setup with multi-service architecture
- Configurable worker scaling and resource management
- Comprehensive management CLI with all operations
- External database support for production deployments
- Automated backup and restore functionality
- Full integration with Vrooli resource management system