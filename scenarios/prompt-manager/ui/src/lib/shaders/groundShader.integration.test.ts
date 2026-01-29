/**
 * Ground Shader Integration Tests
 *
 * Tests the complete shader + material binding pipeline end-to-end:
 * - Material creation and shader binding
 * - onBeforeCompile injection process
 * - Uniform synchronization
 * - Config updates without rebinding
 *
 * These tests complement the unit tests in groundShader.test.ts by verifying
 * the full integration flow rather than individual GLSL validation.
 */

import { describe, it, expect } from 'vitest'
import * as THREE from 'three'
import { bindGroundShader, syncGroundShader, type GroundShaderConfig, type GroundShaderUserData } from './groundShader'
import {
  GROUND_SHADER_MARKER,
  GROUND_SHADER_UNIFORMS,
  VERTEX_COMMON_INJECTION,
  FRAGMENT_COMMON_INJECTION,
} from './glsl/ground.glsl'

// =============================================================================
// TEST FIXTURES
// =============================================================================

/** Create a minimal test configuration */
function createTestConfig(overrides?: Partial<GroundShaderConfig>): GroundShaderConfig {
  const mockTexture = new THREE.DataTexture(new Uint8Array([255, 255, 255, 255]), 1, 1)

  return {
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
    ...overrides,
  }
}

/** Create a mock shader object as Three.js would provide */
function createMockShader(): THREE.Shader {
  return {
    uniforms: {} as Record<string, THREE.IUniform>,
    vertexShader: `
      #include <common>
      void main() {
        vec3 transformed = position;
        #include <worldpos_vertex>
        gl_Position = projectionMatrix * modelViewMatrix * vec4(transformed, 1.0);
      }
    `,
    fragmentShader: `
      #define USE_MAP
      #include <common>
      void main() {
        vec4 diffuseColor = vec4(1.0);
        #include <map_fragment>
        gl_FragColor = diffuseColor;
      }
    `,
  }
}

/** Type-safe uniform access - uniforms are guaranteed to be set after onBeforeCompile */
function getUniform<T>(shader: THREE.Shader, name: string): T {
  const uniform = shader.uniforms[name]
  if (!uniform) throw new Error(`Uniform ${name} not found`)
  return uniform.value as T
}

// =============================================================================
// INTEGRATION TESTS
// =============================================================================

