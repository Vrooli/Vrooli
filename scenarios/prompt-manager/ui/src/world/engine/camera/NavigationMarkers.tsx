import type { Ref } from 'react'
import type { Group, Mesh } from 'three'
import { NAV_VISUALS } from '../../config/navigation'
import type { NavigationMode } from './navigation'

export function NavigationMarkers({ pivot, avatar, mode }: { pivot: Ref<Mesh>; avatar: Ref<Group>; mode: NavigationMode }) {
  return <>
    <mesh ref={pivot} rotation={[-Math.PI / 2, 0, 0]} renderOrder={NAV_VISUALS.pivotOrder} raycast={() => {}}>
      <ringGeometry args={[...NAV_VISUALS.pivotGeometry]} />
      <meshBasicMaterial color={NAV_VISUALS.accent} depthTest={false} transparent opacity={NAV_VISUALS.pivotOpacity} />
    </mesh>
    <group ref={avatar} visible={mode === 'third-person'}>
      <mesh position={[...NAV_VISUALS.bodyPosition]} castShadow raycast={() => {}}><capsuleGeometry args={[...NAV_VISUALS.bodyGeometry]} /><meshStandardMaterial color={NAV_VISUALS.accent} /></mesh>
      <mesh position={[...NAV_VISUALS.facePosition]} raycast={() => {}}><sphereGeometry args={[...NAV_VISUALS.faceGeometry]} /><meshStandardMaterial color={NAV_VISUALS.faceColor} /></mesh>
    </group>
  </>
}
