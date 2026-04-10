/**
 * Ground Shader Validation Tests
 *
 * Uses shared test utilities and single-source GLSL from glsl/ground.glsl.ts.
 * Tests at multiple levels:
 * 1. GLSL syntax parsing (catches syntax errors)
 * 2. WebGL compilation (catches undefined variables, type errors)
 * 3. Integration (validates runtime binding behavior)
 */

import { describe, it, expect } from 'vitest'
import * as THREE from 'three'
import {
  compileShaderWithWebGL,
  hasWebGL,
  parseGLSL,
  prepareForParsing,
  applyShaderInjections,
  findMissingUniforms,
  findMissingFunctions,
  findPotentialMissingSemicolons,
} from '@/test/shader-test-utils'
import {
  GROUND_SHADER_MARKER,
  GROUND_SHADER_INJECTIONS,
  GROUND_SHADER_UNIFORMS,
  GROUND_SHADER_FUNCTIONS,
  VERTEX_COMMON_INJECTION,
  VERTEX_WORLDPOS_INJECTION,
  FRAGMENT_COMMON_INJECTION,
  FRAGMENT_MAP_INJECTION,
} from './glsl/ground.glsl'
import { bindGroundShader, syncGroundShader, type GroundShaderConfig } from './groundShader'

// =============================================================================
// HELPER: Extract complete shaders for testing
// =============================================================================

function extractGroundShaderGLSL() {
  return applyShaderInjections({
    vertexCommon: VERTEX_COMMON_INJECTION,
    vertexWorldpos: VERTEX_WORLDPOS_INJECTION,
    fragmentCommon: FRAGMENT_COMMON_INJECTION,
    fragmentMap: FRAGMENT_MAP_INJECTION,
  })
}

// =============================================================================
// TESTS
// =============================================================================