describe('groundShader integration', () => {
  describe('full binding pipeline', () => {
    it('completes full bind → compile → sync cycle', () => {
      const material = new THREE.MeshStandardMaterial({ color: 0xffffff })
      const config = createTestConfig()

      // Step 1: Bind shader to material
      bindGroundShader(material, config)

      // Verify binding state
      expect(material.onBeforeCompile).toBeDefined()
      expect(material.userData.groundShader).toBeDefined()
      // needsUpdate is set by bindGroundShader

      // Step 2: Simulate Three.js compilation
      const mockShader = createMockShader()
      material.onBeforeCompile(mockShader, {} as THREE.WebGLRenderer)

      // Verify shader was transformed
      expect(mockShader.vertexShader).toContain(GROUND_SHADER_MARKER)
      expect(mockShader.fragmentShader).toContain(GROUND_SHADER_MARKER)

      // Verify uniforms were added
      expect(mockShader.uniforms.uBaseUvRepeat).toBeDefined()
      expect(getUniform<number>(mockShader, 'uBaseUvRepeat')).toBe(10)

      // Step 3: Sync with new config
      const newConfig = createTestConfig({ baseUvRepeat: 20, stochasticEnabled: false })
      syncGroundShader(material, newConfig)

      // Verify uniforms were updated
      expect(getUniform<number>(mockShader, 'uBaseUvRepeat')).toBe(20)
      expect(getUniform<number>(mockShader, 'uStochasticEnabled')).toBe(0)
    })

    it('handles multiple materials independently', () => {
      const material1 = new THREE.MeshStandardMaterial({ color: 0xff0000 })
      const material2 = new THREE.MeshStandardMaterial({ color: 0x00ff00 })

      const config1 = createTestConfig({ baseUvRepeat: 10 })
      const config2 = createTestConfig({ baseUvRepeat: 20 })

      // Bind different configs
      bindGroundShader(material1, config1)
      bindGroundShader(material2, config2)

      // Each material should have its own config
      expect((material1.userData as GroundShaderUserData).groundShader?.config.baseUvRepeat).toBe(10)
      expect((material2.userData as GroundShaderUserData).groundShader?.config.baseUvRepeat).toBe(20)

      // Compile both
      const shader1 = createMockShader()
      const shader2 = createMockShader()

      material1.onBeforeCompile(shader1, {} as THREE.WebGLRenderer)
      material2.onBeforeCompile(shader2, {} as THREE.WebGLRenderer)

      // Each should have correct uniform values
      expect(getUniform<number>(shader1, 'uBaseUvRepeat')).toBe(10)
      expect(getUniform<number>(shader2, 'uBaseUvRepeat')).toBe(20)
    })
  })

  describe('projection mode switching', () => {
    it('switches from UV to triplanar projection', () => {
      const material = new THREE.MeshStandardMaterial()
      const uvConfig = createTestConfig({ projection: 'uv' })

      bindGroundShader(material, uvConfig)
      const shader = createMockShader()
      material.onBeforeCompile(shader, {} as THREE.WebGLRenderer)

      // UV projection: uUseTriplanar = 0
      expect(getUniform<number>(shader, 'uUseTriplanar')).toBe(0)

      // Switch to triplanar
      const triConfig = createTestConfig({ projection: 'triplanar' })
      syncGroundShader(material, triConfig)

      // Now uUseTriplanar = 1
      expect(getUniform<number>(shader, 'uUseTriplanar')).toBe(1)
    })

    it('preserves other uniforms when switching projection', () => {
      const material = new THREE.MeshStandardMaterial()
      const config = createTestConfig({
        projection: 'uv',
        baseUvRepeat: 15,
        macroIntensity: 0.25,
      })

      bindGroundShader(material, config)
      const shader = createMockShader()
      material.onBeforeCompile(shader, {} as THREE.WebGLRenderer)

      // Switch projection only
      syncGroundShader(material, { ...config, projection: 'triplanar' })

      // Other values unchanged
      expect(getUniform<number>(shader, 'uBaseUvRepeat')).toBe(15)
      expect(getUniform<number>(shader, 'uMacroIntensity')).toBe(0.25)
    })
  })

  describe('stochastic tiling toggle', () => {
    it('enables stochastic tiling via uniform', () => {
      const material = new THREE.MeshStandardMaterial()
      const config = createTestConfig({ stochasticEnabled: true })

      bindGroundShader(material, config)
      const shader = createMockShader()
      material.onBeforeCompile(shader, {} as THREE.WebGLRenderer)

      expect(getUniform<number>(shader, 'uStochasticEnabled')).toBe(1)
    })

    it('disables stochastic tiling via uniform', () => {
      const material = new THREE.MeshStandardMaterial()
      const config = createTestConfig({ stochasticEnabled: false })

      bindGroundShader(material, config)
      const shader = createMockShader()
      material.onBeforeCompile(shader, {} as THREE.WebGLRenderer)

      expect(getUniform<number>(shader, 'uStochasticEnabled')).toBe(0)
    })
  })

  describe('texture updates', () => {
    it('updates macro map texture reference', () => {
      const material = new THREE.MeshStandardMaterial()
      const texture1 = new THREE.DataTexture(new Uint8Array([255, 0, 0, 255]), 1, 1)
      const texture2 = new THREE.DataTexture(new Uint8Array([0, 255, 0, 255]), 1, 1)

      const config = createTestConfig({ macroMap: texture1 })
      bindGroundShader(material, config)

      const shader = createMockShader()
      material.onBeforeCompile(shader, {} as THREE.WebGLRenderer)

      expect(getUniform<THREE.DataTexture>(shader, 'uMacroMap')).toBe(texture1)

      // Update texture
      syncGroundShader(material, { ...config, macroMap: texture2 })

      expect(getUniform<THREE.DataTexture>(shader, 'uMacroMap')).toBe(texture2)
    })
  })

  describe('rotation handling', () => {
    it('updates rotation uniform in radians', () => {
      const material = new THREE.MeshStandardMaterial()
      const config = createTestConfig({ rotation: Math.PI / 4 }) // 45 degrees

      bindGroundShader(material, config)
      const shader = createMockShader()
      material.onBeforeCompile(shader, {} as THREE.WebGLRenderer)

      expect(getUniform<number>(shader, 'uRotation')).toBeCloseTo(Math.PI / 4, 5)

      // Update rotation
      syncGroundShader(material, { ...config, rotation: Math.PI / 2 })

      expect(getUniform<number>(shader, 'uRotation')).toBeCloseTo(Math.PI / 2, 5)
    })
  })

  describe('error resilience', () => {
    it('handles sync before compile gracefully', () => {
      const material = new THREE.MeshStandardMaterial()
      const config = createTestConfig()

      bindGroundShader(material, config)

      // Sync BEFORE onBeforeCompile is called (no shader yet)
      const newConfig = createTestConfig({ baseUvRepeat: 30 })

      // Should not throw
      expect(() => {
        syncGroundShader(material, newConfig)
      }).not.toThrow()

      // Config should be stored for when shader compiles
      expect((material.userData as GroundShaderUserData).groundShader?.config.baseUvRepeat).toBe(30)
    })

    it('handles sync on unbound material gracefully', () => {
      const material = new THREE.MeshStandardMaterial()
      const config = createTestConfig()

      // Sync on material that was never bound
      expect(() => {
        syncGroundShader(material, config)
      }).not.toThrow()

      // Should create userData entry
      expect((material.userData as GroundShaderUserData).groundShader?.config).toEqual(config)
    })

    it('prevents double injection on multiple compile calls', () => {
      const material = new THREE.MeshStandardMaterial()
      const config = createTestConfig()

      bindGroundShader(material, config)
      const shader = createMockShader()

      // First compile
      material.onBeforeCompile(shader, {} as THREE.WebGLRenderer)
      const vertexAfterFirst = shader.vertexShader

      // Second compile (Three.js might call this during hot reload etc.)
      material.onBeforeCompile(shader, {} as THREE.WebGLRenderer)

      // Should be identical (no double injection)
      expect(shader.vertexShader).toBe(vertexAfterFirst)
    })
  })

  describe('all uniforms coverage', () => {
    it('sets all documented uniforms', () => {
      const material = new THREE.MeshStandardMaterial()
      const config = createTestConfig({
        projection: 'triplanar',
        rotation: 0.5,
        baseUvRepeat: 12,
        baseWorldScale: 0.15,
        macroUvRepeat: 6,
        macroWorldScale: 0.06,
        macroIntensity: 0.2,
        triplanarSharpness: 5,
        stochasticEnabled: true,
      })

      bindGroundShader(material, config)
      const shader = createMockShader()
      material.onBeforeCompile(shader, {} as THREE.WebGLRenderer)

      // All uniforms from GROUND_SHADER_UNIFORMS should be set
      const uniformsSet = new Set(Object.keys(shader.uniforms))

      for (const uniformName of GROUND_SHADER_UNIFORMS) {
        expect(uniformsSet.has(uniformName), `Missing uniform: ${uniformName}`).toBe(true)
      }

      // Verify specific values
      expect(getUniform<number>(shader, 'uBaseUvRepeat')).toBe(12)
      expect(getUniform<number>(shader, 'uBaseWorldScale')).toBe(0.15)
      expect(getUniform<number>(shader, 'uMacroUvRepeat')).toBe(6)
      expect(getUniform<number>(shader, 'uMacroWorldScale')).toBe(0.06)
      expect(getUniform<number>(shader, 'uMacroIntensity')).toBe(0.2)
      expect(getUniform<number>(shader, 'uUseTriplanar')).toBe(1)
      expect(getUniform<number>(shader, 'uRotation')).toBe(0.5)
      expect(getUniform<number>(shader, 'uTriplanarSharpness')).toBe(5)
      expect(getUniform<number>(shader, 'uStochasticEnabled')).toBe(1)
    })
  })

  describe('custom program cache key', () => {
    it('sets customProgramCacheKey for shader sharing', () => {
      const material = new THREE.MeshStandardMaterial()
      const config = createTestConfig()

      bindGroundShader(material, config)

      // Should have custom cache key
      expect(material.customProgramCacheKey).toBeDefined()
      expect(typeof material.customProgramCacheKey).toBe('function')

      // Key should be consistent
      const key = material.customProgramCacheKey()
      expect(key).toBe('ground-texture-v2')
    })
  })
})

