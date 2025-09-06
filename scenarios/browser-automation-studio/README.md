# Browser Automation Studio

Visual browser automation workflow builder with AI-powered generation and debugging.

## 🎯 Overview

Browser Automation Studio transforms browser automation from code-based scripts to visual, self-healing workflows. It provides a drag-and-drop interface for creating browser automation workflows, real-time execution monitoring with screenshots, and AI assistance for both generation and debugging.

## ✨ Features

- **Visual Workflow Builder**: Drag-and-drop nodes to create automation flows using React Flow
- **Real-time Execution**: Watch screenshots and logs stream as workflows execute
- **AI Generation**: Create workflows from natural language descriptions
- **AI Debugging**: Automatic debugging and fixing with Claude Code integration
- **Folder Organization**: Organize workflows in a tree structure
- **Scheduling**: Calendar integration for recurring automations
- **Full API/CLI**: Complete programmatic access to all features

## 🚀 Quick Start

### Prerequisites

```bash
# Ensure required resources are running
vrooli resource browserless start
vrooli resource postgres start
vrooli resource minio start
```

### Installation

```bash
# Install CLI
cd cli
./install.sh
source ~/.bashrc

# Setup database
cd ../api
go run cmd/migrate/main.go up

# Install UI dependencies
cd ../ui
npm install
```

### Running the Scenario

```bash
# Start the scenario
vrooli scenario run browser-automation-studio

# Or start components individually:
# Start API
cd api && go run main.go

# Start UI
cd ui && npm run dev
```

### Using the CLI

```bash
# Check status
browser-automation-studio status

# Create workflow from AI prompt
browser-automation-studio workflow create "test-flow" \
  --ai-prompt "Navigate to google.com and search for automation"

# Execute workflow
browser-automation-studio workflow execute "test-flow" --wait

# List workflows
browser-automation-studio workflow list
```

## 🏗️ Architecture

### Components

1. **UI (React + Vite + TypeScript)**
   - React Flow for visual workflow building
   - WebSocket for real-time updates
   - Zustand for state management
   - Tailwind CSS for styling

2. **API (Go + Chi)**
   - RESTful API for workflow management
   - WebSocket server for live streaming
   - PostgreSQL for persistence
   - MinIO for screenshot storage

3. **CLI (Bash)**
   - Thin wrapper around API
   - Full feature parity with UI
   - JSON and human-readable output

### Workflow Node Types

- **Navigate**: Go to URL
- **Click**: Click elements by selector
- **Type**: Enter text in inputs
- **Screenshot**: Capture page screenshots
- **Wait**: Wait for time/elements/navigation
- **Extract**: Extract data from pages

## 🤖 AI Integration

### Workflow Generation
The AI can generate complete workflows from natural language:
```bash
"Navigate to amazon.com, search for laptops, 
click the first result, extract the price"
```

### Debugging Assistant
Failed workflows can be debugged with Claude Code:
- Analyzes error logs and screenshots
- Suggests fixes for selectors
- Handles website changes automatically

## 📁 Workflow Organization

Workflows are organized in a folder structure:
```
/ui-validation
  /checkout
    - checkout-test
    - payment-validation
  /login
    - login-flow
/data-collection
  - competitor-pricing
  - news-scraper
/automation
  - invoice-download
```

## 🔌 Integration with Other Scenarios

Browser Automation Studio provides automation capabilities to all Vrooli scenarios:

```javascript
// From any scenario
const result = await fetch('http://localhost:8090/api/v1/workflows/execute', {
  method: 'POST',
  body: JSON.stringify({
    workflow_id: 'ui-validation-checkout',
    parameters: { productId: '123' }
  })
});
```

## 🎨 UI Style

The UI follows a technical, developer-focused aesthetic:
- Dark theme with syntax highlighting
- Split-pane layout for workflow building and execution viewing
- Console-style log output
- Filmstrip screenshot timeline

## 🧪 Testing

```bash
# Run scenario tests
vrooli scenario test browser-automation-studio

# Run unit tests
cd api && go test ./...
cd ../ui && npm test
```

## 📊 Metrics

The scenario tracks:
- Workflows created
- Workflows executed
- Success rate
- Average execution time
- Screenshots captured

## 🔒 Security

- Screenshots encrypted at rest in MinIO
- Role-based access to workflows
- Audit trail for all executions
- No secrets in workflow definitions

## 🚧 Known Limitations

- Browserless supports ~10 concurrent sessions
- Screenshots can consume significant storage
- WebSocket connections required for real-time features

## 🔮 Future Enhancements

- [ ] Visual regression testing with diff highlights
- [ ] Workflow marketplace for sharing
- [ ] Parallel execution across browsers
- [ ] Cloud execution options
- [ ] Advanced AI with auto-fix capabilities

## 📚 Documentation

- [API Documentation](docs/api.md)
- [CLI Reference](docs/cli.md)
- [Workflow Schema](docs/workflow-schema.md)
- [Integration Guide](docs/integration.md)

## 🤝 Contributing

This scenario is a core Vrooli capability. Improvements here benefit all UI-based scenarios.

---

**Part of the Vrooli Ecosystem** - Every workflow created becomes permanent intelligence the system uses forever.