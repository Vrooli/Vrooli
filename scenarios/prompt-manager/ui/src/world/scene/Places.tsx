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

function roomSlabs(rooms: Place[], layout: LayoutTuning, height: (point: Vec2) => number): { walls: Slab[] } {
  const walls: Slab[] = []
  for (const room of rooms) {
    const [w, d] = room.size
    const [x, z] = room.position
    const yaw = room.rotation
    const wallY = height(room.position) + layout.wallHeight / 2
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
  }
  return { walls }
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
  const commons = state.places.commons
  const { walls } = useMemo(() => {
    const rooms = state.placeOrder.map((id) => state.places[id]).filter((p): p is Place => p?.kind === 'room')
    return roomSlabs(rooms, layout, (point) => heightAt(state.terrain, point[0], point[1]))
  }, [state.placeOrder, state.places, state.terrain, layout])
  return (
    <group name="places">
      <SlabInstances slabs={walls} color={scene.palette.roomWall} roughness={0.85} castShadow />
      {commons && (
        <mesh position={[commons.position[0], heightAt(state.terrain, commons.position[0], commons.position[1]) + 0.01, commons.position[1]]} rotation={[-Math.PI / 2, 0, 0]} receiveShadow>
          <circleGeometry args={[layout.commonsRadius, COMMONS_SEGMENTS]} />
          <meshStandardMaterial color={scene.palette.commons} roughness={1} />
        </mesh>
      )}
    </group>
  )
}
