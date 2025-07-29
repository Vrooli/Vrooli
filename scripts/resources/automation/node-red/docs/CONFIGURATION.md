# Node-RED Configuration Guide

This guide covers all configuration options for Node-RED, including installation parameters, runtime settings, and advanced customization.

## Installation Configuration

### Basic Installation Options

```bash
# Install with custom image (recommended for host access)
./manage.sh --action install --build-image yes

# Install with standard Node-RED image
./manage.sh --action install --build-image no

# Force reinstall if already exists
./manage.sh --action install --build-image yes --force yes
```

### Installation Parameters

| Parameter | Description | Default | Options |
|-----------|-------------|---------|---------|
| `--build-image` | Build custom image with host access | `yes` | `yes`, `no` |
| `--force` | Force reinstall if already exists | `no` | `yes`, `no` |
| `--port` | Override default port | `1880` | Any available port |
| `--flow-file` | Default flow file name | `flows.json` | Any filename |

## Environment Variables

### Core Configuration

```bash
# Service Configuration
export NODE_RED_PORT=1880                    # Override default port
export NODE_RED_FLOW_FILE=flows.json        # Flow file name
export NODE_RED_CREDENTIAL_SECRET=mysecret  # Encryption key for credentials
export TZ=America/New_York                  # Timezone setting

# Resource Limits
export NODE_RED_MEMORY_LIMIT=1g             # Container memory limit
export NODE_RED_CPU_LIMIT=1.0               # Container CPU limit

# Feature Toggles
export NODE_RED_ENABLE_PROJECTS=true        # Enable project mode
export NODE_RED_ENABLE_TOUR=false           # Disable welcome tour
export NODE_RED_LOGGING_LEVEL=info          # Logging level
```

### Host Integration Configuration

```bash
# Host Access Settings
export NODE_RED_HOST_ACCESS=true            # Enable host command access
export NODE_RED_DOCKER_SOCKET=true          # Enable Docker socket access
export NODE_RED_WORKSPACE_PATH=/workspace   # Workspace mount path

# Security Settings
export NODE_RED_ADMIN_AUTH=false            # Enable admin authentication
export NODE_RED_HTTP_AUTH=false             # Enable HTTP authentication
export NODE_RED_HTTPS_ENABLED=false         # Enable HTTPS
```

## Docker Configuration

### Volume Mounts

The Node-RED container uses several volume mounts:

```bash
# Data persistence
-v node-red-data:/data                       # Node-RED user data and flows

# Host integration (when enabled)
-v /var/run/docker.sock:/var/run/docker.sock # Docker control
-v "${PWD}:/workspace:ro"                    # Workspace access (read-only)
-v /usr/bin:/host/usr/bin:ro                 # Host binaries access
-v /bin:/host/bin:ro                         # System binaries access

# Optional mounts
-v /etc/localtime:/etc/localtime:ro          # Timezone sync
-v ./custom-nodes:/data/node_modules         # Custom node modules
```

### Custom Docker Options

```bash
# Advanced Docker configuration
export DOCKER_EXTRA_ARGS="--privileged --device /dev/ttyUSB0"

# Custom network configuration
export DOCKER_NETWORK="vrooli-network"

# Additional environment variables
export DOCKER_ENV_VARS="-e CUSTOM_VAR=value"
```

## Node-RED Settings

### settings.js Configuration

The main configuration file is located at `/data/settings.js`:

```javascript
module.exports = {
    // HTTP settings
    uiPort: process.env.PORT || 1880,
    uiHost: "0.0.0.0",
    
    // Runtime settings
    flowFile: 'flows.json',
    flowFilePretty: true,
    credentialSecret: process.env.NODE_RED_CREDENTIAL_SECRET || false,
    
    // Function node settings
    functionGlobalContext: {
        util: require('util'),
        exec: require('child_process').exec,
        fs: require('fs'),
        path: require('path')
    },
    
    // Logging
    logging: {
        console: {
            level: "info",
            metrics: false,
            audit: false
        }
    },
    
    // Editor settings
    editorTheme: {
        projects: {
            enabled: true
        },
        tours: false
    },
    
    // Context storage
    contextStorage: {
        default: {
            module: "memory"
        },
        file: {
            module: "localfilesystem"
        }
    }
};
```

### Authentication Configuration

