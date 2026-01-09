import { logger } from '@/services/logger';
import React from 'react'
import ReactDOM from 'react-dom/client'
import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child'
import App from './App.tsx'
import { SnackStackProvider } from '@/notifications/SnackStackProvider'
import './index.css'

const sendDebugEvent = (event: string, detail?: Record<string, unknown>) => {
  try {
    if (typeof navigator !== 'undefined' && typeof navigator.sendBeacon === 'function') {
      const blob = new Blob([
        JSON.stringify({
          event,
          timestamp: Date.now(),
          detail: detail ?? null,
          userAgent: typeof navigator !== 'undefined' ? navigator.userAgent : undefined,
        }),
      ], { type: 'application/json' })
      navigator.sendBeacon('/__debug/client-event', blob)
    } else {
      void fetch('/__debug/client-event', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          event,
          timestamp: Date.now(),
          detail: detail ?? null,
          userAgent: typeof navigator !== 'undefined' ? navigator.userAgent : undefined,
        }),
        keepalive: true,
      })
    }
  } catch (error) {
    // best-effort debug logging
  }
}

if (typeof window !== 'undefined' && typeof window.history !== 'undefined') {
  const history = window.history as typeof window.history & { __appMonitorDebugPatched?: boolean }
  const globalWindow = window as typeof window & {
    __appMonitorPreviewGuard?: {
      active: boolean
      armedAt: number
      ttl: number
      key: string | null
      appId: string | null
      recoverPath: string | null
      ignoreNextPopstate?: boolean
      lastSuppressedAt?: number
      recoverState?: unknown
    }
  }
  if (!globalWindow.__appMonitorPreviewGuard) {
    globalWindow.__appMonitorPreviewGuard = {
      active: false,
      armedAt: 0,
      ttl: 15000,
      key: null,
      appId: null,
      recoverPath: null,
      ignoreNextPopstate: false,
      lastSuppressedAt: 0,
      recoverState: null,
    }
  }
  if (!history.__appMonitorDebugPatched) {
    const resolvePathWithSearch = (target?: string | URL | null) => {
      try {
        if (typeof target === 'string') {
          const resolved = target.startsWith('http') ? new URL(target) : new URL(target, window.location.origin)
          return `${resolved.pathname}${resolved.search ?? ''}`
        }
        if (target instanceof URL) {
          return `${target.pathname}${target.search ?? ''}`
        }
      } catch (error) {
        // Ignore malformed URLs and fall back to current location
      }
      return `${window.location.pathname}${window.location.search ?? ''}`
    }

    const wrapHistoryMethod = <T extends 'pushState' | 'replaceState'>(method: T) => {
      const original = history[method]
      return function patched(this: typeof history, state: unknown, title: string, url?: string | URL | null) {
        const normalizedUrl = typeof url === 'string' ? url : url?.toString() ?? null
        sendDebugEvent(`history-${method}`, {
          state,
          title,
          url: normalizedUrl,
        })
        const result = original.apply(this, [state, title, url])
        try {
          const guard = globalWindow.__appMonitorPreviewGuard
          if (guard?.active && guard.recoverPath) {
            const targetPath = resolvePathWithSearch(url)
            if (targetPath === guard.recoverPath) {
              guard.recoverState = state
              if (
                typeof state === 'object'
                && state !== null
                && 'key' in state
                && typeof (state as Record<string, unknown>).key === 'string'
              ) {
                guard.key = (state as Record<string, unknown>).key as string
              }
              globalWindow.__appMonitorPreviewGuard = guard
              sendDebugEvent('history-guard-primed', {
                method,
                targetPath,
              })
            }
          }
        } catch (error) {
          // Guard instrumentation errors are non-fatal
        }
        return result
      }
    }

    history.pushState = wrapHistoryMethod('pushState')
    history.replaceState = wrapHistoryMethod('replaceState')

    const extractStateKey = (state: unknown): string | null => {
      if (!state || typeof state !== 'object') {
        return null
      }
      if ('key' in state && typeof (state as Record<string, unknown>).key === 'string') {
        return (state as Record<string, unknown>).key as string
      }
      return null
    }

    window.addEventListener('popstate', (event) => {
      sendDebugEvent('history-popstate', {
        state: event.state,
      })
      const guard = globalWindow.__appMonitorPreviewGuard!
      if (guard.ignoreNextPopstate) {
        guard.ignoreNextPopstate = false
        sendDebugEvent('history-popstate-ignored', {
          reason: 'recover-forward',
          state: event.state,
        })
        return
      }

      const now = Date.now()
      const currentPath = `${window.location.pathname}${window.location.search ?? ''}`
      const withinGuard = guard.active && now - guard.armedAt <= guard.ttl
      const guardKey = guard.key ?? null
      const poppedStateKey = extractStateKey(event.state)
      const stateMismatch = Boolean(guardKey && guardKey !== poppedStateKey)

      if (withinGuard) {
        sendDebugEvent('history-popstate-suppressed', {
          state: event.state,
          currentPath,
          guard,
          stateMismatch,
        })
        guard.lastSuppressedAt = now
        guard.ignoreNextPopstate = true
        guard.active = true
        guard.armedAt = now
        const recoverUrl = guard.recoverPath ?? currentPath
        const recoverState = guard.recoverState ?? history.state
        history.pushState(recoverState ?? {}, '', recoverUrl)
        window.dispatchEvent(new PopStateEvent('popstate', { state: recoverState }))
        return
      }
    })
    history.__appMonitorDebugPatched = true
  }
}

const BRIDGE_FLAG = '__appMonitorBridgeInitialized'

if (
  typeof window !== 'undefined' &&
  window.parent !== window &&
  !((window as unknown as Record<string, unknown>)[BRIDGE_FLAG] ?? false)
) {
  let parentOrigin: string | undefined

  try {
    if (document.referrer) {
      parentOrigin = new URL(document.referrer).origin
    }
  } catch (error) {
    logger.warn('[app-monitor] Unable to determine parent origin for iframe bridge', error)
  }

  initIframeBridgeChild({ parentOrigin, appId: 'app-monitor' })
  ;(window as unknown as Record<string, unknown>)[BRIDGE_FLAG] = true
}

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <SnackStackProvider>
      <App />
    </SnackStackProvider>
  </React.StrictMode>,
)
