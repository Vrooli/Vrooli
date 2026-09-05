import { describe, expect, it } from 'vitest'
import { tuning, withTuningOverride } from '../config'
import { terrainMaterialSettings, terrainTintVariation } from './terrainAppearance'
import { createWaterMaterial } from './waterAppearance'

describe('terrain and water appearance settings', () => {
  it('preserves the former terrain colour formula exactly across coordinates and weather strengths', () => {
    for (let x = -90; x <= 90; x += 9) for (let z = -90; z <= 90; z += 9) for (const strength of [0, 0.25, 1]) {
      const expected = strength * (0.7 + Math.sin(x * 0.17 + z * 0.11) * 0.15 + Math.sin(x * 0.07 - z * 0.19) * 0.15)
      expect(terrainTintVariation(x, z, strength, tuning.visual.terrain)).toBe(expected)
    }
    for (const wetness of [0, 0.25, 0.8, 1]) {
      expect(terrainMaterialSettings(wetness, tuning.visual.terrain)).toEqual({
        color: wetness > 0 ? '#d4dce0' : '#ffffff', roughness: Math.max(0.3, 1 - wetness * 0.65),
      })
    }
  })

  it('applies terrain appearance overrides', () => {
    const settings = withTuningOverride({ visual: { terrain: { tintBase: 0.4, tintAmplitude: 0, wetColor: '#123456', wetRoughnessScale: 1, minimumRoughness: 0.1 } } }).visual.terrain
    expect(terrainTintVariation(7, 9, 0.5, settings)).toBe(0.2)
    expect(terrainMaterialSettings(1, settings)).toEqual({ color: '#123456', roughness: 0.1 })
  })

  it('preserves default water shader parameters', () => {
    const material = createWaterMaterial(tuning.visual.water, true)
    const values = Object.fromEntries(Object.entries(material.uniforms).map(([key, uniform]) => [key, key === 'uColor' ? uniform.value.getHexString() : uniform.value]))
    expect(values).toEqual({
      uTime: 0, uWobble: 1, uColor: '4f9db8', uWaveFrequencyX: 0.18, uWaveFrequencyZ: 0.16,
      uWaveSpeed: 1, uCrossWaveSpeed: 0.7, uWaveAmplitude: 0.025, uShoreFadeWidth: 1.25,
      uShoreBrightness: 0.82, uShoreOpacity: 0.18, uDeepOpacity: 0.72,
    })
    material.dispose()
  })

  it('applies water overrides independently of profile wobble', () => {
    const settings = withTuningOverride({ visual: { water: { color: '#123456', waveAmplitude: 0.05, shoreOpacity: 0.4 } } }).visual.water
    const material = createWaterMaterial(settings, false)
    expect(material.uniforms.uColor?.value.getHexString()).toBe('123456')
    expect(material.uniforms.uWaveAmplitude?.value).toBe(0.05)
    expect(material.uniforms.uShoreOpacity?.value).toBe(0.4)
    expect(material.uniforms.uWobble?.value).toBe(0)
    material.dispose()
  })

  it('preserves postprocessing defaults and validates their bounds', () => {
    expect(tuning.visual.post).toEqual({ aoRadius: 1.6, aoIntensity: 2.2, aoFalloff: 1, bloomThreshold: 1, bloomIntensity: 0.55, bloomRadius: 0.65 })
    expect(() => withTuningOverride({ visual: { post: { bloomRadius: 2 } } })).toThrow(/visual.post.bloomRadius/)
  })
})
