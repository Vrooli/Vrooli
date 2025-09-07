# Product Requirements Document (PRD) - Game Dialog Generator

## 🎯 Capability Definition

### Core Capability
**What permanent capability does this scenario add to Vrooli?**

The Game Dialog Generator adds AI-powered character dialog generation for video games and interactive media. This capability enables:
- Context-aware dialog generation based on character personality, relationships, and scene context
- Real-time dynamic dialog for adaptive gameplay experiences
- Batch generation for traditional fixed-dialog games
- Character voice synthesis with distinct personality-based audio profiles
- Semantic search and retrieval of existing dialog content
- Character relationship modeling that evolves dialog over time

### Intelligence Amplification
**How does this capability make future agents smarter?**

This capability compounds with existing Vrooli capabilities to enable:
- **Game Development Scenarios**: Provides dialog generation as a service to game-making scenarios
- **Interactive Storytelling**: Powers narrative-driven scenarios with dynamic character interactions
- **Content Creation Pipelines**: Integrates with writing and creative scenarios for dialog scripting
- **Character AI Systems**: Enables scenarios that need persistent character personalities
- **Voice-Enabled Applications**: Provides character voice synthesis for any scenario needing audio personas
- **Narrative Analytics**: Enables scenarios to analyze dialog patterns and character development

### Recursive Value
**What new scenarios become possible after this exists?**

1. **Interactive Fiction Studio** - Visual novel and interactive story creation with dynamic dialog trees
2. **Game Master AI** - Tabletop RPG assistant that voices NPCs with distinct personalities
3. **Voice Acting Studio** - Character voice synthesis for audiobooks, podcasts, and media
4. **Personality-Driven Chatbots** - Customer service or entertainment bots with game character personalities
5. **Dialog Tree Designer** - Visual tool for creating complex branching conversations

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [x] Character personality definition and storage system
  - [x] Scene context management with semantic search via Qdrant
  - [x] Real-time dialog generation API with streaming responses
  - [x] Batch dialog generation for traditional game workflows
  - [x] PostgreSQL integration for persistent character and project data
  - [x] Ollama integration for local LLM-powered dialog generation
  
- **Should Have (P1)**
  - [ ] Voice synthesis with character-specific audio profiles
  - [ ] Character relationship modeling affecting dialog dynamics
  - [ ] Dialog history tracking and consistency checks
  - [ ] Export formats for popular game engines (Unity, Unreal, Godot)
  - [ ] Character emotion state tracking influencing dialog tone
  - [ ] Dialog validation against character personality constraints
  
- **Nice to Have (P2)**
  - [ ] Visual character relationship graphs
  - [ ] Dialog analytics and quality metrics
  - [ ] Multi-language dialog generation
  - [ ] Integration with existing game dialog systems
  - [ ] Collaborative character development features
  - [ ] Dialog A/B testing capabilities

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Real-time Dialog Response | < 2s for 95% of requests | API monitoring |
| Batch Dialog Throughput | 100+ dialogs/minute | Load testing |
| Character Consistency | > 85% personality adherence | AI validation scoring |
| Resource Usage | < 2GB memory, < 50% CPU | System monitoring |
| Voice Generation Speed | < 5s per dialog line | Audio pipeline timing |

### Quality Gates
- [x] All P0 requirements implemented and tested
- [ ] Integration tests pass with all required resources
- [ ] Performance targets met under load
- [ ] Documentation complete (README, API docs, CLI help)
- [ ] Scenario can be invoked by other agents via API/CLI

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  - resource_name: postgres
    purpose: Store character definitions, dialog history, and project data
    integration_pattern: Direct SQL via Go database/sql package
    access_method: PostgreSQL connection string
    
  - resource_name: qdrant
    purpose: Vector embeddings for character traits, scene context, and semantic dialog search
    integration_pattern: Qdrant REST API via Go client
    access_method: Qdrant HTTP API endpoints
    
  - resource_name: ollama
    purpose: Local LLM for dialog generation and character personality modeling
    integration_pattern: Ollama REST API for inference
    access_method: resource-ollama CLI wrapper and direct HTTP API
    
optional:
  - resource_name: whisper
    purpose: Voice synthesis for character-specific audio generation
    fallback: Text-only dialog generation without voice synthesis
    access_method: Whisper API for TTS conversion
