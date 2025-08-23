# PostgreSQL Storage Resource

A managed PostgreSQL resource that enables isolated database instances for client-specific development and multi-tenant deployments in Vrooli's platform factory architecture.

## 📋 Quick Reference

| Task | Command |
|------|---------|
| Create new client DB | `./manage.sh --action create --instance client-name` |
| Start instance | `./manage.sh --action start --instance client-name` |
| Get connection info | `./manage.sh --action connect --instance client-name` |
| Open GUI | `./manage.sh --action gui --instance client-name` |
| Create database | `./manage.sh --action create-db --instance client-name --database db_name` |
| Backup instance | `./manage.sh --action backup --instance client-name` |
| List all instances | `./manage.sh --action list` |
| Check health | `./manage.sh --action status --instance client-name` |
| View logs | `./manage.sh --action logs --instance client-name` |
| Stop instance | `./manage.sh --action stop --instance client-name` |

## 🎯 Purpose

Unlike Vrooli's core PostgreSQL infrastructure (which stores platform data), this resource provides:

- **Client Isolation**: Separate database instances for each client project
- **Parallel Development**: Work on multiple client solutions simultaneously
- **Template-based Configuration**: Quick setup with predefined configurations
- **Web GUI Management**: Integrated pgweb interface for visual database management
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

### GUI Management

```bash
# Start web GUI for an instance
./manage.sh --action gui --instance real-estate

# Start GUI on specific port
./manage.sh --action gui --instance real-estate --gui-port 8085

# Check GUI status
./manage.sh --action gui-status --instance real-estate

# List all running GUIs
./manage.sh --action gui-list

# Stop GUI for an instance
./manage.sh --action gui-stop --instance real-estate
```

## 🏗️ Architecture

### Instance Isolation
Each PostgreSQL instance runs in its own Docker container with:
- Unique port allocation (5433-5499 range)
- Separate data volume
- Independent credentials
- Isolated network namespace (optional)

### GUI Containers
Each instance can have an optional pgweb GUI container:
- One GUI per PostgreSQL instance
- Port range: 8080-8099 (auto-assigned)
- Same Docker networks as associated PostgreSQL instance
- Web-based database management interface

### Templates
Pre-configured PostgreSQL settings for different use cases:
- `development` - Fast startup, verbose logging, Docker networking enabled (default)
- `production` - Optimized for performance, Docker networking enabled
- `testing` - Ephemeral, in-memory focused, Docker networking enabled
- `minimal` - Low resource usage, Docker networking enabled

All templates include `listen_addresses = '*'` for proper Docker container communication.

## 📦 Client Packaging

When packaging a Vrooli instance for client deployment:

1. Export instance configuration:
   ```bash
   # Get connection details for packaging
   ./manage.sh --action connect --instance real-estate --format json > client-db-config.json
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

3. Package with client-specific data:
   ```bash
   # Backup client database
   ./manage.sh --action backup --instance real-estate --backup-name client-data
   
   # Include in deployment package
   cp -r ~/.vrooli/backups/postgres/real-estate/client-data ./package/
   ```

## 🎯 Real-World Example: Upwork Project Workflow

### Scenario: Building AI-Powered Real Estate Lead Generation
```bash
# 1. Create isolated database for real estate client
./manage.sh --action create --instance real-estate-leads --template production

# 2. Initialize migration system
./manage.sh --action migrate-init --instance real-estate-leads

# 3. Create project-specific databases
./manage.sh --action create-db --instance real-estate-leads --database lead_tracking
./manage.sh --action create-db --instance real-estate-leads --database property_data
./manage.sh --action create-db --instance real-estate-leads --database analytics

# 4. Create users with specific permissions
./manage.sh --action create-user --instance real-estate-leads \
  --username lead_api_user --password $(openssl rand -base64 32)

./manage.sh --action create-user --instance real-estate-leads \
  --username analytics_user --password $(openssl rand -base64 32)

