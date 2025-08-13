# Browserless Chrome Service

High-performance headless Chrome automation service for web scraping, screenshot generation, PDF creation, and browser automation at scale.

## Quick Reference

| Component | Details |
|-----------|---------|
| **Port** | 4110 |
| **Container** | vrooli-browserless |
| **Status** | `./manage.sh --action status` |
| **Health Check** | http://localhost:4110/pressure |
| **Dashboard** | http://localhost:4110 |

## Quick Start

### Prerequisites
- Docker installed and running
- At least 2GB RAM available
- Port 4110 available

### Installation

```bash
# Install with default settings
./manage.sh --action install

# Install with custom configuration
./manage.sh --action install --max-browsers 10 --timeout 60000

# Install with non-headless mode (for debugging)
./manage.sh --action install --headless no
```

## Core Features

- 🚀 **High-Performance Browser Automation** - Concurrent browser instances with resource pooling
- 📸 **Screenshot Generation** - Capture full-page or viewport screenshots with custom options
- 📄 **PDF Generation** - Convert web pages to PDF with formatting control
- 🔍 **Web Scraping** - Extract content with JavaScript execution support
- ⚡ **Function Execution** - Run custom JavaScript in isolated browser contexts
- 🔄 **Session Management** - Reusable browser sessions for improved performance
- 📊 **Performance Monitoring** - Real-time metrics and pressure monitoring
- 🛡️ **Security Isolation** - Sandboxed Chrome instances with security profiles

## When to Use

### Use Browserless When You Need:
- **Browser Automation at Scale** - Multiple concurrent browser operations
- **JavaScript Rendering** - Pages that require JavaScript execution
- **Visual Testing** - Screenshot comparison and visual regression testing
- **PDF Generation** - High-quality PDF conversion with browser rendering
- **Complex Scraping** - Sites with dynamic content or anti-bot measures

### Alternatives to Consider:
- **Simple HTTP Requests**: Use curl/wget for static content
- **API Access**: Prefer official APIs when available
- **Puppeteer/Playwright**: For programmatic control in Node.js
- **Selenium**: For cross-browser testing requirements

## Integration Examples

### With n8n Automation
```javascript
// n8n HTTP Request node configuration
{
  "method": "POST",
  "url": "http://vrooli-browserless:4110/screenshot",
  "body": {
    "url": "https://example.com",
    "fullPage": true,
    "type": "png"
  }
}
```

### With Ollama AI
```bash
# Generate screenshot for AI analysis
curl -X POST http://localhost:4110/screenshot \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com"}' \
  --output screenshot.png

# Send to Ollama for analysis
# (Requires vision-capable model)
```

### With Custom Scripts
```python
import requests
import base64

# Take screenshot
response = requests.post(
    'http://localhost:4110/screenshot',
    json={
        'url': 'https://example.com',
        'fullPage': True,
        'encoding': 'base64'
    }
)

# Process the base64 image
image_data = base64.b64decode(response.text)
```

## API Reference

### Screenshot API
```bash
POST /screenshot
{
  "url": "https://example.com",
  "fullPage": true,
  "type": "png",
  "quality": 80,
  "width": 1920,
  "height": 1080,
  "deviceScaleFactor": 2
}
```

### PDF API
```bash
POST /pdf
{
  "url": "https://example.com",
  "format": "A4",
  "printBackground": true,
  "landscape": false,
  "margin": {
    "top": "20px",
    "bottom": "20px"
  }
}
```

### Content/Scrape API
```bash
POST /content
{
  "url": "https://example.com",
  "selector": ".main-content",
  "waitForSelector": ".loaded",
  "timeout": 30000
}
```

### Function API
```bash
POST /function
{
  "code": "async () => { return document.title; }",
  "context": {},
  "timeout": 10000
}
```

## Architecture

### Resource Management
Browserless uses a pool of pre-warmed Chrome instances to minimize startup time and maximize throughput. Each request is assigned to an available browser from the pool.

```
Client Request → Load Balancer → Browser Pool → Chrome Instance
                                       ↓
                                 Resource Monitor
```

### Concurrency Control
- **MAX_BROWSERS**: Maximum concurrent browser instances
- **Queue Management**: Requests queued when at capacity
- **Auto-scaling**: Dynamic resource allocation based on load

### Security Model
- Sandboxed Chrome processes
- Network isolation per instance
- Automatic cleanup of browser data
- Rate limiting and access control

## Performance Tuning

### Memory Optimization
```bash
# Increase shared memory for heavy workloads
./manage.sh --action install --max-browsers 20

# Monitor memory usage
./manage.sh --action status
```

### Timeout Configuration
```bash
# Increase timeout for slow pages
./manage.sh --action install --timeout 60000

# Per-request timeout override
curl -X POST http://localhost:4110/screenshot \
  -d '{"url": "...", "timeout": 45000}'
```

### Browser Pooling
```bash
# Check pool status
curl http://localhost:4110/pressure

# Response:
{
  "running": 2,
  "queued": 0,
  "maxConcurrent": 5,
  "cpu": 0.15,
  "memory": 0.42
}
```

## Testing & Examples

### Basic Health Check
```bash
./manage.sh --action test
```

### Run All Examples
```bash
./manage.sh --action usage --usage-type all
```

### Specific Examples
```bash
# Screenshot example
./manage.sh --action screenshot --url https://google.com

# PDF generation
./manage.sh --action pdf --url https://example.com --output document.pdf

# Web scraping
./manage.sh --action scrape --url https://news.ycombinator.com --selector ".storylink"
```

