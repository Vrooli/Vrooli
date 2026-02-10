# 3D Asset Generation Guide

How to create new 3D objects (decorations, furniture, props) for the prompt-manager world using the Node.js GLB generator pipeline.

---

## Why Generated Assets?

External model downloads (poly.pizza, Quaternius, etc.) are blocked by Cloudflare bot protection when automated. Instead, we generate stylized low-poly GLB models via a pure Node.js script — no browser APIs, no three.js exporter, no external tools. Models are checked into the repo so builds have zero download dependencies.

This approach also ensures consistent art style across all objects.

---

## Architecture Overview

```
scripts/generate-tree-models.mjs     ← Generator script (rename/extend for new objects)
        │
        │  Creates raw geometry (vertices, normals, indices)
        │  Groups by material color
        │  Writes binary GLB files
        ▼
public/assets/models/decorations/    ← Output GLB files
        │
        │  Registered in asset manifest
        ▼
src/config/assetManifest.ts          ← MODEL_ASSETS registry + getAssetPath()
        │
        │  Referenced by type system
        ▼
src/types/decoration.ts              ← DecorationType union + DECORATION_CONFIGS
        │
        │  Loaded and rendered
        ▼
src/components/world/decorations/    ← DecorationItem.tsx renders via useGLTF
DecorationItem.tsx                     + Suspense fallback + shadow traversal
        │
        │  Exposed in editor
        ▼
src/components/world/editor/         ← ObjectPalette.tsx shows in the UI
ObjectPalette.tsx
```

[CODE: ui/scripts/generate-tree-models.mjs]
[CODE: ui/src/config/assetManifest.ts]
[CODE: ui/src/types/decoration.ts]
[CODE: ui/src/components/world/decorations/DecorationItem.tsx]
[CODE: ui/src/components/world/editor/ObjectPalette.tsx]

---

## Step-by-Step: Adding a New Object

### 1. Define the geometry in the generator script

Add a new function to `scripts/generate-tree-models.mjs` (or create a dedicated script like `scripts/generate-props.mjs`):

```js
function createCampfire() {
  seed = 500 // Unique seed for reproducibility

  const logColor = 0x5c3a1e
  const stoneColor = 0x808080
  const emberColor = 0xff4500

  const logGeos = [
    // Logs arranged in a cross pattern
    // ... compose with createCylinder, rotateZ, translateGeometry, etc.
  ]

  const stoneGeos = [
    // Ring of stones
    // ... use createIcosphere with addNoise for organic rocks
  ]

  const emberGeos = [
    // Glowing ember bed
    // ... small noisy icospheres at ground level
  ]

  return [
    { geometries: logGeos, color: logColor, roughness: 0.9 },
    { geometries: stoneGeos, color: stoneColor, roughness: 0.95 },
    { geometries: emberGeos, color: emberColor, roughness: 0.3, metallic: 0.0 },
    // Note: emissive glow is added at render time in DecorationItem.tsx, not in the GLB
  ]
}
```

### 2. Generate the GLB

```bash
cd scenarios/prompt-manager/ui
node scripts/generate-tree-models.mjs   # Or your new script
```

Output goes to `public/assets/models/decorations/<name>.glb`.

### 3. Register in the asset manifest

In `src/config/assetManifest.ts`, add to `MODEL_ASSETS`:

```ts
campfire: {
  id: 'campfire',
  category: 'model',
  path: '/assets/models/decorations/campfire.glb',
  name: 'Campfire',
},
```

### 4. Add to the type system

In `src/types/decoration.ts`:

1. Add `'campfire'` to the `DecorationType` union
2. Add config to `DECORATION_CONFIGS`:
   ```ts
   campfire: {
     type: 'campfire',
     displayName: 'Campfire',
     emitsLight: true,       // Important for day/night behavior
     movable: true,
     size: [0.8, 0.6, 0.8], // Bounding box [width, height, depth]
     defaultY: 0,
   },
   ```
3. Add fallback color to `DEFAULT_DECORATION_COLORS`:
   ```ts
   campfire: '#ff4500',
   ```

### 5. Add rendering in DecorationItem.tsx

Add a switch case in `renderDecoration()`:

```tsx
case 'campfire': {
  const modelPath = getAssetPath('campfire')
  if (!modelPath) return <CampfireFallback castShadow={castShadow} />
  return (
    <Suspense fallback={<CampfireFallback castShadow={castShadow} />}>
      <TreeModel modelPath={modelPath} castShadow={castShadow} />
      {/* Add dynamic elements not possible in static GLB */}
      {lightOn && (
        <pointLight position={[0, 0.3, 0]} intensity={2} color="#ff6600" distance={8} decay={2} />
      )}
    </Suspense>
  )
}
```

Add preload at the bottom of the file:

```tsx
const campfirePath = getAssetPath('campfire')
if (campfirePath) useGLTF.preload(campfirePath)
```

**Key pattern**: The `TreeModel` component works for any GLB, not just trees — it loads the model via `useGLTF`, clones the scene, and enables shadows. Reuse it for all GLB-based objects.

### 6. Add to the Object Palette

