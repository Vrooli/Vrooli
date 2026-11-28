import { WorldClock } from '../../config/clock'

/** One scene-owned timer, armed at the next event boundary. A resumed tab seeks
 * current state rather than replaying timer callbacks for missed events.
 */
export function bindAmbientWake(nextBoundary: (now: number) => number | null, invalidate: () => void, clock = new WorldClock()): () => void {
  let timer: number | undefined
  let disposed = false
  const clear = () => { if (timer !== undefined) window.clearTimeout(timer); timer = undefined }
  const arm = () => {
    clear()
    if (disposed || document.hidden) return
    const snapshot = clock.snapshot()
    if (snapshot.timeScale === 0) return
    const nowMs = snapshot.utcMilliseconds
    const now = nowMs / 1000
    const boundary = nextBoundary(now)
    if (boundary === null) return
    if (!Number.isFinite(boundary) || boundary <= now) throw new Error('Ambient boundary must be in the future')
    timer = window.setTimeout(() => {
      timer = undefined
      // Long waits may hit the browser's signed-32-bit timer limit. Re-arm
      // without rendering until the actual boundary, including backwards clock jumps.
      if (clock.snapshot().utcMilliseconds / 1000 >= boundary) invalidate()
      arm()
    }, Math.min(2147483647, Math.max(1, Math.ceil((boundary * 1000 - nowMs) / snapshot.timeScale))))
  }
  const resume = () => {
    clock.refreshTimeZone()
    if (!document.hidden) invalidate()
    arm()
  }
  document.addEventListener('visibilitychange', resume)
  window.addEventListener('focus', resume)
  const offClock = clock.subscribe(resume)
  arm()
  return () => {
    disposed = true
    offClock()
    clear()
    document.removeEventListener('visibilitychange', resume)
    window.removeEventListener('focus', resume)
  }
}
