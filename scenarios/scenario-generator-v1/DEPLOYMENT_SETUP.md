# Scenario Generator V1 - Deployment Setup Guide

This guide provides step-by-step instructions for setting up the Scenario Generator V1, including required credential configuration for n8n workflows.

## Prerequisites

- PostgreSQL database running and accessible
- Redis running (optional but recommended)
- Vrooli installation with Claude resource configured
- Go 1.21+ for compiling the API

## Database Setup

### PostgreSQL Configuration

Ensure PostgreSQL is running and create the required database:

```bash
psql -U postgres -c "CREATE DATABASE vrooli;"
```

### Initialize Schema

```bash
psql -U postgres -d vrooli < initialization/postgres/schema.sql
```

## Environment Variables

Set the following environment variables before running the deployment:

```bash
export VROOLI_ROOT="/path/to/your/vrooli/installation"
export POSTGRES_HOST="localhost"
export POSTGRES_PORT="5433"
export POSTGRES_DB="vrooli"
export POSTGRES_USER="postgres"
export POSTGRES_PASSWORD="your_password"
export API_PORT="8080"
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

3. **Build and Start API**:
   ```bash
   cd api
   go build -o scenario-generator-api
   ./scenario-generator-api
   ```

4. **Verify Deployment**:
   ```bash
   # Run integration tests
   ./test.sh
   ```

## Troubleshooting

### Pipeline Errors
If generation fails:

1. **Check Vrooli Claude Resource**: Ensure `vrooli resource claude` command works:
   ```bash
   echo "Hello" | vrooli resource claude chat
   ```

2. **Check Database Connection**: Verify PostgreSQL is accessible:
   ```bash
   psql -h localhost -p 5433 -U postgres -d vrooli
   ```

### Service Connection Issues

1. **PostgreSQL**: 
   - Verify connection: `psql -h localhost -p 5433 -U postgres -d vrooli`
   - Check if database exists and schema is applied

2. **API Service**:
   - Verify API is running: `curl http://localhost:8080/health`
   - Check pipeline status in health response

3. **Claude Code**:
   - Verify installation: `claude --version`
   - Check authentication: `claude auth whoami`

### API Build Issues

1. **Go Module Dependencies**: Update dependencies if needed:
   ```bash
   cd api
   go mod tidy
   go mod download
   ```

2. **Pipeline Package**: Ensure pipeline package is properly imported and compiled

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
   - View API logs for generation progress
   - Check database for scenario status updates
   - Monitor backlog folders for item movement

## Support

- Check logs in n8n workflow execution history
- Run health checks: `./deployment/startup.sh health`  
- Review database records in `scenarios` table
- Examine MinIO bucket contents for generated artifacts

For additional help, refer to the main README.md file or check the troubleshooting section in the deployment startup script.