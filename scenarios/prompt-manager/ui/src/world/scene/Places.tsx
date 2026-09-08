import { Instance, Instances } from '@react-three/drei'
import { useMemo, useState } from 'react'
import { useFrame, useThree } from '@react-three/fiber'
import { Vector3 } from 'three'
import type { LayoutTuning, Scene } from '../config'
import { heightAt, type Place } from '../sim'
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
 * Full room architecture with directional Explore cutaways and the commons disc. Terrain owns the
 * level pads and baked path mask; props and actors are separate layers.
 */
export function Places({ scene, layout, walking = false, revealedSpaceId, onSelectSpace }: { scene: Scene; layout: LayoutTuning; walking?: boolean; revealedSpaceId?: string | null; onSelectSpace?: (id: string) => void }) {
  const camera = useThree(state => state.camera)
  const direction = useMemo(() => new Vector3(), [])
  const sectorForCamera = () => {
    camera.getWorldDirection(direction)
    return Math.round(Math.atan2(-direction.x, -direction.z) / (Math.PI / 8))
  }
  const [sector, setSector] = useState(sectorForCamera)
  useFrame(() => {
    if (walking) return
    const next = sectorForCamera()
    if (Math.hypot(direction.x, direction.z) > .01 && next !== sector) setSector(next)
  })
  const surfaces = layout.surfaces
  const { floorThickness, corridorLift } = surfaces
  const store = useWorldStore()
  const state = store.getState()
  const commons = state.placeOrder.map((id) => state.places[id]).find((place) => place?.kind === 'gathering')
  const { walls, floors, corridors, windows } = useMemo(() => {
    const rooms = state.placeOrder.map((id) => state.places[id]).filter((p): p is Place => p?.kind === 'room')
    const result: { walls: Slab[]; floors: Slab[] } = { walls: [], floors: [] }
    const windows: Slab[] = []
    for (const room of rooms.filter(room => room.space)) {
      const space = room.space
      if (!space) continue
      const ground = heightAt(state.terrain, room.position[0], room.position[1])
      for (const box of spaceStructures(room)) {
        const reveal = !walking && (space.kind === 'office' || room.id === revealedSpaceId)
        // Remove the roof and the sides facing the viewer; retain complete far
        // walls and their windows so the room still reads as architecture.
        if (reveal && (box.surface === 'ceiling' || box.outward &&
          box.outward[0] * Math.sin(sector * Math.PI / 8) + box.outward[1] * Math.cos(sector * Math.PI / 8) > .15)) continue
        const scale = [...box.size] as Slab['scale']
        const position = [...box.position] as Slab['position']
        position[1] += ground
        const color = box.color ?? (box.surface === 'window' ? architecture.palette.glass : box.surface === 'floor' ? scene.palette.roomFloor : space.variant === 'tent' ? architecture.palette.canvas : space.variant === 'rv' ? architecture.palette.rv : space.kind === 'office' ? scene.palette.roomWall : architecture.palette.timber)
        const slab = { key: box.id, position, rotation: box.rotation, scale, roomId: room.id, color }
        if (box.surface === 'window') windows.push(slab)
        else if (box.surface === 'floor') result.floors.push(slab)
        else result.walls.push(slab)
      }
    }
    const corridorSlabs = state.placeOrder.map((id) => state.places[id]).filter((p): p is Place => p?.kind === 'corridor').map((place) => ({ key: `${place.id}:floor`, position: [place.position[0], heightAt(state.terrain, place.position[0], place.position[1]) + corridorLift, place.position[1]] as [number, number, number], rotation: place.rotation, scale: [place.size[0], floorThickness, place.size[1]] as [number, number, number] }))
    if (walking && scene.environment === 'indoor') {
      for (const floor of corridorSlabs) result.walls.push({ ...floor, key: floor.key.replace(':floor', ':ceiling'),
        position: [floor.position[0], floor.position[1] - corridorLift + architecture.wallHeight + architecture.office.ceilingThickness / 2, floor.position[2]],
        scale: [floor.scale[0], architecture.office.ceilingThickness, floor.scale[2]], color: scene.palette.roomWall })
    }
    return { ...result, corridors: corridorSlabs, windows }
  }, [state.placeOrder, state.places, state.terrain, corridorLift, floorThickness, scene.environment, scene.palette.roomFloor, scene.palette.roomWall, walking, revealedSpaceId, sector])
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
