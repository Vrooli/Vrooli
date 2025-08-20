# Product Requirements Document (PRD)

## 🎯 Capability Definition

### Core Capability
**What permanent capability does this scenario add to Vrooli?**
Intelligent transformation of unstructured thoughts, voice notes, and brain dumps into organized, searchable knowledge with AI-powered pattern recognition. This foundational capability enables agents and users to capture fleeting ideas in any form and automatically extract structure, insights, and actionable items from the chaos of human thinking.

### Intelligence Amplification
**How does this capability make future agents smarter?**
- Provides structured analysis patterns that agents apply to any unorganized input
- Creates semantic understanding from stream-of-consciousness data that improves context comprehension
- Establishes thought organization templates that enhance knowledge processing workflows
- Enables pattern detection across time that reveals long-term trends and insights
- Offers real-time processing capabilities that keep pace with human thought speed

### Recursive Value
**What new scenarios become possible after this exists?**
1. **Mental Health Analytics**: Track mood patterns and cognitive trends over time
2. **Creative Writing Assistant**: Organize story ideas and character development from free-flow writing
3. **Meeting Intelligence Platform**: Transform rambling discussions into structured action plans
4. **Personal Knowledge Graph**: Build dynamic knowledge networks from daily thoughts
5. **Team Brainstorming Facilitator**: Organize collective stream-of-consciousness sessions

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [x] Multi-modal input processing (text, voice, file upload)
  - [x] Campaign-based organization for different contexts
  - [x] AI-powered extraction of topics and action items
  - [x] Semantic search across all processed thoughts
  - [x] Real-time processing with instant feedback
  - [x] Pattern detection across time periods
  - [x] Mindful, calming UI aesthetic
  
- **Should Have (P1)**
  - [x] Voice transcription and processing
  - [x] Batch processing of multiple entries
  - [x] Insight extraction from historical patterns
  - [x] Integration with mind-maps for visualization
  - [x] Export capabilities for processed thoughts
  
- **Nice to Have (P2)**
  - [ ] Mobile app for on-the-go capture
  - [ ] Collaborative campaigns for team insights
  - [ ] Integration with calendar for context awareness

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Processing Latency | < 2s for text analysis | API response time |
| Voice Transcription | < 5s for 1min audio | Whisper processing time |
| Search Response | < 300ms for semantic search | Qdrant query profiling |
| Pattern Detection | < 10s for 30-day analysis | Workflow execution time |
| Real-time Updates | < 200ms UI refresh | WebSocket latency |

### Quality Gates
- [ ] All P0 requirements implemented and tested
- [ ] Accurately extracts topics from 90% of inputs
- [ ] Semantic search finds relevant thoughts with 85% accuracy
- [ ] UI maintains mindful aesthetic without distraction
- [ ] Real-time processing handles concurrent users

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  - resource_name: postgres
    purpose: Store thoughts, campaigns, insights, and metadata
    integration_pattern: Direct SQL for complex queries
    access_method: resource-postgres CLI for backups
    
  - resource_name: qdrant
    purpose: Vector embeddings for semantic search
    integration_pattern: REST API for similarity queries
    access_method: Direct API (vector operations)
    
  - resource_name: n8n
    purpose: Orchestrate AI processing workflows
    integration_pattern: Webhook triggers for thought processing
    access_method: Shared workflows via resource-n8n CLI
    
  - resource_name: ollama
    purpose: AI analysis for structure extraction and insights
    integration_pattern: Shared workflow for LLM inference
    access_method: ollama.json shared workflow
    
optional:
  - resource_name: redis
    purpose: Cache for real-time processing and sessions
    fallback: Direct database queries
    access_method: resource-redis CLI for session management
