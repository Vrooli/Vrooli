# Huginn Test Fixtures

This directory contains mock data and test helpers for the Huginn resource management tests.

## Structure

- `test_helper.bash` - Main test helper with mocking functions and assertions
- `mock-responses/` - Mock API responses for testing
- `sample-data/` - Sample configuration and data files

## Mock Responses

### agents.json
Mock response for agent listing operations

### scenarios.json
Mock response for scenario listing operations

### events.json
Mock response for event queries

### system-stats.json
Mock response for system statistics

## Usage

All bats test files should load the test helper:

```bash
load test-fixtures/test_helper
```

This provides:
- Mock Docker commands
- Mock curl commands  
- Mock PostgreSQL commands
- Test environment setup/teardown
- Common assertions

## Mock Modes

The test helper supports different mock modes:

- `success` - Normal successful operation
- `not_installed` - Huginn not installed
- `not_running` - Huginn installed but stopped
- `failure` - Command failures

Example:
```bash
mock_docker "not_running"
mock_curl "timeout"
```