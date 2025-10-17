# Wiki.js Product Requirements Document (PRD)

## Executive Summary
**What**: Modern, Git-backed wiki platform for centralized documentation and knowledge management
**Why**: Scenarios need persistent, searchable, version-controlled documentation that agents can programmatically access and update
**Who**: All Vrooli scenarios, agents, and developers requiring documentation storage and retrieval
**Value**: Enables self-documenting scenarios, knowledge persistence, and collaborative documentation workflows
**Priority**: P1 - Critical infrastructure for scenario documentation and knowledge management

## Requirements Checklist

### P0 Requirements (Must Have)
- [ ] **Health Check**: Wiki.js responds to health checks within 5 seconds timeout
- [ ] **v2.0 Contract Compliance**: All required CLI commands and file structure per universal.yaml
- [ ] **Database Integration**: Connects to PostgreSQL and initializes schema
- [ ] **Port Registry**: Properly registers and uses port from port_registry.sh
- [ ] **Container Lifecycle**: Clean start/stop/restart with proper state management
- [ ] **Basic API Access**: GraphQL endpoint accessible for programmatic content management
- [ ] **Data Persistence**: Content and configuration survive container restarts

### P1 Requirements (Should Have)
- [ ] **Git Backend**: Synchronizes content with Git repository for version control
- [ ] **Search Functionality**: Full-text search across all wiki content
- [ ] **Authentication**: Basic auth setup for API access
- [ ] **Content Templates**: Pre-configured templates for scenario documentation

### P2 Requirements (Nice to Have)
- [ ] **Backup/Restore**: Automated backup and restore functionality
- [ ] **Bulk Import**: Ability to import multiple documents at once
- [ ] **API Key Management**: Secure API key generation and rotation

## Technical Specifications

### Architecture
- **Container**: requarks/wiki:2 Docker image
- **Database**: PostgreSQL for metadata and configuration
- **Storage**: Local volumes for content and data persistence
- **Network**: Host network mode for database connectivity
- **Port**: Dynamically allocated via port_registry.sh

### Dependencies
- **Required**: PostgreSQL (for metadata storage)
- **Optional**: Redis (for caching), Git (for version control)

### API Endpoints
- **GraphQL**: `/graphql` - Full CRUD operations on wiki content
- **REST**: `/api` - Alternative REST endpoints for simpler integrations
- **Health**: `/` - Health check endpoint

### Configuration
```yaml
Database:
  type: postgres
  host: localhost (via host network)
  port: from postgres resource
  name: wikijs
  user: wikijs
  
Storage:
  data: /var/lib/vrooli/resources/wikijs/data
  content: /var/lib/vrooli/resources/wikijs/content
  
Network:
  mode: host
  port: from port_registry.sh
```

## Success Metrics

### Completion Criteria
- **P0 Complete**: 0% (0/7 requirements)
- **P1 Complete**: 0% (0/4 requirements)
- **P2 Complete**: 0% (0/3 requirements)
- **Overall**: 0% complete

### Quality Metrics
- Startup time: < 20 seconds
- Health check response: < 1 second
- API response time: < 500ms
- Memory usage: < 512MB
- Test coverage: > 80%

### Performance Targets
- Concurrent users: 50+
- Page load time: < 2 seconds
- Search response: < 1 second
- Content sync: < 30 seconds

## Revenue Justification

### Direct Value
- **Documentation Automation**: $15K - Eliminates manual documentation overhead
- **Knowledge Retention**: $20K - Preserves institutional knowledge permanently
- **API Integration**: $10K - Enables programmatic documentation workflows

### Indirect Value
- **Scenario Self-Documentation**: Every scenario can auto-generate its own docs
- **Agent Knowledge Base**: Persistent memory for all agent interactions
- **Compliance Documentation**: Audit trails and change history

### Total Estimated Value: $45K+

## Implementation Notes

### Current Issues
- Missing v2.0 test phase structure (test/phases/*.sh files)
- Port not registered in port_registry.sh
- No defaults.sh or schema.json in config/
- Missing test/run-tests.sh orchestrator
- Health check implementation needs timeout handling

### Next Steps
1. Register port in port_registry.sh
2. Create v2.0 compliant test structure
3. Add missing config files (defaults.sh, schema.json)
4. Implement proper health check with timeout
5. Test complete lifecycle operations
6. Verify GraphQL API functionality

## Version History
- 2025-01-12: Initial PRD creation
- 2025-01-12: Assessment of current implementation status