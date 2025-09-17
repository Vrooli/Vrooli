# OpenTripPlanner Product Requirements Document (PRD)

## Executive Summary
**What**: OpenTripPlanner (OTP) is an open-source multimodal trip planning engine supporting transit, biking, walking, and demand-responsive transport with GTFS/OSM inputs.  
**Why**: Unlocks the Transportation & Mobility branch, enabling route simulation, transit evaluation, and dispatch solutions integrated with fleet telemetry and smart-city dashboards.  
**Who**: Transportation agencies, mobility providers, urban planners, and scenarios requiring routing capabilities.  
**Value**: $35K-50K per deployment through transit optimization, fleet coordination, and mobility-as-a-service orchestration.  
**Priority**: High - foundational for transportation scenarios.

## Progress Summary
- **Current Status**: 60% Complete (Docker working, graph building functional, API serving)
- **Phase**: Core Implementation - P0 Requirements Progressing
- **Last Updated**: 2025-09-17
- **Revenue Potential**: $35K-50K per deployment

### Completed Work
- ✓ Basic v2.0 contract structure created
- ✓ CLI commands implemented (help, info, manage, test, content, status, logs)
- ✓ Docker Compose configuration prepared
- ✓ Test infrastructure scaffolded (smoke, integration, unit)
- ✓ Port allocation registered (8080)
- ✓ Configuration files created (defaults.sh, schema.json, runtime.json)

## Requirements Checklist

### P0 Requirements (Must Have - Core Functionality)
- [x] **Dockerized Stack**: Ship a Dockerized OpenTripPlanner stack with sample GTFS and OpenStreetMap data plus CLI commands for building graphs and planning trips. ✅ COMPLETE
  - Acceptance Criteria: Docker Compose running OTP with Portland sample data, accessible API on configured port
  - Test Command: `vrooli resource opentripplanner test smoke`
  - Verified: 2025-09-17 - Container runs, Portland data downloaded and loaded
  
- [x] **Graph Building**: CLI commands for importing GTFS/OSM data and building routing graphs. ✅ COMPLETE
  - Acceptance Criteria: Can build graph from GTFS + OSM files via CLI
  - Test Command: `vrooli resource opentripplanner content execute --action build-graph`
  - Verified: 2025-09-17 - Built graph with Portland GTFS + OSM data successfully
  
- [x] **Trip Planning API**: Expose OTP's REST API for multimodal journey planning. (PARTIAL: API running, needs route testing)
  - Acceptance Criteria: API endpoint returns valid itineraries for test coordinates
  - Test Command: `curl -s http://localhost:${OTP_PORT}/otp/routers/default/plan?fromPlace=45.5,-122.6&toPlace=45.52,-122.65&mode=TRANSIT,WALK`
  - Status: API server running, debug UI accessible, routing endpoint needs validation
  
- [ ] **Smoke Tests**: Provide smoke tests that execute representative itineraries (e.g., bus + rail) and validate routing responses.
  - Acceptance Criteria: Tests validate bus+rail, bike+transit, walk-only routes
  - Test Command: `vrooli resource opentripplanner test integration`
  
- [x] **Health Checks**: Expose health/status checks ensuring the OTP API and graph builder are up before scenarios depend on them. ✅ COMPLETE
  - Acceptance Criteria: Health endpoint responds with graph status and readiness
  - Test Command: `timeout 5 curl -sf http://localhost:${OTP_PORT}/`
  - Verified: 2025-09-17 - Health checks working, debug UI accessible

- [x] **Lifecycle Management**: Complete v2.0 lifecycle support (install/start/stop/restart/uninstall). ✅ COMPLETE
  - Acceptance Criteria: All lifecycle commands work per universal contract
  - Test Command: `vrooli resource opentripplanner manage start --wait`
  - Verified: 2025-09-17 - All lifecycle commands working correctly

- [x] **Content Management**: GTFS/OSM data import and management through standardized CLI. ✅ COMPLETE
  - Acceptance Criteria: Can add, list, remove transit feeds and map data
  - Test Command: `vrooli resource opentripplanner content list`
  - Verified: 2025-09-17 - Content management commands working, graph building tested

### P1 Requirements (Should Have - Enhanced Features)
- [ ] **GTFS-RT Support**: Add support for GTFS-RT ingestion and document how to replay feeds for simulation scenarios.
  - Acceptance Criteria: Real-time updates affect routing results
  - Test Command: `vrooli resource opentripplanner content add --type gtfs-rt --url http://feed.example.com`
  
