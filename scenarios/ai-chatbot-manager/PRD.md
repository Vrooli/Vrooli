# Product Requirements Document (PRD)

## 🎯 Capability Definition

### Core Capability
**What permanent capability does this scenario add to Vrooli?**
AI Chatbot Manager adds the ability to create, deploy, and manage intelligent chatbots that serve as 24/7 sales assistants and support representatives. This creates a complete SaaS chatbot platform capability within Vrooli, enabling businesses to integrate AI-powered conversational interfaces directly into their websites with zero external dependencies.

### Intelligence Amplification  
**How does this capability make future agents smarter?**
This capability creates a reusable conversational AI infrastructure that other scenarios can leverage. Future agents gain access to:
- Proven conversation management patterns and context handling
- Lead qualification and customer interaction workflows
- Real-time analytics and performance optimization insights  
- Multi-tenant chatbot deployment patterns for scaling customer interactions
- Embeddable widget architecture that can be adapted for other interactive tools

### Recursive Value
**What new scenarios become possible after this exists?**
- **sales-funnel-optimizer**: Automatically optimizes chatbot conversation flows based on conversion data
- **customer-insights-analyzer**: Deep analysis of conversation patterns to understand customer needs
- **omnichannel-support-hub**: Extends chatbot capability to SMS, email, social media platforms
- **ai-training-data-curator**: Uses conversation data to improve AI model performance across Vrooli
- **lead-scoring-engine**: Advanced lead qualification using conversational AI insights

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [ ] Create and configure multiple chatbots with custom personalities and knowledge bases
  - [ ] Generate embeddable JavaScript widgets for website integration
  - [ ] Real-time WebSocket-powered conversations with Ollama AI models  
  - [ ] Conversation storage and retrieval with full chat history
  - [ ] Lead capture and contact information collection
  - [ ] Basic analytics dashboard showing conversation volume and engagement
  
- **Should Have (P1)**
  - [ ] Multi-tenant architecture supporting different client configurations
  - [ ] A/B testing different chatbot personalities and conversation flows
  - [ ] Integration with external CRMs and lead management systems
  - [ ] Advanced analytics with conversion tracking and user journey mapping
  - [ ] Automated escalation to human agents when AI confidence is low
  - [ ] Custom branding and styling for chat widgets
  
- **Nice to Have (P2)**
  - [ ] Voice chat integration with speech-to-text and text-to-speech
  - [ ] Multi-language support with automatic language detection
  - [ ] Sentiment analysis and conversation quality scoring
  - [ ] API marketplace for third-party integrations
  - [ ] White-label deployment options for resellers

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Response Time | < 500ms for 95% of chat messages | WebSocket latency monitoring |
| Throughput | 1000 concurrent conversations | Load testing with simulated users |
| Widget Load Time | < 2s initial widget render | Browser performance API |
| Accuracy | > 85% appropriate responses | Manual conversation review |
| Uptime | > 99.5% availability | Health check monitoring |
| Resource Usage | < 2GB memory, < 50% CPU under load | System monitoring |

### Quality Gates
- [ ] All P0 requirements implemented and tested
- [ ] Integration tests pass with Ollama and PostgreSQL
- [ ] Performance targets met under simulated load
- [ ] Documentation complete (README, API docs, widget integration guide)
- [ ] Scenario can be invoked by other agents via API/CLI
- [ ] Widget successfully embeds in test websites

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  - resource_name: ollama
    purpose: AI conversation engine for natural language processing
    integration_pattern: Direct API calls via HTTP
    access_method: HTTP POST to localhost:11434/api/chat
    
  - resource_name: postgres
    purpose: Store chatbot configs, conversations, user data, analytics
    integration_pattern: Direct SQL via Go database/sql
    access_method: PostgreSQL connection pool
    
optional:
  - resource_name: redis  
    purpose: Session management and real-time conversation caching
    fallback: In-memory session storage with periodic persistence
    access_method: Redis client library
```

### Resource Integration Standards
```yaml
# Priority order for resource access:
integration_priorities:
  1_resource_cli:        # FIRST: Use resource CLI when available
    - command: resource-ollama chat
      purpose: Structured AI conversation interface
  
  2_direct_api:          # SECOND: Direct API for real-time requirements
    - justification: WebSocket requires direct HTTP for low-latency responses
      endpoint: http://localhost:11434/api/chat
      purpose: Real-time conversation processing
