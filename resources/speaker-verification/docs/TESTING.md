# Speaker Verification Testing

## Running Tests

```bash
# Run all tests
resource-speaker-verification test all

# Run individual test phases
resource-speaker-verification test smoke        # <30s - quick health validation
resource-speaker-verification test integration  # <120s - end-to-end behavior
resource-speaker-verification test unit         # <60s - library internals

# Using test runner directly
./resources/speaker-verification/test/run-tests.sh all
./resources/speaker-verification/test/run-tests.sh smoke
```

## Test Phases

### Smoke Tests

Quick lifecycle and health validation:
- Container running
- `/health` endpoint responsive
- `/ready` endpoint responsive
- `/v1/info` returns valid data
- Config files exist and parse

### Integration Tests

End-to-end behavior with real API calls:
- Profile CRUD (create, read, list, delete)
- Enrollment with generated fixture audio
- Verification against enrolled profile
- Response structure validation
- Error handling for invalid inputs

### Unit Tests

Library and configuration validation:
- Config defaults properly exported
- `runtime.json` and `schema.json` valid JSON
- Threshold is valid float in [0, 1]
- Sample rate is valid integer
- Enrollment minimum seconds is positive

## Test Fixtures

Tests generate synthetic WAV fixtures (sine waves at 440Hz, 16kHz mono) using Python's `wave` module. No external fixture files are required.

Fixture classes:
- **Enrollment**: 3-5 second sine waves for profile creation
- **Verification**: 2 second sine waves for verification
- **Error cases**: Empty files, missing profiles, invalid requests

## Contract Validation

```bash
# Layer 1: Syntax and file structure
scripts/resources/tools/validate-universal-contract.sh --resource speaker-verification --layer 1

# Layer 2: Command registration and functionality
scripts/resources/tools/validate-universal-contract.sh --resource speaker-verification --layer 2
```

## Prerequisites

Tests require the service to be running:

```bash
resource-speaker-verification manage install
resource-speaker-verification manage start
resource-speaker-verification test all
```

Host requirements:
- `curl` for API calls
- `jq` for JSON parsing
- `python3` for fixture generation
- Docker running with the speaker-verification container