describe('groundShader GLSL content validation', () => {
  describe('injection markers', () => {
    it('vertex common injection contains marker', () => {
      expect(VERTEX_COMMON_INJECTION).toContain(GROUND_SHADER_MARKER)
    })

    it('fragment common injection contains marker', () => {
      expect(FRAGMENT_COMMON_INJECTION).toContain(GROUND_SHADER_MARKER)
    })
  })

  describe('varying consistency', () => {
    it('vertex declares vGroundWorldPosition', () => {
      expect(VERTEX_COMMON_INJECTION).toContain('varying vec3 vGroundWorldPosition')
    })

    it('fragment declares vGroundWorldPosition', () => {
      expect(FRAGMENT_COMMON_INJECTION).toContain('varying vec3 vGroundWorldPosition')
    })

    it('vertex declares vGroundWorldNormal', () => {
      expect(VERTEX_COMMON_INJECTION).toContain('varying vec3 vGroundWorldNormal')
    })

    it('fragment declares vGroundWorldNormal', () => {
      expect(FRAGMENT_COMMON_INJECTION).toContain('varying vec3 vGroundWorldNormal')
    })
  })

  describe('function definitions', () => {
    it('defines rotateUv function', () => {
      expect(FRAGMENT_COMMON_INJECTION).toContain('vec2 rotateUv(')
    })

    it('defines hash22 function for stochastic tiling', () => {
      expect(FRAGMENT_COMMON_INJECTION).toContain('vec2 hash22(')
    })

    it('defines sampleStochastic function', () => {
      expect(FRAGMENT_COMMON_INJECTION).toContain('vec4 sampleStochastic(')
    })

    it('defines sampleTriplanar function', () => {
      expect(FRAGMENT_COMMON_INJECTION).toContain('vec4 sampleTriplanar(')
    })

    it('defines sampleProjected function', () => {
      expect(FRAGMENT_COMMON_INJECTION).toContain('vec4 sampleProjected(')
    })
  })
})
