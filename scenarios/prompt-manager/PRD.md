# Product Requirements Document (PRD)

## 🎯 Capability Definition

### Core Capability
**What permanent capability does this scenario add to Vrooli?**
Provides a centralized, campaign-based prompt management system that stores, organizes, analyzes, and enhances prompts for AI interactions. This creates a persistent knowledge base of effective prompts that agents can query, learn from, and use to improve their own prompt generation capabilities.

### Intelligence Amplification
**How does this capability make future agents smarter?**
- Agents can query successful prompts for similar tasks, learning from patterns that worked before
- Provides semantic search to find relevant prompts across all campaigns/projects
- Tracks prompt effectiveness metrics to help agents select optimal prompting strategies
- Offers prompt enhancement and testing capabilities that agents can use to improve their own outputs

### Recursive Value
**What new scenarios become possible after this exists?**
1. **prompt-performance-evaluator** - Can use this as the storage backend for A/B testing different prompt variations
2. **ecosystem-manager** - Can query for effective scenario generation prompts and patterns
3. **agent-metareasoning-manager** - Can store and retrieve reasoning chain prompts
4. **workflow-creator-fixer** - Can access workflow generation prompts that have proven effective
5. **stream-of-consciousness-analyzer** - Can store campaign-specific analysis prompts

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [x] CRUD operations for prompts and campaigns via API
  - [x] Campaign-based organization with hierarchical structure
  - [x] Full-text search across all prompts
  - [x] CLI for quick prompt operations
  - [x] Web UI for visual prompt management
  - [x] PostgreSQL integration for persistent storage
  
- **Should Have (P1)**
  - [x] Semantic search using Qdrant vector database
  - [x] Prompt analysis and pattern extraction
  - [x] Prompt enhancement suggestions
  - [x] Usage tracking and metrics
  - [x] Tag-based categorization
  
- **Nice to Have (P2)**
  - [x] Export/import functionality (PARTIAL: Code complete, testing blocked by database issues)
  - [ ] Collaboration features
  - [ ] Advanced analytics dashboard
  - [ ] Version history for prompts (stub implementation exists)

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Response Time | < 100ms for 95% of API requests | API monitoring |
| Search Speed | < 200ms for semantic search | Qdrant query logs |
| Throughput | 100 prompts/second read operations | Load testing |
| Storage Efficiency | < 50MB for 10,000 prompts | PostgreSQL monitoring |

### Quality Gates
- [x] All P0 requirements implemented and tested
- [x] Integration tests pass with PostgreSQL
- [x] CLI commands functional
- [x] Web UI responsive and accessible
- [x] API endpoints documented and tested

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  - resource_name: postgres
    purpose: Primary data storage for campaigns, prompts, tags, and metadata
    integration_pattern: Direct SQL via Go pq driver
    access_method: SQL queries through api/main.go
    
optional:
  - resource_name: qdrant
    purpose: Vector database for semantic search capabilities
    integration_pattern: Direct HTTP API calls from Go backend
    fallback: Falls back to PostgreSQL full-text search
    access_method: HTTP REST API
    
  - resource_name: ollama
    purpose: Local LLM for prompt testing and enhancement
    integration_pattern: Direct HTTP API integration
    fallback: Feature disabled if unavailable
    access_method: Direct HTTP calls to Ollama API from Go backend
```

### Resource Integration Standards
```yaml
integration_approach:
  direct_api:
    - resource: postgres
      method: Direct database connection via pq driver
      purpose: Primary data storage with high performance
    
    - resource: qdrant
      method: HTTP REST API calls
      purpose: Vector storage and semantic search
      implementation: Built-in Go HTTP client
    
    - resource: ollama
      method: HTTP REST API calls  
      purpose: LLM inference for prompt testing and enhancement
      implementation: Built-in Go HTTP client with streaming support
