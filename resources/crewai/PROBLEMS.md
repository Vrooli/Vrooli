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