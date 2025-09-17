# OpenMRS Clinical Records Platform PRD

## Executive Summary
**What**: OpenMRS is an open-source electronic medical record platform with modular architecture for healthcare data management  
**Why**: Enables healthcare scenarios with EHR workflows, clinical decision support, and population health analytics  
**Who**: Healthcare providers, clinics, telemedicine platforms, public health agencies, and health system integrators  
**Value**: $30K+ per deployment - replaces proprietary EMR systems costing $100-500K annually  
**Priority**: High - Unlocks healthcare vertical with immediate revenue potential and ties into emergency response intelligence

## Requirements Checklist

### P0 Requirements (Must Have) ☐ 60%
- [x] **Docker Deployment**: Package OpenMRS reference application with Docker Compose and MySQL
- [x] **Demo Data**: Seed demo patients, providers, and clinical concepts via CLI command
- [x] **CLI Utilities**: Provide CLI commands for patient, encounter, and concept management
- [ ] **Authentication Integration**: Integrate with Keycloak for secure authentication
- [x] **MySQL Persistence**: Configure MySQL as primary database (PostgreSQL not supported)
- [x] **Health Checks**: Implement smoke tests for service health and connectivity
- [ ] **API Validation**: Create tests that verify REST API and data flow functionality

### P1 Requirements (Should Have) ☐ 0%
- [ ] **AI Integration**: Connectors to push clinical events to Haystack/Ollama workflows
- [ ] **Interoperability**: Document FHIR/HL7 patterns with n8n automation examples
- [ ] **Analytics Dashboards**: Integrate with Apache Superset for population health monitoring
- [ ] **Referral Automation**: Sample workflows for referrals and telemedicine scheduling

### P2 Requirements (Nice to Have) ☐ 0%
- [ ] **Patient Notifications**: Twilio/Pushover integration for follow-ups
- [ ] **Geospatial Analysis**: GeoNode integration for outbreak mapping

## Technical Specifications

### Architecture
```
┌──────────────────────────────────────────┐
│            OpenMRS Resource              │
├──────────────────────────────────────────┤
│  Web Layer                               │
│  - Patient Portal (Port 8005)            │
│  - Admin Interface                       │
├──────────────────────────────────────────┤
│  API Layer                               │
│  - REST API (Port 8006)                  │
│  - FHIR R4 API (Port 8007)              │
├──────────────────────────────────────────┤
│  Database (PostgreSQL)                   │
│  - Clinical Data (Port 5444)             │
│  - Demo Clinic Seed Data                 │
├──────────────────────────────────────────┤
│  CLI Interface                           │
│  - Patient CRUD Operations               │
│  - Encounter Management                  │
│  - FHIR Export/Import                    │
└──────────────────────────────────────────┘
```

### Dependencies
- Docker & Docker Compose (container orchestration)
- PostgreSQL 14+ (clinical database)
- OpenMRS 3.0+ Backend (reference application)
- jq (JSON processing for API responses)
- curl (API testing and health checks)

### API Endpoints

#### REST API (Port 8006)
- `GET /openmrs/ws/rest/v1/session` - Authentication session
- `GET /openmrs/ws/rest/v1/patient` - List patients
- `POST /openmrs/ws/rest/v1/patient` - Create patient
- `GET /openmrs/ws/rest/v1/encounter` - List encounters
- `POST /openmrs/ws/rest/v1/encounter` - Create encounter
- `GET /openmrs/ws/rest/v1/concept` - Clinical concepts
- `GET /openmrs/ws/rest/v1/provider` - Healthcare providers
- `GET /openmrs/ws/rest/v1/location` - Facility locations

#### FHIR API (Port 8007)
- `GET /openmrs/ws/fhir2/R4/Patient` - FHIR patient resources
- `GET /openmrs/ws/fhir2/R4/Encounter` - FHIR encounter resources
- `GET /openmrs/ws/fhir2/R4/Practitioner` - FHIR practitioner resources
- `GET /openmrs/ws/fhir2/R4/Observation` - Clinical observations
- `POST /openmrs/ws/fhir2/R4/$export` - Bulk data export

