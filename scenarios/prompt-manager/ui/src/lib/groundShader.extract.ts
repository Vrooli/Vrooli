/**
 * Extracts GLSL code from groundShader for validation testing.
 * Simulates Three.js MeshStandardMaterial shader chunks and applies
 * the same injections that bindGroundShader performs.
 */

/**
 * Simplified Three.js MeshStandardMaterial vertex shader structure.
 * Uses WebGL 1 / GLSL ES 1.0 syntax (no version directive, attribute/varying).
 * Three.js uses this format by default.
 */
const BASE_VERTEX_SHADER = `precision highp float;

// Three.js built-in attributes
attribute vec3 position;
attribute vec3 normal;
attribute vec2 uv;

// Three.js built-in uniforms
uniform mat4 modelMatrix;
uniform mat4 modelViewMatrix;
uniform mat4 projectionMatrix;
uniform mat3 normalMatrix;

// Outputs
varying vec2 vUv;

#include <common>

void main() {
  vUv = uv;
  vec3 transformed = position;
  #include <worldpos_vertex>
  gl_Position = projectionMatrix * modelViewMatrix * vec4(transformed, 1.0);
}
`

/**
 * Simplified Three.js MeshStandardMaterial fragment shader structure.
 * Uses WebGL 1 / GLSL ES 1.0 syntax (no version directive, varying).
 * Three.js r150+ uses vMapUv for map textures.
 */
const BASE_FRAGMENT_SHADER = `precision highp float;

// Three.js built-in uniforms
uniform sampler2D map;
uniform vec3 diffuse;

// Inputs (Three.js r150+ uses vMapUv for map textures)
varying vec2 vMapUv;

#define USE_MAP

#include <common>

void main() {
  vec4 diffuseColor = vec4(diffuse, 1.0);
  #include <map_fragment>
  gl_FragColor = diffuseColor;
}
`

/**
 * Marker comment to prevent double-injection
 */
const GROUND_SHADER_MARKER = '/* GROUND_SHADER_INJECTED */'

/**
 * Vertex shader injection - adds world position and normal varyings.
 */
const VERTEX_COMMON_INJECTION = `
${GROUND_SHADER_MARKER}
varying vec3 vGroundWorldPosition;
varying vec3 vGroundWorldNormal;
`

const VERTEX_WORLDPOS_INJECTION = `
vec4 groundWorldPosition = modelMatrix * vec4( transformed, 1.0 );
vGroundWorldPosition = groundWorldPosition.xyz;
vGroundWorldNormal = normalize( normalMatrix * normal );
`

/**
 * Fragment shader injection - adds all custom uniforms and GLSL functions.
 */
const FRAGMENT_COMMON_INJECTION = `
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
}
`

/**
 * Fragment shader map_fragment replacement - custom texture sampling.
 * Uses vMapUv (Three.js r150+ naming) instead of vUv.
 */
const FRAGMENT_MAP_INJECTION = `
  // Three.js r150+ uses vMapUv for map textures
  vec4 groundColor = sampleProjected(map, vMapUv, vGroundWorldPosition, normalize(vGroundWorldNormal), uBaseUvRepeat, uBaseWorldScale, uRotation, uUseTriplanar, uTriplanarSharpness);
  diffuseColor *= groundColor;

  vec4 macroSample = sampleProjected(uMacroMap, vMapUv, vGroundWorldPosition, normalize(vGroundWorldNormal), uMacroUvRepeat, uMacroWorldScale, uRotation, uUseTriplanar, uTriplanarSharpness);
  float macroFactor = 1.0 + (macroSample.r - 0.5) * 2.0 * uMacroIntensity;
  diffuseColor.rgb *= macroFactor;
`

/**
 * Extracts the complete GLSL shaders with all injections applied.
 * This simulates what Three.js does at runtime with onBeforeCompile.
 * Produces compilable GLSL for WebGL 1 testing.
 */
export function extractGroundShaderGLSL(): { vertex: string; fragment: string } {
  // Apply vertex shader injections
  let vertex = BASE_VERTEX_SHADER
    .replace('#include <common>', VERTEX_COMMON_INJECTION)
    .replace('#include <worldpos_vertex>', VERTEX_WORLDPOS_INJECTION)

  // Apply fragment shader injections
  let fragment = BASE_FRAGMENT_SHADER
    .replace('#include <common>', FRAGMENT_COMMON_INJECTION)
    .replace('#include <map_fragment>', FRAGMENT_MAP_INJECTION)

  return { vertex, fragment }
}

/**
 * Extracts GLSL that can be compiled standalone by WebGL.
 * Removes any Three.js-specific constructs.
 */
export function extractCompilableGLSL(): { vertex: string; fragment: string } {
  const { vertex, fragment } = extractGroundShaderGLSL()

  // The extracted shaders should already be compilable since we replaced
  // the #include directives with actual code
  return { vertex, fragment }
}

/**
 * Returns the raw injection strings for inspection.
 */
export function getShaderInjections() {
  return {
    vertexCommon: VERTEX_COMMON_INJECTION,
    vertexWorldpos: VERTEX_WORLDPOS_INJECTION,
    fragmentCommon: FRAGMENT_COMMON_INJECTION,
    fragmentMap: FRAGMENT_MAP_INJECTION,
  }
}
