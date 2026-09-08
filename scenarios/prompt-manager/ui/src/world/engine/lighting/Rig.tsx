import { celestialDirections, celestialKey, lunarPhase } from '../../config/celestial'
import type { WorldClock } from '../../config/clock'
import type { PeriodId } from '../../config'
import { Environment, Lightformer } from '@react-three/drei'
import { useFrame, useLoader, useThree } from '@react-three/fiber'
import { useEffect, useMemo, useRef } from 'react'
import { Color, EquirectangularReflectionMapping, MathUtils, type DirectionalLight } from 'three'
import { HDRLoader } from 'three/examples/jsm/loaders/HDRLoader.js'
import type { LightingPeriod, LightingTuning, QualityProfile, Scene } from '../../config'
import { WORLD_ASSETS, worldAssetUrl } from '../assets/urls'
import type { WorldBounds } from '../types'
import { resizeShadowTarget, useShadowRefresh, type ShadowWorldStore } from './shadowRefresh'
import { applyPeriodBackground } from './background'

interface LightingRigProps {
  scene: Scene
  clock: WorldClock
  mode: 'clock' | PeriodId
  sunElevationDegrees: number
  cloudCoverage: number
  period: LightingPeriod
  lighting: LightingTuning
  profile: QualityProfile
  bounds: WorldBounds
  /** Vertical field of view, so fog distances follow the framing. */
  fovDeg: number
  store: ShadowWorldStore
}

/**
 * One directional key light with a shadow frustum fitted to the slab, a
 * hemisphere fill, an HDRI environment with Lightformer rim panels, the sky
 * dome for outdoor scenes and exponential fog. Every number comes from the
 * resolved lighting period.
 */
export function LightingRig({ scene, period, lighting, profile, bounds, fovDeg, store, clock, mode, sunElevationDegrees, cloudCoverage }: LightingRigProps) {
  const rig = lighting.rig
  const threeScene = useThree((s) => s.scene)
  const gl = useThree((s) => s.gl)
  const invalidate = useThree((s) => s.invalidate)
  const keyLight = useRef<DirectionalLight | null>(null)
  useEffect(() => {
    if (keyLight.current && resizeShadowTarget(keyLight.current.shadow, profile.shadowMapSize)) {
      gl.shadowMap.needsUpdate = true
      invalidate()
    }
  }, [gl, invalidate, profile.shadowMapSize])
  // Load the HDRI before the environment portal mounts: the portal captures
  // its cube map once, so a texture that arrives later would be missed.
  const sky = useLoader(HDRLoader, worldAssetUrl(WORLD_ASSETS.skyHdr))
  sky.mapping = EquirectangularReflectionMapping

  const half = Math.max(bounds.footprint.width, bounds.footprint.depth) * rig.shadowExtentScale + rig.shadowExtentPadding
  const shadowCenter = bounds.footprint.center
  // Fog is framed relative to the slab so a bigger world does not sink into it.
  const fogExtent = Math.hypot(bounds.width, bounds.depth) / (2 * Math.sin(MathUtils.degToRad(fovDeg) / 2))
  const celestial = useMemo(() => ({ phaseKey: NaN, phase: lunarPhase(0), shadowKey: NaN, sunColor: new Color(), moonColor: new Color('#b6c9ef') }), [])
  useEffect(() => clock.subscribe(invalidate), [clock, invalidate])
  useFrame(() => {
    const light = keyLight.current
    if (!light) return
    const snapshot = clock.snapshot()
    const phaseKey = snapshot.timeScale === 0 ? snapshot.utcMilliseconds : Math.floor(snapshot.utcMilliseconds / 60000)
    if (phaseKey !== celestial.phaseKey) { celestial.phaseKey = phaseKey; celestial.phase = lunarPhase(snapshot.utcMilliseconds) }
    const directions = celestialDirections(snapshot.localMinutes, celestial.phase, mode === 'clock' ? undefined : { elevationDegrees: sunElevationDegrees, setting: mode === 'dusk' })
    const key = celestialKey(directions, celestial.phase.illumination, cloudCoverage, period.keyIntensity)
    light.position.set(shadowCenter[0] + key.direction[0] * rig.sunDistance, key.direction[1] * rig.sunDistance, shadowCenter[1] + key.direction[2] * rig.sunDistance)
    light.intensity = key.intensity
    light.color.copy(celestial.sunColor.set(period.keyColor)).lerp(celestial.moonColor, key.intensity > 0 ? key.moonIntensity / key.intensity : 1)
    // Smooth body motion does not require rebuilding shadow maps every frame.
    const shadowKey = snapshot.timeScale === 0 ? snapshot.utcMilliseconds : Math.floor(snapshot.utcMilliseconds / 30000)
    if (shadowKey !== celestial.shadowKey) { celestial.shadowKey = shadowKey; gl.shadowMap.needsUpdate = true }
  })

  useShadowRefresh(store, scene, profile, `${mode}:${sunElevationDegrees}:${cloudCoverage}:${period.backgroundColor}:${period.keyColor}:${period.keyIntensity}:${period.exposure}`)

  const outdoor = scene.environment === 'outdoor'

  useEffect(() => {
    applyPeriodBackground(threeScene, gl, outdoor, period)
  }, [threeScene, gl, outdoor, period])

  return (
    <>
      <fog attach="fog" args={[period.fogColor, fogExtent * period.fogNear, fogExtent * period.fogFar]} />
      <hemisphereLight
        args={[period.fogColor, scene.palette.ground, period.ambientIntensity]}
        position={[0, rig.hemisphereHeight, 0]}
      />
      <directionalLight
        ref={keyLight}
        name="celestial-key"
        castShadow={profile.shadows}
        target-position={[shadowCenter[0], 0, shadowCenter[1]]}
        intensity={period.keyIntensity}
        color={period.keyColor}
        shadow-mapSize={[profile.shadowMapSize, profile.shadowMapSize]}
        shadow-bias={lighting.keyLight.shadowBias}
        shadow-normalBias={lighting.keyLight.shadowNormalBias}
        shadow-camera-left={-half}
        shadow-camera-right={half}
        shadow-camera-top={half}
        shadow-camera-bottom={-half}
        shadow-camera-near={rig.sunDistance - half * 2}
        shadow-camera-far={rig.sunDistance + half * 2}
      />
      <Environment
        map={sky}
        resolution={rig.environmentResolution}
        environmentIntensity={period.envIntensity}
        background={false}
        backgroundIntensity={period.skyIntensity}
        backgroundBlurriness={period.skyBlur}
      >
        <Lightformer form="rect" {...rig.keyPanel} color={period.keyColor} target={[0, 0, 0]} />
        <Lightformer form="rect" {...rig.fillPanel} color={period.fogColor} target={[0, 0, 0]} />
        <Lightformer form="ring" {...rig.topPanel} color={rig.topPanelColor} target={[0, 0, 0]} />
      </Environment>
    </>
  )
}
