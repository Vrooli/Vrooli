# Product Requirements Document (PRD)

> **Version**: 2.0.0
> **Canonical Template**: v2.0.0
> **Status**: Published

## 🎯 Overview

The Game Dialog Generator adds AI-powered character dialog generation for video games and interactive media. This capability enables context-aware dialog generation based on character personality, relationships, and scene context with real-time dynamic dialog for adaptive gameplay experiences.

**Purpose**: Provides AI-powered character dialog generation with personality consistency, semantic search, and optional voice synthesis as a reusable capability for game development and interactive storytelling scenarios.

**Primary Users**: Independent game developers, narrative designers, game studios, interactive fiction creators

**Deployment Surfaces**:
- CLI for scripting and batch operations
- REST API for integration with game engines and other scenarios
- Web UI for character creation, dialog management, and project export

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability

- [ ] OT-P0-001 | Character personality definition and storage system | Create and store character personalities with traits, backgrounds, speech patterns, and relationships in PostgreSQL
- [ ] OT-P0-002 | Scene context management with semantic search via Qdrant | Manage scene contexts and enable semantic search using Qdrant vector embeddings
- [ ] OT-P0-003 | Real-time dialog generation API with streaming responses | Generate character dialog in real-time using Ollama with < 2s response time
- [ ] OT-P0-004 | Batch dialog generation for traditional game workflows | Support batch generation of multiple dialog lines for fixed-dialog games
- [ ] OT-P0-005 | PostgreSQL integration for persistent character and project data | Store characters, scenes, projects, and dialog history in PostgreSQL
- [ ] OT-P0-006 | Ollama integration for local LLM-powered dialog generation | Use Ollama for AI-powered dialog generation with personality constraints

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | Voice synthesis with character-specific audio profiles | Generate character-specific voice audio for dialog lines
- [ ] OT-P1-002 | Character relationship modeling affecting dialog dynamics | Model character relationships that influence dialog tone and content
- [ ] OT-P1-003 | Dialog history tracking and consistency checks | Track dialog history and validate consistency with character personality
- [ ] OT-P1-004 | Export formats for popular game engines | Export dialog in formats compatible with Unity, Unreal, Godot, and JSON
- [ ] OT-P1-005 | Character emotion state tracking influencing dialog tone | Track and incorporate character emotion states in dialog generation
- [ ] OT-P1-006 | Dialog validation against character personality constraints | Validate generated dialog matches character personality profiles

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | Visual character relationship graphs | Display character relationships as interactive graphs
- [ ] OT-P2-002 | Dialog analytics and quality metrics | Provide analytics on dialog quality, character consistency, and usage patterns
- [ ] OT-P2-003 | Multi-language dialog generation | Generate dialog in multiple languages with cultural context
- [ ] OT-P2-004 | Integration with existing game dialog systems | Connect to existing game dialog systems for enhanced generation
- [ ] OT-P2-005 | Collaborative character development features | Enable team collaboration on character creation and dialog review
- [ ] OT-P2-006 | Dialog A/B testing capabilities | Support A/B testing of different dialog variations

## 🧱 Tech Direction Snapshot

**Preferred Stack**:
- API: Go with net/http, PostgreSQL via database/sql, Qdrant Go client
- UI: React + Vite with TypeScript, jungle platformer theme
- CLI: Go binary with command structure for character and dialog management

**Data Storage**:
- PostgreSQL for structured data (characters, scenes, projects, dialog history)
- Qdrant for vector embeddings (character traits, scene context, semantic search)
- Local filesystem for generated audio files (P1 feature)

**Integration Strategy**:
- Direct resource access via CLI commands (resource-ollama, resource-qdrant)
- Direct API calls for performance-critical operations (real-time dialog generation)
- No n8n workflows per project direction

**Non-Goals**:
- Not a general-purpose creative writing tool (focused on game dialog)
- Not a game engine (provides dialog generation as a service)
- Not a multi-modal AI platform (focused on text and voice for characters)

## 🤝 Dependencies & Launch Plan

**Required Local Resources**:
- **postgres**: Character definitions, dialog history, project data storage
- **qdrant**: Vector embeddings for semantic search and character trait matching
- **ollama**: LLM inference for AI-powered dialog generation with personality constraints

**Optional Resources**:
- **whisper**: Voice synthesis for character-specific audio (P1 feature, graceful degradation to text-only)

**Scenario Dependencies**: None (standalone capability)

**Launch Sequencing**:
1. P0 features: Core character and dialog management with Ollama integration
2. P1 features: Voice synthesis, relationship modeling, game engine exports
3. P2 features: Analytics, multi-language, collaborative features

**Risks**:
- Ollama service availability impacts dialog generation (mitigation: cached responses, clear error messages)
- Character consistency may degrade with complex personalities (mitigation: personality validation scoring)
- Voice synthesis quality may vary (mitigation: multiple TTS backends, quality controls)

