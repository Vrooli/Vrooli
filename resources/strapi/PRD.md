# Product Requirements Document: Strapi Resource

## Executive Summary

**What**: Strapi v5 headless CMS providing structured content management with REST/GraphQL APIs  
**Why**: Enable scenarios to rapidly build content-driven applications without custom backend development  
**Who**: Scenarios needing content management, API-first development, or multi-channel content delivery  
**Value**: $35,000 - Eliminates 3-6 months of custom CMS development per scenario  
**Priority**: Medium - Essential for content-heavy applications and rapid prototyping

## Problem Statement

### Current Challenges
1. Scenarios building custom content management from scratch waste resources
2. No unified way to manage structured content across scenarios
3. Difficult to build content-driven APIs without significant backend work
4. Content versioning and multi-language support requires complex implementation
5. No visual content modeling tool for non-technical content creation

### Impact
- Development time: 3-6 months per custom CMS implementation
- Maintenance burden: Custom content systems require ongoing support
- Content silos: Each scenario manages content differently
- Limited scalability: Custom solutions struggle with complex content relationships

## Success Metrics

### Completion Metrics
- [ ] **P0 Requirements**: 50% complete (3/6)
- [ ] **P1 Requirements**: 50% complete (2/4)  
- [ ] **P2 Requirements**: 0% complete (0/3)
- [ ] **Documentation**: 40% complete
- [ ] **Test Coverage**: 25% complete

### Quality Metrics
- Health check response time: <500ms
- Startup time: <30 seconds
- API response time: <200ms for basic queries
- Memory usage: <512MB baseline
- Database connection pool: Properly managed

### Business Metrics
- Setup time reduction: From 3-6 months to 1 hour
- Content API generation: Instant vs weeks of development
- Multi-channel support: Built-in vs custom implementation
- Development cost savings: $35K per scenario

## Requirements

### P0 Requirements (Must Have)
- [x] **Health Check Endpoint**: Service responds to health checks at /health
- [x] **Basic Lifecycle**: Start/stop/restart commands work reliably
- [ ] **PostgreSQL Integration**: Connects to Vrooli's PostgreSQL instance (requires PostgreSQL running)
- [ ] **Admin Panel Access**: Web-based admin panel accessible for content management
- [ ] **REST API**: Content APIs automatically generated from content types
- [x] **Environment Configuration**: All settings via environment variables (no hardcoded values)

### P1 Requirements (Should Have)
- [ ] **GraphQL API**: GraphQL endpoint for flexible content queries
- [x] **S3 Storage Integration**: Connect to MinIO for media storage (storage.sh library implemented)
- [x] **Backup/Restore**: Content and configuration backup capabilities (backup.sh library implemented)
- [ ] **Multi-environment Support**: Dev/staging/production configurations

### P2 Requirements (Nice to Have)
- [ ] **Webhook Support**: Trigger external actions on content changes
- [ ] **Content Import/Export**: Bulk content operations
- [ ] **Custom Plugin Support**: Ability to extend with custom functionality

## Technical Specifications

### Architecture Overview
```
┌─────────────────┐     ┌──────────────┐     ┌──────────────┐
│   Scenarios     │────▶│  Strapi API  │────▶│  PostgreSQL  │
└─────────────────┘     └──────────────┘     └──────────────┘
                               │                      │
                               ▼                      ▼
                        ┌──────────────┐     ┌──────────────┐
                        │ Admin Panel  │     │    MinIO     │
                        └──────────────┘     └──────────────┘
```

### Technology Stack
- **Runtime**: Node.js 20 (LTS)
- **Framework**: Strapi v5.x
- **Database**: PostgreSQL 14+ (via Vrooli postgres resource)
- **Storage**: Local filesystem or MinIO (S3-compatible)
- **Process Manager**: PM2 for production stability
- **Port**: 1337 (default Strapi port)

### Integration Points
- **PostgreSQL**: Database connection via environment variables
- **MinIO**: Optional S3-compatible storage for media
- **Redis**: Optional caching layer for performance
- **N8n**: Webhook integration for workflow automation

### Configuration Schema
```json
{
  "database": {
    "client": "postgres",
    "connection": {
      "host": "${POSTGRES_HOST}",
      "port": "${POSTGRES_PORT}",
      "database": "${STRAPI_DATABASE_NAME}",
      "user": "${POSTGRES_USER}",
      "password": "${POSTGRES_PASSWORD}"
    }
  },
  "server": {
    "host": "${STRAPI_HOST:-0.0.0.0}",
    "port": "${STRAPI_PORT:-1337}",
    "admin": {
      "auth": {
        "secret": "${STRAPI_ADMIN_JWT_SECRET}"
      }
    }
  },
  "storage": {
    "provider": "${STORAGE_PROVIDER:-local}",
    "s3": {
      "endpoint": "${MINIO_ENDPOINT}",
      "accessKey": "${MINIO_ACCESS_KEY}",
      "secretKey": "${MINIO_SECRET_KEY}"
    }
  }
}
```

## Implementation Plan

