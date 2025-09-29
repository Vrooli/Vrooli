# Problems & Solutions

## Issues Discovered During Implementation

### 1. API Port Configuration
**Problem**: The API was expected on port 19840 but was actually assigned port 19914.
**Solution**: Use `vrooli scenario port data-tools API_PORT` to get the dynamically assigned port.

### 2. Go Build Errors
**Problem**: Building API with just `main.go` failed due to missing handlers.
**Solution**: Build with all Go files: `go build -o api ./cmd/server/*.go`

### 3. Dataset Loading Not Implemented
**Problem**: Transform and validate endpoints couldn't load datasets from storage.
**Solution**: Currently requires inline data. Future enhancement needed for MinIO integration.

### 4. Authentication Token
**Problem**: API requires Bearer token authentication but wasn't documented.
**Solution**: Use `Authorization: Bearer data-tools-secret-token` header or set `DATA_TOOLS_API_TOKEN` environment variable.

## Performance Considerations

- **Large Dataset Handling**: Currently limited to in-memory processing. Datasets > 10GB should use streaming.
- **Query Performance**: No query optimization or indexing implemented yet.
- **Streaming Latency**: WebSocket connections not implemented, using polling for now.

## Future Enhancements

1. **MinIO Integration**: Store and retrieve large datasets from object storage
2. **Query Optimization**: Add query planner and execution optimization
3. **Advanced Analytics**: Implement P1 features like correlation and regression
4. **Natural Language Queries**: Integrate with Ollama for NL to SQL conversion
5. **Distributed Processing**: Support for multi-node processing for large datasets