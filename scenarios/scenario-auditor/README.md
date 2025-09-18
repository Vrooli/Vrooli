# Scenario Auditor

> Comprehensive standards enforcement system for Vrooli scenarios

The Scenario Auditor is Vrooli's permanent quality gatekeeper, ensuring every scenario maintains consistent, high-quality patterns that compound across the entire ecosystem. It provides unified auditing of API security, configuration compliance, UI testing practices, and development standards.

## ✨ Features

### 🛡️ Comprehensive Standards Enforcement
- **API Standards**: Go best practices, security patterns, documentation requirements
- **Configuration Standards**: service.json schema compliance, lifecycle completeness  
- **UI Standards**: Browserless testing practices, accessibility, performance
- **Testing Standards**: Phase-based structure, coverage requirements, integration patterns

### 🔧 Rule Engine
- **Modular YAML Rules**: Easy to read, write, and maintain
- **Toggleable Preferences**: Enable/disable rules with persistent storage
- **Category Organization**: Rules grouped by api, config, ui, testing
- **AI-Powered Creation**: Generate and edit rules using natural language

### 📊 Quality Dashboard
- **Real-time Monitoring**: Live standards compliance tracking
- **Violation Analysis**: Detailed breakdown by category and severity
- **Quality Scoring**: 0-100 score based on violations and importance
- **Trend Analysis**: Historical compliance data and improvements

### 🤖 AI Integration
- **Automated Fixes**: Claude Code agent spawning for violation fixes
- **Rule Generation**: Create new rules from natural language descriptions
- **Rule Enhancement**: AI-assisted modification of existing rules
- **Intelligent Analysis**: Smart categorization and prioritization

## 🚀 Quick Start

### Prerequisites
- Go 1.21+
- Node.js 18+ (for UI)
- Git
- curl, jq (for testing)

### Installation
```bash
# Full setup
make setup

# Or step by step
make install  # Install dependencies
make build    # Build binaries
make install-cli  # Install CLI globally
```

### Usage
```bash
# Start the auditor
make run

# Access the dashboard
open http://localhost:35001

# Use the CLI
scenario-auditor scan current
scenario-auditor rules --category config
scenario-auditor fix ecosystem-manager --auto
```

## 📋 CLI Commands

```bash
scenario-auditor scan [scenario]     # Scan scenario for violations
scenario-auditor rules               # List available rules
scenario-auditor fix [scenario]      # Generate fixes for violations
scenario-auditor health              # Check API health
scenario-auditor version             # Show version
scenario-auditor help                # Show help
```

### Examples
```bash
# Scan current scenario
scenario-auditor scan

# Scan specific scenario
scenario-auditor scan ecosystem-manager

# List rules by category
scenario-auditor rules --category config

# Generate automated fixes
scenario-auditor fix ecosystem-manager --auto

# Check system health
scenario-auditor health
```

## 🏗️ Architecture

### Core Components
- **Rule Engine**: Executes configurable quality rules against scenario files
- **Standards Scanner**: Validates scenarios against established best practices
- **AI Integration**: Generates new rules and fixes violations automatically
- **Preferences Manager**: Maintains user-configurable rule toggles
- **Dashboard Interface**: Provides comprehensive standards management UI

### Rule Categories
1. **API Standards** (`rules/api/`): Go best practices, security patterns
2. **Configuration Standards** (`rules/config/`): service.json compliance
3. **UI Standards** (`rules/ui/`): Browserless testing best practices
4. **Testing Standards** (`rules/testing/`): Phase-based testing structure

### Technology Stack
- **Backend**: Go with Gorilla Mux, PostgreSQL
- **Frontend**: React + TypeScript + Tailwind CSS
- **CLI**: Go with structured output
- **Rules**: YAML-based definitions
- **AI**: Claude Code agent integration

## 📊 Standards Categories

### service.json Validation
- ✅ Schema compliance with project-level schema
- ✅ Lifecycle completeness (setup, develop, test, stop)
- ✅ Binary naming conventions (`<scenario>-api`, `<scenario>`)
- ✅ Health check configuration (API + UI endpoints)
- ✅ Required step ordering and presence

### UI Testing Best Practices
- ✅ Unique element IDs for reliable testing
- ✅ data-testid attributes for interactive elements
- ✅ Semantic HTML for accessibility
- ✅ Loading state indicators
- ✅ Error handling display
- ✅ Consistent naming conventions

### Phase-Based Testing
- ✅ test/phases/ directory structure
- ✅ Unit, integration, business, dependencies tests
- ✅ Performance and structure validation
- ✅ Test artifacts organization
- ✅ Executable test scripts

