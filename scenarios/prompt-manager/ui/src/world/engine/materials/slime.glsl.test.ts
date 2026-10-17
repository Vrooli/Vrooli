import { describe, expect, it } from 'vitest'
import {
  FRAGMENT_COLOR_INJECTION,
  FRAGMENT_COMMON_INJECTION,
  SLIME_SHADER_MARKER,
  VERTEX_COMMON_INJECTION,
  VERTEX_DISPLACEMENT_INJECTION,
} from './slime.glsl'

describe('slime GLSL', () => {
  it('carries the double-injection marker', () => {
    expect(VERTEX_COMMON_INJECTION).toContain(SLIME_SHADER_MARKER)
  })

  it('declares the uniforms and per-instance attributes the material binds', () => {
    for (const name of ['uTime', 'uWobbleIntensity', 'uWobbleScale', 'uWobbleSpeed']) {
      expect(VERTEX_COMMON_INJECTION).toContain(`uniform float ${name}`)
    }
    for (const name of ['aSeed', 'aTimeShift', 'aSquash']) {
      expect(VERTEX_COMMON_INJECTION).toContain(`attribute float ${name}`)
    }
    expect(VERTEX_COMMON_INJECTION).toContain('attribute vec3 aColor')
  })

  it('contains the simplex noise implementation', () => {
    expect(VERTEX_COMMON_INJECTION).toContain('float snoise(vec3 v)')
    expect(VERTEX_COMMON_INJECTION).toContain('taylorInvSqrt')
  })

  it('applies squash exactly once, in the shader', () => {
    expect(VERTEX_DISPLACEMENT_INJECTION.match(/transformed\.y \*= aSquash/g)).toHaveLength(1)
    expect(VERTEX_DISPLACEMENT_INJECTION).toContain('transformed += normal * wobble')
  })

  it('routes the instance colour into the fragment stage', () => {
    expect(FRAGMENT_COMMON_INJECTION).toContain('varying vec3 vSlimeColor')
    expect(FRAGMENT_COLOR_INJECTION).toContain('diffuseColor.rgb *= vSlimeColor')
  })
})
