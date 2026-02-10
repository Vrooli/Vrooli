/**
 * Generates low-poly stylized tree GLB models for the 3D world.
 * Run: node scripts/generate-tree-models.mjs
 *
 * Creates tree-oak.glb, tree-pine.glb, tree-birch.glb in public/assets/models/decorations/
 * Uses a minimal GLB builder (no browser APIs required).
 */
import { writeFileSync, mkdirSync } from 'fs'
import { join, dirname } from 'path'
import { fileURLToPath } from 'url'

const __dirname = dirname(fileURLToPath(import.meta.url))
const OUTPUT_DIR = join(__dirname, '..', 'public', 'assets', 'models', 'decorations')
mkdirSync(OUTPUT_DIR, { recursive: true })

// Seeded PRNG for reproducible "random" trees
let seed = 42
function rand() {
  seed = (seed * 16807 + 0) % 2147483647
  return (seed - 1) / 2147483646
}
function randRange(min, max) { return min + rand() * (max - min) }

// ─── Geometry Helpers ─────────────────────────────────────────────────────────

function createCylinder(radiusTop, radiusBottom, height, segments) {
  const positions = []
  const normals = []
  const indices = []

  // Two rings of vertices + 2 center vertices for caps
  for (let ring = 0; ring < 2; ring++) {
    const r = ring === 0 ? radiusBottom : radiusTop
    const y = ring === 0 ? 0 : height
    for (let i = 0; i < segments; i++) {
      const angle = (i / segments) * Math.PI * 2
      const nx = Math.cos(angle)
      const nz = Math.sin(angle)
      positions.push(nx * r, y, nz * r)
      normals.push(nx, 0, nz)
    }
  }

  // Side faces
  for (let i = 0; i < segments; i++) {
    const a = i
    const b = (i + 1) % segments
    const c = i + segments
    const d = (i + 1) % segments + segments
    indices.push(a, b, c, b, d, c)
  }

  // Bottom cap
  const bCenter = positions.length / 3
  positions.push(0, 0, 0)
  normals.push(0, -1, 0)
  for (let i = 0; i < segments; i++) {
    const next = (i + 1) % segments
    indices.push(bCenter, next, i)
  }

  // Top cap
  const tCenter = positions.length / 3
  positions.push(0, height, 0)
  normals.push(0, 1, 0)
  for (let i = 0; i < segments; i++) {
    const next = (i + 1) % segments
    indices.push(tCenter, i + segments, next + segments)
  }

  return {
    positions: new Float32Array(positions),
    normals: new Float32Array(normals),
    indices: new Uint16Array(indices),
  }
}

function createCone(radius, height, segments) {
  const positions = []
  const normals = []
  const indices = []

  // Base ring
  for (let i = 0; i < segments; i++) {
    const angle = (i / segments) * Math.PI * 2
    const nx = Math.cos(angle)
    const nz = Math.sin(angle)
    positions.push(nx * radius, 0, nz * radius)
    // Cone side normal
    const slope = radius / height
    const ny = slope
    const len = Math.sqrt(nx * nx + ny * ny + nz * nz)
    normals.push(nx / len, ny / len, nz / len)
  }

  // Apex
  const apex = positions.length / 3
  positions.push(0, height, 0)
  normals.push(0, 1, 0)

  // Side faces
  for (let i = 0; i < segments; i++) {
    const next = (i + 1) % segments
    indices.push(i, next, apex)
  }

  // Bottom cap
  const bCenter = positions.length / 3
  positions.push(0, 0, 0)
  normals.push(0, -1, 0)
  for (let i = 0; i < segments; i++) {
    const next = (i + 1) % segments
    indices.push(bCenter, next, i)
  }

  return {
    positions: new Float32Array(positions),
    normals: new Float32Array(normals),
    indices: new Uint16Array(indices),
  }
}

