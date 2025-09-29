import { StrictMode } from 'react'
import ReactDOM from 'react-dom/client'
import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child'
import App from './App.tsx'

declare global {
  interface Window {
    __textToolsBridgeInitialized?: boolean
  }
}

if (typeof window !== 'undefined' && window.parent !== window && !window.__textToolsBridgeInitialized) {
  let parentOrigin: string | undefined
  try {
    if (document.referrer) {
      parentOrigin = new URL(document.referrer).origin
    }
  } catch (error) {
    console.warn('[TextTools] Unable to parse parent origin for iframe bridge', error)
  }

  initIframeBridgeChild({ parentOrigin, appId: 'text-tools' })
  window.__textToolsBridgeInitialized = true
}

ReactDOM.createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
