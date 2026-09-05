import { useFrame } from '@react-three/fiber'
import { useEffect, useMemo, useRef } from 'react'
import { Group } from 'three'
import { createParticleGeometry, createParticleMaterial } from './weatherParticles'
import type { QualityProfile, WeatherId, WeatherPreset, WeatherTuning } from '../config'

export function Weather({ id, preset, tuning, profile, getTarget }: { id: WeatherId; preset: WeatherPreset; tuning: WeatherTuning; profile: QualityProfile; getTarget: () => [number, number, number] }) {
  const group = useRef<Group | null>(null)
  const count = Math.floor(tuning.particleBaseCount * preset.particleRate * profile.weatherParticleScale)
  const geometry = useMemo(() => createParticleGeometry(count, tuning.particles), [count, tuning.particles])
  const material = useMemo(() => createParticleMaterial(preset, tuning.particles), [preset, tuning.particles])
  useEffect(() => () => geometry.dispose(), [geometry])
  useEffect(() => () => material.dispose(), [material])
  useFrame((state) => {
    const target = getTarget()
    group.current?.position.set(target[0], target[1], target[2])
    const time = material.uniforms.uTime
    if (time) time.value = state.clock.elapsedTime
  })
  if (count === 0 || (id !== 'rain' && id !== 'snow')) return null
  return <group ref={group} name="weather"><points geometry={geometry} material={material} /></group>
}