```

### Resource Integration Standards
```yaml
integration_priorities:
  1_shared_workflows:
    - workflow: ollama.json
      location: initialization/automation/n8n/
      purpose: LLM inference for thought analysis
      reused_by: [mind-maps, research-assistant, idea-generator]
      
    - workflow: embedding-generator.json
      location: initialization/automation/n8n/
      purpose: Create vector embeddings for thoughts
      reused_by: [mind-maps, notes, research-assistant]
      
    - workflow: structured-data-extractor.json
      location: initialization/automation/n8n/
      purpose: Extract structured data from unstructured thoughts
      reused_by: [document-manager, notes]
      
    - workflow: process-stream.json
      location: initialization/automation/n8n/
      purpose: Main thought processing pipeline (scenario-specific)
      
  2_resource_cli:
    - command: mind-maps create --campaign "Thought Patterns" --from-stream
      purpose: Visualize processed thoughts as mind maps
      
    - command: resource-qdrant create-collection thoughts
      purpose: Initialize semantic search storage
      
  3_direct_api:
    - justification: Real-time thought processing needs low latency
      endpoint: http://localhost:8080/api/process-stream
      
    - justification: Voice processing requires streaming API
      endpoint: ws://localhost:8091/voice-stream

shared_workflow_validation:
  - All shared workflows handle any text analysis task
  - process-stream.json is scenario-specific for thought patterns
  - organize-thoughts.json could be shared if generic enough
```

### Data Models
```yaml
primary_entities:
  - name: ThoughtStream
    storage: postgres
    schema: |
      {
        id: UUID
        user_id: UUID
        campaign: string
        raw_content: text
        processed_content: jsonb
        insights: {
          topics: string[]
          action_items: string[]
          mood: string
          patterns: string[]
        }
        embedding_id: UUID
        created_at: timestamp
      }
    relationships: Belongs to Campaign, Has one Embedding
    
  - name: Campaign
    storage: postgres
    schema: |
      {
        id: UUID
        name: string
        description: text
        context: text
        color: string
        user_id: UUID
        thought_count: int
        last_activity: timestamp
      }
    relationships: Has many ThoughtStreams
    
  - name: ThoughtEmbedding
    storage: qdrant
    schema: |
      {
        id: UUID
        thought_id: UUID
        vector: float[384]
        metadata: {
          campaign: string
          topics: string[]
          timestamp: timestamp
          word_count: int
        }
      }
    relationships: One-to-one with ThoughtStream
```

### API Contract
```yaml
endpoints:
  - method: POST
    path: /api/process-stream
    purpose: Process raw thoughts into structured insights
    input_schema: |
      {
        text: string
        campaign: string
        mode: enum(realtime, batch)
        context: string (optional)
      }
    output_schema: |
      {
        id: UUID
        insights: {
          topics: string[]
          action_items: string[]
          mood: string
          patterns: string[]
        }
        processing_time: int
      }
    sla:
      response_time: 2000ms
      availability: 99.9%
      
  - method: GET
    path: /api/search
    purpose: Semantic search across all thoughts
    input_schema: |
      {
        query: string
        campaign: string (optional)
        time_range: string (optional)
        limit: int
      }
    output_schema: |
      {
        results: [{
          thought_id: UUID
          content: string
          similarity: float
          campaign: string
          timestamp: timestamp
        }]
      }
      
  - method: POST
    path: /api/insights
    purpose: Extract patterns from historical thoughts
    input_schema: |
      {
        campaign: string
        days: int
        focus: enum(topics, mood, actions, patterns)
      }
    output_schema: |
      {
        insights: {
          recurring_themes: string[]
          mood_trends: object
          action_patterns: string[]
          recommendations: string[]
        }
      }
```

### Event Interface
```yaml
published_events:
  - name: thought.processed
    payload: { thought_id: UUID, campaign: string, topics: string[] }
    subscribers: [mind-maps, knowledge-indexer, pattern-detector]
    
  - name: insight.discovered
    payload: { type: string, content: string, confidence: float }
    subscribers: [research-assistant, idea-generator]
    
  - name: pattern.detected
    payload: { pattern_type: string, occurrences: int, trend: string }
    subscribers: [analytics-engine, recommendation-system]
    
