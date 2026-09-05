/**
 * The instanced slime material: MeshPhysicalMaterial with clearcoat and sheen
 * for the jelly look, plus the ported wobble/squash vertex injection driven
 * by per-instance attributes (aColor, aSeed, aTimeShift, aSquash).
 */
import { Color, MeshPhysicalMaterial, type WebGLProgramParametersWithUniforms } from 'three'
import type { ActorTuning } from '../../config'
import {
  FRAGMENT_COLOR_INJECTION,
  FRAGMENT_COMMON_INJECTION,
  SLIME_SHADER_MARKER,
  VERTEX_COMMON_INJECTION,
  VERTEX_DISPLACEMENT_INJECTION,
} from './slime.glsl'

export interface SlimeUniforms {
  uTime: { value: number }
  uWobbleIntensity: { value: number }
  uWobbleScale: { value: number }
  uWobbleSpeed: { value: number }
}

export interface SlimeMaterial extends MeshPhysicalMaterial {
  slime: SlimeUniforms
}

/** Build the material once per world; call `setSlimeWobble` when the profile changes. */
export function createSlimeMaterial(actor: ActorTuning, wobbleEnabled: boolean): SlimeMaterial {
  const material = new MeshPhysicalMaterial({
    color: new Color(actor.material.color),
    roughness: actor.material.roughness,
    metalness: 0,
    clearcoat: actor.material.clearcoat,
    clearcoatRoughness: actor.material.clearcoatRoughness,
    sheen: actor.material.sheen,
    sheenColor: new Color(actor.material.sheenColor),
  }) as SlimeMaterial
  material.slime = {
    uTime: { value: 0 },
    uWobbleIntensity: { value: wobbleEnabled ? actor.wobbleIntensity : 0 },
    uWobbleScale: { value: actor.material.wobbleScale },
    uWobbleSpeed: { value: actor.material.wobbleSpeed },
  }
  material.onBeforeCompile = (shader: WebGLProgramParametersWithUniforms) => {
    if (shader.vertexShader.includes(SLIME_SHADER_MARKER)) return
    Object.assign(shader.uniforms, material.slime)
    shader.vertexShader = shader.vertexShader
      .replace('#include <common>', `#include <common>${VERTEX_COMMON_INJECTION}`)
      .replace('#include <begin_vertex>', `#include <begin_vertex>${VERTEX_DISPLACEMENT_INJECTION}`)
    shader.fragmentShader = shader.fragmentShader
      .replace('#include <common>', `#include <common>${FRAGMENT_COMMON_INJECTION}`)
      .replace('#include <color_fragment>', `#include <color_fragment>${FRAGMENT_COLOR_INJECTION}`)
  }
  material.customProgramCacheKey = () => 'world-slime'
  return material
}

export function setSlimeWobble(material: SlimeMaterial, actor: ActorTuning, enabled: boolean): void {
  material.slime.uWobbleIntensity.value = enabled ? actor.wobbleIntensity : 0
}
