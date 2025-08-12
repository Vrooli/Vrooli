# Node-RED Configuration Guide

Essential configuration options for Node-RED installation and runtime settings.

## Installation Options

```bash
# Standard installation with host access (recommended)
./manage.sh --action install --build-image yes

# Force reinstall
./manage.sh --action install --force yes
```

## Key Environment Variables

```bash
# Basic Configuration
export NODE_RED_PORT=1880                    # Service port
export NODE_RED_CREDENTIAL_SECRET=mysecret  # Encryption key
export TZ=America/New_York                  # Timezone

# Host Integration
export NODE_RED_HOST_ACCESS=true            # Enable host commands
export NODE_RED_DOCKER_SOCKET=true          # Enable Docker access

# Security (optional)
export NODE_RED_ADMIN_AUTH=false            # Admin authentication
export NODE_RED_HTTPS_ENABLED=false         # HTTPS mode
```

## Docker Configuration

### Key Volume Mounts
```bash
-v node-red-data:/data                       # Data persistence
-v /var/run/docker.sock:/var/run/docker.sock # Docker access
-v "${PWD}:/workspace:ro"                    # Workspace access
```

## Node-RED Settings

### settings.js Configuration (located in data/ directory)

```javascript
// Essential settings.js configuration
module.exports = {
    uiPort: process.env.PORT || 1880,
    flowFile: 'flows.json',
    credentialSecret: process.env.NODE_RED_CREDENTIAL_SECRET || "default-secret",
    userDir: '/data',
    
    // Optional: Enable authentication
    // adminAuth: {
    //     type: "credentials",
    //     users: [{
    //         username: "admin",
    //         password: "$2a$08$zZWtXTja0fB1pzD4sHCMyOCMYz2Z6dNbM6tl8sJogENOMcxWV9DN.",
    //         permissions: "*"
    //     }]
    // },
    
    functionGlobalContext: {
        os: require('os')
    },
    
    logging: {
        console: {
            level: "info",
            metrics: false,
            audit: false
        }
    }
};
```

## Custom Node Installation

```bash
# Install additional nodes in running container
docker exec node-red-vrooli npm install node-red-contrib-dashboard
docker exec node-red-vrooli npm install node-red-contrib-influxdb

# Restart to load new nodes
./manage.sh --action restart
```

## Common Configuration Examples

### Enable Authentication
```javascript
// Add to settings.js
adminAuth: {
    type: "credentials",
    users: [{
        username: "admin", 
        password: "$2a$08$zZWtXTja0fB1pzD4sHCMyOCMYz2Z6dNbM6tl8sJogENOMcxWV9DN.",
        permissions: "*"
    }]
}
```

### Enable HTTPS
```bash
export NODE_RED_HTTPS_ENABLED=true
# Requires SSL certificates in data/certs/
```

### Project Mode
```bash
export NODE_RED_ENABLE_PROJECTS=true
# Enables Git-based project management
```

For detailed configuration options, see the [Node-RED documentation](https://nodered.org/docs/user-guide/runtime/configuration).