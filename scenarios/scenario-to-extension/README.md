# Browser Extension Generator

Transform Vrooli scenarios into powerful browser extensions with intelligent templates and automated build processes.

## 🎯 Overview

The `scenario-to-extension` scenario provides a complete system for generating browser extensions from Vrooli scenarios. It bridges the gap between AI intelligence and web interaction, enabling any scenario to extend its reach into browsers with sophisticated extension capabilities.

### Core Capabilities

- **🔧 Full Extension Generation**: Complete extensions with background services, content scripts, and popup interfaces
- **📄 Specialized Templates**: Content script only, background only, or popup only variants
- **🚀 Automated Build System**: Complete development tooling with hot reload and production builds  
- **🧪 Integrated Testing**: Browserless-powered testing with screenshot validation
- **⚙️ API Integration**: Seamless connection between extensions and scenario APIs
- **🎨 Modern UI**: Web-based management interface for generation and testing

## 🚀 Quick Start

### Prerequisites

- Vrooli platform running locally
- Browserless resource available (for testing)
- Node.js 18+ (for extension builds)
- Chrome or Firefox (for testing)

### Installation

1. **Install the CLI**:
   ```bash
   cd scenarios/scenario-to-extension
   ./cli/install.sh
   ```

2. **Start the service**:
   ```bash
   vrooli scenario run scenario-to-extension
   ```

3. **Open the web UI**:
   ```
   http://localhost:3202
   ```

### Generate Your First Extension

```bash
# Generate a full extension for web-scraper scenario
scenario-to-extension generate web-scraper \
  --template full \
  --permissions storage,activeTab,scripting \
  --api-endpoint http://localhost:3000

# Build the extension  
scenario-to-extension build ./platforms/extension

# Test the extension
scenario-to-extension test ./platforms/extension --sites https://example.com
```

## 📋 Extension Templates

### Full Extension (`full`)
Complete extension with all components:
- ✅ Background service worker
- ✅ Content script injection  
- ✅ Popup interface
- ✅ Cross-component messaging
- **Best for**: Complex scenarios needing full browser integration

### Content Script Only (`content-script-only`)  
Page-focused extensions:
- ✅ DOM manipulation capabilities
- ✅ Data extraction tools
- ✅ Page interaction monitoring
- **Best for**: Web scraping, data analysis, page enhancement

### Background Only (`background-only`)
Headless automation:
- ✅ API polling and monitoring
- ✅ Cross-tab coordination
- ✅ Scheduled tasks and alarms
- **Best for**: Monitoring, automation, cross-site orchestration

### Popup Only (`popup-only`)
Simple UI tools:
- ✅ Quick access interface
- ✅ Settings and controls
- ✅ Information display
- **Best for**: Utilities, calculators, simple tools

## 🛠️ CLI Commands

### Generation
```bash
# Basic generation
scenario-to-extension generate <scenario_name>

# Advanced generation
scenario-to-extension generate my-scenario \
  --template content-script-only \
  --permissions storage,activeTab \
  --host-permissions "https://*.example.com/*" \
  --output ./extensions/my-scenario \
  --api-endpoint http://localhost:3001 \
  --debug
```

### Building
```bash
# Build extension
scenario-to-extension build ./platforms/extension

# Development mode with hot reload
scenario-to-extension build ./platforms/extension --watch

# Production build
scenario-to-extension build ./platforms/extension --minify
```

### Testing  
```bash
# Basic testing
scenario-to-extension test ./platforms/extension

# Advanced testing
scenario-to-extension test ./platforms/extension \
  --sites https://example.com,https://google.com \
  --browser chrome \
  --screenshot \
  --headless
```

### Status & Management
```bash
# Check system status
scenario-to-extension status --verbose

# List available templates
scenario-to-extension templates

# List recent builds
scenario-to-extension builds
```

## 🌐 Web Interface

Access the web UI at `http://localhost:3202` for:

### 🎛️ Extension Generator
- Visual template selection
- Form-based configuration
- Real-time validation
- Progress monitoring

### 📦 Build Management  
- Build status tracking
- Error log viewing
- Download and testing
- Build history

### 🧪 Testing Suite
- Multi-site testing
- Screenshot capture
- Error detection
- Performance metrics

### 📚 Template Library
- Template documentation
- File structure preview
- Usage examples

## 🔌 API Integration

Extensions communicate with scenarios via REST APIs:

```javascript
// In your extension's background script
const response = await ScenarioAPI.executeAction('analyze-page', {
  url: tab.url,
  content: pageData
});

// In your scenario's API handler
app.post('/api/v1/extension/action/analyze-page', (req, res) => {
  const { url, content } = req.body;
  // Process page data with AI
  res.json({ success: true, analysis: result });
});
```