#### Admin Authentication
```javascript
adminAuth: {
    type: "credentials",
    users: [{
        username: "admin",
        password: "$2a$08$zZWtXTja0fB1pzD4sHCMyOCMYz2Z6dNbM6tl8sJogENOMcxWV9DN.",
        permissions: "*"
    }]
}
```

#### HTTP Node Authentication
```javascript
httpNodeAuth: {
    user: "user",
    pass: "$2a$08$zZWtXTja0fB1pzD4sHCMyOCMYz2Z6dNbM6tl8sJogENOMcxWV9DN."
}
```

### HTTPS Configuration

```javascript
https: {
    key: require("fs").readFileSync('/data/certs/private.key'),
    cert: require("fs").readFileSync('/data/certs/certificate.pem')
}
```

## Custom Image Configuration

### Dockerfile Customization

The custom Node-RED image includes:

```dockerfile
FROM nodered/node-red:latest

# Install system tools
USER root
RUN apk add --no-cache \
    bash \
    curl \
    jq \
    python3 \
    docker-cli \
    git \
    openssh-client

# Install additional Node.js packages
USER node-red
RUN npm install --no-audit --no-update-notifier \
    node-red-contrib-exec \
    node-red-contrib-dockerode \
    node-red-contrib-fs \
    node-red-contrib-cpu \
    node-red-dashboard \
    node-red-contrib-string

# Copy custom entrypoint
COPY docker-entrypoint.sh /usr/local/bin/
USER root
RUN chmod +x /usr/local/bin/docker-entrypoint.sh
USER node-red

ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
```

### Custom Entrypoint Script

The `docker-entrypoint.sh` script provides:

```bash
#!/bin/bash
set -e

# Add host directories to PATH for transparent command access
export PATH="/host/usr/bin:/host/bin:$PATH"

# Set up command fallback handling
export NODE_RED_HOST_COMMANDS=true

# Start Node-RED with original entrypoint
exec /usr/src/node-red/docker-entrypoint.sh "$@"
```

## Flow Configuration

### Flow File Management

```bash
# Flow file locations
/data/flows.json                    # Main flow file
/data/flows_cred.json              # Encrypted credentials
/data/.flows.json.backup           # Automatic backup

# Flow management via manage.sh
./manage.sh --action flow-export --output backup-flows.json
./manage.sh --action flow-import --flow-file new-flows.json
./manage.sh --action flow-list    # List all flows
```

### Default Flows

Pre-configured flows included:

```json
{
  "flows": [
    {
      "id": "test-basic",
      "type": "http in",
      "url": "/test/hello",
      "method": "get"
    },
    {
      "id": "test-exec", 
      "type": "http in",
      "url": "/test/exec",
      "method": "post"
    },
    {
      "id": "vrooli-monitor",
      "type": "inject",
      "cron": "*/30 * * * * *",
      "topic": "resource-health"
    }
  ]
}
```

## Dashboard Configuration

### Dashboard Settings

```javascript
// In settings.js
ui: {
    path: "ui",
    middleware: function(req, res, next) {
        // Custom middleware
        next();
    }
}
```

### Dashboard Themes

```javascript
ui: {
    path: "ui",
    theme: {
        name: "theme-dark",
        lightTheme: {
            default: "#0094CE",
            baseColor: "#0094CE",
            baseFont: "-apple-system,BlinkMacSystemFont,Segoe UI,Roboto,Oxygen-Sans,Ubuntu,Cantarell,Helvetica Neue,sans-serif"
        },
        darkTheme: {
            default: "#097479",
            baseColor: "#097479",
            baseFont: "-apple-system,BlinkMacSystemFont,Segoe UI,Roboto,Oxygen-Sans,Ubuntu,Cantarell,Helvetica Neue,sans-serif"
        }
    }
}
```

## Performance Configuration

### Memory Management

```bash
# Container memory settings
docker update node-red --memory 2g

# Node.js memory settings
export NODE_OPTIONS="--max-old-space-size=1024"

# Flow execution settings
export NODE_RED_MAX_STACK_SIZE=1000
```

### CPU Optimization

```bash
# CPU allocation
docker update node-red --cpus 1.5

# Process priority
export NODE_RED_PROCESS_PRIORITY=0
```

### Context Storage Optimization

