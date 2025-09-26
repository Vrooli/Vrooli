# CrewAI Resource - Known Issues and Solutions

## Common Issues

### 1. Test Function Namespacing
**Problem**: Test functions were not properly namespaced, causing `command not found` errors
**Solution**: All functions now use proper namespace prefix `crewai::` 
**Status**: ✅ Fixed

### 2. CrewAI Library Dependencies
**Problem**: CrewAI requires specific versions of Python libraries that may conflict
**Solution**: Mock mode provides fallback functionality when real CrewAI unavailable
**Status**: ✅ Mitigated with dual-mode support

### 3. Memory Persistence
**Problem**: Qdrant connection may timeout on first use
**Solution**: Health check includes Qdrant availability status
**Status**: ✅ Working with proper timeout handling

### 4. Datetime Deprecation Warnings
**Problem**: Python 3.12+ shows deprecation warnings for datetime.utcnow()
**Solution**: Updated to use datetime.now(timezone.utc) throughout
**Status**: ✅ Fixed

### 5. Inject Endpoint Compatibility
**Problem**: Inject endpoint only accepted 'file_path' parameter
**Solution**: Now accepts both 'path' and 'file_path' with auto-detection of file type
**Status**: ✅ Enhanced

### 6. Security - Directory Traversal Prevention
**Problem**: Inject endpoint accepted any file path without validation (security risk)
**Solution**: Added path validation to restrict access to workspace and /tmp directories only
**Status**: ✅ Fixed - Path validation implemented and tested

### 7. LLM Configuration - OpenAI API Key Requirement
**Problem**: CrewAI agents required OpenAI API key even when Ollama was available
**Solution**: Configured agents to use Ollama via litellm when no OpenAI key is set
**Status**: ✅ Fixed - Agents now default to ollama/llama3.2:3b model

### 8. Agent Creation Timeout - Ollama Connectivity
**Problem**: Agent creation would hang indefinitely when trying to connect to unavailable Ollama
**Solution**: Added 0.5s timeout check for Ollama availability before configuring LLM settings
**Status**: ✅ Fixed - Agent creation fails gracefully with error message instead of hanging

## Performance Considerations

### API Response Times
- Health endpoint: <100ms typical
- Crew creation: <200ms typical  
- Task execution: Varies based on complexity (1-30s)
- Memory operations: <500ms typical with Qdrant

### Resource Usage
- Memory: ~50MB base Python process
- CPU: Minimal when idle, spikes during task execution
- Disk: Minimal (JSON storage for crews/agents)

## Integration Notes

### Ollama Integration
CrewAI can use Ollama for local LLM inference:
- Ensure Ollama resource is running
- Models are automatically detected
- Use `/tools` endpoint to verify LLM availability

### Qdrant Integration  
For persistent agent memory:
- Qdrant must be running on port 6333
- Collections are created automatically
- Memory persists across restarts

## Debugging Tips

### Check Service Health
```bash
vrooli resource crewai status
curl http://localhost:8084/health | jq .
```

### View Logs
```bash
vrooli resource crewai logs
tail -f ~/.crewai/logs/crewai.log
```

### Test Endpoints
```bash
# List available tools
curl http://localhost:8084/tools | jq .

# Check crews
curl http://localhost:8084/crews | jq .

# View metrics
curl http://localhost:8084/metrics | jq .
```

### 9. Input Validation - Missing Required Fields
**Problem**: API endpoints accepted empty or invalid values for required fields (name, role, goal)
**Solution**: Added comprehensive validation for agent/crew creation with proper error messages
**Status**: ✅ Fixed - Input validation implemented for all required fields

### 10. Performance - Configuration Caching
**Problem**: Configuration files were loaded from disk on every request
**Solution**: Implemented caching for agent and crew configurations to reduce I/O operations
**Status**: ✅ Fixed - Configuration caching implemented

### 11. Connection Resilience - Qdrant Retry Logic
**Problem**: Single connection attempt to Qdrant could fail on startup
**Solution**: Added retry logic with 3 attempts and delay between retries
**Status**: ✅ Fixed - Connection retry logic implemented

### 12. Health Check Completeness
**Problem**: Health endpoint didn't check all dependencies
**Solution**: Enhanced health check to include Ollama availability status
**Status**: ✅ Fixed - Comprehensive health check implemented

### 13. Test Variable Cleanup
**Problem**: Unused test_crew_response variable causing shellcheck warning
**Solution**: Removed unused variable declaration from integration test
**Status**: ✅ Fixed - Test code cleaned up

## Future Enhancements

### P2 Features Not Implemented
1. **UI Dashboard**: Would require significant frontend work
   - Estimated effort: 2-3 days
   - Value: Medium (CLI/API sufficient for most uses)
   
2. **Workflow Designer**: Complex visual tool
   - Estimated effort: 4-5 days  
   - Value: Medium-High (would improve UX)

### Potential Improvements
- Add webhook support for async task completion
- Implement rate limiting for API endpoints
- Add authentication/authorization layer
- Support for custom tool development
- Integration with more LLM providers