## 🎨 UX & Branding

**Visual Style**: Jungle platformer theme inspired by classic games like Donkey Kong Country, featuring rich jungle greens, warm earth tones, and bright accent colors. Card-based character management with jungle vine decorative elements and bouncy organic animations.

**Color Palette**:
- Primary: #2F5233, #4A7C59, #7BA35C, #B8D67A
- Accent: #FF6B35, #F7931E, #FFD23F
- Backgrounds: Layered jungle scenery with parallax scrolling effects

**Typography**: Playful but readable mix of adventure game fonts with modern UI text for professional game development workflows.

**UI Elements**:
- Navigation: Vine-style breadcrumbs that swing on hover
- Buttons: Wooden signs with carved text, stone tablets for important actions
- Character cards: Framed like retro platformer character select screens
- Dialog bubbles: Leaf-shaped speech bubbles with organic borders
- Loading states: Swinging monkey animations, growing plant progress bars

**Tone & Personality**: Playful and adventurous but professional for game developers. Energetic exploration of creative possibilities while maintaining serious game development functionality.

**Accessibility**: WCAG 2.1 AA compliance with high contrast options for readability. Desktop-first design for game development workflow, tablet-friendly for creative sessions.

**Brand Positioning**: Adventure-themed game development tool that sparks creativity while providing professional-grade AI dialog generation. Balances whimsical jungle aesthetics with technical excellence.

## 📎 Appendix

**Technical Architecture Details**

**Resource Integration Patterns**:
```yaml
integration_priorities:
  1_resource_cli:
    - command: resource-ollama generate --model llama3.2
      purpose: Dialog generation with personality constraints
    - command: resource-qdrant search --collection characters
      purpose: Character and scene context retrieval

  2_direct_api:
    - justification: Real-time dialog streaming requires direct HTTP connections
      endpoint: /v1/chat/completions (Ollama), /collections/search (Qdrant)
```

**Data Models**:
- Character: UUID, name, personality_traits (jsonb), background_story, speech_patterns, relationships, voice_profile
- Scene: UUID, project_id, name, context, setting, mood, participants (UUID array)
- DialogLine: UUID, character_id, scene_id, content, emotion, context_embedding (Qdrant), audio_file_path
- Project: UUID, name, description, characters (UUID array), settings, export_format

**API Endpoints**:
- POST /api/v1/dialog/generate - Real-time dialog generation (< 2s SLA)
- POST /api/v1/dialog/batch - Batch dialog generation (< 30s SLA)
- GET /api/v1/characters/{id}/personality - Character personality retrieval
- POST /api/v1/characters - Character creation
- POST /api/v1/projects/{id}/export - Project export for game engines

**CLI Commands**:
- game-dialog-generator status - Show operational status
- game-dialog-generator help - Command help
- game-dialog-generator version - Version information
- game-dialog-generator character-create - Create character with personality
- game-dialog-generator dialog-generate - Generate dialog for character
- game-dialog-generator project-export - Export project dialog

**Performance Targets**:
- Real-time dialog response: < 2s for 95% of requests
- Batch dialog throughput: 100+ dialogs/minute
- Character consistency: > 85% personality adherence
- Resource usage: < 2GB memory, < 50% CPU
- Voice generation speed: < 5s per dialog line (P1)

**Business Context**

**Value Proposition**:
- Eliminates dialog writing bottleneck in game development (20-40% of narrative work)
- Revenue potential: $25K - $75K per deployment for game studios
- Cost savings: Reduces dialog writing time from weeks to hours
- Market differentiator: Local AI-powered dialog generator with character consistency

**Revenue Model**:
- Indie tier: $29/month (5 projects, 100 characters)
- Studio tier: $99/month (unlimited projects, 1000 characters)
- Enterprise tier: $299/month (unlimited, voice synthesis, API access)
- Trial period: 14 days

**Evolution Path**:
- v1.0: Core character and dialog management, Ollama integration, jungle-themed UI
- v2.0: Voice synthesis, relationship modeling, game engine integrations, analytics
- Long-term: Multi-modal character AI, real-time game engine integration, community templates

**Capability Discovery**

**Registry Entry**:
- Name: game-dialog-generator
- Category: generation
- Capabilities: dialog-generation, character-ai, voice-synthesis, game-development
- Interfaces: API (http://localhost:{API_PORT}/api/v1), CLI (game-dialog-generator), Events (dialog.*)
- Keywords: game-development, dialog, character-ai, narrative, voice-synthesis, storytelling
- Dependencies: ollama, qdrant, postgres
- Enhances: interactive-fiction, game-engines, content-creation

**Related Documentation**

- README.md - User-facing overview with jungle platformer theme introduction
- docs/api.md - Complete API specification with character modeling details
- docs/cli.md - CLI documentation with game development workflows
- docs/architecture.md - Technical deep-dive on personality modeling and dialog generation
