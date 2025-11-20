# Product Requirements Document (PRD)

> **Version**: 2.0.0
> **Last Updated**: 2025-11-18
> **Status**: Production
> **Canonical Reference**: PRD Control Tower

## 🎯 Overview

**Purpose**: Bedtime Story Generator delivers an immersive, AI-powered bedtime story experience combining autonomous story generation with a cinematic 3D WebGL environment. The scenario demonstrates production-quality integration of LLM-based content generation with interactive 3D visualization.

**Primary Users**:
- Parents seeking engaging, age-appropriate bedtime content for children (ages 6-12)
- Children exploring interactive story experiences
- Developers learning WebGL/Three.js integration patterns

**Deployment Surfaces**:
- React UI with 3D WebGL scene (primary interface)
- RESTful API for story CRUD operations
- CLI for database management and story generation
- PostgreSQL for persistence

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability

- [x] OT-P0-001 | Story generation with age-appropriate content | Generate stories for 6-8 and 9-12 age groups with content filtering
- [x] OT-P0-002 | Database persistence | Store and retrieve stories, reading history, and favorites via PostgreSQL
- [x] OT-P0-003 | RESTful API with health checks | Provide API endpoints for story operations with service health monitoring
- [x] OT-P0-004 | React UI for browsing and reading | Deliver functional UI for story interaction at dynamically assigned port
- [x] OT-P0-005 | Theme selection | Support 10+ kid-friendly themes (Adventure, Animals, Magic, Space, Ocean, etc.)
- [x] OT-P0-006 | Reading history tracking | Track reading sessions via /api/v1/stories/{id}/read endpoint
- [x] OT-P0-007 | Performance targets | Achieve story generation in ~2-3s with local LLM (target: <8s)
- [x] OT-P0-008 | Basic 3D environment | Render cinematic bedroom scene with Three.js integration

### 🟠 P1 – Should have post-launch

- [x] OT-P1-001 | Character name customization | Accept and integrate custom character names in generated stories
- [x] OT-P1-002 | Reading time estimates | Calculate and display reading_time_minutes for each story
- [x] OT-P1-003 | Favorite stories feature | Toggle and persist favorite status via /favorite endpoint
- [x] OT-P1-004 | Story length options | Support short/medium/long length parameters for story generation
- [x] OT-P1-005 | Interactive 3D bookshelf | Render 3-level bookshelf with 15 interactive books and animations
- [x] OT-P1-006 | Time-of-day transitions | Implement day/evening/night lighting with smooth fog and color transitions
- [x] OT-P1-007 | Dynamic particle system | Display 100+ floating magical particles with theme-based animations
- [x] OT-P1-008 | Camera rail system | Provide cinematic camera presets (intro, bookshelf, window, story orbit, lamp zoom)
- [x] OT-P1-009 | Spatial audio ambience | Deliver 3D audio with time-of-day and story mood adaptations
- [x] OT-P1-010 | Performance dashboard | Display real-time FPS, draw calls, triangles, memory in developer console

### 🟢 P2 – Future / expansion

- [x] OT-P2-001 | Emoji-based illustrations | Generate and display emoji art illustrations for stories
- [x] OT-P2-002 | Parent dashboard | Integrate parent controls and metrics in UI
- [x] OT-P2-003 | PDF export functionality | Export stories to PDF via /export endpoint
- [x] OT-P2-004 | Text-to-speech | Implement Web Speech API for story narration
- [x] OT-P2-005 | Parent narration mode | Stream text-to-speech into environment with lamp synchronization
- [x] OT-P2-006 | Procedural star map | Generate dynamic constellations based on reading history and story themes
- [x] OT-P2-007 | Photo-mode exports | Capture marketing screenshots at multi-resolution (up to 6K) with presets
- [x] OT-P2-008 | Asset kit generator | Export Blender starter scenes (bedroom, classroom, office, outdoor) with Python scripts
- [ ] OT-P2-009 | Advanced AI illustration generation | Generate custom illustrations beyond emoji art

