# Scenario Generator V1 - Deployment Setup Guide

This guide provides step-by-step instructions for setting up the Scenario Generator V1, including required credential configuration for n8n workflows.

## Prerequisites

- n8n instance running and accessible at `http://localhost:5678`
- PostgreSQL database running and accessible
- MinIO running and accessible
- Redis running (optional but recommended)
- Claude Code CLI installed and authenticated

## Required Credentials Setup

Before deploying the scenario, you must configure the following credentials in your n8n instance:

### 1. PostgreSQL Credential

1. Open n8n web interface at `http://localhost:5678`
2. Go to **Settings** → **Credentials**
3. Click **+ Create New**
4. Select **PostgreSQL** credential type
5. Configure with these details:
   - **Name**: `PostgreSQL - Scenario Generator`
   - **ID**: `postgres-scenario-generator`
   - **Host**: `localhost` (or your PostgreSQL host)
   - **Port**: `5433` (or your PostgreSQL port)
   - **Database**: `vrooli` (or your database name)
   - **User**: `postgres` (or your username)
   - **Password**: Your PostgreSQL password
   - **SSL**: `false` (unless you're using SSL)

### 2. MinIO S3 Credential

1. In n8n, create a new **S3** credential
2. Configure with these details:
   - **Name**: `MinIO - Scenario Generator`
   - **ID**: `minio-scenario-generator`
   - **Access Key ID**: Your MinIO access key
   - **Secret Access Key**: Your MinIO secret key
   - **Region**: `us-east-1` (or your preferred region)
   - **Custom Endpoint**: `http://localhost:9000` (or your MinIO URL)
   - **Force Path Style**: `true` (important for MinIO)

### 3. Redis Credential (Optional)

1. In n8n, create a new **Redis** credential
2. Configure with these details:
   - **Name**: `Redis - Scenario Generator`
   - **ID**: `redis-scenario-generator`
   - **Host**: `localhost` (or your Redis host)
   - **Port**: `6380` (or your Redis port)
   - **Password**: Your Redis password (if set)
   - **Database**: `0`

## Environment Variables

Set the following environment variables before running the deployment:

```bash
export VROOLI_ROOT="/path/to/your/vrooli/installation"
export POSTGRES_HOST="localhost"
export POSTGRES_PORT="5433"
export POSTGRES_DB="vrooli"
export POSTGRES_USER="postgres"
export POSTGRES_PASSWORD="your_password"
export N8N_HOST="localhost"
export N8N_PORT="5678"
export MINIO_HOST="localhost"
export MINIO_PORT="9000"
export REDIS_HOST="localhost"
export REDIS_PORT="6380"
```

## Deployment Steps

1. **Verify Prerequisites**:
   ```bash
   # Check that all services are running
   ./deployment/startup.sh check
   ```

2. **Deploy the Scenario**:
   ```bash
   # Run the deployment script
   ./deployment/startup.sh deploy
   ```

3. **Import n8n Workflows**:
   - The workflows should be automatically imported during deployment
   - If manual import is needed:
     1. Open n8n at `http://localhost:5678`
     2. Go to **Workflows**
     3. Click **Import from file**
     4. Import each workflow from `initialization/automation/n8n/`

4. **Verify Deployment**:
   ```bash
   # Run integration tests
   ./test.sh
   ```

## Troubleshooting

### Credential Errors
If workflows fail with credential errors:

1. **Check credential IDs**: Ensure the credential IDs in n8n match exactly:
   - `postgres-scenario-generator`
   - `minio-scenario-generator` 
   - `redis-scenario-generator`

2. **Test connections**: In n8n, test each credential connection before saving

3. **Check permissions**: Ensure database user has sufficient permissions for the scenario schema

### Service Connection Issues

1. **PostgreSQL**: 
   - Verify connection: `psql -h localhost -p 5433 -U postgres -d vrooli`
   - Check if database exists and schema is applied

2. **MinIO**:
   - Verify connection: `curl http://localhost:9000/minio/health/live`
   - Check if buckets exist (they should be auto-created)

3. **Claude Code**:
   - Verify installation: `claude --version`
   - Check authentication: `claude auth whoami`

### Workflow Import Issues

1. **Manual Import**: If automatic import fails, import workflows manually:
   ```bash
   # Using n8n CLI (if installed)
   n8n import:workflow --file=initialization/automation/n8n/main-workflow.json
   n8n import:workflow --file=initialization/automation/n8n/planning-workflow.json  
   n8n import:workflow --file=initialization/automation/n8n/building-workflow.json
   n8n import:workflow --file=initialization/automation/n8n/validation-workflow.json
   ```

2. **Credential References**: After import, check that all nodes have their credentials properly assigned

## Usage

Once deployed successfully:

1. **Access the Dashboard**: 
   - Open React UI at `http://localhost:3000`
   - Navigate to the Scenario Generator app

2. **Generate a Scenario**:
   - Enter customer requirements in the idea input form
   - Select complexity level and category
   - Click "Generate Scenario"

3. **Monitor Progress**:
   - View workflow execution in n8n at `http://localhost:5678`
   - Check database for scenario progress updates
   - Monitor MinIO buckets for generated files

## Support

- Check logs in n8n workflow execution history
- Run health checks: `./deployment/startup.sh health`  
- Review database records in `scenarios` table
- Examine MinIO bucket contents for generated artifacts

For additional help, refer to the main README.md file or check the troubleshooting section in the deployment startup script.