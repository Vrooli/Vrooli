import { Bvh } from '@react-three/drei'
import { useCallback, useLayoutEffect, useMemo, useRef, useState } from 'react'
import { BufferAttribute, BufferGeometry, Color, Sphere, Vector3 } from 'three'
import { type QualityProfile, type Scene, type TerrainResolver, type TerrainVisualTuning, type WeatherPreset } from '../config'
import type { TerrainMeshInput, TerrainMeshData } from '../sim/terrain/mesh'
import { beginTerrainPreparation, updateDiagnostics } from '../engine/diagnostics/store'
import { useThree } from '@react-three/fiber'
import { useWorldStore } from './WorldStoreContext'
import { useOwnedResource } from '../engine/assets/useOwnedResource'
import { terrainMaterialSettings } from './terrainAppearance'

export function Terrain({ scene, tuning, profile, weather, visual, prepareMesh }: { prepareMesh: (input: TerrainMeshInput, options: { signal: AbortSignal; onMode: (mode: 'worker' | 'fallback' | 'cache') => void }) => Promise<TerrainMeshData>; scene: Scene; tuning: TerrainResolver; profile: QualityProfile; weather: WeatherPreset; visual: TerrainVisualTuning }) {
  const store = useWorldStore()
  const state = store.getState()
  const invalidate = useThree(value => value.invalidate)
  const input = useMemo<TerrainMeshInput | null>(() => {
    if (scene.environment === 'indoor' && !scene.centre) return null
    const tint = new Color(weather.terrainTint)
    const shadowTint = new Color(weather.terrainShadowTint)
    return {
      field: state.terrain, surface: state.terrainSurface, innerRadiusSetting: tuning.base().innerRadius,
      profile: { terrainInnerRadius: profile.terrainInnerRadius, terrainCellScale: profile.terrainCellScale },
      visual, weather: { terrainTintMix: weather.terrainTintMix, terrainTintVariation: weather.terrainTintVariation }, tint: [tint.r, tint.g, tint.b], shadowTint: [shadowTint.r, shadowTint.g, shadowTint.b],
    }
  }, [profile.terrainCellScale, profile.terrainInnerRadius, scene, state.terrainSurface, state.terrain, tuning, visual, weather.terrainShadowTint, weather.terrainTint, weather.terrainTintMix, weather.terrainTintVariation])
  const [prepared, setPrepared] = useState<{ input: TerrainMeshInput; mesh: TerrainMeshData } | null>(null)
  const [failure, setFailure] = useState<{ input: TerrainMeshInput; error: Error } | null>(null)
  const release = useRef<(() => void) | null>(null)
  useLayoutEffect(() => {
    if (!input) return
    const controller = new AbortController()
    const finish = beginTerrainPreparation()
    release.current = finish
    void prepareMesh(input, {
      signal: controller.signal,
      onMode: terrainMeshMode => updateDiagnostics({ terrainMeshMode }),
    }).then(mesh => {
      if (!controller.signal.aborted) setPrepared({ input, mesh })
    }).catch((error: unknown) => {
      if (!controller.signal.aborted) setFailure({ input, error: error instanceof Error ? error : new Error(String(error)) })
    })
    return () => { controller.abort(); finish() }
  }, [input, prepareMesh])
  const createGeometry = useCallback(() => {
    const field = state.terrain
    if (scene.environment === 'indoor' && !scene.centre) {
      const half = field.radius
      const result = new BufferGeometry()
      result.setAttribute('position', new BufferAttribute(new Float32Array([-half, 0, -half, -half, 0, half, half, 0, -half, half, 0, half]), 3))
      result.setAttribute('normal', new BufferAttribute(new Float32Array([0, 1, 0, 0, 1, 0, 0, 1, 0, 0, 1, 0]), 3))
      const ground = new Color(scene.palette.ground)
      result.setAttribute('color', new BufferAttribute(new Float32Array([ground.r, ground.g, ground.b, ground.r, ground.g, ground.b, ground.r, ground.g, ground.b, ground.r, ground.g, ground.b]), 3))
      result.setIndex([0, 1, 2, 2, 1, 3])
      result.computeBoundingSphere()
      return result
    }
    if (!prepared || prepared.input.field !== field) return null
    const { vertices, normals, colours, indices, sphere } = prepared.mesh
    const result = new BufferGeometry()
    result.setAttribute('position', new BufferAttribute(vertices, 3))
    result.setAttribute('normal', new BufferAttribute(normals, 3))
    result.setAttribute('color', new BufferAttribute(colours, 3))
    result.setIndex(new BufferAttribute(indices, 1))
    result.boundingSphere = new Sphere(new Vector3(...sphere.center), sphere.radius)
    return result
  }, [prepared, scene, state.terrain])
  const geometry = useOwnedResource(createGeometry)
  useLayoutEffect(() => {
    if (!geometry || prepared?.input !== input) return
    release.current?.()
    invalidate()
  }, [geometry, prepared, input, invalidate])

  if (failure?.input === input) throw failure.error
  if (!geometry) return null
  return (
    <Bvh key={geometry.uuid} name="terrain" indirect>
      <mesh userData={{ cameraSurface: true }} geometry={geometry} receiveShadow>
        <meshStandardMaterial vertexColors {...terrainMaterialSettings(weather.wetness, visual)} metalness={0} />
      </mesh>
    </Bvh>
  )
}
