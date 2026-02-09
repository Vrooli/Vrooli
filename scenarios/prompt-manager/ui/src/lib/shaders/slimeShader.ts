/**
 * Slime Shader Runtime Binding
 *
 * Binds custom slime vertex wobble shader to Three.js MeshPhysicalMaterial.
 * Uses onBeforeCompile to inject GLSL for:
 * - Simplex 3D noise vertex displacement (organic wobble)
 * - Squash/stretch Y-axis deformation
 *
 * GLSL source is imported from ./glsl/slime.glsl.ts (single source of truth).
 */

import * as THREE from 'three'
import {
  SLIME_SHADER_MARKER,
  VERTEX_COMMON_INJECTION,
  VERTEX_DISPLACEMENT_INJECTION,
} from './glsl/slime.glsl'

export interface SlimeShaderConfig {
  /** Intensity of vertex wobble displacement (0 = off) */
  wobbleIntensity: number
  /** Speed multiplier for wobble animation */
  wobbleSpeed: number
}

export interface SlimeShaderUserData {
  slimeShader?: {
    config: SlimeShaderConfig
    shader?: THREE.Shader
  }
}

interface SlimeShaderUniforms {
  uTime: THREE.IUniform<number>
  uWobbleIntensity: THREE.IUniform<number>
  uSquashY: THREE.IUniform<number>
}

const SHADER_KEY = 'slime-wobble-v1'

const getUserData = (material: THREE.Material): SlimeShaderUserData =>
  material.userData as SlimeShaderUserData

/**
 * Binds the slime shader to a MeshPhysicalMaterial.
 *
 * Sets up onBeforeCompile to inject custom GLSL when Three.js
 * compiles the material's shader program.
 *
 * @param material - The material to bind the shader to
 * @param config - Shader configuration (wobble intensity, speed)
 */
export const bindSlimeShader = (material: THREE.MeshPhysicalMaterial, config: SlimeShaderConfig) => {
  const userData = getUserData(material)
  userData.slimeShader = { config }
  material.customProgramCacheKey = () => SHADER_KEY

  material.onBeforeCompile = (shader) => {
    const compileUserData = getUserData(material)

    // Set uniforms (safe to call multiple times)
    shader.uniforms.uTime = { value: 0 }
    shader.uniforms.uWobbleIntensity = { value: config.wobbleIntensity }
    shader.uniforms.uSquashY = { value: 1.0 }

    // Only transform shader if not already transformed (prevents double-injection)
    if (!shader.vertexShader.includes(SLIME_SHADER_MARKER)) {
      shader.vertexShader = shader.vertexShader.replace(
        '#include <common>',
        `#include <common>${VERTEX_COMMON_INJECTION}`
      )

      shader.vertexShader = shader.vertexShader.replace(
        '#include <begin_vertex>',
        `#include <begin_vertex>${VERTEX_DISPLACEMENT_INJECTION}`
      )
    }

    if (!compileUserData.slimeShader) {
      compileUserData.slimeShader = { config, shader }
      return
    }

    compileUserData.slimeShader.shader = shader
  }

  material.needsUpdate = true
}

/**
 * Syncs shader uniforms with runtime values.
 *
 * Call this each frame to update time and squash/stretch.
 * Only works after the shader has been compiled at least once.
 *
 * @param material - The material with bound slime shader
 * @param time - Current elapsed time
 * @param squashY - Y-axis scale factor (1.0 = normal, <1 = squash, >1 = stretch)
 * @param wobbleIntensity - Override wobble intensity (for LOD)
 */
export const syncSlimeShader = (
  material: THREE.MeshPhysicalMaterial,
  time: number,
  squashY = 1.0,
  wobbleIntensity?: number,
) => {
  const userData = getUserData(material)
  const data = userData.slimeShader
  if (!data?.shader) return

  const uniforms = data.shader.uniforms as unknown as SlimeShaderUniforms
  uniforms.uTime.value = time
  uniforms.uSquashY.value = squashY
  if (wobbleIntensity !== undefined) {
    uniforms.uWobbleIntensity.value = wobbleIntensity
  }
}
