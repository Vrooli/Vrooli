# 3D World Architecture

This document describes the architecture of the 3D world visualization system in the prompt-manager UI, built with React Three Fiber.

---

## Technology Stack

```
┌─────────────────────────────────────────────┐
│            React Three Fiber                │
│         (React renderer for Three.js)       │
├─────────────────────────────────────────────┤
│               Three.js                      │
│         (Base 3D rendering engine)          │
├─────────────────────────────────────────────┤
│  @react-three/drei    @react-three/postproc │
│  (Helpers/presets)    (Effects pipeline)    │
└─────────────────────────────────────────────┘
```

**Package Versions:**
- `three` (v0.151.3) - Base 3D rendering engine
- `@react-three/fiber` (v8.15) - React renderer for Three.js
- `@react-three/drei` (v9.88) - Reusable Three.js components and helpers
- `@react-three/postprocessing` (v3.0.4) - Post-processing effects pipeline

---

## Directory Structure

```
ui/src/
├── components/world/
│   ├── WorldCanvas.tsx          # Main entry point (canvas + state orchestration)
│   ├── WorldScene.tsx           # 3D scene composition (lights, controls, agents)
│   ├── AgentProvider.tsx        # Dependency injection for swappable agent types
│   │
│   ├── agents/
│   │   ├── SlimeAgent.tsx             # Cute blob creature with jelly shader
│   │   └── AgentWithAccessories.tsx   # Wraps agent + accessories + overlays
│   │
│   ├── accessories/
│   │   ├── BackpackAccessory.tsx   # Auto-sized by skill count (paper->backpack)
│   │   ├── HeadAccessory.tsx       # Hat, glasses, crown, etc.
│   │   └── HeldItemAccessory.tsx   # Book, orb, wand, etc.
│   │
│   ├── overlays/
│   │   ├── AgentOverlayGroup.tsx  # Stacks 2D UI in 3D space
│   │   ├── NameTag.tsx
│   │   ├── StatusIcon.tsx
│   │   └── SpeechBubble.tsx
│   │
│   ├── materials/
│   │   ├── MaterialProvider.tsx    # Caches & manages material instances
│   │   └── presets.ts              # metallic, matte, plastic, emissive...
│   │
│   └── rendering/
│       ├── RenderPipeline.tsx      # Post-processing (bloom, vignette, SMAA)
│       └── EnvironmentSetup.tsx    # HDR environment + contact shadows
│       ├── GroundSurface.tsx       # Grid/plane ground composition
│       └── GroundMaterial.tsx      # Textured ground materials + shader wiring
│
├── lib/shaders/
│   ├── glsl/
│   │   ├── ground.glsl.ts         # Ground texture shader GLSL
│   │   ├── sky.glsl.ts            # Sky shader GLSL
│   │   └── slime.glsl.ts          # Slime wobble shader GLSL (simplex noise)
│   ├── groundShader.ts            # Ground shader binding/sync
│   └── slimeShader.ts             # Slime shader binding/sync
│
├── stores/                  # Zustand state management
│   ├── cameraStore.ts       # Position, zoom, modes (freeform/zoomed/top-down)
│   ├── graphicsStore.ts     # Performance tiers (low->ultra)
│   ├── environmentStore.ts  # HDR presets, time of day
│   ├── accessoryStore.ts    # Per-agent accessories
│   └── interactionStore.ts  # Hover/drag state
│
├── hooks/
│   ├── useHoverHighlight.ts    # Hover effects on 3D objects
│   ├── useGraphicsSettings.ts  # Access/modify graphics tier
│   └── useDeviceCapability.ts  # Auto-detect GPU tier
│
├── types/
│   ├── world.ts       # AgentProps, CameraState
│   ├── agent.ts       # Agent interface, colors
│   ├── graphics.ts    # PerformanceTier, GraphicsConfig
│   ├── accessory.ts   # Accessory types, backpack thresholds
│   └── environment.ts # TimeOfDay, SceneType
│
└── config/
    ├── graphics.ts      # PERFORMANCE_TIERS configs
    └── environments.ts  # Lighting/fog presets
```

---

## Component Hierarchy

