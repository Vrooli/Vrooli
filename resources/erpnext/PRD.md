# ERPNext Resource - Product Requirements Document

## Executive Summary
**What**: Complete open-source Enterprise Resource Planning (ERP) system  
**Why**: Enable businesses to manage operations (accounting, inventory, HR, CRM, projects) in a unified platform  
**Who**: SMBs and enterprises needing comprehensive business automation  
**Value**: $15,000-50,000 in annual operational efficiency gains per deployment  
**Priority**: High - Core business operations platform

## Requirements

### P0 Requirements (Must Have)
- [x] **Health Check Endpoint**: Respond to health checks within 1 second on port 8020 ✅ 2025-01-12
- [x] **v2.0 Contract Compliance**: Full implementation of universal contract (cli.sh, lib/core.sh, test structure) ✅ 2025-01-12
- [ ] **Core ERP Modules**: Accounting, Inventory, HR, CRM, Project Management functional
- [ ] **API Access**: REST API available at http://localhost:8020/api
- [ ] **Authentication**: Working login/logout with session management
- [x] **Database Integration**: PostgreSQL connection for data persistence ✅ 2025-01-12
- [x] **Redis Integration**: Redis for caching and queue management ✅ 2025-01-12

### P1 Requirements (Should Have)
- [ ] **Workflow Engine**: Custom business process automation
- [ ] **Reporting Module**: Built-in analytics and custom report builder
- [ ] **Multi-tenant Support**: Multiple companies/organizations
- [ ] **Content Management**: Add/list/get/remove custom apps and DocTypes

### P2 Requirements (Nice to Have)
- [ ] **Mobile Responsive UI**: Fully responsive design for all devices
- [ ] **E-commerce Module**: Online store capabilities
- [ ] **Manufacturing Module**: Production planning and control

## Technical Specifications

### Architecture
- **Service Type**: Multi-container Docker application
- **Primary Container**: ERPNext application server
- **Supporting Containers**: MariaDB, Redis, Nginx
- **Port**: 8020 (main application)
- **Dependencies**: postgres, redis (external resources)

### API Endpoints
- `GET /health` - Health check endpoint
- `POST /api/method/login` - Authentication
- `GET /api/resource/:doctype` - Get records
- `POST /api/resource/:doctype` - Create records
- `PUT /api/resource/:doctype/:name` - Update records
- `DELETE /api/resource/:doctype/:name` - Delete records

### Configuration
```yaml
startup_order: 580
dependencies: ["postgres", "redis"]
startup_timeout: 90
startup_time_estimate: "30-60s"
recovery_attempts: 2
priority: medium
```

## Success Metrics

### Completion Targets
- **P0 Completion**: 57% (4/7 requirements)
- **P1 Completion**: 0% (0/4 requirements)  
- **P2 Completion**: 0% (0/3 requirements)
- **Overall**: 29% (4/14 requirements)

### Quality Metrics
- Health check response time: <1s required
- API response time: <500ms for basic operations
- Startup time: <60s to healthy state
- Test coverage: >80% for core functionality

### Performance Targets
- Concurrent users: 50+ simultaneous
- Transaction throughput: 100+ per second
- Database size: Handle 10GB+ datasets
- Memory usage: <2GB under normal load

## Development Progress

### History
- 2025-01-12: Initial PRD created, beginning v2.0 compliance work
- 2025-01-12: 0% → 29% - Implemented v2.0 contract compliance, health checks, test structure, verified Redis/PostgreSQL integration

### Next Steps
1. Test and fix API endpoint availability
2. Verify core ERP modules are accessible
3. Implement authentication testing
4. Add content management functionality
5. Document API usage and integration points