import { useFrame } from '@react-three/fiber'
import { useEffect, useMemo } from 'react'
import { BufferAttribute, BufferGeometry, Color, ShaderMaterial } from 'three'
import type { QualityProfile, Scene, TerrainTuning } from '../config'
import { waterSurfaceComponents } from '../sim/terrain/waterSurface'
import { useWorldStore } from './WorldStoreContext'

export function Water({ scene, tuning, profile }: { scene: Scene; tuning: TerrainTuning; profile: QualityProfile }) {
  if (scene.environment === 'indoor') return null
  return <OutdoorWater tuning={tuning} profile={profile} />
}

function OutdoorWater({ tuning, profile }: { tuning: TerrainTuning; profile: QualityProfile }) {
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
  const material = useMemo(() => new ShaderMaterial({
    transparent: true,
    depthWrite: false,
    uniforms: { uTime: { value: 0 }, uWobble: { value: profile.wobble ? 1 : 0 }, uColor: { value: new Color('#4f9db8') } },
    vertexShader: `
      attribute float shore;
      uniform float uTime;
      uniform float uWobble;
      varying float vShore;
      void main() {
        vec3 p = position;
        p.y += sin(p.x * 0.18 + uTime) * cos(p.z * 0.16 + uTime * 0.7) * 0.025 * uWobble;
        vShore = shore;
        gl_Position = projectionMatrix * modelViewMatrix * vec4(p, 1.0);
      }
    `,
    fragmentShader: `
      uniform vec3 uColor;
      varying float vShore;
      void main() {
        float edge = smoothstep(0.0, 1.25, vShore);
        gl_FragColor = vec4(uColor * mix(0.82, 1.0, edge), mix(0.18, 0.72, edge));
      }
    `,
  }), [profile.wobble])
  useEffect(() => () => { for (const geometry of geometries) geometry.dispose(); material.dispose() }, [geometries, material])
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
