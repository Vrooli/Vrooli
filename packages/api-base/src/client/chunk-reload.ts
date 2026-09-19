/**
 * Stale-chunk recovery for code-split (React.lazy / dynamic import) UIs.
 *
 * Vite builds emit content-hashed chunk filenames. When a scenario is
 * rebuilt/restarted, every hash changes and the old chunks are deleted —
 * but any browser tab that loaded the previous index.html still points at
 * the old filenames. The next lazy navigation then fails with a dynamic
 * import error and the app crashes into its error boundary, even though
 * nothing is actually wrong: a new version was simply deployed.
 *
 * The fix is client-side: reload once to pick up the fresh index.html.
 * A cooldown marker prevents reload loops when a chunk is missing for a
 * real reason (broken deploy, offline).
 *
 * Usage — one call at the app entry point, before React mounts:
 * ```ts
 * import { installChunkReloadGuard } from '@vrooli/api-base'
 * installChunkReloadGuard()
 * ```
 * Error boundaries can additionally use `isStaleChunkError` to classify
 * failures that reach them through paths Vite's event does not cover.
 */

const RELOAD_MARKER_KEY = 'vrooli:chunk-reload-at'
const RELOAD_COOLDOWN_MS = 60_000

/** Vite dispatches `vite:preloadError` with the failure on `payload`. */
interface VitePreloadErrorEvent extends Event {
  payload?: unknown
}

/**
 * True when an error is a stale-chunk failure: the running page was built
 * before the last deploy and is importing a hashed chunk that no longer
 * exists. Covers the Chrome, Firefox, and Safari message shapes.
 */
export function isStaleChunkError(error: unknown): boolean {
  const message =
    error instanceof Error ? error.message : typeof error === 'string' ? error : ''
  return (
    message.includes('Failed to fetch dynamically imported module') || // Chromium
    message.includes('error loading dynamically imported module') || // Firefox
    message.includes('Importing a module script failed') // Safari
  )
}

function readReloadMarker(win: Window): number {
  try {
    return Number(win.sessionStorage.getItem(RELOAD_MARKER_KEY) ?? 0)
  } catch {
    // Storage can be unavailable (sandboxed iframe, privacy mode). Treat as
    // "never reloaded" — worst case is one extra reload attempt.
    return 0
  }
}

function writeReloadMarker(win: Window, now: number): void {
  try {
    win.sessionStorage.setItem(RELOAD_MARKER_KEY, String(now))
  } catch {
    // Ignore: without storage we cannot rate-limit, but we also do not loop
    // because the caller only invokes this on an actual chunk failure.
  }
}

/**
 * Reload the page to pick up the new deploy — at most once per cooldown
 * window, so a genuinely broken bundle cannot cause a reload loop.
 *
 * @returns true when a reload was triggered; false when the cooldown
 * suppressed it (the caller should fall back to its normal error UI).
 */
export function reloadForStaleChunk(win: Window = window): boolean {
  const now = Date.now()
  if (now - readReloadMarker(win) < RELOAD_COOLDOWN_MS) {
    return false
  }
  writeReloadMarker(win, now)
  win.location.reload()
  return true
}

/**
 * Install once at the app entry point, before React mounts.
 *
 * Listens for Vite's `vite:preloadError` (fired when a code-split chunk or
 * one of its preloaded dependencies fails to load) and self-heals with a
 * rate-limited reload instead of letting the failure crash the app into an
 * error boundary.
 */
export function installChunkReloadGuard(win: Window = window): void {
  win.addEventListener('vite:preloadError', (event) => {
    if (reloadForStaleChunk(win)) {
      // We are reloading; suppress Vite's re-throw so the doomed render
      // tree does not also flash an error boundary.
      ;(event as VitePreloadErrorEvent).preventDefault()
    }
  })
}