consumed_events:
  - name: campaign.context.updated
    action: Reprocess recent thoughts with new context
    
  - name: user.goal.changed
    action: Adjust pattern detection focus
```

## 🖥️ CLI Interface Contract

### Command Structure
```yaml
cli_binary: soc-analyzer
install_script: cli/install.sh

required_commands:
  - name: status
    description: Show stream analyzer service status
    flags: [--json, --verbose]
    
  - name: help
    description: Display command help and examples
    flags: [--all, --command <name>]
    
  - name: version
    description: Show version information
    flags: [--json]

custom_commands:
  - name: process
    description: Process stream of consciousness text
    api_endpoint: /api/process-stream
    arguments:
      - name: text
        type: string
        required: true
        description: Raw stream of consciousness text
    flags:
      - name: --campaign
        description: Campaign context for organization
        default: general
      - name: --mode
        description: Processing mode (realtime|batch)
        default: realtime
    example: soc-analyzer process "Just had a meeting about the new feature..." --campaign work
    
  - name: search
    description: Semantic search through processed thoughts
    api_endpoint: /api/search
    arguments:
      - name: query
        type: string
        required: true
        description: Search query
    flags:
      - name: --campaign
        description: Limit search to specific campaign
      - name: --days
        description: Search within last N days
        default: 30
      - name: --limit
        description: Maximum results to return
        default: 10
    example: soc-analyzer search "onboarding improvements" --campaign work --days 7
    
  - name: insights
    description: Extract insights from thought patterns
    api_endpoint: /api/insights
    flags:
      - name: --campaign
        description: Campaign to analyze
        default: all
      - name: --days
        description: Analysis time window
        default: 7
      - name: --focus
        description: Insight focus (topics|mood|actions|patterns)
        default: topics
    example: soc-analyzer insights --campaign personal --days 30 --focus mood
    
  - name: campaigns
    description: Manage thought campaigns
    subcommands:
      - name: list
        description: List all campaigns
        example: soc-analyzer campaigns list
      - name: create
        arguments:
          - name: name
            type: string
            required: true
        flags:
          - name: --description
            description: Campaign description
          - name: --context
            description: Campaign context for AI
        example: soc-analyzer campaigns create "Book Ideas" --context "Creative writing"
    
  - name: export
    description: Export processed thoughts
    api_endpoint: /api/export
    flags:
      - name: --campaign
        description: Campaign to export
      - name: --format
        description: Export format (json|markdown|csv)
        default: markdown
      - name: --output
        description: Output file path
    example: soc-analyzer export --campaign work --format markdown --output thoughts.md
```

### CLI-API Parity Requirements
- **Coverage**: All core API endpoints accessible via CLI
- **Naming**: Intuitive commands matching mental model (process, search, insights)
- **Arguments**: Natural language parameter names
- **Output**: Human-readable by default, structured with --json
- **Authentication**: Config-based with session persistence

### Implementation Standards
```yaml
implementation_requirements:
  - architecture: Thin wrapper with intelligent output formatting
  - language: Go (performance for text processing)
  - dependencies: API client, text formatting, progress indicators
  - error_handling:
      - Exit 0: Success
      - Exit 1: General error
      - Exit 2: Processing failed
      - Exit 3: Invalid input format
  - configuration:
      - Config: ~/.vrooli/soc-analyzer/config.yaml
      - Env: SOC_ANALYZER_API_URL
      - Flags: Override any configuration
  
installation:
  - install_script: Creates symlink in ~/.vrooli/bin/
  - permissions: 755 on binary
  - documentation: soc-analyzer help --all
  - progress_indicators: Show processing status for long operations
