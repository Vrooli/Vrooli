# PostgreSQL Storage Resource

A managed PostgreSQL resource that enables isolated database instances for client-specific development and multi-tenant deployments in Vrooli's platform factory architecture.

## 🎯 Purpose

Unlike Vrooli's core PostgreSQL infrastructure (which stores platform data), this resource provides:

- **Client Isolation**: Separate database instances for each client project
- **Parallel Development**: Work on multiple client solutions simultaneously
- **Template-based Configuration**: Quick setup with predefined configurations
- **Deployment Packaging**: Bundle database configuration with client solutions
- **Schema Management**: Per-instance migrations and seeding

## 🚀 Quick Start

### Install the Resource

```bash
# Install PostgreSQL resource (pulls Docker image, creates directories)
./manage.sh --action install
```

### Create a Client Instance

```bash
# Create instance for real estate client
./manage.sh --action create \
  --instance real-estate \
  --template production \
  --port 5434

# Create instance with auto-assigned port
./manage.sh --action create \
  --instance ecommerce \
  --template development
```

### Instance Management

```bash
# List all instances
./manage.sh --action list

# Check instance status
./manage.sh --action status --instance real-estate

# Get connection string
./manage.sh --action connect --instance real-estate

# Start/stop instances
./manage.sh --action start --instance real-estate
./manage.sh --action stop --instance real-estate

# Start all instances
./manage.sh --action start --instance all
```

## 🏗️ Architecture

### Instance Isolation
Each PostgreSQL instance runs in its own Docker container with:
- Unique port allocation (5433-5499 range)
- Separate data volume
- Independent credentials
- Isolated network namespace (optional)

### Templates
Pre-configured PostgreSQL settings for different use cases:
- `development` - Fast startup, verbose logging (default)
- `production` - Optimized for performance
- `testing` - Ephemeral, in-memory focused
- `minimal` - Low resource usage

## 📦 Client Packaging

When packaging a Vrooli instance for client deployment:

1. Export instance configuration:
   ```bash
   # Get connection details for packaging
   ./manage.sh --action connect --instance real-estate
   ```

2. Include in client's resource configuration:
   ```json
   {
     "storage": {
       "postgres": {
         "enabled": true,
         "instances": {
           "main": {
             "port": 5432,
             "template": "production",
             "migrations": "./migrations",
             "seeds": "./seeds"
           }
         }
       }
     }
   }
   ```

## 🔧 Advanced Usage

### Multi-tenant Development
```bash
# Create instances for multiple clients
for client in real-estate ecommerce healthcare; do
  ./manage.sh --action create --instance "$client"
done

# Start all instances
./manage.sh --action start --instance all
```

### A/B Testing Schemas
```bash
# Create two instances with different configurations
./manage.sh --action create --instance design-a --template development
./manage.sh --action create --instance design-b --template production

# Compare performance and behavior
```

### Monitoring and Diagnostics
```bash
# Monitor all instances continuously
./manage.sh --action monitor

# Run diagnostic checks
./manage.sh --action diagnose

# View instance logs
./manage.sh --action logs --instance real-estate --lines 100
```

## 📋 Complete Command Reference

### Resource Management
```bash
# Install/uninstall resource
./manage.sh --action install
./manage.sh --action uninstall

# Upgrade PostgreSQL Docker image
./manage.sh --action upgrade
```

### Instance Lifecycle
```bash
# Create new instance
./manage.sh --action create --instance <name> [--port <port>] [--template <template>]

# Destroy instance (with confirmation)
./manage.sh --action destroy --instance <name>

# Force destroy without confirmation
./manage.sh --action destroy --instance <name> --force yes
```

### Instance Operations
```bash
# Start/stop/restart instances
./manage.sh --action start --instance <name|all>
./manage.sh --action stop --instance <name|all>
./manage.sh --action restart --instance <name|all>
```

