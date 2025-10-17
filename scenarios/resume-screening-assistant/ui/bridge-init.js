import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child'

const BRIDGE_STATE_KEY = '__hyperRecruitBridgeInitialized'

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
    console.warn('[HyperRecruit3000] Unable to parse parent origin for iframe bridge', error)
  }

  initIframeBridgeChild({ parentOrigin, appId: 'resume-screening-assistant' })
  window[BRIDGE_STATE_KEY] = true
}

if (document.readyState === 'complete' || document.readyState === 'interactive') {
  initializeBridge()
} else {
  document.addEventListener('DOMContentLoaded', initializeBridge, { once: true })
}