# 5. Connect to n8n for workflow automation
# n8n can now access the database at: vrooli-postgres-real-estate-leads:5432

# 6. Start GUI for development
./manage.sh --action gui --instance real-estate-leads

# 7. Monitor during development
./manage.sh --action monitor --interval 10

# 8. Package for client deployment
./manage.sh --action backup --instance real-estate-leads --backup-name final-deploy
./manage.sh --action connect --instance real-estate-leads --format json > db-config.json
```

### Working on Multiple Clients Simultaneously
```bash
# Client A: Real Estate Automation
./manage.sh --action create --instance client-a-realestate --template production

# Client B: E-commerce Analytics  
./manage.sh --action create --instance client-b-ecommerce --template production

# Client C: Healthcare Dashboard
./manage.sh --action create --instance client-c-healthcare --template production

# Start all client databases
./manage.sh --action multi-start --instance "client-"

# Monitor all clients
./manage.sh --action multi-status --instance "client-"

# Backup all client data nightly
./manage.sh --action multi-backup --instance "client-"
```

## 🔧 Advanced Usage

### Multi-tenant Development
```bash
# Create instances for multiple clients
for client in real-estate ecommerce healthcare; do
  ./manage.sh --action create --instance "client-$client"
done

# Start all instances
./manage.sh --action start --instance all

# Start only client instances using pattern matching
./manage.sh --action multi-start --instance "client-*"

# Check status of all client instances
./manage.sh --action multi-status --instance "client-"

# Backup all test instances
./manage.sh --action multi-backup --instance "*-test"

# Stop all development instances
./manage.sh --action multi-stop --instance "*-dev"
```

#### Pattern Matching Support
Multi-instance commands support several pattern types:
- **Exact match**: `--instance myinstance`
- **Prefix match**: `--instance client-` (matches client-realestate, client-ecommerce, etc.)
- **Wildcard patterns**: `--instance "*-test"` or `--instance "client-*-prod"`
- **Comma-separated**: `--instance "instance1,instance2,instance3"`
- **All instances**: `--instance all`

### A/B Testing Schemas
```bash
# Create two instances with different configurations
./manage.sh --action create --instance design-a --template development
./manage.sh --action create --instance design-b --template production

# Start GUIs for easy comparison
./manage.sh --action gui --instance design-a    # Available at http://localhost:8080
./manage.sh --action gui --instance design-b    # Available at http://localhost:8081

# Compare performance and behavior using both GUI and command line
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

### GUI Management
```bash
# Start GUI for instance
./manage.sh --action gui --instance <name> [--gui-port <port>]

# Show GUI status
./manage.sh --action gui-status --instance <name>

# List all GUI instances
./manage.sh --action gui-list

# Stop GUI for instance
./manage.sh --action gui-stop --instance <name>
```

### Database Operations
```bash
# Create a new database within an instance
./manage.sh --action create-db --instance <name> --database <db_name>

# Drop a database (use --yes yes to skip confirmation)
./manage.sh --action drop-db --instance <name> --database <db_name> [--yes yes]

# Create a new user
./manage.sh --action create-user --instance <name> --username <username> --password <password>

# Drop a user (use --yes yes to skip confirmation)
./manage.sh --action drop-user --instance <name> --username <username> [--yes yes]

# Show database statistics
./manage.sh --action db-stats --instance <name> [--database <db_name>]
```

### Migration Management
```bash
# Initialize migration system for an instance
./manage.sh --action migrate-init --instance <name>

# Run migrations from a directory
./manage.sh --action migrate --instance <name> --migrations-dir <path>

# Check migration status and history
./manage.sh --action migrate-status --instance <name>

# Rollback a specific migration
./manage.sh --action migrate-rollback --instance <name> --migration <version>

# List available migrations in a directory
./manage.sh --action migrate-list --migrations-dir <path>

# Seed database with initial data
./manage.sh --action seed --instance <name> --seed-path <file_or_dir> [--database <db_name>]
```

