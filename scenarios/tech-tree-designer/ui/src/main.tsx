import React from 'react'
import ReactDOM from 'react-dom/client'
import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child'
import { resolveApiBase, buildApiUrl } from '@vrooli/api-base'
import App from './App'
import './index.css'
import './styles/SectorGroupNode.css'

type HealthStatus = 'healthy' | 'degraded'

type TechTreeHealth = {
  status: HealthStatus
  service: string
  timestamp: string
  readiness: boolean
  version: string
  api_connectivity: {
    connected: boolean
    api_url: string
    last_check: string
    error: null | {
      code: string
      message: string
      category: string
      retryable: boolean
    }
    latency_ms: number | null
  }
}

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

const resolveApiHealthUrl = () =>
  buildApiUrl('/health', {
    baseUrl: API_BASE
  })

if (typeof window !== 'undefined') {
  const healthSnapshot: TechTreeHealth = {
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
  }

  ;(window as typeof window & { __TECH_TREE_HEALTH__?: TechTreeHealth }).__TECH_TREE_HEALTH__ = healthSnapshot
}

if (typeof window !== 'undefined' && window.parent !== window) {
  initIframeBridgeChild({ appId: 'tech-tree-designer' })
}

ReactDOM.createRoot(document.getElementById('root') as HTMLElement).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
)
