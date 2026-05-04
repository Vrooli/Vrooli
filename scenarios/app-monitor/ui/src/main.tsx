import { logger } from '@/services/logger';
import React from 'react'
import ReactDOM from 'react-dom/client'
import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child'
import App from './App.tsx'
import { SnackStackProvider } from '@/notifications/SnackStackProvider'
import { onProfilerRender } from '@/lib/profiler'
import './index.css'
import './shared.css'
import './components/app-modal/TabStateView.css'

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
  } catch {
    // best-effort debug logging
  }
}

if (typeof window !== 'undefined' && typeof window.history !== 'undefined') {
  const history = window.history as typeof window.history & { __appMonitorDebugPatched?: boolean }
  if (!history.__appMonitorDebugPatched) {
    const wrapHistoryMethod = <T extends 'pushState' | 'replaceState'>(method: T) => {
      const original = history[method]
      return function patched(this: typeof history, state: unknown, title: string, url?: string | URL | null) {
        const normalizedUrl = typeof url === 'string' ? url : url?.toString() ?? null
        sendDebugEvent(`history-${method}`, {
          state,
          title,
          url: normalizedUrl,
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

// DOC: scenarios/app-monitor/docs/internal/SEAMS.md#recursive-self-embedding-prevention
const APP_MONITOR_DEPTH_KEY = '__appMonitorDepth'
const APP_MONITOR_MAX_DEPTH = 1

function getIframeDepth(): number {
  if (window.parent === window) return 0
  try {
    const parentDepth = (window.parent as unknown as Record<string, unknown>)[APP_MONITOR_DEPTH_KEY]
    return typeof parentDepth === 'number' ? parentDepth + 1 : 1
  } catch {
    return 1 // Cross-origin = assume depth 1
  }
}

const currentDepth = getIframeDepth()
;(window as unknown as Record<string, unknown>)[APP_MONITOR_DEPTH_KEY] = currentDepth

const rootEl = document.getElementById('root')
if (currentDepth > APP_MONITOR_MAX_DEPTH) {
  sendDebugEvent('recursive-embed-blocked', { depth: currentDepth })
  if (rootEl) {
    rootEl.innerHTML = '<div style="padding:2rem;font-family:system-ui;color:#ef4444">'
      + '<h2>Recursive Embedding Detected</h2>'
      + '<p>App Monitor cannot render inside itself.</p></div>'
  }
} else if (rootEl) {
  ReactDOM.createRoot(rootEl).render(
    <React.StrictMode>
      <SnackStackProvider>
        <React.Profiler id="App" onRender={onProfilerRender}>
          <App />
        </React.Profiler>
      </SnackStackProvider>
    </React.StrictMode>,
  )
}
