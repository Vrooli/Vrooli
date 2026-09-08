import { Text, createInstances } from '@react-three/drei'
import { useMemo } from 'react'
import { roofGeometry } from './roofGeometry'
import { architecture as A } from '../config/architecture'
import { hashString } from '../sim/rng'
import { heightAt, type Place } from '../sim'
import { useWorldStore } from './WorldStoreContext'
import { propRecord, WORLD_ASSETS, worldAssetUrl } from '../engine/assets'

const [DetailBoxes, DetailBox] = createInstances()
const [SolidBoxes, SolidBox] = createInstances()
const [Roofs, Roof] = createInstances()
const [TentRoofs, TentRoof] = createInstances()
const noopRaycast = () => undefined


function Box({ position, size, color, obstacle = false }: { position: [number, number, number]; size: [number, number, number]; color: string; obstacle?: boolean }) {
  const Component = obstacle ? SolidBox : DetailBox
  return <Component position={position} scale={size} color={color} raycast={noopRaycast} />
}

function Details({ room, walking, onSelectSpace }: { room: Place; walking: boolean; onSelectSpace?: (id: string) => void }) {
  const space = room.space
  if (!space) return null
  const camp = space.kind === 'campsite'
  return <group name={`space:${room.id}`} position={[room.position[0], 0, room.position[1]]} rotation={[0, room.rotation, 0]}>
    <group position={[space.entrance[0] - (camp ? 1.9 : 2), 0, space.entrance[1] + .2]}
      onClick={event => { event.stopPropagation(); onSelectSpace?.(room.id) }}>
      <Box position={[0, .8, 0]} size={[.12, 1.6, .12]} color={A.palette.timber} />
      <mesh position={[0, A.signHeight, 0]} castShadow><boxGeometry args={[2.8, .65, .12]} /><meshStandardMaterial color={camp ? A.palette.roof : A.palette.accent} /></mesh>
      <Text position={[0, A.signHeight, .07]} font={worldAssetUrl(WORLD_ASSETS.labelFont)} fontSize={.23} maxWidth={2.5} textAlign="center" anchorX="center" anchorY="middle" color="#fff2ce">{room.label}</Text>
    </group>
    {camp && space.shelters.map(shell => <group key={shell.id} name={shell.id} position={[shell.position[0], 0, shell.position[1]]}
      onClick={event => { event.stopPropagation(); onSelectSpace?.(room.id) }}>
      {walking && (space.variant === 'rv'
        ? <Box position={[0, 2.9, 0]} size={[shell.size[0] + .25, .22, shell.size[1] + .25]} color={A.palette.rv} obstacle />
        : space.variant === 'tent'
          ? <TentRoof position={[0, .65, 0]} scale={[shell.size[0] + .5, 2.45, shell.size[1] + .6]} color={A.palette.canvas} />
          : <Roof position={[0, A.wallHeight, 0]} scale={[shell.size[0] + .5, 1.25, shell.size[1] + .6]} color={A.palette.roof} />)}
      {space.variant === 'rv' && <>
        <Box position={[0, .95, shell.size[1] / 2 + .13]} size={[shell.size[0], .28, .12]} color={A.palette.accent} />
        <Box position={[0, 2.1, -shell.size[1] / 2 - .1]} size={[shell.size[0] * .7, .8, .12]} color={A.palette.glass} />
        <Box position={[0, .4, shell.size[1] / 2 + .7]} size={[.18, .18, 1.4]} color={A.palette.metal} obstacle />
      </>}
      {[-1, 1].map(sign => <group key={sign}>
        <Box position={[sign * 1.65, walking && space.variant !== 'tent' ? 1.5 : .35, shell.size[1] / 2 + .11]} size={[.8, walking && space.variant !== 'tent' ? .8 : .15, .08]} color={A.palette.glass} />
        {space.variant === 'rv' && [-1.4, 1.4].map(z => <mesh key={z} position={[sign * (shell.size[0] / 2 + .05), .3, z]} rotation={[0, 0, Math.PI / 2]} raycast={noopRaycast}>
          <cylinderGeometry args={[.35, .35, .2, 12]} /><meshStandardMaterial color={A.palette.metal} />
        </mesh>)}
      </group>)}
      <Box position={[0, .06, shell.size[1] / 2 + .45]} size={[2.1, .12, .8]} color={A.palette.trim} obstacle />
    </group>)}
    {!camp && <>
      {/* Skirting boards and open door leaves give the architecture depth without narrowing the entrance. */}
      <Box position={[0, .13, -room.size[1] / 2 + .11]} size={[room.size[0], .25, .07]} color={A.palette.trim} />
      {[-1, 1].map(sign => <group key={sign}>
        <Box position={[sign * (room.size[0] / 2 - .11), .13, 0]} size={[.07, .25, room.size[1]]} color={A.palette.trim} />
        {walking && <>
          <Box position={[sign * room.size[0] / 2, A.office.windowSill + A.office.windowHeight / 2, 0]} size={[.24, A.office.windowHeight, .06]} color={A.palette.trim} />
          <Box position={[sign * room.size[0] / 2, A.office.windowSill, 0]} size={[.3, .07, room.size[1] * A.office.windowFraction]} color={A.palette.trim} />
        </>}
      </group>)}
      {walking && <>
        <Box position={[-A.doorWidth / 2 - .05, 1.1, room.size[1] / 2 + .65]} size={[.13, .04, .12]} color={A.palette.metal} />
        <Box position={[0, A.wallHeight - .04, 0]} size={[1.6, .08, .55]} color="#f4e8cb" />
      </>}
      {room.id === 'shared:kitchen' && <>
        <Box position={[0, .9, -room.size[1] / 2 + .55]} size={[5, .08, 1.1]} color={A.palette.trim} />
        <Box position={[0, .9, .8]} size={[3, .08, 1]} color={A.palette.trim} />
        <Box position={[-.65, .95, -room.size[1] / 2 + .55]} size={[.8, .03, .55]} color={A.palette.metal} />
        <Box position={[-.65, 1.1, -room.size[1] / 2 + .2]} size={[.05, .35, .05]} color={A.palette.glass} />
        <Box position={[-.65, 1.26, -room.size[1] / 2 + .35]} size={[.05, .04, .3]} color={A.palette.glass} />
        <Box position={[1.4, 1.18, -room.size[1] / 2 + .55]} size={[.6, .46, .55]} color={A.palette.metal} />
        <Box position={[1.4, 1.18, -room.size[1] / 2 + .84]} size={[.45, .3, .02]} color={A.palette.glass} />
        {[-1.7, 0, 1.7].map(x => <Box key={x} position={[x, .55, -room.size[1] / 2 + 1.065]} size={[.3, .04, .03]} color={A.palette.trim} />)}
        <Box position={[room.size[0] / 2 - .3, 1.12, -room.size[1] / 2 + 1.24]} size={[.04, .42, .04]} color={A.palette.metal} />
      </>}
    </>}
  </group>
}

