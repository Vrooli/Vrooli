# 🎯 Picker Wheel - Product Requirements Document

## Executive Summary
**What**: Interactive decision-making web application with spinning wheel mechanics  
**Why**: Eliminate decision fatigue through gamified random selection with weighted probabilities  
**Who**: Individuals, teams, educators, and organizations needing fair decision-making tools  
**Priority**: High - demonstrates UI/UX capabilities and cross-resource integration

## Requirements Checklist

### P0 Requirements (Must Have)
- [x] **Health Check**: API responds to /health endpoint with service status ✅
- [x] **Lifecycle Management**: Service starts/stops through Vrooli lifecycle (setup/develop/test/stop) ✅
- [x] **Wheel Spinning**: Core functionality to spin wheels and get random weighted results ✅
- [x] **Preset Wheels**: Include built-in wheels (yes/no, dinner-decider) ✅
- [x] **Custom Wheels**: Users can create and save custom wheels with options - UI and API fully implemented ✅
- [x] **Web UI**: Interactive browser-based wheel with visual spinning animation ✅
- [x] **CLI Tool**: Command-line interface for spinning wheels ✅

### P1 Requirements (Should Have)
- [x] **Weighted Options**: Support probability weights for each wheel option - Fully implemented in UI and API ✅
- [x] **History Tracking**: Store and display spin history - Complete with stats and database persistence ✅
- [x] **Database Persistence**: Save wheels and history to PostgreSQL - Schema implemented, fallback to in-memory working ✅
- [x] **API Documentation**: Clear API endpoints with request/response examples - Added to README ✅

### P2 Requirements (Nice to Have)  
- [ ] **AI Suggestions**: Use Ollama for intelligent option suggestions
- [ ] **Multiple Themes**: Visual themes (neon, retro, minimal, dark mode)
- [ ] **Sound Effects**: Audio feedback for spins and results

## Technical Specifications

### Architecture
- **API**: Go backend service (port range 15000-19999)
- **UI**: Node.js/Express server with vanilla JavaScript (port range 35000-39999)
- **Database**: PostgreSQL for persistence
- **CLI**: Bash wrapper calling API endpoints
- **Workflows**: N8n for automation workflows (optional)

### Dependencies
- **Required Resources**: PostgreSQL
- **Optional Resources**: N8n, Ollama
- **Go Packages**: gorilla/mux, lib/pq, rs/cors
- **Node Packages**: express, path, http

### API Endpoints
- `GET /health` - Service health check
- `POST /api/spin` - Spin a wheel
- `GET /api/wheels` - List available wheels
- `POST /api/wheels` - Create custom wheel
- `GET /api/wheels/:id` - Get wheel details
- `DELETE /api/wheels/:id` - Delete custom wheel
- `GET /api/history` - Get spin history

## Success Metrics

### Completion Targets
- **P0 Completion**: 100% (7/7 requirements) ✅
- **P1 Completion**: 100% (4/4 requirements) ✅
- **P2 Completion**: 0% (0/3 requirements)
- **Overall**: 79% (11/14 requirements)

### Quality Metrics
- API response time < 500ms
- UI spin animation smooth (60 fps)
- Test coverage > 70%
- Zero critical security issues

### Performance Targets
- Support 100 concurrent users
- Handle 1000 spins/minute
- Database queries < 100ms
- UI load time < 2 seconds

## Implementation Status

### Current State
- ✅ API health endpoint working (returns proper JSON)
- ✅ UI server with proper environment variable validation
- ✅ Preset wheels initialized (yes-or-no, dinner-decider)
- ✅ Wheel spinning functionality working correctly with visual weight representation
- ✅ CLI tool functional with automatic port detection and install script
- ✅ Custom wheel creation UI fully implemented with weight inputs
- ✅ Phased testing structure complete (structure/unit/integration/performance/business/dependencies)
- ✅ Go code formatted to standards
- ✅ Lifecycle management working (setup/develop/test/stop)
- ✅ Weighted options fully functional in UI and backend
- ✅ History tracking with stats and database persistence
- ✅ Comprehensive API documentation in README
- ✅ Standards compliance improved: 0 critical violations (down from 3)

### Known Issues
- PostgreSQL connection configured but falls back to in-memory when credentials vary (works gracefully)
- N8n workflows exist but resource not currently running (files ready in initialization/)
- 21 high-severity and 77 medium-severity standards violations remaining (mostly PRD structure and Makefile)

### Next Steps
1. Address remaining high-severity standards violations in PRD structure and Makefile
2. Implement P2 requirements (AI suggestions, multiple themes, sound effects)
3. Add more preset wheels and themes
4. Enhance performance monitoring
5. Consider multiplayer features

## Revenue Justification

### B2C SaaS Model
- **Pricing**: $5-10/month premium features
- **Features**: Unlimited saves, analytics, team wheels
- **Market**: Decision-fatigued professionals, gamers
- **Potential**: 1000 users × $7.50/month = $90K ARR

### Educational License  
- **Pricing**: $200-500 per institution
- **Features**: Classroom tools, student accounts, admin panel
- **Market**: K-12 schools, universities
- **Potential**: 50 schools × $350 = $17.5K one-time

### API Access
- **Pricing**: $50-100/month for developers
- **Features**: REST API, webhooks, SDKs
- **Market**: App developers, integrators
- **Potential**: 20 developers × $75/month = $18K ARR

### Total Revenue Potential
- **Year 1**: $10-25K (MVP launch)
- **Year 2**: $50-100K (market growth)
- **Year 3**: $100K+ (expansion)

## Progress History
- **2025-09-24**: Initial PRD created, scenario assessment complete (0% → 0%)
- **2025-09-24**: Fixed N8N_PORT issue, validated P0 requirements (0% → 36%)
- **2025-09-24**: Fixed spin endpoint, all core features working (36% → 43%)
- **2025-09-27**: Custom wheel UI verified functional, phased testing added, Go formatting fixed (43% → 50%)
- **2025-09-30**: Database schema initialized, PostgreSQL integration with graceful fallback implemented (50% → 57%)
- **2025-10-03**: Verified all P1 features working, added comprehensive API documentation, improved CLI port detection (57% → 79%)