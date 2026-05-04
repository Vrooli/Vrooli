export type WorldRenderReason =
  | 'scene-mount'
  | 'scene-data'
  | 'camera-state'
  | 'orbit-start'
  | 'orbit-change'
  | 'orbit-end'
  | 'pointer-move'
  | 'pointer-leave'
  | 'drag-active'
  | 'placement-active'
  | 'overlay-open'
  | 'visibility'
  | 'diagnostics'
  | 'manual'

type CustomWorldRenderReason = string & {}
type RenderReason = WorldRenderReason | CustomWorldRenderReason
type RenderRequester = (reason: RenderReason, frames?: number) => void

let activeRequester: RenderRequester | null = null

export function registerWorldRenderRequester(requester: RenderRequester): () => void {
  activeRequester = requester
  return () => {
    if (activeRequester === requester) {
      activeRequester = null
    }
  }
}

export function requestWorldRender(reason: RenderReason, frames = 1): void {
  activeRequester?.(reason, frames)
}