### Authentication
Extensions authenticate using bearer tokens:
```javascript
// Extension requests token
const auth = await ScenarioAPI.authenticate({
  extensionId: chrome.runtime.id,
  scenario: 'my-scenario'
});

// All subsequent requests include token
headers: { 'Authorization': `Bearer ${auth.token}` }
```

## 🔒 Security Best Practices

### Permission Minimization
- Request only necessary permissions
- Use `activeTab` instead of `tabs` when possible  
- Prefer specific host patterns over `<all_urls>`

### Content Security Policy
Generated extensions include strict CSP:
```json
{
  "content_security_policy": {
    "extension_pages": "script-src 'self'; object-src 'self'; connect-src 'self' https://your-api.com"
  }
}
```

### Data Protection
- API tokens stored in encrypted Chrome storage
- No sensitive data logged in production
- All API communication over HTTPS

## 🧪 Testing & Validation

### Automated Testing
The scenario includes comprehensive testing:
- ✅ Manifest validation
- ✅ Extension loading verification
- ✅ API integration testing  
- ✅ Cross-browser compatibility
- ✅ Performance benchmarking
- ✅ Security auditing

### Manual Testing Workflow
1. Generate extension
2. Load in Chrome developer mode  
3. Test on target websites
4. Verify API communication
5. Check browser console for errors
6. Validate user experience

## 📈 Performance Optimization

### Generated Extensions
- Minimal permission requests
- Efficient content script injection
- Debounced API calls
- Memory leak prevention
- Background script optimization

### Build System
- Template variable substitution
- Code minification (production)
- Asset optimization
- Hot reload (development)
- Incremental builds

## 🔄 Integration with Other Scenarios

This scenario enhances multiple other scenarios:

### web-scraper-manager
```bash
scenario-to-extension generate web-scraper-manager \
  --template content-script-only \
  --permissions storage,activeTab,scripting
```

### productivity-enhancer  
```bash
scenario-to-extension generate productivity-enhancer \
  --template full \
  --permissions storage,activeTab,contextMenus
```

### social-media-monitor
```bash
scenario-to-extension generate social-media-monitor \
  --template background-only \
  --permissions storage,alarms,notifications
```

## 🛠️ Development

### Project Structure
```
scenario-to-extension/
├── api/                    # Go API server
├── cli/                    # CLI implementation  
├── ui/                     # Web interface
├── templates/              # Extension templates
│   ├── vanilla/           # Base templates
│   └── advanced/          # Specialized variants
├── prompts/               # AI generation prompts
├── initialization/        # N8n workflows
└── test/                  # Test suites
```

### Contributing

1. **Template Development**: Add new templates in `templates/advanced/`
2. **Feature Enhancement**: Extend API endpoints and CLI commands
3. **Testing**: Add test cases in `test/` directory
4. **Documentation**: Update README and inline docs

### Custom Templates

Create specialized templates for specific use cases:

```json
{
  "name": "my-custom-template",
  "description": "Custom template for specific scenario type",
  "manifest_template": { /* custom manifest */ },
  "files": ["manifest.json", "custom.js"],
  "template_variables": { /* custom variables */ }
}
```

## 🎯 Roadmap

### v1.1 - Enhanced Templates
- Firefox WebExtensions support
- Safari extension templates  
- Advanced content script patterns

### v1.2 - Store Integration
- Chrome Web Store publishing
- Firefox Add-ons submission
- Automated store deployment

### v1.3 - Enterprise Features  
- Extension management dashboard
- Enterprise distribution
- Analytics and monitoring

## 📚 Resources

- **Extension Documentation**: Generated extensions include complete README files
- **Chrome Extension APIs**: https://developer.chrome.com/extensions
- **Firefox WebExtensions**: https://developer.mozilla.org/en-US/docs/Mozilla/Add-ons/WebExtensions
- **Browserless Testing**: https://browserless.io/docs
- **Vrooli Platform**: https://github.com/vrooli/vrooli

## 🐛 Troubleshooting

### Common Issues

**Extension won't load:**
- Check manifest.json syntax
- Verify all referenced files exist
- Look for JavaScript errors in console

**API calls failing:**  
- Verify scenario is running
- Check CORS configuration
- Validate authentication tokens

**Content script not working:**
- Verify host permissions match website
- Check content script injection timing
- Look for CSP violations

### Debug Mode
Enable debug mode for verbose logging:
```bash
scenario-to-extension generate my-scenario --debug
```

### Getting Help
- Check the web UI status page
- Review CLI help: `scenario-to-extension help`
- Examine API health: `curl http://localhost:3201/api/v1/health`
- Report issues: GitHub repository issues page

---

**Generated by scenario-to-extension v1.0.0**  
**Part of the [Vrooli](https://vrooli.com) AI intelligence platform**