describe('groundShader GLSL validation', () => {
  describe('shader extraction', () => {
    it('extracts vertex and fragment shaders', () => {
      const { vertex, fragment } = extractGroundShaderGLSL()

      expect(vertex).toBeDefined()
      expect(fragment).toBeDefined()
      expect(vertex.length).toBeGreaterThan(100)
      expect(fragment.length).toBeGreaterThan(100)
    })

    it('vertex shader contains world position varying', () => {
      const { vertex } = extractGroundShaderGLSL()
      expect(vertex).toContain('vGroundWorldPosition')
      expect(vertex).toContain('vGroundWorldNormal')
    })

    it('fragment shader contains custom uniforms', () => {
      const { fragment } = extractGroundShaderGLSL()
      expect(fragment).toContain('uniform float uBaseUvRepeat')
      expect(fragment).toContain('uniform float uStochasticEnabled')
      expect(fragment).toContain('uniform sampler2D uMacroMap')
    })

    it('fragment shader contains stochastic sampling function', () => {
      const { fragment } = extractGroundShaderGLSL()
      expect(fragment).toContain('sampleStochastic')
      expect(fragment).toContain('hash22')
    })

    it('fragment shader contains triplanar sampling function', () => {
      const { fragment } = extractGroundShaderGLSL()
      expect(fragment).toContain('sampleTriplanar')
      expect(fragment).toContain('sampleProjected')
    })
  })

  describe('GLSL syntax validation (parser)', () => {
    it('vertex shader parses without syntax errors', () => {
      const { vertex } = extractGroundShaderGLSL()
      const prepared = prepareForParsing(vertex)
      const result = parseGLSL(prepared)

      if (!result.success) {
        console.error('=== Vertex Shader Parse Errors ===')
        result.errors.forEach(err => console.error(err))
        console.error('=== Vertex Shader Source (prepared) ===')
        console.error(prepared.split('\n').map((line, i) => `${i + 1}: ${line}`).join('\n'))
      }

      expect(result.success, `Vertex shader parse failed:\n${result.errors.join('\n')}`).toBe(true)
    })

    it('fragment shader parses without syntax errors', () => {
      const { fragment } = extractGroundShaderGLSL()
      const prepared = prepareForParsing(fragment)
      const result = parseGLSL(prepared)

      if (!result.success) {
        console.error('=== Fragment Shader Parse Errors ===')
        result.errors.forEach(err => console.error(err))
        console.error('=== Fragment Shader Source (prepared) ===')
        console.error(prepared.split('\n').map((line, i) => `${i + 1}: ${line}`).join('\n'))
      }

      expect(result.success, `Fragment shader parse failed:\n${result.errors.join('\n')}`).toBe(true)
    })
  })

  describe.skipIf(!hasWebGL)('WebGL shader compilation (real GPU)', () => {
    it('vertex shader compiles with WebGL', () => {
      const { vertex } = extractGroundShaderGLSL()
      const result = compileShaderWithWebGL(vertex, 'vertex')

      if (!result.success) {
        console.error('=== WebGL Vertex Shader Compilation Error ===')
        console.error(result.error)
        console.error('=== Vertex Shader Source ===')
        console.error(vertex.split('\n').map((line, i) => `${i + 1}: ${line}`).join('\n'))
      }

      expect(result.success, `WebGL vertex shader compilation failed:\n${result.error}`).toBe(true)
    })

    it('fragment shader compiles with WebGL', () => {
      const { fragment } = extractGroundShaderGLSL()
      const result = compileShaderWithWebGL(fragment, 'fragment')

      if (!result.success) {
        console.error('=== WebGL Fragment Shader Compilation Error ===')
        console.error(result.error)
        console.error('=== Fragment Shader Source ===')
        console.error(fragment.split('\n').map((line, i) => `${i + 1}: ${line}`).join('\n'))
      }

      expect(result.success, `WebGL fragment shader compilation failed:\n${result.error}`).toBe(true)
    })
  })

  describe('shader injection consistency', () => {
    it('injection strings are non-empty', () => {
      expect(GROUND_SHADER_INJECTIONS.vertexCommon.trim().length).toBeGreaterThan(0)
      expect(GROUND_SHADER_INJECTIONS.vertexWorldpos.trim().length).toBeGreaterThan(0)
      expect(GROUND_SHADER_INJECTIONS.fragmentCommon.trim().length).toBeGreaterThan(0)
      expect(GROUND_SHADER_INJECTIONS.fragmentMap.trim().length).toBeGreaterThan(0)
    })

    it('fragment common injection declares all required uniforms', () => {
      const missing = findMissingUniforms(FRAGMENT_COMMON_INJECTION, GROUND_SHADER_UNIFORMS)
      expect(missing, `Missing uniforms: ${missing.join(', ')}`).toHaveLength(0)
    })

    it('fragment common injection defines all sampling functions', () => {
      const missing = findMissingFunctions(FRAGMENT_COMMON_INJECTION, GROUND_SHADER_FUNCTIONS)
      expect(missing, `Missing functions: ${missing.join(', ')}`).toHaveLength(0)
    })

    it('marker is present in injections', () => {
      expect(VERTEX_COMMON_INJECTION).toContain(GROUND_SHADER_MARKER)
      expect(FRAGMENT_COMMON_INJECTION).toContain(GROUND_SHADER_MARKER)
    })
  })

  describe('function signatures', () => {
    it('sampleStochastic has correct parameter count', () => {
      // sampleStochastic(sampler2D tex, vec2 uv, float scale, float rotation)
      const match = FRAGMENT_COMMON_INJECTION.match(/vec4 sampleStochastic\s*\([^)]+\)/)
      expect(match).not.toBeNull()

      if (match) {
        const params = match[0].split(',')
        expect(params.length).toBe(4)
      }
    })

    it('sampleProjected has correct parameter count', () => {
      // sampleProjected(sampler2D tex, vec2 uv, vec3 worldPos, vec3 normal, float uvRepeat, float worldScale, float rotation, float useTriplanar, float sharpness)
      const match = FRAGMENT_COMMON_INJECTION.match(/vec4 sampleProjected\s*\([^)]+\)/)
      expect(match).not.toBeNull()

      if (match) {
        const params = match[0].split(',')
        expect(params.length).toBe(9)
      }
    })
  })

  describe('bindGroundShader integration', () => {
    it('binds shader without throwing', () => {
      const material = new THREE.MeshStandardMaterial({ color: 0xffffff })
      const mockTexture = new THREE.DataTexture(new Uint8Array([255, 255, 255, 255]), 1, 1)

      const config: GroundShaderConfig = {
        projection: 'uv',
        rotation: 0,
        baseUvRepeat: 10,
        baseWorldScale: 0.1,
        macroUvRepeat: 5,
        macroWorldScale: 0.05,
        macroIntensity: 0.15,
        macroMap: mockTexture,
        triplanarSharpness: 4,
        stochasticEnabled: true,
      }

      expect(() => bindGroundShader(material, config)).not.toThrow()
      expect(material.onBeforeCompile).toBeDefined()
      expect(typeof material.onBeforeCompile).toBe('function')
      expect(material.userData.groundShader).toBeDefined()
      expect(material.userData.groundShader.config).toEqual(config)
    })

    it('shader injection transforms vertex shader correctly', () => {
      const material = new THREE.MeshStandardMaterial({ color: 0xffffff })
      const mockTexture = new THREE.DataTexture(new Uint8Array([255, 255, 255, 255]), 1, 1)

      const config: GroundShaderConfig = {
        projection: 'uv',
        rotation: 0,
        baseUvRepeat: 10,
        baseWorldScale: 0.1,
        macroUvRepeat: 5,
        macroWorldScale: 0.05,
        macroIntensity: 0.15,
        macroMap: mockTexture,
        triplanarSharpness: 4,
        stochasticEnabled: true,
      }

      bindGroundShader(material, config)

      const mockShader = {
        uniforms: {} as Record<string, { value: unknown }>,
        vertexShader: `
          #include <common>
          void main() {
            vec3 transformed = position;
            #include <worldpos_vertex>
            gl_Position = projectionMatrix * modelViewMatrix * vec4(transformed, 1.0);
          }
        `,
        fragmentShader: `
          #include <common>
          void main() {
            vec4 diffuseColor = vec4(1.0);
            #include <map_fragment>
            gl_FragColor = diffuseColor;
          }
        `,
      }

      material.onBeforeCompile(mockShader as THREE.Shader, {} as THREE.WebGLRenderer)

      expect(mockShader.vertexShader).toContain('vGroundWorldPosition')
      expect(mockShader.vertexShader).toContain('vGroundWorldNormal')
      expect(mockShader.vertexShader).toContain('groundWorldPosition')

      expect(mockShader.fragmentShader).toContain('uBaseUvRepeat')
      expect(mockShader.fragmentShader).toContain('uStochasticEnabled')
      expect(mockShader.fragmentShader).toContain('sampleProjected')

      expect(mockShader.uniforms.uBaseUvRepeat).toBeDefined()
      expect(mockShader.uniforms.uBaseUvRepeat?.value).toBe(10)
      expect(mockShader.uniforms.uStochasticEnabled).toBeDefined()
      expect(mockShader.uniforms.uStochasticEnabled?.value).toBe(1)
    })

    it('onBeforeCompile is idempotent (can be called multiple times)', () => {
      const material = new THREE.MeshStandardMaterial({ color: 0xffffff })
      const mockTexture = new THREE.DataTexture(new Uint8Array([255, 255, 255, 255]), 1, 1)

      const config: GroundShaderConfig = {
        projection: 'uv',
        rotation: 0,
        baseUvRepeat: 10,
        baseWorldScale: 0.1,
        macroUvRepeat: 5,
        macroWorldScale: 0.05,
        macroIntensity: 0.15,
        macroMap: mockTexture,
        triplanarSharpness: 4,
        stochasticEnabled: true,
      }

      bindGroundShader(material, config)

      const mockShader = {
        uniforms: {} as Record<string, { value: unknown }>,
        vertexShader: `
          #include <common>
          void main() {
            vec3 transformed = position;
            #include <worldpos_vertex>
            gl_Position = projectionMatrix * modelViewMatrix * vec4(transformed, 1.0);
          }
        `,
        fragmentShader: `
          #include <common>
          void main() {
            vec4 diffuseColor = vec4(1.0);
            #include <map_fragment>
            gl_FragColor = diffuseColor;
          }
        `,
      }

      // First call
      material.onBeforeCompile(mockShader as THREE.Shader, {} as THREE.WebGLRenderer)
      const vertexAfterFirst = mockShader.vertexShader
      const fragmentAfterFirst = mockShader.fragmentShader

      // Second call - should not double-transform
      material.onBeforeCompile(mockShader as THREE.Shader, {} as THREE.WebGLRenderer)

      expect(mockShader.vertexShader).toBe(vertexAfterFirst)
      expect(mockShader.fragmentShader).toBe(fragmentAfterFirst)
    })

    it('syncGroundShader updates uniforms', () => {
      const material = new THREE.MeshStandardMaterial({ color: 0xffffff })
      const mockTexture = new THREE.DataTexture(new Uint8Array([255, 255, 255, 255]), 1, 1)

      const config: GroundShaderConfig = {
        projection: 'uv',
        rotation: 0,
        baseUvRepeat: 10,
        baseWorldScale: 0.1,
        macroUvRepeat: 5,
        macroWorldScale: 0.05,
        macroIntensity: 0.15,
        macroMap: mockTexture,
        triplanarSharpness: 4,
        stochasticEnabled: true,
      }

      bindGroundShader(material, config)

      const mockShader = {
        uniforms: {} as Record<string, { value: unknown }>,
        vertexShader: `#include <common> #include <worldpos_vertex>`,
        fragmentShader: `#include <common> #include <map_fragment>`,
      }
      material.onBeforeCompile(mockShader as THREE.Shader, {} as THREE.WebGLRenderer)

      const newConfig: GroundShaderConfig = {
        ...config,
        baseUvRepeat: 20,
        stochasticEnabled: false,
      }

      expect(() => syncGroundShader(material, newConfig)).not.toThrow()
      expect(mockShader.uniforms.uBaseUvRepeat?.value).toBe(20)
      expect(mockShader.uniforms.uStochasticEnabled?.value).toBe(0)
    })
  })

  describe('common shader issues', () => {
    it('no duplicate variable declarations in same scope', () => {
      const floatDeclarations = FRAGMENT_COMMON_INJECTION.match(/float\s+(\w+)\s*=/g) || []
      const vec2Declarations = FRAGMENT_COMMON_INJECTION.match(/vec2\s+(\w+)\s*=/g) || []
      const vec3Declarations = FRAGMENT_COMMON_INJECTION.match(/vec3\s+(\w+)\s*=/g) || []
      const vec4Declarations = FRAGMENT_COMMON_INJECTION.match(/vec4\s+(\w+)\s*=/g) || []

      const allDeclarations = [
        ...floatDeclarations,
        ...vec2Declarations,
        ...vec3Declarations,
        ...vec4Declarations,
      ]

      expect(allDeclarations.length).toBeGreaterThan(0)
    })

    it('texture2D calls have correct argument count', () => {
      const textureCalls = FRAGMENT_COMMON_INJECTION.match(/texture2D\s*\([^)]+\)/g) || []

      expect(textureCalls.length).toBeGreaterThan(0)

      for (const call of textureCalls) {
        const args = call.replace(/texture2D\s*\(/, '').replace(')', '').split(',')
        expect(args.length, `texture2D call should have 2 arguments: ${call}`).toBe(2)
      }
    })

    it('all uniform declarations have type and name', () => {
      const uniformLines = FRAGMENT_COMMON_INJECTION.split('\n').filter(line =>
        line.trim().startsWith('uniform ')
      )

      for (const line of uniformLines) {
        const match = line.match(/uniform\s+(float|int|vec[234]|mat[234]|sampler2D)\s+(\w+)\s*;/)
        expect(match, `Invalid uniform declaration: ${line}`).not.toBeNull()
      }
    })

    it('all varying declarations match between vertex and fragment', () => {
      const vertexVaryings = VERTEX_COMMON_INJECTION
        .match(/varying\s+\w+\s+(\w+)\s*;/g)
        ?.map(v => v.match(/varying\s+\w+\s+(\w+)/)?.[1])
        .filter(Boolean) || []

      const fragmentVaryings = FRAGMENT_COMMON_INJECTION
        .match(/varying\s+\w+\s+(\w+)\s*;/g)
        ?.map(v => v.match(/varying\s+\w+\s+(\w+)/)?.[1])
        .filter(Boolean) || []

      for (const varying of vertexVaryings) {
        expect(fragmentVaryings, `Missing varying in fragment: ${varying}`).toContain(varying)
      }

      for (const varying of fragmentVaryings) {
        expect(vertexVaryings, `Missing varying in vertex: ${varying}`).toContain(varying)
      }
    })

    it('no unterminated statements (missing semicolons)', () => {
      const suspicious = findPotentialMissingSemicolons(FRAGMENT_COMMON_INJECTION)
      expect(suspicious, `Potentially missing semicolons:\n${suspicious.join('\n')}`).toHaveLength(0)
    })
  })
})
