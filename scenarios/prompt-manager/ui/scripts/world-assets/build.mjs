#!/usr/bin/env node
/**
 * World asset pipeline.
 *
 * For every prop each scene names (src/world/config/scenes/*.json), take the
 * source GLB from assets-src/world (mapped by sources.json), optimise it with
 * gltf-transform (join, palette, weld, prune, meshopt) into
 * public/assets/world/<scene>/<prop>.glb, and write
 * src/world/engine/assets/registry.generated.json with path, bounds and
 * triangle count. Fails when a prop exceeds tuning.budgets.propTriangles or
 * when a scene names a prop with no source.
 *
 *   pnpm world:assets            build everything
 *   pnpm world:assets --check    verify outputs and registry are current
 */
import { execFileSync } from 'node:child_process'
import { existsSync, mkdirSync, readFileSync, readdirSync, statSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'

const uiRoot = resolve(import.meta.dirname, '..', '..')
const srcRoot = resolve(uiRoot, 'assets-src', 'world')
const outRoot = resolve(uiRoot, 'public', 'assets', 'world')
const registryPath = resolve(uiRoot, 'src', 'world', 'engine', 'assets', 'registry.generated.json')
const tuning = JSON.parse(readFileSync(resolve(uiRoot, 'src/world/config/world.tuning.json'), 'utf8'))
const sources = JSON.parse(readFileSync(resolve(srcRoot, 'sources.json'), 'utf8'))
const checkOnly = process.argv.includes('--check')
const gltfTransform = resolve(uiRoot, 'node_modules', '.bin', 'gltf-transform')

function sceneFiles() {
  const dir = resolve(uiRoot, 'src/world/config/scenes')
  return readdirSync(dir).filter((f) => f.endsWith('.json')).map((f) => JSON.parse(readFileSync(resolve(dir, f), 'utf8')))
}

function propIds(scene) {
  const p = scene.props
  const named = [p.desk, p.chair, p.table, p.seat, p.hearth, p.lamp, p.board, p.door, ...(p.filler ?? [])].filter(Boolean)
  // Biome vegetation also resolves through this registry. Build the full
  // governed source map, while retaining scene prop ids so missing mappings
  // still fail below instead of silently dropping a configured prop.
  return [...new Set([...named, ...Object.keys(sources.props[scene.id] ?? {})])]
}

/** Read bounds and triangle count straight from the GLB JSON chunk; no runtime dependency. */
function inspectGlb(path) {
  const buf = readFileSync(path)
  if (buf.readUInt32LE(0) !== 0x46546c67) throw new Error(`${path}: not a GLB`)
  const jsonLength = buf.readUInt32LE(12)
  const json = JSON.parse(buf.subarray(20, 20 + jsonLength).toString('utf8'))
  const min = [Infinity, Infinity, Infinity]
  const max = [-Infinity, -Infinity, -Infinity]
  let triangles = 0
  const nodes = json.nodes ?? []
  const roots = json.scenes?.[json.scene ?? 0]?.nodes ?? nodes.map((_, i) => i)
  const visit = (index, parent) => {
    const node = nodes[index]
    if (!node) return
    const world = multiply(parent, nodeMatrix(node))
    if (node.mesh !== undefined) {
      for (const prim of json.meshes?.[node.mesh]?.primitives ?? []) {
        const pos = json.accessors?.[prim.attributes?.POSITION]
        if (pos?.min && pos?.max) {
          // KHR_mesh_quantization: normalized integer accessors report min/max in the
          // integer domain; three divides by the type's max before the node transform.
          const divisor = pos.normalized ? NORMALIZED_DIVISOR[pos.componentType] ?? 1 : 1
          for (const corner of corners(pos.min.map((v) => v / divisor), pos.max.map((v) => v / divisor))) {
            const p = apply(world, corner)
            for (let i = 0; i < 3; i += 1) {
              min[i] = Math.min(min[i], p[i])
              max[i] = Math.max(max[i], p[i])
            }
          }
        }
        const count = prim.indices !== undefined ? json.accessors?.[prim.indices]?.count ?? 0 : pos?.count ?? 0
        triangles += Math.floor(count / 3)
      }
    }
    for (const child of node.children ?? []) visit(child, world)
  }
  for (const root of roots) visit(root, identity())
  return { min, max, triangles, materials: (json.materials ?? []).length, meshes: (json.meshes ?? []).length }
}

const NORMALIZED_DIVISOR = { 5120: 127, 5121: 255, 5122: 32767, 5123: 65535 }
const LIT_METALLIC = 0
const LIT_ROUGHNESS = 0.9

/**
 * Kenney kits export KHR_materials_unlit, which three loads as MeshBasicMaterial:
 * no lighting, no shadows. Rewrite the GLB's JSON chunk so every material is a
 * lit PBR material with the same base colour.
 */
function litMaterials(path) {
  const buf = readFileSync(path)
  const jsonLength = buf.readUInt32LE(12)
  const json = JSON.parse(buf.subarray(20, 20 + jsonLength).toString('utf8'))
  let changed = false
  for (const material of json.materials ?? []) {
    if (material.extensions?.KHR_materials_unlit) {
      delete material.extensions.KHR_materials_unlit
      if (Object.keys(material.extensions).length === 0) delete material.extensions
      changed = true
    }
    material.pbrMetallicRoughness = { ...(material.pbrMetallicRoughness ?? {}), metallicFactor: LIT_METALLIC, roughnessFactor: LIT_ROUGHNESS }
    changed = true
  }
  for (const key of ['extensionsUsed', 'extensionsRequired']) {
    if (Array.isArray(json[key])) {
      const next = json[key].filter((e) => e !== 'KHR_materials_unlit')
      if (next.length !== json[key].length) changed = true
      if (next.length === 0) delete json[key]
      else json[key] = next
    }
  }
  if (!changed) return
  let text = Buffer.from(JSON.stringify(json), 'utf8')
  const pad = (4 - (text.length % 4)) % 4
  if (pad) text = Buffer.concat([text, Buffer.alloc(pad, 0x20)])
  const rest = buf.subarray(20 + jsonLength)
  const header = Buffer.alloc(20)
  header.writeUInt32LE(0x46546c67, 0)
  header.writeUInt32LE(2, 4)
  header.writeUInt32LE(20 + text.length + rest.length, 8)
  header.writeUInt32LE(text.length, 12)
  header.writeUInt32LE(0x4e4f534a, 16)
  writeFileSync(path, Buffer.concat([header, text, rest]))
}

function identity() {
  return [1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1]
}

/** Column-major 4x4 from a glTF node's matrix or TRS. */
function nodeMatrix(node) {
  if (node.matrix) return node.matrix
  const [tx, ty, tz] = node.translation ?? [0, 0, 0]
  const [qx, qy, qz, qw] = node.rotation ?? [0, 0, 0, 1]
  const [sx, sy, sz] = node.scale ?? [1, 1, 1]
  const xx = qx * qx, yy = qy * qy, zz = qz * qz, xy = qx * qy, xz = qx * qz, yz = qy * qz, wx = qw * qx, wy = qw * qy, wz = qw * qz
  return [
    (1 - 2 * (yy + zz)) * sx, 2 * (xy + wz) * sx, 2 * (xz - wy) * sx, 0,
    2 * (xy - wz) * sy, (1 - 2 * (xx + zz)) * sy, 2 * (yz + wx) * sy, 0,
    2 * (xz + wy) * sz, 2 * (yz - wx) * sz, (1 - 2 * (xx + yy)) * sz, 0,
    tx, ty, tz, 1,
  ]
}

function multiply(a, b) {
  const out = new Array(16).fill(0)
  for (let col = 0; col < 4; col += 1) {
    for (let row = 0; row < 4; row += 1) {
      let sum = 0
      for (let k = 0; k < 4; k += 1) sum += a[k * 4 + row] * b[col * 4 + k]
      out[col * 4 + row] = sum
    }
  }
  return out
}

function apply(m, [x, y, z]) {
  return [m[0] * x + m[4] * y + m[8] * z + m[12], m[1] * x + m[5] * y + m[9] * z + m[13], m[2] * x + m[6] * y + m[10] * z + m[14]]
}

function corners(min, max) {
  const out = []
  for (const x of [min[0], max[0]]) for (const y of [min[1], max[1]]) for (const z of [min[2], max[2]]) out.push([x, y, z])
  return out
}

const registry = { generatedAt: new Date().toISOString(), props: {} }
const failures = []
for (const scene of sceneFiles()) {
  const map = sources.props[scene.id] ?? {}
  mkdirSync(resolve(outRoot, scene.assetSet), { recursive: true })
  for (const id of propIds(scene)) {
    const source = map[id]
    if (!source) {
      failures.push(`${scene.id}: prop "${id}" has no entry in assets-src/world/sources.json`)
      continue
    }
    const src = resolve(srcRoot, source)
    if (!existsSync(src)) {
      failures.push(`${scene.id}/${id}: missing source ${source}`)
      continue
    }
    const out = resolve(outRoot, scene.assetSet, `${id}.glb`)
    const stale = !existsSync(out) || statSync(out).mtimeMs < statSync(src).mtimeMs
    if (stale) {
      if (checkOnly) {
        failures.push(`${scene.id}/${id}: output missing or older than source; run pnpm world:assets`)
        continue
      }
      execFileSync(gltfTransform, [
        'optimize', src, out,
        '--join', 'true', '--join-meshes', 'true', '--join-named', 'true',
        '--palette', 'true', '--weld', 'true', '--prune', 'true', '--flatten', 'true',
        '--compress', 'meshopt', '--simplify', 'false', '--instance', 'false', '--texture-compress', 'false',
      ], { stdio: 'pipe' })
    }
    litMaterials(out)
    const info = inspectGlb(out)
    if (info.triangles > tuning.budgets.propTriangles) {
      failures.push(`${scene.id}/${id}: ${info.triangles} triangles exceeds budgets.propTriangles ${tuning.budgets.propTriangles}`)
    }
    const key = `${scene.assetSet}/${id}`
    registry.props[key] = {
      scene: scene.id,
      id,
      path: `${scene.assetSet}/${id}.glb`,
      source,
      kit: source.split('/')[0],
      size: info.max.map((v, i) => Number((v - info.min[i]).toFixed(4))),
      bounds: { min: info.min.map((v) => Number(v.toFixed(4))), max: info.max.map((v) => Number(v.toFixed(4))) },
      triangles: info.triangles,
      materials: info.materials,
      bytes: statSync(out).size,
    }
  }
}

const rendered = `${JSON.stringify({ ...registry, generatedAt: undefined, props: Object.fromEntries(Object.entries(registry.props).sort(([a], [b]) => a.localeCompare(b))) }, null, 2)}\n`
if (checkOnly) {
  const current = existsSync(registryPath) ? readFileSync(registryPath, 'utf8') : ''
  if (current !== rendered) failures.push('registry.generated.json is stale; run pnpm world:assets')
} else {
  writeFileSync(registryPath, rendered)
}

for (const f of failures) console.error(`✗ ${f}`)
const total = Object.values(registry.props).reduce((sum, p) => sum + p.bytes, 0)
console.log(`${Object.keys(registry.props).length} props, ${(total / 1024).toFixed(0)} KB under public/assets/world`)
process.exit(failures.length > 0 ? 1 : 0)
