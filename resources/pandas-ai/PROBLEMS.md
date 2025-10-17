# Known Problems and Solutions - Pandas AI

## Resource Limit Issues
**Problem**: Setting memory limits with `resource.setrlimit(resource.RLIMIT_AS, ...)` causes thread creation failures.
**Impact**: Analysis endpoints fail with "can't start new thread" errors
**Solution**: Removed memory limits. Consider external sandboxing for production deployments.

## Security Considerations for Direct Execution
**Problem**: Direct pandas code execution poses security risks if not properly validated.
**Mitigations Implemented**:
- Comprehensive dangerous pattern detection (25+ patterns)
- File path protection blocking system directories
- Command injection prevention
- Syntax validation before execution
- 10-second timeout protection
- Safe mode enforcement by default

**Recommendations**:
- Always use safe_mode=true in production
- Consider running in isolated containers or VMs
- Implement rate limiting for execution endpoints
- Monitor for suspicious patterns in logs

## Performance Considerations
**Problem**: Large DataFrames can cause memory issues or slow responses
**Mitigations**:
- Automatic truncation to 1000 rows for large results
- Configurable MAX_ROWS setting
- Smart caching to avoid repeated computations
- Execution time tracking for monitoring

## Integration Notes
**Matplotlib Backend**: Uses non-interactive backend (Agg) to avoid display issues in containerized environments.
**Thread Safety**: FastAPI handles concurrent requests, but matplotlib operations should be properly cleaned up with `plt.close('all')`.

## Testing Challenges
**Timeout Testing**: Some security tests may appear to timeout due to curl behavior with piped output. The API actually responds correctly with security errors.

## Future Improvements Needed
1. Add resource monitoring and automatic cleanup for long-running processes
2. Implement user session management for multi-user scenarios
3. Add support for more data sources (S3, BigQuery, etc.)
4. Implement execution history and audit logging
5. Add support for custom pandas extensions and plugins