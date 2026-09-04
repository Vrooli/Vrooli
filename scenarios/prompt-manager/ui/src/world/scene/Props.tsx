import { Instance, Instances } from '@react-three/drei'
import { useFrame, useThree } from '@react-three/fiber'
import { useEffect, useMemo, useRef, type RefObject } from 'react'
import { Color, Frustum, InstancedBufferAttribute, InstancedMesh, Matrix4, MeshStandardMaterial, Object3D, Vector3, type Material } from 'three'
import type { LayoutTuning, LightingPeriod, Scene } from '../config'
import { propRecord, usePropParts, type PropRecord } from '../engine/assets'
import { heightAt, type Place, type PlaceKind, type TerrainField, type Vec2 } from '../sim'
import { interiorFor } from '../sim/layout/interior'
import { useWorldStore } from './WorldStoreContext'
import { cullVegetation, matrixBufferChanged, type VegetationCullItem } from './vegetationCull'

export interface Placement {
  key: string
  position: Vec2
  y?: number
  rotation: number
  scale: number
  color?: readonly [number, number, number]
}

export interface CulledPlacement extends Placement {
  radius: number
}

interface PropInstancesProps {
  record: PropRecord
  placements: Placement[]
  scale: number
  castShadow?: boolean
  /** Emissive glow for lamps and fire: colour and intensity from the lighting period. */
  emissive?: { color: string; intensity: number }
  frustumCulled?: boolean
}

const BLOOM_THRESHOLD = 1

/** A material copy with emissive set; above the bloom threshold it bypasses tone mapping so bloom catches it. */
function glowing(material: Material, emissive: { color: string; intensity: number }): Material {
  if (!(material instanceof MeshStandardMaterial)) return material
  const copy = material.clone()
  copy.emissive = new Color(emissive.color)
  copy.emissiveIntensity = emissive.intensity
  copy.toneMapped = emissive.intensity < BLOOM_THRESHOLD
  return copy
}

/** One instanced draw per material part of one prop id; the prop's own origin sits on the ground plane. */
export function PropInstances({ record, placements, scale, castShadow = true, emissive, frustumCulled = false }: PropInstancesProps) {
  const parts = usePropParts(record)
  const lift = -record.bounds.min[1] * scale
  const materials = useMemo(
    () => parts.map((part) => (emissive ? glowing(part.material, emissive) : part.material)),
    [parts, emissive],
  )
  useEffect(() => () => {
    if (emissive) for (const m of materials) m.dispose()
  }, [materials, emissive])
  if (placements.length === 0) return null
  return (
    <>
      {parts.map((part, index) => (
        <Instances
          key={`${record.id}:${index}`}
          geometry={part.geometry}
          material={materials[index] ?? part.material}
          limit={Math.max(placements.length, 1)}
          castShadow={castShadow}
          receiveShadow
          frustumCulled={frustumCulled}
        >
          {placements.map((p) => (
            <Instance key={p.key} position={[p.position[0], (p.y ?? 0) + lift, p.position[1]]} rotation={[0, p.rotation, 0]} scale={scale * p.scale} />
          ))}
        </Instances>
      ))}
    </>
  )
}

/** One draw per material part, with camera visibility owned by the CPU cull. */
export function CulledPropInstances({ record, placements, scale, budget, visibleKeys }: { record: PropRecord; placements: CulledPlacement[]; scale: number; budget: number; visibleKeys?: RefObject<ReadonlySet<string>> }) {
  const parts = usePropParts(record)
  const meshes = useRef<Array<InstancedMesh | null>>([])
  const camera = useThree((state) => state.camera)
  const lift = -record.bounds.min[1] * scale
  const items = useMemo<VegetationCullItem[]>(() => {
    const dummy = new Object3D()
    return placements.map((placement) => {
      dummy.position.set(placement.position[0], (placement.y ?? 0) + lift, placement.position[1])
      dummy.rotation.set(0, placement.rotation, 0)
      dummy.scale.setScalar(scale * placement.scale)
      dummy.updateMatrix()
      return {
        key: placement.key,
        center: [dummy.position.x, dummy.position.y + (record.size[1] * scale * placement.scale) / 2, dummy.position.z],
        radius: placement.radius,
        matrix: new Float32Array(dummy.matrix.elements),
        color: placement.color,
      }
    })
  }, [lift, placements, record.size, scale])
  const capacity = Math.min(items.length, budget)
  const next = useMemo(() => new Float32Array(Math.max(capacity, 1) * 16), [capacity])
  const uploaded = useMemo(() => new Float32Array(Math.max(capacity, 1) * 16), [capacity])
  const colors = useMemo(() => new Float32Array(Math.max(capacity, 1) * 3), [capacity])
  const previousCount = useRef(-1)
  const frustum = useMemo(() => new Frustum(), [])
  const viewProjection = useMemo(() => new Matrix4(), [])
  const cameraPosition = useMemo(() => new Vector3(), [])

  useFrame(() => {
    viewProjection.multiplyMatrices(camera.projectionMatrix, camera.matrixWorldInverse)
    frustum.setFromProjectionMatrix(viewProjection)
    camera.getWorldPosition(cameraPosition)
    const count = cullVegetation(items, frustum, cameraPosition, next, capacity, visibleKeys?.current, colors)
    if (!matrixBufferChanged(uploaded, next, count, previousCount.current)) return
    uploaded.set(next.subarray(0, count * 16), 0)
    for (const mesh of meshes.current) {
      if (!mesh) continue
      mesh.count = count
      mesh.instanceMatrix.array.set(next.subarray(0, count * 16))
      mesh.instanceMatrix.needsUpdate = true
      mesh.instanceColor?.array.set(colors.subarray(0, count * 3))
      if (mesh.instanceColor) mesh.instanceColor.needsUpdate = true
    }
    previousCount.current = count
  }, -2)

  if (capacity === 0) return null
  return <>{parts.map((part, index) => (
    <instancedMesh
      key={`${record.id}:${index}`}
      ref={(mesh) => {
        meshes.current[index] = mesh
        if (mesh && !mesh.instanceColor) mesh.instanceColor = new InstancedBufferAttribute(new Float32Array(capacity * 3).fill(1), 3)
      }}
      args={[part.geometry, part.material, capacity]}
      castShadow
      receiveShadow
      frustumCulled={false}
    />
  ))}</>
}

