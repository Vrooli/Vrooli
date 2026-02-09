/* eslint-disable @typescript-eslint/no-non-null-assertion */
/**
 * Slime Shader Validation Tests
 *
 * Tests shader GLSL strings, binding behavior, sync, and double-injection prevention.
 */

import { describe, it, expect } from 'vitest'
import * as THREE from 'three'
import {
  SLIME_SHADER_MARKER,
  VERTEX_COMMON_INJECTION,
  VERTEX_DISPLACEMENT_INJECTION,
} from '../glsl/slime.glsl'
import { bindSlimeShader, syncSlimeShader, type SlimeShaderConfig } from '../slimeShader'

// =============================================================================
// GLSL STRING VALIDATION
// =============================================================================

describe('slime GLSL strings', () => {
  it('marker is present in vertex common injection', () => {
    expect(VERTEX_COMMON_INJECTION).toContain(SLIME_SHADER_MARKER)
  })

  it('vertex common injection declares required uniforms', () => {
    expect(VERTEX_COMMON_INJECTION).toContain('uniform float uTime')
    expect(VERTEX_COMMON_INJECTION).toContain('uniform float uWobbleIntensity')
    expect(VERTEX_COMMON_INJECTION).toContain('uniform float uSquashY')
  })

  it('vertex common injection contains simplex noise function', () => {
    expect(VERTEX_COMMON_INJECTION).toContain('float snoise(vec3 v)')
    expect(VERTEX_COMMON_INJECTION).toContain('mod289')
    expect(VERTEX_COMMON_INJECTION).toContain('permute')
    expect(VERTEX_COMMON_INJECTION).toContain('taylorInvSqrt')
  })

  it('vertex displacement injection uses noise for wobble', () => {
    expect(VERTEX_DISPLACEMENT_INJECTION).toContain('snoise')
    expect(VERTEX_DISPLACEMENT_INJECTION).toContain('uWobbleIntensity')
    expect(VERTEX_DISPLACEMENT_INJECTION).toContain('normal')
  })

  it('vertex displacement injection applies squash/stretch', () => {
    expect(VERTEX_DISPLACEMENT_INJECTION).toContain('transformed.y *= uSquashY')
  })

  it('injection strings are non-empty', () => {
    expect(VERTEX_COMMON_INJECTION.trim().length).toBeGreaterThan(0)
    expect(VERTEX_DISPLACEMENT_INJECTION.trim().length).toBeGreaterThan(0)
  })
})

// =============================================================================
// BINDING TESTS
// =============================================================================

