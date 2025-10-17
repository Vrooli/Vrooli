import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child'

const BRIDGE_FLAG = '__agentDashboardBridgeInitialized'

function deriveParentOrigin() {
  if (!document.referrer) {
    return undefined
  }
  try {
    return new URL(document.referrer).origin
  } catch (error) {
    console.warn('[AgentDashboard] Unable to parse parent origin for iframe bridge', error)
    return undefined
  }
}

function initializeBridge() {
  if (typeof window === 'undefined') {
    return
  }
  if (window.parent === window) {
    return
  }
  if (window[BRIDGE_FLAG]) {
    return
  }

  const parentOrigin = deriveParentOrigin()
  initIframeBridgeChild({ parentOrigin, appId: 'agent-dashboard' })
  window[BRIDGE_FLAG] = true
}

try {
  initializeBridge()
} catch (error) {
  console.error('[AgentDashboard] Failed to initialize iframe bridge', error)
}

export {}
