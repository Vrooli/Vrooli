import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child'

const BRIDGE_STATE_KEY = '__recommendationEngineBridgeInitialized'

function initializeBridge() {
  if (typeof window === 'undefined' || window.parent === window) {
    return
  }

  if (window[BRIDGE_STATE_KEY]) {
    return
  }

  let parentOrigin
  try {
    if (document.referrer) {
      parentOrigin = new URL(document.referrer).origin
    }
  } catch (error) {
    console.warn('[RecommendationEngine] Unable to determine parent origin for iframe bridge', error)
  }

  initIframeBridgeChild({ parentOrigin, appId: 'recommendation-engine' })
  window[BRIDGE_STATE_KEY] = true
}

if (document.readyState === 'complete' || document.readyState === 'interactive') {
  initializeBridge()
} else {
  document.addEventListener('DOMContentLoaded', initializeBridge, { once: true })
}