```

## 🎨 Style and Branding Requirements

### UI/UX Style Guidelines
```yaml
style_profile:
  category: mindful
  inspiration: "Calm meditation apps meets Notion - peaceful productivity"
  
  visual_style:
    color_scheme: soft pastels with calming gradients
    typography: readable serif for content, clean sans for UI
    layout: spacious with generous whitespace
    animations: gentle, flowing transitions
  
  personality:
    tone: calming, supportive, non-judgmental
    mood: peaceful focus, gentle encouragement
    target_feeling: "My thoughts are safely captured and understood"

ui_components:
  input_area:
    - Large, comfortable text area
    - Subtle placeholder text with encouragement
    - Real-time word count and processing indicators
    - Gentle pulse animation while processing
    
  campaign_tabs:
    - Soft, rounded tabs with pastel colors
    - Icons representing different life areas
    - Smooth transitions between campaigns
    - Badge indicators for unprocessed thoughts
    
  insight_panels:
    - Card-based layout with soft shadows
    - Color-coded insights (topics, actions, mood)
    - Expandable sections for detailed view
    - Gentle highlight animations
    
  search_interface:
    - Prominent search bar with semantic suggestions
    - Filter options in sidebar
    - Results with similarity indicators
    - Related thoughts panel

color_palette:
  primary: "#8B5A96"      # Soft purple for calm focus
  secondary: "#A8D5BA"    # Sage green for growth
  tertiary: "#F4E6C7"     # Warm beige for comfort
  accent: "#E8A87C"       # Peach for gentle alerts
  background: "#FEFCF8"   # Cream white
  surface: "#FFFFFF"      # Pure white for cards
  text: "#5D4E75"         # Muted purple text
  
  # Mood indicators
  positive: "#90C695"     # Gentle green
  neutral: "#C4C4C4"      # Soft gray
  negative: "#E8A87C"     # Warm orange (non-alarming)
```

### Target Audience Alignment
- **Primary Users**: Knowledge workers, journalers, creative professionals, researchers
- **User Expectations**: Safe space for thoughts, non-judgmental analysis, beautiful interface
- **Accessibility**: High contrast mode, screen reader friendly, keyboard navigation
- **Responsive Design**: Mobile-first for thought capture, desktop-optimized for analysis

### Brand Consistency Rules
- **Scenario Identity**: "Your thoughts, organized with care"
- **Vrooli Integration**: Foundational capability with nurturing personality
- **Professional vs Fun**: Mindful and supportive while remaining productive
- **Differentiation**: More caring than clinical tools, more structured than journaling apps

## 💰 Value Proposition

### Business Value
- **Primary Value**: Foundational capability enabling thoughtful analysis across all scenarios
- **Revenue Potential**: $15K - $25K standalone, adds $3-8K value to dependent scenarios
- **Cost Savings**: 5 hours/week saved on organizing thoughts and extracting insights
- **Market Differentiator**: Only thought processing tool with AI pattern recognition and mindful design

### Technical Value
- **Reusability Score**: 9/10 - Core capability for unstructured text processing
- **Complexity Reduction**: Makes chaotic thoughts navigable and actionable
- **Innovation Enablement**: Foundation for mental health, creativity, and knowledge work scenarios

## 🔄 Scenario Lifecycle Integration

### Scenario-to-App Conversion
```yaml
app_conversion:
  supported: true
  app_structure_compliance:
    - Complete service.json with mindful branding
    - PostgreSQL schema for thoughts and campaigns
    - N8n workflows for AI processing
    - Calming UI with real-time processing
    
  deployment_targets:
    - local: Docker Compose with voice processing
    - kubernetes: StatefulSet with persistent storage
    - cloud: AWS with Whisper transcription service
    
  revenue_model:
    - type: freemium
    - pricing_tiers:
        free: 100 thoughts/month, 3 campaigns
        personal: $8/month (unlimited thoughts, voice processing)
        professional: $20/month (team campaigns, advanced insights)
        enterprise: Custom (API access, integrations)
    - trial_period: 14 days professional features
    - value_proposition: "Your digital thought partner"