```

### Resource Integration Standards
```yaml
# Priority order for resource access (follows Vrooli patterns):
integration_priorities:  
  1_resource_cli:        # FIRST: Use resource CLI commands
    - command: resource-ollama generate --model llama3.2
      purpose: Dialog generation with personality constraints
    - command: resource-qdrant search --collection characters
      purpose: Character and scene context retrieval
  
  2_direct_api:          # SECOND: Direct API for performance-critical operations
    - justification: Real-time dialog streaming requires direct HTTP connections
      endpoint: /v1/chat/completions (Ollama), /collections/search (Qdrant)

# NO n8n workflows - direct integration only
shared_workflow_criteria:
  - NOTE: This scenario does NOT use n8n workflows per project direction
  - All resource access via CLI commands or direct API calls
```

### Data Models
```yaml
# Core data structures that define the capability
primary_entities:
  - name: Character
    storage: postgres
    schema: |
      {
        id: UUID,
        name: string,
        personality_traits: jsonb,
        background_story: text,
        speech_patterns: jsonb,
        relationships: jsonb,
        voice_profile: jsonb,
        created_at: timestamp,
        updated_at: timestamp
      }
    relationships: Many-to-many with Projects, one-to-many with DialogHistory
    
  - name: Scene
    storage: postgres
    schema: |
      {
        id: UUID,
        project_id: UUID,
        name: string,
        context: text,
        setting: jsonb,
        mood: string,
        participants: UUID[],
        created_at: timestamp
      }
    relationships: Belongs to Project, references Characters via participants
    
  - name: DialogLine
    storage: postgres
    schema: |
      {
        id: UUID,
        character_id: UUID,
        scene_id: UUID,
        content: text,
        emotion: string,
        context_embedding: vector(stored in qdrant),
        audio_file_path: string,
        generated_at: timestamp
      }
    relationships: Belongs to Character and Scene
    
  - name: Project
    storage: postgres
    schema: |
      {
        id: UUID,
        name: string,
        description: text,
        characters: UUID[],
        settings: jsonb,
        export_format: string,
        created_at: timestamp
      }
    relationships: Has many Characters, Scenes, and DialogLines
```

### API Contract
```yaml
# Defines how other scenarios/agents can use this capability
endpoints:
  - method: POST
    path: /api/v1/dialog/generate
    purpose: Generate dialog in real-time for dynamic games
    input_schema: |
      {
        character_id: UUID,
        scene_context: string,
        previous_dialog: string[],
        emotion_state: string,
        constraints: object
      }
    output_schema: |
      {
        dialog: string,
        emotion: string,
        character_consistency_score: float,
        audio_url?: string
      }
    sla:
      response_time: 2000ms
      availability: 99%
      
  - method: POST
    path: /api/v1/dialog/batch
    purpose: Generate multiple dialog lines for traditional game development
    input_schema: |
      {
        scene_id: UUID,
        dialog_requests: array,
        export_format: string
      }
    output_schema: |
      {
        dialog_set: array,
        export_file_url: string,
        generation_metrics: object
      }
    sla:
      response_time: 30000ms
      availability: 95%
      
  - method: GET
    path: /api/v1/characters/{id}/personality
    purpose: Retrieve character personality for consistency checking
    input_schema: |
      {
        character_id: UUID
      }
    output_schema: |
      {
        personality_traits: object,
        speech_patterns: object,
        relationship_dynamics: object
      }
    sla:
      response_time: 500ms
      availability: 99%
```

### Event Interface
```yaml
# Events this capability publishes for others to consume
published_events:
  - name: dialog.generation.completed
    payload: { dialog_id: UUID, character_id: UUID, quality_score: float }
    subscribers: [game-engines, analytics-scenarios]
    
  - name: character.personality.updated
    payload: { character_id: UUID, changes: object }
    subscribers: [dialog-consistency-checkers]
    
consumed_events:
  - name: game.state.changed
    action: Update character emotion states based on game events
```

## 🖥️ CLI Interface Contract

### Command Structure
```yaml
# Primary CLI executable name and pattern
cli_binary: game-dialog-generator
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
  - name: character-create
    description: Create a new character with personality traits
    api_endpoint: /api/v1/characters
    arguments:
      - name: name
        type: string
        required: true
        description: Character name
      - name: personality-file
        type: string
        required: false
        description: JSON file with personality traits
    flags:
      - name: --interactive
        description: Interactive character creation wizard
    output: Character ID and confirmation
    
  - name: dialog-generate
    description: Generate dialog for a character in a scene
    api_endpoint: /api/v1/dialog/generate
    arguments:
      - name: character-id
        type: string
        required: true
        description: UUID of the character
      - name: scene-context
        type: string
        required: true
        description: Scene description or context
    flags:
      - name: --voice
        description: Generate audio file for dialog
      - name: --streaming
        description: Stream dialog generation in real-time
    output: Generated dialog text and optional audio file path
    
  - name: project-export
    description: Export project dialog for game engines
    api_endpoint: /api/v1/projects/{id}/export
    arguments:
      - name: project-id
        type: string
        required: true
        description: Project UUID to export
      - name: format
        type: string
        required: true
        description: Export format (unity, unreal, godot, json)
    output: Export file path and format information
