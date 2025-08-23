[← Back to README](../README.md) | [Documentation Index](./README.md)

# Browserless API Reference

## Overview

Browserless provides a REST API for headless Chrome automation. All browser-related endpoints use the `/chrome/` prefix.

**Base URL**: `http://localhost:4110`

## Table of Contents

- [Authentication](#authentication)
- [Core Endpoints](#core-endpoints)
  - [Screenshot](#screenshot---chromescreenshot)
  - [PDF Generation](#pdf-generation---chromepdf)
  - [Content Extraction](#content-extraction---chromecontent)
  - [Web Scraping](#web-scraping---chromescrape)
  - [Custom Function](#custom-function---chromefunction)
- [Monitoring Endpoints](#monitoring-endpoints)
  - [System Pressure](#health-check---pressure)
  - [Metrics](#metrics---metrics)
  - [Configuration](#configuration---config)
- [Advanced Examples](#advanced-examples)
- [AI/Automation Safety](#aiautomation-safety)
- [Error Handling](#error-handling)
- [Best Practices](#best-practices)

## Authentication

The default Browserless setup has no authentication. For production use, set a token in the configuration.

```bash
# With authentication token
curl -X POST http://localhost:4110/chrome/screenshot?token=your-secret-token
```

## Core Endpoints

### Screenshot - `/chrome/screenshot`

Captures a screenshot of a webpage.

**Method**: `POST`

**Request Body**:
```json
{
  "url": "https://example.com",
  "options": {
    "fullPage": false,
    "type": "png",
    "quality": 80,
    "width": 1280,
    "height": 720,
    "deviceScaleFactor": 1
  },
  "gotoOptions": {
    "waitUntil": "networkidle2",
    "timeout": 30000
  },
  "waitFor": 1000
}
```

**Parameters**:
- `url` (required): The URL to screenshot
- `options` (optional): Puppeteer screenshot options
  - `fullPage`: Capture full scrollable page (default: false)
  - `type`: Image format - "png" or "jpeg" (default: "png")
  - `quality`: JPEG quality 0-100 (default: 80)
  - `width`: Viewport width (default: 1280)
  - `height`: Viewport height (default: 720)
- `gotoOptions` (optional): Page navigation options
  - `waitUntil`: When to consider navigation done (default: "networkidle2")
  - `timeout`: Maximum navigation time in ms (default: 30000)
- `waitFor`: Additional wait time in ms after page load

**Response**: Binary image data

**Example**:
```bash
curl -X POST http://localhost:4110/chrome/screenshot \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com", "options": {"fullPage": true}}' \
  -o screenshot.png
```

### Content Extraction - `/chrome/content`

Extracts the HTML content of a webpage.

**Method**: `POST`

**Request Body**:
```json
{
  "url": "https://example.com",
  "gotoOptions": {
    "waitUntil": "networkidle2"
  },
  "waitFor": 1000
}
```

**Parameters**:
- `url` (required): The URL to extract content from
- `gotoOptions` (optional): Page navigation options
- `waitFor` (optional): Additional wait time in ms

**Response**: HTML content as text

**Example**:
```bash
curl -X POST http://localhost:4110/chrome/content \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com"}' \
  -o content.html
```

### PDF Generation - `/chrome/pdf`

Generates a PDF of a webpage.

**Method**: `POST`

**Request Body**:
```json
{
  "url": "https://example.com",
  "options": {
    "format": "A4",
    "printBackground": true,
    "margin": {
      "top": "20px",
      "right": "20px",
      "bottom": "20px",
      "left": "20px"
    }
  },
  "gotoOptions": {
    "waitUntil": "networkidle2"
  }
}
```

**Parameters**:
- `url` (required): The URL to convert to PDF
- `options` (optional): Puppeteer PDF options
  - `format`: Paper format (A4, Letter, etc.)
  - `printBackground`: Include background graphics
  - `margin`: Page margins
- `gotoOptions` (optional): Page navigation options

**Response**: Binary PDF data

**Example**:
```bash
curl -X POST http://localhost:4110/chrome/pdf \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com", "options": {"format": "A4"}}' \
  -o document.pdf
```

### Web Scraping - `/chrome/scrape`

Scrapes specific elements from a webpage using CSS selectors.

**Method**: `POST`

**Request Body**:
```json
{
  "url": "https://example.com",
  "elements": [
    {
      "selector": "h1",
      "property": "innerText"
    },
    {
      "selector": "img",
      "property": "src"
    }
  ],
  "gotoOptions": {
    "waitUntil": "networkidle2"
  },
  "waitFor": 1000
}
```

**Parameters**:
- `url` (required): The URL to scrape
- `elements` (required): Array of elements to extract
  - `selector`: CSS selector
  - `property`: Property to extract (innerText, href, src, etc.)
- `gotoOptions` (optional): Page navigation options
- `waitFor` (optional): Additional wait time

**Response**: JSON array of scraped data

**Example**:
```bash
curl -X POST http://localhost:4110/chrome/scrape \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://example.com",
    "elements": [{"selector": "h1", "property": "innerText"}]
  }'
```

### Custom Function - `/chrome/function`

Executes custom JavaScript in the browser context.

**Method**: `POST`

**Request Body**:
```json
{
  "code": "async () => { const title = await page.title(); return { title }; }",
  "context": {
    "url": "https://example.com"
  }
}
```

**Parameters**:
- `code` (required): JavaScript function as string
- `context` (optional): Context object with url and other options

**Response**: Result of the JavaScript execution

**Example**:
```bash
curl -X POST http://localhost:4110/chrome/function \
  -H "Content-Type: application/json" \
  -d '{
    "code": "async () => { return await page.evaluate(() => document.title); }",
    "context": {"url": "https://example.com"}
  }'
```

## Utility Endpoints

### Health Check - `/pressure`

Returns system pressure and availability status.

**Method**: `GET`

**Response**:
```json
{
  "pressure": {
    "cpu": 15,
    "memory": 45,
    "isAvailable": true,
    "maxConcurrent": 5,
    "running": 1,
    "queued": 0
  }
}
```

### Configuration - `/config`

Returns current Browserless configuration.

**Method**: `GET`

**Response**:
```json
{
  "concurrent": 5,
  "timeout": 30000,
  "queued": 10,
  "port": 3000
}
```

## Error Handling

All endpoints return appropriate HTTP status codes:
- `200`: Success
- `400`: Bad Request (invalid parameters)
- `429`: Too Many Requests (queue full)
- `500`: Internal Server Error

Error responses include a message:
```json
{
  "error": "Error message description"
}
```

## Best Practices

1. **Timeouts**: Set appropriate timeouts for your use case
2. **Wait Strategies**: Use `waitUntil: "networkidle2"` for SPAs
3. **Error Handling**: Always handle potential timeouts and errors
4. **Resource Management**: Monitor `/pressure` endpoint for system load
5. **Concurrent Requests**: Respect the concurrent limit (default: 5)

## Common Use Cases

### Capturing Dashboard Screenshots
```bash
# Local Grafana dashboard
curl -X POST http://localhost:4110/chrome/screenshot \
  -H "Content-Type: application/json" \
  -d '{
    "url": "http://localhost:3000/dashboard/db/system-metrics",
    "options": {"fullPage": true},
    "waitFor": 2000
  }' \
  -o grafana-dashboard.png
```

### Generating Reports
```bash
# Convert internal wiki to PDF
curl -X POST http://localhost:4110/chrome/pdf \
  -H "Content-Type: application/json" \
  -d '{
    "url": "http://internal-wiki.local/quarterly-report",
    "options": {
      "format": "A4",
      "printBackground": true
    }
  }' \
  -o quarterly-report.pdf
```

### Monitoring Internal Services
```bash
# Check if service is rendering correctly
curl -X POST http://localhost:4110/chrome/scrape \
  -H "Content-Type: application/json" \
  -d '{
    "url": "http://localhost:8080/health",
    "elements": [
      {"selector": ".status", "property": "innerText"},
      {"selector": ".version", "property": "innerText"}
    ]
  }'
```

## Monitoring Endpoints

### Metrics - `/metrics`

Prometheus-compatible metrics endpoint.

**Method**: `GET`

**Response**: Prometheus text format metrics

**Example**:
```bash
curl http://localhost:4110/metrics
```

## Advanced Examples

### Authentication with Screenshots

Capture screenshots of authenticated pages using cookies:

```bash
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

### Complex Multi-Step Scraping

Execute complex scraping workflows with wait conditions:

```bash
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

Generate PDFs with custom headers and footers:

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

Capture screenshots with mobile device emulation:

```bash
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

## AI/Automation Safety

### Screenshot Validation

**Important**: The Browserless resource includes enhanced screenshot validation to prevent issues with AI tools:

- Validates HTTP response codes (only accepts 200 OK)
- Checks file size (minimum 1KB) to reject error text responses
- Verifies MIME type to ensure actual image content
- Automatically cleans up invalid files

### Safe Screenshot Function

For AI/automation use cases, use the management script's safe screenshot wrapper:

```bash
# Safe screenshot capture with validation
./manage.sh --action usage --usage-type screenshot --url https://example.com --output safe.png
```

### Why Validation Matters

When Browserless fails to capture a page (e.g., 404, 500 errors), it may return error text instead of an image. Without validation:
- A `.png` file might contain "Not Found" or HTML error text
- AI tools attempting to read such files as images can experience context corruption
- This can break automation workflows and cause unexpected behavior

The validation ensures that only genuine image files are saved, preventing these issues.

### Programmatic Usage

When integrating with Browserless programmatically:

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

---
**See also:** [Configuration](./CONFIGURATION.md) | [Usage Guide](./USAGE.md) | [Troubleshooting](./TROUBLESHOOTING.md)