In `ObjectPalette.tsx`, add to the appropriate `DECORATION_CATEGORIES` group and `DECORATION_ICONS`:

```ts
// In DECORATION_CATEGORIES:
outdoor: ['campfire', ...] as DecorationType[],

// In DECORATION_ICONS:
campfire: '🔥',
```

### 7. Type check

```bash
cd scenarios/prompt-manager/ui && npx tsc --noEmit
```

---

## Available Geometry Primitives

All primitives return `{ positions: Float32Array, normals: Float32Array, indices: Uint16Array }`.

| Primitive | Function | Parameters | Use For |
|-----------|----------|------------|---------|
| Cylinder | `createCylinder(rTop, rBot, height, segments)` | `segments`: 8-12 typical | Trunks, poles, logs, pillars |
| Cone | `createCone(radius, height, segments)` | Base at y=0, apex at y=height | **Avoid for foliage** (single-sided, see-through). Use for solid caps/tips only |
| Icosphere | `createIcosphere(radius, subdivisions)` | `subdivisions`: 1=42 verts, 2=162 verts | Foliage, rocks, organic shapes |

### Subdivision Levels

| Level | Vertices | Faces | Quality | Use When |
|-------|----------|-------|---------|----------|
| 0 | 12 | 20 | Icosahedron | Very small/distant objects |
| 1 | 42 | 80 | Low-poly | Visible faceting, retro style |
| **2** | **162** | **320** | **Smooth** | **Default for most objects** |
| 3 | 642 | 1280 | High | Close-up hero objects only |

**Recommendation**: Use subdivision 2 for most visible objects. Subdivision 1 produces noticeably faceted surfaces.

---

## Transform Helpers

Apply transforms **before** merging geometries. Order matters (scale, then rotate, then translate — or compose as needed).

| Helper | Signature | Notes |
|--------|-----------|-------|
| `translateGeometry(geo, tx, ty, tz)` | Offsets all vertices | Most common — position parts |
| `scaleGeometry(geo, sx, sy, sz)` | Non-uniform scale + normal fix | Squash/stretch shapes |
| `rotateX(geo, angle)` | Radians | Tilt forward/back |
| `rotateY(geo, angle)` | Radians | Spin left/right |
| `rotateZ(geo, angle)` | Radians | Tilt sideways — use for angled branches |
| `addNoise(geo, amount)` | Displaces along normals | **Critical for organic look** |
| `flattenBottom(geo, cutY, factor)` | Squashes vertices below cutY | Flat-bottomed foliage |

---

## The `foliageClump` Helper

This is the workhorse for creating natural-looking foliage and organic shapes:

```js
foliageClump(radius, subdivisions, noiseAmount, tx, ty, tz, squashY)
```

It composes: `createIcosphere` → `scaleGeometry(1, squashY, 1)` → `addNoise` → `translateGeometry`.

| Parameter | Typical Range | Effect |
|-----------|---------------|--------|
| `radius` | 0.3-1.3 | Size of the clump |
| `subdivisions` | 2 | Smoothness (always use 2) |
| `noiseAmount` | 0.03-0.08 | Higher = more bumpy/organic |
| `squashY` | 0.4-0.8 | Lower = flatter (pine boughs: 0.45, oak canopy: 0.7) |

---

## The `createBranch` Helper

Creates angled cylindrical branches:

```js
createBranch(length, radiusBase, radiusTip, segments, tiltAngle, yawAngle, attachY)
```

| Parameter | Description |
|-----------|-------------|
| `tiltAngle` | Radians from vertical. Negative = outward. `-0.5` to `-0.9` for natural branches |
| `yawAngle` | Rotation around Y axis (0 to 2*PI). Spread branches evenly |
| `attachY` | Height on the trunk where the branch starts |

---

## Material System

Each mesh part has a material defined by:

```js
{ geometries: [...], color: 0xRRGGBB, roughness: 0.0-1.0, metallic: 0.0-1.0 }
```

The GLB builder creates **PBR metallic-roughness** materials (glTF 2.0 standard). These map directly to Three.js `MeshStandardMaterial`.

### Material Guidelines

| Object Type | Roughness | Metallic | Example |
|-------------|-----------|----------|---------|
| Bark/wood | 0.85-0.95 | 0.0 | Tree trunks, logs |
| Foliage | 0.75-0.85 | 0.0 | Leaves, grass |
| Stone/rock | 0.9-1.0 | 0.0 | Boulders, cobblestones |
| Metal | 0.2-0.4 | 0.7-0.9 | Lamp poles, fences |
| Ceramic/glass | 0.1-0.3 | 0.0-0.2 | Vases, decorative items |
| Embers/glow | 0.2-0.4 | 0.0 | Hot coals (add emissive at render time) |

**Important**: GLB materials are static. For dynamic effects (glow, emissive animation, lights), add them in `DecorationItem.tsx` at render time using Three.js components.

---

## Tips for Good-Looking Objects

### 1. Always use noise on organic shapes

```js
// Bad: perfectly smooth sphere
const g = createIcosphere(0.5, 2)

// Good: organic, natural-looking shape
const g = createIcosphere(0.5, 2)
addNoise(g, 0.05)
```

