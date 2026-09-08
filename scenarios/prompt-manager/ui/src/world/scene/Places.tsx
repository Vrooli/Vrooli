import { Instance, Instances } from '@react-three/drei'
import { useMemo } from 'react'
import type { LayoutTuning, Scene } from '../config'
import { heightAt, type Place, type Vec2 } from '../sim'
import { useWorldStore } from './WorldStoreContext'
import { architecture } from '../config/architecture'
import { spaceStructures } from '../sim/layout/spaces'
import { SpaceDetails } from './SpaceDetails'

interface Slab {
  key: string
  position: [number, number, number]
  rotation: number
  scale: [number, number, number]
  roomId?: string
  color?: string
}

/** Rotate a local (x, z) offset by a yaw about +y (Three's convention). */
function rotate([x, z]: Vec2, yaw: number): Vec2 {
  const cos = Math.cos(yaw)
  const sin = Math.sin(yaw)
  return [x * cos + z * sin, -x * sin + z * cos]
}

type SlabLayout = Pick<LayoutTuning, 'wallHeight'> & {
  floorplan: Pick<LayoutTuning['floorplan'], 'doorWidth'>
  surfaces: Pick<LayoutTuning['surfaces'], 'wallThickness' | 'floorLift' | 'floorThickness' | 'doorFrameScale'>
}