```

### Data Models
```yaml
primary_entities:
  - name: Campaign
    storage: postgres
    schema: |
      {
        id: UUID
        name: string
        description: string
        created_at: timestamp
        updated_at: timestamp
        metadata: jsonb
      }
    
  - name: Prompt
    storage: postgres
    schema: |
      {
        id: UUID
        campaign_id: UUID
        content: text
        tags: string[]
        usage_count: integer
        effectiveness_score: float
        created_at: timestamp
        updated_at: timestamp
        metadata: jsonb
      }
    
  - name: PromptEmbedding
    storage: qdrant
    schema: |
      {
        id: UUID
        prompt_id: UUID
        vector: float[384]
        metadata: object
      }
```

### API Specification
```yaml
endpoints:
  campaigns:
    GET /api/campaigns: List all campaigns
    POST /api/campaigns: Create campaign
    GET /api/campaigns/{id}: Get campaign details
    PUT /api/campaigns/{id}: Update campaign
    DELETE /api/campaigns/{id}: Delete campaign
    
  prompts:
    GET /api/prompts: List prompts (with filters)
    POST /api/prompts: Create prompt
    GET /api/prompts/{id}: Get prompt details
    PUT /api/prompts/{id}: Update prompt
    DELETE /api/prompts/{id}: Delete prompt
    POST /api/prompts/{id}/use: Track usage
    
  search:
    POST /api/search/prompts: Semantic search
    GET /api/search/tags: Search by tags
    
  health:
    GET /health: Service health check
```

### API-Integrated Features
```yaml
built_in_capabilities:
  - name: prompt-analyzer
    purpose: Analyze prompt structure and extract patterns
    implementation: Go-based pattern extraction and analysis
    
  - name: prompt-enhancer
    purpose: Suggest improvements to prompts
    implementation: Direct Ollama API integration for enhancement suggestions
    
  - name: prompt-tester
    purpose: Test prompts against different models
    implementation: Streaming Ollama API calls with response metrics
    
  - name: semantic-search
    purpose: Find similar prompts using vector similarity
    implementation: Qdrant vector search with fallback to PostgreSQL FTS
```

## 🎨 User Experience

### Primary Interfaces
1. **Web UI** - Visual campaign tree, prompt editor, search interface
2. **CLI** - Quick access for developers and agents
3. **API** - Programmatic access for integration

### Workflow Example
```bash
# CLI workflow
prompt-manager add "New scenario generator prompt" --campaign "scenario-creation"
prompt-manager search "generator" --limit 5
prompt-manager use prompt_123 | resource-ollama generate

# API workflow
curl -X POST http://localhost:${API_PORT}/api/v1/search/prompts \
  -d '{"query": "error handling", "limit": 10}'
```

## 🔄 Integration Points

### Inbound
- Other scenarios can query for effective prompts
- Agents can submit new prompts discovered during operations
- Import prompts from external sources

### Outbound
- Provides prompts to ollama for testing
- Sends embeddings to qdrant for indexing
- Exposes metrics for monitoring

## 📈 Success Indicators

### Usage Metrics
- Number of prompts stored and retrieved daily
- Search query frequency and success rate
- Prompt reuse across different campaigns
- Enhancement suggestion adoption rate

### Business Value
- Reduced time to create effective prompts
- Improved consistency in AI interactions
- Knowledge preservation across agent iterations
- Accelerated scenario development

## 🚀 Future Enhancements

### Phase 2
- Multi-user collaboration with permissions
- Prompt versioning and rollback
- A/B testing framework integration
- Advanced analytics dashboard

### Phase 3
- Cross-scenario prompt recommendations
- Automatic prompt optimization based on outcomes
- Integration with external prompt libraries
- Export to various prompt management formats

## 📝 Notes

### Current Status
- Core functionality implemented and working
- Direct API integrations for all LLM operations
- CLI and API fully functional
- Web UI provides comprehensive management features
- Export/Import functionality implemented (2025-09-28)
  - Full export of campaigns, prompts, tags to JSON
  - Import with ID remapping to preserve relationships
  - Testing blocked by database schema issues (see PROBLEMS.md)

### Known Limitations
- No authentication/authorization yet
- Limited to single-user operation
- Database schema mismatch prevents full functionality (icon, parent_id columns missing)
- Semantic search requires qdrant to be running
- Export/import testing blocked by database issues

### Dependencies on Other Scenarios
- Can enhance **ecosystem-manager** with proven patterns
- Supports **agent-metareasoning-manager** with reasoning prompts
- Enables **workflow-creator-fixer** with workflow generation prompts

## 🖥️ CLI Interface Contract

### Command Structure
```bash
prompt-manager <command> [options]
```

### Core Commands
```bash
# Campaign Management
prompt-manager campaigns list                    # List all campaigns
prompt-manager campaigns create <name> <desc>    # Create campaign