```

## 🎨 Style and Branding Requirements

### UI/UX Style Guidelines
```yaml
# Define the visual and experiential personality of this scenario
style_profile:
  category: playful
  inspiration: "Classic jungle platformers like Donkey Kong Country meets modern game dev tools"
  
  # Visual characteristics:
  visual_style:
    color_scheme: "Rich jungle greens, warm earth tones, bright accent colors"
    primary_colors: ["#2F5233", "#4A7C59", "#7BA35C", "#B8D67A"]
    accent_colors: ["#FF6B35", "#F7931E", "#FFD23F"]
    background: "Layered jungle scenery with parallax scrolling effects"
    typography: "Playful but readable - mix of adventure game fonts with modern UI text"
    layout: "Card-based character management with jungle vine decorative elements"
    animations: "Bouncy, organic movements inspired by swinging vines and jumping primates"
  
  # Personality traits:
  personality:
    tone: "Playful and adventurous, but professional for game developers"
    mood: "Energetic exploration of creative possibilities"
    target_feeling: "Like you're building characters for the next great jungle adventure game"
    
  # Specific UI elements:
  jungle_platformer_elements:
    navigation: "Vine-style breadcrumbs that swing when hovered"
    buttons: "Wooden signs with carved text, stone tablets for important actions"
    character_cards: "Framed like character select screens in retro platformers"
    dialog_bubbles: "Leaf-shaped speech bubbles with organic borders"
    loading_states: "Swinging monkey animations, growing plant progress bars"
    backgrounds: "Layered jungle parallax with subtle animated elements"
    sounds: "Jungle ambient sounds, wood/stone clicks for interactions"
```

### Target Audience Alignment
- **Primary Users**: Independent game developers, narrative designers, game studios
- **User Expectations**: Fun, creative tool that doesn't feel corporate or sterile
- **Accessibility**: WCAG 2.1 AA compliance with high contrast options for readability
- **Responsive Design**: Desktop-first for game development workflow, tablet-friendly for creative sessions

### Brand Consistency Rules
- **Scenario Identity**: Adventure-themed game development tool that sparks creativity
- **Vrooli Integration**: Maintains Vrooli's technical excellence under a playful jungle theme
- **Professional vs Fun**: Balances whimsical aesthetics with serious game development functionality
  - UI feels like a game but provides professional-grade dialog generation tools
  - Character creation feels like designing RPG characters but with AI personality modeling
  - Export features are clearly marked and professional despite jungle theming

## 💰 Value Proposition

### Business Value
- **Primary Value**: Eliminates dialog writing bottleneck in game development (typically 20-40% of narrative work)
- **Revenue Potential**: $25K - $75K per deployment for game studios
- **Cost Savings**: Reduces dialog writing time from weeks to hours for indie developers
- **Market Differentiator**: Only local AI-powered game dialog generator with character consistency

### Technical Value
- **Reusability Score**: 9/10 - Character AI and dialog generation useful across many scenarios
- **Complexity Reduction**: Turns complex character personality modeling into simple API calls
- **Innovation Enablement**: Enables dynamic, AI-driven narrative games and interactive fiction

## 🧬 Evolution Path

### Version 1.0 (Current)
- Core character and dialog management
- Basic Ollama integration for text generation
- PostgreSQL storage and Qdrant embeddings
- Jungle-themed UI with character creation

### Version 2.0 (Planned)
- Voice synthesis with character-specific audio profiles
- Advanced relationship modeling affecting dialog dynamics
- Game engine export integrations (Unity, Unreal, Godot)
- Dialog analytics and quality metrics

### Long-term Vision
- Multi-modal character AI (text, voice, visual expressions)
- Real-time integration with game engines for live dialog generation
- Community character template sharing and remixing
- Advanced narrative AI that understands story arcs and character development

## 🔄 Integration Requirements

### Upstream Dependencies
**What capabilities must exist before this can function?**
- **Ollama Resource**: Required for LLM-powered dialog generation
- **Qdrant Resource**: Essential for character trait and scene context embeddings
- **PostgreSQL Resource**: Necessary for persistent character and project data storage

### Downstream Enablement
**What future capabilities does this unlock?**
- **Interactive Fiction Studio**: Provides character dialog engine
- **Game Master AI**: Enables NPC character voices and personalities
- **Voice Acting Studio**: Character voice synthesis foundation
- **Narrative Analytics**: Dialog pattern analysis and character development tracking

### Cross-Scenario Interactions
```yaml
# How this scenario enhances other scenarios
provides_to:
  - scenario: interactive-fiction-studio
    capability: Character dialog generation with personality consistency
    interface: API
    
  - scenario: game-master-ai
    capability: NPC voice synthesis and character modeling
    interface: CLI/API
    
  - scenario: voice-acting-studio
    capability: Character-specific voice synthesis
    interface: API
    
