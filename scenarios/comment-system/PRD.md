# Product Requirements Document (PRD)

## 🎯 Capability Definition

### Core Capability
**What permanent capability does this scenario add to Vrooli?**
A universal comment system microservice that provides collaborative discussion capabilities to all Vrooli scenarios. This creates a social layer for the entire ecosystem, enabling user engagement, feedback collection, and community building across every generated app.

### Intelligence Amplification
**How does this capability make future agents smarter?**
- **User Feedback Loop**: Comments become training data for understanding user needs and preferences
- **Cross-Scenario Learning**: User discussions reveal patterns of how capabilities connect
- **Community Intelligence**: Collective user knowledge enhances problem-solving capabilities
- **Engagement Metrics**: Comment activity guides priority decisions for scenario improvements

### Recursive Value
**What new scenarios become possible after this exists?**
1. **Community-Manager**: Automated community management with sentiment analysis and trend detection
2. **User-Research-Hub**: Aggregate user feedback across all apps for product insights
3. **Social-Analytics**: Deep analysis of user engagement patterns to optimize experiences
4. **Collaborative-Workspaces**: Multi-user scenarios with real-time discussion capabilities
5. **AI-Moderation-System**: Intelligent content moderation leveraging comment patterns and user behavior

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [x] Core CRUD operations for comments (create, read, update, delete)
  - [x] Thread/reply support with nested comment structure
  - [x] Integration with session-authenticator for user authentication
  - [x] Integration with notification-hub for comment notifications
  - [x] PostgreSQL persistence with robust schema design
  - [x] REST API for programmatic access by other scenarios
  - [x] JavaScript SDK for easy integration into any web UI
  - [x] Admin dashboard for per-scenario configuration management
  - [x] Markdown rendering support for comment display
  - [x] Per-scenario authentication requirements (optional vs required login)
  
- **Should Have (P1)**
  - [ ] Real-time comment updates via WebSocket
  - [ ] Comment search and filtering capabilities
  - [ ] User mention system (@username notifications)
  - [ ] Comment edit history and version tracking
  - [ ] Configurable comment themes per scenario
  - [ ] Rich media attachment support (images, files) - configurable
  - [ ] Comment voting/rating system
  - [ ] Export functionality for comment data
  
- **Nice to Have (P2)**
  - [ ] Integration hooks for future AI-moderation-system scenario
  - [ ] Comment analytics and engagement metrics
  - [ ] Advanced formatting toolbar for comment composition
  - [ ] Comment templates and quick replies
  - [ ] Bulk moderation operations
  - [ ] Comment scheduling functionality
  - [ ] Mobile-responsive comment widget

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Response Time | < 200ms for 95% of comment CRUD operations | API monitoring |
| Throughput | 1000 comments/second sustained | Load testing |
| Real-time Latency | < 50ms WebSocket message delivery | Real-time monitoring |
| Resource Usage | < 512MB memory, < 20% CPU under normal load | System monitoring |
| Database Query Time | < 100ms for comment fetching with 10k+ comments | Database monitoring |

### Quality Gates
- [x] All P0 requirements implemented and tested
- [ ] Integration tests pass with session-authenticator and notification-hub
- [ ] Performance targets met under load testing
- [ ] Documentation complete (README, API docs, CLI help, integration guide)
- [ ] Scenario can be invoked by other scenarios via API/SDK
- [ ] JavaScript SDK works across different framework environments

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  - resource_name: postgres
    purpose: Persistent storage for comments, configurations, and user data
    integration_pattern: Direct SQL via Go database/sql
    access_method: Connection pooling with prepared statements
    
optional:
  - resource_name: redis
    purpose: Caching frequently accessed comments and real-time updates
    fallback: Direct database queries with pagination
    access_method: Redis client for caching layer