```javascript
contextStorage: {
    default: {
        module: "localfilesystem",
        config: {
            dir: "/data/context",
            cache: true,
            flushInterval: 30
        }
    },
    memory: {
        module: "memory",
        config: {
            maxEntries: 1000
        }
    }
}
```

## Network Configuration

### Port Configuration

```bash
# Change default port
export NODE_RED_PORT=8080
./manage.sh --action restart

# Multiple port exposure
docker run -p 1880:1880 -p 1881:1881 node-red
```

### Proxy Configuration

```javascript
// In settings.js for reverse proxy
httpAdminRoot: '/node-red',
httpNodeRoot: '/api',
httpStatic: '/static'
```

### CORS Configuration

```javascript
httpNodeCors: {
    origin: "*",
    methods: "GET,PUT,POST,DELETE"
}
```

## Security Configuration

### Input Validation

```javascript
// Function node input validation
if (!msg.payload || typeof msg.payload !== 'object') {
    node.error("Invalid input payload", msg);
    return null;
}

// Command validation
const allowedCommands = ['ls', 'ps', 'docker ps'];
if (!allowedCommands.includes(msg.payload.command)) {
    node.error("Command not allowed", msg);
    return null;
}
```

### Rate Limiting

```javascript
httpNodeMiddleware: function(req, res, next) {
    // Simple rate limiting
    const ip = req.ip;
    const now = Date.now();
    
    if (!global.rateLimits) global.rateLimits = {};
    if (!global.rateLimits[ip]) global.rateLimits[ip] = [];
    
    // Clean old requests
    global.rateLimits[ip] = global.rateLimits[ip].filter(time => now - time < 60000);
    
    if (global.rateLimits[ip].length >= 60) {
        return res.status(429).send('Rate limit exceeded');
    }
    
    global.rateLimits[ip].push(now);
    next();
}
```

## Backup and Recovery

### Configuration Backup

```bash
# Backup flows and settings
./manage.sh --action flow-export --output "backup-$(date +%Y%m%d).json"

# Backup entire data directory
docker cp node-red:/data ./node-red-backup-$(date +%Y%m%d)

# Backup to tar.gz
tar -czf node-red-backup-$(date +%Y%m%d).tar.gz ./node-red-backup-$(date +%Y%m%d)
```

### Automated Backup

```bash
# Cron job for daily backup
0 2 * * * cd /path/to/vrooli && ./scripts/resources/automation/node-red/manage.sh --action flow-export --output "backups/flows-$(date +\%Y\%m\%d).json"
```

### Recovery Procedure

```bash
# Restore flows
./manage.sh --action flow-import --flow-file backup-flows.json

# Restore full data directory
docker cp ./node-red-backup/ node-red:/data

# Restart service
./manage.sh --action restart
```

## Troubleshooting Configuration

### Configuration Validation

```bash
# Validate configuration
./manage.sh --action test

# Check settings file syntax
docker exec node-red node -c "require('/data/settings.js')"

# Validate flows
docker exec node-red node-red-admin flows validate
```

### Debug Configuration

```javascript
// Enhanced logging in settings.js
logging: {
    console: {
        level: "debug",
        metrics: true,
        audit: true
    },
    file: {
        level: "info",
        filename: "/data/node-red.log",
        maxFiles: 5,
        maxSize: "10m"
    }
}
```

### Configuration Reset

```bash
# Reset to defaults (preserves flows)
./manage.sh --action install --force yes

# Complete reset (removes all data)
docker volume rm node-red-data
./manage.sh --action install --force yes
```

## Integration Configuration

### Vrooli Resource Integration

```javascript
// Function node for resource integration
const resourceConfig = {
    ollama: { host: 'ollama', port: 11434, path: '/api/tags' },
    n8n: { host: 'n8n', port: 5678, path: '/healthz' },
    minio: { host: 'minio', port: 9000, path: '/minio/health/live' }
};

// Auto-discovery of Vrooli resources
global.set('vrooliResources', resourceConfig);
```

### External Service Configuration

```javascript
// External API configuration
const externalServices = {
    github: {
        baseUrl: 'https://api.github.com',
        headers: { 'Authorization': 'token YOUR_TOKEN' }
    },
    slack: {
        webhook: 'https://hooks.slack.com/services/...'
    }
};

global.set('externalServices', externalServices);
```

This configuration guide provides comprehensive coverage of all Node-RED configuration options. For specific use cases, refer to the examples in the flows directory.