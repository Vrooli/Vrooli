import { RoundedBox } from '@react-three/drei'
import type { Scene } from '../config'
import type { WorldBounds } from '../engine'

interface StageProps {
  scene: Scene
  bounds: WorldBounds
  /** Radius of the ground disc; the fog hides its edge well inside the far plane. */
  horizon: number
}

const GROUND_SEGMENTS = 48
/** How far the terrain sits below the slab top so the lot reads as a low kerb. */
const KERB = 0.25

/**
 * The ground: a terrain disc out to the horizon (fog blends it into the sky)
 * with the lot on top as a rounded slab one kerb high. Both receive the key
 * light's shadow map. (drei AccumulativeShadows was tried and dropped: its
 * material swap leaves meshes that mount during accumulation invisible.)
 */
export function Stage({ scene, bounds, horizon }: StageProps) {
  const [cx, cz] = bounds.center
  const kerb = Math.min(KERB, scene.slab.thickness)
  // A bevel larger than half the thickness would lift the top face above y=0
  // and swallow every short prop; the ground plane is the contract.
  const cornerRadius = Math.min(scene.slab.cornerRadius, kerb / 2)
  return (
    <group name="stage" position={[cx, 0, cz]}>
      <mesh rotation={[-Math.PI / 2, 0, 0]} position={[0, -kerb, 0]} receiveShadow>
        <circleGeometry args={[horizon, GROUND_SEGMENTS]} />
        <meshStandardMaterial color={scene.palette.horizon} roughness={1} metalness={0} />
      </mesh>
      <RoundedBox args={[bounds.width, kerb, bounds.depth]} radius={cornerRadius} smoothness={3} position={[0, -kerb / 2, 0]} receiveShadow>
        <meshStandardMaterial color={scene.palette.ground} roughness={0.95} metalness={0} />
      </RoundedBox>
    </group>
  )
}
