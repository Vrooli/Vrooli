# React Component Library

**AI-powered React component showcase with accessibility testing, performance benchmarking, and live component playground**

The React Component Library scenario provides a comprehensive platform for building, testing, showcasing, and sharing reusable React components across all Vrooli scenarios. It combines a live component playground with AI-powered generation, automated accessibility testing, performance analysis, and export capabilities.

## 🎯 Purpose & Value

### Core Capability
Every React component built becomes a **permanent asset** that accelerates all future UI development across the Vrooli ecosystem. This creates a recursive improvement cycle where each component makes all future scenarios more capable.

### Key Features
- **🎨 Interactive Component Showcase** - Storybook-like interface with live code playground
- **🤖 AI-Powered Generation** - Create components using natural language descriptions
- **♿ Accessibility Testing** - Automated WCAG 2.1 compliance checking
- **⚡ Performance Benchmarking** - Real-time render time, bundle size, and memory analysis  
- **🔍 Semantic Search** - Find components using natural language queries
- **📦 Multi-Format Export** - Export as npm packages, CDN links, or raw code
- **🔄 Version Management** - Track component evolution and breaking changes
- **📊 Usage Analytics** - Monitor component adoption across scenarios

## 🏗️ Architecture

### Technology Stack
- **Backend API**: Go with Gin framework
- **Frontend UI**: React with TypeScript
- **Database**: PostgreSQL for component metadata and analytics
- **Vector Search**: Qdrant for semantic component discovery
- **File Storage**: MinIO for screenshots and exported packages
- **AI Integration**: Claude-code for component generation and improvement

### Resource Dependencies
```yaml
Required:
  - postgres: Component metadata, versions, test results
  - qdrant: Semantic search for component patterns
  - claude-code: AI-powered component generation
  - minio: Component screenshots and export storage

Optional:
  - redis: Performance caching (improves search speed)
  - browserless: Automated screenshot generation
```

## 🚀 Getting Started

### Prerequisites
- Vrooli platform with required resources enabled
- Node.js 16+ and Go 1.21+ (for development)

### Quick Setup
```bash
# Setup the scenario (includes all dependencies)
vrooli scenario setup react-component-library

# Start the component library
vrooli scenario run react-component-library

# Install CLI globally
cd cli && ./install.sh
```

### Access Points
- **Web Interface**: http://localhost:31012
- **API Documentation**: http://localhost:8090/api/docs
- **CLI Help**: `react-component-library --help`

## 💻 Usage

### Web Interface
The primary interface provides:
- **Dashboard**: Overview of components, recent activity, and quick actions
- **Component Library**: Browse and search existing components
- **Live Playground**: Interactive component testing with real-time preview
- **AI Generator**: Natural language component generation
- **Testing Suite**: Accessibility and performance analysis
- **Analytics**: Usage patterns and component popularity

### CLI Commands
```bash
# Create a new component
react-component-library create Button form --template button

# Search for components
react-component-library search "modal dialog component"

# Generate component with AI
react-component-library generate "responsive data table with sorting"

# Run accessibility tests
react-component-library test MyComponent --accessibility

# Export component
react-component-library export Button --format npm-package

# Check system status
react-component-library status --verbose
```

### API Integration
Other scenarios can programmatically access components:
```bash
# Search for components
curl "http://localhost:8090/api/v1/components/search?query=button"

# Generate new component
curl -X POST http://localhost:8090/api/v1/components/generate \
  -H "Content-Type: application/json" \
  -d '{"description": "modal dialog with backdrop", "accessibility_level": "AA"}'

# Run tests on component  
curl -X POST http://localhost:8090/api/v1/components/{id}/test \
  -H "Content-Type: application/json" \
  -d '{"test_types": ["accessibility", "performance"]}'
```

## 🧪 Component Testing

### Automated Test Types
- **Accessibility**: WCAG 2.1 compliance using axe-core
- **Performance**: Render time, bundle size, memory usage
- **Visual Regression**: Screenshot comparison for UI consistency
- **Unit Tests**: Jest/React Testing Library integration
- **Code Quality**: ESLint analysis with best practices

### Test Configuration
```javascript
// Example test request
{
  "test_types": ["accessibility", "performance"],
  "test_config": {
    "accessibility_level": "AA",
    "performance_budget": {
      "render_time_ms": 16,
      "bundle_size_kb": 10
    }
  }
}
```

