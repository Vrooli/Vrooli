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
  - [ ] Export/import functionality
  - [ ] Collaboration features
  - [ ] Advanced analytics dashboard
  - [ ] Version history for prompts

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
    integration_pattern: HTTP API calls
    fallback: Falls back to PostgreSQL full-text search
    access_method: HTTP API from Go backend
    
  - resource_name: ollama
    purpose: Local LLM for prompt testing and enhancement
    integration_pattern: Shared workflow via n8n
    fallback: Feature disabled if unavailable
    access_method: Via shared ollama.json workflow
```

### Resource Integration Standards
```yaml
integration_priorities:
  1_shared_workflows:
    - workflow: ollama.json
      location: initialization/n8n/
      purpose: Prompt testing and enhancement
    - workflow: structured-data-extractor.json
      location: initialization/n8n/
      purpose: Extract patterns and entities from prompts
    - workflow: embedding-generator.json
      location: initialization/n8n/
      purpose: Generate embeddings for semantic search
  
  2_resource_cli:
    - command: resource-postgres query
      purpose: Database operations when needed from workflows
  
  3_direct_api:
    - justification: Go backend needs direct DB access for performance
      endpoint: PostgreSQL connection via pq driver
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

### Workflow Integration
```yaml
n8n_workflows:
  - name: prompt-analyzer
    purpose: Analyze prompt structure and extract patterns
    uses_shared: [structured-data-extractor.json]
    
  - name: prompt-enhancer
    purpose: Suggest improvements to prompts
    uses_shared: [ollama.json]
    
  - name: prompt-tester
    purpose: Test prompts against different models
    uses_shared: [ollama.json]
    
  - name: campaign-manager
    purpose: Automate campaign operations
    uses_shared: []
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
curl -X POST http://localhost:8085/api/search/prompts \
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
- Using shared workflows for LLM operations
- CLI and API fully functional
- Web UI provides basic management features

### Known Limitations
- No authentication/authorization yet
- Limited to single-user operation
- No backup/restore functionality
- Semantic search requires qdrant to be running

### Dependencies on Other Scenarios
- Can enhance **ecosystem-manager** with proven patterns
- Supports **agent-metareasoning-manager** with reasoning prompts
- Enables **workflow-creator-fixer** with workflow generation prompts