function createIcosphere(radius, subdivisions) {
  // Start with icosahedron
  const t = (1 + Math.sqrt(5)) / 2

  let verts = [
    [-1, t, 0], [1, t, 0], [-1, -t, 0], [1, -t, 0],
    [0, -1, t], [0, 1, t], [0, -1, -t], [0, 1, -t],
    [t, 0, -1], [t, 0, 1], [-t, 0, -1], [-t, 0, 1],
  ]

  let faces = [
    [0, 11, 5], [0, 5, 1], [0, 1, 7], [0, 7, 10], [0, 10, 11],
    [1, 5, 9], [5, 11, 4], [11, 10, 2], [10, 7, 6], [7, 1, 8],
    [3, 9, 4], [3, 4, 2], [3, 2, 6], [3, 6, 8], [3, 8, 9],
    [4, 9, 5], [2, 4, 11], [6, 2, 10], [8, 6, 7], [9, 8, 1],
  ]

  // Normalize initial vertices to unit sphere
  verts = verts.map(([x, y, z]) => {
    const len = Math.sqrt(x * x + y * y + z * z)
    return [x / len, y / len, z / len]
  })

  // Subdivide
  const midPointCache = new Map()
  function getMidPoint(a, b) {
    const key = Math.min(a, b) + '_' + Math.max(a, b)
    if (midPointCache.has(key)) return midPointCache.get(key)
    const [ax, ay, az] = verts[a]
    const [bx, by, bz] = verts[b]
    let mx = (ax + bx) / 2, my = (ay + by) / 2, mz = (az + bz) / 2
    const len = Math.sqrt(mx * mx + my * my + mz * mz)
    mx /= len; my /= len; mz /= len
    const idx = verts.length
    verts.push([mx, my, mz])
    midPointCache.set(key, idx)
    return idx
  }

  for (let s = 0; s < subdivisions; s++) {
    const newFaces = []
    for (const [a, b, c] of faces) {
      const ab = getMidPoint(a, b)
      const bc = getMidPoint(b, c)
      const ca = getMidPoint(c, a)
      newFaces.push([a, ab, ca], [b, bc, ab], [c, ca, bc], [ab, bc, ca])
    }
    faces = newFaces
    midPointCache.clear()
  }

  // Scale to radius
  const positions = new Float32Array(verts.length * 3)
  const normals = new Float32Array(verts.length * 3)
  for (let i = 0; i < verts.length; i++) {
    const [x, y, z] = verts[i]
    positions[i * 3] = x * radius
    positions[i * 3 + 1] = y * radius
    positions[i * 3 + 2] = z * radius
    // Normal = normalized position for sphere
    normals[i * 3] = x
    normals[i * 3 + 1] = y
    normals[i * 3 + 2] = z
  }

  const indices = new Uint16Array(faces.flat())

  return { positions, normals, indices }
}

// ─── Transform Helpers ────────────────────────────────────────────────────────

function translateGeometry(geo, tx, ty, tz) {
  const p = geo.positions
  for (let i = 0; i < p.length; i += 3) {
    p[i] += tx
    p[i + 1] += ty
    p[i + 2] += tz
  }
  return geo
}

function scaleGeometry(geo, sx, sy, sz) {
  const p = geo.positions
  for (let i = 0; i < p.length; i += 3) {
    p[i] *= sx
    p[i + 1] *= sy
    p[i + 2] *= sz
  }
  // Adjust normals for non-uniform scale
  const n = geo.normals
  for (let i = 0; i < n.length; i += 3) {
    n[i] /= sx
    n[i + 1] /= sy
    n[i + 2] /= sz
    const len = Math.sqrt(n[i] ** 2 + n[i + 1] ** 2 + n[i + 2] ** 2)
    if (len > 0) { n[i] /= len; n[i + 1] /= len; n[i + 2] /= len }
  }
  return geo
}

/** Rotate geometry around Y axis */
function rotateY(geo, angle) {
  const p = geo.positions
  const n = geo.normals
  const c = Math.cos(angle), s = Math.sin(angle)
  for (let i = 0; i < p.length; i += 3) {
    const x = p[i], z = p[i + 2]
    p[i] = x * c + z * s
    p[i + 2] = -x * s + z * c
    const nx = n[i], nz = n[i + 2]
    n[i] = nx * c + nz * s
    n[i + 2] = -nx * s + nz * c
  }
  return geo
}

