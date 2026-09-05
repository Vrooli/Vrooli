import { useFrame } from '@react-three/fiber'
import { useEffect, useMemo } from 'react'
import { BufferAttribute, BufferGeometry } from 'three'
import type { QualityProfile, Scene, TerrainResolver, WaterVisualTuning } from '../config'
import { waterSurfaceComponents } from '../sim/terrain/waterSurface'
import { createWaterMaterial } from './waterAppearance'
import { useWorldStore } from './WorldStoreContext'

export function Water({ scene, tuning, profile, visual }: { scene: Scene; tuning: TerrainResolver; profile: QualityProfile; visual: WaterVisualTuning }) {
  if (scene.environment === 'indoor' && !scene.centre) return null
  return <OutdoorWater tuning={tuning} profile={profile} visual={visual} />
}

function OutdoorWater({ tuning, profile, visual }: { tuning: TerrainResolver; profile: QualityProfile; visual: WaterVisualTuning }) {
  const store = useWorldStore()
  const state = store.getState()
  const geometries = useMemo(() => {
    return waterSurfaceComponents(state.terrain, tuning).map((surface) => {
      const geometry = new BufferGeometry()
      geometry.name = `water:${surface.component}`
      geometry.setAttribute('position', new BufferAttribute(new Float32Array(surface.positions), 3))
      geometry.setAttribute('shore', new BufferAttribute(new Float32Array(surface.shore), 1))
      geometry.setIndex(surface.indices)
      geometry.computeVertexNormals()
      geometry.computeBoundingSphere()
      return geometry
    })
  }, [state.terrain, tuning])
  const material = useMemo(() => createWaterMaterial(visual, profile.wobble), [visual, profile.wobble])
  useEffect(() => () => { for (const geometry of geometries) geometry.dispose() }, [geometries])
  useEffect(() => () => material.dispose(), [material])
  useFrame((state) => {
    const time = material.uniforms.uTime
    if (time) time.value = state.clock.elapsedTime
  })
  if (!profile.waterEnabled || geometries.length === 0) return null
  return (
    <group name="water">
      {geometries.map((geometry) => <mesh key={geometry.name} name={geometry.name} geometry={geometry} material={material} renderOrder={2} />)}
    </group>
  )
}
