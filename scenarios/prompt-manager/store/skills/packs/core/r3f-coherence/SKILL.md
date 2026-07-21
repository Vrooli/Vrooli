## Steer focus: React Three Fiber Coherence

Prioritize **performance-first 3D development** using React Three Fiber patterns that prevent re-renders, optimize animation loops, and maintain architectural coherence across 3D scenes.

Your goal is to ensure 3D codebases achieve **60fps rendering without React re-render overhead** - the single most common performance killer in R3F applications.

Do **not** break existing functionality, regress visual fidelity, or introduce new features. All changes must maintain or improve performance and structural consistency.

Required reading:
- `prompt-manager skill read visited-tracker-tools`

---

### **0. Why This Skill Exists**

When React Three Fiber applications grow, common performance issues emerge:
- **Re-render cascades:** State updates in `useFrame` triggering 60 re-renders per second
- **Store over-subscription:** Components re-rendering on any Zustand store change instead of selective updates
- **Inline object creation:** New arrays/objects in JSX props causing unnecessary reconciliation
- **Missing isolation:** Stateful components causing entire scene graphs to re-render
- **Asset loading chaos:** Models and textures loading synchronously, blocking the main thread

The root cause: **React's reconciliation model conflicts with Three.js's imperative mutation model.**

This skill provides concrete patterns that ensure agents understand when to use React patterns vs. direct Three.js mutation.

---

### **1. The Core Tension: React vs. Three.js**

```
┌─────────────────────────────────────────────────────────────────┐
│                    THE FUNDAMENTAL CONFLICT                     │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  REACT WANTS:              THREE.JS WANTS:                      │
│  ─────────────             ───────────────                      │
│  • Immutable state         • Mutable objects                    │
│  • Re-render on change     • Direct property mutation           │
│  • Declarative updates     • Imperative animation loops         │
│  • Component lifecycle     • Continuous frame updates           │
│                                                                 │
│  R3F BRIDGES BOTH - but you must know WHEN to use WHICH        │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

#### **Decision Flow: State vs. Mutation**

```
                    Does this value change every frame?
                              │
              ┌───────────────┴───────────────┐
            YES                               NO
              │                                │
              ▼                                ▼
     MUTATE VIA REF              Does it affect scene structure?
     (useFrame + ref.current)           (add/remove objects)
              │                                │
              │                    ┌───────────┴───────────┐
              │                  YES                       NO
              │                    │                        │
              │                    ▼                        ▼
              │              REACT STATE              Does UI need
              │              (useState/Zustand)       to reflect it?
              │                                           │
              │                               ┌───────────┴───────────┐
              │                             YES                       NO
              │                               │                        │
              │                               ▼                        ▼
              │                         ZUSTAND STORE           MUTATE VIA REF
              │                         (with selectors)        (silent update)
              │
              └─────────────────────────────────────────────────────────────┘
```

---

### **2. Performance Patterns (CRITICAL)**

These patterns prevent the #1 R3F performance killer: unnecessary re-renders.

#### **Pattern 1: Never setState in useFrame**

```tsx
// ❌ CATASTROPHIC: 60 re-renders per second
function BadRotatingBox() {
  const [rotation, setRotation] = useState(0);

  useFrame(() => {
    setRotation(r => r + 0.01);  // React re-renders EVERY FRAME
  });

  return <mesh rotation-y={rotation}><boxGeometry /></mesh>;
}

// ✅ CORRECT: Zero re-renders, direct mutation
function GoodRotatingBox() {
  const meshRef = useRef<Mesh>(null);

  useFrame(() => {
    if (meshRef.current) {
      meshRef.current.rotation.y += 0.01;  // Direct Three.js mutation
    }
  });

  return <mesh ref={meshRef}><boxGeometry /></mesh>;
}
```

#### **Pattern 2: Zustand Selectors (Granular Subscriptions)**

```tsx
// ❌ BAD: Re-renders on ANY store change
function BadComponent() {
  const store = useGameStore();  // Subscribes to everything
  return <mesh position-x={store.playerX} />;
}

// ✅ GOOD: Re-renders ONLY when playerX changes
function GoodComponent() {
  const playerX = useGameStore(state => state.playerX);
  return <mesh position-x={playerX} />;
}

