# Browserless API Reference

## Overview

Browserless provides a REST API for headless Chrome automation. All browser-related endpoints use the `/chrome/` prefix.

**Base URL**: `http://localhost:4110`

## Authentication

The default Browserless setup has no authentication. For production use, set a token in the configuration.

## Endpoints

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