---
name: "r3f-coherence"
description: "React Three Fiber performance patterns - prevent re-renders, optimize useFrame, and maintain 60fps rendering in 3D scenes"
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["steer","react","performance"]
  tags: ["skill"]
  icon: "box"
  status: "active"
  revision: 46
  createdAt: "2026-01-25T00:00:00Z"
  updatedAt: "2026-02-05T01:02:03Z"
  requires:
    scenarios: ["prompt-manager"]
    commands: ["prompt-manager skill", "prompt-manager skill read"]
  origin:
    kind: "authored"
---
## Steer focus: React Three Fiber Coherence

Prioritize **performance-first 3D development** using React Three Fiber patterns that prevent re-renders, optimize animation loops, and maintain architectural coherence across 3D scenes.

Your goal is to ensure 3D codebases achieve **60fps rendering without React re-render overhead** - the single most common performance killer in R3F applications.

Do **not** break existing functionality, regress visual fidelity, or introduce new features. All changes must maintain or improve performance and structural consistency.

Required reading:
- `prompt-manager skill read visited-tracker-tools`

---

### **0. The core tension**

React reconciles immutable state; Three.js mutates objects in a frame loop. R3F bridges both and the failures are the same few: re-render cascades from state set in `useFrame`, whole-store subscriptions, inline arrays and objects in JSX props, stateful parents re-rendering whole scene graphs, and synchronous asset loading.

**Decision flow.** A value that changes every frame is mutated through a ref in `useFrame`. A value that changes scene structure (adds or removes objects) is React state or Zustand. A value the UI must reflect is a Zustand store read with a selector. Anything else is a silent ref mutation.

---

### **1. Performance patterns (critical)**

1. **Never `setState` in `useFrame`.** It re-renders 60 times a second; mutate `meshRef.current.rotation.y += delta` instead.
2. **Zustand selectors.** `useGameStore(s => s.playerX)`, never `useGameStore()`. For continuous values read `useGameStore.getState()` inside `useFrame` and write to a ref: zero re-renders.
3. **No inline objects in JSX.** `position={[0,1,0]}` allocates every render: use module constants (`as const`), `useMemo`, or per-axis props (`position-y={1}`).
4. **Isolate stateful components.** State in a scene-level parent re-renders `Environment`, `Player` and `Enemies`; keep state in a store and let a leaf such as `ScoreDisplay` subscribe to just `s.score`.

### **2. `useFrame`**

Mutate refs and scale by the `delta` argument (`position.x += speed * delta`) so motion is frame-rate independent. `useFrame(cb, { active })` pauses when idle, `{ priority }` orders callbacks (lower runs first), `<Canvas frameloop="demand">` with `invalidate()` serves static scenes, and the state argument (`camera`, `clock`) is read-only access.

### **3. Component architecture**

One `Canvas` per app (camera, lights, environment, post-processing; rarely changes). The scene layer holds Suspense boundaries and structure, with static parts (environment, lights, helpers) apart from dynamic ones. Object components animate through refs, keep props minimal and stable, and `forwardRef` for external control. Hoist shared geometry and materials to module scope or `useMemo` with `dispose={null}`. Use `<primitive object={...} />` for existing Three.js objects and `extend({ CustomMaterial })` for custom classes.

### **4. Assets**

Preload at module level (`useGLTF.preload`, `useTexture.preload`), load with `useGLTF` and `useTexture`, wrap in `<Suspense fallback={...}>`, and show progress with `useProgress`. Useful drei helpers: `Environment`, `Html`, `Instances` (many identical objects), `Bounds`, `Center`, `Float`, `ContactShadows`, `OrbitControls`.

---

### **5. Audit**

Run when joining a project or before performance work, with `rg` over `ui/src` (tsx):
- Re-renders: `setState` inside `useFrame` (fix immediately), `= use\w+Store\(\)` with no selector, `position=\[|scale=\[|rotation=\[` in frequently rendered components.
- Animation loop: `useFrame(` without `delta` (motion varies by frame rate), animating components without refs (likely mutating state).
- Assets: `useGLTF` or `useTexture` without a nearby `<Suspense`, missing `.preload(` calls.

Record durable findings in `docs/internal/R3F_COHERENCE_NOTES.md` (issues with `file:line`, `useFrame` count and delta usage, preload and Suspense coverage, priority fixes). Track visited files with `visited-tracker-tools` (location `scenarios/{{TARGET}}/ui`, tag `r3f-coherence`). Pair with `ui-health` (error boundaries, defensive access) and `performance` (bundle size, code splitting for 3D assets).

### **6. Constraints and output**

Do not change scene structure or visual design without request, swap rendering libraries, or remove features for speed without discussion. Prefer incremental optimization; every performance change is measurable (before and after frame time). You may move animation to refs, store subscriptions to selectors, asset loading to preload plus Suspense, stateful logic into isolated components, and inline objects to stable references. Keep the scene visually identical, interactive and at least as fast; avoid superficial changes that do not measurably improve performance or clarity.

---

### **7. Testing R3F components**

Standard React Testing Library cannot handle a WebGL context, animation tests need frame-loop simulation, and store patterns differ. Pyramid: unit tests (store logic, selectors, utilities), component tests (animation behavior, store connectivity), integration tests (shader and material binding, full render cycles). Use the scenario's R3F test utilities (see its `ui/src` test helpers): `simulator.tickFrames(n)` then check ref values for animation; `tracker.assertMaxRenders(n)` for render counts; `assertOnlyFieldsChanged(before, after, ['field'])` for store actions; `tracker.getEventCount()` for `useFrame` subscriptions; `tracker.assertRefBasedAnimation()` for ref versus state; `simulateDragSequence()` for pointer interaction. Do not unit-test visual appearance, WebGL rendering, complex scene composition or performance numbers; those belong to screenshot, integration-with-gl-mock, end-to-end and real-GPU benchmark tests. Run with `cd scenarios/{{TARGET}} && make test`, or `npx vitest run` in `ui/` (exclude utilities from coverage).
