/**
 * Tests for stale-chunk recovery (chunk-reload.ts)
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import {
  installChunkReloadGuard,
  isStaleChunkError,
  reloadForStaleChunk,
} from '../../client/chunk-reload.js'

interface FakeWindow {
  win: Window
  reload: ReturnType<typeof vi.fn>
  dispatchPreloadError: () => Event
  storage: Map<string, string>
}

/** Minimal window double: sessionStorage + location.reload + event target. */
function fakeWindow(overrides?: { brokenStorage?: boolean }): FakeWindow {
  const storage = new Map<string, string>()
  const listeners = new Map<string, Array<(event: Event) => void>>()
  const reload = vi.fn()

  const sessionStorage = overrides?.brokenStorage
    ? {
        getItem: () => {
          throw new Error('storage disabled')
        },
        setItem: () => {
          throw new Error('storage disabled')
        },
      }
    : {
        getItem: (key: string) => storage.get(key) ?? null,
        setItem: (key: string, value: string) => void storage.set(key, value),
      }

  const win = {
    sessionStorage,
    location: { reload },
    addEventListener: (type: string, handler: (event: Event) => void) => {
      const bucket = listeners.get(type) ?? []
      bucket.push(handler)
      listeners.set(type, bucket)
    },
  } as unknown as Window

  const dispatchPreloadError = (): Event => {
    const event = new Event('vite:preloadError', { cancelable: true })
    for (const handler of listeners.get('vite:preloadError') ?? []) {
      handler(event)
    }
    return event
  }

  return { win, reload, dispatchPreloadError, storage }
}

describe('isStaleChunkError', () => {
  it('recognizes the Chromium message', () => {
    expect(
      isStaleChunkError(
        new TypeError('Failed to fetch dynamically imported module: http://x/assets/Page-abc.js'),
      ),
    ).toBe(true)
  })

  it('recognizes the Firefox message', () => {
    expect(isStaleChunkError(new TypeError('error loading dynamically imported module'))).toBe(true)
  })

  it('recognizes the Safari message', () => {
    expect(isStaleChunkError(new TypeError('Importing a module script failed.'))).toBe(true)
  })

  it('recognizes plain-string errors', () => {
    expect(isStaleChunkError('Failed to fetch dynamically imported module')).toBe(true)
  })

  it('rejects unrelated errors and non-errors', () => {
    expect(isStaleChunkError(new Error('Cannot read properties of undefined'))).toBe(false)
    expect(isStaleChunkError(new Error('Failed to fetch'))).toBe(false)
    expect(isStaleChunkError(undefined)).toBe(false)
    expect(isStaleChunkError({ message: 'Importing a module script failed' })).toBe(false)
  })
})

describe('reloadForStaleChunk', () => {
  beforeEach(() => {
    vi.useRealTimers()
  })

  it('reloads on the first failure and stamps the cooldown marker', () => {
    const { win, reload, storage } = fakeWindow()

    expect(reloadForStaleChunk(win)).toBe(true)
    expect(reload).toHaveBeenCalledTimes(1)
    expect(storage.get('vrooli:chunk-reload-at')).toBeTruthy()
  })

  it('suppresses a second reload inside the cooldown window', () => {
    const { win, reload } = fakeWindow()

    expect(reloadForStaleChunk(win)).toBe(true)
    expect(reloadForStaleChunk(win)).toBe(false)
    expect(reload).toHaveBeenCalledTimes(1)
  })

  it('allows a reload again after the cooldown elapses', () => {
    vi.useFakeTimers()
    const { win, reload } = fakeWindow()

    expect(reloadForStaleChunk(win)).toBe(true)
    vi.advanceTimersByTime(61_000)
    expect(reloadForStaleChunk(win)).toBe(true)
    expect(reload).toHaveBeenCalledTimes(2)
    vi.useRealTimers()
  })

  it('still reloads when sessionStorage is unavailable', () => {
    const { win, reload } = fakeWindow({ brokenStorage: true })

    expect(reloadForStaleChunk(win)).toBe(true)
    expect(reload).toHaveBeenCalledTimes(1)
  })
})

describe('installChunkReloadGuard', () => {
  it('reloads and cancels the event on vite:preloadError', () => {
    const { win, reload, dispatchPreloadError } = fakeWindow()
    installChunkReloadGuard(win)

    const event = dispatchPreloadError()

    expect(reload).toHaveBeenCalledTimes(1)
    expect(event.defaultPrevented).toBe(true)
  })

  it('does not cancel the event when the cooldown suppresses the reload', () => {
    const { win, reload, dispatchPreloadError } = fakeWindow()
    installChunkReloadGuard(win)

    dispatchPreloadError()
    const second = dispatchPreloadError()

    expect(reload).toHaveBeenCalledTimes(1)
    expect(second.defaultPrevented).toBe(false)
  })
})