export function SpaceDetails({ walking, propScale, revealedSpaceId, onSelectSpace }: { walking: boolean; propScale: number; revealedSpaceId?: string | null; onSelectSpace?: (id: string) => void }) {
  const state = useWorldStore().getState()
  // Geometry lifetime follows this presenter. R3F owns disposal on unmount.
  const roof = useMemo(() => roofGeometry(), [])
  const tentRoof = useMemo(() => roofGeometry(true), [])
  const rooms = state.placeOrder.map(id => state.places[id]).filter((p): p is Place => !!p?.space)
  const landmark = state.places.landmark
  const stations = state.placeOrder.map(id => state.places[id]).filter((p): p is Place => p?.kind === 'desk')
  const deskHeight = (propRecord('office', 'desk')?.size[1] ?? .38) * propScale
  const capacity = Math.max(1, stations.length * 12 + rooms.reduce((sum, room) => sum + 48 + (room.space?.shelters.length ?? 0) * 24, 16))
  return <group name="space-details">
    <DetailBoxes key={`detail:${capacity}`} limit={capacity} castShadow receiveShadow frustumCulled={false}>
      <boxGeometry /><meshStandardMaterial roughness={.85} />
    <SolidBoxes key={`solid:${capacity}`} limit={capacity} castShadow receiveShadow frustumCulled={false}>
      <boxGeometry userData={{ cameraObstacle: 'box' }} /><meshStandardMaterial roughness={.85} />
    <Roofs key={`roof:${capacity}`} limit={capacity} geometry={roof} castShadow frustumCulled={false}>
      <meshStandardMaterial roughness={.9} side={2} />
    <TentRoofs key={`tent:${capacity}`} limit={capacity} geometry={tentRoof} castShadow frustumCulled={false}>
      <meshStandardMaterial roughness={.9} side={2} />
    {rooms.map(room => <group key={room.id} position={[0, heightAt(state.terrain, ...room.position), 0]}>
      <Details room={room} walking={walking || room.space?.kind === 'campsite' && room.id !== revealedSpaceId} onSelectSpace={onSelectSpace} />
    </group>)}
    {state.scene === 'office' && stations.map(station => {
      const variant = hashString(station.id) % 3
      return <group key={station.id} name={`workstation:${station.id}`} position={[station.position[0], heightAt(state.terrain, ...station.position), station.position[1]]} rotation={[0, station.rotation, 0]}>
        <Box position={[0, deskHeight + .01, .02]} size={[1, .015, .5]} color={variant === 0 ? A.palette.roof : variant === 1 ? A.palette.accent : A.palette.timber} />
        {(variant === 1 ? [-.29, .29] : [0]).map(x => <group key={x}>
          <Box position={[x, deskHeight + .1, -.12]} size={[.16, .2, .12]} color={A.palette.metal} />
          <Box position={[x, deskHeight + .34, -.12]} size={[variant === 1 ? .52 : A.office.monitorWidth, A.office.monitorHeight, .055]} color={A.palette.metal} />
          <Box position={[x, deskHeight + .34, -.085]} size={[variant === 1 ? .46 : .59, .34, .012]} color={variant === 2 ? '#9ab6ba' : '#526f77'} />
        </group>)}
        <Box position={[0, deskHeight + .035, .22]} size={[.43, .04, .16]} color="#ddd6c8" />
        <Box position={[.38, deskHeight + .09, .19]} size={[.09, .16, .09]} color={A.palette.canvas} />
      </group>
    })}
    {state.scene === 'park' && state.placeOrder.map(id => state.places[id]).filter((p): p is Place => p?.kind === 'desk').map(station => <group key={station.id} position={[station.position[0], heightAt(state.terrain, ...station.position), station.position[1]]} rotation={[0, station.rotation, 0]}>
      <Box position={[0, .08, 0]} size={[station.size[0], .16, station.size[1]]} color={A.palette.roof} obstacle />
      <Box position={[0, .2, -.3]} size={[station.size[0] * .8, .18, .25]} color={A.palette.trim} />
    </group>)}
    {state.scene === 'park' && landmark && <group name="campground-landmark" position={[landmark.position[0], heightAt(state.terrain, ...landmark.position), landmark.position[1]]} raycast={noopRaycast}>
      <Box position={[0, .18, 0]} size={[1.6, .36, 1.6]} color={A.palette.trim} obstacle />
      <Box position={[0, 1.5, 0]} size={[.55, 3, .55]} color={A.palette.timber} obstacle />
      {[0, 1, 2].map(i => <mesh key={i} position={[0, 1.2 + i * .9, 0]} rotation={[0, i * Math.PI / 4, 0]} castShadow raycast={noopRaycast}>
        <octahedronGeometry args={[.7, 0]} /><meshStandardMaterial color={[A.palette.roof, A.palette.canvas, A.palette.accent][i]} roughness={.85} />
      </mesh>)}
    </group>}
    </TentRoofs></Roofs></SolidBoxes></DetailBoxes>
  </group>
}
