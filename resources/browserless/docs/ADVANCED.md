[← Back to README](../README.md) | [Documentation Index](./README.md)

# Browserless Advanced Topics

This guide covers advanced configurations, architectures, and integration patterns for Browserless.

## Table of Contents

- [Architecture Overview](#architecture-overview)
- [Scaling Strategies](#scaling-strategies)
- [Custom Extensions](#custom-extensions)
- [Security Hardening](#security-hardening)
- [Performance Tuning](#performance-tuning)
- [Integration Architectures](#integration-architectures)
- [Monitoring and Observability](#monitoring-and-observability)
- [Developer Notes](#developer-notes)

## Architecture Overview

### Component Architecture

```
┌─────────────────────────────────────────────┐
│            Browserless Container            │
├─────────────────────────────────────────────┤
│         REST API Layer (Express)            │
├─────────────────────────────────────────────┤
│         Queue Manager (Bull)                │
├─────────────────────────────────────────────┤
│      Browser Pool (Puppeteer)               │
├─────────────────────────────────────────────┤
│    Chrome Instances (Headless)              │
└─────────────────────────────────────────────┘
```

### Request Flow

1. **API Request** → REST endpoint receives request
2. **Validation** → Request parameters validated
3. **Queue** → Request added to job queue
4. **Browser Pool** → Available browser assigned
5. **Execution** → Chrome performs requested action
6. **Cleanup** → Browser reset for next request
7. **Response** → Result returned to client

### Resource Management

- **Connection Pooling**: Browsers pre-warmed and reused
- **Memory Management**: Automatic cleanup after each request
- **Queue Management**: Prevents overload with configurable limits
- **Health Monitoring**: Built-in pressure metrics

## Scaling Strategies

### Horizontal Scaling

Deploy multiple Browserless instances behind a load balancer:

```yaml
# docker-compose.yml
version: '3.8'
services:
  browserless-1:
    image: ghcr.io/browserless/chrome:latest
    ports:
      - "4110:3000"
    environment:
      - CONCURRENT=10
      - PREBOOT_CHROME=true
    deploy:
      resources:
        limits:
          memory: 4G
  
  browserless-2:
    image: ghcr.io/browserless/chrome:latest
    ports:
      - "4111:3000"
    environment:
      - CONCURRENT=10
      - PREBOOT_CHROME=true
    deploy:
      resources:
        limits:
          memory: 4G

  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf
```

### Vertical Scaling

Optimize single instance for maximum throughput:

```bash
docker run -d \
  --name browserless \
  --shm-size=8gb \
  --memory="16g" \
  --cpus="8" \
  -e CONCURRENT=50 \
  -e MAX_QUEUE_LENGTH=100 \
  -e PREBOOT_CHROME=true \
  -e KEEPALIVE=true \
  -p 4110:3000 \
  ghcr.io/browserless/chrome:latest
```

### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: browserless
spec:
  replicas: 3
  selector:
    matchLabels:
      app: browserless
  template:
    metadata:
      labels:
        app: browserless
    spec:
      containers:
      - name: browserless
        image: ghcr.io/browserless/chrome:latest
        ports:
        - containerPort: 3000
        env:
        - name: CONCURRENT
          value: "20"
        - name: TOKEN
          valueFrom:
            secretKeyRef:
              name: browserless-secret
              key: token
        resources:
          requests:
            memory: "4Gi"
            cpu: "2"
          limits:
            memory: "8Gi"
            cpu: "4"
        volumeMounts:
        - name: dshm
          mountPath: /dev/shm
      volumes:
      - name: dshm
        emptyDir:
          medium: Memory
          sizeLimit: 4Gi
```

## Custom Extensions

### Custom Chrome Arguments

```javascript
// Custom launch arguments for specific use cases
const customArgs = [
  // Performance
  '--disable-gpu',
  '--disable-dev-shm-usage',
  '--disable-setuid-sandbox',
  '--no-sandbox',
  
  // Security
  '--disable-web-security',
  '--disable-features=IsolateOrigins',
  '--disable-site-isolation-trials',
  
  // Proxy
  '--proxy-server=http://proxy.example.com:8080',
  '--proxy-bypass-list=localhost,127.0.0.1',
  
  // Custom profile
  '--user-data-dir=/tmp/chrome-profile',
  
  // Debugging
  '--enable-logging',
  '--v=1'
];
```

### Custom Headers and User Agents

```bash
# Mobile device emulation with custom headers
curl -X POST http://localhost:4110/chrome/screenshot \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://example.com",
    "viewport": {
      "width": 375,
      "height": 812,
      "deviceScaleFactor": 3,
      "isMobile": true
    },
    "userAgent": "Mozilla/5.0 (iPhone; CPU iPhone OS 14_0 like Mac OS X)",
    "extraHTTPHeaders": {
      "Accept-Language": "en-US",
      "X-Custom-Header": "value"
    }
  }'
```

### Browser Context Isolation

```javascript
// Isolated browser context for security
{
  "url": "https://untrusted-site.com",
  "blockAds": true,
  "stealth": true,
  "args": [
    "--disable-blink-features=AutomationControlled"
  ]
}
```

## Security Hardening

### Production Security Checklist

1. **Authentication**
   ```bash
   # Generate secure token
   TOKEN=$(openssl rand -base64 32)
   
   # Run with token
   docker run -d \
     -e TOKEN=$TOKEN \
     ghcr.io/browserless/chrome:latest
   ```

2. **Network Isolation**
   ```bash
   # Create isolated network
   docker network create --driver bridge browserless-net
   
   # Run in isolated network
   docker run -d \
     --network browserless-net \
     --name browserless \
     ghcr.io/browserless/chrome:latest
   ```

3. **Resource Limits**
   ```bash
   # Strict resource limits
   docker run -d \
     --memory="2g" \
     --memory-swap="2g" \
     --cpu-period=100000 \
     --cpu-quota=200000 \
     --pids-limit=100 \
     ghcr.io/browserless/chrome:latest
   ```

4. **Disable Dangerous Features**
   ```bash
   docker run -d \
     -e FUNCTION_ENABLE=false \
     -e DOWNLOAD_DIR=/dev/null \
     -e WORKSPACE_DELETE_EXPIRED=true \
     -e WORKSPACE_EXPIRE_DAYS=0 \
     ghcr.io/browserless/chrome:latest
   ```

### Security Headers

```nginx
# nginx.conf for reverse proxy
location /browserless/ {
    proxy_pass http://browserless:3000/;
    
    # Security headers
    add_header X-Content-Type-Options nosniff;
    add_header X-Frame-Options DENY;
    add_header X-XSS-Protection "1; mode=block";
    add_header Strict-Transport-Security "max-age=31536000";
    
    # Rate limiting
    limit_req zone=browserless burst=10 nodelay;
    limit_req_status 429;
}
```

## Performance Tuning

### Chrome Optimization Flags

```javascript
const performanceArgs = [
  // Disable unnecessary features
  '--disable-background-networking',
  '--disable-background-timer-throttling',
  '--disable-backgrounding-occluded-windows',
  '--disable-breakpad',
  '--disable-client-side-phishing-detection',
  '--disable-component-extensions-with-background-pages',
  '--disable-default-apps',
  '--disable-extensions',
  '--disable-features=TranslateUI',
  '--disable-hang-monitor',
  '--disable-ipc-flooding-protection',
  '--disable-popup-blocking',
  '--disable-prompt-on-repost',
  '--disable-renderer-backgrounding',
  '--disable-sync',
  
  // Memory optimization
  '--memory-pressure-off',
  '--max_old_space_size=4096',
  
  // Process management
  '--disable-features=site-per-process',
  '--disable-web-resources',
  
  // Font rendering
  '--disable-font-subpixel-positioning',
  '--font-render-hinting=none'
];
```

### Connection Pooling Strategy

```javascript
// Client-side connection pooling
class BrowserlessPool {
  constructor(endpoints, maxRetries = 3) {
    this.endpoints = endpoints;
    this.maxRetries = maxRetries;
    this.currentIndex = 0;
  }
  
  async request(path, options) {
    let lastError;
    
    for (let retry = 0; retry < this.maxRetries; retry++) {
      try {
        const endpoint = this.getNextEndpoint();
        const response = await fetch(`${endpoint}${path}`, options);
        
        if (response.ok) {
          return response;
        }
        
        if (response.status === 429) {
          // Try next endpoint on rate limit
          continue;
        }
        
        throw new Error(`HTTP ${response.status}`);
      } catch (error) {
        lastError = error;
      }
    }
    
    throw lastError;
  }
  
  getNextEndpoint() {
    const endpoint = this.endpoints[this.currentIndex];
    this.currentIndex = (this.currentIndex + 1) % this.endpoints.length;
    return endpoint;
  }
}
```

## Integration Architectures

### Event-Driven Architecture

```javascript
// Redis-based job queue integration
const Queue = require('bull');
const screenshotQueue = new Queue('screenshots', 'redis://localhost:6379');

screenshotQueue.process(async (job) => {
  const { url, options } = job.data;
  
  const response = await fetch('http://browserless:3000/chrome/screenshot', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ url, options })
  });
  
  const buffer = await response.buffer();
  return { screenshot: buffer.toString('base64') };
});
```

### Microservices Pattern

```yaml
# Browserless as part of microservices architecture
services:
  api-gateway:
    depends_on:
      - browserless
      - image-processor
      - storage-service
  
  browserless:
    image: ghcr.io/browserless/chrome:latest
    environment:
      - CONCURRENT=20
  
  image-processor:
    build: ./image-processor
    environment:
      - BROWSERLESS_URL=http://browserless:3000
  
  storage-service:
    build: ./storage
    volumes:
      - ./data:/data
```

## Monitoring and Observability

### Prometheus Integration

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'browserless'
    static_configs:
      - targets: ['browserless:3000']
    metrics_path: '/metrics'
```

### Custom Metrics Collection

```javascript
// Collect and forward metrics
const collectMetrics = async () => {
  const pressure = await fetch('http://localhost:4110/pressure').then(r => r.json());
  const config = await fetch('http://localhost:4110/config').then(r => r.json());
  
  // Forward to monitoring system
  await sendToMonitoring({
    browserless_queue_size: pressure.queued,
    browserless_running: pressure.running,
    browserless_cpu: pressure.cpu,
    browserless_memory: pressure.memory,
    browserless_available: pressure.isAvailable ? 1 : 0
  });
};

setInterval(collectMetrics, 30000);
```

### Logging Pipeline

```bash
# Structured logging with JSON
docker run -d \
  --log-driver=fluentd \
  --log-opt fluentd-address=localhost:24224 \
  --log-opt tag=browserless \
  -e LOG_LEVEL=info \
  ghcr.io/browserless/chrome:latest
```

## Developer Notes

### Local Development Setup

```bash
# Development compose file
cat > docker-compose.dev.yml <<EOF
version: '3.8'
services:
  browserless:
    image: ghcr.io/browserless/chrome:latest
    ports:
      - "4110:3000"
      - "9222:9222"  # Chrome DevTools
    environment:
      - DEBUG=browserless*
      - ENABLE_DEBUGGER=true
      - PREBOOT_CHROME=false
      - CONCURRENT=2
    volumes:
      - ./workspace:/workspace
EOF

docker-compose -f docker-compose.dev.yml up
```

### Debugging Chrome Issues

```javascript
// Enable verbose Chrome logging
{
  "args": [
    "--enable-logging",
    "--v=1",
    "--dump-dom",
    "--log-level=0"
  ],
  "dumpio": true
}
```

### Testing Strategies

```bash
# Load testing script
for i in {1..100}; do
  curl -X POST http://localhost:4110/chrome/screenshot \
    -H "Content-Type: application/json" \
    -d '{"url": "https://example.com"}' \
    -o /dev/null &
done

# Monitor pressure during load
watch -n 1 'curl -s http://localhost:4110/pressure | jq .'
```

---
**See also:** [Configuration](./CONFIGURATION.md) | [API Reference](./API.md) | [Troubleshooting](./TROUBLESHOOTING.md)