```
WorldCanvas
│
├── Canvas (R3F)
│   ├── RenderPipeline ─────────────► EffectComposer
│   │                                  ├── Bloom
│   │                                  ├── Vignette
│   │                                  └── SMAA
│   │
│   ├── EnvironmentSetup ───────────► HDR Environment
│   │                                 └── Contact Shadows
│   │
│   ├── MaterialProvider ───────────► Material cache (Map<key, Material>)
│   │
│   └── AgentProvider ────────────► WorldScene
│                                      │
│       ┌──────────────────────────────┘
│       │
│       ├── Lighting
│       │   ├── ambientLight
│       │   ├── directionalLight (shadow caster)
│       │   └── pointLight
│       │
│       ├── Environment
│       │   ├── Stars (dark mode)
│       │   ├── Fog
│       │   └── GroundSurface (grid or textured plane)
│       │
│       ├── OrbitControls
│       │
│       └── Agents (loop) ─────────► AgentWithAccessories
│                                      │
│           ┌──────────────────────────┘
│           │
│           ├── SlimeAgent ──────► Body (sphere + MeshPhysicalMaterial)
│           │                      Eyes (spheres with tracking pupils)
│           │                      Mouth (torus arc / box variants)
│           │                      Ear nubs (optional, per-agent)
│           │                      Blush marks (optional, per-agent)
│           │                      Floating orbs (when selected)
│           │                      Slime shader (vertex wobble)
│           │
│           ├── BackpackAccessory ──► paper/folder/briefcase/backpack
│           ├── HeadAccessory ──────► hat/glasses/crown/halo
│           ├── HeldItemAccessory ──► book/tool/orb/wand
│           │
│           └── AgentOverlayGroup ─► NameTag (y:1.0)
│                                     StatusIcon (y:1.3)
│                                     ThinkingBubble (y:1.5)
│                                     SpeechBubble (y:1.7)
│
└── UI Overlays (HTML)
    ├── WorldControls
    ├── CombinePanel
    └── AgentOverlay
```

---

## Ground Texturing Pipeline

Ground rendering is split into two responsibilities:

- `GroundSurface` decides **what** to render (grid vs plane) based on environment config.
- `GroundMaterial` wires **how** it renders by applying procedural texture sets and shader projection via `ui/src/lib/groundTextures.ts` + `ui/src/lib/groundShader.ts`.

This keeps world composition free of shader/material concerns and makes texture tuning a local change.

### Texture Generation Flow

```
┌─────────────────────────────────────────────────────────────────────┐
│                    TEXTURE GENERATION PIPELINE                       │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌──────────────┐    ┌─────────────────┐    ┌──────────────────┐   │
│  │   PRESETS    │───▶│  FBM NOISE GEN  │───▶│  DataTexture     │   │
│  │              │    │  (Procedural)   │    │  (512×512)       │   │
│  │ • grass      │    │                 │    │                   │   │
│  │ • concrete   │    │  Multi-octave   │    │  RGBA pixels     │   │
│  │ • wood-plank │    │  Perlin noise   │    │  stored in GPU   │   │
│  │ • stone      │    │                 │    │                   │   │
│  │ • metal-panel│    └─────────────────┘    └──────────────────┘   │
│  └──────────────┘                                                   │
└─────────────────────────────────────────────────────────────────────┘
```

### Five Texture Maps Per Surface

Each ground texture generates **5 maps** that work together:

```
┌─────────────────────────────────────────────────────────────────────┐
│                     TEXTURE SET (e.g., "grass")                     │
├─────────────┬─────────────┬─────────────┬─────────────┬────────────┤
│   ALBEDO    │   NORMAL    │  ROUGHNESS  │     AO      │   MACRO    │
│  (color)    │  (bumps)    │  (shine)    │  (shadows)  │ (variation)│
│             │             │             │             │            │
│  512×512    │   512×512   │   512×512   │   512×512   │  256×256   │
│  sRGB       │  Linear     │  Linear     │  Linear     │  Linear    │
└─────────────┴─────────────┴─────────────┴─────────────┴────────────┘
```

### Rendering Pipeline

