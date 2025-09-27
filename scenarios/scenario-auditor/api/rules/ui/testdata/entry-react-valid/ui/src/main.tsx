import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'
import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child'

if (typeof window !== 'undefined' && window.parent !== window) {
  initIframeBridgeChild({ appId: 'demo' })
}

ReactDOM.createRoot(document.getElementById('root')!).render(<App />)