**📚 Migration Examples Available**: See `examples/migrations/` for:
- User authentication tables with proper indexes
- Role-Based Access Control (RBAC) implementation  
- User preferences with JSONB storage
- Rollback script examples
- Best practices documentation in `examples/migrations/README.md`

### Backup and Restore
```bash
# Create a backup (full, schema-only, or data-only)
./manage.sh --action backup --instance <name> [--backup-name <name>] [--backup-type <full|schema|data>]

# Restore from backup
./manage.sh --action restore --instance <name> --backup-name <backup_name> [--yes yes]

# List available backups
./manage.sh --action list-backups --instance <name>

# Verify backup integrity
./manage.sh --action verify-backup --instance <name> --backup-name <backup_name>

# Delete a specific backup
./manage.sh --action delete-backup --instance <name> --backup-name <backup_name> [--yes yes]

# Clean up old backups (default: keep 7 days)
./manage.sh --action cleanup-backups --instance <name> [--retention-days <days>]
```

### Monitoring and Diagnostics
```bash
# Run comprehensive diagnostics
./manage.sh --action diagnose

# Monitor instances continuously (default: 5-second interval)
./manage.sh --action monitor [--interval <seconds>]
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

### Using Web GUI
```bash
# Start GUI for easy database management
./manage.sh --action gui --instance real-estate

# GUI will be available at http://localhost:8080 (or next available port)
# Check exact URL and port
./manage.sh --action gui-status --instance real-estate

# Access via browser - no additional setup required
# Web interface provides:
# - Table browsing and editing
# - SQL query execution
# - Database schema visualization
# - Export functionality (CSV, JSON, XML)
```

## ⚠️ Important Notes

- This resource is for **development and client-specific** databases
- Do NOT use for Vrooli's core platform data
- Instances are isolated but share host resources
- Default port range: 5433-5499 (max 67 instances)  
- GUI port range: 8080-8099 (max 20 GUIs)
- Maximum 67 instances can run simultaneously (configurable)

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
- Docker networking enabled (`listen_addresses = '*'`)
- Best for: Local development, debugging, GUI access

### Production Template
- Performance optimizations
- Production-ready settings
- Conservative resource usage
- Docker networking enabled (`listen_addresses = '*'`)
- Best for: Client deployments, staging, automation workflows

### Testing Template
- Fastest possible operations
- Reduced durability for speed (`fsync = off`)
- Ephemeral optimizations
- Docker networking enabled (`listen_addresses = '*'`)
- Best for: Automated testing, CI/CD, GUI testing

### Minimal Template
- Lowest resource consumption
- Minimal connection limits
- Basic functionality only
- Docker networking enabled (`listen_addresses = '*'`)
- Best for: Resource-constrained environments, development GUI access

**Note**: All templates include `listen_addresses = '*'` to enable proper communication with GUI containers, n8n workflows, Node-RED flows, and other Docker-based automation tools.

## 🐛 Known Issues & Workarounds

### Permission Issues (Fixed in Latest Version)
- **Previous Issue**: Instance directories created by Docker had root ownership, causing permission errors during destroy
- **Solution**: The destroy action now uses Docker to clean up root-owned files automatically
- **Manual Cleanup**: If you still encounter issues: `sudo rm -rf ~/.vrooli/resources/postgres/instances/<instance-name>`

### Pattern Matching in Multi-Instance Commands
- **Feature**: Multi-instance commands support pattern matching:
  - Prefix matching: `--instance "client-"` matches all instances starting with "client-"
  - Wildcard patterns: `--instance "*-test"` matches all instances ending with "-test"
  - Exact match: `--instance "myinstance"` matches only that specific instance
  - Comma-separated: `--instance "instance1,instance2,instance3"`
- **Example**: `./manage.sh --action multi-status --instance "client-*"`

### GUI Start Feedback
- **Enhanced**: GUI start now shows clear connection details and progress indicators
- **Example Output**:
  ```
  [INFO]    Waiting for pgweb to start...
  ........
  [SUCCESS] ✅ pgweb GUI started successfully
  
  GUI Access Information:
  ========================
  URL: http://localhost:8080
  Instance: client-test
  Database: vrooli_client
  Username: vrooli
  Password: <your-password>
  
  Note: The GUI is already connected. Just open the URL in your browser!
  ```

### Command Line Arguments
- **Format**: Always use `--parameter value` format (not `--parameter=value`)
- **Boolean Flags**: Use `--yes yes` or `--force yes` (not just `--yes`)
- **Example**: `./manage.sh --action destroy --instance test --yes yes`

## 🔧 Troubleshooting

### Instance Won't Start
```bash
# Check diagnostics first
./manage.sh --action diagnose

