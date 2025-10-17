# AutoGPT Resource - Known Problems

## Current Issues

### 1. Port Conflict
**Problem**: Original configuration used port 8080 which conflicts with OpenTripPlanner
**Solution**: Changed default port to 8501 in config/defaults.sh
**Status**: FIXED

### 2. Missing v2.0 Contract Files
**Problem**: Resource was missing required v2.0 contract files:
- PRD.md
- config/defaults.sh
- config/schema.json
- lib/core.sh (partially missing)
- lib/test.sh (incomplete)
- test/run-tests.sh
- test/phases/*.sh
**Solution**: Created all missing files following v2.0 universal contract
**Status**: FIXED

### 3. CLI Integration Issues
**Problem**: CLI was using legacy function names instead of v2.0 compliant core functions
**Solution**: Updated cli.sh to use autogpt::core::* functions
**Status**: FIXED

### 4. log::test Function Not Found
**Problem**: Test scripts reference undefined log::test function
**Solution**: Replaced with log::info for now
**Status**: PARTIAL (works but should use proper test logging)

### 5. Docker Image Not Tested
**Problem**: Unable to fully test Docker container as it requires pulling large image
**Impact**: Cannot verify health checks, agent creation, or task execution
**Status**: PENDING

## Future Improvements Needed

1. **Port Registry Integration**: Add AutoGPT port to scripts/resources/port_registry.sh
2. **Agent Templates**: Create working example agent configurations
3. **LLM Provider Auto-Detection**: Improve autogpt_get_llm_config() to auto-configure
4. **Health Check Implementation**: Verify actual health endpoint exists in container
5. **Memory Backend Integration**: Test Redis/PostgreSQL integration
6. **Tool Integration**: Enable browserless and judge0 connections

## Testing Gaps

- Docker container installation and startup not tested
- Health endpoint response format unknown
- Agent creation/execution API not tested
- LLM provider integration not tested
- Memory persistence not tested

## Dependencies

### Required
- Docker (for container)
- Redis (if using Redis memory backend)

### Optional
- OpenRouter/Ollama (for LLM)
- PostgreSQL (for long-term memory)
- Browserless (for web interaction)
- Judge0 (for code execution)