// ✅ BEST: Zero re-renders for continuous values
function BestComponent() {
  const meshRef = useRef<Mesh>(null);

  useFrame(() => {
    const { playerX, playerY } = useGameStore.getState();
    if (meshRef.current) {
      meshRef.current.position.set(playerX, playerY, 0);
    }
  });

  return <mesh ref={meshRef} />;
}
```

#### **Pattern 3: Avoid Inline Objects in JSX**

```tsx
// ❌ BAD: New array created every render → reconciliation overhead
function BadMesh() {
  return <mesh position={[0, 1, 0]} scale={[1, 2, 1]} />;
}

// ✅ GOOD: Stable references via useMemo or constants
const POSITION = [0, 1, 0] as const;
const SCALE = [1, 2, 1] as const;

function GoodMesh() {
  return <mesh position={POSITION} scale={SCALE} />;
}

// ✅ ALSO GOOD: Individual props for simple cases
function AlsoGoodMesh() {
  return <mesh position-y={1} scale-y={2} />;
}
```

#### **Pattern 4: Isolate Stateful Components**

```tsx
// ❌ BAD: State in parent causes entire scene to re-render
function BadScene() {
  const [score, setScore] = useState(0);

  return (
    <>
      <Environment />           {/* Re-renders on score change! */}
      <Player />                {/* Re-renders on score change! */}
      <ScoreDisplay score={score} />
      <Enemies />               {/* Re-renders on score change! */}
    </>
  );
}

// ✅ GOOD: Isolated state via Zustand + selective subscription
function GoodScene() {
  return (
    <>
      <Environment />           {/* Never re-renders */}
      <Player />                {/* Only re-renders on player state */}
      <ScoreDisplay />          {/* Only re-renders on score state */}
      <Enemies />               {/* Only re-renders on enemy state */}
    </>
  );
}

// ScoreDisplay subscribes only to what it needs
function ScoreDisplay() {
  const score = useGameStore(s => s.score);
  return <Html><div>{score}</div></Html>;
}
```

---

### **3. useFrame Best Practices**

The animation loop is where most R3F applications spend their time. Optimize it ruthlessly.

#### **useFrame Decision Table**

| Scenario | Pattern | Example |
|----------|---------|---------|
| Continuous animation | Mutate ref in useFrame | `ref.current.rotation.y += delta` |
| Frame-rate independence | Use delta parameter | `position.x += speed * delta` |
| Conditional updates | Use active parameter | `useFrame(cb, { active: isPlaying })` |
| Execution order | Use priority parameter | `useFrame(cb, { priority: -1 })` |
| On-demand rendering | Call invalidate() | `invalidate()` after state change |
| Read-only access | Destructure state | `useFrame(({ camera, clock }) => ...)` |

#### **Frame Loop Patterns**

```tsx
// Frame-rate independent motion (runs same speed on 30fps and 144fps)
useFrame((state, delta) => {
  meshRef.current.position.x += velocity * delta;
});

// Conditional subscription (pauses when not needed)
useFrame(
  () => { /* animation logic */ },
  { active: isAnimating }  // Unsubscribes when false
);

// Priority ordering (lower runs first)
useFrame(() => { /* physics */ }, { priority: -1 });  // Runs first
useFrame(() => { /* rendering */ }, { priority: 1 }); // Runs last

// On-demand rendering (for static scenes)
<Canvas frameloop="demand">
  <Scene />
</Canvas>

// Then trigger renders explicitly:
useFrame(({ invalidate }) => {
  if (needsUpdate) invalidate();
});
```

---

### **4. Component Architecture**

```
┌─────────────────────────────────────────────────────────────────┐
│                    R3F COMPONENT LAYERS                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Canvas Layer (one per app)                                     │
│  ├── Camera, lights, environment, post-processing              │
│  ├── Configured via Canvas props, rarely changes               │
│  │                                                              │
│  Scene Layer (composition)                                      │
│  ├── Suspense boundaries, scene structure                       │
│  ├── Static: Environment, lights, helpers                       │
│  ├── Dynamic: Game objects, characters                          │
│  │                                                              │
│  Object Layer (individual meshes/groups)                        │
│  ├── Use refs for animation                                     │
│  ├── Minimal props, stable references                           │
│  ├── forwardRef for external access                             │
│  │                                                              │
│  Material/Geometry Layer (shared resources)                     │
│  ├── Hoist to module scope or useMemo                           │
│  ├── dispose={null} for shared instances                        │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

#### **Component Patterns**