# Check logs for specific errors
./manage.sh --action logs --instance <name> --lines 100

# Verify port availability
./manage.sh --action status --instance <name>

# Common causes:
# - Port already in use: Choose different port or stop conflicting service
# - Docker permission issues: Ensure user is in docker group
# - Resource constraints: Check disk space and memory
```

### Port Conflicts
```bash
# List current instances and ports
./manage.sh --action list

# Find what's using a port
sudo lsof -i :5434
# or
sudo netstat -tulpn | grep :5434

# Create instance with specific available port
./manage.sh --action create --instance <name> --port <available_port>

# Auto-assign port (recommended)
./manage.sh --action create --instance <name>  # Omit --port
```

### Resource Issues
```bash
# Check disk space and system health
./manage.sh --action diagnose

# Monitor resource usage in real-time
./manage.sh --action monitor --interval 2

# Check Docker disk usage
docker system df

# Clean up unused Docker resources
docker system prune -a  # Use with caution!
```

### Connection Issues
```bash
# Verify instance is running and healthy
./manage.sh --action status --instance <name>

# Get current connection details
./manage.sh --action connect --instance <name>

# Test connection with Docker (no psql needed)
docker exec vrooli-postgres-<instance-name> psql -U vrooli -d vrooli_client -c "SELECT version();"

# Check network connectivity (from other containers)
docker exec n8n ping -c 1 vrooli-postgres-<instance-name>

# Common issues:
# - Using external port (5434) instead of internal (5432) in Docker networks
# - Wrong hostname: Use vrooli-postgres-<instance> not localhost from containers
# - Password authentication failed: Check credentials in connection string
```

### GUI Issues
```bash
# GUI won't start - check PostgreSQL instance first
./manage.sh --action status --instance <name>

# GUI shows "connection refused"
# 1. Ensure PostgreSQL instance is running
./manage.sh --action restart --instance <name>
# 2. Wait for PostgreSQL to be ready
sleep 5
# 3. Start GUI
./manage.sh --action gui --instance <name>

# GUI port conflicts
./manage.sh --action gui-list  # Check used ports
./manage.sh --action gui --instance <name> --gui-port 8085  # Use specific port

# GUI container issues
docker logs vrooli-pgweb-<instance-name>  # Check GUI container logs

# Reset GUI completely
./manage.sh --action gui-stop --instance <name>
docker rm -f vrooli-pgweb-<instance-name> 2>/dev/null  # Force remove if stuck
./manage.sh --action gui --instance <name>

# GUI not showing data
# - Clear browser cache
# - Try incognito/private mode
# - Check browser console for errors
```

### Instance Shows as "Missing"
```bash
# This means directory exists but container doesn't

# Option 1: Recreate the container
./manage.sh --action start --instance <name>

# Option 2: Clean up and recreate
./manage.sh --action destroy --instance <name> --yes yes
./manage.sh --action create --instance <name>

# Option 3: Manually check and clean
ls -la instances/<name>/  # Check if directory exists
docker ps -a | grep vrooli-postgres-<name>  # Check container
```

### Database Operations Failing
```bash
# "Database already exists" error
# List existing databases
docker exec vrooli-postgres-<instance> psql -U vrooli -c "\l"

