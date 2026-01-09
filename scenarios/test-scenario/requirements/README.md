# Test Scenario Requirements Registry

This directory contains the requirements registry for test-scenario, a validation fixture for Vrooli infrastructure testing.

## Structure

```
requirements/
└── index.json    # All requirements (scenario is simple, no modular split needed)
```

## Coverage

Test-scenario is a testing fixture with intentional security vulnerabilities. Requirements are validated through:

- **test**: Automated tests verify lifecycle protection and API functionality
- **manual**: Security scanner validation confirms detection of hardcoded secrets

## Requirement Categories

- **TS-P0-001**: Lifecycle protection enforcement
- **TS-P0-002**: Hardcoded secret detection (intentional vulnerabilities)
- **TS-P0-003**: Basic API server
- **TS-P1-001**: Health endpoint
- **TS-P1-002**: CLI installation

**Total**: 5 requirements (3 P0, 2 P1)

## Validation Strategy

All requirements are validated through:
- Automated tests in `api/*_test.go`
- Manual security scanner verification
- Lifecycle system integration tests

## Purpose

This scenario serves as a permanent test fixture to validate:
- Security scanning infrastructure detects hardcoded secrets
- Lifecycle protection prevents direct binary execution
- Health check infrastructure functions correctly
- CLI installation workflows work as expected