/** Rotate geometry around Z axis (tilts branches outward) */
function rotateZ(geo, angle) {
  const p = geo.positions
  const n = geo.normals
  const c = Math.cos(angle), s = Math.sin(angle)
  for (let i = 0; i < p.length; i += 3) {
    const x = p[i], y = p[i + 1]
    p[i] = x * c - y * s
    p[i + 1] = x * s + y * c
    const nx = n[i], ny = n[i + 1]
    n[i] = nx * c - ny * s
    n[i + 1] = nx * s + ny * c
  }
  return geo
}

/** Rotate geometry around X axis */
function rotateX(geo, angle) {
  const p = geo.positions
  const n = geo.normals
  const c = Math.cos(angle), s = Math.sin(angle)
  for (let i = 0; i < p.length; i += 3) {
    const y = p[i + 1], z = p[i + 2]
    p[i + 1] = y * c - z * s
    p[i + 2] = y * s + z * c
    const ny = n[i + 1], nz = n[i + 2]
    n[i + 1] = ny * c - nz * s
    n[i + 2] = ny * s + nz * c
  }
  return geo
}

/** Add noise/jitter to vertex positions for organic look */
function addNoise(geo, amount) {
  const p = geo.positions
  const n = geo.normals
  for (let i = 0; i < p.length; i += 3) {
    // Displace along normal for more natural bulging
    const disp = (rand() - 0.5) * 2 * amount
    p[i] += n[i] * disp
    p[i + 1] += n[i + 1] * disp
    p[i + 2] += n[i + 2] * disp
  }
  return geo
}

/** Squash the bottom of a sphere to make it more flat-bottomed (like foliage clump) */
function flattenBottom(geo, cutY, factor) {
  const p = geo.positions
  for (let i = 0; i < p.length; i += 3) {
    if (p[i + 1] < cutY) {
      p[i + 1] = cutY + (p[i + 1] - cutY) * factor
    }
  }
  return geo
}

/** Create a branch: cylinder going from origin upward, then rotated to angle outward */
function createBranch(length, radiusBase, radiusTip, segments, tiltAngle, yawAngle, attachY) {
  const g = createCylinder(radiusTip, radiusBase, length, segments)
  // Tilt outward from vertical
  rotateZ(g, tiltAngle)
  // Rotate around Y for placement angle
  rotateY(g, yawAngle)
  // Move to attachment point
  translateGeometry(g, 0, attachY, 0)
  return g
}

// ─── GLB Builder ──────────────────────────────────────────────────────────────

/** Merge multiple geometries sharing the same material into one */
function mergeGeometries(geos) {
  let totalVerts = 0, totalIdx = 0
  for (const g of geos) {
    totalVerts += g.positions.length / 3
    totalIdx += g.indices.length
  }
  const positions = new Float32Array(totalVerts * 3)
  const normals = new Float32Array(totalVerts * 3)
  const indices = new Uint16Array(totalIdx)
  let vOffset = 0, iOffset = 0
  for (const g of geos) {
    const nv = g.positions.length / 3
    positions.set(g.positions, vOffset * 3)
    normals.set(g.normals, vOffset * 3)
    for (let i = 0; i < g.indices.length; i++) {
      indices[iOffset + i] = g.indices[i] + vOffset
    }
    vOffset += nv
    iOffset += g.indices.length
  }
  return { positions, normals, indices }
}

function colorToRGBA(hex) {
  const r = ((hex >> 16) & 0xff) / 255
  const g = ((hex >> 8) & 0xff) / 255
  const b = (hex & 0xff) / 255
  return [r, g, b, 1.0]
}

