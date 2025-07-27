# Agent S2 Testing

This directory contains all testing-related files for Agent S2.

## Directory Structure

```
testing/
├── test-outputs/      # Generated test outputs (not tracked in git)
│   ├── screenshots/   # Screenshot test outputs
│   ├── logs/         # Test execution logs
│   └── reports/      # Test reports and summaries
├── integration/      # Integration tests
├── unit/            # Unit tests
└── e2e/             # End-to-end tests
```

## Test Outputs

All test outputs are stored in `test-outputs/` and are ignored by git. This includes:
- Screenshots generated during testing
- Log files from test runs
- Test reports and coverage data

## Running Tests

```bash
# Run all tests
./manage.sh --action test

# Clean test outputs
./testing/cleanup.sh
```

## Important Notes

- Test outputs are not committed to the repository
- Run cleanup.sh periodically to remove old test outputs
- The test-outputs directory has a size limit to prevent disk space issues