```
┌─────────────────────────────────────────────────────────────────────┐
│                        RENDERING FLOW                                │
│                                                                      │
│  ┌─────────────────┐                                                │
│  │ GroundSurface   │  Decides: grid, plane, or none                 │
│  └────────┬────────┘                                                │
│           ▼                                                          │
│  ┌─────────────────┐                                                │
│  │ GroundMaterial  │  Creates MeshStandardMaterial + custom shader  │
│  └────────┬────────┘                                                │
│           ▼                                                          │
│  ┌─────────────────────────────────────────────────────────┐        │
│  │  bindGroundShader() + syncGroundShader()                │        │
│  │  ┌─────────────────────────────────────────────────┐    │        │
│  │  │  • Triplanar projection (no stretching)         │    │        │
│  │  │  • Macro variation overlay (breaks repetition)  │    │        │
│  │  │  • Stochastic tiling (random rotation per tile) │    │        │
│  │  └─────────────────────────────────────────────────┘    │        │
│  └─────────────────────────────────────────────────────────┘        │
└─────────────────────────────────────────────────────────────────────┘
```

### Anti-Tiling Techniques

The system uses multiple techniques to eliminate visible texture repetition:

```
┌─────────────────────────────────────────────────────────────────────┐
│                     ANTI-TILING TECHNIQUES                          │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  1. MACRO VARIATION OVERLAY                                         │
│     ─────────────────────────                                       │
│     Large-scale FBM noise texture (256×256) applied as tint         │
│     multiplier to break up uniform coloring.                        │
│                                                                      │
│     Config: macroVariation.enabled, .scale, .intensity              │
│                                                                      │
│  2. STOCHASTIC TILING                                               │
│     ────────────────────                                            │
│     Per-tile random 90° rotation using hash function.               │
│     Eliminates visible pattern repetition.                          │
│                                                                      │
│     ┌─────┬─────┬─────┐    Each tile gets random rotation:         │
│     │  0° │ 90° │180° │    hash22(tileId) → rotation angle          │
│     ├─────┼─────┼─────┤                                             │
│     │270° │  0° │ 90° │    Result: No visible tiling artifacts      │
│     └─────┴─────┴─────┘                                             │
│                                                                      │
│     Config: stochasticEnabled (default: true)                       │
│                                                                      │
│  3. TRIPLANAR PROJECTION                                            │
│     ──────────────────────                                          │
│     Samples texture from 3 orthogonal planes (X, Y, Z)              │
│     and blends based on surface normal. Eliminates                  │
│     stretching on vertical surfaces.                                │
│                                                                      │
│     Config: projection: 'triplanar' | 'uv'                          │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `id` | GroundTextureId | required | grass, concrete, wood-plank, stone, metal-panel |
| `tileSize` | number | 4 | World units per texture tile |
| `rotation` | number | 0 | UV rotation in radians |
| `projection` | 'uv' \| 'triplanar' | 'uv' | Texture projection method |
| `normalScale` | number | 0.6 | Normal map intensity |
| `stochasticEnabled` | boolean | true | Hash-based anti-tiling |
| `macroVariation.enabled` | boolean | true | Large-scale color variation |
| `macroVariation.scale` | number | 20 | World units per macro tile |
| `macroVariation.intensity` | number | 0.15 | Variation strength (0-1) |

### Key Files

| File | Purpose |
|------|---------|
| `lib/groundTextures.ts` | Procedural texture generation + caching |
| `lib/shaders/groundShader.ts` | Custom shader injection, stochastic + triplanar |
| `rendering/GroundMaterial.tsx` | Wires shader to material |
| `rendering/GroundSurface.tsx` | Ground plane composition |
| `types/environment.ts` | TypeScript interfaces |

[CODE: ui/src/lib/groundTextures.ts]
[CODE: ui/src/lib/shaders/groundShader.ts]

## SlimeAgent Anatomy

The `SlimeAgent` is a cute blob creature with a jelly-like appearance, built with `MeshPhysicalMaterial` and custom vertex shader injection:

```
         ∧  ∧          ← optional ear nubs (50% of agents)
       ╭──────╮
      (  ◕  ◕  )      ← large eyes (r=0.1) with tracking pupils
      (  ○ ○   )      ← optional blush marks (50% of agents)
      (   ω    )      ← mouth (4 style variants)
       ╰──────╯       ← sphere body, vertex-wobbled via slime shader
        ~~~~~         ← contact shadow from EnvironmentSetup