/** Build a GLB buffer from meshParts: [{ geometries: Geometry[], color: number }] */
function buildGLB(meshParts) {
  // Merge each part's geometries
  const mergedParts = meshParts.map((part) => ({
    geo: mergeGeometries(part.geometries),
    color: part.color,
    roughness: part.roughness ?? 0.85,
    metallic: part.metallic ?? 0.0,
  }))

  // Build binary buffer
  const bufferChunks = []
  const bufferViews = []
  const accessors = []
  let byteOffset = 0

  for (const part of mergedParts) {
    const { positions, normals, indices } = part.geo

    // Indices bufferView
    const idxBytes = Buffer.from(indices.buffer.slice(indices.byteOffset, indices.byteOffset + indices.byteLength))
    bufferViews.push({ buffer: 0, byteOffset, byteLength: idxBytes.length, target: 34963 })
    let idxMin = Infinity, idxMax = -Infinity
    for (let i = 0; i < indices.length; i++) { idxMin = Math.min(idxMin, indices[i]); idxMax = Math.max(idxMax, indices[i]) }
    accessors.push({
      bufferView: bufferViews.length - 1, byteOffset: 0,
      componentType: 5123, count: indices.length, type: 'SCALAR',
      min: [idxMin], max: [idxMax],
    })
    bufferChunks.push(idxBytes)
    byteOffset += idxBytes.length
    // Pad to 4 bytes
    const idxPad = (4 - (byteOffset % 4)) % 4
    if (idxPad > 0) { bufferChunks.push(Buffer.alloc(idxPad)); byteOffset += idxPad }

    // Positions bufferView
    const posBytes = Buffer.from(positions.buffer.slice(positions.byteOffset, positions.byteOffset + positions.byteLength))
    bufferViews.push({ buffer: 0, byteOffset, byteLength: posBytes.length, target: 34962 })
    const posMin = [Infinity, Infinity, Infinity], posMax = [-Infinity, -Infinity, -Infinity]
    for (let i = 0; i < positions.length; i += 3) {
      for (let j = 0; j < 3; j++) { posMin[j] = Math.min(posMin[j], positions[i + j]); posMax[j] = Math.max(posMax[j], positions[i + j]) }
    }
    accessors.push({
      bufferView: bufferViews.length - 1, byteOffset: 0,
      componentType: 5126, count: positions.length / 3, type: 'VEC3',
      min: posMin, max: posMax,
    })
    bufferChunks.push(posBytes)
    byteOffset += posBytes.length

    // Normals bufferView
    const normBytes = Buffer.from(normals.buffer.slice(normals.byteOffset, normals.byteOffset + normals.byteLength))
    bufferViews.push({ buffer: 0, byteOffset, byteLength: normBytes.length, target: 34962 })
    accessors.push({
      bufferView: bufferViews.length - 1, byteOffset: 0,
      componentType: 5126, count: normals.length / 3, type: 'VEC3',
    })
    bufferChunks.push(normBytes)
    byteOffset += normBytes.length

    // Pad to 4 bytes
    const normPad = (4 - (byteOffset % 4)) % 4
    if (normPad > 0) { bufferChunks.push(Buffer.alloc(normPad)); byteOffset += normPad }

    part._accessors = {
      indices: accessors.length - 3,
      position: accessors.length - 2,
      normal: accessors.length - 1,
    }
  }

  const totalBinLength = byteOffset

  // Build glTF JSON
  const materials = mergedParts.map((part) => {
    const [r, g, b] = colorToRGBA(part.color)
    return {
      pbrMetallicRoughness: {
        baseColorFactor: [r, g, b, 1.0],
        roughnessFactor: part.roughness,
        metallicFactor: part.metallic,
      },
    }
  })

  const meshPrimitives = mergedParts.map((part, i) => ({
    attributes: { POSITION: part._accessors.position, NORMAL: part._accessors.normal },
    indices: part._accessors.indices,
    material: i,
  }))

  const gltfJson = {
    asset: { version: '2.0', generator: 'vrooli-tree-gen' },
    scene: 0,
    scenes: [{ nodes: [0] }],
    nodes: [{ mesh: 0, name: 'Tree' }],
    meshes: [{ primitives: meshPrimitives, name: 'TreeMesh' }],
    materials,
    accessors,
    bufferViews,
    buffers: [{ byteLength: totalBinLength }],
  }

  const jsonStr = JSON.stringify(gltfJson)
  // Pad JSON to 4-byte alignment
  const jsonPad = (4 - (jsonStr.length % 4)) % 4
  const jsonPadded = jsonStr + ' '.repeat(jsonPad)
  const jsonBuf = Buffer.from(jsonPadded, 'utf8')
  const binBuf = Buffer.concat(bufferChunks)

  // GLB header: magic(4) + version(4) + length(4) = 12
  // JSON chunk: length(4) + type(4) + data = 8 + jsonBuf.length
  // BIN chunk: length(4) + type(4) + data = 8 + binBuf.length
  const totalLength = 12 + 8 + jsonBuf.length + 8 + binBuf.length

  const header = Buffer.alloc(12)
  header.writeUInt32LE(0x46546C67, 0) // "glTF"
  header.writeUInt32LE(2, 4)          // version
  header.writeUInt32LE(totalLength, 8)

  const jsonChunkHeader = Buffer.alloc(8)
  jsonChunkHeader.writeUInt32LE(jsonBuf.length, 0)
  jsonChunkHeader.writeUInt32LE(0x4E4F534A, 4) // "JSON"

  const binChunkHeader = Buffer.alloc(8)
  binChunkHeader.writeUInt32LE(binBuf.length, 0)
  binChunkHeader.writeUInt32LE(0x004E4942, 4) // "BIN\0"

  return Buffer.concat([header, jsonChunkHeader, jsonBuf, binChunkHeader, binBuf])
}