### API Security & Standards
- ✅ Go module configuration
- ✅ Health check endpoints
- ✅ CORS configuration
- ✅ Error handling patterns
- ✅ Structured logging
- ✅ Graceful shutdown
- ✅ Environment variable usage

## 🧪 Testing

### Running Tests
```bash
# Full test suite
make test

# Specific test phases
make test-unit          # Unit tests
make test-integration   # Integration tests
make test-structure     # Directory structure
make test-deps          # Dependencies check
make test-business      # Business logic
make test-performance   # Performance metrics
```

### Test Structure
```
test/
├── run-tests.sh           # Main test runner
├── phases/                # Phase-based tests
│   ├── test-unit.sh       # Unit tests
│   ├── test-integration.sh # Integration tests
│   ├── test-structure.sh   # Structure validation
│   ├── test-dependencies.sh # Dependency checks
│   ├── test-business.sh    # Business logic
│   └── test-performance.sh # Performance tests
└── artifacts/             # Test outputs and logs
```

## 🎯 Development

### Project Structure
```
scenario-auditor/
├── .vrooli/
│   └── service.json       # Lifecycle configuration
├── api/                   # Go API server
│   ├── main.go           # Server entry point
│   ├── handlers_*.go     # API handlers
│   ├── rules/            # Rule engine
│   └── storage/          # Data persistence
├── cli/                  # Command-line tool
│   ├── main.go          # CLI entry point
│   └── install.sh       # Installation script
├── rules/               # Rule definitions
│   ├── api/            # API standards
│   ├── config/         # Configuration rules
│   ├── ui/             # UI standards
│   └── testing/        # Testing requirements
├── ui/                 # React dashboard
│   ├── src/pages/      # Page components
│   ├── src/components/ # Reusable components
│   └── src/services/   # API client
├── test/               # Phase-based testing
└── docs/               # Documentation
```

### Adding New Rules
1. Create YAML rule definition in appropriate category folder
2. Include name, description, condition, and fix information
3. Test rule against sample scenarios
4. Document rule purpose and usage

### Rule Definition Format
```yaml
- id: config-service-json-exists
  name: "Service JSON File Exists"
  description: "Ensures that .vrooli/service.json file exists"
  category: config
  severity: critical
  enabled: true
  condition:
    type: file_exists
    target: ".vrooli/service.json"
  message: "Missing required .vrooli/service.json file"
  fix:
    type: file_create
    action: create
    target: ".vrooli/service.json"
    template: "service-json-template"
    description: "Create service.json from template"
  tags:
    - service-json
    - configuration
    - required
```

## 🔗 Integration

### With Scenario Development
- Automatic validation during scenario creation
- Pre-commit hooks for standards checking
- CI/CD integration for quality gates
- Real-time feedback during development

### With Vrooli Ecosystem
- Cross-scenario learning and pattern recognition
- Standards evolution based on ecosystem feedback
- Integration with maintenance workflows
- Quality metrics for scenario assessment

## 📈 Quality Metrics

### Scoring System
- **100 points**: Perfect compliance
- **Critical violations**: -20 points each
- **High violations**: -10 points each
- **Medium violations**: -5 points each
- **Low violations**: -2 points each
- **Minimum score**: 0 points

### Categories Tracked
- Total violations by severity
- Auto-fixable violation count
- Files scanned and rules executed
- Compliance trends over time
- Category-specific scores

## 🤝 Contributing

### Adding Rules
1. Identify standards gap or new requirement
2. Write rule definition following existing patterns
3. Test rule against multiple scenarios
4. Submit PR with rule and documentation

### Improving Engine
1. Follow Go best practices
2. Maintain backward compatibility
3. Add comprehensive tests
4. Update documentation

## 📚 Resources

- [Product Requirements Document](PRD.md)
- [UI Testing Best Practices](../resources/browserless/docs/UI_TESTING_BEST_PRACTICES.md)
- [Phase-Based Testing Guide](test/README.md)
- [Rule Creation Documentation](docs/RULE_CREATION.md)

## 🔧 Troubleshooting

### Common Issues
- **API not starting**: Check port conflicts, run `make health`
- **Rules not loading**: Verify YAML syntax, check rule files
- **UI build failures**: Ensure Node.js deps installed
- **Test failures**: Check prerequisites, review logs in test/artifacts/

### Getting Help
- Check logs: `make logs`
- Run health check: `make health`
- View test artifacts: `ls test/artifacts/`
- CLI help: `scenario-auditor help`

---

**The Scenario Auditor becomes Vrooli's permanent quality gatekeeper - ensuring every scenario maintains the highest standards and contributes to the ecosystem's continuous improvement.**