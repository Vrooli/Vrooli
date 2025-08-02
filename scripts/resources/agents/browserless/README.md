# Browserless Resource

Chrome-as-a-Service providing headless browser automation via REST API. Browserless handles Chrome lifecycle management, prevents memory leaks, and provides simple endpoints for screenshots, PDFs, web scraping, and browser automation.

## 🚀 Quick Start

```bash
# Install
./manage.sh --action install

# Verify installation
./manage.sh --action status

# Test screenshot capture  
./manage.sh --action usage --usage-type screenshot --url https://example.com --output test.png
```

## 📋 Features

- 📸 **Screenshots** - Full-page captures with validation for AI safety
- 📄 **PDF Generation** - Convert web pages to PDF with customization
- 🔍 **Web Scraping** - Extract data from JavaScript-heavy sites
- 🤖 **Browser Automation** - Execute custom Puppeteer code via API
- 📊 **Resource Management** - Built-in queuing and concurrency control
- 🛡️ **Production Ready** - Health monitoring and automatic cleanup

## 📚 Documentation

- **[Installation Guide](docs/INSTALLATION.md)** - Prerequisites, setup options, verification
- **[Configuration](docs/CONFIGURATION.md)** - Settings, environment variables, customization
- **[API Reference](docs/API.md)** - Endpoints, parameters, request/response examples
- **[Usage Guide](docs/USAGE.md)** - Common tasks, workflows, integration patterns
- **[Troubleshooting](docs/TROUBLESHOOTING.md)** - Common issues, solutions, debugging
- **[Advanced Topics](docs/ADVANCED.md)** - Architecture, scaling, security, performance

## 🔧 Management Commands

| Command | Description |
|---------|-------------|
| `./manage.sh --action install` | Install Browserless service |
| `./manage.sh --action status` | Check service health |
| `./manage.sh --action usage` | Show usage examples menu |
| `./manage.sh --action logs` | View service logs |
| `./manage.sh --action restart` | Restart service |
| `./manage.sh --action uninstall` | Remove service completely |

## 📁 Test Output Files

When running usage examples, output files are managed automatically:

- **Default Location**: `./testing/test-outputs/browserless/`
- **Files Created**:
  - `screenshot_test.png` - Test screenshot captures
  - `document_test.pdf` - Test PDF generation
  - `scrape_test.html` - Test web scraping output
- **Automatic Cleanup**: Files are removed after usage examples complete
- **Custom Output**: Use `--output /path/to/file` to specify custom location (no cleanup)

### Examples

```bash
# Uses default location, auto-cleanup after completion
./manage.sh --action usage --usage-type screenshot --url https://example.com

# Custom output location, no cleanup
./manage.sh --action usage --usage-type screenshot --url https://example.com --output /tmp/my-screenshot.png

# Configure custom test output directory
export BROWSERLESS_TEST_OUTPUT_DIR="/tmp/browserless-tests"
./manage.sh --action usage --usage-type all
```

### Environment Variables

- `BROWSERLESS_TEST_OUTPUT_DIR` - Override default test output directory (default: `./testing/test-outputs/browserless`)

## 💡 Common Use Cases

### 1. Capture Dashboard Screenshot
```bash
# Safe screenshot with validation
./manage.sh --action usage --usage-type screenshot \
  --url http://localhost:3000/dashboard \
  --output dashboard.png
```

### 2. Generate PDF Report
```bash
# Convert web page to PDF
curl -X POST http://localhost:4110/chrome/pdf \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com/report", "options": {"format": "A4"}}' \
  -o report.pdf
```

### 3. Extract Web Data
```bash
# Scrape specific elements
curl -X POST http://localhost:4110/chrome/scrape \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://news.site.com",
    "elements": [{"selector": "h2", "property": "innerText"}]
  }'
```

### 4. Check System Load
```bash
# Monitor browser pool status
curl http://localhost:4110/pressure | jq .
```

## 🔗 Integration Examples

### With n8n Automation
Create visual workflows that capture screenshots, generate PDFs, or scrape data. See [n8n integration example](examples/n8n-integration/).

### With AI Services
Combine with Ollama for visual analysis or Agent-S2 for complex automation. The screenshot validation ensures AI tools receive valid images.

### With Node-RED
Build real-time monitoring dashboards with automated screenshot captures. See [Node-RED flow example](examples/node-red-flow/).

## ⚡ Quick Tips

- **For localhost URLs**: Use Docker bridge IP (usually `172.17.0.1`) instead of `localhost`
- **For slow sites**: Increase timeout with `--timeout 60000` during install
- **For heavy load**: Monitor pressure endpoint and adjust `--max-browsers`
- **For debugging**: Check logs with `./manage.sh --action logs`

## 🛟 Getting Help

- **Installation issues** → [Installation Guide](docs/INSTALLATION.md)
- **API questions** → [API Reference](docs/API.md)  
- **Performance tuning** → [Configuration Guide](docs/CONFIGURATION.md#performance-tuning)
- **Common problems** → [Troubleshooting Guide](docs/TROUBLESHOOTING.md)
- **Advanced scenarios** → [Advanced Topics](docs/ADVANCED.md)

## 📊 Service Information

- **Category**: Agents
- **Default Port**: 4110
- **Container**: `browserless`
- **Health Check**: `http://localhost:4110/pressure`
- **Metrics**: `http://localhost:4110/metrics`

## 🔗 External Resources

- [Browserless.io Documentation](https://www.browserless.io/docs/)
- [API Reference](https://www.browserless.io/docs/api)
- [Examples Directory](examples/)
- [Chrome DevTools Protocol](https://chromedevtools.github.io/devtools-protocol/)