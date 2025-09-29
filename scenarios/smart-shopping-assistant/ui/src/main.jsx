import React from 'react'
import ReactDOM from 'react-dom/client'
import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child'
import App from './App'
import './index.css'

if (typeof window !== 'undefined' && window.parent !== window && !window.__smartShoppingBridgeInitialized) {
  let parentOrigin
  try {
    if (document.referrer) {
      parentOrigin = new URL(document.referrer).origin
    }
  } catch (error) {
    console.warn('[SmartShopping] Unable to parse parent origin for iframe bridge', error)
  }

  initIframeBridgeChild({ parentOrigin, appId: 'smart-shopping-assistant' })
  window.__smartShoppingBridgeInitialized = true
}

ReactDOM.createRoot(document.getElementById('root')).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
)
