import { useFrame, useThree } from '@react-three/fiber'
import { useEffect, useMemo } from 'react'
import type { CameraTuning, LightingPeriod, LightingTuning, QualityProfile, Scene } from '../config'
import { LampLightPool, lampPoolSize, type LampPlacement } from './lampPool'
import { recordLampSelection, updateDiagnostics } from '../engine/diagnostics/store'

interface Props {
  placements: readonly LampPlacement[]
  scene: Scene
  period: LightingPeriod
  lighting: LightingTuning
  profile: QualityProfile
  camera: CameraTuning
}

export function LampLights(props: Props) {
  const color = props.scene.emissive?.lamp
  const count = lampPoolSize(props.profile, props.period, color)
  return count > 0 && color ? <ActiveLampLights {...props} count={count} color={color} /> : null
}

function ActiveLampLights({ placements, period, lighting, camera: cameraTuning, count, color }: Props & { count: number; color: string }) {
  const camera = useThree(state => state.camera)
  const pool = useMemo(() => new LampLightPool(count), [count])
  useEffect(() => {
    updateDiagnostics({ lampLightsMounted: count })
    return () => updateDiagnostics({ lampLightsMounted: 0 })
  }, [count])
  const settings = useMemo(() => ({
    color, intensity: lighting.lampLightIntensity * period.lampEmissive,
    distance: lighting.lampLightDistance, height: lighting.lampLightHeight,
  }), [color, lighting.lampLightIntensity, lighting.lampLightDistance, lighting.lampLightHeight, period.lampEmissive])
  useFrame(() => {
    if (pool.update(camera, placements, settings, cameraTuning.cullEpsilonMetres, cameraTuning.cullEpsilonRadians)) recordLampSelection()
  }, -2)
  return <group name="lamp-lights">{pool.lights.map(light => <primitive key={light.uuid} object={light} />)}</group>
}
