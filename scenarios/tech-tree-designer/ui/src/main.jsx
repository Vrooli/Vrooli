import React from 'react'
import ReactDOM from 'react-dom/client'
import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child'
import { resolveApiBase, buildApiUrl } from '@vrooli/api-base'
import App from './App.jsx'
import './index.css'

const resolveDefaultApiPort = () => {
  const candidates = [
    import.meta.env.VITE_API_PORT,
    import.meta.env.VITE_PROXY_API_PORT,
    import.meta.env.API_PORT
  ]

  for (const candidate of candidates) {
    if (typeof candidate === 'string' && candidate.trim().length > 0) {
      return candidate.trim()
    }
  }

  return '8080'
}

const DEFAULT_API_PORT = resolveDefaultApiPort()

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

const API_BASE = resolveApiBase({
  explicitUrl: typeof import.meta.env.VITE_API_BASE_URL === 'string'
    ? import.meta.env.VITE_API_BASE_URL.trim()
    : undefined,
  defaultPort: DEFAULT_API_PORT,
  appendSuffix: true
})

const resolveApiHealthUrl = () => buildApiUrl('/health', {
  baseUrl: API_BASE,
  defaultPort: DEFAULT_API_PORT,
  appendSuffix: false
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
