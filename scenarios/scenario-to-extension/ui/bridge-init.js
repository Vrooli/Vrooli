import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child'

const BRIDGE_FLAG = '__scenarioToExtensionBridgeInitialized'

(function bootstrapIframeBridge() {
  if (typeof window === 'undefined') {
    return
  }

  if (window.parent === window) {
    return
  }

  if (window[BRIDGE_FLAG]) {
    return
  }

  let parentOrigin
  try {
    if (document.referrer) {
      parentOrigin = new URL(document.referrer).origin
    }
  } catch (error) {
    console.warn('[scenario-to-extension] Unable to determine parent origin for iframe bridge:', error)
  }

  if (typeof window.initIframeBridgeChild !== 'function') {
    window.initIframeBridgeChild = initIframeBridgeChild
  }

  initIframeBridgeChild({
    appId: 'scenario-to-extension-ui',
    parentOrigin,
    captureLogs: {
      enabled: true,
      streaming: true,
      bufferSize: 500,
    },
    captureNetwork: {
      enabled: true,
      streaming: true,
      bufferSize: 250,
    },
  })
  window[BRIDGE_FLAG] = true
})()
