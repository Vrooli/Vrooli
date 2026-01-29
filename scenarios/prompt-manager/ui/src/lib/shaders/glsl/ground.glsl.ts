/**
 * Ground Shader GLSL Injection Strings
 *
 * SINGLE SOURCE OF TRUTH: Both production (groundShader.ts) and
 * tests import from this file. Never duplicate these strings elsewhere.
 *
 * These strings are injected into Three.js MeshStandardMaterial shaders
 * via onBeforeCompile to add:
 * - World position/normal varyings for triplanar projection
 * - Custom texture sampling with stochastic tiling
 * - Macro variation overlay for reduced repetition
 */

/**
 * Marker comment to prevent double-injection.
 * Checked before applying transforms in onBeforeCompile.
 */
export const GROUND_SHADER_MARKER = '/* GROUND_SHADER_INJECTED */'

// =============================================================================
// VERTEX SHADER INJECTIONS
// =============================================================================

/**
 * Vertex shader injection - adds world position and normal varyings.
 * Injected after #include <common>.
 */
export const VERTEX_COMMON_INJECTION = `
${GROUND_SHADER_MARKER}
varying vec3 vGroundWorldPosition;
varying vec3 vGroundWorldNormal;`

/**
 * Vertex shader injection - calculates world position and normal.
 * Injected after #include <worldpos_vertex>.
 */
export const VERTEX_WORLDPOS_INJECTION = `
vec4 groundWorldPosition = modelMatrix * vec4( transformed, 1.0 );
vGroundWorldPosition = groundWorldPosition.xyz;
vGroundWorldNormal = normalize( normalMatrix * normal );`

// =============================================================================
// FRAGMENT SHADER INJECTIONS
// =============================================================================

/**
 * Fragment shader injection - uniforms and GLSL sampling functions.
 * Injected after #include <common>.
 *
 * Uniforms:
 * - uBaseUvRepeat: Texture repeat factor for UV projection
 * - uBaseWorldScale: Scale factor for triplanar world-space projection
 * - uMacroUvRepeat/uMacroWorldScale: Same for macro variation texture
 * - uMacroIntensity: Strength of macro variation effect (0-1)
 * - uUseTriplanar: 1.0 for triplanar, 0.0 for UV projection
 * - uRotation: Texture rotation in radians
 * - uTriplanarSharpness: Blend sharpness for triplanar (higher = sharper)
 * - uStochasticEnabled: 1.0 to enable stochastic tiling
 * - uMacroMap: Macro variation texture sampler
 */
export const FRAGMENT_COMMON_INJECTION = `
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

/**
 * Fragment shader map_fragment replacement - custom texture sampling.
 * Uses vMapUv (Three.js r150+ naming convention).
 * Replaces #include <map_fragment>.
 */
export const FRAGMENT_MAP_INJECTION = `
  // Three.js r150+ uses vMapUv for map textures
  vec4 groundColor = sampleProjected(map, vMapUv, vGroundWorldPosition, normalize(vGroundWorldNormal), uBaseUvRepeat, uBaseWorldScale, uRotation, uUseTriplanar, uTriplanarSharpness);
  diffuseColor *= groundColor;

  vec4 macroSample = sampleProjected(uMacroMap, vMapUv, vGroundWorldPosition, normalize(vGroundWorldNormal), uMacroUvRepeat, uMacroWorldScale, uRotation, uUseTriplanar, uTriplanarSharpness);
  float macroFactor = 1.0 + (macroSample.r - 0.5) * 2.0 * uMacroIntensity;
  diffuseColor.rgb *= macroFactor;`

// =============================================================================
// BUNDLED EXPORTS
// =============================================================================

/**
 * All injection strings bundled for convenience.
 * Use this when you need all injections together.
 */
export const GROUND_SHADER_INJECTIONS = {
  marker: GROUND_SHADER_MARKER,
  vertexCommon: VERTEX_COMMON_INJECTION,
  vertexWorldpos: VERTEX_WORLDPOS_INJECTION,
  fragmentCommon: FRAGMENT_COMMON_INJECTION,
  fragmentMap: FRAGMENT_MAP_INJECTION,
} as const

/**
 * Uniform names for type-safe iteration and validation.
 */
export const GROUND_SHADER_UNIFORMS = [
  'uBaseUvRepeat',
  'uBaseWorldScale',
  'uMacroUvRepeat',
  'uMacroWorldScale',
  'uMacroIntensity',
  'uUseTriplanar',
  'uRotation',
  'uTriplanarSharpness',
  'uStochasticEnabled',
  'uMacroMap',
] as const

export type GroundShaderUniform = (typeof GROUND_SHADER_UNIFORMS)[number]

/**
 * GLSL function names defined in FRAGMENT_COMMON_INJECTION.
 */
export const GROUND_SHADER_FUNCTIONS = [
  'rotateUv',
  'hash22',
  'sampleStochastic',
  'sampleTriplanar',
  'sampleProjected',
] as const