```tsx
// ✅ Export with forwardRef for external animation control
const AnimatedBox = forwardRef<Mesh, BoxProps>((props, ref) => {
  return (
    <mesh ref={ref} {...props}>
      <boxGeometry />
      <meshStandardMaterial />
    </mesh>
  );
});

// ✅ Shared geometry/material (module scope = single instance)
const sharedGeometry = new BoxGeometry(1, 1, 1);
const sharedMaterial = new MeshStandardMaterial({ color: 'hotpink' });

function OptimizedBox() {
  return (
    <mesh geometry={sharedGeometry} material={sharedMaterial} dispose={null} />
  );
}

// ✅ Using primitive for existing Three.js objects
function ExistingObject({ object }: { object: Object3D }) {
  return <primitive object={object} />;
}

// ✅ Extending Three.js with custom classes
extend({ CustomMaterial });
// Then use as: <customMaterial />
```

---

### **5. Asset Loading Patterns**

```
                    Loading an asset?
                          │
          ┌───────────────┴───────────────┐
        GLTF/Model                    Texture
          │                               │
          ▼                               ▼
    useGLTF + preload             useTexture + preload
          │                               │
          ▼                               ▼
    Wrap in Suspense              Wrap in Suspense
          │                               │
          └───────────┬───────────────────┘
                      ▼
              <Suspense fallback={<Loader />}>
                <ModelOrTexture />
              </Suspense>
```

#### **Loading Best Practices**

```tsx
// ✅ Preload at module level (starts loading immediately)
useGLTF.preload('/models/character.glb');
useTexture.preload('/textures/diffuse.jpg');

// ✅ Component with proper Suspense handling
function Character() {
  const { scene, animations } = useGLTF('/models/character.glb');
  return <primitive object={scene} />;
}

// ✅ Scene with Suspense boundary
function Scene() {
  return (
    <Suspense fallback={<LoadingIndicator />}>
      <Character />
      <Environment preset="sunset" />
    </Suspense>
  );
}

// ✅ Loading progress indicator
function LoadingScreen() {
  const { progress } = useProgress();
  return <Html center>{progress.toFixed(0)}%</Html>;
}
```

---

### **6. Drei Helpers Reference**

Essential utilities from `@react-three/drei`:

| Helper | Purpose | When to Use |
|--------|---------|-------------|
| `useGLTF` | Load GLTF/GLB models | Any 3D model loading |
| `useTexture` | Load textures with caching | PBR materials, sprites |
| `Environment` | HDRI lighting + reflections | Realistic lighting |
| `OrbitControls` | Camera orbit/pan/zoom | Development, product viewers |
| `Html` | DOM overlays in 3D space | Labels, UI elements |
| `Instances` | GPU instancing | Many identical objects |
| `Float` | Floating animation | Idle animations |
| `ContactShadows` | Soft ground shadows | Product shots |
| `Bounds` | Auto-fit camera to scene | Model viewers |
| `Center` | Center objects at origin | Imported models |

---

### **7. R3F Coherence Audit**

Run this audit when joining a project or before performance optimization.

#### **Step 1: Re-render Detection**

```bash
# Find setState calls inside useFrame (CRITICAL BUG)
rg "useFrame.*setState|setState.*useFrame" --type tsx

# Find entire-store subscriptions
rg "= use\w+Store\(\)" --type tsx  # Missing selector = bad

# Find inline array/object props
rg "position=\[|scale=\[|rotation=\[" --type tsx
```

**Red flags:**
- [ ] Any `setState` inside `useFrame` → immediate fix required
- [ ] Store usage without selectors → add granular selectors
- [ ] Inline arrays in frequently-rendered components → extract to constants

#### **Step 2: Animation Loop Analysis**

```bash
# Find all useFrame usage
rg "useFrame\(" --type tsx -l

# Check for delta usage (frame-rate independence)
rg "useFrame.*delta" --type tsx

# Find components without refs that animate
rg "useFrame" -A 5 --type tsx | rg -v "useRef|ref"
```

**Red flags:**
- [ ] useFrame without delta → motion varies by frame rate
- [ ] useFrame without refs → likely mutating state instead

#### **Step 3: Asset Loading Check**

```bash
# Find preload calls
rg "\.preload\(" --type tsx

# Find Suspense boundaries
rg "<Suspense" --type tsx -l

# Find synchronous asset usage (missing Suspense)
rg "useGLTF|useTexture" --type tsx -l
```

**Red flags:**
- [ ] useGLTF/useTexture without nearby Suspense → loading flash
- [ ] Missing preload calls → delayed asset loading

#### **Step 4: Document Findings**

After auditing, update `docs/internal/R3F_COHERENCE_NOTES.md`:

```markdown
# R3F Coherence Audit - [Date]

## Performance Issues
- Re-render bugs found: [list with file:line]
- Store subscription issues: [list]
- Inline object violations: [list]

## Animation Loop
- useFrame count: [number]
- Delta usage: [percentage]
- Missing refs: [list]

## Asset Loading
- Preload coverage: [percentage]
- Missing Suspense: [list]

## Priority Fixes
1. [Critical re-render bug]
2. [Performance issue]
3. [Code quality issue]
```

---

### **8. Memory Management with Visited Tracker**

Use the `visited-tracker-tools` skill for tracking visited files, with LOCATION set to `scenarios/{{TARGET}}/ui` and TAG set to `r3f-coherence`.

---

### **9. Relationship to Other Skills**

| Skill | Focus | When to Use Together |
|-------|-------|---------------------|
| react-coherence | General React architecture | R3F coherence extends React coherence patterns to 3D |
| react-stability | Crash prevention | Stability patterns apply equally to R3F components |
| performance | General optimization | After R3F-specific fixes, apply general perf patterns |

Optional reading:
- `prompt-manager skill read react-coherence react-stability performance`

**Recommended sequence for R3F work:**
1. **r3f-coherence** (audit) → find re-render bugs and animation issues
2. **react-stability** → ensure error boundaries and defensive patterns
3. **performance** → bundle size, code splitting for 3D assets

---

### **10. Scenario Constraints**

* Do **not** change scene structure or visual design without explicit request
* Do **not** swap rendering libraries (e.g., replace R3F with vanilla Three.js)
* Do **not** remove features to improve performance without discussion
* Prefer **incremental optimization** over architectural rewrites
* Performance changes must be **measurable** (before/after frame time)

---

### **11. Output Expectations**

You may update:
* Animation patterns to use refs instead of state
* Store subscriptions to use granular selectors
* Asset loading to use preload + Suspense
* Component structure to isolate stateful logic
* Inline objects to use stable references

You **must**:
* Keep the scene visually identical
* Maintain or improve frame rate
* Preserve all interactivity and functionality
* Document significant optimizations in commit messages

**Avoid superficial changes that don't measurably improve performance or code clarity.**

---

### **12. Testing R3F Components**

Testing React Three Fiber components requires specialized utilities because:
- Standard React Testing Library can't handle WebGL context
- Animation tests need frame loop simulation
- Store patterns differ from typical React patterns

#### **Testing Pyramid for R3F**

```
              /\
             /  \  Integration Tests
            /    \  (shader + material binding, full render cycles)
           /──────\
          /        \  Component Tests
         /          \  (animation behavior, store connectivity)
        /────────────\
       /              \  Unit Tests
      /                \  (store logic, utility functions, selectors)
     /────────────────────\
```

**Bottom Layer (Unit Tests):**
- Zustand store actions and selectors
- Utility functions (color conversion, math helpers)
- GLSL validation (syntax, uniforms, varyings)

**Middle Layer (Component Tests):**
- Animation behavior via frame simulation
- Store connectivity patterns
- Prop handling and memoization

**Top Layer (Integration Tests):**
- Shader binding to materials
- Full render → animate → update cycles
- Multi-component interactions

#### **Test Utility Reference**

The following utilities should be available in `@/test/`:

| Utility | File | Purpose |
|---------|------|---------|
| `FrameLoopSimulator` | `r3f-test-utils.ts` | Simulate useFrame callbacks |
| `RenderTracker` | `r3f-test-utils.ts` | Track component re-renders |
| `createMockMesh()` | `r3f-test-utils.ts` | Lightweight mesh stub |
| `createMockPointerEvent()` | `r3f-test-utils.ts` | Simulate R3F pointer events |
| `simulateDragSequence()` | `r3f-test-utils.ts` | Full drag interaction simulation |
| `takeStoreSnapshot()` | `r3f-store-test-utils.ts` | Capture store state |
| `diffSnapshots()` | `r3f-store-test-utils.ts` | Compare before/after states |
| `assertOnlyFieldsChanged()` | `r3f-store-test-utils.ts` | Verify selective updates |
| `StoreSubscriptionTracker` | `r3f-store-test-utils.ts` | Track subscription events |
| `MutationTracker` | `r3f-store-test-utils.ts` | Track ref vs state mutations |
| `R3FTestHarness` | `r3f-component-harness.tsx` | Provider for R3F components |

#### **Animation Testing Pattern**

Test that animations use refs (not state) and are frame-rate independent:

```tsx
import { FrameLoopSimulator, createMockMesh } from '@/test/r3f-test-utils'

it('uses delta for frame-rate independent animation', () => {
  const simulator = new FrameLoopSimulator()
  const meshRef = { current: createMockMesh() }

  // Register animation callback (mimics component's useFrame)
  simulator.registerCallback((_, delta) => {
    if (meshRef.current) {
      meshRef.current.rotation.y += delta * 2 // 2 radians per second
    }
  })

  // Simulate 1 second at 60fps
  simulator.tickFrames(60, 1/60)
  const rotation60fps = meshRef.current.rotation.y

  // Reset and test at 30fps
  meshRef.current.rotation.y = 0
  simulator.reset()

  simulator.registerCallback((_, delta) => {
    if (meshRef.current) {
      meshRef.current.rotation.y += delta * 2
    }
  })

  simulator.tickFrames(30, 1/30)
  const rotation30fps = meshRef.current.rotation.y

  // Both should reach ~2 radians (same elapsed time)
  expect(Math.abs(rotation60fps - rotation30fps)).toBeLessThan(0.01)
})
```

#### **Store Integration Testing Pattern**

Test that components use getState() in useFrame (not subscriptions):

```tsx
import { StoreSubscriptionTracker } from '@/test/r3f-store-test-utils'
import { useLODStore } from '@/stores/lodStore'

it('uses getState() in animation loop', () => {
  const tracker = new StoreSubscriptionTracker(useLODStore)

  // Simulate 60 frames of animation
  tickFrames(60)

  // Should have minimal subscription events
  // (getState() doesn't trigger subscriptions)
  expect(tracker.getEventCount()).toBeLessThan(5)

  tracker.dispose()
})
```

#### **Snapshot Testing for Store Actions**

Test store actions only change expected fields:

```tsx
import {
  takeStoreSnapshot,
  assertOnlyFieldsChanged,
} from '@/test/r3f-store-test-utils'

it('setPosition only updates position field', () => {
  const before = takeStoreSnapshot(useCameraStore)

  useCameraStore.getState().setPosition([5, 10, 15])

  const after = takeStoreSnapshot(useCameraStore)

  // Should ONLY change position, nothing else
  assertOnlyFieldsChanged(before, after, ['position'])
})
```

#### **Interaction Testing Pattern**

Test R3F pointer events:

```tsx
import {
  createMockPointerEvent,
  simulateDragSequence,
} from '@/test/r3f-test-utils'

it('handles drag sequence correctly', () => {
  const onDragStart = vi.fn()
  const onDrag = vi.fn()
  const onDragEnd = vi.fn()

  simulateDragSequence(
    {
      onPointerDown: onDragStart,
      onPointerMove: onDrag,
      onPointerUp: onDragEnd,
    },
    {
      startPoint: [0, 0, 0],
      endPoint: [5, 0, 0],
      steps: 10,
    }
  )

  expect(onDragStart).toHaveBeenCalledTimes(1)
  expect(onDrag).toHaveBeenCalledTimes(10)
  expect(onDragEnd).toHaveBeenCalledTimes(1)
})
```

#### **What NOT to Test in Unit Tests**

| Skip | Why | Test At |
|------|-----|---------|
| Visual appearance | Requires WebGL context | Manual/screenshot tests |
| WebGL rendering | No GPU in test env | Integration tests with gl mock |
| Complex scene composition | Too many moving parts | E2E tests |
| Performance metrics | Unreliable in CI | Benchmarks with real GPU |

#### **Quick Reference: R3F Testing Patterns**

| Pattern | When to Use | Key Assertion |
|---------|-------------|---------------|
| Frame simulation | Animation tests | `simulator.tickFrames(n)` then check ref values |
| Render tracking | Performance tests | `tracker.assertMaxRenders(n)` |
| Store snapshots | Action tests | `assertOnlyFieldsChanged(before, after, ['field'])` |
| Subscription tracking | useFrame patterns | `tracker.getEventCount() < 5` |
| Mutation tracking | Ref vs state | `tracker.assertRefBasedAnimation()` |
| Pointer events | Interaction tests | `simulateDragSequence()` |

#### **Running R3F Tests**

```bash
# Run all tests
cd scenarios/{{TARGET}} && make test

# Run specific R3F component tests
cd scenarios/{{TARGET}}/ui && npx vitest run --reporter=verbose

# Run with coverage (utilities should be excluded)
cd scenarios/{{TARGET}}/ui && npx vitest run --coverage

# Watch mode for development
cd scenarios/{{TARGET}}/ui && npx vitest --watch
```

---