### Configuration
```bash
# Environment Variables
OPENMRS_PORT=8005
OPENMRS_API_PORT=8006
OPENMRS_FHIR_PORT=8007
OPENMRS_DB_PORT=5444
OPENMRS_DB_NAME=openmrs
OPENMRS_DB_USER=openmrs
OPENMRS_ADMIN_USER=admin
OPENMRS_ADMIN_PASS=Admin123
OPENMRS_ENABLE_FHIR=true
OPENMRS_ENABLE_DEMO_DATA=true
```

## Success Metrics

### Completion Targets
- **P0 Completion**: 60% (Docker deployment working, patient CRUD implemented, demo data seeding functional)
- **Test Coverage**: 100% smoke tests implemented
- **API Availability**: REST and FHIR endpoints defined
- **Documentation**: Complete for current implementation

### Performance Requirements
- Health check response time: <1 second
- API response time: <2 seconds for standard queries
- Startup time: 2-3 minutes (initial with demo data)
- Resource usage: 4GB RAM, 10GB disk space

### Quality Gates
- All smoke tests must pass
- Docker containers must start successfully
- Health endpoints must respond
- CLI commands must be available

## Implementation Status

### Completed
- ✅ Basic scaffolding and directory structure
- ✅ CLI interface with all required commands
- ✅ Docker Compose configuration with MySQL
- ✅ Port allocation in registry
- ✅ Test framework (smoke, integration, unit)
- ✅ Configuration schema and runtime.json
- ✅ Health check implementation
- ✅ Patient CRUD operations (create, list, get, update, delete)
- ✅ Demo data seeding via CLI command
- ✅ Encounter creation API
- ✅ Concept listing and search
- ✅ Provider listing

### In Progress
- 🔄 Initial setup automation
- 🔄 REST API authentication
- 🔄 Keycloak authentication integration

### Not Started
- ❌ FHIR module configuration
- ❌ Haystack/Ollama connectors
- ❌ Superset dashboard templates
- ❌ n8n workflow examples
- ❌ Notification integrations

## Revenue Justification

### Direct Value
- **EMR Replacement**: $100-500K annual licensing fees saved
- **Operational Efficiency**: 40% reduction in documentation time
- **Compliance**: HIPAA-ready configuration included
- **Interoperability**: FHIR standard enables health information exchange

### Ecosystem Value
- **Healthcare Vertical**: Unlocks medical scenarios and workflows
- **AI Enhancement**: Clinical data feeds ML models for decision support
- **Population Health**: Analytics drive public health insights
- **Emergency Response**: Real-time data for outbreak detection

### Market Opportunity
- 600,000+ medical practices in US alone
- $40B+ global EMR market
- Growing demand for open-source alternatives
- Telehealth expansion driving adoption

## Next Steps for Improvers

### Priority 1: Complete P0 Requirements
1. Implement demo data seeding with realistic patients/encounters
2. Add Keycloak authentication integration
3. Complete PostgreSQL backup/restore utilities
4. Enhance API client with full CRUD operations

### Priority 2: Enable Integrations
1. Configure FHIR module for data exchange
2. Create Haystack connectors for AI workflows
3. Build Superset dashboard templates
4. Document n8n automation patterns

### Priority 3: Production Readiness
1. Implement data encryption at rest
2. Add audit logging for HIPAA compliance
3. Create backup and disaster recovery procedures
4. Performance optimization for large datasets

## Risk Mitigation

### Technical Risks
- **Docker Image Availability**: Using official OpenMRS images
- **PostgreSQL Compatibility**: Tested with OpenMRS 2.4+
- **API Stability**: Using stable v1 REST endpoints

### Compliance Risks
- **HIPAA Requirements**: Encryption and audit logging planned
- **Data Privacy**: Configurable data retention policies
- **Access Control**: Keycloak provides enterprise-grade auth

## Progress History
- **2025-01-16**: Initial scaffolding created (20% P0 complete)
- **2025-09-16**: Fixed database to use MySQL, implemented patient CRUD, added demo data seeding (60% P0 complete)