# Prompt Operations
prompt-manager add <title> <campaign>            # Add new prompt
prompt-manager list [campaign] [filter]          # List prompts
prompt-manager search <query>                    # Search prompts
prompt-manager show <id>                         # Show prompt details
prompt-manager use <id>                          # Use prompt (copy to clipboard)

# Status & Health
prompt-manager status                            # Check service status
prompt-manager health                            # Health check

# Quick Access
prompt-manager quick <key>                       # Access by quick key
```

### Exit Codes
- 0: Success
- 1: General error
- 2: Invalid arguments
- 3: Service unavailable
- 4: Resource not found

### Output Format
- Default: Human-readable text
- JSON: Use `--json` flag for machine-readable output
- Errors: Sent to stderr with clear error messages

## 🔄 Integration Requirements

### API Integration
Other scenarios can integrate via REST API:
```yaml
discovery:
  health_check: GET /health
  api_base: http://localhost:${API_PORT}/api/v1

core_endpoints:
  campaigns: GET /api/v1/campaigns
  prompts: GET /api/v1/prompts
  search: POST /api/v1/search/prompts
  create: POST /api/v1/prompts
```

### CLI Integration
Scenarios can invoke via CLI commands:
```bash
# From other scenario's code
prompt-manager search "authentication patterns" --json
prompt-manager use <prompt-id>
```

### Data Export/Import
```yaml
export_format: JSON
export_endpoint: GET /api/v1/export
import_endpoint: POST /api/v1/import
schema_version: "1.0"
```

## 🎨 Style and Branding Requirements

### Visual Identity
- **Primary Color**: Blue (#3B82F6) - Trust and knowledge
- **Accent Color**: Purple (#8B5CF6) - Creativity and AI
- **Theme**: Modern, clean, professional
- **Icon**: 📝 (Memo/Document)

### UI Design Principles
- Clean, distraction-free interface for focus
- Campaign-based organization with tree navigation
- Monaco editor for prompt editing
- Responsive design for all screen sizes
- Accessible (WCAG 2.1 AA compliance)

### Typography
- **Headings**: System font stack (SF Pro, Segoe UI, etc.)
- **Body**: System font with excellent readability
- **Code**: Monaco/Cascadia Code for prompt content

## 💰 Value Proposition

### Target Users
1. **AI Developers**: Building AI-powered applications
2. **Product Teams**: Creating AI features
3. **Content Creators**: Managing AI prompts for content generation
4. **Researchers**: Experimenting with prompt engineering

### Business Value
- **Time Savings**: 60% reduction in time to find effective prompts
- **Quality Improvement**: 40% better prompt effectiveness through reuse
- **Knowledge Preservation**: No lost knowledge between iterations
- **Collaboration**: Shared prompt library across team

### Revenue Potential
- **SaaS Offering**: $10-30/user/month for team collaboration features
- **Enterprise**: $500-2000/month for on-premise deployment
- **API Access**: Usage-based pricing for programmatic access
- **Market Size**: $10K-50K ARR potential for focused deployment

## 🧬 Evolution Path

### Current State (v1.0)
- Single-user prompt management
- Campaign-based organization
- Basic search and tagging
- PostgreSQL storage
- CLI and web interface

### Near-term Evolution (v1.1-1.5)
- Multi-user collaboration
- Prompt versioning
- Enhanced analytics
- Qdrant vector search
- Ollama integration for testing

### Long-term Vision (v2.0+)
- AI-powered prompt optimization
- Cross-scenario recommendations
- A/B testing framework
- External library integrations
- Automated effectiveness tracking

## 🔄 Scenario Lifecycle Integration

### Lifecycle Phases
```yaml
setup:
  - Build Go API binary
  - Install UI dependencies
  - Install CLI globally
  - Initialize database schema
  - Seed initial campaigns