```

### Data Models
```yaml
primary_entities:
  - name: Chatbot
    storage: postgres
    schema: |
      {
        id: UUID,
        name: string,
        description: string,
        personality: string,
        knowledge_base: text,
        model_config: jsonb,
        widget_config: jsonb,
        created_at: timestamp,
        updated_at: timestamp
      }
    relationships: [One-to-many conversations]
    
  - name: Conversation
    storage: postgres
    schema: |
      {
        id: UUID,
        chatbot_id: UUID,
        session_id: string,
        user_ip: string,
        user_agent: string,
        started_at: timestamp,
        ended_at: timestamp,
        lead_captured: boolean,
        lead_data: jsonb
      }
    relationships: [One-to-many messages, belongs-to chatbot]
    
  - name: Message
    storage: postgres
    schema: |
      {
        id: UUID,
        conversation_id: UUID,
        role: enum(user, assistant),
        content: text,
        metadata: jsonb,
        timestamp: timestamp
      }
    relationships: [Belongs-to conversation]
    
  - name: Analytics
    storage: postgres
    schema: |
      {
        id: UUID,
        chatbot_id: UUID,
        date: date,
        total_conversations: integer,
        total_messages: integer,
        leads_captured: integer,
        avg_conversation_length: float,
        engagement_score: float
      }
    relationships: [Belongs-to chatbot]
```

### API Contract
```yaml
endpoints:
  - method: POST
    path: /api/v1/chatbots
    purpose: Create new chatbot configuration
    input_schema: |
      {
        name: string,
        personality: string,
        knowledge_base: string,
        model_config: object
      }
    output_schema: |
      {
        id: string,
        widget_embed_code: string,
        api_endpoints: object
      }
    sla:
      response_time: 200ms
      availability: 99.5%
      
  - method: POST  
    path: /api/v1/chat/{chatbot_id}
    purpose: Send message to chatbot and receive AI response
    input_schema: |
      {
        message: string,
        session_id: string,
        context: object
      }
    output_schema: |
      {
        response: string,
        confidence: float,
        should_escalate: boolean,
        lead_qualification: object
      }
    sla:
      response_time: 500ms
      availability: 99.9%
      
  - method: GET
    path: /api/v1/analytics/{chatbot_id}
    purpose: Retrieve conversation analytics and performance metrics
    input_schema: |
      {
        start_date: string,
        end_date: string,
        metrics: array
      }
    output_schema: |
      {
        conversations: integer,
        leads: integer,
        engagement_score: float,
        top_intents: array,
        conversion_rate: float
      }
    sla:
      response_time: 1000ms
      availability: 99%
```

### Event Interface
```yaml
published_events:
  - name: chatbot.conversation.started
    payload: |
      {
        chatbot_id: string,
        conversation_id: string,
        user_context: object
      }
    subscribers: [analytics-engine, lead-scoring-system]
    
  - name: chatbot.lead.captured
    payload: |
      {
        chatbot_id: string,
        conversation_id: string, 
        lead_data: object,
        qualification_score: float
      }
    subscribers: [crm-integrator, email-outreach-manager]
    
consumed_events:
  - name: ollama.model.updated
    action: Refresh chatbot model configurations and restart conversations
```

## 🖥️ CLI Interface Contract

### Command Structure
```yaml
cli_binary: ai-chatbot-manager
install_script: cli/install.sh

required_commands:
  - name: status
    description: Show chatbot service status and active conversations
    flags: [--json, --verbose]
    
  - name: help
    description: Display command help and usage
    flags: [--all, --command <name>]
    
  - name: version
    description: Show CLI and API version information
    flags: [--json]

custom_commands:
  - name: create
    description: Create new chatbot configuration
    api_endpoint: /api/v1/chatbots
    arguments:
      - name: name
        type: string
        required: true
        description: Chatbot display name
    flags:
      - name: --personality
        description: Chatbot personality description
    output: Chatbot ID and embed code
    
  - name: list
    description: List all configured chatbots
    api_endpoint: /api/v1/chatbots
    flags:
      - name: --active-only
        description: Show only active chatbots
    output: Table of chatbot configurations
    
  - name: chat
    description: Test chat with a specific chatbot
    api_endpoint: /api/v1/chat/{chatbot_id}
    arguments:
      - name: chatbot_id
        type: string
        required: true
        description: Target chatbot ID
    output: Interactive chat session
    
  - name: analytics
    description: Show conversation analytics
    api_endpoint: /api/v1/analytics/{chatbot_id}
    arguments:
      - name: chatbot_id
        type: string
        required: true
        description: Target chatbot ID
    flags:
      - name: --days
        description: Number of days to analyze
    output: Analytics summary
