/**
 * Sky Shader Validation Tests
 *
 * Uses shared test utilities to validate the sky gradient shader.
 * Tests WebGL compilation and GLSL syntax.
 */

import { describe, it, expect } from 'vitest'
import {
  compileShaderWithWebGL,
  parseGLSL,
  prepareForParsing,
  findMissingUniforms,
} from '@/test/shader-test-utils'
import {
  SKY_VERTEX_SHADER,
  SKY_FRAGMENT_SHADER,
  SKY_VERTEX_SHADER_STANDALONE,
  SKY_FRAGMENT_SHADER_STANDALONE,
  SKY_SHADER_UNIFORMS,
} from './glsl/sky.glsl'

describe('skyShader GLSL validation', () => {
  describe('WebGL shader compilation (real GPU)', () => {
    it('vertex shader compiles with WebGL', () => {
      // Use standalone version with Three.js built-ins for WebGL compilation
      const result = compileShaderWithWebGL(SKY_VERTEX_SHADER_STANDALONE, 'vertex')

      if (!result.success) {
        console.error('=== WebGL Sky Vertex Shader Compilation Error ===')
        console.error(result.error)
        console.error('=== Vertex Shader Source ===')
        console.error(SKY_VERTEX_SHADER_STANDALONE.split('\n').map((line, i) => `${i + 1}: ${line}`).join('\n'))
      }

      expect(result.success, `WebGL vertex shader compilation failed:\n${result.error}`).toBe(true)
    })

    it('fragment shader compiles with WebGL', () => {
      // Use standalone version with precision qualifier for WebGL compilation
      const result = compileShaderWithWebGL(SKY_FRAGMENT_SHADER_STANDALONE, 'fragment')

      if (!result.success) {
        console.error('=== WebGL Sky Fragment Shader Compilation Error ===')
        console.error(result.error)
        console.error('=== Fragment Shader Source ===')
        console.error(SKY_FRAGMENT_SHADER_STANDALONE.split('\n').map((line, i) => `${i + 1}: ${line}`).join('\n'))
      }

      expect(result.success, `WebGL fragment shader compilation failed:\n${result.error}`).toBe(true)
    })
  })

  describe('GLSL syntax validation (parser)', () => {
    it('vertex shader parses without syntax errors', () => {
      const prepared = prepareForParsing(SKY_VERTEX_SHADER)
      const result = parseGLSL(prepared)

      if (!result.success) {
        console.error('=== Sky Vertex Shader Parse Errors ===')
        result.errors.forEach(err => console.error(err))
      }

      expect(result.success, `Vertex shader parse failed:\n${result.errors.join('\n')}`).toBe(true)
    })

    it('fragment shader parses without syntax errors', () => {
      const prepared = prepareForParsing(SKY_FRAGMENT_SHADER)
      const result = parseGLSL(prepared)

      if (!result.success) {
        console.error('=== Sky Fragment Shader Parse Errors ===')
        result.errors.forEach(err => console.error(err))
      }

      expect(result.success, `Fragment shader parse failed:\n${result.errors.join('\n')}`).toBe(true)
    })
  })

  describe('shader content validation', () => {
    it('fragment shader declares all required uniforms', () => {
      const missing = findMissingUniforms(SKY_FRAGMENT_SHADER, SKY_SHADER_UNIFORMS)
      expect(missing, `Missing uniforms: ${missing.join(', ')}`).toHaveLength(0)
    })

    it('vertex shader outputs vWorldPosition', () => {
      expect(SKY_VERTEX_SHADER).toContain('varying vec3 vWorldPosition')
      expect(SKY_VERTEX_SHADER).toContain('vWorldPosition = worldPosition.xyz')
    })

    it('fragment shader uses vWorldPosition', () => {
      expect(SKY_FRAGMENT_SHADER).toContain('varying vec3 vWorldPosition')
      expect(SKY_FRAGMENT_SHADER).toContain('normalize(vWorldPosition)')
    })

    it('fragment shader uses smoothstep for gradients', () => {
      expect(SKY_FRAGMENT_SHADER).toContain('smoothstep')
    })

    it('fragment shader uses pow for exponential falloff', () => {
      expect(SKY_FRAGMENT_SHADER).toContain('pow')
    })

    it('fragment shader uses mix for color blending', () => {
      expect(SKY_FRAGMENT_SHADER).toContain('mix')
    })
  })

  describe('varying consistency', () => {
    it('vWorldPosition declared in both shaders', () => {
      const vertexVarying = SKY_VERTEX_SHADER.match(/varying\s+vec3\s+vWorldPosition/)
      const fragmentVarying = SKY_FRAGMENT_SHADER.match(/varying\s+vec3\s+vWorldPosition/)

      expect(vertexVarying, 'vWorldPosition not declared in vertex shader').not.toBeNull()
      expect(fragmentVarying, 'vWorldPosition not declared in fragment shader').not.toBeNull()
    })
  })
})
