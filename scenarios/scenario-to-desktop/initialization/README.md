# Scenario-to-Desktop Initialization

This directory contains initialization files and workflows for the scenario-to-desktop system. These files are automatically deployed when the scenario starts up and provide automation and integration capabilities.

## 📁 Directory Structure

```
initialization/
├── README.md                           # This file
└── n8n/                               # N8n workflow definitions
    └── desktop-build-automation.json  # Automated desktop app build pipeline
```

## 🔄 N8n Workflows

### desktop-build-automation.json

**Purpose**: Provides a complete automated pipeline for building desktop applications from scenario configurations.

**Capabilities**:
- Validates desktop build requests
- Generates desktop applications using templates
- Builds TypeScript and installs dependencies
- Packages applications for distribution
- Performs screenshot testing with Browserless
- Sends build completion notifications
- Handles error cases gracefully

**Triggers**:
- Webhook: `POST /webhook/desktop-build`
- Manual trigger for testing

**Usage**:
```bash
# Via webhook
curl -X POST http://localhost:5678/webhook/desktop-build \
  -H "Content-Type: application/json" \
  -d '{
    "scenario_name": "picker-wheel",
    "framework": "electron",
    "template_type": "basic",
    "output_path": "/tmp/picker-wheel-desktop",
    "platforms": ["win", "mac", "linux"],
    "callback_url": "http://localhost:3202/api/v1/desktop/webhook/build-complete"
  }'

# Via scenario-to-desktop CLI (uses this workflow internally)
scenario-to-desktop generate picker-wheel --framework electron --template basic
```

**Workflow Steps**:
1. **Validation**: Validates build request parameters
2. **Generation**: Creates desktop app from templates
3. **Build**: Compiles TypeScript and installs dependencies
4. **Packaging**: Creates distribution packages for target platforms
5. **Testing**: Takes screenshots for UI validation (if Browserless available)
6. **Notification**: Sends completion webhook with results
7. **Response**: Returns build results to caller

**Error Handling**:
- Invalid parameters return immediate error response
- Build failures are captured with full error logs
- Failed builds trigger cleanup procedures
- All errors are reported via webhook callback

**Integration Points**:
- **scenario-to-desktop CLI**: Primary integration point
- **scenario-to-desktop API**: Uses this workflow for async builds
- **Browserless**: Optional screenshot testing
- **Notification services**: Build completion alerts

## 🚀 Deployment

These workflows are automatically deployed when scenario-to-desktop starts:

1. **Automatic Setup**: Service startup reads initialization files
2. **N8n Deployment**: Workflows are imported into N8n instance
3. **Activation**: Workflows are activated and ready for use
4. **Health Check**: System validates workflow deployment

## 🔧 Configuration

### Environment Variables

The workflows use these environment variables:

- `N8N_URL`: N8n instance URL (default: http://localhost:5678)
- `DESKTOP_BUILD_TIMEOUT`: Build timeout in milliseconds (default: 600000)
- `BROWSERLESS_URL`: Browserless instance URL for testing
- `CALLBACK_URL`: Default callback URL for build notifications

### Workflow Settings

Workflows can be customized by modifying the JSON files:

```json
{
  "settings": {
    "executionOrder": "v1",
    "saveManualExecutions": true,
    "callerPolicy": "workflowsFromSameOwner"
  }
}
```

## 🔍 Monitoring

### Workflow Execution

Monitor workflow execution through:
- N8n web interface: `http://localhost:5678`
- Workflow logs via N8n API
- Build notifications via webhooks

### Metrics

Track these key metrics:
- Build success rate
- Average build time
- Template usage patterns
- Platform distribution
- Error frequency

### Health Checks

Workflows include health monitoring:
```bash
# Check workflow status
curl http://localhost:5678/api/v1/workflows/desktop-build-automation

# Test workflow execution
curl -X POST http://localhost:5678/webhook/desktop-build \
  -H "Content-Type: application/json" \
  -d '{"test": true}'
```

## 🐛 Troubleshooting

### Common Issues

**Workflow Not Activated**:
```bash
# Check workflow status
curl http://localhost:5678/api/v1/workflows

# Manually activate
curl -X POST http://localhost:5678/api/v1/workflows/desktop-build-automation/activate
```

**Build Failures**:
- Check Node.js version compatibility
- Verify template file availability
- Ensure output directory permissions
- Review build logs in workflow execution

**Missing Dependencies**:
```bash
# Install build tools
npm install -g electron-builder

# Check system requirements
scenario-to-desktop status --verbose
```

### Debug Mode

Enable debug logging in workflows:
```json
{
  "parameters": {
    "jsCode": "console.log('[DEBUG] Current step:', $input.all());"
  }
}
```

## 🔄 Updates

### Workflow Updates

To update workflows:
1. Modify JSON files in this directory
2. Restart scenario-to-desktop service
3. Workflows are automatically redeployed

### Version Management

Workflows include version tracking:
```json
{
  "versionId": "1.0.0",
  "meta": {
    "templateCredsSetupCompleted": true
  }
}
```

## 📚 Related Documentation

- [Desktop Templates](../templates/README.md) - Template system documentation
- [Build Tools](../templates/build-tools/README.md) - Build system details
- [CLI Commands](../cli/README.md) - Command-line interface
- [API Reference](../api/README.md) - REST API documentation

## 🤝 Contributing

When adding new workflows:
1. Follow existing naming conventions
2. Include comprehensive error handling
3. Add monitoring and logging
4. Test with manual trigger first
5. Document integration points
6. Update this README

### Workflow Template

Use this template for new workflows:
```json
{
  "name": "New Desktop Workflow",
  "nodes": [
    {
      "parameters": {},
      "id": "webhook-trigger",
      "name": "Webhook Trigger",
      "type": "n8n-nodes-base.webhook",
      "typeVersion": 1,
      "position": [240, 300]
    }
  ],
  "connections": {},
  "active": true,
  "settings": {
    "executionOrder": "v1"
  },
  "versionId": "1.0.0"
}
```

---

**Generated by scenario-to-desktop v1.0.0**  
**Part of the [Vrooli](https://vrooli.com) AI intelligence platform**