### Phase 1: Core Setup (Week 1)
1. Resource scaffolding with v2.0 contract compliance
2. Docker container configuration
3. PostgreSQL database initialization
4. Basic health check implementation

### Phase 2: Integration (Week 2)
1. Environment variable configuration
2. Admin panel setup and access
3. REST API validation
4. MinIO storage integration

### Phase 3: Enhancement (Week 3)
1. GraphQL endpoint configuration
2. Backup/restore functionality
3. Performance optimization
4. Documentation and examples

## Security Considerations

### Required Security Measures
- **No hardcoded secrets**: All credentials from environment
- **JWT token rotation**: Admin tokens expire and rotate
- **CORS configuration**: Proper cross-origin settings
- **Rate limiting**: API request throttling
- **SQL injection prevention**: Parameterized queries only
- **File upload validation**: Type and size restrictions

### Access Control
- Admin panel requires authentication
- API endpoints configurable as public/private
- Role-based permissions system
- Content-type level access control

## Rollback Plan

### Failure Scenarios
1. **Database connection failure**: Fall back to SQLite for development
2. **Port conflict**: Use dynamic port allocation
3. **Memory issues**: Reduce cache size and connection pools
4. **Storage failure**: Fall back to local filesystem

### Recovery Steps
1. Stop service: `vrooli resource strapi manage stop`
2. Check logs: `vrooli resource strapi logs --tail 100`
3. Reset configuration: `vrooli resource strapi manage uninstall --keep-data`
4. Reinstall: `vrooli resource strapi manage install`

## Revenue Justification

### Direct Value ($35,000)
- **Development Cost Savings**: $25,000 (3 months @ $100/hour)
- **Maintenance Savings**: $5,000/year (ongoing support)
- **Time-to-Market**: $5,000 (3 months faster deployment)

### Indirect Value
- **Reusability**: Every scenario can leverage the same CMS
- **Standardization**: Consistent content management across platform
- **Scalability**: Handle enterprise content volumes
- **Integration**: Works with existing Vrooli resources

### Use Cases Enabled
1. **Documentation Platforms**: $10K value per implementation
2. **E-commerce Catalogs**: $15K value per implementation
3. **Multi-tenant SaaS**: $20K value per implementation
4. **Content Marketplaces**: $25K value per implementation
5. **Learning Management Systems**: $30K value per implementation

## Dependencies

### Required Resources
- **postgres**: Database storage (must be running)
- **redis** (optional): Caching layer for performance
- **minio** (optional): S3-compatible media storage

### External Dependencies
- Node.js 20 LTS
- PM2 process manager
- PostgreSQL client libraries

## Acceptance Criteria

### Functional Criteria
- [ ] Health endpoint responds within 500ms
- [ ] Admin panel loads and accepts login
- [ ] Can create and retrieve content via API
- [ ] Database persists across restarts
- [ ] Logs are properly captured

### Performance Criteria
- [ ] Starts within 30 seconds
- [ ] Handles 100 concurrent requests
- [ ] Memory usage under 512MB idle
- [ ] API response time under 200ms

### Documentation Criteria
- [ ] README with quickstart guide
- [ ] API endpoint documentation
- [ ] Configuration examples
- [ ] Troubleshooting guide

## Progress History

### 2025-01-15 - 50% → 65% (Storage and Backup features)
- ✅ Implemented S3/MinIO storage integration (lib/storage.sh)
- ✅ Added comprehensive backup/restore functionality (lib/backup.sh)
- ✅ Enhanced CLI with storage and backup commands
- ✅ Created Docker entrypoint script for automatic initialization
- ✅ Updated Docker configuration for better reliability
- ✅ Fixed package.json dependencies for Strapi v5.23.4
- 🔧 P1 requirements 50% complete (2/4 features implemented)

### 2025-01-14 - 0% → 50% (Major improvements)
- ✅ Fixed test script variable naming conflict
- ✅ Implemented full Docker support with docker-compose.yml
- ✅ Added automated admin user creation functionality
- ✅ Enhanced CLI with Docker and admin management commands
- ✅ Implemented health check endpoint support
- ✅ Added environment-based configuration (no hardcoded values)
- ✅ Created Docker management library (lib/docker.sh)
- 🔧 Lifecycle commands functional (install requires Node.js 20+)

### 2025-01-10
- Initial PRD creation
- Research completed, no conflicts found
- Port 1337 allocated for service

## Related Resources
- [Strapi v5 Documentation](https://docs.strapi.io/)
- [Strapi Docker Guide](https://docs.strapi.io/dev-docs/installation/docker)
- [PostgreSQL Integration](https://docs.strapi.io/dev-docs/configurations/database#postgresql)
- [S3 Storage Provider](https://market.strapi.io/providers/@strapi-provider-upload-aws-s3)

## Notes
- Strapi v5 is 100% JavaScript/TypeScript (no more legacy code)
- Supports both REST and GraphQL out of the box
- Plugin ecosystem enables extended functionality
- Can be used as headless CMS or full-stack with frontend