# Browserless Integration Examples

This directory contains practical examples demonstrating how to integrate Browserless into your applications and workflows.

## Directory Structure

```
examples/
├── typescript/           # TypeScript/Node.js examples
│   ├── dashboard-capture.ts    # Capture screenshots of internal dashboards
│   └── service-monitoring.ts   # Monitor internal service health
├── n8n-workflows/       # n8n workflow templates
│   ├── dashboard-monitoring.json    # Automated dashboard monitoring
│   ├── pdf-report-generation.json   # PDF report generation
│   └── content-scraping-pipeline.json # Web scraping with Ollama analysis
└── README.md           # This file
```

## TypeScript Examples

### Dashboard Capture

Captures screenshots of internal dashboards on a schedule or on-demand.

**Features:**
- Captures multiple dashboards in one run
- Configurable viewport sizes
- Health checks before capture
- Saves screenshots with timestamps
- Generates manifest file

**Usage:**
```bash
cd examples/typescript
npm install
npx ts-node dashboard-capture.ts
```

**Environment Variables:**
- `BROWSERLESS_URL`: Browserless endpoint (default: http://localhost:4110)
- `BROWSERLESS_TOKEN`: Optional authentication token

### Service Monitoring

Monitors internal services by checking health endpoints and scraping status information.

**Features:**
- Health endpoint checking
- Status scraping from UI
- Screenshot capture on errors
- Response time measurement
- JSON report generation

**Usage:**
```bash
cd examples/typescript
npm install
npx ts-node service-monitoring.ts
```

## n8n Workflow Examples

### Dashboard Monitoring Workflow

Automated monitoring that captures dashboards hourly and alerts on issues.

**Features:**
- Scheduled execution (hourly)
- Screenshot capture
- Metric scraping
- Conditional alerts
- File storage

**Import Instructions:**
1. Open n8n (http://localhost:5678)
2. Go to Workflows → Import
3. Select `dashboard-monitoring.json`
4. Configure Slack/email credentials for alerts

### PDF Report Generation Workflow

Generates PDF reports from web pages on demand via webhook.

**Features:**
- Webhook triggered
- PDF generation with options
- Google Drive upload
- Email delivery
- Response handling

**Usage:**
```bash
# Trigger via webhook
curl -X POST http://localhost:5678/webhook/pdf-report-trigger \
  -H "Content-Type: application/json" \
  -d '{
    "reportUrl": "http://localhost:3000/reports/quarterly",
    "reportTitle": "Q4 2024 Report",
    "reportName": "quarterly-report",
    "recipientEmail": "manager@example.com",
    "driveFolder": "Reports/2024"
  }'
```

### Content Scraping Pipeline

Daily content scraping with AI analysis using Ollama.

**Features:**
- Daily scheduled execution
- Batch URL processing
- Content extraction
- Element scraping
- Ollama AI analysis
- MongoDB storage
- File backup

**Setup:**
1. Create `/data/urls-to-scrape.json`:
```json
{
  "urls": [
    {
      "url": "http://internal-wiki/latest-updates",
      "title": "Wiki Updates",
      "selectors": [
        {"selector": "h2", "property": "innerText"},
        {"selector": ".update-content", "property": "innerText"}
      ]
    }
  ]
}
```
2. Configure MongoDB connection in n8n
3. Import and activate workflow

## Integration Patterns

### 1. Deciding Between Browserless and Agent-S2

```typescript
function chooseAutomationTool(url: string, requirements: {
  needsAuth?: boolean;
  hasAntiBot?: boolean;
  isInternal?: boolean;
  needsVisualReasoning?: boolean;
}) {
  // Internal services → Browserless
  if (requirements.isInternal || url.includes('localhost')) {
    return 'browserless';
  }
  
  // Complex public sites → Agent-S2
  if (requirements.hasAntiBot || requirements.needsVisualReasoning) {
    return 'agent-s2';
  }
  
  // Default for simple public sites
  return 'browserless';
}
```

### 2. Error Handling Pattern

```typescript
async function robustScreenshot(url: string, retries = 3) {
  for (let i = 0; i < retries; i++) {
    try {
      return await captureScreenshot(url);
    } catch (error) {
      if (i === retries - 1) throw error;
      await new Promise(resolve => setTimeout(resolve, 1000 * (i + 1)));
    }
  }
}
```

### 3. Batch Processing Pattern

```typescript
async function batchCapture(urls: string[], concurrency = 3) {
  const results = [];
  
  for (let i = 0; i < urls.length; i += concurrency) {
    const batch = urls.slice(i, i + concurrency);
    const batchResults = await Promise.all(
      batch.map(url => captureScreenshot(url).catch(e => ({ error: e })))
    );
    results.push(...batchResults);
  }
  
  return results;
}
```

## Best Practices

1. **Timeouts**: Always set appropriate timeouts for your use case
2. **Wait Strategies**: Use `waitUntil: "networkidle2"` for dynamic content
3. **Error Handling**: Implement retry logic for transient failures
4. **Resource Management**: Monitor concurrent browser instances
5. **Authentication**: Use bearer tokens in production environments

## Common Use Cases

### Internal Dashboard Monitoring
- Grafana metrics dashboards
- Kibana log analytics
- Custom admin panels
- Build status pages

### Development Tool Integration
- ComfyUI workflow screenshots
- Jupyter notebook exports
- Node-RED flow documentation
- Code coverage reports

### Report Generation
- Wiki page PDFs
- Analytics reports
- Status page archives
- Documentation exports

## Troubleshooting

### "Request has timed out"
- Increase timeout in request options
- Check if page requires authentication
- Verify the URL is accessible from Browserless container

### Empty Screenshots
- Add `waitFor` parameter to allow content to load
- Check browser console for JavaScript errors
- Verify selectors are correct

### Connection Refused
- Ensure Browserless is running: `docker ps | grep browserless`
- Check port mapping: should be 4110
- Verify no firewall blocking

## Additional Resources

- [Browserless API Reference](../docs/API_REFERENCE.md)
- [Main Browserless README](../README.md)
- [Puppeteer Documentation](https://pptr.dev/) (for advanced options)