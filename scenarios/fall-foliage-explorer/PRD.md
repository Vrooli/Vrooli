# Fall Foliage Explorer - Product Requirements Document

## Executive Summary
**What**: Interactive application for tracking and forecasting fall foliage peaks across North America
**Why**: Help travelers plan autumn trips with data-driven foliage predictions
**Who**: Nature enthusiasts, photographers, travel planners, tourism boards
**Value**: $15K-25K per deployment (tourism partnerships, premium subscriptions)
**Priority**: P0 - Core foliage tracking and prediction functionality must work

## Requirements Checklist

### P0 Requirements (Must Have)
- [x] **Health Check Endpoint**: API responds to /health with status 200
- [x] **Database Connection**: API connects to PostgreSQL and performs CRUD operations
- [x] **Foliage Data API**: Retrieve current foliage status for regions
- [ ] **Weather Integration**: Fetch and store weather data for predictions (n8n workflows missing)
- [ ] **Basic Prediction Engine**: Generate foliage peak predictions using weather data (returns mock data)
- [ ] **Interactive Map UI**: Display regions with foliage status overlays (UI exists but not connected)
- [x] **Lifecycle Management**: setup/develop/test/stop commands work properly

### P1 Requirements (Should Have)
- [x] **User Reports**: Accept and display crowd-sourced foliage reports
- [ ] **Time Slider**: Navigate through past/present/future foliage states
- [ ] **Trip Planning**: Save and manage multi-region trip plans
- [ ] **Photo Gallery**: Display user-submitted photos by region and date

### P2 Requirements (Nice to Have)
- [ ] **AI Predictions**: Use Ollama for advanced pattern analysis
- [ ] **Mobile Responsive**: Optimize UI for mobile devices
- [ ] **Export Features**: Download predictions and trip plans

## Technical Specifications

### Architecture
- **API**: Go service handling data operations and predictions
- **UI**: JavaScript/Node.js with Leaflet.js for interactive maps
- **Storage**: PostgreSQL for historical data, Redis for caching
- **AI**: Ollama for advanced predictions (when implemented)

### Dependencies
- PostgreSQL (required): Historical foliage and weather data
- Redis (required): Real-time data caching
- n8n (required): Weather data workflows
- Ollama (required): AI predictions
- Browserless (optional): Weather data scraping

### API Endpoints
- `GET /health` - Health check ✅
- `GET /api/regions` - List all regions ✅
- `GET /api/foliage?region_id=X` - Get foliage status for region ✅
- `POST /api/predict` - Trigger prediction for region ✅ (mock data)
- `GET /api/weather?region_id=X&date=Y` - Get weather data ✅
- `POST /api/reports` - Submit user report ✅
- `GET /api/reports?region_id=X` - Get user reports for region ✅

### Performance Targets
- API response time: <500ms for data queries
- UI load time: <3s initial load
- Prediction generation: <5s per region
- Map rendering: <2s for overlay updates

## Success Metrics

### Completion Targets
- P0 requirements: 100% required for v1.0
- P1 requirements: 75% target for enhanced version
- P2 requirements: Bonus features for premium tier

### Quality Metrics
- Test coverage: >80% for API endpoints
- Error rate: <1% for API calls
- Uptime: 99.9% for production deployment

### Business Metrics
- User engagement: 5+ minutes average session
- Premium conversion: 10% of free users
- API usage: 1000+ requests/day in season

## Implementation History

### Progress Tracking
**2025-01-29**: 0% → 60% implementation
- Initial assessment complete
- PRD created
- Fixed missing fmt import in API
- Fixed UI server.js syntax error
- Implemented full database integration
- Added proper error handling to all endpoints
- Created user reports functionality
- All API endpoints working (7/7)
- Health checks passing for both API and UI
- Tests passing successfully

### Verified Complete
- Health Check Endpoint: curl http://localhost:17175/health ✅
- Database Connection: Connected to PostgreSQL on port 5433 ✅
- Foliage Data API: curl http://localhost:17175/api/regions ✅
- User Reports: POST/GET /api/reports working ✅
- Lifecycle Management: make run/stop/test all working ✅

### Remaining Work
- Weather Integration: n8n workflow files need creation
- Prediction Engine: Currently returns mock data, needs Ollama integration
- Interactive Map UI: Frontend exists but needs API connection

## Notes
- ✅ FIXED: API now has full database integration
- ✅ FIXED: fmt import added to main.go
- ⚠️ TODO: n8n workflows referenced in service.json but not present
- ⚠️ TODO: UI exists but needs connection to real API endpoints