describe('bindSlimeShader', () => {
  const defaultConfig: SlimeShaderConfig = {
    wobbleIntensity: 0.02,
    wobbleSpeed: 1.5,
  }

  it('sets onBeforeCompile on material', () => {
    const material = new THREE.MeshPhysicalMaterial({ color: 0x6366f1 })

    bindSlimeShader(material, defaultConfig)

    expect(material.onBeforeCompile).toBeDefined()
    expect(typeof material.onBeforeCompile).toBe('function')
  })

  it('stores config in material.userData.slimeShader', () => {
    const material = new THREE.MeshPhysicalMaterial({ color: 0x6366f1 })

    bindSlimeShader(material, defaultConfig)

    expect(material.userData.slimeShader).toBeDefined()
    expect(material.userData.slimeShader.config).toEqual(defaultConfig)
  })

  it('sets customProgramCacheKey on material', () => {
    const material = new THREE.MeshPhysicalMaterial({ color: 0x6366f1 })

    bindSlimeShader(material, defaultConfig)

    expect(material.customProgramCacheKey).toBeDefined()
    expect(typeof material.customProgramCacheKey).toBe('function')
  })

  it('shader injection transforms vertex shader correctly', () => {
    const material = new THREE.MeshPhysicalMaterial({ color: 0x6366f1 })

    bindSlimeShader(material, defaultConfig)

    const mockShader = {
      uniforms: {} as Record<string, { value: unknown }>,
      vertexShader: `
        #include <common>
        void main() {
          #include <begin_vertex>
          gl_Position = projectionMatrix * modelViewMatrix * vec4(transformed, 1.0);
        }
      `,
      fragmentShader: `
        #include <common>
        void main() {
          gl_FragColor = vec4(1.0);
        }
      `,
    }

    material.onBeforeCompile(mockShader as THREE.Shader, {} as THREE.WebGLRenderer)

    expect(mockShader.vertexShader).toContain('uTime')
    expect(mockShader.vertexShader).toContain('uWobbleIntensity')
    expect(mockShader.vertexShader).toContain('uSquashY')
    expect(mockShader.vertexShader).toContain('snoise')
    expect(mockShader.vertexShader).toContain('transformed.y *= uSquashY')

    expect(mockShader.uniforms.uTime).toBeDefined()
    expect(mockShader.uniforms.uTime!.value).toBe(0)
    expect(mockShader.uniforms.uWobbleIntensity).toBeDefined()
    expect(mockShader.uniforms.uWobbleIntensity!.value).toBe(0.02)
    expect(mockShader.uniforms.uSquashY).toBeDefined()
    expect(mockShader.uniforms.uSquashY!.value).toBe(1.0)
  })

  it('prevents double-injection via marker', () => {
    const material = new THREE.MeshPhysicalMaterial({ color: 0x6366f1 })

    bindSlimeShader(material, defaultConfig)

    const mockShader = {
      uniforms: {} as Record<string, { value: unknown }>,
      vertexShader: `
        #include <common>
        void main() {
          #include <begin_vertex>
          gl_Position = projectionMatrix * modelViewMatrix * vec4(transformed, 1.0);
        }
      `,
      fragmentShader: `#include <common> void main() { gl_FragColor = vec4(1.0); }`,
    }

    // First call
    material.onBeforeCompile(mockShader as THREE.Shader, {} as THREE.WebGLRenderer)
    const vertexAfterFirst = mockShader.vertexShader

    // Second call - should not double-transform
    material.onBeforeCompile(mockShader as THREE.Shader, {} as THREE.WebGLRenderer)

    expect(mockShader.vertexShader).toBe(vertexAfterFirst)
  })
})

// =============================================================================
// SYNC TESTS
// =============================================================================

describe('syncSlimeShader', () => {
  it('updates uniform values', () => {
    const material = new THREE.MeshPhysicalMaterial({ color: 0x6366f1 })

    bindSlimeShader(material, {
      wobbleIntensity: 0.02,
      wobbleSpeed: 1.5,
    })

    const mockShader = {
      uniforms: {} as Record<string, { value: unknown }>,
      vertexShader: `#include <common> #include <begin_vertex>`,
      fragmentShader: `#include <common>`,
    }
    material.onBeforeCompile(mockShader as THREE.Shader, {} as THREE.WebGLRenderer)

    syncSlimeShader(material, 2.5, 0.85, 0.01)

    expect(mockShader.uniforms.uTime!.value).toBe(2.5)
    expect(mockShader.uniforms.uSquashY!.value).toBe(0.85)
    expect(mockShader.uniforms.uWobbleIntensity!.value).toBe(0.01)
  })

  it('does not throw when shader has not been compiled', () => {
    const material = new THREE.MeshPhysicalMaterial({ color: 0x6366f1 })

    bindSlimeShader(material, {
      wobbleIntensity: 0.02,
      wobbleSpeed: 1.5,
    })

    // Don't call onBeforeCompile - shader is not yet compiled
    expect(() => syncSlimeShader(material, 1.0, 1.0)).not.toThrow()
  })

  it('preserves wobbleIntensity when not provided', () => {
    const material = new THREE.MeshPhysicalMaterial({ color: 0x6366f1 })

    bindSlimeShader(material, {
      wobbleIntensity: 0.02,
      wobbleSpeed: 1.5,
    })

    const mockShader = {
      uniforms: {} as Record<string, { value: unknown }>,
      vertexShader: `#include <common> #include <begin_vertex>`,
      fragmentShader: `#include <common>`,
    }
    material.onBeforeCompile(mockShader as THREE.Shader, {} as THREE.WebGLRenderer)

    // Sync without wobbleIntensity override
    syncSlimeShader(material, 1.0, 0.9)

    expect(mockShader.uniforms.uTime!.value).toBe(1.0)
    expect(mockShader.uniforms.uSquashY!.value).toBe(0.9)
    // wobbleIntensity should remain at initial value
    expect(mockShader.uniforms.uWobbleIntensity!.value).toBe(0.02)
  })
})
