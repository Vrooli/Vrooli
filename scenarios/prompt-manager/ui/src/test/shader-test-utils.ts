/**
 * Shader Testing Utilities
 *
 * Provides reusable utilities for GLSL shader validation:
 * - WebGL compilation testing (catches GPU-level errors like undefined variables)
 * - GLSL parsing (catches syntax errors)
 * - Shader extraction helpers for Three.js materials
 *
 * @example
 * import { compileShaderWithWebGL, parseGLSL, applyShaderInjections } from '@/test/shader-test-utils'
 *
 * const result = compileShaderWithWebGL(myShaderSource, 'fragment')
 * expect(result.success, result.error).toBe(true)
 */

import { parser } from '@shaderfrog/glsl-parser'
import createGL from 'gl'

// =============================================================================
// TYPES
// =============================================================================

export interface WebGLCompilationResult {
  success: boolean
  error: string
}

export interface GLSLParseResult {
  success: boolean
  errors: string[]
  ast?: unknown
}

export interface ShaderInjections {
  vertexCommon?: string
  vertexWorldpos?: string
  fragmentCommon?: string
  fragmentMap?: string
}

// =============================================================================
// WEBGL AVAILABILITY
// =============================================================================

/** Whether a WebGL context can be created (false in headless environments without GPU) */
export const hasWebGL = (() => {
  try {
    const ctx = createGL(1, 1) as unknown
    return ctx !== null
  } catch {
    return false
  }
})()

// =============================================================================
// WEBGL COMPILATION
// =============================================================================

/**
 * Compiles a shader using a real WebGL context (headless-gl).
 *
 * This catches actual GLSL compilation errors that a parser would miss:
 * - Undefined variables (e.g., using vUv when only vMapUv is defined)
 * - Type mismatches
 * - Invalid operations
 * - Semantic errors
 *
 * @param source - Complete GLSL shader source code
 * @param type - 'vertex' or 'fragment'
 * @returns Result with success boolean and error message if failed
 */
export function compileShaderWithWebGL(
  source: string,
  type: 'vertex' | 'fragment'
): WebGLCompilationResult {
  const gl = createGL(1, 1) as ReturnType<typeof createGL> | null // 1x1 pixel context is enough for compilation
  if (!gl) {
    return { success: false, error: 'WebGL context unavailable (headless environment without GPU)' }
  }

  const shaderType = type === 'vertex' ? gl.VERTEX_SHADER : gl.FRAGMENT_SHADER
  const shader = gl.createShader(shaderType)
  if (!shader) {
    return { success: false, error: 'Failed to create shader' }
  }

  gl.shaderSource(shader, source)
  gl.compileShader(shader)

  const success = gl.getShaderParameter(shader, gl.COMPILE_STATUS) as boolean
  const error = success ? '' : (gl.getShaderInfoLog(shader) || 'Unknown compilation error')

  gl.deleteShader(shader)

  return { success, error }
}

// =============================================================================
// GLSL PARSING
// =============================================================================

/**
 * Parses GLSL code and returns syntax errors.
 *
 * Uses @shaderfrog/glsl-parser which supports GLSL ES 1.0 and 3.0.
 * This catches syntax-level errors but not semantic errors (use WebGL for those).
 *
 * @param source - GLSL source code (should have #version removed first)
 * @returns Result with success boolean, error messages, and AST if successful
 */
export function parseGLSL(source: string): GLSLParseResult {
  const originalError = console.error
  const originalWarn = console.warn
  const ignoreKnownDiagnostic = (args: unknown[]) => {
    const [message] = args
    return (
      typeof message === 'string' &&
      (
        message.includes('Encountered undefined variable: "gl_Position"') ||
        message.includes('Encountered undefined variable: "gl_FragColor"') ||
        message.includes('Encountered undefined variable: "modelMatrix"') ||
        message.includes('Encountered undefined variable: "modelViewMatrix"') ||
        message.includes('Encountered undefined variable: "position"') ||
        message.includes('Encountered undefined variable: "projectionMatrix"')
      )
    )
  }
  console.error = (...args: unknown[]) => {
    if (ignoreKnownDiagnostic(args)) {
      return
    }
    originalError(...args)
  }
  console.warn = (...args: unknown[]) => {
    if (ignoreKnownDiagnostic(args)) {
      return
    }
    originalWarn(...args)
  }

  try {
    const ast = parser.parse(source)
    return { success: true, errors: [], ast }
  } catch (error: unknown) {
    const errorMessage = error instanceof Error ? error.message : String(error)
    const errors = [formatGLSLError(source, errorMessage)]
    return { success: false, errors }
  } finally {
    console.error = originalError
    console.warn = originalWarn
  }
}

/**
 * Prepares shader source for parsing by removing #version directives.
 *
 * The @shaderfrog/glsl-parser handles GLSL ES 1.0/3.0 but sometimes
 * struggles with version directives in certain contexts.
 */