type PropPlaceKind = Exclude<PlaceKind, 'room' | 'gathering' | 'corridor' | 'door' | 'filler'>

function placementsFor(places: Place[], kind: PropPlaceKind, terrain: TerrainField): Placement[] {
  return places.filter((p) => p.kind === kind).map((p) => ({ key: p.id, position: p.position, y: heightAt(terrain, p.position[0], p.position[1]), rotation: p.rotation, scale: 1 }))
}

/** Seats around tables and the campfire get the scene's seat prop; desk seats get a chair behind the desk. */
function seatPlacements(places: Place[], kind: 'table' | 'hearth' | 'desk', terrain: TerrainField): Placement[] {
  const out: Placement[] = []
  for (const place of places) {
    if (place.kind !== kind) continue
    for (const seat of place.seats) {
      out.push({ key: seat.id, position: seat.position, y: heightAt(terrain, seat.position[0], seat.position[1]), rotation: seat.facing + Math.PI, scale: 1 })
    }
  }
  return out
}

/**
 * Every place-bound prop: desks, chairs, tables, seats, the campfire, the
 * board and lamps. Layout comes from the sim's places; props are the scene's
 * data; the registry supplies the baked meshes.
 */
export function Props({ scene, period, tuning }: { scene: Scene; period: LightingPeriod; tuning: LayoutTuning }) {
  const store = useWorldStore()
  const state = store.getState()
  const places = useMemo(() => state.placeOrder.map((id) => state.places[id]).filter((p): p is Place => p !== undefined), [state.placeOrder, state.places])
  const records = useMemo(() => {
    const get = (id: string) => propRecord(scene.assetSet, id)
    return { desk: get(scene.props.desk), chair: get(scene.props.chair), table: get(scene.props.table), seat: get(scene.props.seat), hearth: get(scene.props.hearth), lamp: get(scene.props.lamp), board: get(scene.props.board) }
  }, [scene])
  const lamps = useMemo<Placement[]>(() => {
    const out: Placement[] = []
    for (const room of places) {
      if (room.kind !== 'room') continue
      const [w, d] = room.size
      const inset = Math.min(w, d) * 0.08
      const rotate = (localX: number, localZ: number): Vec2 => [room.position[0] + localX * Math.cos(room.rotation) + localZ * Math.sin(room.rotation), room.position[1] - localX * Math.sin(room.rotation) + localZ * Math.cos(room.rotation)]
      const members = places.filter((place) => place.parentId === room.id && place.kind === 'desk').length
      const choice = interiorFor(state.seed, room.teamId ?? room.id, members, room.size, tuning, scene.props.filler.length)
      const corners: Vec2[] = [[-w / 2 + inset, -d / 2 + inset], [w / 2 - inset, -d / 2 + inset], [w / 2 - inset, d / 2 - inset], [-w / 2 + inset, d / 2 - inset]]
      choice.lampCorners.forEach((cornerIndex, index) => {
        const local = corners[cornerIndex]
        if (!local) return
        const position = rotate(local[0], local[1])
        out.push({ key: `${room.id}:lamp:${index}`, position, y: heightAt(state.terrain, position[0], position[1]), rotation: room.rotation, scale: 1 })
      })
    }
    for (const corridor of places) {
      if (corridor.kind !== 'corridor') continue
      const horizontal = corridor.size[0] >= corridor.size[1]
      const length = horizontal ? corridor.size[0] : corridor.size[1]
      const count = Math.max(1, Math.floor(length / 10))
      for (let index = 0; index < count; index += 1) {
        const offset = ((index + 0.5) / count - 0.5) * length
        out.push({ key: `${corridor.id}:lamp:${index}`, position: horizontal ? [corridor.position[0] + offset, corridor.position[1]] : [corridor.position[0], corridor.position[1] + offset], y: 0, rotation: corridor.rotation, scale: 0.8 })
      }
    }
    return out
  }, [places, scene.props.filler.length, state.seed, state.terrain, tuning])
  const s = scene.propScale
  const glow = period.lampEmissive > 0 ? { color: period.keyColor, intensity: period.lampEmissive } : undefined
  return (
    <group name="props">
      {records.desk && <PropInstances record={records.desk} placements={placementsFor(places, 'desk', state.terrain)} scale={s} />}
      {records.chair && <PropInstances record={records.chair} placements={seatPlacements(places, 'desk', state.terrain)} scale={s} />}
      {records.table && <PropInstances record={records.table} placements={placementsFor(places, 'table', state.terrain)} scale={s} />}
      {records.seat && <PropInstances record={records.seat} placements={[...seatPlacements(places, 'table', state.terrain), ...seatPlacements(places, 'hearth', state.terrain)]} scale={s} />}
      {records.hearth && <PropInstances record={records.hearth} placements={placementsFor(places, 'hearth', state.terrain)} scale={s} emissive={glow ? { color: '#ff9a3c', intensity: glow.intensity } : undefined} />}
      {records.board && <PropInstances record={records.board} placements={placementsFor(places, 'board', state.terrain)} scale={s} />}
      {records.lamp && <PropInstances record={records.lamp} placements={lamps} scale={s} emissive={glow} />}
    </group>
  )
}
