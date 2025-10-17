# Shared Initialization Resources

This directory contains initialization data that can be shared across multiple scenarios, enabling reuse of common workflows, schemas, and configurations.

## Structure

```
initialization/
├── automation/
│   ├── n8n/                # Shared n8n workflows
│   │   ├── embedding-generator.json         # Text to vector embeddings
│   │   ├── structured-data-extractor.json   # Extract structured data from text
│   │   ├── universal-rag-pipeline.json      # Complete RAG document processing
│   │   ├── README.md                        # Workflow documentation
│   │   └── test-structured-extractor.sh     # Test utilities
│   └── windmill/           # Shared Windmill apps (future)
├── storage/
│   ├── postgres/           # Shared database schemas (future)
│   └── qdrant/            # Shared vector collections (future)
└── configuration/         # Shared configs and templates (future)
```

## Usage in Scenarios

To use shared resources in a scenario, reference them in your `.vrooli/service.json`:

```json
{
  "resources": {
    "automation": {
      "n8n": {
        "type": "n8n",
        "enabled": true,
        "initialization": [
          {
            "file": "shared:initialization/automation/n8n/embedding-generator.json",
            "type": "workflow",
            "description": "Generate embeddings for vector storage"
          },
          {
            "file": "shared:initialization/automation/n8n/structured-data-extractor.json",
            "type": "workflow",
            "description": "Extract structured data from text"
          },
          {
            "file": "shared:initialization/automation/n8n/universal-rag-pipeline.json",
            "type": "workflow",
            "description": "Complete RAG document processing pipeline"
          },
          {
            "file": "initialization/automation/n8n/scenario-specific-workflow.json", 
            "type": "workflow",
            "description": "Your scenario's custom workflow"
          }
        ]
      }
    }
  }
}
```

## Shared Resource Conventions

### File Naming
- Prefix shared resources with `shared-` 
- Use descriptive names: `embedding-generator.json`
- Version if needed: `embedding-generator-v2.json`

### Resource IDs
- Use globally unique IDs to prevent conflicts
- Include "shared" in the ID: `embedding-generator`

### Documentation
- Each shared resource should include inline documentation
- Add usage examples and expected input/output formats

## Benefits

1. **Reduced Duplication**: Common patterns implemented once
2. **Consistency**: Standardized implementations across scenarios
3. **Maintainability**: Update shared resources in one place
4. **Compound Intelligence**: Solutions become permanent capabilities

## Merge Behavior

When a scenario runs directly:
1. Shared resources are accessible only if referenced in service.json
2. Scenario-specific resources take precedence over shared ones
3. Resources are accessed in-place, not copied - enabling live updates
4. Both `shared:` prefix and direct path references are supported

## Contributing Shared Resources

When creating a shared resource:
1. Ensure it's truly reusable across multiple scenarios
2. Use generic, configurable implementations
3. Add comprehensive documentation
4. Test with multiple scenarios
5. Consider backward compatibility

## 🚀 Planned Shared Workflows

These workflows are planned for future implementation to provide common functionality across scenarios:

### Priority 1 - Core Infrastructure

#### **Notification Orchestra** (`notification-orchestra.json`)
Multi-channel notification system with user preferences, templates, batching, and delivery tracking.
- **Channels**: Email, Slack, SMS, Discord, Webhooks, In-app
- **Features**: User preferences, scheduling, retry logic, deduplication, analytics
- **Used by**: All production scenarios

#### **Secret Rotation Manager** (`secret-rotation-manager.json`)
Automated credential and API key rotation across services.
- **Features**: Expiry detection, automatic rotation via Vault, service updates, audit trail
- **Used by**: Secure apps, production deployments

### Priority 2 - Data & Analytics

#### **Time-Series Aggregator** (`timeseries-aggregator.json`)
Collect, aggregate, and analyze metrics from multiple sources.
- **Features**: Data collection, aggregation, anomaly detection, QuestDB storage
- **Used by**: Monitoring apps, analytics dashboards

#### **Scheduled Report Generator** (`scheduled-report-generator.json`)
Generate and distribute periodic reports with customizable templates.
- **Features**: Multi-source data, template engine, PDF/Excel export, distribution
- **Used by**: Business apps, analytics platforms

#### **Web Research Aggregator** (`web-research-aggregator.json`)
Search, scrape, and synthesize information from multiple web sources.
- **Features**: SearXNG integration, Browserless scraping, deduplication, synthesis
- **Used by**: Research apps, content generators

### Priority 3 - Developer Tools

#### **Code Analysis Pipeline** (`code-analysis-pipeline.json`)
Analyze code repositories for quality, security, and documentation.
- **Features**: Judge0 execution, security scanning, dependency analysis, test coverage
- **Used by**: Developer tools, CI/CD pipelines

#### **Agent Task Orchestrator** (`agent-task-orchestrator.json`)
Coordinate multi-agent workflows with task distribution and result aggregation.
- **Features**: Task decomposition, agent assignment, progress monitoring, learning
- **Used by**: Complex automation scenarios

### Priority 4 - User Experience

#### **Conversation Memory Manager** (`conversation-memory.json`)
Manage chat history, context windows, and conversational state.
- **Features**: History storage, context management, summarization, entity extraction
- **Used by**: Chat-based apps, AI assistants

#### **Media Processing Pipeline** (`media-processing-pipeline.json`)
Process images, audio, and video through various AI services.
- **Features**: ComfyUI generation, Whisper transcription, OCR, storage in MinIO
- **Used by**: Content creation apps, media managers

### Future Expansion Ideas

#### **API Gateway** (`api-gateway.json`)
Unified API management with rate limiting, authentication, and usage tracking.

#### **Data Validation Pipeline** (`data-validation-pipeline.json`)
Validate and clean data against schemas with error reporting.

#### **Backup & Recovery Orchestrator** (`backup-recovery.json`)
Automated backup scheduling and disaster recovery workflows.

#### **Compliance Auditor** (`compliance-auditor.json`)
Track and report on regulatory compliance (GDPR, HIPAA, etc.).

#### **Performance Monitor** (`performance-monitor.json`)
Real-time performance tracking with alerting and auto-scaling.

#### **Content Moderation Pipeline** (`content-moderation.json`)
AI-powered content moderation for user-generated content.

#### **Payment Processing Orchestra** (`payment-orchestra.json`)
Handle payments, subscriptions, and billing across providers.

#### **Event Stream Processor** (`event-stream-processor.json`)
Process and react to real-time event streams from multiple sources.

## Implementation Guidelines

When implementing these shared workflows:

1. **Use Dynamic Variables**: Replace hardcoded values with `${service.resource.url}` patterns
2. **Include Error Handling**: Comprehensive try-catch blocks and fallback logic
3. **Support Both Sync/Async**: Webhook responses and background processing
4. **Document Schemas**: Clear input/output specifications
5. **Version Control**: Semantic versioning for breaking changes
6. **Test Coverage**: Include test scripts and sample data
7. **Performance Metrics**: Track execution time and resource usage
8. **Enable Composition**: Design for chaining with other workflows