```

### Resource Integration Standards
```yaml
# Priority order for resource access (MUST follow this hierarchy):
integration_priorities:
  1_shared_workflows:     # FIRST: Use existing shared n8n workflows
    - workflow: N/A - This scenario deliberately avoids n8n for performance
      location: N/A
      purpose: N/A - Direct service implementation for speed and reliability
  
  2_resource_cli:        # SECOND: Use resource CLI commands
    - command: resource-postgres [action]
      purpose: Database health checks and administrative operations
  
  3_direct_api:          # LAST: Direct API only when necessary
    - justification: Core service requires direct PostgreSQL and optimal performance
      endpoint: Direct database connection pools

# Integration with other scenarios:
scenario_integrations:
  session_authenticator:
    - purpose: User authentication and profile management
    - interface: REST API calls to verify tokens and get user info
    - fallback: Anonymous commenting if authentication is optional
  
  notification_hub:
    - purpose: Send notifications for comment replies and mentions
    - interface: Event publishing to notification service
    - fallback: Skip notifications if service unavailable
    
  ai_moderation_system:
    - purpose: Future automated content moderation
    - interface: Webhook/API integration for moderation decisions
    - fallback: Manual moderation through admin dashboard
```

### Data Models
```yaml
# Core data structures that define the capability
primary_entities:
  - name: ScenarioConfig
    storage: postgres
    schema: |
      {
        id: UUID PRIMARY KEY,
        scenario_name: VARCHAR(255) UNIQUE NOT NULL,
        auth_required: BOOLEAN DEFAULT true,
        allow_anonymous: BOOLEAN DEFAULT false,
        allow_rich_media: BOOLEAN DEFAULT false,
        moderation_level: ENUM('none', 'manual', 'ai_assisted') DEFAULT 'manual',
        theme_config: JSONB,
        notification_settings: JSONB,
        created_at: TIMESTAMP,
        updated_at: TIMESTAMP
      }
    relationships: One-to-many with Comments
    
  - name: Comment
    storage: postgres
    schema: |
      {
        id: UUID PRIMARY KEY,
        scenario_name: VARCHAR(255) NOT NULL,
        parent_id: UUID REFERENCES comments(id),
        author_id: UUID, # From session-authenticator
        author_name: VARCHAR(255), # Cached from session-authenticator
        content: TEXT NOT NULL,
        content_type: ENUM('plaintext', 'markdown') DEFAULT 'markdown',
        metadata: JSONB, # For attachments, mentions, etc.
        status: ENUM('active', 'deleted', 'moderated') DEFAULT 'active',
        created_at: TIMESTAMP,
        updated_at: TIMESTAMP,
        version: INTEGER DEFAULT 1
      }
    relationships: Self-referential parent/child, belongs to ScenarioConfig
    
  - name: CommentHistory
    storage: postgres
    schema: |
      {
        id: UUID PRIMARY KEY,
        comment_id: UUID NOT NULL REFERENCES comments(id),
        content: TEXT NOT NULL,
        edited_at: TIMESTAMP,
        edited_by: UUID # User who made the edit
      }
    relationships: Belongs to Comment
