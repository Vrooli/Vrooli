/**
 * Sky Gradient Shader GLSL
 *
 * SINGLE SOURCE OF TRUTH: Used by DynamicSky.tsx for procedural sky rendering.
 *
 * Creates a smooth gradient sky dome that transitions between:
 * - topColor: zenith (directly above)
 * - middleColor: horizon
 * - bottomColor: nadir (below horizon, visible at edges)
 *
 * Uses smoothstep with exponential falloff for natural-looking transitions.
 */

/**
 * Sky vertex shader - passes world position to fragment shader.
 *
 * Note: Three.js ShaderMaterial automatically injects common uniforms/attributes,
 * but for standalone testing we need the full shader. This is the minimal version
 * that Three.js expects - it injects the rest.
 */
export const SKY_VERTEX_SHADER = `
varying vec3 vWorldPosition;

void main() {
  vec4 worldPosition = modelMatrix * vec4(position, 1.0);
  vWorldPosition = worldPosition.xyz;
  gl_Position = projectionMatrix * modelViewMatrix * vec4(position, 1.0);
}
`

/**
 * Sky vertex shader with Three.js built-ins for standalone WebGL compilation testing.
 */
export const SKY_VERTEX_SHADER_STANDALONE = `precision highp float;

attribute vec3 position;
uniform mat4 modelMatrix;
uniform mat4 modelViewMatrix;
uniform mat4 projectionMatrix;

varying vec3 vWorldPosition;

void main() {
  vec4 worldPosition = modelMatrix * vec4(position, 1.0);
  vWorldPosition = worldPosition.xyz;
  gl_Position = projectionMatrix * modelViewMatrix * vec4(position, 1.0);
}
`

/**
 * Sky fragment shader - creates gradient based on view angle.
 *
 * Uniforms:
 * - topColor: Color at zenith (top of sky dome)
 * - middleColor: Color at horizon
 * - bottomColor: Color below horizon
 * - offset: Vertical offset for gradient center (default 0.5)
 * - exponent: Controls gradient falloff sharpness (default 0.6)
 *
 * Note: Three.js ShaderMaterial automatically injects precision,
 * but for standalone testing we need the full shader.
 */
export const SKY_FRAGMENT_SHADER = `
uniform vec3 topColor;
uniform vec3 middleColor;
uniform vec3 bottomColor;
uniform float offset;
uniform float exponent;
varying vec3 vWorldPosition;

void main() {
  float h = normalize(vWorldPosition).y;

  // Smoothstep for gradient transitions
  float topFactor = smoothstep(0.0, 1.0, pow(max(h - offset, 0.0) / (1.0 - offset), exponent));
  float bottomFactor = smoothstep(0.0, 1.0, pow(max(-h + offset, 0.0) / (1.0 + offset), exponent));

  vec3 color = mix(middleColor, topColor, topFactor);
  color = mix(color, bottomColor, bottomFactor);

  gl_FragColor = vec4(color, 1.0);
}
`

/**
 * Sky fragment shader with precision qualifier for standalone WebGL compilation testing.
 */
export const SKY_FRAGMENT_SHADER_STANDALONE = `precision highp float;

uniform vec3 topColor;
uniform vec3 middleColor;
uniform vec3 bottomColor;
uniform float offset;
uniform float exponent;
varying vec3 vWorldPosition;

void main() {
  float h = normalize(vWorldPosition).y;

  // Smoothstep for gradient transitions
  float topFactor = smoothstep(0.0, 1.0, pow(max(h - offset, 0.0) / (1.0 - offset), exponent));
  float bottomFactor = smoothstep(0.0, 1.0, pow(max(-h + offset, 0.0) / (1.0 + offset), exponent));

  vec3 color = mix(middleColor, topColor, topFactor);
  color = mix(color, bottomColor, bottomFactor);

  gl_FragColor = vec4(color, 1.0);
}
`

/**
 * Uniform names for type-safe iteration and validation.
 */
export const SKY_SHADER_UNIFORMS = [
  'topColor',
  'middleColor',
  'bottomColor',
  'offset',
  'exponent',
] as const

export type SkyShaderUniform = (typeof SKY_SHADER_UNIFORMS)[number]
