[← Back to README](../README.md) | [Documentation Index](./README.md)

# Browserless Usage Guide

This guide covers common workflows and best practices for using Browserless in the Vrooli ecosystem.

## Table of Contents

- [Management Commands](#management-commands)
- [Usage Examples](#usage-examples)
- [Common Workflows](#common-workflows)
- [Integration Patterns](#integration-patterns)
- [Use Cases](#use-cases)
- [Tips and Tricks](#tips-and-tricks)
- [Comparison with Alternatives](#comparison-with-alternatives)

## Management Commands

### Basic Operations

```bash
# Check service status
./manage.sh --action status

# Start service (if stopped)
./manage.sh --action start

# Stop service
./manage.sh --action stop

# Restart service
./manage.sh --action restart

# View real-time logs
./manage.sh --action logs

# Get service information
./manage.sh --action info
```

### Service Lifecycle

```bash
# Install or reinstall
./manage.sh --action install

# Completely remove service
./manage.sh --action uninstall

# Update to latest version
docker pull ghcr.io/browserless/chrome:latest
./manage.sh --action restart
```

## Usage Examples

The management script includes built-in examples to test Browserless functionality:

### Interactive Usage Menu

```bash
# Show all available usage examples
./manage.sh --action usage
```

This displays an interactive menu with all available examples.

### Screenshot Examples

```bash
# Basic screenshot
./manage.sh --action usage --usage-type screenshot

# Screenshot specific URL
./manage.sh --action usage --usage-type screenshot --url https://github.com --output github.png

# Screenshot with validation (recommended for automation)
./manage.sh --action usage --usage-type screenshot --url https://localhost:8080 --output dashboard.png
```

### PDF Generation

```bash
# Basic PDF generation
./manage.sh --action usage --usage-type pdf

# Convert specific page to PDF
./manage.sh --action usage --usage-type pdf --url https://wikipedia.org --output wiki.pdf
```

### Web Scraping

```bash
# Basic scraping example
./manage.sh --action usage --usage-type scrape

# Scrape specific site
./manage.sh --action usage --usage-type scrape --url https://news.ycombinator.com
```

### System Monitoring

```bash
# Check browser pool status
./manage.sh --action usage --usage-type pressure

# Returns JSON with system load info
{
  "isAvailable": true,
  "queued": 0,
  "running": 1,
  "maxConcurrent": 5
}
```

### Run All Examples

```bash
# Test all features at once
./manage.sh --action usage --usage-type all

# Test all features with custom URL
./manage.sh --action usage --usage-type all --url https://example.com
```

## Common Workflows

### 1. Automated Testing Workflow

```bash
# Capture before/after screenshots for visual regression
BEFORE_URL="http://localhost:3000/old-version"
AFTER_URL="http://localhost:3000/new-version"

# Capture baseline
./manage.sh --action usage --usage-type screenshot \
  --url "$BEFORE_URL" \
  --output before.png

# Deploy changes...

# Capture new version
./manage.sh --action usage --usage-type screenshot \
  --url "$AFTER_URL" \
  --output after.png

# Compare with image diff tool
compare before.png after.png diff.png
```

### 2. Report Generation Workflow

```bash
# Generate weekly reports from dashboard
DASHBOARD_URL="http://grafana:3000/dashboard/db/weekly-metrics"
OUTPUT_DIR="/reports/$(date +%Y-%m-%d)"

mkdir -p "$OUTPUT_DIR"

# Screenshot dashboard
./manage.sh --action usage --usage-type screenshot \
  --url "$DASHBOARD_URL" \
  --output "$OUTPUT_DIR/dashboard.png"

# Generate PDF report
./manage.sh --action usage --usage-type pdf \
  --url "$DASHBOARD_URL" \
  --output "$OUTPUT_DIR/report.pdf"
```

### 3. Content Monitoring Workflow

```bash
# Monitor website changes
URL="https://status.example.com"
SELECTOR=".status-indicator"

# Scrape current status
STATUS=$(curl -s -X POST http://localhost:4110/chrome/scrape \
  -H "Content-Type: application/json" \
  -d "{
    \"url\": \"$URL\",
    \"elements\": [{
      \"selector\": \"$SELECTOR\",
      \"property\": \"innerText\"
    }]
  }" | jq -r '.[0].results[0]')

echo "Current status: $STATUS"
```

## Integration Patterns

### With n8n Automation

Create n8n workflows that use Browserless:

```javascript
// n8n HTTP Request node configuration
{
  "method": "POST",
  "url": "http://browserless:4110/chrome/screenshot",
  "headers": {
    "Content-Type": "application/json"
  },
  "body": {
    "url": "{{ $json.targetUrl }}",
    "options": {
      "fullPage": true
    }
  },
  "responseType": "file"
}
```

### With Node-RED

Node-RED flow for automated screenshots:

```javascript
// Function node to prepare request
msg.payload = {
  url: msg.payload.url,
  options: {
    fullPage: true,
    type: "png"
  }
};
msg.headers = {
  "Content-Type": "application/json"
};
return msg;
```

### With AI Services

Combine with Ollama for visual analysis:

```bash
# Capture screenshot
./manage.sh --action usage --usage-type screenshot \
  --url https://example.com \
  --output site.png

# Analyze with Ollama vision model
curl -X POST http://localhost:11434/api/generate \
  -d '{
    "model": "llava",
    "prompt": "Describe this website screenshot",
    "images": ["site.png"]
  }'
```

## Use Cases

### 1. Automated Testing
- **Visual Regression**: Compare screenshots before/after deployments
- **Cross-Browser Testing**: Verify rendering across different viewports
- **E2E Test Artifacts**: Capture screenshots during test failures

### 2. Web Scraping
- **Dynamic Content**: Scrape JavaScript-rendered content
- **Data Collection**: Extract structured data from websites
- **Price Monitoring**: Track product prices across e-commerce sites

### 3. Report Generation
- **Dashboard Exports**: Convert live dashboards to PDF reports
- **Invoice Generation**: Create PDFs from web-based templates
- **Documentation**: Capture current state of web applications

### 4. Content Verification
- **Uptime Monitoring**: Visual verification of service availability
- **Content Integrity**: Ensure pages render correctly
- **Compliance Checks**: Verify required elements are present

### 5. Development Support
- **Bug Reports**: Capture screenshots for issue tracking
- **Design Reviews**: Generate visual artifacts for review
- **Performance Analysis**: Capture loading states and timings

## Tips and Tricks

### 1. Handling Dynamic Content

Wait for specific elements:
```bash
curl -X POST http://localhost:4110/chrome/screenshot \
  -d '{
    "url": "https://spa-app.com",
    "waitFor": ".content-loaded",
    "options": {"fullPage": true}
  }'
```

### 2. Dealing with Authentication

Use cookies for authenticated pages:
```bash
curl -X POST http://localhost:4110/chrome/screenshot \
  -d '{
    "url": "https://app.com/dashboard",
    "cookies": [{
      "name": "auth_token",
      "value": "your-token",
      "domain": ".app.com"
    }]
  }'
```

### 3. Optimizing Performance

- Reuse browser contexts when possible
- Set appropriate timeouts for your use case
- Monitor the pressure endpoint
- Use concurrent limits wisely

### 4. Debugging Issues

Enable verbose logging:
```bash
# View detailed logs
docker logs -f browserless

# Check browser console errors
curl -X POST http://localhost:4110/chrome/content \
  -d '{"url": "https://problem-site.com"}' \
  | grep -i error
```

## Comparison with Alternatives

### Browserless vs Direct Puppeteer

| Feature | Browserless | Direct Puppeteer |
|---------|-------------|------------------|
| **Setup** | One command | Install Chrome + dependencies |
| **API** | REST endpoints | JavaScript only |
| **Scaling** | Built-in concurrency | Manual management |
| **Memory** | Automatic cleanup | Manual cleanup needed |
| **Monitoring** | Built-in metrics | Custom implementation |
| **Language Support** | Any via REST | JavaScript/TypeScript |
| **Resource Usage** | Optimized | Requires tuning |
| **Error Handling** | Automatic retries | Manual implementation |

### When to Use Browserless

✅ **Use Browserless when:**
- You need language-agnostic browser automation
- You want managed browser lifecycle
- You need built-in queuing and concurrency
- You prefer REST APIs over programmatic control
- You want easy scaling and monitoring

❌ **Consider alternatives when:**
- You need fine-grained browser control
- You're already in a Node.js environment
- You need custom browser extensions
- You require specific Chrome versions

---
**See also:** [API Reference](./API.md) | [Configuration](./CONFIGURATION.md) | [Troubleshooting](./TROUBLESHOOTING.md)