- [ ] **Postgres/Qdrant Integration**: Integrate OTP outputs with QGIS/Mapbox visualizations and store analytics in Postgres/Qdrant for scenario use.
  - Acceptance Criteria: Route analytics stored in PostGIS, semantic search in Qdrant
  - Test Command: `vrooli resource opentripplanner content execute --analyze-routes --store-results`
  
- [ ] **Traccar Integration**: Include automation workflows that coordinate with Traccar for fleet tracking and dispatch alerts.
  - Acceptance Criteria: N8n workflow connects OTP routes to Traccar fleet positions
  - Test Command: `vrooli resource opentripplanner content execute --fleet-dispatch --traccar-sync`

### P2 Requirements (Nice to Have - Advanced Features)
- [ ] **MaaS Templates**: Offer templates for Mobility-as-a-Service orchestration, including fare calculation examples.
  - Acceptance Criteria: Example scenarios for fare integration and booking
  - Documentation: `docs/MAAS_TEMPLATES.md`
  
- [ ] **Digital Twin Connectors**: Provide connectors to digital twin frameworks (Eclipse Ditto) for network-wide status monitoring.
  - Acceptance Criteria: Eclipse Ditto integration for transit network state
  - Test Command: `vrooli resource opentripplanner content execute --sync-digital-twin`
  
- [ ] **Benchmarking Scripts**: Create benchmarking scripts that compare travel-time scenarios across policy or infrastructure changes.
  - Acceptance Criteria: Compare routing before/after network changes
  - Test Command: `vrooli resource opentripplanner content execute --benchmark-scenarios`

## Technical Specifications

### Architecture
- **Core**: OpenTripPlanner 2.x Java application
- **Runtime**: Docker container with JVM
- **Data Storage**: Graph cache in mounted volume
- **API**: RESTful HTTP API on configurable port
- **Dependencies**: PostGIS (optional), Redis (cache), N8n (automation)

### Performance Requirements
- Graph build: <60s for sample data
- Route planning: <500ms for typical queries
- Health check: <1s response time
- Memory: 2-4GB for medium-sized regions

### Security Considerations
- No hardcoded ports or credentials
- API rate limiting for public deployments
- CORS configuration for UI integration
- Input validation for coordinates and parameters

### Integration Points
- **PostGIS**: Spatial analysis and route storage
- **Qdrant**: Semantic search for locations and routes
- **Traccar**: Real-time fleet position integration
- **N8n**: Workflow automation for dispatch and alerts
- **Eclipse Ditto**: Digital twin state synchronization

## Success Metrics

### Completion Targets
- P0: 100% required for v1.0
- P1: 50% target for v1.1
- P2: Stretch goals for v2.0

### Quality Standards
- Test coverage: >80% for P0 features
- Documentation: Complete API reference
- Performance: <500ms route calculation
- Reliability: 99% uptime for health checks

### Business Value
- Transit optimization: 15-20% efficiency gain
- Fleet coordination: 25% reduction in dead miles
- MaaS deployment: $35K-50K per installation
- Carbon tracking: Measurable emission reductions

## Implementation Notes

### Phase 1: Core Setup (Current)
- Docker Compose with OpenTripPlanner
- Sample GTFS/OSM data for Portland
- Basic CLI commands following v2.0 contract
- Health checks and smoke tests

### Phase 2: Integration (Next)
- PostGIS integration for spatial analysis
- N8n workflows for automation
- Real-time feed support

### Phase 3: Advanced Features (Future)
- Digital twin synchronization
- MaaS orchestration templates
- Benchmarking and analytics

## Risk Mitigation
- **Memory Usage**: JVM heap size configuration critical
- **Graph Build Time**: Large regions may timeout - implement async builds
- **Data Quality**: GTFS validation required before import
- **API Rate Limits**: Implement caching for repeated queries

## Dependencies
- Docker and Docker Compose
- Java 17+ runtime
- Sample GTFS/OSM data
- Optional: PostGIS, Redis, N8n

## References
- Official Site: https://www.opentripplanner.org
- GitHub: https://github.com/opentripplanner/OpenTripPlanner
- GTFS Spec: https://gtfs.org
- API Docs: https://docs.opentripplanner.org/api/

## Change History
- 2025-01-16: Initial PRD creation (0% complete)
- 2025-01-16: Core scaffolding complete (20% complete)
  - Implemented v2.0 contract structure
  - Created CLI with all required commands
  - Set up Docker configuration
  - Added test framework
  - Registered port allocation
- 2025-09-17: Major improvements (60% complete)
  - Fixed Docker startup command for OTP 2.x
  - Downloaded Portland GTFS and OSM data
  - Successfully built routing graph
  - Fixed health check endpoints
  - Verified all lifecycle commands
  - Updated smoke tests for proper validation