// ─── Foliage Helpers ─────────────────────────────────────────────────────────

/** Create a foliage clump: noisy, slightly squashed icosphere */
function foliageClump(radius, subdivisions, noiseAmount, tx, ty, tz, squashY = 0.8) {
  const g = createIcosphere(radius, subdivisions)
  scaleGeometry(g, 1, squashY, 1) // Flatten vertically for natural canopy shape
  addNoise(g, noiseAmount)
  translateGeometry(g, tx, ty, tz)
  return g
}

// ─── Tree Definitions ─────────────────────────────────────────────────────────

/** Oak Tree - broad deciduous tree ~3.5-4 units tall */
function createOakTree() {
  seed = 42

  const trunkColor = 0x5c3a1e
  const leafColor = 0x2d6b1e // Slightly brighter green

  // Trunk with root flare at base
  const trunkGeos = [
    // Main trunk (tapers upward)
    createCylinder(0.12, 0.22, 1.8, 10),
    // Root flare - wider base
    (() => {
      const g = createCylinder(0.22, 0.30, 0.3, 10)
      scaleGeometry(g, 1, 0.6, 1) // Squash vertically
      return g
    })(),
  ]
  // Move trunk up so base is at y=0
  // (cylinder starts at y=0, no translation needed for base)

  // Branches angling outward at various angles
  const branchGeos = [
    // Main branches
    createBranch(1.0, 0.07, 0.03, 8, -0.6, 0.0, 1.3),
    createBranch(0.9, 0.06, 0.025, 8, -0.7, Math.PI * 0.6, 1.2),
    createBranch(0.85, 0.06, 0.025, 8, -0.5, Math.PI * 1.2, 1.4),
    createBranch(0.8, 0.05, 0.02, 8, -0.8, Math.PI * 1.7, 1.1),
    // Secondary smaller branches
    createBranch(0.6, 0.04, 0.015, 6, -0.9, Math.PI * 0.3, 1.5),
    createBranch(0.55, 0.035, 0.015, 6, -0.6, Math.PI * 0.9, 1.55),
    createBranch(0.5, 0.035, 0.015, 6, -0.7, Math.PI * 1.5, 1.6),
  ]

  // Dense canopy with many overlapping foliage clumps (subdivision 2 for smooth)
  const leafGeos = [
    // Central large canopy mass
    foliageClump(1.3, 2, 0.08, 0.0, 2.9, 0.0, 0.7),
    // Ring of medium clumps around the center
    foliageClump(0.95, 2, 0.06, 0.7, 2.6, 0.3, 0.75),
    foliageClump(0.9, 2, 0.06, -0.6, 2.7, 0.5, 0.75),
    foliageClump(0.85, 2, 0.06, -0.5, 2.5, -0.6, 0.75),
    foliageClump(0.9, 2, 0.06, 0.5, 2.5, -0.5, 0.75),
    foliageClump(0.85, 2, 0.06, 0.3, 2.8, 0.7, 0.75),
    foliageClump(0.8, 2, 0.06, -0.7, 2.8, -0.2, 0.75),
    // Top clumps
    foliageClump(0.7, 2, 0.05, 0.15, 3.4, 0.1, 0.7),
    foliageClump(0.6, 2, 0.05, -0.2, 3.3, -0.2, 0.7),
    foliageClump(0.55, 2, 0.05, 0.3, 3.2, -0.3, 0.7),
    // Lower drooping clumps
    foliageClump(0.7, 2, 0.05, 0.8, 2.2, 0.0, 0.8),
    foliageClump(0.65, 2, 0.05, -0.3, 2.1, 0.7, 0.8),
    foliageClump(0.6, 2, 0.05, 0.0, 2.2, -0.8, 0.8),
  ]

  return [
    { geometries: [...trunkGeos, ...branchGeos], color: trunkColor, roughness: 0.9 },
    { geometries: leafGeos, color: leafColor, roughness: 0.8 },
  ]
}

