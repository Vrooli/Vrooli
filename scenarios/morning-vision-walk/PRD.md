# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/prd-control-tower/docs/CANONICAL_PRD_TEMPLATE.md`
> **Validation**: Enforced by `prd-control-tower` + `scenario-auditor`
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview
- **Purpose**: AI companion for morning walks that transforms unstructured thoughts into actionable insights, helping brainstorm ideas, understand Vrooli's current state, and collaboratively plan the day ahead through natural voice interaction
- **Primary users/verticals**: Vrooli developers, knowledge workers, daily planners, reflective thinkers who prefer voice-first interaction during walks or commutes
- **Deployment surfaces**: CLI (conversation management and task export), API (conversation handling and context gathering), UI (mobile-optimized conversation interface with voice input)
- **Value promise**: Transform daily walks into productive brainstorming sessions with automatic task extraction, context-aware responses using conversation history, and seamless integration with task-planner and other Vrooli scenarios

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [ ] OT-P0-001 | Voice-first conversation interface | Handle voice input/output for hands-free brainstorming during walks with real-time transcription
- [ ] OT-P0-002 | Context-aware AI responses | Generate responses using Ollama with embeddings from past conversations via universal-rag-pipeline
- [ ] OT-P0-003 | Automatic task extraction | Parse conversations to identify and extract actionable tasks with priority estimation
- [ ] OT-P0-004 | Conversation storage and retrieval | Store conversations in PostgreSQL with searchable embeddings in Qdrant
- [ ] OT-P0-005 | Mobile-optimized UI | Clean, calming interface with nature-inspired colors optimized for mobile use during walks
- [ ] OT-P0-006 | CLI conversation management | Start conversations, review insights, search history, and export tasks via CLI

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-001 | Task-planner integration | Automatically export extracted tasks to task-planner with proper formatting and metadata
- [ ] OT-P1-002 | Insight breakthrough detection | Identify and highlight significant insights or breakthrough ideas during conversations
- [ ] OT-P1-003 | Daily planning workflow | Generate daily plans with priority-based task organization from conversation content
- [ ] OT-P1-004 | Vrooli context gathering | Pull current state of scenarios, resources, and active tasks to inform conversation responses
- [ ] OT-P1-005 | Stream-of-consciousness integration | Feed conversations to stream-of-consciousness-analyzer for deeper pattern analysis

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | Multi-day conversation threading | Track and reference conversation themes across multiple days
- [ ] OT-P2-002 | Mind-map visualization | Generate visual mind maps from conversation insights and connections
- [ ] OT-P2-003 | Idea-generator integration | Push novel ideas to idea-generator for further development
- [ ] OT-P2-004 | Offline voice processing | Enable voice capture and processing without network connectivity
- [ ] OT-P2-005 | Personalized reflection prompts | Learn user patterns and suggest targeted reflection questions

## 🧱 Tech Direction Snapshot
- Preferred stacks / frameworks: Go API (conversation handling and context gathering), React UI (mobile-first with nature-inspired design)
- Data + storage expectations: PostgreSQL (conversations, insights, tasks, and plans), Redis (current conversation state and real-time audio streaming), Qdrant (conversation embeddings for context retrieval), Ollama (llama3.2 for AI responses)
- Integration strategy: Direct API/CLI orchestration for AI capabilities (ollama, chain-of-thought-orchestrator, embedding-generator, universal-rag-pipeline), plus scenario integration with task-planner and stream-of-consciousness-analyzer
- Non-goals / guardrails: Not a general-purpose voice assistant (focused on walks and planning), not a replacement for task management tools (complements task-planner), no complex UI interactions (keep it simple for outdoor use)

## 🤝 Dependencies & Launch Plan
- Required resources: ollama (llama3.2 for conversation), postgres (data storage), qdrant (embeddings), redis (session state), chain-of-thought-orchestrator (reasoning), embedding-generator (vectors), universal-rag-pipeline (context retrieval)
- Scenario dependencies: task-planner (task export), stream-of-consciousness-analyzer (insight analysis), mind-maps (visualization - P2), idea-generator (idea development - P2)
- Operational risks: Voice transcription accuracy in outdoor environments, Ollama availability for real-time responses, mobile network connectivity during walks, conversation context window limits
- Launch sequencing: Phase 1 - Deploy basic voice conversation with task extraction (2 weeks), Phase 2 - Add task-planner integration and daily planning (1 week), Phase 3 - Implement breakthrough detection and advanced integrations (ongoing)

## 🎨 UX & Branding
- Look & feel: Clean, calming interface with nature-inspired color palette (soft greens, earthy browns, sky blues), large touch targets for outdoor use, minimalist design to reduce cognitive load during reflection
- Accessibility: High contrast text for outdoor visibility, voice-first interaction with visual feedback, screen reader support for all features, haptic feedback for key actions, works well in bright sunlight
- Voice & messaging: Thoughtful, reflective, encouraging tone - like a wise companion who listens deeply and asks insightful questions. Concise responses suitable for audio playback while walking
- Branding hooks: 🌅 Morning vision theme, 🚶 walk-centric workflow, 💡 insight highlighting with visual sparkle effect, 🎯 seamless task extraction with gentle confirmation

## 📎 Appendix (optional)

### Success Metrics
- **Engagement**: 5+ conversations per week for active users
- **Value**: 3+ actionable tasks extracted per conversation
- **Quality**: 80%+ user satisfaction with AI responses
- **Integration**: 90%+ successful task exports to task-planner
- **Performance**: <2s response latency for voice interactions

### Voice Interaction Pattern
```
User: (walking, voice input) "I've been thinking about how we could improve the automation workflows..."
System: (audio response) "That's an interesting direction. What specific workflow challenges are you encountering?"
User: "The error handling feels brittle, and we don't have good retry logic..."
System: (audio) "I hear two themes - reliability and resilience. Should I create a task for improving automation error handling?"
User: "Yes, make it high priority"
System: (audio + visual confirmation) "✓ Task created: Improve automation error handling and retry logic - High Priority"
```

### Reference Documentation
- CLI usage patterns in README.md
- Voice UI interaction patterns in ui/src/components/
