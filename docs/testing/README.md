# Vrooli Testing Documentation

Welcome to Vrooli's comprehensive testing documentation. This directory contains all the documentation you need to understand, implement, and maintain testing across the Vrooli ecosystem.

## 🧭 Quick Navigation

**New to testing in Vrooli?** → Start with [Quick Start Guide](guides/quick-start.md)  
**Need to test safely?** → Read [Safety Guidelines](safety/GUIDELINES.md) first  
**Looking for specific info?** → Use the [Testing Hub](../TESTING.md) decision tree

## 📁 Documentation Structure

### [🛡️ Safety](safety/)
**CRITICAL - Read before writing any tests**

Essential safety information to prevent accidental data loss in test scripts.

- **[Guidelines](safety/GUIDELINES.md)** - Complete safety rules and patterns
- **[README](safety/README.md)** - Quick safety checklist and critical warnings
- **[BATS Teardown Bug](safety/BATS_TEARDOWN_BUG.md)** - Real incident case study

### [📚 Guides](guides/)
**Step-by-step instructions for common testing tasks**

Practical guides for getting things done quickly and correctly.

- **[Quick Start](guides/quick-start.md)** - Get testing in 5 minutes
- **[Scenario Testing](guides/scenario-testing.md)** - Test complete applications
- **[Scenario Unit Testing](guides/scenario-unit-testing.md)** - Go, Node.js, Python unit tests
- **[Resource Unit Testing](guides/resource-unit-testing.md)** - Test Vrooli resources
- **[CLI Testing](guides/cli-testing.md)** - BATS framework for command-line tools
- **[Troubleshooting](guides/troubleshooting.md)** - Debug and fix common issues

### [📐 Architecture](architecture/)
**Core concepts and design principles**

Deep dives into how Vrooli's testing system works and why.

- **[Phased Testing](architecture/PHASED_TESTING.md)** - 6-phase progressive validation system
- **[Strategy](architecture/STRATEGY.md)** - Three-layer validation for resources
- **[Infrastructure](architecture/infrastructure.md)** - BATS, stubbing, and testing tools

### [📖 Reference](reference/)
**API documentation and comprehensive examples**

Detailed reference material for testing libraries and implementations.

- **[Shell Libraries](reference/shell-libraries.md)** - Testing library function reference
- **[Test Runners](reference/test-runners.md)** - Available test execution methods
- **[Examples](reference/examples.md)** - Gold standard implementations to study

## 🎯 Common Testing Scenarios

### I need to...

| Task | Start Here | Also Read |
|------|------------|-----------|
| **Write my first test** | [Quick Start](guides/quick-start.md) | [Safety Guidelines](safety/GUIDELINES.md) |
| **Test a complete scenario** | [Scenario Testing](guides/scenario-testing.md) | [Phased Testing](architecture/PHASED_TESTING.md) |
| **Test Go/Node/Python code** | [Scenario Unit Testing](guides/scenario-unit-testing.md) | [Examples](reference/examples.md) |
| **Test a Vrooli resource** | [Resource Unit Testing](guides/resource-unit-testing.md) | [Strategy](architecture/STRATEGY.md) |
| **Test CLI commands safely** | [CLI Testing](guides/cli-testing.md) | [Safety Guidelines](safety/GUIDELINES.md) |
| **Debug failing tests** | [Troubleshooting](guides/troubleshooting.md) | [Examples](reference/examples.md) |
| **Understand the architecture** | [Phased Testing](architecture/PHASED_TESTING.md) | [Infrastructure](architecture/infrastructure.md) |
| **Use testing libraries** | [Shell Libraries](reference/shell-libraries.md) | [Test Runners](reference/test-runners.md) |

## 🚨 Critical Safety Notice

**BEFORE writing any test scripts, read the [Safety Guidelines](safety/GUIDELINES.md).** 

Test scripts can accidentally delete important files if not written properly. We've had real incidents where tests deleted Makefiles, READMEs, and other critical files. The safety documentation exists because these problems have happened in production.

## 📊 Testing Standards

### Coverage Requirements
- **Unit Tests**: 80% warning threshold, 70% minimum
- **Integration Tests**: All endpoints and resources tested
- **Business Logic**: Core workflows validated
- **Performance**: Baselines established

### Time Limits
- **Structure Tests**: <15 seconds
- **Unit Tests**: <60 seconds  
- **Integration Tests**: <120 seconds
- **Business Logic**: <180 seconds
- **Performance Tests**: <60 seconds

### Quality Gates
- Safety linter passes
- All phases complete successfully
- Coverage thresholds met
- No hardcoded values (ports, paths)

## 🔗 Related Documentation

### For Implementation
- **[Testing Library README](/scripts/scenarios/testing/README.md)** - Sourceable testing functions
- **[Test System Documentation](../__test/README.md)** - BATS framework internals

### For Development
- **[Development Environment](../devops/development-environment.md)** - Setting up testing tools
- **[CI/CD Pipeline](../devops/ci-cd.md)** - Automated testing integration

### For Operations
- **[Resource Management](../resources/)** - Individual resource testing
- **[Scenario Development](../scenarios/)** - Application testing patterns

## 🆘 Getting Help

### Quick Fixes
- **Tests deleting files?** → [Safety Guidelines](safety/GUIDELINES.md) immediately
- **Tests failing?** → [Troubleshooting Guide](guides/troubleshooting.md)
- **Need examples?** → [Gold Standard Examples](reference/examples.md)

### Advanced Support
- **GitHub Issues**: Use "TESTING" label for testing-related issues
- **Documentation Issues**: Use "DOCS" label for documentation problems
- **Safety Issues**: Use "SAFETY" label for urgent safety concerns

## 🏆 Success Stories

Learn from our best implementations:
- **[Visited Tracker](../scenarios/visited-tracker/)** - 79.4% Go coverage with comprehensive patterns
- **[Browser Automation Studio](../scenarios/browser-automation-studio/)** - UI testing framework (testing patterns in development)
- **Resource Testing Examples** - PostgreSQL, Redis, Ollama integrations

---

**Remember: Effective testing is about building confidence, not just coverage. Test the behaviors that matter, and always prioritize safety.**