/** Pine Tree - tall conifer ~4.5-5 units tall.
 *  Uses solid icosphere foliage clumps arranged in conical tiers. */
function createPineTree() {
  seed = 137

  const trunkColor = 0x4a2f15
  const leafColor = 0x1a5d2e // Richer green

  // Trunk visible at bottom, tapering
  const trunkGeos = [
    createCylinder(0.06, 0.16, 4.5, 10),
    // Root flare
    (() => {
      const g = createCylinder(0.16, 0.22, 0.2, 10)
      scaleGeometry(g, 1, 0.5, 1)
      return g
    })(),
  ]

  // Solid foliage clumps arranged in conical tiers (no see-through cones).
  // Each tier is a ring of horizontally-flattened icospheres at a given height,
  // with ring radius and clump size decreasing upward to form the conical shape.
  const leafGeos = []

  const tiers = [
    // { y, ringRadius, clumpRadius, count } — count = clumps per ring
    { y: 1.0, ringRadius: 1.05, clumpRadius: 0.65, count: 6 },
    { y: 1.5, ringRadius: 0.85, clumpRadius: 0.58, count: 5 },
    { y: 2.0, ringRadius: 0.68, clumpRadius: 0.52, count: 5 },
    { y: 2.5, ringRadius: 0.52, clumpRadius: 0.45, count: 4 },
    { y: 3.0, ringRadius: 0.35, clumpRadius: 0.38, count: 4 },
    { y: 3.5, ringRadius: 0.2, clumpRadius: 0.32, count: 3 },
    { y: 4.0, ringRadius: 0.08, clumpRadius: 0.25, count: 2 },
  ]

  for (const tier of tiers) {
    // Center clump at each tier
    leafGeos.push(foliageClump(tier.clumpRadius * 0.9, 2, 0.04, 0, tier.y, 0, 0.5))
    // Ring of clumps
    for (let i = 0; i < tier.count; i++) {
      const angle = (i / tier.count) * Math.PI * 2 + tier.y * 0.7 // Offset per tier for natural look
      const x = Math.cos(angle) * tier.ringRadius
      const z = Math.sin(angle) * tier.ringRadius
      const yJitter = (rand() - 0.5) * 0.15
      leafGeos.push(foliageClump(tier.clumpRadius, 2, 0.04, x, tier.y + yJitter, z, 0.45))
    }
  }
  // Crown tip
  leafGeos.push(foliageClump(0.2, 2, 0.03, 0, 4.4, 0, 0.6))

  return [
    { geometries: trunkGeos, color: trunkColor, roughness: 0.9 },
    { geometries: leafGeos, color: leafColor, roughness: 0.8 },
  ]
}

