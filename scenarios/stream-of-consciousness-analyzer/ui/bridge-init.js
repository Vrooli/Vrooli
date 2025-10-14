import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child'

if (typeof window !== 'undefined') {
  if (typeof window.initIframeBridgeChild !== 'function') {
    window.initIframeBridgeChild = initIframeBridgeChild
  }

  if (window.parent !== window && !window.__streamOfConsciousnessBridgeInitialized) {
    window.initIframeBridgeChild({ appId: 'stream-of-consciousness-analyzer-ui' })
    window.__streamOfConsciousnessBridgeInitialized = true
  }
}