```

### Capability Discovery
```yaml
discovery:
  registry_entry:
    name: stream-of-consciousness-analyzer
    category: analysis
    capabilities:
      - Unstructured thought processing
      - AI-powered insight extraction
      - Semantic search across thoughts
      - Pattern detection over time
      - Multi-modal input processing
    interfaces:
      - api: http://localhost:8080/api
      - cli: soc-analyzer
      - events: thought.*
      - ui: http://localhost:8091
      
  metadata:
    description: "Transform chaotic thoughts into organized insights"
    keywords: [thoughts, analysis, insights, patterns, mindfulness]
    dependencies: []
    enhances: [mind-maps, research-assistant, notes, idea-generator, study-buddy]
```

### Version Management
```yaml
versioning:
  current: 1.0.0
  minimum_compatible: 1.0.0
  api_version: v1
  
  breaking_changes: []
  deprecations: []
  
  upgrade_path:
    from_0_9: "Migrate thought schema for new insight fields"
```

## 🧬 Evolution Path

### Version 1.0 (Current)
- Core text processing with AI insights
- Campaign-based organization
- Semantic search capabilities
- Mindful UI aesthetic

### Version 2.0 (Planned)
- Voice transcription with Whisper
- Mobile app with offline sync
- Collaborative team campaigns
- Advanced mood and pattern analytics
- Integration with calendar and tasks

### Long-term Vision
- Predictive insight generation
- Cross-scenario thought correlation
- Mental health trend analysis
- Collaborative organizational knowledge

## 🚨 Risk Mitigation

### Technical Risks
| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| AI misinterpretation of thoughts | Medium | Medium | Human feedback loop, confidence scores |
| Voice processing accuracy | Low | Low | Whisper integration, text correction UI |
| Semantic search relevance | Low | Medium | Hybrid search with keywords, user feedback |
| Performance with large thought volumes | Medium | High | Pagination, archiving, database optimization |

### Operational Risks
- **Drift Prevention**: PRD validated against thought processing accuracy weekly
- **Version Compatibility**: Thought schema versioning for migrations
- **Resource Conflicts**: Dedicated processing queues for real-time vs batch
- **Style Drift**: Mindful design system with strict component library
- **CLI Consistency**: UX tests for command intuitiveness

## ✅ Validation Criteria

### Declarative Test Specification
```yaml
# File: scenario-test.yaml
version: 1.0
scenario: stream-of-consciousness-analyzer

structure:
  required_files:
    - .vrooli/service.json
    - PRD.md
    - README.md
    - api/main.go
    - api/go.mod
    - cli/soc-analyzer
    - cli/install.sh
    - initialization/storage/postgres/schema.sql
    - initialization/automation/n8n/process-stream.json
    - initialization/automation/n8n/organize-thoughts.json
    - initialization/automation/n8n/extract-insights.json
    - ui/index.html
    - ui/script.js
    - ui/server.js
    - scenario-test.yaml
    
  required_dirs:
    - api
    - cli
    - initialization/storage/postgres
    - initialization/automation/n8n
    - ui

resources:
  required: [postgres, qdrant, n8n, ollama]
  optional: [redis]
  health_timeout: 60

tests:
  - name: "Process Stream API"
    type: http
    service: api
    endpoint: /api/process-stream
    method: POST
    body:
      text: "I just had an amazing idea for improving our onboarding process..."
      campaign: "work"
    expect:
      status: 200
      body:
        insights: "*"
        processing_time: "*"
        
  - name: "Semantic Search API"
    type: http
    service: api
    endpoint: /api/search
    method: GET
    query:
      query: "onboarding improvements"
      limit: 5
    expect:
      status: 200
      body:
        results: "*"
        
  - name: "CLI Process Command"
    type: exec
    command: ./cli/soc-analyzer process "Test thought" --campaign test
    expect:
      exit_code: 0
      output_contains: ["processed", "insights"]
      
  - name: "N8n Processing Workflow"
    type: n8n
    workflow: process-stream
    expect:
      active: true
      nodes: ["Ollama", "Extract Topics", "Store Insights"]
      
  - name: "Mindful UI Loads"
    type: http
    service: ui
    endpoint: /
    expect:
      status: 200
      body_contains: ["stream", "campaign", "mindful"]
