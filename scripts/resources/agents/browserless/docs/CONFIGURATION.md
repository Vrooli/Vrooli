[← Back to README](../README.md) | [Documentation Index](./README.md)

# Browserless Configuration Guide

This guide covers all configuration options for Browserless in the Vrooli ecosystem.

## Configuration Overview

Browserless configuration happens at three levels:
1. **Container Environment Variables** - Runtime behavior settings
2. **Vrooli Resource Configuration** - Integration settings
3. **Per-Request Options** - API call customization

## Container Environment Variables

These variables control the Browserless Docker container behavior:

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `CONCURRENT` | Maximum concurrent browser sessions | 5 | `10` |
| `TIMEOUT` | Default timeout in milliseconds | 30000 | `60000` |
| `ENABLE_DEBUGGER` | Enable Chrome DevTools debugger | false | `true` |
| `PREBOOT_CHROME` | Keep Chrome instances warm | true | `false` |
| `MAX_PAYLOAD_SIZE` | Maximum request payload size | 5mb | `10mb` |
| `WORKSPACE_DIR` | Directory for temporary files | /tmp | `/workspace` |
| `FUNCTION_ENABLE` | Allow function execution endpoint | true | `false` |
| `FUNCTION_BUILT_INS` | Built-in modules for functions | [] | `["cheerio"]` |

### Setting Environment Variables

During installation:
```bash
./manage.sh --action install \
  --max-browsers 10 \
  --timeout 60000 \
  --enable-debugger true
```

Or modify the running container:
```bash
docker stop browserless
docker rm browserless
docker run -d \
  --name browserless \
  -p 4110:3000 \
  -e CONCURRENT=10 \
  -e TIMEOUT=60000 \
  -e ENABLE_DEBUGGER=true \
  --shm-size=2gb \
  ghcr.io/browserless/chrome:latest
```

## Vrooli Resource Configuration

Browserless is automatically configured in `~/.vrooli/resources.local.json`:

```json
{
  "services": {
    "agents": {
      "browserless": {
        "enabled": true,
        "baseUrl": "http://localhost:4110",
        "healthCheck": {
          "endpoint": "/pressure",
          "intervalMs": 60000,
          "timeoutMs": 5000,
          "retries": 3
        },
        "features": {
          "screenshots": true,
          "pdf": true,
          "scraping": true,
          "automation": true,
          "function": true
        },
        "browser": {
          "maxConcurrency": 5,
          "headless": true,
          "timeout": 30000,
          "defaultViewport": {
            "width": 1920,
            "height": 1080
          }
        },
        "security": {
          "allowFileAccess": false,
          "allowCodeExecution": true,
          "maxPayloadSize": "5mb"
        }
      }
    }
  }
}
```

### Configuration Properties

#### Base Configuration
- `enabled`: Whether the resource is active
- `baseUrl`: Service endpoint URL
- `healthCheck`: Health monitoring settings

#### Feature Flags
- `screenshots`: Enable screenshot API
- `pdf`: Enable PDF generation
- `scraping`: Enable web scraping
- `automation`: Enable browser automation
- `function`: Enable custom function execution

#### Browser Settings
- `maxConcurrency`: Maximum parallel sessions
- `headless`: Run without GUI
- `timeout`: Default operation timeout
- `defaultViewport`: Default browser window size

#### Security Settings
- `allowFileAccess`: Allow local file access
- `allowCodeExecution`: Allow custom code execution
- `maxPayloadSize`: Maximum request size

## Advanced Configuration

### Memory Optimization

For high-load scenarios:

```bash
# Increase shared memory
docker run -d \
  --name browserless \
  --shm-size=4gb \
  -e CONCURRENT=20 \
  ghcr.io/browserless/chrome:latest
```

### Chrome Launch Arguments

Custom Chrome flags can be set:

```json
{
  "args": [
    "--disable-gpu",
    "--disable-dev-shm-usage",
    "--disable-setuid-sandbox",
    "--no-sandbox",
    "--disable-web-security"
  ]
}
```

### Proxy Configuration

For requests through a proxy:

```json
{
  "args": [
    "--proxy-server=http://proxy.example.com:8080"
  ]
}
```

## Performance Tuning

### Concurrent Sessions

Balance between performance and resource usage:

- **Low (1-5)**: Stable, low memory usage
- **Medium (5-10)**: Good balance for most uses
- **High (10-20)**: Requires monitoring and tuning

### Timeout Settings

Adjust based on your use cases:

- **Screenshots**: 10-30 seconds
- **PDF Generation**: 30-60 seconds
- **Complex Automation**: 60-120 seconds

### Resource Limits

Set Docker resource constraints:

```bash
docker run -d \
  --name browserless \
  --memory="4g" \
  --cpus="2" \
  ghcr.io/browserless/chrome:latest
```

## Monitoring Configuration

### Metrics Collection

Enable Prometheus metrics:

```json
{
  "metrics": {
    "enabled": true,
    "port": 9090,
    "path": "/metrics"
  }
}
```

### Logging Configuration

Control log verbosity:

```bash
docker run -d \
  -e DEBUG=browserless* \
  -e LOG_LEVEL=info \
  ghcr.io/browserless/chrome:latest
```

## Security Configuration

### Authentication

Enable token-based auth:

```bash
docker run -d \
  -e TOKEN=your-secret-token \
  ghcr.io/browserless/chrome:latest
```

Then use in requests:
```bash
curl -X POST http://localhost:4110/chrome/screenshot?token=your-secret-token
```

### Network Isolation

Restrict network access:

```bash
docker run -d \
  --network=isolated_network \
  ghcr.io/browserless/chrome:latest
```

### File System Access

Disable local file access:

```json
{
  "args": ["--disable-file-system"]
}
```

## Integration Examples

### With n8n Automation

```json
{
  "browserless": {
    "baseUrl": "http://browserless:3000",
    "timeout": 60000,
    "concurrency": 3
  }
}
```

### With AI Services

```json
{
  "browserless": {
    "features": {
      "screenshots": true,
      "validation": {
        "minFileSize": 1024,
        "validateMimeType": true,
        "cleanupOnError": true
      }
    }
  }
}
```

## Configuration Best Practices

1. **Start Conservative**: Begin with low concurrency and increase as needed
2. **Monitor Resources**: Watch memory and CPU usage
3. **Set Appropriate Timeouts**: Balance between reliability and performance
4. **Use Health Checks**: Ensure service availability
5. **Secure Production**: Always use authentication in production

## Troubleshooting Configuration

### Verify Current Configuration

```bash
# Check environment variables
docker exec browserless env | grep -E "(CONCURRENT|TIMEOUT)"

# Check Vrooli configuration
cat ~/.vrooli/resources.local.json | jq '.services.agents.browserless'
```

### Common Configuration Issues

1. **Out of Memory**: Reduce CONCURRENT or increase --shm-size
2. **Timeouts**: Increase TIMEOUT environment variable
3. **Port Conflicts**: Change mapping in docker run command
4. **Permission Errors**: Check file system permissions

---
**See also:** [Installation](./INSTALLATION.md) | [API Reference](./API.md) | [Troubleshooting](./TROUBLESHOOTING.md)