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

### 1. n8n workflow initialization
- **Issue**: n8n workflows referenced in service.json but files don't exist
- **Impact**: Weather data fetching and prediction workflows not functioning
- **TODO**: Create the missing workflow files:
  - initialization/n8n/foliage-data-fetcher.json
  - initialization/n8n/weather-integration.json
  - initialization/n8n/foliage-predictor.json

### 2. Ollama integration not implemented
- **Issue**: Prediction endpoint returns mock data instead of using Ollama
- **Impact**: No AI-powered predictions
- **TODO**: Implement actual Ollama integration in predictHandler()

### 3. Missing test infrastructure
- **Issue**: No unit tests, integration tests, or UI tests
- **Impact**: Limited test coverage
- **TODO**: Migrate to phased testing architecture as recommended

## Notes for Future Improvements

1. **Database Initialization**: Should be automated in the lifecycle setup phase
2. **Environment Variables**: Consider documenting required environment variables
3. **Error Handling**: API has good error handling but could benefit from structured logging
4. **UI Enhancement**: Current UI files exist but need connection to real API endpoints