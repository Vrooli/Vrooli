# OpenEMS Energy Management Suite - Product Requirements Document

## Executive Summary
**What**: Open-source energy management system for distributed energy resources (DER) and microgrids  
**Why**: Powers energy & utilities scenarios with real-time control, optimization, and monitoring capabilities  
**Who**: Energy utilities, microgrid operators, solar installers, EV fleet managers, smart city planners  
**Value**: $45K per deployment - Replaces proprietary SCADA systems costing $100K-500K  
**Priority**: HIGH - Critical for energy/utilities vertical and cross-sector simulations

## Requirements Checklist

### P0 Requirements (Must Have)
- [x] **Containerized Backend**: Deploy OpenEMS backend with Edge/Backend components in Docker containers
- [x] **CLI Configuration**: Expose CLI commands for configuring energy systems, assets, and DER controllers
- [x] **Real-time Telemetry**: Wire data persistence into QuestDB and Redis for time-series telemetry
- [x] **DER Simulation Tests**: Build tests that simulate distributed energy resource telemetry
- [x] **API Validation**: Ensure REST/JSON-RPC APIs are reachable from automation workflows
- [x] **Health Monitoring**: Implement health check endpoints for both Edge and Backend components
- [x] **Basic Lifecycle**: Support start/stop/restart operations through CLI

### P1 Requirements (Should Have)
- [x] **n8n Workflow Integration**: Automated energy management workflows with visual automation flows
- [x] **Apache Superset Dashboards**: Energy analytics dashboard templates for real-time monitoring
- [x] **Eclipse Ditto Digital Twins**: Digital twin models for DER assets with co-simulation support
- [x] **Energy Forecast Models**: Solar generation, battery optimization, and consumption forecasting

### P2 Requirements (Nice to Have)
- [x] **Alert Automation**: Implement hooks for automated alerts through Pushover/Twilio
- [x] **Co-simulation**: Enable co-simulation with OpenTripPlanner and GeoNode for energy-aware mobility

## Technical Specifications

### Architecture
- **OpenEMS Edge**: Runs on edge devices, controls DER hardware
- **OpenEMS Backend**: Central management, monitoring, and analytics
- **Communication**: REST API, JSON-RPC, Modbus TCP/RTU
- **Storage**: QuestDB for time-series, Redis for real-time state

### Dependencies
- Docker for containerization
- QuestDB (port 9010) for time-series storage
- Redis (port 6379) for real-time caching
- Java 17+ runtime (embedded in container)

### API Endpoints
- REST API: http://localhost:8084/rest/*
- JSON-RPC: ws://localhost:8085/jsonrpc
- UI Console: http://localhost:8084/

### Integration Points
- Node-RED for workflow automation
- Superset for analytics dashboards
- SimPy/Blender for grid simulations
- GeoNode for spatial analysis

## Success Metrics

### Completion Targets
- P0: 100% (7/7 requirements)
- P1: 100% (4/4 requirements)  
- P2: 100% (2/2 requirements)
- Overall: 100% (13/13 requirements)

### Quality Metrics
- Health check response time < 1s
- API response time < 500ms
- Telemetry ingestion rate > 1000 points/sec
- Container startup time < 30s

### Performance Targets
- Support 100+ DER assets per Edge instance
- Handle 10,000+ telemetry points/minute
- Query historical data < 100ms
- Real-time state updates < 50ms latency

## Revenue Justification

### Market Size
- Global microgrid market: $47B by 2025
- Energy management systems: $62B market
- DER management software: $2.8B growing 15% annually

### Value Proposition
- Replaces $100K-500K proprietary SCADA systems
- Reduces energy costs by 15-30% through optimization
- Enables participation in grid flexibility markets
- Supports renewable energy integration mandates

### Deployment Value
- Small commercial: $10K (solar + battery)
- Medium industrial: $45K (microgrid control)
- Large utility: $150K (DER fleet management)
- Smart city: $300K (district energy systems)

## Implementation History
- 2025-01-16: Initial PRD creation - Generator phase started
- 2025-01-16: Completed P0 requirements (100%)
  - Implemented containerized deployment with Edge/Backend components
  - Added CLI configuration commands for DER assets
  - Integrated QuestDB and Redis for telemetry persistence
  - Created comprehensive DER simulation tests (solar, battery, EV, wind, microgrid)
  - Implemented API validation for REST/JSON-RPC/Modbus
  - Added health monitoring and lifecycle management
- 2025-01-16: Completed P1 requirements (100%)
  - Implemented n8n workflow integration for energy automation (solar optimization, peak shaving, SCADA ingestion)
  - Created Apache Superset dashboard templates (energy overview, solar analytics, battery management, grid interaction)
  - Added Eclipse Ditto digital twin integration with SimPy/Blender co-simulation support
  - Developed energy forecast models for solar generation, battery optimization, and consumption prediction
- 2025-01-16: Completed P2 requirements (100%)
  - Implemented alert automation with Pushover/Twilio integration for critical energy events
  - Added co-simulation capability with OpenTripPlanner and GeoNode for energy-aware mobility planning
  - Created SimPy-based simulation engine for EV charging optimization scenarios
  - Implemented comprehensive alert rules for grid outages, battery faults, and peak demand

## Research Findings
- **Similar Work**: zigbee2mqtt (IoT integration), questdb (time-series), n8n (workflow automation)
- **Template Selected**: v2.0 resource contract with containerized service pattern
- **Unique Value**: First comprehensive energy management platform in Vrooli
- **External References**: 
  - https://openems.io/docs/
  - https://github.com/OpenEMS/openems
  - https://www.openems.de/wp-content/uploads/2020/02/OpenEMS-Whitepaper.pdf
  - https://openems.github.io/openems.io/openems/latest/introduction.html
  - https://community.openems.io/
- **Security Notes**: Isolate Modbus communications, validate all DER commands, rate-limit telemetry