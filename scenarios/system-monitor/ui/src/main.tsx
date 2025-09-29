import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child'
import App from './App.tsx'

declare global {
  interface Window {
    __systemMonitorBridgeInitialized?: boolean;
  }
}

if (typeof window !== 'undefined' && window.parent !== window && !window.__systemMonitorBridgeInitialized) {
  let parentOrigin: string | undefined
  try {
    if (document.referrer) {
      parentOrigin = new URL(document.referrer).origin
    }
  } catch (error) {
    console.warn('[SystemMonitor] Unable to parse parent origin for iframe bridge', error)
  }

  initIframeBridgeChild({ parentOrigin, appId: 'system-monitor' })
  window.__systemMonitorBridgeInitialized = true
}

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <BrowserRouter>
      <App />
    </BrowserRouter>
  </StrictMode>,
)
