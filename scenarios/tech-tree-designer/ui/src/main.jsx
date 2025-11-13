import React from 'react'
import ReactDOM from 'react-dom/client'
import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child'
import { resolveApiBase, buildApiUrl } from '@vrooli/api-base'
import App from './App.jsx'
import './index.css'

const resolveAppVersion = () => {
  if (typeof __APP_VERSION__ !== 'undefined' && `${__APP_VERSION__}`.trim().length > 0) {
    return `${__APP_VERSION__}`.trim()
  }

  const envVersion = import.meta.env && import.meta.env.VITE_APP_VERSION
  if (typeof envVersion === 'string' && envVersion.trim().length > 0) {
    return envVersion.trim()
  }

  return 'development'
}

const APP_VERSION = resolveAppVersion()

const API_BASE = resolveApiBase({ appendSuffix: true })

const resolveApiHealthUrl = () => buildApiUrl('/health', {
  baseUrl: API_BASE
})

// Update health status
if (typeof window !== 'undefined') {
  window.__TECH_TREE_HEALTH__ = {
    status: 'healthy',
    service: 'tech-tree-designer-ui',
    timestamp: new Date().toISOString(),
    readiness: true,
    version: APP_VERSION,
    api_connectivity: {
      connected: true,
      api_url: resolveApiHealthUrl(),
      last_check: new Date().toISOString(),
      error: null,
      latency_ms: null
    }
  };
}

if (typeof window !== 'undefined' && window.parent !== window) {
    initIframeBridgeChild({ appId: 'tech-tree-designer' })
}

ReactDOM.createRoot(document.getElementById('root')).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
)
