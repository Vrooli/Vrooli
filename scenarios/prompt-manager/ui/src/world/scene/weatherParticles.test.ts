import { describe, expect, it } from 'vitest'
import { tuning, withTuningOverride } from '../config'
import { createParticleGeometry, createParticleMaterial } from './weatherParticles'

describe('weather particle levers', () => {
  it('preserves every default particle position exactly', () => {
    const count = 40
    const expected = new Float32Array(count * 3)
    for (let index = 0; index < count; index += 1) {
      const angle = index * 2.399963
      const radius = Math.sqrt(index / count) * 26
      expected[index * 3] = Math.sin(angle) * radius
      expected[index * 3 + 1] = (index * 7.13) % 24
      expected[index * 3 + 2] = Math.cos(angle) * radius
    }
    const geometry = createParticleGeometry(count, tuning.weather.particles)
    expect(geometry.getAttribute('position').array).toEqual(expected)
    geometry.dispose()
  })

  it.each(['rain', 'snow'] as const)('preserves the %s shader defaults as uniforms', (id) => {
    const material = createParticleMaterial(tuning.weather.states[id], tuning.weather.particles)
    expect(material.uniforms.uSpeed?.value).toBe(id === 'snow' ? 2.5 : 12)
    expect(material.uniforms.uSize?.value).toBe(id === 'snow' ? 0.18 : 0.08)
    expect(material.uniforms.uColor?.value.getHexString()).toBe(id === 'snow' ? 'f5fbff' : '82b7d2')
    expect(material.uniforms.uColumnHeight?.value).toBe(24)
    expect(material.uniforms.uPointSizeScale?.value).toBe(300)
    expect(material.uniforms.uOpacity?.value).toBe(0.78)
    material.dispose()
  })

  it('applies overrides to geometry and shader parameters', () => {
    const changed = withTuningOverride({ weather: {
      particles: { columnRadius: 4, columnHeight: 8, verticalStride: 2, pointSizeScale: 200, opacity: 0.4 },
      states: { rain: { particleFallSpeed: 5, particleSize: 0.2, particleColor: '#123456' } },
    } }).weather
    const geometry = createParticleGeometry(20, changed.particles)
    const positions = geometry.getAttribute('position')
    for (let i = 0; i < positions.count; i += 1) {
      expect(Math.hypot(positions.getX(i), positions.getZ(i))).toBeLessThanOrEqual(4)
      expect(positions.getY(i)).toBe((i * 2) % 8)
    }
    const material = createParticleMaterial(changed.states.rain, changed.particles)
    expect(material.uniforms.uSpeed?.value).toBe(5)
    expect(material.uniforms.uSize?.value).toBe(0.2)
    expect(material.uniforms.uColor?.value.getHexString()).toBe('123456')
    expect(material.uniforms.uColumnHeight?.value).toBe(8)
    expect(material.uniforms.uPointSizeScale?.value).toBe(200)
    expect(material.uniforms.uOpacity?.value).toBe(0.4)
    geometry.dispose()
    material.dispose()
  })
})
