# Haystack Resource - Known Issues and Solutions

## Issues Resolved (2025-01-16)

### 1. Batch Indexing Test Format Issue
**Problem**: Integration test for batch_index endpoint was failing due to incorrect payload format.
**Root Cause**: Test was sending flat array instead of array of arrays as required by the endpoint.
**Solution**: Updated test payload from `[{doc1}, {doc2}]` to `[[{doc1}], [{doc2}]]` format.

### 2. Restart Command Non-Zero Exit Code
**Problem**: The restart command was returning non-zero exit code even on successful restart.
**Root Cause**: The restart function wasn't properly handling the return value from the start function.
**Solution**: Added explicit success/failure handling with proper return codes.

### 3. Info Command Test Case Sensitivity
**Problem**: Integration test for info command was failing to detect required fields.
**Root Cause**: Test was looking for lowercase field names but info outputs capitalized names.
**Solution**: Changed test to use case-insensitive grep matching (`grep -qi`).

### 4. Package Validation Incorrect Package Name
**Problem**: Package validation was failing to find Haystack package.
**Root Cause**: Looking for package "haystack" but actual package is "haystack-ai".
**Solution**: Updated package list to use correct package names.

## Current Best Practices

### Health Checks
- Always use `timeout 5` wrapper for all health check calls
- Validate health after service start before declaring success
- Check both process existence and endpoint responsiveness

### Testing
- Run full test suite after any changes: `vrooli resource haystack test all`
- Ensure all three phases pass: smoke, integration, unit
- Document any new test requirements in integration tests

### Error Handling
- Log all errors with appropriate severity level
- Provide recovery hints in error messages
- Ensure graceful degradation when optional dependencies unavailable

## Performance Characteristics

### Observed Metrics
- **Startup Time**: 5-10 seconds typical, 15 seconds max
- **Health Check Response**: <100ms consistently
- **Document Indexing**: ~27ms per document average
- **Query Response**: 10-30ms for vector search
- **Batch Processing**: Can handle 100+ documents efficiently
- **Memory Usage**: ~750MB with Qdrant integration

### Optimization Opportunities
1. Consider lazy loading of embedding models to reduce startup time
2. Implement connection pooling for Qdrant client
3. Add caching layer for frequently accessed documents
4. Consider using lighter embedding models for faster indexing

## Integration Notes

### Qdrant Integration
- Automatically connects if Qdrant is running on localhost:6333
- Falls back gracefully to InMemory store if unavailable
- Collection name: "haystack_docs" by default

### Ollama Integration
- Checks availability on localhost:11434
- Uses llama3.2 model by default
- Enhances queries with LLM-based expansion

### Multi-Language Support
- Uses langdetect library for automatic language detection
- Supports 55+ languages with confidence scoring
- Adds language metadata to all indexed documents

## Known Limitations

1. ~~**File Upload**: Currently only supports text files, not PDFs or other formats~~ **RESOLVED 2025-01-16**
2. **Pipeline Persistence**: Custom pipelines are not persisted across restarts
3. ~~**Concurrent Indexing**: Batch indexing uses ThreadPoolExecutor with fixed 4 workers~~ **RESOLVED 2025-01-16**
4. ~~**Error Recovery**: No automatic retry mechanism for failed operations~~ **RESOLVED 2025-01-16**

## Recent Improvements (2025-01-16)

### Latest Enhancements (Second Iteration)
- **PDF Support Added**: Can now process PDF documents via upload endpoint
- **Configurable Worker Pool**: HAYSTACK_WORKER_POOL_SIZE environment variable controls concurrency
- **Exponential Backoff Retry**: Intelligent retry logic with exponential backoff for resilience
- **Prometheus Metrics Endpoint**: /metrics endpoint exports comprehensive metrics in Prometheus format
- **Enhanced Error Tracking**: Errors tracked by type for better diagnostics

### Added Retry Logic with Recovery Hints (First Iteration)
- Content operations now include 3-attempt retry logic for transient failures
- Each operation provides clear recovery hints on failure
- Improved error messages guide users to resolution steps

### Installation Validation Function
- Added `haystack::validate_installation()` to check system integrity
- Validates: venv existence, Python binary, server script, critical packages
- Automatically called during start to catch issues early
- Provides specific recovery commands for each type of issue

### Enhanced Error Handling
- Service running check before content operations
- HTTP status code validation for API calls
- Proper timeout handling on all network operations (10s for queries, 30s for uploads)
- Graceful degradation with informative error messages

## Future Improvements

1. ~~Add support for PDF and other document formats~~ **COMPLETED 2025-01-16**
2. Implement pipeline persistence using JSON serialization
3. ~~Add configurable worker pool size for batch processing~~ **COMPLETED 2025-01-16**
4. ~~Add exponential backoff for retry delays~~ **COMPLETED 2025-01-16**
5. ~~Add comprehensive metrics endpoint with Prometheus format~~ **COMPLETED 2025-01-16**
6. Consider adding authentication/authorization layer
7. Add support for more document formats (DOCX, ODT, HTML)
8. Implement caching layer for frequently accessed documents
9. Add document versioning and update tracking