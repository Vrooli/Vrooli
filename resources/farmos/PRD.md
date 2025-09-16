# farmOS Resource - Product Requirements Document

## Executive Summary
**What**: farmOS agricultural management platform resource for Vrooli ecosystem  
**Why**: Enable scenarios to orchestrate farm operations, IoT sensor integration, and supply chain management  
**Who**: Agricultural scenarios, IoT workflows, precision agriculture applications  
**Value**: Anchors the Agriculture/Food Systems branch with $30K+ revenue potential  
**Priority**: High - Critical for agricultural and food system scenarios

## Requirements

### P0 Requirements (Must Have) ✅ (100% Complete)
- [x] **Deploy farmOS Stack**: Official farmOS (Drupal + modules) deployment with Docker
- [x] **Demo Data Seeding**: Automatic seeding with demo fields, livestock, and equipment data
- [x] **Field Management CLI**: Commands for creating and managing farm fields
- [x] **Activity Logging CLI**: Commands for logging farm activities and operations
- [x] **Record Export CLI**: Commands for exporting farm records in various formats
- [x] **API Authentication**: OAuth2 and basic auth support for API access
- [x] **CRUD Operations**: Full Create, Read, Update, Delete via API for crops and assets
- [x] **Health Monitoring**: Proper health checks and status reporting

### P1 Requirements (Should Have) ☐
- [ ] **IoT Sensor Integration**: MQTT/Node-RED connectors for sensor data ingestion
- [ ] **QuestDB Integration**: Push farm metrics to QuestDB for time-series analytics
- [ ] **Redis Integration**: Real-time data caching and alerting workflows
- [ ] **Open Data Cube Integration**: Documentation for precision agriculture patterns

### P2 Requirements (Nice to Have) ☐
- [ ] **Financial Dashboards**: Integration with Mifos microfinance workflows
- [ ] **Geospatial Overlays**: Visualization of farmOS data in mapping scenarios
- [ ] **Multi-Farm Templates**: Cooperative resource sharing across farms

## Technical Specifications

### Architecture
- **Core Stack**: farmOS v3.x (Drupal 10 + farmOS modules)
- **Database**: PostgreSQL 14 with PostGIS extensions
- **Caching**: Built-in Drupal caching (Redis optional)
- **API**: REST API v2.x with OAuth2 authentication
- **Port**: 8004 (configurable)

### Dependencies
- Docker and Docker Compose
- PostgreSQL 14
- No other Vrooli resources required (foundational service)

### API Endpoints
- `/api/field` - Field management
- `/api/asset` - Asset management (equipment, livestock)
- `/api/log` - Activity and observation logs
- `/api/taxonomy_term` - Crop types, activity types
- `/oauth/token` - OAuth2 authentication
- `/export` - Data export endpoint

### Integration Points
- **QuestDB**: Time-series sensor data storage
- **Node-RED**: IoT workflow automation
- **Eclipse Ditto**: Digital twins for farm equipment
- **PostgreSQL/PostGIS**: Spatial queries for field mapping
- **Mifos**: Agricultural finance integration

## Success Metrics

### Completion Criteria
- [x] **P0 Completion**: 100% (8 requirements completed)
- [x] **Test Coverage**: Smoke tests pass, integration and unit tests functional
- [x] **Documentation**: Complete PRD, README, and API implementation
- [x] **Demo Validation**: Demo farm with sample data loads successfully

### Quality Metrics
- Health endpoint responds in <1 second
- API authentication successful
- CRUD operations functional
- Docker containers stable
- No critical errors in logs

### Performance Targets
- Startup time: 60-90 seconds
- API response time: <500ms
- Export time: <5 seconds for 1000 records
- Memory usage: <2GB under normal load

## Business Value

### Revenue Potential
- **Direct Deployment**: $10K per farm installation
- **SaaS Model**: $500/month per farm subscription
- **Cooperative Platform**: $50K for multi-farm deployments
- **Total Addressable Market**: 500+ potential farm clients

### Use Cases
1. **Precision Agriculture**: Sensor-driven crop management
2. **Livestock Tracking**: Health and location monitoring
3. **Supply Chain**: Farm-to-table traceability
4. **Compliance Reporting**: Regulatory documentation
5. **Financial Integration**: Loan and insurance workflows

### Scenario Integration
- **smart-agriculture**: Core dependency for farm automation
- **supply-chain-tracker**: Source data for food traceability
- **iot-orchestrator**: Sensor data aggregation point
- **financial-dashboard**: Agricultural metrics and reports

## Implementation Plan

### Phase 1: Core Setup ✅ (Complete)
- Docker Compose configuration
- Basic CLI structure
- v2.0 contract compliance
- Configuration management

### Phase 2: API Integration ✅ (Complete)
- Basic authentication implementation
- CRUD operations via API for fields, assets, and logs
- Automatic demo data seeding on first startup
- Export functionality for farm records

### Phase 3: IoT Integration (Future)
- MQTT broker connection
- Node-RED workflow templates
- QuestDB data pipeline
- Real-time dashboards

### Phase 4: Advanced Features (Future)
- Multi-farm support
- Financial integrations
- Geospatial analytics
- Mobile app support

## Known Issues & Limitations
- Initial installation requires 2-3 minutes on first startup
- IoT integration requires Node-RED resource (P1 requirement)
- Multi-farm support not yet implemented (P2 requirement)
- Some API responses require authentication even for public data

## Security Considerations
- Database passwords from environment variables
- OAuth2 for API authentication
- CORS configuration for cross-origin requests
- No hardcoded credentials
- Docker network isolation

## Testing Strategy
- **Smoke Tests**: Health checks, container status
- **Integration Tests**: API auth, CRUD operations, data persistence
- **Unit Tests**: Library functions, configuration validation
- **Performance Tests**: Load testing with simulated farm data

## Documentation
- README.md with quick start guide
- API documentation with examples
- Integration patterns for IoT sensors
- Troubleshooting guide for common issues

## Progress History
- 2025-01-16: Initial scaffolding (25% complete)
  - Created Docker Compose setup
  - Implemented CLI structure
  - Added test framework
  - Basic health monitoring

- 2025-09-16: Full P0 Implementation (100% complete)
  - Implemented Basic Auth for API access
  - Created comprehensive API library for CRUD operations
  - Added automatic installation completion on first startup
  - Implemented demo data seeding with drush
  - Fixed health checks and smoke tests
  - Integrated all API functions into CLI commands
  - All P0 requirements verified and functional

## Next Steps
1. Implement IoT sensor integration with MQTT (P1)
2. Add QuestDB integration for time-series data (P1)
3. Create Redis caching layer (P1)
4. Add Open Data Cube integration documentation (P1)
5. Implement multi-farm support (P2)