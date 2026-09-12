import type { RootState } from '@react-three/fiber'

export interface WorldCapture {
  /** Freeze only after live performance measurement; return the rendered canvas. */
  snapshot: (seconds: number, frames: number) => string
}

declare global {
  interface Window { __worldCapture?: WorldCapture }
}

/** Local frame callbacks and the real post chain still run; global timing hooks do not. */
export function snapshotWorld(state: Pick<RootState, 'setFrameloop' | 'clock' | 'advance' | 'gl'>, seconds: number, frames: number): string {
  if (!Number.isFinite(seconds) || seconds < 0 || !Number.isInteger(frames) || frames < 1) throw new Error('Invalid snapshot time or frame count')
  state.setFrameloop('never')
  state.clock.elapsedTime = seconds
  state.gl.shadowMap.needsUpdate = true
  for (let frame = 0; frame < frames; frame += 1) state.advance(seconds, false)
  return state.gl.domElement.toDataURL('image/png')
}
