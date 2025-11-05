# Vrooli Testing Documentation

Welcome to Vrooli's comprehensive testing documentation. This directory contains all the documentation you need to understand, implement, and maintain testing across the Vrooli ecosystem.

## 🧭 Quick Navigation

**New to testing in Vrooli?** → Start with [Quick Start Guide](guides/quick-start.md) + [Glossary](GLOSSARY.md)
**Need to test safely?** → Read [Safety Guidelines](safety/GUIDELINES.md) first
**Looking for specific info?** → Use the [Quick Finder](#-quick-finder) below or [Testing Hub](../TESTING.md) decision tree
**Unfamiliar term?** → Check the [Testing Glossary](GLOSSARY.md) for definitions

## 🔍 Quick Finder

**I want to...**

| Goal | Primary Guide | Supporting Docs |
|------|---------------|-----------------|
| 🚀 **Write my first test** | [Quick Start](guides/quick-start.md) | [Safety Guidelines](safety/GUIDELINES.md) |
| 📋 **Understand requirement tracking** | [Requirement Flow](architecture/REQUIREMENT_FLOW.md) | [Requirement Tracking Guide](guides/requirement-tracking.md) |
| 🔧 **Test a scenario (full-stack app)** | [Scenario Testing](guides/scenario-testing.md) | [Phased Testing](architecture/PHASED_TESTING.md) |
| 🧪 **Write unit tests** | [Scenario Unit Testing](guides/scenario-unit-testing.md) | [Examples](reference/examples.md) |
| 🎨 **Design testable UIs** | [Writing Testable UIs](guides/writing-testable-uis.md) | [UI Automation with BAS](guides/ui-automation-with-bas.md) |
| 🤖 **Test UI workflows** | [UI Automation with BAS](guides/ui-automation-with-bas.md) | [End-to-End Example](guides/end-to-end-example.md) |
| 🔮 **Generate tests with AI** | [test-genie Integration](guides/test-genie-integration.md) | [Test Generation](guides/test-generation.md) |
| 🔌 **Test a resource integration** | [Resource Unit Testing](guides/resource-unit-testing.md) | [Strategy](architecture/STRATEGY.md) |
| 💻 **Test CLI commands** | [CLI Testing](guides/cli-testing.md) | [Safety Guidelines](safety/GUIDELINES.md) |
| 🤖 **Generate tests automatically** | [Test Generation](guides/test-generation.md) | [test-genie scenario](/scenarios/test-genie/) |
| 🐛 **Debug failing tests** | [Troubleshooting](guides/troubleshooting.md) | [Examples](reference/examples.md) |
| 🏗️ **Understand the architecture** | [Phased Testing](architecture/PHASED_TESTING.md) | [Requirement Flow](architecture/REQUIREMENT_FLOW.md) |
| 💻 **Use testing helper functions** | [Shell Libraries](reference/shell-libraries.md) | [Testing Library README](/scripts/scenarios/testing/README.md) |

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
- **[End-to-End Example](guides/end-to-end-example.md)** - Complete flow from PRD to coverage ⭐ **NEW**
- **[Scenario Testing](guides/scenario-testing.md)** - Test complete applications
- **[Scenario Unit Testing](guides/scenario-unit-testing.md)** - Go, Node.js, Python unit tests
- **[Resource Unit Testing](guides/resource-unit-testing.md)** - Test Vrooli resources
- **[Writing Testable UIs](guides/writing-testable-uis.md)** - Design UIs for automation ⭐ **NEW**
- **[UI Automation with BAS](guides/ui-automation-with-bas.md)** - Write JSON workflow tests ⭐ **EXPANDED**
- **[test-genie Integration](guides/test-genie-integration.md)** - AI-powered test generation ⭐ **NEW**
- **[CLI Testing](guides/cli-testing.md)** - BATS framework for command-line tools
- **[Troubleshooting](guides/troubleshooting.md)** - Debug and fix common issues

### [📐 Architecture](architecture/)
**Core concepts and design principles**

Deep dives into how Vrooli's testing system works and why.

- **[Requirement Flow](architecture/REQUIREMENT_FLOW.md)** - End-to-end flow from PRD to test validation ⭐ **START HERE**
- **[Phased Testing](architecture/PHASED_TESTING.md)** - 6-phase progressive validation system
- **[Strategy](architecture/STRATEGY.md)** - Three-layer validation for resources
- **[Infrastructure](architecture/infrastructure.md)** - BATS, stubbing, and testing tools

### [📖 Reference](reference/)
**API documentation and comprehensive examples**

Detailed reference material for testing libraries and implementations.

- **[Shell Libraries](reference/shell-libraries.md)** - Testing library function reference
- **[Test Runners](reference/test-runners.md)** - Available test execution methods
- **[Requirement Schema](reference/requirement-schema.md)** - JSON schema for requirement registry
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
| **Track requirements** | [Requirement Tracking](guides/requirement-tracking.md) | [Phased Testing](architecture/PHASED_TESTING.md) |
| **Test UI workflows** | [UI Automation with BAS](guides/ui-automation-with-bas.md) | [Examples](reference/examples.md) |
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

## 🔗 Integration Topics

### 📋 Requirement Tracking
**Essential for production scenarios** - Automatically track PRD requirements through to test validation.

- **[Requirement Tracking Guide](guides/requirement-tracking.md)** - Complete system overview
- **[Requirement Schema Reference](reference/requirement-schema.md)** - Registry structure (Phase 2)
- **[@vrooli/vitest-requirement-reporter](../../packages/vitest-requirement-reporter/README.md)** - Automatic tracking for Vitest

**Key Features:**
- Tag tests with `[REQ:ID]` for automatic coverage tracking
- Live status from phase results
- Parent-child requirement hierarchies
- Auto-sync requirement files from test results

### 🤖 UI Automation
**For complex user journeys** - Use Browser Automation Studio workflows for declarative UI testing.

- **[UI Automation with BAS](guides/ui-automation-with-bas.md)** - Workflow-based testing guide
- **[browser-automation-studio](/scenarios/browser-automation-studio/)** - Self-testing reference implementation

**Key Features:**
- Declarative JSON workflows (version controlled)
- Visual workflow authoring in BAS UI
- Integrated with requirement tracking via `automation` validation type
- API-native execution (no manual clicking)

## 🔗 Related Documentation

### 📦 Package Documentation
Official packages that support Vrooli's testing system.

- **[@vrooli/vitest-requirement-reporter](../../packages/vitest-requirement-reporter/README.md)** - Vitest requirement tracking integration
- **[Testing Shell Libraries](/scripts/scenarios/testing/README.md)** - Sourceable bash testing functions
- **[Requirement Scripts](/scripts/requirements/)** - Registry validation and reporting tools

### For Implementation
- **[Testing Library README](/scripts/scenarios/testing/README.md)** - Sourceable testing functions
- **[Test System Documentation](../../__test/README.md)** - BATS framework internals

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