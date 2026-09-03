import { Instance, Instances } from '@react-three/drei'
import { useEffect, useMemo } from 'react'
import { Color, MeshStandardMaterial, type Material } from 'three'
import type { LightingPeriod, Scene } from '../config'
import { propRecord, usePropParts, type PropRecord } from '../engine/assets'
import type { Place, PlaceKind, Vec2 } from '../sim'
import { useWorldStore } from './WorldStoreContext'

interface Placement {
  key: string
  position: Vec2
  rotation: number
  scale: number
}

interface PropInstancesProps {
  record: PropRecord
  placements: Placement[]
  scale: number
  castShadow?: boolean
  /** Emissive glow for lamps and fire: colour and intensity from the lighting period. */
  emissive?: { color: string; intensity: number }
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
export function PropInstances({ record, placements, scale, castShadow = true, emissive }: PropInstancesProps) {
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
          frustumCulled={false}
        >
          {placements.map((p) => (
            <Instance key={p.key} position={[p.position[0], lift, p.position[1]]} rotation={[0, p.rotation, 0]} scale={scale * p.scale} />
          ))}
        </Instances>
      ))}
    </>
  )
}

type PropPlaceKind = Exclude<PlaceKind, 'room' | 'commons'>

function placementsFor(places: Place[], kind: PropPlaceKind): Placement[] {
  return places.filter((p) => p.kind === kind).map((p) => ({ key: p.id, position: p.position, rotation: p.rotation, scale: 1 }))
}

/** Seats around tables and the campfire get the scene's seat prop; desk seats get a chair behind the desk. */
function seatPlacements(places: Place[], kind: 'table' | 'campfire' | 'desk'): Placement[] {
  const out: Placement[] = []
  for (const place of places) {
    if (place.kind !== kind) continue
    for (const seat of place.seats) {
      out.push({ key: seat.id, position: seat.position, rotation: seat.facing + Math.PI, scale: 1 })
    }
  }
  return out
}

/**
 * Every place-bound prop: desks, chairs, tables, seats, the campfire, the
 * board and lamps. Layout comes from the sim's places; props are the scene's
 * data; the registry supplies the baked meshes.
 */
export function Props({ scene, period }: { scene: Scene; period: LightingPeriod }) {
  const store = useWorldStore()
  const state = store.getState()
  const places = useMemo(() => state.placeOrder.map((id) => state.places[id]).filter((p): p is Place => p !== undefined), [state.placeOrder, state.places])
  const records = useMemo(() => {
    const get = (id: string) => propRecord(scene.assetSet, id)
    return { desk: get(scene.props.desk), chair: get(scene.props.chair), table: get(scene.props.table), seat: get(scene.props.seat), campfire: get(scene.props.campfire), lamp: get(scene.props.lamp), board: get(scene.props.board) }
  }, [scene])
  const lamps = useMemo<Placement[]>(() => {
    // One lamp at each front corner of every room, facing the commons.
    const out: Placement[] = []
    for (const room of places) {
      if (room.kind !== 'room') continue
      const [w, d] = room.size
      const inset = Math.min(w, d) * 0.08
      out.push({ key: `${room.id}:lamp-l`, position: [room.position[0] - w / 2 + inset, room.position[1] + d / 2 - inset], rotation: 0, scale: 1 })
      out.push({ key: `${room.id}:lamp-r`, position: [room.position[0] + w / 2 - inset, room.position[1] + d / 2 - inset], rotation: 0, scale: 1 })
    }
    return out
  }, [places])
  const s = scene.propScale
  const glow = period.lampEmissive > 0 ? { color: period.keyColor, intensity: period.lampEmissive } : undefined
  return (
    <group name="props">
      {records.desk && <PropInstances record={records.desk} placements={placementsFor(places, 'desk')} scale={s} />}
      {records.chair && <PropInstances record={records.chair} placements={seatPlacements(places, 'desk')} scale={s} />}
      {records.table && <PropInstances record={records.table} placements={placementsFor(places, 'table')} scale={s} />}
      {records.seat && <PropInstances record={records.seat} placements={[...seatPlacements(places, 'table'), ...seatPlacements(places, 'campfire')]} scale={s} />}
      {records.campfire && <PropInstances record={records.campfire} placements={placementsFor(places, 'campfire')} scale={s} emissive={glow ? { color: '#ff9a3c', intensity: glow.intensity } : undefined} />}
      {records.board && <PropInstances record={records.board} placements={placementsFor(places, 'board')} scale={s} />}
      {records.lamp && <PropInstances record={records.lamp} placements={lamps} scale={s} emissive={glow} />}
    </group>
  )
}
