import React from 'react'
import ReactDOM from 'react-dom/client'
import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child'
import App from './App.jsx'
import './index.css'

const bootstrapIframeBridge = () => {
  if (window.__scalableAppBridgeInitialized) {
    return
  }

  let parentOrigin
  try {
    if (document.referrer) {
      parentOrigin = new URL(document.referrer).origin
    }
  } catch (error) {
    console.warn('[ScalableAppCookbook] Unable to detect parent origin for iframe bridge', error)
  }

  initIframeBridgeChild({ appId: 'scalable-app-cookbook', parentOrigin })
  window.__scalableAppBridgeInitialized = true
}

if (typeof window !== 'undefined' && window.parent !== window) {
  bootstrapIframeBridge()
}

const rootElement = document.getElementById('root')

if (!rootElement) {
  throw new Error('Root element #root not found')
}

ReactDOM.createRoot(rootElement).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
)