```

**Body**: `sphereGeometry(0.4, 32, 32)` with `MeshPhysicalMaterial`:
- `clearcoat: 1.0`, `clearcoatRoughness: 0.05` (glossy shell)
- `transmission: 0.15`, `thickness: 1.5`, `ior: 1.4` (subtle jelly translucency)
- `iridescence: 0.3`, `iridescenceIOR: 1.2` (color-shifting sheen)
- `roughness: 0.2`, `metalness: 0.0`
- Vertex wobble via `bindSlimeShader()` (simplex 3D noise displacement)

**Geometry Components:**
| Part | Geometry | Dimensions |
|------|----------|------------|
| Body | Sphere | radius 0.4, Y-scaled 0.82-0.88 |
| Eyes | Spheres (x2) | radius 0.1, at [±0.12, 0.1, 0.3] |
| Pupils | Spheres (x2) | radius 0.055, nested inside eyes |
| Mouth | Torus arc / Boxes | 4 variants: smile, cat :3, chevron ^, none |
| Ear nubs | Spheres (x2) | radius 0.08, optional (50% of agents) |
| Blush | Circles (x2) | radius 0.06, semi-transparent pink |

### Slime Shader Injection

The slime shader follows the same `onBeforeCompile` injection pattern as `groundShader.ts`:

```
┌─────────────────────────────────────────────────────────┐
│                  SLIME SHADER PIPELINE                    │
│                                                           │
│  glsl/slime.glsl.ts (GLSL source of truth)               │
│       │                                                   │
│       ├── VERTEX_COMMON_INJECTION                         │
│       │   ├── Simplex 3D noise function (Ashima Arts)     │
│       │   └── uniform declarations (uTime, uWobbleInt.)   │
│       │                                                   │
│       └── VERTEX_DISPLACEMENT_INJECTION                   │
│           ├── Noise-based vertex displacement              │
│           └── Y-axis squash/stretch (uSquashY)            │
│                                                           │
│  slimeShader.ts (runtime binding)                         │
│       ├── bindSlimeShader(material, config)                │
│       │   └── Sets onBeforeCompile + marker check          │
│       └── syncSlimeShader(material, time, squashY, wobble)│
│           └── Updates uniforms each frame                  │
└─────────────────────────────────────────────────────────┘
```

### Per-Agent Variation

Deterministic from `agentId` hash — ensures the same agent always looks the same:

| Feature | Probability | Range |
|---------|-------------|-------|
| Ear nubs | 50% | accent-colored spheres at top-sides |
| Blush marks | 50% | semi-transparent pink circles on cheeks |
| Mouth style | 4 variants | smile arc, cat :3, chevron ^, none |
| Wobble speed | continuous | 1.3 - 1.7 |
| Body aspect Y | continuous | 0.82 - 0.88 (slight height variation) |

### Animation System

All animations run in a single `useFrame` callback with LOD-based optimization:

| Animation | Trigger | LOD | Implementation |
|-----------|---------|-----|----------------|
| Breathing | Always idle | All | `scale.y = 1.0 + sin(t*2)*0.03` |
| Vertex wobble | Always idle | High/Medium | `uTime` uniform driven each frame |
| Lateral sway | Always idle | High/Medium | `position.x += sin(t*0.8)*0.01` |
| Cursor tracking | Cursor present | High/Medium | Pupil offset clamped to ±0.03 |
| Body lean | Cursor present | High only | `rotation.y = cursorDir.x * 0.1` |
| Hop locomotion | Moving | All | `abs(sin(t*6))*0.15` Y bounce |
| Squash on land | Moving, Y↓ | High/Medium | Compress Y scale, expand XZ |
| Stretch on jump | Moving, Y↑ | High/Medium | Extend Y scale, compress XZ |
| Wave | 1-2 skills selected | High | Body tilts side to side |
| Celebration | 3+ skills selected | High | Spin + bounce + increased wobble |

### Accessory Offsets

Offsets are tuned for the slime's spherical body:

| Slot | Position | Notes |
|------|----------|-------|
| head | [0, 0.4, 0] | On top of dome |
| back | [0, 0, -0.35] | Rear surface |
| leftHand | [-0.4, 0, 0.1] | Floating beside body |
| rightHand | [0.4, 0, 0.1] | Floating beside body |

[CODE: ui/src/components/world/agents/SlimeAgent.tsx]
[CODE: ui/src/lib/shaders/slimeShader.ts]

---

## State Management Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                        ZUSTAND STORES                           │
├─────────────────┬─────────────────┬─────────────────────────────┤
│  cameraStore    │  graphicsStore  │  environmentStore           │
│  ─────────────  │  ────────────── │  ─────────────────          │
│  * position     │  * tier (low->  │  * dreiPreset               │
│  * target       │    ultra)       │  * timeOfDay                │
│  * mode         │  * shadows,     │  * syncWithTheme            │
│  * zoom         │    bloom, etc   │                             │
│  * focusedId    │                 │                             │
├─────────────────┴─────────────────┴─────────────────────────────┤
│  accessoryStore              │  interactionStore                │
│  ──────────────────          │  ──────────────────              │
│  * agentAccessories          │  * hoveredObjectId               │
│    (per-agent head/held)     │  * isDragging                    │
│  * agentStatus               │                                  │
└─────────────────────────────────────────────────────────────────┘
             │
             ▼
    ┌────────────────────┐
    │  Components read   │
    │  from stores and   │
    │  react to changes  │
    └────────────────────┘
```

