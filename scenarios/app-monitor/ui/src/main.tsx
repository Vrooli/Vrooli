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
    const wrapHistoryMethod = <T extends 'pushState' | 'replaceState'>(method: T) => {
      const original = history[method]
      return function patched(this: typeof history, state: unknown, title: string, url?: string | URL | null) {
        sendDebugEvent(`history-${method}`, {
          state,
          title,
          url: typeof url === 'string' ? url : url?.toString() ?? null,
        })
        return original.apply(this, [state, title, url])
      }
    }

    history.pushState = wrapHistoryMethod('pushState')
    history.replaceState = wrapHistoryMethod('replaceState')
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
      const returningToApps = guard.recoverPath && currentPath !== guard.recoverPath

      if (withinGuard && returningToApps) {
        sendDebugEvent('history-popstate-suppressed', {
          state: event.state,
          currentPath,
          guard,
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
    console.warn('[app-monitor] Unable to determine parent origin for iframe bridge', error)
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