# "User already exists" error  
# List existing users
docker exec vrooli-postgres-<instance> psql -U vrooli -c "\du"

# Permission denied errors
# Ensure you're using the correct user
./manage.sh --action connect --instance <name>  # Check credentials
```

### Integration Issues with Automation Tools
```bash
# n8n cannot connect
# 1. Verify instance is on n8n-network
docker inspect vrooli-postgres-<instance> | grep -A 10 Networks

# 2. Use internal hostname and port 5432 (not external port)
# Correct: vrooli-postgres-<instance>:5432
# Wrong: localhost:5439

# 3. Get proper connection format
./manage.sh --action connect --instance <name> --format n8n

# Node-RED connection issues
# Install PostgreSQL node first:
# cd ~/.node-red && npm install node-red-contrib-postgresql
# Then restart Node-RED
```

### Docker Network Issues
```bash
# Instance not joining networks
# Check if network exists
docker network ls | grep n8n-network

# Manually connect if needed
docker network connect n8n-network vrooli-postgres-<instance>

# Verify connection
docker exec n8n ping -c 1 vrooli-postgres-<instance>
```

### Migration Issues
```bash
# Migration system not initialized
./manage.sh --action migrate-init --instance <name>

# Migrations failing
# Check migration file syntax
./manage.sh --action migrate-list --migrations-dir <path>

# Rollback if needed
./manage.sh --action migrate-status --instance <name>
./manage.sh --action migrate-rollback --instance <name> --migration <version>
```

## 🏷️ Version Information

- **PostgreSQL Version**: 16-alpine (Docker image)
- **GUI Version**: pgweb latest (sosedoff/pgweb:latest)
- **PostgreSQL Port Range**: 5433-5499
- **GUI Port Range**: 8080-8099
- **Max Instances**: 67 (PostgreSQL), 20 (GUI)
- **Network**: vrooli-network
- **Data Persistence**: Docker volumes

## 🎯 Multi-Client Workflow Guide

### Upwork Automation Use Case

The PostgreSQL resource is designed to support Vrooli's **platform factory** approach - creating customized automation solutions for different clients with complete data isolation.

#### Complete Client Workflow

```bash
# 1. Analyze Upwork job requirements
# Example: "Real estate lead generation and CRM automation"

# 2. Create isolated client database
./manage.sh --action create --instance real-estate-client --template real-estate

# 3. Seed with industry-specific data
./manage.sh --action seed --instance real-estate-client --seed-path ./examples/seeds/real-estate/

# 4. Verify client setup
./manage.sh --action status --instance real-estate-client
./manage.sh --action connect --instance real-estate-client --format n8n

# 5. Build automation workflows (n8n, Node-RED, etc.)
# 6. Package for client deployment
```

#### Client Template Selection Guide

| Client Type | Template | Key Features | Typical Automation |
|-------------|----------|--------------|-------------------|
| **Real Estate** | `real-estate` | Property listings, lead management, agent CRM | Lead capture → qualification → nurturing → showing scheduling |
| **Ecommerce** | `ecommerce` | Product catalog, orders, inventory, customers | Order processing → inventory alerts → customer segmentation → abandoned cart recovery |
| **SaaS Business** | `saas` | Multi-tenant architecture, billing, feature flags | Usage tracking → billing automation → feature rollouts → support ticketing |
| **General Business** | `production` | High-performance, general-purpose | Custom workflows based on client requirements |

#### Data Isolation Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Vrooli Core Platform                     │
│                   (Internal PostgreSQL)                    │
└─────────────────────────────────────────────────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        │                     │                     │
┌───────▼────────┐   ┌────────▼────────┐   ┌───────▼────────┐
│ Client A       │   │ Client B        │   │ Client C       │
│ Real Estate    │   │ Ecommerce       │   │ SaaS Platform  │
│ Port: 5433     │   │ Port: 5434      │   │ Port: 5435     │
│ Networks:      │   │ Networks:       │   │ Networks:      │
│ • n8n-network  │   │ • n8n-network   │   │ • n8n-network  │
│ • browserless  │   │ • minio-network │   │ • node-red     │
│ • searxng      │   │ • qdrant        │   │ • minio        │
└────────────────┘   └─────────────────┘   └────────────────┘
```

