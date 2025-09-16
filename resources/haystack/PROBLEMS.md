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

1. **File Upload**: Currently only supports text files, not PDFs or other formats
2. **Pipeline Persistence**: Custom pipelines are not persisted across restarts
3. **Concurrent Indexing**: Batch indexing uses ThreadPoolExecutor with fixed 4 workers
4. **Error Recovery**: No automatic retry mechanism for failed operations

## Future Improvements

1. Add support for PDF and other document formats
2. Implement pipeline persistence using JSON serialization
3. Add configurable worker pool size for batch processing
4. Implement retry logic with exponential backoff
5. Add comprehensive metrics endpoint with Prometheus format
6. Consider adding authentication/authorization layer