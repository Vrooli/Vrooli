# n8n Resource

n8n is a workflow automation platform that allows you to connect various services and automate tasks. This resource provides automated installation, configuration, and management of n8n for the Vrooli project with enhanced host system access.

## Overview

- **Category**: Automation
- **Default Port**: 5678
- **Service Type**: Docker Container
- **Container Name**: n8n
- **Documentation**: [docs.n8n.io](https://docs.n8n.io)

## Features

- 🔄 **Workflow Automation**: Visual workflow builder with 400+ integrations
- 💻 **Host Command Access**: Run system commands directly from n8n workflows
- 🐳 **Docker Integration**: Control other Docker containers from within n8n
- 📁 **Workspace Access**: Direct access to the Vrooli project files
- 🔧 **Enhanced Tools**: Pre-installed utilities for common automation tasks
- 🔐 **Basic Authentication**: Secure access with username/password
- 💾 **Database Options**: SQLite (default) or PostgreSQL
- 🌐 **Webhook Support**: External webhook endpoints for triggers
- 📊 **API Access**: Full REST API for programmatic control

## Prerequisites

- Docker installed and running
- 1GB+ RAM available
- 2GB+ free disk space
- Port 5678 available (or custom port)
- (Optional) PostgreSQL for production use

## Custom Setup Details

### Custom Docker Image
- Based on official n8n image with additional tools
- Seamless host command execution (no absolute paths needed)
- Pre-installed: bash, git, curl, wget, jq, python3, docker-cli
- Smart PATH handling for transparent host binary access

### Volume Mounts
- `/host/usr/bin`, `/host/bin`: Read-only access to system binaries
- `/host/home`: Read-write access to user home directory
- `/workspace`: Direct access to Vrooli project
- `/var/run/docker.sock`: Docker control capabilities

### Security Note
This setup prioritizes functionality over isolation. It's designed for development and trusted environments where n8n needs full system access.

## Installation

### Quick Start
```bash
# Install with custom image (recommended)
./manage.sh --action install --build-image yes

# Install with standard n8n image
./manage.sh --action install
```

### Installation Options
```bash
./manage.sh --action install \
  --build-image yes \           # Build custom image with host access
  --basic-auth yes \            # Enable authentication (default)
  --username admin \            # Set username (default: admin)
  --password YourPassword \     # Set password (auto-generated if not specified)
  --database postgres \         # Use PostgreSQL (default: sqlite)
  --webhook-url https://...     # Set external webhook URL
```

## Usage

### Management Commands
```bash
# Check status
./manage.sh --action status

# View logs
./manage.sh --action logs

# Stop n8n
./manage.sh --action stop

# Start n8n
./manage.sh --action start

# Restart n8n
./manage.sh --action restart

# Reset admin password
./manage.sh --action reset-password

# Uninstall n8n
./manage.sh --action uninstall

# Show API setup instructions
./manage.sh --action api-setup

# Save API key to configuration
./manage.sh --action save-api-key --api-key YOUR_API_KEY

# Execute workflow via API (smart detection of webhook vs manual trigger)
./manage.sh --action execute --workflow-id WORKFLOW_ID

# Execute webhook workflow with data
./manage.sh --action execute --workflow-id WORKFLOW_ID --data '{"key": "value"}'
```

### Smart Workflow Execution

The manage.sh script now intelligently handles different workflow types:

1. **Webhook Workflows**: Automatically detects webhook path and HTTP method, activates if needed, and executes via webhook URL
   - **Important**: For synchronous execution results, configure your webhook with:
     - Respond: "When Last Node Finishes"
     - Response Data: "Last Node"
2. **Manual Trigger Workflows**: Provides clear explanation that these cannot be executed via API and suggests alternatives
3. **Data Passing**: Supports passing JSON data to webhook workflows via the `--data` parameter

### Important Note: CLI Execute Bug

**⚠️ Known Issue**: The n8n CLI command `n8n execute --id` is broken in versions 1.93.0+ (GitHub issue #15567). 

**Workaround**: Use the manage.sh script with API-based execution:

```bash
# First-time setup
./manage.sh --action api-setup
# Follow instructions to create API key in web UI

# Save API key (recommended - persists across sessions)
./manage.sh --action save-api-key --api-key YOUR_API_KEY

# Alternative: Set API key temporarily
export N8N_API_KEY="your-api-key-here"

# Execute workflows (API key loaded automatically from config)
./manage.sh --action execute --workflow-id YOUR_WORKFLOW_ID
```

**API Key Storage**: The API key is saved securely in `~/.vrooli/resources.local.json` with 600 permissions, ensuring it persists across terminal sessions and system restarts.

### Using Execute Command Node

With the custom image, you can run commands naturally in the Execute Command node:

```bash
# System commands work without paths
tput bel                      # Terminal bell
ls ~/Vrooli                   # List Vrooli directory
whoami                        # Current user

# Docker commands
docker ps                     # List containers
docker logs container-name    # View container logs

# Python scripts
python3 /workspace/script.py  # Run Python scripts

# Git operations
cd /workspace && git status   # Check git status

# Direct file access
cat /host/home/.bashrc       # Read host files
echo "test" > /workspace/output.txt  # Write to workspace
```

### Workflow Examples

#### 1. System Monitoring
```javascript
// Execute Command node
ps aux | grep node
df -h
free -m
```

#### 2. Docker Management
```javascript
// Execute Command node
docker stats --no-stream
docker restart my-container
```

#### 3. File Processing
```javascript
// Execute Command node
find /workspace -name "*.log" -mtime -1
grep -r "ERROR" /workspace/logs/
```

## Programmatic Workflow Management

n8n provides extensive CLI and REST API capabilities for programmatic workflow creation, execution, and management. This enables automation of automation workflows themselves.

### CLI Commands

Access n8n CLI commands through the container:

#### Workflow Execution
```bash
# Execute a workflow by ID
docker exec n8n n8n execute --id=WORKFLOW_ID

# Execute with raw JSON output only
docker exec n8n n8n execute --id=WORKFLOW_ID --rawOutput
```

#### Workflow Management
```bash
# List all workflows
docker exec n8n n8n list:workflow

# List only active workflows with IDs only
docker exec n8n n8n list:workflow --active=true --onlyId

# Update workflow active status
docker exec n8n n8n update:workflow --id=WORKFLOW_ID --active=true
docker exec n8n n8n update:workflow --all --active=false
```

#### Import/Export Operations
```bash
# Export single workflow
docker exec n8n n8n export:workflow --id=WORKFLOW_ID --output=workflow.json --pretty

# Export all workflows
docker exec n8n n8n export:workflow --all --output=backup-dir/ --separate

# Create backup with formatted output
docker exec n8n n8n export:workflow --backup --output=backups/latest/

# Import workflow from file
docker exec n8n n8n import:workflow --input=workflow.json

# Import multiple workflows from directory
docker exec n8n n8n import:workflow --separate --input=backups/latest/
```

#### AI-Powered Workflow Creation
```bash
# Generate workflow from text prompt
docker exec n8n n8n ttwf:generate --prompt "Create a telegram chatbot that can tell current weather in Berlin" --output result.json

# Batch generate from dataset
docker exec n8n n8n ttwf:generate --input dataset.jsonl --output results.jsonl --limit 10 --concurrency 2
```

### REST API Access

n8n provides a comprehensive REST API for programmatic access. **API Key Required**: Create an API key through the web interface at Settings → n8n API.

#### Authentication Setup
1. Access n8n web interface: `http://localhost:5678`
2. Go to Settings → n8n API
3. Create API key with appropriate scopes
4. Use the key in API requests:

```bash
# Set API key as environment variable
export N8N_API_KEY="your-api-key-here"

# Use in curl requests
curl -H "X-N8N-API-KEY: $N8N_API_KEY" http://localhost:5678/api/v1/workflows
```

#### API Endpoints Examples
```bash
# List workflows
curl -H "X-N8N-API-KEY: $N8N_API_KEY" http://localhost:5678/api/v1/workflows

# Get specific workflow
curl -H "X-N8N-API-KEY: $N8N_API_KEY" http://localhost:5678/api/v1/workflows/WORKFLOW_ID

# Execute workflow
curl -X POST -H "X-N8N-API-KEY: $N8N_API_KEY" \
     -H "Content-Type: application/json" \
     http://localhost:5678/api/v1/workflows/WORKFLOW_ID/execute

# Create new workflow
curl -X POST -H "X-N8N-API-KEY: $N8N_API_KEY" \
     -H "Content-Type: application/json" \
     -d @workflow.json \
     http://localhost:5678/api/v1/workflows

# Update workflow active status
curl -X PATCH -H "X-N8N-API-KEY: $N8N_API_KEY" \
     -H "Content-Type: application/json" \
     -d '{"active": true}' \
     http://localhost:5678/api/v1/workflows/WORKFLOW_ID
```

### Complete Workflow Example

Here's a complete example demonstrating programmatic workflow creation and execution:

#### 1. Create Workflow JSON
```json
[{
  "name": "System Notification Workflow",
  "active": false,
  "nodes": [
    {
      "parameters": {
        "path": "notify",
        "responseMode": "lastNode",
        "options": {}
      },
      "type": "n8n-nodes-base.webhook",
      "typeVersion": 1.1,
      "position": [0, 0],
      "id": "webhook-trigger",
      "name": "Webhook"
    },
    {
      "parameters": {
        "command": "echo \"Workflow executed at $(date)\" > /host/home/workflow-notification.txt"
      },
      "type": "n8n-nodes-base.executeCommand",
      "typeVersion": 1,
      "position": [300, 0],
      "id": "create-notification",
      "name": "Create Notification File"
    }
  ],
  "connections": {
    "Webhook": {
      "main": [[{
        "node": "Create Notification File",
        "type": "main",
        "index": 0
      }]]
    }
  },
  "settings": {"executionOrder": "v1"}
}]
```

#### 2. Import and Execute
```bash
# Copy workflow to container
docker cp workflow.json n8n:/tmp/workflow.json

# Import workflow
docker exec n8n n8n import:workflow --input=/tmp/workflow.json

# List to get workflow ID
WORKFLOW_ID=$(docker exec n8n n8n list:workflow --onlyId | tail -1)

# Execute workflow
docker exec n8n n8n execute --id=$WORKFLOW_ID

# Verify execution result
ls -la /home/matthalloran8/workflow-notification.txt
cat /home/matthalloran8/workflow-notification.txt
```

### Advanced Programmatic Patterns

#### Workflow Templates with Parameters
```bash
# Create parameterized workflow template
cat > workflow-template.json << 'EOF'
[{
  "name": "Parameterized Task Runner",
  "nodes": [{
    "parameters": {
      "command": "echo \"{{TASK_MESSAGE}}\" > {{OUTPUT_FILE}}"
    },
    "type": "n8n-nodes-base.executeCommand",
    "name": "Execute Task"
  }]
}]
EOF

# Generate specific workflow instance
sed 's/{{TASK_MESSAGE}}/Build completed successfully/g; s/{{OUTPUT_FILE}}/\/host\/home\/build-status.txt/g' \
  workflow-template.json > build-workflow.json
```

#### Batch Workflow Management
```bash
# Export all workflows for backup
docker exec n8n n8n export:workflow --backup --output=/tmp/backup-$(date +%Y%m%d)/

# Activate all workflows
docker exec n8n n8n update:workflow --all --active=true

# Execute multiple workflows in parallel
for id in $(docker exec n8n n8n list:workflow --active=true --onlyId); do
  docker exec n8n n8n execute --id=$id &
done
wait
```

#### Monitoring and Logging
```bash
# Check execution status and logs
docker exec n8n n8n list:workflow | while read line; do
  id=$(echo $line | cut -d'|' -f1)
  name=$(echo $line | cut -d'|' -f2)
  echo "Workflow: $name (ID: $id)"
  
  # Execute and capture result
  result=$(docker exec n8n n8n execute --id=$id --rawOutput 2>&1)
  if echo "$result" | grep -q "successful"; then
    echo "  ✅ Success"
  else
    echo "  ❌ Failed: $result"
  fi
done
```

### Integration Patterns

#### CI/CD Integration
```bash
#!/bin/bash
# Deploy workflow as part of CI/CD pipeline

# Import deployment workflow
docker exec n8n n8n import:workflow --input=/workspace/ci/deploy-workflow.json

# Get workflow ID
DEPLOY_ID=$(docker exec n8n n8n list:workflow | grep "Deploy Pipeline" | cut -d'|' -f1)

# Execute deployment
if docker exec n8n n8n execute --id=$DEPLOY_ID | grep -q "successful"; then
  echo "Deployment workflow completed successfully"
  exit 0
else
  echo "Deployment workflow failed"
  exit 1
fi
```

#### Scheduled Automation
```bash
# Add to crontab for scheduled workflow execution
# Execute backup workflow every day at 2 AM
# 0 2 * * * docker exec n8n n8n execute --id=BACKUP_WORKFLOW_ID

# Execute system health check every hour
# 0 * * * * docker exec n8n n8n execute --id=HEALTH_CHECK_WORKFLOW_ID
```

### Best Practices for Programmatic Usage

1. **Workflow Versioning**: Always export workflows before making changes
2. **Error Handling**: Check execution results and implement retry logic
3. **Logging**: Use `--rawOutput` for machine-readable execution results
4. **Security**: Limit API key scopes in enterprise environments
5. **Performance**: Use batch operations for multiple workflow management
6. **Backup**: Regular automated backups of all workflows and credentials

## Architecture

### Directory Structure
```
n8n/
├── manage.sh               # Main management script (with API workaround)
├── Dockerfile              # Custom n8n image definition
├── docker-entrypoint.sh    # PATH setup and command handling
├── README.md              # This file
├── example-notification-workflow.json  # Example workflow file
└── config/                # Future configuration files
```

### How It Works

1. **Custom Entrypoint**: The `docker-entrypoint.sh` script:
   - Adds host directories to PATH
   - Sets up command fallback handling
   - Preserves n8n's original functionality

2. **Smart Command Resolution**:
   - Container binaries are preferred
   - Falls back to host binaries if not found
   - Special wrappers for compatibility (e.g., tput)

3. **Volume Strategy**:
   - Read-only mounts for system directories
   - Read-write for workspace and home
   - Conditional mounts based on availability

## Troubleshooting

### Command Not Found
- Verify the command exists on the host: `which command-name`
- Check if the directory is mounted: `ls /host/usr/bin/`
- Try using the full path: `/host/usr/bin/command-name`

### Permission Denied
- Some commands may require specific permissions
- Check file ownership and permissions
- Consider using sudo in the host system if needed

### Container Won't Start
- Check logs: `docker logs n8n`
- Verify ports are available: `ss -tlnp | grep 5678`
- Ensure Docker socket is accessible

### Build Failures
- Check Docker is running: `docker info`
- Verify Dockerfile syntax
- Ensure all files in this directory are present

### Workflow Execution Issues
- CLI execute is broken in v1.93.0+ (use `./manage.sh --action execute` instead)
- Run `./manage.sh --action api-setup` for API configuration help
- Verify API key is set: `echo $N8N_API_KEY`

## Development

### Building Custom Image Manually
```bash
cd scripts/resources/automation/n8n
docker build -t n8n-vrooli:latest .
```

### Testing Commands
```bash
# Test command resolution
docker exec n8n which tput
docker exec n8n tput bel

# Test workspace access
docker exec n8n ls /workspace

# Test Docker access
docker exec n8n docker ps
```

### Extending the Image
Edit `Dockerfile` to add more tools:
```dockerfile
RUN apk add --no-cache \
    your-package-here
```

Then rebuild: `./n8n.sh --action install --build-image yes --force yes`

## Best Practices

1. **Security**: Only use in trusted environments
2. **Commands**: Test commands in Docker first before using in workflows
3. **Paths**: Use `/workspace` for Vrooli files, `/host/home` for user files
4. **Cleanup**: Regularly clean execution data to save space
5. **Backups**: Export workflows regularly

## FAQ

**Q: Why custom image instead of standard n8n?**
A: Standard n8n in Docker can't access host commands. Our custom image bridges this gap.

**Q: Is this secure?**
A: This setup prioritizes functionality over isolation. Use only in trusted environments.

**Q: Can I use this in production?**
A: Recommend standard n8n for production unless you specifically need host access.

**Q: How do I update n8n?**
A: Rebuild the custom image: `./manage.sh --action install --build-image yes --force yes`

**Q: Why can't I execute workflows with the CLI?**
A: This is a known bug in n8n v1.93.0+ (issue #15567). Use `./manage.sh --action execute --workflow-id ID` with API authentication instead.

## Configuration

### Environment Variables
- `N8N_CUSTOM_PORT`: Override default port (default: 5678)
- `N8N_BASIC_AUTH_USER`: Basic auth username (default: admin)
- `N8N_BASIC_AUTH_PASSWORD`: Basic auth password (auto-generated if not set)
- `N8N_DB_TYPE`: Database type: sqlite or postgres (default: sqlite)
- `N8N_WEBHOOK_URL`: External webhook URL for triggers
- `N8N_API_KEY`: API key for REST API access (create in web UI)

### Vrooli Integration
n8n is automatically configured in `~/.vrooli/resources.local.json`:
```json
{
  "services": {
    "automation": {
      "n8n": {
        "enabled": true,
        "baseUrl": "http://localhost:5678",
        "healthCheck": {
          "intervalMs": 60000,
          "timeoutMs": 5000
        },
        "api": {
          "version": "v1",
          "workflowsEndpoint": "/api/v1/workflows",
          "executionsEndpoint": "/api/v1/executions",
          "credentialsEndpoint": "/api/v1/credentials"
        }
      }
    }
  }
}
```

## Support

For issues related to:
- This custom setup: Check this README and manage.sh script
- n8n itself: Visit https://docs.n8n.io
- Vrooli integration: See Vrooli documentation