```

## 🔄 Integration Requirements

### Upstream Dependencies
**What capabilities must exist before this can function?**
- **Ollama Resource**: Provides local LLM inference for conversation processing
- **PostgreSQL Resource**: Database storage for persistent data management

### Downstream Enablement
**What future capabilities does this unlock?**
- **Conversational Analytics Platform**: Deep conversation mining and business intelligence
- **Multi-channel Support Hub**: Extend chatbot capability to SMS, email, social platforms
- **Sales Process Automation**: Automated lead nurturing and qualification workflows

### Cross-Scenario Interactions
```yaml
provides_to:
  - scenario: lead-scoring-engine
    capability: Rich conversation data and user interaction patterns
    interface: API
    
  - scenario: customer-insights-analyzer  
    capability: Natural language conversation analysis
    interface: Event stream
    
consumes_from:
  - scenario: email-outreach-manager
    capability: Lead nurturing automation triggers
    fallback: Manual lead export via CSV
```

## 🎨 Style and Branding Requirements

### UI/UX Style Guidelines
```yaml
style_profile:
  category: professional
  inspiration: Modern SaaS dashboard with clean, business-focused aesthetic
  
  visual_style:
    color_scheme: light
    typography: modern
    layout: dashboard
    animations: subtle
  
  personality:
    tone: professional
    mood: confident
    target_feeling: Users should feel empowered and in control
```

### Target Audience Alignment
- **Primary Users**: Business owners, marketing teams, customer success managers
- **User Expectations**: Professional, reliable, easy-to-use business tool interface
- **Accessibility**: WCAG 2.1 AA compliance for business accessibility requirements
- **Responsive Design**: Desktop-first with tablet support, mobile view for monitoring

### Brand Consistency Rules
- **Scenario Identity**: Clean, professional SaaS platform aesthetic
- **Vrooli Integration**: Consistent with Vrooli's business-focused scenarios
- **Professional Design**: Business/enterprise tool requiring professional design approach

## 💰 Value Proposition

### Business Value
- **Primary Value**: Automated 24/7 customer engagement and lead generation
- **Revenue Potential**: $15K - $50K per deployment (typical chatbot platform value)
- **Cost Savings**: Replace human support agents for common queries, 30-40% cost reduction
- **Market Differentiator**: Privacy-first, self-hosted solution with no external API costs

### Technical Value
- **Reusability Score**: 9/10 - Conversational AI patterns applicable across multiple scenarios
- **Complexity Reduction**: Simplifies customer interaction automation for any business
- **Innovation Enablement**: Creates foundation for advanced conversational AI applications

## 🧬 Evolution Path

### Version 1.0 (Current)
- Core chatbot creation and management
- Basic website widget integration  
- Ollama-powered conversations
- PostgreSQL data persistence

### Version 2.0 (Planned)
- Multi-channel deployment (SMS, email, social)
- Advanced analytics and conversation insights
- CRM integration marketplace
- White-label deployment options

### Long-term Vision
- Voice conversation support with speech processing
- Advanced ML-powered conversation optimization
- Enterprise-grade scalability and multi-tenancy
- AI model fine-tuning based on conversation data

## 🔄 Scenario Lifecycle Integration

### Direct Scenario Deployment
```yaml
direct_execution:
  supported: true
  structure_compliance:
    - service.json with complete metadata
    - PostgreSQL schema and seed data
    - Health check endpoints
    - Widget generation scripts
    
  deployment_targets:
    - local: Docker Compose with Ollama + PostgreSQL
    - kubernetes: Helm chart with persistent storage
    - cloud: AWS/GCP templates with managed databases
    
  revenue_model:
    - type: subscription
    - pricing_tiers: 
        - Starter: $29/month (up to 3 chatbots, 1000 conversations)
        - Professional: $99/month (unlimited chatbots, 10000 conversations)  
        - Enterprise: $299/month (white-label, advanced analytics)
    - trial_period: 14 days
