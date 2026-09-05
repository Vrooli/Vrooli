import { Instance, Instances } from '@react-three/drei'
import { useMemo } from 'react'
import { InstancedBufferAttribute } from 'three'
import type { CameraTuning, LayoutTuning, LightingPeriod, LightingTuning, QualityProfile, Scene } from '../config'
import { propRecord, usePropParts, type PropRecord } from '../engine/assets'
import { heightAt, type Place, type PlaceKind, type TerrainField, type Vec2 } from '../sim'
import { lampPlacements } from './lampPlacements'
import { useWorldStore } from './WorldStoreContext'
import { slotEmissive, usePropMaterials } from './propMaterials'
import { LampLights } from './LampLights'
import { type VegetationBuffer } from './vegetationCull'

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

/** One instanced draw per material part of one prop id; the prop's own origin sits on the ground plane. */
export function PropInstances({ record, placements, scale, castShadow = true, emissive, frustumCulled = false }: PropInstancesProps) {
  const parts = usePropParts(record)
  const lift = -record.bounds.min[1] * scale
  const materials = usePropMaterials(parts, emissive)
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
export function CulledPropInstances({ record, buffer }: { record: PropRecord; buffer: VegetationBuffer }) {
  const parts = usePropParts(record)
  if (buffer.capacity === 0) return null
  return <>{parts.map((part, index) => (
    <instancedMesh
      key={`${record.id}:${index}`}
      ref={(mesh) => {
        buffer.meshes[index] = mesh
        if (mesh) {
          if (!mesh.instanceColor) mesh.instanceColor = new InstancedBufferAttribute(new Float32Array(buffer.capacity * 3), 3)
          buffer.upload(mesh)
        }
      }}
      args={[part.geometry, part.material, buffer.capacity]}
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
export function Props({ scene, period, tuning, lighting, profile, camera }: { scene: Scene; period: LightingPeriod; tuning: LayoutTuning; lighting: LightingTuning; profile: QualityProfile; camera: CameraTuning }) {
  const store = useWorldStore()
  const state = store.getState()
  const places = useMemo(() => state.placeOrder.map((id) => state.places[id]).filter((p): p is Place => p !== undefined), [state.placeOrder, state.places])
  const records = useMemo(() => {
    const get = (id: string) => propRecord(scene.assetSet, id)
    return { desk: get(scene.props.desk), chair: get(scene.props.chair), table: get(scene.props.table), seat: get(scene.props.seat), hearth: get(scene.props.hearth), lamp: get(scene.props.lamp), board: get(scene.props.board) }
  }, [scene])
  const lamps = useMemo(() => lampPlacements(places, state.seed, state.terrain, tuning, scene.props.filler.length), [places, scene.props.filler.length, state.seed, state.terrain, tuning])
  const s = scene.propScale
  const glows = useMemo(() => ({
    desk: slotEmissive(scene, period, 'desk'), chair: slotEmissive(scene, period, 'chair'),
    table: slotEmissive(scene, period, 'table'), seat: slotEmissive(scene, period, 'seat'),
    hearth: slotEmissive(scene, period, 'hearth'), lamp: slotEmissive(scene, period, 'lamp'),
    board: slotEmissive(scene, period, 'board'),
  // Scene roles are immutable data; material caches further key on scalar color/intensity.
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }), [scene.emissive, period.lampEmissive])
  return (
    <group name="props">
      <LampLights placements={lamps} scene={scene} period={period} lighting={lighting} profile={profile} camera={camera} />
      {records.desk && <PropInstances record={records.desk} placements={placementsFor(places, 'desk', state.terrain)} scale={s} emissive={glows.desk} />}
      {records.chair && <PropInstances record={records.chair} placements={seatPlacements(places, 'desk', state.terrain)} scale={s} emissive={glows.chair} />}
      {records.table && <PropInstances record={records.table} placements={placementsFor(places, 'table', state.terrain)} scale={s} emissive={glows.table} />}
      {records.seat && <PropInstances record={records.seat} placements={[...seatPlacements(places, 'table', state.terrain), ...seatPlacements(places, 'hearth', state.terrain)]} scale={s} emissive={glows.seat} />}
      {records.hearth && <PropInstances record={records.hearth} placements={placementsFor(places, 'hearth', state.terrain)} scale={s} emissive={glows.hearth} />}
      {records.board && <PropInstances record={records.board} placements={placementsFor(places, 'board', state.terrain)} scale={s} emissive={glows.board} />}
      {records.lamp && <PropInstances record={records.lamp} placements={lamps} scale={s} emissive={glows.lamp} />}
    </group>
  )
}
