import { useFrame } from '@react-three/fiber'
import { useEffect, useMemo, useRef } from 'react'
import { BufferAttribute, BufferGeometry, Color, Group, ShaderMaterial } from 'three'
import type { QualityProfile, WeatherId, WeatherPreset, WeatherTuning } from '../config'

export function Weather({ id, preset, tuning, profile, getTarget }: { id: WeatherId; preset: WeatherPreset; tuning: WeatherTuning; profile: QualityProfile; getTarget: () => [number, number, number] }) {
  const group = useRef<Group | null>(null)
  const count = Math.floor(tuning.particleBaseCount * preset.particleRate * profile.weatherParticleScale)
  const geometry = useMemo(() => {
    const positions = new Float32Array(Math.max(1, count) * 3)
    for (let index = 0; index < count; index += 1) {
      const angle = index * 2.399963
      const radius = Math.sqrt(index / Math.max(1, count)) * 26
      positions[index * 3] = Math.sin(angle) * radius
      positions[index * 3 + 1] = (index * 7.13) % 24
      positions[index * 3 + 2] = Math.cos(angle) * radius
    }
    const result = new BufferGeometry()
    result.setAttribute('position', new BufferAttribute(positions, 3))
    return result
  }, [count])
  const material = useMemo(() => new ShaderMaterial({
    transparent: true,
    depthWrite: false,
    uniforms: {
      uTime: { value: 0 },
      uSpeed: { value: id === 'snow' ? 2.5 : 12 },
      uSize: { value: id === 'snow' ? 0.18 : 0.08 },
      uColor: { value: new Color(id === 'snow' ? '#f5fbff' : '#82b7d2') },
    },
    vertexShader: `
      uniform float uTime;
      uniform float uSpeed;
      uniform float uSize;
      void main() {
        vec3 p = position;
        p.y = mod(position.y - uTime * uSpeed + 24.0, 24.0);
        vec4 mvPosition = modelViewMatrix * vec4(p, 1.0);
        gl_PointSize = uSize * 300.0 / max(1.0, -mvPosition.z);
        gl_Position = projectionMatrix * mvPosition;
      }
    `,
    fragmentShader: `
      uniform vec3 uColor;
      void main() {
        vec2 point = gl_PointCoord - vec2(0.5);
        if (dot(point, point) > 0.25) discard;
        gl_FragColor = vec4(uColor, 0.78);
      }
    `,
  }), [id])
  useEffect(() => () => { geometry.dispose(); material.dispose() }, [geometry, material])
  useFrame((state) => {
    const target = getTarget()
    group.current?.position.set(target[0], target[1], target[2])
    const time = material.uniforms.uTime
    if (time) time.value = state.clock.elapsedTime
  })
  if (count === 0 || (id !== 'rain' && id !== 'snow')) return null
  return <group ref={group} name="weather"><points geometry={geometry} material={material} /></group>
}