### Information and Monitoring
```bash
# Show resource or instance status
./manage.sh --action status [--instance <name>]

# List all instances
./manage.sh --action list

# Get connection details
./manage.sh --action connect --instance <name>

# Show container logs
./manage.sh --action logs --instance <name> [--lines <num>]

# Continuous monitoring
./manage.sh --action monitor [--interval <seconds>]

# Run diagnostics
./manage.sh --action diagnose
```

## 🔌 Connection Examples

### Using psql
```bash
# Get connection details first
./manage.sh --action connect --instance real-estate

# Connect using psql
psql -h localhost -p 5434 -U vrooli -d vrooli_client
```

### Using Connection String
```bash
# Get full connection string
CONN_STRING=$(./manage.sh --action connect --instance real-estate | grep "postgresql://")

# Use in your application
export DATABASE_URL="$CONN_STRING"
```

### Using Environment Variables
```bash
# Extract connection details
INSTANCE="real-estate"
DB_HOST="localhost"
DB_PORT=$(./manage.sh --action connect --instance $INSTANCE | grep "Port:" | cut -d' ' -f2)
DB_USER="vrooli"
DB_NAME="vrooli_client"
DB_PASS=$(./manage.sh --action connect --instance $INSTANCE | grep "Password:" | cut -d' ' -f2)
```

## ⚠️ Important Notes

- This resource is for **development and client-specific** databases
- Do NOT use for Vrooli's core platform data
- Instances are isolated but share host resources
- Default port range: 5433-5499 (max 67 instances)
- Maximum 20 instances can run simultaneously (configurable)

## 🚫 Common Pitfalls

- Don't confuse with core PostgreSQL (port 5432)
- Remember to backup before destroying instances
- Check port availability before creating instances
- Use appropriate templates for use case
- Instance names can only contain letters, numbers, hyphens, and underscores

## 🎯 Configuration Templates

### Development Template
- Verbose logging (`log_statement = all`)
- Fast startup optimizations
- Development-friendly settings
- Best for: Local development, debugging

### Production Template
- Performance optimizations
- Production-ready settings
- Conservative resource usage
- Best for: Client deployments, staging

### Testing Template
- Fastest possible operations
- Reduced durability for speed
- Ephemeral optimizations
- Best for: Automated testing, CI/CD

### Minimal Template
- Lowest resource consumption
- Minimal connection limits
- Basic functionality only
- Best for: Resource-constrained environments

## 🔧 Troubleshooting

### Instance Won't Start
```bash
# Check diagnostics
./manage.sh --action diagnose

# Check logs for errors
./manage.sh --action logs --instance <name> --lines 100

# Verify port availability
./manage.sh --action status --instance <name>
```

### Port Conflicts
```bash
# List current instances and ports
./manage.sh --action list

# Create instance with specific port
./manage.sh --action create --instance <name> --port <available_port>
```

### Resource Issues
```bash
# Check disk space and system health
./manage.sh --action diagnose

# Monitor resource usage
./manage.sh --action monitor
```

### Connection Issues
```bash
# Verify instance is running and healthy
./manage.sh --action status --instance <name>

# Get current connection details
./manage.sh --action connect --instance <name>

# Test connection
psql -h localhost -p <port> -U vrooli -d vrooli_client -c "SELECT version();"
```

## 🏷️ Version Information

- **PostgreSQL Version**: 16-alpine (Docker image)
- **Port Range**: 5433-5499
- **Max Instances**: 20
- **Network**: vrooli-network
- **Data Persistence**: Docker volumes

## 🔗 Integration

This PostgreSQL resource integrates with:
- Vrooli resource discovery system
- Vrooli configuration management
- Docker networking
- Port registry system

For more information, see the [Vrooli Resources Documentation](../../README.md).

---

**Need Help?**
- Use `./manage.sh --help` for command-line help
- Check the diagnostics: `./manage.sh --action diagnose`
- View instance logs: `./manage.sh --action logs --instance <name>`