```

### API Contract
```yaml
# Defines how other scenarios/agents can use this capability
endpoints:
  - method: GET
    path: /api/v1/comments/{scenario_name}
    purpose: Retrieve comments for a specific scenario
    input_schema: |
      {
        scenario_name: "string",
        parent_id?: "uuid",
        limit?: number,
        offset?: number,
        sort?: "newest" | "oldest" | "threaded"
      }
    output_schema: |
      {
        comments: Comment[],
        total_count: number,
        has_more: boolean
      }
    sla:
      response_time: 150ms
      availability: 99.9%
      
  - method: POST
    path: /api/v1/comments/{scenario_name}
    purpose: Create new comment in specified scenario
    input_schema: |
      {
        content: "string",
        parent_id?: "uuid",
        content_type?: "plaintext" | "markdown",
        author_token: "string" # From session-authenticator
      }
    output_schema: |
      {
        comment: Comment,
        success: boolean,
        message?: "string"
      }
    sla:
      response_time: 200ms
      availability: 99.9%
      
  - method: PUT
    path: /api/v1/comments/{comment_id}
    purpose: Update existing comment
    input_schema: |
      {
        content: "string",
        author_token: "string"
      }
    output_schema: |
      {
        comment: Comment,
        success: boolean
      }
    sla:
      response_time: 150ms
      availability: 99.9%
      
  - method: DELETE
    path: /api/v1/comments/{comment_id}
    purpose: Soft delete a comment
    input_schema: |
      {
        author_token: "string"
      }
    output_schema: |
      {
        success: boolean,
        message?: "string"
      }
    sla:
      response_time: 100ms
      availability: 99.9%
      
  - method: GET
    path: /api/v1/config/{scenario_name}
    purpose: Get comment system configuration for scenario
    output_schema: |
      {
        config: ScenarioConfig
      }
    sla:
      response_time: 50ms
      availability: 99.9%
      
  - method: POST
    path: /api/v1/config/{scenario_name}
    purpose: Update comment system configuration for scenario
    input_schema: |
      {
        auth_required?: boolean,
        allow_anonymous?: boolean,
        allow_rich_media?: boolean,
        moderation_level?: "none" | "manual" | "ai_assisted",
        theme_config?: object,
        notification_settings?: object
      }
    output_schema: |
      {
        config: ScenarioConfig,
        success: boolean
      }
    sla:
      response_time: 100ms
      availability: 99.9%
```

### Event Interface
```yaml
# Events this capability publishes for others to consume
published_events:
  - name: comment.created
    payload: |
      {
        comment_id: "uuid",
        scenario_name: "string",
        author_id: "uuid",
        parent_id?: "uuid",
        created_at: "timestamp"
      }
    subscribers: [notification-hub, analytics scenarios]
    
  - name: comment.reply.created
    payload: |
      {
        reply_id: "uuid",
        parent_comment_id: "uuid",
        scenario_name: "string",
        author_id: "uuid",
        parent_author_id: "uuid"
      }
    subscribers: [notification-hub for reply notifications]
    
  - name: comment.mention.created
    payload: |
      {
        comment_id: "uuid",
        mentioned_user_id: "uuid",
        author_id: "uuid",
        scenario_name: "string"
      }
    subscribers: [notification-hub for mention notifications]
    
consumed_events:
  - name: session_authenticator.user.updated
    action: Update cached user information in comments
    
  - name: ai_moderation_system.decision.made
    action: Update comment status based on moderation decision
```

## 🖥️ CLI Interface Contract

### Command Structure
```yaml
# Primary CLI executable name and pattern
cli_binary: comment-system
install_script: cli/install.sh

# Core commands that MUST be implemented:
required_commands:
  - name: status
    description: Show operational status and resource health
    flags: [--json, --verbose]
    
  - name: help
    description: Display command help and usage
    flags: [--all, --command <name>]
    
  - name: version
    description: Show CLI and API version information
    flags: [--json]

# Scenario-specific commands:
custom_commands:
  - name: list
    description: List comments for a scenario
    api_endpoint: /api/v1/comments/{scenario_name}
    arguments:
      - name: scenario_name
        type: string
        required: true
        description: Name of the scenario to list comments for
    flags:
      - name: --parent
        description: Filter by parent comment ID
      - name: --limit
        description: Maximum number of comments to return
      - name: --format
        description: Output format (json, table, tree)
    output: Formatted comment list
    
  - name: create
    description: Create a new comment
    api_endpoint: /api/v1/comments/{scenario_name}
    arguments:
      - name: scenario_name
        type: string
        required: true
        description: Target scenario for the comment
      - name: content
        type: string
        required: true
        description: Comment content
    flags:
      - name: --parent
        description: Parent comment ID for replies
      - name: --markdown
        description: Content is markdown formatted
    output: Created comment details
    
  - name: config
    description: Manage scenario comment configurations
    api_endpoint: /api/v1/config/{scenario_name}
    arguments:
      - name: scenario_name
        type: string
        required: true
        description: Scenario to configure
    flags:
      - name: --auth-required
        description: Require authentication for comments
      - name: --allow-anonymous
        description: Allow anonymous comments
      - name: --rich-media
        description: Enable rich media attachments
      - name: --moderation
        description: Set moderation level
    output: Configuration status
    
  - name: moderate
    description: Moderate comments (admin operations)
    api_endpoint: /api/v1/comments/{comment_id}
    arguments:
      - name: comment_id
        type: string
        required: true
        description: Comment to moderate
      - name: action
        type: string
        required: true
        description: Moderation action (approve, delete, flag)
    output: Moderation result
