import { describe, expect, it } from 'vitest'
import { tuning } from '../../config'
import { createSlimeMaterial, setSlimeWobble } from './slime'
import { SLIME_SHADER_MARKER } from './slime.glsl'

describe('slime material', () => {
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
    expect(shader.fragmentShader).toContain('diffuseColor.rgb = vSlimeColor')
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
})
