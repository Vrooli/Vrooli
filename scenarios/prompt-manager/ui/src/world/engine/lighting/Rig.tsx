import { Environment, Lightformer } from '@react-three/drei'
import { useLoader, useThree } from '@react-three/fiber'
import { useEffect, useMemo } from 'react'
import { Color, EquirectangularReflectionMapping } from 'three'
import { HDRLoader } from 'three/examples/jsm/loaders/HDRLoader.js'
import type { LightingPeriod, LightingTuning, QualityProfile, Scene } from '../../config'
import { WORLD_ASSETS, worldAssetUrl } from '../assets/urls'
import type { WorldBounds } from '../types'
import { fitDistance } from '../camera/pose'

interface LightingRigProps {
  scene: Scene
  period: LightingPeriod
  lighting: LightingTuning
  profile: QualityProfile
  bounds: WorldBounds
  /** Vertical field of view, so fog distances follow the framing. */
  fovDeg: number
}

const DEG = Math.PI / 180
const ENV_RESOLUTION = 256
const SUN_DISTANCE = 100

function sunDirection(elevationDeg: number, azimuthDeg: number): [number, number, number] {
  const elevation = elevationDeg * DEG
  const azimuth = azimuthDeg * DEG
  return [Math.cos(elevation) * Math.sin(azimuth), Math.sin(elevation), Math.cos(elevation) * Math.cos(azimuth)]
}

/**
 * One directional key light with a shadow frustum fitted to the slab, a
 * hemisphere fill, an HDRI environment with Lightformer rim panels, the sky
 * dome for outdoor scenes and exponential fog. Every number comes from the
 * resolved lighting period.
 */
export function LightingRig({ scene, period, lighting, profile, bounds, fovDeg }: LightingRigProps) {
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
  const half = Math.max(bounds.width, bounds.depth) * 0.62
  // Fog is framed relative to the slab so a bigger world does not sink into it.
  const fit = fitDistance({ width: bounds.width, depth: bounds.depth, fovDeg, aspect: 1 })
  const keyPosition: [number, number, number] = [
    bounds.center[0] + keyDir[0] * SUN_DISTANCE,
    keyDir[1] * SUN_DISTANCE,
    bounds.center[1] + keyDir[2] * SUN_DISTANCE,
  ]

  useEffect(() => {
    threeScene.background = new Color(period.backgroundColor)
    gl.toneMappingExposure = period.exposure
  }, [threeScene, gl, period.backgroundColor, period.exposure])

  const outdoor = scene.environment === 'outdoor'

  return (
    <>
      <fog attach="fog" args={[period.fogColor, fit * period.fogNear, fit * period.fogFar]} />
      <hemisphereLight
        args={[period.fogColor, scene.palette.ground, period.ambientIntensity]}
        position={[0, 50, 0]}
      />
      <directionalLight
        castShadow={profile.shadows}
        position={keyPosition}
        target-position={[bounds.center[0], 0, bounds.center[1]]}
        intensity={period.keyIntensity}
        color={period.keyColor}
        shadow-mapSize={[profile.shadowMapSize, profile.shadowMapSize]}
        shadow-bias={lighting.keyLight.shadowBias}
        shadow-normalBias={lighting.keyLight.shadowNormalBias}
        shadow-camera-left={-half}
        shadow-camera-right={half}
        shadow-camera-top={half}
        shadow-camera-bottom={-half}
        shadow-camera-near={SUN_DISTANCE - half * 2}
        shadow-camera-far={SUN_DISTANCE + half * 2}
      />
      <Environment
        map={sky}
        resolution={ENV_RESOLUTION}
        environmentIntensity={period.envIntensity}
        background={outdoor}
        backgroundIntensity={period.skyIntensity}
        backgroundBlurriness={period.skyBlur}
      >
        <Lightformer form="rect" intensity={1.2} color={period.keyColor} position={[8, 6, -8]} scale={[8, 4, 1]} target={[0, 0, 0]} />
        <Lightformer form="rect" intensity={0.6} color={period.fogColor} position={[-9, 4, 4]} scale={[6, 3, 1]} target={[0, 0, 0]} />
        <Lightformer form="ring" intensity={0.4} color="#ffffff" position={[0, 10, 0]} scale={6} target={[0, 0, 0]} />
      </Environment>
    </>
  )
}
