# Browserless Chrome Service

High-performance headless Chrome automation service for web scraping, screenshot generation, PDF creation, and browser automation at scale.

## Quick Reference

| Component | Details |
|-----------|---------|
| **Port** | 4110 |
| **Container** | vrooli-browserless |
| **Status** | `./cli.sh status` |
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
./cli.sh install

# Install with custom configuration
./cli.sh install --max-browsers 10 --timeout 60000

# Install with non-headless mode (for debugging)
./cli.sh install --headless no
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
- 🔌 **Resource Adapters** - Provide UI automation fallbacks for other resources (NEW!)

## When to Use

### Use Browserless When You Need:
- **Browser Automation at Scale** - Multiple concurrent browser operations
- **JavaScript Rendering** - Pages that require JavaScript execution
- **Visual Testing** - Screenshot comparison and visual regression testing
- **PDF Generation** - High-quality PDF conversion with browser rendering
- **Complex Scraping** - Sites with dynamic content or anti-bot measures
- **As a Backend for BAS** - Headless Chrome/CDP backend for Vrooli Ascension executions

### Alternatives to Consider:
- **Simple HTTP Requests**: Use curl/wget for static content
- **API Access**: Prefer official APIs when available
- **Puppeteer/Playwright**: For programmatic control in Node.js
- **Selenium**: For cross-browser testing requirements

## Resource Adapters (NEW!)

Browserless now provides **UI automation adapters** for other resources, enabling fallback interfaces when APIs are unavailable or broken. This creates an antifragile system where resources can continue operating even during failures.

### Available Adapters

- **vault** - Manage secrets and policies through UI automation (preview)

### Using Adapters

The new "for" pattern allows browserless to act as an adapter:

```bash
# Add Vault secrets via UI
resource-browserless for vault add-secret secret/myapp key=value

# List available adapters
resource-browserless for --help
```

## Browser Pool Management (NEW!)

Browserless now includes automatic browser pool scaling to handle varying loads efficiently with enhanced reliability features:

### Reliability Features
- **Automatic Crash Recovery**: Detects and recovers from browser crashes automatically
- **Health Monitoring**: Continuous health checks integrated into auto-scaler
- **Session Cleanup**: Clears stuck sessions when high rejection rates detected
- **Pre-warming After Recovery**: Automatically pre-warms pool after recovery for immediate readiness
- **Portable Implementation**: No external dependencies (bc replaced with awk)

### Pool Management Commands

```bash
# Show pool statistics
resource-browserless pool

# Start auto-scaler
resource-browserless pool start

# Check and recover unhealthy pool
resource-browserless pool recover

# Stop auto-scaler
resource-browserless pool stop

# Get pool metrics in JSON
resource-browserless pool metrics

# Enable auto-scaling on startup
export BROWSERLESS_ENABLE_AUTOSCALING=true
resource-browserless manage start
```

### Pool Configuration

```bash
# Auto-scaling configuration
export BROWSERLESS_POOL_MIN_SIZE=2         # Minimum pool size
export BROWSERLESS_POOL_MAX_SIZE=20        # Maximum pool size
export BROWSERLESS_POOL_SCALE_UP_THRESHOLD=70    # Scale up at 70% utilization
export BROWSERLESS_POOL_SCALE_DOWN_THRESHOLD=30  # Scale down at 30% utilization
export BROWSERLESS_POOL_SCALE_STEP=2       # Add/remove 2 instances at a time
export BROWSERLESS_POOL_MONITOR_INTERVAL=10      # Check every 10 seconds
export BROWSERLESS_POOL_COOLDOWN_PERIOD=30       # Wait 30s between scaling
```

## Performance Benchmarks (NEW!)

Track and compare performance over time with the new benchmark suite:

### Running Benchmarks

```bash
# Run all benchmarks
resource-browserless benchmark

# Run specific benchmark categories
resource-browserless benchmark navigation
resource-browserless benchmark screenshots
resource-browserless benchmark extraction

# Show benchmark summary
resource-browserless benchmark summary

# Compare two benchmark runs
resource-browserless benchmark compare file1.json file2.json
```

### Benchmark Metrics

- **Navigation**: Page load times for simple and complex pages
- **Screenshots**: Time to capture regular and full-page screenshots  
- **Extraction**: Text, attribute, and element detection performance

Benchmarks track min, max, average, median, and 95th percentile metrics.

### Why Use Adapters?

- **Resilience**: Continue operations when APIs fail
- **Cost Optimization**: Switch between API and UI based on usage
- **Feature Access**: Use UI-only features not exposed via API
- **Emergency Access**: Bypass rate limits or authentication issues

See [docs/ADAPTERS.md](docs/ADAPTERS.md) for complete documentation.

## Integration Examples

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
./cli.sh install --max-browsers 20

# Monitor memory usage
./cli.sh status
```

### Timeout Configuration
```bash
# Increase timeout for slow pages
./cli.sh install --timeout 60000

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
./cli.sh test
```

### Run All Examples
```bash
./cli.sh usage --usage-type all
```

### Specific Examples
```bash
# Screenshot example
./cli.sh screenshot --url https://google.com

# PDF generation
./cli.sh pdf --url https://example.com --output document.pdf

# Web scraping
./cli.sh scrape --url https://news.ycombinator.com --selector ".storylink"
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
./cli.sh create-backup --label "before_upgrade"

