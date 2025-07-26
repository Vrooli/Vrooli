# Browserless Resource

Browserless.io is a web browser automation service that runs headless Chrome as a service, providing a simple REST API for browser automation tasks. This resource provides automated installation, configuration, and management of Browserless for the Vrooli project.

## Overview

- **Category**: Agents
- **Default Port**: 4110 (mapped to container's internal port 3000)
- **Service Type**: Docker Container
- **Container Name**: browserless
- **Documentation**: [browserless.io/docs](https://www.browserless.io/docs/)

## What is Browserless?

Browserless.io provides Chrome-as-a-Service, handling the complex parts of running headless Chrome at scale:
- Manages Chrome lifecycle (prevents memory leaks and zombie processes)
- Provides simple REST API endpoints for common tasks
- Handles concurrent browser sessions with queuing
- Built-in debugging tools and session recording
- Production-ready with health monitoring

## Features

- 📸 **Screenshots**: Capture full-page or element screenshots
- 📄 **PDF Generation**: Convert web pages to PDF with customization
- 🌐 **Content Extraction**: Get HTML content or text from pages
- 🤖 **Function Execution**: Run custom Puppeteer code via API
- 🔍 **Web Scraping**: Extract structured data from websites
- 📊 **Performance Metrics**: Monitor browser resource usage
- 🎯 **Session Management**: Handle multiple concurrent sessions
- 🛡️ **Built-in Security**: Sandboxed browser execution

## Prerequisites

- Docker installed and running
- User in docker group or sudo access
- 2GB+ RAM available
- 1GB+ free disk space
- Port 4110 available (or custom port)

## Installation

### Quick Install
```bash
# Install Browserless service
./manage.sh --action install

# Install with custom settings
./manage.sh --action install --max-browsers 10 --timeout 60000
```

### Installation Options
```bash
./manage.sh --action install \
  --port 4110 \              # Service port (default: 4110)
  --max-browsers 5 \         # Maximum concurrent sessions (default: 5)
  --timeout 30000 \          # Default timeout in ms (default: 30000)
  --headless yes             # Run in headless mode (default: yes)
```

## Usage

### Management Commands
```bash
# Check status
./manage.sh --action status

# Start service
./manage.sh --action start

# Stop service
./manage.sh --action stop

# Restart service
./manage.sh --action restart

# View logs
./manage.sh --action logs

# Service info
./manage.sh --action info

# Uninstall service
./manage.sh --action uninstall
```

### Usage Examples
The management script includes built-in examples to test Browserless functionality:

```bash
# Show all available usage examples
./manage.sh --action usage

# Test screenshot API
./manage.sh --action usage --usage-type screenshot
./manage.sh --action usage --usage-type screenshot --url https://github.com --output github.png

# Test PDF generation
./manage.sh --action usage --usage-type pdf
./manage.sh --action usage --usage-type pdf --url https://wikipedia.org --output wiki.pdf

# Test web scraping
./manage.sh --action usage --usage-type scrape
./manage.sh --action usage --usage-type scrape --url https://news.ycombinator.com

# Check browser pool status
./manage.sh --action usage --usage-type pressure

# Run all usage examples
./manage.sh --action usage --usage-type all
./manage.sh --action usage --usage-type all --url https://example.com
```

These examples help you:
- Verify Browserless is working correctly after installation  
- Learn the API endpoints with working examples
- Test core browser automation features (screenshots, PDFs, scraping)
- Generate sample outputs for development

## API Endpoints

### Core Endpoints

#### Screenshot - `/chrome/screenshot`
Capture screenshots of web pages with built-in validation:
```bash
curl -X POST http://localhost:4110/chrome/screenshot \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://example.com",
    "options": {
      "fullPage": true,
      "type": "png"
    }
  }' \
  --output screenshot.png
```

**Important**: The Browserless resource includes enhanced screenshot validation to prevent issues with AI tools:
- Validates HTTP response codes (only accepts 200 OK)
- Checks file size (minimum 1KB) to reject error text responses
- Verifies MIME type to ensure actual image content
- Automatically cleans up invalid files

For AI/automation use cases, use the safe screenshot wrapper:
```bash
# Safe screenshot capture with validation
./manage.sh --action usage --usage-type screenshot --url https://example.com --output safe.png
```

#### PDF Generation - `/chrome/pdf`
Convert web pages to PDF:
```bash
curl -X POST http://localhost:4110/chrome/pdf \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://example.com",
    "options": {
      "format": "A4",
      "printBackground": true,
      "margin": {
        "top": "1cm",
        "bottom": "1cm"
      }
    }
  }' \
  --output document.pdf
```

#### Content Extraction - `/content`
Get page content as HTML or text:
```bash
curl -X POST http://localhost:4110/content \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://example.com"
  }'
```

#### Function Execution - `/function`
Run custom Puppeteer code:
```bash
curl -X POST http://localhost:4110/function \
  -H "Content-Type: application/json" \
  -d '{
    "code": "async ({ page }) => {
      await page.goto(\"https://example.com\");
      const title = await page.title();
      return { title };
    }"
  }'
```

#### Web Scraping - `/scrape`
Extract structured data:
```bash
curl -X POST http://localhost:4110/scrape \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://example.com",
    "elements": [{
      "selector": "h1",
      "property": "textContent"
    }]
  }'
```

### Monitoring Endpoints

#### System Pressure - `/pressure`
Check system load and availability:
```bash
curl http://localhost:4110/pressure
```

Response:
```json
{
  "isAvailable": true,
  "queued": 0,
  "recentlyRejected": 0,
  "running": 1,
  "maxConcurrent": 5,
  "maxQueued": 10,
  "cpu": 0.15,
  "memory": 0.45
}
```

#### Metrics - `/metrics`
Prometheus-compatible metrics:
```bash
curl http://localhost:4110/metrics
```

## Advanced Usage

### Authentication with Screenshots
```bash
# Screenshot with cookies
curl -X POST http://localhost:4110/chrome/screenshot \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://app.example.com/dashboard",
    "cookies": [{
      "name": "session",
      "value": "abc123",
      "domain": ".example.com"
    }],
    "options": {
      "fullPage": true
    }
  }' \
  --output dashboard.png
```

### Complex Scraping
```bash
# Multi-step scraping with wait conditions
curl -X POST http://localhost:4110/function \
  -H "Content-Type: application/json" \
  -d '{
    "code": "async ({ page }) => {
      await page.goto(\"https://example.com/search\");
      await page.type(\"#search-input\", \"query\");
      await page.click(\"#search-button\");
      await page.waitForSelector(\".results\");
      
      const results = await page.evaluate(() => {
        return Array.from(document.querySelectorAll(\".result-item\")).map(item => ({
          title: item.querySelector(\"h3\")?.textContent,
          link: item.querySelector(\"a\")?.href,
          description: item.querySelector(\"p\")?.textContent
        }));
      });
      
      return { results };
    }"
  }'
```

### PDF with Custom Headers/Footers
```bash
curl -X POST http://localhost:4110/chrome/pdf \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://example.com/report",
    "options": {
      "format": "A4",
      "displayHeaderFooter": true,
      "headerTemplate": "<div style=\"font-size: 10px; text-align: center;\">Report Header</div>",
      "footerTemplate": "<div style=\"font-size: 10px; text-align: center;\">Page <span class=\"pageNumber\"></span> of <span class=\"totalPages\"></span></div>",
      "margin": {
        "top": "2cm",
        "bottom": "2cm"
      }
    }
  }' \
  --output report.pdf
```

### Mobile Device Emulation
```bash
# Screenshot as mobile device
curl -X POST http://localhost:4110/chrome/screenshot \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://example.com",
    "viewport": {
      "width": 375,
      "height": 812,
      "deviceScaleFactor": 3,
      "isMobile": true,
      "hasTouch": true
    },
    "userAgent": "Mozilla/5.0 (iPhone; CPU iPhone OS 14_0 like Mac OS X)"
  }' \
  --output mobile-screenshot.png
```

## Configuration

### Environment Variables
The service uses these environment variables (set in the Docker container):
- `CONCURRENT`: Maximum concurrent browser sessions (default: 5)
- `TIMEOUT`: Default timeout in milliseconds (default: 30000)
- `ENABLE_DEBUGGER`: Enable Chrome DevTools debugger (default: false)
- `PREBOOT_CHROME`: Keep Chrome instances warm (default: true)

### Vrooli Integration
Browserless is automatically configured in `~/.vrooli/resources.local.json`:
```json
{
  "services": {
    "agents": {
      "browserless": {
        "enabled": true,
        "baseUrl": "http://localhost:4110",
        "healthCheck": {
          "intervalMs": 60000,
          "timeoutMs": 5000
        },
        "features": {
          "screenshots": true,
          "pdf": true,
          "scraping": true,
          "automation": true
        },
        "browser": {
          "maxConcurrency": 5,
          "headless": true,
          "timeout": 30000
        }
      }
    }
  }
}
```

## AI/Automation Safety Features

The Browserless resource includes special safety features designed for AI and automation tools that may attempt to read screenshot files:

### Screenshot Validation
When capturing screenshots, the service performs multiple validation checks:

1. **HTTP Status Validation**: Only accepts successful (200 OK) responses
2. **File Size Check**: Rejects files smaller than 1KB (typical of error messages)
3. **MIME Type Verification**: Ensures the output is actually an image file
4. **Automatic Cleanup**: Removes invalid files to prevent accidental reads

### Safe Screenshot Function
For AI tools that need to read screenshots, use the `browserless::safe_screenshot` function:

```bash
# From scripts or automation
source /path/to/browserless/lib/api.sh
browserless::safe_screenshot "http://localhost:8080" "/tmp/screenshot.png"
if [[ $? -eq 0 ]]; then
    # Safe to read the screenshot file
    echo "Screenshot captured and validated"
else
    echo "Screenshot failed validation - file not created"
fi
```

### Why This Matters
When Browserless fails to capture a page (e.g., 404, 500 errors), it may return error text instead of an image. Without validation:
- A `.png` file might contain "Not Found" or HTML error text
- AI tools attempting to read such files as images can experience context corruption
- This can break automation workflows and cause unexpected behavior

The validation ensures that only genuine image files are saved, preventing these issues.

## Troubleshooting

### Common Issues

#### Container Won't Start
```bash
# Check logs
docker logs browserless

# Verify shared memory
df -h /dev/shm

# Check if port is in use
sudo lsof -i :4110
```

#### Chrome Crashes / Out of Memory
```bash
# Restart with more shared memory
docker run -d --shm-size=4gb ...

# Or limit concurrent sessions
./manage.sh --action install --max-browsers 3
```

#### Timeout Errors
```bash
# Increase default timeout
./manage.sh --action install --timeout 60000

# Or per-request timeout
curl -X POST http://localhost:4110/chrome/screenshot \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://slow-site.com",
    "waitUntil": "networkidle0",
    "timeout": 60000
  }'
```

#### Screenshot Validation Failures
If screenshots are failing validation:
```bash
# Check the actual error
./manage.sh --action usage --usage-type screenshot --url https://your-site.com

# Common validation failures:
# - "HTTP status: 500" - Target site returned server error
# - "HTTP status: 404" - Page not found
# - "too small" - Response was error text, not an image
# - "not an image" - Server returned HTML/text instead of image

# For Docker networking issues (can't reach host services):
# Use Docker bridge IP instead of localhost
ip route | grep docker0  # Usually 172.17.0.1
./manage.sh --action usage --usage-type screenshot --url http://172.17.0.1:8080
```

### Performance Optimization

#### Monitor Resource Usage
```bash
# Check container stats
docker stats browserless

# Check pressure endpoint
curl http://localhost:4110/pressure
```

#### Optimize for High Load
```bash
# Install with optimized settings
./manage.sh --action install \
  --max-browsers 10 \
  --timeout 20000
```

## Maintenance

### Regular Tasks
- Monitor container logs: `docker logs -f browserless`
- Check memory usage: `docker stats browserless`
- Monitor pressure: `curl http://localhost:4110/pressure`
- Clean up disk space (Browserless handles this automatically)

### Updates
```bash
# Pull latest image
docker pull ghcr.io/browserless/chrome:latest

# Restart service
./manage.sh --action restart
```

## Security Considerations

- Browserless runs Chrome in a sandboxed environment
- Each session is isolated
- No persistent storage between sessions
- Consider adding authentication for production use
- Monitor for excessive resource usage

## Use Cases

### 1. Automated Testing
- Screenshot regression testing
- PDF generation testing
- Cross-browser compatibility checks

### 2. Web Scraping
- Extract data from JavaScript-heavy sites
- Monitor website changes
- Collect structured data

### 3. Report Generation
- Convert dashboards to PDFs
- Generate invoices from web templates
- Create documentation snapshots

### 4. Content Verification
- Check if pages render correctly
- Verify deployed changes
- Monitor for visual bugs

## Comparison with Direct Puppeteer

| Feature | Browserless | Direct Puppeteer |
|---------|-------------|------------------|
| Setup | One command | Install Chrome + deps |
| API | REST endpoints | JavaScript only |
| Scaling | Built-in concurrency | Manual management |
| Memory | Automatic cleanup | Manual cleanup needed |
| Monitoring | Built-in metrics | Custom implementation |
| Language Support | Any via REST | JavaScript/TypeScript |

## Related Resources

- [Browserless Documentation](https://www.browserless.io/docs/)
- [API Reference](https://www.browserless.io/docs/api)
- [Chrome DevTools Protocol](https://chromedevtools.github.io/devtools-protocol/)
- [Puppeteer Documentation](https://pptr.dev)