# Integration Test Migration Inventory

## Resources with Existing Integration Tests (Need Migration)

### 1. unstructured-io (AI)
- **Source**: `resources/unstructured-io/lib/test-api.sh`
- **Type**: Comprehensive API integration test
- **Status**: Ready for migration
- **Target**: `resources/unstructured-io/test/integration-test.sh`
- **Notes**: Well-structured test with health checks, API endpoints, and file processing

### 2. unstructured-io (AI) - Comprehensive Suite  
- **Source**: `resources/unstructured-io/lib/test-suite.sh`
- **Type**: Full functionality test suite
- **Status**: Evaluate if should be merged with test-api.sh or kept separate
- **Target**: Could be integrated into the main integration-test.sh
- **Notes**: More comprehensive than test-api.sh, includes fixtures and edge cases

### 3. comfyui (Automation)
- **Source**: `resources/comfyui/lib/test_workflow.sh`
- **Type**: Workflow execution integration test
- **Status**: Ready for migration
- **Target**: `resources/comfyui/test/integration-test.sh`
- **Notes**: Tests actual workflow execution with test outputs

## Resources with Test Runners (BATS - No Migration Needed)

These orchestrate BATS tests which will be handled by `pnpm run test:shell`:

- `resources/node-red/lib/run-tests.sh` - BATS orchestrator
- `resources/huginn/lib/run-tests.sh` - BATS orchestrator  
- `resources/agent-s2/tests/run_tests.sh` - Python/BATS orchestrator
- `resources/huginn/lib/testing.sh` - Test utilities
- `resources/node-red/lib/testing.sh` - Test utilities

## Resources with Integration Tests Already Working

### 1. ollama (AI)
- **Location**: `__test/integration/services/ollama.sh`
- **Status**: Needs to be moved to standardized location
- **Target**: `resources/ollama/test/integration-test.sh`
- **Notes**: Already follows the standard interface

## Resources Needing Integration Tests Created

Based on the 6 healthy resources detected in testing:

### 1. whisper (AI)
- **Status**: No integration test exists
- **Target**: `resources/whisper/test/integration-test.sh`
- **Needs**: API health check, transcription test with sample audio

### 2. node-red (Automation)  
- **Status**: Has BATS tests but no integration test
- **Target**: `resources/node-red/test/integration-test.sh`
- **Needs**: UI availability, flow execution, API endpoint tests

### 3. vault (Storage)
- **Status**: No integration test exists  
- **Target**: `resources/vault/test/integration-test.sh`
- **Needs**: Seal status, auth, secret operations

### 4. qdrant (Storage)
- **Status**: No integration test exists
- **Target**: `resources/qdrant/test/integration-test.sh`
- **Needs**: Collection operations, vector search, health check

## Migration Priority

### High Priority (Have Existing Tests)
1. **unstructured-io**: Migrate test-api.sh → integration-test.sh
2. **comfyui**: Migrate test_workflow.sh → integration-test.sh  
3. **ollama**: Move from __test/integration/services/ → resources/ai/ollama/lib/

### Medium Priority (Create New Tests)
4. **whisper**: Create integration-test.sh from scratch
5. **vault**: Create integration-test.sh from scratch
6. **qdrant**: Create integration-test.sh from scratch
7. **node-red**: Create integration-test.sh from scratch

## Standard Interface Requirements

All migrated/created tests must follow:
- **Exit codes**: 0 (pass), 1 (fail), 2 (skip)
- **Location**: `scripts/resources/{category}/{resource}/test/integration-test.sh`
- **Format**: Based on integration-test-template.sh
- **Timeout**: 120 seconds max
- **Environment**: Accept standard environment variables (BASE_URL, TIMEOUT, etc.)