/** Birch Tree - slender white-barked tree ~3-3.5 units tall */
function createBirchTree() {
  seed = 271

  const trunkColor = 0xe8dcc8  // White bark
  const markColor = 0x3a3a3a   // Dark lenticels
  const leafColor = 0x6aaf3c   // Bright spring green

  // Slender trunk
  const trunkGeos = [
    createCylinder(0.05, 0.10, 2.8, 10),
    // Slight root flare
    (() => {
      const g = createCylinder(0.10, 0.14, 0.15, 10)
      scaleGeometry(g, 1, 0.5, 1)
      return g
    })(),
  ]

  // Bark marks (horizontal lenticel bands) - wider/taller for visibility
  const markGeos = []
  const markYPositions = [0.3, 0.6, 0.95, 1.25, 1.6, 1.9, 2.2]
  for (const my of markYPositions) {
    // Each mark is a thin ring slightly larger than trunk at that height
    const trunkR = 0.10 - (my / 2.8) * 0.05 // Trunk tapers
    const g = createCylinder(trunkR + 0.008, trunkR + 0.008, 0.015, 10)
    translateGeometry(g, 0, my, 0)
    markGeos.push(g)
  }

  // Graceful branches angling upward and slightly out (birch branches droop less)
  const branchGeos = [
    // Upper branches
    createBranch(0.7, 0.03, 0.01, 6, -0.5, 0.0, 2.0),
    createBranch(0.65, 0.025, 0.01, 6, -0.6, Math.PI * 0.5, 2.1),
    createBranch(0.6, 0.025, 0.01, 6, -0.4, Math.PI * 1.0, 2.2),
    createBranch(0.55, 0.02, 0.008, 6, -0.7, Math.PI * 1.5, 1.9),
    // Mid branches
    createBranch(0.5, 0.02, 0.008, 6, -0.8, Math.PI * 0.3, 1.6),
    createBranch(0.45, 0.02, 0.008, 6, -0.5, Math.PI * 1.3, 1.7),
  ]

  // Delicate, airy foliage clusters - smaller and more spread out than oak
  const leafGeos = [
    // Crown cluster
    foliageClump(0.55, 2, 0.06, 0.0, 3.0, 0.0, 0.75),
    // Upper ring
    foliageClump(0.45, 2, 0.05, 0.4, 2.8, 0.2, 0.8),
    foliageClump(0.4, 2, 0.05, -0.35, 2.9, -0.15, 0.8),
    foliageClump(0.42, 2, 0.05, 0.1, 2.7, -0.4, 0.8),
    foliageClump(0.38, 2, 0.05, -0.2, 2.85, 0.35, 0.8),
    // Mid ring (where branches end)
    foliageClump(0.5, 2, 0.05, 0.5, 2.4, 0.3, 0.8),
    foliageClump(0.45, 2, 0.05, -0.45, 2.5, 0.4, 0.8),
    foliageClump(0.4, 2, 0.05, -0.3, 2.3, -0.5, 0.8),
    foliageClump(0.42, 2, 0.05, 0.35, 2.5, -0.35, 0.8),
    // Lower sparse clusters
    foliageClump(0.35, 2, 0.04, 0.55, 2.1, -0.1, 0.85),
    foliageClump(0.3, 2, 0.04, -0.5, 2.15, -0.2, 0.85),
  ]

  return [
    { geometries: trunkGeos, color: trunkColor, roughness: 0.6 },
    { geometries: markGeos, color: markColor, roughness: 0.8 },
    { geometries: branchGeos, color: 0xd4c4a8, roughness: 0.7 },
    { geometries: leafGeos, color: leafColor, roughness: 0.75 },
  ]
}

// ─── Main ─────────────────────────────────────────────────────────────────────

console.log('Generating tree models...')

const trees = [
  { parts: createOakTree(), file: 'tree-oak.glb' },
  { parts: createPineTree(), file: 'tree-pine.glb' },
  { parts: createBirchTree(), file: 'tree-birch.glb' },
]

for (const { parts, file } of trees) {
  const glb = buildGLB(parts)
  const outPath = join(OUTPUT_DIR, file)
  writeFileSync(outPath, glb)
  console.log(`  Written: ${outPath} (${(glb.length / 1024).toFixed(1)} KB)`)
}

console.log('Done! Models ready for use in the 3D world.')