export function buildRoomSlabs(rooms: Place[], doors: Place[], layout: SlabLayout, height: (point: Vec2) => number, enclosed: boolean): { walls: Slab[]; floors: Slab[] } {
  const { wallThickness, floorLift, floorThickness, doorFrameScale } = layout.surfaces
  const walls: Slab[] = []
  const floors: Slab[] = []
  for (const room of rooms) {
    const [w, d] = room.size
    const [x, z] = room.position
    const yaw = room.rotation
    const wallY = height(room.position) + layout.wallHeight / 2
    floors.push({ key: `${room.id}:floor`, position: [x, height(room.position) + floorLift, z], rotation: yaw, scale: [w, floorThickness, d] })
    // Three low walls; the front (toward +z locally) stays open.
    const sides: Array<{ id: string; local: Vec2; scale: [number, number, number] }> = [
      { id: 'back', local: [0, -d / 2], scale: [w + wallThickness, layout.wallHeight, wallThickness] },
      { id: 'left', local: [-w / 2, 0], scale: [wallThickness, layout.wallHeight, d] },
      { id: 'right', local: [w / 2, 0], scale: [wallThickness, layout.wallHeight, d] },
    ]
    for (const side of sides) {
      const [dx, dz] = rotate(side.local, yaw)
      walls.push({ key: `${room.id}:${side.id}`, position: [x + dx, wallY, z + dz], rotation: yaw, scale: side.scale })
    }
    if (enclosed) {
      const door = doors.find((candidate) => candidate.parentId === room.id)
      const gap = Math.min(w - wallThickness * 2, door?.size[0] ?? layout.floorplan.doorWidth)
      const segment = (w - gap) / 2
      for (const sign of [-1, 1]) {
        const local: Vec2 = [sign * (gap / 2 + segment / 2), d / 2]
        const [dx, dz] = rotate(local, yaw)
        walls.push({ key: `${room.id}:front:${sign}`, position: [x + dx, wallY, z + dz], rotation: yaw, scale: [segment, layout.wallHeight, wallThickness] })
      }
      const frameThickness = wallThickness * doorFrameScale
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

function SlabInstances({ slabs, color, roughness, glass = false, castShadow = false, onSelectSpace }: { slabs: Slab[]; color: string; roughness: number; glass?: boolean; castShadow?: boolean; onSelectSpace?: (id: string) => void }) {
  const instances = useMemo(() => slabs.map(slab => (
    <Instance key={slab.key} position={slab.position} rotation={[0, slab.rotation, 0]} scale={slab.scale} color={slab.color ?? color}
      onClick={slab.roomId && onSelectSpace ? event => { event.stopPropagation(); if (slab.roomId) onSelectSpace(slab.roomId) } : undefined} />
  )), [slabs, onSelectSpace, color])
  if (slabs.length === 0) return null
  return (
    <Instances key={slabs.length} limit={slabs.length} castShadow={castShadow} receiveShadow frustumCulled={false}>
      <boxGeometry args={[1, 1, 1]} userData={{ cameraObstacle: 'box' }} />
      <meshStandardMaterial color="white" roughness={roughness} transparent={glass} opacity={glass ? .3 : 1} depthWrite={!glass} />
      {instances}
    </Instances>
  )
}

/**
 * Three low room walls (front open) and the commons disc. Terrain owns the
 * level pads and baked path mask; props and actors are separate layers.
 */
export function Places({ scene, layout, walking = false, revealedSpaceId, onSelectSpace }: { scene: Scene; layout: LayoutTuning; walking?: boolean; revealedSpaceId?: string | null; onSelectSpace?: (id: string) => void }) {
  const surfaces = layout.surfaces
  const { wallThickness, floorLift, floorThickness, doorFrameScale, corridorLift } = surfaces
  const wallHeight = layout.wallHeight, doorWidth = layout.floorplan.doorWidth
  const slabLayout = useMemo<SlabLayout>(() => ({ wallHeight, floorplan: { doorWidth },
    surfaces: { wallThickness, floorLift, floorThickness, doorFrameScale } }),
  [wallHeight, doorWidth, wallThickness, floorLift, floorThickness, doorFrameScale])
  const store = useWorldStore()
  const state = store.getState()
  const commons = state.placeOrder.map((id) => state.places[id]).find((place) => place?.kind === 'gathering')
  const { walls, floors, corridors, windows } = useMemo(() => {
    const rooms = state.placeOrder.map((id) => state.places[id]).filter((p): p is Place => p?.kind === 'room')
    const doors = state.placeOrder.map((id) => state.places[id]).filter((p): p is Place => p?.kind === 'door')
    const result = buildRoomSlabs(rooms.filter(room => !room.space), doors, slabLayout, (point) => heightAt(state.terrain, point[0], point[1]), scene.environment === 'indoor')
    const windows: Slab[] = []
    for (const room of rooms.filter(room => room.space)) {
      const space = room.space
      if (!space) continue
      const ground = heightAt(state.terrain, room.position[0], room.position[1])
      for (const box of spaceStructures(room)) {
        const reveal = !walking && (space.kind === 'office' || room.id === revealedSpaceId)
        if (reveal && (box.surface === 'lintel' || box.surface === 'ceiling' || box.surface === 'window')) continue
        const cut = reveal && box.surface === 'wall'
        const scale = [...box.size] as Slab['scale']
        const position = [...box.position] as Slab['position']
        if (cut) {
          const bottom = position[1] - scale[1] / 2
          if (bottom >= architecture.cutawayHeight) continue
          scale[1] = Math.min(scale[1], architecture.cutawayHeight - bottom)
          position[1] = bottom + scale[1] / 2
        }
        position[1] += ground
        const color = box.color ?? (box.surface === 'window' ? architecture.palette.glass : box.surface === 'floor' ? scene.palette.roomFloor : space.variant === 'tent' ? architecture.palette.canvas : space.variant === 'rv' ? architecture.palette.rv : space.kind === 'office' ? scene.palette.roomWall : architecture.palette.timber)
        const slab = { key: box.id, position, rotation: box.rotation, scale, roomId: room.id, color }
        if (box.surface === 'window') windows.push(slab)
        else if (box.surface === 'floor') result.floors.push(slab)
        else result.walls.push(slab)
      }
    }
    const corridorSlabs = state.placeOrder.map((id) => state.places[id]).filter((p): p is Place => p?.kind === 'corridor').map((place) => ({ key: `${place.id}:floor`, position: [place.position[0], heightAt(state.terrain, place.position[0], place.position[1]) + corridorLift, place.position[1]] as [number, number, number], rotation: place.rotation, scale: [place.size[0], floorThickness, place.size[1]] as [number, number, number] }))
    return { ...result, corridors: corridorSlabs, windows }
  }, [state.placeOrder, state.places, state.terrain, slabLayout, corridorLift, floorThickness, scene.environment, scene.palette.roomFloor, scene.palette.roomWall, walking, revealedSpaceId])
  return (
    <group name="places" userData={{ cameraSurface: true }}>
      <SlabInstances slabs={walls} color={scene.environment === 'outdoor' ? architecture.palette.timber : scene.palette.roomWall} roughness={surfaces.wallRoughness} castShadow onSelectSpace={onSelectSpace} />
      <SlabInstances slabs={floors} color={scene.palette.roomFloor} roughness={surfaces.floorRoughness} onSelectSpace={onSelectSpace} />
      <SlabInstances slabs={corridors} color={scene.palette.path} roughness={surfaces.corridorRoughness} />
      <SlabInstances slabs={windows} color={architecture.palette.glass} roughness={.2} glass onSelectSpace={onSelectSpace} />
      <SpaceDetails walking={walking} propScale={scene.propScale} revealedSpaceId={revealedSpaceId} onSelectSpace={onSelectSpace} />
      {commons && (
        <mesh position={[commons.position[0], heightAt(state.terrain, commons.position[0], commons.position[1]) + surfaces.commonsLift, commons.position[1]]} rotation={[-Math.PI / 2, 0, 0]} receiveShadow>
          <circleGeometry args={[commons.size[0] / 2, surfaces.commonsSegments]} />
          <meshStandardMaterial color={scene.palette.commons} roughness={surfaces.commonsRoughness} />
        </mesh>
      )}
    </group>
  )
}
