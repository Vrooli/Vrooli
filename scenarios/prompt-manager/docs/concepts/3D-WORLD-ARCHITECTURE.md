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
│   ├── WorldScene.tsx           # 3D scene composition (lights, controls, members)
│   ├── MemberProvider.tsx       # Dependency injection for swappable member types
│   │
│   ├── members/
│   │   ├── GeometricMember.tsx        # Procedural 3D character (capsules/spheres)
│   │   └── MemberWithAccessories.tsx  # Wraps member + backpack/head/held items
│   │
│   ├── accessories/
│   │   ├── BackpackAccessory.tsx   # Auto-sized by skill count (paper->backpack)
│   │   ├── HeadAccessory.tsx       # Hat, glasses, crown, etc.
│   │   └── HeldItemAccessory.tsx   # Book, orb, wand, etc.
│   │
│   ├── overlays/
│   │   ├── MemberOverlayGroup.tsx  # Stacks 2D UI in 3D space
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
├── stores/                  # Zustand state management
│   ├── cameraStore.ts       # Position, zoom, modes (freeform/zoomed/top-down)
│   ├── graphicsStore.ts     # Performance tiers (low->ultra)
│   ├── environmentStore.ts  # HDR presets, time of day
│   ├── accessoryStore.ts    # Per-member accessories
│   └── interactionStore.ts  # Hover/drag state
│
├── hooks/
│   ├── useHoverHighlight.ts    # Hover effects on 3D objects
│   ├── useGraphicsSettings.ts  # Access/modify graphics tier
│   └── useDeviceCapability.ts  # Auto-detect GPU tier
│
├── types/
│   ├── world.ts       # MemberProps, CameraState
│   ├── member.ts      # Member interface, colors
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
│   └── MemberProvider ─────────────► WorldScene
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
│       └── Members (loop) ─────────► MemberWithAccessories
│                                      │
│           ┌──────────────────────────┘
│           │
│           ├── GeometricMember ────► Body (capsule)
│           │                         Head (sphere + eyes)
│           │                         Arms (capsules)
│           │                         Hover glow
│           │                         Floating orbs
│           │
│           ├── BackpackAccessory ──► paper/folder/briefcase/backpack
│           ├── HeadAccessory ──────► hat/glasses/crown/halo
│           ├── HeldItemAccessory ──► book/tool/orb/wand
│           │
│           └── MemberOverlayGroup ─► NameTag (y:1.0)
│                                     StatusIcon (y:1.3)
│                                     ThinkingBubble (y:1.5)
│                                     SpeechBubble (y:1.7)
│
└── UI Overlays (HTML)
    ├── WorldControls
    ├── CombinePanel
    └── MemberOverlay
```

---

## Ground Texturing Pipeline

Ground rendering is split into two responsibilities:

- `GroundSurface` decides **what** to render (grid vs plane) based on environment config.
- `GroundMaterial` wires **how** it renders by applying procedural texture sets and shader projection via `ui/src/lib/groundTextures.ts` + `ui/src/lib/groundShader.ts`.

This keeps world composition free of shader/material concerns and makes texture tuning a local change.

## GeometricMember Anatomy

The `GeometricMember` is a procedurally generated 3D character built with Three.js primitives:

```
                    ╭────────────╮
                   ( * antenna * )   <- accent sphere
                    ╰────────────╯
                   ╭──────────────╮
                  (   O       O   )  <- eyes with pupils
                   │    HEAD      │  <- sphere (r=0.3)
                    ╰────────────╯
                   ┌──────────────┐
         arm ───► ═╡    BODY      ╞═  <─── arm
        capsule    │   capsule    │      capsule
                   │  (r=0.25)    │
                   └──────────────┘

Interactive behaviors:
  * Head follows cursor (smooth lerp)
  * Wave animation (1-2 skills selected)
  * Celebration spin+bounce (3+ skills)
  * Floating orbs orbit when selected
```

**Geometry Components:**
| Part | Geometry | Dimensions |
|------|----------|------------|
| Body | Capsule | radius 0.25, height 0.5 |
| Head | Sphere | radius 0.3 |
| Eyes | Spheres (x2) | radius 0.06 |
| Pupils | Spheres (x2) | radius 0.03 |
| Arms | Capsules (x2) | radius 0.06, height 0.25 |
| Antenna | Cylinder + Sphere | height 0.15, sphere r=0.05 |

**Materials:**
| Part | Metalness | Roughness | Special |
|------|-----------|-----------|---------|
| Body | 0.3 | 0.6 | - |
| Head | 0.2 | 0.5 | - |
| Accent | 0.4 | 0.4 | Emissive glow |

[CODE: ui/src/components/world/members/GeometricMember.tsx]

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
│  * memberAccessories         │  * hoveredObjectId               │
│    (per-member head/held)    │  * isDragging                    │
│  * memberStatus              │                                  │
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
| `cameraStore` | Camera position, zoom, focus | position, target, mode, focusedMemberId |
| `graphicsStore` | Performance settings | tier, config, overrides |
| `environmentStore` | HDR and lighting | dreiPreset, timeOfDay, syncWithTheme |
| `accessoryStore` | Member customization | memberAccessories, memberStatus |
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

The backpack accessory automatically scales based on member skill count:

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

The `MemberProvider` implements dependency injection for pluggable member components:

```typescript
const MEMBER_REGISTRY: MemberRegistry = {
  geometric: {
    Component: GeometricMember,
    displayName: 'Geometric Member',
    description: 'Abstract geometric member built with Three.js primitives',
  },
  // Future members (mixamo, rive, etc.) can be registered here
}
```

**Key Functions:**
- `useMember()` - Access current member config
- `useMemberComponent()` - Get the member component type
- `getAvailableMembers()` - List all registered members
- `registerMember()` - Runtime member registration

This pattern allows swapping member implementations without changing consumer code.

[CODE: ui/src/components/world/MemberProvider.tsx]

---

## Key Type Definitions

### MemberProps Interface

```typescript
interface MemberProps {
  position: [number, number, number]
  cursorPosition: { x: number; y: number } | null
  selectedNodes: string[]
  isAnimating: boolean
  onAnimationComplete?: () => void
  onMemberClick?: () => void
  memberId?: string
  colors?: { body: string; head: string; accent: string }
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
- **NameTag** - Displays member name with outline
- **StatusIcon** - Visual indicator (normal, warning, error, info, thinking, speaking)
- **ThinkingBubble** - Animated thinking indicator with dots
- **SpeechBubble** - For member speech/messages

[CODE: ui/src/components/world/overlays/MemberOverlayGroup.tsx]

---

## Camera System

The camera supports three modes:

| Mode | Position | Use Case |
|------|----------|----------|
| freeform | [0, 5, 10] | Default exploration |
| top-down | [0, 20, 0.1] | Overview of all members |
| zoomed-member | Member + [0, 2, 5] | Focus on specific member |

**Key Actions:**
- `zoomToMember(memberId, position)` - Animate camera to focus on member
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

- [DOC: docs/SEAMS.md] - Testing seams and dependency injection patterns
- [DOC: PRD.md] - Product requirements and operational targets