```

### CLI-API Parity Requirements
- **Coverage**: Every API endpoint has a corresponding CLI command
- **Naming**: CLI commands use kebab-case versions of API endpoints
- **Arguments**: CLI arguments map directly to API parameters
- **Output**: Support both human-readable and JSON output (--json flag)
- **Authentication**: Integrate with session-authenticator for user context

### Implementation Standards
```yaml
implementation_requirements:
  - architecture: Thin wrapper over Go API client
  - language: Go for consistency with API service
  - dependencies: Minimal - reuse API client libraries
  - error_handling: Consistent exit codes (0=success, 1=error)
  - configuration: 
      - Read from ~/.vrooli/comment-system/config.yaml
      - Environment variables override config
      - Command flags override everything
  
installation:
  - install_script: Must create symlink in ~/.vrooli/bin/
  - path_update: Must add ~/.vrooli/bin to PATH if not present
  - permissions: Executable permissions (755) required
  - documentation: Generated via --help must be comprehensive
```

## 🔄 Integration Requirements

### Upstream Dependencies
**What capabilities must exist before this can function?**
- **PostgreSQL Resource**: Essential for data persistence and ACID compliance
- **Session-Authenticator Scenario**: Required for user authentication and profile data
- **Notification-Hub Scenario**: Required for sending comment notifications and mentions

### Downstream Enablement
**What future capabilities does this unlock?**
- **AI-Moderation-System**: Automated content moderation with training data from comment patterns
- **Community-Analytics**: Deep insights into user engagement across all Vrooli scenarios
- **Social-Features**: Foundation for likes, follows, user reputation systems
- **Collaborative-Tools**: Multi-user scenarios with built-in discussion capabilities

### Cross-Scenario Interactions
```yaml
# How this scenario enhances other scenarios
provides_to:
  - scenario: "*" # All scenarios
    capability: Comment and discussion functionality
    interface: JavaScript SDK and REST API
    
  - scenario: community-manager
    capability: Comment moderation and analytics data
    interface: API and events
    
  - scenario: user-research-hub
    capability: User feedback and sentiment data
    interface: API for comment analysis
    
consumes_from:
  - scenario: session-authenticator
    capability: User authentication and profile data
    fallback: Anonymous commenting where configured
    
  - scenario: notification-hub
    capability: Push notifications for comments and replies
    fallback: Skip notifications if service unavailable
    
  - scenario: ai-moderation-system
    capability: Automated content moderation decisions
    fallback: Manual moderation through admin dashboard
```

## 🎨 Style and Branding Requirements

### UI/UX Style Guidelines
```yaml
style_profile:
  category: professional
  inspiration: GitHub comments, Discord chat, modern commenting systems
  
  # Visual characteristics:
  visual_style:
    color_scheme: adaptive # Matches host scenario theme
    typography: clean # Sans-serif, readable at small sizes
    layout: threaded # Nested comment structure
    animations: subtle # Smooth expand/collapse, typing indicators
  
  # Personality traits:
  personality:
    tone: friendly # Encouraging participation
    mood: focused # Stays out of the way
    target_feeling: "Safe to contribute and discuss"

# Admin dashboard style:
admin_style:
  category: professional
  inspiration: Modern SaaS dashboard
  visual_style:
    color_scheme: light # Professional admin interface
    typography: modern # Clean, information-dense
    layout: dashboard # Multi-panel configuration view
    animations: none # Fast, functional interface
