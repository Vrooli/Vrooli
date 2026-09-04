import { Instance, Instances } from '@react-three/drei'
import { useMemo } from 'react'
import type { LayoutTuning, Scene } from '../config'
import { heightAt, type Place, type Vec2 } from '../sim'
import { useWorldStore } from './WorldStoreContext'

const WALL_THICKNESS = 0.18
const COMMONS_SEGMENTS = 48

interface Slab {
  key: string
  position: [number, number, number]
  rotation: number
  scale: [number, number, number]
}

/** Rotate a local (x, z) offset by a yaw about +y (Three's convention). */
function rotate([x, z]: Vec2, yaw: number): Vec2 {
  const cos = Math.cos(yaw)
  const sin = Math.sin(yaw)
  return [x * cos + z * sin, -x * sin + z * cos]
}

export function buildRoomSlabs(rooms: Place[], doors: Place[], layout: LayoutTuning, height: (point: Vec2) => number, enclosed: boolean): { walls: Slab[]; floors: Slab[] } {
  const walls: Slab[] = []
  const floors: Slab[] = []
  for (const room of rooms) {
    const [w, d] = room.size
    const [x, z] = room.position
    const yaw = room.rotation
    const wallY = height(room.position) + layout.wallHeight / 2
    floors.push({ key: `${room.id}:floor`, position: [x, height(room.position) + 0.012, z], rotation: yaw, scale: [w, 0.02, d] })
    // Three low walls; the front (toward +z locally) stays open.
    const sides: Array<{ id: string; local: Vec2; scale: [number, number, number] }> = [
      { id: 'back', local: [0, -d / 2], scale: [w + WALL_THICKNESS, layout.wallHeight, WALL_THICKNESS] },
      { id: 'left', local: [-w / 2, 0], scale: [WALL_THICKNESS, layout.wallHeight, d] },
      { id: 'right', local: [w / 2, 0], scale: [WALL_THICKNESS, layout.wallHeight, d] },
    ]
    for (const side of sides) {
      const [dx, dz] = rotate(side.local, yaw)
      walls.push({ key: `${room.id}:${side.id}`, position: [x + dx, wallY, z + dz], rotation: yaw, scale: side.scale })
    }
    if (enclosed) {
      const door = doors.find((candidate) => candidate.parentId === room.id)
      const gap = Math.min(w - WALL_THICKNESS * 2, door?.size[0] ?? layout.floorplan.doorWidth)
      const segment = (w - gap) / 2
      for (const sign of [-1, 1]) {
        const local: Vec2 = [sign * (gap / 2 + segment / 2), d / 2]
        const [dx, dz] = rotate(local, yaw)
        walls.push({ key: `${room.id}:front:${sign}`, position: [x + dx, wallY, z + dz], rotation: yaw, scale: [segment, layout.wallHeight, WALL_THICKNESS] })
      }
      const frameThickness = WALL_THICKNESS * 1.5
      for (const sign of [-1, 1]) {
        const local: Vec2 = [sign * gap / 2, d / 2]
        const [dx, dz] = rotate(local, yaw)
        walls.push({ key: `${room.id}:door-jamb:${sign}`, position: [x + dx, wallY, z + dz], rotation: yaw, scale: [frameThickness, layout.wallHeight, frameThickness] })
      }
      const [lintelX, lintelZ] = rotate([0, d / 2], yaw)
      walls.push({ key: `${room.id}:door-lintel`, position: [x + lintelX, height(room.position) + layout.wallHeight - frameThickness / 2, z + lintelZ], rotation: yaw, scale: [gap + frameThickness, frameThickness, frameThickness] })
    }
  }
  return { walls, floors }
}

function SlabInstances({ slabs, color, roughness, castShadow = false }: { slabs: Slab[]; color: string; roughness: number; castShadow?: boolean }) {
  if (slabs.length === 0) return null
  return (
    <Instances limit={slabs.length} castShadow={castShadow} receiveShadow frustumCulled={false}>
      <boxGeometry args={[1, 1, 1]} />
      <meshStandardMaterial color={color} roughness={roughness} />
      {slabs.map((slab) => (
        <Instance key={slab.key} position={slab.position} rotation={[0, slab.rotation, 0]} scale={slab.scale} />
      ))}
    </Instances>
  )
}

/**
 * Three low room walls (front open) and the commons disc. Terrain owns the
 * level pads and baked path mask; props and actors are separate layers.
 */
export function Places({ scene, layout }: { scene: Scene; layout: LayoutTuning }) {
  const store = useWorldStore()
  const state = store.getState()
  const commons = state.placeOrder.map((id) => state.places[id]).find((place) => place?.kind === 'gathering')
  const { walls, floors, corridors } = useMemo(() => {
    const rooms = state.placeOrder.map((id) => state.places[id]).filter((p): p is Place => p?.kind === 'room')
    const doors = state.placeOrder.map((id) => state.places[id]).filter((p): p is Place => p?.kind === 'door')
    const result = buildRoomSlabs(rooms, doors, layout, (point) => heightAt(state.terrain, point[0], point[1]), scene.environment === 'indoor')
    const corridorSlabs = state.placeOrder.map((id) => state.places[id]).filter((p): p is Place => p?.kind === 'corridor').map((place) => ({ key: `${place.id}:floor`, position: [place.position[0], 0.011, place.position[1]] as [number, number, number], rotation: place.rotation, scale: [place.size[0], 0.02, place.size[1]] as [number, number, number] }))
    return { ...result, corridors: corridorSlabs }
  }, [state.placeOrder, state.places, state.terrain, layout, scene.environment])
  return (
    <group name="places">
      <SlabInstances slabs={walls} color={scene.palette.roomWall} roughness={0.85} castShadow />
      <SlabInstances slabs={floors} color={scene.palette.roomFloor} roughness={0.95} />
      <SlabInstances slabs={corridors} color={scene.palette.path} roughness={0.9} />
      {commons && (
        <mesh position={[commons.position[0], heightAt(state.terrain, commons.position[0], commons.position[1]) + 0.01, commons.position[1]]} rotation={[-Math.PI / 2, 0, 0]} receiveShadow>
          <circleGeometry args={[commons.size[0] / 2, COMMONS_SEGMENTS]} />
          <meshStandardMaterial color={scene.palette.commons} roughness={1} />
        </mesh>
      )}
    </group>
  )
}
