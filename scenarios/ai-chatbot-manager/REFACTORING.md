# AI Chatbot Manager - Refactoring Documentation

## Overview
This document describes the comprehensive refactoring performed on the AI Chatbot Manager scenario to align with Vrooli's production standards as established by the visited-tracker reference implementation.

## Key Changes

### 1. Removed All Hardcoded Values ✅
- **Before**: Had hardcoded `defaultPort = "8090"` and `defaultOllamaURL = "http://localhost:11434"`
- **After**: All configuration comes from environment variables with NO fallbacks
- **Impact**: Service now properly fails if required environment variables are not set

### 2. Modular Code Architecture ✅
Split the monolithic 1065-line `main.go` into:
- `main.go` (49 lines) - Simple entry point
- `config.go` - Configuration management
- `models.go` - Data structures
- `database.go` - Database operations with proper connection pooling
- `handlers.go` - HTTP request handlers
- `websocket.go` - WebSocket implementation
- `widget.go` - Widget generation
- `middleware.go` - Request logging, CORS, rate limiting
- `server.go` - Server initialization and routing
- `logger.go` - Structured logging

### 3. Fixed SQL Issues ✅
- Removed references to non-existent database functions in analytics queries
- Simplified engagement score calculation
- Made analytics queries more robust and efficient

### 4. Proper WebSocket Implementation ✅
- Complete WebSocket chat handler with reconnection logic
- Message queuing and proper connection management
- Fallback to HTTP when WebSocket fails

### 5. Dynamic Widget Generation ✅
- No hardcoded localhost URLs
- Widget auto-detects API URL from environment
- Multiple configuration options for deployment

### 6. Enhanced Error Handling ✅
- Exponential backoff for database connections
- Graceful shutdown handling
- Recovery middleware for panic handling
- Comprehensive error responses

### 7. Production-Ready Features ✅
- Rate limiting middleware
- Request logging with duration tracking
- CORS configuration
- Health check endpoint with dependency verification
- Structured logging with consistent format

## Environment Variables

### Required (No Defaults)
- `API_PORT` - API server port (lifecycle system provides from range 15000-19999)
- `OLLAMA_URL` - Ollama API endpoint for AI responses
- `VROOLI_LIFECYCLE_MANAGED` - Must be "true" (set by lifecycle system)

### Database Configuration
Either provide:
- `POSTGRES_URL` - Complete connection string

OR all of:
- `POSTGRES_HOST` - Database host
- `POSTGRES_PORT` - Database port
- `POSTGRES_USER` - Database username
- `POSTGRES_PASSWORD` - Database password
- `POSTGRES_DB` - Database name

## API Structure

### Endpoints (v1 API)
```
GET  /health                      - Health check
GET  /api/v1/chatbots             - List chatbots
POST /api/v1/chatbots             - Create chatbot
GET  /api/v1/chatbots/{id}        - Get chatbot
PUT  /api/v1/chatbots/{id}        - Update chatbot
DELETE /api/v1/chatbots/{id}      - Delete chatbot
POST /api/v1/chat/{id}            - Send chat message
WS   /api/v1/ws/{id}              - WebSocket chat
GET  /api/v1/analytics/{id}       - Get analytics
GET  /api/v1/chatbots/{id}/widget - Get widget code
```

## Security Improvements
- No default passwords or connection strings
- Input validation on all endpoints
- Rate limiting to prevent abuse
- CORS properly configured
- SQL injection prevention via parameterized queries

## Performance Improvements
- Connection pooling for database
- Concurrent request handling
- Efficient SQL queries
- WebSocket for real-time communication
- Caching where appropriate

## Testing
Run tests with:
```bash
vrooli scenario test ai-chatbot-manager
```

## Deployment Considerations

### Widget Integration
Users must configure the widget with their API URL:

```html
<!-- Option 1: Global variable -->
<script>
  window.CHATBOT_API_URL = 'https://your-api.com';
</script>

<!-- Option 2: Meta tag -->
<meta name="chatbot-api-url" content="https://your-api.com">

<!-- Then include the widget -->
<script src="widget.js"></script>
```

### Database Setup
The schema includes:
- Proper indexes for performance
- Triggers for updated_at timestamps
- Functions for engagement scoring
- Views for analytics summaries

### Monitoring
- Health endpoint for uptime monitoring
- Structured logs for debugging
- Request duration tracking
- Error recovery and reporting

## Remaining Work

### Future Enhancements (Not Critical)
1. **Database Migrations**: Add migration system for schema versioning
2. **Authentication**: Add JWT-based authentication for API
3. **Metrics**: Add Prometheus metrics endpoint
4. **Caching**: Add Redis for session caching
5. **Tests**: Add comprehensive unit and integration tests

## Validation Checklist

- [x] No hardcoded ports or URLs
- [x] Lifecycle system compliance
- [x] Modular code structure
- [x] Proper error handling
- [x] WebSocket implementation
- [x] Dynamic configuration
- [x] Production-ready middleware
- [x] SQL query optimization
- [x] Comprehensive logging
- [x] Graceful shutdown

## Migration Notes

For existing deployments:
1. Update environment variables to remove any reliance on defaults
2. Ensure OLLAMA_URL is explicitly set
3. Database schema is backward compatible
4. Widget code needs update for dynamic URL configuration

## Standards Compliance

This refactoring brings the AI Chatbot Manager in line with:
- Vrooli lifecycle v2.0 standards
- Go best practices for web services
- Production deployment requirements
- Security and performance standards
- Visited-tracker reference implementation patterns