**Store Responsibilities:**

| Store | Purpose | Key State |
|-------|---------|-----------|
| `cameraStore` | Camera position, zoom, focus | position, target, mode, focusedAgentId |
| `graphicsStore` | Performance settings | tier, config, overrides |
| `environmentStore` | HDR and lighting | dreiPreset, timeOfDay, syncWithTheme |
| `accessoryStore` | Agent customization | agentAccessories, agentStatus |
| `interactionStore` | User interaction | hoveredObjectId, isDragging |

[CODE: ui/src/stores/cameraStore.ts]
[CODE: ui/src/stores/graphicsStore.ts]

---

## Performance Tiers

The graphics system supports four performance tiers with graceful degradation:

| Tier   | DPR | Shadows | Post-Proc | Bloom | SSAO | AA   |
|--------|-----|---------|-----------|-------|------|------|
| low    | 1   | No      | No        | No    | No   | none |
| medium | 1.5 | Yes     | Yes       | Yes   | No   | fxaa |
| high   | 2   | Yes     | Yes       | Yes   | Yes  | smaa |
| ultra  | 2   | Yes     | Yes       | Yes   | Yes  | smaa |

**Auto-detection:** The `useDeviceCapability` hook analyzes WebGL renderer strings, device pixel ratio, and mobile detection to recommend an appropriate tier.

[CODE: ui/src/config/graphics.ts]
[CODE: ui/src/hooks/useDeviceCapability.ts]

---

## Backpack Evolution (Skill Count Indicator)

The backpack accessory automatically scales based on agent skill count:

```
Skills: 0        1-2          3-5           6-10          11+
        ━━       📄           📁            💼            🎒
       none     paper       folder      briefcase     backpack
                                                    (with skill
                                                     indicators)
```

**Thresholds:**

| Skill Count | Accessory | Visual |
|-------------|-----------|--------|
| 0 | none | Nothing displayed |
| 1-2 | paper | Stacked paper sheets |
| 3-5 | folder | Folder with tab |
| 6-10 | briefcase | Briefcase with handle and clasps |
| 11+ | backpack | Full backpack with pockets, straps, skill indicator orbs |

