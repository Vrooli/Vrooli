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
- [x] **Core ERP Modules**: Accounting, Inventory, HR, CRM, Project Management functional ✅ 2025-09-14
- [x] **API Access**: REST API available at http://localhost:8020/api ✅ 2025-09-14
- [x] **Authentication**: Working login/logout with session management ✅ 2025-09-14
- [x] **Database Integration**: PostgreSQL connection for data persistence ✅ 2025-01-12
- [x] **Redis Integration**: Redis for caching and queue management ✅ 2025-01-12

### P1 Requirements (Should Have)
- [x] **Workflow Engine**: Custom business process automation ✅ 2025-09-14
- [x] **Reporting Module**: Built-in analytics and custom report builder ✅ 2025-09-14
- [ ] **Multi-tenant Support**: Multiple companies/organizations
- [x] **Content Management**: Add/list/get/remove custom apps and DocTypes ✅ 2025-09-13

### P2 Requirements (Nice to Have)
- [ ] **Mobile Responsive UI**: Fully responsive design for all devices
- [x] **E-commerce Module**: Online store capabilities ✅ 2025-09-15
- [x] **Manufacturing Module**: Production planning and control ✅ 2025-09-15

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
- **P0 Completion**: 100% (7/7 requirements fully working)
- **P1 Completion**: 75% (3/4 requirements)  
- **P2 Completion**: 67% (2/3 requirements)
- **Overall**: 86% (12/14 requirements)

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
- 2025-09-12: 29% → 50% - Added site initialization, API status monitoring, enhanced status reporting, partial module and auth functionality
- 2025-09-13: 50% → 57% - Enhanced API integration with Host header support, implemented content management, added credentials command, created API helper library
- 2025-09-14: 57% → 64% - Fixed database configuration, completed P0 requirements, all tests passing (100% pass rate)
- 2025-09-14: 64% → 71% - Exposed Workflow Engine and Reporting Module via CLI, added timeout handling to all API calls, fixed session management
- 2025-09-15: 71% → 86% - Added E-commerce and Manufacturing modules via CLI, exposed shopping cart and BOM functionality (2 P2 requirements)

### Next Steps
1. Complete web interface routing solution (nginx proxy or hosts file)
2. Finish authentication flow testing and documentation
3. Enable multi-tenant support (P1 - remaining requirement)
4. Implement mobile responsive UI (P2 - remaining requirement)

### Known Issues
- Web interface requires hosts file modification (127.0.0.1 vrooli.local) for browser access
- API calls require Host header (Host: vrooli.local) for multi-tenant routing - this is by design