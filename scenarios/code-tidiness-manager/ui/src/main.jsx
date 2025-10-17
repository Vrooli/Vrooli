import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'
import './index.css'
import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child'

if (typeof window !== 'undefined' && window.parent !== window && !window.__codeTidinessBridgeInitialized) {
  let parentOrigin
  try {
    if (document.referrer) {
      parentOrigin = new URL(document.referrer).origin
    }
  } catch (error) {
    console.warn('[CodeTidinessManager] Unable to determine parent origin for iframe bridge', error)
  }

  initIframeBridgeChild({ parentOrigin, appId: 'code-tidiness-manager' })
  window.__codeTidinessBridgeInitialized = true
}

ReactDOM.createRoot(document.getElementById('root')).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
)