```

### Target Audience Alignment
- **Primary Users**: End users of Vrooli scenarios who want to discuss and collaborate
- **Secondary Users**: Scenario developers who need to configure comment behavior
- **Admin Users**: System administrators managing content and moderation
- **User Expectations**: Familiar commenting patterns similar to GitHub, Reddit, or Slack
- **Accessibility**: WCAG 2.1 AA compliance for keyboard navigation and screen readers
- **Responsive Design**: Mobile-first design for comment widget, desktop-optimized admin

### Brand Consistency Rules
- **Scenario Identity**: Clean, universal design that adapts to host scenario
- **Vrooli Integration**: Uses consistent API patterns and error handling
- **Professional Focus**: Business tool that enhances other scenarios without distraction

## 💰 Value Proposition

### Business Value
- **Primary Value**: Transforms every Vrooli scenario into a collaborative platform
- **Revenue Potential**: $15K - $30K per deployment (adds social features to every app)
- **Cost Savings**: Eliminates need for custom comment systems in each scenario
- **Market Differentiator**: Universal social layer across all generated applications

### Technical Value
- **Reusability Score**: 10/10 - Used by potentially every scenario in the ecosystem
- **Complexity Reduction**: Complex commenting features become one-line integrations
- **Innovation Enablement**: Enables community-driven scenarios and collaborative features

## 🧬 Evolution Path

### Version 1.0 (Current)
- Core CRUD operations and threading
- Session-authenticator integration
- Notification-hub integration
- Basic admin configuration
- JavaScript SDK and widget

### Version 2.0 (Planned)
- Real-time WebSocket updates
- AI-moderation-system integration
- Advanced search and filtering
- Rich media support
- Mobile app SDK

### Long-term Vision
- Voice comments and transcription
- AI-powered comment suggestions
- Cross-scenario user reputation system
- Automated community moderation

## 🔄 Scenario Lifecycle Integration

### Direct Scenario Deployment
```yaml
direct_execution:
  supported: true
  structure_compliance:
    - .vrooli/service.json with PostgreSQL dependency
    - Complete REST API with health endpoints
    - CLI tool with installation script
    - JavaScript SDK for web integration
    
  deployment_targets:
    - local: Docker Compose with PostgreSQL
    - kubernetes: StatefulSet for database persistence
    - cloud: RDS/Cloud SQL for managed database
    
  revenue_model:
    - type: usage-based
    - pricing_tiers: Based on comment volume and scenarios
    - trial_period: 30 days with full features
```

### Capability Discovery
```yaml
discovery:
  registry_entry:
    name: comment-system
    category: social
    capabilities: [comments, discussions, notifications, moderation]
    interfaces:
      - api: /api/v1/comments
      - cli: comment-system
      - events: comment.*
      - sdk: VrooliComments JavaScript library
      
  metadata:
    description: Universal comment system for all Vrooli scenarios
    keywords: [comments, discussions, collaboration, social, community]
    dependencies: [session-authenticator, notification-hub]
    enhances: ["*"] # All scenarios
```

### Version Management
```yaml
versioning:
  current: 1.0.0
  minimum_compatible: 1.0.0
  
  breaking_changes: []
  
  deprecations: []
