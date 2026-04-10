/**
 * Ground Shader Runtime Binding
 *
 * Binds custom ground texture shader to Three.js MeshStandardMaterial.
 * Uses onBeforeCompile to inject GLSL for:
 * - Triplanar projection (eliminates UV stretching on slopes)
 * - Stochastic tiling (eliminates visible repetition)
 * - Macro variation overlay (adds large-scale detail)
 *
 * GLSL source is imported from ./glsl/ground.glsl.ts (single source of truth).
 */

import * as THREE from 'three'
import type { GroundProjection } from '@/types/environment'
import {
  GROUND_SHADER_MARKER,
  VERTEX_COMMON_INJECTION,
  VERTEX_WORLDPOS_INJECTION,
  FRAGMENT_COMMON_INJECTION,
  FRAGMENT_MAP_INJECTION,
} from './glsl/ground.glsl'

export interface GroundShaderConfig {
  projection: GroundProjection
  rotation: number
  baseUvRepeat: number
  baseWorldScale: number
  macroUvRepeat: number
  macroWorldScale: number
  macroIntensity: number
  macroMap: THREE.Texture
  triplanarSharpness: number
  /** Enable stochastic tiling to eliminate visible repetition */
  stochasticEnabled: boolean
}

export interface GroundShaderUserData {
  groundShader?: {
    config: GroundShaderConfig
    shader?: THREE.Shader
  }
}

interface GroundShaderUniforms {
  uBaseUvRepeat: THREE.IUniform<number>
  uBaseWorldScale: THREE.IUniform<number>
  uMacroUvRepeat: THREE.IUniform<number>
  uMacroWorldScale: THREE.IUniform<number>
  uMacroIntensity: THREE.IUniform<number>
  uMacroMap: THREE.IUniform<THREE.Texture>
  uUseTriplanar: THREE.IUniform<number>
  uRotation: THREE.IUniform<number>
  uTriplanarSharpness: THREE.IUniform<number>
  uStochasticEnabled: THREE.IUniform<number>
}

const SHADER_KEY = 'ground-texture-v2'

const getUserData = (material: THREE.Material): GroundShaderUserData =>
  material.userData as GroundShaderUserData

/**
 * Binds the ground shader to a MeshStandardMaterial.
 *
 * This sets up onBeforeCompile to inject custom GLSL when Three.js
 * compiles the material's shader program.
 *
 * @param material - The material to bind the shader to
 * @param config - Shader configuration (projection mode, texture settings, etc.)
 */
export const bindGroundShader = (material: THREE.MeshStandardMaterial, config: GroundShaderConfig) => {
  const userData = getUserData(material)
  userData.groundShader = { config }
  material.customProgramCacheKey = () => SHADER_KEY

  material.onBeforeCompile = (shader) => {
    const compileUserData = getUserData(material)

    // Set uniforms (safe to call multiple times)
    shader.uniforms.uBaseUvRepeat = { value: config.baseUvRepeat }
    shader.uniforms.uBaseWorldScale = { value: config.baseWorldScale }
    shader.uniforms.uMacroUvRepeat = { value: config.macroUvRepeat }
    shader.uniforms.uMacroWorldScale = { value: config.macroWorldScale }
    shader.uniforms.uMacroIntensity = { value: config.macroIntensity }
    shader.uniforms.uMacroMap = { value: config.macroMap }
    shader.uniforms.uUseTriplanar = { value: config.projection === 'triplanar' ? 1 : 0 }
    shader.uniforms.uRotation = { value: config.rotation }
    shader.uniforms.uTriplanarSharpness = { value: config.triplanarSharpness }
    shader.uniforms.uStochasticEnabled = { value: config.stochasticEnabled ? 1 : 0 }

    // Only transform shader if not already transformed (prevents double-injection)
    // which can happen if onBeforeCompile is called multiple times
    if (!shader.vertexShader.includes(GROUND_SHADER_MARKER)) {
      shader.vertexShader = shader.vertexShader.replace(
        '#include <common>',
        `#include <common>${VERTEX_COMMON_INJECTION}`
      )

      shader.vertexShader = shader.vertexShader.replace(
        '#include <worldpos_vertex>',
        `#include <worldpos_vertex>${VERTEX_WORLDPOS_INJECTION}`
      )
    }

    if (!shader.fragmentShader.includes(GROUND_SHADER_MARKER)) {
      shader.fragmentShader = shader.fragmentShader.replace(
        '#include <common>',
        `#include <common>${FRAGMENT_COMMON_INJECTION}`
      )

      shader.fragmentShader = shader.fragmentShader.replace(
        '#include <map_fragment>',
        `#ifdef USE_MAP${FRAGMENT_MAP_INJECTION}
#endif`
      )
    }

    if (!compileUserData.groundShader) {
      compileUserData.groundShader = { config, shader }
      return
    }

    compileUserData.groundShader.shader = shader
  }

  material.needsUpdate = true
}

/**
 * Syncs shader uniforms with a new configuration.
 *
 * Use this to update shader parameters without re-binding.
 * Only works after the shader has been compiled at least once.
 *
 * @param material - The material with bound ground shader
 * @param config - New shader configuration
 */
export const syncGroundShader = (material: THREE.MeshStandardMaterial, config: GroundShaderConfig) => {
  const userData = getUserData(material)
  const data = userData.groundShader
  if (!data?.shader) {
    userData.groundShader = { config }
    return
  }

  data.config = config
  const uniforms = data.shader.uniforms as unknown as GroundShaderUniforms
  uniforms.uBaseUvRepeat.value = config.baseUvRepeat
  uniforms.uBaseWorldScale.value = config.baseWorldScale
  uniforms.uMacroUvRepeat.value = config.macroUvRepeat
  uniforms.uMacroWorldScale.value = config.macroWorldScale
  uniforms.uMacroIntensity.value = config.macroIntensity
  uniforms.uMacroMap.value = config.macroMap
  uniforms.uUseTriplanar.value = config.projection === 'triplanar' ? 1 : 0
  uniforms.uRotation.value = config.rotation
  uniforms.uTriplanarSharpness.value = config.triplanarSharpness
  uniforms.uStochasticEnabled.value = config.stochasticEnabled ? 1 : 0
}
