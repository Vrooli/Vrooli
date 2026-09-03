import { Cloud, Clouds } from '@react-three/drei'
import { MeshBasicMaterial } from 'three'
import type { LightingPeriod, QualityProfile, Scene } from '../config'
import type { WorldBounds } from '../engine'

const CLOUD_HEIGHT = 26
const CLOUD_SEGMENTS = 18
const CLOUD_OPACITY = 0.55

/** Sky dressing: soft clouds over outdoor scenes when the profile allows. */
export function SceneEnvironment({ scene, profile, period, bounds }: { scene: Scene; profile: QualityProfile; period: LightingPeriod; bounds: WorldBounds }) {
  if (scene.environment !== 'outdoor' || !profile.clouds) return null
  const span = Math.max(bounds.width, bounds.depth)
  return (
    <Clouds material={MeshBasicMaterial} limit={CLOUD_SEGMENTS * 3} frustumCulled={false}>
      <Cloud seed={1} segments={CLOUD_SEGMENTS} bounds={[span * 0.9, 3, span * 0.6]} volume={span * 0.35} color={period.fogColor} opacity={CLOUD_OPACITY} position={[bounds.center[0], CLOUD_HEIGHT, bounds.center[1] - span * 0.3]} speed={0.08} />
      <Cloud seed={2} segments={CLOUD_SEGMENTS} bounds={[span * 0.6, 2, span * 0.4]} volume={span * 0.25} color={period.fogColor} opacity={CLOUD_OPACITY * 0.8} position={[bounds.center[0] + span * 0.4, CLOUD_HEIGHT + 4, bounds.center[1] + span * 0.2]} speed={0.05} />
    </Clouds>
  )
}
