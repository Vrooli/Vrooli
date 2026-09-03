import { Instance, Instances } from '@react-three/drei'
import { useMemo } from 'react'
import type { LayoutTuning, Scene } from '../config'
import type { Place, Vec2 } from '../sim'
import { useWorldStore } from './WorldStoreContext'

const FLOOR_THICKNESS = 0.02
const WALL_THICKNESS = 0.18
const PATH_THICKNESS = 0.01
const PATH_WIDTH_FACTOR = 0.35
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

function roomSlabs(rooms: Place[], layout: LayoutTuning, commons: Place | undefined): { floors: Slab[]; walls: Slab[]; paths: Slab[] } {
  const floors: Slab[] = []
  const walls: Slab[] = []
  const paths: Slab[] = []
  const wallY = layout.wallHeight / 2
  for (const room of rooms) {
    const [w, d] = room.size
    const [x, z] = room.position
    const yaw = room.rotation
    floors.push({ key: room.id, position: [x, FLOOR_THICKNESS / 2, z], rotation: yaw, scale: [w, FLOOR_THICKNESS, d] })
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
    if (!commons) continue
    // A path from the open front to the commons edge, along the line between them.
    const [fx, fz] = rotate([0, d / 2], yaw)
    const front: Vec2 = [x + fx, z + fz]
    const toCommons: Vec2 = [commons.position[0] - front[0], commons.position[1] - front[1]]
    const reach = Math.hypot(toCommons[0], toCommons[1])
    const length = reach - layout.commonsRadius
    if (length <= 0) continue
    const heading = Math.atan2(toCommons[0], toCommons[1])
    const mid: Vec2 = [front[0] + (toCommons[0] / reach) * (length / 2), front[1] + (toCommons[1] / reach) * (length / 2)]
    paths.push({ key: `${room.id}:path`, position: [mid[0], PATH_THICKNESS / 2, mid[1]], rotation: heading, scale: [layout.deskPitch * PATH_WIDTH_FACTOR * 3, PATH_THICKNESS, length] })
  }
  return { floors, walls, paths }
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
 * Room floors with three low walls (front open), the commons disc, and a
 * path from each room's opening to the commons. One instanced draw per
 * kind of slab regardless of team count; props and actors are separate layers.
 */
export function Places({ scene, layout }: { scene: Scene; layout: LayoutTuning }) {
  const store = useWorldStore()
  const state = store.getState()
  const commons = state.places.commons
  const { floors, walls, paths } = useMemo(() => {
    const rooms = state.placeOrder.map((id) => state.places[id]).filter((p): p is Place => p?.kind === 'room')
    return roomSlabs(rooms, layout, commons)
  }, [state.placeOrder, state.places, layout, commons])
  return (
    <group name="places">
      <SlabInstances slabs={floors} color={scene.palette.roomFloor} roughness={0.9} />
      <SlabInstances slabs={walls} color={scene.palette.roomWall} roughness={0.85} castShadow />
      <SlabInstances slabs={paths} color={scene.palette.path} roughness={1} />
      {commons && (
        <mesh position={[commons.position[0], FLOOR_THICKNESS, commons.position[1]]} rotation={[-Math.PI / 2, 0, 0]} receiveShadow>
          <circleGeometry args={[layout.commonsRadius, COMMONS_SEGMENTS]} />
          <meshStandardMaterial color={scene.palette.commons} roughness={1} />
        </mesh>
      )}
    </group>
  )
}
