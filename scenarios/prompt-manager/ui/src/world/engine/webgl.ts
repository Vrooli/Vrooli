export type WebGLProbeFailureReason = 'no-context' | 'context-lost' | 'threw'

export type WebGLProbeResult =
  | { ok: true }
  | { ok: false; reason: WebGLProbeFailureReason; detail: string }

/** Probe WebGL2 without retaining one of the browser's limited live contexts. */
export function probeWebGL(forceFailure = false): WebGLProbeResult {
  if (forceFailure) {
    return { ok: false, reason: 'no-context', detail: 'WebGL2 failure forced by the forceWebglFail test lever.' }
  }

  const canvas = document.createElement('canvas')
  canvas.width = 1
  canvas.height = 1

  let context: WebGL2RenderingContext | null = null
  try {
    context = canvas.getContext('webgl2')
    if (!context) return { ok: false, reason: 'no-context', detail: 'The browser did not provide a WebGL2 context.' }
    if (context.isContextLost()) return { ok: false, reason: 'context-lost', detail: 'The WebGL2 context was already lost.' }
    return { ok: true }
  } catch (error) {
    return {
      ok: false,
      reason: 'threw',
      detail: error instanceof Error ? error.message : String(error),
    }
  } finally {
    context?.getExtension('WEBGL_lose_context')?.loseContext()
  }
}

export function retryWebGL(forceFailure = false): WebGLProbeResult {
  return probeWebGL(forceFailure)
}

interface ViewChoice {
  webglAvailable: boolean
  userChoice: boolean | null
  requestedTwoD: boolean | null
  storedTwoD: boolean
  narrow: boolean
}

/** Resolve the view without allowing a stored preference to defeat an explicit deep link. */
export function resolveTwoD({ webglAvailable, userChoice, requestedTwoD, storedTwoD, narrow }: ViewChoice): boolean {
  if (!webglAvailable) return true
  return userChoice ?? requestedTwoD ?? (storedTwoD || narrow)
}