# Automatic backups are created before risky operations
```

### Restore from Backup
```bash
# List available backups
./cli.sh list-backups

# Recover from latest backup
./cli.sh recover
```

## Known Issues & Version Selection

### Chrome Process Leak in Latest Version

**Issue**: The `latest` tag (sha256:96cc9039..., published ~7 days ago) has a Chrome process leak that causes process accumulation over time. This leads to "fork: Resource temporarily unavailable" errors after extended use.

**Symptoms**:
- Chrome/defunct processes accumulate in container
- After ~600+ processes: browserless fails to spawn new Chrome instances
- Error messages: "fork: Resource temporarily unavailable" or "pthread_create: Resource temporarily unavailable"
- UI smoke tests fail with "Browserless returned invalid response"

**Root Cause**: Regression introduced in the `latest` version between v2.38.2 release and current `latest` build.

**Solution**: We've pinned to **v2.38.2** which does NOT have this issue.

### Version Testing Results

Extensive testing (50+ consecutive UI smoke tests) confirms:

**v2.38.2 (RECOMMENDED)**:
- ✅ Processes: Stable at 4 total, 0 Chrome/defunct
- ✅ Memory: Stable at ~170-190MiB
- ✅ Health: No degradation over 50+ tests
- ✅ All tests pass consistently

**latest (NOT RECOMMENDED)**:
- ❌ Process leak: ~1 Chrome process per test accumulated
- ❌ After 10 tests: 17 total, 11 Chrome/defunct processes
- ❌ After 600+ tests: Container unusable, requires restart
- ❌ Memory accumulation and eventual failure

### Current Configuration

We've updated the configuration to use v2.38.2:
- Image: `ghcr.io/browserless/chrome:v2.38.2`
- Digest: `sha256:7c206dfaca4781bb477c6495ff2b5477a932c07a79fd7504bab3cf149e3e4be4`
- Published: ~29 days ago
- Status: Stable, no known process leak issues

### Monitoring for Process Leaks

Our enhanced health diagnostics now detect process accumulation:

```bash
# Check for process leaks
vrooli resource browserless status

# Automated detection thresholds:
# - >50 Chrome processes: Status = degraded, restart recommended
# - >20 Chrome processes: Status = warning, monitor for accumulation
# - >10 defunct processes: Status = warning, cleanup issues detected
```

The UI smoke test framework also provides detailed diagnostics when browserless failures occur, including process leak detection and recommended recovery steps.

### Recovery from Process Leak

If you encounter a process leak (shouldn't happen with v2.38.2):

```bash
# Restart browserless to clear accumulated processes
docker restart vrooli-browserless

# Verify clean state
vrooli resource browserless status
# Should show: "Processes: 4 total, 0 Chrome/defunct"

# If using latest tag, downgrade to v2.38.2
# (Already done in current configuration)
```

## Troubleshooting

### Container Won't Start
```bash
# Check Docker logs
./cli.sh logs --lines 100

# Verify port availability
sudo lsof -i :4110

# Reset and reinstall
./cli.sh uninstall --force yes
./cli.sh install
```

### High Memory Usage
```bash
# Reduce concurrent browsers
./cli.sh uninstall
./cli.sh install --max-browsers 3

# Monitor pressure
watch curl -s http://localhost:4110/pressure
```

### Timeout Errors
```bash
# Increase timeout
export TIMEOUT=60000
./cli.sh restart

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
./cli.sh monitor
```

### Logging
```bash
# View recent logs
./cli.sh logs

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
WORKSPACE_DELETE_EXPIRED=true  # Auto-cleanup
WORKSPACE_EXPIRE_DAYS=7    # Cleanup after 7 days

# Note: PREBOOT_CHROME, KEEP_ALIVE, and CHROME_REFRESH_TIME are deprecated
# in browserless v2 and have been removed from our configuration
```

### Network Configuration
```bash
# Use custom network
docker network create custom-network
./cli.sh install --network custom-network

# Connect to existing services
docker network connect vrooli-network other-container
```

### Volume Mounts
```bash
# Custom workspace directory
mkdir -p /data/browserless
./cli.sh install --data-dir /data/browserless
```

## Management Commands

| Command | Description |
|---------|-------------|
| `./cli.sh install` | Install Browserless service |
| `./cli.sh uninstall` | Remove service completely |
| `./cli.sh start` | Start the container |
| `./cli.sh stop` | Stop the container |
| `./cli.sh restart` | Restart the container |
| `./cli.sh status` | Check service health |
| `./cli.sh logs` | View service logs |
| `./cli.sh info` | Show service information |
| `./cli.sh version` | Display version |
| `./cli.sh test` | Run functionality tests |
| `./cli.sh usage` | Show usage examples menu |
| `./cli.sh screenshot` | Take a screenshot |
| `./cli.sh pdf` | Generate PDF |
| `./cli.sh scrape` | Scrape content |
| `./cli.sh create-backup` | Create backup |
| `./cli.sh recover` | Restore from backup |

## 📚 Documentation

- **[Installation Guide](docs/INSTALLATION.md)** - Prerequisites, setup options, verification
- **[Configuration](docs/CONFIGURATION.md)** - Settings, environment variables, customization
- **[API Reference](docs/API.md)** - Endpoints, parameters, request/response examples
- **[Usage Guide](docs/USAGE.md)** - Common tasks and integration patterns
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
