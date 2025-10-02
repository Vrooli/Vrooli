# Fall Foliage Explorer - Known Issues & Solutions

## Issues Resolved (2025-01-29)

### 1. Missing fmt import in API
- **Issue**: API main.go was missing the fmt import needed for lifecycle check
- **Solution**: Added fmt import to the imports list
- **Status**: ✅ Fixed

### 2. UI server.js syntax error
- **Issue**: Mixed Express and http server code causing syntax error
- **Solution**: Refactored to use only http module for consistency
- **Status**: ✅ Fixed

### 3. Database connection failure
- **Issue**: API couldn't connect to PostgreSQL with default credentials
- **Solution**: Updated connection to use vrooli user and port 5433
- **Status**: ✅ Fixed

### 4. Missing database tables
- **Issue**: Database schema not initialized automatically
- **Solution**: Manually ran schema.sql against the database
- **Status**: ✅ Fixed

## Outstanding Issues

### 1. n8n workflow initialization (Deferred - Low Priority)
- **Issue**: n8n workflows referenced in service.json but files don't exist
- **Impact**: Weather data fetching workflows not functioning (not critical as predictions work via Ollama)
- **Status**: ⚠️ Deferred - Core functionality works without workflows
- **Note**: Ollama AI predictions handle forecasting without requiring separate n8n workflows

### 2. Test infrastructure migration (Recommended)
- **Issue**: Legacy scenario-test.yaml format
- **Impact**: Could benefit from phased testing architecture
- **Status**: 🟡 Recommended - Current tests pass, migration would improve structure
- **TODO**: Migrate to phased testing architecture as recommended in PHASED_TESTING_ARCHITECTURE.md

## Resolved Issues (2025-10-02)

### 3. Ollama integration
- **Issue**: Prediction endpoint returns mock data instead of using Ollama
- **Solution**: Implemented full Ollama integration with llama3.2:latest model
- **Status**: ✅ Fixed

### 4. Export functionality
- **Issue**: No way to export predictions or trip plans
- **Solution**: Implemented CSV and JSON export for both predictions and trip plans
- **Status**: ✅ Fixed

### 5. Photo gallery
- **Issue**: No photo sharing or gallery functionality
- **Solution**: Implemented full photo gallery with upload, filtering, and display
- **Status**: ✅ Fixed

## Notes for Future Improvements

1. **Database Initialization**: Consider automating in the lifecycle setup phase
2. **Environment Variables**: Document required environment variables for production deployment
3. **Error Handling**: API has good error handling but could benefit from structured logging
4. **n8n Workflows**: Consider implementing weather data workflows if real-time weather integration becomes a priority