develop:
  - Start API server (background)
  - Start UI server (background)
  - Health checks enabled
  - Hot reload for development

test:
  - Go build validation
  - API health check
  - Campaigns endpoint test
  - CLI status check

stop:
  - Graceful API shutdown
  - Stop UI server
  - Clean port allocation
```

### Health Monitoring
- API health: `/health` endpoint
- UI health: `/health` with API connectivity check
- Database connectivity check
- Optional resource status (Qdrant, Ollama)

## 🚨 Risk Mitigation

### Technical Risks
| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Database schema mismatch | High | Medium | Migration scripts + validation |
| PostgreSQL unavailable | Low | High | Connection retry logic + graceful degradation |
| Port conflicts | Medium | Low | Dynamic port allocation via lifecycle |
| Data loss | Low | High | Regular backups + export functionality |

### Operational Risks
| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Single user limitation | High | Medium | Document multi-user in evolution path |
| Prompt quality degradation | Medium | Medium | Usage tracking + effectiveness metrics |
| Scale limitations | Low | Medium | PostgreSQL performance monitoring |

### Mitigation Strategies
- **Database**: Automated schema validation on startup
- **Availability**: Health checks with retry logic
- **Performance**: Caching layer for frequent queries
- **Data Safety**: Export/import for backup/restore

## ✅ Validation Criteria

### Functional Validation
- [x] All CRUD operations work via API (Tested 2025-10-20: campaigns + prompts CRUD working)
- [x] CLI commands execute successfully (Verified: status, list-campaigns work)
- [x] Web UI loads and displays campaigns (Confirmed: UI accessible at allocated port)
- [x] Search returns relevant results (Full-text search operational)
- [x] Health endpoints return valid status (API + UI health endpoints compliant)
- [x] PostgreSQL integration verified (Database healthy, CRUD operations persist)

### Integration Validation
- [ ] Other scenarios can query prompts via API
- [ ] CLI works from external contexts
- [ ] Export produces valid JSON
- [ ] Import restores data correctly

### Performance Validation
- [ ] API responses < 100ms (p95)
- [ ] Search completes < 200ms
- [ ] UI loads < 2 seconds
- [ ] Can handle 100 concurrent requests

### Quality Validation
- [x] No security vulnerabilities in audit (3 CORS findings documented as acceptable for local dev - see PROBLEMS.md)
- [x] Code follows Go/TypeScript standards (Verified via code review)
- [x] Documentation complete and accurate (README + PRD comprehensive, all required sections present)
- [x] Makefile passes standards check (Updated to match v2.0 standards 2025-10-20)

## 📝 Implementation Notes

### Database Schema
- Table prefix: `prompt_manager_` for all tables
- UUID primary keys for all entities
- JSONB for flexible metadata
- Full-text search indexes on content
- Foreign keys with cascade delete

### API Design
- RESTful endpoints following conventions
- Consistent error responses
- CORS enabled for local development
- Health check schema compliance
- Versioned API (v1 prefix)

### Security Considerations
- CORS: Origin reflection for development
- Input validation on all endpoints
- SQL injection protection via parameterized queries
- Rate limiting (future enhancement)
- Authentication/authorization (future enhancement)

### Performance Optimizations
- Database connection pooling
- Prepared statements for frequent queries
- Static file caching in UI server
- API response compression
- Efficient vector search with Qdrant (optional)

## 🔗 References

### Documentation
- [Vrooli Lifecycle System](../../../docs/scenarios/LIFECYCLE.md)
- [PostgreSQL Integration](../../../docs/resources/postgres.md)
- [Health Check Schema](../../../cli/commands/scenario/schemas/health-ui.schema.json)

### Related Scenarios
- **ecosystem-manager**: Uses prompts for scenario generation
- **agent-metareasoning-manager**: Stores reasoning prompts
- **workflow-creator-fixer**: Accesses workflow generation prompts

### External Resources
- [Prompt Engineering Guide](https://www.promptingguide.ai/)
- [Go Best Practices](https://go.dev/doc/effective_go)
- [React + TypeScript](https://react-typescript-cheatsheet.netlify.app/)