export function prepareForParsing(source: string): string {
  return source.replace(/^\s*#version\s+\d+\s*(es)?\s*\n/gmi, '')
}

/**
 * Formats a GLSL error with source context.
 * Shows the problematic line with surrounding context.
 */
function formatGLSLError(source: string, errorMessage: string): string {
  const lineMatch = errorMessage.match(/line (\d+)/i)

  if (!lineMatch?.[1]) {
    return errorMessage
  }

  const lineNum = parseInt(lineMatch[1], 10)
  const lines = source.split('\n')
  const startLine = Math.max(0, lineNum - 3)
  const endLine = Math.min(lines.length, lineNum + 2)

  let formatted = errorMessage + '\n\nContext:\n'
  for (let i = startLine; i < endLine; i++) {
    const prefix = i === lineNum - 1 ? '>>> ' : '    '
    formatted += `${prefix}${i + 1}: ${lines[i]}\n`
  }

  return formatted
}

// =============================================================================
// THREE.JS SHADER TEMPLATES
// =============================================================================

/**
 * Minimal Three.js MeshStandardMaterial vertex shader structure.
 * Uses WebGL 1 / GLSL ES 1.0 syntax (attribute/varying, no version directive).
 *
 * Used for testing shader injections without running full Three.js.
 */
export const BASE_VERTEX_SHADER = `precision highp float;

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
 * Minimal Three.js MeshStandardMaterial fragment shader structure.
 * Uses WebGL 1 / GLSL ES 1.0 syntax.
 * Three.js r150+ uses vMapUv for map textures.
 */
export const BASE_FRAGMENT_SHADER = `precision highp float;

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
 * Applies shader injections to base shaders.
 *
 * Simulates what Three.js does at runtime with onBeforeCompile.
 * Use this to extract complete, compilable GLSL for testing.
 *
 * @example
 * import { GROUND_SHADER_INJECTIONS } from '@/lib/shaders/glsl/ground.glsl'
 *
 * const { vertex, fragment } = applyShaderInjections({
 *   vertexCommon: GROUND_SHADER_INJECTIONS.vertexCommon,
 *   vertexWorldpos: GROUND_SHADER_INJECTIONS.vertexWorldpos,
 *   fragmentCommon: GROUND_SHADER_INJECTIONS.fragmentCommon,
 *   fragmentMap: GROUND_SHADER_INJECTIONS.fragmentMap,
 * })
 */
export function applyShaderInjections(injections: ShaderInjections): {
  vertex: string
  fragment: string
} {
  let vertex = BASE_VERTEX_SHADER
  let fragment = BASE_FRAGMENT_SHADER

  if (injections.vertexCommon) {
    vertex = vertex.replace('#include <common>', injections.vertexCommon)
  }
  if (injections.vertexWorldpos) {
    vertex = vertex.replace('#include <worldpos_vertex>', injections.vertexWorldpos)
  }
  if (injections.fragmentCommon) {
    fragment = fragment.replace('#include <common>', injections.fragmentCommon)
  }
  if (injections.fragmentMap) {
    fragment = fragment.replace('#include <map_fragment>', injections.fragmentMap)
  }

  return { vertex, fragment }
}

// =============================================================================
// VALIDATION HELPERS
// =============================================================================

/**
 * Validates that a shader source contains expected uniform declarations.
 *
 * @param source - GLSL source code
 * @param uniformNames - Array of uniform names to check for
 * @returns Array of missing uniform names (empty if all present)
 */
export function findMissingUniforms(source: string, uniformNames: readonly string[]): string[] {
  return uniformNames.filter(name => !source.includes(name))
}

/**
 * Validates that a shader source contains expected function definitions.
 *
 * @param source - GLSL source code
 * @param functionNames - Array of function names to check for
 * @returns Array of missing function names (empty if all present)
 */
export function findMissingFunctions(source: string, functionNames: readonly string[]): string[] {
  return functionNames.filter(name => !source.includes(name))
}

/**
 * Checks if shader has proper semicolon termination.
 * Returns lines that appear to be missing semicolons.
 */
export function findPotentialMissingSemicolons(source: string): string[] {
  const suspicious: string[] = []
  const lines = source.split('\n')

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i]?.trim() ?? ''

    // Skip empty lines, comments, function signatures, braces
    if (
      !line ||
      line.startsWith('//') ||
      line.startsWith('/*') ||
      line.includes('{') ||
      line === '}' ||
      line.endsWith(',')
    ) {
      continue
    }

    // If line has assignment but doesn't end properly, flag it
    if (
      line.includes('=') &&
      !line.includes('==') &&
      !line.endsWith(';') &&
      !line.endsWith('{') &&
      !line.endsWith(',')
    ) {
      const nextLine = lines[i + 1]?.trim() || ''
      if (
        !nextLine.startsWith('.') &&
        !nextLine.startsWith('+') &&
        !nextLine.startsWith('-') &&
        !nextLine.startsWith('*')
      ) {
        suspicious.push(`Line ${i + 1}: ${line}`)
      }
    }
  }

  return suspicious
}