```

## 🚨 Risk Mitigation

### Technical Risks
| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| Database performance under high comment volume | Medium | High | Connection pooling, indexing, pagination |
| Session-authenticator service unavailability | Low | Medium | Graceful fallback to anonymous comments |
| Cross-origin issues with SDK | Medium | High | Comprehensive CORS configuration and testing |
| WebSocket connection limits | Low | Medium | Connection pooling and fallback to polling |

### Operational Risks
- **Spam and Abuse**: Rate limiting, integration hooks for AI moderation
- **Data Privacy**: Compliance with user data deletion requests
- **Cross-Scenario Conflicts**: Isolated configurations per scenario
- **Performance Degradation**: Caching strategies and database optimization

## ✅ Validation Criteria

### Declarative Test Specification
Phased testing is orchestrated via `test/run-tests.sh`, which sources the shared runner in
`scripts/scenarios/testing/shell/runner.sh` and emits live artifacts beneath
`coverage/phase-results/*.json` for requirements reporting.

- **Structure** — verifies presence/permissions of core files (`api/main.go`, `cli/comment-system`, schema, SDK assets) and required directories.
- **Dependencies** — runs `go mod verify`, smoke checks npm dependency resolution, and exercises `comment-system help` to ensure the CLI is callable.
- **Unit** — executes Go unit tests with coverage aggregation through `testing::unit::run_all_tests`, exporting reports to `coverage/comment-system/go/`.
- **Integration** — drives CRUD + external dependency flows by running `test/test-comment-crud.sh` and `test/test-integration.sh` against the runtime-discovered API URL.
- **Business** — validates threaded conversations (`test/test-threading.sh`) and CLI workflows (`comment-system status --json`).
- **Performance** — samples health/latency for `/api/v1/comments/{scenario}` and raises warnings when baseline thresholds are exceeded.

The lifecycle `test` step now delegates to this phased suite, ensuring `make test` and `vrooli scenario test comment-system` share identical coverage-aware execution.

### Performance Validation
- [x] API response times meet SLA targets (<200ms for CRUD operations)
- [ ] Resource usage within defined limits (<512MB memory, <20% CPU)
- [ ] Throughput meets minimum requirements (1000 comments/second)
- [ ] No memory leaks detected over 24-hour test

### Integration Validation
- [ ] Integration with session-authenticator for user authentication
- [ ] Integration with notification-hub for comment notifications
- [ ] JavaScript SDK works across different web frameworks
- [ ] All API endpoints documented and functional
- [ ] All CLI commands executable with --help

### Capability Verification
- [x] Universal commenting capability for all scenarios
- [x] Thread/reply functionality with proper nesting
- [x] Markdown rendering and display
- [x] Admin configuration management
- [x] Configurable authentication requirements

## 📝 Implementation Notes

### Design Decisions
**Database Choice**: PostgreSQL chosen for ACID compliance and complex querying needs
- Alternative considered: MongoDB for document storage
- Decision driver: Need for consistent transactions and relational integrity
- Trade-offs: More complex setup vs. data consistency guarantees

**No n8n Usage**: Direct service implementation for optimal performance
- Alternative considered: n8n workflows for processing
- Decision driver: Performance requirements and reliability concerns
- Trade-offs: More code to maintain vs. faster, more reliable service

**JavaScript SDK**: Universal SDK for easy integration across frameworks
- Alternative considered: Framework-specific components
- Decision driver: Maximum compatibility and ease of adoption
- Trade-offs: Larger bundle size vs. universal compatibility

### Known Limitations
- **Real-time Updates**: Initial version uses polling, WebSocket in v2
  - Workaround: Configurable polling intervals
  - Future fix: WebSocket implementation in version 2.0
- **Rich Media**: File uploads require additional storage configuration
  - Workaround: Link-based media embedding initially
  - Future fix: Integration with file storage service

### Security Considerations
- **Data Protection**: All user data encrypted at rest and in transit
- **Access Control**: Comment authors can edit/delete their own comments
- **Audit Trail**: Complete comment history and edit tracking
- **XSS Prevention**: Markdown sanitization to prevent script injection
- **Rate Limiting**: Per-user and per-IP rate limits to prevent spam

## 🔗 References

### Documentation
- README.md - User-facing overview and integration guide
- docs/api.md - Complete API specification
- docs/cli.md - CLI command documentation
- docs/sdk.md - JavaScript SDK documentation
- docs/integration.md - How to integrate into scenarios

### Related PRDs
- session-authenticator - User authentication dependency
- notification-hub - Notification delivery dependency  
- ai-moderation-system - Future moderation integration

### External Resources
- [CommonMark Specification](https://commonmark.org/) - Markdown parsing standard
- [WebSocket RFC 6455](https://tools.ietf.org/html/rfc6455) - Real-time communication
- [WCAG 2.1](https://www.w3.org/WAI/WCAG21/quickref/) - Accessibility guidelines

---

**Last Updated**: 2025-09-09  
**Status**: Draft  
**Owner**: AI Agent (Claude Code)  
**Review Cycle**: Every implementation milestone