#### Client Project Lifecycle

1. **Requirement Analysis**
   ```bash
   # Identify client industry and automation needs
   # Select appropriate template: real-estate, ecommerce, saas
   ```

2. **Database Setup**
   ```bash
   # Create isolated instance with industry template
   ./manage.sh --action create --instance client-name --template industry-type
   
   # Seed with sample data for testing
   ./manage.sh --action seed --instance client-name --seed-path ./examples/seeds/industry-type/
   ```

3. **Development Phase**
   ```bash
   # Build workflows using client-specific database
   # Test with seeded data
   # Develop custom migrations if needed
   ./manage.sh --action migrate --instance client-name --migrations-dir ./custom-migrations/
   ```

4. **Client Delivery**
   ```bash
   # Export client database for deployment
   ./manage.sh --action backup --instance client-name --backup-name client-deployment
   
   # Generate deployment package
   ./examples/package-deployment.sh --instance client-name --target production
   ```

#### Network Integration Benefits

Each client template automatically connects to relevant automation networks:

- **n8n-network**: Core workflow automation
- **node-red-network**: Real-time monitoring and dashboards  
- **browserless-network**: Web scraping and automation
- **minio-network**: File storage and document management
- **qdrant-network**: Vector search and AI features
- **searxng-network**: Privacy-respecting search integration

This enables **zero-configuration connectivity** between client databases and automation tools.

#### Scaling Considerations

- **Max Clients**: 67 simultaneous client databases per Vrooli instance
- **Resource Isolation**: Each client gets dedicated container, port, and data
- **Network Segmentation**: Clients can't access each other's data
- **Performance**: Templates optimized for specific workloads
- **Backup Strategy**: Individual client backups for deployment packaging

## 🔗 Integration with Automation Tools

### n8n Integration
```bash
# Create instance with n8n network access
./manage.sh --action create --instance client-project --template development

# Get n8n-formatted credentials
./manage.sh --action connect --instance client-project --format n8n

# In n8n PostgreSQL node, use:
# Host: vrooli-postgres-client-project
# Port: 5432 (internal Docker port)
# Database: vrooli_client
# User: vrooli
# Password: <from connection output>
```

### Node-RED Integration  
```bash
# Get Node-RED formatted connection
./manage.sh --action connect --instance client-project --format node-red

# Install node-red-contrib-postgresql in Node-RED
# Use internal hostname and port 5432
```

### Windmill Integration
```bash
# Windmill scripts can access databases using internal hostnames
# Connection URL: postgresql://vrooli:password@vrooli-postgres-client:5432/vrooli_client
```

### Cross-Resource Workflows
```bash
# Example: n8n → PostgreSQL → MinIO workflow
# 1. n8n webhook receives data
# 2. Process and store in PostgreSQL
# 3. Generate reports and store in MinIO
# 4. All resources on same Docker network
```

## 🔗 Technical Integration Details

This PostgreSQL resource integrates with:
- Vrooli resource discovery system
- Vrooli configuration management  
- Docker networking (automatic network joining)
- Port registry system (conflict prevention)
- pgweb for web-based database management
- n8n, Node-RED, and other automation tools via Docker networks

For more information, see the [Vrooli Resources Documentation](../../README.md).

---

**Need Help?**
- Use `./manage.sh --help` for command-line help
- Check the diagnostics: `./manage.sh --action diagnose`
- View instance logs: `./manage.sh --action logs --instance <name>`