[CODE: ui/src/types/accessory.ts#SKILL_BACKPACK_THRESHOLDS]
[CODE: ui/src/components/world/accessories/BackpackAccessory.tsx]

---

## Dependency Injection Pattern

The `AgentProvider` implements dependency injection for pluggable agent components:

```typescript
const AGENT_REGISTRY: AgentRegistry = {
  slime: {
    Component: SlimeAgent,
    displayName: 'Slime Agent',
    description: 'Cute blob creature with jelly-like appearance and organic animations',
  },
}
```

**Key Functions:**
- `useAgent()` - Access current agent config
- `useAgentComponent()` - Get the agent component type
- `getAvailableAgents()` - List all registered agents
- `registerAgent()` - Runtime agent registration

`AgentWithAccessories` uses `useAgentComponent()` to resolve the agent component at runtime, making the system truly pluggable for future agent types.

[CODE: ui/src/components/world/AgentProvider.tsx]

---

## Key Type Definitions

### AgentProps Interface

```typescript
interface AgentProps {
  position: [number, number, number]
  cursorPosition: { x: number; y: number } | null
  selectedNodes: string[]
  isAnimating: boolean
  onAnimationComplete?: () => void
  onAgentClick?: () => void
  agentId?: string
  colors?: { body: string; head: string; accent: string }
  isSeated?: boolean
  seatRotation?: number
}
```

[CODE: ui/src/types/world.ts]

### GraphicsConfig Interface

```typescript
interface GraphicsConfig {
  dpr: number | [number, number]
  shadows: boolean
  shadowMapSize: number
  postProcessing: boolean
  materialQuality: MaterialQuality
  envMap: boolean
  bloom: boolean
  ssao: boolean
  antialiasing: AntialiasingMethod
  vignette: boolean
  contactShadows: boolean
}
```

[CODE: ui/src/types/graphics.ts]

---

## Material System

### MaterialProvider

The `MaterialProvider` implements a caching system for Three.js materials:

- **Cache Key Format:** `${preset}:${color}:${quality}`
- **Auto-creates:** `MeshStandardMaterial` or `MeshBasicMaterial` based on quality tier
- **Cleanup:** Implements proper disposal on unmount
- **Fallback:** Creates materials directly if provider unavailable

### Material Presets

| Preset | Metalness | Roughness | Special |
|--------|-----------|-----------|---------|
| metallic | 0.8 | 0.2 | envMapIntensity 1.5 |
| matte | 0.0 | 0.9 | envMapIntensity 0.3 |
| plastic | 0.0 | 0.4 | envMapIntensity 0.8 |
| ceramic | 0.0 | 0.3 | envMapIntensity 1.0 |
| skin | 0.0 | 0.6 | envMapIntensity 0.4 |
| emissive | - | - | emissiveIntensity 0.5, toneMapped false |
| glowing | - | - | emissiveIntensity 1.0, toneMapped false |

[CODE: ui/src/components/world/materials/MaterialProvider.tsx]
[CODE: ui/src/components/world/materials/presets.ts]

---

## Rendering Pipeline

### RenderPipeline Component

Wraps the scene with `EffectComposer` for post-processing effects:

```
Scene
  └── EffectComposer
        ├── Bloom (threshold, smoothing)
        ├── Vignette (edge darkening)
        └── SMAA (antialiasing)
```

Effects are conditionally applied based on the current graphics tier configuration.

[CODE: ui/src/components/world/rendering/RenderPipeline.tsx]

### EnvironmentSetup Component

Sets up scene environment:
- **HDR Environment** - drei preset-based (apartment, city, dawn, forest, night, studio, sunset)
- **Contact Shadows** - Ground shadows with blur
- **Theme Sync** - Auto-matches dark/light theme to appropriate preset

[CODE: ui/src/components/world/rendering/EnvironmentSetup.tsx]

---

## Overlay System

The overlay system renders 2D UI elements in 3D space using HTML overlays:

```
y offset:
  1.7    SpeechBubble      (topmost)
  1.5    ThinkingBubble
  1.3    StatusIcon
  1.0    NameTag           (lowest)
```

**Overlay Types:**
- **NameTag** - Displays agent name with outline
- **StatusIcon** - Visual indicator (normal, warning, error, info, thinking, speaking)
- **ThinkingBubble** - Animated thinking indicator with dots
- **SpeechBubble** - For agent speech/messages

[CODE: ui/src/components/world/overlays/AgentOverlayGroup.tsx]

---

## Camera System

The camera supports three modes:

| Mode | Position | Use Case |
|------|----------|----------|
| freeform | [0, 5, 10] | Default exploration |
| top-down | [0, 20, 0.1] | Overview of all agents |
| zoomed-agent | Agent + [0, 2, 5] | Focus on specific agent |

**Key Actions:**
- `zoomToAgent(agentId, position)` - Animate camera to focus on agent
- `exitZoom()` - Return to previous mode
- `cycleCameraMode()` - Toggle between modes
- `setTopDown()` / `setFreeform()` - Direct mode setting

The camera store maintains a history stack (last 10 positions) for navigation.

[CODE: ui/src/stores/cameraStore.ts]

---

## Testing

For testing strategies and dependency injection patterns, see [DOC: docs/SEAMS.md#3d-world-testing-seams].

---

## Related Documentation

- [DOC: docs/guides/ASSET-GENERATION.md] - How to create new 3D objects using the GLB generator pipeline
- [DOC: docs/SEAMS.md] - Testing seams and dependency injection patterns
- [DOC: PRD.md] - Product requirements and operational targets
