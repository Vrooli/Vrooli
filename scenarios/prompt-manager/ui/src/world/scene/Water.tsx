import { useFrame } from '@react-three/fiber'
import { useEffect, useMemo } from 'react'
import { BufferAttribute, BufferGeometry, Sphere, Vector3 } from 'three'
import type { QualityProfile, Scene, TerrainResolver, WaterVisualTuning } from '../config'
import { createWaterMaterial } from './waterAppearance'
import { useWorldStore } from './WorldStoreContext'

export function Water({ scene, profile, visual }: { scene: Scene; tuning: TerrainResolver; profile: QualityProfile; visual: WaterVisualTuning }) {
  if (scene.environment === 'indoor' && !scene.centre) return null
  return <OutdoorWater profile={profile} visual={visual} />
}

function OutdoorWater({ profile, visual }: { profile: QualityProfile; visual: WaterVisualTuning }) {
  const store = useWorldStore()
  const state = store.getState()
  const geometries = useMemo(() => {
    return state.waterGeometry.map((surface) => {
      const geometry = new BufferGeometry()
      geometry.name = `water:${surface.component}`
      geometry.setAttribute('position', new BufferAttribute(surface.positions, 3))
      geometry.setAttribute('shore', new BufferAttribute(surface.shore, 1))
      geometry.setIndex(new BufferAttribute(surface.indices, 1))
      geometry.setAttribute('normal', new BufferAttribute(surface.normals, 3))
      geometry.boundingSphere = new Sphere(new Vector3(...surface.sphere.center), surface.sphere.radius)
      return geometry
    })
  }, [state.waterGeometry])
  const material = useMemo(() => createWaterMaterial(visual, profile.wobble), [visual, profile.wobble])
  useEffect(() => () => { for (const geometry of geometries) geometry.dispose() }, [geometries])
  useEffect(() => () => material.dispose(), [material])
  useFrame((state) => {
    const time = material.uniforms.uTime
    if (time) time.value = state.clock.elapsedTime
  })
  if (!profile.waterEnabled || geometries.length === 0) return null
  return (
    <group name="water" userData={{ cameraSurface: true }}>
      {geometries.map((geometry) => <mesh key={geometry.name} name={geometry.name} geometry={geometry} material={material} renderOrder={2} />)}
    </group>
  )
}
