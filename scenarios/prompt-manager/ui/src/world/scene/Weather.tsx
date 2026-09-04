import { useFrame } from '@react-three/fiber'
import { useEffect, useMemo, useRef } from 'react'
import { BufferAttribute, BufferGeometry, Group, PointsMaterial } from 'three'
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
  const material = useMemo(() => new PointsMaterial({ color: id === 'snow' ? '#f5fbff' : '#82b7d2', size: id === 'snow' ? 0.18 : 0.08, transparent: true, opacity: 0.78, depthWrite: false }), [id])
  useEffect(() => () => { geometry.dispose(); material.dispose() }, [geometry, material])
  useFrame((_, delta) => {
    const target = getTarget()
    group.current?.position.set(target[0], target[1], target[2])
    const positions = geometry.getAttribute('position') as BufferAttribute
    const speed = id === 'snow' ? 2.5 : 12
    for (let index = 0; index < count; index += 1) {
      const y = positions.getY(index) - delta * speed
      positions.setY(index, y < 0 ? y + 24 : y)
    }
    positions.needsUpdate = true
  })
  if (count === 0 || (id !== 'rain' && id !== 'snow')) return null
  return <group ref={group} name="weather"><points geometry={geometry} material={material} /></group>
}
