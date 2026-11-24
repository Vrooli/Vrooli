import { z } from 'zod'
import { WALK, type CameraMemory, type SavedCameraView } from '../config/navigation'

const coordinate = z.number().min(-1e6).max(1e6)
const position = z.tuple([coordinate, coordinate, coordinate])
const schema = z.object({
  version: z.literal(1), mode: z.enum(['explore', 'first-person', 'third-person']),
  explore: z.object({ position, target: position, zoom: z.number().min(.01).max(100) }),
  body: z.object({ position, yaw: coordinate, pitch: z.number().min(-WALK.maxPitch).max(WALK.maxPitch),
    boom: z.number().min(WALK.minBoom).max(WALK.maxBoom) }).optional(),
}).refine(view => view.mode === 'explore' || view.body !== undefined)

/** Device-local camera continuity. Scene/seed isolation prevents restoring a
 * position from a different world; imported captures use their own pose.
 */
export function createCameraMemory(scene: string, seed: number, storage: () => Pick<Storage, 'getItem' | 'setItem'> = () => window.localStorage): CameraMemory {
  const key = `prompt-manager.world.view.v1:${scene}:${seed}`
  let pending: string | null = null, last: string | null = null
  let timer: ReturnType<typeof setTimeout> | undefined
  const flush = () => {
    clearTimeout(timer); timer = undefined
    if (pending === null) return
    try { storage().setItem(key, pending); last = pending; pending = null } catch { /* Private/unavailable storage must not disable navigation. */ }
  }
  return {
    read() {
      try {
        const value = storage().getItem(key)
        const result = schema.safeParse(value === null ? null : JSON.parse(value))
        if (!result.success) return null
        last = JSON.stringify(result.data)
        return result.data as SavedCameraView
      } catch { return null }
    },
    save(view) {
      const value = JSON.stringify(view)
      if (value === last || value === pending) return
      pending = value
      // Bound writes during continuous movement, rather than postponing forever.
      timer ??= setTimeout(flush, 600)
    },
    flush,
  }
}