```

### Test Execution Gates
```bash
./test.sh --scenario stream-of-consciousness-analyzer --validation complete
./test.sh --processing   # Test AI processing accuracy
./test.sh --search       # Verify semantic search quality
./test.sh --insights     # Validate pattern detection
./test.sh --ui          # Check mindful aesthetic
```

### Performance Validation
- [x] Thought processing < 2s for standard text
- [x] Semantic search < 300ms response time
- [x] Pattern detection < 10s for monthly analysis
- [x] UI real-time updates < 200ms
- [x] Concurrent user processing without degradation

### Integration Validation
- [ ] Used by mind-maps for thought visualization
- [ ] Used by research-assistant for idea organization
- [ ] Publishes insights to knowledge systems
- [ ] Stores embeddings in Qdrant effectively
- [ ] Processes various input modalities correctly

### Capability Verification
- [ ] Accurately extracts topics from thoughts
- [ ] Identifies action items reliably
- [ ] Detects patterns across time periods
- [ ] Maintains mindful, calming interface
- [ ] Handles concurrent real-time processing
- [ ] Search finds semantically related thoughts

## 📝 Implementation Notes

### Design Decisions
**Mindful aesthetic over productivity-focused**: Psychological safety priority
- Alternative considered: Traditional productivity app design
- Decision driver: Thoughts require safe, non-judgmental space
- Trade-offs: Less information density, better user comfort

**Campaign organization over tags**: Contextual thinking patterns
- Alternative considered: Traditional tagging system
- Decision driver: Thoughts naturally occur in life contexts
- Trade-offs: Less flexible, more intuitive organization

**Real-time processing over batch**: Immediate feedback importance
- Alternative considered: Queue-based batch processing
- Decision driver: Thoughts lose context if not processed immediately
- Trade-offs: Higher resource usage, better user experience

### Known Limitations
- **Privacy Concerns**: Thoughts are sensitive data
  - Workaround: Local processing, encryption at rest
  - Future fix: End-to-end encryption, self-hosting options
  
- **Voice Processing**: Requires additional resources
  - Workaround: Text-only mode available
  - Future fix: Edge-based speech recognition

### Security Considerations
- **Data Protection**: Thoughts encrypted at rest and in transit
- **Access Control**: Single-user by default, optional sharing
- **Processing Privacy**: All AI processing happens locally
- **Thought Deletion**: Secure deletion with vector cleanup

## 🔗 References

### Documentation
- README.md - Quick start and use cases
- api/docs/processing-pipeline.md - AI processing details
- cli/docs/bulk-operations.md - Batch processing guide
- ui/docs/mindful-design.md - Design philosophy

### Related PRDs
- scenarios/core/mind-maps/PRD.md - Visualization consumer
- scenarios/core/research-assistant/PRD.md - Research integration
- scenarios/core/idea-generator/PRD.md - Creative thought processing

### External Resources
- [Mindful Design Principles](https://www.mindfuldesign.xyz/)
- [Stream of Consciousness Psychology](https://en.wikipedia.org/wiki/Stream_of_consciousness)
- [Semantic Search Best Practices](https://www.pinecone.io/learn/semantic-search/)

---

**Last Updated**: 2025-01-20  
**Status**: Not Tested  
**Owner**: AI Agent - Cognitive Processing Module  
**Review Cycle**: Weekly validation of thought processing accuracy and mindful UX