consumes_from:
  - scenario: content-creator-hub
    capability: Character background story generation
    fallback: Manual character creation interface
```

## 🔧 Implementation Requirements

### Direct Scenario Deployment
```yaml
# Requirements for direct scenario execution:
direct_execution:
  supported: true
  structure_compliance:
    - service.json with complete metadata
    - All required initialization files
    - Deployment scripts (startup.sh, monitor.sh)
    - Health check endpoints
    
  deployment_targets:
    - local: Docker Compose based
    - kubernetes: Helm chart generation
    - cloud: AWS/GCP/Azure templates
    
  revenue_model:
    - type: subscription
    - pricing_tiers: 
      - indie: $29/month (5 projects, 100 characters)
      - studio: $99/month (unlimited projects, 1000 characters)
      - enterprise: $299/month (unlimited, voice synthesis, API access)
    - trial_period: 14 days
```

### Capability Discovery
```yaml
# How other scenarios/agents discover and use this capability:
discovery:
  registry_entry:
    name: game-dialog-generator
    category: generation
    capabilities: [dialog-generation, character-ai, voice-synthesis, game-development]
    interfaces:
      - api: "http://localhost:{API_PORT}/api/v1"
      - cli: game-dialog-generator
      - events: dialog.*
      
  metadata:
    description: "AI-powered character dialog generation for video games and interactive media"
    keywords: [game-development, dialog, character-ai, narrative, voice-synthesis, storytelling]
    dependencies: [ollama, qdrant, postgres]
    enhances: [interactive-fiction, game-engines, content-creation]
```

## 🚨 Risk Mitigation

### Technical Risks
| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| Ollama service unavailability | Medium | High | Graceful degradation to cached responses, clear error messages |
| Character consistency degradation | Low | Medium | Personality validation scoring, consistency alerts |
| Voice synthesis quality issues | Low | Medium | Multiple TTS backends, quality scoring, fallback to text-only |

### Operational Risks
- **Drift Prevention**: PRD serves as single source of truth, validated by scenario-test.yaml
- **Version Compatibility**: Semantic versioning with clear breaking change documentation  
- **Resource Conflicts**: Resource allocation managed through service.json priorities
- **Style Drift**: UI components must pass jungle platformer theme validation
- **CLI Consistency**: Automated testing ensures CLI-API parity

## ✅ Validation Criteria

### Declarative Test Specification
```yaml
# REQUIRED: scenario-test.yaml in scenario root
version: 1.0
scenario: game-dialog-generator

# Structure validation - files and directories that MUST exist:
structure:
  required_files:
    - .vrooli/service.json
    - PRD.md
    - api/main.go
    - api/go.mod
    - cli/game-dialog-generator
    - cli/install.sh
    - initialization/storage/postgres/schema.sql
    - scenario-test.yaml
    
  required_dirs:
    - api
    - cli
    - initialization
    - initialization/storage
    - ui
    - tests
    - data

# Resource validation:
resources:
  required: [postgres, qdrant, ollama]
  optional: [whisper]
  health_timeout: 60

