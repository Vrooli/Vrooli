# Test Genie Quick Start

Welcome to Test Genie! This guide will help you get started with running and generating tests for your scenarios.

## Overview

Test Genie is a test coverage dashboard that helps you:

- **Run tests** - Execute test suites with configurable presets
- **Track coverage** - Monitor test health across all scenarios
- **Generate tests** - Create prompts for AI-powered test generation

## Getting Started

### 1. View Your Scenarios

Navigate to the **Runs** tab to see all tracked scenarios. Each scenario shows:

- Current status (passing, failing, idle)
- Pending test requests
- Last run result and timing
- Last failure (if any)

### 2. Run Tests

Click on any scenario to open the detail view, then:

1. Select a preset (Quick, Smoke, or Comprehensive)
2. Optionally enable "Stop on first failed phase"
3. Click "Run tests"

**Presets:**

| Preset | Phases | Use Case |
|--------|--------|----------|
| Quick | Structure, Unit | Fast sanity check |
| Smoke | Structure, Integration | Verify core functionality |
| Comprehensive | All phases | Full coverage |

### 3. Generate Test Prompts

Use the **Generate** tab to create prompts for AI test generation:

1. Select a scenario
2. Choose test phases (Unit, Integration, E2E, Business)
3. Optionally select a preset template
4. Copy the generated prompt
5. Use with your preferred AI assistant

## Test Phases

Test Genie supports several test phases:

### Structure Validation
Validates the scenario's file structure, manifest, and configuration.

### Dependencies
Checks that required commands and resources are available.

### Unit Tests
Runs Go/shell unit tests for individual functions and modules.

### Integration Tests
Executes CLI and Bats integration test suites.

### E2E Playbooks
Runs Browser Automation Studio workflows for end-to-end testing.

### Business Validation
Audits requirements coverage and business rules.

### Performance
Runs build and duration budget checks.

## Tips

- Use the **Dashboard** for quick actions and status overview
- Filter scenarios by searching in the **Runs** tab
- Check **History** to see all past test runs
- Use the **Docs** tab for detailed reference information

## Next Steps

- Read the [Synchronous Execution Guide](synchronous-execution-guide.md) for advanced usage
- Check the [Execution Cheatsheet](sync-execution-cheatsheet.md) for quick reference
- Review [Known Issues](PROBLEMS.md) if you encounter problems
