import React from 'react'
import ReactDOM from 'react-dom/client'
import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child'
import App from './App.tsx'
import './index.css'

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
    <App />
  </React.StrictMode>,
)
