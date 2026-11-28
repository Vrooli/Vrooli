import { describe, expect, it } from 'vitest'
import { tuning, withTuningOverride } from '../../config'
import { createSlimeMaterial, setSlimeWobble, updateSlimeMaterial } from './slime'
import { SLIME_SHADER_MARKER } from './slime.glsl'

describe('slime material', () => {
  it('preserves physical material and animation defaults', () => {
    const material = createSlimeMaterial(tuning.actor, true)
    expect(material.color.getHexString()).toBe('ffffff')
    expect(material.sheenColor.getHexString()).toBe('ffffff')
    expect(material.roughness).toBe(0.55)
    expect(material.clearcoat).toBe(0.6)
    expect(material.clearcoatRoughness).toBe(0.35)
    expect(material.sheen).toBe(0.4)
    expect(material.slime.uWobbleScale.value).toBe(3)
    expect(material.slime.uWobbleSpeed.value).toBe(1.5)
    material.dispose()
  })

  it('applies colour, surface and animation overrides to the material', () => {
    const actor = withTuningOverride({ actor: { material: { color: '#123456', sheenColor: '#abcdef', roughness: 0.2, wobbleSpeed: 4 } } }).actor
    const material = createSlimeMaterial(actor, true)
    expect(material.color.getHexString()).toBe('123456')
    expect(material.sheenColor.getHexString()).toBe('abcdef')
    expect(material.roughness).toBe(0.2)
    expect(material.slime.uWobbleSpeed.value).toBe(4)
    material.dispose()
  })

  it('injects the wobble, squash and instance colour once', () => {
    const material = createSlimeMaterial(tuning.actor, true)
    const shader = {
      uniforms: {} as Record<string, { value: unknown }>,
      vertexShader: '#include <common>\nvoid main() {\n#include <begin_vertex>\n}',
      fragmentShader: '#include <common>\nvoid main() {\n#include <color_fragment>\n}',
    }
    material.onBeforeCompile(shader as never, {} as never)
    expect(shader.vertexShader).toContain(SLIME_SHADER_MARKER)
    expect(shader.vertexShader).toContain('transformed.y *= aSquash')
    expect(shader.fragmentShader).toContain('diffuseColor.rgb *= vSlimeColor')
    expect(shader.uniforms.uWobbleIntensity?.value).toBe(tuning.actor.wobbleIntensity)
    const before = shader.vertexShader
    material.onBeforeCompile(shader as never, {} as never)
    expect(shader.vertexShader).toBe(before)
  })

  it('wobble follows the profile flag through the shared uniform', () => {
    const material = createSlimeMaterial(tuning.actor, false)
    expect(material.slime.uWobbleIntensity.value).toBe(0)
    setSlimeWobble(material, tuning.actor, true)
    expect(material.slime.uWobbleIntensity.value).toBe(tuning.actor.wobbleIntensity)
    expect(material.customProgramCacheKey()).toBe('world-slime')
  })
  it('updates surface and wobble values without replacing animation handles or recompiling scalars', () => {
    const material = createSlimeMaterial(tuning.actor, true)
    const uniforms = material.slime, color = material.color, sheenColor = material.sheenColor, version = material.version
    material.slime.uTime.value = 123
    const actor = withTuningOverride({ actor: { wobbleIntensity: .2, material: { color: '#123456', sheenColor: '#abcdef', roughness: .27, clearcoat: .3, clearcoatRoughness: .4, sheen: .2, wobbleScale: 7, wobbleSpeed: 5 } } }).actor
    updateSlimeMaterial(material, actor, true)
    expect(material.slime).toBe(uniforms)
    expect(material.color).toBe(color)
    expect(material.sheenColor).toBe(sheenColor)
    expect(material.color.getHexString()).toBe('123456')
    expect(material.sheenColor.getHexString()).toBe('abcdef')
    expect(material.roughness).toBe(.27)
    expect(material.clearcoatRoughness).toBe(.4)
    expect(material.slime.uTime.value).toBe(123)
    expect(material.slime.uWobbleIntensity.value).toBe(.2)
    expect(material.slime.uWobbleScale.value).toBe(7)
    expect(material.slime.uWobbleSpeed.value).toBe(5)
    expect(material.version).toBe(version)
    updateSlimeMaterial(material, actor, false)
    expect(material.slime.uWobbleIntensity.value).toBe(0)
    expect(material.slime.uTime.value).toBe(123)
    material.dispose()
  })

  it('retains Three feature invalidation when clearcoat and sheen cross zero', () => {
    const material = createSlimeMaterial(tuning.actor, true), version = material.version
    const actor = withTuningOverride({ actor: { material: { clearcoat: 0, sheen: 0 } } }).actor
    updateSlimeMaterial(material, actor, true)
    expect(material.version).toBeGreaterThan(version)
    expect(material.clearcoat).toBe(0)
    expect(material.sheen).toBe(0)
    material.dispose()
  })

})