### 2. Overlap foliage clumps densely

Sparse clumps with visible gaps look unfinished. Overlap by 30-50% for full, lush canopies. The oak tree uses 13 overlapping clumps; the pine uses 37.

### 3. Avoid hollow cones for foliage

Cone primitives are single-layer surfaces. Due to back-face culling, the inside is invisible — you can see right through them from below. Use `foliageClump` instead for any foliage shape.

### 4. Squash foliage vertically

Real canopies are wider than they are tall. Use `squashY` of 0.4-0.8:
- Pine boughs: 0.45 (very flat)
- Oak canopy: 0.7 (moderate)
- Round bushes: 0.85 (nearly spherical)

### 5. Add root flare to trunks

Trees and posts look grounded when the base is wider:

```js
// Root flare
const flare = createCylinder(trunkBaseRadius, trunkBaseRadius * 1.5, 0.2, 10)
scaleGeometry(flare, 1, 0.5, 1) // Squash vertically
```

### 6. Angle branches with rotateZ, not translate

```js
// Bad: vertical cylinder offset sideways (looks wrong)
centeredCylinder(0.03, 0.06, 0.7, 6, 0.2, 1.6, 0.1)

// Good: cylinder rotated to angle outward naturally
createBranch(0.7, 0.06, 0.03, 8, -0.6, 0.0, 1.3)
//                                    tiltAngle  yawAngle  attachHeight
```

### 7. Use seeded PRNG for reproducibility

Always reset `seed` at the start of each object function so the same model is generated every run:

```js
function createMyObject() {
  seed = 42  // Unique per object type
  // ... geometry that uses rand()/randRange()
}
```

### 8. Keep file sizes reasonable

| Object Complexity | Expected Size | Example |
|-------------------|---------------|---------|
| Simple (few parts) | 2-20 KB | Bench, lamp post |
| Medium (many parts) | 50-100 KB | Single tree, campfire |
| Complex (dense foliage) | 100-250 KB | Dense conifer, bush cluster |

If a single object exceeds 300 KB, consider reducing icosphere subdivisions from 2 to 1 for smaller/distant clumps, or reducing the number of foliage clumps.

---

## GLB Binary Format Reference

The `buildGLB()` function constructs a valid glTF 2.0 Binary (GLB) file:

```
┌──────────────────────────┐
│  Header (12 bytes)       │  magic: "glTF", version: 2, totalLength
├──────────────────────────┤
│  JSON Chunk              │  chunk header (8 bytes) + padded JSON
│  - scene graph           │  Materials, accessors, bufferViews,
│  - materials (PBR)       │  mesh primitives, node hierarchy
│  - accessors             │
├──────────────────────────┤
│  BIN Chunk               │  chunk header (8 bytes) + binary data
│  - indices (Uint16)      │  Interleaved per-mesh-part:
│  - positions (Float32)   │  indices → positions → normals
│  - normals (Float32)     │  (4-byte aligned between sections)
└──────────────────────────┘
```

You generally don't need to modify `buildGLB()`. Just define mesh parts as `{ geometries, color, roughness, metallic }` and pass them in.

---

## Rendering Pipeline Integration

### How GLB models are loaded at runtime

```
useGLTF(modelPath)           ← drei hook, loads + caches GLB
    │
    ▼
scene.clone()                ← via useMemo, allows multiple instances
    │
    ▼
traverse → castShadow=true   ← useEffect enables shadows on all meshes
    │
    ▼
<primitive object={...} />   ← React Three Fiber renders the cloned scene
```

### Suspense pattern

All GLB-based objects must be wrapped in `<Suspense>` with a procedural fallback:

```tsx
<Suspense fallback={<ProceduralFallback />}>
  <TreeModel modelPath={path} castShadow={castShadow} />
</Suspense>
```

The fallback renders instantly while the GLB loads asynchronously.

### Preloading

Add `useGLTF.preload(path)` at module scope so models start loading immediately when the component file is imported, not when the object is first placed:

```tsx
// At bottom of DecorationItem.tsx
const path = getAssetPath('campfire')
if (path) useGLTF.preload(path)
```

---

## Existing Objects as Reference

| Object | Function | Clumps | Key Techniques |
|--------|----------|--------|----------------|
| Oak tree | `createOakTree()` | 13 leaf + 7 branches | Dense overlapping foliage, angled branches, root flare |
| Pine tree | `createPineTree()` | 37 (7 tiers × ring + center) | Flat squashed clumps (0.45) in conical arrangement |
| Birch tree | `createBirchTree()` | 11 leaf + 6 branches + 7 marks | Airy sparse clumps, lenticel bark marks, delicate branches |

Study these in `scripts/generate-tree-models.mjs` before creating new objects.

---

## Related Documentation

- [DOC: docs/concepts/3D-WORLD-ARCHITECTURE.md] - Full 3D world architecture
- [DOC: docs/concepts/3D-WORLD-ARCHITECTURE.md#material-system] - Material presets and caching
- [DOC: docs/internal/SEAMS.md] - Testing boundaries