## 🤖 AI-Powered Features

### Component Generation
Describe components in natural language:
```bash
react-component-library generate "responsive navigation menu with dropdown, mobile hamburger, and accessibility support"
```

### Intelligent Improvement
Get AI suggestions for existing components:
```bash
react-component-library improve MyComponent --focus accessibility,performance
```

### Semantic Search
Find components using natural language:
```bash
react-component-library search "data visualization with charts and filtering"
```

## 📦 Export & Sharing

### Export Formats
- **npm Package**: Ready-to-install package with dependencies
- **CDN**: Hosted JavaScript for script tag inclusion
- **Raw Code**: Copy-paste TypeScript/JSX code
- **ZIP Archive**: Complete component with dependencies and examples

### Cross-Scenario Integration
```typescript
// Import component in other scenarios
import { Button, Modal, DataTable } from '@vrooli/react-components';

// Use in your React applications
function MyApp() {
  return (
    <div>
      <Button variant="primary">Click me</Button>
      <Modal isOpen={true}>Content</Modal>
    </div>
  );
}
```

## 📊 Analytics & Insights

### Usage Metrics
- Component popularity and adoption rates
- Performance trends over time
- Accessibility compliance scores
- Cross-scenario usage patterns

### Quality Dashboard
- Test pass/fail rates
- Performance benchmarks
- Code quality scores
- AI generation success rates

## 🔧 Development

### Local Development
```bash
# Install dependencies
cd api && go mod download
cd ui && npm install

# Start development servers
vrooli scenario develop react-component-library
```

### Testing
```bash
# Run all tests
vrooli scenario test react-component-library

# Individual test suites
cd api && go test ./...
cd ui && npm test
react-component-library test --all
```

### API Endpoints
Full API documentation available at `/api/docs` when running.

Key endpoints:
- `GET /api/v1/components` - List components
- `POST /api/v1/components` - Create component
- `GET /api/v1/components/search` - Search components
- `POST /api/v1/components/generate` - AI generate component
- `POST /api/v1/components/{id}/test` - Run tests

## 🌟 Best Practices

### Component Development
- Use TypeScript for better type safety
- Include comprehensive prop interfaces
- Add accessibility attributes by default
- Optimize for performance and bundle size
- Include usage examples and documentation

### Testing Strategy
- Test accessibility from the start
- Monitor performance budgets
- Use visual regression testing for UI changes
- Maintain high test coverage

### AI Generation Tips
- Provide specific, detailed descriptions
- Include accessibility requirements
- Specify style preferences
- Mention any dependencies or constraints

## 🔮 Future Roadmap

### Upcoming Features
- **Visual Component Composer**: Drag-and-drop component builder
- **Design System Generator**: Automated branded component libraries  
- **Component Marketplace**: Rating and review system
- **Advanced Analytics**: Usage prediction and optimization
- **Multi-framework Support**: Vue and Svelte component generation

### Integration Plans
- **Figma Integration**: Import designs as components
- **Storybook Export**: Generate Storybook configurations
- **Testing Automation**: Continuous component quality monitoring
- **Performance Monitoring**: Real-time component performance tracking

## 🤝 Contributing

### Adding New Component Templates
1. Create template in `templates/` directory
2. Add template metadata and preview
3. Update template registry
4. Test generation and export

### Extending AI Capabilities
1. Enhance generation prompts in `api/services/ai.go`
2. Add new improvement focus areas
3. Implement custom model integrations
4. Update CLI commands for new features

## 📚 Resources

- **PRD**: Complete product requirements in `PRD.md`
- **API Docs**: Live documentation at `/api/docs`
- **CLI Reference**: `react-component-library help`
- **Architecture**: Technical details in `docs/architecture.md`

## 🆘 Troubleshooting

### Common Issues
- **API Connection Failed**: Ensure all required resources are running
- **Component Not Found**: Check component ID and search index
- **Test Failures**: Verify component code syntax and dependencies
- **Export Errors**: Check MinIO storage availability

### Getting Help
- Check component library status: `react-component-library status`
- View logs: `vrooli scenario logs react-component-library`
- API health: `curl http://localhost:8090/health`
- Web interface: http://localhost:31012

---

**Built with ❤️ for the Vrooli ecosystem**

*Every component you create becomes a permanent capability that makes all future development faster and more consistent across the platform.*