# Declarative tests:
tests:
  # Resource health checks:
  - name: "Postgres is accessible"
    type: http
    service: postgres
    endpoint: /health
    method: GET
    expect:
      status: 200
      
  - name: "Qdrant is accessible"
    type: http
    service: qdrant
    endpoint: /health
    method: GET
    expect:
      status: 200
      
  - name: "Ollama is accessible"
    type: http
    service: ollama
    endpoint: /api/health
    method: GET
    expect:
      status: 200
      
  # API endpoint tests:
  - name: "API health endpoint responds"
    type: http
    service: api
    endpoint: /health
    method: GET
    expect:
      status: 200
      body:
        status: "healthy"
        
  - name: "Character creation endpoint works"
    type: http
    service: api
    endpoint: /api/v1/characters
    method: POST
    body:
      name: "Test Character"
      personality_traits: {"brave": 0.8, "humorous": 0.6}
    expect:
      status: 201
      body:
        success: true
        character_id: "*"
        
  - name: "Dialog generation endpoint works"
    type: http
    service: api
    endpoint: /api/v1/dialog/generate
    method: POST
    body:
      character_id: "test-uuid"
      scene_context: "A peaceful forest clearing"
      emotion_state: "calm"
    expect:
      status: 200
      body:
        dialog: "*"
        
  # CLI command tests:
  - name: "CLI help command executes"
    type: exec
    command: ./cli/game-dialog-generator --help
    expect:
      exit_code: 0
      output_contains: ["Game Dialog Generator", "Commands:", "character-create"]
      
  - name: "CLI status command works"
    type: exec
    command: ./cli/game-dialog-generator status --json
    expect:
      exit_code: 0
      output_contains: ["healthy"]
      
  # Database tests:
  - name: "Database schema is initialized"
    type: sql
    service: postgres
    query: "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name IN ('characters', 'scenes', 'dialog_lines', 'projects')"
    expect:
      rows: 
        - count: 4
```

### Performance Validation
- [x] API response times meet SLA targets (< 2s for dialog generation)
- [ ] Resource usage within defined limits (< 2GB memory, < 50% CPU)
- [ ] Throughput meets minimum requirements (100+ dialogs/minute batch)
- [ ] No memory leaks detected over 24-hour test

### Integration Validation
- [x] Discoverable via resource registry
- [ ] All API endpoints documented and functional
- [ ] All CLI commands executable with --help
- [ ] Events published/consumed correctly

### Capability Verification
- [x] Solves game dialog generation problem completely
- [x] Integrates with required resources (Ollama, Qdrant, PostgreSQL)
- [x] Enables downstream game development capabilities
- [ ] Maintains character personality consistency
- [x] UI matches jungle platformer aesthetic expectations

## 📝 Implementation Notes

### Design Decisions
**Character Personality Modeling**: Vector embeddings + structured personality traits
- Alternative considered: Pure LLM prompt engineering without structured data
- Decision driver: Consistency and searchability require structured + semantic approach
- Trade-offs: More complex setup but significantly better character consistency

**No n8n Workflows**: Direct resource integration via CLI and APIs
- Alternative considered: n8n workflow orchestration
- Decision driver: Project direction moving away from n8n complexity
- Trade-offs: More direct code but less visual workflow management

**Jungle Platformer Theme**: Fun, game-inspired UI for game developers
- Alternative considered: Standard professional SaaS interface
- Decision driver: Target audience appreciates game aesthetics, differentiation from generic tools
- Trade-offs: More complex UI development but much better user engagement

### Known Limitations
- **Voice Quality**: Initial TTS integration may have robotic qualities
  - Workaround: Provide multiple voice models, quality sliders
  - Future fix: Integration with advanced TTS services in v2.0

- **Real-time Performance**: Complex personality modeling may cause latency
  - Workaround: Implement caching layer for common character/scene combinations
  - Future fix: Optimized embedding strategies and faster inference models

### Security Considerations
- **Data Protection**: Character data and dialog content treated as intellectual property
- **Access Control**: Project-based access control for multi-user environments
- **Audit Trail**: All dialog generation requests logged for debugging and analytics

## 🔗 References

### Documentation
- README.md - User-facing overview with jungle platformer theme introduction
- docs/api.md - Complete API specification with character modeling details
- docs/cli.md - CLI documentation with game development workflows
- docs/architecture.md - Technical deep-dive on personality modeling and dialog generation

### Related PRDs
- Link to interactive-fiction-studio PRD (when created)
- Link to game-master-ai PRD (when created)

### External Resources
- [Personality Psychology Research](https://example.com) - Big Five personality model implementation
- [Game Dialog Design Patterns](https://example.com) - Industry best practices for character dialog
- [Voice Synthesis Standards](https://example.com) - Technical requirements for game audio

---

**Last Updated**: 2025-09-07  
**Status**: Draft  
**Owner**: AI Agent (Claude Code)  
**Review Cycle**: Weekly during development, monthly post-deployment