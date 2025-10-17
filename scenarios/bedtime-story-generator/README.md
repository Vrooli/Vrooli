# 🌙 Bedtime Story Generator – Immersive Edition

Bedtime Story Generator is evolving from a “storybook app” into a Bruno-level, cinematic bedtime suite. We now pair age-aware storytelling with a hand-crafted 3D bedroom that breathes, glows, and reacts to every tale you spin. Think curated glTF assets, baked lighting blends, diegetic HUDs, and developer tooling that lives inside the scene—without sacrificing the production-grade API, CLI, and database backbone we already shipped.

## ✨ What’s New & Why It Matters
- **Cinematic 3D Children’s Room** – Custom Experience engine (Three.js + React) renders a light-baked bedroom with day/evening/night palettes, dynamic particles, and camera choreography inspired by Bruno Simon’s “My Room in 3D.”
- **Diegetic Story Surfaces** – The bookshelf hotspot summons a holographic HUD; a projector streams story previews; the reader unfolds as an open book on the bed while DOM fallbacks keep everything accessible.
- **Living Developer Console** – Developer mode becomes a floating Tweakpane-style console inside the room, synchronized with a DOM mirror for quick tweaks, preset management, and performance metrics.
- **State-Driven Atmosphere** – Story metadata drives shader accents, ambient audio, sparkles, and screen content so the room reflects the tale being told.
- **Reusable Engine Blueprint** – The new `/ui/src/three` Experience layer documents an asset + renderer workflow that future scenarios can clone to reach this level of polish fast.

## 🌟 Core Feature Set (vNext)
- Cinematic room with baked lightmaps and HDR blends (day/evening/night).
- Hotspot navigation + camera rail that focuses on bookshelf, bed stage, window, and ceiling mobile.
- Story generator & reader embedded inside the environment, mirrored in React HUD and CLI.
- Developer/debug holographic console with preset saving, stats, and asset toggles.
- Asset loader & manifest supporting GLB, KTX2 textures, HDR skies, video loops.
- Accessibility coverage: keyboard navigation, SR-friendly DOM, reduced-motion mode, low-spec fallback.

## 🧠 Purpose & Value
This scenario still anchors Vrooli’s bedtime storytelling capability, but now it also teaches agents how to:
- Build unforgettable 3D experiences tied to business value.
- Operate a modern 3D asset pipeline (Blender ➜ glTF ➜ engine).
- Present operational tooling (developer modes, analytics) in-world without losing clarity or accessibility.
- Manage performance budgets while layering AI-generated content into a real-time environment.

## 🚀 Getting Started
```bash
# 1. Start the scenario (manages API, UI, resources)
vrooli scenario run bedtime-story-generator

# 2. Visit the immersive room (port shown in logs)
# Check status to see assigned ports:
vrooli scenario status bedtime-story-generator

# 3. Generate a story via CLI (still works!)
bedtime-story generate --age-group "6-8" --theme "Adventure"
```

> ⚠️ The immersive UI refactor is in-flight. You can run the current app today; as we land each phase, the Experience engine will progressively replace the existing React room.

Inside the UI you can launch the **Immersive Prototype** panel to preview the new Experience engine and watch the room react to live story data while we build out the full environment.

### Local Asset Pipeline (coming online during Phase 1)
```bash
# Export Blender scene ➜ glTF with baked textures
npm run assets:export

# Compress textures to KTX2 / Basis
npm run assets:textures

# Validate manifests & hot reload into the Experience engine
npm run assets:validate
```

Current manifest lives at `ui/src/assets/manifest.js`. The prototype points to `/prototype/...` files—swap these with real exports as soon as the blockout lands.

## 🛠️ Technical Stack
- **API**: Go (REST) with PostgreSQL persistence, Ollama-powered story generation.
- **UI Shell**: React + Zustand shared store bridging DOM panels and Experience engine.
- **Experience Engine**: Three.js, EffectComposer, custom renderer/navigator modeled after Bruno’s architecture.
- **Asset Pipeline**: Blender templates, gltfpack/meshoptimizer, basisu (KTX2 textures), HDR environment maps.
- **Testing**: Go integration suite, Jest/Vitest for HUD + utilities, Playwright visual snapshots, Cypress hotspot smoke tests.

## 🔄 Architecture at a Glance
```
React Shell ──┐
             ├─ Zustand Store ── Story Data, Developer State, Time-of-Day
Experience    │       ▲
 Engine <─────┘       │
  ├─ Renderer (Three.js + PostFX)
  ├─ CameraRig (spherical nav, cinematic rails)
  ├─ Resources (assets manifest, GLB/texture loaders)
  ├─ World Modules (Bookshelf, Bed Stage, Projector, Developer Console)
  └─ Hotspot Controller (hover, activate, analytics)
```

## ✅ Current Capabilities (still live)
- API endpoints for story CRUD/generation (`/api/v1/stories`, `/generate`, `/favorite`, `/read`).
- CLI wrapper (`bedtime-story`) for generation, listing, reading, favorites.
- PostgreSQL schema for stories, reading sessions, user preferences.
- Kids-dashboard integration via `kid-friendly` tag.
- Basic React UI (will be replaced, but remains functional during the transition).

## 📅 Roadmap Snapshot
| Phase | Focus | Highlights |
|-------|-------|------------|
| **1 – Foundations** | Asset prep & engine scaffolding | `/ui/src/three` Experience skeleton, manifests, loader progress HUD |
| **2 – World Build** | Import GLBs, lighting blends, hotspots | Cinematic camera rails, bookshelf HUD, diegetic reader/generator |
| **3 – Developer Mode** | Holographic console, perf dashboards | Preset save/load, particle/audio binding to story metadata |
| **4 – Polish & Launch** | Accessibility, QA, documentation | Visual regression suite, marketing captures, asset pipeline docs |

Track progress and open issues in `PROBLEMS.md` and the scenario Kanban (link forthcoming).

## 🧪 Testing & Quality
- `make test` – Runs Go API tests and CLI BATS suite.
- `npm test` (UI) – Jest/Vitest suite for HUD/Experience utilities.
- `npm run test:e2e` – Cypress hotspot interaction smoke (Phase 2 target).
- `npm run test:visual` – Playwright visual snapshots (Phase 3 target).

## 🤝 Integrations
- **Provides**: Kids dashboard story shelf, Education tracker reading metrics, Text-to-speech scenario content feed.
- **Consumes**: Ollama LLMs, Postgres storage, optional Redis cache for story metadata + asset hints.

## 🛡️ Safety & Accessibility
- Prompt guardrails ensure stories stay age-appropriate.
- No external data collection—everything persists locally.
- Keyboard + SR-friendly fallback mirrored for every diegetic control.
- Low-spec toggle disables heavy post-processing and particle FX.

## 🔮 Future Enhancements
- Parent narration mode (radio/lamp audio playback with generated speech).
- Procedural star map reflecting nightly reading streaks.
- Photo-mode + shareable story posters captured from the cinematic camera rig.
- Asset kit export so other scenarios can bootstrap immersive rooms quickly.

---
**Part of the Vrooli Ecosystem** – Building permanent intelligence, one breathtaking bedtime experience at a time. 🌌📖