## 🧱 Tech Direction Snapshot

**UI Stack**:
- React with Three.js/WebGL for 3D rendering
- Vite for build tooling
- Experience engine pattern (Bruno-inspired) for scene management
- Zustand or similar for state bridging between React and 3D scene

**API Stack**:
- Go with database/sql for PostgreSQL integration
- RESTful endpoints with JSON responses
- Worker pool pattern for LLM invocation

**Data Storage**:
- PostgreSQL for stories, reading sessions, favorites, developer presets
- Migration-based schema management
- Optional Redis for metadata caching and asset prefetch hints

**Integration Strategy**:
- Ollama CLI driver invoked by API worker pool for story generation
- Asset pipeline: Blender → glTF with baked textures and lightmaps
- DOM/3D state sync via shared store with accessibility fallbacks

**Non-Goals**:
- Real-time multiplayer story collaboration
- Mobile native apps (web-first, responsive design)
- Video streaming or live animation generation

## 🤝 Dependencies & Launch Plan

**Required Local Resources**:
- `postgres` – Persist stories, reading history, favorites, developer presets
- `ollama` – Generate bedtime stories with strict prompt guardrails

**Optional Resources**:
- `redis` – Cache story metadata for HUD animations, store asset prefetch hints
- `asset-pipeline` – Blender export toolchain for glTF with baked textures

**Scenario Dependencies**:
- None (standalone scenario)

**Launch Risks**:
- Asset scope creep → Mitigation: Lock asset list early, enforce change control
- Performance regression on low-end GPUs → Mitigation: Budget per module, enable automatic LOD
- Drift between 3D scene state and DOM fallback → Mitigation: Shared store with unit tests ensuring parity
- Large bundle size due to textures → Mitigation: Use KTX2 compression, lazy-load groups, leverage basis loader

**Launch Sequencing**:
1. Core story generation API validated
2. React UI with basic 3D scene functional
3. Performance budgets met on target hardware
4. Accessibility review completed (keyboard navigation, screen reader modes)
5. Visual regression suite passing for key camera setups
6. Demo videos captured for day/evening/night modes

## 🎨 UX & Branding

**Visual Palette**:
- Warm, cozy bedroom aesthetic with soft lighting
- Day/evening/night color schemes with smooth transitions
- Magical particle effects with theme-based color coding
- Volumetric lighting effects (god rays from window, ambient dust particles)

**Typography & Motion**:
- Kid-friendly fonts for story text
- Gentle, cinematic camera movements on rail system
- Smooth transitions between time-of-day states
- Subtle animations on interactive elements (books, lamp, toy chest)

**Accessibility Commitments**:
- WCAG 2.1 AA compliance for DOM fallbacks
- Keyboard navigation for all interactive hotspots
- Screen reader-friendly content mirrors for 3D elements
- Focus management between 3D scene and DOM panels
- Performance guardrails for low-spec devices (automatic LOD, particle reduction)

**Voice & Personality**:
- Warm, encouraging tone in UI copy
- Age-appropriate language matching story content
- Playful but not condescending
- Parent-friendly controls and transparency

**Interaction Patterns**:
- Diegetic UI elements (story projector, bookshelf, lamp) as primary interactions
- Developer console rendered in-world with Tweakpane-style controls
- Hot-reload asset system (Ctrl+Shift+R) for rapid iteration
- Photo-mode with marketing prompt generator for showcase materials

## 📎 Appendix

**References**:
- Three.js documentation: https://threejs.org/docs/
- Ollama API reference: https://github.com/ollama/ollama/blob/main/docs/api.md
- WCAG 2.1 AA guidelines: https://www.w3.org/WAI/WCAG21/quickref/

---

**Owner**: Vrooli Bedtime Pod
**Review Cadence**: Monthly for maintenance
**Last Validated**: 2025-09-27 (Twenty-Second Pass)