### Test Output Files
When running usage examples, output files are managed automatically:

- **Default Location**: `./data/test-outputs/browserless/`
- **Files Created**:
  - `screenshot_test.png` - Test screenshot captures
  - `document_test.pdf` - Test PDF generation
  - `scrape_test.html` - Test web scraping output
- **Automatic Cleanup**: Files are removed after usage examples complete
- **Custom Output**: Use `--output /path/to/file` to specify custom location (no cleanup)

## Backup & Recovery

### Create Backup
```bash
# Manual backup
./manage.sh --action create-backup --label "before_upgrade"

# Automatic backups are created before risky operations
```

### Restore from Backup
```bash
# List available backups
./manage.sh --action list-backups

# Recover from latest backup
./manage.sh --action recover
```

## Troubleshooting

### Container Won't Start
```bash
# Check Docker logs
./manage.sh --action logs --lines 100

# Verify port availability
sudo lsof -i :4110

# Reset and reinstall
./manage.sh --action uninstall --force yes
./manage.sh --action install
```

### High Memory Usage
```bash
# Reduce concurrent browsers
./manage.sh --action uninstall
./manage.sh --action install --max-browsers 3

# Monitor pressure
watch curl -s http://localhost:4110/pressure
```

### Timeout Errors
```bash
# Increase timeout
export TIMEOUT=60000
./manage.sh --action restart

# Check network connectivity
docker exec vrooli-browserless ping -c 1 google.com
```

## Business Value

### Revenue Potential
- **Automation Services**: $5K-15K/month for web scraping solutions
- **Visual Testing**: $3K-8K/month for regression testing services
- **PDF Generation**: $2K-5K/month for document generation APIs
- **Data Extraction**: $10K-30K/month for competitive intelligence

### Use Cases
- E-commerce price monitoring
- Social media content aggregation
- Website change detection
- Automated testing pipelines
- Document generation services
- SEO monitoring tools

### Integration Opportunities
- Combine with AI for visual analysis
- Feed data to business intelligence tools
- Power notification systems
- Enable compliance monitoring

## Monitoring

### Real-time Metrics
```bash
# View metrics
curl http://localhost:4110/metrics

# Monitor continuously
./manage.sh --action monitor
```

### Logging
```bash
# View recent logs
./manage.sh --action logs

# Follow logs
docker logs -f vrooli-browserless
```

### Performance Dashboard
Access the built-in dashboard at http://localhost:4110 for:
- Active sessions
- Queue status
- Resource utilization
- Request history

## Advanced Configuration

### Environment Variables
```bash
# Custom configuration
CONCURRENT=10              # Max concurrent browsers
TIMEOUT=30000              # Default timeout (ms)
PREBOOT_CHROME=true        # Pre-warm browsers
KEEP_ALIVE=true           # Keep browsers alive
WORKSPACE_DELETE_EXPIRED=true  # Auto-cleanup
WORKSPACE_EXPIRE_DAYS=7    # Cleanup after 7 days
```

### Network Configuration
```bash
# Use custom network
docker network create custom-network
./manage.sh --action install --network custom-network

# Connect to existing services
docker network connect vrooli-network other-container
```

### Volume Mounts
```bash
# Custom workspace directory
mkdir -p /data/browserless
./manage.sh --action install --data-dir /data/browserless
```

## Management Commands

| Command | Description |
|---------|-------------|
| `./manage.sh --action install` | Install Browserless service |
| `./manage.sh --action uninstall` | Remove service completely |
| `./manage.sh --action start` | Start the container |
| `./manage.sh --action stop` | Stop the container |
| `./manage.sh --action restart` | Restart the container |
| `./manage.sh --action status` | Check service health |
| `./manage.sh --action logs` | View service logs |
| `./manage.sh --action info` | Show service information |
| `./manage.sh --action version` | Display version |
| `./manage.sh --action test` | Run functionality tests |
| `./manage.sh --action usage` | Show usage examples menu |
| `./manage.sh --action screenshot` | Take a screenshot |
| `./manage.sh --action pdf` | Generate PDF |
| `./manage.sh --action scrape` | Scrape content |
| `./manage.sh --action create-backup` | Create backup |
| `./manage.sh --action recover` | Restore from backup |

## 📚 Documentation

- **[Installation Guide](docs/INSTALLATION.md)** - Prerequisites, setup options, verification
- **[Configuration](docs/CONFIGURATION.md)** - Settings, environment variables, customization
- **[API Reference](docs/API.md)** - Endpoints, parameters, request/response examples
- **[Usage Guide](docs/USAGE.md)** - Common tasks, workflows, integration patterns
- **[Troubleshooting](docs/TROUBLESHOOTING.md)** - Common issues, solutions, debugging
- **[Advanced Topics](docs/ADVANCED.md)** - Architecture, scaling, security, performance

## Security Considerations

- Run in isolated network when possible
- Implement rate limiting for public endpoints
- Regularly update the Browserless image
- Monitor for unusual activity patterns
- Use authentication for production deployments

## Support & Resources

- [Official Documentation](https://www.browserless.io/docs/)
- [API Reference](https://www.browserless.io/docs/api)
- [GitHub Repository](https://github.com/browserless/browserless)
- [Performance Guide](https://www.browserless.io/docs/performance)
- [Chrome DevTools Protocol](https://chromedevtools.github.io/devtools-protocol/)

---

*Browserless is a critical component for browser automation in the Vrooli ecosystem, enabling sophisticated web interactions and content extraction at scale.*