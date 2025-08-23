# Windmill Installation Guide

This guide covers the installation and initial setup of Windmill.

## Prerequisites

- Docker and Docker Compose installed
- At least 4GB of RAM available
- 10GB of free disk space

## Installation Methods

### 1. Quick Install (Recommended)

Install Windmill with default settings:

```bash
./manage.sh --action install
```

This will:
- Start Windmill with 3 workers
- Use an internal PostgreSQL database
- Enable Language Server Protocol (LSP) for code completion
- Create default admin workspace

### 2. Custom Installation

#### With External Database

```bash
./manage.sh --action install \
  --external-db yes \
  --db-url "postgresql://user:pass@host:port/database"
```

#### With Custom Worker Count

```bash
./manage.sh --action install --workers 5
```

#### Without Language Server (Lower Memory Usage)

```bash
./manage.sh --action install --no-lsp
```

#### Combined Options

```bash
./manage.sh --action install \
  --workers 10 \
  --external-db yes \
  --db-url "postgresql://windmill:secret@db.example.com:5432/windmill" \
  --no-lsp
```

## Post-Installation Steps

### 1. Verify Installation

```bash
./manage.sh --action status
```

Expected output:
```
[INFO] Windmill Status:
✅ Windmill server: Running (healthy)
✅ Worker 1: Running
✅ Worker 2: Running
✅ Worker 3: Running
✅ Database: Connected
✅ Redis: Connected
```

### 2. Access Web Interface

Open http://localhost:5681 in your browser.

Default credentials:
- **Email**: admin@windmill.dev
- **Password**: changeme

### 3. Change Default Password

1. Log in with default credentials
2. Go to Account Settings
3. Change your password immediately

### 4. Create Your First Workspace

1. Click "Create Workspace"
2. Enter a workspace ID (e.g., "dev", "prod")
3. Configure workspace settings

## Environment Variables

Key configuration options:

```bash
# Custom port (default: 5681)
export WINDMILL_CUSTOM_PORT=8080

# Database configuration
export WINDMILL_DATABASE_URL="postgresql://user:pass@host:port/db"

# Worker configuration
export WINDMILL_WORKER_COUNT=5
export WINDMILL_WORKER_TAGS="gpu,heavy"

# API key (auto-generated if not set)
export WINDMILL_CUSTOM_API_KEY="your-secure-key"
```

## Troubleshooting Installation

### Port Already in Use

```bash
# Check what's using the port
sudo lsof -i :5681

# Use a different port
WINDMILL_CUSTOM_PORT=8080 ./manage.sh --action install
```

### Database Connection Issues

```bash
# Test database connection
./manage.sh --action test-db

# Check database logs
docker logs windmill-postgres
```

### Insufficient Resources

If installation fails due to resources:

1. Reduce worker count: `--workers 1`
2. Disable LSP: `--no-lsp`
3. Check Docker resource limits

## Next Steps

- [Configure Windmill](CONFIGURATION.md)
- [Create your first script](CONCEPTS.md#scripts)
- [Explore example workflows](../examples/flows/README.md)