# Scenario Documentation

## 📖 Core Guides

### Fundamentals
- **[CONCEPTS.md](CONCEPTS.md)** - Understanding dual-purpose scenarios and resource orchestration
- **[getting-started.md](getting-started.md)** - Step-by-step scenario creation tutorial  

### Development & Testing
- **[VALIDATION.md](VALIDATION.md)** - Testing and validation framework
- **[TESTING_GUIDE.md](TESTING_GUIDE.md)** - Comprehensive testing strategies

### Deployment & Operations  
- **[DEPLOYMENT.md](DEPLOYMENT.md)** - Running scenarios directly in production

### Advanced Topics
- **[ai-generation-guide.md](ai-generation-guide.md)** - AI-powered scenario generation patterns

### Templates
- **[templates/](templates/)** - Enterprise-ready scenario templates

## Quick Links

### Common Commands
```bash
# List available scenarios
vrooli scenario list

# Run a scenario directly
vrooli scenario run <name>

# Test scenario integration
vrooli scenario test <name>

# Direct execution from folder
cd scenarios/<name>
vrooli scenario run <scenario-name>
```

### Key Concepts

**Scenarios** are complete business applications that:
- 🎯 Orchestrate multiple resources to create business value
- 💰 Generate $10K-50K revenue per deployment
- 🚀 Run directly from source without build steps
- ✅ Serve as both integration tests AND production apps

**Direct Execution** means:
- No conversion to standalone apps
- No build artifacts or compilation
- Edit and run immediately
- Single source of truth in scenarios/ folder

## Directory Structure

```
scenarios/
├── README.md               # This file - navigation hub
├── CONCEPTS.md            # Core dual-purpose concepts
├── getting-started.md     # Tutorial for new users
├── VALIDATION.md          # Testing framework
├── TESTING_GUIDE.md       # Testing strategies
├── DEPLOYMENT.md          # Production deployment
├── ai-generation-guide.md # AI generation patterns
├── templates/             # Reusable templates
└── [scenario-folders]/    # Individual scenarios
    ├── .vrooli/service.json
    ├── test.sh
    ├── initialization/
    └── deployment/
```

## Getting Help

- **Issues**: Report at [GitHub Issues](https://github.com/Vrooli/Vrooli/issues)
- **Main Docs**: Return to [Main Documentation](../README.md)
- **Resources**: See [Resource Documentation](../resources/README.md)