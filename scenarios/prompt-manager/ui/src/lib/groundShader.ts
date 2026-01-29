import * as THREE from 'three'
import type { GroundProjection } from '@/types/environment'

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
    const GROUND_SHADER_MARKER = '/* GROUND_SHADER_INJECTED */'

    if (!shader.vertexShader.includes(GROUND_SHADER_MARKER)) {
      shader.vertexShader = shader.vertexShader.replace(
        '#include <common>',
        `#include <common>
${GROUND_SHADER_MARKER}
varying vec3 vGroundWorldPosition;
varying vec3 vGroundWorldNormal;`
      )

      shader.vertexShader = shader.vertexShader.replace(
        '#include <worldpos_vertex>',
        `#include <worldpos_vertex>
vec4 groundWorldPosition = modelMatrix * vec4( transformed, 1.0 );
vGroundWorldPosition = groundWorldPosition.xyz;
vGroundWorldNormal = normalize( normalMatrix * normal );`
      )
    }

    if (!shader.fragmentShader.includes(GROUND_SHADER_MARKER)) {
      shader.fragmentShader = shader.fragmentShader.replace(
        '#include <common>',
        `#include <common>
${GROUND_SHADER_MARKER}
uniform float uBaseUvRepeat;
uniform float uBaseWorldScale;
uniform float uMacroUvRepeat;
uniform float uMacroWorldScale;
uniform float uMacroIntensity;
uniform float uUseTriplanar;
uniform float uRotation;
uniform float uTriplanarSharpness;
uniform float uStochasticEnabled;
uniform sampler2D uMacroMap;
varying vec3 vGroundWorldPosition;
varying vec3 vGroundWorldNormal;

vec2 rotateUv(vec2 uv, float rotation) {
  float s = sin(rotation);
  float c = cos(rotation);
  mat2 rot = mat2(c, -s, s, c);
  return rot * uv;
}

// Hash function for pseudo-random values per tile (Dave Hoskins)
vec2 hash22(vec2 p) {
  vec3 p3 = fract(vec3(p.xyx) * vec3(0.1031, 0.1030, 0.0973));
  p3 += dot(p3, p3.yzx + 33.33);
  return fract((p3.xx + p3.yz) * p3.zy);
}

// Stochastic sampling: random 90-degree rotation and offset per tile
// This breaks visible repetition patterns while maintaining texture quality
vec4 sampleStochastic(sampler2D tex, vec2 uv, float scale, float rotation) {
  vec2 scaledUv = uv * scale;
  vec2 tileId = floor(scaledUv);
  vec2 tileUv = fract(scaledUv);
  vec2 rand = hash22(tileId);

  // Random 90-degree rotation (0, 90, 180, or 270 degrees)
  float angle = floor(rand.x * 4.0) * 1.5707963 + rotation;
  float c = cos(angle);
  float s = sin(angle);
  vec2 centered = tileUv - 0.5;
  vec2 rotated = vec2(centered.x * c - centered.y * s, centered.x * s + centered.y * c) + 0.5;

  // Random offset (prevents alignment artifacts)
  vec2 offset = rand * 0.5;

  return texture2D(tex, rotated + offset);
}

vec4 sampleTriplanar(sampler2D tex, vec3 worldPos, vec3 normal, float scale, float rotation, float sharpness) {
  vec3 blend = pow(abs(normal), vec3(sharpness));
  blend /= (blend.x + blend.y + blend.z);

  vec2 uvX = rotateUv(worldPos.zy * scale, rotation);
  vec2 uvY = rotateUv(worldPos.xz * scale, rotation);
  vec2 uvZ = rotateUv(worldPos.xy * scale, rotation);

  vec4 xColor = texture2D(tex, uvX);
  vec4 yColor = texture2D(tex, uvY);
  vec4 zColor = texture2D(tex, uvZ);

  return xColor * blend.x + yColor * blend.y + zColor * blend.z;
}

vec4 sampleProjected(sampler2D tex, vec2 uv, vec3 worldPos, vec3 normal, float uvRepeat, float worldScale, float rotation, float useTriplanar, float sharpness) {
  // Stochastic sampling path (when enabled)
  if (uStochasticEnabled > 0.5) {
    return sampleStochastic(tex, uv, uvRepeat, rotation);
  }

  // Standard sampling path
  vec2 uvSample = rotateUv(uv * uvRepeat, rotation);
  vec4 planar = texture2D(tex, uvSample);
  vec4 tri = sampleTriplanar(tex, worldPos, normal, worldScale, rotation, sharpness);
  return mix(planar, tri, useTriplanar);
}`
      )

      shader.fragmentShader = shader.fragmentShader.replace(
        '#include <map_fragment>',
        `#ifdef USE_MAP
  // Three.js r150+ uses vMapUv for map textures
  vec4 groundColor = sampleProjected(map, vMapUv, vGroundWorldPosition, normalize(vGroundWorldNormal), uBaseUvRepeat, uBaseWorldScale, uRotation, uUseTriplanar, uTriplanarSharpness);
  diffuseColor *= groundColor;

  vec4 macroSample = sampleProjected(uMacroMap, vMapUv, vGroundWorldPosition, normalize(vGroundWorldNormal), uMacroUvRepeat, uMacroWorldScale, uRotation, uUseTriplanar, uTriplanarSharpness);
  float macroFactor = 1.0 + (macroSample.r - 0.5) * 2.0 * uMacroIntensity;
  diffuseColor.rgb *= macroFactor;
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