```

### Capability Discovery
```yaml
discovery:
  registry_entry:
    name: ai-chatbot-manager
    category: automation
    capabilities: 
      - AI-powered website chatbots
      - Lead capture and qualification
      - Real-time conversation analytics
      - Embeddable widget generation
    interfaces:
      - api: /api/v1
      - cli: ai-chatbot-manager
      - events: chatbot.*
      
  metadata:
    description: Build and manage AI chatbots for 24/7 sales and support
    keywords: [chatbot, ai, sales, support, website, widgets, automation]
    dependencies: [ollama, postgres]
    enhances: [lead-scoring, customer-insights, email-outreach]
```

## 🚨 Risk Mitigation

### Technical Risks
| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| Ollama service unavailable | Medium | High | Graceful fallback with retry logic and error messages |
| WebSocket connection drops | Medium | Medium | Auto-reconnection with conversation state persistence |
| Database connection issues | Low | High | Connection pooling with health checks and failover |
| Widget XSS vulnerabilities | Low | High | Content Security Policy and input sanitization |

### Operational Risks
- **Drift Prevention**: PRD serves as single source of truth, validated by automated testing
- **Version Compatibility**: API versioning with backward compatibility guarantees
- **Resource Conflicts**: Isolated PostgreSQL schema and configurable ports
- **Security**: Input validation, rate limiting, and audit logging for all interactions

## ✅ Validation Criteria

### Performance Validation
- [ ] API response times meet SLA targets (< 500ms for chat)
- [ ] WebSocket connections handle 1000+ concurrent users
- [ ] Widget loads in < 2 seconds on test websites
- [ ] Memory usage stays under 2GB under load
- [ ] No conversation data loss during high traffic

### Integration Validation
- [ ] Discoverable via resource registry
- [ ] All API endpoints documented and functional
- [ ] CLI commands executable with comprehensive --help
- [ ] Events published/consumed correctly
- [ ] Widget embeds successfully in multiple website frameworks

### Capability Verification  
- [ ] Creates functional chatbots that respond intelligently
- [ ] Captures leads with complete contact information
- [ ] Generates accurate analytics and conversion metrics
- [ ] Maintains conversation context across message exchanges
- [ ] Integrates seamlessly into existing websites

## 📝 Implementation Notes

### Design Decisions
**WebSocket vs HTTP**: Chose WebSockets for real-time chat experience
- Alternative considered: Server-sent events or polling
- Decision driver: Better user experience with instant message delivery
- Trade-offs: More complex state management for improved responsiveness

**Local Ollama vs Cloud API**: Using local Ollama for privacy and cost control
- Alternative considered: OpenAI API or other cloud services
- Decision driver: Zero external API costs and complete data privacy
- Trade-offs: Requires local resource management but eliminates vendor lock-in

### Known Limitations
- **Concurrent Conversations**: Limited by local Ollama model capacity
  - Workaround: Implement queue system for high-traffic periods
  - Future fix: Add auto-scaling with multiple Ollama instances

### Security Considerations
- **Data Protection**: All conversations encrypted at rest in PostgreSQL
- **Access Control**: API key authentication for chatbot management
- **Audit Trail**: Complete conversation history with IP and timestamp logging

## 🔗 References

### Documentation
- README.md - User-facing setup and usage guide
- docs/API.md - Complete API specification
- docs/cli.md - CLI command reference  
- docs/widget-integration.md - Website embedding guide ✅ (Comprehensive guide with examples)

### Related PRDs
- [Link to lead-scoring-engine PRD when available]
- [Link to customer-insights-analyzer PRD when available]

### External Resources
- [Ollama API Documentation](https://ollama.ai/docs/api)
- [WebSocket Best Practices](https://developer.mozilla.org/en-US/docs/Web/API/WebSockets_API/Writing_WebSocket_client_applications)
- [Chatbot UX Design Principles](https://www.smashingmagazine.com/2018/02/designing-better-chatbots/)

---

**Last Updated**: 2025-01-13  
**Status**: Implementation Complete  
**Owner**: Claude Code AI  
**Review Cycle**: Weekly validation against implementation progress

### Implementation Status
- ✅ Core API implementation with WebSocket support
- ✅ CLI with dynamic port discovery
- ✅ React UI with component separation
- ✅ PostgreSQL integration with exponential backoff
- ✅ Widget generation and embedding system
- ✅ Widget integration documentation
- ✅ No hardcoded ports/URLs (fixed in v1.0.1)
- ✅ No n8n workflows (verified clean)