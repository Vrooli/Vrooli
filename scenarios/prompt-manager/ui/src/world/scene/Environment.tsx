import type { LightingPeriod, QualityProfile, Scene, WeatherPreset } from '../config'
import type { WorldBounds } from '../engine'

/** Reserved weather environment mount; phase 11 replaces the old view-obscuring clouds. */
export function SceneEnvironment({ scene, profile, period, bounds, weather, altitude }: { scene: Scene; profile: QualityProfile; period: LightingPeriod; bounds: WorldBounds; weather: WeatherPreset; altitude: number }) {
  if (scene.environment !== 'outdoor' || !profile.clouds || weather.cloudCoverage <= 0) return null
  return (
    <mesh name="cloud-layer" position={[bounds.center[0], altitude, bounds.center[1]]} rotation={[Math.PI / 2, 0, 0]}>
      <planeGeometry args={[bounds.width * 3, bounds.depth * 3]} />
      <meshBasicMaterial color={period.fogColor} transparent opacity={weather.cloudCoverage * 0.28} depthWrite={false} />
    </mesh>
  )
}
