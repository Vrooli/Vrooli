import { Environment, Lightformer } from '@react-three/drei'
import { useLoader, useThree } from '@react-three/fiber'
import { useEffect, useMemo } from 'react'
import { EquirectangularReflectionMapping, MathUtils } from 'three'
import { HDRLoader } from 'three/examples/jsm/loaders/HDRLoader.js'
import type { LightingPeriod, LightingTuning, QualityProfile, Scene } from '../../config'
import { WORLD_ASSETS, worldAssetUrl } from '../assets/urls'
import type { WorldBounds } from '../types'
import { useShadowRefresh, type ShadowWorldStore } from './shadowRefresh'
import { applyPeriodBackground } from './background'

interface LightingRigProps {
  scene: Scene
  period: LightingPeriod
  lighting: LightingTuning
  profile: QualityProfile
  bounds: WorldBounds
  /** Vertical field of view, so fog distances follow the framing. */
  fovDeg: number
  store: ShadowWorldStore
}

function sunDirection(elevationDeg: number, azimuthDeg: number): [number, number, number] {
  const elevation = MathUtils.degToRad(elevationDeg)
  const azimuth = MathUtils.degToRad(azimuthDeg)
  return [Math.cos(elevation) * Math.sin(azimuth), Math.sin(elevation), Math.cos(elevation) * Math.cos(azimuth)]
}

/**
 * One directional key light with a shadow frustum fitted to the slab, a
 * hemisphere fill, an HDRI environment with Lightformer rim panels, the sky
 * dome for outdoor scenes and exponential fog. Every number comes from the
 * resolved lighting period.
 */
export function LightingRig({ scene, period, lighting, profile, bounds, fovDeg, store }: LightingRigProps) {
  const rig = lighting.rig
  const threeScene = useThree((s) => s.scene)
  const gl = useThree((s) => s.gl)
  // Load the HDRI before the environment portal mounts: the portal captures
  // its cube map once, so a texture that arrives later would be missed.
  const sky = useLoader(HDRLoader, worldAssetUrl(WORLD_ASSETS.skyHdr))
  sky.mapping = EquirectangularReflectionMapping

  // The key light follows the period's sun elevation but never drops below
  // the rig's minimum so shadows keep reading at dusk and night.
  const keyDir = useMemo(
    () => sunDirection(Math.max(period.sunElevationDeg, lighting.keyLight.elevationDeg), lighting.keyLight.azimuthDeg),
    [period.sunElevationDeg, lighting.keyLight.elevationDeg, lighting.keyLight.azimuthDeg],
  )
  const half = Math.max(bounds.footprint.width, bounds.footprint.depth) * rig.shadowExtentScale + rig.shadowExtentPadding
  const shadowCenter = bounds.footprint.center
  // Fog is framed relative to the slab so a bigger world does not sink into it.
  const fogExtent = Math.hypot(bounds.width, bounds.depth) / (2 * Math.sin(MathUtils.degToRad(fovDeg) / 2))
  const keyPosition: [number, number, number] = [
    shadowCenter[0] + keyDir[0] * rig.sunDistance,
    keyDir[1] * rig.sunDistance,
    shadowCenter[1] + keyDir[2] * rig.sunDistance,
  ]

  useShadowRefresh(store, scene, profile, `${period.backgroundColor}:${period.keyColor}:${period.keyIntensity}:${period.exposure}`)

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
        castShadow={profile.shadows}
        position={keyPosition}
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
        background={outdoor}
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
